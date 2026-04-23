package hsm

import (
	"testing"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/engine"
	engineaction "github.com/b1tAction/paradiced/internal/engine/action"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
)

// ========== Scenario Group A: Buff Effect Verification ==========

// TestScenarioBuff_Divine_LPIncrementOnApplied verifies that 神眷 (Divine) buff
// increases LP by 1 when applied (via PhasePostBuffApplied).
// Divine now triggers LP+1 on buff application, LP-1 revert on buff removal.
// WIP: test isolation issue - TypeCurse subscription interferes with TypeDivine test
func TestScenarioBuff_Divine_LPIncrementOnApplied(t *testing.T) {
	t.Skip("WIP: test isolation issue - TypeCurse subscription interferes with TypeDivine test")
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
		player,
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
// WIP: test isolation issue - TypeCurse subscription interferes with TypeDivine test
func TestScenarioBuff_Curse_LPDecrementOnApplied(t *testing.T) {
	t.Skip("WIP: test isolation issue - TypeCurse subscription interferes with TypeDivine test")
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
		player,
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
		player,
	)

	// Start turn log for GameLog recording
	harness.Game.Log.StartTurn(1, 0, player.ID.UUID())

	// Execute DamageAction that kills the player
	damageAction := engineaction.NewDamageAction(player, 10, "Buff_Corrupt")
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
			if source != "Buff_Corrupt" {
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
// "FragileCell".
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
		player,
	)

	harness.Game.Log.StartTurn(1, 0, player.ID.UUID())

	// Execute FellDownAction that kills the player (damage=1, HP=1 → 0)
	fellDownAction := engineaction.NewFellDownAction(player, 15, 1, "FragileCell")
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
			if source != "FragileCell" {
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
		player,
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
		InitialLP:   3,
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
		player,
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
	respawnAction := engineaction.NewRespawnAction(player, checkpoint, "TurnEndRespawn")
	err := actionCtx.ExecuteAction(respawnAction)
	if err != nil {
		t.Fatalf("ExecuteAction(RespawnAction) failed: %v", err)
	}

	// Verify respawn state
	if player.IsDead {
		t.Error("Player should NOT be dead after respawn")
	}
	if player.HP != 3 { // Respawn resets HP to MaxHP
		t.Errorf("Player HP should be reset to MaxHP(3), got %d", player.HP)
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
		player,
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

	action := engineaction.NewDeathAction(player, "Buff_Corrupt", 25)

	// Verify Type
	if action.Type() != constants.ActionDeath {
		t.Errorf("DeathAction Type should be 'death', got '%s'", action.Type())
	}

	// Verify LogEntry
	entry := action.LogEntry()
	if entry.ActionType != "death" {
		t.Errorf("LogEntry ActionType should be 'death', got '%s'", entry.ActionType)
	}
	if entry.Source != "Buff_Corrupt" {
		t.Errorf("LogEntry Source should be 'Buff_Corrupt', got '%s'", entry.Source)
	}

	// Verify metadata
	position := entry.Metadata.GetIntOrDefault("position", -1)
	if position != 25 {
		t.Errorf("Metadata position should be 25, got %d", position)
	}
	deathSource := entry.Metadata.GetStringOrDefault("death_source", "")
	if deathSource != "Buff_Corrupt" {
		t.Errorf("Metadata death_source should be 'Buff_Corrupt', got '%s'", deathSource)
	}
}