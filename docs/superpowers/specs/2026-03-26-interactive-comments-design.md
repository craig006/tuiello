# Interactive Comments with Mention Autocomplete

**Date:** 2026-03-26
**Status:** Approved
**Feature:** Create, edit, and delete comments with inline username mentions

## Overview

Enhance the Comments tab in the detail panel to allow users to create, edit, and delete comments directly in the TUI. Users can mention other board members using `@` syntax with autocomplete suggestions. The detail panel gains focus management — Enter focuses it, Escape unfocuses it and returns to board navigation.

## Requirements

- **Create comments:** Press `c` to open input at top of comment list
- **Edit comments:** Press `e` on selected comment to edit in place
- **Delete comments:** Press `d` on selected comment (with confirmation)
- **Navigation:** j/k to move between comments when detail panel is focused
- **Focus toggle:** Enter focuses detail panel, Escape returns to board
- **Mentions:** Type `@` to trigger autocomplete popup showing board members
- **Permissions:** Respect API flags for editable comments; default to owner-only
- **Submit:** Enter to submit, Shift+Enter for newline, Escape to cancel

## Architecture

### CommentsList Component

A new sub-component within DetailModel that owns all comment interaction:

```go
type CommentsList struct {
    // Data
    comments  []trello.Comment
    allMembers []trello.Member

    // Selection & modes
    selectedIdx int
    mode        CommentMode  // View, Create, Edit
    editingIdx  int

    // Input
    textInput   textinput.Model

    // Autocomplete
    autocomplete AutocompleteState

    // Rendering
    viewport    viewport.Model
    width       int
    height      int
    theme       Theme
    keyMap      KeyMap
}

type CommentMode int
const (
    ModeView   CommentMode = iota
    ModeCreate
    ModeEdit
)

type AutocompleteState struct {
    active      bool
    matches     []trello.Member
    selectedIdx int
    query       string
    pos         int  // cursor position of @
}
```

### Component Hierarchy

```
App
  └─ DetailModel
      ├─ viewport (for all tab content)
      └─ CommentsList (when Comments tab is active)
          ├─ textInput (Create/Edit mode)
          ├─ autocomplete popup
          └─ comments list
```

### State Ownership

- **DetailModel** owns: tab index, panel focus state, which card is shown
- **CommentsList** owns: selected comment index, edit/create mode, input text, autocomplete state
- **App** owns: board focus state, top-level focus toggle logic

## Interaction Model

### Modes

**View Mode** (default)
- Comments displayed as scrollable list
- Selected comment has blue bar on left (matching card selection style)
- Keybindings:
  - `j`/`k`: Move selection up/down
  - `c`: Enter Create mode
  - `e`: Enter Edit mode on selected comment (if editable)
  - `d`: Delete selected comment (with y/n confirmation)
  - `Escape`: Return focus to board

**Create Mode**
- Text input box appears at top of comment list
- Initial height: 5 lines, expands as user types
- Keybindings:
  - `Shift+Enter`: Insert newline
  - `@`: Trigger autocomplete popup
  - `Enter`: Submit comment, return to View mode
  - `Escape`: Cancel, discard input, return to View mode

**Edit Mode**
- Selected comment replaced with editable text input (in place)
- Same height/expansion as Create mode
- Same keybindings as Create mode
- Submit calls update endpoint, returns to View mode
- Only available if API marks comment as editable

### Focus Management

**Board Focus (default)**
- Board border highlighted in blue
- Detail panel border dimmed
- j/k navigate cards
- Enter focuses detail panel
- Detail panel still visible and updates when card changes

**Detail Focus**
- Detail panel border highlighted in blue
- Board border dimmed
- j/k navigate comments (when Comments tab active)
- Escape returns to board focus
- Board still visible but doesn't respond to j/k

New keybinding:
```yaml
keybinding:
  universal:
    focusDetail: "enter"
    focusBoard: "escape"
```

These bindings work intelligently:
- Enter focuses detail only if board has focus and a card is selected
- Escape unfocuses detail only if detail has focus

## Autocomplete Behavior

### Trigger

Typing `@` in Create or Edit mode opens an autocomplete popup:
- Shows all board members, sorted alphabetically
- Filters in real-time as user types after `@`
- Matches against member name and username

### Navigation & Selection

- `j`/`k` navigate matches
- `Tab` or `Enter` selects highlighted member
- Selection inserts `@username` at cursor, replacing partial text
- Popup closes, cursor positioned after insertion
- User can continue typing or mention more members

### Display

- Popup appears below input (above if space is tight)
- Bordered with theme colors
- Selected option highlighted
- Shows fade-out hint if more matches exist
- Max height: fit available space in panel

## Data Flow

### Creating a Comment

```
User presses 'c'
  → CommentsList.Update()
    → set mode = Create
    → focus textInput

User types text, optionally mentions with @
User presses Enter
  → CommentsList.Update()
    → validate text not empty
    → send CreateCommentCmd(cardID, text)

Trello API responds (async)
  → CreateCommentMsg
  → CommentsList.Update()
    → append comment to list
    → clear input
    → set mode = View
    → stay on newly created comment
```

### Editing a Comment

```
User selects comment, presses 'e'
  → CommentsList.Update()
    → if editable:
        set mode = Edit
        load comment text into input
        editingIdx = selectedIdx
      else:
        show "Not editable"

User modifies text
User presses Enter
  → CommentsList.Update()
    → send UpdateCommentCmd(commentID, text)

Trello API responds (async)
  → UpdateCommentMsg
  → CommentsList.Update()
    → update comment in list
    → clear input
    → set mode = View
```

