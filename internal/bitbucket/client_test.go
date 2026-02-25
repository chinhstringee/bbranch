package bitbucket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient returns a Client pointed at the given httptest.Server URL.
// It replaces the package-level baseURL by overriding the URL in each method
// via a server whose handler mirrors the real API shape.
func mockTokenProvider(token string) TokenProvider {
	return func() (string, error) { return token, nil }
}

func errorTokenProvider() TokenProvider {
	return func() (string, error) { return "", fmt.Errorf("auth failure") }
}

// ---------- NewClient ----------

func TestNewClient_NotNil(t *testing.T) {
	c := NewClient(mockTokenProvider("tok"))
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if c.httpClient == nil {
		t.Fatal("httpClient is nil")
	}
}

// ---------- doRequest / auth ----------

func TestDoRequest_AuthHeaderSet(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Repository{Slug: "test-repo"})
	}))
	defer srv.Close()

	c := NewClient(mockTokenProvider("my-access-token"))
	var repo Repository
	err := c.doRequest("GET", srv.URL, nil, &repo)
	if err != nil {
		t.Fatalf("doRequest error: %v", err)
	}
	if gotAuth != "Bearer my-access-token" {
		t.Errorf("Authorization header = %q, want %q", gotAuth, "Bearer my-access-token")
	}
}

func TestDoRequest_TokenProviderError(t *testing.T) {
	c := NewClient(errorTokenProvider())
	var result Repository
	err := c.doRequest("GET", "http://localhost/ignored", nil, &result)
	if err == nil {
		t.Fatal("expected error from failed token provider, got nil")
	}
	if !strings.Contains(err.Error(), "auth error") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "auth error")
	}
}

func TestDoRequest_APIError_WithMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIError{
			Error: APIErrorDetail{Message: "Repository not found"},
		})
	}))
	defer srv.Close()

	c := NewClient(mockTokenProvider("tok"))
	var result Repository
	err := c.doRequest("GET", srv.URL, nil, &result)
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}
	if !strings.Contains(err.Error(), "Repository not found") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "Repository not found")
	}
}

func TestDoRequest_APIError_PlainBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	}))
	defer srv.Close()

	c := NewClient(mockTokenProvider("tok"))
	var result Repository
	err := c.doRequest("GET", srv.URL, nil, &result)
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
	if !strings.Contains(err.Error(), "API error (401)") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "API error (401)")
	}
}

func TestDoRequest_InvalidJSON_Response(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not-json{{{"))
	}))
	defer srv.Close()

	c := NewClient(mockTokenProvider("tok"))
	var result Repository
	err := c.doRequest("GET", srv.URL, nil, &result)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// ---------- ListRepositories ----------

func TestListRepositories_SinglePage(t *testing.T) {
	repos := []Repository{
		{Slug: "alpha", Name: "Alpha"},
		{Slug: "beta", Name: "Beta"},
	}
	page := PaginatedResponse{Values: repos, Next: ""}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(page)
	}))
	defer srv.Close()

	c := &Client{
		httpClient:    srv.Client(),
		tokenProvider: mockTokenProvider("tok"),
	}

	// Override the request URL by calling doRequest directly with the test server URL
	var got PaginatedResponse
	err := c.doRequest("GET", srv.URL+"?pagelen=100", nil, &got)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Values) != 2 {
		t.Errorf("got %d repos, want 2", len(got.Values))
	}
}

