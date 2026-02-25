package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
)

func resetViper() {
	viper.Reset()
}

func TestExpandEnvVars(t *testing.T) {
	t.Setenv("MY_VAR", "hello")
	t.Setenv("OTHER_VAR", "world")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no placeholder", "plain-value", "plain-value"},
		{"single placeholder", "${MY_VAR}", "hello"},
		{"multiple placeholders", "${MY_VAR}-${OTHER_VAR}", "hello-world"},
		{"placeholder mid-string", "prefix-${MY_VAR}-suffix", "prefix-hello-suffix"},
		{"unset var expands to empty", "${UNSET_ENV_12345}", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := expandEnvVars(tc.input)
			if got != tc.want {
				t.Errorf("expandEnvVars(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestLoad_DefaultSourceBranch(t *testing.T) {
	resetViper()

	// Load with no source_branch set — should default to "master"
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Defaults.SourceBranch != "master" {
		t.Errorf("default SourceBranch = %q, want %q", cfg.Defaults.SourceBranch, "master")
	}
}

func TestLoad_KeepsExplicitSourceBranch(t *testing.T) {
	resetViper()
	viper.Set("defaults.source_branch", "main")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Defaults.SourceBranch != "main" {
		t.Errorf("SourceBranch = %q, want %q", cfg.Defaults.SourceBranch, "main")
	}
}

func TestLoad_EnvVarExpansionInOAuth(t *testing.T) {
	resetViper()

	os.Setenv("BB_CLIENT_ID", "my-client-id")
	os.Setenv("BB_CLIENT_SECRET", "my-secret")
	defer os.Unsetenv("BB_CLIENT_ID")
	defer os.Unsetenv("BB_CLIENT_SECRET")

	viper.Set("oauth.client_id", "${BB_CLIENT_ID}")
	viper.Set("oauth.client_secret", "${BB_CLIENT_SECRET}")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.OAuth.ClientID != "my-client-id" {
		t.Errorf("ClientID = %q, want %q", cfg.OAuth.ClientID, "my-client-id")
	}
	if cfg.OAuth.ClientSecret != "my-secret" {
		t.Errorf("ClientSecret = %q, want %q", cfg.OAuth.ClientSecret, "my-secret")
	}
}

func TestLoad_WorkspaceAndGroups(t *testing.T) {
	resetViper()
	viper.Set("workspace", "myworkspace")
	viper.Set("groups", map[string]interface{}{
		"backend": []interface{}{"repo-a", "repo-b"},
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Workspace != "myworkspace" {
		t.Errorf("Workspace = %q, want %q", cfg.Workspace, "myworkspace")
	}
	if len(cfg.Groups["backend"]) != 2 {
		t.Errorf("Groups[backend] len = %d, want 2", len(cfg.Groups["backend"]))
	}
}

func TestGetReposForGroup_Found(t *testing.T) {
	cfg := &Config{
		Groups: map[string][]string{
			"backend": {"repo-a", "repo-b", "repo-c"},
		},
	}

	repos, err := cfg.GetReposForGroup("backend")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 3 {
		t.Errorf("len(repos) = %d, want 3", len(repos))
	}
}

func TestGetReposForGroup_NotFound(t *testing.T) {
	cfg := &Config{
		Groups: map[string][]string{
			"backend": {"repo-a"},
		},
	}

	_, err := cfg.GetReposForGroup("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing group, got nil")
	}
}

func TestGetReposForGroup_EmptyGroups(t *testing.T) {
	cfg := &Config{
		Groups: map[string][]string{},
	}

	_, err := cfg.GetReposForGroup("anything")
	if err == nil {
		t.Fatal("expected error for empty groups, got nil")
	}
}
