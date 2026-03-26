// internal/tui/board.go
package tui

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/craig006/tuiello/internal/config"
	"github.com/craig006/tuiello/internal/trello"
)

const maxVisibleColumns = 3

// BoardModel manages the kanban board view with a 3-column sliding window.
type BoardModel struct {
	columns       []Column
	board         *trello.Board
	focused       int
	hiddenColumns map[string]struct{}
	width         int
	height        int
	minColWidth   int
	padding       int
	keyMap        KeyMap
	theme         Theme
	filter        Filter
	dimColumns    bool // true when search bar is focused, dims active column border
}

func NewBoardModel(board *trello.Board, cfg config.Config, width, height int) BoardModel {
	km := NewKeyMap(cfg.Keybinding)
	theme := NewTheme(cfg.GUI.Theme)

	if len(board.Lists) == 0 {
		return BoardModel{
			board:         board,
			hiddenColumns: map[string]struct{}{},
			width:         width,
			height:        height,
			minColWidth:   cfg.GUI.ColumnWidth,
			padding:       cfg.GUI.Padding,
			keyMap:        km,
			theme:         theme,
		}
	}

	colWidth := width / min(len(board.Lists), maxVisibleColumns)
	if cfg.GUI.ColumnWidth > 0 && colWidth < cfg.GUI.ColumnWidth {
		colWidth = cfg.GUI.ColumnWidth
	}
	colHeight := height - 7 // 4 for chrome + 3 for breadcrumb

	columns := make([]Column, len(board.Lists))
	for i, l := range board.Lists {
		columns[i] = NewColumn(l, colWidth, colHeight, i == 0)
	}

	bm := BoardModel{
		columns:       columns,
		board:         board,
		focused:       0,
		hiddenColumns: map[string]struct{}{},
		width:         width,
		height:        height,
		minColWidth:   cfg.GUI.ColumnWidth,
		padding:       cfg.GUI.Padding,
		keyMap:        km,
		theme:         theme,
	}
	bm.ResizeColumns()
	return bm
}

// ResizeColumns updates all column dimensions to match current board size.
func (b *BoardModel) ResizeColumns() {
	visibleColumns := b.VisibleColumnIndices()
	if len(visibleColumns) == 0 {
		return
	}
	visible := min(len(visibleColumns), maxVisibleColumns)
	colWidth := b.width / visible
	if b.minColWidth > 0 && colWidth < b.minColWidth {
		colWidth = b.minColWidth
	}
	colHeight := b.height - 2 // 2 for column border
	for i := range b.columns {
		b.columns[i].SetSize(colWidth-2, colHeight)
	}
}

func (b *BoardModel) SetHiddenColumns(names []string) {
	b.hiddenColumns = make(map[string]struct{}, len(names))
	for _, name := range names {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		b.hiddenColumns[strings.ToLower(trimmed)] = struct{}{}
	}

	if len(b.columns) == 0 {
		return
	}

	visible := b.VisibleColumnIndices()
	if len(visible) == 0 {
		b.focused = 0
		return
	}

	if !b.isVisibleColumn(b.focused) {
		b.columns[b.focused].SetFocused(false)
		b.focused = visible[0]
	}

	for i := range b.columns {
		b.columns[i].SetFocused(i == b.focused)
	}
	if slices.Contains(visible, b.focused) {
		b.columns[b.focused].SetFocused(true)
	}
	b.ResizeColumns()
}

func (b BoardModel) isHiddenColumn(name string) bool {
	_, ok := b.hiddenColumns[strings.ToLower(strings.TrimSpace(name))]
	return ok
}

func (b BoardModel) isVisibleColumn(index int) bool {
	if index < 0 || index >= len(b.columns) {
		return false
	}
	return !b.isHiddenColumn(b.columns[index].name)
}

func (b BoardModel) VisibleColumnIndices() []int {
	indices := make([]int, 0, len(b.columns))
	for i := range b.columns {
		if b.isVisibleColumn(i) {
			indices = append(indices, i)
		}
	}
	return indices
}

