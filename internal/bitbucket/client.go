package bitbucket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const baseURL = "https://api.bitbucket.org/2.0"

// TokenProvider is a function that returns a valid access token.
type TokenProvider func() (string, error)

// Client wraps the Bitbucket Cloud REST API.
type Client struct {
	httpClient    *http.Client
	tokenProvider TokenProvider
}

// NewClient creates a new Bitbucket API client.
func NewClient(tokenProvider TokenProvider) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		tokenProvider: tokenProvider,
	}
}

// NewClientWithHTTPClient creates a Bitbucket API client with a custom http.Client.
// Intended for testing with httptest servers.
func NewClientWithHTTPClient(httpClient *http.Client, tokenProvider TokenProvider) *Client {
	return &Client{
		httpClient:    httpClient,
		tokenProvider: tokenProvider,
	}
}

// ListRepositories returns all repos in a workspace (handles pagination).
func (c *Client) ListRepositories(workspace string) ([]Repository, error) {
	const maxPages = 50
	var allRepos []Repository
	nextURL := fmt.Sprintf("%s/repositories/%s?pagelen=100", baseURL, url.PathEscape(workspace))

	for i := 0; nextURL != "" && i < maxPages; i++ {
		var page PaginatedResponse
		if err := c.doRequest("GET", nextURL, nil, &page); err != nil {
			return nil, fmt.Errorf("failed to list repositories: %w", err)
		}
		allRepos = append(allRepos, page.Values...)
		nextURL = page.Next
	}

	return allRepos, nil
}

// GetRepository returns a single repository.
func (c *Client) GetRepository(workspace, repoSlug string) (*Repository, error) {
	url := fmt.Sprintf("%s/repositories/%s/%s", baseURL, url.PathEscape(workspace), url.PathEscape(repoSlug))
	var repo Repository
	if err := c.doRequest("GET", url, nil, &repo); err != nil {
		return nil, fmt.Errorf("failed to get repository %s: %w", repoSlug, err)
	}
	return &repo, nil
}

// CreateBranch creates a new branch in a repository.
func (c *Client) CreateBranch(workspace, repoSlug, branchName, sourceBranch string) (*Branch, error) {
	url := fmt.Sprintf("%s/repositories/%s/%s/refs/branches", baseURL, url.PathEscape(workspace), url.PathEscape(repoSlug))
	body := CreateBranchRequest{
		Name:   branchName,
		Target: BranchTarget{Hash: sourceBranch},
	}

	var branch Branch
	if err := c.doRequest("POST", url, body, &branch); err != nil {
		return nil, err
	}
	return &branch, nil
}

// doRequest performs an authenticated HTTP request and decodes the JSON response.
func (c *Client) doRequest(method, url string, body any, result any) error {
	token, err := c.tokenProvider()
	if err != nil {
		return fmt.Errorf("auth error: %w", err)
	}

	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle error responses
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)

		var apiErr APIError
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Error.Message != "" {
			return fmt.Errorf("API error (%d): %s", resp.StatusCode, apiErr.Error.Message)
		}
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}