### Deleting a Comment

```
User presses 'd' on selected comment
  → CommentsList.Update()
    → if editable:
        show prompt: "Delete comment? (y/n)"
      else:
        show "Not deletable"

User presses 'y'
  → CommentsList.Update()
    → send DeleteCommentCmd(commentID)

Trello API responds (async)
  → DeleteCommentMsg
  → CommentsList.Update()
    → remove comment from list
    → adjust selectedIdx if needed
    → stay in View mode
```

## API Integration

### New Trello Client Methods

```go
// CreateComment posts a new comment to a card
CreateComment(cardID, text string) (Comment, error)

// UpdateComment modifies an existing comment
// Returns error if API doesn't support updates
UpdateComment(commentID, text string) (Comment, error)

// DeleteComment removes a comment
// Returns error if API doesn't support deletion
DeleteComment(commentID string) error

// GetBoardMembers fetches all members for autocomplete
GetBoardMembers(boardID string) ([]Member, error)
```

### Trello Endpoints

- **Create:** `POST /1/cards/{cardID}/actions/comments?text={text}`
- **Update:** `PUT /1/actions/{commentID}?text={text}` (if available)
- **Delete:** `DELETE /1/actions/{commentID}` (if available)
- **Members:** `GET /1/boards/{boardID}?members=open&member_fields=fullName,username,initials,avatarHash`

### API Limitations

Trello's API may not support updating or deleting comments. Handle gracefully:
- If endpoints unavailable: show "Editing/deleting not supported" in status bar
- Check API response for `canDelete` or `editable` flags
- Default: assume user can only edit/delete their own comments (check comment.memberCreator against current user)

## Message Types

New messages for comment operations:

```go
// Request commands
CreateCommentCmd(cardID, text) tea.Cmd
UpdateCommentCmd(commentID, text) tea.Cmd
DeleteCommentCmd(commentID) tea.Cmd

// Response messages
CreateCommentMsg { CommentID string; Comment Comment }
CreateCommentErrMsg { CardID string; Err error }

UpdateCommentMsg { CommentID string; Comment Comment }
UpdateCommentErrMsg { CommentID string; Err error }

DeleteCommentMsg { CommentID string }
DeleteCommentErrMsg { CommentID string; Err error }

// Autocomplete suggestion
AutocompleteMemberMsg { Member Member }
```

## Rendering

### Comment List (View Mode)

```
Author Name (2026-03-25)
│ This is the comment text, word-wrapped to fit the
│ available panel width with proper indentation.
├─────────────────────────
│
Author Name 2 (2026-03-24)
│ Another comment here.
│
├─────────────────────────
│
Press 'c' to create, 'e' to edit, 'd' to delete
```

Selected comment (indicated by blue bar on left):

```
Author Name (2026-03-25)
│ Selected comment text
```

### Create/Edit Mode

```
[New Comment]
┌─────────────────────────┐
│ User's typed text here  │
│ with newlines if        │
│ Shift+Enter was used    │
│                         │
│                         │
└─────────────────────────┘
Mentions: @john, @sarah
Submit: Enter | Cancel: Esc
```

### Autocomplete Popup

```
[New Comment]
┌─────────────────────────┐
│ Type here @jo           │
└─────────────────────────┘
 ┌─ Mentions ──────────────┐
 │ > John Smith (@john)    │
 │   Jo Martin (@jmartin)  │
 │   ...                   │
 └─────────────────────────┘
```

## Edge Cases

- **User edits card while composing comment:** Input is preserved (local state only)
- **Board refreshes while detail panel open:** Detail panel state stays (comment list may be stale until re-fetch)
- **Comment no longer exists:** Show "Comment was deleted" and remove from list
- **Network error during submit:** Show error in status bar, keep input intact for retry
- **Autocomplete matches none:** Show empty list, allow free text submission
- **Very long comment:** Input grows to available height, becomes scrollable
- **Terminal too narrow:** Hide detail panel, show message (existing behavior)
- **Permission denied:** Show "You don't have permission to edit this comment"
- **Rapid focus toggle:** Detail panel state preserved when returning to focus

## Testing Strategy

### Unit Tests

- CommentsList: selection navigation (j/k)
- CommentsList: mode transitions (View → Create → View)
- CommentsList: text input with newlines
- Autocomplete: filtering and selection
- Autocomplete: inserting mention at cursor

### Integration Tests

- Create comment flow: input → submit → list updated
- Edit comment flow: select → edit → submit → list updated
- Delete comment flow: select → confirm → list updated
- Focus toggle: Enter focuses, Escape unfocuses
- Comments persist when switching between cards

### Manual Testing

- Compose with multiple mentions
- Very long comments with wrapping
- Edit and delete permission restrictions
- Network error recovery
- Terminal resize during edit

## Trello API Limitations

Before implementation, verify:
1. Can the Trello API update comments?
2. Can the Trello API delete comments?
3. Does the API expose edit/delete permissions?
4. What fields does comment response include?

If updates/deletes aren't available, we can still support viewing and creating comments, and gracefully hide edit/delete options.

## Future Enhancements (Post-MVP)

- Comment reactions (emoji)
- Edit history / revisions
- Comment threading / replies
- Formatting (markdown, code blocks)
- Bulk comment actions
- Comment search/filter
