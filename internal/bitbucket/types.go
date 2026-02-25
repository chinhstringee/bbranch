package bitbucket

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

// APIError represents an error response from Bitbucket.
type APIError struct {
	Error   APIErrorDetail `json:"error"`
	Type    string         `json:"type"`
	Status  int            `json:"status"`
}

// APIErrorDetail holds the error message and detail.
type APIErrorDetail struct {
	Message string `json:"message"`
	Detail  string `json:"detail"`
}
