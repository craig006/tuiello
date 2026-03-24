# Views Feature Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a "Views" system with named filter presets, a tab bar UI, `@me` token resolution, and configurable view switching keys.

**Architecture:** Views are config-defined presets with fixed base filters and optional GUI overrides. A tab bar renders at the top of the screen. The `@me` token resolves to the authenticated Trello user. View switching is a hard reset to the view's base filter.

**Tech Stack:** Go, Bubble Tea v2 (`charm.land/bubbletea/v2`), Bubbles v2, Lip Gloss v2, Viper

**Spec:** `docs/superpowers/specs/2026-03-24-views-design.md`

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/config/config.go` | Modify | Add `ViewConfig`, `ViewKeys` types; add `Views` to `Config` and `KeybindingConfig`; defaults |
| `internal/config/config_test.go` | Create | Test default views, key assignment, duplicate key rejection |
| `internal/trello/client.go` | Modify | Add `FetchCurrentUser()` method |
| `internal/tui/filter.go` | Modify | Add `currentUser` param to `ParseFilter`, resolve `@me` |
| `internal/tui/filter_test.go` | Modify | Add `@me` tests, update existing call sites |
| `internal/tui/views.go` | Create | View tab bar model, key assignment, rendering |
| `internal/tui/views_test.go` | Create | Test key assignment, rendering |
| `internal/tui/keys.go` | Modify | Add `ViewNext`, `ViewPrev` bindings |
| `internal/tui/app.go` | Modify | Integrate views: state, switching, escape behavior, tab bar rendering |

---

### Task 1: Config Types and Defaults

**Files:**
- Modify: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write failing tests for config defaults and key assignment**

In `internal/config/config_test.go`:

```go
package config

import (
	"testing"
)

func TestDefaultConfigHasDefaultViews(t *testing.T) {
	cfg := DefaultConfig()
	if len(cfg.Views) != 2 {
		t.Fatalf("expected 2 default views, got %d", len(cfg.Views))
	}
	if cfg.Views[0].Title != "My Cards" {
		t.Errorf("expected first view 'My Cards', got %q", cfg.Views[0].Title)
	}
	if cfg.Views[0].Filter != "member:@me" {
		t.Errorf("expected first view filter 'member:@me', got %q", cfg.Views[0].Filter)
	}
	if cfg.Views[0].Key != "m" {
		t.Errorf("expected first view key 'm', got %q", cfg.Views[0].Key)
	}
	if cfg.Views[1].Title != "All Cards" {
		t.Errorf("expected second view 'All Cards', got %q", cfg.Views[1].Title)
	}
}

func TestDefaultConfigHasViewKeys(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Keybinding.Views.NextView != "v" {
		t.Errorf("expected nextView 'v', got %q", cfg.Keybinding.Views.NextView)
	}
	if cfg.Keybinding.Views.PrevView != "V" {
		t.Errorf("expected prevView 'V', got %q", cfg.Keybinding.Views.PrevView)
	}
}

func TestAssignViewKeys(t *testing.T) {
	views := []ViewConfig{
		{Title: "My Cards", Key: "m"},
		{Title: "Mobile Cards"},
		{Title: "All Cards"},
	}
	keys := AssignViewKeys(views)
	if keys[0] != "m" {
		t.Errorf("expected 'm', got %q", keys[0])
	}
	if keys[1] != "1" {
		t.Errorf("expected '1', got %q", keys[1])
	}
	if keys[2] != "2" {
		t.Errorf("expected '2', got %q", keys[2])
	}
}

func TestAssignViewKeysSkipsUsedNumbers(t *testing.T) {
	views := []ViewConfig{
		{Title: "A", Key: "1"},
		{Title: "B"},
		{Title: "C"},
	}
	keys := AssignViewKeys(views)
	if keys[0] != "1" {
		t.Errorf("expected '1', got %q", keys[0])
	}
	if keys[1] != "2" {
		t.Errorf("expected '2', got %q", keys[1])
	}
	if keys[2] != "3" {
		t.Errorf("expected '3', got %q", keys[2])
	}
}

