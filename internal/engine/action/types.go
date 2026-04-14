package action

import (
	"time"

	"github.com/b1tAction/Fated/internal/core"
	"github.com/b1tAction/Fated/internal/core/buff"
	"github.com/b1tAction/Fated/pkg/constants"
	"github.com/b1tAction/Fated/pkg/gamelog"
	"github.com/b1tAction/Fated/pkg/util"
)

// ========== DamageAction ==========

// DamageAction represents HP reduction.
// Can be intercepted by shields/隐匿 to reduce or block damage.
type DamageAction struct {
	TargetPlayer *core.Player // Player receiving damage
	SourceID     string       // Source identifier (e.g., "Buff_Curse", "Event_Trap")
	Amount       int          // Damage amount (can be modified by interceptors)
	IsPiercing   bool         // True if ignores shields (cannot be intercepted)
	BlockedBy    string       // Set by interceptor to identify blocking source
}

// NewDamageAction creates a new DamageAction.
func NewDamageAction(target *core.Player, amount int, sourceID string) *DamageAction {
	return &DamageAction{
		TargetPlayer: target,
		SourceID:     sourceID,
		Amount:       amount,
		IsPiercing:   false,
	}
}

// NewPiercingDamageAction creates a piercing DamageAction that cannot be intercepted.
func NewPiercingDamageAction(target *core.Player, amount int, sourceID string) *DamageAction {
	return &DamageAction{
		TargetPlayer: target,
		SourceID:     sourceID,
		Amount:       amount,
		IsPiercing:   true,
	}
}

func (a *DamageAction) Type() ActionType { return ActionDamage }
func (a *DamageAction) CanModify() bool   { return !a.IsPiercing && a.Amount > 0 }
func (a *DamageAction) Source() string    { return a.SourceID }
func (a *DamageAction) Target() string    { return a.TargetPlayer.UserID }

// PreTriggerPhase returns PhasePreDamage for interception by shields/隐匿.
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
		return nil // Already blocked
	}
	return a.TargetPlayer.ApplyDamage(a.Amount)
}

func (a *DamageAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	metadata.SetString("blocked_by", a.BlockedBy)
	metadata.SetBool("piercing", a.IsPiercing)

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       gamelog.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.TargetPlayer.UserID,
		Delta:      -a.Amount,
		Source:     a.SourceID,
		Metadata:   metadata,
	}
}

// ========== HealAction ==========

// HealAction represents HP restoration.
// Can be intercepted to modify amount (e.g., healing reduction debuff).
type HealAction struct {
	TargetPlayer *core.Player // Player receiving healing
	SourceID     string       // Source identifier (e.g., "Buff_甘霖", "Item_HealingPotion")
	Amount       int          // Heal amount (can be modified)
}

// NewHealAction creates a new HealAction.
func NewHealAction(target *core.Player, amount int, sourceID string) *HealAction {
	return &HealAction{
		TargetPlayer: target,
		SourceID:     sourceID,
		Amount:       amount,
	}
}

func (a *HealAction) Type() ActionType { return ActionHeal }
func (a *HealAction) CanModify() bool   { return a.Amount > 0 }
func (a *HealAction) Source() string    { return a.SourceID }
func (a *HealAction) Target() string    { return a.TargetPlayer.UserID }

// PreTriggerPhase returns PhaseAnyTime (healing typically not intercepted).
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
	a.TargetPlayer.Heal(a.Amount)
	return nil
}

func (a *HealAction) LogEntry() gamelog.LogEntry {
	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       gamelog.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.TargetPlayer.UserID,
		Delta:      a.Amount,
		Source:     a.SourceID,
	}
}

// ========== ModifyLPAction ==========

// ModifyLPAction represents Luck Point modification.
// LP affects event pool weight distribution.
type ModifyLPAction struct {
	TargetPlayer *core.Player // Player receiving LP modification
	SourceID     string       // Source identifier (e.g., "Buff_神眷", "Buff_诅咒")
	Amount       int          // LP amount (+1 or -1)
}

