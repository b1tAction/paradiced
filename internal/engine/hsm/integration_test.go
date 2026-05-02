package hsm

import (
	"encoding/json"
	"testing"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/engine"
	engineaction "github.com/b1tAction/paradiced/internal/engine/action"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/internal/gamemap"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/gamelog"
	"github.com/b1tAction/paradiced/pkg/id"
	"github.com/b1tAction/paradiced/pkg/rng"
)

// ========== Integration Tests: Turn Flow with GameLog ==========

// TestTurnFlow_GameLog_Integration tests complete turn flow with GameLog recording.
func TestTurnFlow_GameLog_Integration(t *testing.T) {
	// Setup game and map
	game := engine.NewGame(id.NewGameID(), 0)
	mapEngine := gamemap.NewMapEngine(100)
	configs := map[int]constants.CellType{30: constants.CellTypeCheckpoint}
	mapEngine.GenerateLinearMap(configs)

	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.HP = 5
	player.LP = 3
	player.Position = 10
	game.AddPlayer(player)

	// Create TurnUpkeep state
	upkeepState := NewTurnUpkeepState()
	hsmInst := NewHSM(game)
	hsmInst.SetMapEngine(mapEngine)
	ctx := NewStateContext().
		WithHSM(hsmInst).
		WithPlayer(player)

	// Execute Enter
	upkeepState.Enter(ctx)

	// Verify GameLog was started
	if !game.Log.IsTurnActive() {
		t.Error("GameLog should have active turn after TurnUpkeep.Enter")
	}

	// Get current turn entries
	entries := game.Log.GetCurrentTurnEntries()
	t.Logf("Turn started with %d entries", len(entries))
}

// TestTurnFlow_BuffEffect_GameLog tests Buff effect triggers and GameLog recording.
// Divine/Curse now trigger on PhasePostBuffApplied (when buff is applied) and
// PhasePreBuffRemoved (when buff is removed), not PhaseBeforeTurn.
func TestTurnFlow_BuffEffect_GameLog(t *testing.T) {
	// Setup
	game := engine.NewGame(id.NewGameID(), 0)
	mapEngine := gamemap.NewMapEngine(100)
	mapEngine.GenerateLinearMap(nil)
	game.Log.StartTurn(1, 0, "test-player")

	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 6})
	player.HP = 5
	player.LP = 3
	player.Position = 10
	game.AddPlayer(player)

	// Create ActionContext for executing Actions
	mapAdapter := mapEngine
	actionCtx := engineaction.NewActionContext(
		game,
		game.Bus,
		mapAdapter,
		game.Draw,
	)
	actionCtx.OnAddBuff = func(p *core.Player, b *core.Buff) { game.ApplyBuffToPlayer(p, b) }
	actionCtx.GetBuffDuration = func(bt constants.BuffType) int {
		def := engine.GetBuffDefinition(bt)
		if def != nil {
			return def.Duration
		}
		return 0
	}

	// Add Divine buff via AddBuffAction (triggers PhasePostBuffApplied → LP+1)
	addAction := engineaction.NewAddBuffAction(player, constants.BuffTypeDivine, "Test_Divine")
	err := actionCtx.ExecuteAction(addAction)
	if err != nil {
		t.Fatalf("ExecuteAction(AddBuffAction) failed: %v", err)
	}

	// Verify LP was modified (Divine buff: LP+1 through PhasePostBuffApplied)
	if player.LP != 4 {
		t.Errorf("LP should be 4 after Divine buff applied, got %d", player.LP)
	}
}

