package pullrequest

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/fatih/color"
	"github.com/chinhstringee/bbranch/internal/bitbucket"
)

// Result holds the outcome of a PR creation for one repo.
type Result struct {
	RepoSlug string
	Success  bool
	Error    string
	PRURL    string
	PRID     int
}

// PRCreator orchestrates parallel pull request creation across repos.
type PRCreator struct {
	client *bitbucket.Client
}

// NewPRCreator creates a new PR orchestrator.
func NewPRCreator(client *bitbucket.Client) *PRCreator {
	return &PRCreator{client: client}
}

// CreatePRs creates pull requests in multiple repos concurrently.
// If destination is empty, each repo's main branch is resolved via the API.
func (pc *PRCreator) CreatePRs(workspace string, repos []string, branchName, destination string) []Result {
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []Result
	)

	for _, repo := range repos {
		wg.Add(1)
		go func(repoSlug string) {
			defer wg.Done()

			dest := destination
			if dest == "" {
				repo, err := pc.client.GetRepository(workspace, repoSlug)
				if err != nil {
					mu.Lock()
					results = append(results, Result{
						RepoSlug: repoSlug,
						Error:    err.Error(),
					})
					mu.Unlock()
					return
				}
				if repo.MainBranch != nil {
					dest = repo.MainBranch.Name
				} else {
					dest = "main"
				}
			}

			// Build description from commits (fallback to static text on error)
			description := "Automated PR created by bbranch"
			commits, err := pc.client.ListCommits(workspace, repoSlug, branchName, dest)
			if err == nil && len(commits) > 0 {
				description = buildDescription(commits)
			}

			req := bitbucket.CreatePullRequestRequest{
				Title:       formatBranchTitle(branchName),
				Description: description,
				Source:      bitbucket.PRBranchRef{Branch: bitbucket.PRBranchName{Name: branchName}},
				Destination: bitbucket.PRBranchRef{Branch: bitbucket.PRBranchName{Name: dest}},
			}

			pr, err := pc.client.CreatePullRequest(workspace, repoSlug, req)

			result := Result{RepoSlug: repoSlug}
			if err != nil {
				result.Error = err.Error()
			} else {
				result.Success = true
				result.PRURL = pr.Links.HTML.Href
				result.PRID = pr.ID
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

// PrintResults displays a colored summary of PR creation results.
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
			fmt.Printf("  %s %-30s %s\n", green("✓"), r.RepoSlug, r.PRURL)
		} else {
			failed++
			// Indent multiline errors (e.g. permission scope details)
			lines := strings.Split(r.Error, "\n")
			fmt.Printf("  %s %-30s %s\n", red("✗"), r.RepoSlug, lines[0])
			for _, line := range lines[1:] {
				fmt.Printf("    %-30s %s\n", "", line)
			}
		}
	}

	fmt.Printf("\n%s %s succeeded, %s failed\n",
		bold("Summary:"),
		green(fmt.Sprintf("%d", succeeded)),
		red(fmt.Sprintf("%d", failed)),
	)
}

// ticketPattern matches JIRA-style ticket numbers like SPT-1298, PROJ-42.
var ticketPattern = regexp.MustCompile(`([A-Z]+)-(\d+)`)

// formatBranchTitle converts a branch name to a human-readable PR title.
// Example: "feature/SPT-1298-increase-api-limit" → "Feature/SPT-1298 increase api limit"
func formatBranchTitle(branchName string) string {
	// Temporarily protect ticket hyphens with a placeholder
	result := ticketPattern.ReplaceAllString(branchName, "${1}\x00${2}")
	// Replace remaining hyphens with spaces
	result = strings.ReplaceAll(result, "-", " ")
	// Restore ticket hyphens
	result = strings.ReplaceAll(result, "\x00", "-")
	// Capitalize first character
	runes := []rune(result)
	if len(runes) > 0 {
		runes[0] = unicode.ToUpper(runes[0])
	}
	return string(runes)
}

// buildDescription creates a markdown unordered list from commit messages.
func buildDescription(commits []bitbucket.Commit) string {
	lines := make([]string, 0, len(commits))
	for _, c := range commits {
		msg := strings.SplitN(c.Message, "\n", 2)[0] // first line only
		lines = append(lines, fmt.Sprintf("* %s", msg))
	}
	return strings.Join(lines, "\n")
}
