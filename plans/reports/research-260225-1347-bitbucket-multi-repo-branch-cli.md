# Research Report: Bitbucket Multi-Repo Branch Creator CLI

**Date:** 2026-02-25
**Sources:** 3 research queries (Bitbucket API, CLI frameworks, existing tools)

## Executive Summary

Building a CLI tool to create branches across multiple Bitbucket repositories is well-supported by Bitbucket Cloud REST API 2.0. The API provides endpoints for listing repos, getting default branches, and creating branches programmatically. No bulk API exists — must iterate repos individually with parallel requests.

**Recommended stack:** Node.js with TypeScript (Commander.js + prompts library) for fastest development, or Go (Cobra + Bubble Tea) for binary distribution. Given this is a dev tool in a demo project, **Node.js/TypeScript** is the pragmatic choice — fast to build, easy to iterate.

## Key Findings

### 1. Bitbucket Cloud REST API 2.0

#### Create Branch
```
POST /2.0/repositories/{workspace}/{repo_slug}/refs/branches
```
```json
{
  "name": "feature/branch-name",
  "target": { "hash": "main" }
}
```
- `target.hash` accepts branch name (e.g., "main") or commit SHA

#### List Repositories
```
GET /2.0/repositories/{workspace}?pagelen=100
```
- Max 100 per page, paginated via `next` link

#### Get Default Branch
```
GET /2.0/repositories/{workspace}/{repo_slug}
```
- Response: `mainbranch.name` field

#### Get Branch SHA
```
GET /2.0/repositories/{workspace}/{repo_slug}/refs/branches/{branch_name}
```
- Response: `target.hash` field

#### Authentication
| Method | Format | Best For |
|--------|--------|----------|
| App Password | Basic Auth (`username:app_password`) | Scripts/CLI tools |
| Repository Access Token | Bearer token | Repo-scoped access |
| Workspace Access Token | Bearer token | Workspace-wide ops |
| OAuth 2.0 | Bearer token | Third-party apps |

**Recommendation:** App Password or Workspace Access Token for CLI tool.

#### Rate Limits
- 1,000 req/hour base per user
- Scaled: +10 RPH per workspace seat (cap 10,000)
- Headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`
- Exceeding → `429 Too Many Requests`

### 2. CLI Framework Comparison

| Criteria | Node.js (Commander) | Go (Cobra) | Python (Typer) | Rust (Clap) |
|----------|-------------------|------------|----------------|-------------|
| Dev speed | Very fast | Fast | Very fast | Moderate |
| Distribution | npm/npx | Single binary | Needs Python | Single binary |
| Parallelism | async/await | goroutines | asyncio (GIL) | tokio |
| Interactive UI | clack/inquirer | bubbletea | questionary | dialoguer |
| Config mgmt | cosmiconfig | viper | pydantic | serde |

**Decision:** Node.js/TypeScript — fastest iteration, native async for parallel API calls, rich prompt libraries, easy distribution via `npx`.

### 3. Existing Tools & Patterns

| Tool | Language | Approach |
|------|----------|----------|
| `meta` | Node.js | Meta-repo with `.meta` JSON config |
| `mu-repo` | Python | INI-style config with repo grouping |
| `git-xargs` | Go | Runs scripts across repos, creates PRs |
| `multi-gitter` | Go | Bulk operations with PR automation |

None are Bitbucket-specific. All iterate repos individually.

#### Common Patterns
- **Config file**: JSON/YAML with repo slugs, groups, base branch overrides
- **Branch naming**: Prefix convention (`feature/`, `bugfix/`) + ticket number
- **Error handling**: Continue on failure, report summary at end
- **Source branch**: Default to repo's main branch, allow override per-repo

## Implementation Recommendations

### Architecture

```
git-branch-creator/
├── src/
│   ├── index.ts              # CLI entry point
│   ├── commands/
│   │   ├── create.ts         # Create branches command
│   │   ├── list.ts           # List repos command
│   │   └── config.ts         # Manage config command
│   ├── services/
│   │   ├── bitbucket-api.ts  # Bitbucket API client
│   │   └── branch-creator.ts # Branch creation orchestrator
│   ├── config/
│   │   └── config-manager.ts # Config file management
│   └── types/
│       └── index.ts          # TypeScript types
├── package.json
├── tsconfig.json
└── .branch-creator.json      # User config (per-project)
```

### Config File Format (`.branch-creator.json`)
```json
{
  "workspace": "my-workspace",
  "auth": {
    "type": "app-password",
    "username": "env:BITBUCKET_USERNAME",
    "password": "env:BITBUCKET_APP_PASSWORD"
  },
  "groups": {
    "backend": ["repo-api", "repo-worker", "repo-shared"],
    "frontend": ["repo-web", "repo-mobile"],
    "all": ["repo-api", "repo-worker", "repo-shared", "repo-web", "repo-mobile"]
  },
  "defaults": {
    "sourceBranch": "main",
    "branchPrefix": "feature/"
  }
}
```

### Core Features
1. **Create branch across repos**: `branch-creator create feature/PROJ-123-new-feature --group backend`
2. **List workspace repos**: `branch-creator list`
3. **Interactive mode**: Select repos via checkbox prompt
4. **Parallel execution**: `Promise.allSettled()` for concurrent API calls
5. **Dry run**: `--dry-run` flag to preview actions
6. **Config management**: `branch-creator config init` for setup wizard

### Key Libraries
- `commander` — CLI argument parsing
- `@clack/prompts` — Beautiful interactive prompts
- `ofetch` or `ky` — HTTP client
- `chalk` — Colored output
- `cosmiconfig` — Config file discovery
- `zod` — Config validation

### Error Handling Strategy
- Pre-flight: Validate auth, check repos exist
- Execution: `Promise.allSettled()` — don't fail-fast
- Report: Summary table (repo → success/fail/reason)
- Rate limiting: Respect `X-RateLimit-Remaining`, add delay if low

## Common Pitfalls
- Branch already exists → API returns 400, handle gracefully
- Repo not found → May be private or misspelled
- Auth scope insufficient → App Password needs "Repositories: Write"
- Pagination → Must follow `next` links for workspaces with 100+ repos
- Rate limits → With 50+ repos, space requests or batch

## Unresolved Questions
1. Should the tool support Bitbucket Server/Data Center in addition to Cloud?
2. Should it support deleting branches across repos (cleanup)?
3. Should config support per-repo base branch overrides?
4. Should it integrate with Jira for ticket-based branch naming?
