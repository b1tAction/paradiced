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
			// New buff handler configs
			{constants.BuffTypeSinking, constants.PhasePreAction, true, 60, false},
			{constants.BuffTypeSinking, constants.PhasePreBuffApplied, true, 60, false},
			{constants.BuffTypeEternal, constants.PhasePreAction, true, 60, false},
			{constants.BuffTypeEternal, constants.PhasePreBuffApplied, true, 60, false},
			{constants.BuffTypeFearless, constants.PhasePreAction, true, 200, false},
			{constants.BuffTypeFearless, constants.PhasePostBuffApplied, true, 200, false},
			{constants.BuffTypeGoldenBody, constants.PhasePreDamage, true, 70, false},
			{constants.BuffTypeWrath, constants.PhasePreAction, true, 60, false},
			{constants.BuffTypeSavior, constants.PhasePreDamage, true, 999, false},
			{constants.BuffTypeSageProtection, constants.PhasePreRespawn, true, 50, false},
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

	// Execute 3 times to trigger LP+1
	for i := 0; i < 3; i++ {
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
	if blockedBy != string(constants.SourceBuffHidden) {
		t.Errorf("blocked_by = %s, expected %s", blockedBy, string(constants.SourceBuffHidden))
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
	// Thorns handler should push derived PiercingDamageAction for reflect damage
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

	// Should produce one derived DamageAction (reflect 4*30%=1.2→rounded=1)
	derivedActions := ctx.GetDerivedActions()
	if len(derivedActions) != 1 {
		t.Fatalf("Expected 1 derived action, got %d", len(derivedActions))
	}

	reflectAction, ok := derivedActions[0].(*engineaction.DamageAction)
	if !ok {
		t.Fatalf("Derived action should be DamageAction, got %T", derivedActions[0])
	}

	if reflectAction.Amount != 1 {
		t.Errorf("Reflect damage = %d, expected 1 (4*0.3 rounded)", reflectAction.Amount)
	}
	if !reflectAction.IsPiercing {
		t.Error("Thorns reflect DamageAction should be piercing (bypasses PhasePreDamage interception)")
	}

	if reflectAction.SourceID != string(constants.SourceBuffThornsReflect) {
		t.Errorf("Reflect source = %s, expected %s", reflectAction.SourceID, string(constants.SourceBuffThornsReflect))
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
				reflectAction, ok := derivedActions[0].(*engineaction.DamageAction)
				if !ok {
					t.Fatalf("damage=%d: derived action should be DamageAction", tt.damage)
				}
				if reflectAction.Amount != tt.expectedReflect {
					t.Errorf("damage=%d: reflect damage = %d, expected %d", tt.damage, reflectAction.Amount, tt.expectedReflect)
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

// ========== createModifyLPHandler Tests ==========

func TestCreateModifyLPHandlerNilContext(t *testing.T) {
	handler := createModifyLPHandler(1, constants.SourceBuffDivine)
	err := handler(constants.PhaseBeforeTurn, nil)
	if err == nil {
		t.Error("createModifyLPHandler should return error for nil context")
	}
}

func TestCreateModifyLPHandlerNilPlayer(t *testing.T) {
	handler := createModifyLPHandler(1, constants.SourceBuffDivine)
	ctx := event.NewContext(nil)
	err := handler(constants.PhaseBeforeTurn, ctx)
	if err == nil {
		t.Error("createModifyLPHandler should return error for nil player")
	}
}

func TestCreateModifyLPHandlerWithActionContext(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	handler := createModifyLPHandler(1, constants.SourceBuffDivine)
	ctx := event.NewContext(player)

	// Set action_context so getActionCtxFromEventCtx succeeds
	actionCtx := engineaction.NewActionContext(nil, nil, nil, nil)
	ctx.Set("action_context", actionCtx)

	err := handler(constants.PhaseBeforeTurn, ctx)
	if err != nil {
		t.Errorf("createModifyLPHandler with valid context should not error: %v", err)
	}

	// Should have a derived action
	actions := ctx.GetDerivedActions()
	if len(actions) == 0 {
		t.Error("createModifyLPHandler should produce a derived action")
	}
}

// ========== Dominance Anti-Recursion Tests ==========

func TestDominanceAmplifySkipAlreadyAmplified(t *testing.T) {
	// Dominance handler should skip actions whose source is "faction_qing_long_dominance"
	// This prevents infinite loops where Dominance amplifies its own derived actions.
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: constants.FactionQingLong,
		MaxHP:   10,
		MaxLP:   5,
	})
	game.AddPlayer(player)

	// Apply Dominance buff to QingLong player
	dominanceBuff := core.NewBuff(constants.BuffTypeDominance, 1)
	game.ApplyBuffToPlayer(player, dominanceBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)

	// Create a BossDamageAction with source "faction_qing_long_dominance" (already amplified)
	amplifiedAction := engineaction.NewBossDamageAction(
		player, game.InitializeBoss(19), 4, false, string(constants.SourceFactionQingLongDominance),
	)
	ctx.Set("current_action", amplifiedAction)

	handler := GetBuffHandlerConfig(constants.BuffTypeDominance).Handler
	err := handler(constants.PhasePreAction, ctx)
	if err != nil {
		t.Fatalf("Handler should not return error for already-amplified action: %v", err)
	}

	// Should NOT produce any derived actions (skip check prevents amplification)
	if len(ctx.GetDerivedActions()) > 0 {
		t.Errorf("Dominance should skip already-amplified action, got %d derived actions", len(ctx.GetDerivedActions()))
	}
}

func TestDominanceAmplifyNormalAction(t *testing.T) {
	// Dominance handler should amplify normal BossDamageAction (not already amplified)
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: constants.FactionQingLong,
		MaxHP:   10,
		MaxLP:   5,
	})
	bossPlayer := game.InitializeBoss(19)
	game.AddPlayer(player)

	// Apply Dominance buff to QingLong player
	dominanceBuff := core.NewBuff(constants.BuffTypeDominance, 1)
	game.ApplyBuffToPlayer(player, dominanceBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)

	// Create a BossDamageAction with normal source
	bossDamageAction := engineaction.NewBossDamageAction(
		player, bossPlayer, 4, false, string(constants.SourceBossDamage),
	)
	ctx.Set("current_action", bossDamageAction)

	handler := GetBuffHandlerConfig(constants.BuffTypeDominance).Handler
	err := handler(constants.PhasePreAction, ctx)
	if err != nil {
		t.Fatalf("Handler should not return error: %v", err)
	}

	// Should produce one derived BossDamageAction (amplified)
	if len(ctx.GetDerivedActions()) != 1 {
		t.Fatalf("Dominance should amplify BossDamageAction, got %d derived actions", len(ctx.GetDerivedActions()))
	}

	derived, ok := ctx.GetDerivedActions()[0].(*engineaction.BossDamageAction)
	if !ok {
		t.Fatalf("Derived action should be BossDamageAction, got %T", ctx.GetDerivedActions()[0])
	}

	// Derived action source should be the Dominance source constant
	if derived.Source() != string(constants.SourceFactionQingLongDominance) {
		t.Errorf("Derived action source = %s, expected %s", derived.Source(), string(constants.SourceFactionQingLongDominance))
	}
}

