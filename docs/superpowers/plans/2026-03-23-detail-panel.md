# Detail Panel Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a toggleable detail panel to the right side of the board that shows card overview, comments, and checklists in tabbed views.

**Architecture:** A self-contained `DetailModel` Bubble Tea component in `internal/tui/detail.go` owns all panel state (tab index, cached data, viewport). `App` holds a `detail DetailModel` field, forwards messages, and detects card selection changes via compare-after-update. New Trello API methods lazily fetch comments and checklists with stale-response detection.

**Tech Stack:** Go, Bubble Tea v2 (`charm.land/bubbletea/v2`), Bubbles v2 (`charm.land/bubbles/v2/viewport`), Lip Gloss v2 (`charm.land/lipgloss/v2`), Trello REST API

---

## File Structure

| Action | File | Responsibility |
|--------|------|---------------|
| Create | `internal/tui/detail.go` | DetailModel component — panel state, tab rendering, viewport |
| Create | `internal/tui/detail_test.go` | Tests for DetailModel (tab cycling, card changes, stale detection) |
| Modify | `internal/trello/types.go` | Add Comment, Checklist, CheckItem types |
| Modify | `internal/trello/client.go` | Add FetchCardComments, FetchCardChecklists methods |
| Modify | `internal/trello/client_test.go` | Tests for new API methods |
| Modify | `internal/config/config.go` | Add DetailKeys to KeybindingConfig |
| Modify | `internal/config/config_test.go` | Test detail key defaults |
| Modify | `internal/tui/keys.go` | Add detail keybindings to KeyMap |
| Modify | `internal/tui/keys_test.go` | Test detail keybinding creation |
| Modify | `internal/tui/app.go` | Integrate DetailModel, handle detail keys, adjust layout |
| Modify | `internal/tui/app_test.go` | Test toggle, tab switching, layout split |
| Modify | `internal/tui/board.go` | No structural changes — width is set by App before View() |

---

### Task 1: Add Trello API Types

**Files:**
- Modify: `internal/trello/types.go`

- [ ] **Step 1: Add Comment, Checklist, and CheckItem types**

Add these types after the existing `CustomFieldValue` struct:

```go
// Comment represents a comment on a Trello card.
type Comment struct {
	ID     string
	Author Member
	Body   string
	Date   time.Time
}

// Checklist represents a checklist on a Trello card.
type Checklist struct {
	ID    string
	Name  string
	Items []CheckItem
}

// CheckItem represents a single item in a checklist.
type CheckItem struct {
	ID       string
	Name     string
	Complete bool
}
```

Also add `"time"` to the import block at the top of the file. The file currently has no imports, so add:

```go
import "time"
```

- [ ] **Step 2: Verify build**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go build ./...`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add internal/trello/types.go
git commit -m "feat: add Comment, Checklist, CheckItem types for detail panel"
```

---

### Task 2: Add Trello API Methods

**Files:**
- Modify: `internal/trello/client.go`
- Modify: `internal/trello/client_test.go`

- [ ] **Step 1: Write failing test for FetchCardComments**

Add to `internal/trello/client_test.go`:

```go
func TestFetchCardComments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/cards/card1/actions" {
			filter := r.URL.Query().Get("filter")
			if filter != "commentCard" {
				t.Errorf("expected filter=commentCard, got %q", filter)
			}
			resp := []map[string]interface{}{
				{
					"id":   "action1",
					"date": "2026-03-20T10:30:00.000Z",
					"data": map[string]interface{}{
						"text": "This is a comment",
					},
					"memberCreator": map[string]interface{}{
						"id":       "member1",
						"fullName": "Craig Thomas",
						"initials": "CT",
						"username": "craigt",
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

	comments, err := c.FetchCardComments("card1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	if comments[0].Body != "This is a comment" {
		t.Errorf("expected body 'This is a comment', got %q", comments[0].Body)
	}
	if comments[0].Author.FullName != "Craig Thomas" {
		t.Errorf("expected author 'Craig Thomas', got %q", comments[0].Author.FullName)
	}
	if comments[0].Date.Year() != 2026 {
		t.Errorf("expected year 2026, got %d", comments[0].Date.Year())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go test ./internal/trello/ -run TestFetchCardComments -v`
Expected: FAIL — `FetchCardComments` not defined

- [ ] **Step 3: Implement FetchCardComments**

Add to `internal/trello/client.go`, after the `ReorderCard` method. Also add `"time"` to the import block.

