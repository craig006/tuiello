package tui

import (
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
// Currently minimal - will be expanded in later tasks.
func (cl CommentsList) Update(msg tea.Msg) (CommentsList, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		if !cl.focused {
			return cl, nil
		}
		// Key handling will be filled in by later tasks
	}
	return cl, nil
}

// View renders the comments list to a string.
// Currently minimal - will be expanded in later tasks.
func (cl CommentsList) View() string {
	if len(cl.comments) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("No comments")
	}
	return "Comments: " + lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Render("TODO")
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