// ========== RobLuck Anti-Recursion Tests ==========

func TestRobLuckRedirectSkipAlreadyRedirected(t *testing.T) {
	// RobLuck handler should skip actions whose source is "faction_bai_hu_rob_luck"
	// This prevents infinite loops from circular redirects (two BaiHu players targeting each other).
	game := NewGame(id.NewGameID(), 0)
	targetPlayer := core.NewPlayer(core.PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: constants.FactionQingLong,
		MaxHP:   10,
		MaxLP:   5,
	})
	baiHuPlayer := core.NewPlayer(core.PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: constants.FactionBaiHu,
		MaxHP:   10,
		MaxLP:   5,
	})
	game.AddPlayer(targetPlayer)
	game.AddPlayer(baiHuPlayer)

	// Apply RobLuck buff to target player, pointing to BaiHu player
	robLuckBuff := core.NewBuff(constants.BuffTypeRobLuck, 1)
	game.ApplyBuffToPlayer(targetPlayer, robLuckBuff)
	robLuckBuff.SetString("rob_luck_source_player", baiHuPlayer.ID.UUID())

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(targetPlayer)
	ctx.Set("action_context", actionCtx)

	// Create a HealAction with source "faction_bai_hu_rob_luck" (already redirected)
	redirectedHeal := engineaction.NewHealAction(baiHuPlayer, 3, string(constants.SourceFactionBaiHuRobLuck))
	ctx.Set("current_action", redirectedHeal)

	handler := GetBuffHandlerConfig(constants.BuffTypeRobLuck).Handler
	err := handler(constants.PhasePreAction, ctx)
	if err != nil {
		t.Fatalf("Handler should not return error for already-redirected action: %v", err)
	}

	// Should NOT block or produce derived actions (skip check prevents redirect)
	if ctx.GetBoolOrDefault("action_blocked", false) {
		t.Error("RobLuck should skip already-redirected action, should not block")
	}
	if len(ctx.GetDerivedActions()) > 0 {
		t.Errorf("RobLuck should skip already-redirected action, got %d derived actions", len(ctx.GetDerivedActions()))
	}
}

func TestRobLuckRedirectPreBuffAppliedSkipAlreadyRedirected(t *testing.T) {
	// RobLuck PreBuffApplied handler should also skip AddBuffActions
	// whose source is "faction_bai_hu_rob_luck"
	game := NewGame(id.NewGameID(), 0)
	targetPlayer := core.NewPlayer(core.PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: constants.FactionQingLong,
		MaxHP:   10,
		MaxLP:   5,
	})
	baiHuPlayer := core.NewPlayer(core.PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: constants.FactionBaiHu,
		MaxHP:   10,
		MaxLP:   5,
	})
	game.AddPlayer(targetPlayer)
	game.AddPlayer(baiHuPlayer)

	// Apply RobLuck buff to target player, pointing to BaiHu player
	robLuckBuff := core.NewBuff(constants.BuffTypeRobLuck, 1)
	game.ApplyBuffToPlayer(targetPlayer, robLuckBuff)
	robLuckBuff.SetString("rob_luck_source_player", baiHuPlayer.ID.UUID())

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(targetPlayer)
	ctx.Set("action_context", actionCtx)
	ctx.Set("applied_buff_type", string(constants.BuffTypeDivine))

	// Create an AddBuffAction with source "faction_bai_hu_rob_luck" (already redirected)
	redirectedBuff := engineaction.NewAddBuffAction(baiHuPlayer, constants.BuffTypeDivine, string(constants.SourceFactionBaiHuRobLuck))
	ctx.Set("current_action", redirectedBuff)

	handler := GetBuffHandlerConfig(constants.BuffTypeRobLuck).Handler
	err := handler(constants.PhasePreBuffApplied, ctx)
	if err != nil {
		t.Fatalf("Handler should not return error for already-redirected buff: %v", err)
	}

	// Should NOT block or produce derived actions (skip check prevents redirect)
	if ctx.GetBoolOrDefault("action_blocked", false) {
		t.Error("RobLuck should skip already-redirected buff, should not block")
	}
	if len(ctx.GetDerivedActions()) > 0 {
		t.Errorf("RobLuck should skip already-redirected buff, got %d derived actions", len(ctx.GetDerivedActions()))
	}
}

