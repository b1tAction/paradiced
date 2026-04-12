package engine

import (
	"sort"

	"github.com/b1tAction/Fated/internal/core"
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
func (sm *StateMachine) TriggerPhase(phase event.Phase, player *core.Player) []*event.Decision {
	return sm.TriggerPhaseWithCtx(phase, player, nil)
}

// TriggerPhaseWithCtx 触发某个Phase，可传入预设的Context
func (sm *StateMachine) TriggerPhaseWithCtx(phase event.Phase, player *core.Player, presetCtx *event.Context) []*event.Decision {
	ctx := presetCtx
	if ctx == nil {
		ctx = event.NewContext(player)
	}

	ctx = ctx.WithState(&event.GameState{
		Round:        sm.Game.State.Round,
		Turn:         sm.Game.State.Turn,
		CurrentPhase: phase,
	})

	sm.CurrentCtx = ctx

	decisions := sm.Game.Bus.Publish(phase, player.UserID, ctx)

	sort.Slice(decisions, func(i, j int) bool {
		return decisions[i].Priority > decisions[j].Priority
	})

	return decisions
}

// TriggerPhaseAndWait 触发Phase并进入等待状态
func (sm *StateMachine) TriggerPhaseAndWait(phase event.Phase, player *core.Player) bool {
	decisions := sm.TriggerPhase(phase, player)

	if len(decisions) > 0 {
		sm.EnterWaitingState(decisions)
		return true
	}

	return false
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
		return
	}

	current := sm.WaitingFor[0]
	current.Execute(choice, sm.CurrentCtx)

	sm.WaitingFor = sm.WaitingFor[1:]

	if len(sm.WaitingFor) > 0 {
		// 继续等待下一个Decision
	} else {
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

// ContinueFlow 继续流程
func (sm *StateMachine) ContinueFlow() {
	sm.ExitWaitingState()
}

// CancelWaiting 取消等待状态
func (sm *StateMachine) CancelWaiting() {
	for _, decision := range sm.WaitingFor {
		decision.Execute(decision.Default, sm.CurrentCtx)
	}
	sm.ExitWaitingState()
}

// ========== Phase执行器 ==========

// ExecuteBeforeTurnPhase 执行回合开始前Phase
func (sm *StateMachine) ExecuteBeforeTurnPhase(player *core.Player) []*event.Decision {
	return sm.TriggerPhase(event.PhaseBeforeTurn, player)
}

// ExecuteOnMovePhase 执行移动Phase
func (sm *StateMachine) ExecuteOnMovePhase(player *core.Player) []*event.Decision {
	return sm.TriggerPhase(event.PhaseOnMove, player)
}

// ExecuteOnLandPhase 执行落地Phase
func (sm *StateMachine) ExecuteOnLandPhase(player *core.Player) []*event.Decision {
	return sm.TriggerPhase(event.PhaseOnLand, player)
}

// ExecutePreEventPhase 执行事件前Phase
func (sm *StateMachine) ExecutePreEventPhase(player *core.Player) []*event.Decision {
	return sm.TriggerPhase(event.PhasePreEvent, player)
}

// ExecutePreDamagePhase 执行受伤前Phase
func (sm *StateMachine) ExecutePreDamagePhase(player *core.Player, damage int) []*event.Decision {
	ctx := event.NewContext(player).WithData(damage)
	return sm.TriggerPhaseWithCtx(event.PhasePreDamage, player, ctx)
}

// ExecuteAfterTurnPhase 执行回合结束后Phase
func (sm *StateMachine) ExecuteAfterTurnPhase(player *core.Player) []*event.Decision {
	return sm.TriggerPhase(event.PhaseAfterTurn, player)
}