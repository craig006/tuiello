// internal/tui/app.go
package tui

import (
	"fmt"
	"strings"

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
		detail:         NewDetailModel(km, NewTheme(cfg.GUI.Theme)),
	}
	a.viewBar = NewViewBar(cfg.Views)
	si := textinput.New()
	si.Placeholder = "Search... (/ to focus, ctrl+m members, ctrl+l labels, esc clear)"
	si.Prompt = "\uf002 "
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
	// border (2) + prompt icon+space (2) = 4 chars of chrome
	a.searchInput.SetWidth(a.board.width - 4)
}

func (a *App) updateDetailLayout() {
	boardHeight := a.height - 3 // 3 for view bar (border top + content + border bottom)
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

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		if a.boardReady {
			boardHeight := msg.Height - 3 // 3 for view bar (border top + content + border bottom)
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
		boardHeight := a.height - 3 // 3 for view bar (border top + content + border bottom)
		a.board = NewBoardModel(msg.Board, a.config, a.width, boardHeight)
		a.updateSearchWidth()
		// Apply active view's filter
		viewCfg := a.viewBar.ActiveConfig()
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
				fetchCmd := a.syncDetailAfterFilter()
				return a, fetchCmd
			}
			return a, nil
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
					a.board.height = a.height - 3
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
		}

		// Direct-jump view switching — checked last so it never shadows standard keys
		if a.boardReady && !a.showPalette && !a.showPrompt && !a.searchFocused && !a.showMemberModal && !a.showLabelModal {
			if a.viewBar.SelectByKey(msg.String()) {
				return a, a.applyActiveView()
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
	if !a.boardReady || a.board.focused == 0 {
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

	cardIdx := fullCardIndex(a.board.columns[colIdx].cards, card.ID)
	if cardIdx < 0 {
		return a, nil
	}
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
			a.board.height = a.height
			a.board.ResizeColumns()
			a.updateSearchWidth()
		}
	}

	return a.syncDetailAfterFilter()
}

func (a App) renderFilterDisplay() string {
	f := ParseFilter(a.searchInput.Value(), a.currentUser)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))
	tokenStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(4))
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(7))

	var parts []string
	if f.Text != "" {
		parts = append(parts, textStyle.Render(f.Text))
	}
	for _, m := range f.Members {
		if strings.Contains(m, " ") {
			parts = append(parts, tokenStyle.Render(fmt.Sprintf(`member:"%s"`, m)))
		} else {
			parts = append(parts, tokenStyle.Render("member:"+m))
		}
	}
	for _, l := range f.Labels {
		if strings.Contains(l, " ") {
			parts = append(parts, tokenStyle.Render(fmt.Sprintf(`label:"%s"`, l)))
		} else {
			parts = append(parts, tokenStyle.Render("label:"+l))
		}
	}

	icon := dimStyle.Render("\uf002 ")
	return icon + strings.Join(parts, " ")
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
		// Render search bar
		searchContent := a.searchInput.View()
		if !a.searchFocused && a.searchInput.Value() != "" {
			// Show the filter text with styled tokens when not focused
			searchContent = a.renderFilterDisplay()
		}
		searchBar := lipgloss.NewStyle().
			Width(a.board.width).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.ANSIColor(8)).
			Render(searchContent)
		a.board.SetSearchBar(searchBar)

		// Render view bar at full screen width, above everything
		boardName := ""
		if a.board.board != nil {
			boardName = a.board.board.Name
		}
		viewBarContent := a.viewBar.View(a.width, boardName)

		var boardContent string
		if a.detail.open {
			boardContent = lipgloss.JoinHorizontal(lipgloss.Top, a.board.View(), " ", a.detail.View())
		} else {
			boardContent = a.board.View()
		}
		content = viewBarContent + "\n" + boardContent

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