func TestRobLuckRedirectNormalAction(t *testing.T) {
	// RobLuck handler should redirect normal HealAction to BaiHu player
	game := NewGame(id.NewGameID(), 0)
	targetPlayer := core.NewPlayer(core.PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: constants.FactionQingLong,
		MaxHP:   10,
		MaxLP:   5,
	})
	baiHuPlayer := core.NewPlayer(core.PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: constants.FactionBaiHu,
		MaxHP:   10,
		MaxLP:   5,
	})
	game.AddPlayer(targetPlayer)
	game.AddPlayer(baiHuPlayer)

	// Apply RobLuck buff to target player, pointing to BaiHu player
	robLuckBuff := core.NewBuff(constants.BuffTypeRobLuck, 1)
	game.ApplyBuffToPlayer(targetPlayer, robLuckBuff)
	robLuckBuff.SetString("rob_luck_source_player", baiHuPlayer.ID.UUID())

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(targetPlayer)
	ctx.Set("action_context", actionCtx)

	// Create a normal HealAction targeting the RobLuck-buffed player
	healAction := engineaction.NewHealAction(targetPlayer, 3, "TestHeal")
	ctx.Set("current_action", healAction)

	handler := GetBuffHandlerConfig(constants.BuffTypeRobLuck).Handler
	err := handler(constants.PhasePreAction, ctx)
	if err != nil {
		t.Fatalf("Handler should not return error: %v", err)
	}

	// Should block original and produce one derived HealAction targeting BaiHu
	if !ctx.GetBoolOrDefault("action_blocked", false) {
		t.Error("RobLuck should block original HealAction")
	}
	if len(ctx.GetDerivedActions()) != 1 {
		t.Fatalf("RobLuck should redirect HealAction, got %d derived actions", len(ctx.GetDerivedActions()))
	}

	derived, ok := ctx.GetDerivedActions()[0].(*engineaction.HealAction)
	if !ok {
		t.Fatalf("Derived action should be HealAction, got %T", ctx.GetDerivedActions()[0])
	}

	// Derived action target should be BaiHu player
	if derived.TargetPlayer() != baiHuPlayer {
		t.Errorf("Derived HealAction should target BaiHu player, got %s", derived.Target())
	}

	// Derived action source should be the RobLuck source constant
	if derived.Source() != string(constants.SourceFactionBaiHuRobLuck) {
		t.Errorf("Derived action source = %s, expected %s", derived.Source(), string(constants.SourceFactionBaiHuRobLuck))
	}
}

// ========== Dominance BossDamage End-to-End Tests ==========

func TestDominanceAmplifyBossDamageViaExecuteAction(t *testing.T) {
	// Full end-to-end test: QingLong with Dominance attacks Boss via ExecuteAction,
	// Dominance should amplify BossDamageAction through PhasePreAction ActorPlayers.
	// Boss total damage = original × 2.

	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: constants.FactionQingLong,
		MaxHP:   10,
		MaxLP:   5,
	})
	game.AddPlayer(player)

	// Apply Dominance buff to QingLong player (subscribes to PhasePreAction)
	dominanceBuff := core.NewBuff(constants.BuffTypeDominance, 1)
	game.ApplyBuffToPlayer(player, dominanceBuff)

	// Initialize Boss player for target
	bossPlayer := game.InitializeBoss(19)
	bossHPBefore := bossPlayer.HP

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)

	// Execute BossDamageAction: QingLong attacks Boss with 5 damage
	action := engineaction.NewBossDamageAction(player, bossPlayer, 5, false, "boss_damage")

	err := actionCtx.ExecuteAction(action)
	if err != nil {
		t.Fatalf("ExecuteAction should not error: %v", err)
	}

	// Dominance should have amplified: original 5 damage + derived 5 damage = 10 total
	damageDealt := bossHPBefore - bossPlayer.HP
	if damageDealt != 10 {
		t.Errorf("Boss damage = %d, expected 10 (5 original + 5 Dominance amplified)", damageDealt)
	}
}
// ========== Sinking Buff Handler Tests ==========

func TestSinkingShareDamageAction(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player1)
	game.AddPlayer(player2)

	// Apply Sinking buff to player1, linked to player2
	sinkingBuff := core.NewBuff(constants.BuffTypeSinking, 2)
	sinkingBuff.Metadata.SetString("sinking_linked_player", player2.ID.UUID())
	game.ApplyBuffToPlayer(player1, sinkingBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player1)
	ctx.Set("action_context", actionCtx)
	ctx.Set("current_action", engineaction.NewDamageAction(player1, 3, "test_damage"))

	handler := GetBuffHandlerConfig(constants.BuffTypeSinking).Handler
	handler(constants.PhasePreAction, ctx)

	// Sinking shares but does NOT block the original action
	if ctx.GetBoolOrDefault("action_blocked", false) {
		t.Error("Sinking should NOT block original DamageAction (share semantics, unlike RobLuck)")
	}

	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived action, got %d", len(derived))
	}
	damageAction, ok := derived[0].(*engineaction.DamageAction)
	if !ok {
		t.Fatalf("expected DamageAction, got %T", derived[0])
	}
	if damageAction.TargetPlayer() != player2 {
		t.Errorf("derived DamageAction should target linked player, got %s", damageAction.Target())
	}
	if damageAction.Amount != 3 {
		t.Errorf("derived DamageAction amount = %d, expected 3", damageAction.Amount)
	}
	if damageAction.Source() != string(constants.SourceBuffSinking) {
		t.Errorf("derived source = %s, expected %s", damageAction.Source(), string(constants.SourceBuffSinking))
	}
}

func TestSinkingShareNegativeModifyLP(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxLP: 5})
	player2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxLP: 5})
	game.AddPlayer(player1)
	game.AddPlayer(player2)

	sinkingBuff := core.NewBuff(constants.BuffTypeSinking, 2)
	sinkingBuff.Metadata.SetString("sinking_linked_player", player2.ID.UUID())
	game.ApplyBuffToPlayer(player1, sinkingBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player1)
	ctx.Set("action_context", actionCtx)
	ctx.Set("current_action", engineaction.NewModifyLPAction(player1, -1, "test"))

	handler := GetBuffHandlerConfig(constants.BuffTypeSinking).Handler
	handler(constants.PhasePreAction, ctx)

	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived action, got %d", len(derived))
	}
	lpAction, ok := derived[0].(*engineaction.ModifyLPAction)
	if !ok {
		t.Fatalf("expected ModifyLPAction, got %T", derived[0])
	}
	if lpAction.TargetPlayer() != player2 {
		t.Error("derived ModifyLPAction should target linked player")
	}
	if lpAction.Amount != -1 {
		t.Errorf("derived amount = %d, expected -1", lpAction.Amount)
	}
}

