package tui

import (
	"fmt"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/craig006/tuillo/internal/trello"
)

// cardItem adapts trello.Card to bubbles/list.DefaultItem interface.
type cardItem struct {
	card trello.Card
}

func (i cardItem) Title() string       { return i.card.Name }
func (i cardItem) Description() string { return "" }
func (i cardItem) FilterValue() string { return i.card.Name }

// Column wraps a bubbles/list.Model for a single Trello list.
type Column struct {
	list   list.Model
	listID string
	name   string
	cards  []trello.Card
}

func NewColumn(l trello.List, width, height int, focused bool) Column {
	items := make([]list.Item, len(l.Cards))
	for i, c := range l.Cards {
		items[i] = cardItem{card: c}
	}
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	m := list.New(items, delegate, width, height)
	m.Title = fmt.Sprintf("%s (%d)", l.Name, len(l.Cards))
	m.SetShowStatusBar(false)
	m.SetFilteringEnabled(false)
	return Column{list: m, listID: l.ID, name: l.Name, cards: l.Cards}
}

func (c Column) Title() string      { return c.name }
func (c Column) ListID() string     { return c.listID }
func (c Column) CardCount() int     { return len(c.cards) }
func (c Column) SelectedIndex() int { return c.list.Index() }

func (c Column) SelectedCard() (trello.Card, bool) {
	if len(c.cards) == 0 {
		return trello.Card{}, false
	}
	item := c.list.SelectedItem()
	if item == nil {
		return trello.Card{}, false
	}
	ci, ok := item.(cardItem)
	if !ok {
		return trello.Card{}, false
	}
	return ci.card, true
}

func (c Column) Cards() []trello.Card { return c.cards }

func (c *Column) SetSize(width, height int) { c.list.SetSize(width, height) }

func (c Column) Update(msg tea.Msg) (Column, tea.Cmd) {
	var cmd tea.Cmd
	c.list, cmd = c.list.Update(msg)
	return c, cmd
}

func (c Column) View() string { return c.list.View() }