// NewModifyLPAction creates a new ModifyLPAction.
func NewModifyLPAction(target *core.Player, amount int, sourceID string) *ModifyLPAction {
	return &ModifyLPAction{
		TargetPlayer: target,
		SourceID:     sourceID,
		Amount:       amount,
	}
}

func (a *ModifyLPAction) Type() ActionType { return ActionModifyLP }
func (a *ModifyLPAction) CanModify() bool   { return false } // LP changes cannot be intercepted
func (a *ModifyLPAction) Source() string    { return a.SourceID }
func (a *ModifyLPAction) Target() string    { return a.TargetPlayer.UserID }

// PreTriggerPhase returns PhaseAnyTime (LP changes cannot be intercepted).
func (a *ModifyLPAction) PreTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

// PostTriggerPhase returns PhaseAnyTime (no post-trigger for LP).
func (a *ModifyLPAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

func (a *ModifyLPAction) Execute(ctx *ActionContext) error {
	if a.Amount > 0 {
		a.TargetPlayer.ModifyLP(a.Amount)
	} else if a.Amount < 0 {
		a.TargetPlayer.ModifyLP(a.Amount)
	}
	return nil
}

func (a *ModifyLPAction) LogEntry() gamelog.LogEntry {
	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       gamelog.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.TargetPlayer.UserID,
		Delta:      a.Amount,
		Source:     a.SourceID,
	}
}

// ========== MoveAction ==========

// MoveAction represents player movement on map.
// Can be intercepted by 迷途 Buff to reverse direction.
type MoveAction struct {
	TargetPlayer *core.Player // Player moving
	Steps        int          // Movement steps (can be negative for reverse)
	SourceID     string       // Source identifier (usually "DiceRoll")
	TargetPos    int          // Final position after movement (set by path calculation)
	Path         []int        // Calculated path (cells visited during movement)
	Overtaken    []*core.Player // Players overtaken during movement (for 白虎劫运)
}

// NewMoveAction creates a new MoveAction.
func NewMoveAction(target *core.Player, steps int, sourceID string) *MoveAction {
	return &MoveAction{
		TargetPlayer: target,
		Steps:        steps,
		SourceID:     sourceID,
		Overtaken:    make([]*core.Player, 0),
	}
}

func (a *MoveAction) Type() ActionType { return ActionMove }
func (a *MoveAction) CanModify() bool   { return a.Steps != 0 }
func (a *MoveAction) Source() string    { return a.SourceID }
func (a *MoveAction) Target() string    { return a.TargetPlayer.UserID }

// PreTriggerPhase returns PhasePreMove for interception by 迷途.
func (a *MoveAction) PreTriggerPhase() constants.Phase {
	return constants.PhasePreMove
}

// PostTriggerPhase returns PhaseAnyTime (no post-trigger for movement).
func (a *MoveAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

func (a *MoveAction) Execute(ctx *ActionContext) error {
	// Movement execution requires MapEngine
	if ctx.MapEngine == nil {
		return nil
	}

	// Calculate path using MapEngine
	startPos := a.TargetPlayer.Position
	result, err := ctx.MapEngine.CalculatePath(startPos, a.Steps)
	if err != nil {
		return err
	}

	// Update player position
	a.TargetPlayer.Position = result.GetTargetIndex()
	a.TargetPos = result.GetTargetIndex()
	a.Path = result.GetPath()

	// Note: Overtaken players detection would need additional logic
	// This could be implemented by checking which players were passed during movement
	// Currently leaving empty as PathResult doesn't track overtaken players
	a.Overtaken = make([]*core.Player, 0)

	return nil
}

func (a *MoveAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	metadata.SetInt("start_pos", a.TargetPlayer.Position-a.Steps) // Approximate start
	metadata.SetInt("end_pos", a.TargetPos)
	metadata.Set("path", a.Path)

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       gamelog.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.TargetPlayer.UserID,
		Delta:      a.Steps,
		Source:     a.SourceID,
		Metadata:   metadata,
	}
}

// Overtook checks if this move overtook a specific player.
func (a *MoveAction) Overtook(player *core.Player) bool {
	for _, p := range a.Overtaken {
		if p.UserID == player.UserID {
			return true
		}
	}
	return false
}

// ========== AddBuffAction ==========

