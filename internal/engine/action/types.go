package action

import (
	"math/rand"
	"strings"
	"time"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/errors"
	"github.com/b1tAction/paradiced/pkg/gamelog"
	"github.com/b1tAction/paradiced/pkg/rng"
	"github.com/b1tAction/paradiced/pkg/util"
)

// ========== DamageAction ==========

// DamageAction represents HP reduction.
// Can be intercepted at PhasePreDamage to reduce or block damage amount.
type DamageAction struct {
	targetPlayer *core.Player // Player receiving damage
	SourceID     string       // Source identifier (e.g., "Buff_Curse", "Event_Trap")
	Amount       int          // Damage amount (can be modified by interceptors)
	IsPiercing   bool         // True if ignores shields (cannot be intercepted)
	BlockedBy    string       // Set by interceptor to identify blocking source
}

// NewDamageAction creates a new DamageAction.
func NewDamageAction(target *core.Player, amount int, sourceID string) *DamageAction {
	return &DamageAction{
		targetPlayer: target,
		SourceID:     sourceID,
		Amount:       amount,
		IsPiercing:   false,
	}
}

// NewPiercingDamageAction creates a piercing DamageAction that cannot be intercepted.
func NewPiercingDamageAction(target *core.Player, amount int, sourceID string) *DamageAction {
	return &DamageAction{
		targetPlayer: target,
		SourceID:     sourceID,
		Amount:       amount,
		IsPiercing:   true,
	}
}

func (a *DamageAction) Type() constants.ActionType { return constants.ActionDamage }
func (a *DamageAction) CanModify() bool            { return !a.IsPiercing && a.Amount > 0 }
func (a *DamageAction) Source() string             { return a.SourceID }
func (a *DamageAction) Target() string             { return a.targetPlayer.ID.UUID() }
func (a *DamageAction) TargetPlayer() *core.Player { return a.targetPlayer }
func (a *DamageAction) PreTriggerPhase() constants.Phase {
	if a.IsPiercing {
		return constants.PhaseAnyTime // Piercing damage cannot be intercepted
	}
	return constants.PhasePreDamage
}

// PostTriggerPhase returns PhaseAnyTime (no post-trigger for damage).
func (a *DamageAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

func (a *DamageAction) Execute(ctx *ActionContext) error {
	if a.Amount <= 0 {
		return nil // Already blocked by interception
	}
	if a.targetPlayer == nil {
		return errors.NewActionExecutionError("damage", "", "target player is nil", nil)
	}
	if err := a.targetPlayer.ApplyDamage(a.Amount); err != nil {
		return errors.NewActionExecutionError("damage", a.targetPlayer.ID.UUID(), "failed to apply damage", err)
	}
	// Derive DeathAction if player died from this damage
	if a.targetPlayer.IsDead {
		ctx.PushDerivedAction(NewDeathAction(a.targetPlayer, a.SourceID, a.targetPlayer.Position))
	}
	return nil
}

func (a *DamageAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	metadata.SetInt("hp_change", -a.Amount)
	metadata.SetString("blocked_by", a.BlockedBy)
	metadata.SetBool("piercing", a.IsPiercing)

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       constants.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.targetPlayer.ID.UUID(),
		Source:     a.SourceID,
		Metadata:   metadata,
	}
}

// ========== HealAction ==========

// HealAction represents HP restoration.
// Can be intercepted to modify amount (e.g., healing reduction debuff).
type HealAction struct {
	targetPlayer *core.Player // Player receiving healing
	SourceID     string       // Source identifier (e.g., "Buff_甘霖", "Item_HealingPotion")
	Amount       int          // Heal amount (can be modified)
}

// NewHealAction creates a new HealAction.
func NewHealAction(target *core.Player, amount int, sourceID string) *HealAction {
	return &HealAction{
		targetPlayer: target,
		SourceID:     sourceID,
		Amount:       amount,
	}
}

func (a *HealAction) Type() constants.ActionType { return constants.ActionHeal }
func (a *HealAction) CanModify() bool            { return a.Amount > 0 }
func (a *HealAction) Source() string             { return a.SourceID }
func (a *HealAction) Target() string             { return a.targetPlayer.ID.UUID() }
func (a *HealAction) TargetPlayer() *core.Player { return a.targetPlayer }
func (a *HealAction) PreTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

// PostTriggerPhase returns PhaseAnyTime (no post-trigger for healing).
func (a *HealAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

func (a *HealAction) Execute(ctx *ActionContext) error {
	if a.Amount <= 0 {
		return nil
	}
	if a.targetPlayer == nil {
		return errors.NewActionExecutionError("heal", "", "target player is nil", nil)
	}
	if err := a.targetPlayer.Heal(a.Amount); err != nil {
		return errors.NewActionExecutionError("heal", a.targetPlayer.ID.UUID(), "failed to heal", err)
	}
	return nil
}

func (a *HealAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	metadata.SetInt("hp_change", a.Amount)

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       constants.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.targetPlayer.ID.UUID(),
		Source:     a.SourceID,
		Metadata:   metadata,
	}
}

// ========== ModifyLPAction ==========

// ModifyLPAction represents Luck Point modification.
// LP affects event pool weight distribution.
type ModifyLPAction struct {
	targetPlayer *core.Player // Player receiving LP modification
	SourceID     string       // Source identifier (e.g., "Buff_神眷", "Buff_诅咒")
	Amount       int          // LP amount (+1 or -1)
}

// NewModifyLPAction creates a new ModifyLPAction.
func NewModifyLPAction(target *core.Player, amount int, sourceID string) *ModifyLPAction {
	return &ModifyLPAction{
		targetPlayer: target,
		SourceID:     sourceID,
		Amount:       amount,
	}
}

func (a *ModifyLPAction) Type() constants.ActionType { return constants.ActionModifyLP }
func (a *ModifyLPAction) CanModify() bool            { return false } // LP changes cannot be intercepted
func (a *ModifyLPAction) Source() string             { return a.SourceID }
func (a *ModifyLPAction) Target() string             { return a.targetPlayer.ID.UUID() }
func (a *ModifyLPAction) TargetPlayer() *core.Player { return a.targetPlayer }
func (a *ModifyLPAction) PreTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

// PostTriggerPhase returns PhaseAnyTime (no post-trigger for LP).
func (a *ModifyLPAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

func (a *ModifyLPAction) Execute(ctx *ActionContext) error {
	if a.targetPlayer == nil {
		return errors.NewActionExecutionError("modify_lp", "", "target player is nil", nil)
	}
	if a.Amount > 0 {
		a.targetPlayer.ModifyLP(a.Amount)
	} else if a.Amount < 0 {
		a.targetPlayer.ModifyLP(a.Amount)
	}
	return nil
}

func (a *ModifyLPAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	metadata.SetInt("lp_change", a.Amount)

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       constants.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.targetPlayer.ID.UUID(),
		Source:     a.SourceID,
		Metadata:   metadata,
	}
}

