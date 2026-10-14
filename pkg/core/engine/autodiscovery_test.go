package engine

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/updatecli/updatecli/pkg/core/config"
	"github.com/updatecli/updatecli/pkg/core/pipeline/condition"
	"github.com/updatecli/updatecli/pkg/core/pipeline/resource"
	"github.com/updatecli/updatecli/pkg/core/pipeline/scm"
	"github.com/updatecli/updatecli/pkg/core/pipeline/source"
	"github.com/updatecli/updatecli/pkg/core/pipeline/target"
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

// newManifestWithSourceSpec builds a manifest whose single source spec is a
// map[string]interface{}, matching the shape a discovered manifest actually has once its
// generated YAML is unmarshaled into a config.Spec (as opposed to a typed Go resource.Spec
// struct, which is how other tests in this file build manifests by hand).
func newManifestWithSourceSpec(spec map[string]interface{}, sourcesCount int) config.Spec {
	manifest := config.Spec{
		Name:    "discovered manifest",
		Sources: map[string]source.Config{},
	}
	for i := 0; i < sourcesCount; i++ {
		manifest.Sources[fmt.Sprintf("source-%d", i)] = source.Config{
			ResourceConfig: resource.ResourceConfig{
				Name: "Get latest version",
				Kind: "helmchart",
				Spec: spec,
			},
		}
	}
	return manifest
}

func TestManifestResourceName(t *testing.T) {
	t.Parallel()

	t.Run("returns the single source spec's name", func(t *testing.T) {
		t.Parallel()

		name, ok := manifestResourceName(newManifestWithSourceSpec(map[string]interface{}{"name": "nginx-website", "url": "https://example.com"}, 1))
		require.True(t, ok)
		require.Equal(t, "nginx-website", name)
	})

	t.Run("returns the single source spec's image, for dockerimage/dockerdigest sources", func(t *testing.T) {
		t.Parallel()

		name, ok := manifestResourceName(newManifestWithSourceSpec(map[string]interface{}{"image": "jenkins/jenkins", "tagfilter": "lts"}, 1))
		require.True(t, ok)
		require.Equal(t, "jenkins/jenkins", name)
	})

	t.Run("returns the single source spec's package, for cargopackage sources", func(t *testing.T) {
		t.Parallel()

		name, ok := manifestResourceName(newManifestWithSourceSpec(map[string]interface{}{"package": "serde"}, 1))
		require.True(t, ok)
		require.Equal(t, "serde", name)
	})

	t.Run("returns groupid:artifactid, for maven sources", func(t *testing.T) {
		t.Parallel()

		name, ok := manifestResourceName(newManifestWithSourceSpec(map[string]interface{}{"groupid": "org.jenkins-ci.plugins", "artifactid": "git"}, 1))
		require.True(t, ok)
		require.Equal(t, "org.jenkins-ci.plugins:git", name)
	})

	t.Run("prefers name over image when, implausibly, a spec has both", func(t *testing.T) {
		t.Parallel()

		name, ok := manifestResourceName(newManifestWithSourceSpec(map[string]interface{}{"name": "nginx-website", "image": "jenkins/jenkins"}, 1))
		require.True(t, ok)
		require.Equal(t, "nginx-website", name)
	})

	t.Run("no name when the spec has none of the known identity fields", func(t *testing.T) {
		t.Parallel()

		_, ok := manifestResourceName(newManifestWithSourceSpec(map[string]interface{}{"url": "https://example.com"}, 1))
		require.False(t, ok)
	})

	t.Run("no name for maven when only one of groupid/artifactid is set", func(t *testing.T) {
		t.Parallel()

		_, ok := manifestResourceName(newManifestWithSourceSpec(map[string]interface{}{"groupid": "org.jenkins-ci.plugins"}, 1))
		require.False(t, ok)

		_, ok = manifestResourceName(newManifestWithSourceSpec(map[string]interface{}{"artifactid": "git"}, 1))
		require.False(t, ok)
	})

	t.Run("no name when the spec's name is empty", func(t *testing.T) {
		t.Parallel()

		_, ok := manifestResourceName(newManifestWithSourceSpec(map[string]interface{}{"name": ""}, 1))
		require.False(t, ok)
	})

	t.Run("no name with zero or more than one source", func(t *testing.T) {
		t.Parallel()

		spec := map[string]interface{}{"name": "nginx-website"}

		_, ok := manifestResourceName(newManifestWithSourceSpec(spec, 0))
		require.False(t, ok)

		_, ok = manifestResourceName(newManifestWithSourceSpec(spec, 2))
		require.False(t, ok)
	})
}

// manifestWithConditionAndSource builds a manifest with one condition (map-shaped spec, with
// the given "value") and one source (map-shaped spec, with the given "name"), matching the
// shape the "helmfile" crawler generates: a condition checking the raw chart identity (e.g.
// "jenkins-infra/rsyncd") and a source resolved against the chart's own short name (e.g.
// "rsyncd").
func manifestWithConditionAndSource(conditionValue, sourceName string) config.Spec {
	manifest := newManifestWithSourceSpec(map[string]interface{}{"name": sourceName}, 1)
	manifest.Conditions = map[string]condition.Config{
		"condition": {
			ResourceConfig: resource.ResourceConfig{
				Kind: "yaml",
				Spec: map[string]interface{}{"value": conditionValue},
			},
		},
	}
	return manifest
}

// manifestWithTarget extends a manifest with a single target whose "name" and "sourceid"
// mirror the "helmfile" crawler's own convention, built from releaseName the same way
// releases.go names its target after the release.
func manifestWithTarget(manifest config.Spec, releaseName string) config.Spec {
	manifest.Targets = map[string]target.Config{
		releaseName: {
			ResourceConfig: resource.ResourceConfig{
				Kind: "yaml",
				Name: fmt.Sprintf(`deps(helmfile): update %q Helm Chart version to {{ source %q}}`, releaseName, releaseName),
			},
			SourceID: releaseName,
		},
	}
	return manifest
}

