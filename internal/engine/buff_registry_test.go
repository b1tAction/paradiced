package engine

import (
	"fmt"
	"testing"

	"github.com/b1tAction/paradiced/internal/core"
	engineaction "github.com/b1tAction/paradiced/internal/engine/action"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/internal/gamemap"
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
		{constants.BuffTypeCurse, constants.PhasePostBuffApplied, true, 50, false},
		{constants.BuffTypeCurse, constants.PhasePreBuffRemoved, true, 50, false},
		{constants.BuffTypeDivine, constants.PhasePostBuffApplied, true, 50, false},
		{constants.BuffTypeDivine, constants.PhasePreBuffRemoved, true, 50, false},
		{constants.BuffTypeLost, constants.PhasePreMove, true, 100, false},
		{constants.BuffTypeHidden, constants.PhasePreBuffApplied, true, 100, false},
		{constants.BuffTypeRain, constants.PhaseAfterTurn, true, 50, false},
		{constants.BuffTypeCorrupt, constants.PhaseAfterTurn, true, 50, false},
		{constants.BuffTypeExorcism, constants.PhasePreEvent, true, 80, false},
		{constants.BuffTypePoison, constants.PhaseBeforeTurn, true, 30, false},
		{constants.BuffTypeDeathMark, constants.PhasePreAction, true, 999, false},
		{constants.BuffTypeThorns, constants.PhasePreDamage, true, 50, false},
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
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: constants.FactionZhuQue,
		MaxLP:   5,
	})
	game.AddPlayer(player)
	game.Log.StartTurn(1, 0, player.ID.UUID())
	initialLP := player.LP

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)

	handler := GetBuffHandlerConfig(constants.BuffTypeFire).Handler

	// Execute 4 times to trigger LP+1
	for i := 0; i < 4; i++ {
		handler(constants.PhaseBeforeTurn, ctx)

		// Bridge derived actions and process
		for _, da := range ctx.GetDerivedActions() {
			if act, ok := da.(engineaction.Action); ok {
				actionCtx.PushDerivedAction(act)
			}
		}
		actionCtx.ProcessQueue()
		ctx.ClearDerivedActions()
	}

	if player.LP != initialLP+1 {
		t.Errorf("LP = %d, expected %d (initial+1)", player.LP, initialLP+1)
	}
	if player.GetFireCounter() != 0 {
		t.Errorf("FireCounter = %d, expected 0 (reset)", player.GetFireCounter())
	}
}

func TestFireBuffHandlerNonBeforeTurnPhase(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: constants.FactionZhuQue,
		MaxLP:   5,
	})
	game.AddPlayer(player)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)

	handler := GetBuffHandlerConfig(constants.BuffTypeFire).Handler

	// Execute in other Phase should be ineffective
	handler(constants.PhaseAfterTurn, ctx)
	if player.GetFireCounter() != 0 {
		t.Errorf("FireCounter should be 0 when not BeforeTurn phase")
	}
	if player.LP != 4 {
		t.Errorf("LP should be initial 4 (MaxLP=5 but InitLP=4 default), got %d", player.LP)
	}
}

// ========== Curse Buff Handler Tests ==========

func TestCurseBuffHandlerBehavior(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxLP: 5})
	player.LP = 5
	game.AddPlayer(player)
	game.Log.StartTurn(1, 0, player.ID.UUID())

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)
	ctx.Set("applied_buff_type", string(constants.BuffTypeCurse))

	handler := GetBuffHandlerConfig(constants.BuffTypeCurse).Handler
	handler(constants.PhasePostBuffApplied, ctx)

	// Bridge derived actions and process
	for _, da := range ctx.GetDerivedActions() {
		if act, ok := da.(engineaction.Action); ok {
			actionCtx.PushDerivedAction(act)
		}
	}
	actionCtx.ProcessQueue()

	if player.LP != 4 {
		t.Errorf("LP = %d, expected 4 (LP-1)", player.LP)
	}
}

// ========== Divine Buff Handler Tests ==========

func TestDivineBuffHandlerBehavior(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxLP: 5})
	player.LP = 3
	game.AddPlayer(player)
	game.Log.StartTurn(1, 0, player.ID.UUID())

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)
	ctx.Set("applied_buff_type", string(constants.BuffTypeDivine))

	handler := GetBuffHandlerConfig(constants.BuffTypeDivine).Handler
	handler(constants.PhasePostBuffApplied, ctx)

	// Bridge derived actions and process
	for _, da := range ctx.GetDerivedActions() {
		if act, ok := da.(engineaction.Action); ok {
			actionCtx.PushDerivedAction(act)
		}
	}
	actionCtx.ProcessQueue()

	if player.LP != 4 {
		t.Errorf("LP = %d, expected 4 (LP+1)", player.LP)
	}
}