// ========== MoveAction ==========

// MoveAction represents player movement on map.
// Pure movement: only sets player position. Path calculation is done by HSM layer.
// Path data is passed via ActionContext.Metadata (target_pos, path).
type MoveAction struct {
	targetPlayer *core.Player   // Player moving
	Steps        int            // Movement steps (may be negative for reverse/迷途)
	SourceID     string         // Source identifier (e.g., "DiceRoll", "DiceRollCheckpoint")
	// Internal fields populated during Execute() from ctx.Metadata (for LogEntry)
	targetPos int
	path      []int
}

// NewMoveAction creates a new MoveAction.
func NewMoveAction(target *core.Player, steps int, sourceID string) *MoveAction {
	return &MoveAction{
		targetPlayer: target,
		Steps:        steps,
		SourceID:     sourceID,
	}
}

func (a *MoveAction) Type() constants.ActionType { return constants.ActionMove }
func (a *MoveAction) CanModify() bool            { return a.Steps != 0 }
func (a *MoveAction) Source() string             { return a.SourceID }
func (a *MoveAction) Target() string             { return a.targetPlayer.ID.UUID() }
func (a *MoveAction) TargetPlayer() *core.Player { return a.targetPlayer }
func (a *MoveAction) PreTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

// PostTriggerPhase returns PhaseAnyTime (no post-trigger for movement).
func (a *MoveAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

func (a *MoveAction) Execute(ctx *ActionContext) error {
	if a.targetPlayer == nil {
		return errors.NewActionExecutionError("move", "", "target player is nil", nil)
	}

	// Read target_pos and path from ActionContext.Metadata (set by HSM TurnMovingState)
	a.targetPos = ctx.GetIntOrDefault("target_pos", a.targetPlayer.Position+a.Steps)
	a.path = ctx.GetIntSliceOrDefault("path", nil)

	// Pure movement: just set player position
	a.targetPlayer.Position = a.targetPos
	return nil
}

func (a *MoveAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	startPos := 0
	if len(a.path) > 0 {
		startPos = a.path[0]
	}
	metadata.SetInt("start_pos", startPos)
	metadata.SetInt("end_pos", a.targetPlayer.Position)
	metadata.Set("path", a.path)
	metadata.SetInt("steps", a.Steps)

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       constants.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.targetPlayer.ID.UUID(),
		Source:     a.SourceID,
		Metadata:   metadata,
	}
}

// Overtook checks if this move overtook a specific player.
// Overtaken detection is done by HSM layer.
func (a *MoveAction) Overtook(player *core.Player) bool {
	// Overtaken data will be managed by HSM in future
	return false
}

// ========== AddBuffAction ==========

// AddBuffAction represents adding a Buff to player.
type AddBuffAction struct {
	targetPlayer *core.Player       // Player receiving Buff
	BuffType     constants.BuffType // Type of Buff to add
	SourceID     string             // Source identifier
	// Internal field populated during Execute() (for LogEntry)
	duration int
}

// NewAddBuffAction creates a new AddBuffAction.
func NewAddBuffAction(target *core.Player, buffType constants.BuffType, sourceID string) *AddBuffAction {
	return &AddBuffAction{
		targetPlayer: target,
		BuffType:     buffType,
		SourceID:     sourceID,
	}
}

func (a *AddBuffAction) Type() constants.ActionType { return constants.ActionAddBuff }
func (a *AddBuffAction) CanModify() bool            { return false } // Buff addition cannot be intercepted
func (a *AddBuffAction) Source() string             { return a.SourceID }
func (a *AddBuffAction) Target() string             { return a.targetPlayer.ID.UUID() }
func (a *AddBuffAction) TargetPlayer() *core.Player { return a.targetPlayer }
func (a *AddBuffAction) PreTriggerPhase() constants.Phase {
	return constants.PhasePreBuffApplied
}

// PostTriggerPhase returns PhasePostBuffApplied for entry effects/chain reactions.
func (a *AddBuffAction) PostTriggerPhase() constants.Phase {
	return constants.PhasePostBuffApplied
}

func (a *AddBuffAction) Execute(ctx *ActionContext) error {
	if a.targetPlayer == nil {
		return errors.NewActionExecutionError("add_buff", "", "target player is nil", nil)
	}
	if ctx.OnAddBuff == nil {
		return errors.NewActionExecutionError("add_buff", "", "OnAddBuff callback is nil", nil)
	}
	if ctx.GetBuffDuration == nil {
		return errors.NewActionExecutionError("add_buff", "", "GetBuffDuration callback is nil", nil)
	}

	duration := ctx.GetBuffDuration(a.BuffType)
	a.duration = duration

	// Check if player already has this buff type (duration extend)
	if a.targetPlayer.HasBuff(a.BuffType) {
		// Duration extend: Player.AddBuff handles duration merge internally
		// OnAddBuff should NOT be called (buff already subscribed on EventBus)
		// PhasePostBuffApplied should NOT be published (not a new buff application)
		newBuff := core.NewBuff(a.BuffType, duration)
		if err := a.targetPlayer.AddBuff(newBuff); err != nil {
			return errors.NewActionExecutionError("add_buff", a.targetPlayer.ID.UUID(), "failed to extend buff duration", err)
		}
		// Mark as duration-extend so ExecuteAction skips PostTrigger
		ctx.SetBool("buff_duration_extended", true)
		return nil
	}

	// New buff: full lifecycle (add + subscribe + PhasePostBuffApplied)
	newBuff := core.NewBuff(a.BuffType, duration)
	ctx.OnAddBuff(a.targetPlayer, newBuff)
	return nil
}

func (a *AddBuffAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	metadata.SetString("buff_type", string(a.BuffType))
	metadata.SetInt("duration", a.duration)

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       constants.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.targetPlayer.ID.UUID(),
		Source:     a.SourceID,
		Metadata:   metadata,
	}
}

// ========== RemoveBuffAction ==========

// RemoveBuffAction represents removing a Buff from player.
type RemoveBuffAction struct {
	targetPlayer *core.Player       // Player losing Buff
	BuffType     constants.BuffType // Type of Buff to remove
	SourceID     string             // Source identifier
}

// NewRemoveBuffAction creates a new RemoveBuffAction.
func NewRemoveBuffAction(target *core.Player, buffType constants.BuffType, sourceID string) *RemoveBuffAction {
	return &RemoveBuffAction{
		targetPlayer: target,
		BuffType:     buffType,
		SourceID:     sourceID,
	}
}

