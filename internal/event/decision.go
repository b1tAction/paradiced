package event

import (
	"time"

	"github.com/b1tAction/paradiced/pkg/id"
)

// Option represents a decision option.
type Option struct {
	ID     string         `json:"id"`
	Label  string         `json:"label"`
	Action func(*Context) `json:"-"`
}

// Decision represents a user decision request.
type Decision struct {
	ID          id.DecisionID       `json:"id"`
	Prompt      string              `json:"prompt"`
	Options     []Option            `json:"options"`
	Priority    int                 `json:"priority"`
	Timeout     time.Duration       `json:"timeout"`
	Default     int                 `json:"default"`
	NeedConfirm bool                `json:"need_confirm"`
	Condition   func() bool         `json:"-"`
	OnChoice    func(int, *Context) `json:"-"`
	SourceID    string              `json:"source_id"`
	SourceType  string              `json:"source_type"`
}

// NewDecision creates a new decision (needs confirmation by default).
func NewDecision(prompt string, options []Option) *Decision {
	return &Decision{
		ID:          id.NewDecisionID(),
		Prompt:      prompt,
		Options:     options,
		Priority:    0,
		Default:     0,
		NeedConfirm: true,
	}
}

// NewAutoDecision creates an auto-executing decision.
func NewAutoDecision(prompt string, options []Option) *Decision {
	return &Decision{
		ID:          id.NewDecisionID(),
		Prompt:      prompt,
		Options:     options,
		Priority:    0,
		Default:     0,
		NeedConfirm: false,
	}
}

func (d *Decision) WithPriority(priority int) *Decision {
	d.Priority = priority
	return d
}

func (d *Decision) WithTimeout(timeout time.Duration, defaultChoice int) *Decision {
	d.Timeout = timeout
	d.Default = defaultChoice
	return d
}

func (d *Decision) WithCondition(condition func() bool) *Decision {
	d.Condition = condition
	return d
}

func (d *Decision) WithOnChoice(onChoice func(int, *Context)) *Decision {
	d.OnChoice = onChoice
	return d
}

func (d *Decision) WithSource(sourceID, sourceType string) *Decision {
	d.SourceID = sourceID
	d.SourceType = sourceType
	return d
}

func (d *Decision) ShouldAsk() bool {
	if !d.NeedConfirm {
		return false
	}
	if d.Condition == nil {
		return true
	}
	return d.Condition()
}

func (d *Decision) WithNeedConfirm(need bool) *Decision {
	d.NeedConfirm = need
	return d
}

func (d *Decision) Execute(choice int, ctx *Context) {
	if choice < 0 || choice >= len(d.Options) {
		choice = d.Default
	}

	if d.Options[choice].Action != nil {
		d.Options[choice].Action(ctx)
	}

	if d.OnChoice != nil {
		d.OnChoice(choice, ctx)
	}
}

func (d *Decision) ExecuteTimeout(ctx *Context) bool {
	if d.Timeout <= 0 {
		return false
	}
	d.Execute(d.Default, ctx)
	return true
}

func (d *Decision) IsTimedOut(startTime time.Time) bool {
	if d.Timeout <= 0 {
		return false
	}
	return time.Since(startTime) >= d.Timeout
}

func (d *Decision) Clone() *Decision {
	return &Decision{
		ID:         id.NewDecisionID(),
		Prompt:     d.Prompt,
		Options:    d.Options,
		Priority:   d.Priority,
		Timeout:    d.Timeout,
		Default:    d.Default,
		Condition:  d.Condition,
		OnChoice:   d.OnChoice,
		SourceID:   "",
		SourceType: d.SourceType,
	}
}

// DecisionBuilder is a decision builder.
type DecisionBuilder struct {
	decision *Decision
}

func NewDecisionBuilder(prompt string) *DecisionBuilder {
	return &DecisionBuilder{
		decision: NewDecision(prompt, []Option{}),
	}
}

func (b *DecisionBuilder) AddOption(id, label string, action func(*Context)) *DecisionBuilder {
	b.decision.Options = append(b.decision.Options, Option{
		ID:     id,
		Label:  label,
		Action: action,
	})
	return b
}

func (b *DecisionBuilder) SetPriority(priority int) *DecisionBuilder {
	b.decision.Priority = priority
	return b
}

func (b *DecisionBuilder) SetTimeout(timeout time.Duration, defaultChoice int) *DecisionBuilder {
	b.decision.Timeout = timeout
	b.decision.Default = defaultChoice
	return b
}

func (b *DecisionBuilder) SetCondition(condition func() bool) *DecisionBuilder {
	b.decision.Condition = condition
	return b
}

func (b *DecisionBuilder) SetOnChoice(onChoice func(int, *Context)) *DecisionBuilder {
	b.decision.OnChoice = onChoice
	return b
}

func (b *DecisionBuilder) SetSource(sourceID, sourceType string) *DecisionBuilder {
	b.decision.SourceID = sourceID
	b.decision.SourceType = sourceType
	return b
}

func (b *DecisionBuilder) Build() *Decision {
	return b.decision
}