func TestSinkingShareNegativeBuff(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player1)
	game.AddPlayer(player2)

	sinkingBuff := core.NewBuff(constants.BuffTypeSinking, 2)
	sinkingBuff.Metadata.SetString("sinking_linked_player", player2.ID.UUID())
	game.ApplyBuffToPlayer(player1, sinkingBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player1)
	ctx.Set("action_context", actionCtx)
	ctx.Set("applied_buff_type", string(constants.BuffTypeCurse))

	handler := GetBuffHandlerConfig(constants.BuffTypeSinking).Handler
	handler(constants.PhasePreBuffApplied, ctx)

	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived action, got %d", len(derived))
	}
	addBuff, ok := derived[0].(*engineaction.AddBuffAction)
	if !ok {
		t.Fatalf("expected AddBuffAction, got %T", derived[0])
	}
	if addBuff.BuffType != constants.BuffTypeCurse {
		t.Errorf("shared buff type = %s, expected curse", addBuff.BuffType)
	}
	if addBuff.TargetPlayer() != player2 {
		t.Error("shared buff should target linked player")
	}
}

func TestSinkingNotShareBossOrHiddenBuff(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player1)
	game.AddPlayer(player2)

	sinkingBuff := core.NewBuff(constants.BuffTypeSinking, 2)
	sinkingBuff.Metadata.SetString("sinking_linked_player", player2.ID.UUID())
	game.ApplyBuffToPlayer(player1, sinkingBuff)

	handler := GetBuffHandlerConfig(constants.BuffTypeSinking).Handler

	for _, bt := range []constants.BuffType{constants.BuffTypeThorns, constants.BuffTypeDeathMark} {
		ctx := event.NewContext(player1)
		ctx.Set("action_context", engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw))
		ctx.Set("applied_buff_type", string(bt))
		handler(constants.PhasePreBuffApplied, ctx)
		if len(ctx.GetDerivedActions()) > 0 {
			t.Errorf("Sinking should NOT share Boss/Hidden buff type %s", bt)
		}
	}
}

func TestSinkingSkipAlreadyShared(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player1)
	game.AddPlayer(player2)

	sinkingBuff := core.NewBuff(constants.BuffTypeSinking, 2)
	sinkingBuff.Metadata.SetString("sinking_linked_player", player2.ID.UUID())
	game.ApplyBuffToPlayer(player1, sinkingBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player1)
	ctx.Set("action_context", actionCtx)
	ctx.Set("current_action", engineaction.NewDamageAction(player2, 3, string(constants.SourceBuffSinking)))

	handler := GetBuffHandlerConfig(constants.BuffTypeSinking).Handler
	handler(constants.PhasePreAction, ctx)

	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("Sinking should skip already-shared action (source=buff_sinking)")
	}
}

func TestSinkingNoLinkedPlayer(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player1)

	sinkingBuff := core.NewBuff(constants.BuffTypeSinking, 2)
	game.ApplyBuffToPlayer(player1, sinkingBuff)
	// No linked_player metadata set

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player1)
	ctx.Set("action_context", actionCtx)
	ctx.Set("current_action", engineaction.NewDamageAction(player1, 3, "test"))

	handler := GetBuffHandlerConfig(constants.BuffTypeSinking).Handler
	handler(constants.PhasePreAction, ctx)

	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("Sinking should produce no actions without linked player")
	}
}

func TestSinkingNotSharePositiveActions(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player1)
	game.AddPlayer(player2)

	sinkingBuff := core.NewBuff(constants.BuffTypeSinking, 2)
	sinkingBuff.Metadata.SetString("sinking_linked_player", player2.ID.UUID())
	game.ApplyBuffToPlayer(player1, sinkingBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player1)
	ctx.Set("action_context", actionCtx)
	ctx.Set("current_action", engineaction.NewHealAction(player1, 3, "test"))

	handler := GetBuffHandlerConfig(constants.BuffTypeSinking).Handler
	handler(constants.PhasePreAction, ctx)

	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("Sinking should NOT share positive HealAction")
	}
}

// ========== Eternal Buff Handler Tests ==========

func TestEternalShareHealAction(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player1)
	game.AddPlayer(player2)

	eternalBuff := core.NewBuff(constants.BuffTypeEternal, 2)
	eternalBuff.Metadata.SetString("eternal_linked_player", player2.ID.UUID())
	game.ApplyBuffToPlayer(player1, eternalBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player1)
	ctx.Set("action_context", actionCtx)
	ctx.Set("current_action", engineaction.NewHealAction(player1, 3, "test_heal"))

	handler := GetBuffHandlerConfig(constants.BuffTypeEternal).Handler
	handler(constants.PhasePreAction, ctx)

	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived action, got %d", len(derived))
	}
	healAction, ok := derived[0].(*engineaction.HealAction)
	if !ok {
		t.Fatalf("expected HealAction, got %T", derived[0])
	}
	if healAction.TargetPlayer() != player2 {
		t.Error("derived HealAction should target linked player")
	}
	if healAction.Amount != 3 {
		t.Errorf("derived amount = %d, expected 3", healAction.Amount)
	}
	if healAction.Source() != string(constants.SourceBuffEternal) {
		t.Errorf("derived source = %s, expected %s", healAction.Source(), string(constants.SourceBuffEternal))
	}
}

func TestEternalSharePositiveModifyLP(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player1)
	game.AddPlayer(player2)

	eternalBuff := core.NewBuff(constants.BuffTypeEternal, 2)
	eternalBuff.Metadata.SetString("eternal_linked_player", player2.ID.UUID())
	game.ApplyBuffToPlayer(player1, eternalBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player1)
	ctx.Set("action_context", actionCtx)
	ctx.Set("current_action", engineaction.NewModifyLPAction(player1, 1, "test"))

	handler := GetBuffHandlerConfig(constants.BuffTypeEternal).Handler
	handler(constants.PhasePreAction, ctx)

	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived action, got %d", len(derived))
	}
	lpAction, ok := derived[0].(*engineaction.ModifyLPAction)
	if !ok {
		t.Fatalf("expected ModifyLPAction, got %T", derived[0])
	}
	if lpAction.TargetPlayer() != player2 {
		t.Error("derived ModifyLPAction should target linked player")
	}
	if lpAction.Amount != 1 {
		t.Errorf("derived amount = %d, expected 1", lpAction.Amount)
	}
}

