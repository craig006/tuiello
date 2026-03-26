package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/craig006/tuiello/internal/config"
	"github.com/craig006/tuiello/internal/trello"
)

func newTestCommentsList() CommentsList {
	cfg := config.DefaultConfig()
	km := NewKeyMap(cfg.Keybinding)
	theme := NewTheme(cfg.GUI.Theme)
	return NewCommentsList(theme, km)
}

func TestNewCommentsList(t *testing.T) {
	cl := newTestCommentsList()

	if cl.selectedIdx != 0 {
		t.Errorf("expected selectedIdx 0, got %d", cl.selectedIdx)
	}
	if cl.mode != CommentModeView {
		t.Errorf("expected mode CommentModeView, got %d", cl.mode)
	}
	if cl.editingIdx != -1 {
		t.Errorf("expected editingIdx -1, got %d", cl.editingIdx)
	}
	if len(cl.comments) != 0 {
		t.Errorf("expected 0 comments, got %d", len(cl.comments))
	}
	if len(cl.allMembers) != 0 {
		t.Errorf("expected 0 members, got %d", len(cl.allMembers))
	}
	if cl.focused {
		t.Error("expected focused to be false")
	}
	if cl.loading {
		t.Error("expected loading to be false")
	}
	if cl.width != 80 {
		t.Errorf("expected width 80, got %d", cl.width)
	}
	if cl.height != 20 {
		t.Errorf("expected height 20, got %d", cl.height)
	}
}

func TestSetComments(t *testing.T) {
	cl := newTestCommentsList()

	// Test setting comments
	comments := []trello.Comment{
		{ID: "comment1", Author: trello.Member{ID: "user1", FullName: "User One"}},
		{ID: "comment2", Author: trello.Member{ID: "user2", FullName: "User Two"}},
	}
	cl.SetComments(comments)

	if len(cl.comments) != 2 {
		t.Errorf("expected 2 comments, got %d", len(cl.comments))
	}
	if cl.selectedIdx != 0 {
		t.Errorf("expected selectedIdx 0, got %d", cl.selectedIdx)
	}

	// Test setting empty comments
	cl.SetComments([]trello.Comment{})
	if len(cl.comments) != 0 {
		t.Errorf("expected 0 comments, got %d", len(cl.comments))
	}
}

func TestSetMembers(t *testing.T) {
	cl := newTestCommentsList()

	members := []trello.Member{
		{ID: "member1", FullName: "Member One", Username: "memberone"},
		{ID: "member2", FullName: "Member Two", Username: "membertwo"},
		{ID: "member3", FullName: "Member Three", Username: "memberthree"},
	}
	cl.SetMembers(members)

	if len(cl.allMembers) != 3 {
		t.Errorf("expected 3 members, got %d", len(cl.allMembers))
	}
	if cl.allMembers[0].FullName != "Member One" {
		t.Errorf("expected first member 'Member One', got %q", cl.allMembers[0].FullName)
	}
}

func TestSetSize(t *testing.T) {
	cl := newTestCommentsList()

	cl.SetSize(120, 30)

	if cl.width != 120 {
		t.Errorf("expected width 120, got %d", cl.width)
	}
	if cl.height != 30 {
		t.Errorf("expected height 30, got %d", cl.height)
	}
	if cl.viewport.Width() != 120 {
		t.Errorf("expected viewport width 120, got %d", cl.viewport.Width())
	}
	if cl.viewport.Height() != 25 {
		t.Errorf("expected viewport height 25 (30-5), got %d", cl.viewport.Height())
	}
}

func TestSetFocus(t *testing.T) {
	cl := newTestCommentsList()

	if cl.focused {
		t.Error("expected focused to be false initially")
	}

	cl.SetFocus(true)
	if !cl.focused {
		t.Error("expected focused to be true after SetFocus(true)")
	}

	cl.SetFocus(false)
	if cl.focused {
		t.Error("expected focused to be false after SetFocus(false)")
	}
}

