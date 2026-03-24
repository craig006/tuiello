// internal/tui/board_test.go
package tui

import (
	"fmt"
	"testing"

	"github.com/craig006/tuiello/internal/config"
	"github.com/craig006/tuiello/internal/trello"
)

func makeTestBoard(numLists int) *trello.Board {
	board := &trello.Board{ID: "b1", Name: "Test"}
	for i := 0; i < numLists; i++ {
		list := trello.List{
			ID:   fmt.Sprintf("list%d", i),
			Name: fmt.Sprintf("List %d", i),
			Pos:  float64(i),
			Cards: []trello.Card{
				{ID: fmt.Sprintf("c%d-1", i), Name: fmt.Sprintf("Card %d-1", i), Pos: 1.0, ListID: fmt.Sprintf("list%d", i)},
				{ID: fmt.Sprintf("c%d-2", i), Name: fmt.Sprintf("Card %d-2", i), Pos: 2.0, ListID: fmt.Sprintf("list%d", i)},
			},
		}
		board.Lists = append(board.Lists, list)
	}
	return board
}

func TestWindowStartMiddle(t *testing.T) {
	board := makeTestBoard(5)
	cfg := config.DefaultConfig()
	b := NewBoardModel(board, cfg, 90, 30)

	start, end := b.VisibleRange()
	if start != 0 || end != 3 {
		t.Errorf("expected range [0,3), got [%d,%d)", start, end)
	}
}

func TestWindowNavigateRight(t *testing.T) {
	board := makeTestBoard(5)
	cfg := config.DefaultConfig()
	b := NewBoardModel(board, cfg, 90, 30)

	b.FocusRight()
	start, end := b.VisibleRange()
	if start != 0 || end != 3 {
		t.Errorf("expected range [0,3), got [%d,%d)", start, end)
	}

	b.FocusRight()
	start, end = b.VisibleRange()
	if start != 1 || end != 4 {
		t.Errorf("expected range [1,4), got [%d,%d)", start, end)
	}
}

func TestWindowLastColumn(t *testing.T) {
	board := makeTestBoard(5)
	cfg := config.DefaultConfig()
	b := NewBoardModel(board, cfg, 90, 30)

	for i := 0; i < 4; i++ {
		b.FocusRight()
	}

	if b.FocusedColumn() != 4 {
		t.Errorf("expected focused column 4, got %d", b.FocusedColumn())
	}

	start, end := b.VisibleRange()
	if start != 2 || end != 5 {
		t.Errorf("expected range [2,5), got [%d,%d)", start, end)
	}
}

func TestWindowTwoColumns(t *testing.T) {
	board := makeTestBoard(2)
	cfg := config.DefaultConfig()
	b := NewBoardModel(board, cfg, 90, 30)

	start, end := b.VisibleRange()
	if start != 0 || end != 2 {
		t.Errorf("expected range [0,2), got [%d,%d)", start, end)
	}
}

func TestWindowOneColumn(t *testing.T) {
	board := makeTestBoard(1)
	cfg := config.DefaultConfig()
	b := NewBoardModel(board, cfg, 90, 30)

	start, end := b.VisibleRange()
	if start != 0 || end != 1 {
		t.Errorf("expected range [0,1), got [%d,%d)", start, end)
	}
}

func TestFocusLeftBoundary(t *testing.T) {
	board := makeTestBoard(5)
	cfg := config.DefaultConfig()
	b := NewBoardModel(board, cfg, 90, 30)

	b.FocusLeft()
	if b.FocusedColumn() != 0 {
		t.Errorf("expected focused column 0, got %d", b.FocusedColumn())
	}
}

func TestFocusRightBoundary(t *testing.T) {
	board := makeTestBoard(3)
	cfg := config.DefaultConfig()
	b := NewBoardModel(board, cfg, 90, 30)

	b.FocusRight()
	b.FocusRight()
	b.FocusRight()
	if b.FocusedColumn() != 2 {
		t.Errorf("expected focused column 2, got %d", b.FocusedColumn())
	}
}

func TestPositionIndicator(t *testing.T) {
	board := makeTestBoard(5)
	cfg := config.DefaultConfig()
	b := NewBoardModel(board, cfg, 90, 30)

	indicator := b.PositionIndicator()
	if indicator != "[1/5]" {
		t.Errorf("expected '[1/5]', got %q", indicator)
	}

	b.FocusRight()
	b.FocusRight()
	indicator = b.PositionIndicator()
	if indicator != "[3/5]" {
		t.Errorf("expected '[3/5]', got %q", indicator)
	}
}

func TestCalcNewPosEmpty(t *testing.T) {
	pos := CalcNewPos(nil, 0)
	if pos != 65536.0 {
		t.Errorf("expected 65536.0, got %f", pos)
	}
}

func TestCalcNewPosTop(t *testing.T) {
	cards := []trello.Card{{Pos: 100.0}, {Pos: 200.0}}
	pos := CalcNewPos(cards, 0)
	if pos != 50.0 {
		t.Errorf("expected 50.0, got %f", pos)
	}
}

func TestCalcNewPosBottom(t *testing.T) {
	cards := []trello.Card{{Pos: 100.0}, {Pos: 200.0}}
	pos := CalcNewPos(cards, 2)
	if pos != 65736.0 {
		t.Errorf("expected 65736.0, got %f", pos)
	}
}

func TestCalcNewPosMiddle(t *testing.T) {
	cards := []trello.Card{{Pos: 100.0}, {Pos: 200.0}, {Pos: 300.0}}
	pos := CalcNewPos(cards, 1)
	if pos != 150.0 {
		t.Errorf("expected 150.0, got %f", pos)
	}
}

func TestMoveCardUpTwice(t *testing.T) {
	board := makeTestBoard(3)
	// Give list0 three cards
	board.Lists[0].Cards = []trello.Card{
		{ID: "c0", Name: "Card 0", Pos: 1.0, ListID: "list0"},
		{ID: "c1", Name: "Card 1", Pos: 2.0, ListID: "list0"},
		{ID: "c2", Name: "Card 2", Pos: 3.0, ListID: "list0"},
	}
	cfg := config.DefaultConfig()
	b := NewBoardModel(board, cfg, 90, 30)

	// Select card at index 2
	b.columns[0].Select(2)
	if b.columns[0].SelectedIndex() != 2 {
		t.Fatalf("expected selected index 2, got %d", b.columns[0].SelectedIndex())
	}

	// First move up: swap index 2 and 1, select 1
	cards := b.columns[0].cards
	cards[2], cards[1] = cards[1], cards[2]
	b.rebuildColumnItems(0)
	b.columns[0].Select(1)

	sel := b.columns[0].SelectedIndex()
	if sel != 1 {
		t.Fatalf("after first move: expected selected index 1, got %d", sel)
	}
	card, ok := b.columns[0].SelectedCard()
	if !ok || card.ID != "c2" {
		t.Fatalf("after first move: expected card c2, got %v", card)
	}

	// Second move up: swap index 1 and 0, select 0
	cards = b.columns[0].cards
	cards[1], cards[0] = cards[0], cards[1]
	b.rebuildColumnItems(0)
	b.columns[0].Select(0)

	sel = b.columns[0].SelectedIndex()
	if sel != 0 {
		t.Fatalf("after second move: expected selected index 0, got %d", sel)
	}
	card, ok = b.columns[0].SelectedCard()
	if !ok || card.ID != "c2" {
		t.Fatalf("after second move: expected card c2, got %v (ok=%v)", card, ok)
	}
}
