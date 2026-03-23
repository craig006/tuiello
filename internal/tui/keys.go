package tui

import (
	"charm.land/bubbles/v2/key"
	"github.com/craig006/tuillo/internal/config"
)

type KeyMap struct {
	Quit, Help, Refresh                                    key.Binding
	MoveLeft, MoveRight, MoveUp, MoveDown                 key.Binding
	MoveCardLeft, MoveCardRight, MoveCardUp, MoveCardDown  key.Binding
	Enter, CustomCommand                                    key.Binding
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
	}
}
