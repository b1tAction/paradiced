package engine

import (
	"testing"

	"github.com/b1tAction/fated/internal/core"
	engineaction "github.com/b1tAction/fated/internal/engine/action"
	"github.com/b1tAction/fated/pkg/constants"
	"github.com/b1tAction/fated/pkg/event"
	"github.com/b1tAction/fated/pkg/id"
)

// ========== EventHandler Tests ==========

func TestHasBuffHandler(t *testing.T) {
	// Fire has custom handler
	if !core.HasBuffHandler(core.BuffTypeFire) {
		t.Error("BuffTypeFire should have custom handler")
	}

	// Curse has no custom handler (uses default)
	if core.HasBuffHandler(core.BuffTypeCurse) {
		t.Error("BuffTypeCurse should not have custom handler")
	}

	// Hidden has no custom handler
	if core.HasBuffHandler(core.BuffTypeHidden) {
		t.Error("BuffTypeHidden should not have custom handler")
	}
}

func TestGetBuffHandler(t *testing.T) {
	// Get Fire handler
	handler := core.GetBuffHandler(core.BuffTypeFire)
	if handler == nil {
		t.Error("GetBuffHandler(Fire) should return handler")
	}

	// Get Curse handler (none)
	handler = core.GetBuffHandler(core.BuffTypeCurse)
	if handler != nil {
		t.Error("GetBuffHandler(Curse) should return nil")
	}
}

func TestFireBuffHandlerBehavior(t *testing.T) {
	// Test Fire buff handler behavior via GetBuffHandler
	player := core.NewPlayer(core.PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: core.FactionZhuQue,
		MaxLP:   5,
	})
	initialLP := player.LP

	// Get Fire handler
	handler := core.GetBuffHandler(core.BuffTypeFire)
	if handler == nil {
		t.Fatal("Fire handler should not be nil")
	}

	// Create context
	ctx := event.NewContext(player)

	// Execute handler (BeforeTurn Phase)
	handler(constants.PhaseBeforeTurn, ctx)

	// First execution, counter should be 1, LP unchanged
	if player.GetFireCounter() != 1 {
		t.Errorf("FireCounter = %d, expected 1", player.GetFireCounter())
	}
	if player.LP != initialLP {
		t.Errorf("LP should not change after first trigger")
	}

	// Execute 3 more times (reach 4 triggers)
	for i := 0; i < 3; i++ {
		handler(constants.PhaseBeforeTurn, ctx)
	}

	// After 4 times, LP should +1, counter reset
	if player.LP != initialLP+1 {
		t.Errorf("LP = %d, expected %d (initial+1)", player.LP, initialLP+1)
	}
	if player.GetFireCounter() != 0 {
		t.Errorf("FireCounter = %d, expected 0 (reset)", player.GetFireCounter())
	}
}

func TestFireBuffHandlerNonBeforeTurnPhase(t *testing.T) {
	// Fire only executes in BeforeTurn Phase
	player := core.NewPlayer(core.PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: core.FactionZhuQue,
		MaxLP:   5,
	})
	ctx := event.NewContext(player)

	handler := core.GetBuffHandler(core.BuffTypeFire)

	// Execute in other Phase should be ineffective
	handler(constants.PhaseAfterTurn, ctx)
	if player.GetFireCounter() != 0 {
		t.Errorf("FireCounter should be 0 when not BeforeTurn phase")
	}
	if player.LP != 5 {
		t.Errorf("LP should not change when not BeforeTurn phase")
	}
}

func TestExecuteDefaultBuffAction(t *testing.T) {
	// Test default handler HP/LP modification using Action system
	tests := []struct {
		name       string
		buffType   core.BuffType
		initHP     int
		initLP     int
		expectedHP int
		expectedLP int
	}{
		{
			name:       "Curse LP-1",
			buffType:   core.BuffTypeCurse,
			initHP:     6,
			initLP:     5,
			expectedHP: 6,
			expectedLP: 4,
		},
		{
			name:       "Divine LP+1",
			buffType:   core.BuffTypeDivine,
			initHP:     6,
			initLP:     5,
			expectedHP: 6,
			expectedLP: 6,
		},
		{
			name:       "Rain HP+1",
			buffType:   core.BuffTypeRain,
			initHP:     6,
			initLP:     5,
			expectedHP: 7,
			expectedLP: 5,
		},
		{
			name:       "Corrupt HP-1",
			buffType:   core.BuffTypeCorrupt,
			initHP:     6,
			initLP:     5,
			expectedHP: 5,
			expectedLP: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player := core.NewPlayer(core.PlayerConfig{
				ID:    id.NewPlayerID(),
				MaxHP: tt.initHP,
				MaxLP: tt.initLP,
			})
			player.HP = tt.initHP
			player.LP = tt.initLP

			def := core.GetBuffDefinition(tt.buffType)

			// Create ActionContext
			actionCtx := engineaction.NewActionContext(nil, nil, nil)
			executeDefaultBuffAction(def, player, actionCtx)

			// Process the queue to execute actions
			actionCtx.ProcessQueue()

			if player.HP != tt.expectedHP {
				t.Errorf("HP = %d, expected %d", player.HP, tt.expectedHP)
			}
			if player.LP != tt.expectedLP {
				t.Errorf("LP = %d, expected %d", player.LP, tt.expectedLP)
			}
		})
	}
}

func TestExecuteDefaultBuffActionWithHPDamage(t *testing.T) {
	// Test HPPerTurn negative calling ApplyDamage
	player := core.NewPlayer(core.PlayerConfig{
		ID:    id.NewPlayerID(),
		MaxHP: 6,
	})
	player.HP = 6

	// Create a local definition for testing (don't modify global registry)
	def := &core.BuffDefinition{
		Type:      core.BuffTypeCorrupt,
		HPPerTurn: -2, // Damage 2 HP per turn
	}

	// Create ActionContext
	actionCtx := engineaction.NewActionContext(nil, nil, nil)
	executeDefaultBuffAction(def, player, actionCtx)

	// Process the queue to execute actions
	actionCtx.ProcessQueue()

	if player.HP != 4 {
		t.Errorf("HP = %d, expected 4 (damage 2 HP)", player.HP)
	}
}

func TestExecuteDefaultBuffActionWithHealing(t *testing.T) {
	// Test HPPerTurn positive calling Heal
	player := core.NewPlayer(core.PlayerConfig{
		ID:    id.NewPlayerID(),
		MaxHP: 6,
	})
	player.HP = 6

	// Create a local definition for testing (don't modify global registry)
	def := &core.BuffDefinition{
		Type:      core.BuffTypeRain,
		HPPerTurn: 2, // Heal 2 HP per turn
	}

	// Create ActionContext
	actionCtx := engineaction.NewActionContext(nil, nil, nil)
	executeDefaultBuffAction(def, player, actionCtx)

	// Process the queue to execute actions
	actionCtx.ProcessQueue()

	if player.HP != 8 {
		t.Errorf("HP = %d, expected 8 (heal 2 HP)", player.HP)
	}
}
