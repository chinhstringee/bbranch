package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/chinhstringee/buck/internal/pullrequest"
)

var prApproveCmd = &cobra.Command{
	Use:   "approve [branch-name]",
	Short: "Approve pull requests by branch name across repos",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runPRApprove,
}

func init() {
	prCmd.AddCommand(prApproveCmd)
}

func runPRApprove(cmd *cobra.Command, args []string) error {
	var branchArg string
	if len(args) > 0 {
		branchArg = args[0]
	}

	ctx, err := resolvePRContext(branchArg)
	if err != nil {
		return err
	}

	bold := color.New(color.Bold)

	if prFlagDryRun {
		bold.Printf("Dry run: would approve PRs from branch %q in:\n", ctx.branchName)
		for _, r := range ctx.repos {
			fmt.Printf("  - %s/%s\n", ctx.workspace, r)
		}
		return nil
	}

	bold.Printf("Approving PRs from %q across %d repos...\n", ctx.branchName, len(ctx.repos))

	mgr := pullrequest.NewPRManager(ctx.client)
	results := mgr.ApprovePRs(ctx.workspace, ctx.repos, ctx.branchName)
	pullrequest.PrintActionResults("Approve", results)

	return nil
}
