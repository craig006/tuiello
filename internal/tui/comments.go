package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"
	"github.com/craig006/tuiello/internal/trello"
)

type CommentMode int

const (
	CommentModeView CommentMode = iota
	CommentModeCreate
	CommentModeEdit
)

// Message types for comment operations
type CreateCommentRequestMsg struct {
	Text string
}

type UpdateCommentRequestMsg struct {
	CommentID string
	Text      string
}

type DeleteCommentRequestMsg struct {
	CommentID string
}

type CommentCreatedMsg struct {
	Comment trello.Comment
}

type CommentUpdatedMsg struct {
	Comment trello.Comment
}

type CommentDeletedMsg struct {
	CommentID string
}

type CommentOperationErrMsg struct {
	Operation string // "create", "update", "delete"
	Err       error
}

// AutocompleteState tracks the state of @ mention autocomplete
type AutocompleteState struct {
	Active      bool
	Matches     []trello.Member
	SelectedIdx int
	Query       string
	Pos         int // cursor position of @
}

// CommentsList is a self-contained Bubble Tea component for displaying and managing comments.
type CommentsList struct {
	// Data
	comments   []trello.Comment
	allMembers []trello.Member

	// Selection & modes
	selectedIdx int
	mode        CommentMode
	editingIdx  int

	// Input
	textInput textinput.Model

	// Autocomplete
	autocomplete AutocompleteState

	// Rendering
	viewport viewport.Model
	width    int
	height   int
	theme    Theme
	keyMap   KeyMap

	// State
	focused        bool
	loading        bool
	loadingErr     string
	deleteConfirming bool
}

// NewCommentsList creates a new CommentsList component with default settings.
func NewCommentsList(theme Theme, keyMap KeyMap) CommentsList {
	ti := textinput.New()
	ti.Placeholder = "Type comment..."

	vp := viewport.New()
	vp.SetWidth(80)
	vp.SetHeight(20)

	return CommentsList{
		comments:         []trello.Comment{},
		allMembers:       []trello.Member{},
		selectedIdx:      0,
		mode:             CommentModeView,
		editingIdx:       -1,
		textInput:        ti,
		autocomplete:     AutocompleteState{},
		viewport:         vp,
		width:            80,
		height:           20,
		theme:            theme,
		keyMap:           keyMap,
		focused:          false,
		deleteConfirming: false,
	}
}

