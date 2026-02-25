package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/chinhstringee/bbranch/internal/auth"
	"github.com/chinhstringee/bbranch/internal/config"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Bitbucket via OAuth 2.0",
	Long:  "Opens your browser to authorize bbranch with your Bitbucket account.\nNot needed when using app_password auth method.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if cfg.AuthMethod() == "api_token" {
			return fmt.Errorf("login is not needed for API token auth.\nRun 'bbranch setup' to configure your credentials")
		}

		if cfg.OAuth.ClientID == "" || cfg.OAuth.ClientSecret == "" {
			return fmt.Errorf("OAuth credentials not configured.\nSet them in .bbranch.yaml or via environment variables:\n  BITBUCKET_OAUTH_CLIENT_ID\n  BITBUCKET_OAUTH_CLIENT_SECRET")
		}

		return auth.Login(cfg.OAuth.ClientID, cfg.OAuth.ClientSecret)
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
