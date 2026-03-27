package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"

	"github.com/craig006/tuiello/internal/trello"
)

const (
	tabOverview   = 0
	tabComments   = 1
	tabChecklists = 2
	tabCount      = 3
)

var tabNames = [tabCount]string{"Overview", "Comments", "Checklist"}

// Fetch result messages — all include CardID for stale-response detection.

type CardCommentsMsg struct {
	CardID   string
	Comments []trello.Comment
}

type CardCommentsFetchErrMsg struct {
	CardID string
	Err    error
}

type CardChecklistsMsg struct {
	CardID     string
	Checklists []trello.Checklist
}

type CardChecklistsFetchErrMsg struct {
	CardID string
	Err    error
}

// DetailModel is a self-contained Bubble Tea component for the detail panel.
type DetailModel struct {
	open    bool
	focused bool
	tab     int
	cardID  string

	card       trello.Card
	comments   []trello.Comment
	checklists []trello.Checklist

	commentsLoaded   bool
	checklistsLoaded bool
	commentsLoading  bool
	checklistsLoading bool
	commentsErr    string
	checklistsErr  string

	viewport      viewport.Model
	commentsList  *CommentsList
	width         int
	height        int
	padding       int
	keyMap        KeyMap
	theme         Theme
	boardHasFocus bool  // true when board has focus, false when detail has focus
}

func NewDetailModel(km KeyMap, theme Theme, padding int) DetailModel {
	vp := viewport.New()
	cl := NewCommentsList(theme, km)
	return DetailModel{
		keyMap:       km,
		theme:        theme,
		padding:      padding,
		viewport:     vp,
		commentsList: &cl,
	}
}

func (d *DetailModel) Toggle() {
	d.open = !d.open
}

func (d *DetailModel) NextTab() {
	d.tab = (d.tab + 1) % tabCount
}

func (d *DetailModel) PrevTab() {
	d.tab = (d.tab - 1 + tabCount) % tabCount
}

// SetFocus sets the focus state of the detail panel.
// When defocusing, any active inputs will be blurred (to be implemented in future tasks).
func (d *DetailModel) SetFocus(focused bool) {
	d.focused = focused

	// Set focus on CommentsList if Comments tab is active
	if d.commentsList != nil {
		d.commentsList.SetFocus(focused && d.tab == tabComments)
	}
}

// SetCard updates the displayed card and clears cached data.
// After calling SetCard, check NeedsFetch() to determine if a fetch command is needed.
func (d *DetailModel) SetCard(card trello.Card) {
	d.card = card
	d.cardID = card.ID
	d.comments = nil
	d.checklists = nil
	d.commentsLoaded = false
	d.checklistsLoaded = false
	d.commentsLoading = false
	d.checklistsLoading = false
	d.commentsErr = ""
	d.checklistsErr = ""

	// Clear CommentsList when card changes
	if d.commentsList != nil {
		d.commentsList.SetComments([]trello.Comment{})
	}
}

func (d *DetailModel) SetSize(width, height int) {
	d.width = width
	d.height = height
	// Reserve 3 lines for border (top + bottom) and tab bar
	vpHeight := height - 4
	if vpHeight < 1 {
		vpHeight = 1
	}
	vpWidth := width - 4 - d.padding // border + padding + content padding
	if vpWidth < 1 {
		vpWidth = 1
	}
	d.viewport.SetWidth(vpWidth)
	d.viewport.SetHeight(vpHeight)

	// Also set size on CommentsList
	if d.commentsList != nil {
		d.commentsList.SetSize(vpWidth, vpHeight)
	}
}

func (d *DetailModel) HandleCommentsMsg(msg CardCommentsMsg) {
	if msg.CardID != d.cardID {
		return // stale response
	}
	d.comments = msg.Comments
	d.commentsLoaded = true
	d.commentsLoading = false
	d.commentsErr = ""

	// Sync comments to CommentsList
	if d.commentsList != nil {
		d.commentsList.SetComments(msg.Comments)
	}
}

func (d *DetailModel) HandleCommentsFetchErr(msg CardCommentsFetchErrMsg) {
	if msg.CardID != d.cardID {
		return
	}
	d.commentsLoading = false
	d.commentsErr = fmt.Sprintf("Failed to load comments: %v", msg.Err)
}

func (d *DetailModel) HandleChecklistsMsg(msg CardChecklistsMsg) {
	if msg.CardID != d.cardID {
		return // stale response
	}
	d.checklists = msg.Checklists
	d.checklistsLoaded = true
	d.checklistsLoading = false
	d.checklistsErr = ""
}

func (d *DetailModel) HandleChecklistsFetchErr(msg CardChecklistsFetchErrMsg) {
	if msg.CardID != d.cardID {
		return
	}
	d.checklistsLoading = false
	d.checklistsErr = fmt.Sprintf("Failed to load checklists: %v", msg.Err)
}

