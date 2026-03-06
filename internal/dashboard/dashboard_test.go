package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/chinhstringee/buck/internal/bitbucket"
)

// hostRewriteTransport rewrites all requests to the test server.
type hostRewriteTransport struct {
	base    http.RoundTripper
	srvHost string
}

func (t *hostRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = "http"
	cloned.URL.Host = t.srvHost
	return t.base.RoundTrip(cloned)
}

func newFetcherForServer(srv *httptest.Server) *Fetcher {
	transport := &hostRewriteTransport{
		base:    http.DefaultTransport,
		srvHost: srv.Listener.Addr().String(),
	}
	httpClient := &http.Client{Transport: transport}
	authApplier := bitbucket.BearerAuth(func() (string, error) { return "test-token", nil })
	client := bitbucket.NewClientWithHTTPClient(httpClient, authApplier)
	return NewFetcher(client)
}

func mockDashboardServer(t *testing.T, prsByRepo map[string][]bitbucket.PullRequest, user *bitbucket.User) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := strings.Trim(r.URL.Path, "/")
		parts := strings.Split(path, "/")

		// GET /2.0/user
		if path == "2.0/user" && user != nil {
			json.NewEncoder(w).Encode(user)
			return
		}

		// GET /2.0/repositories/{ws}/{slug}/pullrequests
		if len(parts) >= 5 && parts[4] == "pullrequests" {
			slug := parts[3]
			prs := prsByRepo[slug]
			json.NewEncoder(w).Encode(bitbucket.PaginatedPullRequests{
				Values: prs,
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestFetchAllPRs_Success(t *testing.T) {
	prsByRepo := map[string][]bitbucket.PullRequest{
		"repo-a": {
			{ID: 1, Title: "PR 1", Author: bitbucket.PRAuthor{DisplayName: "Alice", UUID: "{alice}"}},
			{ID: 2, Title: "PR 2", Author: bitbucket.PRAuthor{DisplayName: "Bob", UUID: "{bob}"}},
		},
		"repo-b": {
			{ID: 3, Title: "PR 3", Author: bitbucket.PRAuthor{DisplayName: "Alice", UUID: "{alice}"}},
		},
	}

	srv := mockDashboardServer(t, prsByRepo, nil)
	defer srv.Close()

	f := newFetcherForServer(srv)
	results := f.FetchAllPRs("ws", []string{"repo-a", "repo-b"}, PRFilters{})

	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}

	// Sorted by slug
	if results[0].RepoSlug != "repo-a" {
		t.Errorf("results[0].RepoSlug = %q, want repo-a", results[0].RepoSlug)
	}
	if len(results[0].PRs) != 2 {
		t.Errorf("repo-a PRs = %d, want 2", len(results[0].PRs))
	}
	if len(results[1].PRs) != 1 {
		t.Errorf("repo-b PRs = %d, want 1", len(results[1].PRs))
	}
}

func TestFetchAllPRs_MineFilter(t *testing.T) {
	prsByRepo := map[string][]bitbucket.PullRequest{
		"repo-a": {
			{ID: 1, Title: "My PR", Author: bitbucket.PRAuthor{UUID: "{alice}"}},
			{ID: 2, Title: "Other PR", Author: bitbucket.PRAuthor{UUID: "{bob}"}},
		},
	}
	user := &bitbucket.User{UUID: "{alice}"}

	srv := mockDashboardServer(t, prsByRepo, user)
	defer srv.Close()

	f := newFetcherForServer(srv)
	results := f.FetchAllPRs("ws", []string{"repo-a"}, PRFilters{Mine: true})

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if len(results[0].PRs) != 1 {
		t.Errorf("filtered PRs = %d, want 1", len(results[0].PRs))
	}
	if results[0].PRs[0].ID != 1 {
		t.Errorf("expected PR #1 (mine), got #%d", results[0].PRs[0].ID)
	}
}

func TestFetchAllPRs_AuthorFilter(t *testing.T) {
	prsByRepo := map[string][]bitbucket.PullRequest{
		"repo-a": {
			{ID: 1, Author: bitbucket.PRAuthor{Nickname: "alice"}},
			{ID: 2, Author: bitbucket.PRAuthor{Nickname: "bob"}},
			{ID: 3, Author: bitbucket.PRAuthor{Nickname: "alice"}},
		},
	}

	srv := mockDashboardServer(t, prsByRepo, nil)
	defer srv.Close()

	f := newFetcherForServer(srv)
	results := f.FetchAllPRs("ws", []string{"repo-a"}, PRFilters{Author: "alice"})

	if len(results[0].PRs) != 2 {
		t.Errorf("filtered PRs = %d, want 2", len(results[0].PRs))
	}
}

func TestFetchAllPRs_EmptyRepos(t *testing.T) {
	srv := mockDashboardServer(t, nil, nil)
	defer srv.Close()

	f := newFetcherForServer(srv)
	results := f.FetchAllPRs("ws", []string{}, PRFilters{})

	if len(results) != 0 {
		t.Errorf("len(results) = %d, want 0", len(results))
	}
}

func TestFetchAllPRs_Concurrency(t *testing.T) {
	var requestCount atomic.Int64

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bitbucket.PaginatedPullRequests{})
	}))
	defer srv.Close()

	repos := make([]string, 15)
	for i := range repos {
		repos[i] = "repo"
	}

	f := newFetcherForServer(srv)
	results := f.FetchAllPRs("ws", repos, PRFilters{})

	if len(results) != 15 {
		t.Errorf("len(results) = %d, want 15", len(results))
	}
	if int(requestCount.Load()) != 15 {
		t.Errorf("request count = %d, want 15", requestCount.Load())
	}
}

func TestFetchAllPRs_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(bitbucket.APIError{
			Error: bitbucket.APIErrorDetail{Message: "Forbidden"},
		})
	}))
	defer srv.Close()

	f := newFetcherForServer(srv)
	results := f.FetchAllPRs("ws", []string{"repo-a"}, PRFilters{})

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].Error == "" {
		t.Error("expected error, got none")
	}
}

func TestFilterPRs_NoFilters(t *testing.T) {
	prs := []bitbucket.PullRequest{{ID: 1}, {ID: 2}}
	got := filterPRs(prs, PRFilters{}, "")
	if len(got) != 2 {
		t.Errorf("len = %d, want 2", len(got))
	}
}
