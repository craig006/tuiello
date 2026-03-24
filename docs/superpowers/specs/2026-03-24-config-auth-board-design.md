# Config, Auth & Board Design

## Goal

Add credential storage to config files, restructure project-local config from a single file to a directory, and allow board selection from config so launching tuiello requires no flags.

## Config Directory Structure

Project-local config changes from a single `.tuiello.yml` file to a `.tuiello/` directory. The old `.tuiello.yml` format is dropped (pre-release, no backwards compatibility needed).

```
~/.config/tuiello/          # global
├── config.yml              # settings (GUI, keybindings, views, board, etc.)
└── auth.yml                # credentials (apiKey, token)

<project>/.tuiello/         # project-local
├── config.yml              # project settings overrides
└── auth.yml                # project credential overrides
```

All files are optional. Missing files are silently skipped.

### Merge Order

Each layer overrides the previous:

1. `DefaultConfig()` (hardcoded defaults)
2. `~/.config/tuiello/config.yml`
3. `~/.config/tuiello/auth.yml`
4. `.tuiello/config.yml`
5. `.tuiello/auth.yml`
6. Environment variables (`TRELLO_API_KEY`, `TRELLO_TOKEN`)
7. CLI flags (`--board`, `--board-id`)

## Auth Config

New `AuthConfig` struct:

```go
type AuthConfig struct {
    APIKey string `mapstructure:"apiKey"`
    Token  string `mapstructure:"token"`
}
```

Added to the top-level `Config` struct as `Auth AuthConfig `mapstructure:"auth"``). No defaults — both fields are empty strings by default.

### auth.yml Format

```yaml
auth:
  apiKey: "your-trello-api-key"
  token: "your-trello-token"
```

### Credential Resolution in root.go

All credential and board resolution happens in `PersistentPreRunE` so that any future subcommands also have access to the resolved config.

1. Config cascade fills `appConfig.Auth.APIKey` and `appConfig.Auth.Token`
2. Env vars override: if `TRELLO_API_KEY` is set, it wins; same for `TRELLO_TOKEN`
3. `RunE` reads the final `appConfig.Auth` values — if both are still empty, show error with instructions for both config and env var options

Env vars remain supported for backwards compatibility and for CI/scripting use cases.

## Board Config

`BoardConfig` already exists with `ID` and `Name` fields — no struct changes needed. It now gets populated from config files in addition to CLI flags.

### config.yml Format

```yaml
board:
  name: "My Project Board"
  id: "abc123def456"
```

Either field can be set independently. Both don't need to be set.

### Board Resolution Order

1. Config cascade fills `appConfig.Board.ID` and `appConfig.Board.Name`
2. CLI flags override: `--board-id` overrides `Board.ID`, `--board` overrides `Board.Name`
3. Board lookup: if `ID` is set, fetch by ID; if ID is not set (or returns a "not found" error) and `Name` is set, search by name; if neither is set, show an error. Hard failures (network errors, auth errors) are not retried with name fallback — only "not found" triggers fallback.

## Config Loader Changes

The `Load` function signature stays the same:

```go
func Load(globalDir, projectDir string) (Config, error)
```

Internally it merges four files instead of two. Each file is loaded via `v.SetConfigFile(absolutePath)` followed by `v.MergeInConfig()`:

```go
files := []string{
    filepath.Join(globalDir, "config.yml"),
    filepath.Join(globalDir, "auth.yml"),
    filepath.Join(projectDir, ".tuiello", "config.yml"),
    filepath.Join(projectDir, ".tuiello", "auth.yml"),
}
for _, f := range files {
    v.SetConfigFile(f)
    if err := v.MergeInConfig(); err != nil {
        // skip missing files, return real errors
    }
}
```

Since Viper merges at the key level, `auth.apiKey` in `auth.yml` and `board.name` in `config.yml` don't clobber each other.

The existing test `TestCascadeProjectLocal` writes `.tuiello.yml` and must be updated to write `.tuiello/config.yml` instead.

## .gitignore Update

Add `.tuiello/auth.yml` to `.gitignore` to prevent accidental credential commits:

```
.tuiello/auth.yml
```

This gitignores only the auth file — `.tuiello/config.yml` remains committable so teams can share board/GUI settings.

## Error Message Update

The credential error in `root.go` is updated to mention config files as an alternative to env vars:

```
Missing Trello credentials.

Set credentials in ~/.config/tuiello/auth.yml:
  auth:
    apiKey: <your-api-key>
    token: <your-token>

Or set environment variables:
  export TRELLO_API_KEY=<your-api-key>
  export TRELLO_TOKEN=<your-token>

Get your API key at: https://trello.com/power-ups/admin
```

## README Update

The Configuration section of README.md is updated to document:

- The new directory structure (global and project-local)
- The auth.yml file and credential options
- Board config in config.yml
- The full resolution order (config → env → CLI flags)

## Scope

### Included
- `AuthConfig` struct with `apiKey` and `token`
- Project-local config directory (`.tuiello/`) replacing `.tuiello.yml`
- `auth.yml` at both global and project levels
- Four-file merge in `Load()`
- Env var override of config auth in `root.go`
- Board config from files (existing struct, existing lookup logic)
- Updated error message
- Updated README

### Test Cases

- Four-file merge: global config → global auth → project config → project auth, each layer overrides the previous
- Missing files silently skipped (no error when any of the four files is absent)
- Auth from `auth.yml` populates `Config.Auth.APIKey` and `Config.Auth.Token`
- Project auth overrides global auth
- Env vars override config-file auth values
- CLI flags override config-file board values
- Existing `TestCascadeProjectLocal` updated for `.tuiello/config.yml` directory structure

### Not Included
- `tuiello auth` guided setup command (separate feature, out of scope)
- Encrypted credential storage
- Keychain/keyring integration
- Config file generation or scaffolding commands
