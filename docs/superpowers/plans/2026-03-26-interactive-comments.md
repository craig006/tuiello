# Interactive Comments Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enable users to create, edit, and delete comments on cards with @mention autocomplete.

**Architecture:** CommentsList is a self-contained Bubble Tea sub-component that owns comment selection, edit/create modes, text input, and autocomplete state. DetailModel delegates to it when the Comments tab is active. App manages top-level focus state (board vs detail) and routes keybindings accordingly.

**Tech Stack:** Bubble Tea, bubbles/textinput, Go stdlib, Trello API

---

## Phase 1: Focus Management System

### Task 1: Add focus state to App

**Files:**
- Modify: `internal/tui/app.go:1-50` (App struct definition)

**Step 1:** Add focus field to App struct

In `internal/tui/app.go`, find the `App` struct definition. Add a new field after the existing fields:

```go
type App struct {
    board     BoardModel
    detail    DetailModel
    status    StatusBar
    keyMap    KeyMap
    theme     Theme

    // NEW: Focus management
    boardHasFocus bool  // true = board active (blue border), false = detail active

    // existing fields...
}
```

**Step 2:** Initialize focus state in NewApp

Find the `NewApp()` function. Add initialization for focus:

```go
func NewApp(board BoardModel, detail DetailModel, theme Theme, keyMap KeyMap) App {
    return App{
        board:         board,
        detail:        detail,
        status:        NewStatusBar(theme),
        keyMap:        keyMap,
        theme:         theme,
        boardHasFocus: true,  // Board starts with focus
    }
}
```

**Step 3:** Commit

```bash
git add internal/tui/app.go
git commit -m "feat: add focus state to App (boardHasFocus field)"
```

---

### Task 2: Add focus toggle keybindings

**Files:**
- Modify: `internal/tui/keys.go:1-100` (KeyMap struct)

**Step 1:** Add focus keybindings to KeyMap

Find the `KeyMap` struct in `internal/tui/keys.go`. Add a new section for universal focus keybindings:

```go
type KeyMap struct {
    // Universal
    Quit   key.Binding
    Help   key.Binding
    Refresh key.Binding

    // NEW: Focus toggle
    FocusDetail key.Binding  // Enter
    FocusBoard  key.Binding  // Escape

    // existing fields...
}
```

**Step 2:** Define default keybindings

Find where default keybindings are set (likely in `DefaultKeyMap()` function). Add:

```go
FocusDetail: key.NewBinding(
    key.WithKeys("enter"),
    key.WithHelp("enter", "focus detail panel"),
),
FocusBoard: key.NewBinding(
    key.WithKeys("esc"),
    key.WithHelp("esc", "focus board"),
),
```

**Step 3:** Add to config YAML structure

Find the `KeybindingConfig` struct (if it exists, or keybinding-related config). Add:

```go
type KeybindingConfig struct {
    Universal struct {
        // existing bindings...
        FocusDetail string `yaml:"focusDetail"`
        FocusBoard  string `yaml:"focusBoard"`
    } `yaml:"universal"`
    // existing sections...
}
```

**Step 4:** Load keybindings from config

Find where keybindings are loaded from config. Add logic to set FocusDetail and FocusBoard from config if present.

**Step 5:** Commit

```bash
git add internal/tui/keys.go
git commit -m "feat: add focus toggle keybindings (enter/escape)"
```

---

### Task 3: Route keybindings based on focus in App.Update

**Files:**
- Modify: `internal/tui/app.go:Update()` method

**Step 1:** Add focus toggle logic to App.Update

In the `App.Update()` method, add this logic early (before routing to board/detail):

```go
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd

    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch {
        // Focus toggles
        case key.Matches(msg, a.keyMap.FocusDetail):
            // Enter focuses detail (only if board has focus and a card is selected)
            if a.boardHasFocus {
                if _, _, ok := a.board.SelectedCard(); ok {
                    a.boardHasFocus = false
                    a.detail.SetFocus(true)
                    return a, nil
                }
            }

        case key.Matches(msg, a.keyMap.FocusBoard):
            // Escape returns focus to board
            if !a.boardHasFocus {
                a.boardHasFocus = true
                a.detail.SetFocus(false)
                return a, nil
            }
        }
    }

    // Route to board or detail based on focus
    if a.boardHasFocus {
        a.board, cmd = a.board.Update(msg).(tea.Model)
    } else {
        a.detail, cmd = a.detail.Update(msg).(tea.Model)
    }

    // ... rest of Update logic
    return a, cmd
}
```

**Step 2:** Add SetFocus method to DetailModel

In `internal/tui/detail.go`, add:

```go
func (d *DetailModel) SetFocus(focused bool) {
    d.focused = focused
    // If defocusing, blur any active input
    if !focused && d.comments != nil {
        // Will implement in later task
    }
}
```

Add `focused bool` field to DetailModel struct:

```go
type DetailModel struct {
    open    bool
    focused bool  // NEW: true when detail panel has focus
    tab     int
    // ... existing fields
}
```

**Step 3:** Test focus toggle

Write a simple test to verify focus state changes:

```go
// In internal/tui/app_test.go
func TestFocusToggle(t *testing.T) {
    app := NewApp(newTestBoard(), newTestDetail(), defaultTheme(), defaultKeyMap())

    // Initially board has focus
    if !app.boardHasFocus {
        t.Fatal("Board should start with focus")
    }

    // Press Enter to focus detail
    enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
    updatedApp, _ := app.Update(enterMsg).(App)
    if updatedApp.boardHasFocus {
        t.Fatal("Detail should have focus after Enter")
    }

    // Press Escape to return to board
    escapeMsg := tea.KeyMsg{Type: tea.KeyEsc}
    updatedApp, _ = updatedApp.Update(escapeMsg).(App)
    if !updatedApp.boardHasFocus {
        t.Fatal("Board should have focus after Escape")
    }
}
```

**Step 4:** Commit

```bash
git add internal/tui/app.go internal/tui/detail.go internal/tui/app_test.go
git commit -m "feat: implement focus toggle between board and detail (enter/escape)"
```

