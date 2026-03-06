package gitutil

import (
	"os"
	"os/exec"
	"testing"
)

func TestParseRemoteURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantWS    string
		wantRepo  string
		wantError bool
	}{
		{
			name:     "SSH format",
			url:      "git@bitbucket.org:myworkspace/my-repo.git",
			wantWS:   "myworkspace",
			wantRepo: "my-repo",
		},
		{
			name:     "SSH without .git suffix",
			url:      "git@bitbucket.org:myworkspace/my-repo",
			wantWS:   "myworkspace",
			wantRepo: "my-repo",
		},
		{
			name:     "HTTPS format",
			url:      "https://bitbucket.org/myworkspace/my-repo.git",
			wantWS:   "myworkspace",
			wantRepo: "my-repo",
		},
		{
			name:     "HTTPS without .git suffix",
			url:      "https://bitbucket.org/myworkspace/my-repo",
			wantWS:   "myworkspace",
			wantRepo: "my-repo",
		},
		{
			name:     "HTTPS with user",
			url:      "https://user@bitbucket.org/myworkspace/my-repo.git",
			wantWS:   "myworkspace",
			wantRepo: "my-repo",
		},
		{
			name:      "GitHub URL",
			url:       "git@github.com:user/repo.git",
			wantError: true,
		},
		{
			name:      "non-bitbucket HTTPS",
			url:       "https://gitlab.com/user/repo.git",
			wantError: true,
		},
		{
			name:      "empty URL",
			url:       "",
			wantError: true,
		},
		{
			name:      "malformed URL",
			url:       "not-a-url",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws, repo, err := ParseRemoteURL(tt.url)
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got ws=%q repo=%q", ws, repo)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ws != tt.wantWS {
				t.Errorf("workspace: got %q, want %q", ws, tt.wantWS)
			}
			if repo != tt.wantRepo {
				t.Errorf("repoSlug: got %q, want %q", repo, tt.wantRepo)
			}
		})
	}
}

func TestCurrentBranch(t *testing.T) {
	// Create a temp git repo
	dir := t.TempDir()
	oldDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(oldDir) })

	os.Chdir(dir)

	run := func(name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
		}
	}

	run("git", "init")
	run("git", "checkout", "-b", "test-branch")
	// Need at least one commit for rev-parse to work
	run("git", "commit", "--allow-empty", "-m", "init")

	branch, err := CurrentBranch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "test-branch" {
		t.Errorf("got %q, want %q", branch, "test-branch")
	}
}

func TestCurrentBranch_NotGitRepo(t *testing.T) {
	dir := t.TempDir()
	oldDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(oldDir) })

	os.Chdir(dir)

	_, err := CurrentBranch()
	if err == nil {
		t.Error("expected error for non-git directory")
	}
}
