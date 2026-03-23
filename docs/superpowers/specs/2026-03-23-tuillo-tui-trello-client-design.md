# tuillo — TUI Trello Board Client

**Date:** 2026-03-23
**Status:** Draft
**Working title:** tuillo (subject to change)

## Overview

A terminal-based Trello board viewer and manager, built in Go using Bubble Tea. Follows the configuration conventions of lazygit and lazydocker — YAML config, platform-aware paths, cascading config, Go-template custom commands.

The MVP focuses on three capabilities: viewing a board in kanban layout, moving cards between and within columns, and executing user-defined custom commands.

## Tech Stack

- **Language:** Go
- **TUI framework:** Bubble Tea (charmbracelet/bubbletea)
- **Components:** Bubbles (charmbracelet/bubbles) — list, viewport, help, key
- **Styling:** Lip Gloss (charmbracelet/lipgloss)
- **Trello client:** adlio/trello
- **Config:** YAML via koanf or viper
- **CLI:** Cobra

## Project Structure

```
tuillo/
├── main.go                 # Entry point, CLI flag parsing
├── cmd/
│   └── root.go             # Cobra root command, config loading
├── internal/
│   ├── config/
│   │   └── config.go       # Config structs, loading, cascade logic
│   ├── trello/
│   │   └── client.go       # Trello API wrapper (using adlio/trello)
│   ├── tui/
│   │   ├── app.go          # Root Bubble Tea model
│   │   ├── board.go        # Board model (columns layout)
│   │   ├── column.go       # Column model (wraps bubbles/list)
│   │   └── keys.go         # Keybinding definitions
│   └── commands/
│       └── custom.go       # Custom command execution engine
├── go.mod
└── go.sum
```

## Configuration

### Cascade Order (lowest to highest priority)

1. Built-in defaults
2. `~/.config/tuillo/config.yml` (global)
3. `.tuillo.yml` (project-local, in current directory)
4. CLI flags (`--board`, `--board-id`)

### Config Structure

```yaml
gui:
  theme:
    activeBorderColor:
      - green
      - bold
    inactiveBorderColor:
      - "240"
    selectedCardColor:
      - cyan
    columnTitleColor:
      - magenta
      - bold
  columnWidth: 30
  showCardLabels: true

board:
  id: ""
  name: ""

keybinding:
  universal:
    quit: "q"
    help: "?"
    refresh: "r"
  board:
    moveLeft: "h"
    moveRight: "l"
    moveUp: "k"
    moveDown: "j"
    moveCardLeft: "H"
    moveCardRight: "L"
    moveCardUp: "K"
    moveCardDown: "J"
    enter: "enter"
    customCommand: "x"

customCommands: []
```

### Theme Defaults

The default theme uses ANSI named colors only (green, cyan, magenta, etc.) so the app inherits the user's terminal color scheme automatically. Lip Gloss supports ANSI 4-bit, 8-bit, and true color — users can override with hex values in their config if desired.

### Authentication

Trello credentials are provided via environment variables:

- `TRELLO_API_KEY` — API key from Trello Power-Ups admin
- `TRELLO_TOKEN` — User token with read/write scope

If missing on startup, the app prints a clear error message with instructions on how to obtain credentials. No credentials are stored in config files.

### Board Resolution

The `board` config section accepts either `id` (direct Trello board ID) or `name` (human-readable name, resolved to ID on startup via the API). CLI flags `--board` (name) and `--board-id` (ID) override config values. If resolving by name matches multiple boards, the app errors with a message listing the matches and asking the user to use `--board-id` instead.

## TUI Architecture

### Root Model (`app.go`)

The top-level Bubble Tea model manages:

- The board model (main view)
- Current focus state (which column, which card)
- A status bar (errors, success messages, loading state)
- The help overlay (toggled with `?`)

### Board Model (`board.go`)

Manages the horizontal layout of columns:

- **3-column sliding window** — only the selected column and its immediate neighbors are visible at any time
- Calculates column widths based on terminal size (equal 3-way split, respecting `gui.columnWidth` minimum)
- Tracks which column is focused
- Handles left/right navigation between columns (shifts the window when focus moves beyond visible range)
- Handles card move commands (H/L between columns, K/J within column)
- Renders columns side-by-side using Lip Gloss `JoinHorizontal`

**Sliding window edge cases:**

- First column selected: show columns 1, 2, 3 (selected is left-aligned)
- Last column selected: show last-2, last-1, last (selected is right-aligned)
- Board has 2 columns: show both, extra space distributed
- Board has 1 column: single column, centered
- Visual indicators (`[2/5]` or similar) show position within the full board

### Column Model (`column.go`)

Wraps `bubbles/list` for each Trello list:

- Displays list name as header with card count
- Shows cards as list items with label color indicators
- Handles up/down navigation within the column
- Delegates to `bubbles/list` for scrolling and viewport management

### Data Flow

```
User Input → Root Model
  → Board Model (navigation, move commands)
    → Trello Client (API call as Bubble Tea Cmd)
      → Response Message → Update Board State → Re-render
```

