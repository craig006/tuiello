package tui

import (
	"charm.land/lipgloss/v2"
	"github.com/craig006/tuiello/internal/config"
)

type Theme struct {
	ActiveBorder, InactiveBorder, SelectedCard, ColumnTitle lipgloss.Style
}

func NewTheme(cfg config.ThemeConfig) Theme {
	return Theme{
		ActiveBorder:   buildStyle(cfg.ActiveBorderColor),
		InactiveBorder: buildStyle(cfg.InactiveBorderColor),
		SelectedCard:   buildStyle(cfg.SelectedCardColor),
		ColumnTitle:    buildStyle(cfg.ColumnTitleColor),
	}
}

func buildStyle(attrs []string) lipgloss.Style {
	s := lipgloss.NewStyle()
	if len(attrs) == 0 { return s }
	s = s.Foreground(lipgloss.Color(attrs[0]))
	for _, attr := range attrs[1:] {
		switch attr {
		case "bold": s = s.Bold(true)
		case "italic": s = s.Italic(true)
		case "underline": s = s.Underline(true)
		}
	}
	return s
}
