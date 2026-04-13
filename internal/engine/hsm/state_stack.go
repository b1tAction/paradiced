package hsm

import (
	"errors"
)

// StateStack implements a push-down automaton for interrupt handling.
// When a decision is needed, current state is pushed onto the stack,
// and WaitDecision state becomes active. After decision is handled,
// the stack is popped to restore the previous state.
type StateStack struct {
	// Stack entries (bottom = oldest, top = newest)
	stack []StackEntry

	// Maximum stack depth (prevents infinite nesting)
	maxDepth int
}

// StackEntry represents a saved state on the stack.
type StackEntry struct {
	StateID    StateID
	State      State
	Context    *StateContext
	EntryTime  int64 // Timestamp when pushed (for timeout calculation)
}

// NewStateStack creates a new state stack with default max depth.
func NewStateStack() *StateStack {
	return &StateStack{
		stack:    make([]StackEntry, 0),
		maxDepth: 10, // Allow up to 10 nested interrupts
	}
}

// NewStateStackWithDepth creates a new state stack with custom max depth.
func NewStateStackWithDepth(maxDepth int) *StateStack {
	return &StateStack{
		stack:    make([]StackEntry, 0),
		maxDepth: maxDepth,
	}
}

// Push saves current state onto the stack before entering interrupt state.
// Returns error if stack is at max depth.
func (ss *StateStack) Push(state State, ctx *StateContext) error {
	if ss == nil {
		return errors.New("state stack is nil")
	}
	if state == nil {
		return errors.New("state is nil")
	}
	if len(ss.stack) >= ss.maxDepth {
		return errors.New("stack depth limit reached")
	}

	entry := StackEntry{
		StateID:   state.ID(),
		State:     state,
		Context:   ctx,
		EntryTime: ctx.StartTime.UnixNano(),
	}

	ss.stack = append(ss.stack, entry)
	return nil
}

// Pop restores the most recently saved state from the stack.
// Returns the popped entry, or error if stack is empty.
func (ss *StateStack) Pop() (*StackEntry, error) {
	if ss == nil {
		return nil, errors.New("state stack is nil")
	}
	if len(ss.stack) == 0 {
		return nil, errors.New("stack is empty")
	}

	// Get top element
	index := len(ss.stack) - 1
	entry := ss.stack[index]

	// Remove from stack
	ss.stack = ss.stack[:index]

	return &entry, nil
}

// Peek returns the top entry without removing it.
// Returns nil if stack is empty.
func (ss *StateStack) Peek() *StackEntry {
	if ss == nil || len(ss.stack) == 0 {
		return nil
	}

	index := len(ss.stack) - 1
	entry := ss.stack[index]
	return &entry
}

// Depth returns current stack depth.
func (ss *StateStack) Depth() int {
	if ss == nil {
		return 0
	}
	return len(ss.stack)
}

// IsEmpty checks if stack has no entries.
func (ss *StateStack) IsEmpty() bool {
	return ss == nil || len(ss.stack) == 0
}

// Clear removes all entries from the stack.
func (ss *StateStack) Clear() {
	if ss != nil {
		ss.stack = make([]StackEntry, 0)
	}
}

// GetEntry retrieves an entry at specific index (0 = bottom).
func (ss *StateStack) GetEntry(index int) *StackEntry {
	if ss == nil || index < 0 || index >= len(ss.stack) {
		return nil
	}
	return &ss.stack[index]
}

// GetStackIDs returns all state IDs in the stack (for snapshot).
func (ss *StateStack) GetStackIDs() []StateID {
	if ss == nil {
		return nil
	}
	ids := make([]StateID, len(ss.stack))
	for i, entry := range ss.stack {
		ids[i] = entry.StateID
	}
	return ids
}

// RestoreFromIDs restores stack state from a list of state IDs.
// Note: This requires state factory to recreate state instances.
func (ss *StateStack) RestoreFromIDs(ids []StateID, factory StateFactory) error {
	if ss == nil {
		return errors.New("state stack is nil")
	}

	ss.Clear()
	for _, id := range ids {
		state := factory.CreateState(id)
		if state == nil {
			return errors.New("failed to create state: " + id.String())
		}
		// Push with minimal context (will be populated on restore)
		ctx := NewStateContext()
		if err := ss.Push(state, ctx); err != nil {
			return err
		}
	}
	return nil
}

// StateFactory is an interface for creating state instances from StateID.
type StateFactory interface {
	CreateState(id StateID) State
}

// Contains checks if a state ID exists in the stack.
func (ss *StateStack) Contains(stateID StateID) bool {
	if ss == nil {
		return false
	}
	for _, entry := range ss.stack {
		if entry.StateID == stateID {
			return true
		}
	}
	return false
}

// String returns a human-readable representation of the stack.
func (ss *StateStack) String() string {
	if ss == nil || len(ss.stack) == 0 {
		return "Stack[empty]"
	}

	result := "Stack["
	for i, entry := range ss.stack {
		if i > 0 {
			result += " -> "
		}
		result += entry.StateID.String()
	}
	result += "]"
	return result
}