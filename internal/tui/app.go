// internal/tui/app.go
package tui

import (
	"fmt"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"

	"github.com/craig006/tuillo/internal/commands"
	"github.com/craig006/tuillo/internal/config"
	"github.com/craig006/tuillo/internal/trello"
)

// Messages
type BoardFetchedMsg struct {
	Board *trello.Board
}

type BoardFetchErrMsg struct {
	Err error
}

type CardMovedMsg struct {
	CardID string
}

type CardMoveErrMsg struct {
	Err    error
	// For rollback
	Card   trello.Card
	FromCol int
	FromIdx int
	ToCol   int
}

type BoardResolvedMsg struct {
	ID string
}

type StatusMsg struct {
	Text string
}

// commandItem wraps a custom command for display in the command palette.
type commandItem struct {
	cmd config.CustomCommandConfig
}

func (c commandItem) Title() string       { return c.cmd.Description }
func (c commandItem) Description() string { return c.cmd.Key }
func (c commandItem) FilterValue() string { return c.cmd.Description }

// App is the root Bubble Tea model.
type App struct {
	board      BoardModel
	client     *trello.Client
	config     config.Config
	keyMap     KeyMap
	help       help.Model
	status     string
	loading    bool
	showHelp   bool
	width      int
	height     int
	boardReady bool
	detail     DetailModel

	// Command palette
	commandPalette list.Model
	showPalette    bool
	// Prompt flow
	pendingCommand *config.CustomCommandConfig
	pendingCtx     commands.TemplateContext
	promptIdx      int
	promptInput    textinput.Model
	showPrompt     bool
	promptType     string // "confirm", "input", "menu"
}

func NewApp(client *trello.Client, cfg config.Config) App {
	km := NewKeyMap(cfg.Keybinding)
	palette := list.New(nil, list.NewDefaultDelegate(), 40, 20)
	palette.Title = "Commands"
	return App{
		client:         client,
		config:         cfg,
		keyMap:         km,
		help:           help.New(),
		status:         "Loading board...",
		loading:        true,
		commandPalette: palette,
		detail:         NewDetailModel(km, NewTheme(cfg.GUI.Theme)),
	}
}

func (a App) Init() tea.Cmd {
	boardID := a.config.Board.ID
	if boardID == "" && a.config.Board.Name != "" {
		return a.resolveBoardCmd(a.config.Board.Name)
	}
	if boardID == "" {
		return func() tea.Msg {
			return BoardFetchErrMsg{Err: fmt.Errorf("no board configured — use --board or --board-id, or set board.id in config")}
		}
	}
	return a.fetchBoardCmd(boardID)
}

func (a App) resolveBoardCmd(name string) tea.Cmd {
	return func() tea.Msg {
		id, err := a.client.ResolveBoard(name)
		if err != nil {
			return BoardFetchErrMsg{Err: err}
		}
		return BoardResolvedMsg{ID: id}
	}
}

func (a App) fetchBoardCmd(boardID string) tea.Cmd {
	return func() tea.Msg {
		board, err := a.client.FetchBoard(boardID)
		if err != nil {
			return BoardFetchErrMsg{Err: err}
		}
		return BoardFetchedMsg{Board: board}
	}
}

type moveRollback struct {
	Card    trello.Card
	FromCol int
	FromIdx int
	ToCol   int
}

func (a App) moveCardToListCmd(cardID, listID, pos string, rb moveRollback) tea.Cmd {
	return func() tea.Msg {
		err := a.client.MoveCardToList(cardID, listID, pos)
		if err != nil {
			return CardMoveErrMsg{Err: err, Card: rb.Card, FromCol: rb.FromCol, FromIdx: rb.FromIdx, ToCol: rb.ToCol}
		}
		return CardMovedMsg{CardID: cardID}
	}
}

func (a App) reorderCardCmd(cardID string, pos float64) tea.Cmd {
	return func() tea.Msg {
		err := a.client.ReorderCard(cardID, pos)
		if err != nil {
			return CardMoveErrMsg{Err: err}
		}
		return CardMovedMsg{CardID: cardID}
	}
}

