package hsm

import (
	"testing"
	"time"
)

func TestNewStateStack(t *testing.T) {
	stack := NewStateStack()

	if stack == nil {
		t.Fatal("NewStateStack should return non-nil stack")
	}
	if stack.Depth() != 0 {
		t.Error("New stack should have depth 0")
	}
	if !stack.IsEmpty() {
		t.Error("New stack should be empty")
	}
	if stack.maxDepth != 10 {
		t.Error("Default max depth should be 10")
	}
}

func TestNewStateStackWithDepth(t *testing.T) {
	stack := NewStateStackWithDepth(5)

	if stack.maxDepth != 5 {
		t.Errorf("Max depth should be 5, got %d", stack.maxDepth)
	}
}

func TestStateStackPush(t *testing.T) {
	stack := NewStateStack()
	state := &mockState{id: StateTurnUpkeep}
	ctx := NewStateContext()
	ctx.StartTime = time.Now()

	// Push first state
	err := stack.Push(state, ctx)
	if err != nil {
		t.Errorf("Push failed: %v", err)
	}
	if stack.Depth() != 1 {
		t.Errorf("Depth should be 1, got %d", stack.Depth())
	}

	// Push second state
	state2 := &mockState{id: StateTurnMoving}
	err = stack.Push(state2, ctx)
	if err != nil {
		t.Errorf("Second push failed: %v", err)
	}
	if stack.Depth() != 2 {
		t.Errorf("Depth should be 2, got %d", stack.Depth())
	}
}

func TestStateStackPushMaxDepth(t *testing.T) {
	stack := NewStateStackWithDepth(3)
	ctx := NewStateContext()
	ctx.StartTime = time.Now()

	// Push 3 states (should succeed)
	for i := 0; i < 3; i++ {
		state := &mockState{id: StateID(200 + i)}
		err := stack.Push(state, ctx)
		if err != nil {
			t.Errorf("Push %d failed: %v", i, err)
		}
	}

	// Push 4th state (should fail)
	state := &mockState{id: StateID(203)}
	err := stack.Push(state, ctx)
	if err == nil {
		t.Error("Push beyond max depth should fail")
	}
}

func TestStateStackPop(t *testing.T) {
	stack := NewStateStack()
	ctx := NewStateContext()
	ctx.StartTime = time.Now()

	// Push states
	state1 := &mockState{id: StateTurnUpkeep}
	state2 := &mockState{id: StateTurnMoving}
	stack.Push(state1, ctx)
	stack.Push(state2, ctx)

	// Pop first (should get state2 - top)
	entry, err := stack.Pop()
	if err != nil {
		t.Errorf("Pop failed: %v", err)
	}
	if entry.StateID != StateTurnMoving {
		t.Errorf("Pop should return top state (Moving), got %s", entry.StateID.String())
	}
	if stack.Depth() != 1 {
		t.Errorf("Depth should be 1 after pop, got %d", stack.Depth())
	}

	// Pop second (should get state1)
	entry, err = stack.Pop()
	if err != nil {
		t.Errorf("Second pop failed: %v", err)
	}
	if entry.StateID != StateTurnUpkeep {
		t.Errorf("Pop should return state1 (Upkeep), got %s", entry.StateID.String())
	}
	if !stack.IsEmpty() {
		t.Error("Stack should be empty after all pops")
	}

	// Pop empty stack (should fail)
	entry, err = stack.Pop()
	if err == nil {
		t.Error("Pop empty stack should fail")
	}
	if entry != nil {
		t.Error("Pop empty stack should return nil entry")
	}
}

func TestStateStackPeek(t *testing.T) {
	stack := NewStateStack()
	ctx := NewStateContext()
	ctx.StartTime = time.Now()

	// Peek empty stack
	entry := stack.Peek()
	if entry != nil {
		t.Error("Peek empty stack should return nil")
	}

	// Push and peek
	state := &mockState{id: StateTurnUpkeep}
	stack.Push(state, ctx)
	entry = stack.Peek()
	if entry == nil {
		t.Fatal("Peek should return entry")
	}
	if entry.StateID != StateTurnUpkeep {
		t.Errorf("Peek should return top state, got %s", entry.StateID.String())
	}
	if stack.Depth() != 1 {
		t.Error("Peek should not modify stack depth")
	}
}

