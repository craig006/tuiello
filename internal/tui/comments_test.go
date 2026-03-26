package tui

import (
	"testing"

	"github.com/craig006/tuiello/internal/config"
	"github.com/craig006/tuiello/internal/trello"
)

func newTestCommentsList() CommentsList {
	cfg := config.DefaultConfig()
	km := NewKeyMap(cfg.Keybinding)
	theme := NewTheme(cfg.GUI.Theme)
	return NewCommentsList(theme, km)
}

func TestNewCommentsList(t *testing.T) {
	cl := newTestCommentsList()

	if cl.selectedIdx != 0 {
		t.Errorf("expected selectedIdx 0, got %d", cl.selectedIdx)
	}
	if cl.mode != CommentModeView {
		t.Errorf("expected mode CommentModeView, got %d", cl.mode)
	}
	if cl.editingIdx != -1 {
		t.Errorf("expected editingIdx -1, got %d", cl.editingIdx)
	}
	if len(cl.comments) != 0 {
		t.Errorf("expected 0 comments, got %d", len(cl.comments))
	}
	if len(cl.allMembers) != 0 {
		t.Errorf("expected 0 members, got %d", len(cl.allMembers))
	}
	if cl.focused {
		t.Error("expected focused to be false")
	}
	if cl.loading {
		t.Error("expected loading to be false")
	}
	if cl.width != 80 {
		t.Errorf("expected width 80, got %d", cl.width)
	}
	if cl.height != 20 {
		t.Errorf("expected height 20, got %d", cl.height)
	}
}

func TestSetComments(t *testing.T) {
	cl := newTestCommentsList()

	// Test setting comments
	comments := []trello.Comment{
		{ID: "comment1", Author: trello.Member{ID: "user1", FullName: "User One"}},
		{ID: "comment2", Author: trello.Member{ID: "user2", FullName: "User Two"}},
	}
	cl.SetComments(comments)

	if len(cl.comments) != 2 {
		t.Errorf("expected 2 comments, got %d", len(cl.comments))
	}
	if cl.selectedIdx != 0 {
		t.Errorf("expected selectedIdx 0, got %d", cl.selectedIdx)
	}

	// Test setting empty comments
	cl.SetComments([]trello.Comment{})
	if len(cl.comments) != 0 {
		t.Errorf("expected 0 comments, got %d", len(cl.comments))
	}
}

func TestSetMembers(t *testing.T) {
	cl := newTestCommentsList()

	members := []trello.Member{
		{ID: "member1", FullName: "Member One", Username: "memberone"},
		{ID: "member2", FullName: "Member Two", Username: "membertwo"},
		{ID: "member3", FullName: "Member Three", Username: "memberthree"},
	}
	cl.SetMembers(members)

	if len(cl.allMembers) != 3 {
		t.Errorf("expected 3 members, got %d", len(cl.allMembers))
	}
	if cl.allMembers[0].FullName != "Member One" {
		t.Errorf("expected first member 'Member One', got %q", cl.allMembers[0].FullName)
	}
}

func TestSetSize(t *testing.T) {
	cl := newTestCommentsList()

	cl.SetSize(120, 30)

	if cl.width != 120 {
		t.Errorf("expected width 120, got %d", cl.width)
	}
	if cl.height != 30 {
		t.Errorf("expected height 30, got %d", cl.height)
	}
	if cl.viewport.Width() != 120 {
		t.Errorf("expected viewport width 120, got %d", cl.viewport.Width())
	}
	if cl.viewport.Height() != 25 {
		t.Errorf("expected viewport height 25 (30-5), got %d", cl.viewport.Height())
	}
}

func TestSetFocus(t *testing.T) {
	cl := newTestCommentsList()

	if cl.focused {
		t.Error("expected focused to be false initially")
	}

	cl.SetFocus(true)
	if !cl.focused {
		t.Error("expected focused to be true after SetFocus(true)")
	}

	cl.SetFocus(false)
	if cl.focused {
		t.Error("expected focused to be false after SetFocus(false)")
	}
}
