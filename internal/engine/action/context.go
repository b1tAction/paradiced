package action

import (
	"fmt"

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
type ActionContext struct {
	*util.Metadata // Embedded for extensible storage

	Game          protocol.Game         // Game instance (interface to avoid circular dependency with engine)
	EventBus      *event.EventBus       // EventBus for interception (nil if no interception)
	MapEngine     *gamemap.MapEngine    // MapEngine for movement calculation (direct type)
	DrawEngine    *rng.DrawEngine       // DrawEngine for random draws (events, buffs, items)
	EventPool     []*rng.EvaluatedItem  // Event pool for DrawEventAction (all events)
	ItemPool      []*rng.EvaluatedItem  // Item pool for DrawItemAction (all items)
	BuffPool      []*rng.EvaluatedItem  // Buff pool for DrawBuffAction (drawable buffs)
	ActionQueue   *Queue                // Queue for derived actions
	// Probability weights for cell-based draws
	ProbGood      float64               // Probability weight for Good pool
	ProbNeutral   float64               // Probability weight for Neutral pool
	ProbBad       float64               // Probability weight for Bad pool

	// processedCount tracks total actions processed in this context's lifecycle.
	// Shared across all nested ProcessQueue calls to enforce depth limit.
	processedCount int

	// Buff lifecycle callbacks - injected by HSM layer (engine.Game)
	// These handle EventBus subscription/unsubscription for Buff add/remove.
	OnAddBuff    func(player *core.Player, buff *core.Buff)    // Called after AddBuffAction.Execute
	OnRemoveBuff func(player *core.Player, buffType constants.BuffType) *core.Buff // Called by RemoveBuffAction.Execute

	// GetBuffDuration returns the default duration for a Buff type from BuffDefinition.
	// Injected by engine layer (BuffRegistry). Used by AddBuffAction/DeathAction to
	// look up duration from definition instead of hardcoding it.
	GetBuffDuration func(buffType constants.BuffType) int

	// Item lifecycle callbacks - injected by HSM layer (engine.Game)
	// These handle EventBus subscription/unsubscription for Item add/remove.
	OnAddItem    func(player *core.Player, item *core.Item)                     // Called after AddItemAction.Execute
	OnRemoveItem func(player *core.Player, itemType constants.ItemType) *core.Item // Called by RemoveItemAction.Execute
}

// NewActionContext creates a new ActionContext with required components.
func NewActionContext(game protocol.Game, bus *event.EventBus, mapEngine *gamemap.MapEngine, drawEngine *rng.DrawEngine) *ActionContext {
	return &ActionContext{
		Metadata:    util.NewMetadata(),
		Game:        game,
		EventBus:    bus,
		MapEngine:   mapEngine,
		DrawEngine:  drawEngine,
		ActionQueue: NewQueue(),
	}
}

// SetPools sets the event, item, and buff pools for draw actions.
func (ctx *ActionContext) SetPools(eventPool, itemPool, buffPool []*rng.EvaluatedItem) *ActionContext {
	ctx.EventPool = eventPool
	ctx.ItemPool = itemPool
	ctx.BuffPool = buffPool
	return ctx
}

// SetCellDraw sets the cell-based draw probabilities.
func (ctx *ActionContext) SetCellDraw(probGood, probNeutral, probBad float64) *ActionContext {
	ctx.ProbGood = probGood
	ctx.ProbNeutral = probNeutral
	ctx.ProbBad = probBad
	return ctx
}

// maxQueueProcessingDepth limits the number of actions processed in a single
// ProcessQueue cycle to prevent infinite loops from recursive derived actions.
const maxQueueProcessingDepth = 50

// ExecuteAction executes an action with interception support.
// Flow:
// 1. PhasePreAction - interception for ALL target players (DeathMark blocks dead, Dominance amplifies, RobLuck redirects)
// 2. PreTrigger phase - publish for interception (if not PhaseAnyTime)
// 3. Collect derived actions from handler into queue
// 4. Execute the (possibly modified) action
// 5. PostTrigger phase - publish for lifecycle events (if not PhaseAnyTime)
// 6. Collect derived actions from post-trigger handler
// 7. Record in global game log
// 8. Process any derived actions in queue
// Returns first error from handlers or action execution.
func (ctx *ActionContext) ExecuteAction(action Action) error {
	// Log action execution (nil-safe: GetDebugLog returns nil-safe GameLogger)
	if ctx.Game != nil {
		ctx.Game.GetDebugLog().Debug("Action.ExecuteAction", "type", action.Type(), "target", action.Target(), "source", action.Source())
	}

	// Step 0: PhasePreAction - interception for relevant players
	// DeathMark blocks actions on dead players, Dominance amplifies from actor,
	// RobLuck redirects beneficial actions to BaiHu player.
	if ctx.EventBus != nil {
		// Determine players to publish PhasePreAction to:
		// - Default: [TargetPlayer()] (most actions only need target publication)
		// - ActorPlayerer: custom list (e.g., BossDamageAction needs both Boss + attacker)
		playersToPublish := []*core.Player{}
		if actor, ok := action.(ActorPlayerer); ok {
			playersToPublish = actor.ActorPlayers()
		} else {
			targetPlayer := action.TargetPlayer()
			if targetPlayer != nil {
				playersToPublish = []*core.Player{targetPlayer}
			}
		}

		for _, player := range playersToPublish {
			preCtx := event.NewContext(player)
			preCtx.Set("action_context", ctx)
			preCtx.Set("current_action", action)
			ctx.EventBus.Publish(constants.PhasePreAction, player.ID.UUID(), preCtx)

			// Check for handler errors in PhasePreAction
			if preCtx.HasError() {
				return preCtx.FirstError()
			}

			// Collect derived actions from PhasePreAction handlers
			// (Dominance adds amplified actions, RobLuck adds redirected actions)
			for _, derived := range preCtx.GetDerivedActions() {
				if execAction, ok := derived.(Action); ok {
					ctx.PushDerivedAction(execAction)
				}
			}

			// Check if action was blocked (DeathMark, RobLuck)
			if preCtx.GetBoolOrDefault("action_blocked", false) {
				if ctx.Game != nil {
					blockReason := preCtx.GetStringOrDefault("blocked_by", "unknown")
					ctx.Game.GetDebugLog().Info("Action.ExecuteAction.blocked", "phase", "PhasePreAction", "action_type", action.Type(), "target", action.Target(), "player_id", player.ID.UUID(), "blocked_by", blockReason)
				}
				return nil // Action blocked, derived actions already pushed above
			}
		}
	}

	// Step 1: PreTrigger phase - interception
	prePhase := action.PreTriggerPhase()
	if prePhase != constants.PhaseAnyTime && ctx.EventBus != nil {
		// Create context for interception using action's target player
		triggerCtx := event.NewContext(action.TargetPlayer())
		triggerCtx.Set("current_action", action)
		triggerCtx.Set("action_context", ctx)

		// For RemoveBuffAction, set removed buff type for handler matching
		if removeAction, ok := action.(*RemoveBuffAction); ok {
			triggerCtx.Set("removed_buff_type", string(removeAction.BuffType))
		}

		// Publish to allow Buffs/Items to intercept/modify the action
		ctx.EventBus.Publish(prePhase, action.Target(), triggerCtx)

		// Check for handler errors
		if triggerCtx.HasError() {
			return triggerCtx.FirstError()
		}

		// Check if action was blocked
		if triggerCtx.GetBoolOrDefault("action_blocked", false) {
			if ctx.Game != nil {
				blockReason := triggerCtx.GetStringOrDefault("blocked_by", "unknown")
				ctx.Game.GetDebugLog().Info("Action.ExecuteAction.blocked", "phase", prePhase, "action_type", action.Type(), "target", action.Target(), "blocked_by", blockReason)
			}
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
		// Skip PostTrigger for AddBuffAction duration-extend (not a new buff application)
		if _, ok := action.(*AddBuffAction); ok && ctx.GetBoolOrDefault("buff_duration_extended", false) {
			// Duration extended, skip PhasePostBuffApplied publication
			// No derived actions from PostTrigger needed since buff was just extended, not newly applied
		} else {
			// Create context for post-trigger using action's target player
			triggerCtx := event.NewContext(action.TargetPlayer())
			triggerCtx.Set("current_action", action)
			triggerCtx.Set("action_context", ctx)

			// For AddBuffAction, set applied buff type for handler matching
			if addAction, ok := action.(*AddBuffAction); ok {
				triggerCtx.Set("applied_buff_type", string(addAction.BuffType))
			}

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
// Uses a shared processedCount to limit total depth across all nested calls,
// preventing infinite loops from recursive derived actions.
func (ctx *ActionContext) ProcessQueue() error {
	for !ctx.ActionQueue.IsEmpty() {
		if ctx.processedCount >= maxQueueProcessingDepth {
			return fmt.Errorf("action queue exceeded maximum depth (%d), possible infinite loop", maxQueueProcessingDepth)
		}
		action := ctx.ActionQueue.Pop()
		ctx.processedCount++
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
	ctx.processedCount = 0
}

// GetGameLog returns the global game log (helper method).
func (ctx *ActionContext) GetGameLog() *gamelog.GameLog {
	if ctx.Game == nil {
		return nil
	}
	return ctx.Game.GetGameLog()
}