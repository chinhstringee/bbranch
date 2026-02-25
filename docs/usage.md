# bbranch Usage Guide

Create git branches across multiple Bitbucket Cloud repositories simultaneously using OAuth 2.0 authentication.

## Quick Start

### 1. Install Prerequisites

- **Go 1.26 or later** — [Download Go](https://golang.org/dl/)
- **Bitbucket Cloud workspace** — [Create one](https://bitbucket.org)
- **OAuth consumer** — Configure in workspace settings

### 2. Create OAuth Consumer

1. Go to Bitbucket workspace → Settings → API → OAuth consumers
2. Click "Add consumer"
3. Configure:
   - **Name**: `bbranch`
   - **Callback URL**: `http://localhost:9876/callback`
   - **Permissions**: Repositories (Read + Write)
4. Save client ID and secret

### 3. Build and Configure

```bash
# Build the tool
go build -o bbranch

# Create config file
cp .bbranch.example.yaml .bbranch.yaml
```

Edit `.bbranch.yaml`:

```yaml
workspace: your-workspace-slug

oauth:
  client_id: YOUR_CLIENT_ID
  client_secret: YOUR_CLIENT_SECRET

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

### 4. Authenticate

```bash
bbranch login
```

Opens your browser to authorize. Token saved to `~/.bbranch/token.json`.

## Commands

### `bbranch login`

Authenticate with Bitbucket via OAuth 2.0 browser flow.

```bash
bbranch login
```

**What it does:**
- Opens browser for OAuth authorization
- Stores token in `~/.bbranch/token.json`
- Token reused for all subsequent commands

**Note**: Run when token expires or you need to switch accounts.

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

## Configuration

### File Locations

bbranch looks for `.bbranch.yaml` in this order:

1. Current directory
2. User home directory (`~/.bbranch.yaml`)
3. Custom path via `--config` flag

### Schema

```yaml
workspace: my-workspace              # Required: Bitbucket workspace slug

oauth:
  client_id: YOUR_CLIENT_ID           # Required: OAuth consumer ID
  client_secret: YOUR_CLIENT_SECRET   # Required: OAuth consumer secret

groups:                               # Optional: Named repo groups
  backend:
    - api-repo
    - worker-repo
  frontend:
    - web-repo
    - mobile-repo

defaults:
  source_branch: master               # Optional: Default source branch
  branch_prefix: "feature/"           # Optional: Not used by create command
```

### Environment Variables

Override config values using environment variables:

```bash
BITBUCKET_OAUTH_CLIENT_ID=xxx bbranch list
BITBUCKET_OAUTH_CLIENT_SECRET=yyy bbranch create feature/x
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

### Workflow 4: Verify workspace setup

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

### "OAuth credentials not configured"

**Problem**: Login fails with credentials error.

**Solution**: Set OAuth credentials in `.bbranch.yaml`:

```yaml
oauth:
  client_id: YOUR_CLIENT_ID
  client_secret: YOUR_CLIENT_SECRET
```

Or use environment variables:

```bash
export BITBUCKET_OAUTH_CLIENT_ID=xxx
export BITBUCKET_OAUTH_CLIENT_SECRET=yyy
bbranch login
```

---

### "No repositories found"

**Problem**: `bbranch list` returns empty or create fails.

**Solutions**:
1. Verify workspace slug: `bbranch list`
2. Check OAuth permissions: Consumer needs Repositories (Read + Write)
3. Re-authenticate: `bbranch login`
4. Verify OAuth consumer callback URL is `http://localhost:9876/callback`

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

- **Token storage**: `~/.bbranch/token.json` is readable by your user account only
- **Credentials**: Never commit `.bbranch.yaml` with real credentials to git
- **Environment**: Use environment variables in CI/CD pipelines instead of config files
- **OAuth scope**: Consumer only has Repositories (Read + Write) access—no admin rights

---

## Examples Reference

| Task | Command |
|------|---------|
| Authenticate | `bbranch login` |
| List repos | `bbranch list` |
| Create (interactive) | `bbranch create feature/auth` |
| Create (group) | `bbranch create feature/auth --group backend` |
| Create (specific repos) | `bbranch create feature/auth --repos api-repo,web-repo` |
| From different branch | `bbranch create release/v1 --from develop` |
| Preview | `bbranch create feature/test --dry-run` |
| Custom config | `bbranch list --config ~/.bbranch-prod.yaml` |
