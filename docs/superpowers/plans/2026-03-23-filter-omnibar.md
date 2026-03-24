# Filter/Search Omni Bar Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a filter/search omni bar that filters board cards by text, member, and label — with keyboard text input and multiselect modals.

**Architecture:** The search bar text is the single source of truth for all filters. A parser extracts `member:` and `label:` tokens from the text; the remainder is a substring match against card names. The board model holds a `Filter` struct and rebuilds each column's visible items when filters change. Member/label modals are centered overlays that inject/remove tokens from the search bar.

**Tech Stack:** Go, Bubble Tea v2 (`charm.land/bubbletea/v2`), Bubbles v2 (`charm.land/bubbles/v2/textinput`, `charm.land/bubbles/v2/list`), Lip Gloss v2 (`charm.land/lipgloss/v2`)

**Spec:** `docs/superpowers/specs/2026-03-23-filter-omnibar-design.md`

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/tui/filter.go` | Create | `Filter` struct, `ParseFilter()`, `MatchesCard()`, `BuildFilterText()` |
| `internal/tui/filter_test.go` | Create | Tests for parsing and matching |
| `internal/tui/multiselect.go` | Create | `MultiSelectModel` — reusable checkbox list modal |
| `internal/tui/multiselect_test.go` | Create | Tests for multiselect toggling |
| `internal/tui/board.go` | Modify | Add filter field, apply filter in `View()` and `rebuildColumnItems()` |
| `internal/tui/app.go` | Modify | Search bar rendering, modal state, keybindings, filter application |
| `internal/tui/keys.go` | Modify | Add `FilterFocus`, `FilterMembers`, `FilterLabels` bindings |
| `internal/config/config.go` | Modify | Add `FilterKeys` config struct with defaults |

---

### Task 1: Filter Parser

Create the filter parsing and matching logic as a standalone unit with no UI dependencies.

**Files:**
- Create: `internal/tui/filter.go`
- Create: `internal/tui/filter_test.go`

- [ ] **Step 1: Write tests for ParseFilter**

```go
// internal/tui/filter_test.go
package tui

import (
	"testing"

	"github.com/craig006/tuillo/internal/trello"
)

func TestParseFilterTextOnly(t *testing.T) {
	f := ParseFilter("fix door")
	if f.Text != "fix door" {
		t.Errorf("expected text 'fix door', got %q", f.Text)
	}
	if len(f.Members) != 0 {
		t.Errorf("expected no members, got %v", f.Members)
	}
	if len(f.Labels) != 0 {
		t.Errorf("expected no labels, got %v", f.Labels)
	}
}

func TestParseFilterMemberToken(t *testing.T) {
	f := ParseFilter("member:craig fix")
	if f.Text != "fix" {
		t.Errorf("expected text 'fix', got %q", f.Text)
	}
	if len(f.Members) != 1 || f.Members[0] != "craig" {
		t.Errorf("expected members [craig], got %v", f.Members)
	}
}

func TestParseFilterLabelToken(t *testing.T) {
	f := ParseFilter("label:Bug label:Design")
	if len(f.Labels) != 2 {
		t.Errorf("expected 2 labels, got %v", f.Labels)
	}
}

func TestParseFilterQuotedValue(t *testing.T) {
	f := ParseFilter(`member:"Craig Smith" fix`)
	if len(f.Members) != 1 || f.Members[0] != "Craig Smith" {
		t.Errorf("expected members [Craig Smith], got %v", f.Members)
	}
	if f.Text != "fix" {
		t.Errorf("expected text 'fix', got %q", f.Text)
	}
}