func (a *App) updateDetailLayout() {
	boardWidth := a.width * 60 / 100
	panelWidth := a.width - boardWidth
	a.board.width = boardWidth
	a.board.height = a.height - 4
	a.board.ResizeColumns()
	a.detail.SetSize(panelWidth, a.height-2)
}

func (a App) fetchDetailData() tea.Cmd {
	cardID := a.detail.cardID
	switch a.detail.tab {
	case tabComments:
		return func() tea.Msg {
			comments, err := a.client.FetchCardComments(cardID)
			if err != nil {
				return CardCommentsFetchErrMsg{CardID: cardID, Err: err}
			}
			return CardCommentsMsg{CardID: cardID, Comments: comments}
		}
	case tabChecklists:
		return func() tea.Msg {
			checklists, err := a.client.FetchCardChecklists(cardID)
			if err != nil {
				return CardChecklistsFetchErrMsg{CardID: cardID, Err: err}
			}
			return CardChecklistsMsg{CardID: cardID, Checklists: checklists}
		}
	}
	return nil
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		if a.boardReady {
			if a.detail.open {
				a.updateDetailLayout()
			} else {
				a.board.width = msg.Width
				a.board.height = msg.Height - 4
				a.board.ResizeColumns()
			}
		}
		return a, nil

	case BoardFetchedMsg:
		a.loading = false
		a.boardReady = true
		a.board = NewBoardModel(msg.Board, a.config, a.width, a.height-4)
		a.status = fmt.Sprintf("%s — %s", msg.Board.Name, a.board.PositionIndicator())
		a.detail.open = false
		a.detail.cardID = ""
		return a, nil

	case BoardFetchErrMsg:
		a.loading = false
		a.status = fmt.Sprintf("Error: %v", msg.Err)
		return a, nil

	case BoardResolvedMsg:
		a.config.Board.ID = msg.ID
		return a, a.fetchBoardCmd(msg.ID)

	case StatusMsg:
		a.status = msg.Text
		return a, nil

	case CardMovedMsg:
		a.status = "Card moved"
		return a, nil

	case CardMoveErrMsg:
		a.status = fmt.Sprintf("Move failed: %v", msg.Err)
		// Rollback: remove card from destination, re-insert at source
		if msg.ToCol >= 0 && msg.ToCol < len(a.board.columns) && msg.FromCol >= 0 && msg.FromCol < len(a.board.columns) {
			destCards := a.board.columns[msg.ToCol].cards
			for i, c := range destCards {
				if c.ID == msg.Card.ID {
					a.board.RemoveCard(msg.ToCol, i)
					break
				}
			}
			a.board.InsertCard(msg.FromCol, msg.Card, msg.FromIdx)
		}
		return a, nil

	case CardCommentsMsg:
		a.detail.HandleCommentsMsg(msg)
		return a, nil

	case CardCommentsFetchErrMsg:
		a.detail.HandleCommentsFetchErr(msg)
		return a, nil

	case CardChecklistsMsg:
		a.detail.HandleChecklistsMsg(msg)
		return a, nil

	case CardChecklistsFetchErrMsg:
		a.detail.HandleChecklistsFetchErr(msg)
		return a, nil

	case tea.KeyPressMsg:
		// Handle active prompts
		if a.showPrompt && a.pendingCommand != nil {
			prompt := a.pendingCommand.Prompts[a.promptIdx]
			switch a.promptType {
			case "confirm":
				if msg.String() == "y" {
					a.promptIdx++
					a.showPrompt = false
					return a.showNextPrompt()
				}
				if msg.String() == "n" || msg.String() == "esc" {
					a.pendingCommand = nil
					a.showPrompt = false
					a.status = "Cancelled"
					return a, nil
				}
			case "input":
				if msg.String() == "enter" {
					a.pendingCtx.Prompt[prompt.Key] = a.promptInput.Value()
					a.promptIdx++
					a.showPrompt = false
					return a.showNextPrompt()
				}
				if msg.String() == "esc" {
					a.pendingCommand = nil
					a.showPrompt = false
					a.status = "Cancelled"
					return a, nil
				}
				var cmd tea.Cmd
				a.promptInput, cmd = a.promptInput.Update(msg)
				return a, cmd
			case "menu":
				if msg.String() == "esc" {
					a.pendingCommand = nil
					a.showPrompt = false
					a.showPalette = false
					a.status = "Cancelled"
					return a, nil
				}
				if msg.String() == "enter" {
					if item, ok := a.commandPalette.SelectedItem().(commandItem); ok {
						a.pendingCtx.Prompt[prompt.Key] = item.cmd.Key
						a.promptIdx++
						a.showPrompt = false
						a.showPalette = false
						return a.showNextPrompt()
					}
				}
				var cmd tea.Cmd
				a.commandPalette, cmd = a.commandPalette.Update(msg)
				return a, cmd
			}
		}

		// Handle command palette
		if a.showPalette {
			if msg.String() == "esc" {
				a.showPalette = false
				return a, nil
			}
			if msg.String() == "enter" {
				if item, ok := a.commandPalette.SelectedItem().(commandItem); ok {
					a.showPalette = false
					return a.executeCustomCommand(item.cmd)
				}
			}
			var cmd tea.Cmd
			a.commandPalette, cmd = a.commandPalette.Update(msg)
			return a, cmd
		}

		switch {
		case matchKey(msg, a.keyMap.Quit):
			return a, tea.Quit

		case matchKey(msg, a.keyMap.Help):
			a.showHelp = !a.showHelp
			return a, nil

		case matchKey(msg, a.keyMap.Refresh):
			if a.config.Board.ID != "" {
				a.loading = true
				a.status = "Refreshing..."
				return a, a.fetchBoardCmd(a.config.Board.ID)
			}
			return a, nil

		case matchKey(msg, a.keyMap.CustomCommand):
			if a.boardReady && !a.showPalette {
				filtered := commands.FilterByContext(a.config.CustomCommands, "card")
				items := make([]list.Item, len(filtered))
				for i, cmd := range filtered {
					items[i] = commandItem{cmd: cmd}
				}
				a.commandPalette.SetItems(items)
				a.commandPalette.SetFilteringEnabled(true)
				a.showPalette = true
				return a, nil
			}

		case matchKey(msg, a.keyMap.MoveLeft):
			if a.boardReady {
				a.board.FocusLeft()
				a.status = fmt.Sprintf("%s — %s", a.board.board.Name, a.board.PositionIndicator())
				if a.detail.open {
					if card, _, ok := a.board.SelectedCard(); ok && card.ID != a.detail.cardID {
						a.detail.SetCard(card)
						if a.detail.NeedsFetch() {
							a.detail.MarkLoading()
							return a, a.fetchDetailData()
						}
					}
				}
			}
			return a, nil

		case matchKey(msg, a.keyMap.MoveRight):
			if a.boardReady {
				a.board.FocusRight()
				a.status = fmt.Sprintf("%s — %s", a.board.board.Name, a.board.PositionIndicator())
				if a.detail.open {
					if card, _, ok := a.board.SelectedCard(); ok && card.ID != a.detail.cardID {
						a.detail.SetCard(card)
						if a.detail.NeedsFetch() {
							a.detail.MarkLoading()
							return a, a.fetchDetailData()
						}
					}
				}
			}
			return a, nil

		case matchKey(msg, a.keyMap.MoveCardLeft):
			return a.handleMoveCardLeft()

		case matchKey(msg, a.keyMap.MoveCardRight):
			return a.handleMoveCardRight()

		case matchKey(msg, a.keyMap.MoveCardUp):
			return a.handleMoveCardUp()

		case matchKey(msg, a.keyMap.MoveCardDown):
			return a.handleMoveCardDown()

		case matchKey(msg, a.keyMap.DetailToggle):
			if a.boardReady {
				if !a.detail.open && a.width < 80 {
					a.status = "Terminal too narrow for detail panel"
					return a, nil
				}
				a.detail.Toggle()
				if a.detail.open {
					if card, _, ok := a.board.SelectedCard(); ok {
						a.detail.SetCard(card)
						a.updateDetailLayout()
						if a.detail.NeedsFetch() {
							a.detail.MarkLoading()
							return a, a.fetchDetailData()
						}
						return a, nil
					}
					a.updateDetailLayout()
				} else {
					a.board.width = a.width
					a.board.height = a.height - 4
					a.board.ResizeColumns()
				}
				return a, nil
			}

		case matchKey(msg, a.keyMap.DetailTabNext):
			if a.boardReady && a.detail.open {
				a.detail.NextTab()
				if a.detail.NeedsFetch() {
					a.detail.MarkLoading()
					return a, a.fetchDetailData()
				}
				return a, nil
			}

		case matchKey(msg, a.keyMap.DetailTabPrev):
			if a.boardReady && a.detail.open {
				a.detail.PrevTab()
				if a.detail.NeedsFetch() {
					a.detail.MarkLoading()
					return a, a.fetchDetailData()
				}
				return a, nil
			}

		case matchKey(msg, a.keyMap.DetailScrollDown):
			if a.boardReady && a.detail.open {
				a.detail.ScrollDown()
				return a, nil
			}

		case matchKey(msg, a.keyMap.DetailScrollUp):
			if a.boardReady && a.detail.open {
				a.detail.ScrollUp()
				return a, nil
			}
		}

		// Pass to board for card navigation (up/down via bubbles/list)
		if a.boardReady {
			var cmd tea.Cmd
			a.board, cmd = a.board.Update(msg)
			if a.detail.open {
				if card, _, ok := a.board.SelectedCard(); ok && card.ID != a.detail.cardID {
					a.detail.SetCard(card)
					if a.detail.NeedsFetch() {
						a.detail.MarkLoading()
						return a, tea.Batch(cmd, a.fetchDetailData())
					}
				}
			}
			return a, cmd
		}
	}

	return a, nil
}

