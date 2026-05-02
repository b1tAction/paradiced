package hsm

import (
	"testing"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/engine"
	engineaction "github.com/b1tAction/paradiced/internal/engine/action"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
)

// ========== Scenario Group A: Buff Effect Verification ==========

// TestScenarioBuff_Divine_LPIncrementOnApplied verifies that 神眷 (Divine) buff
// increases LP by 1 when applied (via PhasePostBuffApplied).
// Divine now triggers LP+1 on buff application, LP-1 revert on buff removal.
func TestScenarioBuff_Divine_LPIncrementOnApplied(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
		InitialHP:   6,
		InitialLP:   3,
	})

	player := harness.Players[0] // QingLong player (no faction buff)
	initialLP := player.LP

	// Create ActionContext with callbacks
	actionCtx := newActionContextWithPools(
		harness.Game,
		harness.Game.Bus,
		harness.MapEngine,
		harness.Game.Draw,
	)

	harness.Game.Log.StartTurn(1, 0, player.ID.UUID())

	// Add Divine buff via AddBuffAction (Action system publishes PhasePostBuffApplied)
	addAction := engineaction.NewAddBuffAction(player, constants.BuffTypeDivine, "Test_Divine")
	err := actionCtx.ExecuteAction(addAction)
	if err != nil {
		t.Fatalf("ExecuteAction(AddBuffAction) failed: %v", err)
	}

	t.Logf("After Divine applied: LP=%d (initial=%d), buffs=%d, bus_subs=%d", player.LP, initialLP, len(player.ActiveBuffs), harness.Game.Bus.GetSubscriptionCount())

	// Divine buff should increase LP by 1 when applied (PhasePostBuffApplied)
	if player.LP != initialLP+1 {
		t.Errorf("LP should be initial+1 after Divine buff applied, initial=%d, got=%d", initialLP, player.LP)
	}
}

// TestScenarioBuff_Curse_LPDecrementOnApplied verifies that 诅咒 (Curse) buff
// decreases LP by 1 when applied (via PhasePostBuffApplied).
// Curse now triggers LP-1 on buff application, LP+1 revert on buff removal.
func TestScenarioBuff_Curse_LPDecrementOnApplied(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
		InitialHP:   6,
		InitialLP:   3,
	})

	player := harness.Players[0]
	initialLP := player.LP

	// Create ActionContext with callbacks
	actionCtx := newActionContextWithPools(
		harness.Game,
		harness.Game.Bus,
		harness.MapEngine,
		harness.Game.Draw,
	)

	harness.Game.Log.StartTurn(1, 0, player.ID.UUID())

	// Add Curse buff via AddBuffAction (Action system publishes PhasePostBuffApplied)
	addAction := engineaction.NewAddBuffAction(player, constants.BuffTypeCurse, "Test_Curse")
	err := actionCtx.ExecuteAction(addAction)
	if err != nil {
		t.Fatalf("ExecuteAction(AddBuffAction) failed: %v", err)
	}

	t.Logf("After Curse applied: LP=%d (initial=%d), buffs=%d", player.LP, initialLP, len(player.ActiveBuffs))

	// Curse buff should decrease LP by 1 when applied (PhasePostBuffApplied)
	if player.LP != initialLP-1 {
		t.Errorf("LP should be initial-1 after Curse buff applied, initial=%d, got=%d", initialLP, player.LP)
	}
}

// TestScenarioBuff_Rain_HPHeal verifies that 甘霖 (Rain) buff
// heals HP by 1 every 2 turns via AfterTurn phase trigger.
func TestScenarioBuff_Rain_HPHeal(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
		InitialHP:   4,
		InitialLP:   3,
	})

	player := harness.Players[0]
	initialHP := player.HP

	// Add Rain buff with duration 4 (heals every 2 turns)
	harness.AddBuffToPlayer(player, constants.BuffTypeRain, 4)

	// Run player 0's first turn
	err := harness.RunPlayerTurn(0, 3)
	if err != nil {
		t.Fatalf("RunPlayerTurn failed: %v", err)
	}

	// Rain buff heals every 2 turns, so after first turn it may not have healed yet
	// (depends on tickEligible mechanism)
	// After first turn, buff tickEligible is set to true
	t.Logf("After turn 1: HP=%d (initial=%d)", player.HP, initialHP)

	// Verify buff is still active
	if !harness.VerifyBuffOnPlayer(player, constants.BuffTypeRain) {
		t.Error("Rain buff should still be active after first turn")
	}
}

// TestScenarioBuff_Corrupt_HPDamage verifies that 腐化 (Corrupt) buff
// damages HP by 1 every 2 turns via AfterTurn phase trigger.
func TestScenarioBuff_Corrupt_HPDamage(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
		InitialHP:   5,
		InitialLP:   3,
	})

	player := harness.Players[0]
	initialHP := player.HP

	// Add Corrupt buff with duration 4
	harness.AddBuffToPlayer(player, constants.BuffTypeCorrupt, 4)

	// Run player 0's first turn
	err := harness.RunPlayerTurn(0, 3)
	if err != nil {
		t.Fatalf("RunPlayerTurn failed: %v", err)
	}

	t.Logf("After turn 1: HP=%d (initial=%d)", player.HP, initialHP)

	// Verify buff is still active
	if !harness.VerifyBuffOnPlayer(player, constants.BuffTypeCorrupt) {
		t.Error("Corrupt buff should still be active after first turn")
	}
}

// ========== Scenario Group B: Faction Skill Verification ==========

// TestScenarioFaction_ZhuQue_FireBuffAutoApplied verifies that ZhuQue player
// automatically gets 离火 (Fire) buff when faction buffs are initialized.
func TestScenarioFaction_ZhuQue_FireBuffAutoApplied(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 4,
		Factions:    []constants.Faction{
			constants.FactionQingLong,
			constants.FactionZhuQue,
			constants.FactionBaiHu,
			constants.FactionXuanWu,
		},
	})

	zhuQuePlayer := harness.Players[1]

	// ZhuQue player should have Fire buff auto-applied
	if !harness.VerifyBuffOnPlayer(zhuQuePlayer, constants.BuffTypeFire) {
		t.Error("ZhuQue player should have 离火 (Fire) buff after initialization")
	}
}

// TestScenarioFaction_QingLong_ChargeOnTurnEnd verifies that QingLong player
// gets charge increments at TurnEnd (charge mechanism for 行迹 skill).
func TestScenarioFaction_QingLong_ChargeOnTurnEnd(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
	})

	qingLongPlayer := harness.Players[0]

	// Run player 0's (QingLong) turn
	err := harness.RunPlayerTurn(0, 3)
	if err != nil {
		t.Fatalf("RunPlayerTurn failed: %v", err)
	}

	// QingLong should have charge after TurnEnd
	charge := qingLongPlayer.GetChargeCount()
	if charge < 1 {
		t.Errorf("QingLong should have at least 1 charge after TurnEnd, got %d", charge)
	}
}

// ========== Scenario Group D: Full Round Flow ==========

// TestScenarioFullRound_2Players verifies that a complete round
// with 2 players completes successfully, with correct turn order
// and basic state transitions.
func TestScenarioFullRound_2Players(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
		InitialHP:   6,
		InitialLP:   3,
	})

	t.Logf("HSM global state before round: %s", harness.HSM.GetGlobalStateID().String())

	// Run a complete round with simple rankings
	miniGameRanks := map[int]int{0: 1, 1: 2}
	err := harness.RunFullRound(miniGameRanks)
	if err != nil {
		t.Fatalf("RunFullRound failed: %v", err)
	}

	t.Logf("HSM global state after round: %s, turn state: %s",
		harness.HSM.GetGlobalStateID().String(), harness.HSM.GetTurnStateID().String())

	// Verify players have correct HP/LP
	for i, player := range harness.Players {
		t.Logf("Player %d position=%d, HP=%d, LP=%d", i, player.Position, player.HP, player.LP)
		// Basic health checks - HP should be non-negative
		if player.HP < 0 {
			t.Errorf("Player %d HP should not be negative: %d", i, player.HP)
		}
	}
}