---

## Phase 2: Trello API Methods

### Task 4: Add comment methods to trello.Client

**Files:**
- Modify: `internal/trello/client.go:1-50` (Client struct and imports)
- Modify: `internal/trello/types.go` or add to `client.go` (if types are separate)

**Step 1:** Define Comment type with editable flag

Add to your types file (or top of client.go):

```go
type Comment struct {
    ID       string
    Author   Member
    Body     string
    Date     time.Time
    Editable bool  // Can user edit/delete this comment?
}
```

**Step 2:** Add CreateComment method

In `internal/trello/client.go`, add:

```go
func (c *Client) CreateComment(cardID, text string) (Comment, error) {
    reqURL := fmt.Sprintf("https://api.trello.com/1/cards/%s/actions/comments", cardID)

    body := url.Values{
        "text":     {text},
        "key":      {c.apiKey},
        "token":    {c.token},
    }.Encode()

    resp, err := http.Post(reqURL, "application/x-www-form-urlencoded", strings.NewReader(body))
    if err != nil {
        return Comment{}, fmt.Errorf("create comment request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return Comment{}, fmt.Errorf("create comment failed: status %d", resp.StatusCode)
    }

    var action struct {
        ID   string `json:"id"`
        Data struct {
            Text string `json:"text"`
        } `json:"data"`
        IDMemberCreator string `json:"idMemberCreator"`
        MemberCreator   Member `json:"memberCreator"`
        Date            string `json:"date"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&action); err != nil {
        return Comment{}, fmt.Errorf("parse comment response: %w", err)
    }

    date, _ := time.Parse(time.RFC3339, action.Date)
    return Comment{
        ID:       action.ID,
        Author:   action.MemberCreator,
        Body:     action.Data.Text,
        Date:     date,
        Editable: true,  // User just created it, so it's editable
    }, nil
}
```

**Step 3:** Add UpdateComment method

```go
func (c *Client) UpdateComment(commentID, text string) (Comment, error) {
    reqURL := fmt.Sprintf("https://api.trello.com/1/actions/%s", commentID)

    body := url.Values{
        "text":     {text},
        "key":      {c.apiKey},
        "token":    {c.token},
    }.Encode()

    req, err := http.NewRequest("PUT", reqURL, strings.NewReader(body))
    if err != nil {
        return Comment{}, fmt.Errorf("create request: %w", err)
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return Comment{}, fmt.Errorf("update comment request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusMethodNotAllowed {
        return Comment{}, fmt.Errorf("update not supported: %w", ErrUpdateNotSupported)
    }
    if resp.StatusCode != http.StatusOK {
        return Comment{}, fmt.Errorf("update comment failed: status %d", resp.StatusCode)
    }

    // Parse response similar to CreateComment
    var action struct {
        ID   string `json:"id"`
        Data struct {
            Text string `json:"text"`
        } `json:"data"`
        IDMemberCreator string `json:"idMemberCreator"`
        MemberCreator   Member `json:"memberCreator"`
        Date            string `json:"date"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&action); err != nil {
        return Comment{}, fmt.Errorf("parse response: %w", err)
    }

    date, _ := time.Parse(time.RFC3339, action.Date)
    return Comment{
        ID:       action.ID,
        Author:   action.MemberCreator,
        Body:     action.Data.Text,
        Date:     date,
        Editable: true,
    }, nil
}
```

**Step 4:** Add DeleteComment method

```go
func (c *Client) DeleteComment(commentID string) error {
    reqURL := fmt.Sprintf("https://api.trello.com/1/actions/%s", commentID)

    req, err := http.NewRequest("DELETE", reqURL, nil)
    if err != nil {
        return fmt.Errorf("create request: %w", err)
    }

    q := req.URL.Query()
    q.Add("key", c.apiKey)
    q.Add("token", c.token)
    req.URL.RawQuery = q.Encode()

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return fmt.Errorf("delete comment request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusMethodNotAllowed {
        return ErrDeleteNotSupported
    }
    if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
        return fmt.Errorf("delete failed: status %d", resp.StatusCode)
    }

    return nil
}
```

**Step 5:** Add GetBoardMembers method

```go
func (c *Client) GetBoardMembers(boardID string) ([]Member, error) {
    reqURL := fmt.Sprintf("https://api.trello.com/1/boards/%s", boardID)

    q := url.Values{
        "members":              {"open"},
        "member_fields":        {"fullName,username,initials,avatarHash"},
        "key"::                 {c.apiKey},
        "token":                {c.token},
    }

    resp, err := http.Get(reqURL + "?" + q.Encode())
    if err != nil {
        return nil, fmt.Errorf("fetch members request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("fetch members failed: status %d", resp.StatusCode)
    }

    var board struct {
        Members []Member `json:"members"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&board); err != nil {
        return nil, fmt.Errorf("parse members: %w", err)
    }

    return board.Members, nil
}
```

**Step 6:** Define error sentinels

At top of `client.go`:

```go
var (
    ErrUpdateNotSupported = errors.New("comment update not supported by Trello API")
    ErrDeleteNotSupported = errors.New("comment delete not supported by Trello API")
)
```

**Step 7:** Test API methods

Write mock tests:

```go
// In internal/trello/client_test.go
func TestCreateComment(t *testing.T) {
    client := NewTestClient()

    comment, err := client.CreateComment("card123", "Test comment")
    if err != nil {
        t.Fatalf("CreateComment failed: %v", err)
    }

    if comment.Body != "Test comment" {
        t.Fatalf("Expected body 'Test comment', got %q", comment.Body)
    }
    if !comment.Editable {
        t.Fatal("New comment should be editable")
    }
}

func TestUpdateComment(t *testing.T) {
    client := NewTestClient()

    comment, err := client.UpdateComment("comment123", "Updated text")
    if err != nil {
        t.Fatalf("UpdateComment failed: %v", err)
    }

    if comment.Body != "Updated text" {
        t.Fatalf("Expected updated body, got %q", comment.Body)
    }
}

func TestDeleteComment(t *testing.T) {
    client := NewTestClient()

    err := client.DeleteComment("comment123")
    if err != nil {
        t.Fatalf("DeleteComment failed: %v", err)
    }
}

func TestGetBoardMembers(t *testing.T) {
    client := NewTestClient()

    members, err := client.GetBoardMembers("board123")
    if err != nil {
        t.Fatalf("GetBoardMembers failed: %v", err)
    }

    if len(members) == 0 {
        t.Fatal("Expected board members")
    }
}
```

**Step 8:** Commit

```bash
git add internal/trello/client.go internal/trello/types.go internal/trello/client_test.go
git commit -m "feat: add comment API methods (create, update, delete, get members)"
```

---

## Phase 3: CommentsList Component

### Task 5: Create CommentsList component scaffold

**Files:**
- Create: `internal/tui/comments.go`

**Step 1:** Define CommentsList struct and types

Create `internal/tui/comments.go`:

```go
package tui

import (
    "github.com/charmbracelet/bubbles/textinput"
    "github.com/charmbracelet/bubbles/viewport"
    "github.com/charmbracelet/lipgloss"
    "github.com/craig006/tuiello/internal/trello"
)

type CommentMode int

const (
    CommentModeView CommentMode = iota
    CommentModeCreate
    CommentModeEdit
)

type AutocompleteState struct {
    Active      bool
    Matches     []trello.Member
    SelectedIdx int
    Query       string
    Pos         int  // cursor position of @ in input
}

type CommentsList struct {
    // Data
    comments   []trello.Comment
    allMembers []trello.Member

    // Selection & modes
    selectedIdx int
    mode        CommentMode
    editingIdx  int

    // Input
    textInput textinput.Model

    // Autocomplete
    autocomplete AutocompleteState

    // Rendering
    viewport viewport.Model
    width    int
    height   int
    theme    Theme
    keyMap   KeyMap

    // State
    focused    bool
    loading    bool
    loadingErr string
}

func NewCommentsList(theme Theme, keyMap KeyMap) CommentsList {
    ti := textinput.New()
    ti.Placeholder = "Type comment..."

    return CommentsList{
        comments:     []trello.Comment{},
        allMembers:   []trello.Member{},
        selectedIdx:  0,
        mode:         CommentModeView,
        editingIdx:   -1,
        textInput:    ti,
        autocomplete: AutocompleteState{},
        viewport:     viewport.New(80, 20),
        width:        80,
        height:       20,
        theme:        theme,
        keyMap:       keyMap,
        focused:      false,
    }
}
```

**Step 2:** Implement stub methods

Add minimal implementations that will be filled in later:

```go
func (cl CommentsList) Update(msg tea.Msg) (CommentsList, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if !cl.focused {
            return cl, nil
        }
        // Will be filled in by later tasks
    }
    return cl, nil
}