func (a App) handleMoveCardLeft() (tea.Model, tea.Cmd) {
	if !a.boardReady || a.board.focused == 0 {
		return a, nil
	}

	card, colIdx, ok := a.board.SelectedCard()
	if !ok {
		return a, nil
	}

	cardIdx := a.board.columns[colIdx].SelectedIndex()
	targetCol := colIdx - 1
	rb := moveRollback{Card: card, FromCol: colIdx, FromIdx: cardIdx, ToCol: targetCol}

	a.board.RemoveCard(colIdx, cardIdx)
	a.board.InsertCard(targetCol, card, 0)
	a.board.FocusLeft()
	a.status = fmt.Sprintf("Moving %q...", card.Name)

	targetListID := a.board.columns[targetCol].ListID()
	return a, a.moveCardToListCmd(card.ID, targetListID, "top", rb)
}

func (a App) handleMoveCardRight() (tea.Model, tea.Cmd) {
	if !a.boardReady || a.board.focused >= len(a.board.columns)-1 {
		return a, nil
	}

	card, colIdx, ok := a.board.SelectedCard()
	if !ok {
		return a, nil
	}

	cardIdx := a.board.columns[colIdx].SelectedIndex()
	targetCol := colIdx + 1
	rb := moveRollback{Card: card, FromCol: colIdx, FromIdx: cardIdx, ToCol: targetCol}

	a.board.RemoveCard(colIdx, cardIdx)
	a.board.InsertCard(targetCol, card, 0)
	a.board.FocusRight()
	a.status = fmt.Sprintf("Moving %q...", card.Name)

	targetListID := a.board.columns[targetCol].ListID()
	return a, a.moveCardToListCmd(card.ID, targetListID, "top", rb)
}

