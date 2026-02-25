package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/chinhstringee/bbranch/internal/bitbucket"
	"github.com/chinhstringee/bbranch/internal/config"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List repositories in the configured workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if cfg.Workspace == "" {
			return fmt.Errorf("workspace not configured in .bbranch.yaml")
		}

		authApplier, err := buildAuthApplier(cfg)
		if err != nil {
			return err
		}

		client := bitbucket.NewClient(authApplier)

		fmt.Printf("Fetching repos from workspace %q...\n\n", cfg.Workspace)

		repos, err := client.ListRepositories(cfg.Workspace)
		if err != nil {
			return err
		}

		bold := color.New(color.Bold)
		dim := color.New(color.Faint)

		bold.Printf("%-30s %-15s %s\n", "REPO", "DEFAULT BRANCH", "UPDATED")
		fmt.Println("─────────────────────────────────────────────────────────────")

		for _, r := range repos {
			branch := "n/a"
			if r.MainBranch != nil {
				branch = r.MainBranch.Name
			}

			// Truncate updated_on to date only
			updated := r.UpdatedOn
			if len(updated) > 10 {
				updated = updated[:10]
			}

			fmt.Printf("%-30s %-15s %s\n", r.Slug, branch, dim.Sprint(updated))
		}

		fmt.Printf("\nTotal: %d repositories\n", len(repos))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
