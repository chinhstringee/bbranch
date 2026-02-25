# Phase 5: List & Config Init Commands

**Priority:** Medium | **Status:** Pending

## Overview
Supporting commands: `bbranch list` to view workspace repos and `bbranch config init` for setup wizard.

## Implementation Steps

### 1. List Command (`cmd/list.go`)
```bash
bbranch list              # List all repos in workspace
bbranch list --group backend  # List repos in a group
```
- Fetch repos via `client.ListRepositories(workspace)`
- Display as table: slug, default branch, last updated
- If `--group` specified, filter to group repos and mark which exist on remote

### 2. Config Init Command (`cmd/config_init.go`)
```bash
bbranch config init
```
- Interactive wizard using `huh` library:
  1. Ask workspace name
  2. Ask auth method (env vars recommended)
  3. Fetch and display repos from workspace
  4. Ask user to create groups (multi-select repos for each group)
  5. Ask default source branch and prefix
- Write `.bbranch.yaml` to current directory

## Files to Create
- `cmd/list.go`
- `cmd/config_init.go`

## Success Criteria
- [ ] `bbranch list` shows workspace repos in table format
- [ ] `bbranch config init` creates valid `.bbranch.yaml`
- [ ] Interactive wizard completes without errors
