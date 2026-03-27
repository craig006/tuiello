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

// SetFocusedElement sets the focused element within a section.
// Returns true if successful (section matches the currently focused section),
// false if the section doesn't match.
func (fm *FocusManager) SetFocusedElement(section, element string) bool {
	if section != fm.focusedSection {
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

// NotifyContentChanged validates that the focused element still exists in the given section.
// If the section doesn't match the focused section or the element is not in the list,
// the element focus is cleared.
func (fm *FocusManager) NotifyContentChanged(section string, currentElements []string) {
	if section != fm.focusedSection {
		return
	}

	// Check if focused element exists in the current elements list
	found := false
	for _, elem := range currentElements {
		if elem == fm.focusedElement {
			found = true
			break
		}
	}

	if !found {
		fm.focusedElement = ""
	}
}