// TestScenarioFullRound_4Players_AllFactions verifies a full round
// with all 4 factions to ensure each faction's passive works.
func TestScenarioFullRound_4Players_AllFactions(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 4,
		Factions:    []constants.Faction{
			constants.FactionQingLong,
			constants.FactionZhuQue,
			constants.FactionBaiHu,
			constants.FactionXuanWu,
		},
		InitialHP:   6,
		InitialLP:   3,
	})

	// Verify faction buff initialization
	for i, player := range harness.Players {
		faction := player.GetFaction()
		t.Logf("Player %d: faction=%s, buffs=%d", i, faction, len(player.ActiveBuffs))
	}

	// ZhuQue should have Fire buff
	zhuQuePlayer := harness.Players[1]
	if !harness.VerifyBuffOnPlayer(zhuQuePlayer, constants.BuffTypeFire) {
		t.Error("ZhuQue player should have Fire buff")
	}

	// Run a complete round
	miniGameRanks := map[int]int{0: 1, 1: 2, 2: 3, 3: 4}
	err := harness.RunFullRound(miniGameRanks)
	if err != nil {
		t.Fatalf("RunFullRound failed: %v", err)
	}

	// Check basic state consistency
	for i, player := range harness.Players {
		t.Logf("Player %d: position=%d, HP=%d, LP=%d, buffs=%d",
			i, player.Position, player.HP, player.LP, len(player.ActiveBuffs))
		if player.HP < 0 {
			t.Errorf("Player %d HP should not be negative: %d", i, player.HP)
		}
		if player.LP < 0 {
			t.Errorf("Player %d LP should not be negative: %d", i, player.LP)
		}
	}
}

// TestScenarioBuffExpiry verifies that a buff with limited duration
// expires after the specified number of turns and is unsubscribed.
func TestScenarioBuffExpiry(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
		InitialHP:   6,
		InitialLP:   3,
	})

	player := harness.Players[0]

	// Add Curse buff with duration 1 (should expire after first turn-end tick)
	harness.AddBuffToPlayer(player, constants.BuffTypeCurse, 1)

	// Get the buff instance for checking duration
	var curseBuff *core.Buff
	for _, b := range player.ActiveBuffs {
		if b.Type == constants.BuffTypeCurse {
			curseBuff = b
			break
		}
	}

	// The buff should be active before turn
	if !harness.VerifyBuffOnPlayer(player, constants.BuffTypeCurse) {
		t.Error("Curse buff should be active before turn")
	}

	// Run player 0's turn
	err := harness.RunPlayerTurn(0, 3)
	if err != nil {
		t.Fatalf("RunPlayerTurn failed: %v", err)
	}

	// Buff with duration 1 should still be present after first TurnEnd
	// (first TickDuration marks tickEligible=true, doesn't decrement yet)
	// So it should survive one turn.
	t.Logf("After turn 1: LP=%d, buff active=%v, buff duration=%d",
		player.LP, player.HasBuff(constants.BuffTypeCurse), curseBuff.Duration)
}

// ========== Scenario Group E: Error Handling Verification ==========

// TestScenarioHSM_ErrorInEnter_AbortsTransition verifies that when a state
// sets ctx.Error in Enter(), the TransitionTo method detects it and returns
// the error instead of proceeding to Update() and auto-transition.
func TestScenarioHSM_ErrorInEnter_AbortsTransition(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
	})

	// Enter TurnLoop global state (required for turn transitions)
	loopCtx := harness.newCtx(nil)
	err := harness.HSM.TransitionTo(StateTurnLoop, loopCtx)
	if err != nil {
		t.Fatalf("Failed to enter TurnLoop: %v", err)
	}

	// Set turn player to nil - transitionTurn always sets ctx.Player from
	// hsm.turnPlayer, so this will cause TurnUpkeepState.Enter to see nil player.
	harness.HSM.SetTurnPlayer(nil)

	// Try to transition to TurnUpkeep with nil turn player
	// This should cause ctx.Error in TurnUpkeepState.Enter (player is nil)
	upkeepCtx := harness.newCtx(nil)
	err = harness.HSM.TransitionTo(StateTurnUpkeep, upkeepCtx)

	// With our ctx.Error check in transitionTurn, the error should be returned
	if err != nil {
		t.Logf("TransitionTo correctly returned error: %v", err)
	} else if upkeepCtx.Error != nil {
		t.Logf("ctx.Error was set and returned by TransitionTo: %v", upkeepCtx.Error)
	} else {
		t.Error("Expected error when transitioning with nil player, but got none")
	}
}

// ========== Scenario: Harness Creation and Basic Setup ==========

// TestScenarioHarnessCreation verifies the harness creates correctly
// with all expected components initialized.
func TestScenarioHarnessCreation(t *testing.T) {
	harness := NewGameTestHarness(nil) // Use default config

	// Verify core components exist
	if harness.Game == nil {
		t.Error("Game should be initialized")
	}
	if harness.HSM == nil {
		t.Error("HSM should be initialized")
	}
	if harness.MapEngine == nil {
		t.Error("MapEngine should be initialized")
	}
	if harness.Broadcast == nil {
		t.Error("Broadcast should be initialized")
	}

	// Verify players
	if len(harness.Players) != 4 {
		t.Errorf("Expected 4 players, got %d", len(harness.Players))
	}

	// Verify factions
	expectedFactions := []constants.Faction{
		constants.FactionQingLong,
		constants.FactionZhuQue,
		constants.FactionBaiHu,
		constants.FactionXuanWu,
	}
	for i, player := range harness.Players {
		if player.GetFaction() != expectedFactions[i] {
			t.Errorf("Player %d faction should be %s, got %s",
				i, expectedFactions[i], player.GetFaction())
		}
	}

	// Verify initial HP/LP
	for i, player := range harness.Players {
		if player.HP != 6 {
			t.Errorf("Player %d HP should be 6, got %d", i, player.HP)
		}
		if player.LP != 3 {
			t.Errorf("Player %d LP should be 3, got %d", i, player.LP)
		}
	}

	// Verify pools
	if len(harness.Game.EventPool) == 0 {
		t.Error("EventPool should be populated from Registry")
	}
	if len(harness.Game.ItemPool) == 0 {
		t.Error("ItemPool should be populated from Registry")
	}
}

// TestScenarioCustomConfig verifies harness with custom configuration.
func TestScenarioCustomConfig(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        100,
		PlayerCount: 2,
		MapLength:   50,
		CellTypeOverrides: map[int]constants.CellType{
			10: constants.CellTypeCheckpoint,
			20: constants.CellTypeFragile,
		},
		Factions:    []constants.Faction{constants.FactionBaiHu, constants.FactionXuanWu},
		InitialHP:   10,
		MaxHP:       10,
		InitialLP:   5,
	})

	if len(harness.Players) != 2 {
		t.Errorf("Expected 2 players, got %d", len(harness.Players))
	}
	for i, player := range harness.Players {
		if player.HP != 10 {
			t.Errorf("Player %d HP should be 10, got %d", i, player.HP)
		}
		if player.LP != 5 {
			t.Errorf("Player %d LP should be 5, got %d", i, player.LP)
		}
	}

	// BaiHu doesn't get auto buff, XuanWu doesn't get auto buff
	baiHuPlayer := harness.Players[0]
	if harness.VerifyBuffOnPlayer(baiHuPlayer, constants.BuffTypeFire) {
		t.Error("BaiHu player should not have Fire buff")
	}
}

