// internal/tui/app_test.go
package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/craig006/tuiello/internal/config"
	"github.com/craig006/tuiello/internal/trello"
)

func TestAppBoardHasFocusInitialized(t *testing.T) {
	cfg := config.DefaultConfig()
	client := trello.NewClient("key", "token")
	app := NewApp(client, cfg)

	if !app.boardHasFocus {
		t.Error("expected boardHasFocus to be true after NewApp initialization")
	}
}

func TestAppInitNoBoard(t *testing.T) {
	cfg := config.DefaultConfig()
	client := trello.NewClient("key", "token")
	app := NewApp(client, cfg)

	cmd := app.Init()
	if cmd == nil {
		t.Fatal("expected a command for missing board error")
	}

	// Init now returns tea.Batch, which produces a BatchMsg containing multiple commands.
	// One of the batched commands should produce a BoardFetchErrMsg.
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected tea.BatchMsg, got %T", msg)
	}
	foundErr := false
	for _, c := range batch {
		if c == nil {
			continue
		}
		m := c()
		if _, ok := m.(BoardFetchErrMsg); ok {
			foundErr = true
		}
	}
	if !foundErr {
		t.Error("expected one of the batched commands to produce BoardFetchErrMsg")
	}
}

func TestAppBoardFetchedMsg(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Board.ID = "board1"
	client := trello.NewClient("key", "token")
	app := NewApp(client, cfg)
	app.width = 90
	app.height = 30

	board := &trello.Board{
		ID:   "board1",
		Name: "Test Board",
		Lists: []trello.List{
			{ID: "l1", Name: "Todo", Cards: []trello.Card{
				{ID: "c1", Name: "Card 1", Pos: 1.0},
			}},
			{ID: "l2", Name: "Done", Cards: []trello.Card{}},
		},
	}

	model, _ := app.Update(BoardFetchedMsg{Board: board})
	updated := model.(App)

	if !updated.boardReady {
		t.Error("expected boardReady to be true")
	}
	if updated.loading {
		t.Error("expected loading to be false")
	}
}

func TestAppMoveCardUpTwice(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Board.ID = "board1"
	cfg.Views = []config.ViewConfig{{Title: "All Cards"}} // No filter so cards aren't hidden
	client := trello.NewClient("key", "token")
	app := NewApp(client, cfg)
	app.width = 90
	app.height = 30

	board := &trello.Board{
		ID:   "board1",
		Name: "Test Board",
		Lists: []trello.List{
			{ID: "l1", Name: "Todo", Cards: []trello.Card{
				{ID: "c0", Name: "Card 0", Pos: 1.0, ListID: "l1"},
				{ID: "c1", Name: "Card 1", Pos: 2.0, ListID: "l1"},
				{ID: "c2", Name: "Card 2", Pos: 3.0, ListID: "l1"},
			}},
			{ID: "l2", Name: "Done", Cards: []trello.Card{}},
		},
	}

	// Load board
	model, _ := app.Update(BoardFetchedMsg{Board: board})
	app = model.(App)

	// Select card at index 2
	app.board.columns[0].Select(2)
	t.Logf("Before moves: selectedIndex=%d", app.board.columns[0].SelectedIndex())

	card, _, ok := app.board.SelectedCard()
	if !ok || card.ID != "c2" {
		t.Fatalf("expected c2 selected, got %v (ok=%v)", card, ok)
	}

	// Simulate pressing K (MoveCardUp)
	msg := tea.KeyPressMsg{Code: -1, Text: "K"}
	model, _ = app.Update(msg)
	app = model.(App)

	t.Logf("After first move: selectedIndex=%d", app.board.columns[0].SelectedIndex())
	card, _, ok = app.board.SelectedCard()
	if !ok {
		t.Fatal("no card selected after first move")
	}
	t.Logf("After first move: selected card=%s", card.ID)
	if card.ID != "c2" {
		t.Fatalf("expected c2 still selected after first move, got %s", card.ID)
	}

	// Second press of K
	model, _ = app.Update(msg)
	app = model.(App)

	t.Logf("After second move: selectedIndex=%d", app.board.columns[0].SelectedIndex())
	card, _, ok = app.board.SelectedCard()
	if !ok {
		t.Fatal("no card selected after second move")
	}
	t.Logf("After second move: selected card=%s", card.ID)
	if card.ID != "c2" {
		t.Fatalf("expected c2 still selected after second move, got %s", card.ID)
	}
}

func TestDetailToggleKey(t *testing.T) {
	cfg := config.DefaultConfig()
	client := trello.NewClient("key", "token")
	app := NewApp(client, cfg)

	app.width = 120
	app.height = 30
	board := makeTestBoard(3)
	app.boardReady = true
	app.board = NewBoardModel(board, cfg, 120, 26)

	msg := tea.KeyPressMsg{Code: -1, Text: "d"}
	result, _ := app.Update(msg)
	a := result.(App)
	if !a.detail.open {
		t.Error("expected detail panel to be open after pressing 'd'")
	}

	result, _ = a.Update(msg)
	a = result.(App)
	if a.detail.open {
		t.Error("expected detail panel to be closed after pressing 'd' again")
	}
}

