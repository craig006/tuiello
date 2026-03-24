# Config, Auth & Board Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add credential storage to config files, restructure project-local config from `.tuiello.yml` to a `.tuiello/` directory, and allow board selection from config so `tuiello` can launch with no flags.

**Architecture:** Add `AuthConfig` struct to config, rewrite `Load()` to merge four files (global config, global auth, project config, project auth) via Viper's `SetConfigFile`/`MergeInConfig` loop, move credential resolution from `RunE` to `PersistentPreRunE` with env var overrides.

**Tech Stack:** Go 1.26.1, Viper (config), Cobra (CLI)

**Spec:** `docs/superpowers/specs/2026-03-24-config-auth-board-design.md`

---

## File Structure

### Files to Modify
- `internal/config/config.go` — add `AuthConfig` struct, rewrite `Load()` for four-file merge
- `internal/config/config_test.go` — update `TestCascadeProjectLocal`, add new tests for auth and four-file merge
- `cmd/root.go` — move credential resolution to `PersistentPreRunE`, use `appConfig.Auth` with env var override
- `.gitignore` — add `.tuiello/auth.yml`
- `README.md` — update Configuration section

---

### Task 1: Add AuthConfig and Rewrite Config Loader

**Files:**
- Modify: `internal/config/config.go:12-18` (add Auth field to Config struct)
- Modify: `internal/config/config.go:198-233` (rewrite Load function)
- Modify: `internal/config/config_test.go:56-85` (update TestCascadeProjectLocal)
- Test: `internal/config/config_test.go` (new tests)

- [ ] **Step 1: Write test for AuthConfig loading from auth.yml**

Add to `internal/config/config_test.go`:

```go
func TestAuthFromFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "auth.yml"), []byte(`
auth:
  apiKey: "test-key"
  token: "test-token"
`), 0644)

	cfg, err := Load(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Auth.APIKey != "test-key" {
		t.Errorf("expected apiKey 'test-key', got %q", cfg.Auth.APIKey)
	}
	if cfg.Auth.Token != "test-token" {
		t.Errorf("expected token 'test-token', got %q", cfg.Auth.Token)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestAuthFromFile -v`
Expected: FAIL — `cfg.Auth` field doesn't exist yet.

- [ ] **Step 3: Add AuthConfig struct and Auth field to Config**

In `internal/config/config.go`, add the struct after `BoardConfig`:

```go
type AuthConfig struct {
	APIKey string `mapstructure:"apiKey"`
	Token  string `mapstructure:"token"`
}
```

Add `Auth` field to the `Config` struct:

```go
type Config struct {
	Auth           AuthConfig            `mapstructure:"auth"`
	GUI            GUIConfig             `mapstructure:"gui"`
	Board          BoardConfig           `mapstructure:"board"`
	Keybinding     KeybindingConfig      `mapstructure:"keybinding"`
	CustomCommands []CustomCommandConfig `mapstructure:"customCommands"`
	Views          []ViewConfig          `mapstructure:"views"`
}
```

- [ ] **Step 4: Rewrite Load() for four-file merge**

Replace the entire `Load` function in `internal/config/config.go`:

```go
// Load reads config with cascade: globalDir/config.yml → globalDir/auth.yml →
// projectDir/.tuiello/config.yml → projectDir/.tuiello/auth.yml.
// Either path can be empty to skip that layer.
func Load(globalDir, projectDir string) (Config, error) {
	cfg := DefaultConfig()

	v := viper.New()
	v.SetConfigType("yaml")

	var files []string
	if globalDir != "" {
		files = append(files,
			filepath.Join(globalDir, "config.yml"),
			filepath.Join(globalDir, "auth.yml"),
		)
	}
	if projectDir != "" {
		files = append(files,
			filepath.Join(projectDir, ".tuiello", "config.yml"),
			filepath.Join(projectDir, ".tuiello", "auth.yml"),
		)
	}

	for _, f := range files {
		v.SetConfigFile(f)
		if err := v.MergeInConfig(); err != nil {
			if !os.IsNotExist(err) {
				if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
					return cfg, err
				}
			}
		}
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/config/ -run TestAuthFromFile -v`
Expected: PASS

- [ ] **Step 6: Write test for four-file merge order**

Add to `internal/config/config_test.go`:

