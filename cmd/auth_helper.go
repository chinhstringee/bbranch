package cmd

import (
	"fmt"

	"github.com/chinhstringee/bbranch/internal/auth"
	"github.com/chinhstringee/bbranch/internal/bitbucket"
	"github.com/chinhstringee/bbranch/internal/config"
)

// buildAuthApplier creates the appropriate AuthApplier based on config.
func buildAuthApplier(cfg *config.Config) (bitbucket.AuthApplier, error) {
	switch cfg.AuthMethod() {
	case "api_token":
		if cfg.ApiToken.Email == "" || cfg.ApiToken.Token == "" {
			return nil, fmt.Errorf("api_token credentials not configured.\nSet them in .bbranch.yaml:\n  api_token:\n    email: your-email@example.com\n    token: your-api-token")
		}
		return bitbucket.BasicAuth(cfg.ApiToken.Email, cfg.ApiToken.Token), nil

	case "oauth":
		if cfg.OAuth.ClientID == "" || cfg.OAuth.ClientSecret == "" {
			return nil, fmt.Errorf("OAuth credentials not configured.\nSet them in .bbranch.yaml or via environment variables:\n  BITBUCKET_OAUTH_CLIENT_ID\n  BITBUCKET_OAUTH_CLIENT_SECRET")
		}
		tokenFn := func() (string, error) {
			return auth.GetToken(cfg.OAuth.ClientID, cfg.OAuth.ClientSecret)
		}
		return bitbucket.BearerAuth(tokenFn), nil

	default:
		return nil, fmt.Errorf("unknown auth method %q. Use \"oauth\" or \"api_token\"", cfg.AuthMethod())
	}
}
