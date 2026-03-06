package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/chinhstringee/buck/internal/pullrequest"
)

var prDeclineFlagYes bool

var prDeclineCmd = &cobra.Command{
	Use:   "decline [branch-name]",
	Short: "Decline (close) pull requests by branch name across repos",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runPRDecline,
}

func init() {
	prDeclineCmd.Flags().BoolVarP(&prDeclineFlagYes, "yes", "y", false, "skip confirmation prompt")
	prCmd.AddCommand(prDeclineCmd)
}

func runPRDecline(cmd *cobra.Command, args []string) error {
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
		bold.Printf("Dry run: would decline PRs from branch %q in:\n", ctx.branchName)
		for _, r := range ctx.repos {
			fmt.Printf("  - %s/%s\n", ctx.workspace, r)
		}
		return nil
	}

	if !prDeclineFlagYes {
		bold.Printf("Will decline PRs from branch %q across %d repos\n", ctx.branchName, len(ctx.repos))
		if !confirmAction("Proceed?") {
			fmt.Println("Aborted.")
			return nil
		}
	}

	bold.Printf("Declining PRs from %q across %d repos...\n", ctx.branchName, len(ctx.repos))

	mgr := pullrequest.NewPRManager(ctx.client)
	results := mgr.DeclinePRs(ctx.workspace, ctx.repos, ctx.branchName)
	pullrequest.PrintActionResults("Decline", results)

	return nil
}
