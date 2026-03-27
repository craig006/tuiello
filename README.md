# tuiello

A TUI client for Trello.

<!-- TODO: Add screenshot -->

## Features

- Kanban board view with sliding column window
- Detail panel with comments and checklists
- **Interactive Comments** — Create, edit, and delete comments on cards with @mention autocomplete
- Views — named filter presets with keyboard shortcuts
- Search and filter by member, label, or text
- Custom commands with templated shell execution
- **Focus Management** — Press Enter to focus detail panel, Esc to return to board
- Vim-style navigation
- Configurable keybindings and theming

## Installation

### Homebrew

```bash
brew install craig006/tuiello/tuiello
```

### AUR (Arch Linux)

```bash
yay -S tuiello
```

### From Source

```bash
go install github.com/craig006/tuiello@latest
```

## Quick Start

1. Get your Trello API key at https://trello.com/power-ups/admin
2. Generate a token from the API key page
3. Set environment variables:

```bash
export TRELLO_API_KEY=<your-api-key>
export TRELLO_TOKEN=<your-token>
```

4. Launch:

```bash
tuiello --board "Board Name"
```

## Configuration

tuiello uses a two-level config system. Each level has a settings file and an optional auth file:

```
~/.config/tuiello/          # global
├── config.yml              # settings
└── auth.yml                # credentials

<project>/.tuiello/         # project-local (overrides global)
├── config.yml              # project settings
└── auth.yml                # project credentials
```

All files are optional. Values merge in order: global config → global auth → project config → project auth. Environment variables and CLI flags override everything.

### Credentials

Set your Trello credentials in `auth.yml`:

```yaml
auth:
  apiKey: your-api-key
  token: your-token
```

Or use environment variables: `TRELLO_API_KEY` and `TRELLO_TOKEN`.

### Board

Set a default board so you can launch with just `tuiello`:

```yaml
board:
  name: "My Board"
```

Or by ID:

```yaml
board:
  id: "abc123"
```

CLI flags `--board` and `--board-id` override config values.

### Settings

```yaml
gui:
  columnWidth: 30
  showCardLabels: true
  showDetailPanel: true
  padding: 1
  theme:
    activeBorderColor: ["4", "bold"]
    inactiveBorderColor: ["8"]

views:
  - title: "My Cards"
    filter: "member:@me"
    key: "m"
  - title: "All Cards"

keybinding:
  universal:
    quit: "q"
    help: "?"
    refresh: "r"
```

## Keybindings

| Key | Action | Context |
|-----|--------|---------|
| `h` / `l` | Move between columns | Board |
| `j` / `k` | Move between cards | Board |
| `H` / `L` | Move card to adjacent column | Board with card selected |
| `J` / `K` | Move card up/down in column | Board with card selected |
| `enter` | Focus detail panel | Board with card selected |
| `esc` | Focus board | Detail panel active |
| `d` | Toggle detail panel | Board |
| `[` / `]` | Detail panel: previous/next tab | Detail panel active |
| `Ctrl+j` / `Ctrl+k` | Detail panel: scroll down/up | Detail panel active |
| `j` / `k` | Navigate comments | Detail (Comments tab active) |
| `c` | Create comment | Detail (Comments tab active, View mode) |
| `e` | Edit comment | Detail (Comments tab active, View mode) |
| `d` | Delete comment | Detail (Comments tab active, View mode) |
| `@` | Mention user (in comment) | Creating/editing comment |
| `Tab` / `Enter` | Select mention | Autocomplete popup active |
| `/` | Focus search bar | Board |
| `Ctrl+m` | Filter by member (multi-select) | Board |
| `Ctrl+l` | Filter by label (multi-select) | Board |
| `v` / `V` | Next/previous view | Board |
| `x` | Custom commands | Board |
| `r` | Refresh board | Board |
| `q` | Quit | Universal |

## License

MIT
