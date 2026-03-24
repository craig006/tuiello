package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/craig006/tuillo/internal/config"
)

// ViewBar manages the views tab bar state.
type ViewBar struct {
	views  []config.ViewConfig
	keys   []string // assigned shortcut keys per view
	active int
}

// NewViewBar creates a view bar from the config views.
// Filters out views with empty titles.
func NewViewBar(views []config.ViewConfig) ViewBar {
	var filtered []config.ViewConfig
	for _, v := range views {
		if v.Title != "" {
			filtered = append(filtered, v)
		}
	}
	if len(filtered) == 0 {
		// Fallback to defaults
		filtered = []config.ViewConfig{
			{Title: "My Cards", Filter: "member:@me", Key: "m"},
			{Title: "All Cards"},
		}
	}
	return ViewBar{
		views: filtered,
		keys:  config.AssignViewKeys(filtered),
	}
}

// Active returns the index of the active view.
func (v *ViewBar) Active() int { return v.active }

// ActiveConfig returns the config of the active view.
func (v *ViewBar) ActiveConfig() config.ViewConfig {
	return v.views[v.active]
}

// Next cycles to the next view (wraps around).
func (v *ViewBar) Next() {
	v.active = (v.active + 1) % len(v.views)
}

// Prev cycles to the previous view (wraps around).
func (v *ViewBar) Prev() {
	v.active = (v.active - 1 + len(v.views)) % len(v.views)
}

// SelectByKey selects a view by its shortcut key. Returns true if found.
func (v *ViewBar) SelectByKey(key string) bool {
	for i, k := range v.keys {
		if k == key {
			v.active = i
			return true
		}
	}
	return false
}

// Keys returns the assigned shortcut keys.
func (v *ViewBar) Keys() []string { return v.keys }

// View renders the tab bar at the given width.
func (v ViewBar) View(width int) string {
	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(15))
	activeKeyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(7))
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))

	// Calculate max title length for truncation
	sepWidth := 5 // "  │  "
	totalSepWidth := sepWidth * (len(v.views) - 1)
	shortcutWidth := 4 // " ‹k›" per view
	availableForTitles := width - 2 - totalSepWidth - (shortcutWidth * len(v.views)) // 2 for padding
	maxTitleLen := availableForTitles / len(v.views)
	if maxTitleLen < 5 {
		maxTitleLen = 5
	}

	var parts []string
	for i, view := range v.views {
		title := view.Title
		if len([]rune(title)) > maxTitleLen {
			title = string([]rune(title)[:maxTitleLen-1]) + "…"
		}
		shortcut := " ‹" + v.keys[i] + "›"

		var tab string
		if i == v.active {
			tab = activeStyle.Render(title) + activeKeyStyle.Render(shortcut)
		} else {
			tab = inactiveStyle.Render(title + shortcut)
		}
		parts = append(parts, tab)
	}

	sep := sepStyle.Render("  │  ")
	content := strings.Join(parts, sep)

	bar := lipgloss.NewStyle().
		Width(width).
		Background(lipgloss.ANSIColor(0)).
		Render(" " + content)

	return bar
}
