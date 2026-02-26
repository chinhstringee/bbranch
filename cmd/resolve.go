package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/fatih/color"
	"github.com/chinhstringee/bbranch/internal/bitbucket"
	"github.com/chinhstringee/bbranch/internal/config"
	"github.com/chinhstringee/bbranch/internal/matcher"
)

// resolveTargetRepos determines which repos to target based on the given flags.
func resolveTargetRepos(reposFlag, groupFlag string, interactive bool, cfg *config.Config, client *bitbucket.Client) ([]string, error) {
	// --interactive flag forces interactive selection
	if interactive {
		return selectInteractively(cfg, client)
	}

	// Explicit --repos flag takes priority — fuzzy match against workspace repos
	if reposFlag != "" {
		return resolveWithFuzzyMatch(cfg, client, reposFlag)
	}

	// --group flag
	if groupFlag != "" {
		return cfg.GetReposForGroup(groupFlag)
	}

	// Default: interactive mode (core use case)
	return selectInteractively(cfg, client)
}

// selectInteractively fetches workspace repos and shows a multi-select.
func selectInteractively(cfg *config.Config, client *bitbucket.Client) ([]string, error) {
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
				Title("Select repositories (type to filter)").
				Options(options...).
				Filterable(true).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("selection cancelled")
	}

	return selected, nil
}

// resolveWithFuzzyMatch fetches workspace repos and fuzzy-matches patterns.
func resolveWithFuzzyMatch(cfg *config.Config, client *bitbucket.Client, reposFlag string) ([]string, error) {
	patterns := strings.Split(reposFlag, ",")

	fmt.Printf("Fetching repos from workspace %q...\n", cfg.Workspace)
	repos, err := client.ListRepositories(cfg.Workspace)
	if err != nil {
		return nil, fmt.Errorf("failed to list repos: %w", err)
	}

	slugs := make([]string, len(repos))
	for i, r := range repos {
		slugs[i] = r.Slug
	}

	result := matcher.Match(slugs, patterns)

	warn := color.New(color.FgYellow)
	bold := color.New(color.Bold)

	for _, p := range result.Unmatched {
		warn.Printf("Warning: no repos matched pattern %q\n", p)
	}

	if len(result.Matched) > 0 {
		bold.Println("Matched repos:")
		for _, s := range result.Matched {
			fmt.Printf("  - %s\n", s)
		}
	}

	return result.Matched, nil
}
