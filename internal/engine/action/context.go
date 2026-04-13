package action

import (
	"github.com/b1tAction/Fated/internal/core"
	"github.com/b1tAction/Fated/pkg/event"
	"github.com/b1tAction/Fated/pkg/util"
)

// GameInterface defines the interface ActionContext needs from Game.
// Avoids circular dependency by not importing engine package directly.
type GameInterface interface {
	// GetCurrentPlayer returns the current active player.
	GetCurrentPlayer() *core.Player
	// GetPlayer returns a player by ID.
	GetPlayer(id string) *core.Player
	// GetPlayers returns all players.
	GetPlayers() []*core.Player
}

// PathResultInterface defines the interface for movement path results.
// Used by MoveAction to get path calculation results.
type PathResultInterface interface {
	// GetTargetIndex returns the target position.
	GetTargetIndex() int
	// GetPath returns the path of visited cells.
	GetPath() []int
}

// MapEngineInterface defines the minimal interface ActionContext needs for movement.
// Avoids circular dependency by not importing hsm or gamemap packages.
type MapEngineInterface interface {
	// CalculatePath calculates movement path from start position with given steps.
	CalculatePath(startPos int, steps int) (PathResultInterface, error)
}

// ActionContext provides context for action execution.
// Contains references to game engine, event bus, and map engine for executing actions.
// Embeds util.Metadata for extensible type-safe key-value storage.
type ActionContext struct {
	*util.Metadata // Embedded for extensible storage

	Game        GameInterface      // Game instance (interface to avoid circular dependency)
	EventBus    *event.EventBus    // EventBus for interception (nil if no interception)
	MapEngine   MapEngineInterface // MapEngine for movement calculation
	ActionQueue *Queue             // Queue for derived actions
	EventLog    *TurnEventLog      // Log for recording events
}

// NewActionContext creates a new ActionContext with required components.
func NewActionContext(game GameInterface, bus *event.EventBus, mapEngine MapEngineInterface) *ActionContext {
	return &ActionContext{
		Metadata:    util.NewMetadata(),
		Game:        game,
		EventBus:    bus,
		MapEngine:   mapEngine,
		ActionQueue: NewQueue(),
		EventLog:    NewTurnEventLog(),
	}
}

// ExecuteAction executes an action with interception support.
// Flow:
// 1. PreTrigger phase - publish for interception (if not PhaseAnyTime)
// 2. Execute the (possibly modified) action
// 3. PostTrigger phase - publish for lifecycle events (if not PhaseAnyTime)
// 4. Record in event log
// 5. Process any derived actions in queue
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
			return nil // Action blocked, skip execution
		}
	}

	// Step 2: Execute the action
	err := action.Execute(ctx)
	if err != nil {
		return err
	}

	// Step 3: PostTrigger phase - lifecycle events
	postPhase := action.PostTriggerPhase()
	if postPhase != event.PhaseAnyTime && ctx.EventBus != nil {
		// Create context for post-trigger
		triggerCtx := event.NewContext(ctx.Game.GetCurrentPlayer())
		triggerCtx.Set("current_action", action)
		triggerCtx.Set("action_context", ctx)

		// Publish for buff entry effects, death effects, chain reactions
		ctx.EventBus.Publish(postPhase, action.Target(), triggerCtx)
	}

	// Step 4: Record in event log
	ctx.EventLog.AddEntry(action.LogEntry())

	// Step 5: Process derived actions in queue
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
	ctx.EventLog.Clear()
}