func (a *RemoveBuffAction) Type() constants.ActionType { return constants.ActionRemoveBuff }
func (a *RemoveBuffAction) CanModify() bool            { return false }
func (a *RemoveBuffAction) Source() string             { return a.SourceID }
func (a *RemoveBuffAction) Target() string             { return a.targetPlayer.ID.UUID() }
func (a *RemoveBuffAction) TargetPlayer() *core.Player { return a.targetPlayer }
func (a *RemoveBuffAction) PreTriggerPhase() constants.Phase {
	return constants.PhasePreBuffRemoved
}

// PostTriggerPhase returns PhasePostBuffRemoved.
func (a *RemoveBuffAction) PostTriggerPhase() constants.Phase {
	return constants.PhasePostBuffRemoved
}

func (a *RemoveBuffAction) Execute(ctx *ActionContext) error {
	if a.targetPlayer == nil {
		return errors.NewActionExecutionError("remove_buff", "", "target player is nil", nil)
	}
	if ctx.OnRemoveBuff == nil {
		return errors.NewActionExecutionError("remove_buff", "", "OnRemoveBuff callback is nil", nil)
	}
	// OnRemoveBuff handles EventBus unsubscription and player.RemoveBuff
	ctx.OnRemoveBuff(a.targetPlayer, a.BuffType)
	return nil
}

func (a *RemoveBuffAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	metadata.SetString("buff_type", string(a.BuffType))

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       constants.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.targetPlayer.ID.UUID(),
		Source:     a.SourceID,
		Metadata:   metadata,
	}
}

// ========== TeleportAction ==========

// TeleportAction represents instant teleport to specific position.
// Used by items like 任意门.
type TeleportAction struct {
	targetPlayer *core.Player // Player teleporting
	TargetPos    int          // Destination position
	SourcePos    int          // Original position (captured before Execute)
	SourceID     string       // Source identifier (e.g., "Item_AnyDoor")
}

// NewTeleportAction creates a new TeleportAction.
func NewTeleportAction(target *core.Player, targetPos int, sourceID string) *TeleportAction {
	return &TeleportAction{
		targetPlayer: target,
		TargetPos:    targetPos,
		SourcePos:    target.Position,
		SourceID:     sourceID,
	}
}

func (a *TeleportAction) Type() constants.ActionType { return constants.ActionTeleport }
func (a *TeleportAction) CanModify() bool            { return false }
func (a *TeleportAction) Source() string             { return a.SourceID }
func (a *TeleportAction) Target() string             { return a.targetPlayer.ID.UUID() }
func (a *TeleportAction) TargetPlayer() *core.Player { return a.targetPlayer }
func (a *TeleportAction) PreTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

// PostTriggerPhase returns PhaseAnyTime (no post-trigger for teleport).
func (a *TeleportAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

func (a *TeleportAction) Execute(ctx *ActionContext) error {
	if a.targetPlayer == nil {
		return errors.NewActionExecutionError("teleport", "", "target player is nil", nil)
	}
	a.targetPlayer.Position = a.TargetPos
	return nil
}

func (a *TeleportAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	metadata.SetInt("from_pos", a.SourcePos)
	metadata.SetInt("to_pos", a.TargetPos)

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       constants.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.targetPlayer.ID.UUID(),
		Source:     a.SourceID,
		Metadata:   metadata,
	}
}

// ========== StealBuffAction ==========

// StealBuffAction represents stealing a Buff from another player.
// Used by 白虎"劫运" faction passive.
type StealBuffAction struct {
	targetPlayer *core.Player // Player being stolen from
	SourcePlayer *core.Player // Player stealing (owner of 白虎 faction)
	SourceID     string       // Source identifier (e.g., "faction_bai_hu")
	StolenBuff   *core.Buff   // Buff that was stolen (set after execution)
}

// NewStealBuffAction creates a new StealBuffAction.
func NewStealBuffAction(target, source *core.Player, sourceID string) *StealBuffAction {
	return &StealBuffAction{
		targetPlayer: target,
		SourcePlayer: source,
		SourceID:     sourceID,
	}
}

func (a *StealBuffAction) Type() constants.ActionType { return constants.ActionStealBuff }
func (a *StealBuffAction) CanModify() bool            { return false }
func (a *StealBuffAction) Source() string             { return a.SourceID }
func (a *StealBuffAction) Target() string             { return a.targetPlayer.ID.UUID() }
func (a *StealBuffAction) TargetPlayer() *core.Player { return a.targetPlayer }
func (a *StealBuffAction) PreTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

// PostTriggerPhase returns PhaseAnyTime (no post-trigger for steal).
func (a *StealBuffAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

func (a *StealBuffAction) Execute(ctx *ActionContext) error {
	// Steal a random buff from target
	if a.targetPlayer == nil {
		return errors.NewActionExecutionError("steal_buff", "", "target player is nil", nil)
	}
	if a.SourcePlayer == nil {
		return errors.NewActionExecutionError("steal_buff", "", "source player is nil", nil)
	}
	if len(a.targetPlayer.ActiveBuffs) == 0 {
		return nil // No buffs to steal
	}

	// Take first buff (in real implementation, would be random selection)
	stolen := a.targetPlayer.ActiveBuffs[0]
	a.targetPlayer.RemoveBuff(stolen.Type)
	if err := a.SourcePlayer.AddBuff(stolen); err != nil {
		return errors.NewActionExecutionError("steal_buff", a.SourcePlayer.ID.UUID(), "failed to add stolen buff to source player", err)
	}
	a.StolenBuff = stolen

	return nil
}

func (a *StealBuffAction) LogEntry() gamelog.LogEntry {
	buffType := ""
	if a.StolenBuff != nil {
		buffType = string(a.StolenBuff.Type)
	}

	metadata := util.NewMetadata()
	metadata.SetString("stolen_by", a.SourcePlayer.ID.UUID())
	metadata.SetString("buff_type", buffType)

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       constants.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.targetPlayer.ID.UUID(),
		Source:     a.SourceID,
		Metadata:   metadata,
	}
}

// ========== DrawEventAction ==========

// DrawEventAction represents drawing a random event.
// Can be intercepted by 辟邪/玄武 to block bad events.
type DrawEventAction struct {
	targetPlayer *core.Player        // Player drawing event
	SourceID     string              // Source identifier
	DrawnType    constants.EventType // Event type drawn (set after Execute)
}

// NewDrawEventAction creates a new DrawEventAction.
func NewDrawEventAction(target *core.Player, sourceID string) *DrawEventAction {
	return &DrawEventAction{
		targetPlayer: target,
		SourceID:     sourceID,
	}
}

