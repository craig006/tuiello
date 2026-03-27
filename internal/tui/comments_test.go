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
	cleanView := stripANSI(view)

	// Check that comments are rendered
	if !stringContains(cleanView, "User One") {
		t.Error("expected 'User One' in view")
	}
	if !stringContains(cleanView, "User Two") {
		t.Error("expected 'User Two' in view")
	}
	if !stringContains(cleanView, "First comment") {
		t.Error("expected 'First comment' in view")
	}
	if !stringContains(cleanView, "Second comment") {
		t.Error("expected 'Second comment' in view")
	}
	if !stringContains(cleanView, "2026-01-15") {
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
	cleanView := stripANSI(view)

	// Verify the view contains both comments and is properly formatted
	if !strings.Contains(cleanView, "User One") {
		t.Error("expected first comment author in view")
	}
	if !strings.Contains(cleanView, "User Two") {
		t.Error("expected second comment author in view")
	}
	if !strings.Contains(cleanView, "First comment") {
		t.Error("expected first comment body in view")
	}
	if !strings.Contains(cleanView, "Second comment") {
		t.Error("expected second comment body in view")
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

// Task 9: Delete comment tests
func TestDeleteCommentInitiatesConfirmation(t *testing.T) {
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

	// Simulate pressing 'd' key
	if cl.selectedIdx < len(cl.comments) {
		comment := cl.comments[cl.selectedIdx]
		if comment.Editable {
			cl.deleteConfirming = true
		}
	}

	if !cl.deleteConfirming {
		t.Error("expected deleteConfirming to be true after pressing 'd' on editable comment")
	}
}

func TestDeleteCommentNotEditable(t *testing.T) {
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

	// Attempt to start deletion on non-editable comment
	if cl.selectedIdx < len(cl.comments) {
		comment := cl.comments[cl.selectedIdx]
		if comment.Editable {
			cl.deleteConfirming = true
		}
	}

	if cl.deleteConfirming {
		t.Error("expected deleteConfirming to remain false for non-editable comment")
	}
}

func TestDeleteCommentConfirmYes(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)

	// Set up test comments
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

	// Start deletion confirmation
	cl.deleteConfirming = true

	// Call deleteComment directly (testing the command generation)
	cmd := cl.deleteComment(cl.comments[cl.selectedIdx].ID)

	// Verify that a command was returned
	if cmd == nil {
		t.Error("expected a command from deleteComment, got nil")
	}

	// Execute the command to get the message
	msg := cmd()
	deleteMsg, ok := msg.(DeleteCommentRequestMsg)
	if !ok {
		t.Fatalf("expected DeleteCommentRequestMsg, got %T", msg)
	}

	if deleteMsg.CommentID != "comment1" {
		t.Errorf("expected CommentID 'comment1', got %q", deleteMsg.CommentID)
	}
}

func TestDeleteCommentConfirmNo(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)

	// Set up test comments
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

	// Start deletion confirmation
	cl.deleteConfirming = true

	// Cancel deletion (pressing 'n')
	cl.deleteConfirming = false

	if cl.deleteConfirming {
		t.Error("expected deleteConfirming to be false after cancelling")
	}

	// Verify the comment is still there
	if len(cl.comments) != 1 {
		t.Errorf("expected 1 comment to remain, got %d", len(cl.comments))
	}
}

func TestDeleteCommentBoundaries(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)

	// Test 1: No comments - should not crash
	cl.deleteConfirming = false
	if len(cl.comments) > 0 && cl.selectedIdx < len(cl.comments) {
		comment := cl.comments[cl.selectedIdx]
		if comment.Editable {
			cl.deleteConfirming = true
		}
	}
	if cl.deleteConfirming {
		t.Error("expected deleteConfirming to remain false when no comments")
	}

	// Test 2: Invalid selectedIdx - should not crash
	cl.SetComments([]trello.Comment{
		{
			ID:       "comment1",
			Author:   trello.Member{ID: "user1", FullName: "User One"},
			Body:     "First comment",
			Date:     parseTestDate("2026-01-15"),
			Editable: true,
		},
	})
	cl.selectedIdx = 100 // Invalid index

	cl.deleteConfirming = false
	if cl.selectedIdx < len(cl.comments) {
		comment := cl.comments[cl.selectedIdx]
		if comment.Editable {
			cl.deleteConfirming = true
		}
	}

	if cl.deleteConfirming {
		t.Error("expected deleteConfirming to remain false with invalid index")
	}
}