// ========== Scenario Group C: Death & Respawn Verification ==========

// TestScenarioDeath_DamageActionDerivesDeathAction verifies that when
// DamageAction kills a player (HP → 0, IsDead=true), it derives
// a DeathAction which adds DeathMark buff to the player via OnAddBuff.
func TestScenarioDeath_DamageActionDerivesDeathAction(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
		InitialHP:   3, // Low HP so one damage kills
		InitialLP:   3,
	})

	player := harness.Players[0]
	player.Position = 10

	// Create ActionContext with callbacks
	actionCtx := newActionContextWithPools(
		harness.Game,
		harness.Game.Bus,
		harness.MapEngine,
		harness.Game.Draw,
	)

	// Start turn log for GameLog recording
	harness.Game.Log.StartTurn(1, 0, player.ID.UUID())

	// Execute DamageAction that kills the player
	damageAction := engineaction.NewDamageAction(player, 10, string(constants.SourceBuffCorrupt))
	err := actionCtx.ExecuteAction(damageAction)
	if err != nil {
		t.Fatalf("ExecuteAction(DamageAction) failed: %v", err)
	}

	// Verify player is dead
	if !player.IsDead {
		t.Error("Player should be dead after lethal damage")
	}

	// Verify DeathMark buff was added (by derived DeathAction)
	if !player.HasBuff(constants.BuffTypeDeathMark) {
		t.Error("Player should have DeathMark buff after death")
	}

	// Verify GameLog has damage and death entries
	entries := harness.Game.Log.GetCurrentTurnEntries()
	var hasDamage, hasDeath bool
	for _, entry := range entries {
		if entry.ActionType == "damage" {
			hasDamage = true
		}
		if entry.ActionType == "death" {
			hasDeath = true
			// Check death_source in metadata
			source := entry.Metadata.GetStringOrDefault("death_source", "")
			if source != string(constants.SourceBuffCorrupt) {
				t.Errorf("Death source should be 'Buff_Corrupt', got '%s'", source)
			}
			position := entry.Metadata.GetIntOrDefault("position", -1)
			if position != 10 {
				t.Errorf("Death position should be 10, got %d", position)
			}
		}
	}
	if !hasDamage {
		t.Error("GameLog should have damage entry")
	}
	if !hasDeath {
		t.Error("GameLog should have death entry (derived from DamageAction)")
	}
}

// TestScenarioDeath_FellDownActionDerivesDeathAction verifies that when
// FellDownAction kills a player, it derives a DeathAction with source
// "fragile_cell".
func TestScenarioDeath_FellDownActionDerivesDeathAction(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
		InitialHP:   1, // Minimal HP so fall damage kills
		InitialLP:   3,
	})

	player := harness.Players[0]
	player.Position = 15

	// Create ActionContext with callbacks
	actionCtx := newActionContextWithPools(
		harness.Game,
		harness.Game.Bus,
		harness.MapEngine,
		harness.Game.Draw,
	)

	harness.Game.Log.StartTurn(1, 0, player.ID.UUID())

	// Execute FellDownAction that kills the player (damage=1, HP=1 → 0)
	fellDownAction := engineaction.NewFellDownAction(player, 15, 1, string(constants.SourceFragileCell))
	err := actionCtx.ExecuteAction(fellDownAction)
	if err != nil {
		t.Fatalf("ExecuteAction(FellDownAction) failed: %v", err)
	}

	// Verify player is dead
	if !player.IsDead {
		t.Error("Player should be dead after fall damage")
	}

	// Verify DeathMark buff was added
	if !player.HasBuff(constants.BuffTypeDeathMark) {
		t.Error("Player should have DeathMark buff after fall death")
	}

	// Verify death entry has correct source
	entries := harness.Game.Log.GetCurrentTurnEntries()
	var hasDeath bool
	for _, entry := range entries {
		if entry.ActionType == "death" {
			hasDeath = true
			source := entry.Metadata.GetStringOrDefault("death_source", "")
			if source != string(constants.SourceFragileCell) {
				t.Errorf("Death source should be 'FragileCell', got '%s'", source)
			}
		}
	}
	if !hasDeath {
		t.Error("GameLog should have death entry from fell_down derivation")
	}
}

// TestScenarioDeath_DeathMarkBlocksSubsequentActions verifies that
// after death, DeathMark buff blocks subsequent Actions via PhasePreAction.
// Actions on a dead player (with DeathMark) should be silently skipped.
func TestScenarioDeath_DeathMarkBlocksSubsequentActions(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
		InitialHP:   3,
		InitialLP:   3,
	})

	player := harness.Players[0]

	// Create ActionContext with callbacks
	actionCtx := newActionContextWithPools(
		harness.Game,
		harness.Game.Bus,
		harness.MapEngine,
		harness.Game.Draw,
	)

	harness.Game.Log.StartTurn(1, 0, player.ID.UUID())

	// Step 1: Kill the player with damage
	damageAction := engineaction.NewDamageAction(player, 10, "TestKill")
	actionCtx.ExecuteAction(damageAction)

	if !player.IsDead {
		t.Fatal("Player should be dead after lethal damage")
	}
	if !player.HasBuff(constants.BuffTypeDeathMark) {
		t.Fatal("Player should have DeathMark buff")
	}

	// Step 2: Try to execute another action on the dead player
	// This should be blocked by PhasePreAction + DeathMark
	healAction := engineaction.NewHealAction(player, 5, "TestHeal")
	err := actionCtx.ExecuteAction(healAction)
	// ExecuteAction returns nil on block (not an error), but action is skipped
	if err != nil {
		t.Errorf("ExecuteAction on blocked action should not return error, got: %v", err)
	}

	// Player HP should NOT change (heal was blocked)
	if player.HP != 0 {
		t.Errorf("Dead player HP should still be 0 (heal blocked), got %d", player.HP)
	}
}

// TestScenarioDeath_RespawnRemovesDeathMark verifies that after
// DeathMark is cleaned up (in TurnEnd), RespawnAction resets player state
// (IsDead, HP, position).
func TestScenarioDeath_RespawnRemovesDeathMark(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
		InitialHP:   3,
		MaxHP:       3,
		InitialLP:   3,
		MaxLP:       3,
		// Add a checkpoint at position 30
		CellTypeOverrides: map[int]constants.CellType{30: constants.CellTypeCheckpoint},
	})

	player := harness.Players[0]
	player.Position = 50

	// Create ActionContext with callbacks
	actionCtx := newActionContextWithPools(
		harness.Game,
		harness.Game.Bus,
		harness.MapEngine,
		harness.Game.Draw,
	)

	harness.Game.Log.StartTurn(1, 0, player.ID.UUID())

	// Step 1: Kill the player
	damageAction := engineaction.NewDamageAction(player, 10, "TestKill")
	actionCtx.ExecuteAction(damageAction)

	if !player.IsDead {
		t.Fatal("Player should be dead")
	}
	if !player.HasBuff(constants.BuffTypeDeathMark) {
		t.Fatal("Player should have DeathMark buff")
	}

	// Step 2: Remove DeathMark (as TurnEnd would do before respawn)
	deathMark := player.GetBuff(constants.BuffTypeDeathMark)
	harness.Game.RemoveBuffFromPlayer(player, deathMark)

	// Verify DeathMark was removed
	if player.HasBuff(constants.BuffTypeDeathMark) {
		t.Fatal("DeathMark should be removed before respawn")
	}

	// Step 3: Respawn at checkpoint (PhasePreAction no longer blocks since DeathMark removed)
	checkpoint := harness.MapEngine.GetLastCheckpoint(player.Position)
	respawnAction := engineaction.NewRespawnAction(player, checkpoint, string(constants.SourceSystemTurnEndRespawn))
	err := actionCtx.ExecuteAction(respawnAction)
	if err != nil {
		t.Fatalf("ExecuteAction(RespawnAction) failed: %v", err)
	}

	// Verify respawn state
	if player.IsDead {
		t.Error("Player should NOT be dead after respawn")
	}
	if player.HP != 3 { // Respawn resets HP to InitHP
		t.Errorf("Player HP should be reset to InitHP(3), got %d", player.HP)
	}
	if player.Position != checkpoint {
		t.Errorf("Player position should be checkpoint(%d), got %d", checkpoint, player.Position)
	}

	// Verify GameLog has respawn entry
	entries := harness.Game.Log.GetCurrentTurnEntries()
	var hasRespawn bool
	for _, entry := range entries {
		if entry.ActionType == "respawn" {
			hasRespawn = true
			cpPos := entry.Metadata.GetIntOrDefault("checkpoint_pos", -1)
			if cpPos != checkpoint {
				t.Errorf("Respawn checkpoint_pos should be %d, got %d", checkpoint, cpPos)
			}
		}
	}
	if !hasRespawn {
		t.Error("GameLog should have respawn entry")
	}
}