// ========== Rain Buff Handler Tests ==========

func TestRainBuffHandlerBehavior(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	player.HP = 6
	game.AddPlayer(player)
	game.Log.StartTurn(1, 0, player.ID.UUID())

	// Add Rain buff to player (required for counter persistence in Buff.Metadata)
	rainBuff := core.NewBuff(constants.BuffTypeRain, 4)
	player.ActiveBuffs = append(player.ActiveBuffs, rainBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	handler := GetBuffHandlerConfig(constants.BuffTypeRain).Handler

	// First execution with new context - counter 1 in Buff.Metadata, no HP change
	ctx1 := event.NewContext(player)
	ctx1.Set("action_context", actionCtx)
	handler(constants.PhaseAfterTurn, ctx1)
	if len(ctx1.GetDerivedActions()) != 0 {
		t.Errorf("Should have no derived actions on first turn, got %d", len(ctx1.GetDerivedActions()))
	}
	if player.HP != 6 {
		t.Errorf("HP should not change on first turn, got HP=%d", player.HP)
	}
	// Verify counter persisted in Buff.Metadata
	if rainBuff.GetIntOrDefault("buff_turn_counter", 0) != 1 {
		t.Errorf("buff_turn_counter should be 1 after first call, got %d", rainBuff.GetIntOrDefault("buff_turn_counter", 0))
	}

	// Second execution with new context - counter reaches 2 in Buff.Metadata, HP+1
	ctx2 := event.NewContext(player)
	ctx2.Set("action_context", actionCtx)
	handler(constants.PhaseAfterTurn, ctx2)

	// Bridge derived actions and process
	for _, da := range ctx2.GetDerivedActions() {
		if act, ok := da.(engineaction.Action); ok {
			actionCtx.PushDerivedAction(act)
		}
	}
	actionCtx.ProcessQueue()

	if player.HP != 7 {
		t.Errorf("HP = %d, expected 7 (HP+1)", player.HP)
	}
	// Counter should be reset to 0 after trigger
	if rainBuff.GetIntOrDefault("buff_turn_counter", 0) != 0 {
		t.Errorf("buff_turn_counter should be 0 after reset, got %d", rainBuff.GetIntOrDefault("buff_turn_counter", 0))
	}
}

// ========== Corrupt Buff Handler Tests ==========

func TestCorruptBuffHandlerBehavior(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	player.HP = 6
	game.AddPlayer(player)
	game.Log.StartTurn(1, 0, player.ID.UUID())

	// Add Corrupt buff to player (required for counter persistence in Buff.Metadata)
	corruptBuff := core.NewBuff(constants.BuffTypeCorrupt, 4)
	player.ActiveBuffs = append(player.ActiveBuffs, corruptBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	handler := GetBuffHandlerConfig(constants.BuffTypeCorrupt).Handler

	// First execution with new context - counter 1 in Buff.Metadata, no HP change
	ctx1 := event.NewContext(player)
	ctx1.Set("action_context", actionCtx)
	handler(constants.PhaseAfterTurn, ctx1)
	if len(ctx1.GetDerivedActions()) != 0 {
		t.Errorf("Should have no derived actions on first turn, got %d", len(ctx1.GetDerivedActions()))
	}
	if player.HP != 6 {
		t.Errorf("HP should not change on first turn, got HP=%d", player.HP)
	}

	// Second execution with new context - counter reaches 2 in Buff.Metadata, HP-1
	ctx2 := event.NewContext(player)
	ctx2.Set("action_context", actionCtx)
	handler(constants.PhaseAfterTurn, ctx2)

	// Bridge derived actions and process
	for _, da := range ctx2.GetDerivedActions() {
		if act, ok := da.(engineaction.Action); ok {
			actionCtx.PushDerivedAction(act)
		}
	}
	actionCtx.ProcessQueue()

	if player.HP != 5 {
		t.Errorf("HP = %d, expected 5 (HP-1)", player.HP)
	}
	// Counter should be reset to 0 after trigger
	if corruptBuff.GetIntOrDefault("buff_turn_counter", 0) != 0 {
		t.Errorf("buff_turn_counter should be 0 after reset, got %d", corruptBuff.GetIntOrDefault("buff_turn_counter", 0))
	}
}

// ========== Lost Buff Handler Tests ==========

func TestLostBuffHandlerBehavior(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	mockState := &mockStepsModifier{steps: 3}

	ctx := event.NewContext(player)
	ctx.Set("current_state", mockState)

	handler := GetBuffHandlerConfig(constants.BuffTypeLost).Handler
	handler(constants.PhasePreMove, ctx)

	// Should signal reverse movement
	reverse, err := ctx.GetBool("reverse_movement")
	if err != nil {
		t.Error("reverse_movement should be set")
	}
	if !reverse {
		t.Error("reverse_movement should be true")
	}

	// Steps should be reversed
	if mockState.steps != -3 {
		t.Errorf("steps should be -3 (reversed), got %d", mockState.steps)
	}
}

func TestLostBuffHandlerOtherPhase(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	ctx := event.NewContext(player)

	handler := GetBuffHandlerConfig(constants.BuffTypeLost).Handler

	// Execute in BeforeTurn phase - should not trigger
	handler(constants.PhaseBeforeTurn, ctx)

	_, err := ctx.GetBool("reverse_movement")
	if err == nil {
		t.Error("reverse_movement should not be set in BeforeTurn phase")
	}
}

// ========== Hidden Buff Handler Tests ==========

func TestHiddenBuffHandlerBehavior(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	ctx := event.NewContext(player)

	handler := GetBuffHandlerConfig(constants.BuffTypeHidden).Handler
	handler(constants.PhasePreBuffApplied, ctx)

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
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	ctx := event.NewContext(player)

	handler := GetBuffHandlerConfig(constants.BuffTypeExorcism).Handler
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
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	ctx := event.NewContext(player)

	handler := GetBuffHandlerConfig(constants.BuffTypePoison).Handler
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

func TestCurseAndDivineDerivedActions(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxLP: 5})
	player.LP = 3
	game.AddPlayer(player)
	game.Log.StartTurn(1, 0, player.ID.UUID())

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)

	ctxDivine := event.NewContext(player)
	ctxDivine.Set("action_context", actionCtx)
	ctxDivine.Set("applied_buff_type", string(constants.BuffTypeDivine))

	ctxCurse := event.NewContext(player)
	ctxCurse.Set("action_context", actionCtx)
	ctxCurse.Set("applied_buff_type", string(constants.BuffTypeCurse))

	GetBuffHandlerConfig(constants.BuffTypeDivine).Handler(constants.PhasePostBuffApplied, ctxDivine)
	GetBuffHandlerConfig(constants.BuffTypeCurse).Handler(constants.PhasePostBuffApplied, ctxCurse)

	totalDerived := len(ctxDivine.GetDerivedActions()) + len(ctxCurse.GetDerivedActions())
	if totalDerived != 2 {
		t.Fatalf("total derived actions = %d, want 2", totalDerived)
	}

	for _, d := range ctxDivine.GetDerivedActions() {
		if act, ok := d.(engineaction.Action); ok {
			actionCtx.PushDerivedAction(act)
		}
	}
	for _, d := range ctxCurse.GetDerivedActions() {
		if act, ok := d.(engineaction.Action); ok {
			actionCtx.PushDerivedAction(act)
		}
	}
	actionCtx.ProcessQueue()

	if player.LP != 3 {
		t.Fatalf("LP = %d, want 3 (divine +1 and curse -1)", player.LP)
	}
}