// AddBuffAction represents adding a Buff to player.
type AddBuffAction struct {
	TargetPlayer *core.Player  // Player receiving Buff
	BuffType     constants.BuffType // Type of Buff to add
	Duration     int           // Buff duration in turns
	SourceID     string        // Source identifier
}

// NewAddBuffAction creates a new AddBuffAction.
func NewAddBuffAction(target *core.Player, buffType constants.BuffType, duration int, sourceID string) *AddBuffAction {
	return &AddBuffAction{
		TargetPlayer: target,
		BuffType:     buffType,
		Duration:     duration,
		SourceID:     sourceID,
	}
}

func (a *AddBuffAction) Type() ActionType { return ActionAddBuff }
func (a *AddBuffAction) CanModify() bool   { return false } // Buff addition cannot be intercepted
func (a *AddBuffAction) Source() string    { return a.SourceID }
func (a *AddBuffAction) Target() string    { return a.TargetPlayer.UserID }

// PreTriggerPhase returns PhaseAnyTime (buff addition not intercepted).
func (a *AddBuffAction) PreTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

// PostTriggerPhase returns PhaseOnBuffApplied for entry effects/chain reactions.
func (a *AddBuffAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseOnBuffApplied
}

func (a *AddBuffAction) Execute(ctx *ActionContext) error {
	newBuff := buff.NewBuff(a.BuffType, a.Duration)
	a.TargetPlayer.AddBuff(newBuff)
	return nil
}

func (a *AddBuffAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	metadata.SetString("buff_type", string(a.BuffType))

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       gamelog.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.TargetPlayer.UserID,
		Delta:      a.Duration,
		Source:     a.SourceID,
		Metadata:   metadata,
	}
}

// ========== RemoveBuffAction ==========

// RemoveBuffAction represents removing a Buff from player.
type RemoveBuffAction struct {
	TargetPlayer *core.Player  // Player losing Buff
	BuffType     constants.BuffType // Type of Buff to remove
	SourceID     string        // Source identifier
}

// NewRemoveBuffAction creates a new RemoveBuffAction.
func NewRemoveBuffAction(target *core.Player, buffType constants.BuffType, sourceID string) *RemoveBuffAction {
	return &RemoveBuffAction{
		TargetPlayer: target,
		BuffType:     buffType,
		SourceID:     sourceID,
	}
}

func (a *RemoveBuffAction) Type() ActionType { return ActionRemoveBuff }
func (a *RemoveBuffAction) CanModify() bool   { return false }
func (a *RemoveBuffAction) Source() string    { return a.SourceID }
func (a *RemoveBuffAction) Target() string    { return a.TargetPlayer.UserID }

// PreTriggerPhase returns PhaseOnBuffRemoved for death effects/亡语.
func (a *RemoveBuffAction) PreTriggerPhase() constants.Phase {
	return constants.PhaseOnBuffRemoved
}

// PostTriggerPhase returns PhaseAnyTime (no post-trigger after removal).
func (a *RemoveBuffAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

func (a *RemoveBuffAction) Execute(ctx *ActionContext) error {
	a.TargetPlayer.RemoveBuff(a.BuffType)
	return nil
}

func (a *RemoveBuffAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	metadata.SetString("buff_type", string(a.BuffType))

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       gamelog.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.TargetPlayer.UserID,
		Delta:      0,
		Source:     a.SourceID,
		Metadata:   metadata,
	}
}

// ========== TeleportAction ==========

// TeleportAction represents instant teleport to specific position.
// Used by items like 任意门.
type TeleportAction struct {
	TargetPlayer *core.Player // Player teleporting
	TargetPos    int          // Destination position
	SourceID     string       // Source identifier (e.g., "Item_AnyDoor")
}

// NewTeleportAction creates a new TeleportAction.
func NewTeleportAction(target *core.Player, targetPos int, sourceID string) *TeleportAction {
	return &TeleportAction{
		TargetPlayer: target,
		TargetPos:    targetPos,
		SourceID:     sourceID,
	}
}

func (a *TeleportAction) Type() ActionType { return ActionTeleport }
func (a *TeleportAction) CanModify() bool   { return false }
func (a *TeleportAction) Source() string    { return a.SourceID }
func (a *TeleportAction) Target() string    { return a.TargetPlayer.UserID }

