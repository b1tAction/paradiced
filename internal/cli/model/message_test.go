// Package model provides data models for CLI protocol handling.
package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/b1tAction/paradiced/pkg/gamelog"
)

// ========== Message Parse Tests ==========

func TestParseMessage(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		data := `{"op_code": 1, "timestamp": 1234567890, "data": {"key": "value"}}`
		msg, err := ParseMessage([]byte(data))
		if err != nil {
			t.Fatalf("ParseMessage failed: %v", err)
		}
		if msg.OpCode != 1 {
			t.Errorf("OpCode = %d, expected 1", msg.OpCode)
		}
		if msg.Timestamp != 1234567890 {
			t.Errorf("Timestamp = %d, expected 1234567890", msg.Timestamp)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		data := `invalid json`
		_, err := ParseMessage([]byte(data))
		if err == nil {
			t.Error("ParseMessage should return error for invalid JSON")
		}
	})

	t.Run("empty data", func(t *testing.T) {
		data := `{"op_code": 1, "timestamp": 0, "data": null}`
		msg, err := ParseMessage([]byte(data))
		if err != nil {
			t.Fatalf("ParseMessage failed: %v", err)
		}
		if msg.OpCode != 1 {
			t.Errorf("OpCode = %d, expected 1", msg.OpCode)
		}
	})
}

func TestMessageParseData(t *testing.T) {
	t.Run("parse to struct", func(t *testing.T) {
		msg := &Message{
			OpCode: 1,
			Data:   []byte(`{"global_state": "match_init", "round": 1}`),
		}

		var stateSync StateSync
		err := msg.ParseData(&stateSync)
		if err != nil {
			t.Fatalf("ParseData failed: %v", err)
		}
		if stateSync.GlobalState != "match_init" {
			t.Errorf("GlobalState = %s, expected match_init", stateSync.GlobalState)
		}
		if stateSync.Round != 1 {
			t.Errorf("Round = %d, expected 1", stateSync.Round)
		}
	})

	t.Run("empty data", func(t *testing.T) {
		msg := &Message{
			OpCode: 1,
			Data:   nil,
		}

		var stateSync StateSync
		err := msg.ParseData(&stateSync)
		if err != nil {
			t.Fatalf("ParseData with empty data should not error: %v", err)
		}
	})

	t.Run("invalid JSON data", func(t *testing.T) {
		msg := &Message{
			OpCode: 1,
			Data:   []byte(`invalid json`),
		}

		var stateSync StateSync
		err := msg.ParseData(&stateSync)
		if err == nil {
			t.Error("ParseData should return error for invalid JSON")
		}
	})
}

// ========== StateSync Tests ==========

func TestStateSyncJSON(t *testing.T) {
	stateSync := StateSync{
		GlobalState:     "turn_loop",
		TurnState:       "main_action",
		CurrentPlayerID: "player-001",
		Round:           1,
		Turn:            2,
		Paused:          false,
		Players: []Player{
			{
				PlayerID:    "player-001",
				Faction:     "qing_long",
				Position:    10,
				HP:          8,
				LP:          5,
				Charge:      2,
				FireCounter: 0,
				IsDead:      false,
				SkipTurn:    false,
			},
		},
	}

	// Serialize
	data, err := json.Marshal(stateSync)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Deserialize
	var parsed StateSync
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.GlobalState != stateSync.GlobalState {
		t.Errorf("GlobalState = %s, expected %s", parsed.GlobalState, stateSync.GlobalState)
	}
	if parsed.CurrentPlayerID != stateSync.CurrentPlayerID {
		t.Errorf("CurrentPlayerID = %s, expected %s", parsed.CurrentPlayerID, stateSync.CurrentPlayerID)
	}
	if len(parsed.Players) != 1 {
		t.Errorf("Players count = %d, expected 1", len(parsed.Players))
	}
	if parsed.Players[0].PlayerID != "player-001" {
		t.Errorf("PlayerID = %s, expected player-001", parsed.Players[0].PlayerID)
	}
}

func TestStateSyncEmptyPlayers(t *testing.T) {
	stateSync := StateSync{
		GlobalState: "match_init",
		Round:        0,
		Turn:         0,
		Players:      []Player{},
	}

	data, err := json.Marshal(stateSync)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed StateSync
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(parsed.Players) != 0 {
		t.Errorf("Players count = %d, expected 0", len(parsed.Players))
	}
}

// ========== Player Tests ==========

