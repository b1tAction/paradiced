package engine

import (
	engineaction "github.com/b1tAction/paradiced/internal/engine/action"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/pkg/constants"
)

// ========== Achievement Handler Config ==========

// AchievementHandlerConfig contains handler logic and execution configuration
// for achievement detection via EventBus.
// Mirrors BuffHandlerConfig pattern: Phases, Priority, Handler.
type AchievementHandlerConfig struct {
	Phases   []constants.Phase    // Phases to subscribe to (typically PhasePreAction)
	Priority int                  // Priority (low value like -10, so core handlers run first)
	Handler  AchievementHandler   // Condition detection + grant logic
}

// AchievementHandler defines the handler signature for achievement detection.
// Follows EffectHandler pattern from buff_registry.go.
type AchievementHandler func(phase constants.Phase, ctx *event.Context) error

// GetPhases returns the achievement's trigger phase list.
func (c *AchievementHandlerConfig) GetPhases() []constants.Phase {
	return c.Phases
}

// ========== Achievement Registry ==========

// AchievementRegistry manages achievement definitions and handler configs.
// Definitions contain static metadata (name, description, points).
// HandlerConfigs contain runtime detection logic (EventBus subscription + condition checks).
type AchievementRegistry struct {
	definitions map[constants.AchievementType]*constants.AchievementDefinition
	configs     map[constants.AchievementType]*AchievementHandlerConfig
}

// GlobalAchievementRegistry is the global achievement registry instance.
// Initialized lazily via NewAchievementRegistry() to avoid init cycle with GlobalScoreRegistry.
var GlobalAchievementRegistry *AchievementRegistry

// initAchievementRegistry initializes the global registry on first access.
// Called by EnsureAchievementRegistryInitialized() to avoid init cycle.
func initAchievementRegistry() {
	if GlobalAchievementRegistry == nil {
		GlobalAchievementRegistry = NewAchievementRegistry()
	}
}

// EnsureAchievementRegistryInitialized guarantees the global registry is ready.
// Safe to call multiple times. Resolves init cycle between score_registry and achievement_registry.
func EnsureAchievementRegistryInitialized() {
	initAchievementRegistry()
}

// NewAchievementRegistry creates a new AchievementRegistry with all default definitions.
func NewAchievementRegistry() *AchievementRegistry {
	r := &AchievementRegistry{
		definitions: make(map[constants.AchievementType]*constants.AchievementDefinition),
		configs:     make(map[constants.AchievementType]*AchievementHandlerConfig),
	}
	r.registerDefaultDefinitions()
	r.registerDefaultConfigs()
	return r
}

// ========== Achievement Definitions ==========

// registerDefaultDefinitions registers all achievement definitions with static metadata.
func (r *AchievementRegistry) registerDefaultDefinitions() {
	defs := []constants.AchievementDefinition{
		{Type: constants.AchievementTripleOne, Name: "三连一", Desc: "连续3次掷骰结果为1", Points: 5, EnglishName: "triple_one"},
		{Type: constants.AchievementTripleSix, Name: "三连六", Desc: "连续3次掷骰结果为6", Points: 5, EnglishName: "triple_six"},
		{Type: constants.AchievementBossKillShot, Name: "K头", Desc: "击败Boss（对Boss造成致命一击）", Points: 5, EnglishName: "boss_kill_shot"},
		{Type: constants.AchievementBossDamageTen, Name: "勇者之路", Desc: "对Boss累积伤害达到10点", Points: 5, EnglishName: "boss_damage_ten"},
		{Type: constants.AchievementItemCollector, Name: "道具收藏家", Desc: "同时持有3个或更多道具", Points: 5, EnglishName: "item_collector"},
		{Type: constants.AchievementSurvivor, Name: "生存大师", Desc: "游戏结束时从未死亡", Points: 8, EnglishName: "survivor"},
		{Type: constants.AchievementLuckMaster, Name: "幸运之星", Desc: "游戏结束时LP达到最大值", Points: 5, EnglishName: "luck_master"},
		{Type: constants.AchievementFirstToBoss, Name: "先行者", Desc: "第一个到达Boss格的玩家", Points: 5, EnglishName: "first_to_boss"},
		{Type: constants.AchievementMiniGameWinnerThree, Name: "小游戏之王", Desc: "小游戏获得第1名累计达到3次", Points: 8, EnglishName: "mini_game_winner_three"},
	}
	for i := range defs {
		r.definitions[defs[i].Type] = &defs[i]
	}
}