// PreTriggerPhase returns PhaseAnyTime (teleport not intercepted).
func (a *TeleportAction) PreTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

// PostTriggerPhase returns PhaseAnyTime (no post-trigger for teleport).
func (a *TeleportAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

func (a *TeleportAction) Execute(ctx *ActionContext) error {
	a.TargetPlayer.Position = a.TargetPos
	return nil
}

func (a *TeleportAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	metadata.SetInt("from_pos", a.TargetPlayer.Position)
	metadata.SetInt("to_pos", a.TargetPos)

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       gamelog.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.TargetPlayer.UserID,
		Delta:      a.TargetPos,
		Source:     a.SourceID,
		Metadata:   metadata,
	}
}

// ========== StealBuffAction ==========

// StealBuffAction represents stealing a Buff from another player.
// Used by 白虎"劫运" faction passive.
type StealBuffAction struct {
	TargetPlayer *core.Player // Player being stolen from
	SourcePlayer *core.Player // Player stealing (owner of 白虎 faction)
	SourceID     string       // Source identifier (e.g., "Faction_BaiHu")
	StolenBuff   *core.Buff   // Buff that was stolen (set after execution)
}

// NewStealBuffAction creates a new StealBuffAction.
func NewStealBuffAction(target, source *core.Player, sourceID string) *StealBuffAction {
	return &StealBuffAction{
		TargetPlayer: target,
		SourcePlayer: source,
		SourceID:     sourceID,
	}
}

func (a *StealBuffAction) Type() ActionType { return ActionStealBuff }
func (a *StealBuffAction) CanModify() bool   { return false }
func (a *StealBuffAction) Source() string    { return a.SourceID }
func (a *StealBuffAction) Target() string    { return a.TargetPlayer.UserID }

// PreTriggerPhase returns PhaseAnyTime (steal not intercepted).
func (a *StealBuffAction) PreTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

// PostTriggerPhase returns PhaseAnyTime (no post-trigger for steal).
func (a *StealBuffAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

func (a *StealBuffAction) Execute(ctx *ActionContext) error {
	// Steal a random buff from target
	if len(a.TargetPlayer.ActiveBuffs) == 0 {
		return nil // No buffs to steal
	}

	// Take first buff (in real implementation, would be random selection)
	stolen := a.TargetPlayer.ActiveBuffs[0]
	a.TargetPlayer.RemoveBuff(stolen.Type)
	a.SourcePlayer.AddBuff(stolen)
	a.StolenBuff = stolen

	return nil
}

func (a *StealBuffAction) LogEntry() gamelog.LogEntry {
	buffType := ""
	if a.StolenBuff != nil {
		buffType = string(a.StolenBuff.Type)
	}

	metadata := util.NewMetadata()
	metadata.SetString("stolen_by", a.SourcePlayer.UserID)
	metadata.SetString("buff_type", buffType)

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       gamelog.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.TargetPlayer.UserID,
		Delta:      0,
		Source:     a.SourceID,
		Metadata:   metadata,
	}
}

// ========== DrawEventAction ==========

// DrawEventAction represents drawing a random event.
// Can be intercepted by 辟邪/玄武 to block bad events.
type DrawEventAction struct {
	TargetPlayer *core.Player // Player drawing event
	SourceID     string       // Source identifier
}

// NewDrawEventAction creates a new DrawEventAction.
func NewDrawEventAction(target *core.Player, sourceID string) *DrawEventAction {
	return &DrawEventAction{
		TargetPlayer: target,
		SourceID:     sourceID,
	}
}

func (a *DrawEventAction) Type() ActionType { return ActionDrawEvent }
func (a *DrawEventAction) CanModify() bool   { return true }
func (a *DrawEventAction) Source() string    { return a.SourceID }
func (a *DrawEventAction) Target() string    { return a.TargetPlayer.UserID }

// PreTriggerPhase returns PhasePreEvent for interception by 辟邪/玄武.
func (a *DrawEventAction) PreTriggerPhase() constants.Phase {
	return constants.PhasePreEvent
}