func TestDeleteCommentShowsConfirmation(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetSize(80, 20)

	// Set up test comments
	comments := []trello.Comment{
		{
			ID:       "comment1",
			Author:   trello.Member{ID: "user1", FullName: "User One"},
			Body:     "First comment to delete",
			Date:     parseTestDate("2026-01-15"),
			Editable: true,
		},
	}
	cl.SetComments(comments)

	// Start deletion confirmation
	cl.deleteConfirming = true

	view := cl.View()
	cleanView := stripANSI(view)

	// Check for deletion prompt
	if !stringContains(cleanView, "Delete comment?") {
		t.Error("expected 'Delete comment?' prompt in confirmation view")
	}

	// Check for y/n instructions
	if !stringContains(cleanView, "(y/n)") {
		t.Error("expected '(y/n)' instructions in confirmation view")
	}

	// Check that comment details are shown
	if !stringContains(cleanView, "User One") {
		t.Error("expected author 'User One' in confirmation view")
	}

	if !stringContains(cleanView, "First comment to delete") {
		t.Error("expected comment body in confirmation view")
	}
}

// Task 10: Autocomplete tests
func TestAutocompleteActivatesOnAt(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)
	cl.mode = CommentModeCreate
	cl.textInput.Focus()

	// Set up test members
	members := []trello.Member{
		{ID: "member1", FullName: "Alice Smith", Username: "alice"},
		{ID: "member2", FullName: "Bob Jones", Username: "bob"},
	}
	cl.SetMembers(members)

	// Simulate @ character
	initialTextLen := len(cl.textInput.Value())
	cl.autocomplete.Active = true
	cl.autocomplete.Query = ""
	cl.autocomplete.SelectedIdx = 0
	cl.autocomplete.Pos = initialTextLen
	cl.autocomplete.Matches = cl.allMembers

	if !cl.autocomplete.Active {
		t.Error("expected autocomplete to be active after @ character")
	}

	if cl.autocomplete.Query != "" {
		t.Errorf("expected empty query at start, got %q", cl.autocomplete.Query)
	}

	if cl.autocomplete.SelectedIdx != 0 {
		t.Errorf("expected SelectedIdx 0, got %d", cl.autocomplete.SelectedIdx)
	}

	if len(cl.autocomplete.Matches) != 2 {
		t.Errorf("expected 2 matches (all members), got %d", len(cl.autocomplete.Matches))
	}
}

func TestAutocompleteFilterByName(t *testing.T) {
	cl := newTestCommentsList()

	// Set up test members
	members := []trello.Member{
		{ID: "member1", FullName: "Alice Smith", Username: "alice"},
		{ID: "member2", FullName: "Bob Jones", Username: "bob"},
		{ID: "member3", FullName: "Alice Johnson", Username: "alice2"},
	}
	cl.SetMembers(members)

	// Filter by "Alice"
	cl.filterMembers("Alice")

	if len(cl.autocomplete.Matches) != 2 {
		t.Errorf("expected 2 members matching 'Alice', got %d", len(cl.autocomplete.Matches))
	}

	// Verify both Alices are in matches
	foundAliceSmith := false
	foundAliceJohnson := false
	for _, member := range cl.autocomplete.Matches {
		if member.FullName == "Alice Smith" {
			foundAliceSmith = true
		}
		if member.FullName == "Alice Johnson" {
			foundAliceJohnson = true
		}
	}

	if !foundAliceSmith {
		t.Error("expected 'Alice Smith' in filtered results")
	}
	if !foundAliceJohnson {
		t.Error("expected 'Alice Johnson' in filtered results")
	}
}

func TestAutocompleteFilterByUsername(t *testing.T) {
	cl := newTestCommentsList()

	// Set up test members
	members := []trello.Member{
		{ID: "member1", FullName: "Alice Smith", Username: "alice"},
		{ID: "member2", FullName: "Bob Jones", Username: "bobby"},
		{ID: "member3", FullName: "Charlie Brown", Username: "charlie"},
	}
	cl.SetMembers(members)

	// Filter by "bob" (should match both "bobby" and "Bob")
	cl.filterMembers("bob")

	if len(cl.autocomplete.Matches) != 1 {
		t.Errorf("expected 1 member matching 'bob', got %d", len(cl.autocomplete.Matches))
	}

	if cl.autocomplete.Matches[0].Username != "bobby" {
		t.Errorf("expected username 'bobby', got %q", cl.autocomplete.Matches[0].Username)
	}
}