func (b BoardModel) visibleColumnPosition(index int) int {
	visible := b.VisibleColumnIndices()
	for i, colIdx := range visible {
		if colIdx == index {
			return i
		}
	}
	return -1
}

func (b *BoardModel) FocusedColumn() int { return b.focused }

func (b *BoardModel) FocusLeft() {
	visible := b.VisibleColumnIndices()
	current := b.visibleColumnPosition(b.focused)
	if current > 0 {
		b.columns[b.focused].SetFocused(false)
		b.focused = visible[current-1]
		b.columns[b.focused].SetFocused(true)
	}
}

func (b *BoardModel) FocusRight() {
	visible := b.VisibleColumnIndices()
	current := b.visibleColumnPosition(b.focused)
	if current >= 0 && current < len(visible)-1 {
		b.columns[b.focused].SetFocused(false)
		b.focused = visible[current+1]
		b.columns[b.focused].SetFocused(true)
	}
}

// VisibleRange returns the [start, end) indices of visible columns.
func (b *BoardModel) VisibleRange() (int, int) {
	visible := b.VisibleColumnIndices()
	total := len(visible)
	if total <= maxVisibleColumns {
		return 0, total
	}

	current := b.visibleColumnPosition(b.focused)
	if current < 0 {
		current = 0
	}

	start := current - 1
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
	visible := b.VisibleColumnIndices()
	if len(visible) == 0 {
		return "[0/0]"
	}
	return fmt.Sprintf("[%d/%d]", b.visibleColumnPosition(b.focused)+1, len(visible))
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
	var items []list.Item
	filteredIdx := -1
	fi := 0
	for i, c := range col.cards {
		if b.filter.MatchesCard(c) {
			items = append(items, cardItem{card: c})
			if i == selectIdx {
				filteredIdx = fi
			}
			fi++
		}
	}
	if items == nil {
		items = []list.Item{}
	}
	col.list.SetItems(items)
	if filteredIdx >= 0 {
		col.list.Select(filteredIdx)
	} else if len(items) > 0 && col.list.Index() >= len(items) {
		col.list.Select(len(items) - 1)
	}
}

// ApplyFilter updates the filter and rebuilds all column item lists.
func (b *BoardModel) ApplyFilter(f Filter) {
	b.filter = f
	for i := range b.columns {
		b.rebuildFilteredItems(i)
	}
}

// rebuildFilteredItems rebuilds a column's list items, applying the current filter.
func (b *BoardModel) rebuildFilteredItems(colIdx int) {
	col := &b.columns[colIdx]
	var items []list.Item
	for _, c := range col.cards {
		if b.filter.MatchesCard(c) {
			items = append(items, cardItem{card: c})
		}
	}
	if items == nil {
		items = []list.Item{}
	}
	col.list.SetItems(items)
	// Clamp selection to valid range
	if col.list.Index() >= len(items) && len(items) > 0 {
		col.list.Select(len(items) - 1)
	}
}

// ClearFilter removes all filters and rebuilds column items,
// preserving the currently selected card in each column.
func (b *BoardModel) ClearFilter() {
	// Remember the selected card in each column before clearing
	selectedIDs := make([]string, len(b.columns))
	for i, col := range b.columns {
		item := col.list.SelectedItem()
		if item != nil {
			if ci, ok := item.(cardItem); ok {
				selectedIDs[i] = ci.card.ID
			}
		}
	}

	b.filter = Filter{}
	for i := range b.columns {
		b.rebuildFilteredItems(i)
		// Re-select the card that was selected before clearing
		if selectedIDs[i] != "" {
			idx := fullCardIndex(b.columns[i].cards, selectedIDs[i])
			if idx >= 0 {
				b.columns[i].list.Select(idx)
			}
		}
	}
}