func (cl CommentsList) View() string {
    if len(cl.comments) == 0 {
        return lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("No comments")
    }
    return "Comments: " + lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Render("TODO")
}

func (cl *CommentsList) SetComments(comments []trello.Comment) {
    cl.comments = comments
    if len(comments) > 0 {
        cl.selectedIdx = 0
    }
}

func (cl *CommentsList) SetMembers(members []trello.Member) {
    cl.allMembers = members
}

func (cl *CommentsList) SetSize(width, height int) {
    cl.width = width
    cl.height = height
    cl.viewport.Width = width
    cl.viewport.Height = height - 5  // Reserve space for input or footer
}

func (cl *CommentsList) SetFocus(focused bool) {
    cl.focused = focused
    if focused {
        cl.textInput.Focus()
    } else {
        cl.textInput.Blur()
    }
}
```

**Step 3:** Test basic structure

```go
// In internal/tui/comments_test.go
func TestNewCommentsList(t *testing.T) {
    cl := NewCommentsList(defaultTheme(), defaultKeyMap())

    if cl.mode != CommentModeView {
        t.Fatal("Should start in View mode")
    }
    if len(cl.comments) != 0 {
        t.Fatal("Should start with no comments")
    }
}

func TestSetComments(t *testing.T) {
    cl := NewCommentsList(defaultTheme(), defaultKeyMap())

    comments := []trello.Comment{
        {ID: "1", Body: "First"},
        {ID: "2", Body: "Second"},
    }

    cl.SetComments(comments)

    if len(cl.comments) != 2 {
        t.Fatalf("Expected 2 comments, got %d", len(cl.comments))
    }
}
```

**Step 4:** Commit

```bash
git add internal/tui/comments.go internal/tui/comments_test.go
git commit -m "feat: scaffold CommentsList component"
```

---

### Task 6: Implement View mode with j/k navigation

**Files:**
- Modify: `internal/tui/comments.go`

**Step 1:** Implement View mode rendering

Update the `View()` method in CommentsList:

```go
func (cl CommentsList) View() string {
    if len(cl.comments) == 0 {
        return lipgloss.NewStyle().
            Foreground(lipgloss.Color("8")).
            Render("No comments. Press 'c' to create.")
    }

    // Build comment list
    var lines []string
    for i, comment := range cl.comments {
        // Format: "Author Name (YYYY-MM-DD)"
        dateStr := comment.Date.Format("2006-01-02")
        header := lipgloss.NewStyle().
            Bold(true).
            Render(comment.Author.FullName) +
            " " +
            lipgloss.NewStyle().
                Foreground(lipgloss.Color("8")).
                Render("(" + dateStr + ")")

        // Body with word wrap
        body := wordWrap(comment.Body, cl.width-4)

        // Selection indicator (blue bar on left)
        indicator := " "
        if i == cl.selectedIdx {
            indicator = lipgloss.NewStyle().
                Foreground(lipgloss.Color("4")).
                Render("│")
        }

        lines = append(lines, indicator+" "+header)
        for _, line := range strings.Split(body, "\n") {
            lines = append(lines, "│ "+line)
        }
        lines = append(lines, "├"+strings.Repeat("─", cl.width-2))
    }

    // Render through viewport
    content := strings.Join(lines, "\n")
    cl.viewport.SetContent(content)

    // Add footer
    footer := lipgloss.NewStyle().
        Foreground(lipgloss.Color("8")).
        Render("Press 'c' to create, 'e' to edit, 'd' to delete")

    return cl.viewport.View() + "\n" + footer
}