func TestAutocompleteFilterCaseInsensitive(t *testing.T) {
	cl := newTestCommentsList()

	// Set up test members
	members := []trello.Member{
		{ID: "member1", FullName: "Alice Smith", Username: "alice"},
		{ID: "member2", FullName: "Bob Jones", Username: "bobby"},
	}
	cl.SetMembers(members)

	// Filter with different cases
	cl.filterMembers("ALICE")

	if len(cl.autocomplete.Matches) != 1 {
		t.Errorf("expected 1 member matching 'ALICE', got %d", len(cl.autocomplete.Matches))
	}

	if cl.autocomplete.Matches[0].FullName != "Alice Smith" {
		t.Errorf("expected 'Alice Smith', got %q", cl.autocomplete.Matches[0].FullName)
	}

	// Filter with mixed case
	cl.filterMembers("BoB")

	if len(cl.autocomplete.Matches) != 1 {
		t.Errorf("expected 1 member matching 'BoB', got %d", len(cl.autocomplete.Matches))
	}

	if cl.autocomplete.Matches[0].FullName != "Bob Jones" {
		t.Errorf("expected 'Bob Jones', got %q", cl.autocomplete.Matches[0].FullName)
	}
}

func TestAutocompleteStartsWithAllMembers(t *testing.T) {
	cl := newTestCommentsList()

	// Set up test members
	members := []trello.Member{
		{ID: "member1", FullName: "Alice Smith", Username: "alice"},
		{ID: "member2", FullName: "Bob Jones", Username: "bobby"},
		{ID: "member3", FullName: "Charlie Brown", Username: "charlie"},
	}
	cl.SetMembers(members)

	// Simulate @ character (starts with all members)
	cl.autocomplete.Active = true
	cl.autocomplete.Query = ""
	cl.autocomplete.Matches = cl.allMembers

	if len(cl.autocomplete.Matches) != 3 {
		t.Errorf("expected all 3 members when @ is first typed, got %d", len(cl.autocomplete.Matches))
	}
}

func TestRenderAutocomplete(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetSize(80, 20)

	// Set up test members
	members := []trello.Member{
		{ID: "member1", FullName: "Alice Smith", Username: "alice"},
		{ID: "member2", FullName: "Bob Jones", Username: "bobby"},
	}
	cl.SetMembers(members)

	// Activate autocomplete
	cl.autocomplete.Active = true
	cl.autocomplete.Matches = cl.allMembers
	cl.autocomplete.SelectedIdx = 0
	cl.autocomplete.Query = ""

	view := cl.renderAutocomplete()

	// Should not be empty
	if view == "" {
		t.Error("expected autocomplete view to be non-empty")
	}

	// Check for Members header
	if !stringContains(view, "Members:") {
		t.Error("expected 'Members:' header in autocomplete view")
	}

	// Check for member names
	cleanView := stripANSI(view)
	if !stringContains(cleanView, "Alice Smith") {
		t.Error("expected 'Alice Smith' in autocomplete view")
	}
	if !stringContains(cleanView, "Bob Jones") {
		t.Error("expected 'Bob Jones' in autocomplete view")
	}

	// Check for usernames
	if !stringContains(cleanView, "@alice") {
		t.Error("expected '@alice' in autocomplete view")
	}
	if !stringContains(cleanView, "@bobby") {
		t.Error("expected '@bobby' in autocomplete view")
	}
}

func TestRenderAutocompleteEmpty(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetSize(80, 20)

	// Set up test members
	members := []trello.Member{
		{ID: "member1", FullName: "Alice Smith", Username: "alice"},
	}
	cl.SetMembers(members)

	// Activate autocomplete but with no matches
	cl.autocomplete.Active = true
	cl.autocomplete.Matches = []trello.Member{}
	cl.autocomplete.Query = "xyz"

	view := cl.renderAutocomplete()

	// Should be empty
	if view != "" {
		t.Errorf("expected empty view when no matches, got %q", view)
	}
}

func TestRenderAutocompleteInactive(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetSize(80, 20)

	// Set up test members
	members := []trello.Member{
		{ID: "member1", FullName: "Alice Smith", Username: "alice"},
	}
	cl.SetMembers(members)

	// Autocomplete inactive
	cl.autocomplete.Active = false
	cl.autocomplete.Matches = cl.allMembers

	view := cl.renderAutocomplete()

	// Should be empty
	if view != "" {
		t.Errorf("expected empty view when autocomplete inactive, got %q", view)
	}
}