func TestRainAndCorruptDerivedActions(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	player.HP = 5
	game.AddPlayer(player)
	game.Log.StartTurn(1, 0, player.ID.UUID())

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)

	ctxRain := event.NewContext(player)
	ctxRain.Set("action_context", actionCtx)
	rainHandler := GetBuffHandlerConfig(constants.BuffTypeRain).Handler
	rainHandler(constants.PhaseAfterTurn, ctxRain)
	rainHandler(constants.PhaseAfterTurn, ctxRain)

	ctxCorrupt := event.NewContext(player)
	ctxCorrupt.Set("action_context", actionCtx)
	corruptHandler := GetBuffHandlerConfig(constants.BuffTypeCorrupt).Handler
	corruptHandler(constants.PhaseAfterTurn, ctxCorrupt)
	corruptHandler(constants.PhaseAfterTurn, ctxCorrupt)

	for _, d := range ctxRain.GetDerivedActions() {
		if act, ok := d.(engineaction.Action); ok {
			actionCtx.PushDerivedAction(act)
		}
	}
	for _, d := range ctxCorrupt.GetDerivedActions() {
		if act, ok := d.(engineaction.Action); ok {
			actionCtx.PushDerivedAction(act)
		}
	}
	actionCtx.ProcessQueue()

	if player.HP != 5 {
		t.Fatalf("HP = %d, want 5 (rain +1 and corrupt -1)", player.HP)
	}
}

