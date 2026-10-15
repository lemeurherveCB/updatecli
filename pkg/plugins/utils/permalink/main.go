// Package permalink builds permalinks to a file at a specific commit for the git forges
// Updatecli knows how to format a blob URL for.
package permalink

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// scpLikeURL matches the scp-like git remote syntax, e.g. "git@github.com:owner/repo.git"
var scpLikeURL = regexp.MustCompile(`^(?:[^@/]+@)?([^:/]+):(.+)$`)

// New builds a permalink to relativePath at commitHash in the git repository referenced by
// remoteURL. It returns ok=false when the forge hosting remoteURL couldn't be identified.
func New(remoteURL, commitHash, relativePath string) (permalink string, ok bool) {
	host, repositoryPath, err := parseRemoteURL(remoteURL)
	if err != nil {
		return "", false
	}

	escapedRelativePath := (&url.URL{Path: relativePath}).EscapedPath()
	lowerHost := strings.ToLower(host)

	switch {
	case strings.Contains(lowerHost, "github"):
		repo := cleanRepositoryPath(repositoryPath)
		return fmt.Sprintf("https://%s/%s/blob/%s/%s", host, repo, commitHash, escapedRelativePath), true

	case strings.Contains(lowerHost, "gitlab"):
		repo := cleanRepositoryPath(repositoryPath)
		return fmt.Sprintf("https://%s/%s/-/blob/%s/%s", host, repo, commitHash, escapedRelativePath), true

	case strings.EqualFold(host, "bitbucket.org"):
		repo := cleanRepositoryPath(repositoryPath)
		return fmt.Sprintf("https://%s/%s/src/%s/%s", host, repo, commitHash, escapedRelativePath), true

	case isAzureDevOpsHost(lowerHost):
		org, project, repo, ok := parseAzureDevOpsPath(lowerHost, repositoryPath)
		if !ok {
			return "", false
		}
		escapedQueryPath := url.QueryEscape("/" + strings.TrimPrefix(relativePath, "/"))
		return fmt.Sprintf("https://dev.azure.com/%s/%s/_git/%s?path=%s&version=GC%s",
			url.PathEscape(org), url.PathEscape(project), url.PathEscape(repo), escapedQueryPath, commitHash), true

	default:
		// Bitbucket Server / Data Center clone URLs are identified by their "/scm/" path
		// segment regardless of hostname, since they're almost always self-hosted.
		projectOrUser, repo, ok := parseBitbucketServerPath(repositoryPath)
		if !ok {
			return "", false
		}
		if username, ok := strings.CutPrefix(projectOrUser, "~"); ok {
			return fmt.Sprintf("https://%s/users/%s/repos/%s/browse/%s?at=%s", host, username, repo, escapedRelativePath, commitHash), true
		}
		return fmt.Sprintf("https://%s/projects/%s/repos/%s/browse/%s?at=%s", host, projectOrUser, repo, escapedRelativePath, commitHash), true
	}
}

// cleanRepositoryPath trims the leading/trailing slashes and the ".git" suffix off a
// repository path extracted from a remote URL.
func cleanRepositoryPath(repositoryPath string) string {
	return strings.Trim(strings.TrimSuffix(repositoryPath, ".git"), "/")
}

// isAzureDevOpsHost reports whether host is a known Azure DevOps/Visual Studio Team
// Services hostname, covering both the current (dev.azure.com) and legacy
// (*.visualstudio.com) domains, as well as their SSH counterparts.
func isAzureDevOpsHost(lowerHost string) bool {
	return strings.Contains(lowerHost, "dev.azure.com") || strings.Contains(lowerHost, "visualstudio.com")
}

// parseAzureDevOpsPath extracts the organization, project and repository from the path of
// an Azure DevOps git remote URL, across its HTTPS and SSH, current and legacy forms:
//   - https://dev.azure.com/{org}/{project}/_git/{repo}
//   - git@ssh.dev.azure.com:v3/{org}/{project}/{repo}
//   - https://{org}.visualstudio.com/{project}/_git/{repo} (optionally under /DefaultCollection/)
//   - {org}@vs-ssh.visualstudio.com:v3/{org}/{project}/{repo}
func parseAzureDevOpsPath(lowerHost, rawPath string) (org, project, repo string, ok bool) {
	cleanPath := strings.Trim(rawPath, "/")

	switch {
	case strings.Contains(lowerHost, "ssh.dev.azure.com"), strings.Contains(lowerHost, "vs-ssh.visualstudio.com"):
		parts := strings.Split(strings.TrimPrefix(cleanPath, "v3/"), "/")
		if len(parts) != 3 {
			return "", "", "", false
		}
		return parts[0], parts[1], strings.TrimSuffix(parts[2], ".git"), true

	case strings.Contains(lowerHost, "dev.azure.com"):
		parts := strings.SplitN(cleanPath, "/_git/", 2)
		if len(parts) != 2 {
			return "", "", "", false
		}
		orgProject := strings.Split(parts[0], "/")
		if len(orgProject) != 2 {
			return "", "", "", false
		}
		return orgProject[0], orgProject[1], strings.TrimSuffix(parts[1], ".git"), true

	case strings.HasSuffix(lowerHost, ".visualstudio.com"):
		org := strings.TrimSuffix(lowerHost, ".visualstudio.com")
		cleanPath = strings.TrimPrefix(cleanPath, "DefaultCollection/")
		parts := strings.SplitN(cleanPath, "/_git/", 2)
		if len(parts) != 2 {
			return "", "", "", false
		}
		return org, parts[0], strings.TrimSuffix(parts[1], ".git"), true
	}

	return "", "", "", false
}

// parseBitbucketServerPath extracts the project key (or "~username" for a personal
// repository) and the repository name from the path of a Bitbucket Server/Data Center
// HTTPS git remote URL, e.g. "/scm/PROJECT/repo.git" or "/scm/~username/repo.git".
func parseBitbucketServerPath(rawPath string) (projectOrUser, repo string, ok bool) {
	cleanPath := strings.Trim(rawPath, "/")

	const scmPrefix = "scm/"
	if !strings.HasPrefix(cleanPath, scmPrefix) {
		return "", "", false
	}

	parts := strings.SplitN(strings.TrimPrefix(cleanPath, scmPrefix), "/", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	return parts[0], strings.TrimSuffix(parts[1], ".git"), true
}

// parseRemoteURL extracts the host and the repository path from a git remote URL, supporting
// both URL syntax (https://, ssh://) and the scp-like syntax (e.g. git@host:owner/repo.git).
func parseRemoteURL(remoteURL string) (host, repositoryPath string, err error) {
	if u, parseErr := url.Parse(remoteURL); parseErr == nil && u.Host != "" {
		return u.Hostname(), u.Path, nil
	}

	if matches := scpLikeURL.FindStringSubmatch(remoteURL); matches != nil {
		return matches[1], matches[2], nil
	}

	return "", "", fmt.Errorf("unrecognized git remote URL format: %q", remoteURL)
}