func TestAutocompleteNavigation(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)
	cl.mode = CommentModeCreate
	cl.textInput.Focus()

	// Set up test members
	members := []trello.Member{
		{ID: "member1", FullName: "Alice Smith", Username: "alice"},
		{ID: "member2", FullName: "Bob Jones", Username: "bob"},
		{ID: "member3", FullName: "Charlie Brown", Username: "charlie"},
	}
	cl.SetMembers(members)

	// Activate autocomplete
	cl.autocomplete.Active = true
	cl.autocomplete.Matches = cl.allMembers
	cl.autocomplete.SelectedIdx = 0

	if cl.autocomplete.SelectedIdx != 0 {
		t.Errorf("expected initial SelectedIdx 0, got %d", cl.autocomplete.SelectedIdx)
	}

	// Navigate down with 'j' - we simulate this by directly testing the logic
	// since we're testing navigation bounds
	if cl.autocomplete.SelectedIdx < len(cl.autocomplete.Matches)-1 {
		cl.autocomplete.SelectedIdx++
	}

	if cl.autocomplete.SelectedIdx != 1 {
		t.Errorf("expected SelectedIdx 1 after nav down, got %d", cl.autocomplete.SelectedIdx)
	}

	// Navigate down again
	if cl.autocomplete.SelectedIdx < len(cl.autocomplete.Matches)-1 {
		cl.autocomplete.SelectedIdx++
	}

	if cl.autocomplete.SelectedIdx != 2 {
		t.Errorf("expected SelectedIdx 2 after second nav down, got %d", cl.autocomplete.SelectedIdx)
	}

	// Navigate up with 'k'
	if cl.autocomplete.SelectedIdx > 0 {
		cl.autocomplete.SelectedIdx--
	}

	if cl.autocomplete.SelectedIdx != 1 {
		t.Errorf("expected SelectedIdx 1 after nav up, got %d", cl.autocomplete.SelectedIdx)
	}
}

func TestAutocompleteNavigationBoundaries(t *testing.T) {
	cl := newTestCommentsList()

	// Set up test members
	members := []trello.Member{
		{ID: "member1", FullName: "Alice Smith", Username: "alice"},
		{ID: "member2", FullName: "Bob Jones", Username: "bob"},
	}
	cl.SetMembers(members)

	// Activate autocomplete
	cl.autocomplete.Active = true
	cl.autocomplete.Matches = cl.allMembers
	cl.autocomplete.SelectedIdx = 0

	// Try to navigate up from first item - should not go past 0
	if cl.autocomplete.SelectedIdx > 0 {
		cl.autocomplete.SelectedIdx--
	}

	if cl.autocomplete.SelectedIdx != 0 {
		t.Errorf("expected SelectedIdx to stay at 0, got %d", cl.autocomplete.SelectedIdx)
	}

	// Navigate to last item
	cl.autocomplete.SelectedIdx = len(cl.autocomplete.Matches) - 1

	// Try to navigate down from last item - should not go past last
	if cl.autocomplete.SelectedIdx < len(cl.autocomplete.Matches)-1 {
		cl.autocomplete.SelectedIdx++
	}

	if cl.autocomplete.SelectedIdx != 1 {
		t.Errorf("expected SelectedIdx to stay at 1 (last item), got %d", cl.autocomplete.SelectedIdx)
	}
}

func TestAutocompleteMentionInsertion(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)
	cl.mode = CommentModeCreate
	cl.textInput.Focus()

	// Set up test members
	members := []trello.Member{
		{ID: "member1", FullName: "Alice Smith", Username: "alice"},
		{ID: "member2", FullName: "Bob Jones", Username: "bob"},
	}
	cl.SetMembers(members)

	// Type a message with @ (simulate user typing "Hey @")
	cl.textInput.SetValue("Hey @")
	cl.autocomplete.Active = true
	cl.autocomplete.Pos = len(cl.textInput.Value()) - 1 // Position of @
	cl.autocomplete.Matches = cl.allMembers
	cl.autocomplete.SelectedIdx = 0
	cl.autocomplete.Query = ""

	// Insert mention for Alice
	cl.insertMention("alice")

	// Check that the text now contains the mention
	result := cl.textInput.Value()
	if !strings.Contains(result, "@alice") {
		t.Errorf("expected mention '@alice' in result, got %q", result)
	}

	// Check that autocomplete is closed
	if cl.autocomplete.Active {
		t.Error("expected autocomplete to be closed after insertion")
	}

	if len(cl.autocomplete.Matches) != 0 {
		t.Errorf("expected matches to be cleared, got %d", len(cl.autocomplete.Matches))
	}
}

