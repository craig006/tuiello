# tuillo Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a TUI Trello board client in Go that displays a kanban board, supports card movement, and allows user-defined custom commands.

**Architecture:** Bubble Tea v2 app with a root model managing a board view composed of column models (each wrapping bubbles/list). Trello API calls are async via tea.Cmd. Config is YAML with cascading load (global → project-local → CLI flags) via Viper.

**Tech Stack:** Go, Bubble Tea v2 (`charm.land/bubbletea/v2`), Bubbles v2 (`charm.land/bubbles/v2`), Lip Gloss v2 (`charm.land/lipgloss/v2`), Cobra, Viper. Note: the Trello API client is implemented directly (not using adlio/trello) to keep the dependency tree minimal and give full control over request/response handling.

**Spec:** `docs/superpowers/specs/2026-03-23-tuillo-tui-trello-client-design.md`

---

## File Map

| File | Responsibility |
|------|----------------|
| `main.go` | Entry point, calls `cmd.Execute()` |
| `cmd/root.go` | Cobra root command, config init, credential validation, launches TUI |
| `internal/config/config.go` | Config structs, defaults, cascade loading (global → local → flags) |
| `internal/config/config_test.go` | Config loading, cascade, defaults |
| `internal/trello/client.go` | Trello API wrapper, returns tea.Cmd-compatible functions |
| `internal/trello/client_test.go` | Client tests with HTTP mock server |
| `internal/trello/types.go` | Board, List, Card domain types (decoupled from adlio/trello) |
| `internal/tui/app.go` | Root Bubble Tea model, status bar, help overlay, message routing |
| `internal/tui/app_test.go` | Root model tests |
| `internal/tui/board.go` | Board model: 3-column sliding window, column navigation, card move dispatch |
| `internal/tui/board_test.go` | Board model tests: window sliding, navigation, move logic |
| `internal/tui/column.go` | Column model: wraps bubbles/list, card items, up/down nav |
| `internal/tui/column_test.go` | Column model tests |
| `internal/tui/keys.go` | KeyMap struct, default bindings, config-driven overrides |
| `internal/tui/keys_test.go` | Keybinding tests |
| `internal/tui/theme.go` | Theme struct, ANSI defaults, config-driven overrides, Lip Gloss styles |
| `internal/commands/custom.go` | Custom command engine: template parsing, execution, prompts |
| `internal/commands/custom_test.go` | Custom command tests |

---

### Task 1: Project Scaffolding & Go Module Init

**Files:**
- Create: `main.go`
- Create: `cmd/root.go`
- Create: `go.mod`

- [ ] **Step 1: Initialize Go module**

Run:
```bash
cd /Users/craig/GitHub/craig006/tuillo/main
go mod init github.com/craig006/tuillo
```

- [ ] **Step 2: Write main.go**

```go
// main.go
package main

import (
	"os"

	"github.com/craig006/tuillo/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Write cmd/root.go with minimal Cobra command**

```go
// cmd/root.go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	boardFlag   string
	boardIDFlag string
)

var rootCmd = &cobra.Command{
	Use:   "tuillo",
	Short: "TUI client for Trello boards",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("tuillo — TUI Trello client")
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&boardFlag, "board", "", "Trello board name")
	rootCmd.PersistentFlags().StringVar(&boardIDFlag, "board-id", "", "Trello board ID")
}

