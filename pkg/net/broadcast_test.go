// Package net provides network message protocol definitions for client-server communication.
package net

import (
	"testing"

	"github.com/b1tAction/paradiced/pkg/gamelog"
)

func TestMockBroadcastAdapterStateSync(t *testing.T) {
	mock := NewMockBroadcastAdapter()

	stateSync := &StateSync{
		GlobalState: "turn_loop",
		Round:       1,
	}

	err := mock.BroadcastStateSync(stateSync)
	if err != nil {
		t.Fatalf("BroadcastStateSync() error: %v", err)
	}
	if len(mock.StateSyncs) != 1 {
		t.Errorf("len(mock.StateSyncs) = %d, want 1", len(mock.StateSyncs))
	}
	if mock.StateSyncs[0].GlobalState != "turn_loop" {
		t.Errorf("mock.StateSyncs[0].GlobalState = %s, want turn_loop", mock.StateSyncs[0].GlobalState)
	}
}

func TestMockBroadcastAdapterTurnSync(t *testing.T) {
	mock := NewMockBroadcastAdapter()

	// Create LogEntry using gamelog
	entry := gamelog.NewActionEntry("damage", "player-001", "Cell_Fragile")

	turnSync := &TurnSync{
		Round:   1,
		Turn:    0,
		Player:  "player-001",
		Entries: []gamelog.LogEntry{entry},
	}

	err := mock.BroadcastTurnSync(turnSync)
	if err != nil {
		t.Fatalf("BroadcastTurnSync() error: %v", err)
	}
	if len(mock.TurnSyncs) != 1 {
		t.Errorf("len(mock.TurnSyncs) = %d, want 1", len(mock.TurnSyncs))
	}
	if mock.TurnSyncs[0].Player != "player-001" {
		t.Errorf("mock.TurnSyncs[0].Player = %s, want player-001", mock.TurnSyncs[0].Player)
	}
	if len(mock.TurnSyncs[0].Entries) != 1 {
		t.Errorf("len(mock.TurnSyncs[0].Entries) = %d, want 1", len(mock.TurnSyncs[0].Entries))
	}
	if mock.TurnSyncs[0].Entries[0].ActionType != "damage" {
		t.Errorf("mock.TurnSyncs[0].Entries[0].ActionType = %s, want damage", mock.TurnSyncs[0].Entries[0].ActionType)
	}
}

func TestMockBroadcastAdapterDecision(t *testing.T) {
	mock := NewMockBroadcastAdapter()

	decision := &Decision{
		ID:      "dec-001",
		Prompt:  "选择目标",
		Context: "Item_AnyDoor",
	}

	err := mock.SendDecision("player-001", decision)
	if err != nil {
		t.Fatalf("SendDecision() error: %v", err)
	}
	if mock.Decisions["player-001"] == nil {
		t.Fatal("mock.Decisions[player-001] should not be nil")
	}
	if mock.Decisions["player-001"].ID != "dec-001" {
		t.Errorf("mock.Decisions[player-001].ID = %s, want dec-001", mock.Decisions["player-001"].ID)
	}
	if mock.Decisions["player-001"].Context != "Item_AnyDoor" {
		t.Errorf("mock.Decisions[player-001].Context = %s, want Item_AnyDoor", mock.Decisions["player-001"].Context)
	}
}

func TestMockBroadcastAdapterAvailable(t *testing.T) {
	mock := NewMockBroadcastAdapter()

	available := &Available{
		Items:       []Item{{ID: "item-001", Type: "any_door", Name: "任意门"}},
		CanUseSkill: false,
		DiceType:    "gold",
	}

	err := mock.SendAvailable("player-001", available)
	if err != nil {
		t.Fatalf("SendAvailable() error: %v", err)
	}
	if mock.Availables["player-001"] == nil {
		t.Fatal("mock.Availables[player-001] should not be nil")
	}
	if mock.Availables["player-001"].DiceType != "gold" {
		t.Errorf("mock.Availables[player-001].DiceType = %s, want gold", mock.Availables["player-001"].DiceType)
	}
}

func TestMockBroadcastAdapterMiniGameStart(t *testing.T) {
	mock := NewMockBroadcastAdapter()

	start := &MiniGameStart{
		GameType: "dice_race",
		Players:  []string{"p1", "p2", "p3", "p4"},
	}

	err := mock.BroadcastMiniGameStart(start)
	if err != nil {
		t.Fatalf("BroadcastMiniGameStart() error: %v", err)
	}
	if len(mock.MiniGameStarts) != 1 {
		t.Errorf("len(mock.MiniGameStarts) = %d, want 1", len(mock.MiniGameStarts))
	}
	if mock.MiniGameStarts[0].GameType != "dice_race" {
		t.Errorf("mock.MiniGameStarts[0].GameType = %s, want dice_race", mock.MiniGameStarts[0].GameType)
	}
}

func TestMockBroadcastAdapterMiniGameResult(t *testing.T) {
	mock := NewMockBroadcastAdapter()

	result := &MiniGameResult{
		Rankings: []RankingEntry{
			{PlayerID: "p1", Rank: 1},
			{PlayerID: "p2", Rank: 2},
		},
	}

	err := mock.BroadcastMiniGameResult(result)
	if err != nil {
		t.Fatalf("BroadcastMiniGameResult() error: %v", err)
	}
	if len(mock.MiniGameResults) != 1 {
		t.Errorf("len(mock.MiniGameResults) = %d, want 1", len(mock.MiniGameResults))
	}
	if len(mock.MiniGameResults[0].Rankings) != 2 {
		t.Errorf("len(mock.MiniGameResults[0].Rankings) = %d, want 2", len(mock.MiniGameResults[0].Rankings))
	}
}