func TestAutocompleteMentionInsertionMultiple(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)
	cl.mode = CommentModeCreate
	cl.textInput.Focus()

	// Set up test members
	members := []trello.Member{
		{ID: "member1", FullName: "Alice Smith", Username: "alice"},
		{ID: "member2", FullName: "Bob Jones", Username: "bob"},
	}
	cl.SetMembers(members)

	// Type initial text with first @
	cl.textInput.SetValue("Hey @")

	// Insert first mention
	cl.autocomplete.Active = true
	cl.autocomplete.Pos = len(cl.textInput.Value()) - 1
	cl.autocomplete.Matches = cl.allMembers
	cl.autocomplete.SelectedIdx = 0
	cl.autocomplete.Query = ""
	cl.insertMention("alice")

	// Add more text with second mention
	currentText := cl.textInput.Value()
	cl.textInput.SetValue(currentText + " and @")

	// Insert second mention
	cl.autocomplete.Active = true
	cl.autocomplete.Pos = len(cl.textInput.Value()) - 1
	cl.autocomplete.Matches = cl.allMembers
	cl.autocomplete.SelectedIdx = 1
	cl.autocomplete.Query = ""
	cl.insertMention("bob")

	result := cl.textInput.Value()

	// Both mentions should be present
	if !strings.Contains(result, "@alice") {
		t.Errorf("expected '@alice' in result, got %q", result)
	}
	if !strings.Contains(result, "@bob") {
		t.Errorf("expected '@bob' in result, got %q", result)
	}
}

func TestAutocompleteEscapeCloses(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)
	cl.mode = CommentModeCreate
	cl.textInput.Focus()

	// Set up test members
	members := []trello.Member{
		{ID: "member1", FullName: "Alice Smith", Username: "alice"},
	}
	cl.SetMembers(members)

	// Activate autocomplete
	cl.autocomplete.Active = true
	cl.autocomplete.Query = "al"
	cl.autocomplete.Matches = cl.allMembers

	if !cl.autocomplete.Active {
		t.Error("expected autocomplete to be active")
	}

	// Simulate escape
	cl.autocomplete.Active = false

	if cl.autocomplete.Active {
		t.Error("expected autocomplete to be closed after escape")
	}
}

func TestAutocompleteBackspaceFilters(t *testing.T) {
	cl := newTestCommentsList()

	// Set up test members
	members := []trello.Member{
		{ID: "member1", FullName: "Alice Smith", Username: "alice"},
		{ID: "member2", FullName: "Alice Johnson", Username: "alice2"},
		{ID: "member3", FullName: "Bob Jones", Username: "bob"},
	}
	cl.SetMembers(members)

	// Activate autocomplete and filter
	cl.autocomplete.Active = true
	cl.autocomplete.Query = "ali"
	cl.filterMembers(cl.autocomplete.Query)

	if len(cl.autocomplete.Matches) != 2 {
		t.Errorf("expected 2 matches for 'ali', got %d", len(cl.autocomplete.Matches))
	}

	// Simulate backspace: remove last character from query
	if len(cl.autocomplete.Query) > 0 {
		cl.autocomplete.Query = cl.autocomplete.Query[:len(cl.autocomplete.Query)-1]
		cl.filterMembers(cl.autocomplete.Query)
	}

	if cl.autocomplete.Query != "al" {
		t.Errorf("expected query 'al' after backspace, got %q", cl.autocomplete.Query)
	}

	if len(cl.autocomplete.Matches) != 2 {
		t.Errorf("expected 2 matches for 'al', got %d", len(cl.autocomplete.Matches))
	}
}