// Update handles incoming messages and updates the component state.
func (cl CommentsList) Update(msg tea.Msg) (CommentsList, tea.Cmd) {
	// Handle comment operation responses FIRST
	switch msg := msg.(type) {
	case CommentCreatedMsg:
		cl.comments = append(cl.comments, msg.Comment)
		cl.mode = CommentModeView
		cl.textInput.SetValue("")
		return cl, nil

	case CommentUpdatedMsg:
		if cl.editingIdx >= 0 && cl.editingIdx < len(cl.comments) {
			cl.comments[cl.editingIdx] = msg.Comment
		}
		cl.mode = CommentModeView
		cl.textInput.SetValue("")
		cl.editingIdx = -1
		return cl, nil

	case CommentDeletedMsg:
		// Remove comment from list
		if cl.selectedIdx < len(cl.comments) {
			cl.comments = append(cl.comments[:cl.selectedIdx], cl.comments[cl.selectedIdx+1:]...)
		}
		if cl.selectedIdx >= len(cl.comments) && cl.selectedIdx > 0 {
			cl.selectedIdx--
		}
		return cl, nil
	}

	// Handle key messages
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !cl.focused {
			return cl, nil
		}

		// Handle deletion confirmation
		if cl.deleteConfirming {
			switch msg.String() {
			case "y":
				if cl.selectedIdx < len(cl.comments) {
					comment := cl.comments[cl.selectedIdx]
					cl.deleteConfirming = false
					return cl, cl.deleteComment(comment.ID)
				}
				cl.deleteConfirming = false
				return cl, nil
			case "n":
				cl.deleteConfirming = false
				return cl, nil
			}
		}

		// Handle input in Create/Edit modes
		if cl.mode == CommentModeCreate || cl.mode == CommentModeEdit {
			// First check if this is @ character (before normal text input)
			if msg.String() == "@" {
				cl.autocomplete.Active = true
				cl.autocomplete.Query = ""
				cl.autocomplete.SelectedIdx = 0
				cl.autocomplete.Pos = len(cl.textInput.Value())
				cl.autocomplete.Matches = cl.allMembers // Start with all members
				return cl, nil
			}


		// Handle autocomplete navigation and selection
		if cl.autocomplete.Active {
			switch msg.String() {
			case "j":
				if cl.autocomplete.SelectedIdx < len(cl.autocomplete.Matches)-1 {
					cl.autocomplete.SelectedIdx++
				}
				return cl, nil
			case "k":
				if cl.autocomplete.SelectedIdx > 0 {
					cl.autocomplete.SelectedIdx--
				}
				return cl, nil
			case "enter", "tab":
				if cl.autocomplete.SelectedIdx < len(cl.autocomplete.Matches) {
					member := cl.autocomplete.Matches[cl.autocomplete.SelectedIdx]
					cl.insertMention(member.Username)
				}
				return cl, nil
			case "esc":
				cl.autocomplete.Active = false
				return cl, nil
			case "backspace":
				if len(cl.autocomplete.Query) > 0 {
					cl.autocomplete.Query = cl.autocomplete.Query[:len(cl.autocomplete.Query)-1]
					cl.filterMembers(cl.autocomplete.Query)
				} else {
					// If query is empty, close autocomplete
					cl.autocomplete.Active = false
				}
				return cl, nil
			}

			// When autocomplete is active, handle typing to filter
			keyStr := msg.String()
			if len(keyStr) > 0 && keyStr != "delete" {
				// Add to query and filter
				cl.autocomplete.Query += keyStr
				cl.filterMembers(cl.autocomplete.Query)
				return cl, nil
			}

			return cl, nil
		}

		// Then handle other special keys when autocomplete is not active
		switch msg.String() {
		case "enter":
			text := cl.textInput.Value()
			if strings.TrimSpace(text) == "" {
				return cl, nil // Ignore empty submissions
			}
			return cl, cl.submitComment()
		case "esc":
			cl.mode = CommentModeView
			cl.textInput.SetValue("")
			cl.textInput.Blur()
			cl.autocomplete.Active = false
			return cl, nil
		}

		// Normal text input when not in autocomplete
		var cmd tea.Cmd
		cl.textInput, cmd = cl.textInput.Update(msg)
		return cl, cmd
	}

	// Only handle navigation keys in View mode
	if cl.mode == CommentModeView {
			switch {
			case key.Matches(msg, cl.keyMap.MoveDown):
				if cl.selectedIdx < len(cl.comments)-1 {
					cl.selectedIdx++
				}
				return cl, nil
			case key.Matches(msg, cl.keyMap.MoveUp):
				if cl.selectedIdx > 0 {
					cl.selectedIdx--
				}
				return cl, nil
			case msg.String() == "c":
				cl.mode = CommentModeCreate
				cl.textInput.SetValue("")
				cl.textInput.Focus()
				return cl, textinput.Blink
			case msg.String() == "e":
				if cl.selectedIdx < len(cl.comments) {
					comment := cl.comments[cl.selectedIdx]
					if comment.Editable {
						cl.mode = CommentModeEdit
						cl.editingIdx = cl.selectedIdx
						cl.textInput.SetValue(comment.Body)
						cl.textInput.Focus()
						return cl, textinput.Blink
					}
				}
			case msg.String() == "d":
				if cl.selectedIdx < len(cl.comments) {
					comment := cl.comments[cl.selectedIdx]
					if comment.Editable {
						cl.deleteConfirming = true
						return cl, nil
					}
				}
			}
		}
	}
	return cl, nil
}