func (a App) handleMoveCardUp() (tea.Model, tea.Cmd) {
	if !a.boardReady {
		return a, nil
	}

	card, colIdx, ok := a.board.SelectedCard()
	if !ok {
		return a, nil
	}

	cardIdx := a.board.columns[colIdx].SelectedIndex()
	if cardIdx <= 0 {
		return a, nil
	}

	cards := a.board.columns[colIdx].cards

	// Calculate new position BEFORE the swap using the neighbors at the target index
	var newPos float64
	if cardIdx-1 == 0 {
		newPos = cards[0].Pos / 2.0
	} else {
		newPos = (cards[cardIdx-2].Pos + cards[cardIdx-1].Pos) / 2.0
	}

	cards[cardIdx], cards[cardIdx-1] = cards[cardIdx-1], cards[cardIdx]
	cards[cardIdx-1].Pos = newPos // Update local Pos so subsequent moves calculate correctly
	a.board.rebuildColumnItemsAt(colIdx, cardIdx-1)

	return a, a.reorderCardCmd(card.ID, newPos)
}

func (a App) handleMoveCardDown() (tea.Model, tea.Cmd) {
	if !a.boardReady {
		return a, nil
	}

	card, colIdx, ok := a.board.SelectedCard()
	if !ok {
		return a, nil
	}

	cardIdx := a.board.columns[colIdx].SelectedIndex()
	cards := a.board.columns[colIdx].cards
	if cardIdx >= len(cards)-1 {
		return a, nil
	}

	// Calculate new position BEFORE the swap using the neighbors at the target index
	var newPos float64
	if cardIdx+1 >= len(cards)-1 {
		newPos = cards[len(cards)-1].Pos + 65536.0
	} else {
		newPos = (cards[cardIdx+1].Pos + cards[cardIdx+2].Pos) / 2.0
	}

	cards[cardIdx], cards[cardIdx+1] = cards[cardIdx+1], cards[cardIdx]
	cards[cardIdx+1].Pos = newPos // Update local Pos so subsequent moves calculate correctly
	a.board.rebuildColumnItemsAt(colIdx, cardIdx+1)

	return a, a.reorderCardCmd(card.ID, newPos)
}