func TestParseFilterEmpty(t *testing.T) {
	f := ParseFilter("")
	if !f.IsEmpty() {
		t.Error("expected empty filter")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -run TestParseFilter -v`
Expected: FAIL — `ParseFilter` undefined

- [ ] **Step 3: Implement Filter struct and ParseFilter**

```go
// internal/tui/filter.go
package tui

import (
	"strings"
)

// Filter holds the parsed filter state.
type Filter struct {
	Text    string
	Members []string
	Labels  []string
}

// IsEmpty returns true if no filters are active.
func (f Filter) IsEmpty() bool {
	return f.Text == "" && len(f.Members) == 0 && len(f.Labels) == 0
}

// ParseFilter parses a search string into structured filter components.
// Recognized tokens: member:<value>, label:<value>. Quoted values supported.
// Everything else becomes the text search.
func ParseFilter(input string) Filter {
	var f Filter
	var textParts []string

	tokens := tokenize(input)
	for _, tok := range tokens {
		lower := strings.ToLower(tok)
		if strings.HasPrefix(lower, "member:") {
			val := tok[len("member:"):]
			val = strings.Trim(val, `"`)
			if val != "" {
				f.Members = append(f.Members, val)
			}
		} else if strings.HasPrefix(lower, "label:") {
			val := tok[len("label:"):]
			val = strings.Trim(val, `"`)
			if val != "" {
				f.Labels = append(f.Labels, val)
			}
		} else {
			textParts = append(textParts, tok)
		}
	}

	f.Text = strings.TrimSpace(strings.Join(textParts, " "))
	return f
}

// tokenize splits input into tokens, respecting quoted values after member:/label: prefixes.
func tokenize(input string) []string {
	var tokens []string
	i := 0
	runes := []rune(input)
	for i < len(runes) {
		// Skip whitespace
		if runes[i] == ' ' {
			i++
			continue
		}
		start := i
		// Check for member: or label: prefix with quoted value
		rest := string(runes[i:])
		lowerRest := strings.ToLower(rest)
		if strings.HasPrefix(lowerRest, "member:\"") || strings.HasPrefix(lowerRest, "label:\"") {
			colonIdx := strings.Index(rest, ":")
			i += colonIdx + 1 // past the colon
			if i < len(runes) && runes[i] == '"' {
				i++ // past opening quote
				end := i
				for end < len(runes) && runes[end] != '"' {
					end++
				}
				prefix := string(runes[start : start+colonIdx+1])
				val := string(runes[i:end])
				tokens = append(tokens, prefix+`"`+val+`"`)
				if end < len(runes) {
					end++ // past closing quote
				}
				i = end
				continue
			}
		}
		// Regular token (until next space)
		for i < len(runes) && runes[i] != ' ' {
			i++
		}
		tokens = append(tokens, string(runes[start:i]))
	}
	return tokens
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run TestParseFilter -v`
Expected: PASS

- [ ] **Step 5: Write tests for MatchesCard**

```go
func TestMatchesCardTextMatch(t *testing.T) {
	card := trello.Card{Name: "Fix the back door"}
	f := Filter{Text: "back"}
	if !f.MatchesCard(card) {
		t.Error("expected card to match text filter")
	}
}

func TestMatchesCardTextNoMatch(t *testing.T) {
	card := trello.Card{Name: "Fix the back door"}
	f := Filter{Text: "window"}
	if f.MatchesCard(card) {
		t.Error("expected card not to match text filter")
	}
}

func TestMatchesCardMemberMatch(t *testing.T) {
	card := trello.Card{
		Members: []trello.Member{{Username: "craig", FullName: "Craig Smith"}},
	}
	f := Filter{Members: []string{"craig"}}
	if !f.MatchesCard(card) {
		t.Error("expected card to match member filter")
	}
}

func TestMatchesCardMemberByFullName(t *testing.T) {
	card := trello.Card{
		Members: []trello.Member{{Username: "craig006", FullName: "Craig Smith"}},
	}
	f := Filter{Members: []string{"Craig Smith"}}
	if !f.MatchesCard(card) {
		t.Error("expected card to match member by full name")
	}
}

func TestMatchesCardLabelMatch(t *testing.T) {
	card := trello.Card{
		Labels: []trello.Label{{Name: "Bug"}},
	}
	f := Filter{Labels: []string{"bug"}}
	if !f.MatchesCard(card) {
		t.Error("expected card to match label filter (case-insensitive)")
	}
}

func TestMatchesCardAndLogic(t *testing.T) {
	card := trello.Card{
		Name:    "Fix login bug",
		Members: []trello.Member{{Username: "craig"}},
		Labels:  []trello.Label{{Name: "Bug"}},
	}
	f := Filter{Text: "login", Members: []string{"craig"}, Labels: []string{"Bug"}}
	if !f.MatchesCard(card) {
		t.Error("expected card to match all filters")
	}
}

func TestMatchesCardAndLogicFail(t *testing.T) {
	card := trello.Card{
		Name:    "Fix login bug",
		Members: []trello.Member{{Username: "craig"}},
	}
	// Card has no labels, so label filter should fail
	f := Filter{Text: "login", Members: []string{"craig"}, Labels: []string{"Bug"}}
	if f.MatchesCard(card) {
		t.Error("expected card not to match — missing label")
	}
}

func TestMatchesCardEmptyFilter(t *testing.T) {
	card := trello.Card{Name: "Anything"}
	f := Filter{}
	if !f.MatchesCard(card) {
		t.Error("empty filter should match all cards")
	}
}
```

- [ ] **Step 6: Run tests to verify they fail**

Run: `go test ./internal/tui/ -run TestMatchesCard -v`
Expected: FAIL — `MatchesCard` undefined

- [ ] **Step 7: Implement MatchesCard**

Add to `internal/tui/filter.go`:

```go
import (
	"strings"

	"github.com/craig006/tuillo/internal/trello"
)

// MatchesCard returns true if the card passes all active filters.
func (f Filter) MatchesCard(card trello.Card) bool {
	if f.IsEmpty() {
		return true
	}

	// Text: case-insensitive substring match on card name
	if f.Text != "" {
		if !strings.Contains(strings.ToLower(card.Name), strings.ToLower(f.Text)) {
			return false
		}
	}

	// Members: card must have at least one matching member (OR)
	if len(f.Members) > 0 {
		found := false
		for _, fm := range f.Members {
			for _, cm := range card.Members {
				if strings.EqualFold(cm.Username, fm) || strings.EqualFold(cm.FullName, fm) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	// Labels: card must have at least one matching label (OR)
	if len(f.Labels) > 0 {
		found := false
		for _, fl := range f.Labels {
			for _, cl := range card.Labels {
				name := cl.Name
				if name == "" {
					name = cl.Color
				}
				if strings.EqualFold(name, fl) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}
```

- [ ] **Step 8: Run all filter tests**

Run: `go test ./internal/tui/ -run "TestParseFilter|TestMatchesCard" -v`
Expected: All PASS

- [ ] **Step 9: Write test for BuildFilterText**

```go
func TestBuildFilterText(t *testing.T) {
	f := Filter{Text: "fix", Members: []string{"craig"}, Labels: []string{"Bug"}}
	result := BuildFilterText(f)
	if !strings.Contains(result, "member:craig") {
		t.Errorf("expected member token in %q", result)
	}
	if !strings.Contains(result, "label:Bug") {
		t.Errorf("expected label token in %q", result)
	}
	if !strings.Contains(result, "fix") {
		t.Errorf("expected text in %q", result)
	}
}

func TestBuildFilterTextQuotesSpaces(t *testing.T) {
	f := Filter{Members: []string{"Craig Smith"}}
	result := BuildFilterText(f)
	if !strings.Contains(result, `member:"Craig Smith"`) {
		t.Errorf("expected quoted member in %q", result)
	}
}
```

- [ ] **Step 10: Implement BuildFilterText**

Add to `internal/tui/filter.go`:

```go
// BuildFilterText reconstructs the search bar text from a Filter.
func BuildFilterText(f Filter) string {
	var parts []string
	if f.Text != "" {
		parts = append(parts, f.Text)
	}
	for _, m := range f.Members {
		if strings.Contains(m, " ") {
			parts = append(parts, `member:"`+m+`"`)
		} else {
			parts = append(parts, "member:"+m)
		}
	}
	for _, l := range f.Labels {
		if strings.Contains(l, " ") {
			parts = append(parts, `label:"`+l+`"`)
		} else {
			parts = append(parts, "label:"+l)
		}
	}
	return strings.Join(parts, " ")
}
```

- [ ] **Step 11: Run all filter tests**

Run: `go test ./internal/tui/ -run "TestParseFilter|TestMatchesCard|TestBuildFilter" -v`
Expected: All PASS

- [ ] **Step 12: Commit**

```bash
git add internal/tui/filter.go internal/tui/filter_test.go
git commit -m "feat: add filter parser and card matching logic"
```

---

### Task 2: MultiSelect Model

Create a reusable multiselect checkbox list component for the member/label modals.

**Files:**
- Create: `internal/tui/multiselect.go`
- Create: `internal/tui/multiselect_test.go`

- [ ] **Step 1: Write tests for MultiSelectModel**

```go
// internal/tui/multiselect_test.go
package tui

import "testing"

func TestMultiSelectToggle(t *testing.T) {
	items := []MultiSelectItem{
		{Label: "Alice", Value: "alice"},
		{Label: "Bob", Value: "bob"},
	}
	m := NewMultiSelectModel("Members", items)
	// Toggle first item
	m.Toggle()
	selected := m.Selected()
	if len(selected) != 1 || selected[0] != "alice" {
		t.Errorf("expected [alice], got %v", selected)
	}
	// Toggle again to deselect
	m.Toggle()
	selected = m.Selected()
	if len(selected) != 0 {
		t.Errorf("expected empty, got %v", selected)
	}
}

func TestMultiSelectNavigation(t *testing.T) {
	items := []MultiSelectItem{
		{Label: "Alice", Value: "alice"},
		{Label: "Bob", Value: "bob"},
		{Label: "Charlie", Value: "charlie"},
	}
	m := NewMultiSelectModel("Members", items)
	m.MoveDown()
	m.Toggle()
	selected := m.Selected()
	if len(selected) != 1 || selected[0] != "bob" {
		t.Errorf("expected [bob], got %v", selected)
	}
}

func TestMultiSelectPreselected(t *testing.T) {
	items := []MultiSelectItem{
		{Label: "Alice", Value: "alice"},
		{Label: "Bob", Value: "bob", Checked: true},
	}
	m := NewMultiSelectModel("Members", items)
	selected := m.Selected()
	if len(selected) != 1 || selected[0] != "bob" {
		t.Errorf("expected [bob], got %v", selected)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -run TestMultiSelect -v`
Expected: FAIL — types undefined

- [ ] **Step 3: Implement MultiSelectModel**

```go
// internal/tui/multiselect.go
package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// MultiSelectItem represents a single item in the multiselect list.
type MultiSelectItem struct {
	Label   string
	Value   string
	Checked bool
	Color   lipgloss.TerminalColor // optional, for label indicators
}

// MultiSelectModel is a simple checkbox list for modal overlays.
type MultiSelectModel struct {
	title   string
	items   []MultiSelectItem
	cursor  int
}

// NewMultiSelectModel creates a new multiselect model.
func NewMultiSelectModel(title string, items []MultiSelectItem) MultiSelectModel {
	return MultiSelectModel{
		title: title,
		items: items,
	}
}

// Toggle toggles the checked state of the item at the cursor.
func (m *MultiSelectModel) Toggle() {
	if len(m.items) > 0 {
		m.items[m.cursor].Checked = !m.items[m.cursor].Checked
	}
}

// MoveDown moves the cursor down.
func (m *MultiSelectModel) MoveDown() {
	if m.cursor < len(m.items)-1 {
		m.cursor++
	}
}

// MoveUp moves the cursor up.
func (m *MultiSelectModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

// Selected returns the values of all checked items.
func (m MultiSelectModel) Selected() []string {
	var selected []string
	for _, item := range m.items {
		if item.Checked {
			selected = append(selected, item.Value)
		}
	}
	return selected
}

// View renders the multiselect list.
func (m MultiSelectModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(15))
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(7))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(12))

	var lines []string
	lines = append(lines, titleStyle.Render(m.title))
	lines = append(lines, "")

	for i, item := range m.items {
		checkbox := "[ ] "
		if item.Checked {
			checkbox = "[x] "
		}

		label := item.Label
		if item.Color != nil {
			indicator := lipgloss.NewStyle().Foreground(item.Color).Render("⏺ ")
			label = indicator + label
		}

		var line string
		if i == m.cursor {
			line = cursorStyle.Render("> "+checkbox) + normalStyle.Render(label)
		} else {
			line = dimStyle.Render("  "+checkbox) + normalStyle.Render(label)
		}
		lines = append(lines, line)
	}

	lines = append(lines, "")
	lines = append(lines, dimStyle.Render(fmt.Sprintf("  space: toggle • enter/esc: close • %d selected", len(m.Selected()))))

	return strings.Join(lines, "\n")
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run TestMultiSelect -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/multiselect.go internal/tui/multiselect_test.go
git commit -m "feat: add multiselect checkbox list component"
```

---

### Task 3: Keybindings and Config

Add filter keybindings to the config and KeyMap.

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/tui/keys.go`

- [ ] **Step 1: Add FilterKeys config struct**

In `internal/config/config.go`, add a new `FilterKeys` struct and add it to `KeybindingConfig`:

```go
type FilterKeys struct {
	Focus   string `mapstructure:"focus"`
	Members string `mapstructure:"members"`
	Labels  string `mapstructure:"labels"`
}
```

Add to `KeybindingConfig`:

```go
type KeybindingConfig struct {
	Universal UniversalKeys `mapstructure:"universal"`
	Board     BoardKeys     `mapstructure:"board"`
	Detail    DetailKeys    `mapstructure:"detail"`
	Filter    FilterKeys    `mapstructure:"filter"`
}
```

Add defaults in the defaults function:

```go
Filter: FilterKeys{
	Focus:   "/",
	Members: "ctrl+m",
	Labels:  "ctrl+l",
},
```

- [ ] **Step 2: Add filter bindings to KeyMap**

In `internal/tui/keys.go`, add to the `KeyMap` struct:

```go
FilterFocus, FilterMembers, FilterLabels key.Binding
```

In `NewKeyMap`, add:

```go
FilterFocus:   key.NewBinding(key.WithKeys(cfg.Filter.Focus), key.WithHelp(cfg.Filter.Focus, "search")),
FilterMembers: key.NewBinding(key.WithKeys(cfg.Filter.Members), key.WithHelp(cfg.Filter.Members, "filter members")),
FilterLabels:  key.NewBinding(key.WithKeys(cfg.Filter.Labels), key.WithHelp(cfg.Filter.Labels, "filter labels")),
```

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add internal/config/config.go internal/tui/keys.go
git commit -m "feat: add filter keybindings config"
```

---

### Task 4: Board Filtering Integration

Add filter state to BoardModel and make columns filter their visible cards.

**Files:**
- Modify: `internal/tui/board.go`

- [ ] **Step 1: Add filter field to BoardModel**

Add a `filter Filter` field to the `BoardModel` struct.

- [ ] **Step 2: Add ApplyFilter method**

```go
// ApplyFilter updates the filter and rebuilds all column item lists.
func (b *BoardModel) ApplyFilter(f Filter) {
	b.filter = f
	for i := range b.columns {
		b.rebuildFilteredItems(i)
	}
}

// rebuildFilteredItems rebuilds a column's list items, applying the current filter.
func (b *BoardModel) rebuildFilteredItems(colIdx int) {
	col := &b.columns[colIdx]
	var items []list.Item
	for _, c := range col.cards {
		if b.filter.MatchesCard(c) {
			items = append(items, cardItem{card: c})
		}
	}
	if items == nil {
		items = []list.Item{}
	}
	col.list.SetItems(items)
	// Clamp selection to valid range
	if col.list.Index() >= len(items) && len(items) > 0 {
		col.list.Select(len(items) - 1)
	}
}
```

- [ ] **Step 3: Update rebuildColumnItems to respect filter**

Modify the existing `rebuildColumnItemsAt` to filter cards and find the correct select index within the filtered list:

```go
func (b *BoardModel) rebuildColumnItemsAt(colIdx int, selectIdx int) {
	col := &b.columns[colIdx]
	var items []list.Item
	filteredIdx := -1
	fi := 0
	for i, c := range col.cards {
		if b.filter.MatchesCard(c) {
			items = append(items, cardItem{card: c})
			if i == selectIdx {
				filteredIdx = fi
			}
			fi++
		}
	}
	if items == nil {
		items = []list.Item{}
	}
	col.list.SetItems(items)
	if filteredIdx >= 0 {
		col.list.Select(filteredIdx)
	} else if len(items) > 0 && col.list.Index() >= len(items) {
		col.list.Select(len(items) - 1)
	}
}
```

This maps the `selectIdx` (index into the full `col.cards` slice) to the corresponding index in the filtered list. If the selected card doesn't match the filter, selection clamps to the last visible card.

- [ ] **Step 4: Update SelectedCard to work with filtered list**

The existing `SelectedCard` reads from `col.cards` but should read from the list's selected item (which is already filtered):

Verify `SelectedCard` in `board.go` — it currently calls `b.columns[b.focused].SelectedCard()` which calls `c.list.SelectedItem().(cardItem)`. This already reads from the list items (filtered), so no change needed here.

- [ ] **Step 5: Add ClearFilter method**

```go
// ClearFilter removes all filters and rebuilds column items.
func (b *BoardModel) ClearFilter() {
	b.filter = Filter{}
	for i := range b.columns {
		b.rebuildFilteredItems(i)
	}
}

// HasFilter returns true if any filter is active.
func (b *BoardModel) HasFilter() bool {
	return !b.filter.IsEmpty()
}
```

- [ ] **Step 6: Verify build and existing tests pass**

Run: `go build ./... && go test ./internal/tui/ -v`
Expected: Build passes, all existing tests pass

- [ ] **Step 7: Commit**

```bash
git add internal/tui/board.go
git commit -m "feat: add filter state and card filtering to board model"
```

---

### Task 5: Search Bar UI and App Integration

Add the search bar text input to the app, wire up keybindings, and render the search bar above the breadcrumb.

**Files:**
- Modify: `internal/tui/app.go`
- Modify: `internal/tui/board.go`

- [ ] **Step 1: Add search bar fields to App struct**

Add these fields to the `App` struct in `app.go`:

```go
// Filter search bar
searchInput    textinput.Model
searchFocused  bool
```

- [ ] **Step 2: Initialize searchInput in NewApp or Init**

In the App constructor or initialization, create the textinput:

```go
si := textinput.New()
si.Placeholder = " Search..."
si.Prompt = " "
si.Width = a.width - 6 // borders (2) + prompt icon (2) + padding (2)
a.searchInput = si
```

Note: `` is the nerdfont search icon `U+F002`. The `Prompt` field places the icon before the cursor.

- [ ] **Step 3: Add search bar keybinding handling in Update**

In `app.go` `Update()`, inside the existing `case tea.KeyPressMsg:` block, add handling AFTER the modal/palette/prompt checks but BEFORE the existing keybinding switch. When `searchFocused` is true, intercept all keys:

```go
// Handle search bar input
if a.searchFocused {
	switch msg.String() {
	case "enter":
		a.searchFocused = false
		a.searchInput.Blur()
		return a, nil
	case "esc":
		a.searchFocused = false
		a.searchInput.Blur()
		a.searchInput.SetValue("")
		a.board.ClearFilter()
		a.syncDetailAfterFilter()
		return a, nil
	default:
		var cmd tea.Cmd
		a.searchInput, cmd = a.searchInput.Update(msg)
		// Live filtering on every keystroke
		f := ParseFilter(a.searchInput.Value())
		a.board.ApplyFilter(f)
		a.syncDetailAfterFilter()
		return a, cmd
	}
}
```

In the existing keybinding switch, add cases for the filter keys:

```go
case matchKey(msg, a.keyMap.FilterFocus):
	if a.boardReady && !a.showPalette {
		a.searchFocused = true
		a.searchInput.Focus()
		return a, nil
	}

case matchKey(msg, a.keyMap.FilterMembers):
	if a.boardReady && !a.showPalette {
		// Open member modal — implemented in Task 6
		return a, nil
	}

case matchKey(msg, a.keyMap.FilterLabels):
	if a.boardReady && !a.showPalette {
		// Open label modal — implemented in Task 6
		return a, nil
	}
```

Add `Escape` clear-all when not focused. This must go AFTER the existing palette/prompt escape handlers (which return early) so it only fires when no overlay is open:

```go
case msg.String() == "esc":
	if a.boardReady && !a.showPalette && !a.showPrompt && a.board.HasFilter() {
		a.searchInput.SetValue("")
		a.board.ClearFilter()
		a.syncDetailAfterFilter()
		return a, nil
	}
```

- [ ] **Step 4: Render search bar in board View**

In `internal/tui/board.go`, add search bar rendering above the breadcrumb in `View()`. Add a `searchBar string` field to `BoardModel` that gets set by `App` before rendering:

```go
// SetSearchBar sets the rendered search bar content to display above the breadcrumb.
func (b *BoardModel) SetSearchBar(rendered string) {
	b.searchBar = rendered
}
```

Add `searchBar string` field to `BoardModel` struct.

In `View()`, prepend the search bar above the breadcrumb:

```go
// After building breadcrumb, before building columns:
var header string
if b.searchBar != "" {
	header = b.searchBar + "\n" + breadcrumb
} else {
	header = breadcrumb
}
// ... use header instead of breadcrumb at the end
return header + "\n" + columns
```

Update column height calculation: when search bar is present, subtract 3 more lines.

In `app.go` `View()`, before rendering the board, set the search bar:

```go
if a.boardReady {
	// Render search bar
	searchContent := a.searchInput.View()
	searchBar := lipgloss.NewStyle().
		Width(a.board.width).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.ANSIColor(8)).
		Render(searchContent)
	a.board.SetSearchBar(searchBar)
	// ... existing board render
}
```

- [ ] **Step 5: Adjust height calculations**

In `board.go`, update the column height to account for the search bar:

```go
// In View(), column height calculation:
headerLines := 3 // breadcrumb
if b.searchBar != "" {
	headerLines = 6 // search bar (3) + breadcrumb (3)
}
colH := b.height - headerLines
```

Update `ResizeColumns()` to match. The current code uses `colHeight := b.height - 5` (3 for breadcrumb border, 2 for column border). With the search bar always present, this becomes `colHeight := b.height - 8` (3 for search bar border + 3 for breadcrumb border + 2 for column border). Update both `ResizeColumns()` and `NewBoardModel()` constructor where `colHeight` is calculated.

- [ ] **Step 6: Verify build**

Run: `go build ./...`
Expected: No errors

- [ ] **Step 7: Run all tests**

Run: `go test ./internal/tui/ -v`
Expected: All tests pass. Some existing tests may need `searchBar` field adjustments if they render the board.

- [ ] **Step 8: Commit**

```bash
git add internal/tui/app.go internal/tui/board.go
git commit -m "feat: add search bar UI with live filtering"
```

---

### Task 6: Member and Label Modals

Wire up the multiselect modals for member and label filtering.

**Files:**
- Modify: `internal/tui/app.go`

- [ ] **Step 1: Add modal fields to App struct**

```go
// Filter modals
showMemberModal bool
showLabelModal  bool
memberModal     MultiSelectModel
labelModal      MultiSelectModel
```

- [ ] **Step 2: Implement modal open for members**

First, add `"strings"` to the import block in `app.go` (needed for `strings.EqualFold`).

In the `FilterMembers` keybinding case (Task 5 placeholder), build member items and open modal:

```go
case matchKey(msg, a.keyMap.FilterMembers):
	if a.boardReady && !a.showPalette && !a.searchFocused {
		currentFilter := ParseFilter(a.searchInput.Value())
		var items []MultiSelectItem
		for _, m := range a.board.board.Members {
			checked := false
			for _, fm := range currentFilter.Members {
				if strings.EqualFold(fm, m.Username) || strings.EqualFold(fm, m.FullName) {
					checked = true
					break
				}
			}
			items = append(items, MultiSelectItem{
				Label:   m.FullName,
				Value:   m.Username,
				Checked: checked,
			})
		}
		a.memberModal = NewMultiSelectModel("Filter by Member", items)
		a.showMemberModal = true
		return a, nil
	}
```

- [ ] **Step 3: Implement modal open for labels**

Similar for `FilterLabels`:

```go
case matchKey(msg, a.keyMap.FilterLabels):
	if a.boardReady && !a.showPalette && !a.searchFocused {
		currentFilter := ParseFilter(a.searchInput.Value())
		// Collect unique labels from all cards across all columns
		seen := make(map[string]bool)
		var items []MultiSelectItem
		for _, col := range a.board.columns {
			for _, card := range col.cards {
				for _, lbl := range card.Labels {
					name := lbl.Name
					if name == "" {
						name = lbl.Color
					}
					if seen[name] {
						continue
					}
					seen[name] = true
					checked := false
					for _, fl := range currentFilter.Labels {
						if strings.EqualFold(fl, name) {
							checked = true
							break
						}
					}
					ansiColor, ok := trelloColorToANSI[lbl.Color]
					if !ok {
						ansiColor = lipgloss.ANSIColor(7)
					}
					items = append(items, MultiSelectItem{
						Label:   name,
						Value:   name,
						Checked: checked,
						Color:   ansiColor,
					})
				}
			}
		}
		a.labelModal = NewMultiSelectModel("Filter by Label", items)
		a.showLabelModal = true
		return a, nil
	}
```

- [ ] **Step 4: Handle modal input in Update**

Add modal input handling BEFORE the search bar handling and keybinding switch:

```go
// Handle member/label modals
if a.showMemberModal || a.showLabelModal {
	modal := &a.memberModal
	isLabel := false
	if a.showLabelModal {
		modal = &a.labelModal
		isLabel = true
	}

	switch msg.String() {
	case "j", "down":
		modal.MoveDown()
	case "k", "up":
		modal.MoveUp()
	case " ":
		modal.Toggle()
	case "enter", "esc":
		// Close modal and update search bar with selections
		selected := modal.Selected()
		currentFilter := ParseFilter(a.searchInput.Value())
		if isLabel {
			currentFilter.Labels = selected
		} else {
			currentFilter.Members = selected
		}
		a.searchInput.SetValue(BuildFilterText(currentFilter))
		a.board.ApplyFilter(currentFilter)
		a.showMemberModal = false
		a.showLabelModal = false
		// Update detail panel
		if a.detail.open {
			if card, _, ok := a.board.SelectedCard(); ok && card.ID != a.detail.cardID {
				a.detail.SetCard(card)
			}
		}
	}
	return a, nil
}
```

- [ ] **Step 5: Render modal overlay in View**

In `app.go` `View()`, add modal rendering (similar to command palette pattern):

```go
if a.showMemberModal {
	modalView := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("cyan")).
		Padding(1).
		Width(a.width / 3).
		Render(a.memberModal.View())
	content = lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, modalView)
} else if a.showLabelModal {
	modalView := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("cyan")).
		Padding(1).
		Width(a.width / 3).
		Render(a.labelModal.View())
	content = lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, modalView)
}
```

This should go after the board rendering but before the final view assembly, overriding `content` when a modal is open.

- [ ] **Step 6: Verify build and test**

Run: `go build ./... && go test ./internal/tui/ -v`
Expected: All pass

- [ ] **Step 7: Commit**

```bash
git add internal/tui/app.go
git commit -m "feat: add member and label multiselect filter modals"
```

---

### Task 7: Edge Cases and Polish

Handle empty columns, filter persistence across refresh, and detail panel sync.

**Files:**
- Modify: `internal/tui/board.go`
- Modify: `internal/tui/app.go`
- Modify: `internal/tui/column.go`

- [ ] **Step 1: Empty column display**

In `column.go`, update `View()` or the list delegate to show "No matching cards" when the list has zero items and a filter is active. The simplest approach: in `board.go` `View()`, if a column has zero filtered items, render a placeholder:

In `board.go` `View()`, after rendering each column, check if the column is empty due to filtering and override the content:

```go
// After rendering col.View() into the style:
if len(col.list.Items()) == 0 && !b.filter.IsEmpty() {
	emptyMsg := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Render("No matching cards")
	rendered = style.Render(emptyMsg)
	// Re-apply custom top border as before
}
```

This ensures filtered-empty columns show "No matching cards" in dimmed text as the spec requires.

- [ ] **Step 2: Filter persistence across board refresh**

In `app.go`, when `BoardFetchedMsg` is handled, re-apply the current filter after rebuilding the board:

```go
case BoardFetchedMsg:
	a.loading = false
	a.boardReady = true
	a.board = NewBoardModel(msg.Board, a.config, a.width, a.height)
	// Re-apply current filter if any
	if a.searchInput.Value() != "" {
		f := ParseFilter(a.searchInput.Value())
		a.board.ApplyFilter(f)
	}
	// ... rest of handler
```

- [ ] **Step 3: Detail panel sync after filtering**

Verify that all filter operations (text change, modal close, Escape clear) update the detail panel when open. This was already added in Task 5 and Task 6 key handlers. Add a helper to reduce duplication:

```go
func (a *App) syncDetailAfterFilter() {
	if a.detail.open {
		if card, _, ok := a.board.SelectedCard(); ok {
			if card.ID != a.detail.cardID {
				a.detail.SetCard(card)
				if a.detail.NeedsFetch() {
					a.detail.MarkLoading()
				}
			}
		} else {
			a.detail.SetCard(trello.Card{})
		}
	}
}
```

Replace the inline detail sync blocks in Task 5 and Task 6 with calls to `a.syncDetailAfterFilter()`.

- [ ] **Step 4: Verify build and all tests**

Run: `go build ./... && go test ./... -v`
Expected: All pass

- [ ] **Step 5: Commit**

```bash
git add internal/tui/app.go internal/tui/board.go internal/tui/column.go
git commit -m "feat: handle filter edge cases — empty columns, refresh, detail sync"
```
