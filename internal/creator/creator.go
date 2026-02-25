package creator

import (
	"fmt"
	"sort"
	"sync"

	"github.com/fatih/color"
	"github.com/stringee/git-branch-creator/internal/bitbucket"
)

// Result holds the outcome of a branch creation for one repo.
type Result struct {
	RepoSlug   string
	Success    bool
	Error      string
	CommitHash string
}

// BranchCreator orchestrates parallel branch creation across repos.
type BranchCreator struct {
	client *bitbucket.Client
}

// NewBranchCreator creates a new orchestrator.
func NewBranchCreator(client *bitbucket.Client) *BranchCreator {
	return &BranchCreator{client: client}
}

// CreateBranches creates a branch in multiple repos concurrently.
func (bc *BranchCreator) CreateBranches(workspace string, repos []string, branchName, sourceBranch string) []Result {
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []Result
	)

	for _, repo := range repos {
		wg.Add(1)
		go func(repoSlug string) {
			defer wg.Done()

			branch, err := bc.client.CreateBranch(workspace, repoSlug, branchName, sourceBranch)

			result := Result{RepoSlug: repoSlug}
			if err != nil {
				result.Success = false
				result.Error = err.Error()
			} else {
				result.Success = true
				// Show short hash (first 7 chars)
				if len(branch.Target.Hash) > 7 {
					result.CommitHash = branch.Target.Hash[:7]
				} else {
					result.CommitHash = branch.Target.Hash
				}
			}

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(repo)
	}

	wg.Wait()

	// Sort by repo slug for consistent output
	sort.Slice(results, func(i, j int) bool {
		return results[i].RepoSlug < results[j].RepoSlug
	})

	return results
}

// PrintResults displays a colored summary table of results.
func PrintResults(results []Result) {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	succeeded := 0
	failed := 0

	fmt.Println()
	for _, r := range results {
		if r.Success {
			succeeded++
			fmt.Printf("  %s %-30s created (%s)\n", green("✓"), r.RepoSlug, r.CommitHash)
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