func TestPlayerJSON(t *testing.T) {
	player := Player{
		PlayerID:    "player-001",
		Faction:     "zhu_que",
		Position:    25,
		HP:          10,
		LP:          8,
		Charge:      0,
		FireCounter: 4,
		IsDead:      false,
		SkipTurn:    true,
	}

	data, err := json.Marshal(player)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Player
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.PlayerID != player.PlayerID {
		t.Errorf("PlayerID = %s, expected %s", parsed.PlayerID, player.PlayerID)
	}
	if parsed.Faction != player.Faction {
		t.Errorf("Faction = %s, expected %s", parsed.Faction, player.Faction)
	}
	if parsed.SkipTurn != player.SkipTurn {
		t.Errorf("SkipTurn = %v, expected %v", parsed.SkipTurn, player.SkipTurn)
	}
}

func TestPlayerWithBuffsItems(t *testing.T) {
	player := Player{
		PlayerID: "player-001",
		Faction:  "qing_long",
		Buffs: []Buff{
			{Type: "divine", Name: "神眷", Duration: 3},
			{Type: "curse", Name: "诅咒", Duration: 2},
		},
		Items: []Item{
			{ID: "item-001", Type: "reverse_clock", Name: "逆流沙漏"},
		},
	}

	data, err := json.Marshal(player)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Player
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(parsed.Buffs) != 2 {
		t.Errorf("Buffs count = %d, expected 2", len(parsed.Buffs))
	}
	if parsed.Buffs[0].Type != "divine" {
		t.Errorf("Buff Type = %s, expected divine", parsed.Buffs[0].Type)
	}
	if len(parsed.Items) != 1 {
		t.Errorf("Items count = %d, expected 1", len(parsed.Items))
	}
	if parsed.Items[0].ID != "item-001" {
		t.Errorf("Item ID = %s, expected item-001", parsed.Items[0].ID)
	}
}

// ========== Buff Tests ==========

func TestBuffJSON(t *testing.T) {
	buff := Buff{
		Type:     "hidden",
		Name:     "隐匿",
		Duration: 5,
	}

	data, err := json.Marshal(buff)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Buff
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.Type != buff.Type {
		t.Errorf("Type = %s, expected %s", parsed.Type, buff.Type)
	}
	if parsed.Duration != buff.Duration {
		t.Errorf("Duration = %d, expected %d", parsed.Duration, buff.Duration)
	}
}

// ========== Item Tests ==========

func TestItemJSON(t *testing.T) {
	item := Item{
		ID:   "item-uuid-123",
		Type: "any_door",
		Name: "任意门",
	}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Item
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.ID != item.ID {
		t.Errorf("ID = %s, expected %s", parsed.ID, item.ID)
	}
	if parsed.Type != item.Type {
		t.Errorf("Type = %s, expected %s", parsed.Type, item.Type)
	}
}

// ========== Decision Tests ==========

func TestDecisionJSON(t *testing.T) {
	decision := Decision{
		ID:      "dec-001",
		Prompt:  "选择一个选项",
		Context: "event_context",
		Options: []Option{
			{ID: "opt-1", Label: "选项1", Effect: "效果1"},
			{ID: "opt-2", Label: "选项2", Effect: "效果2"},
		},
		Timeout: 30,
		Default: 0,
	}

	data, err := json.Marshal(decision)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Decision
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.ID != decision.ID {
		t.Errorf("ID = %s, expected %s", parsed.ID, decision.ID)
	}
	if len(parsed.Options) != 2 {
		t.Errorf("Options count = %d, expected 2", len(parsed.Options))
	}
	if parsed.Timeout != decision.Timeout {
		t.Errorf("Timeout = %d, expected %d", parsed.Timeout, decision.Timeout)
	}
}

func TestDecisionEmptyOptions(t *testing.T) {
	decision := Decision{
		ID:      "dec-002",
		Prompt:  "无选项决策",
		Options: []Option{},
	}

	data, err := json.Marshal(decision)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Decision
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(parsed.Options) != 0 {
		t.Errorf("Options count = %d, expected 0", len(parsed.Options))
	}
}

// ========== Option Tests ==========

func TestOptionJSON(t *testing.T) {
	option := Option{
		ID:     "opt-001",
		Label:  "接受",
		Effect: "获得道具",
	}

	data, err := json.Marshal(option)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Option
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.ID != option.ID {
		t.Errorf("ID = %s, expected %s", parsed.ID, option.ID)
	}
	if parsed.Label != option.Label {
		t.Errorf("Label = %s, expected %s", parsed.Label, option.Label)
	}
}

func TestOptionWithoutEffect(t *testing.T) {
	option := Option{
		ID:    "opt-002",
		Label: "拒绝",
	}

	data, err := json.Marshal(option)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Option
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.Effect != "" {
		t.Errorf("Effect = %s, expected empty", parsed.Effect)
	}
}

// ========== Available Tests ==========

