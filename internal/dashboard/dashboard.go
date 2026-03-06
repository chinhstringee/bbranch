package dashboard

import (
	"sort"
	"sync"

	"github.com/chinhstringee/buck/internal/bitbucket"
)

// PRFilters controls which PRs to include.
type PRFilters struct {
	Author string // filter by author nickname/UUID
	Mine   bool   // filter to current user's PRs
	State  string // PR state: OPEN (default), MERGED, DECLINED
}

// RepoPRs holds the fetched PRs for one repository.
type RepoPRs struct {
	RepoSlug string
	PRs      []bitbucket.PullRequest
	Error    string
}

// Fetcher concurrently fetches PRs across repos.
type Fetcher struct {
	client *bitbucket.Client
}

// NewFetcher creates a new dashboard fetcher.
func NewFetcher(client *bitbucket.Client) *Fetcher {
	return &Fetcher{client: client}
}

// FetchAllPRs fetches open PRs from multiple repos concurrently.
func (f *Fetcher) FetchAllPRs(workspace string, repos []string, filters PRFilters) []RepoPRs {
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []RepoPRs
	)

	// Resolve current user for --mine filter
	var currentUser string
	if filters.Mine {
		user, err := f.client.GetCurrentUser()
		if err == nil {
			currentUser = user.UUID
		}
	}

	for _, repo := range repos {
		wg.Add(1)
		go func(repoSlug string) {
			defer wg.Done()

			state := filters.State
			if state == "" {
				state = "OPEN"
			}
			prs, err := f.client.ListPullRequests(workspace, repoSlug, state)

			result := RepoPRs{RepoSlug: repoSlug}
			if err != nil {
				result.Error = err.Error()
			} else {
				result.PRs = filterPRs(prs, filters, currentUser)
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

// filterPRs applies author/mine filters to a PR list.
func filterPRs(prs []bitbucket.PullRequest, filters PRFilters, currentUserUUID string) []bitbucket.PullRequest {
	if filters.Author == "" && !filters.Mine {
		return prs
	}

	filtered := make([]bitbucket.PullRequest, 0, len(prs))
	for _, pr := range prs {
		if filters.Mine && currentUserUUID != "" {
			if pr.Author.UUID != currentUserUUID {
				continue
			}
		}
		if filters.Author != "" {
			if pr.Author.Nickname != filters.Author && pr.Author.UUID != filters.Author {
				continue
			}
		}
		filtered = append(filtered, pr)
	}
	return filtered
}
