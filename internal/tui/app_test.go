// internal/tui/app_test.go
package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/craig006/tuillo/internal/config"
	"github.com/craig006/tuillo/internal/trello"
)

func TestAppInitNoBoard(t *testing.T) {
	cfg := config.DefaultConfig()
	client := trello.NewClient("key", "token")
	app := NewApp(client, cfg)

	cmd := app.Init()
	if cmd == nil {
		t.Fatal("expected a command for missing board error")
	}

	msg := cmd()
	if _, ok := msg.(BoardFetchErrMsg); !ok {
		t.Errorf("expected BoardFetchErrMsg, got %T", msg)
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
