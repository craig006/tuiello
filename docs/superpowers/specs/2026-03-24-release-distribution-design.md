# Release & Distribution Design

## Goal

Rename tuillo to tuiello, set up automated release infrastructure, and distribute the first public release (v0.1.0) via GitHub Releases, Homebrew, and AUR.

## Rename

The project is renamed from `tuillo` to `tuiello`. This affects:

- **Go module:** `github.com/craig006/tuillo` -> `github.com/craig006/tuiello`
- **All import paths** across the codebase (`internal/config`, `internal/tui`, `internal/trello`, `internal/commands`, `cmd`)
- **Binary name:** `tuillo` -> `tuiello` (in `.gitignore` and build output)
- **Branding:** the "tuillo" string rendered in the view bar (`internal/tui/views.go`)

## License

MIT. Standard MIT license file at the repo root with copyright holder `Craig Memory`.

## README

Minimal README for v0.1.0 with the following sections:

1. **Header** — project name, one-line description ("A TUI client for Trello"), screenshot placeholder
2. **Features** — bullet list: kanban board view, detail panel with comments/checklists, views (named filter presets), search/filter (member, label, text), custom commands, vim-style navigation, configurable keybindings and theming
3. **Installation** — three methods:
   - Homebrew: `brew install craig006/tuiello/tuiello`
   - AUR: `yay -S tuiello`
   - From source: `go install github.com/craig006/tuiello@latest`
4. **Quick Start** — get Trello API key from https://trello.com/power-ups/admin, generate token, set `TRELLO_API_KEY` and `TRELLO_TOKEN` env vars (or in config), run `tuiello --board "Board Name"`
5. **Configuration** — config file locations (`~/.config/tuiello/config.yml` and `.tuiello.yml`), brief overview of key config sections (views, keybindings, GUI settings)
6. **Keybindings** — table of default keys
7. **License** — MIT

Not included in v0.1.0 README: contributing guide, detailed theming docs, architecture docs, custom command authoring guide.

## GoReleaser Configuration

File: `.goreleaser.yaml` at repo root.

### Builds

Single build entry:
- **Binary:** `tuiello`
- **Main:** `.` (repo root, where `main.go` lives)
- **GOOS:** `linux`, `darwin`
- **GOARCH:** `amd64`, `arm64`
- **Ldflags:** `-s -w -X github.com/craig006/tuiello/internal/tui.Version={{.Version}}`

`-s -w` strips debug info for smaller binaries. `{{.Version}}` is the tag without the `v` prefix (e.g., `0.1.0`).

### Archives

Format: `.tar.gz`. Each archive contains the binary, `LICENSE`, and `README.md`.

Name template: `tuiello_{{.Version}}_{{.Os}}_{{.Arch}}`

### Changelog

Auto-generated from commit messages since the last tag. Uses GoReleaser's default grouping (features, fixes, etc. based on conventional commit prefixes).

### Homebrew Tap

GoReleaser's `brews` section:
- **Tap:** `craig006/homebrew-tuiello`
- **Description:** "A TUI client for Trello"
- **Homepage:** `https://github.com/craig006/tuiello`
- **Token:** `{{ .Env.HOMEBREW_TAP_TOKEN }}` — a GitHub PAT with write access to the tap repo

GoReleaser generates and pushes the Homebrew formula automatically on each release.

### Not Included

No Docker images, no Snapcraft, no Scoop (Windows), no Flatpak. Windows support is excluded from v0.1.0 — the TUI relies on Unix terminal semantics and has not been tested on Windows. These can be added in future releases.

## GitHub Actions Workflows

### test.yml

**Triggers:** push to `main`, pull requests targeting `main`.

**Job:** single job on `ubuntu-latest`:
1. Checkout code
2. Setup Go (version from `go.mod`)
3. Run `go test ./...`

### release.yml

**Triggers:** push of tags matching `v*`.

**Permissions:** `contents: write` (required for creating GitHub Releases and uploading assets).

