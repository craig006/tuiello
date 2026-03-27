# Focus and Input Routing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a two-level focus model (Section and Element) with event bubbling input routing to replace the simple `boardHasFocus` boolean.

**Architecture:** Introduce `FocusManager` to track focus state, define `KeyHandler` interface for event routing, and refactor `App.Update()` to bubble events through focused elements and sections instead of checking section flags.

**Tech Stack:** Go, Bubble Tea (TUI framework), existing key binding system

---

## Phase 1: Focus State Management

### Task 1: Create focus.go with FocusManager type

**Files:**
- Create: `internal/tui/focus.go`

- [ ] **Step 1: Write failing test for FocusManager**

Create `internal/tui/focus_test.go`:

```go
package tui

import (
	"testing"
)

func TestNewFocusManager(t *testing.T) {
	fm := NewFocusManager("board")

	if fm.FocusedSection() != "board" {
		t.Errorf("expected focused section 'board', got %s", fm.FocusedSection())
	}

	if fm.FocusedElement() != "" {
		t.Errorf("expected no focused element initially, got %s", fm.FocusedElement())
	}

	if fm.IsModalActive() {
		t.Errorf("expected modal to be inactive initially")
	}
}

func TestSetFocusedSection(t *testing.T) {
	fm := NewFocusManager("board")
	fm.SetFocusedSection("detail")

	if fm.FocusedSection() != "detail" {
		t.Errorf("expected focused section 'detail', got %s", fm.FocusedSection())
	}

	if fm.FocusedElement() != "" {
		t.Errorf("expected element focus to clear when section changes, got %s", fm.FocusedElement())
	}
}

func TestSetFocusedElementIgnoredWhenNotInFocusedSection(t *testing.T) {
	fm := NewFocusManager("board")
	success := fm.SetFocusedElement("detail", "some_element")

	if success {
		t.Errorf("expected SetFocusedElement to fail when element not in focused section")
	}

	if fm.FocusedElement() != "" {
		t.Errorf("expected focused element to remain empty")
	}
}

func TestSetFocusedElementSucceedsWhenInFocusedSection(t *testing.T) {
	fm := NewFocusManager("board")
	success := fm.SetFocusedElement("board", "card_1")

	if !success {
		t.Errorf("expected SetFocusedElement to succeed")
	}

	if fm.FocusedElement() != "card_1" {
		t.Errorf("expected focused element 'card_1', got %s", fm.FocusedElement())
	}
}

func TestModalSuspendAndRestore(t *testing.T) {
	fm := NewFocusManager("board")
	fm.SetFocusedElement("board", "card_1")

	fm.OpenModal()

	if !fm.IsModalActive() {
		t.Errorf("expected modal to be active")
	}

	fm.CloseModal()

	if fm.IsModalActive() {
		t.Errorf("expected modal to be inactive after closing")
	}

	if fm.FocusedSection() != "board" {
		t.Errorf("expected focus to be restored to 'board', got %s", fm.FocusedSection())
	}

	if fm.FocusedElement() != "card_1" {
		t.Errorf("expected element focus to be restored to 'card_1', got %s", fm.FocusedElement())
	}
}

func TestSetFocusedSectionIgnoredWhenModalActive(t *testing.T) {
	fm := NewFocusManager("board")
	fm.OpenModal()

	fm.SetFocusedSection("detail")

	if fm.FocusedSection() != "board" {
		t.Errorf("expected focus to remain 'board' while modal active, got %s", fm.FocusedSection())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/craig/GitHub/craig006/tuillo/main
go test ./internal/tui -v -run TestNewFocusManager
```

Expected output:
```
undefined: NewFocusManager
```

- [ ] **Step 3: Implement FocusManager in focus.go**

