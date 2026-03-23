// internal/tui/app_test.go
package tui

import (
	"testing"

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