func (a *DrawEventAction) Type() constants.ActionType { return constants.ActionDrawEvent }
func (a *DrawEventAction) CanModify() bool            { return true }
func (a *DrawEventAction) Source() string             { return a.SourceID }
func (a *DrawEventAction) Target() string             { return a.targetPlayer.ID.UUID() }
func (a *DrawEventAction) TargetPlayer() *core.Player { return a.targetPlayer }
func (a *DrawEventAction) PreTriggerPhase() constants.Phase {
	return constants.PhasePreEvent
}

// PostTriggerPhase returns PhaseAnyTime (no post-trigger for draw).
func (a *DrawEventAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

func (a *DrawEventAction) Execute(ctx *ActionContext) error {
	if a.targetPlayer == nil {
		return errors.NewActionExecutionError("draw_event", "", "target player is nil", nil)
	}

	// If DrawnType is already set (e.g. bound event from CellTypeEvent), skip pool draw
	if a.DrawnType != constants.EventTypeNone && a.DrawnType.IsValid() {
		return nil
	}

	// Draw event from pool requires DrawEngine and EventPool
	if ctx.DrawEngine == nil {
		return errors.NewInternalError("DrawEventAction", "Execute", nil).
			WithContext("reason", "draw engine is nil")
	}
	if ctx.EventPool == nil {
		return errors.NewInternalError("DrawEventAction", "Execute", nil).
			WithContext("reason", "event pool is nil")
	}

	// Draw event type from pool using probability weights and player's LP
	result := ctx.DrawEngine.DrawWithProb(
		ctx.EventPool,
		ctx.ProbGood, ctx.ProbNeutral, ctx.ProbBad,
		a.targetPlayer.LP,
	)
	if result.Item != nil {
		a.DrawnType = constants.ParseEventType(result.Item.Type)
	} else {
		a.DrawnType = constants.EventTypeNone
	}

	return nil
}

func (a *DrawEventAction) LogEntry() gamelog.LogEntry {
	entry := gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       constants.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.targetPlayer.ID.UUID(),
		Source:     a.SourceID,
		Metadata:   util.NewMetadata(),
	}

	// Add event type to metadata (client uses event_type to look up definition from DefinitionsConfig)
	if a.DrawnType.IsValid() {
		entry.Metadata.SetString("event_type", string(a.DrawnType))
	}

	return entry
}

// ========== DrawItemAction ==========

// DrawItemAction represents drawing a random item (e.g. from CheckPoint treasure).
// No interception - auto draw, cannot be blocked.
type DrawItemAction struct {
	targetPlayer *core.Player          // Player drawing item
	SourceID     string                // Source identifier (e.g., "CheckpointTreasure")
	DrawnType    constants.ItemType    // Item type drawn (set after Execute)
}

// NewDrawItemAction creates a new DrawItemAction.
func NewDrawItemAction(target *core.Player, sourceID string) *DrawItemAction {
	return &DrawItemAction{
		targetPlayer: target,
		SourceID:     sourceID,
	}
}

func (a *DrawItemAction) Type() constants.ActionType { return constants.ActionDrawItem }
func (a *DrawItemAction) CanModify() bool            { return false }
func (a *DrawItemAction) Source() string             { return a.SourceID }
func (a *DrawItemAction) Target() string             { return a.targetPlayer.ID.UUID() }
func (a *DrawItemAction) TargetPlayer() *core.Player { return a.targetPlayer }
func (a *DrawItemAction) PreTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

// PostTriggerPhase returns PhaseAnyTime (no post-trigger for draw item).
func (a *DrawItemAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

func (a *DrawItemAction) Execute(ctx *ActionContext) error {
	// Draw item requires DrawEngine and pool data
	if ctx.DrawEngine == nil {
		return errors.NewInternalError("DrawItemAction", "Execute", nil).
			WithContext("reason", "draw engine is nil")
	}
	if ctx.ItemPool == nil {
		return errors.NewInternalError("DrawItemAction", "Execute", nil).
			WithContext("reason", "item pool is nil")
	}
	if a.targetPlayer == nil {
		return errors.NewActionExecutionError("draw_item", "", "target player is nil", nil)
	}

	// Draw item type from pool using probability weights and player's LP
	result := ctx.DrawEngine.DrawWithProb(
		ctx.ItemPool,
		ctx.ProbGood, ctx.ProbNeutral, ctx.ProbBad,
		a.targetPlayer.LP,
	)
	if result.Item != nil {
		a.DrawnType = constants.ParseItemType(result.Item.Type)
	} else {
		a.DrawnType = constants.ItemTypeNone
	}

	// Push AddItemAction as DerivedAction for full item lifecycle (EventBus subscription)
	if a.DrawnType != constants.ItemTypeNone && a.DrawnType.IsValid() {
		ctx.PushDerivedAction(NewAddItemAction(a.targetPlayer, a.DrawnType, a.SourceID))
	}

	return nil
}

func (a *DrawItemAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	metadata.SetString("item_type", string(a.DrawnType))

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       constants.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.targetPlayer.ID.UUID(),
		Source:     a.SourceID,
		Metadata:   metadata,
	}
}

// ========== RespawnAction ==========

// RespawnAction represents player respawn at checkpoint.
// Used when player dies and needs to respawn.
type RespawnAction struct {
	targetPlayer  *core.Player // Player respawning
	CheckpointPos int          // Position to respawn at
	SourceID      string       // Source identifier (e.g., "death_respawn", "system_turn_end_respawn")
}

// NewRespawnAction creates a new RespawnAction.
func NewRespawnAction(target *core.Player, checkpointPos int, sourceID string) *RespawnAction {
	return &RespawnAction{
		targetPlayer:  target,
		CheckpointPos: checkpointPos,
		SourceID:      sourceID,
	}
}

func (a *RespawnAction) Type() constants.ActionType { return constants.ActionRespawn }
func (a *RespawnAction) CanModify() bool            { return false }
func (a *RespawnAction) Source() string             { return a.SourceID }
func (a *RespawnAction) Target() string             { return a.targetPlayer.ID.UUID() }
func (a *RespawnAction) TargetPlayer() *core.Player { return a.targetPlayer }
func (a *RespawnAction) PreTriggerPhase() constants.Phase {
	return constants.PhasePreRespawn
}

// PostTriggerPhase returns PhaseAnyTime (no post-trigger for respawn).
func (a *RespawnAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

func (a *RespawnAction) Execute(ctx *ActionContext) error {
	if a.targetPlayer == nil {
		return errors.NewActionExecutionError("respawn", "", "target player is nil", nil)
	}
	if err := a.targetPlayer.Respawn(a.CheckpointPos); err != nil {
		return errors.NewActionExecutionError("respawn", a.targetPlayer.ID.UUID(), "failed to respawn", err)
	}
	return nil
}

func (a *RespawnAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	metadata.SetInt("checkpoint_pos", a.CheckpointPos)

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       constants.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.targetPlayer.ID.UUID(),
		Source:     a.SourceID,
		Metadata:   metadata,
	}
}

