package tui

import (
	"testing"
)

func TestNewFocusManager(t *testing.T) {
	fm := NewFocusManager("board")

	if fm.FocusedSection() != "board" {
		t.Errorf("expected focused section 'board', got '%s'", fm.FocusedSection())
	}

	if fm.FocusedElement() != "" {
		t.Errorf("expected empty focused element, got '%s'", fm.FocusedElement())
	}

	if fm.IsModalActive() {
		t.Errorf("expected modal to be inactive")
	}
}

func TestSetFocusedSection(t *testing.T) {
	fm := NewFocusManager("board")
	fm.SetFocusedElement("board", "card1")

	if fm.FocusedElement() != "card1" {
		t.Errorf("expected focused element 'card1', got '%s'", fm.FocusedElement())
	}

	fm.SetFocusedSection("detail")

	if fm.FocusedSection() != "detail" {
		t.Errorf("expected focused section 'detail', got '%s'", fm.FocusedSection())
	}

	if fm.FocusedElement() != "" {
		t.Errorf("expected element focus to be cleared when section changes, got '%s'", fm.FocusedElement())
	}
}

func TestSetFocusedElementIgnoredWhenNotInFocusedSection(t *testing.T) {
	fm := NewFocusManager("board")

	ok := fm.SetFocusedElement("detail", "comment1")

	if ok {
		t.Errorf("expected SetFocusedElement to return false when section doesn't match")
	}

	if fm.FocusedElement() != "" {
		t.Errorf("expected element to remain empty, got '%s'", fm.FocusedElement())
	}
}

func TestSetFocusedElementSucceedsWhenInFocusedSection(t *testing.T) {
	fm := NewFocusManager("board")

	ok := fm.SetFocusedElement("board", "card1")

	if !ok {
		t.Errorf("expected SetFocusedElement to return true when section matches")
	}

	if fm.FocusedElement() != "card1" {
		t.Errorf("expected focused element 'card1', got '%s'", fm.FocusedElement())
	}
}

func TestModalSuspendAndRestore(t *testing.T) {
	fm := NewFocusManager("board")
	fm.SetFocusedElement("board", "card1")

	fm.OpenModal()

	if !fm.IsModalActive() {
		t.Errorf("expected modal to be active after OpenModal")
	}

	fm.SetFocusedSection("detail")

	if fm.FocusedSection() != "board" {
		t.Errorf("expected section to remain 'board' while modal is active, got '%s'", fm.FocusedSection())
	}

	fm.CloseModal()

	if fm.IsModalActive() {
		t.Errorf("expected modal to be inactive after CloseModal")
	}

	if fm.FocusedSection() != "board" {
		t.Errorf("expected section to still be 'board', got '%s'", fm.FocusedSection())
	}

	if fm.FocusedElement() != "card1" {
		t.Errorf("expected element to still be 'card1', got '%s'", fm.FocusedElement())
	}
}

func TestSetFocusedSectionIgnoredWhenModalActive(t *testing.T) {
	fm := NewFocusManager("board")
	fm.OpenModal()

	fm.SetFocusedSection("detail")

	if fm.FocusedSection() != "board" {
		t.Errorf("expected section to remain 'board' while modal is active, got '%s'", fm.FocusedSection())
	}
}
