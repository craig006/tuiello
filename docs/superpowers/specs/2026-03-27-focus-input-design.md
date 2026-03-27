# Focus and Input Routing Design

**Date:** 2026-03-27
**Status:** Design approved
**Scope:** TUI keyboard input handling and focus management for multi-section app (board, details panel, search, modals)

---

## Problem Statement

The Trello TUI presents multiple interactive sections:
- **Board** (columns and cards)
- **Details panel** (tabs: description, comments, etc.)
- **Search/filter bar**
- **Modals** (command palette, member/label filters)

Each section has its own focusable elements (e.g., cards within a column, comments within a panel). The app needs to:

1. **Track focus** at two levels: which section is active, and which element within it is selected
2. **Route keyboard events** correctly based on focus, with special handling for text input
3. **Ensure consistency**: only one section and one element have focus at any time (with exceptions for modals)

The current `boardHasFocus` boolean is insufficient for this complexity.

---

## Solution Overview

### Two-Level Focus Model

**Section Focus:** The active container in the app. Exactly one section has focus at all times (when no modal is open).
- Examples: "board", "detail", "search"

**Element Focus:** A focusable item within the focused section. Between 0 and 1 element has focus across the entire app.
- Examples: a card in a column, a comment in the details panel, a member in a filter modal

### Focus Rules

1. **Exactly one section is always focused** (when no modal is open)
2. If a section loses focus, all its elements lose focus
3. **When a section gains focus**, automatically select the first focusable element (if any exist)
4. **When content changes** in a section (elements added/removed/reordered):
   - If the currently focused element still exists, keep it focused
   - If it was deleted, focus the nearest element by index (previous sibling preferred, then next)
   - If the section now has no focusable elements, clear element focus
5. **When a modal opens**, suspend the normal focus model (save current state)
6. **When a modal closes**, restore the saved focus state

### Modal Behavior

Modals (command palette, filters) suspend the normal focus system. While a modal is open:
- Regular section/element focus is inactive
- The modal handles its own input
- When the modal closes, the previous section/element focus is restored

### Input Routing: Event Bubbling

Keyboard events flow through the focus hierarchy with a stop-on-handled pattern:

```
1. Key event received
2. If modal is active:
   → Send to modal's HandleKeyEvent()
   → If handled, done. Otherwise, continue (bubble to app)
3. If modal is NOT active:
   → Send to FocusedElement's HandleKeyEvent()
   → If handled, done. Otherwise, continue
4. Send to FocusedSection's HandleKeyEvent()
   → If handled, done. Otherwise, continue
5. Send to App's HandleKeyEvent() (global shortcuts)
   → If handled, done. Otherwise, ignore
```

**Why modals bubble to app?** Global shortcuts like `Ctrl+Q` (quit) should always work, even in modals.

### Text Input Special Handling

