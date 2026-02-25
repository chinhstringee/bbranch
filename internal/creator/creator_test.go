package creator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/chinhstringee/bbranch/internal/bitbucket"
)

// mockBBServer builds an httptest.Server that handles branch creation requests.
// branchResponses maps repoSlug → Branch to return (status 201).
// branchErrors maps repoSlug → API error message (status 409).
func mockBBServer(t *testing.T, branchResponses map[string]bitbucket.Branch, branchErrors map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Path: /2.0/repositories/{workspace}/{slug}/refs/branches
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		// parts[0]=2.0, parts[1]=repositories, parts[2]=workspace, parts[3]=slug, ...
		if len(parts) < 4 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		slug := parts[3]

		w.Header().Set("Content-Type", "application/json")

		if errMsg, bad := branchErrors[slug]; bad {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(bitbucket.APIError{
				Error: bitbucket.APIErrorDetail{Message: errMsg},
			})
			return
		}

		if branch, ok := branchResponses[slug]; ok {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(branch)
			return
		}

		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(bitbucket.APIError{
			Error: bitbucket.APIErrorDetail{Message: "repo not found"},
		})
	}))
}

// newCreatorForServer builds a BranchCreator whose client uses the given test server.
// It replaces the package baseURL by directly calling doRequest with the server URL.
// Since CreateBranch builds the URL from baseURL, we need a client-level override.
// We achieve this via a custom transport that rewrites the host to the test server.
func newCreatorForServer(srv *httptest.Server) *BranchCreator {
	transport := &hostRewriteTransport{
		base:    http.DefaultTransport,
		srvURL:  srv.URL,
		srvHost: srv.Listener.Addr().String(),
	}
	httpClient := &http.Client{Transport: transport}
	tp := func() (string, error) { return "test-token", nil }
	client := bitbucket.NewClientWithHTTPClient(httpClient, tp)
	return NewBranchCreator(client)
}

// hostRewriteTransport rewrites all requests to go to the test server instead of the real host.
type hostRewriteTransport struct {
	base    http.RoundTripper
	srvURL  string
	srvHost string
}

func (t *hostRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request and rewrite the URL host/scheme to point at the test server.
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = "http"
	cloned.URL.Host = t.srvHost
	return t.base.RoundTrip(cloned)
}

// ---------- CreateBranches ----------

func TestCreateBranches_AllSuccess(t *testing.T) {
	repos := []string{"repo-a", "repo-b", "repo-c"}
	responses := map[string]bitbucket.Branch{
		"repo-a": {Name: "feature/test", Target: bitbucket.BranchTarget{Hash: "aabbccdd1234"}},
		"repo-b": {Name: "feature/test", Target: bitbucket.BranchTarget{Hash: "bbccddee5678"}},
		"repo-c": {Name: "feature/test", Target: bitbucket.BranchTarget{Hash: "ccddeeff9012"}},
	}

	srv := mockBBServer(t, responses, nil)
	defer srv.Close()

	bc := newCreatorForServer(srv)
	results := bc.CreateBranches("my-workspace", repos, "feature/test", "main")

	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}

	for _, r := range results {
		if !r.Success {
			t.Errorf("repo %q failed unexpectedly: %s", r.RepoSlug, r.Error)
		}
		if r.CommitHash == "" {
			t.Errorf("repo %q has empty CommitHash", r.RepoSlug)
		}
		// Hash should be truncated to 7 chars
		if len(r.CommitHash) > 7 {
			t.Errorf("repo %q CommitHash length = %d, want ≤7", r.RepoSlug, len(r.CommitHash))
		}
	}
}

func TestCreateBranches_SortedBySlug(t *testing.T) {
	repos := []string{"zeta", "alpha", "gamma", "beta"}
	responses := map[string]bitbucket.Branch{}
	for _, slug := range repos {
		responses[slug] = bitbucket.Branch{Name: "feature/x", Target: bitbucket.BranchTarget{Hash: "abc1234567"}}
	}

	srv := mockBBServer(t, responses, nil)
	defer srv.Close()

	bc := newCreatorForServer(srv)
	results := bc.CreateBranches("ws", repos, "feature/x", "main")

	expected := []string{"alpha", "beta", "gamma", "zeta"}
	for i, want := range expected {
		if results[i].RepoSlug != want {
			t.Errorf("results[%d].RepoSlug = %q, want %q", i, results[i].RepoSlug, want)
		}
	}
}