// HasFilter returns true if any filter is active.
func (b *BoardModel) HasFilter() bool {
	return !b.filter.IsEmpty()
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

// RenderBreadcrumb renders the nav breadcrumb bar at the given width.
func (b BoardModel) RenderBreadcrumb(width int) string {
	visible := b.VisibleColumnIndices()
	if len(visible) == 0 {
		return ""
	}
	start, end := b.VisibleRange()

	var breadcrumbParts []string
	for pos, colIdx := range visible {
		col := b.columns[colIdx]
		name := col.name
		var style lipgloss.Style
		if colIdx == b.focused {
			style = lipgloss.NewStyle().Bold(true).Underline(true).Foreground(lipgloss.ANSIColor(4))
		} else if pos >= start && pos < end {
			style = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(7))
		} else {
			style = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))
		}
		breadcrumbParts = append(breadcrumbParts, style.Render(name))
	}
	separator := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Render(" • ")
	breadcrumbText := strings.Join(breadcrumbParts, separator)
	return lipgloss.NewStyle().
		Width(width).
		Height(1).
		Padding(1, 0).
		Align(lipgloss.Center).
		Render(breadcrumbText)
}

func (b BoardModel) View() string {
	visible := b.VisibleColumnIndices()
	if len(visible) == 0 {
		return "No visible lists in this view."
	}
	if len(b.columns) == 0 {
		return "No lists on this board."
	}

	start, end := b.VisibleRange()

	colH := b.height

	visibleCount := end - start
	colWidth := b.width / visibleCount
	if b.minColWidth > 0 && colWidth < b.minColWidth {
		colWidth = b.minColWidth
	}

	views := make([]string, 0, visibleCount)
	border := lipgloss.RoundedBorder()
	for _, colIdx := range visible[start:end] {
		col := b.columns[colIdx]

		borderColor := b.theme.InactiveBorder.GetForeground()
		if colIdx == b.focused && !b.dimColumns {
			borderColor = b.theme.ActiveBorder.GetForeground()
		}

		// Give the last visible column any remaining width from rounding
		w := colWidth
		if len(views) == visibleCount-1 {
			w = b.width - colWidth*(visibleCount-1)
		}

		style := lipgloss.NewStyle().
			Width(w).
			Height(colH).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor)

		var colContent string
		if len(col.list.Items()) == 0 {
			msg := "No items."
			if !b.filter.IsEmpty() {
				msg = "No matching cards"
			}
			colContent = lipgloss.NewStyle().PaddingLeft(b.padding).Foreground(lipgloss.ANSIColor(8)).Render(msg)
		} else {
			colContent = col.View()
		}
		rendered := style.Render(colContent)
		lines := strings.Split(rendered, "\n")

		// Build custom top border with embedded title, matching the
		// exact visible width of the original rendered top border.
		if len(lines) > 0 {
			origWidth := lipgloss.Width(lines[0])
			title := fmt.Sprintf(" %s (%d) ", col.name, len(col.list.Items()))
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

			// Embed page indicator in bottom border (right-aligned) if there are multiple pages
			pageInfo := col.PageInfo()
			if pageInfo != "" && len(lines) >= 2 {
				lastIdx := len(lines) - 1
				bottomWidth := lipgloss.Width(lines[lastIdx])
				pageLen := len([]rune(pageInfo))
				leadingBottom := bottomWidth - 2 - 1 - pageLen // 2 corners, 1 trailing dash
				if leadingBottom < 0 {
					leadingBottom = 0
				}
				pageStyle := lipgloss.NewStyle().Foreground(borderColor)
				lines[lastIdx] = borderStyle.Render(border.BottomLeft+strings.Repeat(border.Bottom, leadingBottom)) +
					pageStyle.Render(pageInfo) +
					borderStyle.Render(border.Bottom+border.BottomRight)
			}

			rendered = strings.Join(lines, "\n")
		}

		views = append(views, rendered)
	}

	columns := lipgloss.JoinHorizontal(lipgloss.Top, views...)
	return columns
}