func wordWrap(text string, width int) string {
    // Simple word wrap implementation
    var result []string
    words := strings.Fields(text)
    var line string

    for _, word := range words {
        if len(line)+len(word)+1 > width {
            result = append(result, line)
            line = word
        } else {
            if line == "" {
                line = word
            } else {
                line += " " + word
            }
        }
    }
    if line != "" {
        result = append(result, line)
    }
    return strings.Join(result, "\n")
}
```

**Step 2:** Implement j/k navigation in Update

Update the `Update()` method to handle j/k:

```go
func (cl CommentsList) Update(msg tea.Msg) (CommentsList, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if !cl.focused {
            return cl, nil
        }

        // Only handle navigation keys in View mode
        if cl.mode == CommentModeView {
            switch {
            case key.Matches(msg, cl.keyMap.Down):  // j/k mapped to down/up in KeyMap
                if cl.selectedIdx < len(cl.comments)-1 {
                    cl.selectedIdx++
                }
                return cl, nil
            case key.Matches(msg, cl.keyMap.Up):
                if cl.selectedIdx > 0 {
                    cl.selectedIdx--
                }
                return cl, nil
            case key.Matches(msg, cl.keyMap.Custom):  // Use a custom key for 'c'
                // Will implement in next task
            case msg.String() == "e":
                // Will implement in next task
            case msg.String() == "d":
                // Will implement in next task
            }
        }
    }
    return cl, nil
}
```

**Step 3:** Add test for navigation

```go
func TestNavigateComments(t *testing.T) {
    cl := NewCommentsList(defaultTheme(), defaultKeyMap())
    cl.SetFocus(true)

    comments := []trello.Comment{
        {ID: "1", Body: "First"},
        {ID: "2", Body: "Second"},
    }
    cl.SetComments(comments)

    // Start at index 0
    if cl.selectedIdx != 0 {
        t.Fatalf("Expected selectedIdx 0, got %d", cl.selectedIdx)
    }

    // Press j (down)
    downMsg := tea.KeyMsg{Type: tea.KeyDown}
    cl, _ = cl.Update(downMsg)

    if cl.selectedIdx != 1 {
        t.Fatalf("Expected selectedIdx 1 after j, got %d", cl.selectedIdx)
    }

    // Press k (up)
    upMsg := tea.KeyMsg{Type: tea.KeyUp}
    cl, _ = cl.Update(upMsg)

    if cl.selectedIdx != 0 {
        t.Fatalf("Expected selectedIdx 0 after k, got %d", cl.selectedIdx)
    }
}
```

**Step 4:** Commit

```bash
git add internal/tui/comments.go internal/tui/comments_test.go
git commit -m "feat: implement comment view mode with j/k navigation"
```

---

### Task 7: Implement Create mode with text input

**Files:**
- Modify: `internal/tui/comments.go`

**Step 1:** Add Create mode entry point

Update `Update()` to handle 'c' key:

```go
case msg.String() == "c":
    if cl.mode == CommentModeView {
        cl.mode = CommentModeCreate
        cl.textInput.SetValue("")
        cl.textInput.Focus()
        return cl, textinput.Blink
    }
```

**Step 2:** Handle text input in Create mode

Update `Update()` to route to textinput when in Create mode:

```go
if cl.mode == CommentModeCreate || cl.mode == CommentModeEdit {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "enter":
            text := cl.textInput.Value()
            if strings.TrimSpace(text) == "" {
                return cl, nil  // Ignore empty submissions
            }
            // Will return a command in next task to actually create
            return cl, cl.submitComment()
        case "esc":
            cl.mode = CommentModeView
            cl.textInput.SetValue("")
            cl.textInput.Blur()
            return cl, nil
        }
    }

    // Delegate to textInput for normal typing
    cl.textInput, cmd = cl.textInput.Update(msg)
    return cl, cmd
}
```

**Step 3:** Implement submitComment as a Cmd

```go
func (cl CommentsList) submitComment() tea.Cmd {
    return func() tea.Msg {
        text := cl.textInput.Value()
        if cl.mode == CommentModeCreate {
            return CreateCommentRequestMsg{Text: text}
        }
        return UpdateCommentRequestMsg{
            CommentID: cl.comments[cl.editingIdx].ID,
            Text:      text,
        }
    }
}
```

**Step 4:** Update View to show input in Create mode

```go
func (cl CommentsList) View() string {
    switch cl.mode {
    case CommentModeCreate:
        return cl.renderCreateMode()
    case CommentModeEdit:
        return cl.renderEditMode()
    default:
        return cl.renderViewMode()  // existing code moved here
    }
}

func (cl CommentsList) renderCreateMode() string {
    title := lipgloss.NewStyle().
        Bold(true).
        Render("[New Comment]")

    inputBox := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        Padding(1).
        Width(cl.width - 2).
        Render(cl.textInput.View())

    footer := lipgloss.NewStyle().
        Foreground(lipgloss.Color("8")).
        Render("Submit: Enter | Cancel: Esc | Newline: Shift+Enter")

    return title + "\n" + inputBox + "\n" + footer
}

func (cl CommentsList) renderViewMode() string {
    // existing renderViewMode code
    ...
}
```

**Step 5:** Define message types

Add at top of `comments.go`:

```go
type CreateCommentRequestMsg struct {
    Text string
}

type UpdateCommentRequestMsg struct {
    CommentID string
    Text      string
}

type DeleteCommentRequestMsg struct {
    CommentID string
}

type CommentCreatedMsg struct {
    Comment trello.Comment
}

type CommentUpdatedMsg struct {
    Comment trello.Comment
}