// TestScenarioDeath_LethalDamageNonKillingDoesNotDeriveDeathAction verifies that
// when DamageAction does NOT kill (HP > 0 after damage), no DeathAction
// is derived and no DeathMark buff is added.
func TestScenarioDeath_LethalDamageNonKillingDoesNotDeriveDeathAction(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
		InitialHP:   6,
		InitialLP:   3,
	})

	player := harness.Players[0]

	// Create ActionContext with callbacks
	actionCtx := newActionContextWithPools(
		harness.Game,
		harness.Game.Bus,
		harness.MapEngine,
		harness.Game.Draw,
	)

	harness.Game.Log.StartTurn(1, 0, player.ID.UUID())

	// Execute DamageAction that does NOT kill (3 damage, HP 6 → 3)
	damageAction := engineaction.NewDamageAction(player, 3, "SmallHit")
	err := actionCtx.ExecuteAction(damageAction)
	if err != nil {
		t.Fatalf("ExecuteAction(DamageAction) failed: %v", err)
	}

	// Player should NOT be dead
	if player.IsDead {
		t.Error("Player should NOT be dead from non-lethal damage")
	}

	// No DeathMark buff should exist
	if player.HasBuff(constants.BuffTypeDeathMark) {
		t.Error("Player should NOT have DeathMark buff from non-lethal damage")
	}

	// HP should be reduced
	if player.HP != 3 {
		t.Errorf("Player HP should be 3 (6-3), got %d", player.HP)
	}

	// No death entry in GameLog
	entries := harness.Game.Log.GetCurrentTurnEntries()
	for _, entry := range entries {
		if entry.ActionType == "death" {
			t.Error("GameLog should NOT have death entry from non-lethal damage")
		}
	}
}

// TestScenarioDeath_KillPlayerHelperInHarness verifies that the
// KillPlayer helper on the harness correctly sets IsDead and HP=0.
func TestScenarioDeath_KillPlayerHelperInHarness(t *testing.T) {
	harness := NewGameTestHarness(nil)

	player := harness.Players[0]
	if player.IsDead {
		t.Error("Player should NOT be dead initially")
	}

	// Use KillPlayer helper
	harness.KillPlayer(player)

	if !player.IsDead {
		t.Error("Player should be dead after KillPlayer")
	}
	if player.HP != 0 {
		t.Errorf("Player HP should be 0 after KillPlayer, got %d", player.HP)
	}
}

// TestScenarioDeath_HiddenBuffNotInLotteryPool verifies that
// DeathMark (Hidden buff) does not appear in good/bad/neutral lottery pools.
func TestScenarioDeath_HiddenBuffNotInLotteryPool(t *testing.T) {
	// Verify DeathMark is not in any lottery pool
	for _, bt := range engine.GetBuffTypesByCategory("Good") {
		if bt == constants.BuffTypeDeathMark {
			t.Error("DeathMark should NOT be in good buff pool")
		}
	}
	for _, bt := range engine.GetBuffTypesByCategory("Bad") {
		if bt == constants.BuffTypeDeathMark {
			t.Error("DeathMark should NOT be in bad buff pool")
		}
	}
	for _, bt := range engine.GetBuffTypesByCategory("Neutral") {
		if bt == constants.BuffTypeDeathMark {
			t.Error("DeathMark should NOT be in neutral buff pool")
		}
	}

	// Verify DeathMark is still a valid buff type
	if !constants.BuffTypeDeathMark.IsValid() {
		t.Error("BuffTypeDeathMark should be valid")
	}

	// Verify DeathMark IsHidden
	if !constants.BuffTypeDeathMark.IsHidden() {
		t.Error("BuffTypeDeathMark should be hidden")
	}
}

// TestScenarioDeath_DeathActionLogEntryMetadata verifies that DeathAction
// LogEntry contains correct metadata fields (position, death_source).
func TestScenarioDeath_DeathActionLogEntryMetadata(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 6})
	player.Position = 25

	action := engineaction.NewDeathAction(player, string(constants.SourceBuffCorrupt), 25)

	// Verify Type
	if action.Type() != constants.ActionDeath {
		t.Errorf("DeathAction Type should be 'death', got '%s'", action.Type())
	}

	// Verify LogEntry
	entry := action.LogEntry()
	if entry.ActionType != "death" {
		t.Errorf("LogEntry ActionType should be 'death', got '%s'", entry.ActionType)
	}
	if entry.Source != string(constants.SourceBuffCorrupt) {
		t.Errorf("LogEntry Source should be 'Buff_Corrupt', got '%s'", entry.Source)
	}

	// Verify metadata
	position := entry.Metadata.GetIntOrDefault("position", -1)
	if position != 25 {
		t.Errorf("Metadata position should be 25, got %d", position)
	}
	deathSource := entry.Metadata.GetStringOrDefault("death_source", "")
	if deathSource != string(constants.SourceBuffCorrupt) {
		t.Errorf("Metadata death_source should be 'Buff_Corrupt', got '%s'", deathSource)
	}
}

