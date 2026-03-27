// internal/tui/app.go
package tui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/craig006/tuiello/internal/commands"
	"github.com/craig006/tuiello/internal/config"
	"github.com/craig006/tuiello/internal/trello"
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
	Err error
	// For rollback
	Card    trello.Card
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

type CurrentUserMsg struct {
	Username string
}

type CurrentUserErrMsg struct {
	Err error
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

	// Views
	viewBar     ViewBar
	currentUser string // resolved Trello username for @me

	// Filter search bar
	searchInput   textinput.Model
	searchFocused bool

	// Filter modals
	showMemberModal bool
	showLabelModal  bool
	memberModal     MultiSelectModel
	labelModal      MultiSelectModel

	focusManager *FocusManager
}

func NewApp(client *trello.Client, cfg config.Config) App {
	km := NewKeyMap(cfg.Keybinding)
	palette := list.New(nil, list.NewDefaultDelegate(), 40, 20)
	palette.Title = "Commands"
	a := App{
		client:         client,
		config:         cfg,
		keyMap:         km,
		help:           help.New(),
		status:         "Loading board...",
		loading:        true,
		commandPalette: palette,
		detail:         NewDetailModel(km, NewTheme(cfg.GUI.Theme), cfg.GUI.Padding),
		focusManager:   NewFocusManager("board"),
	}
	a.viewBar = NewViewBar(cfg.Views)
	si := textinput.New()
	si.Placeholder = "Search... (/ to focus, ctrl+m members, ctrl+l labels, esc clear)"
	si.Prompt = "\uf002  "
	searchBg := lipgloss.Color("238")
	styles := si.Styles()
	styles.Focused.Prompt = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(4)).Background(searchBg)
	styles.Focused.Text = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(15)).Background(searchBg)
	styles.Focused.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Background(searchBg)
	styles.Blurred.Prompt = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Background(searchBg)
	styles.Blurred.Text = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(7)).Background(searchBg)
	styles.Blurred.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Background(searchBg)
	si.SetStyles(styles)
	si.SetWidth(0) // will be set on first render
	a.searchInput = si
	return a
}

func (a App) fetchCurrentUserCmd() tea.Cmd {
	return func() tea.Msg {
		user, err := a.client.FetchCurrentUser()
		if err != nil {
			return CurrentUserErrMsg{Err: err}
		}
		return CurrentUserMsg{Username: user.Username}
	}
}