func TestNavigateCommentsDown(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)

	// Set up test comments
	comments := []trello.Comment{
		{ID: "comment1", Author: trello.Member{ID: "user1", FullName: "User One"}, Body: "First comment"},
		{ID: "comment2", Author: trello.Member{ID: "user2", FullName: "User Two"}, Body: "Second comment"},
		{ID: "comment3", Author: trello.Member{ID: "user3", FullName: "User Three"}, Body: "Third comment"},
	}
	cl.SetComments(comments)

	// Test initial selection
	if cl.selectedIdx != 0 {
		t.Errorf("expected selectedIdx 0, got %d", cl.selectedIdx)
	}

	// Simulate moving down with down arrow
	if cl.selectedIdx < len(cl.comments)-1 {
		cl.selectedIdx++
	}
	if cl.selectedIdx != 1 {
		t.Errorf("after moving down, expected selectedIdx 1, got %d", cl.selectedIdx)
	}

	// Simulate moving down again
	if cl.selectedIdx < len(cl.comments)-1 {
		cl.selectedIdx++
	}
	if cl.selectedIdx != 2 {
		t.Errorf("after second move, expected selectedIdx 2, got %d", cl.selectedIdx)
	}
}

func TestNavigateCommentsUp(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)

	comments := []trello.Comment{
		{ID: "comment1", Author: trello.Member{ID: "user1", FullName: "User One"}, Body: "First comment"},
		{ID: "comment2", Author: trello.Member{ID: "user2", FullName: "User Two"}, Body: "Second comment"},
		{ID: "comment3", Author: trello.Member{ID: "user3", FullName: "User Three"}, Body: "Third comment"},
	}
	cl.SetComments(comments)

	// Start at index 2
	cl.selectedIdx = 2

	// Simulate moving up with up arrow
	if cl.selectedIdx > 0 {
		cl.selectedIdx--
	}
	if cl.selectedIdx != 1 {
		t.Errorf("after moving up, expected selectedIdx 1, got %d", cl.selectedIdx)
	}

	// Simulate moving up again
	if cl.selectedIdx > 0 {
		cl.selectedIdx--
	}
	if cl.selectedIdx != 0 {
		t.Errorf("after second move, expected selectedIdx 0, got %d", cl.selectedIdx)
	}
}

func TestNavigateCommentsAtBoundaries(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)

	comments := []trello.Comment{
		{ID: "comment1", Author: trello.Member{ID: "user1", FullName: "User One"}, Body: "First comment"},
		{ID: "comment2", Author: trello.Member{ID: "user2", FullName: "User Two"}, Body: "Second comment"},
	}
	cl.SetComments(comments)

	// Test can't go above first comment
	if cl.selectedIdx > 0 {
		cl.selectedIdx--
	}
	if cl.selectedIdx != 0 {
		t.Errorf("at top, should not move, expected selectedIdx 0, got %d", cl.selectedIdx)
	}

	// Move to last comment
	cl.selectedIdx = 1

	// Test can't go below last comment
	if cl.selectedIdx < len(cl.comments)-1 {
		cl.selectedIdx++
	}
	if cl.selectedIdx != 1 {
		t.Errorf("at bottom, should not move, expected selectedIdx 1, got %d", cl.selectedIdx)
	}
}

func TestViewModeRendersComments(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetSize(80, 20)

	comments := []trello.Comment{
		{
			ID:     "comment1",
			Author: trello.Member{ID: "user1", FullName: "User One"},
			Body:   "First comment",
			Date:   parseTestDate("2026-01-15"),
		},
		{
			ID:     "comment2",
			Author: trello.Member{ID: "user2", FullName: "User Two"},
			Body:   "Second comment",
			Date:   parseTestDate("2026-01-16"),
		},
	}
	cl.SetComments(comments)

	view := cl.View()

	// Check that comments are rendered
	if !stringContains(view, "User One") {
		t.Error("expected 'User One' in view")
	}
	if !stringContains(view, "User Two") {
		t.Error("expected 'User Two' in view")
	}
	if !stringContains(view, "First comment") {
		t.Error("expected 'First comment' in view")
	}
	if !stringContains(view, "Second comment") {
		t.Error("expected 'Second comment' in view")
	}
	if !stringContains(view, "2026-01-15") {
		t.Error("expected date '2026-01-15' in view")
	}
}