```go
// API response types for card actions (comments)
type apiAction struct {
	ID            string          `json:"id"`
	Date          string          `json:"date"`
	Data          apiActionData   `json:"data"`
	MemberCreator apiMember       `json:"memberCreator"`
}

type apiActionData struct {
	Text string `json:"text"`
}

// FetchCardComments retrieves comments on a card.
func (c *Client) FetchCardComments(cardID string) ([]Comment, error) {
	var actions []apiAction
	path := fmt.Sprintf("/1/cards/%s/actions?filter=commentCard&fields=data,date,idMemberCreator,memberCreator&memberCreator_fields=fullName,initials,username", cardID)
	if err := c.get(path, &actions); err != nil {
		return nil, err
	}

	comments := make([]Comment, 0, len(actions))
	for _, a := range actions {
		t, err := time.Parse(time.RFC3339, a.Date)
		if err != nil {
			t = time.Time{} // zero value on parse failure
		}
		comments = append(comments, Comment{
			ID:   a.ID,
			Body: a.Data.Text,
			Date: t,
			Author: Member{
				ID:       a.MemberCreator.ID,
				FullName: a.MemberCreator.FullName,
				Initials: a.MemberCreator.Initials,
				Username: a.MemberCreator.Username,
			},
		})
	}
	return comments, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go test ./internal/trello/ -run TestFetchCardComments -v`
Expected: PASS

- [ ] **Step 5: Write failing test for FetchCardChecklists**

Add to `internal/trello/client_test.go`:

```go
func TestFetchCardChecklists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/cards/card1/checklists" {
			resp := []map[string]interface{}{
				{
					"id":   "cl1",
					"name": "TODO",
					"checkItems": []map[string]interface{}{
						{"id": "ci1", "name": "First item", "state": "complete"},
						{"id": "ci2", "name": "Second item", "state": "incomplete"},
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

	checklists, err := c.FetchCardChecklists("card1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(checklists) != 1 {
		t.Fatalf("expected 1 checklist, got %d", len(checklists))
	}
	if checklists[0].Name != "TODO" {
		t.Errorf("expected name 'TODO', got %q", checklists[0].Name)
	}
	if len(checklists[0].Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(checklists[0].Items))
	}
	if !checklists[0].Items[0].Complete {
		t.Error("expected first item to be complete")
	}
	if checklists[0].Items[1].Complete {
		t.Error("expected second item to be incomplete")
	}
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go test ./internal/trello/ -run TestFetchCardChecklists -v`
Expected: FAIL — `FetchCardChecklists` not defined

- [ ] **Step 7: Implement FetchCardChecklists**

Add to `internal/trello/client.go`, after `FetchCardComments`:

```go
// API response types for checklists
type apiChecklist struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	CheckItems []apiCheckItem   `json:"checkItems"`
}

type apiCheckItem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

// FetchCardChecklists retrieves checklists for a card.
func (c *Client) FetchCardChecklists(cardID string) ([]Checklist, error) {
	var raw []apiChecklist
	path := fmt.Sprintf("/1/cards/%s/checklists?fields=name&checkItem_fields=name,state", cardID)
	if err := c.get(path, &raw); err != nil {
		return nil, err
	}

	checklists := make([]Checklist, 0, len(raw))
	for _, cl := range raw {
		checklist := Checklist{ID: cl.ID, Name: cl.Name}
		for _, ci := range cl.CheckItems {
			checklist.Items = append(checklist.Items, CheckItem{
				ID:       ci.ID,
				Name:     ci.Name,
				Complete: ci.State == "complete",
			})
		}
		checklists = append(checklists, checklist)
	}
	return checklists, nil
}
```

- [ ] **Step 8: Run test to verify it passes**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go test ./internal/trello/ -run TestFetchCardChecklists -v`
Expected: PASS

- [ ] **Step 9: Run all trello tests**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go test ./internal/trello/ -v`
Expected: ALL PASS

- [ ] **Step 10: Commit**

```bash
git add internal/trello/client.go internal/trello/client_test.go
git commit -m "feat: add FetchCardComments and FetchCardChecklists API methods"
```

---

### Task 3: Add Detail Keybinding Config

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `internal/tui/keys.go`
- Modify: `internal/tui/keys_test.go`

- [ ] **Step 1: Write failing test for detail key defaults**

Add to `internal/config/config_test.go`:

```go
func TestDefaultDetailKeys(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Keybinding.Detail.Toggle != "d" {
		t.Errorf("expected detail toggle 'd', got %q", cfg.Keybinding.Detail.Toggle)
	}
	if cfg.Keybinding.Detail.TabPrev != "[" {
		t.Errorf("expected detail tabPrev '[', got %q", cfg.Keybinding.Detail.TabPrev)
	}
	if cfg.Keybinding.Detail.TabNext != "]" {
		t.Errorf("expected detail tabNext ']', got %q", cfg.Keybinding.Detail.TabNext)
	}
	if cfg.Keybinding.Detail.ScrollDown != "ctrl+j" {
		t.Errorf("expected detail scrollDown 'ctrl+j', got %q", cfg.Keybinding.Detail.ScrollDown)
	}
	if cfg.Keybinding.Detail.ScrollUp != "ctrl+k" {
		t.Errorf("expected detail scrollUp 'ctrl+k', got %q", cfg.Keybinding.Detail.ScrollUp)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go test ./internal/config/ -run TestDefaultDetailKeys -v`
