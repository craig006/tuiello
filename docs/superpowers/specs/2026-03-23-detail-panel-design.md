# Detail Panel Design Spec

## Overview

A toggleable detail panel on the right side of the screen that shows full details of the currently selected Trello card. The panel has three tabbed views: Overview, Comments, and Checklist.

## Layout

- **Position:** Fixed to the right side of the screen
- **Width split:** 60% board columns / 40% detail panel
- **Toggle:** Press `d` to open/close
- **Border:** Rounded border matching the active border color (ANSI 4)
- **Tab bar:** Embedded in the top border (same style as column titles)
- When closed, board columns take full width as they do today

## Tabs

Three tabs cycled with `[` (previous) and `]` (next):

1. **Overview** — card metadata and description
2. **Comments** — chronological list of card comments
3. **Checklist** — checklist items with completion status

Active tab is highlighted with the active border color. Inactive tabs are dimmed (ANSI 8).

Tab keybindings only fire when the panel is open.

## Tab Content

### 1. Overview

Displayed top-to-bottom. All data comes from the existing `Card` struct — no additional fetch required.

- **Title** — card name, bold
- **Labels** — horizontal row of colored `⏺` indicators followed by label name. Uses existing `trelloColorToANSI` mapping.
- **Assignees** — horizontal row of member full names (from `Member.FullName`), comma-separated
- **Description** — card description text, word-wrapped to `panelWidth - padding`, in a scrollable viewport. If empty, show dimmed "No description."

### 2. Comments

- **Data source:** `GET /cards/{id}/actions?filter=commentCard` — fetched lazily on first view of this tab for a given card
- **Layout per comment:**
  - Line 1: `Author Name (YYYY-MM-DD)` — author in bold, date dimmed
  - Line 2+: Comment body text, word-wrapped to panel width
  - Separator line between comments
- **Reactions:** Deferred to a follow-up. The Trello reactions API requires a separate request per comment (N+1 problem), adding significant latency for marginal value. The `Reaction` type is included in the data model for future use but not fetched or rendered in this iteration.
- **Scrollable** via bubbles/viewport
- If no comments, show dimmed "No comments."
- **Loading state:** Show "Loading comments..." while fetching

### 3. Checklist

- **Data source:** `GET /cards/{id}/checklists` — fetched lazily on first view of this tab for a given card
- **Layout:**
  - If multiple checklists, show checklist name as a header (bold)
  - Each item: `[x]` or `[ ]` followed by item name
  - Completed items shown in dimmed color (ANSI 8)
- **Scrollable** via bubbles/viewport
- If no checklists, show dimmed "No checklists."
- **Loading state:** Show "Loading checklists..." while fetching

## Architecture

### DetailModel (internal/tui/detail.go)

`DetailModel` is a self-contained Bubble Tea component that owns all detail panel state. `App` holds a single `detail DetailModel` field and forwards relevant messages to it.

```go
type DetailModel struct {
    // Panel state
    open      bool
    tab       int    // 0=Overview, 1=Comments, 2=Checklist
    cardID    string // ID of the card currently displayed

    // Cached data
    card       trello.Card        // copy of the currently displayed card
    comments   []trello.Comment
    checklists []trello.Checklist

    // Loading flags
    commentsLoaded   bool
    checklistsLoaded bool
    loading          bool
    loadingErr       string

    // Rendering
    viewport   viewport.Model  // scrollable content viewport
    width      int
    height     int
    keyMap     KeyMap
    theme      Theme
}
```

Methods:
- `Update(msg tea.Msg) (DetailModel, tea.Cmd)` — handles tab switching, viewport scrolling
- `View() string` — renders the panel with border, tab bar, and content
- `SetCard(card trello.Card) tea.Cmd` — called when selected card changes; copies card, clears caches, returns fetch command if needed
- `Toggle() tea.Cmd` — toggles open/closed, returns fetch command if opening with stale data
- `SetSize(width, height int)` — updates dimensions

### State Ownership

- `DetailModel` owns: tab index, cached comments/checklists, loading flags, viewport, card reference
- `App` owns: `detail DetailModel` field, `detailOpen` convenience accessor via `detail.open`
- `App` is responsible for: detecting card selection changes, forwarding messages, calling `detail.SetCard()` when needed

### Detecting Card Selection Changes

After forwarding key messages to `BoardModel.Update()` (for j/k navigation), `App` compares the currently selected card ID against `detail.cardID`. If they differ and the panel is open, `App` calls `detail.SetCard()` with the new card. This compare-after-update approach avoids adding a new message type and works with the existing board navigation.