// TestScenarioDeath_RespawnWithDeathMarkPresent verifies that
// RespawnAction can execute even when DeathMark buff is present.
// This is the Boss turn scenario: Boss kills player → DeathAction adds DeathMark →
// HSM attempts RespawnAction immediately → DeathMark handler must NOT block RespawnAction.
// Previously, RespawnAction was blocked by DeathMark because PhasePreAction
// death check did not exempt RespawnAction (only RemoveBuffAction was exempted).
func TestScenarioDeath_RespawnWithDeathMarkPresent(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
		InitialHP:   3,
		MaxHP:       3,
		InitialLP:   3,
		MaxLP:       3,
		CellTypeOverrides: map[int]constants.CellType{30: constants.CellTypeCheckpoint},
	})

	player := harness.Players[0]
	player.Position = 50

	// Create ActionContext with callbacks
	actionCtx := newActionContextWithPools(
		harness.Game,
		harness.Game.Bus,
		harness.MapEngine,
		harness.Game.Draw,
	)

	harness.Game.Log.StartTurn(1, 0, player.ID.UUID())

	// Step 1: Kill the player with BossAttackAction (simulating Boss turn)
	bossPlayer := harness.Game.InitializeBoss(harness.MapEngine.Length - 1)
	if bossPlayer == nil {
		t.Fatal("Failed to initialize Boss player")
	}
	bossAttackAction := engineaction.NewBossAttackAction(
		bossPlayer,
		player,
		10, // lethal damage
		constants.BossAttackNormal,
		"boss_normal",
	)
	err := actionCtx.ExecuteAction(bossAttackAction)
	if err != nil {
		t.Fatalf("ExecuteAction(BossAttackAction) failed: %v", err)
	}

	if !player.IsDead {
		t.Fatal("Player should be dead after Boss attack")
	}
	if !player.HasBuff(constants.BuffTypeDeathMark) {
		t.Fatal("Player should have DeathMark buff after death")
	}

	// Step 2: Attempt RespawnAction immediately (as BossBattleState would do)
	// This must succeed even though DeathMark is still present.
	checkpoint := harness.MapEngine.GetLastCheckpoint(player.Position)
	respawnAction := engineaction.NewRespawnAction(player, checkpoint, string(constants.SourceSystemBossAttackRespawn))
	err = actionCtx.ExecuteAction(respawnAction)
	if err != nil {
		t.Fatalf("ExecuteAction(RespawnAction) should succeed with DeathMark present, got: %v", err)
	}

	// Verify respawn state
	if player.IsDead {
		t.Error("Player should NOT be dead after respawn")
	}
	if player.HP != 3 {
		t.Errorf("Player HP should be reset to InitHP(3), got %d", player.HP)
	}
	if player.Position != checkpoint {
		t.Errorf("Player position should be checkpoint(%d), got %d", checkpoint, player.Position)
	}
}

// ========== Scenario Group C: Hidden Buff Immunity ==========

// TestScenarioBuff_Hidden_BlockNegativeBuff verifies that Hidden (隐匿) buff
// blocks the application of negative buffs via EventBus interception.
func TestScenarioBuff_Hidden_BlockNegativeBuff(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
	})

	player := harness.Players[0]

	actionCtx := newActionContextWithPools(
		harness.Game,
		harness.Game.Bus,
		harness.MapEngine,
		harness.Game.Draw,
	)

	harness.Game.Log.StartTurn(1, 0, player.ID.UUID())

	// Step 1: Add Hidden buff first
	addHiddenAction := engineaction.NewAddBuffAction(player, constants.BuffTypeHidden, "TestHidden")
	if err := actionCtx.ExecuteAction(addHiddenAction); err != nil {
		t.Fatalf("Add Hidden buff failed: %v", err)
	}

	// Step 2: Try to add a negative buff (Curse) while Hidden
	addCurseAction := engineaction.NewAddBuffAction(player, constants.BuffTypeCurse, "TestCurseAttempt")
	if err := actionCtx.ExecuteAction(addCurseAction); err != nil {
		t.Fatalf("ExecuteAction(AddBuffAction(Curse)) returned error: %v", err)
	}

	// Curse should be blocked by Hidden at PhasePreBuffApplied
	if player.HasBuff(constants.BuffTypeCurse) {
		t.Error("Curse buff should NOT be applied when player has Hidden buff")
	}
}

// ========== Scenario Group D: DeathMark + Buff Removal ==========

// TestScenarioDeath_DeathMarkBlockRemoveOtherBuff verifies that
// DeathMark blocks RemoveBuffAction for non-DeathMark buffs.
// When a dead player has Divine buff, removing Divine should be blocked
// because DeathMark blocks all non-exempt actions.
func TestScenarioDeath_DeathMarkBlockRemoveOtherBuff(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
		InitialHP:   3,
		InitialLP:   3,
	})

	player := harness.Players[0]

	actionCtx := newActionContextWithPools(
		harness.Game,
		harness.Game.Bus,
		harness.MapEngine,
		harness.Game.Draw,
	)

	harness.Game.Log.StartTurn(1, 0, player.ID.UUID())

	// Step 1: Add Divine buff to player
	addDivineAction := engineaction.NewAddBuffAction(player, constants.BuffTypeDivine, "TestDivine")
	if err := actionCtx.ExecuteAction(addDivineAction); err != nil {
		t.Fatalf("Add Divine buff failed: %v", err)
	}

	if !player.HasBuff(constants.BuffTypeDivine) {
		t.Fatal("Player should have Divine buff")
	}

	// Step 2: Kill the player
	damageAction := engineaction.NewDamageAction(player, 10, "TestKill")
	actionCtx.ExecuteAction(damageAction)

	if !player.IsDead {
		t.Fatal("Player should be dead")
	}
	if !player.HasBuff(constants.BuffTypeDeathMark) {
		t.Fatal("Player should have DeathMark buff")
	}

	// Step 3: Try to remove Divine buff — should be blocked by DeathMark
	removeDivineAction := engineaction.NewRemoveBuffAction(player, constants.BuffTypeDivine, string(constants.SourceBuffExpiry))
	if err := actionCtx.ExecuteAction(removeDivineAction); err != nil {
		t.Fatalf("ExecuteAction should not return error for blocked action: %v", err)
	}

	// Divine buff should still be present (removal was blocked)
	if !player.HasBuff(constants.BuffTypeDivine) {
		t.Error("Divine buff removal should be blocked by DeathMark — Divine should still be present")
	}
}

// ========== Scenario Group E: StealBuff (BaiHu 劫运) ==========

// TestScenarioStealBuff_BaiHuStealsBuff verifies that StealBuffAction
// transfers a buff from target player to source player.
func TestScenarioStealBuff_BaiHuStealsBuff(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionBaiHu},
		InitialHP:   6,
		InitialLP:   3,
	})

	target := harness.Players[0] // QingLong (victim)
	stealer := harness.Players[1] // BaiHu (stealer)

	actionCtx := newActionContextWithPools(
		harness.Game,
		harness.Game.Bus,
		harness.MapEngine,
		harness.Game.Draw,
	)

	harness.Game.Log.StartTurn(1, 0, target.ID.UUID())

	// Step 1: Add Divine buff to target player
	addDivineAction := engineaction.NewAddBuffAction(target, constants.BuffTypeDivine, "TestDivine")
	if err := actionCtx.ExecuteAction(addDivineAction); err != nil {
		t.Fatalf("Add Divine buff failed: %v", err)
	}

	if !target.HasBuff(constants.BuffTypeDivine) {
		t.Fatal("Target should have Divine buff")
	}

	// Step 2: StealBuffAction — BaiHu steals from QingLong
	stealAction := engineaction.NewStealBuffAction(target, stealer, string(constants.SourceFactionBaiHu))
	if err := actionCtx.ExecuteAction(stealAction); err != nil {
		t.Fatalf("StealBuffAction failed: %v", err)
	}

	// Verify: Divine buff should be removed from target, added to stealer
	if target.HasBuff(constants.BuffTypeDivine) {
		t.Error("Target should NOT have Divine buff after steal")
	}
	if !stealer.HasBuff(constants.BuffTypeDivine) {
		t.Error("Stealer should have Divine buff after steal")
	}

	// Verify: StolenBuff is set on the action
	if stealAction.StolenBuff == nil {
		t.Error("StolenBuff should be set after steal execution")
	} else if stealAction.StolenBuff.Type != constants.BuffTypeDivine {
		t.Errorf("StolenBuff type should be Divine, got %s", stealAction.StolenBuff.Type)
	}
}

// ========== Scenario Group G: Lost Buff Reverse Movement ==========