Expected: FAIL — `Detail` field does not exist

- [ ] **Step 3: Add DetailKeys config struct and defaults**

In `internal/config/config.go`, add the `DetailKeys` struct after `BoardKeys`:

```go
type DetailKeys struct {
	Toggle     string `mapstructure:"toggle"`
	TabPrev    string `mapstructure:"tabPrev"`
	TabNext    string `mapstructure:"tabNext"`
	ScrollDown string `mapstructure:"scrollDown"`
	ScrollUp   string `mapstructure:"scrollUp"`
}
```

Add `Detail DetailKeys \`mapstructure:"detail"\`` field to `KeybindingConfig`:

```go
type KeybindingConfig struct {
	Universal UniversalKeys `mapstructure:"universal"`
	Board     BoardKeys     `mapstructure:"board"`
	Detail    DetailKeys    `mapstructure:"detail"`
}
```

Add defaults in `DefaultConfig()`, inside the `Keybinding` block:

```go
Detail: DetailKeys{
	Toggle:     "d",
	TabPrev:    "[",
	TabNext:    "]",
	ScrollDown: "ctrl+j",
	ScrollUp:   "ctrl+k",
},
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go test ./internal/config/ -run TestDefaultDetailKeys -v`
Expected: PASS

- [ ] **Step 5: Add detail keybindings to KeyMap**

In `internal/tui/keys.go`, add five new fields to the `KeyMap` struct:

```go
DetailToggle, DetailTabPrev, DetailTabNext key.Binding
DetailScrollDown, DetailScrollUp          key.Binding
```

In `NewKeyMap`, add the bindings:

```go
DetailToggle:     key.NewBinding(key.WithKeys(cfg.Detail.Toggle), key.WithHelp(cfg.Detail.Toggle, "detail panel")),
DetailTabPrev:    key.NewBinding(key.WithKeys(cfg.Detail.TabPrev), key.WithHelp(cfg.Detail.TabPrev, "prev tab")),
DetailTabNext:    key.NewBinding(key.WithKeys(cfg.Detail.TabNext), key.WithHelp(cfg.Detail.TabNext, "next tab")),
DetailScrollDown: key.NewBinding(key.WithKeys(cfg.Detail.ScrollDown), key.WithHelp(cfg.Detail.ScrollDown, "scroll down")),
DetailScrollUp:   key.NewBinding(key.WithKeys(cfg.Detail.ScrollUp), key.WithHelp(cfg.Detail.ScrollUp, "scroll up")),
```

- [ ] **Step 6: Verify build**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go build ./...`
Expected: BUILD SUCCESS

- [ ] **Step 7: Run all config and keys tests**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go test ./internal/config/ ./internal/tui/ -v`
Expected: ALL PASS

- [ ] **Step 8: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go internal/tui/keys.go internal/tui/keys_test.go
git commit -m "feat: add detail panel keybinding config and KeyMap entries"
```

---

### Task 4: Create DetailModel Component

**Files:**
- Create: `internal/tui/detail.go`
- Create: `internal/tui/detail_test.go`

- [ ] **Step 1: Write failing tests for DetailModel**

Create `internal/tui/detail_test.go`:

```go
package tui

import (
	"strings"
	"testing"

	"github.com/craig006/tuillo/internal/config"
	"github.com/craig006/tuillo/internal/trello"
)

func newTestDetail() DetailModel {
	cfg := config.DefaultConfig()
	km := NewKeyMap(cfg.Keybinding)
	theme := NewTheme(cfg.GUI.Theme)
	return NewDetailModel(km, theme)
}

func TestDetailToggle(t *testing.T) {
	d := newTestDetail()
	if d.open {
		t.Error("expected panel to start closed")
	}
	d.Toggle()
	if !d.open {
		t.Error("expected panel to be open after toggle")
	}
	d.Toggle()
	if d.open {
		t.Error("expected panel to be closed after second toggle")
	}
}

func TestDetailTabCycling(t *testing.T) {
	d := newTestDetail()
	d.open = true
	if d.tab != 0 {
		t.Errorf("expected tab 0, got %d", d.tab)
	}
	d.NextTab()
	if d.tab != 1 {
		t.Errorf("expected tab 1, got %d", d.tab)
	}
	d.NextTab()
	if d.tab != 2 {
		t.Errorf("expected tab 2, got %d", d.tab)
	}
	d.NextTab()
	if d.tab != 0 {
		t.Errorf("expected tab to wrap to 0, got %d", d.tab)
	}
	d.PrevTab()
	if d.tab != 2 {
		t.Errorf("expected tab to wrap to 2, got %d", d.tab)
	}
}