func TestAvailableJSON(t *testing.T) {
	available := Available{
		Items: []Item{
			{ID: "item-001", Type: "reverse_clock", Name: "逆流沙漏"},
		},
		CanUseSkill: true,
		DiceType:    "gold",
	}

	data, err := json.Marshal(available)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Available
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(parsed.Items) != 1 {
		t.Errorf("Items count = %d, expected 1", len(parsed.Items))
	}
	if parsed.CanUseSkill != true {
		t.Errorf("CanUseSkill = %v, expected true", parsed.CanUseSkill)
	}
	if parsed.DiceType != "gold" {
		t.Errorf("DiceType = %s, expected gold", parsed.DiceType)
	}
}

func TestAvailableNoItems(t *testing.T) {
	available := Available{
		Items:       []Item{},
		CanUseSkill: false,
		DiceType:    "normal",
	}

	data, err := json.Marshal(available)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Available
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(parsed.Items) != 0 {
		t.Errorf("Items count = %d, expected 0", len(parsed.Items))
	}
}

// ========== MiniGameStart Tests ==========

func TestMiniGameStartJSON(t *testing.T) {
	start := MiniGameStart{
		GameType: "dice_race",
		Players:  []string{"player-001", "player-002", "player-003"},
	}

	data, err := json.Marshal(start)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed MiniGameStart
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.GameType != start.GameType {
		t.Errorf("GameType = %s, expected %s", parsed.GameType, start.GameType)
	}
	if len(parsed.Players) != 3 {
		t.Errorf("Players count = %d, expected 3", len(parsed.Players))
	}
}

// ========== MiniGameResult Tests ==========

