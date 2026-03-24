# Release & Distribution Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename tuillo to tuiello, add LICENSE/README, configure GoReleaser and GitHub Actions CI/CD for automated releases via GitHub Releases and Homebrew.

**Architecture:** Rename all Go module paths and branding from `tuillo` to `tuiello`. Add release infrastructure as config files (`.goreleaser.yaml`, `.github/workflows/`). README and LICENSE are static files at repo root.

**Tech Stack:** Go 1.26.1, GoReleaser, GitHub Actions, Homebrew tap

**Spec:** `docs/superpowers/specs/2026-03-24-release-distribution-design.md`

---

## File Structure

### Files to Create
- `LICENSE` — MIT license
- `README.md` — project README
- `.goreleaser.yaml` — GoReleaser configuration
- `.github/workflows/test.yml` — CI test workflow
- `.github/workflows/release.yml` — Release workflow

### Files to Modify (Rename: tuillo → tuiello)
- `go.mod` — module path
- `main.go` — import path
- `cmd/root.go` — import paths, `Use` field, config directory name, config file name
- `internal/config/config.go` — local config file name
- `internal/tui/app.go` — import paths
- `internal/tui/board.go` — import paths
- `internal/tui/column.go` — import paths
- `internal/tui/detail.go` — import paths
- `internal/tui/filter.go` — import paths
- `internal/tui/keys.go` — import paths
- `internal/tui/theme.go` — import paths
- `internal/tui/views.go` — import paths, branding string
- `internal/commands/custom.go` — import paths
- `internal/tui/app_test.go` — import paths
- `internal/tui/board_test.go` — import paths
- `internal/tui/column_test.go` — import paths
- `internal/tui/detail_test.go` — import paths
- `internal/tui/filter_test.go` — import paths
- `internal/tui/keys_test.go` — import paths
- `internal/tui/views_test.go` — import paths
- `internal/commands/custom_test.go` — import paths
- `.gitignore` — binary name

---

### Task 1: Rename Go Module (tuillo → tuiello)

This task renames all references to the old module path across the entire codebase.

**Files:**
- Modify: `go.mod:1` (module declaration)
- Modify: `main.go:7` (import)
- Modify: `cmd/root.go:10-13,23,30` (imports, Use field, config dir)
- Modify: `internal/config/config.go:219` (local config file name)
- Modify: All `internal/tui/*.go` files (imports)
- Modify: All `internal/commands/*.go` files (imports)
- Modify: All `*_test.go` files (imports)
- Modify: `internal/tui/views.go:112` (branding string)
- Modify: `.gitignore:1` (binary name)

- [ ] **Step 1: Update go.mod module path**

In `go.mod`, change line 1:

```
module github.com/craig006/tuillo
```

to:

```
module github.com/craig006/tuiello
```

- [ ] **Step 2: Replace all import paths across the codebase**

Run a find-and-replace across all `.go` files:

Replace `github.com/craig006/tuillo` with `github.com/craig006/tuiello`

This affects all 21 `.go` files listed above. Every import of `github.com/craig006/tuillo/internal/config`, `github.com/craig006/tuillo/internal/trello`, `github.com/craig006/tuillo/internal/tui`, `github.com/craig006/tuillo/internal/commands`, and `github.com/craig006/tuillo/cmd` must be updated.

- [ ] **Step 3: Update cobra Use field and config directory name**

In `cmd/root.go`, change line 23:

```go
Use:   "tuillo",
```

to:

```go
Use:   "tuiello",
```

Change line 30:

```go
globalDir = globalDir + "/tuillo"
```

to:

```go
globalDir = globalDir + "/tuiello"
```

- [ ] **Step 4: Update local config file name and comment**

In `internal/config/config.go`, change the comment on line 198:

```go
// Load reads config with cascade: globalDir/config.yml → projectDir/.tuillo.yml.
```

to:

```go
// Load reads config with cascade: globalDir/config.yml → projectDir/.tuiello.yml.
```

And change line 219:

```go
v.SetConfigFile(filepath.Join(projectDir, ".tuillo.yml"))
```

to:

```go
v.SetConfigFile(filepath.Join(projectDir, ".tuiello.yml"))
```

- [ ] **Step 5: Update branding string in view bar**

In `internal/tui/views.go`, change line 112:

```go
appBrand := appNameStyle.Render("tuillo") + appVerStyle.Render(" "+Version)
```

to:

```go
appBrand := appNameStyle.Render("tuiello") + appVerStyle.Render(" "+Version)
```

- [ ] **Step 6: Update .gitignore binary name**

In `.gitignore`, change:

```
tuillo
```

to:

```
tuiello
```

- [ ] **Step 7: Verify the rename compiles and tests pass**

Run:
```bash
go build -o tuiello .
go test ./...
```

Expected: Build succeeds, all tests pass. Delete the built binary after verification.

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "rename: tuillo → tuiello across all module paths, config, and branding"
```

---

### Task 2: Add MIT License

**Files:**
- Create: `LICENSE`

- [ ] **Step 1: Create LICENSE file**

Create `LICENSE` at the repo root with the standard MIT license text:

```
MIT License

Copyright (c) 2026 Craig Memory

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

- [ ] **Step 2: Commit**

```bash
git add LICENSE
git commit -m "add MIT license"
```

---

### Task 3: Add README

**Files:**
- Create: `README.md`

- [ ] **Step 1: Create README.md**