func TestAssignViewKeysDuplicateCustomKey(t *testing.T) {
	views := []ViewConfig{
		{Title: "A", Key: "m"},
		{Title: "B", Key: "m"},
	}
	keys := AssignViewKeys(views)
	// First keeps the key, second duplicate gets auto-assigned
	if keys[0] != "m" {
		t.Errorf("expected 'm', got %q", keys[0])
	}
	if keys[1] != "1" {
		t.Errorf("expected '1' (duplicate overridden), got %q", keys[1])
	}
}

func TestAssignViewKeysEmptyViews(t *testing.T) {
	keys := AssignViewKeys(nil)
	if len(keys) != 0 {
		t.Errorf("expected empty keys, got %v", keys)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go test ./internal/config/ -v -run "TestDefault|TestAssign"`
Expected: FAIL — types and functions don't exist yet

- [ ] **Step 3: Add ViewConfig and ViewKeys types to config.go**

In `internal/config/config.go`, add after the `OptionConfig` type (around line 96):

```go
type ViewConfig struct {
	Title           string `mapstructure:"title"`
	Filter          string `mapstructure:"filter"`
	Key             string `mapstructure:"key"`
	ShowDetailPanel *bool  `mapstructure:"showDetailPanel"`
	ColumnWidth     *int   `mapstructure:"columnWidth"`
	ShowCardLabels  *bool  `mapstructure:"showCardLabels"`
}

type ViewKeys struct {
	NextView string `mapstructure:"nextView"`
	PrevView string `mapstructure:"prevView"`
}
```

Add `Views ViewKeys` to `KeybindingConfig`:

```go
type KeybindingConfig struct {
	Universal UniversalKeys `mapstructure:"universal"`
	Board     BoardKeys     `mapstructure:"board"`
	Detail    DetailKeys    `mapstructure:"detail"`
	Filter    FilterKeys    `mapstructure:"filter"`
	Views     ViewKeys      `mapstructure:"views"`
}
```

Add `Views []ViewConfig` to `Config`:

```go
type Config struct {
	GUI            GUIConfig             `mapstructure:"gui"`
	Board          BoardConfig           `mapstructure:"board"`
	Keybinding     KeybindingConfig      `mapstructure:"keybinding"`
	CustomCommands []CustomCommandConfig `mapstructure:"customCommands"`
	Views          []ViewConfig          `mapstructure:"views"`
}
```

Update `DefaultConfig()` to include view defaults:

```go
Views: []ViewConfig{
	{Title: "My Cards", Filter: "member:@me", Key: "m"},
	{Title: "All Cards"},
},
```

And in the Keybinding section:

```go
Views: ViewKeys{
	NextView: "v",
	PrevView: "V",
},
```

- [ ] **Step 4: Add AssignViewKeys function**

Add to `internal/config/config.go`:

```go
// AssignViewKeys assigns shortcut keys to views. Views with a custom Key
// keep it (first occurrence wins for duplicates). Views without a Key get
// auto-assigned incrementing numbers, skipping already-used keys.
func AssignViewKeys(views []ViewConfig) []string {
	used := map[string]bool{}
	keys := make([]string, len(views))
	for i, v := range views {
		if v.Key != "" && !used[v.Key] {
			keys[i] = v.Key
			used[v.Key] = true
		}
	}
	next := 1
	for i := range views {
		if keys[i] == "" {
			for used[strconv.Itoa(next)] {
				next++
			}
			keys[i] = strconv.Itoa(next)
			used[strconv.Itoa(next)] = true
			next++
		}
	}
	return keys
}
```

Add `"strconv"` to the imports.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go test ./internal/config/ -v -run "TestDefault|TestAssign"`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat: add ViewConfig, ViewKeys types and AssignViewKeys to config"
```

---

### Task 2: FetchCurrentUser on Trello Client

**Files:**
- Modify: `internal/trello/client.go`

- [ ] **Step 1: Add FetchCurrentUser method**

Add to `internal/trello/client.go` after the `ValidateCredentials` method:

```go
// FetchCurrentUser retrieves the authenticated user's member record.
func (c *Client) FetchCurrentUser() (Member, error) {
	var am apiMember
	if err := c.get("/1/members/me?fields=id,fullName,initials,username", &am); err != nil {
		return Member{}, err
	}
	return Member{
		ID:       am.ID,
		FullName: am.FullName,
		Initials: am.Initials,
		Username: am.Username,
	}, nil
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go build ./internal/trello/`
Expected: Success

- [ ] **Step 3: Commit**

```bash
git add internal/trello/client.go
git commit -m "feat: add FetchCurrentUser method to Trello client"
```

---

### Task 3: @me Token Resolution in ParseFilter

**Files:**
- Modify: `internal/tui/filter.go`
- Modify: `internal/tui/filter_test.go`
- Modify: `internal/tui/app.go` (update call sites)

- [ ] **Step 1: Write failing tests for @me resolution**

Add to `internal/tui/filter_test.go`:

```go
func TestParseFilterAtMe(t *testing.T) {
	f := ParseFilter("member:@me fix", "craig")
	if len(f.Members) != 1 || f.Members[0] != "craig" {
		t.Errorf("expected @me resolved to 'craig', got %v", f.Members)
	}
	if f.Text != "fix" {
		t.Errorf("expected text 'fix', got %q", f.Text)
	}
}

func TestParseFilterAtMeEmpty(t *testing.T) {
	f := ParseFilter("member:@me", "")
	if len(f.Members) != 1 || f.Members[0] != "@me" {
		t.Errorf("expected literal '@me' when no user, got %v", f.Members)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go test ./internal/tui/ -v -run "TestParseFilterAtMe"`
Expected: FAIL — `ParseFilter` signature doesn't accept second param

- [ ] **Step 3: Update ParseFilter signature**

In `internal/tui/filter.go`, change the `ParseFilter` function signature:

```go
// ParseFilter parses a search string into structured filter components.
// Recognized tokens: member:<value>, label:<value>. Quoted values supported.
// The currentUser param resolves member:@me tokens; pass "" to keep @me literal.
func ParseFilter(input string, currentUser string) Filter {
```

Inside the `member:` handling block, add `@me` resolution after extracting `val`:

```go
		if strings.HasPrefix(lower, "member:") {
			val := tok[len("member:"):]
			val = strings.Trim(val, `"`)
			if val == "@me" && currentUser != "" {
				val = currentUser
			}
			if val != "" {
				f.Members = append(f.Members, val)
			}
```

- [ ] **Step 4: Update all existing tests to pass "" as currentUser**

Update all existing `ParseFilter` calls in `filter_test.go` to include the second argument `""`:

- `ParseFilter("fix door", "")`
- `ParseFilter("member:craig fix", "")`
- `ParseFilter("label:Bug label:Design", "")`
- `ParseFilter(`member:"Craig Smith" fix`, "")`
- `ParseFilter("", "")`

- [ ] **Step 5: Update all ParseFilter call sites in app.go**

Search for `ParseFilter(` in `app.go` and add the current user argument. The `App` struct doesn't have `currentUser` yet, so use `""` for now — Task 6 will wire it up. There are approximately 5 call sites:

- Line ~240: `ParseFilter(a.searchInput.Value())` → `ParseFilter(a.searchInput.Value(), "")`
- Line ~401: `ParseFilter(a.searchInput.Value())` → `ParseFilter(a.searchInput.Value(), "")`
- Line ~435: `ParseFilter(a.searchInput.Value())` → `ParseFilter(a.searchInput.Value(), "")`
- Line ~586: `ParseFilter(a.searchInput.Value())` → `ParseFilter(a.searchInput.Value(), "")`
- Line ~609: `ParseFilter(a.searchInput.Value())` → `ParseFilter(a.searchInput.Value(), "")`
- Line ~970: `ParseFilter(a.searchInput.Value())` in `renderFilterDisplay()` → `ParseFilter(a.searchInput.Value(), "")`

- [ ] **Step 6: Run all tests**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go test ./internal/tui/ -v`
Expected: ALL PASS

- [ ] **Step 7: Verify full build**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go build ./...`
Expected: Success

- [ ] **Step 8: Commit**

```bash
git add internal/tui/filter.go internal/tui/filter_test.go internal/tui/app.go
git commit -m "feat: add @me token resolution to ParseFilter"
```

---

### Task 4: View Tab Bar Model

**Files:**
- Create: `internal/tui/views.go`
- Create: `internal/tui/views_test.go`

- [ ] **Step 1: Write failing tests for view model**

Create `internal/tui/views_test.go`:

```go
package tui

import (
	"strings"
	"testing"

	"github.com/craig006/tuillo/internal/config"
)

func TestNewViewBarDefaultViews(t *testing.T) {
	views := []config.ViewConfig{
		{Title: "My Cards", Filter: "member:@me", Key: "m"},
		{Title: "All Cards"},
	}
	vb := NewViewBar(views)
	if vb.Active() != 0 {
		t.Errorf("expected active view 0, got %d", vb.Active())
	}
	if len(vb.views) != 2 {
		t.Errorf("expected 2 views, got %d", len(vb.views))
	}
}

func TestViewBarCycleForward(t *testing.T) {
	views := []config.ViewConfig{
		{Title: "A"},
		{Title: "B"},
		{Title: "C"},
	}
	vb := NewViewBar(views)
	vb.Next()
	if vb.Active() != 1 {
		t.Errorf("expected 1, got %d", vb.Active())
	}
	vb.Next()
	vb.Next() // wraps
	if vb.Active() != 0 {
		t.Errorf("expected 0 after wrap, got %d", vb.Active())
	}
}

func TestViewBarCycleBackward(t *testing.T) {
	views := []config.ViewConfig{
		{Title: "A"},
		{Title: "B"},
	}
	vb := NewViewBar(views)
	vb.Prev() // wraps to last
	if vb.Active() != 1 {
		t.Errorf("expected 1 after wrap, got %d", vb.Active())
	}
}

func TestViewBarSelectByKey(t *testing.T) {
	views := []config.ViewConfig{
		{Title: "A", Key: "m"},
		{Title: "B"},
		{Title: "C"},
	}
	vb := NewViewBar(views)
	if !vb.SelectByKey("1") {
		t.Error("expected to select view with key '1'")
	}
	if vb.Active() != 1 {
		t.Errorf("expected 1, got %d", vb.Active())
	}
	if vb.SelectByKey("z") {
		t.Error("expected false for unknown key")
	}
}

func TestViewBarActiveConfig(t *testing.T) {
	showDetail := true
	views := []config.ViewConfig{
		{Title: "A", Filter: "member:craig", ShowDetailPanel: &showDetail},
		{Title: "B"},
	}
	vb := NewViewBar(views)
	cfg := vb.ActiveConfig()
	if cfg.Filter != "member:craig" {
		t.Errorf("expected filter, got %q", cfg.Filter)
	}
}

func TestViewBarRender(t *testing.T) {
	views := []config.ViewConfig{
		{Title: "My Cards", Key: "m"},
		{Title: "All Cards"},
	}
	vb := NewViewBar(views)
	rendered := vb.View(80)
	if !strings.Contains(rendered, "My Cards") {
		t.Error("expected 'My Cards' in rendered output")
	}
	if !strings.Contains(rendered, "All Cards") {
		t.Error("expected 'All Cards' in rendered output")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go test ./internal/tui/ -v -run "TestViewBar|TestNewViewBar"`
Expected: FAIL

- [ ] **Step 3: Implement ViewBar model**

Create `internal/tui/views.go`:

```go
package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/craig006/tuillo/internal/config"
)

// ViewBar manages the views tab bar state.
type ViewBar struct {
	views  []config.ViewConfig
	keys   []string // assigned shortcut keys per view
	active int
}

// NewViewBar creates a view bar from the config views.
// Filters out views with empty titles.
func NewViewBar(views []config.ViewConfig) ViewBar {
	var filtered []config.ViewConfig
	for _, v := range views {
		if v.Title != "" {
			filtered = append(filtered, v)
		}
	}
	if len(filtered) == 0 {
		// Fallback to defaults
		filtered = []config.ViewConfig{
			{Title: "My Cards", Filter: "member:@me", Key: "m"},
			{Title: "All Cards"},
		}
	}
	return ViewBar{
		views: filtered,
		keys:  config.AssignViewKeys(filtered),
	}
}

// Active returns the index of the active view.
func (v *ViewBar) Active() int { return v.active }

// ActiveConfig returns the config of the active view.
func (v *ViewBar) ActiveConfig() config.ViewConfig {
	return v.views[v.active]
}

// Next cycles to the next view (wraps around).
func (v *ViewBar) Next() {
	v.active = (v.active + 1) % len(v.views)
}

// Prev cycles to the previous view (wraps around).
func (v *ViewBar) Prev() {
	v.active = (v.active - 1 + len(v.views)) % len(v.views)
}

// SelectByKey selects a view by its shortcut key. Returns true if found.
func (v *ViewBar) SelectByKey(key string) bool {
	for i, k := range v.keys {
		if k == key {
			v.active = i
			return true
		}
	}
	return false
}

// Keys returns the assigned shortcut keys.
func (v *ViewBar) Keys() []string { return v.keys }

// View renders the tab bar at the given width.
func (v ViewBar) View(width int) string {
	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(15))
	activeKeyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(7))
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))

	// Calculate max title length for truncation
	sepWidth := 5 // "  │  "
	totalSepWidth := sepWidth * (len(v.views) - 1)
	shortcutWidth := 4 // " ‹k›" per view
	availableForTitles := width - 2 - totalSepWidth - (shortcutWidth * len(v.views)) // 2 for padding
	maxTitleLen := availableForTitles / len(v.views)
	if maxTitleLen < 5 {
		maxTitleLen = 5
	}

	var parts []string
	for i, view := range v.views {
		title := view.Title
		if len([]rune(title)) > maxTitleLen {
			title = string([]rune(title)[:maxTitleLen-1]) + "…"
		}
		shortcut := " \u2039" + v.keys[i] + "\u203a" // ‹key›

		var tab string
		if i == v.active {
			tab = activeStyle.Render(title) + activeKeyStyle.Render(shortcut)
		} else {
			tab = inactiveStyle.Render(title + shortcut)
		}
		parts = append(parts, tab)
	}

	sep := sepStyle.Render("  │  ")
	content := strings.Join(parts, sep)

	bar := lipgloss.NewStyle().
		Width(width).
		Background(lipgloss.ANSIColor(0)).
		Render(" " + content)

	return bar
}
```

- [ ] **Step 4: Run tests**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go test ./internal/tui/ -v -run "TestViewBar|TestNewViewBar"`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/views.go internal/tui/views_test.go
git commit -m "feat: add ViewBar model with tab bar rendering and key assignment"
```

---

### Task 5: Add View Keybindings to KeyMap

**Files:**
- Modify: `internal/tui/keys.go`

- [ ] **Step 1: Add ViewNext and ViewPrev bindings to KeyMap**

In `internal/tui/keys.go`, add to the `KeyMap` struct:

```go
ViewNext, ViewPrev key.Binding
```

In `NewKeyMap`, add:

```go
ViewNext: key.NewBinding(key.WithKeys(cfg.Views.NextView), key.WithHelp(cfg.Views.NextView, "next view")),
ViewPrev: key.NewBinding(key.WithKeys(cfg.Views.PrevView), key.WithHelp(cfg.Views.PrevView, "prev view")),
```

- [ ] **Step 2: Verify build**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go build ./internal/tui/`
Expected: Success

- [ ] **Step 3: Commit**

```bash
git add internal/tui/keys.go
git commit -m "feat: add ViewNext and ViewPrev keybindings"
```

---

### Task 6: Integrate Views into App

This is the largest task. It wires everything together.

**Files:**
- Modify: `internal/tui/app.go`

- [ ] **Step 1: Add view-related fields to App struct**

Add to the `App` struct:

```go
// Views
viewBar     ViewBar
currentUser string // resolved Trello username for @me
```

- [ ] **Step 2: Initialize ViewBar in NewApp**

In `NewApp`, after creating the app struct, add:

```go
a.viewBar = NewViewBar(cfg.Views)
```

- [ ] **Step 3: Add FetchCurrentUser to Init**

Add a new message type at the top of app.go:

```go
type CurrentUserMsg struct {
	Username string
}

type CurrentUserErrMsg struct {
	Err error
}
```

Add a fetch method:

```go
func (a App) fetchCurrentUserCmd() tea.Cmd {
	return func() tea.Msg {
		user, err := a.client.FetchCurrentUser()
		if err != nil {
			return CurrentUserErrMsg{Err: err}
		}
		return CurrentUserMsg{Username: user.Username}
	}
}
```

Update `Init()` to fire both the board fetch and user fetch in parallel using `tea.Batch`:

```go
func (a App) Init() tea.Cmd {
	userCmd := a.fetchCurrentUserCmd()

	boardID := a.config.Board.ID
	if boardID == "" && a.config.Board.Name != "" {
		return tea.Batch(a.resolveBoardCmd(a.config.Board.Name), userCmd)
	}
	if boardID == "" {
		return tea.Batch(func() tea.Msg {
			return BoardFetchErrMsg{Err: fmt.Errorf("no board configured — use --board or --board-id, or set board.id in config")}
		}, userCmd)
	}
	return tea.Batch(a.fetchBoardCmd(boardID), userCmd)
}
```

- [ ] **Step 4: Handle CurrentUserMsg in Update**

Add cases in the `Update` switch:

```go
case CurrentUserMsg:
	a.currentUser = msg.Username
	// Re-apply active view's filter now that @me can resolve
	if a.boardReady {
		viewCfg := a.viewBar.ActiveConfig()
		if viewCfg.Filter != "" {
			a.searchInput.SetValue(viewCfg.Filter)
			f := ParseFilter(viewCfg.Filter, a.currentUser)
			a.board.ApplyFilter(f)
		}
	}
	return a, nil

case CurrentUserErrMsg:
	// Graceful degradation — @me won't resolve but app continues
	return a, nil
```

- [ ] **Step 5: Apply active view filter on BoardFetchedMsg**

In the `BoardFetchedMsg` handler, after `a.board = NewBoardModel(...)` and `a.updateSearchWidth()`, replace the existing filter re-application block with view-aware logic:

```go
// Apply active view's filter
viewCfg := a.viewBar.ActiveConfig()
if viewCfg.Filter != "" {
	a.searchInput.SetValue(viewCfg.Filter)
	f := ParseFilter(viewCfg.Filter, a.currentUser)
	a.board.ApplyFilter(f)
} else if a.searchInput.Value() != "" {
	f := ParseFilter(a.searchInput.Value(), a.currentUser)
	a.board.ApplyFilter(f)
}
```

Also apply view GUI overrides for `ShowDetailPanel`:

```go
showDetail := a.config.GUI.ShowDetailPanel
if viewCfg.ShowDetailPanel != nil {
	showDetail = *viewCfg.ShowDetailPanel
}
a.detail.open = false
a.detail.cardID = ""
if showDetail && a.width >= 80 {
```

- [ ] **Step 6: Update all remaining ParseFilter calls to pass currentUser**

Replace all `ParseFilter(..., "")` calls added in Task 3 with `ParseFilter(..., a.currentUser)`:

Search for `ParseFilter(` in `app.go` and update each call site.

- [ ] **Step 7: Add view switching to key handler**

Add `ViewNext`/`ViewPrev` cases inside the existing `switch` block (alongside other keybinding cases like `Quit`, `Help`, etc.):

```go
case matchKey(msg, a.keyMap.ViewNext):
	if a.boardReady {
		a.viewBar.Next()
		return a, a.applyActiveView()
	}

case matchKey(msg, a.keyMap.ViewPrev):
	if a.boardReady {
		a.viewBar.Prev()
		return a, a.applyActiveView()
	}
```

Then, AFTER the main `switch` block but BEFORE the final `a.board.Update(msg)` fallthrough (around line 658), add direct-jump key handling. This ensures standard keybindings (quit, help, refresh, etc.) always take precedence over view direct-jump keys:

```go
// Direct-jump view switching — checked last so it never shadows standard keys
if a.boardReady && !a.showPalette && !a.showPrompt && !a.searchFocused && !a.showMemberModal && !a.showLabelModal {
	if a.viewBar.SelectByKey(msg.String()) {
		return a, a.applyActiveView()
	}
}
```

- [ ] **Step 8: Implement applyActiveView helper**

Add to `app.go`:

```go
// applyActiveView applies the active view's filter and GUI overrides.
func (a *App) applyActiveView() tea.Cmd {
	viewCfg := a.viewBar.ActiveConfig()

	// Reset search bar to view's base filter
	a.searchInput.SetValue(viewCfg.Filter)
	if viewCfg.Filter != "" {
		f := ParseFilter(viewCfg.Filter, a.currentUser)
		a.board.ApplyFilter(f)
	} else {
		a.board.ClearFilter()
	}

	// Apply GUI overrides
	if viewCfg.ColumnWidth != nil {
		a.board.minColWidth = *viewCfg.ColumnWidth
		a.board.ResizeColumns()
	}

	if viewCfg.ShowDetailPanel != nil {
		shouldShow := *viewCfg.ShowDetailPanel
		if shouldShow && !a.detail.open && a.width >= 80 {
			a.detail.open = true
			if card, _, ok := a.board.SelectedCard(); ok {
				a.detail.SetCard(card)
				a.updateDetailLayout()
				if a.detail.NeedsFetch() {
					a.detail.MarkLoading()
					return a.fetchDetailData()
				}
			} else {
				a.updateDetailLayout()
			}
		} else if !shouldShow && a.detail.open {
			a.detail.open = false
			a.detail.cardID = ""
			a.board.width = a.width
			a.board.height = a.height
			a.board.ResizeColumns()
			a.updateSearchWidth()
		}
	}

	// Note: showCardLabels override is a render-time concern — store it on the
	// board and check during column rendering. This can be deferred if card
	// label rendering doesn't yet consult a dynamic flag.

	return a.syncDetailAfterFilter()
}
```

- [ ] **Step 9: Update Escape behavior for view base filter**

In the search-focused handler, update the `esc` case:

```go
case "esc":
	a.searchFocused = false
	a.searchInput.Blur()
	// Reset to view's base filter instead of clearing completely
	viewCfg := a.viewBar.ActiveConfig()
	a.searchInput.SetValue(viewCfg.Filter)
	if viewCfg.Filter != "" {
		f := ParseFilter(viewCfg.Filter, a.currentUser)
		a.board.ApplyFilter(f)
	} else {
		a.board.ClearFilter()
	}
	fetchCmd := a.syncDetailAfterFilter()
	return a, fetchCmd
```

Also update the non-focused Escape handler (around line 649) to reset to view base filter:

```go
case msg.String() == "esc":
	if a.boardReady && !a.showPalette && !a.showPrompt {
		viewCfg := a.viewBar.ActiveConfig()
		if a.searchInput.Value() != viewCfg.Filter {
			// Reset to view's base filter
			a.searchInput.SetValue(viewCfg.Filter)
			if viewCfg.Filter != "" {
				f := ParseFilter(viewCfg.Filter, a.currentUser)
				a.board.ApplyFilter(f)
			} else {
				a.board.ClearFilter()
			}
			fetchCmd := a.syncDetailAfterFilter()
			return a, fetchCmd
		}
	}
```

- [ ] **Step 10: Render tab bar in View()**

In the `View()` method, in the `a.boardReady` block, add the view bar rendering before the search bar:

```go
// Render view tab bar
viewBarContent := a.viewBar.View(a.board.width)
```

Then update the search bar and board rendering to include it. Change the `a.board.SetSearchBar(searchBar)` line to also include the view bar:

```go
a.board.SetSearchBar(viewBarContent + "\n" + searchBar)
```

Note: The view bar adds 1 line to the header. Since the view bar is always included in the `SetSearchBar` content (view bar + newline + search bar), `b.searchBar` will always be non-empty when views are active. Update `board.go View()`:

- The `headerLines` when `b.searchBar != ""` changes from `6` to `7` (view bar adds 1 line)
- The `headerLines` when `b.searchBar == ""` stays at `3` (this branch won't be hit once views are wired, but keep it correct for safety)
- `ResizeColumns`: change `height - 8` to `height - 9`
- `NewBoardModel`: change `height - 10` to `height - 11`

- [ ] **Step 11: Update board.go height calculations**

In `board.go`:

- `NewBoardModel`: change `height - 10` to `height - 11`
- `ResizeColumns`: change `height - 8` to `height - 9`
- `View()` `headerLines`: change `3` to `4` and `6` to `7`

- [ ] **Step 12: Add view keys to help screen**

In `renderHelp()`, add a "Views" section with cycle keys and per-view direct jump keys:

```go
{a.keyMap.ViewNext.Keys()[0] + "/" + a.keyMap.ViewPrev.Keys()[0], "Cycle views"},
```

Also add per-view direct jump keys dynamically:

```go
// After the static keys slice, add view-specific keys
for i, view := range a.viewBar.views {
	keys = append(keys, struct{ key, desc string }{
		a.viewBar.keys[i], "View: " + view.Title,
	})
}
```

- [ ] **Step 13: Verify full build and tests**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go build ./... && go test ./... -v`
Expected: All pass, build succeeds

- [ ] **Step 14: Commit**

```bash
git add internal/tui/app.go internal/tui/board.go
git commit -m "feat: integrate views with tab bar, view switching, @me resolution, and escape-to-view-base"
```

---

### Task 7: Manual Testing and Polish

- [ ] **Step 1: Run the app and verify default views render**

Run the app and confirm:
- Tab bar shows at top with "My Cards" and "All Cards"
- "My Cards" is active on startup (bold/bright)
- `@me` resolves to the current user's cards

- [ ] **Step 2: Test view switching**

- Press `1` or `m` for direct jump
- Press `v` to cycle forward, `V` to cycle backward
- Confirm search bar updates to view's base filter on switch

- [ ] **Step 3: Test filter appending and escape**

- While on "My Cards" view, focus search bar and add extra text
- Press Escape — should reset to `member:@me` (not clear completely)
- Switch to "All Cards", press Escape — should clear completely

- [ ] **Step 4: Fix any visual issues**

Adjust styling, spacing, or height calculations as needed based on actual terminal rendering.

- [ ] **Step 5: Final commit if any polish changes**

```bash
git add -A
git commit -m "fix: polish views tab bar rendering and height calculations"
```
