# Views Feature Design

## Goal

Add a "Views" system to tuillo that provides named filter presets with overridable GUI settings, presented as a tab bar across the top of the screen. Inspired by gh-dash's `prSections`.

## Architecture

Views are named presets defined in config, each with a fixed base filter and optional GUI overrides. A tab bar always renders at the top of the screen showing all available views. The active view determines the base filter and GUI settings; users can append additional filters on top, and Escape resets to the view's base state. Switching views is a hard reset — any user-appended filters are discarded.

A special `@me` token resolves to the authenticated Trello user at filter evaluation time, enabling a default "My Cards" view without requiring the user to know their username.

## Config Structure

### View Definition

```yaml
views:
  - title: My Cards
    filter: "member:@me"
    key: m
    showDetailPanel: true
  - title: Mobile Cards
    filter: "label:apple label:android"
  - title: All Cards
    showDetailPanel: false
```

**Fields:**
- `title` (required) — display name in the tab bar. Views without a title are silently skipped.
- `filter` (optional) — base filter string using existing filter syntax (`member:`, `label:`, free text). Supports `@me` token.
- `key` (optional) — custom shortcut key for direct jump. Must be a single character. Views without a custom key are auto-assigned incrementing numbers starting at 1. If two views define the same custom key, the first keeps it and the second is auto-assigned a number.
- GUI override keys (optional) — e.g., `showDetailPanel`, `columnWidth`, `showCardLabels`. Overrides the global value when this view is active. Adding new GUI settings to views requires adding a pointer field to `ViewConfig` and updating the override-merge logic.

`Views` is a top-level field on the `Config` struct (alongside `GUI`, `Board`, `Keybinding`, `CustomCommands`).

### View Keybindings

```yaml
keybinding:
  views:
    nextView: v
    prevView: V
```

### Default Views

When no views are defined in config, two defaults are provided:

```yaml
views:
  - title: My Cards
    filter: "member:@me"
    key: m
  - title: All Cards
```

This ensures the tab bar always renders and the feature is discoverable.

## Tab Bar Rendering

Full-width bar at the very top of the screen, styled to match gh-dash's section tabs:

```
 All Cards ‹1›  │  My Cards ‹m›  │  Mobile Cards ‹2›
```

- **Active view:** Bold, bright text (ANSIColor 15), bordered/boxed
- **Inactive views:** Dimmed text (ANSIColor 8)
- **Separators:** `│` pipe character in dimmed color between tabs
- **Shortcuts:** Rendered as `‹key›` after the title, slightly dimmer than the title text to distinguish from column card counts `(n)` which use parentheses
- Active view's shortcut inherits the brighter active styling
- Tab bar occupies one row of vertical space

## View Switching

### Keyboard Controls

- **Direct jump:** Press the view's shortcut key (`1`, `m`, etc.) from default board mode — only processed when no modal is open (member/label select, command palette, help overlay), search bar is not focused, and no prompt flow is active
- **Cycling:** `v` cycles forward, `V` cycles backward (wraps around at ends)
- All keys are configurable via `keybinding.views` config
- Direct-jump view keys are checked after all standard keybindings (quit, help, detail toggle, etc.), so they cannot shadow built-in keys. Users should still avoid using standard keys as view shortcuts for clarity.

### Behavior on Switch

1. Any user-appended filters are discarded
2. Search bar text is replaced with the view's base `filter` string (or cleared if no filter)
3. Board filters are re-applied from the base filter only
4. GUI overrides from the view config take effect (e.g., detail panel toggles on/off)

### Filter Interaction

- **Appending filters:** User focuses search bar and types additional terms after the base filter text. Filters stack (base + user-added).
- **Escape key (search focused):** Resets search bar to the active view's base filter (not fully cleared, unless the view has no base filter). Any user-appended terms are removed. Escape behavior when the search bar is NOT focused remains unchanged.
- **Switching views:** Hard reset to the new view's base filter. User-appended filters from the previous view are lost.

## @me Token Resolution

### Trello Client

New method on the Trello client:

```go
func (c *Client) FetchCurrentUser() (Member, error)
```

Calls `/1/members/me` and returns a `Member` struct. This endpoint is already used by `ValidateCredentials`.

### App Integration

- `FetchCurrentUser()` is called once at app startup
- The current user's `Member` is stored on the `App` struct
- If the call fails (network error, invalid credentials), the app continues without `@me` support — `member:@me` is treated as a literal username "@me" which will match no cards. The default "My Cards" view will simply show an empty board. No startup failure.
- When `ParseFilter` encounters `member:@me` and a current user is available, it resolves to the current user's username before matching cards
- Works in both view configs and manual search bar input

### ParseFilter Signature

The existing `ParseFilter(input string) Filter` signature changes to accept the current username for `@me` resolution:

```go
func ParseFilter(input string, currentUser string) Filter
```

When `currentUser` is empty (user not resolved), `@me` tokens are left as literal text. All existing call sites in `app.go` are updated to pass the stored current username.

## View Config Type

```go
type ViewConfig struct {
    Title           string `mapstructure:"title"`
    Filter          string `mapstructure:"filter"`
    Key             string `mapstructure:"key"`
    ShowDetailPanel *bool  `mapstructure:"showDetailPanel"`
    ColumnWidth     *int   `mapstructure:"columnWidth"`
    ShowCardLabels  *bool  `mapstructure:"showCardLabels"`
}
```

GUI override fields use pointers so that unset values (nil) are distinguishable from explicit false/0. Only non-nil overrides replace the global config value.

### View Keybinding Config

```go
type ViewKeys struct {
    NextView string `mapstructure:"nextView"`
    PrevView string `mapstructure:"prevView"`
}
```

Added to `KeybindingConfig` as `Views ViewKeys`.

## Shortcut Key Assignment

For views without a custom `key`, auto-assign incrementing numbers starting at 1, skipping any numbers already used as custom keys:

```go
func assignViewKeys(views []ViewConfig) []string {
    used := map[string]bool{}
    keys := make([]string, len(views))
    for i, v := range views {
        if v.Key != "" {
            keys[i] = v.Key
            used[v.Key] = true
        }
    }
    next := 1
    for i, v := range views {
        if v.Key == "" {
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

## Components Modified

- **`internal/config/config.go`** — Add `ViewConfig`, `ViewKeys` types; add `Views []ViewConfig` to `Config`; add `Views ViewKeys` to `KeybindingConfig`; provide defaults
- **`internal/trello/client.go`** — Add `FetchCurrentUser() (Member, error)` method
- **`internal/tui/filter.go`** — Update `ParseFilter` / `MatchesCard` to handle `@me` resolution (accept current username parameter)
- **`internal/tui/app.go`** — Add view tab bar state (`activeView`, `views`, `currentUser`); handle view switching keys; render tab bar; apply view GUI overrides on switch; modify Escape behavior to reset to view base filter
- **`internal/tui/keys.go`** — Add `ViewNext`, `ViewPrev` bindings plus per-view dynamic bindings

## Startup and Session Behavior

- The active view on startup is always the first view in the list (index 0)
- Active view does not persist across sessions — always resets to the first view on launch
- Tab bar truncates long view titles with ellipsis if they would overflow the terminal width

## Help Overlay

View keys (both `nextView`/`prevView` and per-view direct jump keys) are displayed in the help overlay under a "Views" section.

## Testing

- `filter_test.go` — Add tests for `@me` token resolution
- `config_test.go` — Test default views, custom key assignment, GUI override merging, duplicate key rejection, missing title handling
- View switching integration tested manually via the TUI