func (a App) Init() tea.Cmd {
	userCmd := a.fetchCurrentUserCmd()
	boardID := a.config.Board.ID
	if boardID == "" && a.config.Board.Name != "" {
		return tea.Batch(a.resolveBoardCmd(a.config.Board.Name), userCmd)
	}
	if boardID == "" {
		return tea.Batch(func() tea.Msg {
			return BoardFetchErrMsg{Err: fmt.Errorf("no board configured — use --board or --board-id, or set board.id in config")}
		}, userCmd)
	}
	return tea.Batch(a.fetchBoardCmd(boardID), userCmd)
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

func (a *App) updateSearchWidth() {
	// padding + focus bar (1) + prompt icon+spaces (3) = ~5 chars of chrome
	a.searchInput.SetWidth(a.width - a.config.GUI.Padding - 5)
}

func (a *App) updateDetailLayout() {
	boardHeight := a.height - 10 // 3 for view bar + 3 for search bar + 1 margin + 3 for breadcrumb
	boardWidth := a.width * 60 / 100
	panelWidth := a.width - boardWidth - 1 // 1 char spacer between board and detail
	a.board.width = boardWidth
	a.board.height = boardHeight
	a.board.ResizeColumns()
	a.detail.SetSize(panelWidth, boardHeight)
	a.updateSearchWidth()
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

// HandleKeyEvent processes app-level keyboard shortcuts
// Returns true if handled, false to continue bubbling
// NOTE: Quit (q, ctrl+c) is NOT handled here - it's handled in the main Update switch
// to ensure tea.Quit() is properly returned
func (a *App) HandleKeyEvent(key string) bool {
	// Global shortcuts that always work (but NOT quit - see note above)
	switch key {
	case "?": // Help
		a.showHelp = !a.showHelp
		return true
	}
	return false
}

// HandleSearchKeyEvent processes keyboard events for the search input
// Returns true if handled, false to continue
func (a *App) HandleSearchKeyEvent(key string) bool {
	// Search input handling remains minimal for now
	// Focus management will be enhanced in later refactoring
	switch key {
	case "esc":
		a.searchFocused = false
		return true
	}
	return false
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		if a.boardReady {
			boardHeight := msg.Height - 6 // 3 for view bar + 3 for breadcrumb
			if a.detail.open {
				a.updateDetailLayout()
			} else {
				a.board.width = msg.Width
				a.board.height = boardHeight
				a.board.ResizeColumns()
				a.updateSearchWidth()
			}
		}
		return a, nil

	case BoardFetchedMsg:
		a.loading = false
		a.boardReady = true
		boardHeight := a.height - 7 // 3 for view bar + 3 for search bar + 1 margin
		a.board = NewBoardModel(msg.Board, a.config, a.width, boardHeight)
		a.updateSearchWidth()
		// Apply active view's filter
		viewCfg := a.viewBar.ActiveConfig()
		a.board.SetHiddenColumns(viewCfg.HideColumns)
		if viewCfg.Filter != "" {
			a.searchInput.SetValue(viewCfg.Filter)
			f := ParseFilter(viewCfg.Filter, a.currentUser)
			a.board.ApplyFilter(f)
		} else if a.searchInput.Value() != "" {
			f := ParseFilter(a.searchInput.Value(), a.currentUser)
			a.board.ApplyFilter(f)
		}
		a.status = fmt.Sprintf("%s — %s", msg.Board.Name, a.board.PositionIndicator())
		showDetail := a.config.GUI.ShowDetailPanel
		if viewCfg.ShowDetailPanel != nil {
			showDetail = *viewCfg.ShowDetailPanel
		}
		a.detail.open = false
		a.detail.cardID = ""
		if showDetail && a.width >= 80 {
			a.detail.open = true
			if card, _, ok := a.board.SelectedCard(); ok {
				a.detail.SetCard(card)
				a.updateDetailLayout()
				if a.detail.NeedsFetch() {
					a.detail.MarkLoading()
					return a, a.fetchDetailData()
				}
			} else {
				a.updateDetailLayout()
			}
		}
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

	case CreateCommentRequestMsg:
		return a, a.createCommentCmd(msg.Text)

	case UpdateCommentRequestMsg:
		return a, a.updateCommentCmd(msg.CommentID, msg.Text)

	case DeleteCommentRequestMsg:
		return a, a.deleteCommentCmd(msg.CommentID)

	case CommentOperationErrMsg:
		a.status = fmt.Sprintf("Failed to %s comment: %v", msg.Operation, msg.Err)
		return a, nil

	case CurrentUserMsg:
		a.currentUser = msg.Username
		// Re-apply active view's filter now that @me can resolve
		if a.boardReady {
			viewCfg := a.viewBar.ActiveConfig()
			if viewCfg.Filter != "" {
				a.searchInput.SetValue(viewCfg.Filter)
				f := ParseFilter(viewCfg.Filter, a.currentUser)
				a.board.ApplyFilter(f)
			}
		}
		return a, nil

	case CurrentUserErrMsg:
		// Graceful degradation — @me won't resolve but app continues
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
				a.focusManager.CloseModal()
					a.status = "Cancelled"
					return a, nil
				}
				if msg.String() == "enter" {
					if item, ok := a.commandPalette.SelectedItem().(commandItem); ok {
						a.pendingCtx.Prompt[prompt.Key] = item.cmd.Key
						a.promptIdx++
						a.showPrompt = false
						a.showPalette = false
				a.focusManager.CloseModal()
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
				a.focusManager.CloseModal()
				return a, nil
			}
			if msg.String() == "enter" {
				if item, ok := a.commandPalette.SelectedItem().(commandItem); ok {
					a.showPalette = false
					a.focusManager.CloseModal()
					return a.executeCustomCommand(item.cmd)
				}
			}
			var cmd tea.Cmd
			a.commandPalette, cmd = a.commandPalette.Update(msg)
			return a, cmd
		}

		// Handle member/label modals
		if a.showMemberModal || a.showLabelModal {
			modal := &a.memberModal
			isLabel := false
			if a.showLabelModal {
				modal = &a.labelModal
				isLabel = true
			}

			switch msg.String() {
			case "j", "down":
				modal.MoveDown()
			case "k", "up":
				modal.MoveUp()
			case " ", "space":
				modal.Toggle()
			case "enter", "esc":
				// Close modal and update search bar with selections
				selected := modal.Selected()
				currentFilter := ParseFilter(a.searchInput.Value(), a.currentUser)
				if isLabel {
					currentFilter.Labels = selected
				} else {
					currentFilter.Members = selected
				}
				a.searchInput.SetValue(BuildFilterText(currentFilter))
				a.board.ApplyFilter(currentFilter)
				a.showMemberModal = false
				a.showLabelModal = false
			a.focusManager.CloseModal()
				fetchCmd := a.syncDetailAfterFilter()
				return a, fetchCmd
			}
			return a, nil
		}

		// Route keyboard events through focus hierarchy
		// 1. Check if app-level shortcuts handle it
		if a.HandleKeyEvent(msg.String()) {
			return a, nil
		}

		// 2. Handle focus toggles (board <-> detail) — only if no other UI is active
		switch {
		// Focus toggles
		case matchKey(msg, a.keyMap.FocusDetail):
			// Enter focuses detail (only if board has focus and a card is selected)
			if a.focusManager.FocusedSection() == "board" {
				if _, _, ok := a.board.SelectedCard(); ok {
					a.focusManager.SetFocusedSection("detail")
					a.detail.SetFocus(true)
					return a, nil
				}
			}

		case matchKey(msg, a.keyMap.FocusBoard):
			// Escape returns focus to board
			if a.focusManager.FocusedSection() == "detail" {
				a.focusManager.SetFocusedSection("board")
				a.detail.SetFocus(false)
				return a, nil
			}
		}

		// Handle search bar input
		if a.searchFocused {
			switch msg.String() {
			case "enter":
				a.searchFocused = false
				a.searchInput.Blur()
				return a, nil
			case "esc":
				a.searchFocused = false
				a.searchInput.Blur()
				// Reset to view's base filter instead of clearing completely
				viewCfg := a.viewBar.ActiveConfig()
				a.searchInput.SetValue(viewCfg.Filter)
				if viewCfg.Filter != "" {
					f := ParseFilter(viewCfg.Filter, a.currentUser)
					a.board.ApplyFilter(f)
				} else {
					a.board.ClearFilter()
				}
				fetchCmd := a.syncDetailAfterFilter()
				return a, fetchCmd
			default:
				var cmd tea.Cmd
				a.searchInput, cmd = a.searchInput.Update(msg)
				// Live filtering on every keystroke
				f := ParseFilter(a.searchInput.Value(), a.currentUser)
				a.board.ApplyFilter(f)
				fetchCmd := a.syncDetailAfterFilter()
				if fetchCmd != nil {
					return a, tea.Batch(cmd, fetchCmd)
				}
				return a, cmd
			}
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

		case matchKey(msg, a.keyMap.OpenCard):
			return a.handleOpenSelectedCard()

		case matchKey(msg, a.keyMap.CopyCardURL):
			return a.handleCopySelectedCardURL()

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
			a.focusManager.OpenModal()
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

		case msg.String() == "ctrl+g":
			return a.handleMoveCardToTop()

		case msg.String() == "ctrl+shift+g":
			return a.handleMoveCardToBottom()

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
					a.board.height = a.height - 10
					a.board.ResizeColumns()
					a.updateSearchWidth()
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

		case matchKey(msg, a.keyMap.FilterFocus):
			if a.boardReady && !a.showPalette {
				a.searchFocused = true
				a.searchInput.Focus()
				// Add trailing space so new terms start fresh
				if val := a.searchInput.Value(); val != "" && !strings.HasSuffix(val, " ") {
					a.searchInput.SetValue(val + " ")
				}
				a.searchInput.CursorEnd()
				return a, nil
			}

		case matchKey(msg, a.keyMap.FilterMembers):
			if a.boardReady && !a.showPalette && !a.searchFocused {
				currentFilter := ParseFilter(a.searchInput.Value(), a.currentUser)
				var items []MultiSelectItem
				for _, m := range a.board.board.Members {
					checked := false
					for _, fm := range currentFilter.Members {
						if strings.EqualFold(fm, m.Username) || strings.EqualFold(fm, m.FullName) {
							checked = true
							break
						}
					}
					items = append(items, MultiSelectItem{
						Label:   m.FullName,
						Value:   m.Username,
						Checked: checked,
					})
				}
				a.memberModal = NewMultiSelectModel("Filter by Member", items)
				a.showMemberModal = true
			a.focusManager.OpenModal()
				return a, nil
			}

		case matchKey(msg, a.keyMap.FilterLabels):
			if a.boardReady && !a.showPalette && !a.searchFocused {
				currentFilter := ParseFilter(a.searchInput.Value(), a.currentUser)
				// Collect unique labels from all cards across all columns
				seen := make(map[string]bool)
				var items []MultiSelectItem
				for _, col := range a.board.columns {
					for _, card := range col.cards {
						for _, lbl := range card.Labels {
							name := lbl.Name
							if name == "" {
								name = lbl.Color
							}
							if seen[name] {
								continue
							}
							seen[name] = true
							checked := false
							for _, fl := range currentFilter.Labels {
								if strings.EqualFold(fl, name) {
									checked = true
									break
								}
							}
							ansiColor, ok := trelloColorToANSI[lbl.Color]
							if !ok {
								ansiColor = lipgloss.ANSIColor(7)
							}
							items = append(items, MultiSelectItem{
								Label:   name,
								Value:   name,
								Checked: checked,
								Color:   ansiColor,
							})
						}
					}
				}
				a.labelModal = NewMultiSelectModel("Filter by Label", items)
				a.showLabelModal = true
			a.focusManager.OpenModal()
				return a, nil
			}

		case matchKey(msg, a.keyMap.ViewNext):
			if a.boardReady && !a.showPalette && !a.showPrompt && !a.searchFocused && !a.showMemberModal && !a.showLabelModal {
				a.viewBar.Next()
				return a, a.applyActiveView()
			}

		case matchKey(msg, a.keyMap.ViewPrev):
			if a.boardReady && !a.showPalette && !a.showPrompt && !a.searchFocused && !a.showMemberModal && !a.showLabelModal {
				a.viewBar.Prev()
				return a, a.applyActiveView()
			}

		case msg.String() == "esc":
			if a.showHelp {
				a.showHelp = false
				return a, nil
			}
			if a.boardReady && !a.showPalette && !a.showPrompt {
				viewCfg := a.viewBar.ActiveConfig()
				if a.searchInput.Value() != viewCfg.Filter {
					// Reset to view's base filter
					a.searchInput.SetValue(viewCfg.Filter)
					if viewCfg.Filter != "" {
						f := ParseFilter(viewCfg.Filter, a.currentUser)
						a.board.ApplyFilter(f)
					} else {
						a.board.ClearFilter()
					}
					fetchCmd := a.syncDetailAfterFilter()
					return a, fetchCmd
				}
			}
			return a, nil
		}

		// Direct-jump view switching — checked last so it never shadows standard keys
		if a.boardReady && !a.showPalette && !a.showPrompt && !a.searchFocused && !a.showMemberModal && !a.showLabelModal {
			if a.viewBar.SelectByKey(msg.String()) {
				return a, a.applyActiveView()
			}
		}

		// Route to board or detail based on focus
		if a.boardReady {
			var cmd tea.Cmd
			if a.focusManager.FocusedSection() == "board" {
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
			} else {
				a.detail, cmd = a.detail.Update(msg)
			}
			return a, cmd
		}
	}

	return a, nil
}