func TestEternalShareAddItemAction(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player1)
	game.AddPlayer(player2)

	eternalBuff := core.NewBuff(constants.BuffTypeEternal, 2)
	eternalBuff.Metadata.SetString("eternal_linked_player", player2.ID.UUID())
	game.ApplyBuffToPlayer(player1, eternalBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player1)
	ctx.Set("action_context", actionCtx)
	ctx.Set("current_action", engineaction.NewAddItemAction(player1, constants.ItemTypeAnyDoor, "test"))

	handler := GetBuffHandlerConfig(constants.BuffTypeEternal).Handler
	handler(constants.PhasePreAction, ctx)

	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived AddItemAction, got %d", len(derived))
	}
	addItem, ok := derived[0].(*engineaction.AddItemAction)
	if !ok {
		t.Fatalf("expected AddItemAction, got %T", derived[0])
	}
	if addItem.TargetPlayer() != player2 {
		t.Error("derived AddItemAction should target linked player")
	}
	if addItem.ItemType != constants.ItemTypeAnyDoor {
		t.Errorf("derived ItemType = %s, expected %s", addItem.ItemType, constants.ItemTypeAnyDoor)
	}
	if addItem.Source() != string(constants.SourceBuffEternal) {
		t.Errorf("derived source = %s, expected %s", addItem.Source(), string(constants.SourceBuffEternal))
	}
}

func TestEternalSharePositiveBuff(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player1)
	game.AddPlayer(player2)

	eternalBuff := core.NewBuff(constants.BuffTypeEternal, 2)
	eternalBuff.Metadata.SetString("eternal_linked_player", player2.ID.UUID())
	game.ApplyBuffToPlayer(player1, eternalBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player1)
	ctx.Set("action_context", actionCtx)
	ctx.Set("applied_buff_type", string(constants.BuffTypeDivine))

	handler := GetBuffHandlerConfig(constants.BuffTypeEternal).Handler
	handler(constants.PhasePreBuffApplied, ctx)

	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived AddBuffAction, got %d", len(derived))
	}
	addBuff, ok := derived[0].(*engineaction.AddBuffAction)
	if !ok {
		t.Fatalf("expected AddBuffAction, got %T", derived[0])
	}
	if addBuff.BuffType != constants.BuffTypeDivine {
		t.Errorf("shared buff type = %s, expected divine", addBuff.BuffType)
	}
	if addBuff.TargetPlayer() != player2 {
		t.Error("shared buff should target linked player")
	}
}

func TestEternalNotShareFactionBossHiddenBuff(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player1)
	game.AddPlayer(player2)

	eternalBuff := core.NewBuff(constants.BuffTypeEternal, 2)
	eternalBuff.Metadata.SetString("eternal_linked_player", player2.ID.UUID())
	game.ApplyBuffToPlayer(player1, eternalBuff)

	handler := GetBuffHandlerConfig(constants.BuffTypeEternal).Handler
	for _, bt := range []constants.BuffType{constants.BuffTypeDominance, constants.BuffTypeThorns, constants.BuffTypeDeathMark} {
		ctx := event.NewContext(player1)
		ctx.Set("action_context", engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw))
		ctx.Set("applied_buff_type", string(bt))
		handler(constants.PhasePreBuffApplied, ctx)
		if len(ctx.GetDerivedActions()) > 0 {
			t.Errorf("Eternal should NOT share Faction/Boss/Hidden buff type %s", bt)
		}
	}
}

func TestEternalSkipAlreadyShared(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player1)
	game.AddPlayer(player2)

	eternalBuff := core.NewBuff(constants.BuffTypeEternal, 2)
	eternalBuff.Metadata.SetString("eternal_linked_player", player2.ID.UUID())
	game.ApplyBuffToPlayer(player1, eternalBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player1)
	ctx.Set("action_context", actionCtx)
	ctx.Set("current_action", engineaction.NewHealAction(player2, 3, string(constants.SourceBuffEternal)))

	handler := GetBuffHandlerConfig(constants.BuffTypeEternal).Handler
	handler(constants.PhasePreAction, ctx)

	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("Eternal should skip already-shared action (source=buff_eternal)")
	}
}

func TestEternalNoLinkedPlayer(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player1)

	eternalBuff := core.NewBuff(constants.BuffTypeEternal, 2)
	game.ApplyBuffToPlayer(player1, eternalBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player1)
	ctx.Set("action_context", actionCtx)
	ctx.Set("current_action", engineaction.NewHealAction(player1, 3, "test"))

	handler := GetBuffHandlerConfig(constants.BuffTypeEternal).Handler
	handler(constants.PhasePreAction, ctx)

	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("Eternal should produce no actions without linked player")
	}
}

func TestEternalNotShareNegativeActions(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player1 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player2 := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player1)
	game.AddPlayer(player2)

	eternalBuff := core.NewBuff(constants.BuffTypeEternal, 2)
	eternalBuff.Metadata.SetString("eternal_linked_player", player2.ID.UUID())
	game.ApplyBuffToPlayer(player1, eternalBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player1)
	ctx.Set("action_context", actionCtx)
	ctx.Set("current_action", engineaction.NewDamageAction(player1, 3, "test"))

	handler := GetBuffHandlerConfig(constants.BuffTypeEternal).Handler
	handler(constants.PhasePreAction, ctx)

	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("Eternal should NOT share negative DamageAction")
	}
}

// ========== Fearless Buff Handler Tests ==========

func TestFearlessBlockDamage(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	game.AddPlayer(player)

	fearlessBuff := core.NewBuff(constants.BuffTypeFearless, 3)
	game.ApplyBuffToPlayer(player, fearlessBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)
	ctx.Set("current_action", engineaction.NewDamageAction(player, 5, "test_damage"))

	handler := GetBuffHandlerConfig(constants.BuffTypeFearless).Handler
	handler(constants.PhasePreAction, ctx)

	if !ctx.GetBoolOrDefault("action_blocked", false) {
		t.Error("Fearless should block DamageAction")
	}
	blockedBy, _ := ctx.GetString("blocked_by")
	if blockedBy != string(constants.SourceBuffFearless) {
		t.Errorf("blocked_by = %s, expected %s", blockedBy, string(constants.SourceBuffFearless))
	}
}

