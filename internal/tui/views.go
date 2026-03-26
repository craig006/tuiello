package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/craig006/tuiello/internal/config"
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
func (v ViewBar) View(width int, boardName string, padding int) string {
	bg := lipgloss.Color("236")
	activeStyle := lipgloss.NewStyle().Bold(true).Underline(true).Foreground(lipgloss.ANSIColor(4)).Background(bg)
	activeKeyStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(6)).Background(bg)
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(15)).Background(bg)
	inactiveKeyStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(6)).Background(bg)
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Background(bg)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Background(bg)
	valueStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(15)).Background(bg)
	dividerStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Background(bg)
	appNameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(4)).Background(bg)
	appVerStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Background(bg)

	// Build the left side: " Board: <name> | Views: <tabs> "
	boardSection := labelStyle.Render("Board: ") + valueStyle.Render(boardName)
	divider := dividerStyle.Render(" | ")

	var viewParts []string
	for i, view := range v.views {
		title := view.Title
		shortcut := " [" + v.keys[i] + "]"

		var tab string
		if i == v.active {
			tab = activeStyle.Render(title) + activeKeyStyle.Render(shortcut)
		} else {
			tab = inactiveStyle.Render(title) + inactiveKeyStyle.Render(shortcut)
		}
		viewParts = append(viewParts, tab)
	}

	viewSep := sepStyle.Render(" • ")
	viewsSection := labelStyle.Render("Views: ") + strings.Join(viewParts, viewSep)

	leftContent := boardSection + divider + viewsSection

	// Build the right side: app name + version
	appBrand := appNameStyle.Render("tuiello") + appVerStyle.Render(" "+Version)

	// Calculate gap between left and right
	bgSpace := lipgloss.NewStyle().Background(bg)
	leftWidth := lipgloss.Width(leftContent)
	rightWidth := lipgloss.Width(appBrand)
	rightPad := 1 // space after version number
	innerWidth := width - padding - rightPad
	gap := innerWidth - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}

	content := leftContent + bgSpace.Render(strings.Repeat(" ", gap)) + appBrand

	bar := lipgloss.NewStyle().
		Width(width).
		Padding(1, rightPad, 1, padding). // top 1, right 1, bottom 1, left padding
		Background(bg).
		Render(content)

	return bar
}