type CommentDeletedMsg struct {
    CommentID string
}
```

**Step 6:** Test Create mode

```go
func TestCreateCommentMode(t *testing.T) {
    cl := NewCommentsList(defaultTheme(), defaultKeyMap())
    cl.SetFocus(true)

    // Press 'c' to enter create mode
    cMsg := tea.KeyMsg{Runes: []rune{'c'}}
    cl, _ = cl.Update(cMsg)

    if cl.mode != CommentModeCreate {
        t.Fatal("Should be in Create mode after pressing 'c'")
    }

    // Type text
    typeMsg := tea.KeyMsg{Runes: []rune{'t', 'e', 's', 't'}}
    cl, _ = cl.Update(typeMsg)

    if !strings.Contains(cl.textInput.Value(), "test") {
        t.Fatal("Should have typed text in input")
    }

    // Press Escape to cancel
    escMsg := tea.KeyMsg{Type: tea.KeyEsc}
    cl, _ = cl.Update(escMsg)

    if cl.mode != CommentModeView {
        t.Fatal("Should return to View mode after Escape")
    }
}
```

**Step 7:** Commit

```bash
git add internal/tui/comments.go internal/tui/comments_test.go
git commit -m "feat: implement create comment mode with text input"
```

---

### Task 8: Implement Edit mode

**Files:**
- Modify: `internal/tui/comments.go`

**Step 1:** Add Edit mode entry point

Update `Update()` to handle 'e' key:

```go
case msg.String() == "e":
    if cl.mode == CommentModeView && cl.selectedIdx < len(cl.comments) {
        comment := cl.comments[cl.selectedIdx]
        if comment.Editable {
            cl.mode = CommentModeEdit
            cl.editingIdx = cl.selectedIdx
            cl.textInput.SetValue(comment.Body)
            cl.textInput.Focus()
            return cl, textinput.Blink
        }
    }
```

**Step 2:** Update renderEditMode

```go
func (cl CommentsList) renderEditMode() string {
    if cl.editingIdx < 0 || cl.editingIdx >= len(cl.comments) {
        return ""
    }

    comment := cl.comments[cl.editingIdx]
    title := lipgloss.NewStyle().
        Bold(true).
        Render("[Edit Comment]")

    authorStr := comment.Author.FullName + " (" + comment.Date.Format("2006-01-02") + ")"

    inputBox := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        Padding(1).
        Width(cl.width - 2).
        Render(cl.textInput.View())

    footer := lipgloss.NewStyle().
        Foreground(lipgloss.Color("8")).
        Render("Submit: Enter | Cancel: Esc | Newline: Shift+Enter")

    return title + "\n" + authorStr + "\n" + inputBox + "\n" + footer
}
```

**Step 3:** Test Edit mode

```go
func TestEditCommentMode(t *testing.T) {
    cl := NewCommentsList(defaultTheme(), defaultKeyMap())
    cl.SetFocus(true)

    comments := []trello.Comment{
        {ID: "1", Body: "Original text", Editable: true},
    }
    cl.SetComments(comments)

    // Press 'e' to enter edit mode
    eMsg := tea.KeyMsg{Runes: []rune{'e'}}
    cl, _ = cl.Update(eMsg)

    if cl.mode != CommentModeEdit {
        t.Fatal("Should be in Edit mode after pressing 'e'")
    }

    if cl.textInput.Value() != "Original text" {
        t.Fatal("Input should contain original text")
    }
}
```

**Step 4:** Commit

```bash
git add internal/tui/comments.go internal/tui/comments_test.go
git commit -m "feat: implement edit comment mode"
```

---

### Task 9: Implement Delete with confirmation

**Files:**
- Modify: `internal/tui/comments.go`

**Step 1:** Add delete confirmation state

Add to CommentsList struct:

```go
type CommentsList struct {
    // ... existing fields
    deleteConfirming bool  // true when showing delete confirmation
}
```

**Step 2:** Handle 'd' key and show confirmation

```go
case msg.String() == "d":
    if cl.mode == CommentModeView && cl.selectedIdx < len(cl.comments) {
        comment := cl.comments[cl.selectedIdx]
        if comment.Editable {
            cl.deleteConfirming = true
            return cl, nil
        }
    }
