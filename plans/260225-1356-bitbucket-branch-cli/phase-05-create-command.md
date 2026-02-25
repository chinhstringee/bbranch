# Phase 5: Create Branch Command (Core)

**Priority:** High | **Status:** Pending

## Overview
The main command: `bbranch create <branch-name>` with parallel execution across repos.

## CLI Interface
```bash
bbranch create <branch-name> [flags]

Flags:
  --group, -g       Repo group from config
  --repos, -r       Comma-separated repo slugs (override group)
  --from, -f        Source branch (default: from config or "master")
  --dry-run         Preview actions without executing
  --interactive, -i Select repos interactively
```

## Implementation Steps

1. **Create `cmd/create.go`** — Cobra command:
   - Parse branch name (positional arg, required)
   - Resolve target repos: `--repos` > `--interactive` > `--group`
   - Resolve source branch: `--from` > config defaults > "master"
   - If `--dry-run`: print plan and exit
   - Call `creator.CreateBranches()`

2. **Create `internal/creator/creator.go`** — Orchestrator:
   - `BranchCreator` struct — holds bitbucket client + config
   - `Result` struct — RepoSlug, Success, Error, BranchName
   - `CreateBranches(repos []string, branchName, sourceBranch string) []Result`
     - Launch goroutine per repo (with `sync.WaitGroup`)
     - Collect results via channel
     - Return sorted results slice
   - `PrintResults(results []Result)` — colored summary table

3. **Interactive mode** (`--interactive`):
   - Use `huh` library for multi-select form
   - Fetch repo list from API (`client.ListRepositories`)
   - Present checkboxes, return selected slugs

4. **Output format**:
   ```
   Creating branch "feature/PROJ-123" across 3 repos...

   ✓ repo-api         created from master (a1b2c3d)
   ✓ repo-worker      created from master (d4e5f6g)
   ✗ repo-shared      branch already exists

   Summary: 2 succeeded, 1 failed
   ```

## Files to Create
- `cmd/create.go`
- `internal/creator/creator.go`

## Success Criteria
- [ ] Creates branches in parallel across multiple repos
- [ ] `--dry-run` shows plan without executing
- [ ] `--interactive` shows repo selection UI
- [ ] Gracefully handles partial failures
- [ ] Colored summary table output
