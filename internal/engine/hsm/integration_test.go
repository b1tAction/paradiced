package hsm

import (
	"encoding/json"
	"testing"

	"github.com/b1tAction/fated/internal/core"
	"github.com/b1tAction/fated/internal/core/buff"
	"github.com/b1tAction/fated/internal/engine"
	engineaction "github.com/b1tAction/fated/internal/engine/action"
	"github.com/b1tAction/fated/internal/gamemap"
	"github.com/b1tAction/fated/pkg/constants"
	"github.com/b1tAction/fated/pkg/event"
	"github.com/b1tAction/fated/pkg/gamelog"
	"github.com/b1tAction/fated/pkg/id"
)

// ========== Integration Tests: Turn Flow with GameLog ==========

// TestTurnFlow_GameLog_Integration tests complete turn flow with GameLog recording.
func TestTurnFlow_GameLog_Integration(t *testing.T) {
	// Setup game and map
	game := engine.NewGame(id.NewGameID(), 0)
	mapEngine := gamemap.NewMapEngine(100)
	configs := map[int]gamemap.CellType{30: gamemap.CellTypeCheckpoint}
	mapEngine.GenerateLinearMap(configs)

	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID()})
	player.HP = 5
	player.LP = 3
	player.Position = 10
	game.AddPlayer(player)

	// Setup HSM adapters
	busAdapter := NewEventBusWrapper(game.Bus)
	mapAdapter := NewMapEngineWrapper(mapEngine)

	// Create TurnUpkeep state
	upkeepState := NewTurnUpkeepState()
	ctx := NewStateContext().
		WithGame(game).
		WithPlayer(player).
		WithBus(busAdapter).
		WithMapEngine(mapAdapter)

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
func TestTurnFlow_BuffEffect_GameLog(t *testing.T) {
	// Setup
	game := engine.NewGame(id.NewGameID(), 0)
	mapEngine := gamemap.NewMapEngine(100)
	mapEngine.GenerateLinearMap(nil)

	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 6})
	player.HP = 5
	player.LP = 3
	player.Position = 10
	game.AddPlayer(player)

	// Add Divine buff (神眷: LP+1 each turn)
	divineBuff := buff.NewBuff(constants.BuffTypeDivine, 3)
	player.AddBuff(divineBuff)

	// Subscribe buff to EventBus
	game.SubscribeBuff(player, divineBuff)

	// Create ActionContext for executing Actions
	mapAdapter := NewMapEngineWrapper(mapEngine)
	actionCtx := engineaction.NewActionContext(
		NewGameWrapper(game),
		game.Bus,
		NewProtocolMapEngineWrapper(mapAdapter),
	)

	// Simulate BeforeTurn phase trigger
	triggerCtx := event.NewContext(player)
	triggerCtx.Set("action_context", actionCtx)

	// Publish PhaseBeforeTurn
	game.Bus.Publish(constants.PhaseBeforeTurn, player.ID.UUID(), triggerCtx)

	// Process any derived actions
	actionCtx.ProcessQueue()

	// Verify LP was modified (Divine buff: LP+1)
	// Note: Current handler uses direct modification, not Action system
	if player.LP != 4 {
		t.Errorf("LP should be 4 after Divine buff, got %d", player.LP)
	}
}

// TestTurnFlow_Respawn_GameLog tests RespawnAction and GameLog recording.
func TestTurnFlow_Respawn_GameLog(t *testing.T) {
	// Setup
	game := engine.NewGame(id.NewGameID(), 0)
	mapEngine := gamemap.NewMapEngine(100)
	configs := map[int]gamemap.CellType{30: gamemap.CellTypeCheckpoint}
	mapEngine.GenerateLinearMap(configs)

	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 6})
	player.HP = 5
	player.Position = 50
	player.IsDead = true
	game.AddPlayer(player)

	// Start turn log
	game.Log.StartTurn(1, 0, player.ID.UUID())

	// Create ActionContext
	mapAdapter := NewMapEngineWrapper(mapEngine)
	actionCtx := engineaction.NewActionContext(
		NewGameWrapper(game),
		game.Bus,
		NewProtocolMapEngineWrapper(mapAdapter),
	)

	// Execute RespawnAction
	checkpoint := mapEngine.GetLastCheckpoint(player.Position)
	respawnAction := engineaction.NewRespawnAction(player, checkpoint, "DeathRespawn")
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
		if respawnEntry.Source != "DeathRespawn" {
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
	mapAdapter := NewMapEngineWrapper(mapEngine)
	actionCtx := engineaction.NewActionContext(
		NewGameWrapper(game),
		game.Bus,
		NewProtocolMapEngineWrapper(mapAdapter),
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
		if damageEntry.Delta != -2 {
			t.Errorf("Damage delta should be -2, got %d", damageEntry.Delta)
		}
		if damageEntry.Source != "Event_Trap" {
			t.Errorf("Damage source should be Event_Trap, got %s", damageEntry.Source)
		}
	}
}