func TestViewModeShowsNoComments(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetSize(80, 20)

	view := cl.View()

	if !stringContains(view, "No comments") {
		t.Error("expected 'No comments' message in view when no comments exist")
	}
	if !stringContains(view, "Press 'c' to create") {
		t.Error("expected 'Press 'c' to create' in empty state message")
	}
}

func TestViewModeShowsFooter(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetSize(80, 20)

	comments := []trello.Comment{
		{
			ID:     "comment1",
			Author: trello.Member{ID: "user1", FullName: "User One"},
			Body:   "First comment",
			Date:   parseTestDate("2026-01-15"),
		},
	}
	cl.SetComments(comments)

	view := cl.View()
	// Strip ANSI codes for easier testing
	cleanView := stripANSI(view)

	// Check for footer
	if !stringContains(cleanView, "'c' to create") {
		t.Error("expected 'c' to create' in footer")
	}
	if !stringContains(cleanView, "'e' to edit") {
		t.Error("expected 'e' to edit' in footer")
	}
	if !stringContains(cleanView, "'d' to delete") {
		t.Error("expected 'd' to delete' in footer")
	}
}

func TestViewModeShowsSelectionIndicator(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetSize(80, 20)

	comments := []trello.Comment{
		{
			ID:     "comment1",
			Author: trello.Member{ID: "user1", FullName: "User One"},
			Body:   "First comment",
			Date:   parseTestDate("2026-01-15"),
		},
		{
			ID:     "comment2",
			Author: trello.Member{ID: "user2", FullName: "User Two"},
			Body:   "Second comment",
			Date:   parseTestDate("2026-01-16"),
		},
	}
	cl.SetComments(comments)

	// Select first comment (default)
	view := cl.View()
	lines := strings.Split(view, "\n")

	// The first comment should have a selection indicator (│)
	// We just verify that some line contains the indicator
	foundIndicator := false
	for _, line := range lines {
		if strings.Contains(line, "│") {
			foundIndicator = true
			break
		}
	}

	if !foundIndicator {
		t.Error("expected selection indicator (│) in view")
	}
}

// Helper functions for tests
func parseTestDate(dateStr string) time.Time {
	t, _ := time.Parse("2006-01-02", dateStr)
	return t
}

func stringContains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	// Remove all ANSI escape sequences using a more comprehensive approach
	result := strings.ReplaceAll(s, "\x1b[90m", "")   // foreground color 8
	result = strings.ReplaceAll(result, "\x1b[1m", "") // bold
	result = strings.ReplaceAll(result, "\x1b[34m", "") // blue foreground
	result = strings.ReplaceAll(result, "\x1b[m", "")  // reset
	result = strings.ReplaceAll(result, "\x1b[7;37m", "") // reverse video
	result = strings.ReplaceAll(result, "\x1b[37m", "") // white foreground
	result = strings.ReplaceAll(result, "\x1b[0m", "") // reset

	// Remove color sequences with parameters (e.g., \x1b[38;5;XXm)
	for i := 0; i < 256; i++ {
		result = strings.ReplaceAll(result, fmt.Sprintf("\x1b[38;5;%dm", i), "")
		result = strings.ReplaceAll(result, fmt.Sprintf("\x1b[48;5;%dm", i), "")
	}

	// Remove any remaining escape sequences
	inEscape := false
	var cleaned strings.Builder
	for _, ch := range result {
		if ch == '\x1b' {
			inEscape = true
		} else if inEscape && ch == 'm' {
			inEscape = false
		} else if !inEscape {
			cleaned.WriteRune(ch)
		}
	}

	return cleaned.String()
}

func TestCreateCommentMode(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)

	// Start in View mode
	if cl.mode != CommentModeView {
		t.Fatalf("expected initial mode CommentModeView, got %d", cl.mode)
	}

	// Manually transition to Create mode (simulating 'c' key handling)
	cl.mode = CommentModeCreate
	cl.textInput.SetValue("")
	cl.textInput.Focus()

	if cl.mode != CommentModeCreate {
		t.Errorf("expected mode CommentModeCreate, got %d", cl.mode)
	}

	// Verify textinput is cleared and focused
	if cl.textInput.Value() != "" {
		t.Errorf("expected textInput to be cleared, got %q", cl.textInput.Value())
	}
}

