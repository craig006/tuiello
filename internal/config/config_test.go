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
	os.WriteFile(filepath.Join(projectDir, ".tuillo.yml"), []byte(`
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