```go
func TestFourFileMergeOrder(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()

	// Global config sets auth and board
	os.WriteFile(filepath.Join(globalDir, "config.yml"), []byte(`
board:
  id: "global-board"
gui:
  columnWidth: 25
`), 0644)

	os.WriteFile(filepath.Join(globalDir, "auth.yml"), []byte(`
auth:
  apiKey: "global-key"
  token: "global-token"
`), 0644)

	// Project config overrides board
	projectCfgDir := filepath.Join(projectDir, ".tuiello")
	os.MkdirAll(projectCfgDir, 0755)

	os.WriteFile(filepath.Join(projectCfgDir, "config.yml"), []byte(`
board:
  id: "project-board"
`), 0644)

	// Project auth overrides token but not apiKey
	os.WriteFile(filepath.Join(projectCfgDir, "auth.yml"), []byte(`
auth:
  token: "project-token"
`), 0644)

	cfg, err := Load(globalDir, projectDir)
	if err != nil {
		t.Fatal(err)
	}

	// Board overridden by project config
	if cfg.Board.ID != "project-board" {
		t.Errorf("expected project-board, got %q", cfg.Board.ID)
	}
	// GUI preserved from global (project didn't override)
	if cfg.GUI.ColumnWidth != 25 {
		t.Errorf("expected columnWidth 25, got %d", cfg.GUI.ColumnWidth)
	}
	// APIKey from global auth (project auth didn't override)
	if cfg.Auth.APIKey != "global-key" {
		t.Errorf("expected global-key, got %q", cfg.Auth.APIKey)
	}
	// Token overridden by project auth
	if cfg.Auth.Token != "project-token" {
		t.Errorf("expected project-token, got %q", cfg.Auth.Token)
	}
}
```

- [ ] **Step 7: Run test to verify it passes**

Run: `go test ./internal/config/ -run TestFourFileMergeOrder -v`
Expected: PASS

- [ ] **Step 8: Write test for missing files silently skipped**

Add to `internal/config/config_test.go`:

```go
func TestMissingFilesSkipped(t *testing.T) {
	// Both dirs exist but contain no config files
	globalDir := t.TempDir()
	projectDir := t.TempDir()

	cfg, err := Load(globalDir, projectDir)
	if err != nil {
		t.Fatalf("expected no error with missing files, got %v", err)
	}
	// Defaults should still apply
	if cfg.GUI.ColumnWidth != 30 {
		t.Errorf("expected default columnWidth 30, got %d", cfg.GUI.ColumnWidth)
	}
	if cfg.Auth.APIKey != "" {
		t.Errorf("expected empty apiKey, got %q", cfg.Auth.APIKey)
	}
}
```

- [ ] **Step 9: Run test to verify it passes**

Run: `go test ./internal/config/ -run TestMissingFilesSkipped -v`
Expected: PASS

- [ ] **Step 10: Update TestCascadeProjectLocal for new directory structure**

Replace the existing `TestCascadeProjectLocal` in `internal/config/config_test.go`:

```go
func TestCascadeProjectLocal(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()

	// global config sets board id
	os.WriteFile(filepath.Join(globalDir, "config.yml"), []byte(`
board:
  id: "global-board"
gui:
  columnWidth: 25
`), 0644)

	// project-local overrides board id
	projectCfgDir := filepath.Join(projectDir, ".tuiello")
	os.MkdirAll(projectCfgDir, 0755)
	os.WriteFile(filepath.Join(projectCfgDir, "config.yml"), []byte(`
board:
  id: "project-board"
`), 0644)

	cfg, err := Load(globalDir, projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Board.ID != "project-board" {
		t.Errorf("expected project-board, got %q", cfg.Board.ID)
	}
	// global columnWidth preserved since project didn't override
	if cfg.GUI.ColumnWidth != 25 {
		t.Errorf("expected columnWidth 25, got %d", cfg.GUI.ColumnWidth)
	}
}
```

- [ ] **Step 11: Run all config tests**

Run: `go test ./internal/config/ -v`
Expected: All tests PASS.

