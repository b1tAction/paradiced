package engine

import (
	"testing"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
)

// ========== BuffHandler Tests ==========

func TestAllBuffsHaveHandlerConfig(t *testing.T) {
	// All Buffs should have HandlerConfig registered
	allBuffs := GetAllBuffTypes()
	for _, bt := range allBuffs {
		config := GetBuffHandlerConfig(bt)
		if config == nil {
			t.Errorf("BuffType(%s) should have HandlerConfig", bt)
		}
	}
}

func TestAllBuffsHaveHandler(t *testing.T) {
	// All Buffs should have Handler set
	allBuffs := GetAllBuffTypes()
	for _, bt := range allBuffs {
		if !HasBuffHandler(bt) {
			t.Errorf("BuffType(%s) should have Handler", bt)
		}
	}
}

func TestBuffHandlerConfigPhases(t *testing.T) {
	tests := []struct {
		buffType    constants.BuffType
		phase       constants.Phase
		hasPhase    bool
		priority    int
		needConfirm bool
	}{
		{constants.BuffTypeFire, constants.PhaseBeforeTurn, true, 10, false},
		{constants.BuffTypeCurse, constants.PhaseBeforeTurn, true, 50, false},
		{constants.BuffTypeDivine, constants.PhaseBeforeTurn, true, 50, false},
		{constants.BuffTypeLost, constants.PhasePreMove, true, 100, false},
		{constants.BuffTypeHidden, constants.PhasePreDamage, true, 100, false},
		{constants.BuffTypeRain, constants.PhaseAfterTurn, true, 50, false},
		{constants.BuffTypeCorrupt, constants.PhaseAfterTurn, true, 50, false},
		{constants.BuffTypeExorcism, constants.PhasePreEvent, true, 80, false},
		{constants.BuffTypePoison, constants.PhaseBeforeTurn, true, 30, false},
	}

	for _, tt := range tests {
		config := GetBuffHandlerConfig(tt.buffType)
		if config == nil {
			t.Errorf("BuffType(%s) has no HandlerConfig", tt.buffType)
			continue
		}

		if config.HasPhase(tt.phase) != tt.hasPhase {
			t.Errorf("%s.HasPhase(%s) = %v, expected %v", tt.buffType, tt.phase, config.HasPhase(tt.phase), tt.hasPhase)
		}

		if config.Priority != tt.priority {
			t.Errorf("%s.Priority = %d, expected %d", tt.buffType, config.Priority, tt.priority)
		}

		if config.NeedConfirm != tt.needConfirm {
			t.Errorf("%s.NeedConfirm = %v, expected %v", tt.buffType, config.NeedConfirm, tt.needConfirm)
		}
	}
}

// ========== Fire Buff Handler Tests ==========

func TestFireBuffHandlerBehavior(t *testing.T) {
	// Test Fire buff handler behavior
	player := core.NewPlayer(core.PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: constants.FactionZhuQue,
		MaxLP:   5,
	})
	initialLP := player.LP

	// Get Fire handler config
	config := GetBuffHandlerConfig(constants.BuffTypeFire)
	if config == nil {
		t.Fatal("Fire should have HandlerConfig")
	}
	handler := config.Handler
	if handler == nil {
		t.Fatal("Fire should have Handler")
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
		Faction: constants.FactionZhuQue,
		MaxLP:   5,
	})
	ctx := event.NewContext(player)

	config := GetBuffHandlerConfig(constants.BuffTypeFire)
	handler := config.Handler

	// Execute in other Phase should be ineffective
	handler(constants.PhaseAfterTurn, ctx)
	if player.GetFireCounter() != 0 {
		t.Errorf("FireCounter should be 0 when not BeforeTurn phase")
	}
	if player.LP != 5 {
		t.Errorf("LP should not change when not BeforeTurn phase")
	}
}

// ========== Curse Buff Handler Tests ==========