func TestMiniGameResultJSON(t *testing.T) {
	result := MiniGameResult{
		Rankings: []RankingEntry{
			{PlayerID: "player-001", Rank: 1},
			{PlayerID: "player-002", Rank: 2},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed MiniGameResult
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(parsed.Rankings) != 2 {
		t.Errorf("Rankings count = %d, expected 2", len(parsed.Rankings))
	}
	if parsed.Rankings[0].Rank != 1 {
		t.Errorf("Rank = %d, expected 1", parsed.Rankings[0].Rank)
	}
}

func TestRankingEntryJSON(t *testing.T) {
	entry := RankingEntry{
		PlayerID: "player-uuid",
		Rank:     3,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed RankingEntry
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.PlayerID != entry.PlayerID {
		t.Errorf("PlayerID = %s, expected %s", parsed.PlayerID, entry.PlayerID)
	}
	if parsed.Rank != entry.Rank {
		t.Errorf("Rank = %d, expected %d", parsed.Rank, entry.Rank)
	}
}

// ========== GameOver Tests ==========

func TestGameOverJSON(t *testing.T) {
	gameOver := GameOver{
		WinnerID: "player-001",
		Stats: []PlayerStats{
			{PlayerID: "player-001", RoundsWon: 3, EventsDrawn: 5, ItemsUsed: 2},
			{PlayerID: "player-002", RoundsWon: 1, EventsDrawn: 4, ItemsUsed: 1},
		},
	}

	data, err := json.Marshal(gameOver)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed GameOver
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.WinnerID != gameOver.WinnerID {
		t.Errorf("WinnerID = %s, expected %s", parsed.WinnerID, gameOver.WinnerID)
	}
	if len(parsed.Stats) != 2 {
		t.Errorf("Stats count = %d, expected 2", len(parsed.Stats))
	}
}

func TestPlayerStatsJSON(t *testing.T) {
	stats := PlayerStats{
		PlayerID:    "player-001",
		RoundsWon:   2,
		EventsDrawn: 10,
		ItemsUsed:   3,
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed PlayerStats
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.PlayerID != stats.PlayerID {
		t.Errorf("PlayerID = %s, expected %s", parsed.PlayerID, stats.PlayerID)
	}
	if parsed.RoundsWon != stats.RoundsWon {
		t.Errorf("RoundsWon = %d, expected %d", parsed.RoundsWon, stats.RoundsWon)
	}
}

// ========== Client Messages Tests ==========

func TestRollDiceJSON(t *testing.T) {
	rollDice := RollDice{}

	data, err := json.Marshal(rollDice)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Empty struct should serialize to {}
	if string(data) != "{}" {
		t.Errorf("RollDice JSON = %s, expected {}", string(data))
	}
}

func TestUseItemJSON(t *testing.T) {
	t.Run("with target", func(t *testing.T) {
		useItem := UseItem{
			ItemID:   "item-uuid-123",
			TargetID: "player-002",
		}

		data, err := json.Marshal(useItem)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		var parsed UseItem
		err = json.Unmarshal(data, &parsed)
		if err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}

		if parsed.ItemID != useItem.ItemID {
			t.Errorf("ItemID = %s, expected %s", parsed.ItemID, useItem.ItemID)
		}
		if parsed.TargetID != useItem.TargetID {
			t.Errorf("TargetID = %s, expected %s", parsed.TargetID, useItem.TargetID)
		}
	})

	t.Run("without target", func(t *testing.T) {
		useItem := UseItem{
			ItemID: "item-uuid-123",
		}

		data, err := json.Marshal(useItem)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		var parsed UseItem
		err = json.Unmarshal(data, &parsed)
		if err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}

		if parsed.TargetID != "" {
			t.Errorf("TargetID = %s, expected empty", parsed.TargetID)
		}
	})
}

func TestUseSkillJSON(t *testing.T) {
	useSkill := UseSkill{}

	data, err := json.Marshal(useSkill)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if string(data) != "{}" {
		t.Errorf("UseSkill JSON = %s, expected {}", string(data))
	}
}

func TestUserChoiceJSON(t *testing.T) {
	userChoice := UserChoice{
		DecisionID: "dec-uuid-123",
		Choice:     1,
	}

	data, err := json.Marshal(userChoice)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed UserChoice
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.DecisionID != userChoice.DecisionID {
		t.Errorf("DecisionID = %s, expected %s", parsed.DecisionID, userChoice.DecisionID)
	}
	if parsed.Choice != userChoice.Choice {
		t.Errorf("Choice = %d, expected %d", parsed.Choice, userChoice.Choice)
	}
}

func TestMiniGameDataSubmitJSON(t *testing.T) {
	submit := MiniGameDataSubmit{
		GameType: "dice_race",
		GameData: map[string]interface{}{"score": 100, "time": 3.5},
	}

	data, err := json.Marshal(submit)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed MiniGameDataSubmit
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.GameType != submit.GameType {
		t.Errorf("GameType = %s, expected %s", parsed.GameType, submit.GameType)
	}
}

// ========== ActionRejected Tests ==========

func TestActionRejectedJSON(t *testing.T) {
	rejected := ActionRejected{
		OpCode:  100,
		Reason:  "not_current_player",
		Message: "当前不是你的回合",
	}

	data, err := json.Marshal(rejected)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed ActionRejected
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.OpCode != rejected.OpCode {
		t.Errorf("OpCode = %d, expected %d", parsed.OpCode, rejected.OpCode)
	}
	if parsed.Reason != rejected.Reason {
		t.Errorf("Reason = %s, expected %s", parsed.Reason, rejected.Reason)
	}
	if parsed.Message != rejected.Message {
		t.Errorf("Message = %s, expected %s", parsed.Message, rejected.Message)
	}
}

// ========== TurnSync Tests ==========

func TestTurnSyncJSON(t *testing.T) {
	turnSync := TurnSync{
		Round:           1,
		Turn:            2,
		CurrentPlayerID: "player-001",
		Entries: []gamelog.LogEntry{
			{Type: "action", ActionType: "move"},
		},
	}

	data, err := json.Marshal(turnSync)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed TurnSync
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.Round != turnSync.Round {
		t.Errorf("Round = %d, expected %d", parsed.Round, turnSync.Round)
	}
	if parsed.CurrentPlayerID != turnSync.CurrentPlayerID {
		t.Errorf("CurrentPlayerID = %s, expected %s", parsed.CurrentPlayerID, turnSync.CurrentPlayerID)
	}
	if len(parsed.Entries) != 1 {
		t.Errorf("Entries count = %d, expected 1", len(parsed.Entries))
	}
}

func TestTurnSyncEmptyEntries(t *testing.T) {
	turnSync := TurnSync{
		Round:           1,
		Turn:            0,
		CurrentPlayerID: "player-001",
		Entries:         []gamelog.LogEntry{},
	}

	data, err := json.Marshal(turnSync)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed TurnSync
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(parsed.Entries) != 0 {
		t.Errorf("Entries count = %d, expected 0", len(parsed.Entries))
	}
}

// ========== LogEntryWithTime Tests ==========

func TestLogEntryWithTimeJSON(t *testing.T) {
	entry := LogEntryWithTime{
		LogEntry: gamelog.LogEntry{
			Type:       "action",
			ActionType: "damage",
		},
		Time: time.Now(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed LogEntryWithTime
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.Type != entry.Type {
		t.Errorf("Type = %s, expected %s", parsed.Type, entry.Type)
	}
	if parsed.ActionType != entry.ActionType {
		t.Errorf("ActionType = %s, expected %s", parsed.ActionType, entry.ActionType)
	}
}