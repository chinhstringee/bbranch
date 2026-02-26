package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/chinhstringee/bbranch/internal/bitbucket"
	"github.com/chinhstringee/bbranch/internal/config"
	"github.com/chinhstringee/bbranch/internal/pullrequest"
)

var (
	prFlagGroup       string
	prFlagRepos       string
	prFlagDryRun      bool
	prFlagDestination string
	prFlagInteractive bool
)

var prCmd = &cobra.Command{
	Use:   "pr <branch-name>",
	Short: "Create pull requests across multiple Bitbucket repos",
	Args:  cobra.ExactArgs(1),
	RunE:  runPR,
}

func init() {
	prCmd.Flags().StringVarP(&prFlagGroup, "group", "g", "", "repo group from config")
	prCmd.Flags().StringVarP(&prFlagRepos, "repos", "r", "", "comma-separated repo slugs")
	prCmd.Flags().BoolVar(&prFlagDryRun, "dry-run", false, "preview actions without executing")
	prCmd.Flags().StringVarP(&prFlagDestination, "destination", "d", "", "destination branch (default: repo's main branch)")
	prCmd.Flags().BoolVarP(&prFlagInteractive, "interactive", "i", false, "select repos interactively")

	rootCmd.AddCommand(prCmd)
}

func runPR(cmd *cobra.Command, args []string) error {
	branchName := args[0]

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

	repos, err := resolveTargetRepos(prFlagRepos, prFlagGroup, prFlagInteractive, cfg, client)
	if err != nil {
		return err
	}

	if len(repos) == 0 {
		return fmt.Errorf("no repositories selected")
	}

	bold := color.New(color.Bold)

	if prFlagDryRun {
		dest := prFlagDestination
		if dest == "" {
			dest = "(each repo's default branch)"
		}
		bold.Printf("Dry run: would create PRs from %q to %s in:\n", branchName, dest)
		for _, r := range repos {
			fmt.Printf("  - %s\n", r)
		}
		return nil
	}

	bold.Printf("Creating PRs from %q across %d repos...\n", branchName, len(repos))

	pc := pullrequest.NewPRCreator(client)
	results := pc.CreatePRs(cfg.Workspace, repos, branchName, prFlagDestination)
	pullrequest.PrintResults(results)

	return nil
}
