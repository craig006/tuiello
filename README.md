# tuiello

A TUI client for Trello.

<!-- TODO: Add screenshot -->

## Features

- Kanban board view with sliding column window
- Detail panel with comments and checklists
- Views — named filter presets with keyboard shortcuts
- Search and filter by member, label, or text
- Custom commands with templated shell execution
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

tuiello looks for configuration in two places (merged in order):

1. `~/.config/tuiello/config.yml` — global config
2. `.tuiello.yml` — project-local config (in current directory)

Key config sections:

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

| Key | Action |
|-----|--------|
| `h` / `l` | Move between columns |
| `j` / `k` | Move between cards |
| `H` / `L` | Move card to adjacent column |
| `J` / `K` | Move card up/down in column |
| `d` | Toggle detail panel |
| `[` / `]` | Detail panel: previous/next tab |
| `Ctrl+j` / `Ctrl+k` | Detail panel: scroll down/up |
| `/` | Focus search bar |
| `Ctrl+m` | Filter by member (multi-select) |
| `Ctrl+l` | Filter by label (multi-select) |
| `v` / `V` | Next/previous view |
| `x` | Custom commands |
| `r` | Refresh board |
| `q` | Quit |

## License

MIT
