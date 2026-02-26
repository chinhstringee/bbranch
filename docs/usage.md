# bbranch Usage Guide

Create git branches across multiple Bitbucket Cloud repositories simultaneously.

## Quick Start

### 1. Install Prerequisites

- **Go 1.25 or later** — [Download Go](https://golang.org/dl/)
- **Bitbucket Cloud workspace** — [Create one](https://bitbucket.org)

### 2. Build and Configure

```bash
go build -o bbranch
cp .bbranch.example.yaml .bbranch.yaml
```

### 3. Authentication Setup

#### Option A: API Token (default, recommended)

1. Go to [Bitbucket > Personal settings > Security > API tokens](https://bitbucket.org/account/settings/api-tokens/)
2. Click "Create API token with scopes"
3. Select scopes: `read:repository:bitbucket`, `write:repository:bitbucket`
4. Copy the token (shown only once)

```yaml
workspace: your-workspace-slug

api_token:
  email: your-email@example.com
  token: YOUR_API_TOKEN
```

No `bbranch login` needed — works immediately.

#### Option B: OAuth 2.0 + PKCE

1. Go to Bitbucket workspace → Settings → API → OAuth consumers
2. Click "Add consumer"
3. Configure:
   - **Name**: `bbranch`
   - **Callback URL**: `http://localhost:9876/callback`
   - **Permissions**: Repositories (Read + Write)
4. Save client ID and secret

```yaml
workspace: your-workspace-slug

auth:
  method: oauth

oauth:
  client_id: YOUR_CLIENT_ID
  client_secret: YOUR_CLIENT_SECRET
```

Then run `bbranch login` to authenticate via browser.

### 4. Add Groups (optional)

```yaml
groups:
  backend:
    - api-repo
    - worker-repo
  frontend:
    - web-repo

defaults:
  source_branch: master
  branch_prefix: "feature/"
```

## Commands

### `bbranch login`

Authenticate with Bitbucket via OAuth 2.0 browser flow. Only needed when using `auth.method: oauth`.

```bash
bbranch login
```

**What it does:**
- Opens browser for OAuth authorization
- Stores token in `~/.bbranch/token.json`
- Token reused for all subsequent commands

**Note**: Not needed for API token auth. Run when token expires or you need to switch accounts.

---

### `bbranch list`

List all repositories in your workspace with metadata.

```bash
bbranch list
```

**Output example:**

```
Fetching repos from workspace "my-workspace"...

REPO                           DEFAULT BRANCH     UPDATED
─────────────────────────────────────────────────────────────
api-repo                       main               2025-02-20
web-repo                       master             2025-02-18
worker-repo                    main               2025-02-15
mobile-repo                    develop            2025-02-10

Total: 4 repositories
```

**Use cases:**
- Verify workspace access
- Find exact repo slugs for `--repos` flag
- Check default branches

---

### `bbranch create <branch-name>`

Create a branch across selected repositories.

```bash
bbranch create feature/new-feature
```

**By default**, prompts interactive multi-select of repos. Navigate with arrow keys, toggle with space, confirm with enter.

#### Options

| Flag | Short | Description |
|------|-------|-------------|
| `--group` | `-g` | Use predefined repo group from config |
| `--repos` | `-r` | Comma-separated repo slugs |
| `--from` | `-f` | Source branch (overrides config default) |
| `--dry-run` | | Preview without executing |
| `--interactive` | `-i` | Force interactive selection |
| `--config` | | Custom config file path |

#### Examples

**Interactive mode (default):**

```bash
bbranch create release/v1.2.3
```

**Using a group from config:**

```bash
bbranch create feature/auth --group backend
```

Groups must be defined in `.bbranch.yaml`:

```yaml
groups:
  backend:
    - api-repo
    - worker-repo
```

**Specific repositories:**

```bash
bbranch create bugfix/cors --repos api-repo,web-repo
```

**From non-default branch:**

```bash
bbranch create release/v2.0 --from develop
```

**Preview without creating:**

```bash
bbranch create feature/test --dry-run
```

Output:

```
Dry run: would create branch "feature/test" from "master" in:
  - api-repo
  - web-repo
```

**Custom config file:**

```bash
bbranch create feature/x --config /path/to/.bbranch.yaml
```

---

### `bbranch pr <branch-name>`

Create pull requests across selected repositories from a specific branch to main/default branch.

```bash
bbranch pr feature/auth
```

**By default**, prompts interactive multi-select of repos. Navigate with arrow keys, toggle with space, confirm with enter.

#### Options

| Flag | Short | Description |
|------|-------|-------------|
| `--group` | `-g` | Use predefined repo group from config |
| `--repos` | `-r` | Comma-separated repo slugs |
| `--source` | `-s` | Source branch (defaults to target branch name) |
| `--destination` | `-d` | Destination branch (defaults to main/default per repo) |
| `--dry-run` | | Preview without creating |
| `--interactive` | `-i` | Force interactive selection |
| `--config` | | Custom config file path |

#### Examples

**Interactive mode (default):**

```bash
bbranch pr feature/auth
```

Creates PRs from `feature/auth` to each repo's main branch.

**Using a group from config:**

```bash
bbranch pr feature/auth --group backend
```

**Specific repositories:**

```bash
bbranch pr feature/cors --repos api-repo,web-repo
```

**Custom destination branch:**

```bash
bbranch pr feature/hotfix --destination develop --repos api-repo,worker-repo
```

**Preview without creating:**

```bash
bbranch pr feature/test --dry-run
```

Output:

```
Dry run: would create PR from "feature/test" to main/default branch in:
  - api-repo (main)
  - web-repo (master)
  - worker-repo (main)
```

**Force interactive selection:**

```bash
bbranch pr feature/auth --interactive
```

---

## Configuration

### File Locations

bbranch looks for `.bbranch.yaml` in this order:

1. Current directory
2. User home directory (`~/.bbranch.yaml`)
3. Custom path via `--config` flag

### Schema

```yaml
workspace: my-workspace              # Required: Bitbucket workspace slug

# Auth method: "api_token" (default) or "oauth"
auth:
  method: api_token                  # Optional: defaults to api_token

# For API token auth
api_token:
  email: user@example.com            # Atlassian account email
  token: YOUR_API_TOKEN              # API token with repo scopes

# For OAuth auth
oauth:
  client_id: YOUR_CLIENT_ID
  client_secret: YOUR_CLIENT_SECRET

groups:                               # Optional: Named repo groups
  backend:
    - api-repo
    - worker-repo

defaults:
  source_branch: master               # Optional: Default source branch
  branch_prefix: "feature/"           # Optional: Not used by create command
```

### Environment Variables

All credential fields support `${ENV_VAR}` expansion:

```bash
export BITBUCKET_EMAIL=user@example.com
export BITBUCKET_API_TOKEN=your-token
bbranch list
```

---

## Common Workflows

### Workflow 1: Create feature branch across backend services

```bash
# One-time: add group to config
# Then run:
bbranch create feature/auth --group backend

# Or interactively:
bbranch create feature/auth
```

### Workflow 2: Create release branch from develop

```bash
bbranch create release/v1.5.0 --from develop --repos api-repo,web-repo
```

### Workflow 3: Preview before creating

```bash
# Preview
bbranch create feature/risky-change --dry-run

# If satisfied:
bbranch create feature/risky-change
```

### Workflow 4: Create pull requests across repos

```bash
# Create branches first
bbranch create feature/auth --group backend

# Then create PRs from those branches
bbranch pr feature/auth --group backend
```

Or in one group:

```bash
bbranch pr feature/auth --repos api-repo,web-repo,worker-repo
```

### Workflow 5: Create PR with custom destination

```bash
# PR from feature branch to develop (not main)
bbranch pr feature/newfeature --destination develop --repos api-repo,web-repo
```

### Workflow 6: Verify workspace setup

```bash
bbranch list
```

---

## Troubleshooting

### "Workspace not configured"

**Problem**: Error when running commands.

**Solution**: Add `workspace:` to `.bbranch.yaml`:

```yaml
workspace: your-workspace-slug
```

Find your workspace slug in Bitbucket URL: `bitbucket.org/{workspace-slug}`

---

### "api_token credentials not configured"

**Problem**: Commands fail with credentials error.

**Solution**: Set API token in `.bbranch.yaml`:

```yaml
api_token:
  email: your-email@example.com
  token: YOUR_API_TOKEN
```

Create an API token at: Bitbucket > Personal settings > Security > API tokens.
Required scopes: `read:repository:bitbucket`, `write:repository:bitbucket`.

---

### "No repositories found"

**Problem**: `bbranch list` returns empty or create fails.

**Solutions**:
1. Verify workspace slug: `bbranch list`
2. Check API token scopes: needs `read:repository:bitbucket` + `write:repository:bitbucket`
3. If using OAuth: re-authenticate with `bbranch login`

---

### "Selection cancelled"

**Problem**: Interactive select closed without choosing repos.

**Solutions**:
- Try again and use arrow keys + space + enter
- Use `--repos` flag instead: `bbranch create feature/x --repos api-repo,web-repo`
- Use `--group` flag: `bbranch create feature/x --group backend`

---

### "Port 9876 in use"

**Problem**: OAuth login fails because callback port is taken.

**Solution**: Kill process using port 9876 or restart your system.

---

## Global Flags

| Flag | Description |
|------|-------------|
| `--config` | Path to config file (default: `.bbranch.yaml` in current dir or home) |
| `--help` | Show command help |
| `--version` | Show tool version |

---

## Security Notes

- **Token storage**: `~/.bbranch/token.json` (OAuth only) is readable by your user account only
- **Credentials**: Never commit `.bbranch.yaml` with real credentials to git
- **Environment**: Use `${ENV_VAR}` expansion or environment variables in CI/CD pipelines
- **API token scope**: Use minimal scopes — only `read:repository` + `write:repository`

---

## Examples Reference

| Task | Command |
|------|---------|
| Authenticate | `bbranch login` |
| List repos | `bbranch list` |
| Create (interactive) | `bbranch create feature/auth` |
| Create (group) | `bbranch create feature/auth --group backend` |
| Create (specific repos) | `bbranch create feature/auth --repos api-repo,web-repo` |
| Create from different branch | `bbranch create release/v1 --from develop` |
| Create (preview) | `bbranch create feature/test --dry-run` |
| PR (interactive) | `bbranch pr feature/auth` |
| PR (group) | `bbranch pr feature/auth --group backend` |
| PR (specific repos) | `bbranch pr feature/auth --repos api-repo,web-repo` |
| PR (custom destination) | `bbranch pr feature/auth --destination develop` |
| PR (preview) | `bbranch pr feature/test --dry-run` |
| Custom config | `bbranch list --config ~/.bbranch-prod.yaml` |
