package pullrequest

import (
	"fmt"
	"sort"
	"sync"

	"github.com/chinhstringee/buck/internal/bitbucket"
)

// PRManager orchestrates PR operations (merge, decline, approve, reviewers) across repos.
type PRManager struct {
	client *bitbucket.Client
}

// NewPRManager creates a new PR manager.
func NewPRManager(client *bitbucket.Client) *PRManager {
	return &PRManager{client: client}
}

// MergePRs merges PRs by branch name across repos concurrently.
func (m *PRManager) MergePRs(workspace string, repos []string, branchName string, req bitbucket.MergePRRequest) []Result {
	return m.forEachRepo(workspace, repos, branchName, func(ws, slug string, pr *bitbucket.PullRequest) error {
		return m.client.MergePR(ws, slug, pr.ID, req)
	})
}

// DeclinePRs declines PRs by branch name across repos concurrently.
func (m *PRManager) DeclinePRs(workspace string, repos []string, branchName string) []Result {
	return m.forEachRepo(workspace, repos, branchName, func(ws, slug string, pr *bitbucket.PullRequest) error {
		return m.client.DeclinePR(ws, slug, pr.ID)
	})
}

// ApprovePRs approves PRs by branch name across repos concurrently.
func (m *PRManager) ApprovePRs(workspace string, repos []string, branchName string) []Result {
	return m.forEachRepo(workspace, repos, branchName, func(ws, slug string, pr *bitbucket.PullRequest) error {
		return m.client.ApprovePR(ws, slug, pr.ID)
	})
}

// AddReviewers adds reviewers to PRs by branch name across repos concurrently.
func (m *PRManager) AddReviewers(workspace string, repos []string, branchName string, reviewers []bitbucket.PRReviewer) []Result {
	return m.forEachRepo(workspace, repos, branchName, func(ws, slug string, pr *bitbucket.PullRequest) error {
		// Merge existing reviewers with new ones
		existing := make(map[string]bool)
		allReviewers := make([]bitbucket.PRReviewer, 0, len(pr.Reviewers)+len(reviewers))
		for _, r := range pr.Reviewers {
			key := r.UUID + r.AccountID
			if !existing[key] {
				existing[key] = true
				allReviewers = append(allReviewers, r)
			}
		}
		for _, r := range reviewers {
			key := r.UUID + r.AccountID
			if !existing[key] {
				existing[key] = true
				allReviewers = append(allReviewers, r)
			}
		}
		_, err := m.client.UpdatePR(ws, slug, pr.ID, bitbucket.PRUpdateRequest{Reviewers: allReviewers})
		return err
	})
}

// forEachRepo finds a PR by branch and performs an action, concurrently across repos.
func (m *PRManager) forEachRepo(workspace string, repos []string, branchName string, action func(ws, slug string, pr *bitbucket.PullRequest) error) []Result {
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []Result
	)

	for _, repo := range repos {
		wg.Add(1)
		go func(repoSlug string) {
			defer wg.Done()

			result := Result{RepoSlug: repoSlug}

			pr, err := m.client.FindPRByBranch(workspace, repoSlug, branchName, "OPEN")
			if err != nil {
				result.Error = err.Error()
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
				return
			}

			result.PRID = pr.ID
			result.PRURL = pr.Links.HTML.Href

			if err := action(workspace, repoSlug, pr); err != nil {
				result.Error = err.Error()
			} else {
				result.Success = true
			}

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(repo)
	}

	wg.Wait()

	sort.Slice(results, func(i, j int) bool {
		return results[i].RepoSlug < results[j].RepoSlug
	})

	return results
}

// PrintActionResults displays results for merge/decline/approve operations.
func PrintActionResults(action string, results []Result) {
	fmt.Println()
	printResultLines(results, func(r Result) string {
		return fmt.Sprintf("%sd PR #%d", action, r.PRID)
	})
}

// printResultLines is the shared result printer with a custom success message formatter.
func printResultLines(results []Result, successMsg func(Result) string) {
	green := colorGreen()
	red := colorRed()
	bold := colorBold()

	succeeded := 0
	failed := 0

	for _, r := range results {
		if r.Success {
			succeeded++
			fmt.Printf("  %s %-30s %s\n", green("✓"), r.RepoSlug, successMsg(r))
		} else {
			failed++
			fmt.Printf("  %s %-30s %s\n", red("✗"), r.RepoSlug, r.Error)
		}
	}

	fmt.Printf("\n%s %s succeeded, %s failed\n",
		bold("Summary:"),
		green(fmt.Sprintf("%d", succeeded)),
		red(fmt.Sprintf("%d", failed)),
	)
}