func TestFearlessBlockHeal(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	game.AddPlayer(player)

	fearlessBuff := core.NewBuff(constants.BuffTypeFearless, 3)
	game.ApplyBuffToPlayer(player, fearlessBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)
	ctx.Set("current_action", engineaction.NewHealAction(player, 3, "test_heal"))

	handler := GetBuffHandlerConfig(constants.BuffTypeFearless).Handler
	handler(constants.PhasePreAction, ctx)

	if !ctx.GetBoolOrDefault("action_blocked", false) {
		t.Error("Fearless should block HealAction")
	}
}

func TestFearlessAllowSelfDamage(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	game.AddPlayer(player)

	fearlessBuff := core.NewBuff(constants.BuffTypeFearless, 3)
	game.ApplyBuffToPlayer(player, fearlessBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)
	ctx.Set("current_action", engineaction.NewPiercingDamageAction(player, 5, string(constants.SourceBuffFearless)))

	handler := GetBuffHandlerConfig(constants.BuffTypeFearless).Handler
	handler(constants.PhasePreAction, ctx)

	if ctx.GetBoolOrDefault("action_blocked", false) {
		t.Error("Fearless should NOT block its own HP-setting damage (source=buff_fearless)")
	}
}

func TestFearlessReduceHPTo1(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	player.HP = 6
	game.AddPlayer(player)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)
	ctx.Set("applied_buff_type", string(constants.BuffTypeFearless))

	handler := GetBuffHandlerConfig(constants.BuffTypeFearless).Handler
	handler(constants.PhasePostBuffApplied, ctx)

	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 PiercingDamageAction, got %d", len(derived))
	}
	damageAction, ok := derived[0].(*engineaction.DamageAction)
	if !ok {
		t.Fatalf("expected DamageAction, got %T", derived[0])
	}
	if !damageAction.IsPiercing {
		t.Error("Fearless HP reduction should be piercing (unblockable)")
	}
	if damageAction.Amount != 5 {
		t.Errorf("damage amount = %d, expected 5 (HP-1=6-1)", damageAction.Amount)
	}
	if damageAction.Source() != string(constants.SourceBuffFearless) {
		t.Errorf("source = %s, expected %s", damageAction.Source(), string(constants.SourceBuffFearless))
	}
}

func TestFearlessReduceHPAlreadyAtOne(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	player.HP = 1
	game.AddPlayer(player)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)
	ctx.Set("applied_buff_type", string(constants.BuffTypeFearless))

	handler := GetBuffHandlerConfig(constants.BuffTypeFearless).Handler
	handler(constants.PhasePostBuffApplied, ctx)

	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("Fearless should produce no derived PiercingDamageAction when HP already at 1")
	}
}

func TestFearlessWrongPhase(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := event.NewContext(player)

	handler := GetBuffHandlerConfig(constants.BuffTypeFearless).Handler
	handler(constants.PhaseBeforeTurn, ctx)

	if ctx.GetBoolOrDefault("action_blocked", false) {
		t.Error("Fearless should not block on wrong phase")
	}
}

func TestFearlessAllowRemoveBuffAction(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	game.AddPlayer(player)

	fearlessBuff := core.NewBuff(constants.BuffTypeFearless, 3)
	game.ApplyBuffToPlayer(player, fearlessBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)
	ctx.Set("current_action", engineaction.NewRemoveBuffAction(player, constants.BuffTypeFearless, "test"))

	handler := GetBuffHandlerConfig(constants.BuffTypeFearless).Handler
	handler(constants.PhasePreAction, ctx)

	if ctx.GetBoolOrDefault("action_blocked", false) {
		t.Error("Fearless should allow RemoveBuffAction(Fearless) through")
	}
}

// ========== GoldenBody Buff Handler Tests ==========

func TestGoldenBodyReduceDamage(t *testing.T) {
	tests := []struct {
		damage    int
		expected  int
	}{
		{1, 1},  // floor(1/2)+1 = 0+1 = 1
		{3, 2},  // floor(3/2)+1 = 1+1 = 2
		{5, 3},  // floor(5/2)+1 = 2+1 = 3
		{7, 4},  // floor(7/2)+1 = 3+1 = 4
	}

	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	game.AddPlayer(player)

	goldenBuff := core.NewBuff(constants.BuffTypeGoldenBody, 2)
	game.ApplyBuffToPlayer(player, goldenBuff)

	handler := GetBuffHandlerConfig(constants.BuffTypeGoldenBody).Handler

	for _, tt := range tests {
		t.Run(fmt.Sprintf("damage_%d", tt.damage), func(t *testing.T) {
			damageAction := engineaction.NewDamageAction(player, tt.damage, "test")
			ctx := event.NewContext(player)
			ctx.Set("current_action", damageAction)
			handler(constants.PhasePreDamage, ctx)
			if damageAction.Amount != tt.expected {
				t.Errorf("damage %d: reduced amount = %d, expected %d", tt.damage, damageAction.Amount, tt.expected)
			}
		})
	}
}

func TestGoldenBodySkipPiercing(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	game.AddPlayer(player)

	goldenBuff := core.NewBuff(constants.BuffTypeGoldenBody, 2)
	game.ApplyBuffToPlayer(player, goldenBuff)

	piercingAction := engineaction.NewPiercingDamageAction(player, 5, "test_piercing")
	ctx := event.NewContext(player)
	ctx.Set("current_action", piercingAction)

	handler := GetBuffHandlerConfig(constants.BuffTypeGoldenBody).Handler
	handler(constants.PhasePreDamage, ctx)

	if piercingAction.Amount != 5 {
		t.Errorf("piercing damage should not be reduced, got %d", piercingAction.Amount)
	}
}