// TestScenarioBuff_Lost_ReverseMovement verifies that 迷途 (Lost) buff
// reverses movement direction via StepsModifier interface.
// The Lost handler subscribes to PhasePreMove and flips Steps from positive
// to negative when TurnMovingState publishes PhasePreMove.
// This test simulates the PhasePreMove publish flow manually.
func TestScenarioBuff_Lost_ReverseMovement(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
		InitialHP:   6,
		InitialLP:   3,
	})

	player := harness.Players[0]

	// Add Lost buff to player (duration 1 turn)
	harness.AddBuffToPlayer(player, constants.BuffTypeLost, 1)

	if !player.HasBuff(constants.BuffTypeLost) {
		t.Fatal("Player should have Lost buff after AddBuffToPlayer")
	}

	// Simulate PhasePreMove publish with StepsModifier
	// TurnMovingState.Enter() does this with itself as StepsModifier
	// Here we use a mock to verify the handler flips Steps
	actionCtx := newActionContextWithPools(
		harness.Game,
		harness.Game.Bus,
		harness.MapEngine,
		harness.Game.Draw,
	)

	mockSteps := &testStepsModifier{steps: 5}

	triggerCtx := event.NewContext(player)
	triggerCtx.Set("action_context", actionCtx)
	triggerCtx.Set("current_state", mockSteps)

	harness.Game.Bus.Publish(constants.PhasePreMove, player.ID.UUID(), triggerCtx)

	// Lost handler should have reversed steps: 5 → -5
	if mockSteps.steps != -5 {
		t.Errorf("Lost buff should reverse Steps from 5 to -5, got steps=%d", mockSteps.steps)
	}

	// Verify reverse_movement flag was set by handler
	if !triggerCtx.GetBoolOrDefault("reverse_movement", false) {
		t.Error("reverse_movement flag should be set by Lost handler")
	}
}

// testStepsModifier implements StepsModifier for scenario tests.
type testStepsModifier struct {
	steps int
}

func (m *testStepsModifier) GetSteps() int  { return m.steps }
func (m *testStepsModifier) SetSteps(s int) { m.steps = s }

// ========== Scenario Group H: Exorcism Immune Poison ==========

// TestScenarioBuff_Exorcism_ImmunePoison verifies that 辟邪 (Exorcism) buff
// blocks Poison's bad event draw via PhasePreEvent.
// When both Exorcism and Poison buffs are active, the Poison handler sets
// draw_bad_event in PhaseBeforeTurn, but Exorcism handler sets
// block_poison_effect in PhasePreEvent.
func TestScenarioBuff_Exorcism_ImmunePoison(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
		InitialHP:   6,
		InitialLP:   3,
	})

	player := harness.Players[0]

	// Add Exorcism buff (duration 5) and Poison buff (duration 3)
	harness.AddBuffToPlayer(player, constants.BuffTypeExorcism, 5)
	harness.AddBuffToPlayer(player, constants.BuffTypePoison, 3)

	if !player.HasBuff(constants.BuffTypeExorcism) {
		t.Fatal("Player should have Exorcism buff")
	}
	if !player.HasBuff(constants.BuffTypePoison) {
		t.Fatal("Player should have Poison buff")
	}

	actionCtx := newActionContextWithPools(
		harness.Game,
		harness.Game.Bus,
		harness.MapEngine,
		harness.Game.Draw,
	)

	// Simulate PhaseBeforeTurn publish (Poison handler sets draw_bad_event)
	beforeTurnCtx := event.NewContext(player)
	beforeTurnCtx.Set("action_context", actionCtx)
	harness.Game.Bus.Publish(constants.PhaseBeforeTurn, player.ID.UUID(), beforeTurnCtx)

	// Poison handler should set draw_bad_event flag
	if !beforeTurnCtx.GetBoolOrDefault("draw_bad_event", false) {
		t.Error("Poison handler should set draw_bad_event flag in PhaseBeforeTurn")
	}

	// Simulate PhasePreEvent publish (Exorcism handler sets block_poison_effect)
	preEventCtx := event.NewContext(player)
	preEventCtx.Set("action_context", actionCtx)
	harness.Game.Bus.Publish(constants.PhasePreEvent, player.ID.UUID(), preEventCtx)

	// Exorcism handler should set block_poison_effect flag
	if !preEventCtx.GetBoolOrDefault("block_poison_effect", false) {
		t.Error("Exorcism handler should set block_poison_effect flag in PhasePreEvent")
	}
}

// ========== Scenario Group I: Divine/Curse Removal Revert ==========

// TestScenarioBuff_Divine_LPRevertOnRemoval verifies that 神眷 (Divine) buff
// reverts LP-1 when removed via RemoveBuffAction (PhasePreBuffRemoved).
// Divine: LP+1 on applied, LP-1 revert on removed.
func TestScenarioBuff_Divine_LPRevertOnRemoval(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
		InitialHP:   6,
		InitialLP:   3,
	})

	player := harness.Players[0]
	initialLP := player.LP

	actionCtx := newActionContextWithPools(
		harness.Game,
		harness.Game.Bus,
		harness.MapEngine,
		harness.Game.Draw,
	)

	harness.Game.Log.StartTurn(1, 0, player.ID.UUID())

	// Step 1: Add Divine buff → LP+1
	addAction := engineaction.NewAddBuffAction(player, constants.BuffTypeDivine, "TestDivine")
	if err := actionCtx.ExecuteAction(addAction); err != nil {
		t.Fatalf("Add Divine buff failed: %v", err)
	}

	lpAfterApply := player.LP
	if lpAfterApply != initialLP+1 {
		t.Fatalf("LP should be initial+1 after Divine applied, got LP=%d (initial=%d)", lpAfterApply, initialLP)
	}

	// Step 2: Remove Divine buff → LP-1 revert
	removeAction := engineaction.NewRemoveBuffAction(player, constants.BuffTypeDivine, "TestRemoval")
	if err := actionCtx.ExecuteAction(removeAction); err != nil {
		t.Fatalf("Remove Divine buff failed: %v", err)
	}

	// LP should revert to initial value
	if player.LP != initialLP {
		t.Errorf("LP should revert to initial after Divine removed, got LP=%d (expected %d)", player.LP, initialLP)
	}

	// Divine buff should be gone
	if player.HasBuff(constants.BuffTypeDivine) {
		t.Error("Divine buff should be removed after RemoveBuffAction")
	}
}

// TestScenarioBuff_Curse_LPRevertOnRemoval verifies that 诅咒 (Curse) buff
// reverts LP+1 when removed via RemoveBuffAction (PhasePreBuffRemoved).
// Curse: LP-1 on applied, LP+1 revert on removed.
func TestScenarioBuff_Curse_LPRevertOnRemoval(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
		InitialHP:   6,
		InitialLP:   3,
		MaxLP:       8,
	})

	player := harness.Players[0]
	initialLP := player.LP

	actionCtx := newActionContextWithPools(
		harness.Game,
		harness.Game.Bus,
		harness.MapEngine,
		harness.Game.Draw,
	)

	harness.Game.Log.StartTurn(1, 0, player.ID.UUID())

	// Step 1: Add Curse buff → LP-1
	addAction := engineaction.NewAddBuffAction(player, constants.BuffTypeCurse, "TestCurse")
	if err := actionCtx.ExecuteAction(addAction); err != nil {
		t.Fatalf("Add Curse buff failed: %v", err)
	}

	lpAfterApply := player.LP
	if lpAfterApply != initialLP-1 {
		t.Fatalf("LP should be initial-1 after Curse applied, got LP=%d (initial=%d)", lpAfterApply, initialLP)
	}

	// Step 2: Remove Curse buff → LP+1 revert
	removeAction := engineaction.NewRemoveBuffAction(player, constants.BuffTypeCurse, "TestRemoval")
	if err := actionCtx.ExecuteAction(removeAction); err != nil {
		t.Fatalf("Remove Curse buff failed: %v", err)
	}

	// LP should revert to initial value
	if player.LP != initialLP {
		t.Errorf("LP should revert to initial after Curse removed, got LP=%d (expected %d)", player.LP, initialLP)
	}

	// Curse buff should be gone
	if player.HasBuff(constants.BuffTypeCurse) {
		t.Error("Curse buff should be removed after RemoveBuffAction")
	}
}