// NeedsFetch returns true if the active tab needs data fetched.
func (d *DetailModel) NeedsFetch() bool {
	switch d.tab {
	case tabComments:
		return !d.commentsLoaded
	case tabChecklists:
		return !d.checklistsLoaded
	}
	return false
}

// ScrollDown scrolls the viewport down by one line.
func (d *DetailModel) ScrollDown() {
	d.viewport.ScrollDown(1)
}

// ScrollUp scrolls the viewport up by one line.
func (d *DetailModel) ScrollUp() {
	d.viewport.ScrollUp(1)
}

// MarkLoading sets the loading flag for the active tab.
func (d *DetailModel) MarkLoading() {
	switch d.tab {
	case tabComments:
		d.commentsLoading = true
	case tabChecklists:
		d.checklistsLoading = true
	}
}

// Update handles viewport scrolling messages.
func (d DetailModel) Update(msg tea.Msg) (DetailModel, tea.Cmd) {
	// If Comments tab is active and detail is focused, delegate to CommentsList
	if d.open && d.focused && d.tab == tabComments && d.commentsList != nil {
		*d.commentsList, msg = d.commentsList.Update(msg)
		// Return early if it was a message CommentsList handled
		// Otherwise fall through to normal viewport handling
		switch msg.(type) {
		case CreateCommentRequestMsg, UpdateCommentRequestMsg, DeleteCommentRequestMsg:
			// CommentsList has handled these - return them as commands to parent
			return d, func() tea.Msg { return msg }
		}
	}

	var cmd tea.Cmd
	d.viewport, cmd = d.viewport.Update(msg)
	return d, cmd
}

// View renders the detail panel with border, tab bar, and content.
// SetBoardFocus sets whether the board has focus (false means detail has focus)
func (d *DetailModel) SetBoardFocus(boardHasFocus bool) {
	d.boardHasFocus = boardHasFocus
}

func (d DetailModel) View() string {
	if !d.open {
		return ""
	}

	contentWidth := d.width - 4 - d.padding // border + padding + content padding
	if contentWidth < 1 {
		contentWidth = 1
	}

	// Render content based on active tab
	var content string
	if d.cardID == "" {
		content = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Render("No card selected")
	} else {
		switch d.tab {
		case tabOverview:
			content = d.renderOverview(contentWidth)
		case tabComments:
			if d.commentsList != nil {
				content = d.renderCommentsList()
			} else {
				content = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Render("Loading comments...")
			}
		case tabChecklists:
			if d.checklistsErr != "" {
				content = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(1)).Render(d.checklistsErr)
			} else {
				content = d.renderChecklists(contentWidth)
			}
		}
	}

	d.viewport.SetContent(content)

	// Build border with tab bar
	// Detail border is active (blue) when detail has focus (!boardHasFocus)
	borderColor := d.theme.InactiveBorder.GetForeground()
	if !d.boardHasFocus {
		borderColor = d.theme.ActiveBorder.GetForeground()
	}
	tabActiveColor := d.theme.ActiveBorder.GetForeground()
	border := lipgloss.RoundedBorder()

	// Build tab bar string with dynamic labels
	var tabBar string
	for i, name := range tabNames {
		label := name
		if i == tabComments && d.card.CommentCount > 0 {
			label = fmt.Sprintf("%s (%d)", name, d.card.CommentCount)
		}
		if i == tabChecklists && d.card.CheckItemCount > 0 {
			label = fmt.Sprintf("%s (%d/%d)", name, d.card.CheckItemsChecked, d.card.CheckItemCount)
		}
		if i == d.tab {
			tabBar += lipgloss.NewStyle().Bold(true).Foreground(tabActiveColor).Render(" "+label+" ")
		} else {
			tabBar += lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Render(" "+label+" ")
		}
	}

	// Render panel with border
	style := lipgloss.NewStyle().
		Width(d.width).
		PaddingLeft(d.padding).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Height(d.height)

	rendered := style.Render(d.viewport.View())
	lines := strings.Split(rendered, "\n")

	// Replace top border with tab bar
	if len(lines) > 0 {
		origWidth := lipgloss.Width(lines[0])
		tabBarWidth := lipgloss.Width(tabBar)
		trailing := origWidth - 2 - 1 - tabBarWidth // corners + leading dash
		if trailing < 0 {
			trailing = 0
		}
		borderStyle := lipgloss.NewStyle().Foreground(borderColor)
		lines[0] = borderStyle.Render(border.TopLeft+border.Top) +
			tabBar +
			borderStyle.Render(strings.Repeat(border.Top, trailing)+border.TopRight)
		rendered = strings.Join(lines, "\n")
	}

	return rendered
}