func TestStateStackContains(t *testing.T) {
	stack := NewStateStack()
	ctx := NewStateContext()
	ctx.StartTime = time.Now()

	// Empty stack
	if stack.Contains(StateTurnUpkeep) {
		t.Error("Empty stack should not contain any state")
	}

	// Push states
	state1 := &mockState{id: StateTurnUpkeep}
	state2 := &mockState{id: StateTurnMoving}
	stack.Push(state1, ctx)
	stack.Push(state2, ctx)

	// Check contains
	if !stack.Contains(StateTurnUpkeep) {
		t.Error("Stack should contain TurnUpkeep")
	}
	if !stack.Contains(StateTurnMoving) {
		t.Error("Stack should contain TurnMoving")
	}
	if stack.Contains(StateTurnEnd) {
		t.Error("Stack should not contain TurnEnd")
	}
}

func TestStateStackGetStackIDs(t *testing.T) {
	stack := NewStateStack()
	ctx := NewStateContext()
	ctx.StartTime = time.Now()

	// Empty stack
	ids := stack.GetStackIDs()
	if len(ids) != 0 {
		t.Error("Empty stack IDs should be empty")
	}

	// Push states
	state1 := &mockState{id: StateTurnUpkeep}
	state2 := &mockState{id: StateTurnMoving}
	stack.Push(state1, ctx)
	stack.Push(state2, ctx)

	ids = stack.GetStackIDs()
	if len(ids) != 2 {
		t.Errorf("IDs length should be 2, got %d", len(ids))
	}
	if ids[0] != StateTurnUpkeep {
		t.Errorf("First ID should be TurnUpkeep, got %s", ids[0].String())
	}
	if ids[1] != StateTurnMoving {
		t.Errorf("Second ID should be TurnMoving, got %s", ids[1].String())
	}
}

func TestStateStackClear(t *testing.T) {
	stack := NewStateStack()
	ctx := NewStateContext()
	ctx.StartTime = time.Now()

	// Push states
	for i := 0; i < 3; i++ {
		state := &mockState{id: StateID(200 + i)}
		stack.Push(state, ctx)
	}

	// Clear
	stack.Clear()
	if stack.Depth() != 0 {
		t.Error("Stack should be empty after clear")
	}
	if !stack.IsEmpty() {
		t.Error("IsEmpty should return true after clear")
	}
}

func TestStateStackString(t *testing.T) {
	stack := NewStateStack()
	ctx := NewStateContext()
	ctx.StartTime = time.Now()

	// Empty stack
	if stack.String() != "Stack[empty]" {
		t.Errorf("Empty stack string = %s, want 'Stack[empty]'", stack.String())
	}

	// Push states
	state1 := &mockState{id: StateTurnUpkeep}
	state2 := &mockState{id: StateTurnMoving}
	stack.Push(state1, ctx)
	stack.Push(state2, ctx)

	expected := "Stack[TurnUpkeep -> TurnMoving]"
	if stack.String() != expected {
		t.Errorf("Stack string = %s, want '%s'", stack.String(), expected)
	}
}

func TestStateStackGetEntry(t *testing.T) {
	stack := NewStateStack()
	ctx := NewStateContext()
	ctx.StartTime = time.Now()

	// Push states
	state1 := &mockState{id: StateTurnUpkeep}
	state2 := &mockState{id: StateTurnMoving}
	stack.Push(state1, ctx)
	stack.Push(state2, ctx)

	// Get valid entries
	entry0 := stack.GetEntry(0)
	if entry0 == nil || entry0.StateID != StateTurnUpkeep {
		t.Error("Entry 0 should be TurnUpkeep")
	}

	entry1 := stack.GetEntry(1)
	if entry1 == nil || entry1.StateID != StateTurnMoving {
		t.Error("Entry 1 should be TurnMoving")
	}

	// Get invalid entries
	entryNeg := stack.GetEntry(-1)
	if entryNeg != nil {
		t.Error("Negative index should return nil")
	}

	entryLarge := stack.GetEntry(10)
	if entryLarge != nil {
		t.Error("Large index should return nil")
	}
}