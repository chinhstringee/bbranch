# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

**bbranch** — CLI tool for creating Git branches and pull requests across multiple Bitbucket Cloud repositories simultaneously. Supports API token (default) and OAuth 2.0 with PKCE authentication.

- **Module**: `github.com/chinhstringee/bbranch`
- **Go version**: 1.25.0

## Commands

```bash
# Build
go build -o bbranch

# Run all tests (86 tests across 7 packages)
go test ./...

# Run single test
go test -run TestFunctionName ./internal/auth/

# Run with verbose output
go test -v ./...

# Run directly without building
go run main.go <subcommand>
```

No Makefile or linter configuration exists. Standard `go vet` and `gofmt` apply.

## CLI Usage

```bash
# Create branches across repos
bbranch create <branch-name> --repos repo-a,repo-b --from main
bbranch create <branch-name> --group backend
bbranch create <branch-name> --dry-run

# Create pull requests across repos
bbranch pr <branch-name> --repos repo-a,repo-b
bbranch pr <branch-name> --group backend --destination develop
bbranch pr <branch-name> --dry-run

# Common flags for both create and pr
#   -r, --repos       comma-separated repo slugs (supports fuzzy match)
#   -g, --group       repo group from .bbranch.yaml config
#   -i, --interactive  force interactive repo selection
#       --dry-run      preview without executing

# Other commands
bbranch list          # List workspace repos
bbranch login         # OAuth login flow
bbranch setup         # Interactive API token setup
```

## Release

Tag-based via GoReleaser. To release a new version:
```bash
git tag v0.X.0 && git push origin v0.X.0
# GitHub Actions runs GoReleaser → builds binaries + updates Homebrew tap
# Users upgrade: brew upgrade bbranch
```

## Architecture

```
main.go → cmd.Execute()
  │
  cmd/          (Cobra CLI commands)
  ├── root.go         Viper config init (.bbranch.yaml)
  ├── auth_helper.go  Builds AuthApplier from config (API token or OAuth)
  ├── resolve.go      Shared repo resolution (--repos/--group/interactive)
  ├── login.go        OAuth login flow
  ├── list.go         List workspace repos
  ├── create.go       Create branches across repos
  └── pr.go           Create pull requests across repos
  │
  internal/     (Private packages)
  ├── auth/         OAuth 2.0 + PKCE flow, token persistence (~/.bbranch/token.json)
  ├── bitbucket/    REST API client + types + AuthApplier (api.bitbucket.org/2.0)
  ├── config/       YAML config loading with env var expansion (${VAR_NAME})
  ├── creator/      Parallel branch creation orchestrator (goroutines + sync)
  └── pullrequest/  Parallel PR creation orchestrator (goroutines + sync)
```

**Key data flow for `create` command**: Config loading → Token retrieval (auto-refresh) → Repo resolution (flags/groups/interactive) → Concurrent branch creation → Colored result display.

**Key data flow for `pr` command**: Config loading → Token retrieval → Repo resolution → Per-repo: GetRepository (mainbranch) + CreatePullRequest → Colored result display with PR URLs.

**Repo resolution order**: `--interactive` flag > `--repos` flag > `--group` flag > interactive multi-select (charmbracelet/huh).

## Config

Config file: `.bbranch.yaml` (searched in cwd, then home dir). Real config is gitignored; `.bbranch.example.yaml` is the template. Supports `${ENV_VAR}` expansion for credential fields.

Auth methods: `api_token` (default, Basic auth) or `oauth` (Bearer token). OAuth token stored at `~/.bbranch/token.json` with 0600 permissions.

## Testing Patterns

- `httptest.Server` for Bitbucket API mocking
- `t.TempDir()` for file system isolation
- `t.Setenv()` for env var isolation
- Mock `AuthApplier func(req *http.Request) error` for auth injection
- Creator tests verify concurrency safety with stress tests (20 repos)

## Dependencies

- `spf13/cobra` — CLI framework
- `spf13/viper` — Config management
- `charmbracelet/huh` — Interactive TUI forms
- `fatih/color` — Colored terminal output