// fullCardIndex finds the index of a card by ID in the full (unfiltered) cards slice.
func fullCardIndex(cards []trello.Card, id string) int {
	for i, c := range cards {
		if c.ID == id {
			return i
		}
	}
	return -1
}

func (a App) handleMoveCardLeft() (tea.Model, tea.Cmd) {
	if !a.boardReady {
		return a, nil
	}

	card, colIdx, ok := a.board.SelectedCard()
	if !ok {
		return a, nil
	}

	cardIdx := fullCardIndex(a.board.columns[colIdx].cards, card.ID)
	if cardIdx < 0 {
		return a, nil
	}
	visiblePos := a.board.visibleColumnPosition(colIdx)
	if visiblePos <= 0 {
		return a, nil
	}
	targetCol := a.board.VisibleColumnIndices()[visiblePos-1]
	rb := moveRollback{Card: card, FromCol: colIdx, FromIdx: cardIdx, ToCol: targetCol}

	a.board.RemoveCard(colIdx, cardIdx)
	a.board.InsertCard(targetCol, card, 0)
	a.board.FocusLeft()
	a.status = fmt.Sprintf("Moving %q...", card.Name)

	targetListID := a.board.columns[targetCol].ListID()
	return a, a.moveCardToListCmd(card.ID, targetListID, "top", rb)
}