// TestTurnFlow_Respawn_GameLog tests RespawnAction and GameLog recording.
func TestTurnFlow_Respawn_GameLog(t *testing.T) {
	// Setup
	game := engine.NewGame(id.NewGameID(), 0)
	mapEngine := gamemap.NewMapEngine(100)
	configs := map[int]constants.CellType{30: constants.CellTypeCheckpoint}
	mapEngine.GenerateLinearMap(configs)

	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 6})
	player.HP = 5
	player.Position = 50
	player.IsDead = true
	game.AddPlayer(player)

	// Start turn log
	game.Log.StartTurn(1, 0, player.ID.UUID())

	// Create ActionContext
	mapAdapter := mapEngine
	actionCtx := engineaction.NewActionContext(
		game,
		game.Bus,
		mapAdapter,
		game.Draw,
	)

	// Execute RespawnAction
	checkpoint := mapEngine.GetLastCheckpoint(player.Position)
	respawnAction := engineaction.NewRespawnAction(player, checkpoint, string(constants.SourceDeathRespawn))
	actionCtx.ExecuteAction(respawnAction)

	// Verify player respawned
	if player.IsDead {
		t.Error("Player should not be dead after respawn")
	}
	if player.Position != checkpoint {
		t.Errorf("Position should be checkpoint %d, got %d", checkpoint, player.Position)
	}

	// Verify GameLog has respawn entry
	entries := game.Log.GetCurrentTurnEntries()
	if len(entries) == 0 {
		t.Fatal("Should have at least one entry (respawn)")
	}

	// Find respawn entry
	var respawnEntry *gamelog.LogEntry
	for i := range entries {
		if entries[i].ActionType == "respawn" {
			respawnEntry = &entries[i]
			break
		}
	}

	if respawnEntry == nil {
		t.Error("Should have respawn entry in GameLog")
	} else {
		if respawnEntry.Target != player.ID.UUID() {
			t.Errorf("Respawn target should be %s, got %s", player.ID.UUID(), respawnEntry.Target)
		}
		if respawnEntry.Source != string(constants.SourceDeathRespawn) {
			t.Errorf("Respawn source should be DeathRespawn, got %s", respawnEntry.Source)
		}
		// Check metadata for checkpoint_pos
		checkpointPos := respawnEntry.Metadata.GetIntOrDefault("checkpoint_pos", -1)
		if checkpointPos != checkpoint {
			t.Errorf("Checkpoint pos should be %d, got %d", checkpoint, checkpointPos)
		}
	}
}

// TestTurnFlow_Damage_GameLog tests DamageAction and GameLog recording.
func TestTurnFlow_Damage_GameLog(t *testing.T) {
	// Setup
	game := engine.NewGame(id.NewGameID(), 0)
	mapEngine := gamemap.NewMapEngine(100)

	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 6})
	player.HP = 5
	game.AddPlayer(player)

	// Start turn log
	game.Log.StartTurn(1, 0, player.ID.UUID())

	// Create ActionContext
	mapAdapter := mapEngine
	actionCtx := engineaction.NewActionContext(
		game,
		game.Bus,
		mapAdapter,
		game.Draw,
	)

	// Execute DamageAction (trap damage)
	damageAction := engineaction.NewDamageAction(player, 2, "Event_Trap")
	actionCtx.ExecuteAction(damageAction)

	// Verify HP decreased
	if player.HP != 3 {
		t.Errorf("HP should be 3 after 2 damage, got %d", player.HP)
	}

	// Verify GameLog has damage entry
	entries := game.Log.GetCurrentTurnEntries()
	if len(entries) == 0 {
		t.Fatal("Should have at least one entry (damage)")
	}

	var damageEntry *gamelog.LogEntry
	for i := range entries {
		if entries[i].ActionType == "damage" {
			damageEntry = &entries[i]
			break
		}
	}

	if damageEntry == nil {
		t.Error("Should have damage entry in GameLog")
	} else {
		if damageEntry.Metadata.GetIntOrDefault("hp_change", 0) != -2 {
			t.Errorf("Damage hp_change should be -2, got %d", damageEntry.Metadata.GetIntOrDefault("hp_change", 0))
		}
		if damageEntry.Source != "Event_Trap" {
			t.Errorf("Damage source should be Event_Trap, got %s", damageEntry.Source)
		}
	}
}