// GetDefinition returns the achievement definition for a given type.
// Returns nil if the achievement type is not registered.
func (r *AchievementRegistry) GetDefinition(achievementType constants.AchievementType) *constants.AchievementDefinition {
	return r.definitions[achievementType]
}

// AllDefinitions returns all registered achievement definitions.
func (r *AchievementRegistry) AllDefinitions() []*constants.AchievementDefinition {
	result := make([]*constants.AchievementDefinition, 0, len(r.definitions))
	for _, def := range r.definitions {
		result = append(result, def)
	}
	return result
}

// ========== Achievement Handler Configs ==========

// GetConfig returns the handler config for a given achievement type.
// Returns nil for HSM-direct-detection achievements (no EventBus handler).
func (r *AchievementRegistry) GetConfig(achievementType constants.AchievementType) *AchievementHandlerConfig {
	return r.configs[achievementType]
}

// AllConfigs returns all registered handler configs.
// Only includes EventBus-subscribed achievements.
func (r *AchievementRegistry) AllConfigs() []*AchievementHandlerConfig {
	result := make([]*AchievementHandlerConfig, 0, len(r.configs))
	for _, config := range r.configs {
		result = append(result, config)
	}
	return result
}

// EventBusAchievementTypes returns achievement types that use EventBus PhasePreAction subscriptions.
// Boss achievements use PhasePreAction with BossDamageAction type assertion:
// - boss_kill_shot: predicts kill by comparing bossPlayer.HP <= bossAction.Damage
// - boss_damage_ten: tracks cumulative boss damage via player.AddBossDamageDealt
func (r *AchievementRegistry) EventBusAchievementTypes() []constants.AchievementType {
	return []constants.AchievementType{
		constants.AchievementTripleOne,
		constants.AchievementTripleSix,
		constants.AchievementBossKillShot,
		constants.AchievementBossDamageTen,
		constants.AchievementItemCollector,
	}
}

// ========== Handler Registration ==========

// registerDefaultConfigs registers handler configs for EventBus-subscribed achievements.
// Handlers use type assertion on current_action (set by ActionContext.ExecuteAction during
// PhasePreAction publish) to determine the action type and extract action-specific data.
func (r *AchievementRegistry) registerDefaultConfigs() {
	// triple_one: detect 3 consecutive dice rolls of value 1
	r.configs[constants.AchievementTripleOne] = &AchievementHandlerConfig{
		Phases:   []constants.Phase{constants.PhasePreAction},
		Priority: -10,
		Handler:  makeDiceRollTripleHandler(1, constants.AchievementTripleOne),
	}

	// triple_six: detect 3 consecutive dice rolls of value 6
	r.configs[constants.AchievementTripleSix] = &AchievementHandlerConfig{
		Phases:   []constants.Phase{constants.PhasePreAction},
		Priority: -10,
		Handler:  makeDiceRollTripleHandler(6, constants.AchievementTripleSix),
	}

	// boss_kill_shot: detect fatal blow on Boss (predicted via bossPlayer.HP <= Damage)
	// PhasePreAction runs BEFORE BossDamageAction.Execute, so boss HP hasn't been reduced yet.
	// We predict kill by comparing current boss HP with the damage amount.
	r.configs[constants.AchievementBossKillShot] = &AchievementHandlerConfig{
		Phases:   []constants.Phase{constants.PhasePreAction},
		Priority: -10,
		Handler:  makeBossKillShotHandler(),
	}

	// boss_damage_ten: track cumulative boss damage (>= 10 points) and add boss scores.
	// PhasePreAction runs BEFORE action execution, so damage hasn't been applied yet.
	// We track damage via player.AddBossDamageDealt and add boss damage/crit scores here.
	r.configs[constants.AchievementBossDamageTen] = &AchievementHandlerConfig{
		Phases:   []constants.Phase{constants.PhasePreAction},
		Priority: -10,
		Handler:  makeBossDamageTenHandler(),
	}

	// item_collector: detect inventory size >= 3 when item is added
	r.configs[constants.AchievementItemCollector] = &AchievementHandlerConfig{
		Phases:   []constants.Phase{constants.PhasePreAction},
		Priority: -10,
		Handler:  makeItemCollectorHandler(),
	}
}

