// internal/tui/board.go
package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"

	"github.com/craig006/tuillo/internal/config"
	"github.com/craig006/tuillo/internal/trello"
)

const maxVisibleColumns = 3

// BoardModel manages the kanban board view with a 3-column sliding window.
type BoardModel struct {
	columns    []Column
	board      *trello.Board
	focused    int
	width      int
	height     int
	minColWidth int
	keyMap     KeyMap
	theme      Theme
}

func NewBoardModel(board *trello.Board, cfg config.Config, width, height int) BoardModel {
	km := NewKeyMap(cfg.Keybinding)
	theme := NewTheme(cfg.GUI.Theme)

	if len(board.Lists) == 0 {
		return BoardModel{
			board:       board,
			width:       width,
			height:      height,
			minColWidth: cfg.GUI.ColumnWidth,
			keyMap:      km,
			theme:       theme,
		}
	}

	colWidth := width / min(len(board.Lists), maxVisibleColumns)
	if cfg.GUI.ColumnWidth > 0 && colWidth < cfg.GUI.ColumnWidth {
		colWidth = cfg.GUI.ColumnWidth
	}
	colHeight := height - 4

	columns := make([]Column, len(board.Lists))
	for i, l := range board.Lists {
		columns[i] = NewColumn(l, colWidth, colHeight, i == 0)
	}

	bm := BoardModel{
		columns:     columns,
		board:       board,
		focused:     0,
		width:       width,
		height:      height,
		minColWidth: cfg.GUI.ColumnWidth,
		keyMap:      km,
		theme:       theme,
	}
	bm.ResizeColumns()
	return bm
}

// ResizeColumns updates all column dimensions to match current board size.
func (b *BoardModel) ResizeColumns() {
	if len(b.columns) == 0 {
		return
	}
	visible := min(len(b.columns), maxVisibleColumns)
	colWidth := b.width / visible
	if b.minColWidth > 0 && colWidth < b.minColWidth {
		colWidth = b.minColWidth
	}
	colHeight := b.height - 4
	for i := range b.columns {
		b.columns[i].SetSize(colWidth-2, colHeight)
	}
}

func (b *BoardModel) FocusedColumn() int { return b.focused }

func (b *BoardModel) FocusLeft() {
	if b.focused > 0 {
		b.columns[b.focused].SetFocused(false)
		b.focused--
		b.columns[b.focused].SetFocused(true)
	}
}

func (b *BoardModel) FocusRight() {
	if b.focused < len(b.columns)-1 {
		b.columns[b.focused].SetFocused(false)
		b.focused++
		b.columns[b.focused].SetFocused(true)
	}
}

// VisibleRange returns the [start, end) indices of visible columns.
func (b *BoardModel) VisibleRange() (int, int) {
	total := len(b.columns)
	if total <= maxVisibleColumns {
		return 0, total
	}

	start := b.focused - 1
	if start < 0 {
		start = 0
	}
	end := start + maxVisibleColumns
	if end > total {
		end = total
		start = end - maxVisibleColumns
	}
	return start, end
}

func (b *BoardModel) PositionIndicator() string {
	return fmt.Sprintf("[%d/%d]", b.focused+1, len(b.columns))
}

// SelectedCard returns the currently focused card and its list index.
func (b *BoardModel) SelectedCard() (trello.Card, int, bool) {
	if len(b.columns) == 0 {
		return trello.Card{}, 0, false
	}
	card, ok := b.columns[b.focused].SelectedCard()
	return card, b.focused, ok
}

// RemoveCard removes a card from the given column at the given index, returning the card.
func (b *BoardModel) RemoveCard(colIdx, cardIdx int) trello.Card {
	col := &b.columns[colIdx]
	card := col.cards[cardIdx]
	col.cards = append(col.cards[:cardIdx], col.cards[cardIdx+1:]...)
	b.rebuildColumnItems(colIdx)
	return card
}