// PostTriggerPhase returns PhaseAnyTime (no post-trigger for draw).
func (a *DrawEventAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

func (a *DrawEventAction) Execute(ctx *ActionContext) error {
	// Event drawing logic would be implemented here
	// This would use RNG engine to draw from event pool
	return nil
}

func (a *DrawEventAction) LogEntry() gamelog.LogEntry {
	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       gamelog.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.TargetPlayer.UserID,
		Delta:      0,
		Source:     a.SourceID,
	}
}

// ========== RespawnAction ==========

// RespawnAction represents player respawn at checkpoint.
// Used when player dies and needs to respawn.
type RespawnAction struct {
	TargetPlayer  *core.Player // Player respawning
	CheckpointPos int          // Position to respawn at
	SourceID      string       // Source identifier (e.g., "DeathRespawn", "FragileRespawn")
}

// NewRespawnAction creates a new RespawnAction.
func NewRespawnAction(target *core.Player, checkpointPos int, sourceID string) *RespawnAction {
	return &RespawnAction{
		TargetPlayer:  target,
		CheckpointPos: checkpointPos,
		SourceID:      sourceID,
	}
}

func (a *RespawnAction) Type() ActionType { return ActionRespawn }
func (a *RespawnAction) CanModify() bool   { return false }
func (a *RespawnAction) Source() string    { return a.SourceID }
func (a *RespawnAction) Target() string    { return a.TargetPlayer.UserID }

// PreTriggerPhase returns PhasePreRespawn (respawn can be intercepted by Undying等).
func (a *RespawnAction) PreTriggerPhase() constants.Phase {
	return constants.PhasePreRespawn
}

// PostTriggerPhase returns PhaseAnyTime (no post-trigger for respawn).
func (a *RespawnAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

func (a *RespawnAction) Execute(ctx *ActionContext) error {
	a.TargetPlayer.Respawn(a.CheckpointPos)
	return nil
}

func (a *RespawnAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	metadata.SetInt("checkpoint_pos", a.CheckpointPos)

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       gamelog.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.TargetPlayer.UserID,
		Delta:      0,
		Source:     a.SourceID,
		Metadata:   metadata,
	}
}

// ========== FellDownAction ==========

// FellDownAction represents player falling from Fragile cell.
// Used when player lands on a broken fragile cell.
type FellDownAction struct {
	TargetPlayer *core.Player // Player falling
	Position     int          // Position where player fell
	Damage       int          // Damage amount from falling
	SourceID     string       // Source identifier (e.g., "FragileCell")
}

// NewFellDownAction creates a new FellDownAction.
func NewFellDownAction(target *core.Player, position int, damage int, sourceID string) *FellDownAction {
	return &FellDownAction{
		TargetPlayer: target,
		Position:     position,
		Damage:       damage,
		SourceID:     sourceID,
	}
}

func (a *FellDownAction) Type() ActionType { return ActionFellDown }
func (a *FellDownAction) CanModify() bool   { return false }
func (a *FellDownAction) Source() string    { return a.SourceID }
func (a *FellDownAction) Target() string    { return a.TargetPlayer.UserID }

// PreTriggerPhase returns PhaseAnyTime (fell down not intercepted).
func (a *FellDownAction) PreTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

// PostTriggerPhase returns PhaseAnyTime (no post-trigger for fell down).
func (a *FellDownAction) PostTriggerPhase() constants.Phase {
	return constants.PhaseAnyTime
}

func (a *FellDownAction) Execute(ctx *ActionContext) error {
	// Falling damage
	if a.Damage > 0 {
		return a.TargetPlayer.ApplyDamage(a.Damage)
	}
	return nil
}

func (a *FellDownAction) LogEntry() gamelog.LogEntry {
	metadata := util.NewMetadata()
	metadata.SetInt("position", a.Position)

	return gamelog.LogEntry{
		Timestamp:  time.Now(),
		Type:       gamelog.EntryTypeAction,
		ActionType: string(a.Type()),
		Target:     a.TargetPlayer.UserID,
		Delta:      -a.Damage,
		Source:     a.SourceID,
		Metadata:   metadata,
	}
}