package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/chinhstringee/buck/internal/bitbucket"
	"github.com/chinhstringee/buck/internal/pullrequest"
)

var prReviewersFlagAdd string

var prReviewersCmd = &cobra.Command{
	Use:   "reviewers [branch-name]",
	Short: "Add reviewers to pull requests by branch name across repos",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runPRReviewers,
}

func init() {
	prReviewersCmd.Flags().StringVar(&prReviewersFlagAdd, "add", "", "comma-separated account IDs or UUIDs to add as reviewers")
	prCmd.AddCommand(prReviewersCmd)
}

func runPRReviewers(cmd *cobra.Command, args []string) error {
	if prReviewersFlagAdd == "" {
		return fmt.Errorf("--add flag is required (comma-separated account IDs or UUIDs)")
	}

	var branchArg string
	if len(args) > 0 {
		branchArg = args[0]
	}

	ctx, err := resolvePRContext(branchArg)
	if err != nil {
		return err
	}

	// Parse reviewer identifiers
	parts := strings.Split(prReviewersFlagAdd, ",")
	reviewers := make([]bitbucket.PRReviewer, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// UUIDs are wrapped in {}, account IDs are not
		if strings.HasPrefix(p, "{") {
			reviewers = append(reviewers, bitbucket.PRReviewer{UUID: p})
		} else {
			reviewers = append(reviewers, bitbucket.PRReviewer{AccountID: p})
		}
	}

	if len(reviewers) == 0 {
		return fmt.Errorf("no valid reviewer identifiers provided")
	}

	bold := color.New(color.Bold)

	if prFlagDryRun {
		bold.Printf("Dry run: would add %d reviewers to PRs from branch %q in:\n", len(reviewers), ctx.branchName)
		for _, r := range ctx.repos {
			fmt.Printf("  - %s/%s\n", ctx.workspace, r)
		}
		return nil
	}

	bold.Printf("Adding %d reviewers to PRs from %q across %d repos...\n", len(reviewers), ctx.branchName, len(ctx.repos))

	mgr := pullrequest.NewPRManager(ctx.client)
	results := mgr.AddReviewers(ctx.workspace, ctx.repos, ctx.branchName, reviewers)
	pullrequest.PrintActionResults("Updated reviewers on", results)

	return nil
}