// ========== FellDownAction ==========

// FellDownAction represents player falling from Fragile cell.
// Used when player lands on a broken fragile cell.
type FellDownAction struct {
	targetPlayer *core.Player // Player falling
	Position     int          // Position where player fell
	Damage       int          // Damage amount from falling
	SourceID     string       // Source identifier (e.g., "FragileCell")
}

// NewFellDownAction creates a new FellDownAction.
func NewFellDownAction(target *core.Player, position int, damage int, sourceID string) *FellDownAction {
	return &FellDownAction{
		targetPlayer: target,
		Position:     position,
		Damage:       damage,
		SourceID:     sourceID,
	}
}

func (a *FellDownAction) Type() constants.ActionType { return constants.ActionFellDown }
func (a *FellDownAction) CanModify() bool            { return false }
func (a *FellDownAction) Source() string             { return a.SourceID }
func (a *FellDownAction) Target() string             { return a.targetPlayer.ID.UUID() }
func (a *FellDownAction) TargetPlayer() *core.Player { return a.targetPlayer }
func (a *FellDownAction) PreTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

// PostTriggerPhase returns PhaseAnyTime (no post-trigger for fell down).
func (a *FellDownAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

func (a *FellDownAction) Execute(ctx *ActionContext) error {
	if a.targetPlayer == nil {
		return errors.NewActionExecutionError("fell_down", "", "target player is nil", nil)
	}
	// Derive PiercingDamageAction for actual HP deduction.
	// FellDownAction itself is a semantic signal for client animation;
	// the damage effect is delegated to PiercingDamageAction (unblockable).
	if a.Damage > 0 {
		ctx.PushDerivedAction(NewPiercingDamageAction(a.targetPlayer, a.Damage, a.SourceID))
	}
	return nil
}

func (a *FellDownAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	metadata.SetInt("position", a.Position)

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       constants.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.targetPlayer.ID.UUID(),
		Source:     a.SourceID,
		Metadata:   metadata,
	}
}

// ========== DeathAction ==========

// DeathAction represents a player death event for client animation.
// Pure rendering signal - does NOT modify IsDead (that's done by DamageAction).
// SourceID identifies what caused the death (e.g., "Buff_Corrupt", "FragileCell", "Event_Trap").
type DeathAction struct {
	targetPlayer *core.Player // Player who died
	SourceID     string       // What caused the death (for client rendering)
	Position     int          // Where the death occurred (for client animation)
}

// NewDeathAction creates a new DeathAction.
func NewDeathAction(target *core.Player, source string, position int) *DeathAction {
	return &DeathAction{
		targetPlayer: target,
		SourceID:     source,
		Position:     position,
	}
}

func (a *DeathAction) Type() constants.ActionType { return constants.ActionDeath }
func (a *DeathAction) CanModify() bool            { return false }
func (a *DeathAction) Source() string             { return a.SourceID }
func (a *DeathAction) Target() string             { return a.targetPlayer.ID.UUID() }
func (a *DeathAction) TargetPlayer() *core.Player { return a.targetPlayer }
func (a *DeathAction) PreTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

// PostTriggerPhase returns PhaseAnyTime (no post-trigger for death).
func (a *DeathAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

func (a *DeathAction) Execute(ctx *ActionContext) error {
	// Add DeathMark buff via OnAddBuff callback (like AddBuffAction)
	// DeathMark blocks all subsequent Actions via PhasePreAction on EventBus.
	if a.targetPlayer == nil {
		return errors.NewActionExecutionError("death", "", "target player is nil", nil)
	}
	if ctx.OnAddBuff == nil {
		return errors.NewActionExecutionError("death", a.targetPlayer.ID.UUID(), "OnAddBuff callback is nil", nil)
	}
	if ctx.GetBuffDuration == nil {
		return errors.NewActionExecutionError("death", a.targetPlayer.ID.UUID(), "GetBuffDuration callback is nil", nil)
	}
	duration := ctx.GetBuffDuration(constants.BuffTypeDeathMark)
	deathMark := core.NewBuff(constants.BuffTypeDeathMark, duration)
	ctx.OnAddBuff(a.targetPlayer, deathMark)
	return nil
}

func (a *DeathAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	metadata.SetInt("position", a.Position)
	metadata.SetString("death_source", a.SourceID)

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       constants.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.targetPlayer.ID.UUID(),
		Source:     a.SourceID,
		Metadata:   metadata,
	}
}

// ========== BossDamageAction ==========

// BossDamageAction represents a player attacking the Boss (damage to Boss player).
// Used in TurnBossBattleState when a player on the Boss cell rolls dice.
type BossDamageAction struct {
	SourcePlayer  *core.Player // Player attacking the boss
	targetPlayer  *core.Player // Boss player receiving damage
	Damage        int          // Damage amount (dice steps, x2 if crit)
	IsCrit        bool         // Whether this is a critical hit
	SourceID      string       // Source identifier (e.g., "boss_damage")
}

// NewBossDamageAction creates a new BossDamageAction.
func NewBossDamageAction(source *core.Player, boss *core.Player, damage int, isCrit bool, sourceID string) *BossDamageAction {
	return &BossDamageAction{
		SourcePlayer: source,
		targetPlayer: boss,
		Damage:       damage,
		IsCrit:       isCrit,
		SourceID:     sourceID,
	}
}

func (a *BossDamageAction) Type() constants.ActionType { return constants.ActionBossDamage }
func (a *BossDamageAction) CanModify() bool            { return false }
func (a *BossDamageAction) Source() string             { return a.SourceID }
func (a *BossDamageAction) Target() string             { return a.targetPlayer.ID.UUID() }
func (a *BossDamageAction) TargetPlayer() *core.Player { return a.targetPlayer }

// ActorPlayers returns players that should receive PhasePreAction publication.
// BossDamageAction needs both Boss (target for DeathMark/Thorns interception)
// and SourcePlayer (attacker for Dominance amplification).
func (a *BossDamageAction) ActorPlayers() []*core.Player {
	players := []*core.Player{}
	if a.targetPlayer != nil {
		players = append(players, a.targetPlayer)
	}
	if a.SourcePlayer != nil && a.SourcePlayer.ID.UUID() != a.targetPlayer.ID.UUID() {
		players = append(players, a.SourcePlayer)
	}
	return players
}

