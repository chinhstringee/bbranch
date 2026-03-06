package gitutil

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var (
	sshRemoteRe   = regexp.MustCompile(`git@bitbucket\.org:([^/]+)/(.+?)(?:\.git)?$`)
	httpsRemoteRe = regexp.MustCompile(`https?://(?:[^@]+@)?bitbucket\.org/([^/]+)/(.+?)(?:\.git)?$`)
)

// CurrentBranch returns the current git branch name.
// Returns an error if not in a git repo or in detached HEAD state.
func CurrentBranch() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository or git not installed")
	}

	branch := strings.TrimSpace(string(out))
	if branch == "HEAD" {
		return "", fmt.Errorf("detached HEAD state — checkout a branch first")
	}

	return branch, nil
}

// ParseBitbucketRemote parses the origin remote URL and extracts workspace and repo slug.
func ParseBitbucketRemote() (workspace, repoSlug string, err error) {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return "", "", fmt.Errorf("no 'origin' remote found")
	}

	url := strings.TrimSpace(string(out))
	return ParseRemoteURL(url)
}

// ParseRemoteURL extracts workspace and repo slug from a Bitbucket remote URL.
// Supports SSH and HTTPS formats.
func ParseRemoteURL(url string) (workspace, repoSlug string, err error) {
	if m := sshRemoteRe.FindStringSubmatch(url); m != nil {
		return m[1], m[2], nil
	}

	if m := httpsRemoteRe.FindStringSubmatch(url); m != nil {
		return m[1], m[2], nil
	}

	return "", "", fmt.Errorf("not a Bitbucket remote URL: %s", url)
}
