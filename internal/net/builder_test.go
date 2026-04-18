// Package net provides synchronization data builder for converting internal game structures to protocol messages.
package net

import (
	"testing"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/engine"
	"github.com/b1tAction/paradiced/internal/engine/hsm"
	pkgnet "github.com/b1tAction/paradiced/pkg/net"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/gamelog"
	"github.com/b1tAction/paradiced/pkg/id"
	"github.com/b1tAction/paradiced/pkg/rng"
	"github.com/b1tAction/paradiced/pkg/util"
)

// Helper function to create a test setup
func newTestBuilder() (*Builder, *engine.Game, *hsm.HSM) {
	game := engine.NewGame(id.NewGameID(), 12345)
	hsmInstance := hsm.NewHSM(game)
	hsm.RegisterGlobalStates(hsmInstance)
	hsm.RegisterTurnStates(hsmInstance)
	hsm.RegisterInterruptStates(hsmInstance)
	return NewBuilder(hsmInstance), game, hsmInstance
}

// Helper function to create a player
func newTestPlayer(faction constants.Faction) *core.Player {
	return core.NewPlayer(core.PlayerConfig{
		ID:      id.NewPlayerID(),
		Faction: faction,
	})
}

func TestBuildStateSync(t *testing.T) {
	builder, game, hsmInstance := newTestBuilder()

	player := newTestPlayer(constants.FactionQingLong)
	game.AddPlayer(player)
	hsmInstance.SetTurnPlayer(player)

	stateSync := builder.BuildStateSync()

	if stateSync.Round != 1 {
		t.Errorf("stateSync.Round = %d, want 1", stateSync.Round)
	}
	if len(stateSync.Players) != 1 {
		t.Errorf("len(stateSync.Players) = %d, want 1", len(stateSync.Players))
	}
	if stateSync.Players[0].UserID != player.ID.UUID() {
		t.Errorf("stateSync.Players[0].UserID = %s, want %s", stateSync.Players[0].UserID, player.ID.UUID())
	}
}

func TestBuildTurnSync(t *testing.T) {
	builder, game, hsmInstance := newTestBuilder()

	player := newTestPlayer(constants.FactionZhuQue)
	game.AddPlayer(player)
	hsmInstance.SetTurnPlayer(player)

	// Start turn log
	game.Log.StartTurn(1, 0, player.ID.UUID())

	// Add some log entries
	meta1 := util.NewMetadata()
	meta1.SetInt("lp_change", 1)
	entry1 := gamelog.NewActionEntryWithMetadata("modify_lp", player.ID.UUID(), "Buff_Divine", meta1)
	game.Log.AddEntry(entry1)

	meta := util.NewMetadata()
	meta.Set("path", []int{10, 11, 12, 13, 14, 15})
	entry2 := gamelog.NewActionEntryWithMetadata("move", player.ID.UUID(), "DiceRoll", meta)
	game.Log.AddEntry(entry2)

	turnSync := builder.BuildTurnSync()

	if turnSync.Round != 1 {
		t.Errorf("turnSync.Round = %d, want 1", turnSync.Round)
	}
	if len(turnSync.Entries) != 2 {
		t.Errorf("len(turnSync.Entries) = %d, want 2", len(turnSync.Entries))
	}
	if turnSync.Entries[0].ActionType != "modify_lp" {
		t.Errorf("turnSync.Entries[0].ActionType = %s, want modify_lp", turnSync.Entries[0].ActionType)
	}
	if turnSync.Entries[0].Metadata.GetIntOrDefault("lp_change", 0) != 1 {
		t.Errorf("turnSync.Entries[0].lp_change = %d, want 1", turnSync.Entries[0].Metadata.GetIntOrDefault("lp_change", 0))
	}
	if turnSync.Entries[1].ActionType != "move" {
		t.Errorf("turnSync.Entries[1].ActionType = %s, want move", turnSync.Entries[1].ActionType)
	}
}