func TestLostBuffMutatesSteps(t *testing.T) {
	// Test 迷途 handler reverses Steps via StepsModifier interface
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	// Create a mock StepsModifier for testing (since TurnMovingState is in hsm package)
	mockState := &mockStepsModifier{steps: 4}

	ctx := event.NewContext(player)
	ctx.Set("current_state", mockState)

	handler := GetBuffHandlerConfig(constants.BuffTypeLost).Handler
	handler(constants.PhasePreMove, ctx)

	if mockState.steps != -4 {
		t.Fatalf("steps = %d, want -4 (reversed by 迷途)", mockState.steps)
	}

	// Test double-flip protection: Steps < 0 should NOT be flipped again
	mockState.steps = -3
	ctx2 := event.NewContext(player)
	ctx2.Set("current_state", mockState)
	handler(constants.PhasePreMove, ctx2)

	if mockState.steps != -3 {
		t.Fatalf("steps = %d, want -3 (double-flip protection, should not change)", mockState.steps)
	}
}

// mockStepsModifier implements StepsModifier for testing 迷途 handler.
type mockStepsModifier struct {
	steps int
}

func (m *mockStepsModifier) GetSteps() int {
	return m.steps
}

func (m *mockStepsModifier) SetSteps(steps int) {
	m.steps = steps
}

// ========== Edge Case Tests: nil player/context ==========

func TestHandlerWithNilContext(t *testing.T) {
	// All handlers should gracefully handle nil context (no panic)
	buffTypes := GetAllBuffTypes()
	for _, bt := range buffTypes {
		handler := GetBuffHandlerConfig(bt).Handler
		if handler == nil {
			continue
		}
		// Should not panic with nil context
		handler(constants.PhaseBeforeTurn, nil)
	}
}

func TestHandlerWithNilPlayer(t *testing.T) {
	// All handlers should gracefully handle nil player in context (no panic, no derived actions)
	buffTypes := GetAllBuffTypes()
	for _, bt := range buffTypes {
		handler := GetBuffHandlerConfig(bt).Handler
		if handler == nil {
			continue
		}
		ctx := event.NewContext(nil) // nil player
		handler(constants.PhaseBeforeTurn, ctx)
		// Should not produce derived actions
		if len(ctx.GetDerivedActions()) > 0 {
			t.Errorf("BuffType %s should not produce actions with nil player", bt)
		}
	}
}

func TestModifyLPHandlerNilActionContext(t *testing.T) {
	// Curse handler requires ActionContext for derived actions
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := event.NewContext(player)
	ctx.Set("applied_buff_type", string(constants.BuffTypeCurse))
	// No action_context set

	handler := GetBuffHandlerConfig(constants.BuffTypeCurse).Handler
	handler(constants.PhasePostBuffApplied, ctx)

	// Should not produce derived actions without ActionContext
	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("Curse handler should not produce actions without ActionContext")
	}
}

func TestModifyHPHandlerNilActionContext(t *testing.T) {
	// createModifyHPHandler requires ActionContext
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := event.NewContext(player)
	// No action_context set

	handler := GetBuffHandlerConfig(constants.BuffTypeRain).Handler
	handler(constants.PhaseAfterTurn, ctx)

	// Should not produce derived actions without ActionContext
	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("Rain handler should not produce actions without ActionContext")
	}
}

func TestFireHandlerNilPlayer(t *testing.T) {
	// Fire handler requires player for FireCounter
	ctx := event.NewContext(nil)
	handler := GetBuffHandlerConfig(constants.BuffTypeFire).Handler
	handler(constants.PhaseBeforeTurn, ctx)

	// Should not panic or produce actions
	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("Fire handler should not produce actions with nil player")
	}
}

func TestLostHandlerNilAction(t *testing.T) {
	// Lost handler requires current_action in context
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := event.NewContext(player)
	// No current_action set

	handler := GetBuffHandlerConfig(constants.BuffTypeLost).Handler
	handler(constants.PhasePreMove, ctx)

	// Should not panic
}