func (a App) executeCustomCommand(cmd config.CustomCommandConfig) (tea.Model, tea.Cmd) {
	card, colIdx, ok := a.board.SelectedCard()
	if !ok {
		a.status = "No card selected"
		return a, nil
	}

	col := a.board.columns[colIdx]
	ctx := commands.BuildContext(card, trello.List{ID: col.ListID(), Name: col.Title()}, *a.board.board)

	// If command has prompts, start the prompt flow
	if len(cmd.Prompts) > 0 {
		a.pendingCommand = &cmd
		a.pendingCtx = ctx
		a.promptIdx = 0
		return a.showNextPrompt()
	}

	// No prompts — execute immediately
	return a.runCommand(cmd, ctx)
}

func (a App) showNextPrompt() (tea.Model, tea.Cmd) {
	if a.promptIdx >= len(a.pendingCommand.Prompts) {
		// All prompts done, execute the command
		cmd := *a.pendingCommand
		ctx := a.pendingCtx
		a.pendingCommand = nil
		a.showPrompt = false
		return a.runCommand(cmd, ctx)
	}

	prompt := a.pendingCommand.Prompts[a.promptIdx]
	a.showPrompt = true
	a.promptType = prompt.Type

	// Render the title template with current context
	title, err := commands.RenderTemplate(prompt.Title, a.pendingCtx)
	if err != nil {
		a.status = fmt.Sprintf("Prompt template error: %v", err)
		a.pendingCommand = nil
		a.showPrompt = false
		return a, nil
	}

	switch prompt.Type {
	case "confirm":
		a.status = title + " (y/n)"
	case "input":
		ti := textinput.New()
		ti.Placeholder = title
		ti.Focus()
		a.promptInput = ti
	case "menu":
		// Reuse command palette for menu options
		items := make([]list.Item, len(prompt.Options))
		for i, opt := range prompt.Options {
			items[i] = commandItem{cmd: config.CustomCommandConfig{Description: opt.Name, Key: opt.Value}}
		}
		a.commandPalette.SetItems(items)
		a.showPalette = true
	}

	return a, nil
}

