# Phase 1: Project Setup & Scaffolding

**Priority:** High | **Status:** Pending

## Overview
Initialize Go module, install dependencies, create project structure with Cobra root command.

## Implementation Steps

1. **Init Go module**
   ```bash
   go mod init github.com/stringee/git-branch-creator
   ```

2. **Install dependencies**
   ```bash
   go get github.com/spf13/cobra@latest
   go get github.com/spf13/viper@latest
   go get github.com/fatih/color@latest
   go get github.com/charmbracelet/huh@latest
   ```

3. **Create `main.go`** — calls `cmd.Execute()`

4. **Create `cmd/root.go`** — Root Cobra command with:
   - App name: `bbranch`
   - Description
   - `--config` persistent flag (config file path override)
   - Viper config initialization in `initConfig()`

5. **Create directory structure**
   ```
   cmd/
   internal/bitbucket/
   internal/config/
   internal/creator/
   ```

6. **Verify** — `go build -o bbranch . && ./bbranch --help`

## Files to Create
- `main.go`
- `cmd/root.go`
- `internal/bitbucket/` (empty, placeholder)
- `internal/config/` (empty, placeholder)
- `internal/creator/` (empty, placeholder)

## Success Criteria
- [x] `go build` compiles without errors
- [x] `./bbranch --help` shows help text
- [x] `--config` flag recognized