- [ ] **Step 12: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat: add AuthConfig, rewrite config loader for four-file merge with .tuiello/ directory"
```

---

### Task 2: Move Credential Resolution to PersistentPreRunE

**Files:**
- Modify: `cmd/root.go:25-73` (PersistentPreRunE and RunE)

- [ ] **Step 1: Update PersistentPreRunE to resolve auth from config + env vars**

In `cmd/root.go`, add the env var override logic at the end of `PersistentPreRunE` (after the existing CLI flag overrides, around line 45):

```go
		// Env vars override config auth
		if envKey := os.Getenv("TRELLO_API_KEY"); envKey != "" {
			appConfig.Auth.APIKey = envKey
		}
		if envToken := os.Getenv("TRELLO_TOKEN"); envToken != "" {
			appConfig.Auth.Token = envToken
		}

		return nil
```

- [ ] **Step 2: Update RunE to use appConfig.Auth instead of reading env vars directly**

Replace the credential section in `RunE` (lines 49-61) with:

```go
	RunE: func(cmd *cobra.Command, args []string) error {
		if appConfig.Auth.APIKey == "" || appConfig.Auth.Token == "" {
			return fmt.Errorf("missing Trello credentials.\n\n" +
				"Set credentials in ~/.config/tuiello/auth.yml:\n" +
				"  auth:\n" +
				"    apiKey: <your-api-key>\n" +
				"    token: <your-token>\n\n" +
				"Or set environment variables:\n" +
				"  export TRELLO_API_KEY=<your-api-key>\n" +
				"  export TRELLO_TOKEN=<your-token>\n\n" +
				"Get your API key at: https://trello.com/power-ups/admin")
		}

		client := trello.NewClient(appConfig.Auth.APIKey, appConfig.Auth.Token)

		if err := client.ValidateCredentials(); err != nil {
			return fmt.Errorf("invalid credentials: %w", err)
		}

		app := tui.NewApp(client, appConfig)
		p := tea.NewProgram(app)
		_, err := p.Run()
		return err
	},
```

- [ ] **Step 3: Run all tests**

Run: `go test ./...`
Expected: All tests PASS. (No behavioral test changes here — this is a wiring change.)

- [ ] **Step 4: Commit**

```bash
git add cmd/root.go
git commit -m "feat: resolve credentials from config with env var override in PersistentPreRunE"
```

---

### Task 3: Update .gitignore and README

**Files:**
- Modify: `.gitignore`
- Modify: `README.md`

- [ ] **Step 1: Add .tuiello/auth.yml to .gitignore**

Add to `.gitignore`:

```
.tuiello/auth.yml
```

The full `.gitignore` should now be:

```
tuiello
.superpowers/
.worktrees/
.tuiello/auth.yml
```

- [ ] **Step 2: Update README.md Configuration section**

Replace the Configuration section in `README.md` (between `## Configuration` and `## Keybindings`) with:

````markdown
## Configuration

tuiello uses a two-level config system. Each level has a settings file and an optional auth file:

```
~/.config/tuiello/          # global
├── config.yml              # settings
└── auth.yml                # credentials

<project>/.tuiello/         # project-local (overrides global)
├── config.yml              # project settings
└── auth.yml                # project credentials
```

All files are optional. Values merge in order: global config → global auth → project config → project auth. Environment variables and CLI flags override everything.

### Credentials

Set your Trello credentials in `auth.yml`:

```yaml
auth:
  apiKey: your-api-key
  token: your-token
```

Or use environment variables: `TRELLO_API_KEY` and `TRELLO_TOKEN`.

### Board

Set a default board so you can launch with just `tuiello`:

```yaml
board:
  name: "My Board"
```

Or by ID:

```yaml
board:
  id: "abc123"
```

CLI flags `--board` and `--board-id` override config values.

### Settings

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
````

- [ ] **Step 3: Run all tests**

Run: `go test ./...`
Expected: All tests PASS (no code changes, just docs).

- [ ] **Step 4: Commit**

```bash
git add .gitignore README.md
git commit -m "docs: update .gitignore and README for new config directory structure and auth"
```

---

### Task 4: Final Verification

- [ ] **Step 1: Run full test suite**

Run: `go test ./...`
Expected: All tests pass.

- [ ] **Step 2: Build and verify help output**

Run: `go build -o tuiello . && ./tuiello --help`
Expected: Shows `tuiello` usage with `--board` and `--board-id` flags.

- [ ] **Step 3: Verify error message without credentials**

Run: `TRELLO_API_KEY= TRELLO_TOKEN= ./tuiello`
Expected: Error message mentions both `auth.yml` and env var options.

- [ ] **Step 4: Clean up**

```bash
rm -f tuiello
```
