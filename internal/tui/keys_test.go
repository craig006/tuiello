package tui

import (
	"github.com/craig006/tuiello/internal/config"
	"testing"
)

func TestDefaultKeyMap(t *testing.T) {
	cfg := config.DefaultConfig()
	km := NewKeyMap(cfg.Keybinding)
	if km.Quit.Keys()[0] != "q" {
		t.Errorf("expected quit key 'q', got %q", km.Quit.Keys()[0])
	}
	if km.MoveLeft.Keys()[0] != "h" {
		t.Errorf("expected moveLeft 'h', got %q", km.MoveLeft.Keys()[0])
	}
	if km.MoveCardLeft.Keys()[0] != "H" {
		t.Errorf("expected moveCardLeft 'H', got %q", km.MoveCardLeft.Keys()[0])
	}
	if km.OpenCard.Keys()[0] != "o" {
		t.Errorf("expected openCard 'o', got %q", km.OpenCard.Keys()[0])
	}
	if km.CopyCardURL.Keys()[0] != "u" {
		t.Errorf("expected copyCardURL 'u', got %q", km.CopyCardURL.Keys()[0])
	}
}

func TestCustomKeyMap(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Keybinding.Board.MoveLeft = "a"
	cfg.Keybinding.Board.OpenCard = "O"
	cfg.Keybinding.Universal.Quit = "Q"
	km := NewKeyMap(cfg.Keybinding)
	if km.MoveLeft.Keys()[0] != "a" {
		t.Errorf("expected moveLeft 'a', got %q", km.MoveLeft.Keys()[0])
	}
	if km.OpenCard.Keys()[0] != "O" {
		t.Errorf("expected openCard 'O', got %q", km.OpenCard.Keys()[0])
	}
	if km.Quit.Keys()[0] != "Q" {
		t.Errorf("expected quit 'Q', got %q", km.Quit.Keys()[0])
	}
}