func TestCurseBuffHandlerBehavior(t *testing.T) {
	// Test Curse buff LP-1 effect
	player := core.NewPlayer(core.PlayerConfig{
		ID:    id.NewPlayerID(),
		MaxLP: 5,
	})
	player.LP = 5

	config := GetBuffHandlerConfig(constants.BuffTypeCurse)
	if config == nil {
		t.Fatal("Curse should have HandlerConfig")
	}
	handler := config.Handler
	if handler == nil {
		t.Fatal("Curse should have Handler")
	}

	ctx := event.NewContext(player)

	// Execute handler
	handler(constants.PhaseBeforeTurn, ctx)

	// LP should be 4 after handler execution
	if player.LP != 4 {
		t.Errorf("LP = %d, expected 4 (LP-1)", player.LP)
	}
}

// ========== Divine Buff Handler Tests ==========

func TestDivineBuffHandlerBehavior(t *testing.T) {
	// Test Divine buff LP+1 effect
	player := core.NewPlayer(core.PlayerConfig{
		ID:    id.NewPlayerID(),
		MaxLP: 5,
	})
	player.LP = 3

	config := GetBuffHandlerConfig(constants.BuffTypeDivine)
	if config == nil {
		t.Fatal("Divine should have HandlerConfig")
	}
	handler := config.Handler
	if handler == nil {
		t.Fatal("Divine should have Handler")
	}

	ctx := event.NewContext(player)

	// Execute handler
	handler(constants.PhaseBeforeTurn, ctx)

	// LP should be 4 after handler execution
	if player.LP != 4 {
		t.Errorf("LP = %d, expected 4 (LP+1)", player.LP)
	}
}

// ========== Rain Buff Handler Tests ==========

func TestRainBuffHandlerBehavior(t *testing.T) {
	// Test Rain buff HP+1 every 2 turns effect
	player := core.NewPlayer(core.PlayerConfig{
		ID:    id.NewPlayerID(),
		MaxHP: 10,
	})
	player.HP = 6

	config := GetBuffHandlerConfig(constants.BuffTypeRain)
	if config == nil {
		t.Fatal("Rain should have HandlerConfig")
	}
	handler := config.Handler
	if handler == nil {
		t.Fatal("Rain should have Handler")
	}

	ctx := event.NewContext(player)

	// First execution - counter 1, no HP change
	handler(constants.PhaseAfterTurn, ctx)
	counter, _ := ctx.GetInt("buff_turn_counter")
	if counter != 1 {
		t.Errorf("counter = %d, expected 1", counter)
	}
	// HP change is signaled via context, not applied directly
	hpChange, err := ctx.GetInt("hp_change")
	if err == nil {
		t.Errorf("hp_change should not be set on first turn, got %d", hpChange)
	}

	// Second execution - counter reaches 2, HP+1
	handler(constants.PhaseAfterTurn, ctx)
	hpChange, err = ctx.GetInt("hp_change")
	if err != nil {
		t.Error("hp_change should be set on second turn")
	}
	if hpChange != 1 {
		t.Errorf("hp_change = %d, expected 1", hpChange)
	}
}

// ========== Corrupt Buff Handler Tests ==========

func TestCorruptBuffHandlerBehavior(t *testing.T) {
	// Test Corrupt buff HP-1 every 2 turns effect
	player := core.NewPlayer(core.PlayerConfig{
		ID:    id.NewPlayerID(),
		MaxHP: 10,
	})
	player.HP = 6

	config := GetBuffHandlerConfig(constants.BuffTypeCorrupt)
	if config == nil {
		t.Fatal("Corrupt should have HandlerConfig")
	}
	handler := config.Handler
	if handler == nil {
		t.Fatal("Corrupt should have Handler")
	}

	ctx := event.NewContext(player)

	// First execution - counter 1, no HP change
	handler(constants.PhaseAfterTurn, ctx)
	hpChange, err := ctx.GetInt("hp_change")
	if err == nil {
		t.Errorf("hp_change should not be set on first turn, got %d", hpChange)
	}

	// Second execution - counter reaches 2, HP-1 signaled
	handler(constants.PhaseAfterTurn, ctx)
	hpChange, err = ctx.GetInt("hp_change")
	if err != nil {
		t.Error("hp_change should be set on second turn")
	}
	if hpChange != -1 {
		t.Errorf("hp_change = %d, expected -1", hpChange)
	}
}

// ========== Lost Buff Handler Tests ==========