func TestCreateBranches_PartialFailure(t *testing.T) {
	repos := []string{"repo-ok", "repo-fail", "repo-ok2"}
	responses := map[string]bitbucket.Branch{
		"repo-ok":  {Name: "feature/x", Target: bitbucket.BranchTarget{Hash: "abc1234567"}},
		"repo-ok2": {Name: "feature/x", Target: bitbucket.BranchTarget{Hash: "def5678901"}},
	}
	errors := map[string]string{
		"repo-fail": "Branch already exists",
	}

	srv := mockBBServer(t, responses, errors)
	defer srv.Close()

	bc := newCreatorForServer(srv)
	results := bc.CreateBranches("ws", repos, "feature/x", "main")

	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}

	var succeeded, failed int
	for _, r := range results {
		if r.Success {
			succeeded++
		} else {
			failed++
			if r.RepoSlug != "repo-fail" {
				t.Errorf("unexpected failure: %q", r.RepoSlug)
			}
			if r.Error == "" {
				t.Errorf("failed result %q has empty Error field", r.RepoSlug)
			}
		}
	}
	if succeeded != 2 {
		t.Errorf("succeeded = %d, want 2", succeeded)
	}
	if failed != 1 {
		t.Errorf("failed = %d, want 1", failed)
	}
}

func TestCreateBranches_AllFailure(t *testing.T) {
	repos := []string{"repo-a", "repo-b"}
	errors := map[string]string{
		"repo-a": "not found",
		"repo-b": "unauthorized",
	}

	srv := mockBBServer(t, nil, errors)
	defer srv.Close()

	bc := newCreatorForServer(srv)
	results := bc.CreateBranches("ws", repos, "feature/x", "main")

	for _, r := range results {
		if r.Success {
			t.Errorf("repo %q should have failed but Success=true", r.RepoSlug)
		}
		if r.Error == "" {
			t.Errorf("repo %q has empty Error field on failure", r.RepoSlug)
		}
	}
}

func TestCreateBranches_EmptyRepoList(t *testing.T) {
	srv := mockBBServer(t, nil, nil)
	defer srv.Close()

	bc := newCreatorForServer(srv)
	results := bc.CreateBranches("ws", []string{}, "feature/x", "main")

	if len(results) != 0 {
		t.Errorf("len(results) = %d, want 0", len(results))
	}
}

func TestCreateBranches_Concurrency(t *testing.T) {
	// 20 repos — verify all are processed by counting HTTP requests.
	var requestCount atomic.Int64
	repos := make([]string, 20)
	responses := map[string]bitbucket.Branch{}
	for i := range repos {
		slug := fmt.Sprintf("repo-%02d", i)
		repos[i] = slug
		responses[slug] = bitbucket.Branch{
			Name:   "feature/x",
			Target: bitbucket.BranchTarget{Hash: "abc1234567890"},
		}
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(bitbucket.Branch{
			Name:   "feature/x",
			Target: bitbucket.BranchTarget{Hash: "abc1234567890"},
		})
	}))
	defer srv.Close()

	bc := newCreatorForServer(srv)
	results := bc.CreateBranches("ws", repos, "feature/x", "main")

	if len(results) != 20 {
		t.Errorf("len(results) = %d, want 20", len(results))
	}
	if int(requestCount.Load()) != 20 {
		t.Errorf("HTTP request count = %d, want 20", requestCount.Load())
	}
}

func TestCreateBranches_CommitHashTruncation(t *testing.T) {
	tests := []struct {
		fullHash string
		wantHash string
	}{
		{"abc1234def5678", "abc1234"}, // > 7 chars → truncate
		{"abc1234", "abc1234"},        // exactly 7 → unchanged
		{"abc12", "abc12"},            // < 7 → unchanged
		{"", ""},                      // empty → empty
	}

	for _, tc := range tests {
		srv := mockBBServer(t, map[string]bitbucket.Branch{
			"test-repo": {Name: "feature/x", Target: bitbucket.BranchTarget{Hash: tc.fullHash}},
		}, nil)

		bc := newCreatorForServer(srv)
		results := bc.CreateBranches("ws", []string{"test-repo"}, "feature/x", "main")

		srv.Close()

		if len(results) != 1 {
			t.Errorf("hash %q: len(results) = %d, want 1", tc.fullHash, len(results))
			continue
		}
		if results[0].CommitHash != tc.wantHash {
			t.Errorf("hash %q: CommitHash = %q, want %q", tc.fullHash, results[0].CommitHash, tc.wantHash)
		}
	}
}

// ---------- NewBranchCreator ----------

func TestNewBranchCreator_NotNil(t *testing.T) {
	bc := NewBranchCreator(nil)
	if bc == nil {
		t.Fatal("NewBranchCreator returned nil")
	}
}
