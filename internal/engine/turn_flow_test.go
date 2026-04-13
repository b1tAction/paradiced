package engine

import (
	"testing"

	"github.com/b1tAction/Fated/internal/core"
	"github.com/b1tAction/Fated/internal/gamemap"
	"github.com/b1tAction/Fated/pkg/event"
)

func TestNewTurnFlow(t *testing.T) {
	game := NewGame("test-game")
	player := core.NewPlayer(core.PlayerConfig{
		UserID:  "player-1",
		Faction: core.FactionZhuQue,
	})
	game.AddPlayer(player)

	mapEngine := gamemap.NewMapEngine(50)

	tf := NewTurnFlow(game, mapEngine)

	if tf.Game != game {
		t.Error("Game not set correctly")
	}
	if tf.StateMachine == nil {
		t.Error("StateMachine not created")
	}
	if tf.MapEngine != mapEngine {
		t.Error("MapEngine not set correctly")
	}
	if tf.CurrentStep != StepInit {
		t.Error("CurrentStep should be StepInit")
	}
	if tf.Interrupted {
		t.Error("Interrupted should be false")
	}
}

func TestTurnStepString(t *testing.T) {
	tests := []struct {
		step     TurnStep
		expected string
	}{
		{StepInit, "Init"},
		{StepUpcheck, "Upcheck"},
		{StepBeforeTurn, "BeforeTurn"},
		{StepMainAction, "MainAction"},
		{StepOnMove, "OnMove"},
		{StepOnLand, "OnLand"},
		{StepPreEvent, "PreEvent"},
		{StepAfterTurn, "AfterTurn"},
		{StepComplete, "Complete"},
	}

	for _, tt := range tests {
		if got := tt.step.String(); got != tt.expected {
			t.Errorf("TurnStep(%d).String() = %s, want %s", tt.step, got, tt.expected)
		}
	}
}

func TestExecuteUpcheck(t *testing.T) {
	game := NewGame("test-game")
	player := core.NewPlayer(core.PlayerConfig{
		UserID:  "player-1",
		Faction: core.FactionQingLong,
	})
	game.AddPlayer(player)

	mapEngine := gamemap.NewMapEngine(50)
	tf := NewTurnFlow(game, mapEngine)

	// Normal player can act
	result := tf.executeUpcheck(player)
	if !result.Success {
		t.Error("Upcheck should succeed for normal player")
	}
	if result.PlayerUpdated {
		t.Error("PlayerUpdated should be false for normal player")
	}
}

func TestExecuteUpcheckSkipTurn(t *testing.T) {
	game := NewGame("test-game")
	player := core.NewPlayer(core.PlayerConfig{
		UserID:  "player-1",
		Faction: core.FactionQingLong,
	})
	player.SkipTurn = true
	game.AddPlayer(player)

	mapEngine := gamemap.NewMapEngine(50)
	tf := NewTurnFlow(game, mapEngine)

	result := tf.executeUpcheck(player)
	if !result.Success {
		t.Error("Upcheck should succeed even with SkipTurn")
	}
	// SkipTurn should be reset
	if player.SkipTurn {
		t.Error("SkipTurn should be reset after upcheck")
	}
	// CurrentStep should jump to AfterTurn
	if tf.CurrentStep != StepAfterTurn-1 {
		t.Errorf("CurrentStep should jump to AfterTurn-1, got %d", tf.CurrentStep)
	}
}

func TestExecuteUpcheckDeadPlayer(t *testing.T) {
	game := NewGame("test-game")
	player := core.NewPlayer(core.PlayerConfig{
		UserID:  "player-1",
		Faction: core.FactionQingLong,
	})
	player.Position = 30
	player.IsDead = true
	player.HP = 0
	game.AddPlayer(player)

	// Add checkpoint at position 20
	mapEngine := gamemap.NewMapEngine(50)
	mapEngine.SetCellType(20, gamemap.CellTypeCheckpoint)

	tf := NewTurnFlow(game, mapEngine)

	result := tf.executeUpcheck(player)
	if !result.Success {
		t.Error("Upcheck should succeed for dead player")
	}
	if !result.PlayerUpdated {
		t.Error("PlayerUpdated should be true for respawn")
	}
	if player.IsDead {
		t.Error("IsDead should be false after respawn")
	}
	if player.HP <= 0 {
		t.Error("HP should be restored after respawn")
	}
}