func TestDetailLayoutSplit(t *testing.T) {
	cfg := config.DefaultConfig()
	client := trello.NewClient("key", "token")
	app := NewApp(client, cfg)
	app.width = 100
	app.height = 30

	board := makeTestBoard(3)
	app.boardReady = true
	app.board = NewBoardModel(board, cfg, 100, 26)

	msg := tea.KeyPressMsg{Code: -1, Text: "d"}
	result, _ := app.Update(msg)
	a := result.(App)

	expectedBoardWidth := 100 * 60 / 100
	if a.board.width != expectedBoardWidth {
		t.Errorf("expected board width %d, got %d", expectedBoardWidth, a.board.width)
	}
}

func TestApplyActiveViewHidesColumns(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Views = []config.ViewConfig{{Title: "Working Cards", HideColumns: []string{"To Do"}}}
	client := trello.NewClient("key", "token")
	app := NewApp(client, cfg)
	app.width = 90
	app.height = 30
	app.boardReady = true
	app.board = NewBoardModel(&trello.Board{ID: "b1", Name: "Test", Lists: []trello.List{
		{ID: "l1", Name: "To Do", Cards: []trello.Card{{ID: "c1", Name: "Card 1", Pos: 1}}},
		{ID: "l2", Name: "Doing", Cards: []trello.Card{{ID: "c2", Name: "Card 2", Pos: 1}}},
	}}, cfg, 90, 23)

	app.applyActiveView()
	if got := app.board.VisibleColumnIndices(); len(got) != 1 || got[0] != 1 {
		t.Fatalf("expected only visible column index 1, got %#v", got)
	}
	if strings.Contains(app.board.View(), "To Do") {
		t.Fatal("expected hidden column to be absent after applying active view")
	}
	if app.board.FocusedColumn() != 1 {
		t.Fatalf("expected focus to move to first visible column, got %d", app.board.FocusedColumn())
	}
}

func TestEscClosesHelpWithoutQuitting(t *testing.T) {
	cfg := config.DefaultConfig()
	client := trello.NewClient("key", "token")
	app := NewApp(client, cfg)
	app.showHelp = true

	result, cmd := app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	updated := result.(App)

	if updated.showHelp {
		t.Fatal("expected esc to close help")
	}
	if cmd != nil {
		t.Fatalf("expected esc not to return a quit command, got %T", cmd)
	}
}

func TestOpenSelectedCardKeyAction(t *testing.T) {
	prevOpen := openExternalURL
	defer func() { openExternalURL = prevOpen }()

	var opened string
	openExternalURL = func(url string) error {
		opened = url
		return nil
	}

	cfg := config.DefaultConfig()
	client := trello.NewClient("key", "token")
	app := NewApp(client, cfg)
	app.width = 120
	app.height = 30
	app.boardReady = true
	board := makeTestBoard(1)
	board.Lists[0].Cards[0].URL = "https://trello.com/c/card-1"
	app.board = NewBoardModel(board, cfg, 120, 26)

	result, cmd := app.Update(tea.KeyPressMsg{Code: -1, Text: "o"})
	updated := result.(App)
	if updated.status != "Opening \"Card 0-1\"..." {
		t.Fatalf("unexpected status: %q", updated.status)
	}
	if cmd == nil {
		t.Fatal("expected open command")
	}
	msg := cmd()
	status, ok := msg.(StatusMsg)
	if !ok {
		t.Fatalf("expected StatusMsg, got %T", msg)
	}
	if opened != "https://trello.com/c/card-1" {
		t.Fatalf("expected card URL to open, got %q", opened)
	}
	if status.Text != "Opened \"Card 0-1\" in Trello" {
		t.Fatalf("unexpected status message: %q", status.Text)
	}
}

func TestCopySelectedCardURLKeyAction(t *testing.T) {
	prevClipboard := writeClipboard
	defer func() { writeClipboard = prevClipboard }()

	var copied string
	writeClipboard = func(text string) error {
		copied = text
		return nil
	}

	cfg := config.DefaultConfig()
	client := trello.NewClient("key", "token")
	app := NewApp(client, cfg)
	app.width = 120
	app.height = 30
	app.boardReady = true
	board := makeTestBoard(1)
	board.Lists[0].Cards[0].URL = "https://trello.com/c/card-1"
	app.board = NewBoardModel(board, cfg, 120, 26)

	result, cmd := app.Update(tea.KeyPressMsg{Code: -1, Text: "u"})
	updated := result.(App)
	if updated.status != "Copying URL for \"Card 0-1\"..." {
		t.Fatalf("unexpected status: %q", updated.status)
	}
	if cmd == nil {
		t.Fatal("expected copy command")
	}
	msg := cmd()
	status, ok := msg.(StatusMsg)
	if !ok {
		t.Fatalf("expected StatusMsg, got %T", msg)
	}
	if copied != "https://trello.com/c/card-1" {
		t.Fatalf("expected card URL copied, got %q", copied)
	}
	if status.Text != "Copied URL for \"Card 0-1\"" {
		t.Fatalf("unexpected status message: %q", status.Text)
	}
}

