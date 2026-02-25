# Phase 4: Config Management

**Priority:** High | **Status:** Pending

## Overview
Viper-based config loading from `.bbranch.yaml` with env var expansion for OAuth credentials.

## Config File (`.bbranch.yaml`)
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
defaults:
  source_branch: master
  branch_prefix: "feature/"
```

## Implementation Steps

1. **Create `internal/config/config.go`**:
   - `Config` struct — Workspace, OAuth, Groups, Defaults
   - `OAuth` struct — ClientID, ClientSecret
   - `Defaults` struct — SourceBranch, BranchPrefix
   - `Load(configPath string) (*Config, error)` — Viper load + env expansion
   - `GetReposForGroup(groupName string) ([]string, error)`
   - `expandEnvVars(val string) string` — replace `${VAR}` with `os.Getenv("VAR")`

2. **Config search order** (Viper):
   - `--config` flag path (explicit)
   - `.bbranch.yaml` in current directory
   - `~/.bbranch.yaml` in home directory

3. **Validation**:
   - Workspace must be non-empty
   - OAuth client_id must resolve (after env expansion)
   - Group referenced in CLI must exist in config

## Files to Create
- `internal/config/config.go`

## Success Criteria
- [ ] Loads config from `.bbranch.yaml`
- [ ] Expands `${ENV_VAR}` in OAuth fields
- [ ] Falls back to home directory config
- [ ] Returns clear error if config missing or invalid
