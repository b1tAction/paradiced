package engine

import (
	"sort"

	"github.com/b1tAction/Fated/internal/core"
	"github.com/b1tAction/Fated/pkg/event"
)

// StateMachine is the state machine for handling Phase triggers and user decision waiting.
type StateMachine struct {
	Game        *Game
	WaitingFor  []*event.Decision // Current waiting decision list
	CurrentCtx  *event.Context    // Current context
	FlowState   string            // Flow state marker
}

// NewStateMachine creates a state machine.
func NewStateMachine(game *Game) *StateMachine {
	return &StateMachine{
		Game:       game,
		WaitingFor: make([]*event.Decision, 0),
		FlowState:  "idle",
	}
}

// TriggerPhase triggers a Phase.
func (sm *StateMachine) TriggerPhase(phase event.Phase, player *core.Player) []*event.Decision {
	return sm.TriggerPhaseWithCtx(phase, player, nil)
}

// TriggerPhaseWithCtx triggers a Phase with preset Context.
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

// TriggerPhaseAndWait triggers Phase and enters waiting state.
func (sm *StateMachine) TriggerPhaseAndWait(phase event.Phase, player *core.Player) bool {
	decisions := sm.TriggerPhase(phase, player)

	if len(decisions) > 0 {
		sm.EnterWaitingState(decisions)
		return true
	}

	return false
}

// EnterWaitingState enters waiting state.
func (sm *StateMachine) EnterWaitingState(decisions []*event.Decision) {
	sm.WaitingFor = decisions
	sm.FlowState = "waiting"
	sm.Game.State.Waiting = true
}

// OnUserChoice handles user choice.
func (sm *StateMachine) OnUserChoice(choice int) {
	if len(sm.WaitingFor) == 0 {
		return
	}

	current := sm.WaitingFor[0]
	current.Execute(choice, sm.CurrentCtx)

	sm.WaitingFor = sm.WaitingFor[1:]

	if len(sm.WaitingFor) > 0 {
		// Continue waiting for next Decision
	} else {
		sm.ExitWaitingState()
	}
}

// ExitWaitingState exits waiting state.
func (sm *StateMachine) ExitWaitingState() {
	sm.WaitingFor = make([]*event.Decision, 0)
	sm.FlowState = "running"
	sm.Game.State.Waiting = false
}

// GetCurrentDecision returns the current Decision needing user confirmation.
func (sm *StateMachine) GetCurrentDecision() *event.Decision {
	if len(sm.WaitingFor) > 0 {
		return sm.WaitingFor[0]
	}
	return nil
}

// IsWaiting checks if waiting for user input.
func (sm *StateMachine) IsWaiting() bool {
	return sm.FlowState == "waiting"
}

// GetWaitingCount returns waiting Decision count.
func (sm *StateMachine) GetWaitingCount() int {
	return len(sm.WaitingFor)
}

// ContinueFlow continues flow.
func (sm *StateMachine) ContinueFlow() {
	sm.ExitWaitingState()
}

// CancelWaiting cancels waiting state.
func (sm *StateMachine) CancelWaiting() {
	for _, decision := range sm.WaitingFor {
		decision.Execute(decision.Default, sm.CurrentCtx)
	}
	sm.ExitWaitingState()
}

// ========== Phase Executors ==========

// ExecuteBeforeTurnPhase executes BeforeTurn Phase.
func (sm *StateMachine) ExecuteBeforeTurnPhase(player *core.Player) []*event.Decision {
	return sm.TriggerPhase(event.PhaseBeforeTurn, player)
}

// ExecuteOnMovePhase executes OnMove Phase.
func (sm *StateMachine) ExecuteOnMovePhase(player *core.Player) []*event.Decision {
	return sm.TriggerPhase(event.PhaseOnMove, player)
}

// ExecuteOnLandPhase executes OnLand Phase.
func (sm *StateMachine) ExecuteOnLandPhase(player *core.Player) []*event.Decision {
	return sm.TriggerPhase(event.PhaseOnLand, player)
}

// ExecutePreEventPhase executes PreEvent Phase.
func (sm *StateMachine) ExecutePreEventPhase(player *core.Player) []*event.Decision {
	return sm.TriggerPhase(event.PhasePreEvent, player)
}

// ExecutePreDamagePhase executes PreDamage Phase.
func (sm *StateMachine) ExecutePreDamagePhase(player *core.Player, damage int) []*event.Decision {
	ctx := event.NewContext(player).WithData(damage)
	return sm.TriggerPhaseWithCtx(event.PhasePreDamage, player, ctx)
}

// ExecuteAfterTurnPhase executes AfterTurn Phase.
func (sm *StateMachine) ExecuteAfterTurnPhase(player *core.Player) []*event.Decision {
	return sm.TriggerPhase(event.PhaseAfterTurn, player)
}