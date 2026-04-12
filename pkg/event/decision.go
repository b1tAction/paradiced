package event

import (
	"time"
)

// Option 决策选项
type Option struct {
	ID     string         `json:"id"`    // 选项ID
	Label  string         `json:"label"` // 选项显示文本
	Action func(*Context) `json:"-"`     // 选择后执行的动作
}

// Decision 用户决策请求
// 用于需要用户确认的效果（如是否使用护盾）
type Decision struct {
	ID         string              `json:"id"`           // 决策ID
	Prompt     string              `json:"prompt"`       // 提示文本
	Options    []Option            `json:"options"`      // 可选项列表
	Priority   int                 `json:"priority"`     // 执行优先级（高先执行）
	Timeout    time.Duration       `json:"timeout"`      // 超时时间（可选）
	Default    int                 `json:"default"`      // 超时默认选项索引
	NeedConfirm bool               `json:"need_confirm"` // 是否需要用户确认（默认true）
	Condition  func() bool         `json:"-"`            // 动态判断条件（可选）
	OnChoice   func(int, *Context) `json:"-"`            // 用户选择后回调
	SourceID   string              `json:"source_id"`    // 来源ID（Buff/道具）
	SourceType string              `json:"source_type"`  // 来源类型 "buff" / "item"
}

// NewDecision 创建新的决策（默认需要确认）
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

// NewAutoDecision 创建自动执行的决策（不需要确认）
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

// WithPriority 设置优先级
func (d *Decision) WithPriority(priority int) *Decision {
	d.Priority = priority
	return d
}

// WithTimeout 设置超时
func (d *Decision) WithTimeout(timeout time.Duration, defaultChoice int) *Decision {
	d.Timeout = timeout
	d.Default = defaultChoice
	return d
}

// WithCondition 设置条件
func (d *Decision) WithCondition(condition func() bool) *Decision {
	d.Condition = condition
	return d
}

// WithOnChoice 设置选择回调
func (d *Decision) WithOnChoice(onChoice func(int, *Context)) *Decision {
	d.OnChoice = onChoice
	return d
}

// WithSource 设置来源信息
func (d *Decision) WithSource(sourceID, sourceType string) *Decision {
	d.SourceID = sourceID
	d.SourceType = sourceType
	return d
}

// ShouldAsk 判断是否需要询问用户
func (d *Decision) ShouldAsk() bool {
	// NeedConfirm=false时直接执行，不询问
	if !d.NeedConfirm {
		return false
	}
	// NeedConfirm=true时，检查Condition
	if d.Condition == nil {
		return true // 无条件则总是询问
	}
	return d.Condition()
}

// WithNeedConfirm 设置是否需要确认
func (d *Decision) WithNeedConfirm(need bool) *Decision {
	d.NeedConfirm = need
	return d
}

// Execute 执行用户选择的选项
func (d *Decision) Execute(choice int, ctx *Context) {
	if choice < 0 || choice >= len(d.Options) {
		choice = d.Default // 使用默认选项
	}

	// 执行选项动作
	if d.Options[choice].Action != nil {
		d.Options[choice].Action(ctx)
	}

	// 执行回调
	if d.OnChoice != nil {
		d.OnChoice(choice, ctx)
	}
}

// Clone 克隆决策（用于模板）
func (d *Decision) Clone() *Decision {
	return &Decision{
		ID:         newID(),
		Prompt:     d.Prompt,
		Options:    d.Options, // 共享Options引用，Action函数不可变
		Priority:   d.Priority,
		Timeout:    d.Timeout,
		Default:    d.Default,
		Condition:  d.Condition,
		OnChoice:   d.OnChoice,
		SourceID:   "", // 克隆时清空，由调用者设置
		SourceType: d.SourceType,
	}
}

// DecisionBuilder 决策构建器（简化创建流程）
type DecisionBuilder struct {
	decision *Decision
}

// NewDecisionBuilder 创建决策构建器
func NewDecisionBuilder(prompt string) *DecisionBuilder {
	return &DecisionBuilder{
		decision: NewDecision(prompt, []Option{}),
	}
}

// AddOption 添加选项
func (b *DecisionBuilder) AddOption(id, label string, action func(*Context)) *DecisionBuilder {
	b.decision.Options = append(b.decision.Options, Option{
		ID:     id,
		Label:  label,
		Action: action,
	})
	return b
}

// SetPriority 设置优先级
func (b *DecisionBuilder) SetPriority(priority int) *DecisionBuilder {
	b.decision.Priority = priority
	return b
}

// SetTimeout 设置超时
func (b *DecisionBuilder) SetTimeout(timeout time.Duration, defaultChoice int) *DecisionBuilder {
	b.decision.Timeout = timeout
	b.decision.Default = defaultChoice
	return b
}

// SetCondition 设置条件
func (b *DecisionBuilder) SetCondition(condition func() bool) *DecisionBuilder {
	b.decision.Condition = condition
	return b
}

// SetOnChoice 设置回调
func (b *DecisionBuilder) SetOnChoice(onChoice func(int, *Context)) *DecisionBuilder {
	b.decision.OnChoice = onChoice
	return b
}

// SetSource 设置来源
func (b *DecisionBuilder) SetSource(sourceID, sourceType string) *DecisionBuilder {
	b.decision.SourceID = sourceID
	b.decision.SourceType = sourceType
	return b
}

// Build 构建决策
func (b *DecisionBuilder) Build() *Decision {
	return b.decision
}