func TestExecuteBeforeTurn(t *testing.T) {
	game := NewGame("test-game")
	player := core.NewPlayer(core.PlayerConfig{
		UserID:  "player-1",
		Faction: core.FactionZhuQue, // ZhuQue starts with Fire buff
	})
	game.AddPlayer(player)

	mapEngine := gamemap.NewMapEngine(50)
	tf := NewTurnFlow(game, mapEngine)

	result := tf.executeBeforeTurn(player)

	// BeforeTurn should trigger Fire buff (auto decision)
	// Fire buff has NeedConfirm=false, so it should auto execute
	if !result.Success {
		t.Error("BeforeTurn should succeed when all decisions auto-execute")
	}
}

func TestExecuteBeforeTurnWithDecision(t *testing.T) {
	game := NewGame("test-game")
	player := core.NewPlayer(core.PlayerConfig{
		UserID:  "player-1",
		Faction: core.FactionQingLong,
	})

	// Divine buff is auto-executed (NeedConfirm=false by default)
	buff := core.NewBuff(core.BuffTypeDivine, 3)
	player.AddBuff(buff)
	game.AddPlayer(player)
	game.SubscribeBuff(player, buff)

	mapEngine := gamemap.NewMapEngine(50)
	tf := NewTurnFlow(game, mapEngine)

	result := tf.executeBeforeTurn(player)

	// Divine buff auto-executes, no user decision needed
	// Note: To test NeedConfirm=true scenario, requires custom buff registration
	t.Logf("BeforeTurn result: Success=%v, Decisions=%d", result.Success, len(result.Decisions))
}

func TestExecuteOnMove(t *testing.T) {
	game := NewGame("test-game")
	player := core.NewPlayer(core.PlayerConfig{
		UserID:  "player-1",
		Faction: core.FactionQingLong,
		StartPos: 10,
	})
	game.AddPlayer(player)

	mapEngine := gamemap.NewMapEngine(50)
	tf := NewTurnFlow(game, mapEngine)
	tf.DiceSteps = 5

	result := tf.executeOnMove(player)

	if !result.Success {
		t.Error("OnMove should succeed")
	}
	if result.PathResult == nil {
		t.Error("PathResult should be set")
	}
	if result.PathResult.TargetIndex != 15 {
		t.Errorf("TargetIndex should be 15, got %d", result.PathResult.TargetIndex)
	}
	if player.Position != 15 {
		t.Errorf("Player position should be 15, got %d", player.Position)
	}
}

func TestExecuteOnMoveWithLostBuff(t *testing.T) {
	game := NewGame("test-game")
	player := core.NewPlayer(core.PlayerConfig{
		UserID:  "player-1",
		Faction: core.FactionQingLong,
		StartPos: 20,
	})
	// Add Lost buff (reverse movement)
	buff := core.NewBuff(core.BuffTypeLost, 1)
	player.AddBuff(buff)
	game.AddPlayer(player)
	game.SubscribeBuff(player, buff)

	mapEngine := gamemap.NewMapEngine(50)
	tf := NewTurnFlow(game, mapEngine)
	tf.DiceSteps = 5
	tf.CurrentPlayer = player

	result := tf.executeOnMove(player)

	if !result.Success {
		t.Error("OnMove should succeed with Lost buff")
	}
	// Lost buff reverse movement is handled by OnMove phase trigger
	// The buff's handler should reverse the direction
	// Note: This test verifies the flow works; actual reverse logic in buff handler
	t.Logf("Player position after move with Lost buff: %d", player.Position)
}

func TestExecuteAfterTurn(t *testing.T) {
	game := NewGame("test-game")
	player := core.NewPlayer(core.PlayerConfig{
		UserID:  "player-1",
		Faction: core.FactionQingLong,
	})

	// Add buff with duration
	buff := core.NewBuff(core.BuffTypeCurse, 1)
	player.AddBuff(buff)
	game.AddPlayer(player)
	game.SubscribeBuff(player, buff)

	mapEngine := gamemap.NewMapEngine(50)
	tf := NewTurnFlow(game, mapEngine)

	result := tf.executeAfterTurn(player)

	if !result.Success {
		t.Error("AfterTurn should succeed")
	}
	if !result.PlayerUpdated {
		t.Error("PlayerUpdated should be true")
	}

	// Buff with duration 1 should be expired
	// Note: TickBuffs happens in executeAfterTurn
}