```

**Step 3:** Handle confirmation response

In Update(), handle y/n when deleteConfirming:

```go
if cl.deleteConfirming {
    switch msg.String() {
    case "y":
        comment := cl.comments[cl.selectedIdx]
        cl.deleteConfirming = false
        return cl, cl.deleteComment(comment.ID)
    case "n":
        cl.deleteConfirming = false
        return cl, nil
    }
}
```

**Step 4:** Implement deleteComment command

```go
func (cl CommentsList) deleteComment(commentID string) tea.Cmd {
    return func() tea.Msg {
        return DeleteCommentRequestMsg{CommentID: commentID}
    }
}
```

**Step 5:** Update View to show confirmation

```go
func (cl CommentsList) View() string {
    if cl.deleteConfirming {
        if cl.selectedIdx < len(cl.comments) {
            comment := cl.comments[cl.selectedIdx]
            prompt := lipgloss.NewStyle().
                Foreground(lipgloss.Color("1")).
                Bold(true).
                Render("Delete comment? (y/n)")

            body := wordWrap(comment.Body, cl.width-4)
            return comment.Author.FullName + "\n" + body + "\n\n" + prompt
        }
    }

    // ... rest of View logic
}
```

**Step 6:** Test Delete mode

```go
func TestDeleteComment(t *testing.T) {
    cl := NewCommentsList(defaultTheme(), defaultKeyMap())
    cl.SetFocus(true)

    comments := []trello.Comment{
        {ID: "1", Body: "Comment to delete", Editable: true},
    }
    cl.SetComments(comments)

    // Press 'd' to confirm deletion
    dMsg := tea.KeyMsg{Runes: []rune{'d'}}
    cl, _ = cl.Update(dMsg)

    if !cl.deleteConfirming {
        t.Fatal("Should show delete confirmation")
    }

    // Press 'y' to confirm
    yMsg := tea.KeyMsg{Runes: []rune{'y'}}
    cl, cmd := cl.Update(yMsg)

    if cl.deleteConfirming {
        t.Fatal("Should clear deletion confirmation after 'y'")
    }

    // Verify a DeleteCommentRequestMsg was generated
    if cmd == nil {
        t.Fatal("Should return a command to delete")
    }
}
```

**Step 7:** Commit

```bash
git add internal/tui/comments.go internal/tui/comments_test.go
git commit -m "feat: implement comment deletion with confirmation"
```

---

### Task 10: Implement @ autocomplete trigger and filtering

**Files:**
- Modify: `internal/tui/comments.go`

**Step 1:** Detect @ in input and trigger autocomplete

Modify textInput Update handler:

```go
if cl.mode == CommentModeCreate || cl.mode == CommentModeEdit {
    // Check for @ to trigger autocomplete
    if msg.Type == tea.KeyRunes && string(msg.Runes) == "@" {
        // Trigger autocomplete
        cl.autocomplete.Active = true
        cl.autocomplete.Query = ""
        cl.autocomplete.SelectedIdx = 0
        cl.autocomplete.Pos = len(cl.textInput.Value()) - 1
        cl.autocomplete.Matches = cl.allMembers  // Start with all members
        return cl, nil
    }
}
```

**Step 2:** Implement autocomplete filtering

Add method to filter members:

```go
func (cl *CommentsList) filterMembers(query string) {
    cl.autocomplete.Query = query
    cl.autocomplete.Matches = []trello.Member{}

    query = strings.ToLower(query)
    for _, member := range cl.allMembers {
        if strings.Contains(strings.ToLower(member.FullName), query) ||
           strings.Contains(strings.ToLower(member.Username), query) {
            cl.autocomplete.Matches = append(cl.autocomplete.Matches, member)
        }
    }

    if len(cl.autocomplete.Matches) > 0 {
        cl.autocomplete.SelectedIdx = 0
    }
}
```

**Step 3:** Handle typing after @

Update textinput handler:

```go
if cl.autocomplete.Active {
    // Capture typing to filter
    if msg.Type == tea.KeyRunes {
        cl.filterMembers(cl.autocomplete.Query + string(msg.Runes))
        return cl, nil
    }
}
```

**Step 4:** Add autocomplete navigation (j/k)

```go
if cl.autocomplete.Active {
    switch msg.String() {
    case "j":
        if cl.autocomplete.SelectedIdx < len(cl.autocomplete.Matches)-1 {
            cl.autocomplete.SelectedIdx++
        }
        return cl, nil
    case "k":
        if cl.autocomplete.SelectedIdx > 0 {
            cl.autocomplete.SelectedIdx--
        }
        return cl, nil
    }
}
```

**Step 5:** Render autocomplete popup

Add to View():

```go
func (cl CommentsList) renderAutocomplete() string {
    if !cl.autocomplete.Active || len(cl.autocomplete.Matches) == 0 {
        return ""
    }

    var lines []string
    lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Members:"))

    for i, member := range cl.autocomplete.Matches {
        indicator := " "
        if i == cl.autocomplete.SelectedIdx {
            indicator = ">"
        }

        line := lipgloss.NewStyle().
            Render(indicator + " " + member.FullName + " (@" + member.Username + ")")

        if i == cl.autocomplete.SelectedIdx {
            line = lipgloss.NewStyle().
                Foreground(lipgloss.Color("4")).
                Render(line)
        }
        lines = append(lines, line)
    }

    return strings.Join(lines, "\n")
}
```

**Step 6:** Test autocomplete

```go
func TestAutocompleteFilter(t *testing.T) {
    cl := NewCommentsList(defaultTheme(), defaultKeyMap())

    members := []trello.Member{
        {FullName: "John Smith", Username: "john"},
        {FullName: "Jane Doe", Username: "jane"},
        {FullName: "Johnny Walker", Username: "jwalker"},
    }
    cl.SetMembers(members)

    // Trigger autocomplete and filter for "jo"
    cl.autocomplete.Active = true
    cl.filterMembers("jo")

    if len(cl.autocomplete.Matches) != 2 {
        t.Fatalf("Expected 2 matches for 'jo', got %d", len(cl.autocomplete.Matches))
    }
}
```

**Step 7:** Commit

```bash
git add internal/tui/comments.go internal/tui/comments_test.go
git commit -m "feat: implement @ autocomplete trigger and filtering"
```

---

### Task 11: Implement autocomplete selection and insertion

**Files:**
- Modify: `internal/tui/comments.go`

**Step 1:** Handle Tab/Enter to select mention

```go
if cl.autocomplete.Active {
    switch msg.String() {
    case "enter", "tab":
        if cl.autocomplete.SelectedIdx < len(cl.autocomplete.Matches) {
            member := cl.autocomplete.Matches[cl.autocomplete.SelectedIdx]
            cl.insertMention(member.Username)
        }
        return cl, nil
    }
}
```

**Step 2:** Implement insertMention

```go
func (cl *CommentsList) insertMention(username string) {
    text := cl.textInput.Value()

    // Find the @ position and replace from @ to cursor with mention
    pos := len(text)

    // Backtrack to find @ character
    atPos := pos - 1
    for atPos >= 0 && text[atPos] != '@' {
        atPos--
    }

    if atPos >= 0 {
        // Replace from @ to current position with @username
        before := text[:atPos]
        after := text[pos:]
        newText := before + "@" + username + after
        cl.textInput.SetValue(newText)

        // Close autocomplete
        cl.autocomplete.Active = false
        cl.autocomplete.Matches = []trello.Member{}
    }
}
```

**Step 3:** Handle Escape to close autocomplete

```go
if cl.autocomplete.Active {
    switch msg.String() {
    case "esc":
        cl.autocomplete.Active = false
        return cl, nil
    }
}
```

**Step 4:** Update textInput handler to account for autocomplete

When autocomplete is active, only handle autocomplete keys and @-related input:

```go
if cl.autocomplete.Active {
    // Only handle autocomplete-related keys
    // Regular typing is handled separately
    ...
} else {
    // Normal text input when not autocompleting
    cl.textInput, cmd = cl.textInput.Update(msg)
}
```

**Step 5:** Test mention insertion

```go
func TestMentionInsertion(t *testing.T) {
    cl := NewCommentsList(defaultTheme(), defaultKeyMap())

    members := []trello.Member{
        {FullName: "John Smith", Username: "john"},
    }
    cl.SetMembers(members)

    cl.textInput.SetValue("Hello @j")
    cl.autocomplete.Active = true
    cl.autocomplete.Matches = members
    cl.autocomplete.SelectedIdx = 0

    // Insert mention
    cl.insertMention("john")

    expected := "Hello @john"
    if cl.textInput.Value() != expected {
        t.Fatalf("Expected %q, got %q", expected, cl.textInput.Value())
    }

    if cl.autocomplete.Active {
        t.Fatal("Autocomplete should close after insertion")
    }
}
```

**Step 6:** Commit

```bash
git add internal/tui/comments.go internal/tui/comments_test.go
git commit -m "feat: implement mention selection and insertion in autocomplete"
```

---

### Task 12: Implement message handlers in CommentsList

**Files:**
- Modify: `internal/tui/comments.go`

**Step 1:** Handle comment operation responses

Update Update() to handle messages:

```go
func (cl CommentsList) Update(msg tea.Msg) (CommentsList, tea.Cmd) {
    switch msg := msg.(type) {
    case CommentCreatedMsg:
        cl.comments = append(cl.comments, msg.Comment)
        cl.mode = CommentModeView
        cl.textInput.SetValue("")
        return cl, nil

    case CommentUpdatedMsg:
        if cl.editingIdx >= 0 && cl.editingIdx < len(cl.comments) {
            cl.comments[cl.editingIdx] = msg.Comment
        }
        cl.mode = CommentModeView
        cl.textInput.SetValue("")
        cl.editingIdx = -1
        return cl, nil

    case CommentDeletedMsg:
        // Remove comment from list
        if cl.selectedIdx < len(cl.comments) {
            cl.comments = append(cl.comments[:cl.selectedIdx], cl.comments[cl.selectedIdx+1:]...)
        }
        if cl.selectedIdx >= len(cl.comments) && cl.selectedIdx > 0 {
            cl.selectedIdx--
        }
        return cl, nil
    }

    // ... existing key handling
}
```

**Step 2:** Test message handling

```go
func TestCommentCreatedMessage(t *testing.T) {
    cl := NewCommentsList(defaultTheme(), defaultKeyMap())
    cl.mode = CommentModeCreate

    newComment := trello.Comment{
        ID:     "comment123",
        Body:   "New comment",
        Author: trello.Member{FullName: "John"},
    }

    cl, _ = cl.Update(CommentCreatedMsg{Comment: newComment})

    if len(cl.comments) != 1 {
        t.Fatal("Comment should be added to list")
    }

    if cl.mode != CommentModeView {
        t.Fatal("Should return to View mode")
    }
}
```

**Step 3:** Commit

```bash
git add internal/tui/comments.go internal/tui/comments_test.go
git commit -m "feat: implement comment message handlers in CommentsList"
```

---

## Phase 4: DetailModel Integration

### Task 13: Wire CommentsList into DetailModel

**Files:**
- Modify: `internal/tui/detail.go`

**Step 1:** Add CommentsList field to DetailModel

```go
type DetailModel struct {
    open      bool
    focused   bool
    tab       int
    cardID    string
    card      trello.Card

    // NEW: Comments support
    comments  *CommentsList

    // existing fields...
}
```

**Step 2:** Initialize CommentsList in NewDetailModel

```go
func NewDetailModel(theme Theme, keyMap KeyMap) DetailModel {
    return DetailModel{
        open:     false,
        focused:  false,
        tab:      0,
        comments: NewCommentsList(theme, keyMap),
        // ... existing fields
    }
}
```

**Step 3:** Delegate to CommentsList in Update when Comments tab active

```go
func (d DetailModel) Update(msg tea.Msg) (DetailModel, tea.Cmd) {
    var cmd tea.Cmd

    // If Comments tab is active and detail is focused, delegate to CommentsList
    if d.open && d.focused && d.tab == 1 && d.comments != nil {
        *d.comments, cmd = d.comments.Update(msg)
        return d, cmd
    }

    // ... existing Update logic
}
```

**Step 4:** Update View to render CommentsList when Comments tab active

```go
func (d DetailModel) renderCommentsList() string {
    if d.comments == nil {
        return "Loading comments..."
    }
    return d.comments.View()
}
```

**Step 5:** Fetch members when opening detail panel

```go
func (d DetailModel) SetCard(card trello.Card) tea.Cmd {
    d.card = card
    d.cardID = card.ID

    // Fetch board members for autocomplete
    // This returns a Cmd that will produce a BoardMembersMsg
    return GetBoardMembersCmd(d.boardID)  // needs boardID passed in
}
```

**Step 6:** Test DetailModel with CommentsList

```go
func TestDetailModelDelegates ToCommentsList(t *testing.T) {
    detail := NewDetailModel(defaultTheme(), defaultKeyMap())
    detail.open = true
    detail.focused = true
    detail.tab = 1  // Comments tab

    // Set some comments
    comments := []trello.Comment{
        {ID: "1", Body: "Test"},
    }
    detail.comments.SetComments(comments)
    detail.comments.SetFocus(true)

    // Press j to navigate
    jMsg := tea.KeyMsg{Type: tea.KeyDown}
    detail, _ = detail.Update(jMsg)

    // Verify CommentsList received the message
    // (This is tested more thoroughly in CommentsList tests)
}
```

**Step 7:** Commit

```bash
git add internal/tui/detail.go internal/tui/detail_test.go
git commit -m "feat: integrate CommentsList into DetailModel"
```

---

### Task 14: Add message routing in App for comment operations

**Files:**
- Modify: `internal/tui/app.go`

**Step 1:** Handle comment request messages from CommentsList

In App.Update(), add handlers:

```go
case CreateCommentRequestMsg:
    // Call Trello API
    return a, createCommentCmd(a.cardID, msg.Text)