// TestTurnFlow_CompleteTurn tests a complete turn with multiple actions.
// Divine/Curse now trigger LP changes on buff application/removal, not BeforeTurn.
func TestTurnFlow_CompleteTurn(t *testing.T) {
	// Setup
	game := engine.NewGame(id.NewGameID(), 0)
	mapEngine := gamemap.NewMapEngine(100)
	configs := map[int]constants.CellType{30: constants.CellTypeCheckpoint}
	mapEngine.GenerateLinearMap(configs)

	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 6})
	player.HP = 5
	player.LP = 3
	player.Position = 10
	game.AddPlayer(player)

	mapAdapter := mapEngine

	// === Step 1: Start Turn ===
	game.Log.StartTurn(1, 0, player.ID.UUID())

	// === Step 2: Add Divine buff via AddBuffAction (triggers LP+1) ===
	actionCtx := engineaction.NewActionContext(
		game,
		game.Bus,
		mapAdapter,
		game.Draw,
	)
	actionCtx.OnAddBuff = func(p *core.Player, b *core.Buff) { game.ApplyBuffToPlayer(p, b) }
	actionCtx.GetBuffDuration = func(bt constants.BuffType) int {
		def := engine.GetBuffDefinition(bt)
		if def != nil {
			return def.Duration
		}
		return 0
	}

	addBuffAction := engineaction.NewAddBuffAction(player, constants.BuffTypeDivine, "Test_Divine")
	err := actionCtx.ExecuteAction(addBuffAction)
	if err != nil {
		t.Fatalf("ExecuteAction(AddBuffAction) failed: %v", err)
	}

	// === Step 3: Simulate Movement ===
	// MoveAction now reads from ActionContext.Metadata
	result, _ := mapEngine.CalculatePath(player.Position, 5)
	actionCtx.SetInt("target_pos", result.TargetIndex)
	actionCtx.Set("path", result.Path)
	moveAction := engineaction.NewMoveAction(player, 5, string(constants.SourceSystemDiceRoll))
	actionCtx.ExecuteAction(moveAction)

	// === Step 4: Simulate Trap Damage ===
	damageAction := engineaction.NewDamageAction(player, 1, "Event_Trap")
	actionCtx.ExecuteAction(damageAction)

	// === Step 5: End Turn ===
	game.Log.EndTurn()

	// === Verify Complete GameLog ===
	segments := game.Log.GetTurnSegments()
	if len(segments) != 1 {
		t.Fatalf("Should have 1 segment, got %d", len(segments))
	}

	segment := segments[0]
	if segment.Round != 1 {
		t.Errorf("Round should be 1, got %d", segment.Round)
	}
	if segment.Turn != 0 {
		t.Errorf("Turn should be 0, got %d", segment.Turn)
	}
	if segment.PlayerID != player.ID.UUID() {
		t.Errorf("PlayerID should be %s, got %s", player.ID.UUID(), segment.PlayerID)
	}

	// Verify entries count
	t.Logf("Segment has %d entries", len(segment.Entries))

	// Find add_buff entry (Divine)
	var addBuffEntry *gamelog.LogEntry
	for i := range segment.Entries {
		if segment.Entries[i].ActionType == "add_buff" {
			addBuffEntry = &segment.Entries[i]
			break
		}
	}

	if addBuffEntry != nil {
		if addBuffEntry.Metadata.GetStringOrDefault("buff_type", "") != "divine" {
			t.Errorf("AddBuff buff_type should be divine, got %s", addBuffEntry.Metadata.GetStringOrDefault("buff_type", ""))
		}
	}

	// Find modify_lp entry (Divine LP+1)
	var modifyLPEntry *gamelog.LogEntry
	for i := range segment.Entries {
		if segment.Entries[i].ActionType == "modify_lp" {
			modifyLPEntry = &segment.Entries[i]
			break
		}
	}

	if modifyLPEntry != nil {
		if modifyLPEntry.Metadata.GetIntOrDefault("lp_change", 0) != 1 {
			t.Errorf("ModifyLP lp_change should be 1 (Divine LP+1), got %d", modifyLPEntry.Metadata.GetIntOrDefault("lp_change", 0))
		}
	}

	// Find move entry
	var moveEntry *gamelog.LogEntry
	for i := range segment.Entries {
		if segment.Entries[i].ActionType == "move" {
			moveEntry = &segment.Entries[i]
			break
		}
	}

	if moveEntry != nil {
		if moveEntry.Metadata.GetIntOrDefault("steps", 0) != 5 {
			t.Errorf("Move steps should be 5, got %d", moveEntry.Metadata.GetIntOrDefault("steps", 0))
		}
		if moveEntry.Source != string(constants.SourceSystemDiceRoll) {
			t.Errorf("Move source should be DiceRoll, got %s", moveEntry.Source)
		}
	}

	// Find damage entry
	var damageEntry *gamelog.LogEntry
	for i := range segment.Entries {
		if segment.Entries[i].ActionType == "damage" {
			damageEntry = &segment.Entries[i]
			break
		}
	}

	if damageEntry != nil {
		if damageEntry.Metadata.GetIntOrDefault("hp_change", 0) != -1 {
			t.Errorf("Damage hp_change should be -1, got %d", damageEntry.Metadata.GetIntOrDefault("hp_change", 0))
		}
	}

	// === Verify Final State ===
	if player.HP != 4 {
		t.Errorf("HP should be 4 (5 - 1 damage), got %d", player.HP)
	}
	if player.LP != 4 {
		t.Errorf("LP should be 4 (3 + 1 Divine on application), got %d", player.LP)
	}

	// === Verify JSON Serialization ===
	jsonBytes, err := game.Log.ToJSON()
	if err != nil {
		t.Errorf("ToJSON failed: %v", err)
	}

	// Parse and verify
	var parsed map[string]interface{}
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Errorf("JSON unmarshal failed: %v", err)
	}

	segmentsArray, ok := parsed["segments"].([]interface{})
	if !ok {
		t.Error("JSON should have segments array")
	}
	if len(segmentsArray) != 1 {
		t.Errorf("JSON segments should have 1 element, got %d", len(segmentsArray))
	}
}