// ========== Handler Factories ==========

// makeDiceRollTripleHandler detects consecutive dice rolls of a specific value.
// Uses type assertion on current_action to extract RollDiceAction.Steps.
// Records each dice result via Player.PushDiceResult, checks for 3 consecutive matches.
func makeDiceRollTripleHandler(targetValue int, achievementType constants.AchievementType) AchievementHandler {
	return func(phase constants.Phase, ctx *event.Context) error {
		if ctx.Player == nil || ctx.Player.HasAchievement(achievementType) {
			return nil
		}

		// Type-assert current_action to RollDiceAction
		actionRef, ok := ctx.Get("current_action")
		if !ok {
			return nil
		}
		rollAction, ok := actionRef.(*engineaction.RollDiceAction)
		if !ok {
			return nil
		}

		// Record dice result for triple detection
		ctx.Player.PushDiceResult(rollAction.Steps)

		// Check for 3 consecutive target values
		results := ctx.Player.GetLastDiceResults()
		if len(results) >= 3 && results[0] == targetValue && results[1] == targetValue && results[2] == targetValue {
			grantAchievementAndScore(ctx, achievementType)
		}

		return nil
	}
}

// makeItemCollectorHandler detects when player inventory reaches 3+ items.
// Uses type assertion on current_action to check AddItemAction.
// Note: PhasePreAction runs BEFORE AddItemAction.Execute, so the new item hasn't been
// added yet. We check len(player.Inventory) >= 2, which becomes 3 after the item is added.
// Item acquisition score (ScoreItemAcquired) is also added here for consistency.
func makeItemCollectorHandler() AchievementHandler {
	return func(phase constants.Phase, ctx *event.Context) error {
		if ctx.Player == nil {
			return nil
		}

		// Type-assert current_action to AddItemAction
		actionRef, ok := ctx.Get("current_action")
		if !ok {
			return nil
		}
		_, ok = actionRef.(*engineaction.AddItemAction)
		if !ok {
			return nil
		}

		// Add item acquisition score
		addScoreFromContext(ctx, constants.ScoreCategoryItem, constants.ScoreItemAcquired, "获得道具")

		// Check inventory size for item_collector achievement
		// At PhasePreAction, the item hasn't been added yet, so len(Inventory) >= 2
		// means the player will have 3+ items after this action.
		if ctx.Player.HasAchievement(constants.AchievementItemCollector) {
			return nil
		}
		if len(ctx.Player.Inventory) >= 2 {
			grantAchievementAndScore(ctx, constants.AchievementItemCollector)
		}

		return nil
	}
}

// makeBossKillShotHandler detects the fatal blow on Boss.
// Uses type assertion on current_action to check BossDamageAction.
// At PhasePreAction, BossDamageAction.Execute hasn't run yet, so boss HP hasn't been
// reduced. We predict the kill by comparing bossPlayer.HP <= bossAction.Damage.
// Only processes for SourcePlayer (not Boss) since ActorPlayers returns [Boss, SourcePlayer].
func makeBossKillShotHandler() AchievementHandler {
	return func(phase constants.Phase, ctx *event.Context) error {
		if ctx.Player == nil || ctx.Player.ID.IsBoss() {
			// Skip: no player or Boss player (we only want SourcePlayer)
			return nil
		}

		if ctx.Player.HasAchievement(constants.AchievementBossKillShot) {
			return nil
		}

		// Type-assert current_action to BossDamageAction
		actionRef, ok := ctx.Get("current_action")
		if !ok {
			return nil
		}
		bossAction, ok := actionRef.(*engineaction.BossDamageAction)
		if !ok {
			return nil
		}

		// Get boss player to compare HP with damage
		game := getGameFromContext(ctx)
		if game == nil {
			return nil
		}
		bossPlayer := game.GetBossPlayer()
		if bossPlayer == nil || bossPlayer.IsDead {
			return nil
		}

		// Predict kill: boss will die if HP <= damage
		if bossPlayer.HP <= bossAction.Damage {
			// Add kill shot score (15 pts)
			addScoreFromContext(ctx, constants.ScoreCategoryBoss, constants.ScoreBossKillShot, "K头")
			grantAchievementAndScore(ctx, constants.AchievementBossKillShot)
		}

		return nil
	}
}