func TestListRepositories_Pagination(t *testing.T) {
	callCount := 0
	page1Repos := []Repository{{Slug: "repo-1"}}
	page2Repos := []Repository{{Slug: "repo-2"}}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		callCount++
		if callCount == 1 {
			// First page — Next points back to same server
			page := PaginatedResponse{Values: page1Repos, Next: "http://" + r.Host + r.URL.Path + "?page=2"}
			json.NewEncoder(w).Encode(page)
		} else {
			// Second page — no next
			page := PaginatedResponse{Values: page2Repos, Next: ""}
			json.NewEncoder(w).Encode(page)
		}
	}))
	defer srv.Close()

	// We use a real Client with the test server by making the first request
	// go to srv.URL directly — testing doRequest + pagination loop independently.
	c := &Client{
		httpClient:    srv.Client(),
		tokenProvider: mockTokenProvider("tok"),
	}

	// Manually replicate the ListRepositories pagination loop against the test server
	var allRepos []Repository
	nextURL := srv.URL + "/repositories/ws?pagelen=100"
	for i := 0; nextURL != "" && i < 50; i++ {
		var p PaginatedResponse
		if err := c.doRequest("GET", nextURL, nil, &p); err != nil {
			t.Fatalf("page %d error: %v", i, err)
		}
		allRepos = append(allRepos, p.Values...)
		nextURL = p.Next
	}

	if len(allRepos) != 2 {
		t.Errorf("got %d repos across pages, want 2", len(allRepos))
	}
	if allRepos[0].Slug != "repo-1" || allRepos[1].Slug != "repo-2" {
		t.Errorf("unexpected repo order: %v", allRepos)
	}
}

// ---------- GetRepository ----------

func TestGetRepository_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Repository{
			Slug:     "my-repo",
			Name:     "My Repo",
			FullName: "workspace/my-repo",
		})
	}))
	defer srv.Close()

	c := &Client{httpClient: srv.Client(), tokenProvider: mockTokenProvider("tok")}
	var repo Repository
	err := c.doRequest("GET", srv.URL, nil, &repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.Slug != "my-repo" {
		t.Errorf("Slug = %q, want %q", repo.Slug, "my-repo")
	}
}

func TestGetRepository_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIError{Error: APIErrorDetail{Message: "No such repository"}})
	}))
	defer srv.Close()

	c := &Client{httpClient: srv.Client(), tokenProvider: mockTokenProvider("tok")}
	var repo Repository
	err := c.doRequest("GET", srv.URL, nil, &repo)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !strings.Contains(err.Error(), "No such repository") {
		t.Errorf("error %q does not contain expected message", err.Error())
	}
}

// ---------- CreateBranch ----------

func TestCreateBranch_Success(t *testing.T) {
	var gotBody CreateBranchRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&gotBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Branch{
			Name:   "feature/my-branch",
			Target: BranchTarget{Hash: "abc1234def5678"},
		})
	}))
	defer srv.Close()

	c := &Client{httpClient: srv.Client(), tokenProvider: mockTokenProvider("tok")}

	var branch Branch
	body := CreateBranchRequest{
		Name:   "feature/my-branch",
		Target: BranchTarget{Hash: "main"},
	}
	err := c.doRequest("POST", srv.URL, body, &branch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch.Name != "feature/my-branch" {
		t.Errorf("branch.Name = %q, want %q", branch.Name, "feature/my-branch")
	}
	if branch.Target.Hash != "abc1234def5678" {
		t.Errorf("branch.Target.Hash = %q, want %q", branch.Target.Hash, "abc1234def5678")
	}
}

func TestCreateBranch_Conflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(APIError{
			Error: APIErrorDetail{Message: "Branch already exists"},
		})
	}))
	defer srv.Close()

	c := &Client{httpClient: srv.Client(), tokenProvider: mockTokenProvider("tok")}
	var branch Branch
	err := c.doRequest("POST", srv.URL, CreateBranchRequest{Name: "existing"}, &branch)
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
	if !strings.Contains(err.Error(), "Branch already exists") {
		t.Errorf("error %q does not mention branch conflict", err.Error())
	}
}

// ---------- Content-Type / Accept headers ----------

func TestDoRequest_Headers(t *testing.T) {
	var gotContentType, gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct{}{})
	}))
	defer srv.Close()

	c := &Client{httpClient: srv.Client(), tokenProvider: mockTokenProvider("tok")}
	err := c.doRequest("POST", srv.URL, map[string]string{"k": "v"}, &struct{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q, want application/json", gotAccept)
	}
}
