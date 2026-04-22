package action

import (
	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/internal/gamemap"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/gamelog"
	"github.com/b1tAction/paradiced/pkg/protocol"
	"github.com/b1tAction/paradiced/pkg/rng"
	"github.com/b1tAction/paradiced/pkg/util"
)

// ActionContext provides context for action execution.
// Contains references to game engine, event bus, and map engine for executing actions.
// Embeds util.Metadata for extensible type-safe key-value storage.
// CurrentPlayer is set by HSM when creating the context (from HSM.GetTurnPlayer()).
type ActionContext struct {
	*util.Metadata // Embedded for extensible storage

	Game          protocol.Game         // Game instance (interface to avoid circular dependency with engine)
	EventBus      *event.EventBus       // EventBus for interception (nil if no interception)
	MapEngine     *gamemap.MapEngine    // MapEngine for movement calculation (direct type)
	DrawEngine    *rng.DrawEngine       // DrawEngine for random draws (events, buffs, items)
	EventPool     []*rng.EvaluatedItem  // Event pool for DrawEventAction (all events)
	ItemPool      []*rng.EvaluatedItem  // Item pool for DrawItemAction (all items)
	ActionQueue   *Queue                // Queue for derived actions
	CurrentPlayer *core.Player          // Current player (set by HSM, nil if not in turn)
	// Probability weights for cell-based draws
	ProbGood      float64               // Probability weight for Good pool
	ProbNeutral   float64               // Probability weight for Neutral pool
	ProbBad       float64               // Probability weight for Bad pool
}

// NewActionContext creates a new ActionContext with required components.
func NewActionContext(game protocol.Game, bus *event.EventBus, mapEngine *gamemap.MapEngine, drawEngine *rng.DrawEngine) *ActionContext {
	return &ActionContext{
		Metadata:      util.NewMetadata(),
		Game:          game,
		EventBus:      bus,
		MapEngine:     mapEngine,
		DrawEngine:    drawEngine,
		ActionQueue:   NewQueue(),
		CurrentPlayer: nil, // Set separately via SetCurrentPlayer
	}
}

// NewActionContextWithPlayer creates a new ActionContext with current player.
func NewActionContextWithPlayer(game protocol.Game, bus *event.EventBus, mapEngine *gamemap.MapEngine, drawEngine *rng.DrawEngine, player *core.Player) *ActionContext {
	return &ActionContext{
		Metadata:      util.NewMetadata(),
		Game:          game,
		EventBus:      bus,
		MapEngine:     mapEngine,
		DrawEngine:    drawEngine,
		ActionQueue:   NewQueue(),
		CurrentPlayer: player,
	}
}

// SetPools sets the event and item pools for draw actions.
func (ctx *ActionContext) SetPools(eventPool, itemPool []*rng.EvaluatedItem) *ActionContext {
	ctx.EventPool = eventPool
	ctx.ItemPool = itemPool
	return ctx
}

// SetCellDraw sets the cell-based draw probabilities.
func (ctx *ActionContext) SetCellDraw(probGood, probNeutral, probBad float64) *ActionContext {
	ctx.ProbGood = probGood
	ctx.ProbNeutral = probNeutral
	ctx.ProbBad = probBad
	return ctx
}

// SetCurrentPlayer sets the current player for trigger context.
func (ctx *ActionContext) SetCurrentPlayer(player *core.Player) {
	ctx.CurrentPlayer = player
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
// Returns first error from handlers or action execution.
func (ctx *ActionContext) ExecuteAction(action Action) error {
	// Step 1: PreTrigger phase - interception
	prePhase := action.PreTriggerPhase()
	if prePhase != constants.PhaseAnyTime && ctx.EventBus != nil {
		// Create context for interception
		// Use CurrentPlayer set by HSM (from HSM.GetTurnPlayer())
		triggerCtx := event.NewContext(ctx.CurrentPlayer)
		triggerCtx.Set("current_action", action)
		triggerCtx.Set("action_context", ctx)

		// Publish to allow Buffs/Items to intercept/modify the action
		ctx.EventBus.Publish(prePhase, action.Target(), triggerCtx)

		// Check for handler errors
		if triggerCtx.HasError() {
			return triggerCtx.FirstError()
		}

		// Check if action was blocked
		if triggerCtx.GetBoolOrDefault("action_blocked", false) {
			// Action blocked, but still process any derived actions from the interceptor
			for _, derived := range triggerCtx.GetDerivedActions() {
				if execAction, ok := derived.(Action); ok {
					ctx.PushDerivedAction(execAction)
				}
			}
			return nil
		}

		// Step 2: Collect derived actions from PreTrigger handler
		for _, derived := range triggerCtx.GetDerivedActions() {
			if execAction, ok := derived.(Action); ok {
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
	if postPhase != constants.PhaseAnyTime && ctx.EventBus != nil {
		// Create context for post-trigger
		triggerCtx := event.NewContext(ctx.CurrentPlayer)
		triggerCtx.Set("current_action", action)
		triggerCtx.Set("action_context", ctx)

		// Publish for buff entry effects, death effects, chain reactions
		ctx.EventBus.Publish(postPhase, action.Target(), triggerCtx)

		// Check for handler errors
		if triggerCtx.HasError() {
			return triggerCtx.FirstError()
		}

		// Step 5: Collect derived actions from PostTrigger handler
		for _, derived := range triggerCtx.GetDerivedActions() {
			if execAction, ok := derived.(Action); ok {
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
	if err := ctx.ProcessQueue(); err != nil {
		return err
	}

	return nil
}

// ProcessQueue executes all actions in the queue.
// Returns first error encountered, or nil if all succeeded.
func (ctx *ActionContext) ProcessQueue() error {
	for !ctx.ActionQueue.IsEmpty() {
		action := ctx.ActionQueue.Pop()
		if err := ctx.ExecuteAction(action); err != nil {
			return err
		}
	}
	return nil
}

// PushDerivedAction adds a derived action to the queue.
// Used by handlers to generate new actions during interception.
func (ctx *ActionContext) PushDerivedAction(action Action) {
	ctx.ActionQueue.Push(action)
}

// Clear resets the context for a new turn.
func (ctx *ActionContext) Clear() {
	ctx.ActionQueue.Clear()
	ctx.Metadata.Clear()
}

// GetGameLog returns the global game log (helper method).
func (ctx *ActionContext) GetGameLog() *gamelog.GameLog {
	if ctx.Game == nil {
		return nil
	}
	return ctx.Game.GetGameLog()
}