package tui

import (
	"strings"
	"testing"

	"github.com/craig006/tuillo/internal/config"
)

func TestNewViewBarDefaultViews(t *testing.T) {
	views := []config.ViewConfig{
		{Title: "My Cards", Filter: "member:@me", Key: "m"},
		{Title: "All Cards"},
	}
	vb := NewViewBar(views)
	if vb.Active() != 0 {
		t.Errorf("expected active view 0, got %d", vb.Active())
	}
	if len(vb.views) != 2 {
		t.Errorf("expected 2 views, got %d", len(vb.views))
	}
}

func TestViewBarCycleForward(t *testing.T) {
	views := []config.ViewConfig{
		{Title: "A"},
		{Title: "B"},
		{Title: "C"},
	}
	vb := NewViewBar(views)
	vb.Next()
	if vb.Active() != 1 {
		t.Errorf("expected 1, got %d", vb.Active())
	}
	vb.Next()
	vb.Next() // wraps
	if vb.Active() != 0 {
		t.Errorf("expected 0 after wrap, got %d", vb.Active())
	}
}

func TestViewBarCycleBackward(t *testing.T) {
	views := []config.ViewConfig{
		{Title: "A"},
		{Title: "B"},
	}
	vb := NewViewBar(views)
	vb.Prev() // wraps to last
	if vb.Active() != 1 {
		t.Errorf("expected 1 after wrap, got %d", vb.Active())
	}
}

func TestViewBarSelectByKey(t *testing.T) {
	views := []config.ViewConfig{
		{Title: "A", Key: "m"},
		{Title: "B"},
		{Title: "C"},
	}
	vb := NewViewBar(views)
	if !vb.SelectByKey("1") {
		t.Error("expected to select view with key '1'")
	}
	if vb.Active() != 1 {
		t.Errorf("expected 1, got %d", vb.Active())
	}
	if vb.SelectByKey("z") {
		t.Error("expected false for unknown key")
	}
}

func TestViewBarActiveConfig(t *testing.T) {
	showDetail := true
	views := []config.ViewConfig{
		{Title: "A", Filter: "member:craig", ShowDetailPanel: &showDetail},
		{Title: "B"},
	}
	vb := NewViewBar(views)
	cfg := vb.ActiveConfig()
	if cfg.Filter != "member:craig" {
		t.Errorf("expected filter, got %q", cfg.Filter)
	}
}

func TestViewBarRender(t *testing.T) {
	views := []config.ViewConfig{
		{Title: "My Cards", Key: "m"},
		{Title: "All Cards"},
	}
	vb := NewViewBar(views)
	rendered := vb.View(80)
	if !strings.Contains(rendered, "My Cards") {
		t.Error("expected 'My Cards' in rendered output")
	}
	if !strings.Contains(rendered, "All Cards") {
		t.Error("expected 'All Cards' in rendered output")
	}
}

func TestViewBarEmptyViewsFallsBackToDefaults(t *testing.T) {
	vb := NewViewBar(nil)
	if len(vb.views) != 2 {
		t.Fatalf("expected 2 default views, got %d", len(vb.views))
	}
	if vb.views[0].Title != "My Cards" {
		t.Errorf("expected 'My Cards', got %q", vb.views[0].Title)
	}
}

func TestViewBarFiltersEmptyTitles(t *testing.T) {
	views := []config.ViewConfig{
		{Title: "A"},
		{Title: ""},
		{Title: "B"},
	}
	vb := NewViewBar(views)
	if len(vb.views) != 2 {
		t.Fatalf("expected 2 views (empty title filtered), got %d", len(vb.views))
	}
}
