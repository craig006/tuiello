package tui

import (
	"testing"
	"github.com/craig006/tuiello/internal/trello"
)

func TestNewColumn(t *testing.T) {
	cards := []trello.Card{
		{ID: "c1", Name: "Card 1", Pos: 1.0},
		{ID: "c2", Name: "Card 2", Pos: 2.0},
	}
	col := NewColumn(trello.List{ID: "list1", Name: "Backlog", Cards: cards}, 30, 20, false)
	if col.Title() != "Backlog" { t.Errorf("expected title 'Backlog', got %q", col.Title()) }
	if col.CardCount() != 2 { t.Errorf("expected 2 cards, got %d", col.CardCount()) }
}

func TestColumnSelectedCard(t *testing.T) {
	cards := []trello.Card{
		{ID: "c1", Name: "Card 1", Pos: 1.0},
		{ID: "c2", Name: "Card 2", Pos: 2.0},
	}
	col := NewColumn(trello.List{ID: "list1", Name: "Backlog", Cards: cards}, 30, 20, false)
	card, ok := col.SelectedCard()
	if !ok { t.Fatal("expected a selected card") }
	if card.ID != "c1" { t.Errorf("expected 'c1', got %q", card.ID) }
}

func TestColumnEmptyList(t *testing.T) {
	col := NewColumn(trello.List{ID: "list1", Name: "Empty"}, 30, 20, false)
	_, ok := col.SelectedCard()
	if ok { t.Error("expected no selected card in empty list") }
	if col.CardCount() != 0 { t.Errorf("expected 0 cards, got %d", col.CardCount()) }
}
