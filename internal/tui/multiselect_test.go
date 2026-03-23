package tui

import "testing"

func TestMultiSelectToggle(t *testing.T) {
	items := []MultiSelectItem{
		{Label: "Alice", Value: "alice"},
		{Label: "Bob", Value: "bob"},
	}
	m := NewMultiSelectModel("Members", items)
	// Toggle first item
	m.Toggle()
	selected := m.Selected()
	if len(selected) != 1 || selected[0] != "alice" {
		t.Errorf("expected [alice], got %v", selected)
	}
	// Toggle again to deselect
	m.Toggle()
	selected = m.Selected()
	if len(selected) != 0 {
		t.Errorf("expected empty, got %v", selected)
	}
}

func TestMultiSelectNavigation(t *testing.T) {
	items := []MultiSelectItem{
		{Label: "Alice", Value: "alice"},
		{Label: "Bob", Value: "bob"},
		{Label: "Charlie", Value: "charlie"},
	}
	m := NewMultiSelectModel("Members", items)
	m.MoveDown()
	m.Toggle()
	selected := m.Selected()
	if len(selected) != 1 || selected[0] != "bob" {
		t.Errorf("expected [bob], got %v", selected)
	}
}

func TestMultiSelectPreselected(t *testing.T) {
	items := []MultiSelectItem{
		{Label: "Alice", Value: "alice"},
		{Label: "Bob", Value: "bob", Checked: true},
	}
	m := NewMultiSelectModel("Members", items)
	selected := m.Selected()
	if len(selected) != 1 || selected[0] != "bob" {
		t.Errorf("expected [bob], got %v", selected)
	}
}
