# Phase 3: Bitbucket API Client

**Priority:** High | **Status:** Pending

## Overview
HTTP client wrapping Bitbucket Cloud REST API 2.0 using `net/http`. Uses OAuth Bearer token from auth module.

## Key Insights
- Base URL: `https://api.bitbucket.org/2.0`
- Auth: `Authorization: Bearer <access_token>` (from OAuth)
- Pagination: `next` field in response JSON
- Rate limit: 1,000 req/hour, check `X-RateLimit-Remaining` header

## Implementation Steps

1. **Create `internal/bitbucket/types.go`** — API structs:
   - `Repository` — slug, name, mainbranch
   - `Branch` — name, target hash
   - `PaginatedResponse[T]` — values, next, page, size
   - `CreateBranchRequest` — name, target
   - `APIError` — message, detail

2. **Create `internal/bitbucket/client.go`** — Client struct:
   - `NewClient(tokenFunc func() (string, error)) *Client` — takes token provider from auth module
   - `ListRepositories(workspace string) ([]Repository, error)` — handles pagination
   - `GetRepository(workspace, repoSlug string) (*Repository, error)`
   - `CreateBranch(workspace, repoSlug, branchName, sourceBranch string) (*Branch, error)`
   - `GetBranch(workspace, repoSlug, branchName string) (*Branch, error)`
   - Internal: `doRequest(method, path string, body any) (*http.Response, error)` — sets Bearer token, content-type, checks rate limits

3. **Error handling**:
   - 400 → branch already exists or invalid params
   - 401 → token expired (trigger refresh)
   - 403 → insufficient permissions
   - 404 → repo/branch not found
   - 429 → rate limited (log remaining, retry-after)

## Files to Create
- `internal/bitbucket/types.go`
- `internal/bitbucket/client.go`

## Success Criteria
- [ ] Client compiles
- [ ] Uses Bearer token from OAuth auth module
- [ ] Handles pagination for repos with 100+ entries
- [ ] Returns typed errors for common HTTP status codes
