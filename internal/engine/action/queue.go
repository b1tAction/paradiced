package action

// Queue manages derived actions generated during action execution.
// When an Action executes, it may generate additional Actions (e.g., landing on a trap cell generates DamageAction).
// These derived actions are queued and processed after the main action completes.
type Queue struct {
	items []Action
}

// NewQueue creates a new empty action queue.
func NewQueue() *Queue {
	return &Queue{
		items: make([]Action, 0),
	}
}

// Push adds a derived action to the queue.
func (q *Queue) Push(action Action) {
	q.items = append(q.items, action)
}

// Pop removes and returns the next action from the queue.
// Returns nil if queue is empty.
func (q *Queue) Pop() Action {
	if len(q.items) == 0 {
		return nil
	}
	action := q.items[0]
	q.items = q.items[1:]
	return action
}

// Peek returns the next action without removing it.
// Returns nil if queue is empty.
func (q *Queue) Peek() Action {
	if len(q.items) == 0 {
		return nil
	}
	return q.items[0]
}

// Len returns the number of actions in the queue.
func (q *Queue) Len() int {
	return len(q.items)
}

// Clear removes all actions from the queue.
func (q *Queue) Clear() {
	q.items = make([]Action, 0)
}

// IsEmpty returns true if the queue has no actions.
func (q *Queue) IsEmpty() bool {
	return len(q.items) == 0
}