# Filter/Search Omni Bar Design

## Goal

Add a filter/search bar to tuillo that allows users to filter visible cards by text (case-insensitive substring match on card name), member, and label. Filters can be entered via keyboard text input or through multiselect modals. The search bar is the single source of truth for all active filters.

## Architecture

Filtering is a view-layer concern. The `BoardModel` holds a `Filter` struct. When filters are active, each column rebuilds its visible item list by running cards through the filter. The underlying card data is untouched. Navigation indices operate on the filtered subset.

The search bar, member modal, and label modal are UI components owned by `App`. They push filter state down to the board model.

## Search Bar

### Visual

A `textinput.Model` rendered above the breadcrumb nav bar, inside a single-line bordered box matching the breadcrumb's style (rounded border, grey). The box spans the full board width (`b.width`).

Layout (top to bottom):
1. Search bar (3 lines: border + content + border)
2. Breadcrumb nav (3 lines: border + content + border)
3. Columns (remaining height: `b.height - 6`)

The search bar is always visible. When empty and unfocused, it shows a dimmed placeholder: ` Search...` (where `` is nerdfont `U+F002`).

### Active Filter Display

When filters are active and the bar is not focused, it displays the full filter string:

```
 fix door member:craig member:andy label:Design label:Bug
```

The `member:` and `label:` tokens are styled in a distinct color to separate them from the free-text search.

### Keybindings

| Key | Context | Action |
|-----|---------|--------|
| `/` | Board (unfocused, no modal open) | Focus the search bar |
| `Enter` | Search bar (focused) | Confirm text, defocus, apply filter |
| `Escape` | Search bar (focused) | Clear all filters, defocus |
| `Escape` | Board (unfocused, no modal open) | Clear all filters |
| `ctrl+m` | Board (unfocused, no modal open) | Open member multiselect modal |
| `ctrl+l` | Board (unfocused, no modal open) | Open label multiselect modal |

### Text Parsing

The search bar text is parsed to extract structured filter tokens:

- `member:<value>` — extracted as a member filter (case-insensitive match against username or full name). Multi-word values use quotes: `member:"Craig Smith"`
- `label:<value>` — extracted as a label filter (case-insensitive match against label name). Multi-word values use quotes: `label:"In Progress"`
- Everything else — treated as free-text, case-insensitive substring match against card names

Typing `member:craig label:Bug fix door` is equivalent to selecting Craig via the member modal, Bug via the label modal, and typing "fix door".

When modals add/remove selections, the corresponding tokens are added/removed from the end of the text input. The cursor position is preserved where possible. When the user types tokens manually, the parser extracts them. The text input is always the single source of truth.

## Member & Label Modals

Both modals are identical in behavior, differing only in data source.

### Visual

Rendered as a centered overlay (following the existing command palette pattern). Shows a list with checkboxes.

- **Member modal**: Lists all board members by full name
- **Label modal**: Lists all board labels by name with their colored indicator (`⏺`). Labels without a name are listed by their color (e.g., "green", "red")

Items currently in the filter are pre-checked.

### Interaction

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate list |
| `Space` | Toggle selection |
| `Enter` / `Escape` | Close modal, apply selections |

There is no cancel action. Toggling is immediate. Both `Enter` and `Escape` close the modal and update the search bar tokens to reflect the current selections.

## Filtering Logic

### Filter Struct

```go
type Filter struct {
    Text    string   // substring match against card names
    Members []string // member usernames or full names to match
    Labels  []string // label names to match
}
```

### Matching Rules

1. **Members** (if set): Card must have at least one matching member (OR within members). Matching is case-insensitive against both username and full name.
2. **Labels** (if set): Card must have at least one matching label (OR within labels). Matching is case-insensitive against label name.
3. **Text** (if set): Card name must contain the text as a case-insensitive substring.
4. All three filter types are ANDed together.

A card is visible only if it matches all active filter types.

### Application

Filtering is live — the filter is parsed and applied on every keystroke while the search bar is focused. There is no separate "confirm" step for filtering; `Enter` simply defocuses the bar.

The filtered card list replaces the column's `list.SetItems()`. Navigation (j/k) operates on the filtered set. Card move operations work on the underlying full card data; after a move, the filter is re-applied.

When a filter is applied and the currently selected card is no longer visible, selection moves to the first visible card in the column. If the column has no visible cards, the column shows "No matching cards" in dimmed text.

The filter is re-applied whenever:
- Search text changes (on each keystroke while focused)
- Modal selections change (on modal close)
- Board data is refreshed
- A card is moved

Filter state persists across board refreshes. If a member or label in the filter no longer exists after a refresh, the stale token remains in the search bar (allowing the user to see and clear it) but matches no cards.

### Detail Panel Interaction

When filtering causes the selected card to change, the detail panel updates to show the newly selected card. If a column has no visible cards and the detail panel was showing a card from that column, the detail panel shows "No card selected".

## Layout Impact

The search bar adds 3 lines above the breadcrumb. Column height adjusts from `b.height - 3` to `b.height - 6`. The search bar is always rendered (no layout shifts when activating/deactivating filters).

## Components

| Component | Location | Responsibility |
|-----------|----------|---------------|
| Search bar (`textinput.Model`) | `App` | Text input, display filter state |
| Filter parser | New utility | Parse text into `Filter` struct |
| Member modal | `App` | Multiselect overlay for members |
| Label modal | `App` | Multiselect overlay for labels |
| `Filter` struct + matching | `BoardModel` | Apply filter to card visibility |