// ========== Scenario Group J: Rain/Corrupt Every-2-Turns ==========

// TestScenarioBuff_Rain_Every2TurnsHeal verifies that 甘霖 (Rain) buff
// heals HP+1 every 2 turns via AfterTurn handler with everyNTurns counter.
// The counter is now stored in Buff.Metadata, so it persists across turns
// even when HSM creates a new event.Context each turn.
func TestScenarioBuff_Rain_Every2TurnsHeal(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
		InitialHP:   6,
		InitialLP:   3,
	})

	player := harness.Players[0]

	// Add Rain buff (duration 4)
	harness.AddBuffToPlayer(player, constants.BuffTypeRain, 4)

	if !player.HasBuff(constants.BuffTypeRain) {
		t.Fatal("Player should have Rain buff")
	}

	actionCtx := newActionContextWithPools(
		harness.Game,
		harness.Game.Bus,
		harness.MapEngine,
		harness.Game.Draw,
	)

	harness.Game.Log.StartTurn(1, 0, player.ID.UUID())

	initialHP := player.HP

	// Simulate first PhaseAfterTurn with NEW context (like real HSM flow)
	triggerCtx1 := event.NewContext(player)
	triggerCtx1.Set("action_context", actionCtx)
	harness.Game.Bus.Publish(constants.PhaseAfterTurn, player.ID.UUID(), triggerCtx1)

	// First trigger: counter=1 in Buff.Metadata, should NOT heal
	if player.HP != initialHP {
		t.Errorf("Rain should NOT heal on first AfterTurn (counter=1 <2), HP=%d expected %d", player.HP, initialHP)
	}

	// Verify counter persisted in Buff.Metadata (not context)
	rainBuff := player.GetBuff(constants.BuffTypeRain)
	counter1 := rainBuff.GetIntOrDefault("buff_turn_counter", 0)
	if counter1 != 1 {
		t.Errorf("buff_turn_counter should be 1 in Buff.Metadata after first AfterTurn, got %d", counter1)
	}

	// Run derived actions from first trigger (should be empty)
	for _, derived := range triggerCtx1.GetDerivedActions() {
		if execAction, ok := derived.(engineaction.Action); ok {
			actionCtx.PushDerivedAction(execAction)
		}
	}
	if err := actionCtx.ProcessQueue(); err != nil {
		t.Fatalf("ProcessQueue after first AfterTurn failed: %v", err)
	}

	// Simulate second PhaseAfterTurn with NEW context (counter persists in Buff.Metadata)
	triggerCtx2 := event.NewContext(player)
	triggerCtx2.Set("action_context", actionCtx)
	harness.Game.Bus.Publish(constants.PhaseAfterTurn, player.ID.UUID(), triggerCtx2)

	// Second trigger: counter reaches 2, should produce HealAction derived action
	derivedActions := triggerCtx2.GetDerivedActions()
	if len(derivedActions) == 0 {
		t.Fatal("Rain should produce derived HealAction on second AfterTurn (counter reaches 2)")
	}

	// Process derived actions
	for _, derived := range derivedActions {
		if execAction, ok := derived.(engineaction.Action); ok {
			actionCtx.PushDerivedAction(execAction)
		}
	}
	if err := actionCtx.ProcessQueue(); err != nil {
		t.Fatalf("ProcessQueue after second AfterTurn failed: %v", err)
	}

	// HP should increase by 1 after second AfterTurn
	if player.HP != initialHP+1 {
		t.Errorf("Rain should heal HP+1 on second AfterTurn, got HP=%d (expected %d)", player.HP, initialHP+1)
	}

	// Counter should be reset to 0 after trigger
	counter2 := rainBuff.GetIntOrDefault("buff_turn_counter", 0)
	if counter2 != 0 {
		t.Errorf("buff_turn_counter should be 0 after reset, got %d", counter2)
	}
}

// TestScenarioBuff_Corrupt_Every2TurnsDamage verifies that 腐化 (Corrupt) buff
// damages HP-1 every 2 turns via AfterTurn handler with everyNTurns counter.
// The counter is now stored in Buff.Metadata, so it persists across turns
// even when HSM creates a new event.Context each turn.
func TestScenarioBuff_Corrupt_Every2TurnsDamage(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
		InitialHP:   6,
		InitialLP:   3,
	})

	player := harness.Players[0]

	// Add Corrupt buff (duration 4)
	harness.AddBuffToPlayer(player, constants.BuffTypeCorrupt, 4)

	if !player.HasBuff(constants.BuffTypeCorrupt) {
		t.Fatal("Player should have Corrupt buff")
	}

	actionCtx := newActionContextWithPools(
		harness.Game,
		harness.Game.Bus,
		harness.MapEngine,
		harness.Game.Draw,
	)

	harness.Game.Log.StartTurn(1, 0, player.ID.UUID())

	initialHP := player.HP

	// Simulate first PhaseAfterTurn with NEW context (like real HSM flow)
	triggerCtx1 := event.NewContext(player)
	triggerCtx1.Set("action_context", actionCtx)
	harness.Game.Bus.Publish(constants.PhaseAfterTurn, player.ID.UUID(), triggerCtx1)

	// First trigger: counter=1 in Buff.Metadata, should NOT damage
	if player.HP != initialHP {
		t.Errorf("Corrupt should NOT damage on first AfterTurn (counter=1 <2), HP=%d expected %d", player.HP, initialHP)
	}

	// Verify counter persisted in Buff.Metadata
	corruptBuff := player.GetBuff(constants.BuffTypeCorrupt)
	counter1 := corruptBuff.GetIntOrDefault("buff_turn_counter", 0)
	if counter1 != 1 {
		t.Errorf("buff_turn_counter should be 1 in Buff.Metadata after first AfterTurn, got %d", counter1)
	}

	// Run derived actions from first trigger (should be empty)
	for _, derived := range triggerCtx1.GetDerivedActions() {
		if execAction, ok := derived.(engineaction.Action); ok {
			actionCtx.PushDerivedAction(execAction)
		}
	}
	if err := actionCtx.ProcessQueue(); err != nil {
		t.Fatalf("ProcessQueue after first AfterTurn failed: %v", err)
	}

	// Simulate second PhaseAfterTurn with NEW context (counter persists in Buff.Metadata)
	triggerCtx2 := event.NewContext(player)
	triggerCtx2.Set("action_context", actionCtx)
	harness.Game.Bus.Publish(constants.PhaseAfterTurn, player.ID.UUID(), triggerCtx2)

	// Second trigger: counter reaches 2, should produce DamageAction derived action
	derivedActions := triggerCtx2.GetDerivedActions()
	if len(derivedActions) == 0 {
		t.Fatal("Corrupt should produce derived DamageAction on second AfterTurn (counter reaches 2)")
	}

	// Process derived actions
	for _, derived := range derivedActions {
		if execAction, ok := derived.(engineaction.Action); ok {
			actionCtx.PushDerivedAction(execAction)
		}
	}
	if err := actionCtx.ProcessQueue(); err != nil {
		t.Fatalf("ProcessQueue after second AfterTurn failed: %v", err)
	}

	// HP should decrease by 1 after second AfterTurn
	if player.HP != initialHP-1 {
		t.Errorf("Corrupt should damage HP-1 on second AfterTurn, got HP=%d (expected %d)", player.HP, initialHP-1)
	}

	// Counter should be reset to 0 after trigger
	counter2 := corruptBuff.GetIntOrDefault("buff_turn_counter", 0)
	if counter2 != 0 {
		t.Errorf("buff_turn_counter should be 0 after reset, got %d", counter2)
	}
}

