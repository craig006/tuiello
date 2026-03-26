package tui

import (
	"strings"
	"testing"

	"github.com/craig006/tuiello/internal/config"
	"github.com/craig006/tuiello/internal/trello"
)

func newTestDetail() DetailModel {
	cfg := config.DefaultConfig()
	km := NewKeyMap(cfg.Keybinding)
	theme := NewTheme(cfg.GUI.Theme)
	return NewDetailModel(km, theme, cfg.GUI.Padding)
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

func TestDetailModelInitializesCommentsList(t *testing.T) {
	d := newTestDetail()
	if d.commentsList == nil {
		t.Error("expected CommentsList to be initialized in NewDetailModel")
	}
}

func TestDetailModelDelegatesCommentMessages(t *testing.T) {
	d := newTestDetail()
	d.open = true
	d.focused = true
	d.tab = tabComments
	d.SetCard(trello.Card{ID: "card1"})

	// Create a comment
	testComment := trello.Comment{ID: "c1", Body: "Test comment"}

	// Simulate delegation by sending a message
	msg := CommentCreatedMsg{Comment: testComment}
	d2, _ := d.Update(msg)

	// After Update, d2.commentsList should have received the message
	// We can verify by checking if the comment was added to commentsList
	if len(d2.commentsList.comments) == 0 {
		t.Error("expected CommentsList to receive and handle CommentCreatedMsg")
	}
}

func TestDetailModelSetsFocusOnCommentsList(t *testing.T) {
	d := newTestDetail()
	d.SetCard(trello.Card{ID: "card1"})

	// Set focus on Comments tab
	d.tab = tabComments
	d.SetFocus(true)

	if !d.commentsList.focused {
		t.Error("expected CommentsList to be focused when Comments tab is active and detail is focused")
	}

	// Defocus
	d.SetFocus(false)
	if d.commentsList.focused {
		t.Error("expected CommentsList to be unfocused when detail is defocused")
	}

	// Set focus on different tab
	d.SetFocus(true)
	d.tab = tabOverview
	d.SetFocus(true)
	if d.commentsList.focused {
		t.Error("expected CommentsList to be unfocused when Comments tab is not active")
	}
}

func TestDetailModelRendersCommentsList(t *testing.T) {
	d := newTestDetail()
	d.open = true
	d.focused = true
	d.tab = tabComments
	d.SetSize(60, 30)
	d.SetCard(trello.Card{ID: "card1"})

	// Set some comments in CommentsList
	d.commentsList.SetComments([]trello.Comment{
		{ID: "c1", Body: "First comment", Author: trello.Member{FullName: "Alice"}},
	})

	view := d.View()
	if !strings.Contains(view, "Comments") {
		t.Error("expected view to contain 'Comments' tab label")
	}
	// The content should come from CommentsList rendering
	if !strings.Contains(view, "Alice") && !strings.Contains(view, "No comments") {
		t.Error("expected view to contain comments content from CommentsList")
	}
}

func TestDetailModelCommentsFocusOnlyWhenTabActive(t *testing.T) {
	d := newTestDetail()
	d.SetCard(trello.Card{ID: "card1"})

	// Initially on Overview tab
	d.tab = tabOverview
	d.SetFocus(true)

	if d.commentsList.focused {
		t.Error("expected CommentsList to not be focused when on Overview tab")
	}

	// Switch to Comments tab
	d.NextTab() // Now on tabComments (1)
	d.SetFocus(true)

	if !d.commentsList.focused {
		t.Error("expected CommentsList to be focused when on Comments tab")
	}

	// Switch to Checklist tab
	d.NextTab() // Now on tabChecklists (2)
	d.SetFocus(true)

	if d.commentsList.focused {
		t.Error("expected CommentsList to not be focused when on Checklist tab")
	}
}
