package engine

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/updatecli/updatecli/pkg/core/config"
	"github.com/updatecli/updatecli/pkg/core/pipeline/resource"
	"github.com/updatecli/updatecli/pkg/core/pipeline/scm"
	"github.com/updatecli/updatecli/pkg/core/pipeline/source"
	"github.com/updatecli/updatecli/pkg/plugins/resources/updateclihttp"
	githubscm "github.com/updatecli/updatecli/pkg/plugins/scms/github"
)

func TestManifestResourceIdentity(t *testing.T) {
	t.Parallel()

	newManifest := func(url string, sourcesCount int) config.Spec {
		manifest := config.Spec{
			Name:    "discovered manifest",
			Sources: map[string]source.Config{},
		}
		for i := 0; i < sourcesCount; i++ {
			manifest.Sources[fmt.Sprintf("source-%d", i)] = source.Config{
				ResourceConfig: resource.ResourceConfig{
					Name: "version",
					Kind: "http",
					Spec: updateclihttp.Spec{
						Url:                  url,
						ReturnResponseHeader: "etag",
					},
				},
			}
		}
		return manifest
	}

	t.Run("a manifest with exactly one source has a resource identity", func(t *testing.T) {
		t.Parallel()

		identity, ok := manifestResourceIdentity(newManifest("https://example.com/releases/latest", 1))
		require.True(t, ok)
		require.NotEmpty(t, identity)
	})

	t.Run("the resource identity is deterministic", func(t *testing.T) {
		t.Parallel()

		first, ok := manifestResourceIdentity(newManifest("https://example.com/releases/latest", 1))
		require.True(t, ok)

		second, ok := manifestResourceIdentity(newManifest("https://example.com/releases/latest", 1))
		require.True(t, ok)

		require.Equal(t, first, second)
	})

	t.Run("two different resources have a different identity", func(t *testing.T) {
		t.Parallel()

		first, ok := manifestResourceIdentity(newManifest("https://example.com/releases/latest", 1))
		require.True(t, ok)

		second, ok := manifestResourceIdentity(newManifest("https://example.com/releases/stable", 1))
		require.True(t, ok)

		require.NotEqual(t, first, second)
	})

	t.Run("the resource's own display name does not affect the identity", func(t *testing.T) {
		t.Parallel()

		newManifestWithName := func(name, url string) config.Spec {
			return config.Spec{
				Name: "discovered manifest",
				Sources: map[string]source.Config{
					"source": {
						ResourceConfig: resource.ResourceConfig{
							Name: name,
							Kind: "http",
							Spec: updateclihttp.Spec{
								Url:                  url,
								ReturnResponseHeader: "etag",
							},
						},
					},
				},
			}
		}

		// Regression test: the helmfile crawler sets the source "name" to a per-release
		// display label (e.g. `Get latest "alpha-docs-jenkins-io" Helm Chart version`), which
		// must not be part of the identity, or two releases of the exact same chart would be
		// wrongly treated as different resources and never grouped together.
		first, ok := manifestResourceIdentity(newManifestWithName("Get latest release-a version", "https://example.com/releases/latest"))
		require.True(t, ok)

		second, ok := manifestResourceIdentity(newManifestWithName("Get latest release-b version", "https://example.com/releases/latest"))
		require.True(t, ok)

		require.Equal(t, first, second)
	})

	t.Run("a manifest with no source has no resource identity", func(t *testing.T) {
		t.Parallel()

		_, ok := manifestResourceIdentity(newManifest("https://example.com/releases/latest", 0))
		require.False(t, ok)
	})

	t.Run("a manifest with more than one source has no resource identity", func(t *testing.T) {
		t.Parallel()

		_, ok := manifestResourceIdentity(newManifest("https://example.com/releases/latest", 2))
		require.False(t, ok)
	})
}

func TestAutodiscoveryManifestFingerprintIgnoresSecrets(t *testing.T) {
	t.Parallel()

	manifestWithFirstSecrets := config.Spec{
		Name: "discovered manifest",
		Sources: map[string]source.Config{
			"version": {
				ResourceConfig: resource.ResourceConfig{
					Name: "version",
					Kind: "http",
					// #nosec G101 -- This URL is not real and is only used for testing the fingerprinting logic.
					Spec: updateclihttp.Spec{
						Url:                  "https://user:first-secret@example.com/releases/latest",
						ReturnResponseHeader: "etag",
					},
				},
			},
		},
		SCMs: map[string]scm.Config{
			"default": {
				Kind: "github",
				Spec: githubscm.Spec{
					Owner:      "updatecli",
					Repository: "updatecli",
					Token:      "first-token",
					Branch:     "main",
				},
			},
		},
	}

	manifestWithSecondSecrets := config.Spec{
		Name: "discovered manifest",
		Sources: map[string]source.Config{
			"version": {
				ResourceConfig: resource.ResourceConfig{
					Name: "version",
					Kind: "http",
					Spec: updateclihttp.Spec{
						// #nosec G101 -- This URL is not real and is only used for testing the fingerprinting logic.
						Url:                  "https://user:second-secret@example.com/releases/latest",
						ReturnResponseHeader: "etag",
					},
				},
			},
		},
		SCMs: map[string]scm.Config{
			"default": {
				Kind: "github",
				Spec: githubscm.Spec{
					Owner:      "updatecli",
					Repository: "updatecli",
					Token:      "second-token",
					Branch:     "main",
				},
			},
		},
	}

	firstFingerprint, err := autodiscoveryManifestFingerprint(manifestWithFirstSecrets, "parent-pipeline")
	require.NoError(t, err)

	secondFingerprint, err := autodiscoveryManifestFingerprint(manifestWithSecondSecrets, "parent-pipeline")
	require.NoError(t, err)

	require.Equal(t, firstFingerprint, secondFingerprint)
}

func TestAutodiscoveryManifestFingerprintTracksSanitizedIdentity(t *testing.T) {
	t.Parallel()

	manifestA := config.Spec{
		Name: "discovered manifest",
		Sources: map[string]source.Config{
			"version": {
				ResourceConfig: resource.ResourceConfig{
					Name: "version",
					Kind: "http",
					Spec: updateclihttp.Spec{
						Url:                  "https://example.com/releases/latest",
						ReturnResponseHeader: "etag",
					},
				},
			},
		},
		SCMs: map[string]scm.Config{
			"default": {
				Kind: "github",
				Spec: githubscm.Spec{
					Owner:      "updatecli",
					Repository: "updatecli",
					Branch:     "main",
				},
			},
		},
	}

	manifestB := config.Spec{
		Name: "discovered manifest",
		Sources: map[string]source.Config{
			"version": {
				ResourceConfig: resource.ResourceConfig{
					Name: "version",
					Kind: "http",
					Spec: updateclihttp.Spec{
						Url:                  "https://example.com/releases/stable",
						ReturnResponseHeader: "etag",
					},
				},
			},
		},
		SCMs: map[string]scm.Config{
			"default": {
				Kind: "github",
				Spec: githubscm.Spec{
					Owner:      "updatecli",
					Repository: "website",
					Branch:     "main",
				},
			},
		},
	}

	fingerprintA, err := autodiscoveryManifestFingerprint(manifestA, "parent-pipeline")
	require.NoError(t, err)

	fingerprintB, err := autodiscoveryManifestFingerprint(manifestB, "parent-pipeline")
	require.NoError(t, err)

	require.NotEqual(t, fingerprintA, fingerprintB)
}