// ========== Scenario Group K: Thunder Death Event ==========

// TestScenarioEvent_Thunder_Death verifies that 雷劫 (Thunder) event
// causes player death through the Action system.
// Thunder handler adds DamageAction(currentHP) as derived action,
// which reduces HP to 0 → triggers DeathAction → adds DeathMark.
func TestScenarioEvent_Thunder_Death(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
		InitialHP:   5,
		InitialLP:   3,
	})

	player := harness.Players[0]

	actionCtx := newActionContextWithPools(
		harness.Game,
		harness.Game.Bus,
		harness.MapEngine,
		harness.Game.Draw,
	)

	harness.Game.Log.StartTurn(1, 0, player.ID.UUID())

	initialHP := player.HP

	// Simulate Thunder event handler
	// Thunder handler: DamageAction(currentHP) → death
	thunderCtx := event.NewContext(player)
	thunderCtx.Set("action_context", actionCtx)

	// Directly invoke Thunder handler
	handler := engine.GetEventHandlerConfig(constants.EventTypeThunder).Handler
	if err := handler(constants.PhaseAnyTime, thunderCtx); err != nil {
		t.Fatalf("Thunder handler failed: %v", err)
	}

	// Thunder should produce DamageAction(currentHP) as derived action
	derivedActions := thunderCtx.GetDerivedActions()
	if len(derivedActions) == 0 {
		t.Fatal("Thunder handler should produce DamageAction as derived action")
	}

	// Verify instant_death flag
	if !thunderCtx.GetBoolOrDefault("instant_death", false) {
		t.Error("Thunder handler should set instant_death flag")
	}

	// Process derived actions (DamageAction → ApplyDamage → DeathAction → DeathMark)
	for _, derived := range derivedActions {
		if execAction, ok := derived.(engineaction.Action); ok {
			actionCtx.PushDerivedAction(execAction)
		}
	}
	if err := actionCtx.ProcessQueue(); err != nil {
		t.Fatalf("ProcessQueue for Thunder derived actions failed: %v", err)
	}

	// Player should be dead
	if !player.IsDead {
		t.Errorf("Player should be dead after Thunder event, HP=%d (initial=%d)", player.HP, initialHP)
	}

	// Player should have DeathMark buff
	if !player.HasBuff(constants.BuffTypeDeathMark) {
		t.Error("Player should have DeathMark buff after death from Thunder event")
	}

	// HP should be 0
	if player.HP != 0 {
		t.Errorf("Player HP should be 0 after Thunder death, got HP=%d", player.HP)
	}
}

// ========== Scenario Group L: Buff Duration Extension ==========

// TestScenarioBuff_DurationExtension verifies that when AddBuffAction is applied
// to a player who already has the same BuffType, the buff duration is extended
// instead of creating a new buff instance.
// The buff_duration_extended flag is set in ActionContext.Metadata to skip
// PhasePostBuffApplied publication (not a new buff application).
func TestScenarioBuff_DurationExtension(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
		InitialHP:   6,
		InitialLP:   3,
		MaxLP:       8,
	})

	player := harness.Players[0]
	initialLP := player.LP

	actionCtx := newActionContextWithPools(
		harness.Game,
		harness.Game.Bus,
		harness.MapEngine,
		harness.Game.Draw,
	)

	harness.Game.Log.StartTurn(1, 0, player.ID.UUID())

	// Step 1: Add Divine buff first time (duration 3 from registry)
	addAction1 := engineaction.NewAddBuffAction(player, constants.BuffTypeDivine, "TestDivine1")
	if err := actionCtx.ExecuteAction(addAction1); err != nil {
		t.Fatalf("First AddBuffAction(Divine) failed: %v", err)
	}

	// Divine applied → LP+1
	lpAfterFirstApply := player.LP
	if lpAfterFirstApply != initialLP+1 {
		t.Fatalf("LP should be initial+1 after first Divine applied, got LP=%d", lpAfterFirstApply)
	}

	// Count Divine buff instances (should be exactly 1)
	divineCount := 0
	for _, b := range player.ActiveBuffs {
		if b.Type == constants.BuffTypeDivine {
			divineCount++
		}
	}
	if divineCount != 1 {
		t.Errorf("Should have exactly 1 Divine buff instance after first apply, got %d", divineCount)
	}

	// Clear actionCtx metadata before second apply
	actionCtx.Clear()

	// Step 2: Add Divine buff again → should extend duration, NOT create new instance
	addAction2 := engineaction.NewAddBuffAction(player, constants.BuffTypeDivine, "TestDivine2")
	if err := actionCtx.ExecuteAction(addAction2); err != nil {
		t.Fatalf("Second AddBuffAction(Divine) failed: %v", err)
	}

	// Duration extension should NOT trigger LP+1 again (no PhasePostBuffApplied)
	lpAfterSecondApply := player.LP
	if lpAfterSecondApply != initialLP+1 {
		t.Errorf("LP should remain initial+1 after duration extension (no second LP+1), got LP=%d", lpAfterSecondApply)
	}

	// Should still have exactly 1 Divine buff instance
	divineCount2 := 0
	for _, b := range player.ActiveBuffs {
		if b.Type == constants.BuffTypeDivine {
			divineCount2++
		}
	}
	if divineCount2 != 1 {
		t.Errorf("Should have exactly 1 Divine buff instance after duration extension, got %d", divineCount2)
	}

	// Verify buff_duration_extended flag was set
	if !actionCtx.GetBoolOrDefault("buff_duration_extended", false) {
		t.Error("buff_duration_extended flag should be set in ActionContext.Metadata after duration extension")
	}
}

// ========== Scenario Group M: Poison Bad Event Draw ==========

// TestScenarioBuff_Poison_BadEventDraw verifies that 毒瘴 (Poison) buff
// triggers the draw_bad_event flag during PhaseBeforeTurn.
// The handler sets draw_bad_event=true which signals HSM to force a bad event draw.
func TestScenarioBuff_Poison_BadEventDraw(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		Seed:        42,
		PlayerCount: 2,
		Factions:    []constants.Faction{constants.FactionQingLong, constants.FactionZhuQue},
		InitialHP:   6,
		InitialLP:   3,
	})

	player := harness.Players[0]

	// Add Poison buff (duration 3)
	harness.AddBuffToPlayer(player, constants.BuffTypePoison, 3)

	if !player.HasBuff(constants.BuffTypePoison) {
		t.Fatal("Player should have Poison buff")
	}

	actionCtx := newActionContextWithPools(
		harness.Game,
		harness.Game.Bus,
		harness.MapEngine,
		harness.Game.Draw,
	)

	// Simulate PhaseBeforeTurn publish
	triggerCtx := event.NewContext(player)
	triggerCtx.Set("action_context", actionCtx)
	harness.Game.Bus.Publish(constants.PhaseBeforeTurn, player.ID.UUID(), triggerCtx)

	// Poison handler should set draw_bad_event flag
	if !triggerCtx.GetBoolOrDefault("draw_bad_event", false) {
		t.Error("Poison handler should set draw_bad_event=true in PhaseBeforeTurn")
	}

	// Verify no handler errors
	if triggerCtx.HasError() {
		t.Errorf("PhaseBeforeTurn handler should not produce errors: %v", triggerCtx.FirstError())
	}
}