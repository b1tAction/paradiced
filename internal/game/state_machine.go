package game

import (
	"sort"

	"github.com/b1tAction/Fated/pkg/event"
)

// StateMachine 状态机，处理Phase触发和用户决策等待
type StateMachine struct {
	Game        *Game
	WaitingFor  []*event.Decision // 当前等待的决策列表
	CurrentCtx  *event.Context    // 当前上下文
	FlowState   string            // 流程状态标记
}

// NewStateMachine 创建状态机
func NewStateMachine(game *Game) *StateMachine {
	return &StateMachine{
		Game:       game,
		WaitingFor: make([]*event.Decision, 0),
		FlowState:  "idle",
	}
}

// TriggerPhase 触发某个Phase
// 返回需要用户确认的Decision列表
// 如果返回非空列表，状态机进入等待状态，流程暂停
func (sm *StateMachine) TriggerPhase(phase event.Phase, player *Player) []*event.Decision {
	return sm.TriggerPhaseWithCtx(phase, player, nil)
}

// TriggerPhaseWithCtx 触发某个Phase，可传入预设的Context
func (sm *StateMachine) TriggerPhaseWithCtx(phase event.Phase, player *Player, presetCtx *event.Context) []*event.Decision {
	// 创建或使用预设上下文
	ctx := presetCtx
	if ctx == nil {
		ctx = event.NewContext(player)
	}

	// 设置游戏状态
	ctx = ctx.WithState(&event.GameState{
		Round:        sm.Game.State.Round,
		Turn:         sm.Game.State.Turn,
		CurrentPhase: phase,
	})

	sm.CurrentCtx = ctx

	// 发布Phase事件
	decisions := sm.Game.Bus.Publish(phase, player.UserID, ctx)

	// 按Priority排序（高优先级先处理）
	sort.Slice(decisions, func(i, j int) bool {
		return decisions[i].Priority > decisions[j].Priority
	})

	return decisions
}

// TriggerPhaseAndWait 触发Phase并进入等待状态
// 如果有需要确认的Decision，返回true表示需要等待用户输入
func (sm *StateMachine) TriggerPhaseAndWait(phase event.Phase, player *Player) bool {
	decisions := sm.TriggerPhase(phase, player)

	if len(decisions) > 0 {
		sm.EnterWaitingState(decisions)
		return true // 需要等待用户输入
	}

	return false // 无需等待，流程继续
}

// EnterWaitingState 进入等待状态
func (sm *StateMachine) EnterWaitingState(decisions []*event.Decision) {
	sm.WaitingFor = decisions
	sm.FlowState = "waiting"
	sm.Game.State.Waiting = true
}

// OnUserChoice 处理用户选择
func (sm *StateMachine) OnUserChoice(choice int) {
	if len(sm.WaitingFor) == 0 {
		return // 没有等待的决策
	}

	// 处理第一个Decision
	current := sm.WaitingFor[0]
	current.Execute(choice, sm.CurrentCtx)

	// 移除已处理的Decision
	sm.WaitingFor = sm.WaitingFor[1:]

	if len(sm.WaitingFor) > 0 {
		// 继续等待下一个Decision
		// 这里可以发送下一个Decision给客户端
	} else {
		// 所有Decision处理完毕，退出等待状态
		sm.ExitWaitingState()
	}
}

// ExitWaitingState 退出等待状态
func (sm *StateMachine) ExitWaitingState() {
	sm.WaitingFor = make([]*event.Decision, 0)
	sm.FlowState = "running"
	sm.Game.State.Waiting = false
}

// GetCurrentDecision 获取当前需要用户确认的Decision
func (sm *StateMachine) GetCurrentDecision() *event.Decision {
	if len(sm.WaitingFor) > 0 {
		return sm.WaitingFor[0]
	}
	return nil
}

// IsWaiting 检查是否在等待用户输入
func (sm *StateMachine) IsWaiting() bool {
	return sm.FlowState == "waiting"
}

// GetWaitingCount 获取等待的Decision数量
func (sm *StateMachine) GetWaitingCount() int {
	return len(sm.WaitingFor)
}

// ContinueFlow 继续流程（用于外部调用）
func (sm *StateMachine) ContinueFlow() {
	sm.ExitWaitingState()
}

// CancelWaiting 取消等待状态（超时或异常情况）
func (sm *StateMachine) CancelWaiting() {
	// 执行所有剩余Decision的默认选项
	for _, decision := range sm.WaitingFor {
		decision.Execute(decision.Default, sm.CurrentCtx)
	}
	sm.ExitWaitingState()
}

// ========== Phase执行器 ==========

// ExecuteBeforeTurnPhase 执行回合开始前Phase
func (sm *StateMachine) ExecuteBeforeTurnPhase(player *Player) []*event.Decision {
	return sm.TriggerPhase(event.PhaseBeforeTurn, player)
}

// ExecuteOnMovePhase 执行移动Phase
func (sm *StateMachine) ExecuteOnMovePhase(player *Player) []*event.Decision {
	return sm.TriggerPhase(event.PhaseOnMove, player)
}

// ExecuteOnLandPhase 执行落地Phase
func (sm *StateMachine) ExecuteOnLandPhase(player *Player) []*event.Decision {
	return sm.TriggerPhase(event.PhaseOnLand, player)
}

// ExecutePreEventPhase 执行事件前Phase
func (sm *StateMachine) ExecutePreEventPhase(player *Player) []*event.Decision {
	return sm.TriggerPhase(event.PhasePreEvent, player)
}

// ExecutePreDamagePhase 执行受伤前Phase
func (sm *StateMachine) ExecutePreDamagePhase(player *Player, damage int) []*event.Decision {
	ctx := event.NewContext(player).WithData(damage)
	return sm.TriggerPhaseWithCtx(event.PhasePreDamage, player, ctx)
}

// ExecuteAfterTurnPhase 执行回合结束后Phase
func (sm *StateMachine) ExecuteAfterTurnPhase(player *Player) []*event.Decision {
	return sm.TriggerPhase(event.PhaseAfterTurn, player)
}