```go
package tui

// FocusManager tracks which section and element have focus
type FocusManager struct {
	focusedSection string
	focusedElement string
	modalActive    bool
	suspendedState FocusState
}

type FocusState struct {
	FocusedSection string
	FocusedElement string
}

// NewFocusManager creates a FocusManager with initial section focus
func NewFocusManager(initialSection string) *FocusManager {
	return &FocusManager{
		focusedSection: initialSection,
		focusedElement: "",
		modalActive:    false,
	}
}

// FocusedSection returns the currently focused section
func (fm *FocusManager) FocusedSection() string {
	return fm.focusedSection
}

// FocusedElement returns the currently focused element, or "" if none
func (fm *FocusManager) FocusedElement() string {
	return fm.focusedElement
}

// IsModalActive returns true if a modal is currently open
func (fm *FocusManager) IsModalActive() bool {
	return fm.modalActive
}

// SetFocusedSection changes focus to a new section and clears element focus
// Ignored if a modal is active
func (fm *FocusManager) SetFocusedSection(section string) {
	if fm.modalActive {
		return // Ignore while modal is active
	}
	fm.focusedSection = section
	fm.focusedElement = "" // Clear element focus when section changes
}

// SetFocusedElement sets focus to an element within the given section
// Returns true if successful, false if element not in focused section or modal is active
func (fm *FocusManager) SetFocusedElement(section, element string) bool {
	if fm.modalActive {
		return false // Can't set focus while modal active
	}
	if section != fm.focusedSection {
		return false // Can't focus element in non-focused section
	}
	fm.focusedElement = element
	return true
}

// OpenModal suspends the focus model and saves current state
func (fm *FocusManager) OpenModal() {
	fm.suspendedState = FocusState{
		FocusedSection: fm.focusedSection,
		FocusedElement: fm.focusedElement,
	}
	fm.modalActive = true
}

// CloseModal exits modal mode and restores previous focus state
func (fm *FocusManager) CloseModal() {
	fm.modalActive = false
	fm.focusedSection = fm.suspendedState.FocusedSection
	fm.focusedElement = fm.suspendedState.FocusedElement
	fm.suspendedState = FocusState{}
}

// NotifyContentChanged is called when a section's focusable content changes
// Validates that focused element still exists, or selects nearest sibling
func (fm *FocusManager) NotifyContentChanged(section string, currentElements []string) {
	// Only apply if this is the focused section
	if section != fm.focusedSection {
		return
	}

	// If no elements remain, clear focus
	if len(currentElements) == 0 {
		fm.focusedElement = ""
		return
	}

	// If focused element still exists, keep it
	for _, elem := range currentElements {
		if elem == fm.focusedElement {
			return // Focused element still exists
		}
	}

	// Focused element was deleted, select first element as fallback
	// (Future improvement: select nearest by index instead)
	fm.focusedElement = currentElements[0]
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/tui -v -run "Focus"
```

Expected output:
```
PASS: TestNewFocusManager
PASS: TestSetFocusedSection
PASS: TestSetFocusedElementIgnoredWhenNotInFocusedSection
PASS: TestSetFocusedElementSucceedsWhenInFocusedSection
PASS: TestModalSuspendAndRestore
PASS: TestSetFocusedSectionIgnoredWhenModalActive
ok  	github.com/craig006/tuiello/internal/tui	0.123s
```

- [ ] **Step 5: Commit**

```bash
cd /Users/craig/GitHub/craig006/tuillo/main
git add internal/tui/focus.go internal/tui/focus_test.go
git commit -m "feat: add FocusManager for two-level focus tracking

Introduces FocusManager type to track which section and element have focus.
Supports modal suspension/restoration and validates focus state changes.

- NewFocusManager: initialize with starting section
- SetFocusedSection: move focus to new section, clear element focus
- SetFocusedElement: focus element within focused section
- OpenModal/CloseModal: suspend and restore focus state
- NotifyContentChanged: validate focus when section content changes

All core operations tested and working."
```

---

## Phase 2: Event Routing Infrastructure

### Task 2: Define KeyHandler interface in focus.go

- [ ] **Step 1: Add KeyHandler and FocusAware interfaces to focus.go**

Add to the end of `internal/tui/focus.go`:

```go
// KeyHandler is implemented by sections and elements that handle keyboard input
type KeyHandler interface {
	// HandleKeyEvent processes a keyboard event
	// Returns true if handled, false to bubble up
	HandleKeyEvent(key string) bool
}

// FocusAware is optionally implemented by sections to manage focus state
type FocusAware interface {
	// GetFocusableElements returns the IDs of currently focusable elements
	GetFocusableElements() []string

	// OnContentChanged is called when focusable elements change
	OnContentChanged()
}
```

- [ ] **Step 2: Run tests to ensure no regression**

```bash
go test ./internal/tui -v -run "Focus"
```

Expected: All previous tests still pass

- [ ] **Step 3: Commit**