func TestRewriteHelmfileTargetNames(t *testing.T) {
	t.Parallel()

	t.Run("replaces the release name with the chart identity, keeping the source reference", func(t *testing.T) {
		t.Parallel()

		manifest := manifestWithTarget(manifestWithConditionAndSource("jenkins-infra/nginx-website", "nginx-website"), "pkg-origin-jenkins-io")

		rewriteHelmfileTargetNames("helmfile", manifest)

		require.Equal(t,
			`deps(helmfile): update "jenkins-infra/nginx-website" Helm Chart version to {{ source "pkg-origin-jenkins-io"}}`,
			manifest.Targets["pkg-origin-jenkins-io"].Name)
	})

	t.Run("leaves target names untouched for any other crawler", func(t *testing.T) {
		t.Parallel()

		manifest := manifestWithTarget(manifestWithConditionAndSource("jenkins-infra/nginx-website", "nginx-website"), "pkg-origin-jenkins-io")
		original := manifest.Targets["pkg-origin-jenkins-io"].Name

		rewriteHelmfileTargetNames("npm", manifest)

		require.Equal(t, original, manifest.Targets["pkg-origin-jenkins-io"].Name)
	})

	t.Run("leaves target names untouched when there is no chart identity to use", func(t *testing.T) {
		t.Parallel()

		manifest := manifestWithTarget(newManifestWithSourceSpec(map[string]interface{}{"name": "nginx-website"}, 1), "pkg-origin-jenkins-io")
		original := manifest.Targets["pkg-origin-jenkins-io"].Name

		rewriteHelmfileTargetNames("helmfile", manifest)

		require.Equal(t, original, manifest.Targets["pkg-origin-jenkins-io"].Name)
	})
}

func TestHelmfileChartIdentity(t *testing.T) {
	t.Parallel()

	t.Run("returns the single condition spec's value", func(t *testing.T) {
		t.Parallel()

		identity, ok := helmfileChartIdentity(manifestWithConditionAndSource("jenkins-infra/rsyncd", "rsyncd"))
		require.True(t, ok)
		require.Equal(t, "jenkins-infra/rsyncd", identity)
	})

	t.Run("no identity with zero or more than one condition", func(t *testing.T) {
		t.Parallel()

		manifest := manifestWithConditionAndSource("jenkins-infra/rsyncd", "rsyncd")

		manifest.Conditions = nil
		_, ok := helmfileChartIdentity(manifest)
		require.False(t, ok)

		manifest = manifestWithConditionAndSource("jenkins-infra/rsyncd", "rsyncd")
		manifest.Conditions["other"] = manifest.Conditions["condition"]
		_, ok = helmfileChartIdentity(manifest)
		require.False(t, ok)
	})
}

func TestManifestResourceTitle(t *testing.T) {
	t.Parallel()

	t.Run("helmfile prefers the chart identity over the release name", func(t *testing.T) {
		t.Parallel()

		// Two different release names, same chart identity: the title must be identical, or
		// grouping their pull requests together under "groupby: resource" would still end up
		// with an arbitrary, release-specific title.
		titleA, okA := manifestResourceTitle("helmfile", manifestWithConditionAndSource("jenkins-infra/rsyncd", "rsyncd"))
		titleB, okB := manifestResourceTitle("helmfile", manifestWithConditionAndSource("jenkins-infra/rsyncd", "rsyncd"))

		require.True(t, okA)
		require.True(t, okB)
		require.Equal(t, titleA, titleB)
		require.Contains(t, titleA, "jenkins-infra/rsyncd")
	})

	t.Run("other crawlers fall back to the resource name", func(t *testing.T) {
		t.Parallel()

		title, ok := manifestResourceTitle("npm", newManifestWithSourceSpec(map[string]interface{}{"name": "express"}, 1))
		require.True(t, ok)
		require.Contains(t, title, "express")
	})

	t.Run("no title when there is neither a chart identity nor a resource name", func(t *testing.T) {
		t.Parallel()

		_, ok := manifestResourceTitle("helmfile", newManifestWithSourceSpec(map[string]interface{}{}, 1))
		require.False(t, ok)
	})
}

func TestAppendActionLabel(t *testing.T) {
	t.Parallel()

	t.Run("appends to an existing labels list", func(t *testing.T) {
		t.Parallel()

		spec := map[string]interface{}{"labels": []interface{}{"dependencies"}}

		result := appendActionLabel(spec, "chart:nginx-website")

		require.Equal(t, []interface{}{"dependencies", "chart:nginx-website"}, result.(map[string]interface{})["labels"])
	})

	t.Run("creates a labels list when there is none", func(t *testing.T) {
		t.Parallel()

		spec := map[string]interface{}{}

		result := appendActionLabel(spec, "chart:nginx-website")

		require.Equal(t, []interface{}{"chart:nginx-website"}, result.(map[string]interface{})["labels"])
	})

	t.Run("does not mutate the original spec", func(t *testing.T) {
		t.Parallel()

		original := map[string]interface{}{"labels": []interface{}{"dependencies"}}

		appendActionLabel(original, "chart:nginx-website")

		require.Equal(t, []interface{}{"dependencies"}, original["labels"])
	})

	t.Run("returns the spec unchanged when it is not a map", func(t *testing.T) {
		t.Parallel()

		result := appendActionLabel("not a map", "chart:nginx-website")
		require.Equal(t, "not a map", result)
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