func TestGoldenBodyNotBossDamageAction(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	bossPlayer := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 50})
	game.AddPlayer(player)
	game.AddPlayer(bossPlayer)

	goldenBuff := core.NewBuff(constants.BuffTypeGoldenBody, 2)
	game.ApplyBuffToPlayer(player, goldenBuff)

	bossDamageAction := engineaction.NewBossDamageAction(player, bossPlayer, 5, false, "test")
	ctx := event.NewContext(player)
	ctx.Set("current_action", bossDamageAction)

	handler := GetBuffHandlerConfig(constants.BuffTypeGoldenBody).Handler
	handler(constants.PhasePreDamage, ctx)

	// BossDamageAction should not be reduced (type assertion fails for *BossDamageAction)
	if bossDamageAction.Damage != 5 {
		t.Errorf("BossDamageAction should not be reduced by GoldenBody, got %d", bossDamageAction.Damage)
	}
}

func TestGoldenBodyWrongPhase(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := event.NewContext(player)
	ctx.Set("current_action", engineaction.NewDamageAction(player, 5, "test"))

	handler := GetBuffHandlerConfig(constants.BuffTypeGoldenBody).Handler
	handler(constants.PhaseBeforeTurn, ctx)

	// Should not modify action on wrong phase
}

// ========== Wrath Buff Handler Tests ==========

func TestWrathAmplifyOutgoingDamage(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	wrathPlayer := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	targetPlayer := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	game.AddPlayer(wrathPlayer)
	game.AddPlayer(targetPlayer)

	wrathBuff := core.NewBuff(constants.BuffTypeWrath, 2)
	game.ApplyBuffToPlayer(wrathPlayer, wrathBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(wrathPlayer)
	ctx.Set("action_context", actionCtx)
	ctx.Set("current_action", engineaction.NewDamageActionWithSource(targetPlayer, 3, wrathPlayer, "test"))

	handler := GetBuffHandlerConfig(constants.BuffTypeWrath).Handler
	handler(constants.PhasePreAction, ctx)

	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived action, got %d", len(derived))
	}
	damageAction, ok := derived[0].(*engineaction.DamageAction)
	if !ok {
		t.Fatalf("expected DamageAction, got %T", derived[0])
	}
	if damageAction.Amount != 1 {
		t.Errorf("derived damage = %d, expected 1", damageAction.Amount)
	}
	if damageAction.TargetPlayer() != targetPlayer {
		t.Error("derived damage should target same target")
	}
	if damageAction.Source() != string(constants.SourceBuffWrath) {
		t.Errorf("derived source = %s, expected %s", damageAction.Source(), string(constants.SourceBuffWrath))
	}
}

func TestWrathAmplifyOutgoingBossDamage(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	wrathPlayer := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	bossPlayer := game.InitializeBoss(19)
	game.AddPlayer(wrathPlayer)

	wrathBuff := core.NewBuff(constants.BuffTypeWrath, 2)
	game.ApplyBuffToPlayer(wrathPlayer, wrathBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(wrathPlayer)
	ctx.Set("action_context", actionCtx)
	ctx.Set("current_action", engineaction.NewBossDamageAction(wrathPlayer, bossPlayer, 4, false, "boss_damage"))

	handler := GetBuffHandlerConfig(constants.BuffTypeWrath).Handler
	handler(constants.PhasePreAction, ctx)

	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 derived action, got %d", len(derived))
	}
	bossDamage, ok := derived[0].(*engineaction.BossDamageAction)
	if !ok {
		t.Fatalf("expected BossDamageAction, got %T", derived[0])
	}
	if bossDamage.Damage != 1 {
		t.Errorf("derived boss damage = %d, expected 1", bossDamage.Damage)
	}
	if bossDamage.Source() != string(constants.SourceBuffWrath) {
		t.Errorf("derived source = %s, expected %s", bossDamage.Source(), string(constants.SourceBuffWrath))
	}
}

func TestWrathSkipAlreadyAmplified(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	wrathPlayer := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	targetPlayer := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	game.AddPlayer(wrathPlayer)
	game.AddPlayer(targetPlayer)

	wrathBuff := core.NewBuff(constants.BuffTypeWrath, 2)
	game.ApplyBuffToPlayer(wrathPlayer, wrathBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(wrathPlayer)
	ctx.Set("action_context", actionCtx)
	ctx.Set("current_action", engineaction.NewDamageAction(targetPlayer, 1, string(constants.SourceBuffWrath)))

	handler := GetBuffHandlerConfig(constants.BuffTypeWrath).Handler
	handler(constants.PhasePreAction, ctx)

	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("Wrath should skip already-amplified action (source=buff_wrath)")
	}
}

func TestWrathNotAmplifyNonSourcePlayer(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	wrathPlayer := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	otherPlayer := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	targetPlayer := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	game.AddPlayer(wrathPlayer)
	game.AddPlayer(otherPlayer)
	game.AddPlayer(targetPlayer)

	wrathBuff := core.NewBuff(constants.BuffTypeWrath, 2)
	game.ApplyBuffToPlayer(wrathPlayer, wrathBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(wrathPlayer)
	ctx.Set("action_context", actionCtx)
	ctx.Set("current_action", engineaction.NewDamageActionWithSource(targetPlayer, 3, otherPlayer, "test"))

	handler := GetBuffHandlerConfig(constants.BuffTypeWrath).Handler
	handler(constants.PhasePreAction, ctx)

	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("Wrath should NOT amplify damage where SourcePlayer != Wrath holder")
	}
}

func TestWrathWrongPhase(t *testing.T) {
	wrathPlayer := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := event.NewContext(wrathPlayer)

	handler := GetBuffHandlerConfig(constants.BuffTypeWrath).Handler
	handler(constants.PhaseBeforeTurn, ctx)

	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("Wrath should not amplify on wrong phase")
	}
}

// ========== Savior Buff Handler Tests ==========