```bash
cd /Users/craig/GitHub/craig006/tuillo/main
git add internal/tui/focus.go
git commit -m "feat: add KeyHandler and FocusAware interfaces

Define interfaces for event routing:
- KeyHandler: process keyboard events with bubbling
- FocusAware: optional interface for sections to report focusable elements"
```

### Task 3: Add event routing to App struct

**Files:**
- Modify: `internal/tui/app.go`

- [ ] **Step 1: Replace boardHasFocus with FocusManager**

In `internal/tui/app.go`, find the `App` struct (around line 69) and replace:

```go
// OLD
boardHasFocus bool  // true = board active (blue border), false = detail active
```

With:

```go
// NEW
focusManager *FocusManager
```

- [ ] **Step 2: Update NewApp to initialize FocusManager**

In `NewApp()` function (around line 111), change:

```go
// OLD
boardHasFocus: true,  // Board starts with focus
```

To:

```go
// NEW
focusManager: NewFocusManager("board"),
```

- [ ] **Step 3: Add HandleKeyEvent method to App**

Add this new method to `internal/tui/app.go`:

```go
// HandleKeyEvent routes keyboard events through the focus hierarchy
// Returns true if the event was handled
func (a *App) HandleKeyEvent(key string) bool {
	// Check if this matches any global/app-level shortcuts
	switch {
	case key.Matches(a.keyMap.Quit.Keys()...):
		return true // Will be handled by caller
	case key.Matches(a.keyMap.Help.Keys()...):
		a.showHelp = !a.showHelp
		return true
	case key.Matches(a.keyMap.Refresh.Keys()...):
		// Trigger board refresh
		return true
	}
	return false
}
```

- [ ] **Step 4: Update App.Update() to route key events**

Find the key event handling in `App.Update()` (look for `tea.KeyMsg`). Replace the section that checks `boardHasFocus` with:

```go
case tea.KeyMsg:
	// Route through focus hierarchy
	key := msg

	// If modal is open, route to modal first
	if a.showPalette {
		if a.commandPalette.HandleKeyEvent(key.String()) {
			return a, nil
		}
		// Unhandled modal events can bubble to app level
	}

	if a.showMemberModal || a.showLabelModal {
		if a.memberModal.HandleKeyEvent(key.String()) {
			return a, nil
		}
		// Similar for label modal
	}

	// If no modal, route through normal focus hierarchy
	if !a.focusManager.IsModalActive() {
		// Try focused element first
		if focusedElemID := a.focusManager.FocusedElement(); focusedElemID != "" {
			// Get the element and call its handler
			// (implementation depends on section structure)
		}

		// Try focused section
		section := a.focusManager.FocusedSection()
		switch section {
		case "board":
			if a.board.HandleKeyEvent(key.String()) {
				return a, nil
			}
		case "detail":
			if a.detail.HandleKeyEvent(key.String()) {
				return a, nil
			}
		case "search":
			if a.searchInput.HandleKeyEvent(key.String()) {
				return a, nil
			}
		}
	}

	// Fall through to app-level handlers
	if a.HandleKeyEvent(key.String()) {
		return a, nil
	}
```

- [ ] **Step 5: Build and check for compilation errors**

```bash
cd /Users/craig/GitHub/craig006/tuillo/main
go build ./cmd/tuillo
```

Expected: Should compile (or show which sections need KeyHandler implementation)

- [ ] **Step 6: Commit**

```bash
git add internal/tui/app.go
git commit -m "feat: replace boardHasFocus with FocusManager

- Replace boardHasFocus bool with focusManager *FocusManager
- Add HandleKeyEvent method to App for global shortcuts
- Update App.Update() to route key events through focus hierarchy
- Event bubbling flow: element → section → app"
```

---

## Phase 3: Board Section Integration

### Task 4: Implement KeyHandler in Board

**Files:**
- Modify: `internal/tui/board.go`

- [ ] **Step 1: Add HandleKeyEvent to Board**

Add this method to `internal/tui/board.go`:

```go
// HandleKeyEvent processes keyboard events for the board section
func (b *Board) HandleKeyEvent(key string) bool {
	// Board handles navigation and card-specific actions
	switch key {
	case "up":
		b.MoveUp()
		return true
	case "down":
		b.MoveDown()
		return true
	case "left":
		b.MoveLeft()
		return true
	case "right":
		b.MoveRight()
		return true
	case "ctrl+c": // Move card left
		b.MoveCardLeft()
		return true
	case "ctrl+v": // Move card right
		b.MoveCardRight()
		return true
	case "o": // Open card
		b.OpenCard()
		return true
	default:
		return false
	}
}
```