// View renders the comments list to a string, dispatching to the correct render method.
func (cl CommentsList) View() string {
	// Show delete confirmation if active
	if cl.deleteConfirming {
		if cl.selectedIdx < len(cl.comments) {
			comment := cl.comments[cl.selectedIdx]
			prompt := lipgloss.NewStyle().
				Foreground(lipgloss.Color("1")).
				Bold(true).
				Render("Delete comment? (y/n)")

			dateStr := comment.Date.Format("2006-01-02 15:04:05")
			body := wordWrap(comment.Body, cl.width-4)
			return comment.Author.FullName + " (" + dateStr + ")\n" + body + "\n\n" + prompt
		}
	}

	switch cl.mode {
	case CommentModeCreate:
		return cl.renderCreateMode()
	case CommentModeEdit:
		return cl.renderEditMode()
	default:
		return cl.renderViewMode()
	}
}

// renderViewMode renders the comments list in view mode
func (cl CommentsList) renderViewMode() string {
	if len(cl.comments) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Render("No comments. Press 'c' to create.")
	}

	// Build comment blocks
	var blocks []string
	selBg := lipgloss.Color("236") // subtle dark grey highlight

	for i, comment := range cl.comments {
		isSelected := i == cl.selectedIdx

		// Format: "Author Name (YYYY-MM-DD HH:MM:SS)"
		dateStr := comment.Date.Format("2006-01-02 15:04:05")
		header := comment.Author.FullName + " (" + dateStr + ")"

		// Body with word wrap - account for padding
		body := wordWrap(comment.Body, cl.width-4)

		// Build lines for this comment (unrendered strings)
		var lines []string
		lines = append(lines, header)
		for _, line := range strings.Split(body, "\n") {
			lines = append(lines, line)
		}

		// Add action shortcuts for selected comments
		if isSelected && comment.Editable {
			lines = append(lines, "")
			lines = append(lines, "Edit (e) | Delete (d)")
		}

		// Join all lines
		commentContent := strings.Join(lines, "\n")

		// Apply styling to the entire block
		if isSelected {
			// Selected: blue left bar + background
			// Use PaddingLeft to push content right, border overlays it
			selBorder := lipgloss.Border{Left: "▎"}
			style := lipgloss.NewStyle().
				Background(selBg).
				Foreground(lipgloss.ANSIColor(15)).
				Width(cl.width).
				BorderLeft(true).
				BorderStyle(selBorder).
				BorderForeground(lipgloss.ANSIColor(4)).
				BorderBackground(selBg).
				Padding(0, 1, 0, 0). // right=1, top=0, bottom=0, left=0
				MarginLeft(0)
			blocks = append(blocks, style.Render(commentContent))
		} else {
			// Unfocused: reserve space with padding to align with selected
			// 2 chars padding to reserve space for border width
			style := lipgloss.NewStyle().
				Foreground(lipgloss.ANSIColor(7)).
				Width(cl.width).
				Padding(0, 1, 0, 2). // right=1, top=0, bottom=0, left=2
				MarginLeft(0)
			blocks = append(blocks, style.Render(commentContent))
		}

		// Add empty line between comments (but not after the last one)
		if i < len(cl.comments)-1 {
			blocks = append(blocks, "")
		}
	}

	// Render through viewport
	content := strings.Join(blocks, "\n")
	cl.viewport.SetContent(content)

	// Add footer
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("Press 'c' to create, 'e' to edit, 'd' to delete")

	return cl.viewport.View() + "\n" + footer
}

// renderEditMode renders the edit comment input interface
func (cl CommentsList) renderEditMode() string {
	if cl.editingIdx < 0 || cl.editingIdx >= len(cl.comments) {
		return ""
	}

	comment := cl.comments[cl.editingIdx]
	title := lipgloss.NewStyle().
		Bold(true).
		Render("[Edit Comment]")

	authorStr := comment.Author.FullName + " (" + comment.Date.Format("2006-01-02 15:04:05") + ")"

	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1).
		Width(cl.width - 2).
		Render(cl.textInput.View())

	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("Submit: Enter | Cancel: Esc | Newline: Shift+Enter")

	// Include autocomplete if active
	autocompleteView := cl.renderAutocomplete()
	if autocompleteView != "" {
		return title + "\n" + authorStr + "\n" + inputBox + "\n" + autocompleteView + "\n" + footer
	}
	return title + "\n" + authorStr + "\n" + inputBox + "\n" + footer
}