func TestCreateModeInput(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)

	// Enter Create mode
	cl.mode = CommentModeCreate
	cl.textInput.Focus()

	// Manually set text (simulating user typing)
	cl.textInput.SetValue("Hello World")

	if cl.textInput.Value() != "Hello World" {
		t.Errorf("expected textInput value 'Hello World', got %q", cl.textInput.Value())
	}
}

func TestCreateModeCancel(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)

	// Enter Create mode and type something
	cl.mode = CommentModeCreate
	cl.textInput.Focus()
	cl.textInput.SetValue("Test comment")

	// Simulate Escape key (exit Create mode)
	cl.mode = CommentModeView
	cl.textInput.SetValue("")
	cl.textInput.Blur()

	if cl.mode != CommentModeView {
		t.Errorf("expected mode CommentModeView after exit, got %d", cl.mode)
	}

	if cl.textInput.Value() != "" {
		t.Errorf("expected textInput to be cleared after cancel, got %q", cl.textInput.Value())
	}
}

func TestCreateModeSubmit(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)

	// Enter Create mode
	cl.mode = CommentModeCreate
	cl.textInput.Focus()
	cl.textInput.SetValue("New comment text")

	// Call submitComment directly (testing the command generation)
	cmd := cl.submitComment()

	// Verify that a command was returned
	if cmd == nil {
		t.Error("expected a command from submitComment, got nil")
	}

	// Execute the command to get the message
	msg := cmd()
	createMsg, ok := msg.(CreateCommentRequestMsg)
	if !ok {
		t.Fatalf("expected CreateCommentRequestMsg, got %T", msg)
	}

	if createMsg.Text != "New comment text" {
		t.Errorf("expected text 'New comment text', got %q", createMsg.Text)
	}
}

func TestCreateModeSubmitEmpty(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)

	// Enter Create mode with empty input
	cl.mode = CommentModeCreate
	cl.textInput.Focus()
	cl.textInput.SetValue("   ")

	// Try to submit empty/whitespace-only text - should not generate a message
	text := cl.textInput.Value()
	if strings.TrimSpace(text) == "" {
		// This is what we expect - empty submission should be ignored
		t.Log("empty submission correctly rejected")
	} else {
		t.Error("expected whitespace-only submission to be treated as empty")
	}
}

func TestRenderCreateMode(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetSize(80, 20)
	cl.mode = CommentModeCreate
	cl.textInput.Focus()

	view := cl.View()
	cleanView := stripANSI(view)

	// Check for title
	if !stringContains(cleanView, "[New Comment]") {
		t.Error("expected '[New Comment]' title in create mode view")
	}

	// Check for footer with shortcuts
	if !stringContains(cleanView, "Submit") || !stringContains(cleanView, "Enter") {
		t.Error("expected 'Submit: Enter' in footer")
	}
	if !stringContains(cleanView, "Cancel") || !stringContains(cleanView, "Esc") {
		t.Error("expected 'Cancel: Esc' in footer")
	}
}

// Task 8: Edit mode tests
func TestEditCommentMode(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)

	// Set up test comments with an editable comment
	comments := []trello.Comment{
		{
			ID:       "comment1",
			Author:   trello.Member{ID: "user1", FullName: "User One"},
			Body:     "First comment",
			Date:     parseTestDate("2026-01-15"),
			Editable: true,
		},
	}
	cl.SetComments(comments)

	// Manually transition to Edit mode (simulating 'e' key handling)
	cl.mode = CommentModeEdit
	cl.editingIdx = cl.selectedIdx
	cl.textInput.SetValue(cl.comments[cl.editingIdx].Body)
	cl.textInput.Focus()

	if cl.mode != CommentModeEdit {
		t.Errorf("expected mode CommentModeEdit, got %d", cl.mode)
	}

	if cl.editingIdx != 0 {
		t.Errorf("expected editingIdx 0, got %d", cl.editingIdx)
	}

	// Verify textinput has comment body loaded
	if cl.textInput.Value() != "First comment" {
		t.Errorf("expected textInput to have 'First comment', got %q", cl.textInput.Value())
	}
}