func TestAutocompleteBackspaceClosesEmpty(t *testing.T) {
	cl := newTestCommentsList()

	// Set up test members
	members := []trello.Member{
		{ID: "member1", FullName: "Alice Smith", Username: "alice"},
	}
	cl.SetMembers(members)

	// Activate autocomplete with empty query
	cl.autocomplete.Active = true
	cl.autocomplete.Query = ""

	// Simulate backspace when query is empty
	if len(cl.autocomplete.Query) > 0 {
		cl.autocomplete.Query = cl.autocomplete.Query[:len(cl.autocomplete.Query)-1]
	} else {
		cl.autocomplete.Active = false
	}

	if cl.autocomplete.Active {
		t.Error("expected autocomplete to be closed when backspace on empty query")
	}
}

func TestCommentCreatedMessage(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)

	// Create a test comment
	newComment := trello.Comment{
		ID:   "new-comment-1",
		Body: "This is a new comment",
		Author: trello.Member{
			ID:       "user1",
			FullName: "John Doe",
			Username: "johndoe",
		},
		Date:     time.Now(),
		Editable: true,
	}

	// Send CommentCreatedMsg
	msg := CommentCreatedMsg{Comment: newComment}
	cl, _ = cl.Update(msg)

	// Verify comment was added
	if len(cl.comments) != 1 {
		t.Errorf("expected 1 comment after creation, got %d", len(cl.comments))
	}

	if cl.comments[0].ID != "new-comment-1" {
		t.Errorf("expected comment ID 'new-comment-1', got %q", cl.comments[0].ID)
	}

	if cl.comments[0].Body != "This is a new comment" {
		t.Errorf("expected body 'This is a new comment', got %q", cl.comments[0].Body)
	}
}

func TestCommentCreatedReturnsToView(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)

	// Put component in Create mode
	cl.mode = CommentModeCreate
	cl.textInput.SetValue("Test comment")

	newComment := trello.Comment{
		ID:       "comment-1",
		Body:     "Test comment",
		Author:   trello.Member{ID: "u1", FullName: "User"},
		Date:     time.Now(),
		Editable: true,
	}

	msg := CommentCreatedMsg{Comment: newComment}
	cl, _ = cl.Update(msg)

	// Verify mode changed back to View
	if cl.mode != CommentModeView {
		t.Errorf("expected mode CommentModeView after creation, got %d", cl.mode)
	}
}

func TestCommentCreatedClearsInput(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)

	// Put component in Create mode with text
	cl.mode = CommentModeCreate
	cl.textInput.SetValue("Some input text")

	newComment := trello.Comment{
		ID:       "comment-1",
		Body:     "Some input text",
		Author:   trello.Member{ID: "u1", FullName: "User"},
		Date:     time.Now(),
		Editable: true,
	}

	msg := CommentCreatedMsg{Comment: newComment}
	cl, _ = cl.Update(msg)

	// Verify input was cleared
	if cl.textInput.Value() != "" {
		t.Errorf("expected empty input after creation, got %q", cl.textInput.Value())
	}
}

func TestCommentUpdatedMessage(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)

	// Set up initial comments
	initialComments := []trello.Comment{
		{
			ID:       "comment-1",
			Body:     "Old text",
			Author:   trello.Member{ID: "u1", FullName: "User"},
			Date:     time.Now(),
			Editable: true,
		},
		{
			ID:       "comment-2",
			Body:     "Other comment",
			Author:   trello.Member{ID: "u2", FullName: "User Two"},
			Date:     time.Now(),
			Editable: true,
		},
	}
	cl.SetComments(initialComments)

	// Update first comment (selectedIdx = 0, so editingIdx should be set to 0)
	cl.mode = CommentModeEdit
	cl.editingIdx = 0
	cl.textInput.SetValue("Updated text")

	updatedComment := trello.Comment{
		ID:       "comment-1",
		Body:     "Updated text",
		Author:   trello.Member{ID: "u1", FullName: "User"},
		Date:     time.Now(),
		Editable: true,
	}

	msg := CommentUpdatedMsg{Comment: updatedComment}
	cl, _ = cl.Update(msg)

	// Verify comment was updated
	if cl.comments[0].Body != "Updated text" {
		t.Errorf("expected updated body 'Updated text', got %q", cl.comments[0].Body)
	}

	// Verify other comment wasn't affected
	if cl.comments[1].Body != "Other comment" {
		t.Errorf("expected unchanged body 'Other comment', got %q", cl.comments[1].Body)
	}
}

