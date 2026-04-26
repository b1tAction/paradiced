package hsm

import (
	"testing"

	"github.com/b1tAction/paradiced/pkg/constants"
)

// ========== Harness Helper Method Tests ==========

func TestHarnessAddItemToPlayer(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		PlayerCount: 2,
		MapLength:   20,
		Seed:        42,
	})

	player := harness.Players[0]
	harness.AddItemToPlayer(player, constants.ItemTypeAnyDoor)

	if !player.HasItem(constants.ItemTypeAnyDoor) {
		t.Error("Player should have AnyDoor item after AddItemToPlayer")
	}
}

func TestHarnessVerifyPlayerHP(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		PlayerCount: 2,
		MapLength:   20,
		Seed:        42,
	})

	player := harness.Players[0]
	player.HP = 4

	if !harness.VerifyPlayerHP(player, 4) {
		t.Error("VerifyPlayerHP should return true for matching HP")
	}
	if harness.VerifyPlayerHP(player, 5) {
		t.Error("VerifyPlayerHP should return false for non-matching HP")
	}
}

func TestHarnessVerifyPlayerLP(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		PlayerCount: 2,
		MapLength:   20,
		Seed:        42,
	})

	player := harness.Players[0]
	player.LP = 6

	if !harness.VerifyPlayerLP(player, 6) {
		t.Error("VerifyPlayerLP should return true for matching LP")
	}
	if harness.VerifyPlayerLP(player, 3) {
		t.Error("VerifyPlayerLP should return false for non-matching LP")
	}
}

func TestHarnessVerifyPlayerPosition(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		PlayerCount: 2,
		MapLength:   20,
		Seed:        42,
	})

	player := harness.Players[0]
	player.Position = 15

	if !harness.VerifyPlayerPosition(player, 15) {
		t.Error("VerifyPlayerPosition should return true for matching position")
	}
	if harness.VerifyPlayerPosition(player, 0) {
		t.Error("VerifyPlayerPosition should return false for non-matching position")
	}
}

func TestHarnessVerifyBuffNotOnPlayer(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		PlayerCount: 2,
		MapLength:   20,
		Seed:        42,
	})

	player := harness.Players[0]

	if !harness.VerifyBuffNotOnPlayer(player, constants.BuffTypeCurse) {
		t.Error("VerifyBuffNotOnPlayer should return true when buff not present")
	}

	harness.AddBuffToPlayer(player, constants.BuffTypeCurse, 3)
	if harness.VerifyBuffNotOnPlayer(player, constants.BuffTypeCurse) {
		t.Error("VerifyBuffNotOnPlayer should return false when buff is present")
	}
}

func TestHarnessGetGameLogEntries(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		PlayerCount: 2,
		MapLength:   20,
		Seed:        42,
	})

	// Without starting a turn, entries may be nil
	entries := harness.GetGameLogEntries()
	// After starting a turn, entries should be available
	harness.Game.Log.StartTurn(1, 0, harness.Players[0].ID.UUID())
	entries = harness.GetGameLogEntries()
	if len(entries) != 0 {
		t.Errorf("GetGameLogEntries should return 0 entries with no log entries added, got %d", len(entries))
	}
}

func TestHarnessGetGameLogSegments(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		PlayerCount: 2,
		MapLength:   20,
		Seed:        42,
	})

	segments := harness.GetGameLogSegments()
	if segments == nil {
		t.Error("GetGameLogSegments should not return nil")
	}
}

func TestHarnessRunPlayerTurnWithBuff(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		PlayerCount: 2,
		MapLength:   20,
		Seed:        42,
	})

	err := harness.RunPlayerTurnWithBuff(0, 3, constants.BuffTypeDivine, 3)
	if err != nil {
		t.Errorf("RunPlayerTurnWithBuff failed: %v", err)
	}

	player := harness.Players[0]
	if !player.HasBuff(constants.BuffTypeDivine) {
		t.Error("Player should have Divine buff after RunPlayerTurnWithBuff")
	}
}

func TestHarnessKillPlayer(t *testing.T) {
	harness := NewGameTestHarness(&HarnessConfig{
		PlayerCount: 2,
		MapLength:   20,
		Seed:        42,
	})

	player := harness.Players[0]
	harness.KillPlayer(player)

	if !player.IsDead {
		t.Error("Player should be dead after KillPlayer")
	}
	if player.HP != 0 {
		t.Errorf("Player HP = %d, expected 0 after KillPlayer", player.HP)
	}
}