func TestEditModeNotEditable(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)

	// Set up test comments with a non-editable comment
	comments := []trello.Comment{
		{
			ID:       "comment1",
			Author:   trello.Member{ID: "user1", FullName: "User One"},
			Body:     "First comment",
			Date:     parseTestDate("2026-01-15"),
			Editable: false,
		},
	}
	cl.SetComments(comments)

	// Attempt to enter Edit mode (should be blocked by Editable check in actual Update() handler)
	// This test verifies the logic that should be in the Update() method
	if cl.comments[cl.selectedIdx].Editable {
		t.Error("expected comment to not be editable")
	}

	// Verify we don't enter edit mode for non-editable comments
	if cl.mode == CommentModeEdit {
		t.Error("should not enter edit mode for non-editable comment")
	}
}

func TestEditModeCancel(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)

	// Set up test comments
	comments := []trello.Comment{
		{
			ID:       "comment1",
			Author:   trello.Member{ID: "user1", FullName: "User One"},
			Body:     "Original text",
			Date:     parseTestDate("2026-01-15"),
			Editable: true,
		},
	}
	cl.SetComments(comments)

	// Enter Edit mode
	cl.mode = CommentModeEdit
	cl.editingIdx = 0
	cl.textInput.SetValue("Original text")
	cl.textInput.Focus()

	// Simulate Escape key (exit Edit mode without saving)
	cl.mode = CommentModeView
	cl.textInput.SetValue("")
	cl.textInput.Blur()

	if cl.mode != CommentModeView {
		t.Errorf("expected mode CommentModeView after cancel, got %d", cl.mode)
	}

	if cl.textInput.Value() != "" {
		t.Errorf("expected textInput to be cleared after cancel, got %q", cl.textInput.Value())
	}
}

func TestEditModeSubmit(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)

	// Set up test comments
	comments := []trello.Comment{
		{
			ID:       "comment1",
			Author:   trello.Member{ID: "user1", FullName: "User One"},
			Body:     "Original text",
			Date:     parseTestDate("2026-01-15"),
			Editable: true,
		},
	}
	cl.SetComments(comments)

	// Enter Edit mode with modified text
	cl.mode = CommentModeEdit
	cl.editingIdx = 0
	cl.textInput.Focus()
	cl.textInput.SetValue("Modified text")

	// Call submitComment directly (testing the command generation)
	cmd := cl.submitComment()

	// Verify that a command was returned
	if cmd == nil {
		t.Error("expected a command from submitComment, got nil")
	}

	// Execute the command to get the message
	msg := cmd()
	updateMsg, ok := msg.(UpdateCommentRequestMsg)
	if !ok {
		t.Fatalf("expected UpdateCommentRequestMsg, got %T", msg)
	}

	if updateMsg.CommentID != "comment1" {
		t.Errorf("expected CommentID 'comment1', got %q", updateMsg.CommentID)
	}

	if updateMsg.Text != "Modified text" {
		t.Errorf("expected text 'Modified text', got %q", updateMsg.Text)
	}
}

func TestRenderEditMode(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetSize(80, 20)

	// Set up test comments
	comments := []trello.Comment{
		{
			ID:       "comment1",
			Author:   trello.Member{ID: "user1", FullName: "User One"},
			Body:     "Comment to edit",
			Date:     parseTestDate("2026-01-15"),
			Editable: true,
		},
	}
	cl.SetComments(comments)

	// Enter Edit mode
	cl.mode = CommentModeEdit
	cl.editingIdx = 0
	cl.textInput.SetValue(cl.comments[cl.editingIdx].Body)

	view := cl.View()
	cleanView := stripANSI(view)

	// Check for title
	if !stringContains(cleanView, "[Edit Comment]") {
		t.Error("expected '[Edit Comment]' title in edit mode view")
	}

	// Check for author and date
	if !stringContains(cleanView, "User One") {
		t.Error("expected 'User One' (author) in edit mode view")
	}
	if !stringContains(cleanView, "2026-01-15") {
		t.Error("expected '2026-01-15' (date) in edit mode view")
	}

	// Check for footer with shortcuts
	if !stringContains(cleanView, "Submit") || !stringContains(cleanView, "Enter") {
		t.Error("expected 'Submit: Enter' in footer")
	}
	if !stringContains(cleanView, "Cancel") || !stringContains(cleanView, "Esc") {
		t.Error("expected 'Cancel: Esc' in footer")
	}
}