// InsertCard inserts a card into the given column at the given position and selects it.
func (b *BoardModel) InsertCard(colIdx int, card trello.Card, pos int) {
	col := &b.columns[colIdx]
	if pos > len(col.cards) {
		pos = len(col.cards)
	}
	newCards := make([]trello.Card, 0, len(col.cards)+1)
	newCards = append(newCards, col.cards[:pos]...)
	newCards = append(newCards, card)
	newCards = append(newCards, col.cards[pos:]...)
	col.cards = newCards
	b.rebuildColumnItemsAt(colIdx, pos)
}

func (b *BoardModel) rebuildColumnItems(colIdx int) {
	b.rebuildColumnItemsAt(colIdx, -1)
}

func (b *BoardModel) rebuildColumnItemsAt(colIdx int, selectIdx int) {
	col := &b.columns[colIdx]
	items := make([]list.Item, len(col.cards))
	for i, c := range col.cards {
		items[i] = cardItem{card: c}
	}
	col.list.SetItems(items)
	if selectIdx >= 0 {
		col.list.Select(selectIdx)
	}
}

// CalcNewPos calculates the position value for inserting a card at a given index in a column.
func CalcNewPos(cards []trello.Card, targetIdx int) float64 {
	if len(cards) == 0 {
		return 65536.0
	}
	if targetIdx <= 0 {
		return cards[0].Pos / 2.0
	}
	if targetIdx >= len(cards) {
		return cards[len(cards)-1].Pos + 65536.0
	}
	return (cards[targetIdx-1].Pos + cards[targetIdx].Pos) / 2.0
}

func (b BoardModel) Update(msg tea.Msg) (BoardModel, tea.Cmd) {
	if len(b.columns) == 0 {
		return b, nil
	}

	var cmd tea.Cmd
	b.columns[b.focused], cmd = b.columns[b.focused].Update(msg)
	return b, cmd
}

func (b BoardModel) View() string {
	if len(b.columns) == 0 {
		return "No lists on this board."
	}

	start, end := b.VisibleRange()
	colWidth := b.width / (end - start)
	if b.minColWidth > 0 && colWidth < b.minColWidth {
		colWidth = b.minColWidth
	}

	views := make([]string, 0, end-start)
	border := lipgloss.RoundedBorder()
	for i := start; i < end; i++ {
		col := b.columns[i]

		borderColor := b.theme.InactiveBorder.GetForeground()
		if i == b.focused {
			borderColor = b.theme.ActiveBorder.GetForeground()
		}

		// Render content with border but we'll replace the top line
		style := lipgloss.NewStyle().
			Width(colWidth - 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor)

		rendered := style.Render(col.View())
		lines := strings.Split(rendered, "\n")

		// Build custom top border with embedded title, matching the
		// exact visible width of the original rendered top border.
		if len(lines) > 0 {
			origWidth := lipgloss.Width(lines[0])
			title := fmt.Sprintf(" %s (%d) ", col.name, len(col.cards))
			titleLen := len([]rune(title))
			// origWidth = left corner + dashes + right corner
			// new line  = left corner + 1 dash + title + trailing dashes + right corner
			trailingDashes := origWidth - 2 - 1 - titleLen // 2 corners, 1 leading dash
			if trailingDashes < 0 {
				trailingDashes = 0
			}

			borderStyle := lipgloss.NewStyle().Foreground(borderColor)
			titleStyle := lipgloss.NewStyle().Bold(true).Foreground(borderColor)

			lines[0] = borderStyle.Render(border.TopLeft+border.Top) +
				titleStyle.Render(title) +
				borderStyle.Render(strings.Repeat(border.Top, trailingDashes)+border.TopRight)

			rendered = strings.Join(lines, "\n")
		}

		views = append(views, rendered)
	}

	// Equalize column heights so all columns fill the available space
	targetHeight := b.height
	for i, v := range views {
		lines := strings.Split(v, "\n")
		if len(lines) < targetHeight {
			// Measure the visible width of any content line for padding
			padWidth := 0
			if len(lines) > 1 {
				padWidth = lipgloss.Width(lines[1])
			}
			for len(lines) < targetHeight {
				lines = append(lines[:len(lines)-1], strings.Repeat(" ", padWidth), lines[len(lines)-1])
			}
			views[i] = strings.Join(lines, "\n")
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, views...)
}