func (a *BossDamageAction) PreTriggerPhase() constants.Phase {
	return constants.PhasePreDamage // Thorns handler intercepts at PreDamage on BossPlayer
}

// PostTriggerPhase returns PhaseAnyTime (no post-trigger for boss damage).
func (a *BossDamageAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

func (a *BossDamageAction) Execute(ctx *ActionContext) error {
	if a.targetPlayer == nil {
		return errors.NewActionExecutionError("boss_damage", "", "boss player is nil", nil)
	}
	if a.Damage <= 0 {
		return nil
	}

	// Apply damage to boss player
	if err := a.targetPlayer.ApplyDamage(a.Damage); err != nil {
		return errors.NewActionExecutionError("boss_damage", a.targetPlayer.ID.UUID(), "failed to apply damage to boss", err)
	}

	// Thorns reflect is handled by BuffThorns handler via EventBus (PhasePreDamage).
	// The PreTrigger publishes PhasePreDamage to BossPlayer, and Thorns handler
	// pushes a derived PiercingDamageAction for reflect damage to the attacking player.

	return nil
}

func (a *BossDamageAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	metadata.SetInt("damage", a.Damage)
	metadata.SetBool("is_crit", a.IsCrit)
	metadata.SetInt("boss_remaining_hp", a.targetPlayer.HP)

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       constants.EntryTypeBoss,
		ActionType: string(a.Type()),
		Target:     a.targetPlayer.ID.UUID(),
		Source:     a.SourcePlayer.ID.UUID(),
		Metadata:   metadata,
	}
}

// ========== BossAttackAction ==========

// BossAttackAction represents the Boss physically attacking a player (normal or critical).
// Only used for Boss normal/crit attacks in TurnBossBattleState.
// Skill effects (Thunder, etc.) use BossSkillAction + derived DamageAction instead.
type BossAttackAction struct {
	SourcePlayer *core.Player         // Boss player (attacker)
	targetPlayer *core.Player         // Player receiving damage
	Damage       int                  // Damage amount (1 for normal, 2 for crit)
	AttackType   constants.BossAttackType // Attack type (normal/crit)
	SourceID     string               // Source identifier
}

// NewBossAttackAction creates a new BossAttackAction.
func NewBossAttackAction(boss *core.Player, target *core.Player, damage int, attackType constants.BossAttackType, sourceID string) *BossAttackAction {
	return &BossAttackAction{
		SourcePlayer: boss,
		targetPlayer: target,
		Damage:       damage,
		AttackType:   attackType,
		SourceID:     sourceID,
	}
}

func (a *BossAttackAction) Type() constants.ActionType { return constants.ActionBossAttack }
func (a *BossAttackAction) CanModify() bool            { return false }
func (a *BossAttackAction) Source() string             { return a.SourceID }
func (a *BossAttackAction) Target() string             { return a.targetPlayer.ID.UUID() }
func (a *BossAttackAction) TargetPlayer() *core.Player { return a.targetPlayer }
func (a *BossAttackAction) PreTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime // No handler intercepts BossAttackAction at PhasePreDamage
}

// PostTriggerPhase returns PhaseAnyTime (no post-trigger for boss attack).
func (a *BossAttackAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

func (a *BossAttackAction) Execute(ctx *ActionContext) error {
	if a.targetPlayer == nil {
		return errors.NewActionExecutionError("boss_attack", "", "target player is nil", nil)
	}
	// Derive DamageAction for actual HP deduction.
	// BossAttackAction itself is a semantic signal for client animation;
	// the damage effect is delegated to DamageAction.
	if a.Damage > 0 {
		ctx.PushDerivedAction(NewDamageAction(a.targetPlayer, a.Damage, a.SourceID))
	}
	return nil
}

func (a *BossAttackAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	metadata.SetString("attack_type", string(a.AttackType))
	metadata.SetString("target", a.targetPlayer.ID.UUID())

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       constants.EntryTypeBoss,
		ActionType: string(a.Type()),
		Target:     a.targetPlayer.ID.UUID(),
		Source:     a.SourcePlayer.ID.UUID(),
		Metadata:   metadata,
	}
}

// ========== BossSkillAction ==========

// BossSkillAction represents the Boss using a skill.
// The actual skill effect is handled by BossRegistry skill handlers.
type BossSkillAction struct {
	SourcePlayer *core.Player         // Boss player
	SkillType    constants.BossSkillType // Skill type
	TargetIDs    []string             // Target player IDs
	SourceID     string               // Source identifier
	Targets      []*core.Player       // Target players (for handler execution)
}

// NewBossSkillAction creates a new BossSkillAction.
func NewBossSkillAction(boss *core.Player, skillType constants.BossSkillType, targets []*core.Player, sourceID string) *BossSkillAction {
	targetIDs := make([]string, len(targets))
	for i, t := range targets {
		targetIDs[i] = t.ID.UUID()
	}
	return &BossSkillAction{
		SourcePlayer: boss,
		SkillType:    skillType,
		TargetIDs:    targetIDs,
		SourceID:     sourceID,
		Targets:      targets,
	}
}

func (a *BossSkillAction) Type() constants.ActionType { return constants.ActionBossSkill }
func (a *BossSkillAction) CanModify() bool            { return false }
func (a *BossSkillAction) Source() string             { return a.SourceID }
func (a *BossSkillAction) Target() string {
	if len(a.TargetIDs) > 0 {
		return a.TargetIDs[0] // Primary target
	}
	return ""
}
func (a *BossSkillAction) TargetPlayer() *core.Player { return a.SourcePlayer } // Boss is the actor

// PreTriggerPhase returns PhaseAnyTime (Boss skills cannot be intercepted).
func (a *BossSkillAction) PreTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

// PostTriggerPhase returns PhaseAnyTime (no post-trigger for boss skill).
func (a *BossSkillAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

func (a *BossSkillAction) Execute(ctx *ActionContext) error {
	// Boss skill execution is delegated to BossRegistry handlers.
	// The handler is called by TurnBossBattleState, not here.
	// This action is used as a LogEntry record only.
	return nil
}

func (a *BossSkillAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	metadata.SetString("skill_type", string(a.SkillType))
	// Store target IDs as comma-separated string
	targetStrs := make([]string, len(a.TargetIDs))
	copy(targetStrs, a.TargetIDs)
	metadata.SetString("targets", strings.Join(targetStrs, ","))

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       constants.EntryTypeBoss,
		ActionType: string(a.Type()),
		Target:     a.Target(),
		Source:     a.SourcePlayer.ID.UUID(),
		Metadata:   metadata,
	}
}

// ========== RollDiceAction ==========