func TestHiddenHandlerNilAction(t *testing.T) {
	// Hidden handler requires current_action in context
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := event.NewContext(player)
	// No current_action set

	handler := GetBuffHandlerConfig(constants.BuffTypeHidden).Handler
	handler(constants.PhasePreBuffApplied, ctx)

	// Should not panic
}

func TestExorcismHandlerNilAction(t *testing.T) {
	// Exorcism handler requires current_action in context
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := event.NewContext(player)
	// No current_action set

	handler := GetBuffHandlerConfig(constants.BuffTypeExorcism).Handler
	handler(constants.PhasePreEvent, ctx)

	// Should not panic
}

func TestPoisonHandlerNilPlayer(t *testing.T) {
	// Poison handler sets flag for bad event
	ctx := event.NewContext(nil)
	handler := GetBuffHandlerConfig(constants.BuffTypePoison).Handler
	handler(constants.PhaseBeforeTurn, ctx)

	// Should not panic
}

func TestCurseHandlerWrongPhase(t *testing.T) {
	// Curse handler should only work on PhasePostBuffApplied and PhasePreBuffRemoved
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := event.NewContext(player)
	ctx.Set("applied_buff_type", string(constants.BuffTypeCurse))

	handler := GetBuffHandlerConfig(constants.BuffTypeCurse).Handler
	handler(constants.PhaseAfterTurn, ctx) // Wrong phase

	// Should not produce derived actions on wrong phase
	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("Curse handler should not produce actions on wrong phase")
	}
}

func TestDivineHandlerWrongPhase(t *testing.T) {
	// Divine handler should only work on PhasePostBuffApplied and PhasePreBuffRemoved
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := event.NewContext(player)
	ctx.Set("applied_buff_type", string(constants.BuffTypeDivine))

	handler := GetBuffHandlerConfig(constants.BuffTypeDivine).Handler
	handler(constants.PhaseAfterTurn, ctx) // Wrong phase

	// Should not produce derived actions on wrong phase
	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("Divine handler should not produce actions on wrong phase")
	}
}

// ========== DeathMark Handler Tests ==========

func TestDeathMarkBlockRespawnAction(t *testing.T) {
	// RespawnAction must NOT be blocked by DeathMark
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.IsDead = true
	ctx := event.NewContext(player)
	ctx.Set("current_action", engineaction.NewRespawnAction(player, 30, "BossAttackRespawn"))

	handler := GetBuffHandlerConfig(constants.BuffTypeDeathMark).Handler
	err := handler(constants.PhasePreAction, ctx)
	if err != nil {
		t.Fatalf("DeathMark handler should not return error for RespawnAction: %v", err)
	}

	// RespawnAction should NOT be blocked
	if ctx.GetBoolOrDefault("action_blocked", false) {
		t.Error("RespawnAction should not be blocked by DeathMark")
	}
}

func TestDeathMarkBlockRemoveDeathMarkAction(t *testing.T) {
	// RemoveBuffAction(DeathMark) must NOT be blocked (removing self should not block itself)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.IsDead = true
	ctx := event.NewContext(player)
	ctx.Set("current_action", engineaction.NewRemoveBuffAction(player, constants.BuffTypeDeathMark, "DeathMarkCleanup"))

	handler := GetBuffHandlerConfig(constants.BuffTypeDeathMark).Handler
	err := handler(constants.PhasePreAction, ctx)
	if err != nil {
		t.Fatalf("DeathMark handler should not return error for RemoveBuffAction(DeathMark): %v", err)
	}

	// RemoveBuffAction(DeathMark) should NOT be blocked
	if ctx.GetBoolOrDefault("action_blocked", false) {
		t.Error("RemoveBuffAction(DeathMark) should not be blocked by DeathMark")
	}
}

func TestDeathMarkBlockRemoveOtherBuffAction(t *testing.T) {
	// RemoveBuffAction for OTHER buffs should still be blocked
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.IsDead = true
	ctx := event.NewContext(player)
	ctx.Set("current_action", engineaction.NewRemoveBuffAction(player, constants.BuffTypeDivine, "Buff_Expiry"))

	handler := GetBuffHandlerConfig(constants.BuffTypeDeathMark).Handler
	err := handler(constants.PhasePreAction, ctx)
	if err != nil {
		t.Fatalf("DeathMark handler should not return error: %v", err)
	}

	// RemoveBuffAction(Divine) should be blocked
	if !ctx.GetBoolOrDefault("action_blocked", false) {
		t.Error("RemoveBuffAction(Divine) should be blocked by DeathMark")
	}
}