// makeBossDamageTenHandler tracks cumulative boss damage and adds boss battle scores.
// Uses type assertion on current_action to check BossDamageAction.
// At PhasePreAction, BossDamageAction.Execute hasn't run yet, so we:
// - Add damage to cumulative tracker (player.AddBossDamageDealt)
// - Add boss damage score (damage * ScoreBossDamagePerPt)
// - Add crit bonus if applicable (ScoreBossCritBonus)
// - Check if cumulative damage >= 10 for boss_damage_ten achievement
// Only processes for SourcePlayer (not Boss) since ActorPlayers returns [Boss, SourcePlayer].
func makeBossDamageTenHandler() AchievementHandler {
	return func(phase constants.Phase, ctx *event.Context) error {
		if ctx.Player == nil || ctx.Player.ID.IsBoss() {
			// Skip: no player or Boss player (we only want SourcePlayer)
			return nil
		}

		// Type-assert current_action to BossDamageAction
		actionRef, ok := ctx.Get("current_action")
		if !ok {
			return nil
		}
		bossAction, ok := actionRef.(*engineaction.BossDamageAction)
		if !ok {
			return nil
		}

		damage := bossAction.Damage
		isCrit := bossAction.IsCrit

		// Track cumulative boss damage
		ctx.Player.AddBossDamageDealt(damage)

		// Add boss damage score
		addScoreFromContext(ctx, constants.ScoreCategoryBoss, damage*constants.ScoreBossDamagePerPt, "Boss伤害")

		// Add crit bonus
		if isCrit {
			addScoreFromContext(ctx, constants.ScoreCategoryBoss, constants.ScoreBossCritBonus, "Boss暴击")
		}

		// Check boss_damage_ten achievement: cumulative boss damage >= 10
		if !ctx.Player.HasAchievement(constants.AchievementBossDamageTen) && ctx.Player.GetBossDamageDealt() >= 10 {
			grantAchievementAndScore(ctx, constants.AchievementBossDamageTen)
		}

		return nil
	}
}

// ========== Helper Functions ==========

// grantAchievementAndScore grants an achievement and adds its score points.
// Uses Game reference stored in context metadata to call engine.Game methods.
func grantAchievementAndScore(ctx *event.Context, achievementType constants.AchievementType) {
	game := getGameFromContext(ctx)
	if game == nil {
		return
	}
	game.GrantAchievementToPlayer(ctx.Player, achievementType)
	def := GlobalAchievementRegistry.GetDefinition(achievementType)
	if def != nil {
		game.AddScoreToPlayer(ctx.Player, constants.ScoreCategoryAchievement, def.Points, def.Name, 0)
	}
}

// addScoreFromContext adds score points via Game reference stored in context metadata.
func addScoreFromContext(ctx *event.Context, category constants.ScoreCategory, points int, reason string) {
	game := getGameFromContext(ctx)
	if game == nil {
		return
	}
	game.AddScoreToPlayer(ctx.Player, category, points, reason, 0)
}

// getGameFromContext extracts the Game reference from context metadata.
// The Game reference is injected by InitializePlayerAchievements handler closure.
func getGameFromContext(ctx *event.Context) *Game {
	gameRef, ok := ctx.Get("game")
	if !ok {
		return nil
	}
	game, ok := gameRef.(*Game)
	if !ok {
		return nil
	}
	return game
}