// TestTurnFlow_CompleteTurn tests a complete turn with multiple actions.
func TestTurnFlow_CompleteTurn(t *testing.T) {
	// Setup
	game := engine.NewGame(id.NewGameID(), 0)
	mapEngine := gamemap.NewMapEngine(100)
	configs := map[int]gamemap.CellType{30: gamemap.CellTypeCheckpoint}
	mapEngine.GenerateLinearMap(configs)

	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 6})
	player.HP = 5
	player.LP = 3
	player.Position = 10
	game.AddPlayer(player)

	// Add Divine buff
	divineBuff := buff.NewBuff(constants.BuffTypeDivine, 3)
	player.AddBuff(divineBuff)
	game.SubscribeBuff(player, divineBuff)

	mapAdapter := NewMapEngineWrapper(mapEngine)

	// === Step 1: Start Turn ===
	game.Log.StartTurn(1, 0, player.ID.UUID())

	// === Step 2: BeforeTurn Phase ===
	actionCtx := engineaction.NewActionContext(
		NewGameWrapper(game),
		game.Bus,
		NewProtocolMapEngineWrapper(mapAdapter),
	)

	triggerCtx := event.NewContext(player)
	triggerCtx.Set("action_context", actionCtx)
	game.Bus.Publish(constants.PhaseBeforeTurn, player.ID.UUID(), triggerCtx)
	actionCtx.ProcessQueue()

	// === Step 3: Simulate Movement ===
	moveAction := engineaction.NewMoveAction(player, 5, "DiceRoll")
	// Calculate path
	result, _ := mapEngine.CalculatePath(player.Position, 5)
	moveAction.TargetPos = result.TargetIndex
	moveAction.Path = result.Path
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

	// Find move entry
	var moveEntry *gamelog.LogEntry
	for i := range segment.Entries {
		if segment.Entries[i].ActionType == "move" {
			moveEntry = &segment.Entries[i]
			break
		}
	}

	if moveEntry != nil {
		if moveEntry.Delta != 5 {
			t.Errorf("Move delta should be 5, got %d", moveEntry.Delta)
		}
		if moveEntry.Source != "DiceRoll" {
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
		if damageEntry.Delta != -1 {
			t.Errorf("Damage delta should be -1, got %d", damageEntry.Delta)
		}
	}

	// === Verify Final State ===
	if player.HP != 4 {
		t.Errorf("HP should be 4 (5 - 1 damage), got %d", player.HP)
	}
	if player.LP != 4 {
		t.Errorf("LP should be 4 (3 + 1 Divine), got %d", player.LP)
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
	configs := map[int]gamemap.CellType{30: gamemap.CellTypeCheckpoint}
	mapEngine.GenerateLinearMap(configs)

	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 6})
	player.HP = 0 // Dead
	player.Position = 50
	player.IsDead = true
	game.AddPlayer(player)

	// Start turn log
	game.Log.StartTurn(1, 0, player.ID.UUID())

	// Create ActionContext
	mapAdapter := NewMapEngineWrapper(mapEngine)
	actionCtx := engineaction.NewActionContext(
		NewGameWrapper(game),
		game.Bus,
		NewProtocolMapEngineWrapper(mapAdapter),
	)

	// Execute RespawnAction (should be interceptable with PhasePreRespawn)
	checkpoint := mapEngine.GetLastCheckpoint(player.Position)
	respawnAction := engineaction.NewRespawnAction(player, checkpoint, "DeathRespawn")

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
	mapAdapter := NewMapEngineWrapper(mapEngine)
	actionCtx := engineaction.NewActionContext(
		NewGameWrapper(game),
		game.Bus,
		NewProtocolMapEngineWrapper(mapAdapter),
	)

	// Simulate a handler adding derived actions via triggerCtx
	triggerCtx := event.NewContext(player)

	// Handler adds multiple actions
	triggerCtx.AddDerivedAction(engineaction.NewHealAction(player, 2, "Buff_Test"))
	triggerCtx.AddDerivedAction(engineaction.NewModifyLPAction(player, 1, "Buff_Test"))

	// Collect derived actions into ActionContext queue
	for _, derived := range triggerCtx.GetDerivedActions() {
		if execAction, ok := derived.(engineaction.ExecutableAction); ok {
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
			if entry.Delta != 2 {
				t.Errorf("Heal delta should be 2, got %d", entry.Delta)
			}
		}
		if entry.ActionType == "modify_lp" {
			hasModifyLP = true
			if entry.Delta != 1 {
				t.Errorf("ModifyLP delta should be 1, got %d", entry.Delta)
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
	game.Log.AddEntry(gamelog.NewActionEntry("modify_lp", player.ID.UUID(), 1, "Buff_Divine"))
	game.Log.AddEntry(gamelog.NewActionEntry("move", player.ID.UUID(), 5, "DiceRoll"))
	game.Log.AddEntry(gamelog.NewActionEntry("damage", player.ID.UUID(), -2, "Event_Trap"))

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
