package permalink

import "testing"

func TestNew(t *testing.T) {
	tests := []struct {
		name         string
		remoteURL    string
		commitHash   string
		relativePath string
		wantURL      string
		wantOk       bool
	}{
		{
			name:         "github https",
			remoteURL:    "https://github.com/updatecli/updatecli.git",
			commitHash:   "b6ec28288d59f66d71ae63bb00da28bf4908ba0c",
			relativePath: "updatecli/updatecli.d/updatecli.yaml",
			wantURL:      "https://github.com/updatecli/updatecli/blob/b6ec28288d59f66d71ae63bb00da28bf4908ba0c/updatecli/updatecli.d/updatecli.yaml",
			wantOk:       true,
		},
		{
			name:         "github scp-like ssh",
			remoteURL:    "git@github.com:updatecli/updatecli.git",
			commitHash:   "abc123",
			relativePath: "updatecli.yaml",
			wantURL:      "https://github.com/updatecli/updatecli/blob/abc123/updatecli.yaml",
			wantOk:       true,
		},
		{
			name:         "github ssh scheme with port",
			remoteURL:    "ssh://git@github.com:22/updatecli/updatecli.git",
			commitHash:   "abc123",
			relativePath: "updatecli.yaml",
			wantURL:      "https://github.com/updatecli/updatecli/blob/abc123/updatecli.yaml",
			wantOk:       true,
		},
		{
			name:         "github enterprise https with credentials",
			remoteURL:    "https://user:token@github.mycompany.com/team/repo.git",
			commitHash:   "abc123",
			relativePath: "a/b.yaml",
			wantURL:      "https://github.mycompany.com/team/repo/blob/abc123/a/b.yaml",
			wantOk:       true,
		},
		{
			name:         "gitlab https",
			remoteURL:    "https://gitlab.com/group/subgroup/repo.git",
			commitHash:   "abc123",
			relativePath: "updatecli.yaml",
			wantURL:      "https://gitlab.com/group/subgroup/repo/-/blob/abc123/updatecli.yaml",
			wantOk:       true,
		},
		{
			name:         "gitlab scp-like ssh",
			remoteURL:    "git@gitlab.com:group/repo.git",
			commitHash:   "abc123",
			relativePath: "updatecli.yaml",
			wantURL:      "https://gitlab.com/group/repo/-/blob/abc123/updatecli.yaml",
			wantOk:       true,
		},
		{
			name:         "bitbucket cloud https",
			remoteURL:    "https://bitbucket.org/owner/repo.git",
			commitHash:   "abc123",
			relativePath: "updatecli.yaml",
			wantURL:      "https://bitbucket.org/owner/repo/src/abc123/updatecli.yaml",
			wantOk:       true,
		},
		{
			name:         "azure devops https",
			remoteURL:    "https://dev.azure.com/myorg/myproject/_git/myrepo",
			commitHash:   "abc123",
			relativePath: "updatecli.yaml",
			wantURL:      "https://dev.azure.com/myorg/myproject/_git/myrepo?path=%2Fupdatecli.yaml&version=GCabc123",
			wantOk:       true,
		},
		{
			name:         "azure devops ssh",
			remoteURL:    "git@ssh.dev.azure.com:v3/myorg/myproject/myrepo",
			commitHash:   "abc123",
			relativePath: "a/b.yaml",
			wantURL:      "https://dev.azure.com/myorg/myproject/_git/myrepo?path=%2Fa%2Fb.yaml&version=GCabc123",
			wantOk:       true,
		},
		{
			name:         "azure devops legacy visualstudio https",
			remoteURL:    "https://myorg.visualstudio.com/myproject/_git/myrepo",
			commitHash:   "abc123",
			relativePath: "updatecli.yaml",
			wantURL:      "https://dev.azure.com/myorg/myproject/_git/myrepo?path=%2Fupdatecli.yaml&version=GCabc123",
			wantOk:       true,
		},
		{
			name:         "azure devops legacy visualstudio ssh",
			remoteURL:    "myorg@vs-ssh.visualstudio.com:v3/myorg/myproject/myrepo",
			commitHash:   "abc123",
			relativePath: "updatecli.yaml",
			wantURL:      "https://dev.azure.com/myorg/myproject/_git/myrepo?path=%2Fupdatecli.yaml&version=GCabc123",
			wantOk:       true,
		},
		{
			name:         "bitbucket server project repo https",
			remoteURL:    "https://bitbucket.mycompany.com/scm/PROJ/myrepo.git",
			commitHash:   "abc123",
			relativePath: "updatecli.yaml",
			wantURL:      "https://bitbucket.mycompany.com/projects/PROJ/repos/myrepo/browse/updatecli.yaml?at=abc123",
			wantOk:       true,
		},
		{
			name:         "bitbucket server personal repo https",
			remoteURL:    "https://bitbucket.mycompany.com/scm/~jdoe/myrepo.git",
			commitHash:   "abc123",
			relativePath: "updatecli.yaml",
			wantURL:      "https://bitbucket.mycompany.com/users/jdoe/repos/myrepo/browse/updatecli.yaml?at=abc123",
			wantOk:       true,
		},
		{
			name:         "unknown forge",
			remoteURL:    "https://git.example.com/owner/repo.git",
			commitHash:   "abc123",
			relativePath: "updatecli.yaml",
			wantOk:       false,
		},
		{
			name:         "unparseable remote",
			remoteURL:    "not a url",
			commitHash:   "abc123",
			relativePath: "updatecli.yaml",
			wantOk:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, gotOk := New(tt.remoteURL, tt.commitHash, tt.relativePath)
			if gotOk != tt.wantOk {
				t.Fatalf("New() ok = %v, want %v", gotOk, tt.wantOk)
			}
			if gotOk && gotURL != tt.wantURL {
				t.Fatalf("New() url = %q, want %q", gotURL, tt.wantURL)
			}
		})
	}
}
