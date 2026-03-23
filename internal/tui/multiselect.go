package tui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// MultiSelectItem represents a single item in the multiselect list.
type MultiSelectItem struct {
	Label   string
	Value   string
	Checked bool
	Color   color.Color // optional, for label indicators
}

// MultiSelectModel is a simple checkbox list for modal overlays.
type MultiSelectModel struct {
	title  string
	items  []MultiSelectItem
	cursor int
}

// NewMultiSelectModel creates a new multiselect model.
func NewMultiSelectModel(title string, items []MultiSelectItem) MultiSelectModel {
	return MultiSelectModel{
		title: title,
		items: items,
	}
}

// Toggle toggles the checked state of the item at the cursor.
func (m *MultiSelectModel) Toggle() {
	if len(m.items) > 0 {
		m.items[m.cursor].Checked = !m.items[m.cursor].Checked
	}
}

// MoveDown moves the cursor down.
func (m *MultiSelectModel) MoveDown() {
	if m.cursor < len(m.items)-1 {
		m.cursor++
	}
}

// MoveUp moves the cursor up.
func (m *MultiSelectModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

// Selected returns the values of all checked items.
func (m MultiSelectModel) Selected() []string {
	var selected []string
	for _, item := range m.items {
		if item.Checked {
			selected = append(selected, item.Value)
		}
	}
	return selected
}

// View renders the multiselect list.
func (m MultiSelectModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(15))
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(7))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(12))

	var lines []string
	lines = append(lines, titleStyle.Render(m.title))
	lines = append(lines, "")

	for i, item := range m.items {
		checkbox := "[ ] "
		if item.Checked {
			checkbox = "[x] "
		}

		label := item.Label
		if item.Color != nil {
			indicator := lipgloss.NewStyle().Foreground(item.Color).Render("⏺ ")
			label = indicator + label
		}

		var line string
		if i == m.cursor {
			line = cursorStyle.Render("> "+checkbox) + normalStyle.Render(label)
		} else {
			line = dimStyle.Render("  "+checkbox) + normalStyle.Render(label)
		}
		lines = append(lines, line)
	}

	lines = append(lines, "")
	lines = append(lines, dimStyle.Render(fmt.Sprintf("  space: toggle • enter/esc: close • %d selected", len(m.Selected()))))

	return strings.Join(lines, "\n")
}