Create `README.md` at the repo root with the following content:

````markdown
# tuiello

A TUI client for Trello.

<!-- TODO: Add screenshot -->

## Features

- Kanban board view with sliding column window
- Detail panel with comments and checklists
- Views — named filter presets with keyboard shortcuts
- Search and filter by member, label, or text
- Custom commands with templated shell execution
- Vim-style navigation
- Configurable keybindings and theming

## Installation

### Homebrew

```bash
brew install craig006/tuiello/tuiello
```

### AUR (Arch Linux)

```bash
yay -S tuiello
```

### From Source

```bash
go install github.com/craig006/tuiello@latest
```

## Quick Start

1. Get your Trello API key at https://trello.com/power-ups/admin
2. Generate a token from the API key page
3. Set environment variables:

```bash
export TRELLO_API_KEY=<your-api-key>
export TRELLO_TOKEN=<your-token>
```

4. Launch:

```bash
tuiello --board "Board Name"
```

## Configuration

tuiello looks for configuration in two places (merged in order):

1. `~/.config/tuiello/config.yml` — global config
2. `.tuiello.yml` — project-local config (in current directory)

Key config sections:

```yaml
gui:
  columnWidth: 30
  showCardLabels: true
  showDetailPanel: true
  padding: 1
  theme:
    activeBorderColor: ["4", "bold"]
    inactiveBorderColor: ["8"]

views:
  - title: "My Cards"
    filter: "member:@me"
    key: "m"
  - title: "All Cards"

keybinding:
  universal:
    quit: "q"
    help: "?"
    refresh: "r"
```

## Keybindings

| Key | Action |
|-----|--------|
| `h` / `l` | Move between columns |
| `j` / `k` | Move between cards |
| `H` / `L` | Move card to adjacent column |
| `J` / `K` | Move card up/down in column |
| `d` | Toggle detail panel |
| `[` / `]` | Detail panel: previous/next tab |
| `Ctrl+j` / `Ctrl+k` | Detail panel: scroll down/up |
| `/` | Focus search bar |
| `Ctrl+m` | Filter by member (multi-select) |
| `Ctrl+l` | Filter by label (multi-select) |
| `v` / `V` | Next/previous view |
| `x` | Custom commands |
| `r` | Refresh board |
| `q` | Quit |

## License

MIT
````

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "add README for v0.1.0"
```

---

### Task 4: Add GoReleaser Configuration

**Files:**
- Create: `.goreleaser.yaml`

- [ ] **Step 1: Create .goreleaser.yaml**

Create `.goreleaser.yaml` at the repo root:

```yaml
version: 2

builds:
  - binary: tuiello
    main: .
    goos:
      - linux
      - darwin
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w -X github.com/craig006/tuiello/internal/tui.Version={{.Version}}

archives:
  - format: tar.gz
    name_template: "tuiello_{{.Version}}_{{.Os}}_{{.Arch}}"
    files:
      - LICENSE
      - README.md

changelog:
  sort: asc

brews:
  - repository:
      owner: craig006
      name: homebrew-tuiello
      token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"
    homepage: "https://github.com/craig006/tuiello"
    description: "A TUI client for Trello"
```

- [ ] **Step 2: Validate GoReleaser config (if goreleaser is installed)**

Run:
```bash
goreleaser check
```

Expected: valid configuration. If `goreleaser` is not installed locally, skip this step — the GitHub Actions workflow will validate it on the first release.

- [ ] **Step 3: Commit**

```bash
git add .goreleaser.yaml
git commit -m "add GoReleaser configuration for cross-platform releases and Homebrew tap"
```

---

### Task 5: Add GitHub Actions Workflows

**Files:**
- Create: `.github/workflows/test.yml`
- Create: `.github/workflows/release.yml`

- [ ] **Step 1: Create .github/workflows directory**

```bash
mkdir -p .github/workflows
```

- [ ] **Step 2: Create test.yml**

Create `.github/workflows/test.yml`:

```yaml
name: Test

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - run: go test ./...
```

- [ ] **Step 3: Create release.yml**

Create `.github/workflows/release.yml`:

```yaml
name: Release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - run: go test ./...

      - uses: goreleaser/goreleaser-action@v6
        with:
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}
```

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/test.yml .github/workflows/release.yml
git commit -m "add GitHub Actions workflows for CI testing and automated releases"
```

---

### Task 6: Final Verification

- [ ] **Step 1: Run full test suite**

```bash
go test ./...
```

Expected: All tests pass.

- [ ] **Step 2: Test build with ldflags**

```bash
go build -ldflags "-s -w -X github.com/craig006/tuiello/internal/tui.Version=0.1.0-test" -o tuiello .
./tuiello --help
```

Expected: Binary builds successfully. Help output shows `tuiello` as the command name. Delete the built binary after verification.

- [ ] **Step 3: Verify all files are present**

Confirm the following new files exist:
- `LICENSE`
- `README.md`
- `.goreleaser.yaml`
- `.github/workflows/test.yml`
- `.github/workflows/release.yml`

And that no file still references `tuillo` (except docs/superpowers/ plan/spec files which are historical):

```bash
grep -r "tuillo" --include="*.go" --include="*.yaml" --include="*.yml" --include="*.mod" .
```

Expected: No matches (or only matches in `docs/superpowers/` files).

- [ ] **Step 4: Clean up**

```bash
rm -f tuiello
```