func (a App) handleMoveCardRight() (tea.Model, tea.Cmd) {
	if !a.boardReady {
		return a, nil
	}

	card, colIdx, ok := a.board.SelectedCard()
	if !ok {
		return a, nil
	}

	cardIdx := fullCardIndex(a.board.columns[colIdx].cards, card.ID)
	if cardIdx < 0 {
		return a, nil
	}
	visible := a.board.VisibleColumnIndices()
	visiblePos := a.board.visibleColumnPosition(colIdx)
	if visiblePos < 0 || visiblePos >= len(visible)-1 {
		return a, nil
	}
	targetCol := visible[visiblePos+1]
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

	cards := a.board.columns[colIdx].cards
	cardIdx := fullCardIndex(cards, card.ID)
	if cardIdx <= 0 {
		return a, nil
	}

	// Find the target position: skip past hidden cards to the previous visible card
	targetIdx := cardIdx - 1
	if a.board.HasFilter() {
		for targetIdx > 0 && !a.board.filter.MatchesCard(cards[targetIdx]) {
			targetIdx--
		}
		// If target is not visible and we're at 0, move above it anyway
	}
	if targetIdx == cardIdx {
		return a, nil
	}

	// Calculate new position BEFORE the move
	var newPos float64
	if targetIdx == 0 {
		newPos = cards[0].Pos / 2.0
	} else {
		newPos = (cards[targetIdx-1].Pos + cards[targetIdx].Pos) / 2.0
	}

	// Remove card from current position and insert at target
	removed := cards[cardIdx]
	copy(cards[cardIdx:], cards[cardIdx+1:])
	a.board.columns[colIdx].cards = cards[:len(cards)-1]
	cards = a.board.columns[colIdx].cards

	newCards := make([]trello.Card, 0, len(cards)+1)
	newCards = append(newCards, cards[:targetIdx]...)
	removed.Pos = newPos
	newCards = append(newCards, removed)
	newCards = append(newCards, cards[targetIdx:]...)
	a.board.columns[colIdx].cards = newCards
	a.board.rebuildColumnItemsAt(colIdx, targetIdx)

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

	cards := a.board.columns[colIdx].cards
	cardIdx := fullCardIndex(cards, card.ID)
	if cardIdx < 0 || cardIdx >= len(cards)-1 {
		return a, nil
	}

	// Find the target position: skip past hidden cards to the next visible card
	targetIdx := cardIdx + 1
	if a.board.HasFilter() {
		for targetIdx < len(cards)-1 && !a.board.filter.MatchesCard(cards[targetIdx]) {
			targetIdx++
		}
	}
	if targetIdx == cardIdx {
		return a, nil
	}

	// Insert AFTER the target, so targetIdx+1
	insertAt := targetIdx + 1

	// Calculate new position BEFORE the move
	var newPos float64
	if insertAt >= len(cards) {
		newPos = cards[len(cards)-1].Pos + 65536.0
	} else {
		newPos = (cards[targetIdx].Pos + cards[insertAt].Pos) / 2.0
	}

	// Remove card from current position and insert after target
	removed := cards[cardIdx]
	copy(cards[cardIdx:], cards[cardIdx+1:])
	a.board.columns[colIdx].cards = cards[:len(cards)-1]
	cards = a.board.columns[colIdx].cards

	// Adjust insertAt since we removed an element before it
	insertAt--

	newCards := make([]trello.Card, 0, len(cards)+1)
	newCards = append(newCards, cards[:insertAt]...)
	removed.Pos = newPos
	newCards = append(newCards, removed)
	newCards = append(newCards, cards[insertAt:]...)
	a.board.columns[colIdx].cards = newCards
	a.board.rebuildColumnItemsAt(colIdx, insertAt)

	return a, a.reorderCardCmd(card.ID, newPos)
}