// RollDiceAction represents dice roll result for client animation.
// Steps is calculated at construction time using RNG (like DamageAction.Amount).
// Interceptable via PhasePreDiceRoll — Buffs can modify Steps field.
// Execute writes final (possibly modified) Steps to ActionContext metadata
// for HSM to read back after ExecuteAction completes.
type RollDiceAction struct {
	targetPlayer *core.Player
	DiceType     rng.DiceType // Dice type (gold/silver/copper/wood/normal)
	Steps        int          // Dice result (calculated at construction, may be modified by interception)
	SourceID     string       // Source identifier (e.g., "DiceRoll", "DiceRollTimeout")
}

// NewRollDiceAction creates a new RollDiceAction with dice roll calculated at construction.
// Uses the provided RNG source to calculate Steps via rng.NewDice(diceType, rngInst).Roll().
func NewRollDiceAction(target *core.Player, diceType rng.DiceType, rngInst *rand.Rand, sourceID string) *RollDiceAction {
	var steps int
	if rngInst != nil {
		dice := rng.NewDice(diceType, rngInst)
		steps = dice.Roll()
	} else {
		steps = 1 // Fallback for nil RNG
	}

	return &RollDiceAction{
		targetPlayer: target,
		DiceType:     diceType,
		Steps:        steps,
		SourceID:     sourceID,
	}
}

func (a *RollDiceAction) Type() constants.ActionType { return constants.ActionDiceRoll }
func (a *RollDiceAction) CanModify() bool            { return true } // Buffs can modify Steps via PhasePreDiceRoll
func (a *RollDiceAction) Source() string             { return a.SourceID }
func (a *RollDiceAction) Target() string             { return a.targetPlayer.ID.UUID() }
func (a *RollDiceAction) TargetPlayer() *core.Player { return a.targetPlayer }

// PreTriggerPhase returns PhasePreDiceRoll for interception (Buffs can modify Steps).
func (a *RollDiceAction) PreTriggerPhase() constants.Phase {
	return constants.PhasePreDiceRoll
}

// PostTriggerPhase returns PhaseAnyTime (no post-trigger lifecycle for dice roll).
func (a *RollDiceAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

func (a *RollDiceAction) Execute(ctx *ActionContext) error {
	// Write final Steps (possibly modified by PhasePreDiceRoll interception) to metadata
	// for HSM to read after ExecuteAction completes.
	ctx.SetInt("dice_steps_result", a.Steps)
	ctx.SetString("dice_type_result", a.DiceType.String())
	return nil
}

func (a *RollDiceAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	metadata.SetString("dice_type", a.DiceType.String())
	metadata.SetInt("dice_steps", a.Steps)

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       constants.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.targetPlayer.ID.UUID(),
		Source:     a.SourceID,
		Metadata:   metadata,
	}
}

// ========== AddItemAction ==========

// AddItemAction represents adding an Item to player inventory.
// Similar lifecycle to AddBuffAction: OnAddItem callback handles EventBus subscription.
type AddItemAction struct {
	targetPlayer *core.Player
	ItemType     constants.ItemType // Item type to add
	SourceID     string             // Source identifier (e.g., "CheckpointTreasure", "Event_Relic")
}

// NewAddItemAction creates a new AddItemAction.
func NewAddItemAction(target *core.Player, itemType constants.ItemType, sourceID string) *AddItemAction {
	return &AddItemAction{
		targetPlayer: target,
		ItemType:     itemType,
		SourceID:     sourceID,
	}
}

func (a *AddItemAction) Type() constants.ActionType { return constants.ActionAddItem }
func (a *AddItemAction) CanModify() bool            { return false }
func (a *AddItemAction) Source() string             { return a.SourceID }
func (a *AddItemAction) Target() string             { return a.targetPlayer.ID.UUID() }
func (a *AddItemAction) TargetPlayer() *core.Player { return a.targetPlayer }
func (a *AddItemAction) PreTriggerPhase() constants.Phase  { return constants.PhaseAnyTime }
func (a *AddItemAction) PostTriggerPhase() constants.Phase { return constants.PhaseAnyTime }

func (a *AddItemAction) Execute(ctx *ActionContext) error {
	if a.targetPlayer == nil {
		return errors.NewActionExecutionError("add_item", "", "target player is nil", nil)
	}
	if ctx.OnAddItem == nil {
		return errors.NewActionExecutionError("add_item", "", "OnAddItem callback is nil", nil)
	}

	newItem := core.NewItem(a.ItemType)
	ctx.OnAddItem(a.targetPlayer, newItem)
	return nil
}

func (a *AddItemAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	metadata.SetString("item_type", string(a.ItemType))

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       constants.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.targetPlayer.ID.UUID(),
		Source:     a.SourceID,
		Metadata:   metadata,
	}
}

// ========== RemoveItemAction ==========

// RemoveItemAction represents removing an Item from player inventory.
// Similar lifecycle to RemoveBuffAction: OnRemoveItem callback handles EventBus unsubscription + player.RemoveItem.
type RemoveItemAction struct {
	targetPlayer *core.Player
	ItemType     constants.ItemType // Item type to remove
	SourceID     string             // Source identifier (e.g., "Item_Consumed", "Event_Thief")
}

// NewRemoveItemAction creates a new RemoveItemAction.
func NewRemoveItemAction(target *core.Player, itemType constants.ItemType, sourceID string) *RemoveItemAction {
	return &RemoveItemAction{
		targetPlayer: target,
		ItemType:     itemType,
		SourceID:     sourceID,
	}
}

func (a *RemoveItemAction) Type() constants.ActionType { return constants.ActionRemoveItem }
func (a *RemoveItemAction) CanModify() bool            { return false }
func (a *RemoveItemAction) Source() string             { return a.SourceID }
func (a *RemoveItemAction) Target() string             { return a.targetPlayer.ID.UUID() }
func (a *RemoveItemAction) TargetPlayer() *core.Player { return a.targetPlayer }
func (a *RemoveItemAction) PreTriggerPhase() constants.Phase  { return constants.PhaseAnyTime }
func (a *RemoveItemAction) PostTriggerPhase() constants.Phase { return constants.PhaseAnyTime }

func (a *RemoveItemAction) Execute(ctx *ActionContext) error {
	if a.targetPlayer == nil {
		return errors.NewActionExecutionError("remove_item", "", "target player is nil", nil)
	}
	if ctx.OnRemoveItem == nil {
		return errors.NewActionExecutionError("remove_item", "", "OnRemoveItem callback is nil", nil)
	}
	// OnRemoveItem handles EventBus unsubscription and player.RemoveItem
	ctx.OnRemoveItem(a.targetPlayer, a.ItemType)
	return nil
}

func (a *RemoveItemAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	metadata.SetString("item_type", string(a.ItemType))

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       constants.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.targetPlayer.ID.UUID(),
		Source:     a.SourceID,
		Metadata:   metadata,
	}
}