func (a App) runCommand(cmd config.CustomCommandConfig, ctx commands.TemplateContext) (tea.Model, tea.Cmd) {
	rendered, err := commands.RenderTemplate(cmd.Command, ctx)
	if err != nil {
		a.status = fmt.Sprintf("Template error: %v", err)
		return a, nil
	}

	switch cmd.Output {
	case "terminal":
		c := commands.ExecuteTerminal(rendered)
		return a, tea.ExecProcess(c, func(err error) tea.Msg {
			if err != nil {
				return StatusMsg{Text: fmt.Sprintf("Command failed: %v", err)}
			}
			return StatusMsg{Text: "Command completed"}
		})
	case "popup":
		return a, func() tea.Msg {
			output, err := commands.ExecuteSilent(rendered)
			if err != nil {
				return StatusMsg{Text: fmt.Sprintf("Error: %v — %s", err, output)}
			}
			return StatusMsg{Text: output}
		}
	default: // "none"
		return a, func() tea.Msg {
			_, err := commands.ExecuteSilent(rendered)
			if err != nil {
				return StatusMsg{Text: fmt.Sprintf("Command failed: %v", err)}
			}
			return StatusMsg{Text: "Command executed"}
		}
	}
}

func matchKey(msg tea.KeyPressMsg, binding key.Binding) bool {
	for _, k := range binding.Keys() {
		if msg.String() == k {
			return true
		}
	}
	return false
}

func (a App) View() tea.View {
	if a.showHelp {
		v := tea.NewView(a.renderHelp())
		v.AltScreen = true
		return v
	}

	var content string
	if a.showPalette {
		paletteView := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("cyan")).
			Padding(1).
			Width(a.width / 2).
			Render(a.commandPalette.View())
		content = lipgloss.Place(a.width, a.height-2, lipgloss.Center, lipgloss.Center, paletteView)
	} else if a.showPrompt && a.promptType == "input" {
		content = lipgloss.Place(a.width, a.height-2, lipgloss.Center, lipgloss.Center, a.promptInput.View())
	} else if a.loading {
		content = "\n  Loading board...\n"
	} else if a.boardReady {
		if a.detail.open {
			content = lipgloss.JoinHorizontal(lipgloss.Top, a.board.View(), a.detail.View())
		} else {
			content = a.board.View()
		}
	} else {
		content = "\n  " + a.status + "\n"
	}

	statusBar := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 1).
		Render(a.status)

	view := lipgloss.JoinVertical(lipgloss.Left, content, statusBar)

	v := tea.NewView(view)
	v.AltScreen = true
	return v
}

func (a App) renderHelp() string {
	title := lipgloss.NewStyle().Bold(true).Padding(1).Render("tuillo — Keyboard Shortcuts")

	keys := []struct{ key, desc string }{
		{a.keyMap.Quit.Keys()[0], "Quit"},
		{a.keyMap.Help.Keys()[0], "Toggle help"},
		{a.keyMap.Refresh.Keys()[0], "Refresh board"},
		{a.keyMap.MoveLeft.Keys()[0] + "/" + "\u2190", "Focus column left"},
		{a.keyMap.MoveRight.Keys()[0] + "/" + "\u2192", "Focus column right"},
		{a.keyMap.MoveUp.Keys()[0] + "/" + "\u2191", "Focus card up"},
		{a.keyMap.MoveDown.Keys()[0] + "/" + "\u2193", "Focus card down"},
		{a.keyMap.MoveCardLeft.Keys()[0], "Move card left"},
		{a.keyMap.MoveCardRight.Keys()[0], "Move card right"},
		{a.keyMap.MoveCardUp.Keys()[0], "Move card up"},
		{a.keyMap.MoveCardDown.Keys()[0], "Move card down"},
		{a.keyMap.CustomCommand.Keys()[0], "Command palette"},
		{a.keyMap.DetailToggle.Keys()[0], "Toggle detail panel"},
		{a.keyMap.DetailTabPrev.Keys()[0] + "/" + a.keyMap.DetailTabNext.Keys()[0], "Switch detail tab"},
		{a.keyMap.DetailScrollDown.Keys()[0], "Scroll detail down"},
		{a.keyMap.DetailScrollUp.Keys()[0], "Scroll detail up"},
	}

	lines := title + "\n\n"
	for _, k := range keys {
		lines += fmt.Sprintf("  %-12s %s\n", k.key, k.desc)
	}
	lines += "\n  Press ? or Esc to close"
	return lines
}