func TestDetailSetCardClearsCache(t *testing.T) {
	d := newTestDetail()
	d.open = true
	d.comments = []trello.Comment{{ID: "old"}}
	d.commentsLoaded = true
	d.checklists = []trello.Checklist{{ID: "old"}}
	d.checklistsLoaded = true

	card := trello.Card{ID: "new-card", Name: "New Card"}
	d.SetCard(card)

	if d.cardID != "new-card" {
		t.Errorf("expected cardID 'new-card', got %q", d.cardID)
	}
	if d.commentsLoaded {
		t.Error("expected commentsLoaded to be false")
	}
	if d.checklistsLoaded {
		t.Error("expected checklistsLoaded to be false")
	}
	if len(d.comments) != 0 {
		t.Error("expected comments to be cleared")
	}
	if len(d.checklists) != 0 {
		t.Error("expected checklists to be cleared")
	}
}

func TestDetailOverviewView(t *testing.T) {
	d := newTestDetail()
	d.open = true
	d.SetSize(40, 20)
	card := trello.Card{
		ID:          "card1",
		Name:        "Test Card",
		Description: "A description",
		Labels:      []trello.Label{{Name: "Bug", Color: "red"}},
		Members:     []trello.Member{{FullName: "Craig Thomas"}},
	}
	d.SetCard(card)

	view := d.View()
	if !strings.Contains(view, "Test Card") {
		t.Error("expected view to contain card title")
	}
	if !strings.Contains(view, "Overview") {
		t.Error("expected view to contain 'Overview' tab")
	}
}