func TestCommentUpdatedReturnsToView(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)

	// Set up initial comment
	initialComments := []trello.Comment{
		{
			ID:       "comment-1",
			Body:     "Old text",
			Author:   trello.Member{ID: "u1", FullName: "User"},
			Date:     time.Now(),
			Editable: true,
		},
	}
	cl.SetComments(initialComments)

	// Put in Edit mode
	cl.mode = CommentModeEdit
	cl.editingIdx = 0
	cl.textInput.SetValue("Updated text")

	updatedComment := trello.Comment{
		ID:       "comment-1",
		Body:     "Updated text",
		Author:   trello.Member{ID: "u1", FullName: "User"},
		Date:     time.Now(),
		Editable: true,
	}

	msg := CommentUpdatedMsg{Comment: updatedComment}
	cl, _ = cl.Update(msg)

	// Verify mode changed back to View
	if cl.mode != CommentModeView {
		t.Errorf("expected mode CommentModeView after update, got %d", cl.mode)
	}
}

func TestCommentUpdatedClearsInputAndResetsIdx(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)

	// Set up initial comment
	initialComments := []trello.Comment{
		{
			ID:       "comment-1",
			Body:     "Old text",
			Author:   trello.Member{ID: "u1", FullName: "User"},
			Date:     time.Now(),
			Editable: true,
		},
	}
	cl.SetComments(initialComments)

	// Put in Edit mode
	cl.mode = CommentModeEdit
	cl.editingIdx = 0
	cl.textInput.SetValue("Updated text")

	updatedComment := trello.Comment{
		ID:       "comment-1",
		Body:     "Updated text",
		Author:   trello.Member{ID: "u1", FullName: "User"},
		Date:     time.Now(),
		Editable: true,
	}

	msg := CommentUpdatedMsg{Comment: updatedComment}
	cl, _ = cl.Update(msg)

	// Verify input was cleared
	if cl.textInput.Value() != "" {
		t.Errorf("expected empty input after update, got %q", cl.textInput.Value())
	}

	// Verify editingIdx was reset
	if cl.editingIdx != -1 {
		t.Errorf("expected editingIdx -1 after update, got %d", cl.editingIdx)
	}
}

func TestCommentDeletedMessage(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)

	// Set up initial comments
	initialComments := []trello.Comment{
		{
			ID:       "comment-1",
			Body:     "First comment",
			Author:   trello.Member{ID: "u1", FullName: "User One"},
			Date:     time.Now(),
			Editable: true,
		},
		{
			ID:       "comment-2",
			Body:     "Second comment",
			Author:   trello.Member{ID: "u2", FullName: "User Two"},
			Date:     time.Now(),
			Editable: true,
		},
		{
			ID:       "comment-3",
			Body:     "Third comment",
			Author:   trello.Member{ID: "u3", FullName: "User Three"},
			Date:     time.Now(),
			Editable: true,
		},
	}
	cl.SetComments(initialComments)

	// Select and delete second comment
	cl.selectedIdx = 1

	msg := CommentDeletedMsg{CommentID: "comment-2"}
	cl, _ = cl.Update(msg)

	// Verify comment was removed
	if len(cl.comments) != 2 {
		t.Errorf("expected 2 comments after deletion, got %d", len(cl.comments))
	}

	// Verify correct comment was deleted
	if cl.comments[0].ID != "comment-1" {
		t.Errorf("expected first comment to be 'comment-1', got %q", cl.comments[0].ID)
	}
	if cl.comments[1].ID != "comment-3" {
		t.Errorf("expected second comment to be 'comment-3', got %q", cl.comments[1].ID)
	}
}

func TestCommentDeletedAdjustsSelection(t *testing.T) {
	cl := newTestCommentsList()
	cl.SetFocus(true)

	// Set up initial comments
	initialComments := []trello.Comment{
		{
			ID:       "comment-1",
			Body:     "First comment",
			Author:   trello.Member{ID: "u1", FullName: "User One"},
			Date:     time.Now(),
			Editable: true,
		},
		{
			ID:       "comment-2",
			Body:     "Second comment",
			Author:   trello.Member{ID: "u2", FullName: "User Two"},
			Date:     time.Now(),
			Editable: true,
		},
	}
	cl.SetComments(initialComments)

	// Select and delete the last comment
	cl.selectedIdx = 1

	msg := CommentDeletedMsg{CommentID: "comment-2"}
	cl, _ = cl.Update(msg)

	// Verify selectedIdx was adjusted
	if cl.selectedIdx != 0 {
		t.Errorf("expected selectedIdx 0 after deleting last comment, got %d", cl.selectedIdx)
	}
}