func TestMockBroadcastAdapterGameOver(t *testing.T) {
	mock := NewMockBroadcastAdapter()

	over := &GameOver{
		WinnerID: "player-001",
		Stats:    []PlayerStats{},
	}

	err := mock.BroadcastGameOver(over)
	if err != nil {
		t.Fatalf("BroadcastGameOver() error: %v", err)
	}
	if len(mock.GameOvers) != 1 {
		t.Errorf("len(mock.GameOvers) = %d, want 1", len(mock.GameOvers))
	}
	if mock.GameOvers[0].WinnerID != "player-001" {
		t.Errorf("mock.GameOvers[0].WinnerID = %s, want player-001", mock.GameOvers[0].WinnerID)
	}
}

func TestMockBroadcastAdapterFullSync(t *testing.T) {
	mock := NewMockBroadcastAdapter()

	stateSync := &StateSync{GlobalState: "turn_loop"}
	turnSync := &TurnSync{Round: 1, Player: "player-001"}

	err := mock.SendFullSync("player-001", stateSync, turnSync)
	if err != nil {
		t.Fatalf("SendFullSync() error: %v", err)
	}
	if mock.FullSyncs["player-001"].State == nil {
		t.Fatal("mock.FullSyncs[player-001].State should not be nil")
	}
	if mock.FullSyncs["player-001"].State.GlobalState != "turn_loop" {
		t.Errorf("mock.FullSyncs[player-001].State.GlobalState = %s, want turn_loop", mock.FullSyncs["player-001"].State.GlobalState)
	}
	if mock.FullSyncs["player-001"].Turn.Round != 1 {
		t.Errorf("mock.FullSyncs[player-001].Turn.Round = %d, want 1", mock.FullSyncs["player-001"].Turn.Round)
	}
}

func TestMockBroadcastAdapterClear(t *testing.T) {
	mock := NewMockBroadcastAdapter()

	// Add some messages
	mock.BroadcastStateSync(&StateSync{})
	mock.BroadcastTurnSync(&TurnSync{})
	mock.SendDecision("p1", &Decision{})
	mock.SendAvailable("p1", &Available{})
	mock.BroadcastMiniGameStart(&MiniGameStart{})
	mock.BroadcastMiniGameResult(&MiniGameResult{})
	mock.BroadcastGameOver(&GameOver{})
	mock.SendFullSync("p1", &StateSync{}, &TurnSync{})

	// Verify messages exist
	if len(mock.StateSyncs) != 1 {
		t.Error("mock.StateSyncs should have 1 entry before Clear")
	}
	if len(mock.TurnSyncs) != 1 {
		t.Error("mock.TurnSyncs should have 1 entry before Clear")
	}
	if mock.Decisions["p1"] == nil {
		t.Error("mock.Decisions[p1] should exist before Clear")
	}
	if mock.Availables["p1"] == nil {
		t.Error("mock.Availables[p1] should exist before Clear")
	}

	// Clear
	mock.Clear()

	// Verify all cleared
	if len(mock.StateSyncs) != 0 {
		t.Errorf("len(mock.StateSyncs) = %d after Clear, want 0", len(mock.StateSyncs))
	}
	if len(mock.TurnSyncs) != 0 {
		t.Errorf("len(mock.TurnSyncs) = %d after Clear, want 0", len(mock.TurnSyncs))
	}
	if mock.Decisions["p1"] != nil {
		t.Error("mock.Decisions[p1] should be nil after Clear")
	}
	if mock.Availables["p1"] != nil {
		t.Error("mock.Availables[p1] should be nil after Clear")
	}
	if len(mock.MiniGameStarts) != 0 {
		t.Errorf("len(mock.MiniGameStarts) = %d after Clear, want 0", len(mock.MiniGameStarts))
	}
	if len(mock.MiniGameResults) != 0 {
		t.Errorf("len(mock.MiniGameResults) = %d after Clear, want 0", len(mock.MiniGameResults))
	}
	if len(mock.GameOvers) != 0 {
		t.Errorf("len(mock.GameOvers) = %d after Clear, want 0", len(mock.GameOvers))
	}
}

func TestMockBroadcastAdapterMultipleCalls(t *testing.T) {
	mock := NewMockBroadcastAdapter()

	// Multiple broadcasts
	for i := 0; i < 5; i++ {
		mock.BroadcastStateSync(&StateSync{Round: i + 1})
		mock.BroadcastTurnSync(&TurnSync{Round: i + 1})
	}

	if len(mock.StateSyncs) != 5 {
		t.Errorf("len(mock.StateSyncs) = %d, want 5", len(mock.StateSyncs))
	}
	if len(mock.TurnSyncs) != 5 {
		t.Errorf("len(mock.TurnSyncs) = %d, want 5", len(mock.TurnSyncs))
	}
	if mock.StateSyncs[0].Round != 1 {
		t.Errorf("mock.StateSyncs[0].Round = %d, want 1", mock.StateSyncs[0].Round)
	}
	if mock.StateSyncs[4].Round != 5 {
		t.Errorf("mock.StateSyncs[4].Round = %d, want 5", mock.StateSyncs[4].Round)
	}
}