func TestLostBuffHandlerBehavior(t *testing.T) {
	// Test Lost buff reverse movement effect
	player := core.NewPlayer(core.PlayerConfig{
		ID: id.NewPlayerID(),
	})

	config := GetBuffHandlerConfig(constants.BuffTypeLost)
	if config == nil {
		t.Fatal("Lost should have HandlerConfig")
	}
	handler := config.Handler
	if handler == nil {
		t.Fatal("Lost should have Handler")
	}

	ctx := event.NewContext(player)

	// Execute handler
	handler(constants.PhasePreMove, ctx)

	// Should signal reverse movement
	reverse, err := ctx.GetBool("reverse_movement")
	if err != nil {
		t.Error("reverse_movement should be set")
	}
	if !reverse {
		t.Error("reverse_movement should be true")
	}
}

func TestLostBuffHandlerOtherPhase(t *testing.T) {
	// Lost only triggers in PreMove phase
	player := core.NewPlayer(core.PlayerConfig{
		ID: id.NewPlayerID(),
	})

	config := GetBuffHandlerConfig(constants.BuffTypeLost)
	handler := config.Handler

	ctx := event.NewContext(player)

	// Execute in BeforeTurn phase - should not trigger
	handler(constants.PhaseBeforeTurn, ctx)

	_, err := ctx.GetBool("reverse_movement")
	if err == nil {
		t.Error("reverse_movement should not be set in BeforeTurn phase")
	}
}

// ========== Hidden Buff Handler Tests ==========

func TestHiddenBuffHandlerBehavior(t *testing.T) {
	// Test Hidden buff immunity effect
	player := core.NewPlayer(core.PlayerConfig{
		ID: id.NewPlayerID(),
	})

	config := GetBuffHandlerConfig(constants.BuffTypeHidden)
	if config == nil {
		t.Fatal("Hidden should have HandlerConfig")
	}
	handler := config.Handler
	if handler == nil {
		t.Fatal("Hidden should have Handler")
	}

	ctx := event.NewContext(player)

	// Execute handler
	handler(constants.PhasePreDamage, ctx)

	// Should signal action blocked
 blocked, err := ctx.GetBool("action_blocked")
	if err != nil {
		t.Error("action_blocked should be set")
	}
	if !blocked {
		t.Error("action_blocked should be true")
	}

 blockedBy, err := ctx.GetString("blocked_by")
	if err != nil {
		t.Error("blocked_by should be set")
	}
	if blockedBy != "Buff_Hidden" {
		t.Errorf("blocked_by = %s, expected Buff_Hidden", blockedBy)
	}
}

// ========== Exorcism Buff Handler Tests ==========

func TestExorcismBuffHandlerBehavior(t *testing.T) {
	// Test Exorcism buff poison immunity effect
	player := core.NewPlayer(core.PlayerConfig{
		ID: id.NewPlayerID(),
	})

	config := GetBuffHandlerConfig(constants.BuffTypeExorcism)
	if config == nil {
		t.Fatal("Exorcism should have HandlerConfig")
	}
	handler := config.Handler
	if handler == nil {
		t.Fatal("Exorcism should have Handler")
	}

	ctx := event.NewContext(player)

	// Execute handler
	handler(constants.PhasePreEvent, ctx)

	// Should signal block poison effect
 blocked, err := ctx.GetBool("block_poison_effect")
	if err != nil {
		t.Error("block_poison_effect should be set")
	}
	if !blocked {
		t.Error("block_poison_effect should be true")
	}
}

// ========== Poison Buff Handler Tests ==========

func TestPoisonBuffHandlerBehavior(t *testing.T) {
	// Test Poison buff bad event effect
	player := core.NewPlayer(core.PlayerConfig{
		ID: id.NewPlayerID(),
	})

	config := GetBuffHandlerConfig(constants.BuffTypePoison)
	if config == nil {
		t.Fatal("Poison should have HandlerConfig")
	}
	handler := config.Handler
	if handler == nil {
		t.Fatal("Poison should have Handler")
	}

	ctx := event.NewContext(player)

	// Execute handler
	handler(constants.PhaseBeforeTurn, ctx)

	// Should signal draw bad event
 drawBad, err := ctx.GetBool("draw_bad_event")
	if err != nil {
		t.Error("draw_bad_event should be set")
	}
	if !drawBad {
		t.Error("draw_bad_event should be true")
	}
}