func TestDeathMarkBlockDamageAction(t *testing.T) {
	// DamageAction should be blocked by DeathMark
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.IsDead = true
	ctx := event.NewContext(player)
	ctx.Set("current_action", engineaction.NewDamageAction(player, 3, "TestKill"))

	handler := GetBuffHandlerConfig(constants.BuffTypeDeathMark).Handler
	err := handler(constants.PhasePreAction, ctx)
	if err != nil {
		t.Fatalf("DeathMark handler should not return error: %v", err)
	}

	if !ctx.GetBoolOrDefault("action_blocked", false) {
		t.Error("DamageAction should be blocked by DeathMark")
	}
}

func TestDeathMarkBlockNilActionContext(t *testing.T) {
	// When current_action is missing, block by default
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.IsDead = true
	ctx := event.NewContext(player)
	// No current_action set

	handler := GetBuffHandlerConfig(constants.BuffTypeDeathMark).Handler
	err := handler(constants.PhasePreAction, ctx)
	if err != nil {
		t.Fatalf("DeathMark handler should not return error: %v", err)
	}

	if !ctx.GetBoolOrDefault("action_blocked", false) {
		t.Error("Missing current_action should default to blocking")
	}
}

func TestDeathMarkBlockNonPreActionPhase(t *testing.T) {
	// DeathMark handler should not act on phases other than PhasePreAction
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := event.NewContext(player)
	ctx.Set("current_action", engineaction.NewDamageAction(player, 3, "Test"))

	handler := GetBuffHandlerConfig(constants.BuffTypeDeathMark).Handler
	err := handler(constants.PhaseBeforeTurn, ctx)
	if err != nil {
		t.Fatalf("DeathMark handler should not return error on wrong phase: %v", err)
	}

	// Should NOT block on wrong phase
	if ctx.GetBoolOrDefault("action_blocked", false) {
		t.Error("DeathMark should not block actions on non-PhasePreAction phases")
	}
}

// ========== Thorns Buff Handler Tests ==========

func TestThornsReflectHandler(t *testing.T) {
	// Thorns handler should push derived BossAttackAction for reflect damage
	// Thorns buff is on BossPlayer, ctx.Player = BossPlayer (PreDamage publishes to BossPlayer)
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	bossPlayer := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 50})
	bossPlayer.HP = 50

	game.AddPlayer(player)
	game.AddPlayer(bossPlayer)

	// Apply Thorns buff to BossPlayer (Boss gives itself Thorns)
	thornsBuff := core.NewBuff(constants.BuffTypeThorns, 2)
	game.ApplyBuffToPlayer(bossPlayer, thornsBuff)

	// Create BossDamageAction (player attacks Boss)
	bossDamageAction := engineaction.NewBossDamageAction(
		player, bossPlayer, 4, false, string(constants.SourceBossDamage),
	)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(bossPlayer) // ctx.Player = BossPlayer (PreDamage published to BossPlayer)
	ctx.Set("current_action", bossDamageAction)
	ctx.Set("action_context", actionCtx)

	handler := GetBuffHandlerConfig(constants.BuffTypeThorns).Handler
	handler(constants.PhasePreDamage, ctx)

	// Should produce one derived BossAttackAction (reflect 4*30%=1.2→rounded=1)
	derivedActions := ctx.GetDerivedActions()
	if len(derivedActions) != 1 {
		t.Fatalf("Expected 1 derived action, got %d", len(derivedActions))
	}

	reflectAction, ok := derivedActions[0].(*engineaction.BossAttackAction)
	if !ok {
		t.Fatalf("Derived action should be BossAttackAction, got %T", derivedActions[0])
	}

	if reflectAction.Damage != 1 {
		t.Errorf("Reflect damage = %d, expected 1 (4*0.3 rounded)", reflectAction.Damage)
	}

	if reflectAction.SourceID != string(constants.SourceThornsReflect) {
		t.Errorf("Reflect source = %s, expected %s", reflectAction.SourceID, string(constants.SourceThornsReflect))
	}
}

