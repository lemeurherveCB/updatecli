package reports

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/require"
)

func TestUpdateManifestURL(t *testing.T) {
	tests := []struct {
		name          string
		remoteURL     string
		manifestPath  string // relative to the repository root
		expectedURL   bool
		expectedParts []string
	}{
		{
			name:          "github remote produces a permalink",
			remoteURL:     "https://github.com/updatecli/updatecli.git",
			manifestPath:  filepath.Join("updatecli.d", "updatecli.yaml"),
			expectedURL:   true,
			expectedParts: []string{"https://github.com/updatecli/updatecli/blob/", "/updatecli.d/updatecli.yaml"},
		},
		{
			name:         "unknown forge produces no permalink",
			remoteURL:    "https://git.example.com/owner/repo.git",
			manifestPath: "updatecli.yaml",
			expectedURL:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoDir := t.TempDir()

			repository, err := git.PlainInit(repoDir, false)
			require.NoError(t, err)

			_, err = repository.CreateRemote(&config.RemoteConfig{
				Name: "origin",
				URLs: []string{tt.remoteURL},
			})
			require.NoError(t, err)

			manifestFilename := filepath.Join(repoDir, tt.manifestPath)
			require.NoError(t, os.MkdirAll(filepath.Dir(manifestFilename), 0o755))
			require.NoError(t, os.WriteFile(manifestFilename, []byte("name: test\n"), 0o644))

			worktree, err := repository.Worktree()
			require.NoError(t, err)
			_, err = worktree.Add(tt.manifestPath)
			require.NoError(t, err)
			_, err = worktree.Commit("test commit", &git.CommitOptions{
				Author: &object.Signature{
					Name:  "test",
					Email: "test@example.com",
					When:  time.Now(),
				},
			})
			require.NoError(t, err)

			action := &Action{}
			action.UpdateManifestURL(manifestFilename)

			if !tt.expectedURL {
				require.Nil(t, action.ManifestURL)
				return
			}

			require.NotNil(t, action.ManifestURL)
			for _, part := range tt.expectedParts {
				require.Contains(t, action.ManifestURL.URL, part)
			}
			require.Equal(t, filepath.ToSlash(tt.manifestPath), action.ManifestURL.Name)
		})
	}
}
