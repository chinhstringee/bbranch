package config

import (
	"fmt"
	"os"
	"regexp"

	"github.com/spf13/viper"
)

// Config represents the .bbranch.yaml configuration.
type Config struct {
	Workspace string              `mapstructure:"workspace"`
	Auth      AuthConfig          `mapstructure:"auth"`
	OAuth     OAuthConfig         `mapstructure:"oauth"`
	ApiToken  ApiTokenConfig      `mapstructure:"api_token"`
	Groups    map[string][]string `mapstructure:"groups"`
	Defaults  Defaults            `mapstructure:"defaults"`
}

// AuthConfig holds the authentication method selection.
type AuthConfig struct {
	Method string `mapstructure:"method"` // "oauth" (default) or "api_token"
}

// OAuthConfig holds OAuth consumer credentials.
type OAuthConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
}

// ApiTokenConfig holds Bitbucket API token credentials.
type ApiTokenConfig struct {
	Email string `mapstructure:"email"`
	Token string `mapstructure:"token"`
}

// Defaults holds default branch creation settings.
type Defaults struct {
	SourceBranch string `mapstructure:"source_branch"`
	BranchPrefix string `mapstructure:"branch_prefix"`
}

// AuthMethod returns the configured auth method, defaulting to "api_token".
func (c *Config) AuthMethod() string {
	if c.Auth.Method == "" {
		return "api_token"
	}
	return c.Auth.Method
}

var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// expandEnvVars replaces ${VAR} patterns with environment variable values.
func expandEnvVars(val string) string {
	return envVarPattern.ReplaceAllStringFunc(val, func(match string) string {
		varName := envVarPattern.FindStringSubmatch(match)[1]
		return os.Getenv(varName)
	})
}

// Load reads the config from Viper and expands env vars.
func Load() (*Config, error) {
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Expand env vars in OAuth fields
	cfg.OAuth.ClientID = expandEnvVars(cfg.OAuth.ClientID)
	cfg.OAuth.ClientSecret = expandEnvVars(cfg.OAuth.ClientSecret)

	// Expand env vars in API Token fields
	cfg.ApiToken.Email = expandEnvVars(cfg.ApiToken.Email)
	cfg.ApiToken.Token = expandEnvVars(cfg.ApiToken.Token)

	// Set defaults
	if cfg.Defaults.SourceBranch == "" {
		cfg.Defaults.SourceBranch = "master"
	}

	return &cfg, nil
}

// GetReposForGroup returns repo slugs for a named group.
func (c *Config) GetReposForGroup(name string) ([]string, error) {
	repos, ok := c.Groups[name]
	if !ok {
		return nil, fmt.Errorf("group %q not found in config", name)
	}
	return repos, nil
}