func TestThornsReflectDamageCalculation(t *testing.T) {
	// Test reflect damage rounding: 30% with math.Round
	// Thorns buff is on BossPlayer, ctx.Player = BossPlayer
	tests := []struct {
		damage        int
		expectedReflect int
	}{
		{1, 0},  // 1*0.3=0.3 → rounded=0 → no reflect
		{2, 1},  // 2*0.3=0.6 → rounded=1
		{4, 1},  // 4*0.3=1.2 → rounded=1
		{6, 2},  // 6*0.3=1.8 → rounded=2
		{10, 3}, // 10*0.3=3.0 → rounded=3
	}

	game := NewGame(id.NewGameID(), 0)
	bossPlayer := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 50})
	game.AddPlayer(bossPlayer)

	// Apply Thorns buff to BossPlayer
	thornsBuff := core.NewBuff(constants.BuffTypeThorns, 2)
	game.ApplyBuffToPlayer(bossPlayer, thornsBuff)

	handler := GetBuffHandlerConfig(constants.BuffTypeThorns).Handler

	for _, tt := range tests {
		t.Run(fmt.Sprintf("damage_%d", tt.damage), func(t *testing.T) {
			player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
			game.AddPlayer(player)

			bossDamageAction := engineaction.NewBossDamageAction(
				player, bossPlayer, tt.damage, false, string(constants.SourceBossDamage),
			)

			ctx := event.NewContext(bossPlayer) // ctx.Player = BossPlayer
			ctx.Set("current_action", bossDamageAction)

			handler(constants.PhasePreDamage, ctx)

			derivedActions := ctx.GetDerivedActions()
			if tt.expectedReflect == 0 {
				if len(derivedActions) != 0 {
					t.Errorf("damage=%d: expected 0 derived actions, got %d", tt.damage, len(derivedActions))
				}
			} else {
				if len(derivedActions) != 1 {
					t.Fatalf("damage=%d: expected 1 derived action, got %d", tt.damage, len(derivedActions))
				}
				reflectAction, ok := derivedActions[0].(*engineaction.BossAttackAction)
				if !ok {
					t.Fatalf("damage=%d: derived action should be BossAttackAction", tt.damage)
				}
				if reflectAction.Damage != tt.expectedReflect {
					t.Errorf("damage=%d: reflect damage = %d, expected %d", tt.damage, reflectAction.Damage, tt.expectedReflect)
				}
			}

			// Clean up player for next iteration
			game.RemovePlayer(player.ID)
		})
	}
}

func TestThornsReflectWrongPhase(t *testing.T) {
	// Thorns handler should only work on PhasePreDamage
	game := NewGame(id.NewGameID(), 0)
	bossPlayer := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 50})
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	game.AddPlayer(bossPlayer)
	game.AddPlayer(player)

	thornsBuff := core.NewBuff(constants.BuffTypeThorns, 2)
	game.ApplyBuffToPlayer(bossPlayer, thornsBuff)

	bossDamageAction := engineaction.NewBossDamageAction(
		player, bossPlayer, 4, false, string(constants.SourceBossDamage),
	)

	ctx := event.NewContext(bossPlayer)
	ctx.Set("current_action", bossDamageAction)

	handler := GetBuffHandlerConfig(constants.BuffTypeThorns).Handler
	handler(constants.PhaseBeforeTurn, ctx) // Wrong phase

	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("Thorns handler should not produce actions on wrong phase")
	}
}

func TestThornsReflectNoBossDamageAction(t *testing.T) {
	// Thorns handler should not produce actions without BossDamageAction in context
	bossPlayer := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 50})
	thornsBuff := core.NewBuff(constants.BuffTypeThorns, 2)

	game := NewGame(id.NewGameID(), 0)
	game.AddPlayer(bossPlayer)
	game.ApplyBuffToPlayer(bossPlayer, thornsBuff)

	ctx := event.NewContext(bossPlayer)
	// No current_action set

	handler := GetBuffHandlerConfig(constants.BuffTypeThorns).Handler
	handler(constants.PhasePreDamage, ctx)

	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("Thorns handler should not produce actions without BossDamageAction")
	}
}

func TestThornsReflectNoThornsBuff(t *testing.T) {
	// Thorns handler should not produce reflect if BossPlayer doesn't have Thorns buff
	bossPlayer := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 50})
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})

	game := NewGame(id.NewGameID(), 0)
	game.AddPlayer(bossPlayer)
	game.AddPlayer(player)

	// BossPlayer does NOT have Thorns buff
	bossDamageAction := engineaction.NewBossDamageAction(
		player, bossPlayer, 4, false, string(constants.SourceBossDamage),
	)

	ctx := event.NewContext(bossPlayer) // ctx.Player = BossPlayer
	ctx.Set("current_action", bossDamageAction)

	handler := GetBuffHandlerConfig(constants.BuffTypeThorns).Handler
	handler(constants.PhasePreDamage, ctx)

	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("Thorns handler should not produce reflect if BossPlayer doesn't have Thorns buff")
	}
}