All Trello API calls are async — they return `tea.Cmd` functions that produce messages. The UI remains responsive during network calls with a loading indicator in the status bar.

## Keybindings & Navigation

| Key | Action | Context |
|-----|--------|---------|
| `h` / `←` | Focus column left | Board |
| `l` / `→` | Focus column right | Board |
| `j` / `↓` | Focus card down | Board |
| `k` / `↑` | Focus card up | Board |
| `H` / `shift+←` | Move card to column left | Board |
| `L` / `shift+→` | Move card to column right | Board |
| `K` / `shift+↑` | Move card up in column | Board |
| `J` / `shift+↓` | Move card down in column | Board |
| `x` | Open command palette | Board |
| `r` | Refresh board | Board |
| `?` | Toggle help overlay | Universal |
| `q` | Quit | Universal |

All keybindings are configurable via the `keybinding` config section. Vim-style defaults with arrow key alternatives built in.

**Help overlay:** `?` shows a full-screen overlay listing all keybindings (including custom commands) grouped by context. Dismiss with `?` or `Escape`.

**Command palette:** `x` opens a filterable list (using `bubbles/list`) of all custom commands available for the current context. Type to filter, Enter to execute.

## Custom Commands Engine

### Config Structure

```yaml
customCommands:
  - key: "g"
    description: "Open card in browser"
    command: "open {{.Card.URL}}"
    context: "card"
    output: "none"
  - key: "b"
    description: "Create branch from card"
    command: "git checkout -b {{.Card.Name | kebab}}"
    context: "card"
    output: "terminal"
    prompts:
      - type: "confirm"
        title: "Create branch '{{.Card.Name | kebab}}'?"
  - key: "n"
    description: "Add note to card"
    command: "echo {{.Prompt.Note}} >> notes/{{.Card.ID}}.md"
    context: "card"
    prompts:
      - type: "input"
        title: "Note:"
        key: "Note"
```

### Template Data

- `{{.Card.ID}}`, `{{.Card.Name}}`, `{{.Card.URL}}`, `{{.Card.Description}}`
- `{{.Card.Labels}}` — comma-separated label names
- `{{.Card.Members}}` — comma-separated usernames
- `{{.List.ID}}`, `{{.List.Name}}`
- `{{.Board.ID}}`, `{{.Board.Name}}`
- `{{.Prompt.<Key>}}` — values from interactive prompts

### Template Functions

`kebab`, `snake`, `camel`, `lower`, `upper`, `trim`, `replace`

### Output Modes

- `none` — run silently, show success/error in status bar
- `terminal` — suspend TUI, run command in full terminal, resume on exit
- `popup` — show command output in a modal overlay

### Prompt Types

- `confirm` — yes/no
- `input` — free text
- `menu` — pick from a static list of options defined via `options`:
  ```yaml
  prompts:
    - type: "menu"
      title: "Select priority:"
      key: "Priority"
      options:
        - name: "High"
          value: "high"
        - name: "Medium"
          value: "medium"
        - name: "Low"
          value: "low"
  ```

## Trello API Integration

### Client Wrapper (`internal/trello/client.go`)

Thin wrapper around `adlio/trello`:

- Initializes from env vars
- Validates credentials on startup with `GET /members/me`
- Exposes methods that return Bubble Tea messages

### Core Operations (MVP)

```go
FetchBoard(id string) → BoardMsg
MoveCardToList(cardID, listID string) → CardMovedMsg
ReorderCard(cardID string, pos float64) → CardReorderedMsg
ResolveBoard(name string) → BoardResolvedMsg
```

**Card position calculation:** When moving a card up/down within a list, the new `pos` value is calculated as the midpoint between the two neighboring cards' positions (Trello uses floating-point positioning). When moving to the top, use half the first card's position. When moving to the bottom, use the last card's position plus a fixed increment (e.g., 65536).

### Board Fetching

Single request: `GET /boards/{id}` with `lists=open&cards=open&card_fields=name,desc,labels,idMembers,url,pos&list_fields=name,pos`. Gets everything needed for the board view in one round trip.

Refresh strategy: full board fetch on startup, targeted refreshes after mutations (e.g., re-fetch affected lists after a card move). Manual full refresh with `r`.

### Optimistic Updates

When the user moves a card, the UI updates immediately. The API call fires async. If it fails, the move is reverted and the error shown in the status bar.

### Error Handling

- **401** — clear error about invalid/expired credentials
- **429** — respect `Retry-After` header, show "Rate limited, retrying..." in status bar
- **Network errors** — show in status bar, board remains visible with stale data

### Rate Limits

Trello enforces 100 requests per 10 seconds per token. The app's usage pattern (one fetch on startup, occasional moves) stays well within limits.

## Deferred Features (Post-MVP)

- Card detail side panel (description, labels, members, comments, due date)
- Assign labels and members to cards
- Filter/search cards by label, member, or text
- Board creation and list management
- Multiple board support / board switcher
- Notification/activity feed