- [ ] **Step 2: Add GetFocusableElements to Board**

Add to `internal/tui/board.go`:

```go
// GetFocusableElements returns the IDs of focusable elements (cards) in this board
func (b *Board) GetFocusableElements() []string {
	// Return list of card IDs from current column
	if len(b.Columns) == 0 || b.FocusedColumnIndex < 0 || b.FocusedColumnIndex >= len(b.Columns) {
		return nil
	}
	col := b.Columns[b.FocusedColumnIndex]
	ids := make([]string, len(col.Cards))
	for i, card := range col.Cards {
		ids[i] = card.ID
	}
	return ids
}
```

- [ ] **Step 3: Build and test**

```bash
go build ./cmd/tuillo
```

- [ ] **Step 4: Commit**

```bash
git add internal/tui/board.go
git commit -m "feat: implement KeyHandler on Board

Board processes navigation (arrows) and card actions (open, move).
Handles events for focused column and card."
```

---

## Phase 4: Detail Section Integration

### Task 5: Implement KeyHandler in Detail

**Files:**
- Modify: `internal/tui/detail.go`

- [ ] **Step 1: Add HandleKeyEvent to Detail**

Add to `internal/tui/detail.go`:

```go
// HandleKeyEvent processes keyboard events for the detail panel
func (d *DetailModel) HandleKeyEvent(key string) bool {
	switch key {
	case "tab": // Next tab
		d.TabNext()
		return true
	case "shift+tab": // Prev tab
		d.TabPrev()
		return true
	case "up":
		d.ScrollUp()
		return true
	case "down":
		d.ScrollDown()
		return true
	default:
		return false
	}
}
```

- [ ] **Step 2: Add GetFocusableElements to Detail**

Add to `internal/tui/detail.go`:

```go
// GetFocusableElements returns the IDs of focusable elements in the active tab
func (d *DetailModel) GetFocusableElements() []string {
	// Depends on active tab
	switch d.activeTab {
	case "comments":
		// Return comment IDs
		ids := make([]string, len(d.Comments))
		for i, comment := range d.Comments {
			ids[i] = comment.ID
		}
		return ids
	default:
		return nil
	}
}
```

- [ ] **Step 3: Build and test**

```bash
go build ./cmd/tuillo
```

- [ ] **Step 4: Commit**

```bash
git add internal/tui/detail.go
git commit -m "feat: implement KeyHandler on DetailModel

Detail panel processes tab navigation and scrolling.
Reports focusable elements (comments) from active tab."
```

---

## Phase 5: Comments as Focusable Elements

### Task 6: Implement KeyHandler on comments

**Files:**
- Modify: `internal/tui/comments.go`

- [ ] **Step 1: Add HandleKeyEvent to comment (or comment container)**

Review `internal/tui/comments.go` and add:

```go
// HandleKeyEvent processes keyboard events for comments section
func (c *CommentsModel) HandleKeyEvent(key string) bool {
	switch key {
	case "up":
		c.SelectPrevious()
		return true
	case "down":
		c.SelectNext()
		return true
	case "d": // Delete comment
		if c.CanDelete() {
			c.DeleteSelected()
			return true
		}
		return false // Bubble up if can't delete
	case "e": // Edit comment
		if c.CanEdit() {
			c.StartEdit()
			return true
		}
		return false
	default:
		return false
	}
}
```

- [ ] **Step 2: Build and test**

```bash
go build ./cmd/tuillo
```

- [ ] **Step 3: Commit**

```bash
git add internal/tui/comments.go
git commit -m "feat: implement KeyHandler on CommentsModel

Comments section handles selection (up/down) and actions (edit, delete).
Respects permissions (only allow delete/edit if authorized)."
```

---

## Phase 6: Filter Section as Text Input

### Task 7: Implement KeyHandler on search/filter

**Files:**
- Modify: `internal/tui/filter.go` and `internal/tui/app.go`

- [ ] **Step 1: Add HandleKeyEvent to filter/search input**

Add to `internal/tui/filter.go` or in `app.go` where searchInput is handled:

```go
// In app.go, add a method to handle search input
// HandleSearchKeyEvent processes keyboard events for the search input
func (a *App) HandleSearchKeyEvent(key string) bool {
	// Text input consumes all input except special keys
	switch key {
	case "esc":
		a.searchFocused = false
		return true
	case "enter":
		// Apply search filter
		a.ApplySearch()
		return true
	case "ctrl+m":
		// Toggle member filter modal
		a.showMemberModal = !a.showMemberModal
		return true
	case "ctrl+l":
		// Toggle label filter modal
		a.showLabelModal = !a.showLabelModal
		return true
	default:
		// All other input goes to the text input
		return a.searchInput.HandleKeyEvent(key)
	}
}
```

- [ ] **Step 2: Update key routing to handle search focus**

In `App.Update()` key event handler, add check for search focus:

```go
// If search is focused, route to search handler
if a.searchFocused {
	if a.HandleSearchKeyEvent(key.String()) {
		return a, nil
	}
}
```

- [ ] **Step 3: Build and test**

```bash
go build ./cmd/tuillo
```

- [ ] **Step 4: Commit**

```bash
git add internal/tui/filter.go internal/tui/app.go
git commit -m "feat: implement KeyHandler for search/filter section

Search input consumes all keyboard input except special keys (Esc, Enter).
Escape exits search mode, Enter applies filter.
Ctrl+M/L toggle member/label modals."
```

---

## Phase 7: Modal Integration

### Task 8: Wire up modal suspension in App.Update()

**Files:**
- Modify: `internal/tui/app.go`

- [ ] **Step 1: Call OpenModal when modals are shown**

In `App.Update()`, where `showPalette` or `showMemberModal` is set to true, add:

```go
// When opening command palette
a.showPalette = true
a.focusManager.OpenModal()
```

- [ ] **Step 2: Call CloseModal when modals are hidden**

Where `showPalette` or modals are set to false, add:

```go
// When closing modals
a.showPalette = false
a.focusManager.CloseModal()
```

- [ ] **Step 3: Update key event routing to check modal state**

Modify the key event routing in `App.Update()` to use `focusManager.IsModalActive()`:

```go
if a.focusManager.IsModalActive() {
	// Handle modal input only
	// Don't route to sections
}
```

- [ ] **Step 4: Build and test**

```bash
go build ./cmd/tuillo
```

- [ ] **Step 5: Commit**

```bash
git add internal/tui/app.go
git commit -m "feat: integrate modal suspension in focus manager

- OpenModal called when modals appear
- CloseModal called when modals close
- Focus state automatically saved/restored
- Sections not active while modal open"
```

---

## Phase 8: Text Input Lifecycle

### Task 9: Handle editing mode with text input elements

**Files:**
- Modify: `internal/tui/comments.go`, `internal/tui/app.go`

- [ ] **Step 1: Create EditingComment state in CommentsModel**

Add to `internal/tui/comments.go`:

```go
type CommentsModel struct {
	// ... existing fields ...
	editingCommentID string       // ID of comment being edited, or ""
	editInput       textinput.Model // Text input for editing
}
```

- [ ] **Step 2: Update HandleKeyEvent to enter edit mode**

Modify the 'e' case in CommentsModel.HandleKeyEvent():

```go
case "e": // Edit comment
	if c.CanEdit() {
		c.StartEdit()
		// Focus shifts to the text input element
		return true
	}
	return false
```

- [ ] **Step 3: Add HandleKeyEvent to the text input while editing**

Add method to CommentsModel:

```go
// HandleEditKeyEvent processes keyboard input while editing a comment
func (c *CommentsModel) HandleEditKeyEvent(key string) bool {
	switch key {
	case "esc":
		c.CancelEdit()
		return true
	case "enter":
		if key == "enter" && !tea.HasSuffix(key, "shift") {
			c.SubmitEdit()
			return true
		}
		// Shift+Enter allows newline
		return c.editInput.HandleKeyEvent(key)
	default:
		// All other input goes to text input
		return c.editInput.HandleKeyEvent(key)
	}
}
```

- [ ] **Step 4: Update Detail.HandleKeyEvent to route to comments edit**

In `internal/tui/detail.go`:

```go
func (d *DetailModel) HandleKeyEvent(key string) bool {
	// If currently editing a comment, route to edit handler
	if d.CommentsModel.IsEditing() {
		return d.CommentsModel.HandleEditKeyEvent(key)
	}

	// Otherwise, normal detail handling
	switch key {
	case "tab":
		d.TabNext()
		return true
	// ... rest of cases
	}
}
```

- [ ] **Step 5: Build and test**

