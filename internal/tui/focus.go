package tui

// FocusState represents the state of focus at a point in time
type FocusState struct {
	FocusedSection string
	FocusedElement string
}

// FocusManager tracks which section and element has focus, and manages modal suspension
type FocusManager struct {
	focusedSection string
	focusedElement string
	modalActive    bool
	suspendedState FocusState
}

// NewFocusManager creates a new FocusManager with the given initial section
func NewFocusManager(initialSection string) *FocusManager {
	return &FocusManager{
		focusedSection: initialSection,
		focusedElement: "",
		modalActive:    false,
	}
}

// FocusedSection returns the currently focused section
func (fm *FocusManager) FocusedSection() string {
	return fm.focusedSection
}

// FocusedElement returns the currently focused element within the focused section
func (fm *FocusManager) FocusedElement() string {
	return fm.focusedElement
}

// IsModalActive returns whether a modal is currently active
func (fm *FocusManager) IsModalActive() bool {
	return fm.modalActive
}

// SetFocusedSection sets the focused section, clearing element focus.
// If a modal is active, this operation is ignored.
func (fm *FocusManager) SetFocusedSection(section string) {
	if fm.modalActive {
		return
	}

	fm.focusedSection = section
	fm.focusedElement = ""
}

// SetFocusedElement sets the focused element within the currently focused section.
// Returns true if successful, false if the element cannot be set (e.g., during modal).
func (fm *FocusManager) SetFocusedElement(element string) bool {
	if fm.modalActive {
		return false
	}

	fm.focusedElement = element
	return true
}

// OpenModal saves the current focus state and marks a modal as active
func (fm *FocusManager) OpenModal() {
	fm.suspendedState = FocusState{
		FocusedSection: fm.focusedSection,
		FocusedElement: fm.focusedElement,
	}
	fm.modalActive = true
}

// CloseModal restores the suspended focus state and marks the modal as inactive
func (fm *FocusManager) CloseModal() {
	fm.focusedSection = fm.suspendedState.FocusedSection
	fm.focusedElement = fm.suspendedState.FocusedElement
	fm.modalActive = false
}

// NotifyContentChanged is called when a section's content changes.
// If the section is currently focused, the focused element is cleared
// to allow the section to re-establish focus on valid content.
func (fm *FocusManager) NotifyContentChanged(section string) {
	if section != fm.focusedSection {
		return
	}

	fm.focusedElement = ""
}

// KeyHandler is implemented by sections and elements that handle keyboard input
type KeyHandler interface {
	// HandleKeyEvent processes a keyboard event
	// Returns true if handled, false to bubble up
	HandleKeyEvent(key string) bool
}

// FocusAware is optionally implemented by sections to manage focus state
type FocusAware interface {
	// GetFocusableElements returns the IDs of currently focusable elements
	GetFocusableElements() []string

	// OnContentChanged is called when focusable elements change
	OnContentChanged()
}