func TestBuildTurnSyncWithMetadata(t *testing.T) {
	builder, game, hsmInstance := newTestBuilder()

	player := newTestPlayer(constants.FactionQingLong)
	game.AddPlayer(player)
	hsmInstance.SetTurnPlayer(player)

	game.Log.StartTurn(1, 0, player.ID.UUID())

	// Test that move entry metadata preserves path
	meta := util.NewMetadata()
	meta.Set("path", []int{5, 6, 7, 8})
	meta.SetInt("dice_steps", 3)
	meta.SetString("dice_type", "silver")

	entry := gamelog.NewActionEntryWithMetadata("move", player.ID.UUID(), "DiceRoll", meta)
	game.Log.AddEntry(entry)

	turnSync := builder.BuildTurnSync()

	if len(turnSync.Entries) != 1 {
		t.Fatalf("len(turnSync.Entries) = %d, want 1", len(turnSync.Entries))
	}

	// Verify entry has metadata
	if turnSync.Entries[0].Metadata == nil {
		t.Fatal("turnSync.Entries[0].Metadata should not be nil")
	}

	// Verify metadata fields preserved
	path := turnSync.Entries[0].Metadata.GetIntOrDefault("dice_steps", 0)
	if path != 3 {
		t.Errorf("dice_steps = %d, want 3", path)
	}

	diceType := turnSync.Entries[0].Metadata.GetStringOrDefault("dice_type", "")
	if diceType != "silver" {
		t.Errorf("dice_type = %s, want silver", diceType)
	}
}

func TestBuildPlayers(t *testing.T) {
	builder, game, _ := newTestBuilder()

	player := newTestPlayer(constants.FactionZhuQue)
	player.LP = 6
	player.Position = 25
	game.AddPlayer(player)

	players := builder.BuildPlayers()

	if len(players) != 1 {
		t.Errorf("len(players) = %d, want 1", len(players))
	}
	if players[0].Faction != "zhu_que" {
		t.Errorf("players[0].Faction = %s, want zhu_que", players[0].Faction)
	}
	if players[0].LP != 6 {
		t.Errorf("players[0].LP = %d, want 6", players[0].LP)
	}
	if players[0].Position != 25 {
		t.Errorf("players[0].Position = %d, want 25", players[0].Position)
	}
}

func TestBuildBuffsWithName(t *testing.T) {
	builder, _, _ := newTestBuilder()

	player := newTestPlayer(constants.FactionQingLong)
	divineBuff := core.NewBuff(constants.BuffTypeDivine, 3)
	player.AddBuff(divineBuff)

	buffs := builder.BuildBuffs(player.ActiveBuffs)

	if len(buffs) != 1 {
		t.Errorf("len(buffs) = %d, want 1", len(buffs))
	}
	if buffs[0].Type != "divine" {
		t.Errorf("buffs[0].Type = %s, want divine", buffs[0].Type)
	}
	if buffs[0].Name != "神眷" {
		t.Errorf("buffs[0].Name = %s, want 神眷", buffs[0].Name)
	}
	if buffs[0].Duration != 3 {
		t.Errorf("buffs[0].Duration = %d, want 3", buffs[0].Duration)
	}
}

func TestBuildItemsWithName(t *testing.T) {
	builder, _, _ := newTestBuilder()

	player := newTestPlayer(constants.FactionQingLong)
	anyDoorItem := core.NewItem(constants.ItemTypeAnyDoor)
	player.AddItem(anyDoorItem)

	items := builder.BuildItems(player.Inventory)

	if len(items) != 1 {
		t.Errorf("len(items) = %d, want 1", len(items))
	}
	if items[0].Type != "any_door" {
		t.Errorf("items[0].Type = %s, want any_door", items[0].Type)
	}
	if items[0].Name != "任意门" {
		t.Errorf("items[0].Name = %s, want 任意门", items[0].Name)
	}
}