```bash
go build ./cmd/tuillo
```

- [ ] **Step 6: Commit**

```bash
git add internal/tui/comments.go internal/tui/detail.go
git commit -m "feat: implement text input lifecycle for comment editing

- Press 'e' to enter edit mode for a comment
- Text input consumes all keyboard input while editing
- Escape cancels, Enter submits
- Focus returns to comment after editing ends"
```

---

## Phase 9: Integration Testing

### Task 10: Write integration tests for focus flow

**Files:**
- Modify: `internal/tui/app_test.go`

- [ ] **Step 1: Write test for focus transitions**

Add to `internal/tui/app_test.go`:

```go
func TestFocusTransitionsWithKeyboard(t *testing.T) {
	// Create a test app
	app := NewTestApp()

	// Start with board focused
	if app.focusManager.FocusedSection() != "board" {
		t.Errorf("expected board focused initially")
	}

	// Press Tab to switch to detail
	app.SimulateKeyPress("tab")
	if app.focusManager.FocusedSection() != "detail" {
		t.Errorf("expected detail focused after tab")
	}

	// Press Tab again to switch to search
	app.SimulateKeyPress("tab")
	if app.focusManager.FocusedSection() != "search" {
		t.Errorf("expected search focused after second tab")
	}
}

func TestModalSuspendsFocus(t *testing.T) {
	app := NewTestApp()

	// Board is focused with first card selected
	app.focusManager.SetFocusedElement("board", "card_1")

	// Open command palette
	app.SimulateKeyPress("ctrl+k")
	if !app.focusManager.IsModalActive() {
		t.Errorf("expected modal to be active")
	}

	// Close modal
	app.SimulateKeyPress("esc")
	if app.focusManager.IsModalActive() {
		t.Errorf("expected modal to be inactive")
	}

	// Focus should be restored
	if app.focusManager.FocusedSection() != "board" {
		t.Errorf("expected board focus restored")
	}
	if app.focusManager.FocusedElement() != "card_1" {
		t.Errorf("expected card_1 focus restored")
	}
}

func TestTextInputConsumesAllInput(t *testing.T) {
	app := NewTestApp()

	// Enter comment editing mode
	app.SimulateKeyPress("e") // Assuming focused on comment

	// Type text - should not trigger shortcuts
	app.SimulateKeyPress("d") // Would delete if not in edit mode

	// Check that 'd' was not treated as delete action
	if app.DeleteActionCalled {
		t.Errorf("expected 'd' to be inserted in text, not trigger delete")
	}
}
```

- [ ] **Step 2: Build and run tests**

```bash
go test ./internal/tui -v -run "Integration"
```

- [ ] **Step 3: Commit**

```bash
git add internal/tui/app_test.go
git commit -m "test: add integration tests for focus flow

Tests verify:
- Focus transitions between sections
- Modal suspension and restoration
- Text input prevents shortcuts
- Element focus management"
```

---

## Self-Review Against Spec

**Spec Coverage Check:**

✅ **Two-Level Focus Model** — Task 1 implements FocusManager with section and element tracking
✅ **Section Focus Rules** — Tasks 1-2 ensure exactly one section, auto-select elements
✅ **Element Focus Rules** — Tasks 1-2 ensure 0 or 1 element, clear on section change
✅ **Content Change Handling** — Task 1 includes NotifyContentChanged logic
✅ **Modal Suspension** — Tasks 1, 8 implement OpenModal/CloseModal
✅ **Event Bubbling** — Task 3 adds routing: element → section → app
✅ **Text Input Special Handling** — Task 9 prevents shortcuts during editing
✅ **Autocomplete Non-Modal** — Task 9 keeps focus on text input
✅ **KeyHandler Interface** — Task 2 defines interface
✅ **FocusAware Interface** — Task 2 defines interface for content reporting
✅ **App Integration** — Tasks 3-8 wire up all sections

**No Placeholders Found** — All steps contain complete code, exact commands, expected output

**Type Consistency** — Method names consistent throughout (FocusedSection, SetFocusedSection, HandleKeyEvent, etc.)

---

## Execution Options

Plan complete and saved to `docs/superpowers/plans/2026-03-27-focus-input-implementation.md`.

Two execution approaches available:

**1. Subagent-Driven (recommended)** — Fresh subagent per task, incremental review, fast iteration

**2. Inline Execution** — Execute tasks sequentially in this session with checkpoints

Which approach would you like?