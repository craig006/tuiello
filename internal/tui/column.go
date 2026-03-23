package tui

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"
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

// cardDelegate renders each card as a bordered box filling the column width.
type cardDelegate struct {
	focused bool
}

// trelloColorToANSI maps Trello label color names to ANSI color slots.
var trelloColorToANSI = map[string]lipgloss.ANSIColor{
	"green":       2,
	"yellow":      3,
	"orange":      3, // closest ANSI match
	"red":         1,
	"purple":      5,
	"blue":        4,
	"sky":         6,
	"lime":        10,
	"pink":        13,
	"black":       0,
	"green_dark":  2,
	"yellow_dark": 3,
	"orange_dark": 3,
	"red_dark":    1,
	"purple_dark": 5,
	"blue_dark":   4,
	"sky_dark":    6,
	"lime_dark":   10,
	"pink_dark":   13,
	"black_dark":  8,
	"green_light": 10,
	"yellow_light":11,
	"orange_light":11,
	"red_light":   9,
	"purple_light":13,
	"blue_light":  12,
	"sky_light":   14,
	"lime_light":  10,
	"pink_light":  13,
	"black_light": 7,
}

// memberInitials returns the Trello-provided initials for a member.
func memberInitials(m trello.Member) string {
	if m.Initials != "" {
		return strings.ToUpper(m.Initials)
	}
	return "?"
}

func (d cardDelegate) Height() int  { return 3 } // title + custom fields + labels/meta row
func (d cardDelegate) Spacing() int { return 1 }
func (d cardDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d cardDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ci, ok := item.(cardItem)
	if !ok {
		return
	}

	isSelected := index == m.Index() && d.focused
	cardWidth := m.Width() - 2 // leave room for column padding
	innerWidth := cardWidth - 2 // account for padding

	selBg := lipgloss.ANSIColor(4)
	styledSpan := func(fg lipgloss.ANSIColor, text string) string {
		s := lipgloss.NewStyle().Foreground(fg)
		if isSelected {
			s = s.Background(selBg)
		}
		return s.Render(text)
	}

	// Truncate title to fit
	title := ci.card.Name
	if innerWidth > 0 && len([]rune(title)) > innerWidth {
		title = string([]rune(title)[:innerWidth-1]) + "…"
	}

	// Custom fields row
	var cfLine string
	for i, cf := range ci.card.CustomFields {
		if i > 0 {
			cfLine += styledSpan(8, "  ")
		}
		cfLine += styledSpan(8, cf.FieldName+": ")
		if cf.Color != "" {
			ansiColor, ok := trelloColorToANSI[cf.Color]
			if !ok {
				ansiColor = 7
			}
			cfLine += styledSpan(ansiColor, "⏺ ")
		}
		cfLine += styledSpan(7, cf.Value)
	}

	// Labels (left side of bottom row)
	var labels string
	if len(ci.card.Labels) == 0 {
		labels = styledSpan(8, "-")
	} else {
		for i, lbl := range ci.card.Labels {
			ansiColor, ok := trelloColorToANSI[lbl.Color]
			if !ok {
				ansiColor = 7
			}
			if i > 0 {
				labels += styledSpan(8, " ")
			}
			labels += styledSpan(ansiColor, "⏺")
		}
	}

	// Build fixed-width right-aligned meta slots so icons line up across cards.
	// Comment slot (rightmost): fixed 4 chars wide (icon + space + digits or blank)
	// Member slot (left of comments): fixed 4 chars wide (icon + space + initials or blank)
	// Layout: [labels...] [gap] [member slot] [comment slot]
	const commentSlotWidth = 4
	const memberSlotWidth = 4

	// Build comment slot content
	var commentSlot string
	var commentSlotLen int
	if ci.card.CommentCount > 0 {
		text := fmt.Sprintf("\uf075 %d", ci.card.CommentCount)
		commentSlot = styledSpan(3, text)
		commentSlotLen = len([]rune(text))
	}
	// Pad comment slot to fixed width
	if commentSlotLen < commentSlotWidth {
		commentSlot = styledSpan(0, strings.Repeat(" ", commentSlotWidth-commentSlotLen)) + commentSlot
	}

	// Build member slot content
	var memberSlot string
	var memberSlotLen int
	if len(ci.card.Members) > 0 {
		content := "\uf007 "
		memberSlotLen = 2
		for i, mem := range ci.card.Members {
			if i > 0 {
				content += ","
				memberSlotLen++
			}
			initials := memberInitials(mem)
			content += initials
			memberSlotLen += len([]rune(initials))
		}
		memberSlot = styledSpan(2, string([]rune(content)[:2])) + styledSpan(2, string([]rune(content)[2:]))
	}
	// Pad member slot to at least fixed width
	actualMemberWidth := memberSlotLen
	if actualMemberWidth < memberSlotWidth {
		actualMemberWidth = memberSlotWidth
	}
	if memberSlotLen < actualMemberWidth {
		memberSlot = styledSpan(0, strings.Repeat(" ", actualMemberWidth-memberSlotLen)) + memberSlot
	}

	// Combine: labels (left) + gap + member slot + comment slot
	var labelsVisibleLen int
	if len(ci.card.Labels) == 0 {
		labelsVisibleLen = 1
	} else {
		labelsVisibleLen = len(ci.card.Labels)*2 - 1
	}

	rightWidth := actualMemberWidth + commentSlotWidth
	gap := innerWidth - labelsVisibleLen - rightWidth
	if gap < 1 {
		gap = 1
	}
	bottomRow := labels + styledSpan(0, strings.Repeat(" ", gap)) + memberSlot + commentSlot

	// Build content lines
	content := title
	if cfLine != "" {
		content += "\n" + cfLine
	}
	content += "\n" + bottomRow

	var style lipgloss.Style
	if isSelected {
		style = lipgloss.NewStyle().
			Background(selBg).
			Foreground(lipgloss.ANSIColor(15)).
			Bold(true).
			Width(cardWidth).
			Padding(0, 1)
	} else {
		style = lipgloss.NewStyle().
			Foreground(lipgloss.ANSIColor(7)).
			Width(cardWidth).
			Padding(0, 1)
	}

	fmt.Fprint(w, style.Render(content))
}

// Column wraps a bubbles/list.Model for a single Trello list.
type Column struct {
	list     list.Model
	delegate *cardDelegate
	listID   string
	name     string
	cards    []trello.Card
}

func NewColumn(l trello.List, width, height int, focused bool) Column {
	items := make([]list.Item, len(l.Cards))
	for i, c := range l.Cards {
		items[i] = cardItem{card: c}
	}
	delegate := &cardDelegate{focused: focused}
	m := list.New(items, delegate, width, height)
	m.SetShowTitle(false)
	m.SetShowStatusBar(false)
	m.SetFilteringEnabled(false)
	return Column{list: m, delegate: delegate, listID: l.ID, name: l.Name, cards: l.Cards}
}

// SetFocused updates the delegate so only the focused column highlights its selected card.
func (c *Column) SetFocused(focused bool) {
	c.delegate.focused = focused
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
func (c *Column) Select(index int)          { c.list.Select(index) }

func (c Column) Update(msg tea.Msg) (Column, tea.Cmd) {
	var cmd tea.Cmd
	c.list, cmd = c.list.Update(msg)
	return c, cmd
}

func (c Column) View() string { return c.list.View() }
