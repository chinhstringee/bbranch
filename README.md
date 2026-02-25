# bbranch

CLI tool for creating Git branches across multiple Bitbucket Cloud repositories simultaneously.

## Features

- **Multi-repo branch creation** — Create the same branch across many repos in parallel
- **OAuth 2.0 with PKCE** — Secure browser-based authentication
- **Repository groups** — Define named groups in config for quick targeting
- **Interactive selection** — Pick repos from a TUI multi-select when no flags given
- **Dry run** — Preview what would happen without making changes

## Quick Start

### Install

```bash
# Homebrew (macOS/Linux)
brew tap chinhstringee/tap
brew install bbranch

# Or with Go
go install github.com/chinhstringee/bbranch@latest
```

### Prerequisites

- A [Bitbucket OAuth consumer](https://support.atlassian.com/bitbucket-cloud/docs/use-oauth-on-bitbucket-cloud/) with callback URL `http://localhost:9876/callback`

### Configure

Copy the example config and fill in your details:

```bash
cp .bbranch.example.yaml .bbranch.yaml
```

```yaml
workspace: my-workspace

oauth:
  client_id: ${BITBUCKET_OAUTH_CLIENT_ID}
  client_secret: ${BITBUCKET_OAUTH_CLIENT_SECRET}

groups:
  backend:
    - repo-api
    - repo-worker
  frontend:
    - repo-web
    - repo-mobile

defaults:
  source_branch: master
```

OAuth fields support `${ENV_VAR}` expansion.

### Authenticate

```bash
bbranch login
```

Opens your browser for OAuth authorization. Token is saved to `~/.bbranch/token.json`.

## Usage

### List repositories

```bash
bbranch list
```

### Create branches

```bash
# Interactive repo selection (default)
bbranch create feature/auth

# Using a config group
bbranch create feature/auth --group backend

# Specific repos (supports fuzzy matching)
bbranch create bugfix/cors --repos "api stringeex,subscription"

# From a different source branch
bbranch create release/v2.0 --from develop

# Preview without creating
bbranch create feature/test --dry-run
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--group` | `-g` | Use a predefined repo group from config |
| `--repos` | `-r` | Comma-separated patterns (fuzzy match, space = AND) |
| `--from` | `-f` | Source branch (overrides config default) |
| `--dry-run` | | Preview without executing |
| `--interactive` | `-i` | Force interactive selection |
| `--config` | | Custom config file path |

## License

MIT