// TestTurnFlow_Interrupt_Respawn tests intercepting RespawnAction.
func TestTurnFlow_Interrupt_Respawn(t *testing.T) {
	// Setup
	game := engine.NewGame(id.NewGameID(), 0)
	mapEngine := gamemap.NewMapEngine(100)
	configs := map[int]constants.CellType{30: constants.CellTypeCheckpoint}
	mapEngine.GenerateLinearMap(configs)

	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 6})
	player.HP = 0 // Dead
	player.Position = 50
	player.IsDead = true
	game.AddPlayer(player)

	// Start turn log
	game.Log.StartTurn(1, 0, player.ID.UUID())

	// Create ActionContext
	mapAdapter := mapEngine
	actionCtx := engineaction.NewActionContext(
		game,
		game.Bus,
		mapAdapter,
		game.Draw,
	)

	// Execute RespawnAction (should be interceptable with PhasePreRespawn)
	checkpoint := mapEngine.GetLastCheckpoint(player.Position)
	respawnAction := engineaction.NewRespawnAction(player, checkpoint, string(constants.SourceDeathRespawn))

	// Verify PreTriggerPhase is PhasePreRespawn
	if respawnAction.PreTriggerPhase() != constants.PhasePreRespawn {
		t.Errorf("RespawnAction PreTriggerPhase should be PhasePreRespawn, got %s", string(respawnAction.PreTriggerPhase()))
	}

	actionCtx.ExecuteAction(respawnAction)

	// Verify respawn happened
	if player.IsDead {
		t.Error("Player should not be dead after respawn")
	}

	// Verify GameLog
	entries := game.Log.GetCurrentTurnEntries()
	var respawnEntry *gamelog.LogEntry
	for i := range entries {
		if entries[i].ActionType == "respawn" {
			respawnEntry = &entries[i]
			break
		}
	}

	if respawnEntry == nil {
		t.Error("Should have respawn entry")
	}
}

// TestDerivedActions_FromHandler tests ctx.AddDerivedAction() from handler.
func TestDerivedActions_FromHandler(t *testing.T) {
	// Setup
	game := engine.NewGame(id.NewGameID(), 0)
	mapEngine := gamemap.NewMapEngine(100)

	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 6, MaxLP: 3}) // Set MaxLP to 3 for testing
	player.HP = 3
	player.LP = 0 // Start with LP=0 to test increment
	game.AddPlayer(player)

	game.Log.StartTurn(1, 0, player.ID.UUID())

	// Create ActionContext
	mapAdapter := mapEngine
	actionCtx := engineaction.NewActionContext(
		game,
		game.Bus,
		mapAdapter,
		game.Draw,
	)

	// Simulate a handler adding derived actions via triggerCtx
	triggerCtx := event.NewContext(player)

	// Handler adds multiple actions
	triggerCtx.AddDerivedAction(engineaction.NewHealAction(player, 2, "Buff_Test"))
	triggerCtx.AddDerivedAction(engineaction.NewModifyLPAction(player, 1, "Buff_Test"))

	// Collect derived actions into ActionContext queue
	for _, derived := range triggerCtx.GetDerivedActions() {
		if execAction, ok := derived.(engineaction.Action); ok {
			actionCtx.PushDerivedAction(execAction)
		}
	}

	// Process queue
	actionCtx.ProcessQueue()

	// Verify player state
	if player.HP != 5 {
		t.Errorf("HP should be 5 after heal, got %d", player.HP)
	}
	if player.LP != 1 {
		t.Errorf("LP should be 1 after modify, got %d", player.LP)
	}

	// Verify GameLog has both entries
	entries := game.Log.GetCurrentTurnEntries()
	var hasHeal, hasModifyLP bool
	for _, entry := range entries {
		if entry.ActionType == "heal" {
			hasHeal = true
			if entry.Metadata.GetIntOrDefault("hp_change", 0) != 2 {
				t.Errorf("Heal hp_change should be 2, got %d", entry.Metadata.GetIntOrDefault("hp_change", 0))
			}
		}
		if entry.ActionType == "modify_lp" {
			hasModifyLP = true
			if entry.Metadata.GetIntOrDefault("lp_change", 0) != 1 {
				t.Errorf("ModifyLP lp_change should be 1, got %d", entry.Metadata.GetIntOrDefault("lp_change", 0))
			}
		}
	}

	if !hasHeal {
		t.Error("GameLog should have heal entry")
	}
	if !hasModifyLP {
		t.Error("GameLog should have modify_lp entry")
	}
}