func TestSaviorBlockFatalDamage(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	player.HP = 3
	game.AddPlayer(player)

	saviorBuff := core.NewBuff(constants.BuffTypeSavior, -1)
	game.ApplyBuffToPlayer(player, saviorBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)
	ctx.Set("current_action", engineaction.NewDamageAction(player, 5, "test_fatal"))

	handler := GetBuffHandlerConfig(constants.BuffTypeSavior).Handler
	handler(constants.PhasePreDamage, ctx)

	if !ctx.GetBoolOrDefault("action_blocked", false) {
		t.Error("Savior should block fatal damage (HP-5 <= 0)")
	}
	blockedBy, _ := ctx.GetString("blocked_by")
	if blockedBy != string(constants.SourceBuffSavior) {
		t.Errorf("blocked_by = %s, expected %s", blockedBy, string(constants.SourceBuffSavior))
	}

	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 RemoveBuffAction, got %d", len(derived))
	}
	removeBuff, ok := derived[0].(*engineaction.RemoveBuffAction)
	if !ok {
		t.Fatalf("expected RemoveBuffAction, got %T", derived[0])
	}
	if removeBuff.BuffType != constants.BuffTypeSavior {
		t.Errorf("remove buff type = %s, expected %s", removeBuff.BuffType, constants.BuffTypeSavior)
	}
}

func TestSaviorNotBlockNonFatalDamage(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	player.HP = 5
	game.AddPlayer(player)

	saviorBuff := core.NewBuff(constants.BuffTypeSavior, -1)
	game.ApplyBuffToPlayer(player, saviorBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)
	ctx.Set("current_action", engineaction.NewDamageAction(player, 3, "test_non_fatal"))

	handler := GetBuffHandlerConfig(constants.BuffTypeSavior).Handler
	handler(constants.PhasePreDamage, ctx)

	if ctx.GetBoolOrDefault("action_blocked", false) {
		t.Error("Savior should NOT block non-fatal damage (5-3 > 0)")
	}
	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("Savior should NOT remove itself for non-fatal damage")
	}
}

func TestSaviorWrongPhase(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := event.NewContext(player)

	handler := GetBuffHandlerConfig(constants.BuffTypeSavior).Handler
	handler(constants.PhaseBeforeTurn, ctx)

	if ctx.GetBoolOrDefault("action_blocked", false) {
		t.Error("Savior should not block on wrong phase")
	}
}

func TestSaviorNotBossDamageAction(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 10})
	player.HP = 1
	bossPlayer := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 50})
	game.AddPlayer(player)
	game.AddPlayer(bossPlayer)

	saviorBuff := core.NewBuff(constants.BuffTypeSavior, -1)
	game.ApplyBuffToPlayer(player, saviorBuff)

	bossDamageAction := engineaction.NewBossDamageAction(bossPlayer, player, 5, false, "test")
	ctx := event.NewContext(player)
	ctx.Set("current_action", bossDamageAction)

	handler := GetBuffHandlerConfig(constants.BuffTypeSavior).Handler
	handler(constants.PhasePreDamage, ctx)

	// BossDamageAction should not be blocked by Savior (type assertion fails)
	if ctx.GetBoolOrDefault("action_blocked", false) {
		t.Error("Savior should not block BossDamageAction (different type)")
	}
}

// ========== SageProtection Buff Handler Tests ==========

func TestSageProtectionRespawnInPlace(t *testing.T) {
	game := NewGame(id.NewGameID(), 0)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.Position = 15
	game.AddPlayer(player)

	sageProtBuff := core.NewBuff(constants.BuffTypeSageProtection, -1)
	game.ApplyBuffToPlayer(player, sageProtBuff)

	actionCtx := engineaction.NewActionContext(game, game.Bus, gamemap.NewMapEngine(20), game.Draw)
	ctx := event.NewContext(player)
	ctx.Set("action_context", actionCtx)
	ctx.Set("current_action", engineaction.NewRespawnAction(player, 30, "death_respawn"))

	handler := GetBuffHandlerConfig(constants.BuffTypeSageProtection).Handler
	handler(constants.PhasePreRespawn, ctx)

	raw, _ := ctx.Get("current_action"); respawnAction := raw.(*engineaction.RespawnAction)
	if respawnAction.CheckpointPos != 15 {
		t.Errorf("CheckpointPos = %d, expected 15 (player's death position)", respawnAction.CheckpointPos)
	}

	derived := ctx.GetDerivedActions()
	if len(derived) != 1 {
		t.Fatalf("expected 1 RemoveBuffAction, got %d", len(derived))
	}
	removeBuff, ok := derived[0].(*engineaction.RemoveBuffAction)
	if !ok {
		t.Fatalf("expected RemoveBuffAction, got %T", derived[0])
	}
	if removeBuff.BuffType != constants.BuffTypeSageProtection {
		t.Errorf("remove buff type = %s, expected %s", removeBuff.BuffType, constants.BuffTypeSageProtection)
	}
}

func TestSageProtectionNotRespawnAction(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := event.NewContext(player)
	ctx.Set("current_action", engineaction.NewDamageAction(player, 3, "test"))

	handler := GetBuffHandlerConfig(constants.BuffTypeSageProtection).Handler
	handler(constants.PhasePreRespawn, ctx)

	if len(ctx.GetDerivedActions()) > 0 {
		t.Error("SageProtection should not produce actions for non-RespawnAction")
	}
}

func TestSageProtectionWrongPhase(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	ctx := event.NewContext(player)
	ctx.Set("current_action", engineaction.NewRespawnAction(player, 30, "test"))

	handler := GetBuffHandlerConfig(constants.BuffTypeSageProtection).Handler
	handler(constants.PhaseBeforeTurn, ctx)

	// Should not modify RespawnAction on wrong phase
	raw, _ := ctx.Get("current_action"); respawnAction := raw.(*engineaction.RespawnAction)
	if respawnAction.CheckpointPos != 30 {
		t.Errorf("CheckpointPos should not be modified on wrong phase, got %d", respawnAction.CheckpointPos)
	}
}

// ========== IsItemOnly Classification Tests ==========

func TestBuffIsItemOnly(t *testing.T) {
	itemOnlyBuffs := []constants.BuffType{
		constants.BuffTypeSinking,
		constants.BuffTypeEternal,
		constants.BuffTypeSavior,
		constants.BuffTypeSageProtection,
	}
	for _, bt := range itemOnlyBuffs {
		if !bt.IsItemOnly() {
			t.Errorf("BuffType(%s) should be IsItemOnly=true", bt)
		}
		if bt.IsDraw() {
			t.Errorf("BuffType(%s) should be IsDraw=false (IsItemOnly buffs are not drawable)", bt)
		}
	}
}