func TestThornsReflectNotBossDamageAction(t *testing.T) {
	// Thorns handler should not produce actions when current_action is not BossDamageAction
	bossPlayer := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 50})
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})

	game := NewGame(id.NewGameID(), 0)
	game.AddPlayer(bossPlayer)
	game.AddPlayer(player)

	thornsBuff := core.NewBuff(constants.BuffTypeThorns, 2)
	game.ApplyBuffToPlayer(bossPlayer, thornsBuff)

	// Use DamageAction instead of BossDamageAction
	damageAction := engineaction.NewDamageAction(player, 4, "TestDamage")

	ctx := event.NewContext(bossPlayer)
	ctx.Set("current_action", damageAction) // Not a BossDamageAction

	handler := GetBuffHandlerConfig(constants.BuffTypeThorns).Handler
	handler(constants.PhasePreDamage, ctx)

	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("Thorns handler should not produce actions when current_action is not BossDamageAction")
	}
}

// ========== Hidden Immune IsBoss Bypass Tests ==========

func TestHiddenImmuneIsBossBypass(t *testing.T) {
	// IsBoss buffs (Thorns, DeathMark) should bypass Hidden immunity
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})

	tests := []struct {
		buffType    constants.BuffType
		shouldBlock bool
	}{
		{constants.BuffTypeThorns, false},    // IsBoss → bypass Hidden
		{constants.BuffTypeDeathMark, false},  // IsBoss → bypass Hidden
		{constants.BuffTypeCurse, true},       // Negative → blocked by Hidden
		{constants.BuffTypeDivine, false},     // Positive → bypass Hidden
	}

	handler := GetBuffHandlerConfig(constants.BuffTypeHidden).Handler

	for _, tt := range tests {
		t.Run(string(tt.buffType), func(t *testing.T) {
			ctx := event.NewContext(player)
			ctx.Set("applied_buff_type", string(tt.buffType))

			handler(constants.PhasePreBuffApplied, ctx)

			blocked := ctx.GetBoolOrDefault("action_blocked", false)
			if blocked != tt.shouldBlock {
				t.Errorf("BuffType(%s): action_blocked = %v, expected %v", tt.buffType, blocked, tt.shouldBlock)
			}
		})
	}
}

// ========== BuildBuffPool Tests ==========

func TestBuildBuffPool(t *testing.T) {
	// BuildBuffPool should produce an EvaluatedItem for every registered BuffDefinition
	// where IsDraw() is true (excludes DeathMark/Thorns which are Boss/Hidden)
	pool := BuildBuffPool()

	// Verify pool only contains drawable buffs (not Boss/Hidden)
	for _, item := range pool {
		bt := constants.ParseBuffType(item.Type)
		if !bt.IsDraw() {
			t.Errorf("pool entry Type=%s should have IsDraw=true but it doesn't", item.Type)
		}
	}

	// Verify Boss/Hidden buffs are excluded
	for _, bt := range GetAllBuffTypes() {
		if bt.IsBoss() || bt.IsHidden() {
			for _, item := range pool {
				if item.Type == string(bt) {
					t.Errorf("Boss/Hidden BuffType(%s) should not be in BuffPool", bt)
				}
			}
		}
	}

	// Verify each pool entry matches its Definition's Type and Eval
	for _, item := range pool {
		def := GetBuffDefinition(constants.ParseBuffType(item.Type))
		if def == nil {
			t.Errorf("pool entry Type=%s has no matching BuffDefinition", item.Type)
			continue
		}
		if item.Eval != def.Eval {
			t.Errorf("pool entry Type=%s Eval=%d, expected %d from Definition", item.Type, item.Eval, def.Eval)
		}
	}
}

func TestHasEventHandler(t *testing.T) {
	// Test HasEventHandler for known event types
	allEvents := GetAllEventTypes()
	for _, et := range allEvents {
		if !HasEventHandler(et) {
			t.Errorf("EventType(%s) should have handler", et)
		}
	}

	// Test HasEventHandler for non-existent event type
	if HasEventHandler(constants.EventTypeNone) {
		t.Error("EventTypeNone should not have handler")
	}
}