// TestGameLog_JSON_Output tests full GameLog JSON output for client.
func TestGameLog_JSON_Output(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 0)

	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 6})
	player.HP = 5
	game.AddPlayer(player)

	// Simulate a full turn with actions
	game.Log.StartTurn(1, 0, player.ID.UUID())

	// Add various entries
	game.Log.AddEntry(gamelog.NewActionEntry("modify_lp", player.ID.UUID(), "Buff_Divine"))
	game.Log.AddEntry(gamelog.NewActionEntry("move", player.ID.UUID(), string(constants.SourceSystemDiceRoll)))
	game.Log.AddEntry(gamelog.NewActionEntry("damage", player.ID.UUID(), "Event_Trap"))

	game.Log.EndTurn()

	// Get JSON
	jsonBytes, err := game.Log.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	t.Logf("GameLog JSON:\n%s", string(jsonBytes))

	// Verify JSON structure
	var logData struct {
		Segments []struct {
			Round    int    `json:"round"`
			Turn     int    `json:"turn"`
			PlayerID string `json:"player_id"`
			Entries  []struct {
				Type       string `json:"type"`
				ActionType string `json:"action_type,omitempty"`
				Target     string `json:"target,omitempty"`
				Delta      int    `json:"delta,omitempty"`
				Source     string `json:"source,omitempty"`
			} `json:"entries"`
		} `json:"segments"`
	}

	err = json.Unmarshal(jsonBytes, &logData)
	if err != nil {
		t.Fatalf("JSON parse failed: %v", err)
	}

	if len(logData.Segments) != 1 {
		t.Fatalf("Should have 1 segment, got %d", len(logData.Segments))
	}

	seg := logData.Segments[0]
	if seg.Round != 1 {
		t.Errorf("Round should be 1, got %d", seg.Round)
	}
	if seg.PlayerID != player.ID.UUID() {
		t.Errorf("PlayerID should be %s, got %s", player.ID.UUID(), seg.PlayerID)
	}
	if len(seg.Entries) != 3 {
		t.Errorf("Should have 3 entries, got %d", len(seg.Entries))
	}

	// Verify entry types
	expectedTypes := []string{"modify_lp", "move", "damage"}
	for i, entry := range seg.Entries {
		if entry.ActionType != expectedTypes[i] {
			t.Errorf("Entry %d ActionType should be %s, got %s", i, expectedTypes[i], entry.ActionType)
		}
		if entry.Type != "action" {
			t.Errorf("Entry %d Type should be action, got %s", i, entry.Type)
		}
	}
}

// ========== CheckPoint Split Flow Integration Test ==========

