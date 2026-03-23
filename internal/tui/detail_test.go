package tui

import (
	"strings"
	"testing"

	"github.com/craig006/tuillo/internal/config"
	"github.com/craig006/tuillo/internal/trello"
)

func newTestDetail() DetailModel {
	cfg := config.DefaultConfig()
	km := NewKeyMap(cfg.Keybinding)
	theme := NewTheme(cfg.GUI.Theme)
	return NewDetailModel(km, theme)
}

func TestDetailToggle(t *testing.T) {
	d := newTestDetail()
	if d.open {
		t.Error("expected panel to start closed")
	}
	d.Toggle()
	if !d.open {
		t.Error("expected panel to be open after toggle")
	}
	d.Toggle()
	if d.open {
		t.Error("expected panel to be closed after second toggle")
	}
}

func TestDetailTabCycling(t *testing.T) {
	d := newTestDetail()
	d.open = true
	if d.tab != 0 {
		t.Errorf("expected tab 0, got %d", d.tab)
	}
	d.NextTab()
	if d.tab != 1 {
		t.Errorf("expected tab 1, got %d", d.tab)
	}
	d.NextTab()
	if d.tab != 2 {
		t.Errorf("expected tab 2, got %d", d.tab)
	}
	d.NextTab()
	if d.tab != 0 {
		t.Errorf("expected tab to wrap to 0, got %d", d.tab)
	}
	d.PrevTab()
	if d.tab != 2 {
		t.Errorf("expected tab to wrap to 2, got %d", d.tab)
	}
}

func TestDetailSetCardClearsCache(t *testing.T) {
	d := newTestDetail()
	d.open = true
	d.comments = []trello.Comment{{ID: "old"}}
	d.commentsLoaded = true
	d.checklists = []trello.Checklist{{ID: "old"}}
	d.checklistsLoaded = true

	card := trello.Card{ID: "new-card", Name: "New Card"}
	d.SetCard(card)

	if d.cardID != "new-card" {
		t.Errorf("expected cardID 'new-card', got %q", d.cardID)
	}
	if d.commentsLoaded {
		t.Error("expected commentsLoaded to be false")
	}
	if d.checklistsLoaded {
		t.Error("expected checklistsLoaded to be false")
	}
	if len(d.comments) != 0 {
		t.Error("expected comments to be cleared")
	}
	if len(d.checklists) != 0 {
		t.Error("expected checklists to be cleared")
	}
}

func TestDetailOverviewView(t *testing.T) {
	d := newTestDetail()
	d.open = true
	d.SetSize(40, 20)
	card := trello.Card{
		ID:          "card1",
		Name:        "Test Card",
		Description: "A description",
		Labels:      []trello.Label{{Name: "Bug", Color: "red"}},
		Members:     []trello.Member{{FullName: "Craig Thomas"}},
	}
	d.SetCard(card)

	view := d.View()
	if !strings.Contains(view, "Test Card") {
		t.Error("expected view to contain card title")
	}
	if !strings.Contains(view, "Overview") {
		t.Error("expected view to contain 'Overview' tab")
	}
}

func TestDetailStaleResponseIgnored(t *testing.T) {
	d := newTestDetail()
	d.open = true
	d.SetCard(trello.Card{ID: "current"})

	// Simulate a stale comments response for a different card
	msg := CardCommentsMsg{CardID: "old-card", Comments: []trello.Comment{{ID: "c1", Body: "stale"}}}
	d.HandleCommentsMsg(msg)

	if d.commentsLoaded {
		t.Error("expected stale response to be ignored")
	}

	// Now a matching response
	msg = CardCommentsMsg{CardID: "current", Comments: []trello.Comment{{ID: "c2", Body: "fresh"}}}
	d.HandleCommentsMsg(msg)

	if !d.commentsLoaded {
		t.Error("expected matching response to be accepted")
	}
	if len(d.comments) != 1 || d.comments[0].Body != "fresh" {
		t.Error("expected comments to contain the fresh comment")
	}
}
