package event

import (
	"time"
)

// Option represents a decision option.
type Option struct {
	ID     string         `json:"id"`    // Option ID
	Label  string         `json:"label"` // Option display text
	Action func(*Context) `json:"-"`     // Action to execute after selection
}

// Decision represents a user decision request.
// Used for effects that require user confirmation (e.g., whether to use a shield).
type Decision struct {
	ID          string              `json:"id"`           // Decision ID
	Prompt      string              `json:"prompt"`       // Prompt text
	Options     []Option            `json:"options"`      // Available options list
	Priority    int                 `json:"priority"`     // Execution priority (higher executes first)
	Timeout     time.Duration       `json:"timeout"`      // Timeout duration (optional)
	Default     int                 `json:"default"`      // Default option index on timeout
	NeedConfirm bool                `json:"need_confirm"` // Whether user confirmation is needed (default true)
	Condition   func() bool         `json:"-"`            // Dynamic condition check (optional)
	OnChoice    func(int, *Context) `json:"-"`            // Callback after user selection
	SourceID    string              `json:"source_id"`    // Source ID (Buff/Item)
	SourceType  string              `json:"source_type"`  // Source type "buff" / "item"
}

// NewDecision creates a new decision (needs confirmation by default).
func NewDecision(prompt string, options []Option) *Decision {
	return &Decision{
		ID:          newID(),
		Prompt:      prompt,
		Options:     options,
		Priority:    0,
		Default:     0,
		NeedConfirm: true,
	}
}

// NewAutoDecision creates an auto-executing decision (no confirmation needed).
func NewAutoDecision(prompt string, options []Option) *Decision {
	return &Decision{
		ID:          newID(),
		Prompt:      prompt,
		Options:     options,
		Priority:    0,
		Default:     0,
		NeedConfirm: false,
	}
}

// WithPriority sets the priority.
func (d *Decision) WithPriority(priority int) *Decision {
	d.Priority = priority
	return d
}

// WithTimeout sets the timeout.
func (d *Decision) WithTimeout(timeout time.Duration, defaultChoice int) *Decision {
	d.Timeout = timeout
	d.Default = defaultChoice
	return d
}

// WithCondition sets the condition.
func (d *Decision) WithCondition(condition func() bool) *Decision {
	d.Condition = condition
	return d
}

// WithOnChoice sets the choice callback.
func (d *Decision) WithOnChoice(onChoice func(int, *Context)) *Decision {
	d.OnChoice = onChoice
	return d
}

// WithSource sets the source information.
func (d *Decision) WithSource(sourceID, sourceType string) *Decision {
	d.SourceID = sourceID
	d.SourceType = sourceType
	return d
}

// ShouldAsk determines if user needs to be asked.
func (d *Decision) ShouldAsk() bool {
	// When NeedConfirm=false, execute directly without asking
	if !d.NeedConfirm {
		return false
	}
	// When NeedConfirm=true, check Condition
	if d.Condition == nil {
		return true // No condition means always ask
	}
	return d.Condition()
}

// WithNeedConfirm sets whether confirmation is needed.
func (d *Decision) WithNeedConfirm(need bool) *Decision {
	d.NeedConfirm = need
	return d
}

// Execute executes the user's selected option.
func (d *Decision) Execute(choice int, ctx *Context) {
	if choice < 0 || choice >= len(d.Options) {
		choice = d.Default // Use default option
	}

	// Execute option action
	if d.Options[choice].Action != nil {
		d.Options[choice].Action(ctx)
	}

	// Execute callback
	if d.OnChoice != nil {
		d.OnChoice(choice, ctx)
	}
}

// Clone clones the decision (used for templates).
func (d *Decision) Clone() *Decision {
	return &Decision{
		ID:         newID(),
		Prompt:     d.Prompt,
		Options:    d.Options, // Shared Options reference, Action functions are immutable
		Priority:   d.Priority,
		Timeout:    d.Timeout,
		Default:    d.Default,
		Condition:  d.Condition,
		OnChoice:   d.OnChoice,
		SourceID:   "", // Cleared when cloning, set by caller
		SourceType: d.SourceType,
	}
}

// DecisionBuilder is a decision builder (simplifies creation flow).
type DecisionBuilder struct {
	decision *Decision
}

// NewDecisionBuilder creates a decision builder.
func NewDecisionBuilder(prompt string) *DecisionBuilder {
	return &DecisionBuilder{
		decision: NewDecision(prompt, []Option{}),
	}
}

// AddOption adds an option.
func (b *DecisionBuilder) AddOption(id, label string, action func(*Context)) *DecisionBuilder {
	b.decision.Options = append(b.decision.Options, Option{
		ID:     id,
		Label:  label,
		Action: action,
	})
	return b
}

// SetPriority sets the priority.
func (b *DecisionBuilder) SetPriority(priority int) *DecisionBuilder {
	b.decision.Priority = priority
	return b
}

// SetTimeout sets the timeout.
func (b *DecisionBuilder) SetTimeout(timeout time.Duration, defaultChoice int) *DecisionBuilder {
	b.decision.Timeout = timeout
	b.decision.Default = defaultChoice
	return b
}

// SetCondition sets the condition.
func (b *DecisionBuilder) SetCondition(condition func() bool) *DecisionBuilder {
	b.decision.Condition = condition
	return b
}

// SetOnChoice sets the callback.
func (b *DecisionBuilder) SetOnChoice(onChoice func(int, *Context)) *DecisionBuilder {
	b.decision.OnChoice = onChoice
	return b
}

// SetSource sets the source.
func (b *DecisionBuilder) SetSource(sourceID, sourceType string) *DecisionBuilder {
	b.decision.SourceID = sourceID
	b.decision.SourceType = sourceType
	return b
}

// Build builds the decision.
func (b *DecisionBuilder) Build() *Decision {
	return b.decision
}