// TestCheckPointSplit_Integration tests the full CheckPoint split flow:
// TurnMoving → TurnCheckpoint → TurnMoving(re-entry) → TurnLanded
// Verifying: two MoveAction entries + one DrawItemAction entry in GameLog.
func TestCheckPointSplit_Integration(t *testing.T) {
	// Setup game with seed for deterministic RNG
	game := engine.NewGame(id.NewGameID(), 42)
	mapEngine := gamemap.NewMapEngine(100)
	configs := map[int]constants.CellType{25: constants.CellTypeCheckpoint}
	mapEngine.GenerateLinearMap(configs)

	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 6})
	player.HP = 5
	player.LP = 3
	player.Position = 20
	game.AddPlayer(player)

	// Setup pools for DrawItemAction
	game.ItemPool = []*rng.EvaluatedItem{
		{Type: "healing_potion", Eval: constants.EvaluationGood},
	}
	game.EventPool = []*rng.EvaluatedItem{
		{Type: "herb", Eval: constants.EvaluationMildGood},
	}

	// Start turn log for GameLog recording
	game.Log.StartTurn(1, 0, player.ID.UUID())

	hsmInst := NewHSM(game)
	hsmInst.SetMapEngine(mapEngine)

	// === Phase 1: TurnMoving (first segment - move to CheckPoint) ===
	movingState1 := NewTurnMovingState()
	ctx1 := NewStateContext().
		WithHSM(hsmInst).
		WithPlayer(player)
	ctx1.SetInt(KeyDiceSteps, 10) // Start at 20, should reach CheckPoint at 25 (5 steps)

	movingState1.Enter(ctx1)

	if ctx1.Error != nil {
		t.Fatalf("TurnMoving.Enter should succeed, got error: %v", ctx1.Error)
	}
	if !movingState1.hasCheckpoint {
		t.Error("TurnMoving should detect CheckPoint at 25")
	}
	if player.Position != 25 {
		t.Errorf("Player should be at CheckPoint 25 after first segment, got %d", player.Position)
	}
	if movingState1.remainingSteps != 5 {
		t.Errorf("Remaining steps should be 5, got %d", movingState1.remainingSteps)
	}

	nextState1 := movingState1.Update(ctx1)
	if nextState1 != StateTurnCheckpoint {
		t.Errorf("TurnMoving.Update should return StateTurnCheckpoint, got %s", nextState1.String())
	}

	// === Phase 2: TurnCheckpoint (DrawItemAction) ===
	checkpointState := NewTurnCheckpointState()
	ctx2 := NewStateContext().
		WithHSM(hsmInst).
		WithPlayer(player)
	// Set remaining steps for re-entry
	ctx2.SetInt(KeyDiceSteps, movingState1.remainingSteps)

	checkpointState.Enter(ctx2)

	if ctx2.Error != nil {
		t.Fatalf("TurnCheckpoint.Enter should succeed, got error: %v", ctx2.Error)
	}

	nextState2 := checkpointState.Update(ctx2)
	if nextState2 != StateTurnMoving {
		t.Errorf("TurnCheckpoint.Update should return StateTurnMoving, got %s", nextState2.String())
	}

	// === Phase 3: TurnMoving (second segment - continue movement) ===
	movingState2 := NewTurnMovingState()
	ctx3 := NewStateContext().
		WithHSM(hsmInst).
		WithPlayer(player)
	ctx3.SetInt(KeyDiceSteps, 5) // Remaining 5 steps from CheckPoint 25 → 30

	movingState2.Enter(ctx3)

	if ctx3.Error != nil {
		t.Fatalf("TurnMoving re-entry should succeed, got error: %v", ctx3.Error)
	}
	// Player should be at 25 + 5 = 30
	if player.Position != 30 {
		t.Errorf("Player should be at 30 after second segment, got %d", player.Position)
	}

	// === Phase 4: TurnLanded ===
	landedState := NewTurnLandedState()
	ctx4 := NewStateContext().
		WithHSM(hsmInst).
		WithPlayer(player)
	landedState.Enter(ctx4)

	if ctx4.Error != nil {
		t.Fatalf("TurnLanded.Enter should succeed, got error: %v", ctx4.Error)
	}

	// === Verify GameLog: two move entries + one draw_item entry ===
	game.Log.EndTurn()
	segments := game.Log.GetTurnSegments()
	if len(segments) == 0 {
		t.Fatal("Should have at least 1 turn segment")
	}
	entries := segments[len(segments)-1].Entries

	moveCount := 0
	drawItemCount := 0
	for _, entry := range entries {
		if entry.ActionType == "move" {
			moveCount++
		}
		if entry.ActionType == "draw_item" {
			drawItemCount++
		}
	}

	if moveCount < 2 {
		t.Errorf("Should have at least 2 move entries for CheckPoint split, got %d", moveCount)
	}
	if drawItemCount < 1 {
		t.Errorf("Should have at least 1 draw_item entry from CheckPoint, got %d", drawItemCount)
	}

	t.Logf("CheckPoint split flow produced %d move entries + %d draw_item entries", moveCount, drawItemCount)
}
