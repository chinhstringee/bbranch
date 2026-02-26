package bitbucket

import "encoding/json"

// Repository represents a Bitbucket repository.
type Repository struct {
	Slug       string     `json:"slug"`
	Name       string     `json:"name"`
	FullName   string     `json:"full_name"`
	MainBranch *BranchRef `json:"mainbranch"`
	UpdatedOn  string     `json:"updated_on"`
}

// BranchRef is a short branch reference (used in Repository.MainBranch).
type BranchRef struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Branch represents a full branch object from the API.
type Branch struct {
	Name   string       `json:"name"`
	Target BranchTarget `json:"target"`
}

// BranchTarget holds the commit hash a branch points to.
type BranchTarget struct {
	Hash string `json:"hash"`
}

// CreateBranchRequest is the POST body for creating a branch.
type CreateBranchRequest struct {
	Name   string       `json:"name"`
	Target BranchTarget `json:"target"`
}

// PaginatedResponse wraps Bitbucket's paginated API responses.
type PaginatedResponse struct {
	Values []Repository `json:"values"`
	Next   string       `json:"next"`
	Page   int          `json:"page"`
	Size   int          `json:"size"`
}

// CreatePullRequestRequest is the POST body for creating a pull request.
type CreatePullRequestRequest struct {
	Title             string      `json:"title"`
	Description       string      `json:"description"`
	Source            PRBranchRef `json:"source"`
	Destination       PRBranchRef `json:"destination"`
	CloseSourceBranch bool        `json:"close_source_branch"`
}

// PRBranchRef wraps a branch name reference for PR source/destination.
type PRBranchRef struct {
	Branch PRBranchName `json:"branch"`
}

// PRBranchName holds a branch name.
type PRBranchName struct {
	Name string `json:"name"`
}

// PullRequest represents a Bitbucket pull request response.
type PullRequest struct {
	ID    int     `json:"id"`
	Title string  `json:"title"`
	State string  `json:"state"`
	Links PRLinks `json:"links"`
}

// PRLinks holds pull request link references.
type PRLinks struct {
	HTML LinkRef `json:"html"`
}

// LinkRef holds an href URL.
type LinkRef struct {
	Href string `json:"href"`
}

// Commit represents a Bitbucket commit.
type Commit struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
}

// PaginatedCommits wraps Bitbucket's paginated commit responses.
type PaginatedCommits struct {
	Values []Commit `json:"values"`
	Next   string   `json:"next"`
}

// APIError represents an error response from Bitbucket.
type APIError struct {
	Error   APIErrorDetail `json:"error"`
	Type    string         `json:"type"`
	Status  int            `json:"status"`
}

// APIErrorDetail holds the error message and detail.
// Detail is json.RawMessage because Bitbucket returns either a string or an object.
type APIErrorDetail struct {
	Message string          `json:"message"`
	Detail  json.RawMessage `json:"detail"`
}

// ScopeDetail holds required/granted permission scopes from 403 errors.
type ScopeDetail struct {
	Required []string `json:"required"`
	Granted  []string `json:"granted"`
}