```go
// In App.Update, after board update:
if a.detail.open {
    if card, _, ok := a.board.SelectedCard(); ok && card.ID != a.detail.cardID {
        cmd = a.detail.SetCard(card)
    }
}
```

### Viewport Scrolling

When the detail panel is open, `j`/`k` continue to navigate cards in the board (the panel updates to reflect the new selection). The viewport within the detail panel scrolls with **`ctrl+j`/`ctrl+k`** (or `ctrl+d`/`ctrl+u` for half-page). These bindings only activate when the panel is open.

## Data Flow

### Event Flow

1. **`d` pressed:** Call `detail.Toggle()`. If opening with no card loaded, call `detail.SetCard()` with the currently selected card.
2. **`[` / `]` pressed (panel open):** Cycle `detail.tab`. If the new tab needs data not yet cached (Comments or Checklists), trigger a lazy fetch.
3. **Card selection changes (panel open):** After `board.Update()`, compare card IDs. If different, call `detail.SetCard()` which clears caches and returns a fetch command for the active tab (if tab 1 or 2). Overview tab (0) never triggers a fetch.
4. **Fetch completes:** Message includes `CardID` for stale-response detection. If `CardID` matches `detail.cardID`, store data and set loaded flag. If mismatched, ignore.
5. **Fetch error:** Show error message in the panel content area (e.g., "Failed to load comments").
6. **Board refresh (`r`):** After `BoardFetchedMsg`, clear detail panel state and close it. User can reopen with `d`.
7. **Window resize:** When `detailOpen`, set `board.width = msg.Width * 60 / 100` and `detail.SetSize(msg.Width - boardWidth, msg.Height)`.

### New Messages

All fetch-result messages include `CardID` for stale-response detection:

- `CardCommentsMsg { CardID string; Comments []trello.Comment }`
- `CardCommentsFetchErrMsg { CardID string; Err error }`
- `CardChecklistsMsg { CardID string; Checklists []trello.Checklist }`
- `CardChecklistsFetchErrMsg { CardID string; Err error }`

## New Trello API Types

```go
type Comment struct {
    ID     string
    Author Member
    Body   string
    Date   time.Time
}

type Checklist struct {
    ID    string
    Name  string
    Items []CheckItem
}

type CheckItem struct {
    ID       string
    Name     string
    Complete bool
}
```

Date is parsed from the API's ISO 8601 string using `time.Parse(time.RFC3339, dateStr)`.

## New Trello API Methods

### FetchCardComments(cardID string) ([]Comment, error)

```
GET /1/cards/{cardID}/actions?filter=commentCard&fields=data,date,idMemberCreator,memberCreator&memberCreator_fields=fullName,initials,username
```

Parse `action.data.text` for comment body, `action.memberCreator` for author, `action.date` for timestamp.

### FetchCardChecklists(cardID string) ([]Checklist, error)

```
GET /1/cards/{cardID}/checklists?fields=name&checkItem_fields=name,state
```

Parse `checkItem.state` — `"complete"` or `"incomplete"`.

## Rendering

### Board View Changes

When `detail.open` is true:

1. Calculate board width as `totalWidth * 60 / 100`
2. Calculate panel width as `totalWidth - boardWidth`
3. Render board at `boardWidth` (columns resize via `ResizeColumns()`)
4. Render detail panel at `panelWidth`
5. Join horizontally with `lipgloss.JoinHorizontal`

When `detail.open` is false:

- Board renders at full width as it does today

### Keybinding Changes

New keybindings added to `KeyMap`:

- `DetailToggle` — `d`
- `DetailTabPrev` — `[`
- `DetailTabNext` — `]`
- `DetailScrollDown` — `ctrl+j`
- `DetailScrollUp` — `ctrl+k`

These are added to the existing `KeyMap` struct and `KeybindingConfig` under a new `Detail` section:

```yaml
keybinding:
  detail:
    toggle: "d"
    tabPrev: "["
    tabNext: "]"
    scrollDown: "ctrl+j"
    scrollUp: "ctrl+k"
```

Handled in `App.Update()`:
- `DetailToggle` always fires (when `boardReady`)
- All other detail keybindings only fire when `detail.open` is true

## Edge Cases

- **No card selected (empty lists):** Panel opens and shows "No card selected" message
- **Board not ready (`boardReady` false):** Panel toggle is a no-op
- **Terminal too narrow:** If width < 80, hide the panel and show a status message
- **Card moved while panel open:** Panel stays showing that card's details (data is still valid)
- **Board refresh while panel open:** Clear detail state and close panel
- **Rapid card switching:** Stale fetch responses are ignored by comparing `CardID` in the message against `detail.cardID`
- **Window resize while panel open:** Recalculate 60/40 split and call `board.ResizeColumns()` and `detail.SetSize()`
