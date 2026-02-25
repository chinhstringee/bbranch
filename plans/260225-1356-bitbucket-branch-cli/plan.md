# Plan: Bitbucket Multi-Repo Branch Creator CLI (Go)

**Date:** 2026-02-25
**Stack:** Go 1.26 + Cobra + Viper
**Research:** [research report](../reports/research-260225-1347-bitbucket-multi-repo-branch-cli.md)

## Overview

CLI tool (`bbranch`) to create git branches across multiple Bitbucket Cloud repositories simultaneously. Supports repo groups, parallel execution, dry-run, and interactive selection.

## Architecture

```
git-branch-creator/
├── main.go                     # Entry point
├── go.mod / go.sum
├── cmd/
│   ├── root.go                 # Root Cobra command + Viper config
│   ├── create.go               # `bbranch create <name> --group <g>`
│   ├── list.go                 # `bbranch list`
│   └── config_init.go          # `bbranch config init`
├── internal/
│   ├── auth/
│   │   └── oauth.go            # OAuth 2.0 Authorization Code + PKCE
│   ├── bitbucket/
│   │   ├── client.go           # HTTP client (net/http)
│   │   └── types.go            # API request/response structs
│   ├── config/
│   │   └── config.go           # Viper config management
│   └── creator/
│       └── creator.go          # Parallel branch creation orchestrator
└── .bbranch.yaml               # User config file (per-project)
```

## Config Format (`.bbranch.yaml`)

```yaml
workspace: my-workspace
oauth:
  client_id: ${BITBUCKET_OAUTH_CLIENT_ID}
  client_secret: ${BITBUCKET_OAUTH_CLIENT_SECRET}
groups:
  backend:
    - repo-api
    - repo-worker
    - repo-shared
  frontend:
    - repo-web
    - repo-mobile
  all:
    - repo-api
    - repo-worker
    - repo-shared
    - repo-web
    - repo-mobile
defaults:
  source_branch: master
  branch_prefix: "feature/"
```

## CLI Commands

```bash
# Login (OAuth browser flow)
bbranch login

# Create branch across a group
bbranch create feature/PROJ-123-login --group backend

# Create branch across specific repos
bbranch create hotfix/urgent-fix --repos repo-api,repo-worker

# Create with custom source branch
bbranch create release/v2.0 --group all --from develop

# Interactive repo selection
bbranch create feature/new-ui --interactive

# Dry run
bbranch create feature/test --group all --dry-run

# List workspace repos
bbranch list

# Initialize config
bbranch config init
```

## Phases

| # | Phase | Status | File |
|---|-------|--------|------|
| 1 | Project Setup & Scaffolding | Pending | [phase-01](phase-01-project-setup.md) |
| 2 | OAuth 2.0 Auth | Pending | [phase-02](phase-02-oauth-auth.md) |
| 3 | Bitbucket API Client | Pending | [phase-03](phase-03-bitbucket-client.md) |
| 4 | Config Management | Pending | [phase-04](phase-04-config-management.md) |
| 5 | Create Branch Command | Pending | [phase-05](phase-05-create-command.md) |
| 6 | List & Config Init Commands | Pending | [phase-06](phase-06-list-config-commands.md) |

## Key Dependencies

- `github.com/spf13/cobra` — CLI framework
- `github.com/spf13/viper` — Config management
- `github.com/fatih/color` — Colored terminal output
- `github.com/charmbracelet/huh` — Interactive forms/prompts
- `golang.org/x/oauth2` — OAuth 2.0 client with PKCE support

## Key Decisions

1. **OAuth 2.0 only** — Authorization Code + PKCE flow (opens browser like `gh auth login`)
2. **Token stored locally** — `~/.bbranch/token.json` (access + refresh tokens)
3. **No external HTTP library** — `net/http` sufficient for REST calls
4. **YAML config** — Viper's best-supported format, human-readable
5. **Goroutines for parallelism** — `sync.WaitGroup` + channels for results
6. **Default source branch: `master`** — overridable via `--from` flag
7. **Bitbucket Cloud only** — No Server/Data Center support (YAGNI)
8. **No branch deletion** — Create-only scope (YAGNI)
