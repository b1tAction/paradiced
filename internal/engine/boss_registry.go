package engine

import (
	"github.com/b1tAction/paradiced/internal/core"
	engineaction "github.com/b1tAction/paradiced/internal/engine/action"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/rng"
)

// BossDefinition contains static metadata for a Boss entity.
type BossDefinition struct {
	Type        constants.BossType
	Name        string // Chinese display name
	EnglishName string // English identifier
	Description string // Description text
	MaxHP       int    // Default max HP
}

// BossSkillHandler is the handler signature for Boss skills.
// Unlike Buff/Item/Event handlers (which use event.Context from EventBus),
// Boss skills are executed directly in TurnBossBattleState using ActionContext.
type BossSkillHandler func(game *Game, actionCtx *engineaction.ActionContext, targets []*core.Player) error

// BossSkillHandlerConfig contains handler logic for a Boss skill.
type BossSkillHandlerConfig struct {
	Type    constants.BossSkillType
	Handler BossSkillHandler
}

// BossRegistry is the registry for Boss definitions and skill handlers.
type BossRegistry struct {
	defs         map[constants.BossType]*BossDefinition
	skillConfigs map[constants.BossSkillType]*BossSkillHandlerConfig
	skillPool    []*rng.EvaluatedItem // Pre-built skill pool for random draw
}

// GlobalBossRegistry is the global Boss registry.
var GlobalBossRegistry *BossRegistry

func init() {
	GlobalBossRegistry = NewBossRegistry()
	registerAllBossData()
}

// NewBossRegistry creates a new Boss registry.
func NewBossRegistry() *BossRegistry {
	return &BossRegistry{
		defs:         make(map[constants.BossType]*BossDefinition),
		skillConfigs: make(map[constants.BossSkillType]*BossSkillHandlerConfig),
		skillPool:    make([]*rng.EvaluatedItem, 0),
	}
}

// RegisterBoss registers a Boss definition.
func (r *BossRegistry) RegisterBoss(def *BossDefinition) {
	if def == nil || def.Type == "" {
		return
	}
	r.defs[def.Type] = def
}

// RegisterBossSkill registers a Boss skill handler.
func (r *BossRegistry) RegisterBossSkill(config *BossSkillHandlerConfig) {
	if config == nil || config.Type == "" {
		return
	}
	r.skillConfigs[config.Type] = config
}

// GetBossDefinition returns the Boss definition for a given type.
func (r *BossRegistry) GetBossDefinition(bossType constants.BossType) *BossDefinition {
	return r.defs[bossType]
}

// GetBossSkillHandler returns the skill handler for a given skill type.
func (r *BossRegistry) GetBossSkillHandler(skillType constants.BossSkillType) *BossSkillHandlerConfig {
	return r.skillConfigs[skillType]
}

// BuildBossSkillPool builds the skill pool for Boss random draw.
// All skills have equal weight (Evaluation=50, neutral).
func (r *BossRegistry) BuildBossSkillPool() []*rng.EvaluatedItem {
	pool := make([]*rng.EvaluatedItem, 0, len(r.skillConfigs))
	for skillType := range r.skillConfigs {
		pool = append(pool, &rng.EvaluatedItem{
			Type: string(skillType),
			Eval: constants.Evaluation(50), // Equal weight for all skills
		})
	}
	r.skillPool = pool
	return pool
}

// GetBossSkillPool returns the pre-built skill pool.
func (r *BossRegistry) GetBossSkillPool() []*rng.EvaluatedItem {
	return r.skillPool
}

// ========== Boss Skill Handlers ==========

// registerAllBossData registers all Boss definitions and skill handlers.
func registerAllBossData() {
	// Register Boss definition
	GlobalBossRegistry.RegisterBoss(&BossDefinition{
		Type:        constants.BossTypeBeast,
		Name:        "凶兽",
		EnglishName: "Beast",
		Description: "The ancient beast that guards the end of the map",
		MaxHP:       50,
	})

	// Thunder: AOE damage 3 to all boss-cell players
	GlobalBossRegistry.RegisterBossSkill(&BossSkillHandlerConfig{
		Type: constants.BossSkillThunder,
		Handler: func(game *Game, actionCtx *engineaction.ActionContext, targets []*core.Player) error {
			bossPlayer := game.GetBossPlayer()
			for _, target := range targets {
				attackAction := engineaction.NewBossAttackAction(
					bossPlayer,
					target,
					3,
					constants.BossAttackSkill,
					string(constants.SourceBossSkillThunder),
				)
				actionCtx.PushDerivedAction(attackAction)
			}
			return nil
		},
	})

	// Curse: add curse buff to all boss-cell players
	GlobalBossRegistry.RegisterBossSkill(&BossSkillHandlerConfig{
		Type: constants.BossSkillCurse,
		Handler: func(game *Game, actionCtx *engineaction.ActionContext, targets []*core.Player) error {
			for _, target := range targets {
				addBuffAction := engineaction.NewAddBuffAction(target, constants.BuffTypeCurse, string(constants.SourceBossSkillCurse))
				actionCtx.PushDerivedAction(addBuffAction)
			}
			return nil
		},
	})

	// Rest: Boss heals 20 HP
	GlobalBossRegistry.RegisterBossSkill(&BossSkillHandlerConfig{
		Type: constants.BossSkillRest,
		Handler: func(game *Game, actionCtx *engineaction.ActionContext, targets []*core.Player) error {
			bossPlayer := game.GetBossPlayer()
			if bossPlayer == nil || bossPlayer.IsDead {
				return nil
			}
			healAction := engineaction.NewHealAction(bossPlayer, 20, string(constants.SourceBossSkillRest))
			actionCtx.PushDerivedAction(healAction)
			return nil
		},
	})

	// Thorns: Add thorns buff to Boss itself (reflect 30% damage to attacking player)
	GlobalBossRegistry.RegisterBossSkill(&BossSkillHandlerConfig{
		Type: constants.BossSkillThorns,
		Handler: func(game *Game, actionCtx *engineaction.ActionContext, targets []*core.Player) error {
			bossPlayer := game.GetBossPlayer()
			if bossPlayer != nil {
				addBuffAction := engineaction.NewAddBuffAction(bossPlayer, constants.BuffTypeThorns, string(constants.SourceBossSkillThorns))
				actionCtx.PushDerivedAction(addBuffAction)
			}
			return nil
		},
	})

	// Build skill pool
	GlobalBossRegistry.BuildBossSkillPool()
}