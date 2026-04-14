package action

import (
	"github.com/b1tAction/Fated/pkg/event"
	"github.com/b1tAction/Fated/pkg/gamelog"
	"github.com/b1tAction/Fated/pkg/protocol"
	"github.com/b1tAction/Fated/pkg/util"
)

// ActionContext provides context for action execution.
// Contains references to game engine, event bus, and map engine for executing actions.
// Embeds util.Metadata for extensible type-safe key-value storage.
type ActionContext struct {
	*util.Metadata // Embedded for extensible storage

	Game        protocol.Game      // Game instance (interface to avoid circular dependency)
	EventBus    *event.EventBus    // EventBus for interception (nil if no interception)
	MapEngine   protocol.MapEngine // MapEngine for movement calculation
	ActionQueue *Queue             // Queue for derived actions
}

// NewActionContext creates a new ActionContext with required components.
func NewActionContext(game protocol.Game, bus *event.EventBus, mapEngine protocol.MapEngine) *ActionContext {
	return &ActionContext{
		Metadata:    util.NewMetadata(),
		Game:        game,
		EventBus:    bus,
		MapEngine:   mapEngine,
		ActionQueue: NewQueue(),
	}
}

// ExecuteAction executes an action with interception support.
// Flow:
// 1. PreTrigger phase - publish for interception (if not PhaseAnyTime)
// 2. Collect derived actions from handler into queue
// 3. Execute the (possibly modified) action
// 4. PostTrigger phase - publish for lifecycle events (if not PhaseAnyTime)
// 5. Collect derived actions from post-trigger handler
// 6. Record in global game log
// 7. Process any derived actions in queue
func (ctx *ActionContext) ExecuteAction(action ExecutableAction) error {
	// Step 1: PreTrigger phase - interception
	prePhase := action.PreTriggerPhase()
	if prePhase != event.PhaseAnyTime && ctx.EventBus != nil {
		// Create context for interception
		triggerCtx := event.NewContext(ctx.Game.GetCurrentPlayer())
		triggerCtx.Set("current_action", action)
		triggerCtx.Set("action_context", ctx)

		// Publish to allow Buffs/Items to intercept/modify the action
		ctx.EventBus.Publish(prePhase, action.Target(), triggerCtx)

		// Check if action was blocked
		if triggerCtx.GetBoolOrDefault("action_blocked", false) {
			// Action blocked, but still process any derived actions from the interceptor
			for _, derived := range triggerCtx.GetDerivedActions() {
				if execAction, ok := derived.(ExecutableAction); ok {
					ctx.PushDerivedAction(execAction)
				}
			}
			return nil
		}

		// Step 2: Collect derived actions from PreTrigger handler
		for _, derived := range triggerCtx.GetDerivedActions() {
			if execAction, ok := derived.(ExecutableAction); ok {
				ctx.PushDerivedAction(execAction)
			}
		}
	}

	// Step 3: Execute the action
	err := action.Execute(ctx)
	if err != nil {
		return err
	}

	// Step 4: PostTrigger phase - lifecycle events
	postPhase := action.PostTriggerPhase()
	if postPhase != event.PhaseAnyTime && ctx.EventBus != nil {
		// Create context for post-trigger
		triggerCtx := event.NewContext(ctx.Game.GetCurrentPlayer())
		triggerCtx.Set("current_action", action)
		triggerCtx.Set("action_context", ctx)

		// Publish for buff entry effects, death effects, chain reactions
		ctx.EventBus.Publish(postPhase, action.Target(), triggerCtx)

		// Step 5: Collect derived actions from PostTrigger handler
		for _, derived := range triggerCtx.GetDerivedActions() {
			if execAction, ok := derived.(ExecutableAction); ok {
				ctx.PushDerivedAction(execAction)
			}
		}
	}

	// Step 6: Record in global game log
	if ctx.Game != nil {
		gameLog := ctx.Game.GetGameLog()
		if gameLog != nil {
			gameLog.AddEntry(action.LogEntry())
		}
	}

	// Step 7: Process derived actions in queue
	ctx.ProcessQueue()

	return nil
}

// ProcessQueue executes all actions in the queue.
func (ctx *ActionContext) ProcessQueue() {
	for !ctx.ActionQueue.IsEmpty() {
		action := ctx.ActionQueue.Pop()
		ctx.ExecuteAction(action)
	}
}

// PushDerivedAction adds a derived action to the queue.
// Used by handlers to generate new actions during interception.
func (ctx *ActionContext) PushDerivedAction(action ExecutableAction) {
	ctx.ActionQueue.Push(action)
}

// Clear resets the context for a new turn.
func (ctx *ActionContext) Clear() {
	ctx.ActionQueue.Clear()
	ctx.Metadata.Clear()
}

// GetGameLog returns the global game log (helper method).
func (ctx *ActionContext) GetGameLog() *gamelog.GameLog {
	return ctx.Game.GetGameLog()
}