func TestDetailStaleResponseIgnored(t *testing.T) {
	d := newTestDetail()
	d.open = true
	d.SetCard(trello.Card{ID: "current"})

	// Simulate a stale comments response for a different card
	msg := CardCommentsMsg{CardID: "old-card", Comments: []trello.Comment{{ID: "c1", Body: "stale"}}}
	d.HandleCommentsMsg(msg)

	if d.commentsLoaded {
		t.Error("expected stale response to be ignored")
	}

	// Now a matching response
	msg = CardCommentsMsg{CardID: "current", Comments: []trello.Comment{{ID: "c2", Body: "fresh"}}}
	d.HandleCommentsMsg(msg)

	if !d.commentsLoaded {
		t.Error("expected matching response to be accepted")
	}
	if len(d.comments) != 1 || d.comments[0].Body != "fresh" {
		t.Error("expected comments to contain the fresh comment")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go test ./internal/tui/ -run TestDetail -v`
Expected: FAIL — `NewDetailModel` not defined

- [ ] **Step 3: Implement DetailModel**

Create `internal/tui/detail.go`:

```go
package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"

	"github.com/craig006/tuillo/internal/trello"
)

const (
	tabOverview   = 0
	tabComments   = 1
	tabChecklists = 2
	tabCount      = 3
)

var tabNames = [tabCount]string{"Overview", "Comments", "Checklist"}

// Fetch result messages — all include CardID for stale-response detection.

type CardCommentsMsg struct {
	CardID   string
	Comments []trello.Comment
}

type CardCommentsFetchErrMsg struct {
	CardID string
	Err    error
}

type CardChecklistsMsg struct {
	CardID     string
	Checklists []trello.Checklist
}

type CardChecklistsFetchErrMsg struct {
	CardID string
	Err    error
}

// DetailModel is a self-contained Bubble Tea component for the detail panel.
type DetailModel struct {
	open   bool
	tab    int
	cardID string

	card       trello.Card
	comments   []trello.Comment
	checklists []trello.Checklist

	commentsLoaded   bool
	checklistsLoaded bool
	loading          bool
	loadingErr       string

	viewport viewport.Model
	width    int
	height   int
	keyMap   KeyMap
	theme    Theme
}

func NewDetailModel(km KeyMap, theme Theme) DetailModel {
	vp := viewport.New(0, 0)
	return DetailModel{
		keyMap:   km,
		theme:    theme,
		viewport: vp,
	}
}

func (d *DetailModel) Toggle() {
	d.open = !d.open
}

func (d *DetailModel) NextTab() {
	d.tab = (d.tab + 1) % tabCount
}

func (d *DetailModel) PrevTab() {
	d.tab = (d.tab - 1 + tabCount) % tabCount
}

// SetCard updates the displayed card and clears cached data.
// Returns a tea.Cmd if the active tab needs a fetch.
func (d *DetailModel) SetCard(card trello.Card) tea.Cmd {
	d.card = card
	d.cardID = card.ID
	d.comments = nil
	d.checklists = nil
	d.commentsLoaded = false
	d.checklistsLoaded = false
	d.loading = false
	d.loadingErr = ""
	return d.fetchForActiveTab()
}

func (d *DetailModel) SetSize(width, height int) {
	d.width = width
	d.height = height
	// Reserve 3 lines for border (top + bottom) and tab bar
	vpHeight := height - 4
	if vpHeight < 1 {
		vpHeight = 1
	}
	vpWidth := width - 4 // border + padding
	if vpWidth < 1 {
		vpWidth = 1
	}
	d.viewport.SetWidth(vpWidth)
	d.viewport.SetHeight(vpHeight)
}

func (d *DetailModel) HandleCommentsMsg(msg CardCommentsMsg) {
	if msg.CardID != d.cardID {
		return // stale response
	}
	d.comments = msg.Comments
	d.commentsLoaded = true
	d.loading = false
	d.loadingErr = ""
}

func (d *DetailModel) HandleCommentsFetchErr(msg CardCommentsFetchErrMsg) {
	if msg.CardID != d.cardID {
		return
	}
	d.loading = false
	d.loadingErr = fmt.Sprintf("Failed to load comments: %v", msg.Err)
}

func (d *DetailModel) HandleChecklistsMsg(msg CardChecklistsMsg) {
	if msg.CardID != d.cardID {
		return // stale response
	}
	d.checklists = msg.Checklists
	d.checklistsLoaded = true
	d.loading = false
	d.loadingErr = ""
}

func (d *DetailModel) HandleChecklistsFetchErr(msg CardChecklistsFetchErrMsg) {
	if msg.CardID != d.cardID {
		return
	}
	d.loading = false
	d.loadingErr = fmt.Sprintf("Failed to load checklists: %v", msg.Err)
}

// fetchForActiveTab returns a fetch command if the current tab has uncached data.
// Does NOT return a command for overview (tab 0) — that uses card data already available.
func (d *DetailModel) fetchForActiveTab() tea.Cmd {
	switch d.tab {
	case tabComments:
		if !d.commentsLoaded {
			d.loading = true
			return nil // caller provides the actual fetch cmd via App
		}
	case tabChecklists:
		if !d.checklistsLoaded {
			d.loading = true
			return nil
		}
	}
	return nil
}

// NeedsFetch returns true if the active tab needs data fetched.
func (d *DetailModel) NeedsFetch() bool {
	switch d.tab {
	case tabComments:
		return !d.commentsLoaded
	case tabChecklists:
		return !d.checklistsLoaded
	}
	return false
}

// Update handles viewport scrolling messages.
func (d DetailModel) Update(msg tea.Msg) (DetailModel, tea.Cmd) {
	var cmd tea.Cmd
	d.viewport, cmd = d.viewport.Update(msg)
	return d, cmd
}

// View renders the detail panel with border, tab bar, and content.
func (d DetailModel) View() string {
	if !d.open {
		return ""
	}

	contentWidth := d.width - 4 // border + padding
	if contentWidth < 1 {
		contentWidth = 1
	}

	// Render content based on active tab
	var content string
	if d.cardID == "" {
		content = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Render("No card selected")
	} else if d.loadingErr != "" {
		content = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(1)).Render(d.loadingErr)
	} else {
		switch d.tab {
		case tabOverview:
			content = d.renderOverview(contentWidth)
		case tabComments:
			content = d.renderComments(contentWidth)
		case tabChecklists:
			content = d.renderChecklists(contentWidth)
		}
	}

	d.viewport.SetContent(content)

	// Build border with tab bar
	borderColor := d.theme.ActiveBorder.GetForeground()
	border := lipgloss.RoundedBorder()

	// Build tab bar string
	var tabBar string
	for i, name := range tabNames {
		if i == d.tab {
			tabBar += lipgloss.NewStyle().Bold(true).Foreground(borderColor).Render(" "+name+" ")
		} else {
			tabBar += lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Render(" "+name+" ")
		}
	}

	// Render panel with border
	style := lipgloss.NewStyle().
		Width(d.width - 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Height(d.height - 2)

	rendered := style.Render(d.viewport.View())
	lines := strings.Split(rendered, "\n")

	// Replace top border with tab bar
	if len(lines) > 0 {
		origWidth := lipgloss.Width(lines[0])
		tabBarWidth := lipgloss.Width(tabBar)
		trailing := origWidth - 2 - 1 - tabBarWidth // corners + leading dash
		if trailing < 0 {
			trailing = 0
		}
		borderStyle := lipgloss.NewStyle().Foreground(borderColor)
		lines[0] = borderStyle.Render(border.TopLeft+border.Top) +
			tabBar +
			borderStyle.Render(strings.Repeat(border.Top, trailing)+border.TopRight)
		rendered = strings.Join(lines, "\n")
	}

	return rendered
}

func (d DetailModel) renderOverview(width int) string {
	var sections []string

	// Title
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(15)).Render(d.card.Name)
	sections = append(sections, title)

	// Labels
	if len(d.card.Labels) > 0 {
		var labels []string
		for _, lbl := range d.card.Labels {
			ansiColor, ok := trelloColorToANSI[lbl.Color]
			if !ok {
				ansiColor = 7
			}
			indicator := lipgloss.NewStyle().Foreground(ansiColor).Render("⏺")
			name := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(7)).Render(" " + lbl.Name)
			labels = append(labels, indicator+name)
		}
		sections = append(sections, strings.Join(labels, "  "))
	}

	// Assignees
	if len(d.card.Members) > 0 {
		var names []string
		for _, m := range d.card.Members {
			names = append(names, m.FullName)
		}
		assigneeLabel := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Render("Assignees: ")
		assigneeText := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(7)).Render(strings.Join(names, ", "))
		sections = append(sections, assigneeLabel+assigneeText)
	}

	// Description
	sections = append(sections, "") // spacer
	if d.card.Description != "" {
		desc := wordWrap(d.card.Description, width)
		sections = append(sections, lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(7)).Render(desc))
	} else {
		sections = append(sections, lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Render("No description."))
	}

	return strings.Join(sections, "\n")
}

