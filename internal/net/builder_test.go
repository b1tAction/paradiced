// Package net provides synchronization data builder for converting internal game structures to protocol messages.
package net

import (
	"testing"
	"time"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/engine"
	"github.com/b1tAction/paradiced/internal/engine/hsm"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/internal/gamemap"
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
	if stateSync.Players[0].PlayerID != player.ID.UUID() {
		t.Errorf("stateSync.Players[0].PlayerID = %s, want %s", stateSync.Players[0].PlayerID, player.ID.UUID())
	}
}

func TestBuildStateSyncWithEntries(t *testing.T) {
	builder, game, hsmInstance := newTestBuilder()

	player := newTestPlayer(constants.FactionZhuQue)
	game.AddPlayer(player)
	hsmInstance.SetTurnPlayer(player)

	// Start turn log
	game.Log.StartTurn(1, 0, player.ID.UUID())

	// Add some log entries
	meta1 := util.NewMetadata()
	meta1.SetInt("lp_change", 1)
	entry1 := gamelog.NewActionEntryWithMetadata("modify_lp", player.ID.UUID(), string(constants.SourceBuffDivine), meta1)
	game.Log.AddEntry(entry1)

	meta := util.NewMetadata()
	meta.Set("path", []int{10, 11, 12, 13, 14, 15})
	entry2 := gamelog.NewActionEntryWithMetadata("move", player.ID.UUID(), string(constants.SourceSystemDiceRoll), meta)
	game.Log.AddEntry(entry2)

	stateSync := builder.BuildStateSync()

	if stateSync.Round != 1 {
		t.Errorf("stateSync.Round = %d, want 1", stateSync.Round)
	}
	if len(stateSync.Entries) != 2 {
		t.Errorf("len(stateSync.Entries) = %d, want 2", len(stateSync.Entries))
	}
	if stateSync.Entries[0].ActionType != "modify_lp" {
		t.Errorf("stateSync.Entries[0].ActionType = %s, want modify_lp", stateSync.Entries[0].ActionType)
	}
	if stateSync.Entries[0].Metadata.GetIntOrDefault("lp_change", 0) != 1 {
		t.Errorf("stateSync.Entries[0].lp_change = %d, want 1", stateSync.Entries[0].Metadata.GetIntOrDefault("lp_change", 0))
	}
	if stateSync.Entries[1].ActionType != "move" {
		t.Errorf("stateSync.Entries[1].ActionType = %s, want move", stateSync.Entries[1].ActionType)
	}

	// Second BuildStateSync should return 0 new entries (already broadcasted)
	stateSync2 := builder.BuildStateSync()
	if len(stateSync2.Entries) != 0 {
		t.Errorf("len(stateSync2.Entries) = %d, want 0 (no new entries after MarkBroadcasted)", len(stateSync2.Entries))
	}
}

func TestBuildStateSyncIncrementalEntries(t *testing.T) {
	builder, game, hsmInstance := newTestBuilder()

	player := newTestPlayer(constants.FactionQingLong)
	game.AddPlayer(player)
	hsmInstance.SetTurnPlayer(player)

	game.Log.StartTurn(1, 0, player.ID.UUID())

	// First broadcast: 1 entry
	game.Log.AddEntry(gamelog.NewActionEntry("damage", player.ID.UUID(), "Test1"))
	stateSync1 := builder.BuildStateSync()
	if len(stateSync1.Entries) != 1 {
		t.Errorf("len(stateSync1.Entries) = %d, want 1", len(stateSync1.Entries))
	}

	// Add 2 more entries, second broadcast should only return the new 2
	game.Log.AddEntry(gamelog.NewActionEntry("heal", player.ID.UUID(), "Test2"))
	game.Log.AddEntry(gamelog.NewActionEntry("move", player.ID.UUID(), "Test3"))
	stateSync2 := builder.BuildStateSync()
	if len(stateSync2.Entries) != 2 {
		t.Errorf("len(stateSync2.Entries) = %d, want 2 (incremental)", len(stateSync2.Entries))
	}
	if stateSync2.Entries[0].ActionType != "heal" {
		t.Errorf("stateSync2.Entries[0].ActionType = %s, want heal", stateSync2.Entries[0].ActionType)
	}
	if stateSync2.Entries[1].ActionType != "move" {
		t.Errorf("stateSync2.Entries[1].ActionType = %s, want move", stateSync2.Entries[1].ActionType)
	}

	// Third broadcast: no new entries
	stateSync3 := builder.BuildStateSync()
	if len(stateSync3.Entries) != 0 {
		t.Errorf("len(stateSync3.Entries) = %d, want 0", len(stateSync3.Entries))
	}
}