When a text input element (e.g., editing a comment) has focus:
- **All regular keyboard input** goes to the text input
- **No shortcuts fire** (typing 'd' inserts text, doesn't trigger delete)
- **Special keys** (Escape, Enter) can be handled specially by the input or bubble up to the section/app

Example flow when editing a comment:
1. User types "Hello" → text input receives all characters
2. User presses Escape → text input can handle it (exit edit mode) or return unhandled (bubble)
3. User presses `d` while editing → goes to text input (inserts 'd'), does NOT trigger delete action

### Autocomplete & Inline UI

Autocomplete popups (e.g., `@username` suggestions while typing a comment) do NOT steal focus:
- They're part of the text input element's internal behavior
- Keyboard routing continues to send input to the text input
- The popup responds to input but doesn't change the focus model

---

## Data Structures

### FocusManager

```go
type FocusManager struct {
  focusedSection string  // "board", "detail", "search", etc.
  focusedElement string  // ID of element within section, or "" if none
  modalActive    bool    // true if modal is open
  suspendedState FocusState  // saved focus state before modal opened
}

type FocusState struct {
  FocusedSection string
  FocusedElement string
}
```

### Interfaces

All sections and elements must implement `KeyHandler`:

```go
type KeyHandler interface {
  HandleKeyEvent(key string) bool  // true = handled, false = bubble up
}
```

Sections optionally implement `FocusAware` to help manage content changes:

```go
type FocusAware interface {
  GetFocusableElements() []ElementID  // return current focusable elements
  OnContentChanged()  // called when focusable elements change
}
```

---

## Core Operations

### SetFocusedSection(sectionID string)

- Set `focusedSection` to `sectionID`
- Clear `focusedElement` (previous section's element loses focus)
- Auto-select first focusable element in new section (if any exist)
- Ignored if a modal is active

### SetFocusedElement(elementID string)

- Only works if `elementID` is in the `focusedSection`
- Set `focusedElement` to `elementID`
- Ignored otherwise (can't focus element in unfocused section)

### OpenModal(modalID string)

- Save current `FocusState` (both `focusedSection` and `focusedElement`) to `suspendedState`
- Set `modalActive = true`
- Modal manages its own element focus internally

### CloseModal()

- Set `modalActive = false`
- Restore `FocusState` from `suspendedState`

### NotifyContentChanged(sectionID string)

Called by a section when its focusable elements change:
- If `sectionID` is not the focused section, do nothing (no action needed)
- If `focusedElement` still exists in the section, keep it focused
- If `focusedElement` was deleted:
  - Get the index of where it was
  - Select the element at index-1 (previous sibling)
  - If no previous sibling, select element at same index (next sibling)
  - If no elements remain, clear `focusedElement`
- If section now has no focusable elements, clear `focusedElement`

---

## Edge Cases

### Focus Request While Modal is Open

If code calls `SetFocusedSection()` while `modalActive = true`:
- **Decision:** Ignore it. Modals are modal—don't process focus changes until modal closes.
- Rationale: Prevents unexpected focus changes beneath a modal.

### Section with No Focusable Elements

Valid state. `focusedElement = ""` is correct.
- The section can still handle keyboard events (e.g., "refresh" action)
- Navigation events that would change focus return unhandled and bubble up

### Text Input Lifecycle

When entering edit mode:
- Create text input element
- Set it as `focusedElement`
- Text input's `HandleKeyEvent()` consumes all input

When exiting edit mode (Escape pressed or Enter submitted):
- Destroy text input element
- Section detects this via `OnContentChanged()` or explicit call
- FocusManager re-applies content change rules
- Focus lands on the (now non-editable) comment element that was being edited

### Switching Between Text Input and Regular Element

```
User selects comment 5
  → focusedElement = "comment_5"

User presses 'e' to edit
  → focusedElement = "textinput_editing_comment_5"

User presses Escape
  → focusedElement = "comment_5" (restored)
```

The section manages this transition by replacing elements and calling `NotifyContentChanged()`.

### Rapid Content Changes

If a section's content changes rapidly (e.g., comments loading in streaming fashion):
- Each change triggers `NotifyContentChanged()`
- FocusManager validates focus and adjusts if needed
- Excessive changes can cause focus to bounce—sections should batch updates when possible

### Initial State on App Start

- `focusedSection = "board"`
- Board's first column auto-selected (if board has columns)
- First card in that column auto-selected
- `modalActive = false`

---

## Implementation Strategy

### Phase 1: Focus State Management

1. Replace `boardHasFocus` with `FocusManager` in `App`
2. Implement core operations: `SetFocusedSection`, `SetFocusedElement`, `OpenModal`, `CloseModal`
3. Add `NotifyContentChanged()` handling
4. Wire up focus transitions when user switches sections (clicks on detail panel, presses Tab, etc.)

### Phase 2: Event Routing

1. Define `KeyHandler` interface
2. Update `App.Update()` to route key events through the bubbling chain
3. Implement `App.HandleKeyEvent()` for global shortcuts
4. Implement `HandleKeyEvent()` in each section and element

### Phase 3: Text Input Integration

1. When editing starts, create text input element and set as focused
2. Text input's `HandleKeyEvent()` returns `true` for all keys (consume all input)
3. Special keys (Escape, Enter) can be handled by text input or bubbled to section
4. When editing ends, notify focus manager and restore focus to previous element

### Phase 4: Modal Integration

1. Wrap modal in focus suspension logic
2. `OpenModal()` saves state before modal takes over
3. `CloseModal()` restores state
4. Modal's `HandleKeyEvent()` bubbles unhandled events to app (for global shortcuts)

---

## Key Decisions

1. **Event Bubbling over Direct Dispatch:** Chosen for simplicity and flexibility. Each component decides if it can handle an event.

2. **Text Input Consumes All Input:** When a text input has focus, no shortcuts fire. This prevents accidental actions while typing.

3. **Modals Suspend, Not Replace:** Modals don't become a "section" in the focus model. They suspend the normal model and restore it on close. This simplifies modal nesting if needed in the future.

4. **Nearest-Element Recovery:** When a focused element is deleted, select the nearest sibling (not the first). Better UX.

5. **Auto-Focus on Section Switch:** When focusing a section, auto-select the first element. Prevents "focus but nothing is selected" state.

---

## Testing Considerations

- **Focus transitions:** Board → Detail → Search → Board (cycle works, no focus lost)
- **Modal lifecycle:** Open modal → close modal → previous focus restored
- **Content changes:** Delete focused comment → focus moves to nearest comment
- **Text input:** Type text → Escape exits → focus restored to comment
- **Keyboard routing:** Test that shortcuts work at each level and bubble correctly
- **Edge case:** Delete all comments while one is focused → no element focus, but section focus persists
- **Global shortcuts:** `Ctrl+Q` works even inside modal

---

## Future Considerations

- **Focus history/undo:** Could add a focus stack to support more complex navigation patterns
- **Keyboard focus visualization:** Update UI to clearly show which section/element has focus
- **Accessibility:** Ensure focus model works with screen readers
- **Configurable auto-focus:** Allow sections to customize which element is auto-selected (not always first)