// SetComments updates the comments list and resets selection.
func (cl *CommentsList) SetComments(comments []trello.Comment) {
	cl.comments = comments
	if len(comments) > 0 {
		cl.selectedIdx = 0
	}
}

// SetMembers updates the available members for autocomplete.
func (cl *CommentsList) SetMembers(members []trello.Member) {
	cl.allMembers = members
}

// filterMembers filters the available members based on a query string.
// Matches against FullName and Username (case-insensitive).
func (cl *CommentsList) filterMembers(query string) {
	cl.autocomplete.Query = query
	cl.autocomplete.Matches = []trello.Member{}

	query = strings.ToLower(query)
	for _, member := range cl.allMembers {
		if strings.Contains(strings.ToLower(member.FullName), query) ||
			strings.Contains(strings.ToLower(member.Username), query) {
			cl.autocomplete.Matches = append(cl.autocomplete.Matches, member)
		}
	}

	if len(cl.autocomplete.Matches) > 0 {
		cl.autocomplete.SelectedIdx = 0
	}
}

// SetSize updates the width and height of the component and viewport.
func (cl *CommentsList) SetSize(width, height int) {
	cl.width = width
	cl.height = height
	cl.viewport.SetWidth(width)
	cl.viewport.SetHeight(height - 5)
}

// SetFocus sets the focus state and manages text input focus.
func (cl *CommentsList) SetFocus(focused bool) {
	cl.focused = focused
	if focused {
		cl.textInput.Focus()
	} else {
		cl.textInput.Blur()
	}
}

// submitComment generates a message for creating or updating a comment
func (cl CommentsList) submitComment() tea.Cmd {
	return func() tea.Msg {
		text := cl.textInput.Value()
		if cl.mode == CommentModeCreate {
			return CreateCommentRequestMsg{Text: text}
		}
		return UpdateCommentRequestMsg{
			CommentID: cl.comments[cl.editingIdx].ID,
			Text:      text,
		}
	}
}

// deleteComment generates a message for deleting a comment
func (cl CommentsList) deleteComment(commentID string) tea.Cmd {
	return func() tea.Msg {
		return DeleteCommentRequestMsg{CommentID: commentID}
	}
}

// renderAutocomplete renders the @ mention autocomplete popup showing matching members.
func (cl CommentsList) renderAutocomplete() string {
	if !cl.autocomplete.Active || len(cl.autocomplete.Matches) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Members:"))

	for i, member := range cl.autocomplete.Matches {
		indicator := " "
		if i == cl.autocomplete.SelectedIdx {
			indicator = ">"
		}

		line := lipgloss.NewStyle().
			Render(indicator + " " + member.FullName + " (@" + member.Username + ")")

		if i == cl.autocomplete.SelectedIdx {
			line = lipgloss.NewStyle().
				Foreground(lipgloss.Color("4")).
				Render(line)
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// insertMention inserts a @username mention into the text input
func (cl *CommentsList) insertMention(username string) {
	text := cl.textInput.Value()

	// Find the @ position and replace from @ to cursor with mention
	pos := len(text)

	// Backtrack to find @ character
	atPos := pos - 1
	for atPos >= 0 && text[atPos] != '@' {
		atPos--
	}

	if atPos >= 0 {
		// Replace from @ to current position with @username
		before := text[:atPos]
		after := text[pos:]
		newText := before + "@" + username + after
		cl.textInput.SetValue(newText)

		// Close autocomplete
		cl.autocomplete.Active = false
		cl.autocomplete.Matches = []trello.Member{}
	}
}

// renderCreateMode renders the create comment input interface
func (cl CommentsList) renderCreateMode() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Render("[New Comment]")

	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1).
		Width(cl.width - 2).
		Render(cl.textInput.View())

	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("Submit: Enter | Cancel: Esc | Newline: Shift+Enter")

	// Include autocomplete if active
	autocompleteView := cl.renderAutocomplete()
	if autocompleteView != "" {
		return title + "\n" + inputBox + "\n" + autocompleteView + "\n" + footer
	}
	return title + "\n" + inputBox + "\n" + footer
}