func TestCopySelectedCardURLFailure(t *testing.T) {
	prevClipboard := writeClipboard
	defer func() { writeClipboard = prevClipboard }()

	writeClipboard = func(text string) error {
		return errors.New("clipboard unavailable")
	}

	cfg := config.DefaultConfig()
	client := trello.NewClient("key", "token")
	app := NewApp(client, cfg)
	app.width = 120
	app.height = 30
	app.boardReady = true
	board := makeTestBoard(1)
	board.Lists[0].Cards[0].URL = "https://trello.com/c/card-1"
	app.board = NewBoardModel(board, cfg, 120, 26)

	_, cmd := app.Update(tea.KeyPressMsg{Code: -1, Text: "u"})
	if cmd == nil {
		t.Fatal("expected copy command")
	}
	msg := cmd()
	status, ok := msg.(StatusMsg)
	if !ok {
		t.Fatalf("expected StatusMsg, got %T", msg)
	}
	if status.Text != "Copy failed: clipboard unavailable" {
		t.Fatalf("unexpected status message: %q", status.Text)
	}
}

func TestMoveCardRightUsesNextVisibleColumn(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Views = []config.ViewConfig{{Title: "Working Cards", HideColumns: []string{"On Hold"}}}
	client := trello.NewClient("key", "token")
	app := NewApp(client, cfg)
	app.width = 90
	app.height = 30
	app.boardReady = true
	app.board = NewBoardModel(&trello.Board{ID: "b1", Name: "Test", Lists: []trello.List{
		{ID: "todo", Name: "To Do", Cards: []trello.Card{{ID: "c1", Name: "Card 1", Pos: 1, ListID: "todo"}}},
		{ID: "hold", Name: "On Hold"},
		{ID: "doing", Name: "Doing"},
	}}, cfg, 90, 23)
	app.board.SetHiddenColumns([]string{"On Hold"})

	model, _ := app.handleMoveCardRight()
	updated := model.(App)

	if updated.board.FocusedColumn() != 2 {
		t.Fatalf("expected focus on next visible column, got %d", updated.board.FocusedColumn())
	}
	if len(updated.board.columns[0].cards) != 0 {
		t.Fatal("expected card removed from source column")
	}
	if len(updated.board.columns[1].cards) != 0 {
		t.Fatal("expected hidden column to remain untouched")
	}
	if len(updated.board.columns[2].cards) != 1 || updated.board.columns[2].cards[0].ID != "c1" {
		t.Fatalf("expected card inserted into next visible column, got %#v", updated.board.columns[2].cards)
	}
}

func TestUserConfigHiddenViewHidesColumnsWhenSelected(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Views = []config.ViewConfig{
		{Title: "Assigned to Me", Filter: "member:@me", Key: "m"},
		{Title: "Everything"},
		{Title: "Hidden Views", HideColumns: []string{"In Review", "Complete"}},
	}
	client := trello.NewClient("key", "token")
	app := NewApp(client, cfg)
	app.width = 120
	app.height = 30

	board := &trello.Board{ID: "b1", Name: "Tuiello", Lists: []trello.List{
		{ID: "l1", Name: "Backlog", Cards: []trello.Card{{ID: "c1", Name: "Card 1", Pos: 1}}},
		{ID: "l2", Name: "In Review", Cards: []trello.Card{{ID: "c2", Name: "Card 2", Pos: 1}}},
		{ID: "l3", Name: "Complete", Cards: []trello.Card{{ID: "c3", Name: "Card 3", Pos: 1}}},
		{ID: "l4", Name: "Doing", Cards: []trello.Card{{ID: "c4", Name: "Card 4", Pos: 1}}},
	}}

	model, _ := app.Update(BoardFetchedMsg{Board: board})
	app = model.(App)
	if !strings.Contains(app.board.View(), "In Review") || !strings.Contains(app.board.View(), "Complete") {
		t.Fatal("expected columns to remain visible before selecting Hidden Views")
	}

	if ok := app.viewBar.SelectByKey("2"); !ok {
		t.Fatal("expected Hidden Views to be assigned shortcut 2")
	}
	app.applyActiveView()

	rendered := app.board.View()
	if strings.Contains(rendered, "In Review") || strings.Contains(rendered, "Complete") {
		t.Fatalf("expected selected hidden view to remove configured columns, got %q", rendered)
	}
	if !strings.Contains(rendered, "Backlog") || !strings.Contains(rendered, "Doing") {
		t.Fatalf("expected remaining visible columns to render, got %q", rendered)
	}
}