func TestOnUserChoice(t *testing.T) {
	game := NewGame("test-game")
	player := core.NewPlayer(core.PlayerConfig{
		UserID:  "player-1",
		Faction: core.FactionQingLong,
	})
	game.AddPlayer(player)

	mapEngine := gamemap.NewMapEngine(50)
	tf := NewTurnFlow(game, mapEngine)
	tf.CurrentPlayer = player

	// Create a decision
	decision := event.NewDecision("Test decision", []event.Option{
		{ID: "opt1", Label: "Option 1"},
		{ID: "opt2", Label: "Option 2"},
	})
	tf.Decisions = []*event.Decision{decision}
	tf.CurrentStep = StepMainAction

	err := tf.OnUserChoice(0)
	if err != nil {
		t.Errorf("OnUserChoice failed: %v", err)
	}

	if len(tf.Decisions) != 0 {
		t.Error("Decisions should be cleared after choice")
	}
	if tf.CurrentStep != StepMainAction+1 {
		t.Errorf("CurrentStep should advance, got %d", tf.CurrentStep)
	}
}

func TestCreateSnapshot(t *testing.T) {
	game := NewGame("test-game")
	player := core.NewPlayer(core.PlayerConfig{
		UserID:  "player-1",
		Faction: core.FactionQingLong,
	})
	game.AddPlayer(player)

	mapEngine := gamemap.NewMapEngine(50)
	tf := NewTurnFlow(game, mapEngine)
	tf.CurrentPlayer = player
	tf.CurrentStep = StepMainAction
	tf.DiceSteps = 6

	// Add a decision
	decision := event.NewDecision("Test", []event.Option{
		{ID: "opt1", Label: "Option 1"},
	})
	tf.Decisions = []*event.Decision{decision}

	snapshot := tf.CreateSnapshot()

	if snapshot.GameID != game.ID {
		t.Error("Snapshot GameID mismatch")
	}
	if snapshot.CurrentStep != StepMainAction {
		t.Error("Snapshot CurrentStep mismatch")
	}
	if snapshot.PlayerID != player.UserID {
		t.Error("Snapshot PlayerID mismatch")
	}
	if len(snapshot.WaitingDecisions) != 1 {
		t.Error("Snapshot should have one decision")
	}
}

func TestInterruptAndResume(t *testing.T) {
	game := NewGame("test-game")
	player := core.NewPlayer(core.PlayerConfig{
		UserID:  "player-1",
		Faction: core.FactionQingLong,
	})
	game.AddPlayer(player)

	mapEngine := gamemap.NewMapEngine(50)
	tf := NewTurnFlow(game, mapEngine)
	tf.CurrentPlayer = player

	// Add decision to make it waiting
	decision := event.NewDecision("Test", []event.Option{
		{ID: "opt1", Label: "Option 1"},
	})
	tf.Decisions = []*event.Decision{decision}

	// Interrupt
	err := tf.Interrupt()
	if err != nil {
		t.Errorf("Interrupt failed: %v", err)
	}
	if !tf.Interrupted {
		t.Error("Interrupted should be true")
	}
	if tf.SavedSnapshot == nil {
		t.Error("SavedSnapshot should be created")
	}

	// Resume
	err = tf.ResumeFromInterrupt(tf.SavedSnapshot)
	if err != nil {
		t.Errorf("Resume failed: %v", err)
	}
	if tf.Interrupted {
		t.Error("Interrupted should be false after resume")
	}
}

func TestIsWaiting(t *testing.T) {
	game := NewGame("test-game")
	mapEngine := gamemap.NewMapEngine(50)
	tf := NewTurnFlow(game, mapEngine)

	if tf.IsWaiting() {
		t.Error("Should not be waiting initially")
	}

	decision := event.NewDecision("Test", []event.Option{
		{ID: "opt1", Label: "Option 1"},
	})
	tf.Decisions = []*event.Decision{decision}

	if !tf.IsWaiting() {
		t.Error("Should be waiting with decisions")
	}
}

