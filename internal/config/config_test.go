// internal/config/config_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.GUI.ColumnWidth != 30 {
		t.Errorf("expected columnWidth 30, got %d", cfg.GUI.ColumnWidth)
	}
	if cfg.GUI.ShowCardLabels != true {
		t.Error("expected showCardLabels true")
	}
	if cfg.Keybinding.Universal.Quit != "q" {
		t.Errorf("expected quit key 'q', got %q", cfg.Keybinding.Universal.Quit)
	}
	if cfg.Keybinding.Board.MoveLeft != "h" {
		t.Errorf("expected moveLeft 'h', got %q", cfg.Keybinding.Board.MoveLeft)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")
	err := os.WriteFile(cfgPath, []byte(`
board:
  id: "abc123"
gui:
  columnWidth: 40
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Board.ID != "abc123" {
		t.Errorf("expected board id 'abc123', got %q", cfg.Board.ID)
	}
	if cfg.GUI.ColumnWidth != 40 {
		t.Errorf("expected columnWidth 40, got %d", cfg.GUI.ColumnWidth)
	}
	// defaults still apply for unset fields
	if cfg.Keybinding.Universal.Quit != "q" {
		t.Errorf("expected quit key 'q', got %q", cfg.Keybinding.Universal.Quit)
	}
}

func TestCascadeProjectLocal(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()

	// global config sets board id
	os.WriteFile(filepath.Join(globalDir, "config.yml"), []byte(`
board:
  id: "global-board"
gui:
  columnWidth: 25
`), 0644)

	// project-local overrides board id
	os.WriteFile(filepath.Join(projectDir, ".tuiello.yml"), []byte(`
board:
  id: "project-board"
`), 0644)

	cfg, err := Load(globalDir, projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Board.ID != "project-board" {
		t.Errorf("expected project-board, got %q", cfg.Board.ID)
	}
	// global columnWidth preserved since project didn't override
	if cfg.GUI.ColumnWidth != 25 {
		t.Errorf("expected columnWidth 25, got %d", cfg.GUI.ColumnWidth)
	}
}

func TestDefaultDetailKeys(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Keybinding.Detail.Toggle != "d" {
		t.Errorf("expected detail toggle 'd', got %q", cfg.Keybinding.Detail.Toggle)
	}
	if cfg.Keybinding.Detail.TabPrev != "[" {
		t.Errorf("expected detail tabPrev '[', got %q", cfg.Keybinding.Detail.TabPrev)
	}
	if cfg.Keybinding.Detail.TabNext != "]" {
		t.Errorf("expected detail tabNext ']', got %q", cfg.Keybinding.Detail.TabNext)
	}
	if cfg.Keybinding.Detail.ScrollDown != "ctrl+j" {
		t.Errorf("expected detail scrollDown 'ctrl+j', got %q", cfg.Keybinding.Detail.ScrollDown)
	}
	if cfg.Keybinding.Detail.ScrollUp != "ctrl+k" {
		t.Errorf("expected detail scrollUp 'ctrl+k', got %q", cfg.Keybinding.Detail.ScrollUp)
	}
}

func TestDefaultConfigHasDefaultViews(t *testing.T) {
	cfg := DefaultConfig()
	if len(cfg.Views) != 2 {
		t.Fatalf("expected 2 default views, got %d", len(cfg.Views))
	}
	if cfg.Views[0].Title != "My Cards" {
		t.Errorf("expected first view 'My Cards', got %q", cfg.Views[0].Title)
	}
	if cfg.Views[0].Filter != "member:@me" {
		t.Errorf("expected first view filter 'member:@me', got %q", cfg.Views[0].Filter)
	}
	if cfg.Views[0].Key != "m" {
		t.Errorf("expected first view key 'm', got %q", cfg.Views[0].Key)
	}
	if cfg.Views[1].Title != "All Cards" {
		t.Errorf("expected second view 'All Cards', got %q", cfg.Views[1].Title)
	}
}

func TestDefaultConfigHasViewKeys(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Keybinding.Views.NextView != "v" {
		t.Errorf("expected nextView 'v', got %q", cfg.Keybinding.Views.NextView)
	}
	if cfg.Keybinding.Views.PrevView != "V" {
		t.Errorf("expected prevView 'V', got %q", cfg.Keybinding.Views.PrevView)
	}
}

func TestAssignViewKeys(t *testing.T) {
	views := []ViewConfig{
		{Title: "My Cards", Key: "m"},
		{Title: "Mobile Cards"},
		{Title: "All Cards"},
	}
	keys := AssignViewKeys(views)
	if keys[0] != "m" {
		t.Errorf("expected 'm', got %q", keys[0])
	}
	if keys[1] != "1" {
		t.Errorf("expected '1', got %q", keys[1])
	}
	if keys[2] != "2" {
		t.Errorf("expected '2', got %q", keys[2])
	}
}

func TestAssignViewKeysSkipsUsedNumbers(t *testing.T) {
	views := []ViewConfig{
		{Title: "A", Key: "1"},
		{Title: "B"},
		{Title: "C"},
	}
	keys := AssignViewKeys(views)
	if keys[0] != "1" {
		t.Errorf("expected '1', got %q", keys[0])
	}
	if keys[1] != "2" {
		t.Errorf("expected '2', got %q", keys[1])
	}
	if keys[2] != "3" {
		t.Errorf("expected '3', got %q", keys[2])
	}
}

func TestAssignViewKeysDuplicateCustomKey(t *testing.T) {
	views := []ViewConfig{
		{Title: "A", Key: "m"},
		{Title: "B", Key: "m"},
	}
	keys := AssignViewKeys(views)
	// First keeps the key, second duplicate gets auto-assigned
	if keys[0] != "m" {
		t.Errorf("expected 'm', got %q", keys[0])
	}
	if keys[1] != "1" {
		t.Errorf("expected '1' (duplicate overridden), got %q", keys[1])
	}
}

func TestAssignViewKeysEmptyViews(t *testing.T) {
	keys := AssignViewKeys(nil)
	if len(keys) != 0 {
		t.Errorf("expected empty keys, got %v", keys)
	}
}