func (d DetailModel) renderComments(width int) string {
	if d.loading {
		return lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Render("Loading comments...")
	}
	if len(d.comments) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Render("No comments.")
	}

	var sections []string
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))
	boldStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(15))
	bodyStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(7))

	for i, c := range d.comments {
		if i > 0 {
			sections = append(sections, dimStyle.Render(strings.Repeat("─", width)))
		}
		dateStr := c.Date.Format("2006-01-02")
		header := boldStyle.Render(c.Author.FullName) + " " + dimStyle.Render("("+dateStr+")")
		body := wordWrap(c.Body, width)
		sections = append(sections, header+"\n"+bodyStyle.Render(body))
	}

	return strings.Join(sections, "\n")
}

func (d DetailModel) renderChecklists(width int) string {
	if d.loading {
		return lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Render("Loading checklists...")
	}
	if len(d.checklists) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Render("No checklists.")
	}

	var sections []string
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))
	boldStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(15))
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(7))
	showHeaders := len(d.checklists) > 1

	for _, cl := range d.checklists {
		if showHeaders {
			sections = append(sections, boldStyle.Render(cl.Name))
		}
		for _, item := range cl.Items {
			var line string
			if item.Complete {
				line = dimStyle.Render("[x] " + item.Name)
			} else {
				line = normalStyle.Render("[ ] " + item.Name)
			}
			sections = append(sections, line)
		}
		if showHeaders {
			sections = append(sections, "") // spacer between checklists
		}
	}

	return strings.Join(sections, "\n")
}

// wordWrap wraps text to the given width.
func wordWrap(text string, width int) string {
	if width <= 0 {
		return text
	}
	var result strings.Builder
	for _, paragraph := range strings.Split(text, "\n") {
		if result.Len() > 0 {
			result.WriteByte('\n')
		}
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			continue
		}
		lineLen := 0
		for i, word := range words {
			wordLen := len([]rune(word))
			if i > 0 && lineLen+1+wordLen > width {
				result.WriteByte('\n')
				lineLen = 0
			} else if i > 0 {
				result.WriteByte(' ')
				lineLen++
			}
			result.WriteString(word)
			lineLen += wordLen
		}
	}
	return result.String()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go test ./internal/tui/ -run TestDetail -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/detail.go internal/tui/detail_test.go
git commit -m "feat: add DetailModel component with tabbed views"
```

---

### Task 5: Integrate DetailModel into App

**Files:**
- Modify: `internal/tui/app.go`
- Modify: `internal/tui/app_test.go`

- [ ] **Step 1: Write failing tests for detail panel integration**

Add to `internal/tui/app_test.go`:

```go
func TestDetailToggleKey(t *testing.T) {
	cfg := config.DefaultConfig()
	client := trello.NewClient("key", "token")
	app := NewApp(client, cfg)

	// Simulate board loaded
	board := makeTestBoard(3)
	app.boardReady = true
	app.board = NewBoardModel(board, cfg, 80, 24)

	// Press 'd' to toggle detail open
	msg := tea.KeyPressMsg{Key: tea.Key{Code: 'd'}}
	result, _ := app.Update(msg)
	a := result.(App)
	if !a.detail.open {
		t.Error("expected detail panel to be open after pressing 'd'")
	}

	// Press 'd' again to close
	result, _ = a.Update(msg)
	a = result.(App)
	if a.detail.open {
		t.Error("expected detail panel to be closed after pressing 'd' again")
	}
}

