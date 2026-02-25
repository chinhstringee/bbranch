package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/chinhstringee/bbranch/internal/auth"
	"github.com/chinhstringee/bbranch/internal/bitbucket"
	"github.com/chinhstringee/bbranch/internal/config"
	"github.com/chinhstringee/bbranch/internal/creator"
)

var (
	flagGroup       string
	flagRepos       string
	flagFrom        string
	flagDryRun      bool
	flagInteractive bool
)

var createCmd = &cobra.Command{
	Use:   "create <branch-name>",
	Short: "Create a branch across multiple Bitbucket repos",
	Args:  cobra.ExactArgs(1),
	RunE:  runCreate,
}

func init() {
	createCmd.Flags().StringVarP(&flagGroup, "group", "g", "", "repo group from config")
	createCmd.Flags().StringVarP(&flagRepos, "repos", "r", "", "comma-separated repo slugs")
	createCmd.Flags().StringVarP(&flagFrom, "from", "f", "", "source branch (default: from config or master)")
	createCmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "preview actions without executing")
	createCmd.Flags().BoolVarP(&flagInteractive, "interactive", "i", false, "select repos interactively")

	rootCmd.AddCommand(createCmd)
}

func runCreate(cmd *cobra.Command, args []string) error {
	branchName := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.Workspace == "" {
		return fmt.Errorf("workspace not configured in .bbranch.yaml")
	}

	// Build token provider
	tokenFn := func() (string, error) {
		return auth.GetToken(cfg.OAuth.ClientID, cfg.OAuth.ClientSecret)
	}

	client := bitbucket.NewClient(tokenFn)

	// Resolve target repos
	repos, err := resolveRepos(cfg, client)
	if err != nil {
		return err
	}

	if len(repos) == 0 {
		return fmt.Errorf("no repositories selected")
	}

	// Resolve source branch
	sourceBranch := cfg.Defaults.SourceBranch
	if flagFrom != "" {
		sourceBranch = flagFrom
	}

	bold := color.New(color.Bold)

	// Dry run — show plan and exit
	if flagDryRun {
		bold.Printf("Dry run: would create branch %q from %q in:\n", branchName, sourceBranch)
		for _, r := range repos {
			fmt.Printf("  - %s\n", r)
		}
		return nil
	}

	bold.Printf("Creating branch %q from %q across %d repos...\n", branchName, sourceBranch, len(repos))

	bc := creator.NewBranchCreator(client)
	results := bc.CreateBranches(cfg.Workspace, repos, branchName, sourceBranch)
	creator.PrintResults(results)

	return nil
}

// resolveRepos determines which repos to target based on flags.
func resolveRepos(cfg *config.Config, client *bitbucket.Client) ([]string, error) {
	// Explicit --repos flag takes priority
	if flagRepos != "" {
		parts := strings.Split(flagRepos, ",")
		repos := make([]string, 0, len(parts))
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				repos = append(repos, trimmed)
			}
		}
		return repos, nil
	}

	// --group flag
	if flagGroup != "" {
		return cfg.GetReposForGroup(flagGroup)
	}

	// Default: interactive mode (core use case)
	return selectReposInteractively(cfg, client)
}

// selectReposInteractively fetches workspace repos and shows a multi-select.
func selectReposInteractively(cfg *config.Config, client *bitbucket.Client) ([]string, error) {
	fmt.Printf("Fetching repos from workspace %q...\n", cfg.Workspace)

	repos, err := client.ListRepositories(cfg.Workspace)
	if err != nil {
		return nil, fmt.Errorf("failed to list repos: %w", err)
	}

	if len(repos) == 0 {
		return nil, fmt.Errorf("no repositories found in workspace %q", cfg.Workspace)
	}

	// Build options for multi-select
	options := make([]huh.Option[string], 0, len(repos))
	for _, r := range repos {
		label := r.Slug
		if r.MainBranch != nil {
			label = fmt.Sprintf("%s (%s)", r.Slug, r.MainBranch.Name)
		}
		options = append(options, huh.NewOption(label, r.Slug))
	}

	var selected []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select repositories").
				Options(options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("selection cancelled")
	}

	return selected, nil
}