func (a App) handleMoveCardToTop() (tea.Model, tea.Cmd) {
	if !a.boardReady {
		return a, nil
	}

	card, colIdx, ok := a.board.SelectedCard()
	if !ok {
		return a, nil
	}

	cards := a.board.columns[colIdx].cards
	cardIdx := fullCardIndex(cards, card.ID)
	if cardIdx <= 0 {
		return a, nil
	}

	newPos := cards[0].Pos / 2.0

	removed := cards[cardIdx]
	copy(cards[cardIdx:], cards[cardIdx+1:])
	a.board.columns[colIdx].cards = cards[:len(cards)-1]
	cards = a.board.columns[colIdx].cards

	newCards := make([]trello.Card, 0, len(cards)+1)
	removed.Pos = newPos
	newCards = append(newCards, removed)
	newCards = append(newCards, cards...)
	a.board.columns[colIdx].cards = newCards
	a.board.rebuildColumnItemsAt(colIdx, 0)

	return a, a.reorderCardCmd(card.ID, newPos)
}

func (a App) handleMoveCardToBottom() (tea.Model, tea.Cmd) {
	if !a.boardReady {
		return a, nil
	}

	card, colIdx, ok := a.board.SelectedCard()
	if !ok {
		return a, nil
	}

	cards := a.board.columns[colIdx].cards
	cardIdx := fullCardIndex(cards, card.ID)
	if cardIdx < 0 || cardIdx >= len(cards)-1 {
		return a, nil
	}

	newPos := cards[len(cards)-1].Pos + 65536.0

	removed := cards[cardIdx]
	copy(cards[cardIdx:], cards[cardIdx+1:])
	a.board.columns[colIdx].cards = cards[:len(cards)-1]
	cards = a.board.columns[colIdx].cards

	newCards := make([]trello.Card, 0, len(cards)+1)
	newCards = append(newCards, cards...)
	removed.Pos = newPos
	newCards = append(newCards, removed)
	a.board.columns[colIdx].cards = newCards
	a.board.rebuildColumnItemsAt(colIdx, len(newCards)-1)

	return a, a.reorderCardCmd(card.ID, newPos)
}