func TestBuildStateSyncEntriesWithMetadata(t *testing.T) {
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

	entry := gamelog.NewActionEntryWithMetadata("move", player.ID.UUID(), string(constants.SourceSystemDiceRoll), meta)
	game.Log.AddEntry(entry)

	stateSync := builder.BuildStateSync()

	if len(stateSync.Entries) != 1 {
		t.Fatalf("len(stateSync.Entries) = %d, want 1", len(stateSync.Entries))
	}

	// Verify entry has metadata
	if stateSync.Entries[0].Metadata == nil {
		t.Fatal("stateSync.Entries[0].Metadata should not be nil")
	}

	// Verify metadata fields preserved
	diceSteps := stateSync.Entries[0].Metadata.GetIntOrDefault("dice_steps", 0)
	if diceSteps != 3 {
		t.Errorf("dice_steps = %d, want 3", diceSteps)
	}

	diceType := stateSync.Entries[0].Metadata.GetStringOrDefault("dice_type", "")
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

func TestBuildFullSyncStateSync(t *testing.T) {
	builder, game, hsmInstance := newTestBuilder()

	player := newTestPlayer(constants.FactionQingLong)
	game.AddPlayer(player)
	hsmInstance.SetTurnPlayer(player)

	game.Log.StartTurn(1, 0, player.ID.UUID())
	game.Log.AddEntry(gamelog.NewActionEntry("heal", player.ID.UUID(), "Test"))

	// MarkBroadcasted so incremental won't return entries
	stateSync := builder.BuildStateSync()
	if len(stateSync.Entries) != 1 {
		t.Errorf("first BuildStateSync.Entries = %d, want 1", len(stateSync.Entries))
	}

	// BuildFullSyncStateSync returns all current entries (not incremental)
	fullSyncState := builder.BuildFullSyncStateSync()
	if fullSyncState == nil {
		t.Fatal("BuildFullSyncStateSync should not return nil")
	}
	if len(fullSyncState.Entries) != 1 {
		t.Errorf("len(fullSyncState.Entries) = %d, want 1 (all current entries)", len(fullSyncState.Entries))
	}
}

func TestBuildDecision(t *testing.T) {
	builder, _, _ := newTestBuilder()

	options := []pkgnet.Option{
		{ID: "apply", Label: "应用", Effect: "HP+1"},
		{ID: "skip", Label: "跳过"},
	}

	decision := builder.BuildDecision("dec-001", "是否应用效果？", string(constants.SourceBuffDivine), options, 30, 0)

	if decision.ID != "dec-001" {
		t.Errorf("decision.ID = %s, want dec-001", decision.ID)
	}
	if decision.Prompt != "是否应用效果？" {
		t.Errorf("decision.Prompt = %s, want 是否应用效果？", decision.Prompt)
	}
	if decision.Context != string(constants.SourceBuffDivine) {
		t.Errorf("decision.Context = %s, want %s", decision.Context, string(constants.SourceBuffDivine))
	}
	if len(decision.Options) != 2 {
		t.Errorf("len(decision.Options) = %d, want 2", len(decision.Options))
	}
}

func TestGetNewEntriesAfterStartTurn(t *testing.T) {
	builder, game, hsmInstance := newTestBuilder()

	player := newTestPlayer(constants.FactionQingLong)
	game.AddPlayer(player)
	hsmInstance.SetTurnPlayer(player)

	// Without starting turn, BuildStateSync should have nil Entries
	stateSync := builder.BuildStateSync()
	if stateSync.Entries != nil {
		t.Errorf("stateSync.Entries = %v, want nil before StartTurn", stateSync.Entries)
	}

	// Start turn and add entries
	game.Log.StartTurn(1, 0, player.ID.UUID())
	game.Log.AddEntry(gamelog.NewActionEntry("damage", player.ID.UUID(), "Test"))

	stateSync = builder.BuildStateSync()
	if len(stateSync.Entries) != 1 {
		t.Errorf("len(stateSync.Entries) = %d, want 1", len(stateSync.Entries))
	}
}

func TestSetDiceType(t *testing.T) {
	builder, _, _ := newTestBuilder()

	// Set dice type via string format
	builder.SetDiceType("gold")
	if builder.turnDiceType != rng.DiceTypeGold {
		t.Errorf("SetDiceType gold: turnDiceType = %v, want DiceTypeGold", builder.turnDiceType)
	}

	builder.SetDiceType("silver")
	if builder.turnDiceType != rng.DiceTypeSilver {
		t.Errorf("SetDiceType silver: turnDiceType = %v, want DiceTypeSilver", builder.turnDiceType)
	}

	builder.SetDiceType("copper")
	if builder.turnDiceType != rng.DiceTypeCopper {
		t.Errorf("SetDiceType copper: turnDiceType = %v, want DiceTypeCopper", builder.turnDiceType)
	}

	builder.SetDiceType("wood")
	if builder.turnDiceType != rng.DiceTypeWood {
		t.Errorf("SetDiceType wood: turnDiceType = %v, want DiceTypeWood", builder.turnDiceType)
	}

	// Unknown dice type defaults to DiceTypeNone
	builder.SetDiceType("unknown")
	if builder.turnDiceType != rng.DiceTypeNone {
		t.Errorf("SetDiceType unknown: turnDiceType = %v, want DiceTypeNone", builder.turnDiceType)
	}
}

func TestBuildAvailableNilPlayer(t *testing.T) {
	builder, game, _ := newTestBuilder()

	// No turn player set
	game.AddPlayer(newTestPlayer(constants.FactionQingLong))

	available := builder.BuildAvailable()
	if available != nil {
		t.Errorf("BuildAvailable with no turn player = %v, want nil", available)
	}
}

func TestBuildAvailableWithNonChargeFaction(t *testing.T) {
	builder, game, hsmInstance := newTestBuilder()

	// ZhuQue doesn't have charge-based skill
	player := newTestPlayer(constants.FactionZhuQue)
	game.AddPlayer(player)
	hsmInstance.SetTurnPlayer(player)

	builder.SetDiceTypeFromRng(rng.DiceTypeCopper)
	available := builder.BuildAvailable()

	if available.CanUseSkill != false {
		t.Errorf("ZhuQue CanUseSkill = %v, want false (no charge mechanic)", available.CanUseSkill)
	}
	if available.DiceType != "copper" {
		t.Errorf("DiceType = %s, want copper", available.DiceType)
	}
}

func TestBuildAvailableWithUnusableItem(t *testing.T) {
	builder, game, hsmInstance := newTestBuilder()

	player := newTestPlayer(constants.FactionQingLong)
	item := core.NewItem(constants.ItemTypeAnyDoor)
	item.Usable = false // Mark item as unusable
	player.AddItem(item)
	game.AddPlayer(player)
	hsmInstance.SetTurnPlayer(player)

	builder.SetDiceTypeFromRng(rng.DiceTypeGold)
	available := builder.BuildAvailable()

	if len(available.Items) != 0 {
		t.Errorf("BuildAvailable with unusable item: len(Items) = %d, want 0", len(available.Items))
	}
}

func TestBuildDecisionFromEvent(t *testing.T) {
	builder, _, _ := newTestBuilder()

	decision := event.NewDecision("Choose action", []event.Option{
		{ID: "apply", Label: "应用"},
		{ID: "skip", Label: "跳过"},
	})
	decision.WithSource(string(constants.SourceBuffDivine), "buff")
	decision.WithTimeout(30*time.Second, 0)

	result := builder.BuildDecisionFromEvent(decision)

	if result == nil {
		t.Fatal("BuildDecisionFromEvent should not return nil")
	}
	if result.Prompt != "Choose action" {
		t.Errorf("Prompt = %s, want 'Choose action'", result.Prompt)
	}
	if result.Context != "buff_buff_divine" {
		t.Errorf("Context = %s, want 'buff_buff_divine'", result.Context)
	}
	if len(result.Options) != 2 {
		t.Errorf("len(Options) = %d, want 2", len(result.Options))
	}
	if result.Options[0].ID != "apply" {
		t.Errorf("Options[0].ID = %s, want 'apply'", result.Options[0].ID)
	}
	if result.Options[0].Label != "应用" {
		t.Errorf("Options[0].Label = %s, want '应用'", result.Options[0].Label)
	}
	if result.Options[1].ID != "skip" {
		t.Errorf("Options[1].ID = %s, want 'skip'", result.Options[1].ID)
	}
	if result.Timeout != 30 {
		t.Errorf("Timeout = %d, want 30", result.Timeout)
	}
	if result.Default != 0 {
		t.Errorf("Default = %d, want 0", result.Default)
	}
}

func TestBuildDecisionFromEventNil(t *testing.T) {
	builder, _, _ := newTestBuilder()

	result := builder.BuildDecisionFromEvent(nil)
	if result != nil {
		t.Errorf("BuildDecisionFromEvent(nil) = %v, want nil", result)
	}
}

func TestBuildDecisionFromEventNoSourceID(t *testing.T) {
	builder, _, _ := newTestBuilder()

	decision := event.NewDecision("Test", []event.Option{{ID: "a", Label: "A"}})
	decision.WithSource("", "event")

	result := builder.BuildDecisionFromEvent(decision)
	if result == nil {
		t.Fatal("BuildDecisionFromEvent should not return nil")
	}
	if result.Context != "event" {
		t.Errorf("Context with empty SourceID = %s, want 'event'", result.Context)
	}
}

func TestBuildMapInfo(t *testing.T) {
	builder, _, hsmInstance := newTestBuilder()

	// Create and set map engine with various cell types
	mapEngine := gamemap.NewMapEngine(20)
	mapEngine.SetCellType(5, constants.CellTypeFragile)
	mapEngine.SetCellType(10, constants.CellTypeFog)
	mapEngine.SetCellType(15, constants.CellTypeCheckpoint)
	mapEngine.SetCellType(19, constants.CellTypeBoss)
	hsmInstance.SetMapEngine(mapEngine)

	mapInfo := builder.BuildMapInfo()

	if mapInfo == nil {
		t.Fatal("BuildMapInfo should not return nil")
	}
	if mapInfo.Length != 20 {
		t.Errorf("mapInfo.Length = %d, want 20", mapInfo.Length)
	}
	if len(mapInfo.Cells) != 20 {
		t.Errorf("len(mapInfo.Cells) = %d, want 20", len(mapInfo.Cells))
	}
	if mapInfo.Cells[0].CellType != "normal" {
		t.Errorf("mapInfo.Cells[0].CellType = %s, want normal", mapInfo.Cells[0].CellType)
	}
	if mapInfo.Cells[5].CellType != "fragile" {
		t.Errorf("mapInfo.Cells[5].CellType = %s, want fragile", mapInfo.Cells[5].CellType)
	}
	if mapInfo.Cells[10].CellType != "fog" {
		t.Errorf("mapInfo.Cells[10].CellType = %s, want fog", mapInfo.Cells[10].CellType)
	}
	if mapInfo.Cells[15].CellType != "checkpoint" {
		t.Errorf("mapInfo.Cells[15].CellType = %s, want checkpoint", mapInfo.Cells[15].CellType)
	}
	if mapInfo.Cells[19].CellType != "boss" {
		t.Errorf("mapInfo.Cells[19].CellType = %s, want boss", mapInfo.Cells[19].CellType)
	}
}

func TestBuildMapInfoWithBrokenFragile(t *testing.T) {
	builder, _, hsmInstance := newTestBuilder()

	mapEngine := gamemap.NewMapEngine(10)
	mapEngine.SetCellType(5, constants.CellTypeFragile)
	mapEngine.Cells[5].IsBroken = true
	hsmInstance.SetMapEngine(mapEngine)

	mapInfo := builder.BuildMapInfo()

	if !mapInfo.Cells[5].IsBroken {
		t.Error("Fragile cell at index 5 should be broken in MapInfo")
	}
	if mapInfo.Cells[3].IsBroken {
		t.Error("Normal cell should not be broken in MapInfo")
	}
}

func TestBuildMapInfoNoMapEngine(t *testing.T) {
	builder, _, _ := newTestBuilder()

	// No map engine set
	mapInfo := builder.BuildMapInfo()

	if mapInfo == nil {
		t.Fatal("BuildMapInfo should not return nil even without map engine")
	}
	if mapInfo.Length != 0 {
		t.Errorf("mapInfo.Length = %d, want 0 (empty MapInfo)", mapInfo.Length)
	}
}

func TestFilterClientEntriesSkipsState(t *testing.T) {
	player := newTestPlayer(constants.FactionQingLong)

	entries := []gamelog.LogEntry{
		gamelog.NewActionEntry("damage", player.ID.UUID(), "Test1"),
		gamelog.NewStateEntry("TurnUpkeep", "MainAction", player.ID.UUID()),
		gamelog.NewActionEntry("heal", player.ID.UUID(), "Test2"),
		gamelog.NewStateEntry("MainAction", "TurnMoving", player.ID.UUID()),
		gamelog.NewActionEntry("move", player.ID.UUID(), "Test3"),
	}

	filtered := filterClientEntries(entries)

	if len(filtered) != 3 {
		t.Errorf("len(filtered) = %d, want 3 (state entries filtered out)", len(filtered))
	}
	for _, e := range filtered {
		if e.Type == constants.EntryTypeState {
			t.Error("filtered entries should not contain state type")
		}
	}
	// Verify order preserved: damage, heal, move
	if filtered[0].ActionType != "damage" {
		t.Errorf("filtered[0].ActionType = %s, want damage", filtered[0].ActionType)
	}
	if filtered[1].ActionType != "heal" {
		t.Errorf("filtered[1].ActionType = %s, want heal", filtered[1].ActionType)
	}
	if filtered[2].ActionType != "move" {
		t.Errorf("filtered[2].ActionType = %s, want move", filtered[2].ActionType)
	}
}

func TestFilterClientEntriesEmpty(t *testing.T) {
	filtered := filterClientEntries(nil)
	if filtered != nil {
		t.Errorf("filterClientEntries(nil) = %v, want nil", filtered)
	}

	empty := []gamelog.LogEntry{}
	filtered = filterClientEntries(empty)
	if len(filtered) != 0 {
		t.Errorf("filterClientEntries([]) len = %d, want 0", len(filtered))
	}
}

func TestBuildStateSyncFiltersStateEntries(t *testing.T) {
	builder, game, hsmInstance := newTestBuilder()

	player := newTestPlayer(constants.FactionQingLong)
	game.AddPlayer(player)
	hsmInstance.SetTurnPlayer(player)

	game.Log.StartTurn(1, 0, player.ID.UUID())

	// Add mixed entries: action + state
	game.Log.AddEntry(gamelog.NewActionEntry("damage", player.ID.UUID(), "Test"))
	game.Log.AddEntry(gamelog.NewStateEntry("TurnUpkeep", "MainAction", player.ID.UUID()))
	game.Log.AddEntry(gamelog.NewActionEntry("heal", player.ID.UUID(), "Test2"))

	stateSync := builder.BuildStateSync()
	if len(stateSync.Entries) != 2 {
		t.Errorf("len(stateSync.Entries) = %d, want 2 (state entry filtered)", len(stateSync.Entries))
	}
	if stateSync.Entries[0].ActionType != "damage" {
		t.Errorf("Entries[0].ActionType = %s, want damage", stateSync.Entries[0].ActionType)
	}
	if stateSync.Entries[1].ActionType != "heal" {
		t.Errorf("Entries[1].ActionType = %s, want heal", stateSync.Entries[1].ActionType)
	}
}

func TestBuildDefinitionsConfigBuffClassification(t *testing.T) {
	defs := BuildDefinitionsConfig()

	// Test IsFaction: only Fire (朱雀离火) should be faction
	if _, ok := defs.Buffs["fire"]; !ok {
		t.Fatal("missing fire buff in DefinitionsConfig")
	}
	fire := defs.Buffs["fire"]
	if !fire.IsFaction {
		t.Error("fire.IsFaction should be true (朱雀 faction passive)")
	}

	// Test other buffs are not faction
	for key, buff := range defs.Buffs {
		if key == "fire" {
			continue
		}
		if buff.IsFaction {
			t.Errorf("%s.IsFaction should be false", key)
		}
	}

	// Test IsDraw: Boss and Hidden buffs should NOT be drawable
	nonDrawBuffs := []string{"death_mark", "thorns", "fire"}
	for _, key := range nonDrawBuffs {
		if _, ok := defs.Buffs[key]; !ok {
			t.Fatalf("missing %s buff in DefinitionsConfig", key)
		}
		buff := defs.Buffs[key]
		if buff.IsDraw {
			t.Errorf("%s.IsDraw should be false (IsBoss/IsHidden/IsFaction)", key)
		}
	}

	// Test regular buffs are drawable
	drawBuffs := []string{"divine", "rain", "exorcism", "curse", "lost", "corrupt", "poison"}
	for _, key := range drawBuffs {
		if _, ok := defs.Buffs[key]; !ok {
			t.Fatalf("missing %s buff in DefinitionsConfig", key)
		}
		buff := defs.Buffs[key]
		if !buff.IsDraw {
			t.Errorf("%s.IsDraw should be true", key)
		}
	}

	// Test IsBoss: only Thorns and DeathMark are Boss buffs
	bossBuffs := []string{"thorns", "death_mark"}
	for _, key := range bossBuffs {
		if _, ok := defs.Buffs[key]; !ok {
			t.Fatalf("missing %s buff in DefinitionsConfig", key)
		}
		buff := defs.Buffs[key]
		if !buff.IsBoss {
			t.Errorf("%s.IsBoss should be true", key)
		}
	}

	// Test IsHidden: only DeathMark is hidden
	if _, ok := defs.Buffs["death_mark"]; !ok {
		t.Fatal("missing death_mark buff in DefinitionsConfig")
	}
	deathMark := defs.Buffs["death_mark"]
	if !deathMark.IsHidden {
		t.Error("death_mark.IsHidden should be true")
	}

	// Verify hidden is false for other buffs
	for key, buff := range defs.Buffs {
		if key == "death_mark" {
			continue
		}
		if buff.IsHidden {
			t.Errorf("%s.IsHidden should be false", key)
		}
	}
}

func TestBuildDefinitionsConfigBuffCategory(t *testing.T) {
	defs := BuildDefinitionsConfig()

	// Good buffs: divine, rain, exorcism, fire
	goodBuffs := []string{"divine", "rain", "exorcism", "fire"}
	for _, key := range goodBuffs {
		if _, ok := defs.Buffs[key]; !ok {
			t.Fatalf("missing %s buff in DefinitionsConfig", key)
		}
		buff := defs.Buffs[key]
		if buff.Category != "good" {
			t.Errorf("%s.Category = %s, want good", key, buff.Category)
		}
	}

	// Neutral buffs: hidden, thorns
	neutralBuffs := []string{"hidden", "thorns"}
	for _, key := range neutralBuffs {
		if _, ok := defs.Buffs[key]; !ok {
			t.Fatalf("missing %s buff in DefinitionsConfig", key)
		}
		buff := defs.Buffs[key]
		if buff.Category != "neutral" {
			t.Errorf("%s.Category = %s, want neutral", key, buff.Category)
		}
	}

	// Bad buffs: curse, lost, corrupt, poison, death_mark
	badBuffs := []string{"curse", "lost", "corrupt", "poison", "death_mark"}
	for _, key := range badBuffs {
		if _, ok := defs.Buffs[key]; !ok {
			t.Fatalf("missing %s buff in DefinitionsConfig", key)
		}
		buff := defs.Buffs[key]
		if buff.Category != "bad" {
			t.Errorf("%s.Category = %s, want bad", key, buff.Category)
		}
	}
}

func TestBuildDefinitionsConfigEventCategory(t *testing.T) {
	defs := BuildDefinitionsConfig()

	goodEvents := []string{"herb", "milk_tea", "relic", "divine_bless", "hidden_buff"}
	for _, key := range goodEvents {
		if _, ok := defs.Events[key]; !ok {
			t.Fatalf("missing %s event in DefinitionsConfig", key)
		}
		event := defs.Events[key]
		if event.Category != "good" {
			t.Errorf("%s.Category = %s, want good", key, event.Category)
		}
	}

	neutralEvents := []string{"exchange", "taste_test"}
	for _, key := range neutralEvents {
		if _, ok := defs.Events[key]; !ok {
			t.Fatalf("missing %s event in DefinitionsConfig", key)
		}
		event := defs.Events[key]
		if event.Category != "neutral" {
			t.Errorf("%s.Category = %s, want neutral", key, event.Category)
		}
	}

	badEvents := []string{"mosquito", "ghost_hit", "dog_poop", "thief", "curse_buddha", "lost_way", "thunder"}
	for _, key := range badEvents {
		if _, ok := defs.Events[key]; !ok {
			t.Fatalf("missing %s event in DefinitionsConfig", key)
		}
		event := defs.Events[key]
		if event.Category != "bad" {
			t.Errorf("%s.Category = %s, want bad", key, event.Category)
		}
	}
}

func TestBuildDefinitionsConfigCompleteness(t *testing.T) {
	defs := BuildDefinitionsConfig()

	// Verify all 14 events present
	if len(defs.Events) != 14 {
		t.Errorf("len(defs.Events) = %d, want 14", len(defs.Events))
	}
	// Verify all 11 buffs present
	if len(defs.Buffs) != 11 {
		t.Errorf("len(defs.Buffs) = %d, want 11", len(defs.Buffs))
	}
	// Verify all 3 items present
	if len(defs.Items) != 3 {
		t.Errorf("len(defs.Items) = %d, want 3", len(defs.Items))
	}

	// Verify all entries have required fields populated
	for key, event := range defs.Events {
		if event.Type == "" {
			t.Errorf("event %s: Type is empty", key)
		}
		if event.Name == "" {
			t.Errorf("event %s: Name is empty", key)
		}
		if event.Desc == "" {
			t.Errorf("event %s: Desc is empty", key)
		}
		if event.EnglishName == "" {
			t.Errorf("event %s: EnglishName is empty", key)
		}
		if event.Category == "" {
			t.Errorf("event %s: Category is empty", key)
		}
	}

	for key, buff := range defs.Buffs {
		if buff.Type == "" {
			t.Errorf("buff %s: Type is empty", key)
		}
		if buff.Name == "" {
			t.Errorf("buff %s: Name is empty", key)
		}
		if buff.Desc == "" {
			t.Errorf("buff %s: Desc is empty", key)
		}
		if buff.EnglishName == "" {
			t.Errorf("buff %s: EnglishName is empty", key)
		}
		if buff.Category == "" {
			t.Errorf("buff %s: Category is empty", key)
		}
	}

	for key, item := range defs.Items {
		if item.Type == "" {
			t.Errorf("item %s: Type is empty", key)
		}
		if item.Name == "" {
			t.Errorf("item %s: Name is empty", key)
		}
		if item.Desc == "" {
			t.Errorf("item %s: Desc is empty", key)
		}
		if item.EnglishName == "" {
			t.Errorf("item %s: EnglishName is empty", key)
		}
		if item.Category == "" {
			t.Errorf("item %s: Category is empty", key)
		}
	}
}

func TestBuildFullSyncFiltersStateEntries(t *testing.T) {
	builder, game, hsmInstance := newTestBuilder()

	player := newTestPlayer(constants.FactionQingLong)
	game.AddPlayer(player)
	hsmInstance.SetTurnPlayer(player)

	game.Log.StartTurn(1, 0, player.ID.UUID())

	game.Log.AddEntry(gamelog.NewActionEntry("damage", player.ID.UUID(), "Test"))
	game.Log.AddEntry(gamelog.NewStateEntry("MainAction", "TurnMoving", player.ID.UUID()))
	game.Log.AddEntry(gamelog.NewActionEntry("move", player.ID.UUID(), "Test2"))

	// MarkBroadcasted first so incremental won't return
	builder.BuildStateSync()

	fullSync := builder.BuildFullSyncStateSync()
	if len(fullSync.Entries) != 2 {
		t.Errorf("len(fullSync.Entries) = %d, want 2 (state entry filtered)", len(fullSync.Entries))
	}
	if fullSync.Entries[0].ActionType != "damage" {
		t.Errorf("Entries[0].ActionType = %s, want damage", fullSync.Entries[0].ActionType)
	}
	if fullSync.Entries[1].ActionType != "move" {
		t.Errorf("Entries[1].ActionType = %s, want move", fullSync.Entries[1].ActionType)
	}
}