func Execute() error {
	return rootCmd.Execute()
}
```

- [ ] **Step 4: Install dependencies and verify build**

Run:
```bash
go get github.com/spf13/cobra
go mod tidy
go build ./...
```
Expected: clean build, no errors.

- [ ] **Step 5: Verify the binary runs**

Run:
```bash
go run . --help
```
Expected: shows usage with `--board` and `--board-id` flags.

- [ ] **Step 6: Commit**

```bash
git add main.go cmd/ go.mod go.sum
git commit -m "feat: scaffold project with Go module and Cobra CLI"
```

---

### Task 2: Configuration System

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Modify: `cmd/root.go`

- [ ] **Step 1: Write config tests**

```go
// internal/config/config_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.GUI.ColumnWidth != 30 {
		t.Errorf("expected columnWidth 30, got %d", cfg.GUI.ColumnWidth)
	}
	if cfg.GUI.ShowCardLabels != true {
		t.Error("expected showCardLabels true")
	}
	if cfg.Keybinding.Universal.Quit != "q" {
		t.Errorf("expected quit key 'q', got %q", cfg.Keybinding.Universal.Quit)
	}
	if cfg.Keybinding.Board.MoveLeft != "h" {
		t.Errorf("expected moveLeft 'h', got %q", cfg.Keybinding.Board.MoveLeft)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")
	err := os.WriteFile(cfgPath, []byte(`
board:
  id: "abc123"
gui:
  columnWidth: 40
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Board.ID != "abc123" {
		t.Errorf("expected board id 'abc123', got %q", cfg.Board.ID)
	}
	if cfg.GUI.ColumnWidth != 40 {
		t.Errorf("expected columnWidth 40, got %d", cfg.GUI.ColumnWidth)
	}
	// defaults still apply for unset fields
	if cfg.Keybinding.Universal.Quit != "q" {
		t.Errorf("expected quit key 'q', got %q", cfg.Keybinding.Universal.Quit)
	}
}

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
	os.WriteFile(filepath.Join(projectDir, ".tuillo.yml"), []byte(`
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

- [ ] **Step 2: Run tests to verify they fail**

Run:
```bash
go test ./internal/config/ -v
```
Expected: compilation errors (package doesn't exist yet).

- [ ] **Step 3: Write config.go**

```go
// internal/config/config.go
package config

import (
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	GUI            GUIConfig            `mapstructure:"gui"`
	Board          BoardConfig          `mapstructure:"board"`
	Keybinding     KeybindingConfig     `mapstructure:"keybinding"`
	CustomCommands []CustomCommandConfig `mapstructure:"customCommands"`
}

type GUIConfig struct {
	Theme          ThemeConfig `mapstructure:"theme"`
	ColumnWidth    int         `mapstructure:"columnWidth"`
	ShowCardLabels bool        `mapstructure:"showCardLabels"`
}

type ThemeConfig struct {
	ActiveBorderColor   []string `mapstructure:"activeBorderColor"`
	InactiveBorderColor []string `mapstructure:"inactiveBorderColor"`
	SelectedCardColor   []string `mapstructure:"selectedCardColor"`
	ColumnTitleColor    []string `mapstructure:"columnTitleColor"`
}

type BoardConfig struct {
	ID   string `mapstructure:"id"`
	Name string `mapstructure:"name"`
}

type KeybindingConfig struct {
	Universal UniversalKeys `mapstructure:"universal"`
	Board     BoardKeys     `mapstructure:"board"`
}

type UniversalKeys struct {
	Quit    string `mapstructure:"quit"`
	Help    string `mapstructure:"help"`
	Refresh string `mapstructure:"refresh"`
}

type BoardKeys struct {
	MoveLeft      string `mapstructure:"moveLeft"`
	MoveRight     string `mapstructure:"moveRight"`
	MoveUp        string `mapstructure:"moveUp"`
	MoveDown      string `mapstructure:"moveDown"`
	MoveCardLeft  string `mapstructure:"moveCardLeft"`
	MoveCardRight string `mapstructure:"moveCardRight"`
	MoveCardUp    string `mapstructure:"moveCardUp"`
	MoveCardDown  string `mapstructure:"moveCardDown"`
	Enter         string `mapstructure:"enter"`
	CustomCommand string `mapstructure:"customCommand"`
}

type CustomCommandConfig struct {
	Key         string         `mapstructure:"key"`
	Description string         `mapstructure:"description"`
	Command     string         `mapstructure:"command"`
	Context     string         `mapstructure:"context"`
	Output      string         `mapstructure:"output"`
	Prompts     []PromptConfig `mapstructure:"prompts"`
}

type PromptConfig struct {
	Type    string         `mapstructure:"type"`
	Title   string         `mapstructure:"title"`
	Key     string         `mapstructure:"key"`
	Options []OptionConfig `mapstructure:"options"`
}

type OptionConfig struct {
	Name  string `mapstructure:"name"`
	Value string `mapstructure:"value"`
}

func DefaultConfig() Config {
	return Config{
		GUI: GUIConfig{
			Theme: ThemeConfig{
				ActiveBorderColor:   []string{"green", "bold"},
				InactiveBorderColor: []string{"240"},
				SelectedCardColor:   []string{"cyan"},
				ColumnTitleColor:    []string{"magenta", "bold"},
			},
			ColumnWidth:    30,
			ShowCardLabels: true,
		},
		Keybinding: KeybindingConfig{
			Universal: UniversalKeys{
				Quit:    "q",
				Help:    "?",
				Refresh: "r",
			},
			Board: BoardKeys{
				MoveLeft:      "h",
				MoveRight:     "l",
				MoveUp:        "k",
				MoveDown:      "j",
				MoveCardLeft:  "H",
				MoveCardRight: "L",
				MoveCardUp:    "K",
				MoveCardDown:  "J",
				Enter:         "enter",
				CustomCommand: "x",
			},
		},
	}
}

// Load reads config with cascade: globalDir/config.yml → projectDir/.tuillo.yml.
// Either path can be empty to skip that layer.
func Load(globalDir, projectDir string) (Config, error) {
	cfg := DefaultConfig()

	v := viper.New()
	v.SetConfigType("yaml")

	// Load global config
	if globalDir != "" {
		v.SetConfigName("config")
		v.AddConfigPath(globalDir)
		if err := v.MergeInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return cfg, err
			}
		}
	}

	// Load project-local config
	if projectDir != "" {
		v.SetConfigFile(filepath.Join(projectDir, ".tuillo.yml"))
		if err := v.MergeInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return cfg, err
			}
		}
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}
```

- [ ] **Step 4: Install viper and run tests**

Run:
```bash
go get github.com/spf13/viper
go test ./internal/config/ -v
```
Expected: all 3 tests pass.

- [ ] **Step 5: Wire config loading into cmd/root.go**

Update `cmd/root.go` to load config in `PersistentPreRunE`:

```go
// cmd/root.go
package cmd

import (
	"fmt"
	"os"

	"github.com/craig006/tuillo/internal/config"
	"github.com/spf13/cobra"
)

var (
	boardFlag   string
	boardIDFlag string
	appConfig   config.Config
)

var rootCmd = &cobra.Command{
	Use:   "tuillo",
	Short: "TUI client for Trello boards",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		globalDir, err := os.UserConfigDir()
		if err != nil {
			globalDir = ""
		} else {
			globalDir = globalDir + "/tuillo"
		}

		cwd, _ := os.Getwd()
		appConfig, err = config.Load(globalDir, cwd)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		// CLI flags override config
		if boardIDFlag != "" {
			appConfig.Board.ID = boardIDFlag
		}
		if boardFlag != "" {
			appConfig.Board.Name = boardFlag
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Board ID: %q, Board Name: %q\n", appConfig.Board.ID, appConfig.Board.Name)
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&boardFlag, "board", "", "Trello board name")
	rootCmd.PersistentFlags().StringVar(&boardIDFlag, "board-id", "", "Trello board ID")
}

func Execute() error {
	return rootCmd.Execute()
}
```

- [ ] **Step 6: Verify build and manual test**

Run:
```bash
go build ./...
go run . --board-id "test123"
```
Expected: prints `Board ID: "test123", Board Name: ""`

- [ ] **Step 7: Commit**

```bash
git add internal/config/ cmd/root.go
git commit -m "feat: add config system with cascade loading and CLI flag overrides"
```

---

### Task 3: Trello Domain Types

**Files:**
- Create: `internal/trello/types.go`

- [ ] **Step 1: Write domain types**

These are our own types, decoupled from adlio/trello, so the TUI layer doesn't depend on the API library.

```go
// internal/trello/types.go
package trello

// Board represents a Trello board with its lists.
type Board struct {
	ID   string
	Name string
	URL  string
	Lists []List
}

// List represents a column on a Trello board.
type List struct {
	ID    string
	Name  string
	Pos   float64
	Cards []Card
}

// Card represents a card on a Trello board.
type Card struct {
	ID          string
	Name        string
	Description string
	Pos         float64
	URL         string
	Labels      []Label
	MemberIDs   []string
	ListID      string
}

// Label represents a Trello label.
type Label struct {
	ID    string
	Name  string
	Color string
}
```

- [ ] **Step 2: Verify build**

Run:
```bash
go build ./internal/trello/
```
Expected: clean build.

- [ ] **Step 3: Commit**

```bash
git add internal/trello/types.go
git commit -m "feat: add Trello domain types"
```

---

### Task 4: Trello API Client

**Files:**
- Create: `internal/trello/client.go`
- Create: `internal/trello/client_test.go`

- [ ] **Step 1: Write client tests using httptest**

```go
// internal/trello/client_test.go
package trello

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/members/me" {
			json.NewEncoder(w).Encode(map[string]string{"id": "user1", "username": "testuser"})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c := NewClient("testkey", "testtoken")
	c.BaseURL = server.URL

	err := c.ValidateCredentials()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateCredentialsUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	c := NewClient("badkey", "badtoken")
	c.BaseURL = server.URL

	err := c.ValidateCredentials()
	if err == nil {
		t.Fatal("expected error for unauthorized credentials")
	}
}

func TestFetchBoard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/boards/board1" {
			resp := map[string]interface{}{
				"id":   "board1",
				"name": "Test Board",
				"url":  "https://trello.com/b/board1",
				"lists": []map[string]interface{}{
					{
						"id": "list1", "name": "Backlog", "pos": 1.0,
						"cards": []map[string]interface{}{
							{"id": "card1", "name": "Card One", "pos": 1.0, "idList": "list1", "url": "https://trello.com/c/card1", "labels": []interface{}{}},
							{"id": "card2", "name": "Card Two", "pos": 2.0, "idList": "list1", "url": "https://trello.com/c/card2", "labels": []interface{}{}},
						},
					},
					{
						"id": "list2", "name": "Done", "pos": 2.0,
						"cards": []map[string]interface{}{},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c := NewClient("testkey", "testtoken")
	c.BaseURL = server.URL

	board, err := c.FetchBoard("board1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if board.Name != "Test Board" {
		t.Errorf("expected 'Test Board', got %q", board.Name)
	}
	if len(board.Lists) != 2 {
		t.Fatalf("expected 2 lists, got %d", len(board.Lists))
	}
	if len(board.Lists[0].Cards) != 2 {
		t.Errorf("expected 2 cards in first list, got %d", len(board.Lists[0].Cards))
	}
	if board.Lists[0].Cards[0].Name != "Card One" {
		t.Errorf("expected 'Card One', got %q", board.Lists[0].Cards[0].Name)
	}
}

func TestResolveBoard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/members/me/boards" {
			resp := []map[string]interface{}{
				{"id": "board1", "name": "My Board", "url": "https://trello.com/b/board1"},
				{"id": "board2", "name": "Other Board", "url": "https://trello.com/b/board2"},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c := NewClient("testkey", "testtoken")
	c.BaseURL = server.URL

	id, err := c.ResolveBoard("My Board")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != "board1" {
		t.Errorf("expected 'board1', got %q", id)
	}
}

func TestResolveBoardAmbiguous(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/members/me/boards" {
			resp := []map[string]interface{}{
				{"id": "board1", "name": "My Board", "url": "https://trello.com/b/board1"},
				{"id": "board2", "name": "My Board", "url": "https://trello.com/b/board2"},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c := NewClient("testkey", "testtoken")
	c.BaseURL = server.URL

	_, err := c.ResolveBoard("My Board")
	if err == nil {
		t.Fatal("expected error for ambiguous board name")
	}
}

func TestMoveCardToList(t *testing.T) {
	var receivedBody map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/cards/card1" && r.Method == http.MethodPut {
			r.ParseForm()
			receivedBody = map[string]string{
				"idList": r.FormValue("idList"),
				"pos":    r.FormValue("pos"),
			}
			json.NewEncoder(w).Encode(map[string]string{"id": "card1"})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c := NewClient("testkey", "testtoken")
	c.BaseURL = server.URL

	err := c.MoveCardToList("card1", "list2", "top")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if receivedBody["idList"] != "list2" {
		t.Errorf("expected idList 'list2', got %q", receivedBody["idList"])
	}
}

func TestReorderCard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/cards/card1" && r.Method == http.MethodPut {
			json.NewEncoder(w).Encode(map[string]string{"id": "card1"})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c := NewClient("testkey", "testtoken")
	c.BaseURL = server.URL

	err := c.ReorderCard("card1", 12345.0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:
```bash
go test ./internal/trello/ -v
```
Expected: compilation errors (Client doesn't exist yet).

- [ ] **Step 3: Write client.go**

```go
// internal/trello/client.go
package trello

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Client wraps the Trello REST API.
type Client struct {
	BaseURL    string
	apiKey     string
	token      string
	httpClient *http.Client
}

func NewClient(apiKey, token string) *Client {
	return &Client{
		BaseURL:    "https://api.trello.com",
		apiKey:     apiKey,
		token:      token,
		httpClient: &http.Client{},
	}
}

func (c *Client) get(path string, target interface{}) error {
	u, err := url.Parse(c.BaseURL + path)
	if err != nil {
		return err
	}
	q := u.Query()
	q.Set("key", c.apiKey)
	q.Set("token", c.token)
	u.RawQuery = q.Encode()

	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("unauthorized: check your TRELLO_API_KEY and TRELLO_TOKEN")
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("rate limited by Trello API, please try again shortly")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func (c *Client) put(path string, form url.Values) error {
	form.Set("key", c.apiKey)
	form.Set("token", c.token)

	req, err := http.NewRequest(http.MethodPut, c.BaseURL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("unauthorized: check your TRELLO_API_KEY and TRELLO_TOKEN")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// ValidateCredentials checks that the API key and token are valid.
func (c *Client) ValidateCredentials() error {
	var result map[string]interface{}
	return c.get("/1/members/me", &result)
}

// apiBoard/apiList/apiCard are intermediate types for JSON unmarshaling.
type apiBoard struct {
	ID    string    `json:"id"`
	Name  string    `json:"name"`
	URL   string    `json:"url"`
	Lists []apiList `json:"lists"`
}

type apiList struct {
	ID    string    `json:"id"`
	Name  string    `json:"name"`
	Pos   float64   `json:"pos"`
	Cards []apiCard `json:"cards"`
}

type apiCard struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Desc        string     `json:"desc"`
	Pos         float64    `json:"pos"`
	URL         string     `json:"url"`
	IDList      string     `json:"idList"`
	IDMembers   []string   `json:"idMembers"`
	IDLabels    []string   `json:"idLabels"`
	Labels      []apiLabel `json:"labels"`
}

type apiLabel struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// FetchBoard retrieves a board with all open lists and cards in a single request.
func (c *Client) FetchBoard(boardID string) (*Board, error) {
	var ab apiBoard
	path := fmt.Sprintf("/1/boards/%s?lists=open&cards=open&card_fields=name,desc,labels,idMembers,url,pos,idList&list_fields=name,pos", boardID)
	if err := c.get(path, &ab); err != nil {
		return nil, err
	}

	board := &Board{
		ID:   ab.ID,
		Name: ab.Name,
		URL:  ab.URL,
	}

	for _, al := range ab.Lists {
		list := List{
			ID:   al.ID,
			Name: al.Name,
			Pos:  al.Pos,
		}
		for _, ac := range al.Cards {
			card := Card{
				ID:          ac.ID,
				Name:        ac.Name,
				Description: ac.Desc,
				Pos:         ac.Pos,
				URL:         ac.URL,
				MemberIDs:   ac.IDMembers,
				ListID:      ac.IDList,
			}
			for _, lbl := range ac.Labels {
				card.Labels = append(card.Labels, Label{
					ID:    lbl.ID,
					Name:  lbl.Name,
					Color: lbl.Color,
				})
			}
			list.Cards = append(list.Cards, card)
		}
		board.Lists = append(board.Lists, list)
	}

	return board, nil
}

// ResolveBoard finds a board ID by name. Errors if zero or multiple matches.
func (c *Client) ResolveBoard(name string) (string, error) {
	var boards []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := c.get("/1/members/me/boards", &boards); err != nil {
		return "", err
	}

	var matches []string
	for _, b := range boards {
		if strings.EqualFold(b.Name, name) {
			matches = append(matches, b.ID)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no board found with name %q", name)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("multiple boards match %q (IDs: %s) — use --board-id to specify", name, strings.Join(matches, ", "))
	}
}

// MoveCardToList moves a card to a different list at the given position ("top", "bottom", or numeric).
func (c *Client) MoveCardToList(cardID, listID, pos string) error {
	form := url.Values{}
	form.Set("idList", listID)
	form.Set("pos", pos)
	return c.put(fmt.Sprintf("/1/cards/%s", cardID), form)
}

// ReorderCard changes a card's position within its current list.
func (c *Client) ReorderCard(cardID string, pos float64) error {
	form := url.Values{}
	form.Set("pos", fmt.Sprintf("%f", pos))
	return c.put(fmt.Sprintf("/1/cards/%s", cardID), form)
}
```

- [ ] **Step 4: Run tests**

Run:
```bash
go test ./internal/trello/ -v
```
Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/trello/
git commit -m "feat: add Trello API client with board fetch, resolve, and card move operations"
```

---

### Task 5: Keybindings & Theme

**Files:**
- Create: `internal/tui/keys.go`
- Create: `internal/tui/keys_test.go`
- Create: `internal/tui/theme.go`

- [ ] **Step 1: Write keybinding tests**

```go
// internal/tui/keys_test.go
package tui

import (
	"testing"

	"github.com/craig006/tuillo/internal/config"
)

func TestDefaultKeyMap(t *testing.T) {
	cfg := config.DefaultConfig()
	km := NewKeyMap(cfg.Keybinding)

	if km.Quit.Keys()[0] != "q" {
		t.Errorf("expected quit key 'q', got %q", km.Quit.Keys()[0])
	}
	if km.MoveLeft.Keys()[0] != "h" {
		t.Errorf("expected moveLeft 'h', got %q", km.MoveLeft.Keys()[0])
	}
	if km.MoveCardLeft.Keys()[0] != "H" {
		t.Errorf("expected moveCardLeft 'H', got %q", km.MoveCardLeft.Keys()[0])
	}
}

func TestCustomKeyMap(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Keybinding.Board.MoveLeft = "a"
	cfg.Keybinding.Universal.Quit = "Q"
	km := NewKeyMap(cfg.Keybinding)

	if km.MoveLeft.Keys()[0] != "a" {
		t.Errorf("expected moveLeft 'a', got %q", km.MoveLeft.Keys()[0])
	}
	if km.Quit.Keys()[0] != "Q" {
		t.Errorf("expected quit 'Q', got %q", km.Quit.Keys()[0])
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:
```bash
go test ./internal/tui/ -v
```
Expected: compilation errors.

- [ ] **Step 3: Write keys.go**

```go
// internal/tui/keys.go
package tui

import (
	"charm.land/bubbles/v2/key"

	"github.com/craig006/tuillo/internal/config"
)

// KeyMap holds all keybindings, constructed from config.
type KeyMap struct {
	Quit          key.Binding
	Help          key.Binding
	Refresh       key.Binding
	MoveLeft      key.Binding
	MoveRight     key.Binding
	MoveUp        key.Binding
	MoveDown      key.Binding
	MoveCardLeft  key.Binding
	MoveCardRight key.Binding
	MoveCardUp    key.Binding
	MoveCardDown  key.Binding
	Enter         key.Binding
	CustomCommand key.Binding
}

func NewKeyMap(cfg config.KeybindingConfig) KeyMap {
	return KeyMap{
		Quit:          key.NewBinding(key.WithKeys(cfg.Universal.Quit), key.WithHelp(cfg.Universal.Quit, "quit")),
		Help:          key.NewBinding(key.WithKeys(cfg.Universal.Help), key.WithHelp(cfg.Universal.Help, "help")),
		Refresh:       key.NewBinding(key.WithKeys(cfg.Universal.Refresh), key.WithHelp(cfg.Universal.Refresh, "refresh")),
		MoveLeft:      key.NewBinding(key.WithKeys(cfg.Board.MoveLeft, "left"), key.WithHelp(cfg.Board.MoveLeft, "column left")),
		MoveRight:     key.NewBinding(key.WithKeys(cfg.Board.MoveRight, "right"), key.WithHelp(cfg.Board.MoveRight, "column right")),
		MoveUp:        key.NewBinding(key.WithKeys(cfg.Board.MoveUp, "up"), key.WithHelp(cfg.Board.MoveUp, "card up")),
		MoveDown:      key.NewBinding(key.WithKeys(cfg.Board.MoveDown, "down"), key.WithHelp(cfg.Board.MoveDown, "card down")),
		MoveCardLeft:  key.NewBinding(key.WithKeys(cfg.Board.MoveCardLeft), key.WithHelp(cfg.Board.MoveCardLeft, "move card left")),
		MoveCardRight: key.NewBinding(key.WithKeys(cfg.Board.MoveCardRight), key.WithHelp(cfg.Board.MoveCardRight, "move card right")),
		MoveCardUp:    key.NewBinding(key.WithKeys(cfg.Board.MoveCardUp), key.WithHelp(cfg.Board.MoveCardUp, "move card up")),
		MoveCardDown:  key.NewBinding(key.WithKeys(cfg.Board.MoveCardDown), key.WithHelp(cfg.Board.MoveCardDown, "move card down")),
		Enter:         key.NewBinding(key.WithKeys(cfg.Board.Enter), key.WithHelp(cfg.Board.Enter, "select")),
		CustomCommand: key.NewBinding(key.WithKeys(cfg.Board.CustomCommand), key.WithHelp(cfg.Board.CustomCommand, "commands")),
	}
}
```

- [ ] **Step 4: Write theme.go**

```go
// internal/tui/theme.go
package tui

import (
	"charm.land/lipgloss/v2"

	"github.com/craig006/tuillo/internal/config"
)

// Theme holds pre-computed Lip Gloss styles.
type Theme struct {
	ActiveBorder   lipgloss.Style
	InactiveBorder lipgloss.Style
	SelectedCard   lipgloss.Style
	ColumnTitle    lipgloss.Style
}

func NewTheme(cfg config.ThemeConfig) Theme {
	return Theme{
		ActiveBorder:   buildStyle(cfg.ActiveBorderColor),
		InactiveBorder: buildStyle(cfg.InactiveBorderColor),
		SelectedCard:   buildStyle(cfg.SelectedCardColor),
		ColumnTitle:    buildStyle(cfg.ColumnTitleColor),
	}
}

// buildStyle creates a Lip Gloss style from a color attribute list.
// First element is the foreground color, subsequent elements are modifiers (bold, italic, etc.).
func buildStyle(attrs []string) lipgloss.Style {
	s := lipgloss.NewStyle()
	if len(attrs) == 0 {
		return s
	}

	s = s.Foreground(lipgloss.Color(attrs[0]))
	for _, attr := range attrs[1:] {
		switch attr {
		case "bold":
			s = s.Bold(true)
		case "italic":
			s = s.Italic(true)
		case "underline":
			s = s.Underline(true)
		}
	}
	return s
}
```

- [ ] **Step 5: Install charm dependencies and run tests**

Run:
```bash
go get charm.land/bubbletea/v2 charm.land/bubbles/v2 charm.land/lipgloss/v2
go test ./internal/tui/ -v
```
Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/keys.go internal/tui/keys_test.go internal/tui/theme.go
git commit -m "feat: add configurable keybindings and ANSI-default theme system"
```

---

### Task 6: Column Model

**Files:**
- Create: `internal/tui/column.go`
- Create: `internal/tui/column_test.go`

- [ ] **Step 1: Write column model tests**

```go
// internal/tui/column_test.go
package tui

import (
	"testing"

	"github.com/craig006/tuillo/internal/trello"
)

func TestNewColumn(t *testing.T) {
	cards := []trello.Card{
		{ID: "c1", Name: "Card 1", Pos: 1.0},
		{ID: "c2", Name: "Card 2", Pos: 2.0},
	}
	col := NewColumn(trello.List{
		ID:    "list1",
		Name:  "Backlog",
		Cards: cards,
	}, 30, 20, false)

	if col.Title() != "Backlog" {
		t.Errorf("expected title 'Backlog', got %q", col.Title())
	}
	if col.CardCount() != 2 {
		t.Errorf("expected 2 cards, got %d", col.CardCount())
	}
}

func TestColumnSelectedCard(t *testing.T) {
	cards := []trello.Card{
		{ID: "c1", Name: "Card 1", Pos: 1.0},
		{ID: "c2", Name: "Card 2", Pos: 2.0},
	}
	col := NewColumn(trello.List{
		ID:    "list1",
		Name:  "Backlog",
		Cards: cards,
	}, 30, 20, false)

	card, ok := col.SelectedCard()
	if !ok {
		t.Fatal("expected a selected card")
	}
	if card.ID != "c1" {
		t.Errorf("expected 'c1', got %q", card.ID)
	}
}

func TestColumnEmptyList(t *testing.T) {
	col := NewColumn(trello.List{
		ID:   "list1",
		Name: "Empty",
	}, 30, 20, false)

	_, ok := col.SelectedCard()
	if ok {
		t.Error("expected no selected card in empty list")
	}
	if col.CardCount() != 0 {
		t.Errorf("expected 0 cards, got %d", col.CardCount())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:
```bash
go test ./internal/tui/ -v -run TestColumn
```
Expected: compilation errors.

- [ ] **Step 3: Write column.go**

```go
// internal/tui/column.go
package tui

import (
	"fmt"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"

	"github.com/craig006/tuillo/internal/trello"
)

// cardItem adapts trello.Card to the bubbles/list.DefaultItem interface.
type cardItem struct {
	card trello.Card
}

func (i cardItem) Title() string       { return i.card.Name }
func (i cardItem) Description() string { return "" }
func (i cardItem) FilterValue() string { return i.card.Name }

// Column wraps a bubbles/list.Model for a single Trello list.
type Column struct {
	list   list.Model
	listID string
	name   string
	cards  []trello.Card
}

func NewColumn(l trello.List, width, height int, focused bool) Column {
	items := make([]list.Item, len(l.Cards))
	for i, c := range l.Cards {
		items[i] = cardItem{card: c}
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false

	m := list.New(items, delegate, width, height)
	m.Title = fmt.Sprintf("%s (%d)", l.Name, len(l.Cards))
	m.SetShowStatusBar(false)
	m.SetFilteringEnabled(false)

	return Column{
		list:   m,
		listID: l.ID,
		name:   l.Name,
		cards:  l.Cards,
	}
}

func (c Column) Title() string    { return c.name }
func (c Column) ListID() string   { return c.listID }
func (c Column) CardCount() int   { return len(c.cards) }
func (c Column) SelectedIndex() int { return c.list.Index() }

func (c Column) SelectedCard() (trello.Card, bool) {
	if len(c.cards) == 0 {
		return trello.Card{}, false
	}
	item := c.list.SelectedItem()
	if item == nil {
		return trello.Card{}, false
	}
	ci, ok := item.(cardItem)
	if !ok {
		return trello.Card{}, false
	}
	return ci.card, true
}

func (c Column) Cards() []trello.Card { return c.cards }

func (c *Column) SetSize(width, height int) {
	c.list.SetSize(width, height)
}

func (c Column) Update(msg tea.Msg) (Column, tea.Cmd) {
	var cmd tea.Cmd
	c.list, cmd = c.list.Update(msg)
	return c, cmd
}

func (c Column) View() string {
	return c.list.View()
}
```

- [ ] **Step 4: Run tests**

Run:
```bash
go test ./internal/tui/ -v -run TestColumn
```
Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/column.go internal/tui/column_test.go
git commit -m "feat: add column model wrapping bubbles/list for card display"
```

---

### Task 7: Board Model (3-Column Sliding Window)

**Files:**
- Create: `internal/tui/board.go`
- Create: `internal/tui/board_test.go`

- [ ] **Step 1: Write board model tests**

```go
// internal/tui/board_test.go
package tui

import (
	"fmt"
	"testing"

	"github.com/craig006/tuillo/internal/config"
	"github.com/craig006/tuillo/internal/trello"
)

func makeTestBoard(numLists int) *trello.Board {
	board := &trello.Board{ID: "b1", Name: "Test"}
	for i := 0; i < numLists; i++ {
		list := trello.List{
			ID:   fmt.Sprintf("list%d", i),
			Name: fmt.Sprintf("List %d", i),
			Pos:  float64(i),
			Cards: []trello.Card{
				{ID: fmt.Sprintf("c%d-1", i), Name: fmt.Sprintf("Card %d-1", i), Pos: 1.0, ListID: fmt.Sprintf("list%d", i)},
				{ID: fmt.Sprintf("c%d-2", i), Name: fmt.Sprintf("Card %d-2", i), Pos: 2.0, ListID: fmt.Sprintf("list%d", i)},
			},
		}
		board.Lists = append(board.Lists, list)
	}
	return board
}

func TestWindowStartMiddle(t *testing.T) {
	board := makeTestBoard(5)
	cfg := config.DefaultConfig()
	b := NewBoardModel(board, cfg, 90, 30)

	// Initially focused on column 0 → window shows [0, 1, 2]
	start, end := b.VisibleRange()
	if start != 0 || end != 3 {
		t.Errorf("expected range [0,3), got [%d,%d)", start, end)
	}
}

func TestWindowNavigateRight(t *testing.T) {
	board := makeTestBoard(5)
	cfg := config.DefaultConfig()
	b := NewBoardModel(board, cfg, 90, 30)

	// Move right to column 1 → window [0, 1, 2]
	b.FocusRight()
	start, end := b.VisibleRange()
	if start != 0 || end != 3 {
		t.Errorf("expected range [0,3), got [%d,%d)", start, end)
	}

	// Move right to column 2 → window [1, 2, 3]
	b.FocusRight()
	start, end = b.VisibleRange()
	if start != 1 || end != 4 {
		t.Errorf("expected range [1,4), got [%d,%d)", start, end)
	}
}

func TestWindowLastColumn(t *testing.T) {
	board := makeTestBoard(5)
	cfg := config.DefaultConfig()
	b := NewBoardModel(board, cfg, 90, 30)

	// Navigate to last column
	for i := 0; i < 4; i++ {
		b.FocusRight()
	}

	if b.FocusedColumn() != 4 {
		t.Errorf("expected focused column 4, got %d", b.FocusedColumn())
	}

	start, end := b.VisibleRange()
	if start != 2 || end != 5 {
		t.Errorf("expected range [2,5), got [%d,%d)", start, end)
	}
}

func TestWindowTwoColumns(t *testing.T) {
	board := makeTestBoard(2)
	cfg := config.DefaultConfig()
	b := NewBoardModel(board, cfg, 90, 30)

	start, end := b.VisibleRange()
	if start != 0 || end != 2 {
		t.Errorf("expected range [0,2), got [%d,%d)", start, end)
	}
}

func TestWindowOneColumn(t *testing.T) {
	board := makeTestBoard(1)
	cfg := config.DefaultConfig()
	b := NewBoardModel(board, cfg, 90, 30)

	start, end := b.VisibleRange()
	if start != 0 || end != 1 {
		t.Errorf("expected range [0,1), got [%d,%d)", start, end)
	}
}

func TestFocusLeftBoundary(t *testing.T) {
	board := makeTestBoard(5)
	cfg := config.DefaultConfig()
	b := NewBoardModel(board, cfg, 90, 30)

	b.FocusLeft() // should not go below 0
	if b.FocusedColumn() != 0 {
		t.Errorf("expected focused column 0, got %d", b.FocusedColumn())
	}
}

func TestFocusRightBoundary(t *testing.T) {
	board := makeTestBoard(3)
	cfg := config.DefaultConfig()
	b := NewBoardModel(board, cfg, 90, 30)

	b.FocusRight()
	b.FocusRight()
	b.FocusRight() // should not exceed 2
	if b.FocusedColumn() != 2 {
		t.Errorf("expected focused column 2, got %d", b.FocusedColumn())
	}
}

func TestPositionIndicator(t *testing.T) {
	board := makeTestBoard(5)
	cfg := config.DefaultConfig()
	b := NewBoardModel(board, cfg, 90, 30)

	indicator := b.PositionIndicator()
	if indicator != "[1/5]" {
		t.Errorf("expected '[1/5]', got %q", indicator)
	}

	b.FocusRight()
	b.FocusRight()
	indicator = b.PositionIndicator()
	if indicator != "[3/5]" {
		t.Errorf("expected '[3/5]', got %q", indicator)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:
```bash
go test ./internal/tui/ -v -run TestWindow -run TestFocus -run TestPosition
```
Expected: compilation errors.

- [ ] **Step 3: Write board.go**

```go
// internal/tui/board.go
package tui

import (
	"fmt"

	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"

	"github.com/craig006/tuillo/internal/config"
	"github.com/craig006/tuillo/internal/trello"
)

const maxVisibleColumns = 3

// BoardModel manages the kanban board view with a 3-column sliding window.
type BoardModel struct {
	columns []Column
	board   *trello.Board
	focused int
	width   int
	height  int
	keyMap  KeyMap
	theme   Theme
}

func NewBoardModel(board *trello.Board, cfg config.Config, width, height int) BoardModel {
	km := NewKeyMap(cfg.Keybinding)
	theme := NewTheme(cfg.GUI.Theme)

	colWidth := width / min(len(board.Lists), maxVisibleColumns)
	colHeight := height - 4 // leave room for header and status bar

	columns := make([]Column, len(board.Lists))
	for i, l := range board.Lists {
		columns[i] = NewColumn(l, colWidth, colHeight, i == 0)
	}

	return BoardModel{
		columns: columns,
		board:   board,
		focused: 0,
		width:   width,
		height:  height,
		keyMap:  km,
		theme:   theme,
	}
}

func (b *BoardModel) FocusedColumn() int { return b.focused }

func (b *BoardModel) FocusLeft() {
	if b.focused > 0 {
		b.focused--
	}
}

func (b *BoardModel) FocusRight() {
	if b.focused < len(b.columns)-1 {
		b.focused++
	}
}

// VisibleRange returns the [start, end) indices of visible columns.
func (b *BoardModel) VisibleRange() (int, int) {
	total := len(b.columns)
	if total <= maxVisibleColumns {
		return 0, total
	}

	// Center the focused column in the window
	start := b.focused - 1
	if start < 0 {
		start = 0
	}
	end := start + maxVisibleColumns
	if end > total {
		end = total
		start = end - maxVisibleColumns
	}
	return start, end
}

func (b *BoardModel) PositionIndicator() string {
	return fmt.Sprintf("[%d/%d]", b.focused+1, len(b.columns))
}

// SelectedCard returns the currently focused card and its list index.
func (b *BoardModel) SelectedCard() (trello.Card, int, bool) {
	if len(b.columns) == 0 {
		return trello.Card{}, 0, false
	}
	card, ok := b.columns[b.focused].SelectedCard()
	return card, b.focused, ok
}

// RemoveCard removes a card from the given column at the given index, returning the card.
func (b *BoardModel) RemoveCard(colIdx, cardIdx int) trello.Card {
	col := &b.columns[colIdx]
	card := col.cards[cardIdx]
	col.cards = append(col.cards[:cardIdx], col.cards[cardIdx+1:]...)
	b.rebuildColumnItems(colIdx)
	return card
}

// InsertCard inserts a card into the given column at the given position.
func (b *BoardModel) InsertCard(colIdx int, card trello.Card, pos int) {
	col := &b.columns[colIdx]
	if pos > len(col.cards) {
		pos = len(col.cards)
	}
	col.cards = append(col.cards[:pos], append([]trello.Card{card}, col.cards[pos:]...)...)
	b.rebuildColumnItems(colIdx)
}

func (b *BoardModel) rebuildColumnItems(colIdx int) {
	col := &b.columns[colIdx]
	items := make([]list.Item, len(col.cards))
	for i, c := range col.cards {
		items[i] = cardItem{card: c}
	}
	col.list.SetItems(items)
	col.list.Title = fmt.Sprintf("%s (%d)", col.name, len(col.cards))
}

// CalcNewPos calculates the position value for inserting a card at a given index in a column.
func CalcNewPos(cards []trello.Card, targetIdx int) float64 {
	if len(cards) == 0 {
		return 65536.0
	}
	if targetIdx <= 0 {
		return cards[0].Pos / 2.0
	}
	if targetIdx >= len(cards) {
		return cards[len(cards)-1].Pos + 65536.0
	}
	return (cards[targetIdx-1].Pos + cards[targetIdx].Pos) / 2.0
}

func (b BoardModel) Update(msg tea.Msg) (BoardModel, tea.Cmd) {
	if len(b.columns) == 0 {
		return b, nil
	}

	// Delegate to focused column for card navigation
	var cmd tea.Cmd
	b.columns[b.focused], cmd = b.columns[b.focused].Update(msg)
	return b, cmd
}

func (b BoardModel) View() string {
	if len(b.columns) == 0 {
		return "No lists on this board."
	}

	start, end := b.VisibleRange()
	colWidth := b.width / (end - start)

	views := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		col := b.columns[i]
		col.SetSize(colWidth-2, b.height-4)

		style := lipgloss.NewStyle().
			Width(colWidth - 2).
			Border(lipgloss.RoundedBorder())

		if i == b.focused {
			style = style.BorderForeground(b.theme.ActiveBorder.GetForeground())
		} else {
			style = style.BorderForeground(b.theme.InactiveBorder.GetForeground())
		}

		views = append(views, style.Render(col.View()))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, views...)
}

// Note: uses Go 1.21+ built-in min() function — no custom helper needed.
```

- [ ] **Step 4: Add fmt import to test file and run tests**

Run:
```bash
go test ./internal/tui/ -v -run "TestWindow|TestFocus|TestPosition"
```
Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/board.go internal/tui/board_test.go
git commit -m "feat: add board model with 3-column sliding window navigation"
```

---

### Task 8: Root App Model

**Files:**
- Create: `internal/tui/app.go`
- Modify: `cmd/root.go`

- [ ] **Step 1: Write app.go**

```go
// internal/tui/app.go
package tui

import (
	"fmt"

	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"

	"github.com/craig006/tuillo/internal/config"
	"github.com/craig006/tuillo/internal/trello"
)

// Messages
type BoardFetchedMsg struct {
	Board *trello.Board
}

type BoardFetchErrMsg struct {
	Err error
}

type CardMovedMsg struct {
	CardID string
}

type CardMoveErrMsg struct {
	Err    error
	// For rollback
	Card   trello.Card
	FromCol int
	FromIdx int
	ToCol   int
}

type BoardResolvedMsg struct {
	ID string
}

type StatusMsg struct {
	Text string
}

// App is the root Bubble Tea model.
type App struct {
	board      BoardModel
	client     *trello.Client
	config     config.Config
	keyMap     KeyMap
	help       help.Model
	status     string
	loading    bool
	showHelp   bool
	width      int
	height     int
	boardReady bool
}

func NewApp(client *trello.Client, cfg config.Config) App {
	km := NewKeyMap(cfg.Keybinding)
	return App{
		client: client,
		config: cfg,
		keyMap: km,
		help:   help.New(),
		status: "Loading board...",
		loading: true,
	}
}

func (a App) Init() tea.Cmd {
	boardID := a.config.Board.ID
	if boardID == "" && a.config.Board.Name != "" {
		// Need to resolve board name first
		return a.resolveBoardCmd(a.config.Board.Name)
	}
	if boardID == "" {
		return func() tea.Msg {
			return BoardFetchErrMsg{Err: fmt.Errorf("no board configured — use --board or --board-id, or set board.id in config")}
		}
	}
	return a.fetchBoardCmd(boardID)
}

func (a App) resolveBoardCmd(name string) tea.Cmd {
	return func() tea.Msg {
		id, err := a.client.ResolveBoard(name)
		if err != nil {
			return BoardFetchErrMsg{Err: err}
		}
		return BoardResolvedMsg{ID: id}
	}
}

func (a App) fetchBoardCmd(boardID string) tea.Cmd {
	return func() tea.Msg {
		board, err := a.client.FetchBoard(boardID)
		if err != nil {
			return BoardFetchErrMsg{Err: err}
		}
		return BoardFetchedMsg{Board: board}
	}
}

type moveRollback struct {
	Card    trello.Card
	FromCol int
	FromIdx int
	ToCol   int
}

func (a App) moveCardToListCmd(cardID, listID, pos string, rb moveRollback) tea.Cmd {
	return func() tea.Msg {
		err := a.client.MoveCardToList(cardID, listID, pos)
		if err != nil {
			return CardMoveErrMsg{Err: err, Card: rb.Card, FromCol: rb.FromCol, FromIdx: rb.FromIdx, ToCol: rb.ToCol}
		}
		return CardMovedMsg{CardID: cardID}
	}
}

func (a App) reorderCardCmd(cardID string, pos float64) tea.Cmd {
	return func() tea.Msg {
		err := a.client.ReorderCard(cardID, pos)
		if err != nil {
			return CardMoveErrMsg{Err: err}
		}
		return CardMovedMsg{CardID: cardID}
	}
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		if a.boardReady {
			a.board.width = msg.Width
			a.board.height = msg.Height - 4
		}
		return a, nil

	case BoardFetchedMsg:
		a.loading = false
		a.boardReady = true
		a.board = NewBoardModel(msg.Board, a.config, a.width, a.height-4)
		a.status = fmt.Sprintf("%s — %s", msg.Board.Name, a.board.PositionIndicator())
		return a, nil

	case BoardFetchErrMsg:
		a.loading = false
		a.status = fmt.Sprintf("Error: %v", msg.Err)
		return a, nil

	case BoardResolvedMsg:
		a.config.Board.ID = msg.ID
		return a, a.fetchBoardCmd(msg.ID)

	case StatusMsg:
		a.status = msg.Text
		return a, nil

	case CardMovedMsg:
		a.status = "Card moved"
		return a, nil

	case CardMoveErrMsg:
		a.status = fmt.Sprintf("Move failed: %v", msg.Err)
		// Rollback: remove card from destination, re-insert at source
		if msg.ToCol >= 0 && msg.ToCol < len(a.board.columns) && msg.FromCol >= 0 && msg.FromCol < len(a.board.columns) {
			// Find and remove the card from the destination column
			destCards := a.board.columns[msg.ToCol].cards
			for i, c := range destCards {
				if c.ID == msg.Card.ID {
					a.board.RemoveCard(msg.ToCol, i)
					break
				}
			}
			// Re-insert at original position
			a.board.InsertCard(msg.FromCol, msg.Card, msg.FromIdx)
		}
		return a, nil

	case tea.KeyPressMsg:
		switch {
		case matchKey(msg, a.keyMap.Quit):
			return a, tea.Quit

		case matchKey(msg, a.keyMap.Help):
			a.showHelp = !a.showHelp
			return a, nil

		case matchKey(msg, a.keyMap.Refresh):
			if a.config.Board.ID != "" {
				a.loading = true
				a.status = "Refreshing..."
				return a, a.fetchBoardCmd(a.config.Board.ID)
			}
			return a, nil

		case matchKey(msg, a.keyMap.MoveLeft):
			if a.boardReady {
				a.board.FocusLeft()
				a.status = fmt.Sprintf("%s — %s", a.board.board.Name, a.board.PositionIndicator())
			}
			return a, nil

		case matchKey(msg, a.keyMap.MoveRight):
			if a.boardReady {
				a.board.FocusRight()
				a.status = fmt.Sprintf("%s — %s", a.board.board.Name, a.board.PositionIndicator())
			}
			return a, nil

		case matchKey(msg, a.keyMap.MoveCardLeft):
			return a.handleMoveCardLeft()

		case matchKey(msg, a.keyMap.MoveCardRight):
			return a.handleMoveCardRight()

		case matchKey(msg, a.keyMap.MoveCardUp):
			return a.handleMoveCardUp()

		case matchKey(msg, a.keyMap.MoveCardDown):
			return a.handleMoveCardDown()
		}

		// Pass to board for card navigation (up/down)
		if a.boardReady {
			var cmd tea.Cmd
			a.board, cmd = a.board.Update(msg)
			return a, cmd
		}
	}

	return a, nil
}

func (a App) handleMoveCardLeft() (tea.Model, tea.Cmd) {
	if !a.boardReady || a.board.focused == 0 {
		return a, nil
	}

	card, colIdx, ok := a.board.SelectedCard()
	if !ok {
		return a, nil
	}

	cardIdx := a.board.columns[colIdx].SelectedIndex()
	targetCol := colIdx - 1
	rb := moveRollback{Card: card, FromCol: colIdx, FromIdx: cardIdx, ToCol: targetCol}

	// Optimistic update
	a.board.RemoveCard(colIdx, cardIdx)
	a.board.InsertCard(targetCol, card, 0)
	a.board.FocusLeft()
	a.status = fmt.Sprintf("Moving %q...", card.Name)

	targetListID := a.board.columns[targetCol].ListID()
	return a, a.moveCardToListCmd(card.ID, targetListID, "top", rb)
}

func (a App) handleMoveCardRight() (tea.Model, tea.Cmd) {
	if !a.boardReady || a.board.focused >= len(a.board.columns)-1 {
		return a, nil
	}

	card, colIdx, ok := a.board.SelectedCard()
	if !ok {
		return a, nil
	}

	cardIdx := a.board.columns[colIdx].SelectedIndex()
	targetCol := colIdx + 1
	rb := moveRollback{Card: card, FromCol: colIdx, FromIdx: cardIdx, ToCol: targetCol}

	// Optimistic update
	a.board.RemoveCard(colIdx, cardIdx)
	a.board.InsertCard(targetCol, card, 0)
	a.board.FocusRight()
	a.status = fmt.Sprintf("Moving %q...", card.Name)

	targetListID := a.board.columns[targetCol].ListID()
	return a, a.moveCardToListCmd(card.ID, targetListID, "top", rb)
}

func (a App) handleMoveCardUp() (tea.Model, tea.Cmd) {
	if !a.boardReady {
		return a, nil
	}

	card, colIdx, ok := a.board.SelectedCard()
	if !ok {
		return a, nil
	}

	cardIdx := a.board.columns[colIdx].SelectedIndex()
	if cardIdx <= 0 {
		return a, nil
	}

	// Optimistic: swap in place
	cards := a.board.columns[colIdx].cards
	cards[cardIdx], cards[cardIdx-1] = cards[cardIdx-1], cards[cardIdx]
	a.board.rebuildColumnItems(colIdx)

	newPos := CalcNewPos(cards, cardIdx-1)
	return a, a.reorderCardCmd(card.ID, newPos)
}

func (a App) handleMoveCardDown() (tea.Model, tea.Cmd) {
	if !a.boardReady {
		return a, nil
	}

	card, colIdx, ok := a.board.SelectedCard()
	if !ok {
		return a, nil
	}

	cardIdx := a.board.columns[colIdx].SelectedIndex()
	cards := a.board.columns[colIdx].cards
	if cardIdx >= len(cards)-1 {
		return a, nil
	}

	// Optimistic: swap in place
	cards[cardIdx], cards[cardIdx+1] = cards[cardIdx+1], cards[cardIdx]
	a.board.rebuildColumnItems(colIdx)

	newPos := CalcNewPos(cards, cardIdx+1)
	return a, a.reorderCardCmd(card.ID, newPos)
}

func matchKey(msg tea.KeyPressMsg, binding key.Binding) bool {
	for _, k := range binding.Keys() {
		if msg.String() == k {
			return true
		}
	}
	return false
}

func (a App) View() tea.View {
	if a.showHelp {
		return tea.NewView(a.renderHelp())
	}

	var content string
	if a.loading {
		content = "\n  Loading board...\n"
	} else if a.boardReady {
		content = a.board.View()
	} else {
		content = "\n  " + a.status + "\n"
	}

	statusBar := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 1).
		Render(a.status)

	view := lipgloss.JoinVertical(lipgloss.Left, content, statusBar)

	v := tea.NewView(view)
	v.AltScreen = true
	return v
}

func (a App) renderHelp() string {
	title := lipgloss.NewStyle().Bold(true).Padding(1).Render("tuillo — Keyboard Shortcuts")

	keys := []struct{ key, desc string }{
		{a.keyMap.Quit.Keys()[0], "Quit"},
		{a.keyMap.Help.Keys()[0], "Toggle help"},
		{a.keyMap.Refresh.Keys()[0], "Refresh board"},
		{a.keyMap.MoveLeft.Keys()[0] + "/" + "←", "Focus column left"},
		{a.keyMap.MoveRight.Keys()[0] + "/" + "→", "Focus column right"},
		{a.keyMap.MoveUp.Keys()[0] + "/" + "↑", "Focus card up"},
		{a.keyMap.MoveDown.Keys()[0] + "/" + "↓", "Focus card down"},
		{a.keyMap.MoveCardLeft.Keys()[0], "Move card left"},
		{a.keyMap.MoveCardRight.Keys()[0], "Move card right"},
		{a.keyMap.MoveCardUp.Keys()[0], "Move card up"},
		{a.keyMap.MoveCardDown.Keys()[0], "Move card down"},
		{a.keyMap.CustomCommand.Keys()[0], "Command palette"},
	}

	lines := title + "\n\n"
	for _, k := range keys {
		lines += fmt.Sprintf("  %-12s %s\n", k.key, k.desc)
	}
	lines += "\n  Press ? or Esc to close"
	return lines
}
```

- [ ] **Step 2: Add missing import for key package in app.go**

Ensure `charm.land/bubbles/v2/key` is imported (used by `matchKey` function).

- [ ] **Step 3: Wire up TUI launch in cmd/root.go**

Update the `RunE` in `cmd/root.go`:

```go
RunE: func(cmd *cobra.Command, args []string) error {
    apiKey := os.Getenv("TRELLO_API_KEY")
    token := os.Getenv("TRELLO_TOKEN")

    if apiKey == "" || token == "" {
        return fmt.Errorf("missing Trello credentials.\n\n" +
            "Set these environment variables:\n" +
            "  export TRELLO_API_KEY=<your-api-key>\n" +
            "  export TRELLO_TOKEN=<your-token>\n\n" +
            "Get your API key at: https://trello.com/power-ups/admin\n" +
            "Then authorize a token at:\n" +
            "  https://trello.com/1/authorize?expiration=never&scope=read,write&response_type=token&key=<YOUR_KEY>")
    }

    client := trello.NewClient(apiKey, token)

    if err := client.ValidateCredentials(); err != nil {
        return fmt.Errorf("invalid credentials: %w", err)
    }

    app := tui.NewApp(client, appConfig)
    p := tea.NewProgram(app)
    _, err := p.Run()
    return err
},
```

- [ ] **Step 4: Build and verify**

Run:
```bash
go build ./...
```
Expected: clean build.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/app.go cmd/root.go
git commit -m "feat: add root app model with board fetch, card moves, and help overlay"
```

---

### Task 9: Custom Command Engine

**Files:**
- Create: `internal/commands/custom.go`
- Create: `internal/commands/custom_test.go`

- [ ] **Step 1: Write custom command tests**

```go
// internal/commands/custom_test.go
package commands

import (
	"testing"

	"github.com/craig006/tuillo/internal/config"
	"github.com/craig006/tuillo/internal/trello"
)

func TestRenderTemplate(t *testing.T) {
	ctx := TemplateContext{
		Card: CardContext{
			ID:   "card1",
			Name: "Fix login bug",
			URL:  "https://trello.com/c/card1",
		},
		List: ListContext{
			ID:   "list1",
			Name: "In Progress",
		},
		Board: BoardContext{
			ID:   "board1",
			Name: "My Board",
		},
	}

	result, err := RenderTemplate("open {{.Card.URL}}", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result != "open https://trello.com/c/card1" {
		t.Errorf("expected 'open https://trello.com/c/card1', got %q", result)
	}
}

func TestRenderTemplateKebab(t *testing.T) {
	ctx := TemplateContext{
		Card: CardContext{Name: "Fix Login Bug"},
	}

	result, err := RenderTemplate("git checkout -b {{.Card.Name | kebab}}", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result != "git checkout -b fix-login-bug" {
		t.Errorf("expected 'git checkout -b fix-login-bug', got %q", result)
	}
}

func TestRenderTemplateSnake(t *testing.T) {
	ctx := TemplateContext{
		Card: CardContext{Name: "Fix Login Bug"},
	}

	result, err := RenderTemplate("{{.Card.Name | snake}}", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result != "fix_login_bug" {
		t.Errorf("expected 'fix_login_bug', got %q", result)
	}
}

func TestRenderTemplateWithPrompt(t *testing.T) {
	ctx := TemplateContext{
		Card: CardContext{ID: "card1"},
		Prompt: map[string]string{"Note": "This is a note"},
	}

	result, err := RenderTemplate("echo {{index .Prompt \"Note\"}} >> notes/{{.Card.ID}}.md", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result != "echo This is a note >> notes/card1.md" {
		t.Errorf("unexpected result: %q", result)
	}
}

func TestFilterCommandsByContext(t *testing.T) {
	cmds := []config.CustomCommandConfig{
		{Key: "g", Context: "card", Description: "Open in browser"},
		{Key: "n", Context: "board", Description: "New list"},
		{Key: "d", Context: "card", Description: "Delete card"},
	}

	filtered := FilterByContext(cmds, "card")
	if len(filtered) != 2 {
		t.Errorf("expected 2 card commands, got %d", len(filtered))
	}
}

func TestBuildContextFromCard(t *testing.T) {
	card := trello.Card{
		ID:   "c1",
		Name: "Test Card",
		URL:  "https://trello.com/c/c1",
		Labels: []trello.Label{
			{Name: "bug", Color: "red"},
			{Name: "urgent", Color: "orange"},
		},
		MemberIDs: []string{"m1", "m2"},
	}
	list := trello.List{ID: "l1", Name: "Todo"}
	board := trello.Board{ID: "b1", Name: "Project"}

	ctx := BuildContext(card, list, board)
	if ctx.Card.Labels != "bug,urgent" {
		t.Errorf("expected 'bug,urgent', got %q", ctx.Card.Labels)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:
```bash
go test ./internal/commands/ -v
```
Expected: compilation errors.

- [ ] **Step 3: Write custom.go**

```go
// internal/commands/custom.go
package commands

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"text/template"

	"github.com/craig006/tuillo/internal/config"
	"github.com/craig006/tuillo/internal/trello"
)

type CardContext struct {
	ID          string
	Name        string
	URL         string
	Description string
	Labels      string
	Members     string
}

type ListContext struct {
	ID   string
	Name string
}

type BoardContext struct {
	ID   string
	Name string
}

type TemplateContext struct {
	Card   CardContext
	List   ListContext
	Board  BoardContext
	Prompt map[string]string
}

var nonAlphaNum = regexp.MustCompile(`[^a-zA-Z0-9]+`)

var funcMap = template.FuncMap{
	"kebab": func(s string) string {
		return strings.Trim(nonAlphaNum.ReplaceAllString(strings.ToLower(s), "-"), "-")
	},
	"snake": func(s string) string {
		return strings.Trim(nonAlphaNum.ReplaceAllString(strings.ToLower(s), "_"), "_")
	},
	"camel": func(s string) string {
		parts := nonAlphaNum.Split(strings.ToLower(s), -1)
		for i := 1; i < len(parts); i++ {
			if len(parts[i]) > 0 {
				parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
			}
		}
		return strings.Join(parts, "")
	},
	"lower":   strings.ToLower,
	"upper":   strings.ToUpper,
	"trim":    strings.TrimSpace,
	"replace": func(old, new, s string) string { return strings.ReplaceAll(s, old, new) },
}

func RenderTemplate(tmplStr string, ctx TemplateContext) (string, error) {
	tmpl, err := template.New("cmd").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("invalid template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("template execution failed: %w", err)
	}

	return buf.String(), nil
}

func FilterByContext(cmds []config.CustomCommandConfig, context string) []config.CustomCommandConfig {
	var result []config.CustomCommandConfig
	for _, cmd := range cmds {
		if cmd.Context == context {
			result = append(result, cmd)
		}
	}
	return result
}

func BuildContext(card trello.Card, list trello.List, board trello.Board) TemplateContext {
	var labelNames []string
	for _, l := range card.Labels {
		labelNames = append(labelNames, l.Name)
	}

	return TemplateContext{
		Card: CardContext{
			ID:          card.ID,
			Name:        card.Name,
			URL:         card.URL,
			Description: card.Description,
			Labels:      strings.Join(labelNames, ","),
			Members:     strings.Join(card.MemberIDs, ","),
		},
		List: ListContext{
			ID:   list.ID,
			Name: list.Name,
		},
		Board: BoardContext{
			ID:   board.ID,
			Name: board.Name,
		},
		Prompt: make(map[string]string),
	}
}

// ExecuteSilent runs a command and returns stdout+stderr.
func ExecuteSilent(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// ExecuteTerminal returns the exec.Cmd so the caller can run it with full terminal access.
func ExecuteTerminal(command string) *exec.Cmd {
	return exec.Command("sh", "-c", command)
}
```

- [ ] **Step 4: Run tests**

Run:
```bash
go test ./internal/commands/ -v
```
Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/commands/
git commit -m "feat: add custom command engine with Go template rendering and context building"
```

---

### Task 10: Command Palette Integration

**Files:**
- Modify: `internal/tui/app.go`

- [ ] **Step 1: Add command palette state to App**

Add to the App struct:

```go
type App struct {
	// ... existing fields ...
	commandPalette list.Model
	showPalette    bool
	customCommands []config.CustomCommandConfig
}
```

- [ ] **Step 2: Add command palette item type**

```go
type commandItem struct {
	cmd config.CustomCommandConfig
}

func (c commandItem) Title() string       { return c.cmd.Description }
func (c commandItem) Description() string { return c.cmd.Key }
func (c commandItem) FilterValue() string { return c.cmd.Description }
```

- [ ] **Step 3: Add palette toggle and execution to Update**

In the `tea.KeyPressMsg` switch, add:

```go
case matchKey(msg, a.keyMap.CustomCommand):
    if a.boardReady && !a.showPalette {
        filtered := commands.FilterByContext(a.config.CustomCommands, "card")
        items := make([]list.Item, len(filtered))
        for i, cmd := range filtered {
            items[i] = commandItem{cmd: cmd}
        }
        a.commandPalette.SetItems(items)
        a.commandPalette.SetFilteringEnabled(true)
        a.showPalette = true
        return a, nil
    }
```

Add Escape handling when palette is shown:

```go
if a.showPalette {
    if msg.String() == "esc" {
        a.showPalette = false
        return a, nil
    }
    if msg.String() == "enter" {
        if item, ok := a.commandPalette.SelectedItem().(commandItem); ok {
            a.showPalette = false
            return a.executeCustomCommand(item.cmd)
        }
    }
    var cmd tea.Cmd
    a.commandPalette, cmd = a.commandPalette.Update(msg)
    return a, cmd
}
```

- [ ] **Step 4: Add prompt state and command execution method**

Add prompt state fields to App:

```go
type App struct {
	// ... existing fields ...
	pendingCommand *config.CustomCommandConfig  // command awaiting prompt completion
	pendingCtx     commands.TemplateContext      // context for pending command
	promptIdx      int                           // current prompt index
	promptInput    textinput.Model               // for input prompts
	showPrompt     bool
	promptType     string                        // "confirm", "input", "menu"
}
```

Add prompt message types:

```go
type PromptCompleteMsg struct {
	Key   string
	Value string
}
```

Add the execution method that handles prompts:

```go
func (a App) executeCustomCommand(cmd config.CustomCommandConfig) (tea.Model, tea.Cmd) {
    card, colIdx, ok := a.board.SelectedCard()
    if !ok {
        a.status = "No card selected"
        return a, nil
    }

    col := a.board.columns[colIdx]
    ctx := commands.BuildContext(card, trello.List{ID: col.ListID(), Name: col.Title()}, *a.board.board)

    // If command has prompts, start the prompt flow
    if len(cmd.Prompts) > 0 {
        a.pendingCommand = &cmd
        a.pendingCtx = ctx
        a.promptIdx = 0
        return a.showNextPrompt()
    }

    // No prompts — execute immediately
    return a.runCommand(cmd, ctx)
}

func (a App) showNextPrompt() (tea.Model, tea.Cmd) {
    if a.promptIdx >= len(a.pendingCommand.Prompts) {
        // All prompts done, execute the command
        cmd := *a.pendingCommand
        ctx := a.pendingCtx
        a.pendingCommand = nil
        a.showPrompt = false
        return a.runCommand(cmd, ctx)
    }

    prompt := a.pendingCommand.Prompts[a.promptIdx]
    a.showPrompt = true
    a.promptType = prompt.Type

    // Render the title template with current context
    title, _ := commands.RenderTemplate(prompt.Title, a.pendingCtx)

    switch prompt.Type {
    case "confirm":
        a.status = title + " (y/n)"
    case "input":
        ti := textinput.New()
        ti.Placeholder = title
        ti.Focus()
        a.promptInput = ti
    case "menu":
        // Reuse command palette for menu options
        items := make([]list.Item, len(prompt.Options))
        for i, opt := range prompt.Options {
            items[i] = commandItem{cmd: config.CustomCommandConfig{Description: opt.Name, Key: opt.Value}}
        }
        a.commandPalette.SetItems(items)
        a.showPalette = true
    }

    return a, nil
}

func (a App) runCommand(cmd config.CustomCommandConfig, ctx commands.TemplateContext) (tea.Model, tea.Cmd) {
    rendered, err := commands.RenderTemplate(cmd.Command, ctx)
    if err != nil {
        a.status = fmt.Sprintf("Template error: %v", err)
        return a, nil
    }

    switch cmd.Output {
    case "terminal":
        c := commands.ExecuteTerminal(rendered)
        return a, tea.ExecProcess(c, func(err error) tea.Msg {
            if err != nil {
                return StatusMsg{Text: fmt.Sprintf("Command failed: %v", err)}
            }
            return StatusMsg{Text: "Command completed"}
        })
    case "popup":
        return a, func() tea.Msg {
            output, err := commands.ExecuteSilent(rendered)
            if err != nil {
                return StatusMsg{Text: fmt.Sprintf("Error: %v — %s", err, output)}
            }
            return StatusMsg{Text: output}
        }
    default: // "none"
        return a, func() tea.Msg {
            _, err := commands.ExecuteSilent(rendered)
            if err != nil {
                return StatusMsg{Text: fmt.Sprintf("Command failed: %v", err)}
            }
            return StatusMsg{Text: "Command executed"}
        }
    }
}
```

Add prompt input handling in the Update method (before the palette check):

```go
// Handle active prompts
if a.showPrompt && a.pendingCommand != nil {
    prompt := a.pendingCommand.Prompts[a.promptIdx]
    switch a.promptType {
    case "confirm":
        if msg.String() == "y" {
            a.promptIdx++
            a.showPrompt = false
            return a.showNextPrompt()
        }
        if msg.String() == "n" || msg.String() == "esc" {
            a.pendingCommand = nil
            a.showPrompt = false
            a.status = "Cancelled"
            return a, nil
        }
    case "input":
        if msg.String() == "enter" {
            a.pendingCtx.Prompt[prompt.Key] = a.promptInput.Value()
            a.promptIdx++
            a.showPrompt = false
            return a.showNextPrompt()
        }
        if msg.String() == "esc" {
            a.pendingCommand = nil
            a.showPrompt = false
            a.status = "Cancelled"
            return a, nil
        }
        var cmd tea.Cmd
        a.promptInput, cmd = a.promptInput.Update(msg)
        return a, cmd
    }
}
```

Note: This requires adding `"charm.land/bubbles/v2/textinput"` to the imports in `app.go`.

- [ ] **Step 5: Update View to render palette**

In the `View()` method, before the board view:

```go
if a.showPalette {
    paletteView := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("cyan")).
        Padding(1).
        Width(a.width / 2).
        Render(a.commandPalette.View())
    content = lipgloss.Place(a.width, a.height-2, lipgloss.Center, lipgloss.Center, paletteView)
}
```

- [ ] **Step 6: Build and verify**

Run:
```bash
go build ./...
```
Expected: clean build.

- [ ] **Step 7: Commit**

```bash
git add internal/tui/app.go
git commit -m "feat: add command palette with filtering and custom command execution"
```

---

### Task 11: Integration Test & Manual Verification

**Files:**
- Create: `internal/tui/app_test.go`

- [ ] **Step 1: Write integration test for app initialization**

```go
// internal/tui/app_test.go
package tui

import (
	"testing"

	"github.com/craig006/tuillo/internal/config"
	"github.com/craig006/tuillo/internal/trello"
)

func TestAppInitNoBoard(t *testing.T) {
	cfg := config.DefaultConfig()
	client := trello.NewClient("key", "token")
	app := NewApp(client, cfg)

	cmd := app.Init()
	if cmd == nil {
		t.Fatal("expected a command for missing board error")
	}

	msg := cmd()
	if _, ok := msg.(BoardFetchErrMsg); !ok {
		t.Errorf("expected BoardFetchErrMsg, got %T", msg)
	}
}

func TestAppBoardFetchedMsg(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Board.ID = "board1"
	client := trello.NewClient("key", "token")
	app := NewApp(client, cfg)
	app.width = 90
	app.height = 30

	board := &trello.Board{
		ID:   "board1",
		Name: "Test Board",
		Lists: []trello.List{
			{ID: "l1", Name: "Todo", Cards: []trello.Card{
				{ID: "c1", Name: "Card 1", Pos: 1.0},
			}},
			{ID: "l2", Name: "Done", Cards: []trello.Card{}},
		},
	}

	model, _ := app.Update(BoardFetchedMsg{Board: board})
	updated := model.(App)

	if !updated.boardReady {
		t.Error("expected boardReady to be true")
	}
	if updated.loading {
		t.Error("expected loading to be false")
	}
}
```

- [ ] **Step 2: Run all tests**

Run:
```bash
go test ./... -v
```
Expected: all tests pass across all packages.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/app_test.go
git commit -m "test: add app model integration tests"
```

---

### Task 12: Final Wiring & Polish

**Files:**
- Modify: `cmd/root.go` (ensure all imports are correct)
- Modify: `internal/tui/app.go` (ensure all imports are correct)

- [ ] **Step 1: Run go vet and fix any issues**

Run:
```bash
go vet ./...
```
Expected: no issues.

- [ ] **Step 2: Run go build for final binary**

Run:
```bash
go build -o tuillo .
```
Expected: produces `tuillo` binary.

- [ ] **Step 3: Add tuillo binary to .gitignore**

```
# .gitignore
tuillo
.superpowers/
```

- [ ] **Step 4: Run full test suite one final time**

Run:
```bash
go test ./... -v -count=1
```
Expected: all tests pass.

- [ ] **Step 5: Final commit**

```bash
git add .gitignore
git commit -m "chore: add .gitignore and finalize build"
```

---

## Summary

| Task | What it builds | Tests |
|------|---------------|-------|
| 1 | Go module, Cobra CLI skeleton | Manual verify |
| 2 | Config system with cascade loading | 3 tests |
| 3 | Trello domain types | Build verify |
| 4 | Trello API client | 6 tests |
| 5 | Keybindings + Theme | 2 tests |
| 6 | Column model (bubbles/list wrapper) | 3 tests |
| 7 | Board model (sliding window) | 8 tests |
| 8 | Root app model + TUI launch | Build verify |
| 9 | Custom command engine | 5 tests |
| 10 | Command palette integration | Build verify |
| 11 | Integration tests | 2 tests |
| 12 | Final polish + .gitignore | Full suite |