func (a App) handleOpenSelectedCard() (tea.Model, tea.Cmd) {
	if !a.boardReady {
		return a, nil
	}

	card, _, ok := a.board.SelectedCard()
	if !ok {
		a.status = "No card selected"
		return a, nil
	}
	if card.URL == "" {
		a.status = "Selected card has no URL"
		return a, nil
	}

	a.status = fmt.Sprintf("Opening %q...", card.Name)
	return a, func() tea.Msg {
		if err := openExternalURL(card.URL); err != nil {
			return StatusMsg{Text: fmt.Sprintf("Open failed: %v", err)}
		}
		return StatusMsg{Text: fmt.Sprintf("Opened %q in Trello", card.Name)}
	}
}

func (a App) handleCopySelectedCardURL() (tea.Model, tea.Cmd) {
	if !a.boardReady {
		return a, nil
	}

	card, _, ok := a.board.SelectedCard()
	if !ok {
		a.status = "No card selected"
		return a, nil
	}
	if card.URL == "" {
		a.status = "Selected card has no URL"
		return a, nil
	}

	a.status = fmt.Sprintf("Copying URL for %q...", card.Name)
	return a, func() tea.Msg {
		if err := writeClipboard(card.URL); err != nil {
			return StatusMsg{Text: fmt.Sprintf("Copy failed: %v", err)}
		}
		return StatusMsg{Text: fmt.Sprintf("Copied URL for %q", card.Name)}
	}
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
	a.focusManager.OpenModal()
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

func (a *App) syncDetailAfterFilter() tea.Cmd {
	if a.detail.open {
		if card, _, ok := a.board.SelectedCard(); ok {
			if card.ID != a.detail.cardID {
				a.detail.SetCard(card)
				if a.detail.NeedsFetch() {
					a.detail.MarkLoading()
					return a.fetchDetailData()
				}
			}
		} else {
			a.detail.SetCard(trello.Card{})
		}
	}
	return nil
}

// applyActiveView applies the active view's filter and GUI overrides.
func (a *App) applyActiveView() tea.Cmd {
	viewCfg := a.viewBar.ActiveConfig()
	a.board.SetHiddenColumns(viewCfg.HideColumns)

	// Reset search bar to view's base filter
	a.searchInput.SetValue(viewCfg.Filter)
	if viewCfg.Filter != "" {
		f := ParseFilter(viewCfg.Filter, a.currentUser)
		a.board.ApplyFilter(f)
	} else {
		a.board.ClearFilter()
	}

	// Apply GUI overrides
	if viewCfg.ColumnWidth != nil {
		a.board.minColWidth = *viewCfg.ColumnWidth
		a.board.ResizeColumns()
	}

	if viewCfg.ShowDetailPanel != nil {
		shouldShow := *viewCfg.ShowDetailPanel
		if shouldShow && !a.detail.open && a.width >= 80 {
			a.detail.open = true
			if card, _, ok := a.board.SelectedCard(); ok {
				a.detail.SetCard(card)
				a.updateDetailLayout()
				if a.detail.NeedsFetch() {
					a.detail.MarkLoading()
					return a.fetchDetailData()
				}
			} else {
				a.updateDetailLayout()
			}
		} else if !shouldShow && a.detail.open {
			a.detail.open = false
			a.detail.cardID = ""
			a.board.width = a.width
			a.board.height = a.height - 6 // 3 for view bar + 3 for breadcrumb
			a.board.ResizeColumns()
			a.updateSearchWidth()
		}
	}

	return a.syncDetailAfterFilter()
}

func (a App) renderSearchBar(searchContent string, bg color.Color) string {
	pad := a.config.GUI.Padding

	if a.searchFocused {
		// Blue left bar when focused (like selected card style)
		focusBorder := lipgloss.Border{Left: "▎"}
		return lipgloss.NewStyle().
			Width(a.width).
			Padding(1, 0, 1, pad).
			MarginBottom(1).
			BorderLeft(true).
			BorderStyle(focusBorder).
			BorderForeground(lipgloss.ANSIColor(4)).
			BorderBackground(bg).
			Background(bg).
			Render(searchContent)
	}

	return lipgloss.NewStyle().
		Width(a.width).
		Padding(1, 0, 1, pad+1). // +1 to match width with focused bar character
		MarginBottom(1).
		Background(bg).
		Render(searchContent)
}

func (a App) renderFilterDisplay() string {
	bg := lipgloss.Color("238")
	tokenStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(4)).Background(bg)
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(7)).Background(bg)

	// Render tokens in their original order from the input
	tokens := tokenize(a.searchInput.Value())
	var parts []string
	for _, tok := range tokens {
		lower := strings.ToLower(tok)
		if strings.HasPrefix(lower, "member:") || strings.HasPrefix(lower, "label:") {
			parts = append(parts, tokenStyle.Render(tok))
		} else {
			parts = append(parts, textStyle.Render(tok))
		}
	}

	iconStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Background(bg)
	spaceStyle := lipgloss.NewStyle().Background(bg)
	icon := iconStyle.Render("\uf002 ")
	return icon + spaceStyle.Render(" ") + strings.Join(parts, " ")
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
		a.board.dimColumns = a.searchFocused

		// Render view bar at full screen width
		boardName := ""
		if a.board.board != nil {
			boardName = a.board.board.Name
		}
		viewBarContent := a.viewBar.View(a.width, boardName, a.config.GUI.Padding)

		// Render search bar at full screen width, directly below view bar
		searchContent := a.searchInput.View()
		if !a.searchFocused && a.searchInput.Value() != "" {
			searchContent = a.renderFilterDisplay()
		}
		searchBg := lipgloss.Color("238") // lighter than view bar (236)
		searchBarContent := a.renderSearchBar(searchContent, searchBg)

		var boardContent string
		if a.detail.open {
			// Update layout for detail panel (60/40 split)
			a.updateDetailLayout()

			// Pass focus state to board for border styling
			a.board.SetFocus(a.focusManager.FocusedSection() == "board")

			boardContent = lipgloss.JoinHorizontal(lipgloss.Top, a.board.View(), " ", a.detail.View())
		} else {
			// When detail is closed, board uses full width
			a.board.width = a.width
			a.board.height = a.height - 10
			a.board.ResizeColumns()
			a.board.SetFocus(true)  // board always has focus when detail is closed
			boardContent = a.board.View()
		}
		breadcrumbContent := a.board.RenderBreadcrumb(a.width)
		content = viewBarContent + "\n" + searchBarContent + "\n" + boardContent + breadcrumbContent

		// Overlay member/label modal if open
		if a.showMemberModal {
			modalView := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("cyan")).
				Padding(1).
				Width(a.width / 3).
				Render(a.memberModal.View())
			content = lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, modalView)
		} else if a.showLabelModal {
			modalView := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("cyan")).
				Padding(1).
				Width(a.width / 3).
				Render(a.labelModal.View())
			content = lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, modalView)
		}
	} else {
		content = "\n  " + a.status + "\n"
	}

	view := content

	v := tea.NewView(view)
	v.AltScreen = true
	return v
}