**Job:** single job on `ubuntu-latest`:
1. Checkout code with full history (`fetch-depth: 0`, required by GoReleaser for changelog)
2. Setup Go (version from `go.mod`)
3. Run `go test ./...` — release aborts if tests fail
4. Run GoReleaser via `goreleaser/goreleaser-action@v6`
5. Environment: `GITHUB_TOKEN` (auto-provided), `HOMEBREW_TAP_TOKEN` (repo secret)

## Homebrew Tap

A separate GitHub repository: `craig006/homebrew-tuiello`.

Created empty. GoReleaser pushes the formula file on each release. Users install with:

```
brew install craig006/tuiello/tuiello
```

Or with an explicit tap step:

```
brew tap craig006/tuiello
brew install tuiello
```

## AUR Package

A PKGBUILD registered on the AUR as `tuiello`.

```bash
pkgname=tuiello
pkgver=0.1.0
pkgrel=1
pkgdesc="A TUI client for Trello"
arch=('x86_64' 'aarch64')
url="https://github.com/craig006/tuiello"
license=('MIT')
makedepends=('go')
source=("$pkgname-$pkgver.tar.gz::https://github.com/craig006/tuiello/archive/v$pkgver.tar.gz")
sha256sums=('SKIP')  # Replace with actual hash after first release

build() {
    cd "$pkgname-$pkgver"
    export CGO_ENABLED=0
    go build -ldflags "-s -w -X github.com/craig006/tuiello/internal/tui.Version=$pkgver" -o tuiello .
}

package() {
    cd "$pkgname-$pkgver"
    install -Dm755 tuiello "$pkgdir/usr/bin/tuiello"
    install -Dm644 LICENSE "$pkgdir/usr/share/licenses/$pkgname/LICENSE"
}
```

AUR updates are manual for v0.1.0: after each GitHub release, update `pkgver` and `sha256sums` in the PKGBUILD, regenerate `.SRCINFO` with `makepkg --printsrcinfo > .SRCINFO`, and push both files to the AUR git repo. Automation can be added later.

## Version Management

The version source of truth is the git tag. The flow:

1. Developer tags a commit on `main`: `git tag v0.1.0`
2. Pushes the tag: `git push origin v0.1.0`
3. GitHub Actions runs `release.yml`
4. GoReleaser reads the tag, injects it into the binary via ldflags
5. The binary reports the correct version in the view bar

No version file, no manual version bumping. Tags drive everything.

## Manual Setup Steps (Not Automatable)

These are one-time setup steps the maintainer performs outside the codebase:

1. **Create GitHub repo** `craig006/tuiello` (or rename existing `craig006/tuillo`)
2. **Create GitHub repo** `craig006/homebrew-tuiello` (empty, for the Homebrew tap)
3. **Create GitHub PAT** with `repo` scope for writing to the tap repo
4. **Add repo secret** `HOMEBREW_TAP_TOKEN` on the `tuiello` repo with the PAT value
5. **Register AUR package** `tuiello` — create an AUR account if needed, `git clone ssh://aur@aur.archlinux.org/tuiello.git`, add PKGBUILD, generate `.SRCINFO` with `makepkg --printsrcinfo > .SRCINFO`, push both files

## Release Checklist (Per Release)

1. Ensure `main` is clean and all tests pass
2. `git tag vX.Y.Z`
3. `git push origin vX.Y.Z`
4. Verify GitHub Actions completes successfully
5. Verify GitHub Release page has correct artifacts
6. Verify `brew install craig006/tuiello/tuiello` works
7. Update AUR PKGBUILD with new version and SHA256, regenerate `.SRCINFO`, push

## Out of Scope

- `tuiello auth` guided authentication command — separate feature, follow-up brainstorm
- Contributing guide and detailed documentation — post-v0.1.0
- AUR release automation — post-v0.1.0
- Docker, Snap, Scoop, Flatpak distribution — post-v0.1.0
