package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/craig006/tuillo/internal/config"
)

// Version is set at build time via ldflags.
var Version = "dev"

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

// View renders the tab bar at the given width with a board name prefix and app branding.
func (v ViewBar) View(width int, boardName string) string {
	activeStyle := lipgloss.NewStyle().Bold(true).Underline(true).Foreground(lipgloss.ANSIColor(4))
	activeKeyStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(7))
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))
	valueStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(15))
	dividerStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))
	appNameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(4))
	appVerStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))

	// Build the left side: " Board: <name> | Views: <tabs> "
	boardSection := labelStyle.Render("Board: ") + valueStyle.Render(boardName)
	divider := dividerStyle.Render(" | ")

	var viewParts []string
	for i, view := range v.views {
		title := view.Title
		shortcut := " ‹" + v.keys[i] + "›"

		var tab string
		if i == v.active {
			tab = activeStyle.Render(title) + activeKeyStyle.Render(shortcut)
		} else {
			tab = inactiveStyle.Render(title + shortcut)
		}
		viewParts = append(viewParts, tab)
	}

	viewSep := sepStyle.Render(" • ")
	viewsSection := labelStyle.Render("Views: ") + strings.Join(viewParts, viewSep)

	leftContent := boardSection + divider + viewsSection

	// Build the right side: app name + version
	appBrand := appNameStyle.Render("tuillo") + " " + appVerStyle.Render(Version)

	// Calculate padding between left and right
	leftWidth := lipgloss.Width(leftContent)
	rightWidth := lipgloss.Width(appBrand)
	innerWidth := width - 2 // 2 for border sides
	padding := innerWidth - leftWidth - rightWidth - 2 // 2 for outer spacing
	if padding < 1 {
		padding = 1
	}

	content := leftContent + strings.Repeat(" ", padding) + appBrand

	bar := lipgloss.NewStyle().
		Width(width).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.ANSIColor(8)).
		Render(content)

	return bar
}
