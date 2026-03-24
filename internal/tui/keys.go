package tui

import (
	"charm.land/bubbles/v2/key"
	"github.com/craig006/tuiello/internal/config"
)

type KeyMap struct {
	Quit, Help, Refresh                                    key.Binding
	MoveLeft, MoveRight, MoveUp, MoveDown                 key.Binding
	MoveCardLeft, MoveCardRight, MoveCardUp, MoveCardDown  key.Binding
	Enter, CustomCommand                                    key.Binding
	DetailToggle, DetailTabPrev, DetailTabNext             key.Binding
	DetailScrollDown, DetailScrollUp                       key.Binding
	FilterFocus, FilterMembers, FilterLabels              key.Binding
	ViewNext, ViewPrev                                     key.Binding
}

func NewKeyMap(cfg config.KeybindingConfig) KeyMap {
	return KeyMap{
		Quit:          key.NewBinding(key.WithKeys(cfg.Universal.Quit), key.WithHelp(cfg.Universal.Quit, "quit")),
		Help:          key.NewBinding(key.WithKeys(cfg.Universal.Help), key.WithHelp(cfg.Universal.Help, "help")),
		Refresh:       key.NewBinding(key.WithKeys(cfg.Universal.Refresh), key.WithHelp(cfg.Universal.Refresh, "refresh")),
		MoveLeft:      key.NewBinding(key.WithKeys(cfg.Board.MoveLeft, "left"), key.WithHelp(cfg.Board.MoveLeft, "column left")),
		MoveRight:     key.NewBinding(key.WithKeys(cfg.Board.MoveRight, "right"), key.WithHelp(cfg.Board.MoveRight, "column right")),
		MoveUp:        key.NewBinding(key.WithKeys(cfg.Board.MoveUp, "up"), key.WithHelp(cfg.Board.MoveUp, "card up")),
		MoveDown:      key.NewBinding(key.WithKeys(cfg.Board.MoveDown, "down"), key.WithHelp(cfg.Board.MoveDown, "card down")),
		MoveCardLeft:  key.NewBinding(key.WithKeys(cfg.Board.MoveCardLeft), key.WithHelp(cfg.Board.MoveCardLeft, "move card left")),
		MoveCardRight: key.NewBinding(key.WithKeys(cfg.Board.MoveCardRight), key.WithHelp(cfg.Board.MoveCardRight, "move card right")),
		MoveCardUp:    key.NewBinding(key.WithKeys(cfg.Board.MoveCardUp), key.WithHelp(cfg.Board.MoveCardUp, "move card up")),
		MoveCardDown:  key.NewBinding(key.WithKeys(cfg.Board.MoveCardDown), key.WithHelp(cfg.Board.MoveCardDown, "move card down")),
		Enter:         key.NewBinding(key.WithKeys(cfg.Board.Enter), key.WithHelp(cfg.Board.Enter, "select")),
		CustomCommand: key.NewBinding(key.WithKeys(cfg.Board.CustomCommand), key.WithHelp(cfg.Board.CustomCommand, "commands")),
		DetailToggle:     key.NewBinding(key.WithKeys(cfg.Detail.Toggle), key.WithHelp(cfg.Detail.Toggle, "detail panel")),
		DetailTabPrev:    key.NewBinding(key.WithKeys(cfg.Detail.TabPrev), key.WithHelp(cfg.Detail.TabPrev, "prev tab")),
		DetailTabNext:    key.NewBinding(key.WithKeys(cfg.Detail.TabNext), key.WithHelp(cfg.Detail.TabNext, "next tab")),
		DetailScrollDown: key.NewBinding(key.WithKeys(cfg.Detail.ScrollDown), key.WithHelp(cfg.Detail.ScrollDown, "scroll down")),
		DetailScrollUp:   key.NewBinding(key.WithKeys(cfg.Detail.ScrollUp), key.WithHelp(cfg.Detail.ScrollUp, "scroll up")),
		FilterFocus:      key.NewBinding(key.WithKeys(cfg.Filter.Focus), key.WithHelp(cfg.Filter.Focus, "search")),
		FilterMembers:    key.NewBinding(key.WithKeys(cfg.Filter.Members), key.WithHelp(cfg.Filter.Members, "filter members")),
		FilterLabels:     key.NewBinding(key.WithKeys(cfg.Filter.Labels), key.WithHelp(cfg.Filter.Labels, "filter labels")),
		ViewNext:         key.NewBinding(key.WithKeys(cfg.Views.NextView), key.WithHelp(cfg.Views.NextView, "next view")),
		ViewPrev:         key.NewBinding(key.WithKeys(cfg.Views.PrevView), key.WithHelp(cfg.Views.PrevView, "prev view")),
	}
}
