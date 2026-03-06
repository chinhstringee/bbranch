package dashboard

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/chinhstringee/buck/internal/bitbucket"
)

// PrintDashboard displays a colored summary of PRs across repos.
func PrintDashboard(results []RepoPRs) {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	bold := color.New(color.Bold)

	totalPRs := 0
	totalApproved := 0
	totalErrors := 0

	for _, r := range results {
		if r.Error != "" {
			totalErrors++
			fmt.Printf("  %s %-25s %s\n", red("✗"), r.RepoSlug, r.Error)
			continue
		}

		if len(r.PRs) == 0 {
			continue
		}

		bold.Printf("\n  %s\n", r.RepoSlug)
		for _, pr := range r.PRs {
			totalPRs++
			approvals := countApprovals(pr)
			reviewerCount := countReviewers(pr)
			approvalStr := formatApprovals(approvals, reviewerCount, green, yellow, red)
			statusIcon := prStatusIcon(approvals, reviewerCount, green, yellow)

			fmt.Printf("    %s #%-4d %-50s %s  %s\n",
				statusIcon,
				pr.ID,
				truncate(pr.Title, 50),
				cyan(pr.Author.DisplayName),
				approvalStr,
			)

			if approvals == reviewerCount && reviewerCount > 0 {
				totalApproved++
			}
		}
	}

	if totalPRs == 0 && totalErrors == 0 {
		fmt.Println("  No open pull requests found.")
		return
	}

	fmt.Printf("\n%s %s open, %s approved, %s errors\n",
		bold.Sprint("Summary:"),
		green(fmt.Sprintf("%d", totalPRs)),
		yellow(fmt.Sprintf("%d", totalApproved)),
		red(fmt.Sprintf("%d", totalErrors)),
	)
}

func countApprovals(pr bitbucket.PullRequest) int {
	count := 0
	for _, p := range pr.Participants {
		if p.Approved {
			count++
		}
	}
	return count
}

func countReviewers(pr bitbucket.PullRequest) int {
	count := 0
	for _, p := range pr.Participants {
		if p.Role == "REVIEWER" {
			count++
		}
	}
	return count
}

func formatApprovals(approvals, reviewers int, green, yellow, red func(a ...interface{}) string) string {
	s := fmt.Sprintf("%d/%d", approvals, reviewers)
	if reviewers == 0 {
		return yellow(s)
	}
	if approvals == reviewers {
		return green(s)
	}
	return yellow(s)
}

func prStatusIcon(approvals, reviewers int, green, yellow func(a ...interface{}) string) string {
	if reviewers > 0 && approvals == reviewers {
		return green("✓")
	}
	return yellow("●")
}

func truncate(s string, max int) string {
	s = strings.SplitN(s, "\n", 2)[0]
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-3]) + "..."
}
