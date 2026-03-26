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
	focused    bool
	loading    bool
	loadingErr string
}

// NewCommentsList creates a new CommentsList component with default settings.
func NewCommentsList(theme Theme, keyMap KeyMap) CommentsList {
	ti := textinput.New()
	ti.Placeholder = "Type comment..."

	vp := viewport.New()
	vp.SetWidth(80)
	vp.SetHeight(20)

	return CommentsList{
		comments:     []trello.Comment{},
		allMembers:   []trello.Member{},
		selectedIdx:  0,
		mode:         CommentModeView,
		editingIdx:   -1,
		textInput:    ti,
		autocomplete: AutocompleteState{},
		viewport:     vp,
		width:        80,
		height:       20,
		theme:        theme,
		keyMap:       keyMap,
		focused:      false,
	}
}

// Update handles incoming messages and updates the component state.
func (cl CommentsList) Update(msg tea.Msg) (CommentsList, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !cl.focused {
			return cl, nil
		}

		// Handle input in Create/Edit modes
		if cl.mode == CommentModeCreate || cl.mode == CommentModeEdit {
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
				return cl, nil
			}

			// Delegate to textInput for normal typing
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
			// e, d will be added in later tasks
			}
		}
	}
	return cl, nil
}

// View renders the comments list to a string, dispatching to the correct render method.
func (cl CommentsList) View() string {
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

	// Build comment list
	var lines []string
	for i, comment := range cl.comments {
		// Format: "Author Name (YYYY-MM-DD)"
		dateStr := comment.Date.Format("2006-01-02")
		header := lipgloss.NewStyle().
			Bold(true).
			Render(comment.Author.FullName) +
			" " +
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("8")).
				Render("(" + dateStr + ")")

		// Body with word wrap
		body := wordWrap(comment.Body, cl.width-4)

		// Selection indicator (blue bar on left)
		indicator := " "
		if i == cl.selectedIdx {
			indicator = lipgloss.NewStyle().
				Foreground(lipgloss.Color("4")).
				Render("│")
		}

		lines = append(lines, indicator+" "+header)
		for _, line := range strings.Split(body, "\n") {
			lines = append(lines, "│ "+line)
		}
		lines = append(lines, "├"+strings.Repeat("─", cl.width-2))
	}

	// Render through viewport
	content := strings.Join(lines, "\n")
	cl.viewport.SetContent(content)

	// Add footer
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("Press 'c' to create, 'e' to edit, 'd' to delete")

	return cl.viewport.View() + "\n" + footer
}

// renderEditMode is a placeholder for edit mode rendering (Task 8)
func (cl CommentsList) renderEditMode() string {
	// TODO: Implement in Task 8
	return ""
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

	return title + "\n" + inputBox + "\n" + footer
}