// ========== DrawBuffAction ==========

// DrawBuffAction represents drawing a random Buff from BuffPool.
// Similar to DrawEventAction: PreTrigger can be intercepted by Hidden buff (PhasePreBuffApplied).
// Execute draws a BuffType from BuffPool, then pushes AddBuffAction as DerivedAction.
// The actual buff application lifecycle (OnAddBuff, PhasePostBuffApplied) is handled by AddBuffAction.
type DrawBuffAction struct {
	targetPlayer *core.Player
	SourceID     string               // Source identifier (e.g., "Event_TasteTest")
	DrawnType    constants.BuffType   // Buff type drawn (set after Execute)
}

// NewDrawBuffAction creates a new DrawBuffAction.
func NewDrawBuffAction(target *core.Player, sourceID string) *DrawBuffAction {
	return &DrawBuffAction{
		targetPlayer: target,
		SourceID:     sourceID,
	}
}

func (a *DrawBuffAction) Type() constants.ActionType { return constants.ActionDrawBuff }
func (a *DrawBuffAction) CanModify() bool            { return true }
func (a *DrawBuffAction) Source() string             { return a.SourceID }
func (a *DrawBuffAction) Target() string             { return a.targetPlayer.ID.UUID() }
func (a *DrawBuffAction) TargetPlayer() *core.Player { return a.targetPlayer }
func (a *DrawBuffAction) PreTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime // No interception needed; AddBuffAction handles PhasePreBuffApplied
}
func (a *DrawBuffAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime // Buff entry effects handled by AddBuffAction's PostTrigger
}

func (a *DrawBuffAction) Execute(ctx *ActionContext) error {
	if a.targetPlayer == nil {
		return errors.NewActionExecutionError("draw_buff", "", "target player is nil", nil)
	}

	// Draw buff requires DrawEngine and BuffPool
	if ctx.DrawEngine == nil {
		return errors.NewInternalError("DrawBuffAction", "Execute", nil).
			WithContext("reason", "draw engine is nil")
	}
	if ctx.BuffPool == nil {
		return errors.NewInternalError("DrawBuffAction", "Execute", nil).
			WithContext("reason", "buff pool is nil")
	}

	// Draw buff type from pool using probability weights and player's LP
	result := ctx.DrawEngine.DrawWithProb(
		ctx.BuffPool,
		ctx.ProbGood, ctx.ProbNeutral, ctx.ProbBad,
		a.targetPlayer.LP,
	)
	if result.Item != nil {
		a.DrawnType = constants.ParseBuffType(result.Item.Type)
	} else {
		a.DrawnType = constants.BuffTypeNone
	}

	// Push AddBuffAction as DerivedAction for full buff lifecycle
	if a.DrawnType.IsValid() && a.DrawnType != constants.BuffTypeNone {
		ctx.PushDerivedAction(NewAddBuffAction(a.targetPlayer, a.DrawnType, a.SourceID))
	}

	return nil
}

func (a *DrawBuffAction) LogEntry() gamelog.LogEntry {
	entry := gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       constants.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.targetPlayer.ID.UUID(),
		Source:     a.SourceID,
		Metadata:   util.NewMetadata(),
	}

	// Add buff type to metadata (client uses buff_type to look up definition from DefinitionsConfig)
	if a.DrawnType.IsValid() {
		entry.Metadata.SetString("buff_type", string(a.DrawnType))
	}

	return entry
}

// ========== DiceUpgradeAction ==========

// DiceUpgradeAction represents upgrading a player's dice type.
// Calculates upgrade target from FromDice and writes result to ActionContext metadata
// for HSM to read back and update DiceManager.
// Upgrade path: Wood → Copper → Silver → Gold (Gold cannot be upgraded further).
type DiceUpgradeAction struct {
	targetPlayer *core.Player
	SourceID     string         // Source identifier (e.g., "Item_DiceUpgrade")
	FromDice    rng.DiceType   // Original dice type
	ToDice      rng.DiceType   // Upgraded dice type (set during Execute)
}

// NewDiceUpgradeAction creates a new DiceUpgradeAction.
func NewDiceUpgradeAction(target *core.Player, sourceID string, fromDice rng.DiceType) *DiceUpgradeAction {
	return &DiceUpgradeAction{
		targetPlayer: target,
		SourceID:     sourceID,
		FromDice:     fromDice,
	}
}

func (a *DiceUpgradeAction) Type() constants.ActionType { return constants.ActionDiceUpgrade }
func (a *DiceUpgradeAction) CanModify() bool            { return false }
func (a *DiceUpgradeAction) Source() string             { return a.SourceID }
func (a *DiceUpgradeAction) Target() string             { return a.targetPlayer.ID.UUID() }
func (a *DiceUpgradeAction) TargetPlayer() *core.Player { return a.targetPlayer }
func (a *DiceUpgradeAction) PreTriggerPhase() constants.Phase  { return constants.PhaseAnyTime }
func (a *DiceUpgradeAction) PostTriggerPhase() constants.Phase { return constants.PhaseAnyTime }

func (a *DiceUpgradeAction) Execute(ctx *ActionContext) error {
	if a.targetPlayer == nil {
		return errors.NewActionExecutionError("dice_upgrade", "", "target player is nil", nil)
	}

	// Calculate upgrade: Wood→Copper→Silver→Gold
	// DiceType enum: Gold=1, Silver=2, Copper=3, Wood=4, Normal=5
	// Upgrade means going one level up (lower DiceType number = better dice)
	switch a.FromDice {
	case rng.DiceTypeWood:
		a.ToDice = rng.DiceTypeCopper
	case rng.DiceTypeCopper:
		a.ToDice = rng.DiceTypeSilver
	case rng.DiceTypeSilver:
		a.ToDice = rng.DiceTypeGold
	case rng.DiceTypeGold:
		// Gold cannot be upgraded further, stay Gold
		a.ToDice = rng.DiceTypeGold
	default:
		// Normal dice upgrades to Copper
		a.ToDice = rng.DiceTypeCopper
	}

	// Write upgrade result to ActionContext metadata for HSM DiceManager update
	ctx.SetString("dice_upgrade_to", a.ToDice.String())
	ctx.SetString("dice_upgrade_from", a.FromDice.String())
	ctx.SetString("dice_upgrade_player", a.targetPlayer.ID.UUID())

	return nil
}

func (a *DiceUpgradeAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	metadata.SetString("from_dice", a.FromDice.String())
	metadata.SetString("to_dice", a.ToDice.String())

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       constants.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.targetPlayer.ID.UUID(),
		Source:     a.SourceID,
		Metadata:   metadata,
	}
}