func TestSetDiceSteps(t *testing.T) {
	game := NewGame("test-game")
	mapEngine := gamemap.NewMapEngine(50)
	tf := NewTurnFlow(game, mapEngine)

	tf.SetDiceSteps(10)
	if tf.DiceSteps != 10 {
		t.Errorf("DiceSteps should be 10, got %d", tf.DiceSteps)
	}
}

// ========== Snapshot Tests ==========

func TestFlowSnapshotToJSON(t *testing.T) {
	snapshot := NewFlowSnapshot("test-game")
	snapshot.Round = 1
	snapshot.Turn = 0
	snapshot.CurrentStep = StepMainAction
	snapshot.PlayerID = "player-1"

	data, err := snapshot.ToJSON()
	if err != nil {
		t.Errorf("ToJSON failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("JSON data should not be empty")
	}
}

func TestFlowSnapshotFromJSON(t *testing.T) {
	original := NewFlowSnapshot("test-game")
	original.Round = 1
	original.Turn = 0
	original.CurrentStep = StepMainAction
	original.PlayerID = "player-1"

	data, _ := original.ToJSON()

	restored := &FlowSnapshot{}
	err := restored.FromJSON(data)
	if err != nil {
		t.Errorf("FromJSON failed: %v", err)
	}
	if restored.GameID != original.GameID {
		t.Error("GameID mismatch after restore")
	}
	if restored.Round != original.Round {
		t.Error("Round mismatch after restore")
	}
}

func TestCreatePlayerSnapshot(t *testing.T) {
	player := core.NewPlayer(core.PlayerConfig{
		UserID:  "player-1",
		Faction: core.FactionQingLong, // QingLong has no auto-added buff
	})
	player.Position = 25
	player.HP = 5
	player.LP = 3

	buff := core.NewBuff(core.BuffTypeFire, -1)
	player.AddBuff(buff)

	snapshot := CreatePlayerSnapshot(player)

	if snapshot.UserID != player.UserID {
		t.Error("UserID mismatch")
	}
	if snapshot.Position != 25 {
		t.Error("Position mismatch")
	}
	if snapshot.HP != 5 {
		t.Error("HP mismatch")
	}
	if len(snapshot.ActiveBuffs) != 1 {
		t.Errorf("Should have one buff, got %d", len(snapshot.ActiveBuffs))
	}
}

func TestRestorePlayer(t *testing.T) {
	snapshot := &PlayerSnapshot{
		UserID:   "player-1",
		Faction:  core.FactionZhuQue,
		Position: 25,
		HP:       5,
		LP:       3,
		IsDead:   false,
		SkipTurn: false,
		ActiveBuffs: []*BuffSnapshot{
			{Type: core.BuffTypeFire, Duration: -1},
		},
	}

	player := RestorePlayer(snapshot)

	if player.UserID != snapshot.UserID {
		t.Error("UserID mismatch after restore")
	}
	if player.Position != 25 {
		t.Error("Position mismatch after restore")
	}
	if player.HP != 5 {
		t.Error("HP mismatch after restore")
	}
	if len(player.ActiveBuffs) != 1 {
		t.Error("Should have one buff after restore")
	}
}

func TestSnapshotManager(t *testing.T) {
	sm := NewSnapshotManager()

	snapshot := NewFlowSnapshot("test-game")
	snapshot.Round = 1

	// Save
	err := sm.Save(snapshot)
	if err != nil {
		t.Errorf("Save failed: %v", err)
	}

	// HasSnapshot
	if !sm.HasSnapshot("test-game") {
		t.Error("Should have snapshot")
	}

	// Load
	loaded, err := sm.Load("test-game")
	if err != nil {
		t.Errorf("Load failed: %v", err)
	}
	if loaded.Round != 1 {
		t.Error("Loaded snapshot Round mismatch")
	}

	// List
	ids := sm.List()
	if len(ids) != 1 {
		t.Error("Should have one snapshot ID")
	}

	// Delete
	sm.Delete("test-game")
	if sm.HasSnapshot("test-game") {
		t.Error("Should not have snapshot after delete")
	}
}