func (d DetailModel) renderOverview(width int) string {
	var sections []string

	// Title
	wrappedName := wordWrap(d.card.Name, width)
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(15)).Render(wrappedName)
	sections = append(sections, "")
	sections = append(sections, title)
	sections = append(sections, "")

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(7))

	// Labels
	labelsLabel := dimStyle.Render("Labels: ")
	if len(d.card.Labels) > 0 {
		var labels []string
		for _, lbl := range d.card.Labels {
			ansiColor, ok := trelloColorToANSI[lbl.Color]
			if !ok {
				ansiColor = 7
			}
			indicator := lipgloss.NewStyle().Foreground(ansiColor).Render("⏺")
			displayName := lbl.Name
			if displayName == "" {
				displayName = "-"
			}
			name := textStyle.Render(" " + displayName)
			labels = append(labels, indicator+name)
		}
		sections = append(sections, labelsLabel+strings.Join(labels, "  "))
	} else {
		sections = append(sections, labelsLabel+dimStyle.Render("-"))
	}

	// Members
	membersLabel := dimStyle.Render("Members: ")
	if len(d.card.Members) > 0 {
		var names []string
		for _, m := range d.card.Members {
			names = append(names, m.FullName)
		}
		sections = append(sections, membersLabel+textStyle.Render(strings.Join(names, ", ")))
	} else {
		sections = append(sections, membersLabel+dimStyle.Render("-"))
	}

	// Custom Fields
	if len(d.card.CustomFields) > 0 {
		for _, cf := range d.card.CustomFields {
			fieldLine := dimStyle.Render(cf.FieldName+": ") + textStyle.Render(cf.Value)
			sections = append(sections, fieldLine)
		}
	}

	// Description
	sections = append(sections, "") // spacer above line
	sections = append(sections, dimStyle.Render(strings.Repeat("─", width)))
	sections = append(sections, "") // spacer below line
	if d.card.Description != "" {
		sections = append(sections, renderMarkdown(d.card.Description, width))
	} else {
		sections = append(sections, lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Render("No description."))
	}

	return strings.Join(sections, "\n")
}

func (d DetailModel) renderComments(width int) string {
	if d.commentsLoading {
		return lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Render("Loading comments...")
	}
	if len(d.comments) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Render("No comments.")
	}

	var sections []string
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))
	boldStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(15))

	for i, c := range d.comments {
		if i > 0 {
			sections = append(sections, dimStyle.Render(strings.Repeat("─", width)))
		}
		dateStr := c.Date.Format("2006-01-02")
		header := boldStyle.Render(c.Author.FullName) + " " + dimStyle.Render("("+dateStr+")")
		body := renderMarkdown(c.Body, width)
		sections = append(sections, header+"\n"+body)
	}

	return strings.Join(sections, "\n")
}

func (d DetailModel) renderChecklists(width int) string {
	if d.checklistsLoading {
		return lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Render("Loading checklists...")
	}
	if len(d.checklists) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Render("No checklists.")
	}

	var sections []string
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))
	boldStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(15))
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(7))
	showHeaders := len(d.checklists) > 1

	for _, cl := range d.checklists {
		if showHeaders {
			sections = append(sections, boldStyle.Render(cl.Name))
		}
		for _, item := range cl.Items {
			var line string
			if item.Complete {
				line = dimStyle.Render("[x] " + item.Name)
			} else {
				line = normalStyle.Render("[ ] " + item.Name)
			}
			sections = append(sections, line)
		}
		if showHeaders {
			sections = append(sections, "") // spacer between checklists
		}
	}

	return strings.Join(sections, "\n")
}

func (d DetailModel) renderCommentsList() string {
	if d.commentsList == nil {
		return lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Render("Loading comments...")
	}
	return d.commentsList.View()
}

// wordWrap wraps text to the given width.
func wordWrap(text string, width int) string {
	if width <= 0 {
		return text
	}
	var result strings.Builder
	for _, paragraph := range strings.Split(text, "\n") {
		if result.Len() > 0 {
			result.WriteByte('\n')
		}
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			continue
		}
		lineLen := 0
		for i, word := range words {
			wordLen := len([]rune(word))
			if i > 0 && lineLen+1+wordLen > width {
				result.WriteByte('\n')
				lineLen = 0
			} else if i > 0 {
				result.WriteByte(' ')
				lineLen++
			}
			result.WriteString(word)
			lineLen += wordLen
		}
	}
	return result.String()
}

// renderMarkdown renders text as markdown via glamour, falling back to plain word-wrapped text.
func renderMarkdown(text string, width int) string {
	style := styles.DarkStyleConfig
	noMargin := uint(0)
	style.Document.Margin = &noMargin
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(style),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(7)).Render(wordWrap(text, width))
	}
	rendered, err := r.Render(text)
	if err != nil {
		return lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(7)).Render(wordWrap(text, width))
	}
	return strings.TrimSpace(rendered)
}