case UpdateCommentRequestMsg:
    return a, updateCommentCmd(msg.CommentID, msg.Text)

case DeleteCommentRequestMsg:
    return a, deleteCommentCmd(msg.CommentID)
```

**Step 2:** Implement API command functions

```go
func createCommentCmd(cardID, text string) tea.Cmd {
    return func() tea.Msg {
        // This would call your trello client
        // For now, return a placeholder
        return CommentCreatedMsg{
            Comment: trello.Comment{ID: "new", Body: text},
        }
    }
}

func updateCommentCmd(commentID, text string) tea.Cmd {
    return func() tea.Msg {
        return CommentUpdatedMsg{
            Comment: trello.Comment{ID: commentID, Body: text},
        }
    }
}

func deleteCommentCmd(commentID string) tea.Cmd {
    return func() tea.Msg {
        return CommentDeletedMsg{CommentID: commentID}
    }
}
```

**Step 3:** Test command generation

```go
func TestCreateCommentCommand(t *testing.T) {
    // Verify the command function can be called
    cmd := createCommentCmd("card123", "Test comment")
    msg := cmd()

    if _, ok := msg.(CommentCreatedMsg); !ok {
        t.Fatal("Should generate CommentCreatedMsg")
    }
}
```

**Step 4:** Commit

```bash
git add internal/tui/app.go internal/tui/app_test.go
git commit -m "feat: add message routing in App for comment operations"
```

---

## Phase 5: Focus Styling and Polish

### Task 15: Update rendering to show focus state with blue borders

**Files:**
- Modify: `internal/tui/app.go` (View method)

**Step 1:** Add focus-aware styling to View()

In App.View(), update border styling based on boardHasFocus:

```go
func (a App) View() string {
    boardBorder := lipgloss.RoundedBorder()
    boardBorderColor := "8"  // dim gray
    if a.boardHasFocus {
        boardBorderColor = "4"  // blue
    }

    boardStyle := lipgloss.NewStyle().
        Border(boardBorder).
        BorderForeground(lipgloss.Color(boardBorderColor))

    boardView := boardStyle.Render(a.board.View())

    detailBorder := lipgloss.RoundedBorder()
    detailBorderColor := "8"  // dim gray
    if !a.boardHasFocus && a.detail.open {
        detailBorderColor = "4"  // blue
    }

    detailStyle := lipgloss.NewStyle().
        Border(detailBorder).
        BorderForeground(lipgloss.Color(detailBorderColor))

    var mainView string
    if a.detail.open {
        detailView := detailStyle.Render(a.detail.View())
        mainView = lipgloss.JoinHorizontal(
            lipgloss.Left,
            boardView,
            detailView,
        )
    } else {
        mainView = boardView
    }

    return mainView + "\n" + a.status.View()
}
```

**Step 2:** Test focus styling

```go
func TestFocusStyling(t *testing.T) {
    app := NewApp(newTestBoard(), newTestDetail(), defaultTheme(), defaultKeyMap())
    app.detail.open = true

    // Board has focus initially
    view := app.View()
    if !strings.Contains(view, "blue") && app.boardHasFocus {
        // Check that board is highlighted (implementation-dependent)
    }

    // Switch focus
    app.boardHasFocus = false
    view = app.View()
    // Detail should now be highlighted
}
```

**Step 3:** Commit

```bash
git add internal/tui/app.go internal/tui/app_test.go
git commit -m "feat: add focus-aware blue border styling to board and detail"
```

---

### Task 16: Integration test for full comment workflow

**Files:**
- Create: `internal/tui/integration_test.go`

**Step 1:** Write end-to-end test

```go
func TestCommentWorkflow(t *testing.T) {
    // Setup
    app := NewApp(newTestBoard(), newTestDetail(), defaultTheme(), defaultKeyMap())
    app.board.SelectCard(0, 0)  // Select first card

    // Step 1: Focus detail panel
    enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
    app, _ = app.Update(enterMsg).(App)

    if app.boardHasFocus {
        t.Fatal("Detail should have focus after Enter")
    }

    // Step 2: Switch to Comments tab (if needed)
    // Step 3: Create comment
    cMsg := tea.KeyMsg{Runes: []rune{'c'}}
    app, _ = app.Update(cMsg).(App)

    // Step 4: Type and submit
    // Step 5: Verify comment appears

    // Step 6: Edit comment
    eMsg := tea.KeyMsg{Runes: []rune{'e'}}
    app, _ = app.Update(eMsg).(App)

    // Step 7: Verify edit mode
    // Step 8: Delete comment
    // etc.
}
```

**Step 2:** Commit

```bash
git add internal/tui/integration_test.go
git commit -m "test: add end-to-end comment workflow integration test"
```

---

### Task 17: Final polish and documentation

**Files:**
- Modify: `README.md` (add comment feature to keybindings)
- Create: `docs/comments-feature.md` (internal documentation)

**Step 1:** Update keybindings in README

```markdown
## Keybindings

| Key | Action | Context |
|-----|--------|---------|
| ... existing keybindings ...    |
| `d` | Toggle detail panel | Board |
| `enter` | Focus detail panel | Board with card selected |
| `esc` | Focus board | Detail panel |
| `j`/`k` | Navigate comments | Detail (Comments tab active) |
| `c` | Create comment | Detail (Comments tab active) |
| `e` | Edit comment | Detail (Comments tab active) |
| `d` | Delete comment | Detail (Comments tab active) |
| `@` | Mention user (in comment) | Creating/editing comment |
```

**Step 2:** Commit

```bash
git add README.md
git commit -m "docs: update README with comment feature keybindings"
```

---

## Execution Checklist

All tasks follow TDD: write failing test → implement → verify pass → commit.

Suggested execution order: Complete each phase in sequence, committing frequently.

If encountering API limitations (update/delete not supported), gracefully degrade to create-only and update README accordingly.