func (a App) renderHelp() string {
	title := lipgloss.NewStyle().Bold(true).Padding(1).Render("tuiello — Keyboard Shortcuts")

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
		{a.keyMap.OpenCard.Keys()[0], "Open selected card in Trello"},
		{a.keyMap.CopyCardURL.Keys()[0], "Copy selected card URL"},
		{a.keyMap.CustomCommand.Keys()[0], "Command palette"},
		{a.keyMap.DetailToggle.Keys()[0], "Toggle detail panel"},
		{a.keyMap.DetailTabPrev.Keys()[0] + "/" + a.keyMap.DetailTabNext.Keys()[0], "Switch detail tab"},
		{a.keyMap.DetailScrollDown.Keys()[0], "Scroll detail down"},
		{a.keyMap.DetailScrollUp.Keys()[0], "Scroll detail up"},
		{a.keyMap.FilterFocus.Keys()[0], "Search/filter"},
		{a.keyMap.FilterMembers.Keys()[0], "Filter by member"},
		{a.keyMap.FilterLabels.Keys()[0], "Filter by label"},
		{a.keyMap.ViewNext.Keys()[0] + "/" + a.keyMap.ViewPrev.Keys()[0], "Cycle views"},
		{"esc", "Clear filters"},
	}

	for i, view := range a.viewBar.views {
		keys = append(keys, struct{ key, desc string }{
			a.viewBar.keys[i], "View: " + view.Title,
		})
	}

	lines := title + "\n\n"
	for _, k := range keys {
		lines += fmt.Sprintf("  %-12s %s\n", k.key, k.desc)
	}
	lines += "\n  Press ? or Esc to close"
	return lines
}