func TestBuildAvailable(t *testing.T) {
	builder, game, hsmInstance := newTestBuilder()

	player := newTestPlayer(constants.FactionQingLong)
	anyDoorItem := core.NewItem(constants.ItemTypeAnyDoor)
	anyDoorItem.Usable = true
	player.AddItem(anyDoorItem)
	game.AddPlayer(player)
	hsmInstance.SetTurnPlayer(player)

	builder.SetDiceTypeFromRng(rng.DiceTypeGold)
	available := builder.BuildAvailableForPlayer(player)

	if len(available.Items) != 1 {
		t.Errorf("len(available.Items) = %d, want 1", len(available.Items))
	}
	if available.DiceType != "gold" {
		t.Errorf("available.DiceType = %s, want gold", available.DiceType)
	}
	// QingLong charge starts at 0
	if available.CanUseSkill != false {
		t.Errorf("available.CanUseSkill = %v, want false", available.CanUseSkill)
	}
}

func TestBuildAvailableWithCharge(t *testing.T) {
	builder, game, hsmInstance := newTestBuilder()

	player := newTestPlayer(constants.FactionQingLong)
	player.SetChargeCount(1) // Set charge >= 1
	game.AddPlayer(player)
	hsmInstance.SetTurnPlayer(player)

	builder.SetDiceTypeFromRng(rng.DiceTypeGold)
	available := builder.BuildAvailableForPlayer(player)

	if available.CanUseSkill != true {
		t.Errorf("available.CanUseSkill = %v, want true (charge >= 1)", available.CanUseSkill)
	}
}

func TestBuildFullSync(t *testing.T) {
	builder, game, hsmInstance := newTestBuilder()

	player := newTestPlayer(constants.FactionQingLong)
	game.AddPlayer(player)
	hsmInstance.SetTurnPlayer(player)

	game.Log.StartTurn(1, 0, player.ID.UUID())
	game.Log.AddEntry(gamelog.NewActionEntry("heal", player.ID.UUID(), "Test"))

	stateSync, turnSync := builder.BuildFullSync()

	if stateSync == nil {
		t.Fatal("stateSync should not be nil")
	}
	if turnSync == nil {
		t.Fatal("turnSync should not be nil")
	}
	if len(turnSync.Entries) != 1 {
		t.Errorf("len(turnSync.Entries) = %d, want 1", len(turnSync.Entries))
	}
}

func TestBuildDecision(t *testing.T) {
	builder, _, _ := newTestBuilder()

	options := []pkgnet.Option{
		{ID: "apply", Label: "应用", Effect: "HP+1"},
		{ID: "skip", Label: "跳过"},
	}

	decision := builder.BuildDecision("dec-001", "是否应用效果？", "Buff_Divine", options, 30, 0)

	if decision.ID != "dec-001" {
		t.Errorf("decision.ID = %s, want dec-001", decision.ID)
	}
	if decision.Prompt != "是否应用效果？" {
		t.Errorf("decision.Prompt = %s, want 是否应用效果？", decision.Prompt)
	}
	if decision.Context != "Buff_Divine" {
		t.Errorf("decision.Context = %s, want Buff_Divine", decision.Context)
	}
	if len(decision.Options) != 2 {
		t.Errorf("len(decision.Options) = %d, want 2", len(decision.Options))
	}
}

func TestGetCurrentTurnEntries(t *testing.T) {
	builder, game, hsmInstance := newTestBuilder()

	player := newTestPlayer(constants.FactionQingLong)
	game.AddPlayer(player)
	hsmInstance.SetTurnPlayer(player)

	// Without starting turn, should return nil
	entries := builder.GetCurrentTurnEntries()
	if entries != nil {
		t.Errorf("GetCurrentTurnEntries() = %v, want nil before StartTurn", entries)
	}

	// Start turn and add entries
	game.Log.StartTurn(1, 0, player.ID.UUID())
	game.Log.AddEntry(gamelog.NewActionEntry("damage", player.ID.UUID(), "Test"))

	entries = builder.GetCurrentTurnEntries()
	if len(entries) != 1 {
		t.Errorf("len(GetCurrentTurnEntries()) = %d, want 1", len(entries))
	}
}