func TestDetailLayoutSplit(t *testing.T) {
	cfg := config.DefaultConfig()
	client := trello.NewClient("key", "token")
	app := NewApp(client, cfg)
	app.width = 100
	app.height = 30

	board := makeTestBoard(3)
	app.boardReady = true
	app.board = NewBoardModel(board, cfg, 100, 26)

	// Toggle detail open via the 'd' key — this triggers updateDetailLayout in Update
	msg := tea.KeyPressMsg{Key: tea.Key{Code: 'd'}}
	result, _ := app.Update(msg)
	a := result.(App)

	// When detail is open, board should use 60% width
	expectedBoardWidth := 100 * 60 / 100
	if a.board.width != expectedBoardWidth {
		t.Errorf("expected board width %d, got %d", expectedBoardWidth, a.board.width)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go test ./internal/tui/ -run "TestDetailToggleKey|TestDetailLayoutSplit" -v`
Expected: FAIL — `detail` field does not exist on App

- [ ] **Step 3: Add detail field to App and integrate**

In `internal/tui/app.go`:

**Add `detail DetailModel` field to App struct** (after `boardReady bool`):

```go
detail DetailModel
```

**Initialize in NewApp** — add after `loading: true,`:

```go
detail: NewDetailModel(km, NewTheme(cfg.GUI.Theme)),
```

**Handle detail keybindings** — in the `case tea.KeyPressMsg` switch, inside the `switch` block after `case matchKey(msg, a.keyMap.MoveCardDown):`, add:

```go
case matchKey(msg, a.keyMap.DetailToggle):
	if a.boardReady {
		a.detail.Toggle()
		if a.detail.open {
			// Load current card if panel just opened
			if card, _, ok := a.board.SelectedCard(); ok {
				cmd := a.detail.SetCard(card)
				a.updateDetailLayout()
				if a.detail.NeedsFetch() {
					return a, tea.Batch(cmd, a.fetchDetailData())
				}
				return a, cmd
			}
			a.updateDetailLayout()
		} else {
			// Restore full width
			a.board.width = a.width
			a.board.height = a.height - 4
			a.board.ResizeColumns()
		}
		return a, nil
	}

case matchKey(msg, a.keyMap.DetailTabNext):
	if a.boardReady && a.detail.open {
		a.detail.NextTab()
		if a.detail.NeedsFetch() {
			return a, a.fetchDetailData()
		}
		return a, nil
	}

case matchKey(msg, a.keyMap.DetailTabPrev):
	if a.boardReady && a.detail.open {
		a.detail.PrevTab()
		if a.detail.NeedsFetch() {
			return a, a.fetchDetailData()
		}
		return a, nil
	}

case matchKey(msg, a.keyMap.DetailScrollDown):
	if a.boardReady && a.detail.open {
		d, cmd := a.detail.Update(tea.KeyPressMsg{Key: tea.Key{Code: 'j'}})
		a.detail = d
		return a, cmd
	}

case matchKey(msg, a.keyMap.DetailScrollUp):
	if a.boardReady && a.detail.open {
		d, cmd := a.detail.Update(tea.KeyPressMsg{Key: tea.Key{Code: 'k'}})
		a.detail = d
		return a, cmd
	}
```

**Handle fetch result messages** — add cases in the outer `switch msg := msg.(type)` block:

```go
case CardCommentsMsg:
	a.detail.HandleCommentsMsg(msg)
	return a, nil

case CardCommentsFetchErrMsg:
	a.detail.HandleCommentsFetchErr(msg)
	return a, nil

case CardChecklistsMsg:
	a.detail.HandleChecklistsMsg(msg)
	return a, nil

case CardChecklistsFetchErrMsg:
	a.detail.HandleChecklistsFetchErr(msg)
	return a, nil
```

**Detect card selection changes** — after the existing board update forwarding block (`if a.boardReady { var cmd tea.Cmd; a.board, cmd = a.board.Update(msg); return a, cmd }`), modify it to check for card changes:

```go
if a.boardReady {
	var cmd tea.Cmd
	a.board, cmd = a.board.Update(msg)
	// Detect card selection change when panel is open
	if a.detail.open {
		if card, _, ok := a.board.SelectedCard(); ok && card.ID != a.detail.cardID {
			fetchCmd := a.detail.SetCard(card)
			if a.detail.NeedsFetch() {
				return a, tea.Batch(cmd, fetchCmd, a.fetchDetailData())
			}
			return a, tea.Batch(cmd, fetchCmd)
		}
	}
	return a, cmd
}
```

**Handle board refresh** — in the `BoardFetchedMsg` case, close the detail panel:

```go
case BoardFetchedMsg:
	a.loading = false
	a.boardReady = true
	a.board = NewBoardModel(msg.Board, a.config, a.width, a.height-4)
	a.status = fmt.Sprintf("%s — %s", msg.Board.Name, a.board.PositionIndicator())
	// Close detail panel on refresh
	a.detail.open = false
	a.detail.cardID = ""
	return a, nil
```

**Handle window resize with detail panel** — update the `tea.WindowSizeMsg` case:

```go
case tea.WindowSizeMsg:
	a.width = msg.Width
	a.height = msg.Height
	if a.boardReady {
		if a.detail.open {
			a.updateDetailLayout()
		} else {
			a.board.width = msg.Width
			a.board.height = msg.Height - 4
			a.board.ResizeColumns()
		}
	}
	return a, nil
```

**Add helper methods** to App:

```go
func (a *App) updateDetailLayout() {
	boardWidth := a.width * 60 / 100
	panelWidth := a.width - boardWidth
	a.board.width = boardWidth
	a.board.height = a.height - 4
	a.board.ResizeColumns()
	a.detail.SetSize(panelWidth, a.height-2)
}

func (a App) fetchDetailData() tea.Cmd {
	cardID := a.detail.cardID
	switch a.detail.tab {
	case tabComments:
		return func() tea.Msg {
			comments, err := a.client.FetchCardComments(cardID)
			if err != nil {
				return CardCommentsFetchErrMsg{CardID: cardID, Err: err}
			}
			return CardCommentsMsg{CardID: cardID, Comments: comments}
		}
	case tabChecklists:
		return func() tea.Msg {
			checklists, err := a.client.FetchCardChecklists(cardID)
			if err != nil {
				return CardChecklistsFetchErrMsg{CardID: cardID, Err: err}
			}
			return CardChecklistsMsg{CardID: cardID, Checklists: checklists}
		}
	}
	return nil
}
```

**Update View()** — modify the board rendering in View() to include detail panel:

```go
} else if a.boardReady {
	if a.detail.open {
		content = lipgloss.JoinHorizontal(lipgloss.Top, a.board.View(), a.detail.View())
	} else {
		content = a.board.View()
	}
}
```

**Update help screen** — add detail keybindings to `renderHelp()`:

```go
{a.keyMap.DetailToggle.Keys()[0], "Toggle detail panel"},
{a.keyMap.DetailTabPrev.Keys()[0] + "/" + a.keyMap.DetailTabNext.Keys()[0], "Switch detail tab"},
{a.keyMap.DetailScrollDown.Keys()[0], "Scroll detail down"},
{a.keyMap.DetailScrollUp.Keys()[0], "Scroll detail up"},
```

- [ ] **Step 4: Verify build**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go build ./...`
Expected: BUILD SUCCESS

- [ ] **Step 5: Run all tests**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go test ./... -v`
Expected: ALL PASS

- [ ] **Step 6: Manual smoke test**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go run .`

Test:
1. Press `d` — detail panel opens on right showing current card's overview
2. Press `j`/`k` — card selection changes, detail panel updates
3. Press `]` — switches to Comments tab, shows "Loading comments..." then content
4. Press `]` — switches to Checklist tab
5. Press `[` — back to Comments
6. Press `d` — panel closes, board returns to full width
7. Press `r` — board refreshes, panel stays closed

- [ ] **Step 7: Commit**

```bash
git add internal/tui/app.go internal/tui/app_test.go
git commit -m "feat: integrate detail panel into app with keybindings and layout split"
```

---

### Task 6: Edge Cases and Polish

**Files:**
- Modify: `internal/tui/app.go`
- Modify: `internal/tui/detail.go`

- [ ] **Step 1: Handle narrow terminal**

In `internal/tui/app.go`, in the `DetailToggle` handler, add a width check:

```go
case matchKey(msg, a.keyMap.DetailToggle):
	if a.boardReady {
		if !a.detail.open && a.width < 80 {
			a.status = "Terminal too narrow for detail panel"
			return a, nil
		}
		// ... rest of toggle logic
```

- [ ] **Step 2: Handle empty board (no card selected)**

The `SetCard` path already handles this — if `SelectedCard()` returns `ok=false`, the panel shows "No card selected". Verify this works by checking the `cardID == ""` case in `detail.View()`.

- [ ] **Step 3: Run all tests**

Run: `cd /Users/craig/GitHub/craig006/tuillo/main && go test ./... -v`
Expected: ALL PASS

- [ ] **Step 4: Commit**

```bash
git add internal/tui/app.go internal/tui/detail.go
git commit -m "feat: add edge case handling for narrow terminal and empty boards"
```