// selectedCardID returns the ID of the currently selected card
func (a *App) selectedCardID() string {
	if card, _, ok := a.board.SelectedCard(); ok {
		return card.ID
	}
	return ""
}

// createCommentCmd generates a command to create a comment on the selected card
func (a *App) createCommentCmd(text string) tea.Cmd {
	return func() tea.Msg {
		comment, err := a.client.CreateComment(a.selectedCardID(), text)
		if err != nil {
			return CommentOperationErrMsg{Operation: "create", Err: err}
		}
		return CommentCreatedMsg{Comment: comment}
	}
}

// updateCommentCmd generates a command to update an existing comment
func (a *App) updateCommentCmd(commentID, text string) tea.Cmd {
	return func() tea.Msg {
		comment, err := a.client.UpdateComment(commentID, text)
		if err != nil {
			return CommentOperationErrMsg{Operation: "update", Err: err}
		}
		return CommentUpdatedMsg{Comment: comment}
	}
}

// deleteCommentCmd generates a command to delete a comment
func (a *App) deleteCommentCmd(commentID string) tea.Cmd {
	return func() tea.Msg {
		err := a.client.DeleteComment(commentID)
		if err != nil {
			return CommentOperationErrMsg{Operation: "delete", Err: err}
		}
		return CommentDeletedMsg{CommentID: commentID}
	}
}
