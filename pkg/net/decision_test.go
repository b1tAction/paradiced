// Package net provides network message protocol definitions for client-server communication.
package net

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecisionJSON(t *testing.T) {
	decision := Decision{
		ID:      "dec-abc123",
		Prompt:  "是否使用神眷效果？",
		Context: "Buff_Divine",
		Options: []Option{
			{ID: "apply", Label: "应用效果", Effect: "LP+1"},
			{ID: "skip", Label: "跳过"},
		},
		Timeout: 30,
		Default: 0,
	}

	jsonBytes, err := json.Marshal(decision)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var parsed Decision
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if parsed.ID != "dec-abc123" {
		t.Errorf("parsed.ID = %s, want dec-abc123", parsed.ID)
	}
	if parsed.Prompt != "是否使用神眷效果？" {
		t.Errorf("parsed.Prompt = %s, want 是否使用神眷效果？", parsed.Prompt)
	}
	if parsed.Context != "Buff_Divine" {
		t.Errorf("parsed.Context = %s, want Buff_Divine", parsed.Context)
	}
	if len(parsed.Options) != 2 {
		t.Errorf("len(parsed.Options) = %d, want 2", len(parsed.Options))
	}
	if parsed.Timeout != 30 {
		t.Errorf("parsed.Timeout = %d, want 30", parsed.Timeout)
	}
}

func TestDecisionWithContext(t *testing.T) {
	decision := Decision{
		ID:      "dec-xyz",
		Prompt:  "选择目标玩家",
		Context: "Item_AnyDoor",
		Options: []Option{
			{ID: "p1", Label: "玩家1", Effect: "传送"},
		},
	}

	if decision.Context != "Item_AnyDoor" {
		t.Errorf("decision.Context = %s, want Item_AnyDoor", decision.Context)
	}
}

func TestOptionWithEffect(t *testing.T) {
	option := Option{
		ID:     "apply",
		Label:  "应用效果",
		Effect: "HP+1",
	}

	jsonBytes, err := json.Marshal(option)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var parsed Option
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if parsed.Effect != "HP+1" {
		t.Errorf("parsed.Effect = %s, want HP+1", parsed.Effect)
	}
}

func TestOptionEffectOmitempty(t *testing.T) {
	option := Option{
		ID:    "skip",
		Label: "跳过",
	}

	jsonBytes, err := json.Marshal(option)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	// effect field should be omitted when empty
	if strings.Contains(string(jsonBytes), `"effect"`) {
		t.Error("JSON should not contain 'effect' field when empty (omitempty)")
	}
}

func TestRollDiceStruct(t *testing.T) {
	// RollDice is empty struct
	roll := RollDice{}
	jsonBytes, err := json.Marshal(roll)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}
	if string(jsonBytes) != "{}" {
		t.Errorf("RollDice JSON = %s, want {}", string(jsonBytes))
	}
}

func TestUseItemJSON(t *testing.T) {
	useItem := UseItem{
		ItemID:   "item-abc123",
		TargetID: "player-001",
	}

	jsonBytes, err := json.Marshal(useItem)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var parsed UseItem
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if parsed.ItemID != "item-abc123" {
		t.Errorf("parsed.ItemID = %s, want item-abc123", parsed.ItemID)
	}
	if parsed.TargetID != "player-001" {
		t.Errorf("parsed.TargetID = %s, want player-001", parsed.TargetID)
	}
}

func TestUseItemWithoutTarget(t *testing.T) {
	useItem := UseItem{
		ItemID: "item-abc123",
	}

	jsonBytes, err := json.Marshal(useItem)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	// target_id should be omitted
	if strings.Contains(string(jsonBytes), `"target_id"`) {
		t.Error("JSON should not contain 'target_id' field when empty (omitempty)")
	}
}

func TestUseSkillStruct(t *testing.T) {
	// UseSkill is empty struct
	skill := UseSkill{}
	jsonBytes, err := json.Marshal(skill)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}
	if string(jsonBytes) != "{}" {
		t.Errorf("UseSkill JSON = %s, want {}", string(jsonBytes))
	}
}

func TestUserChoiceJSON(t *testing.T) {
	choice := UserChoice{
		DecisionID: "dec-abc123",
		Choice:     0,
	}

	jsonBytes, err := json.Marshal(choice)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var parsed UserChoice
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if parsed.DecisionID != "dec-abc123" {
		t.Errorf("parsed.DecisionID = %s, want dec-abc123", parsed.DecisionID)
	}
	if parsed.Choice != 0 {
		t.Errorf("parsed.Choice = %d, want 0", parsed.Choice)
	}
}

func TestMiniGameResultSubmitJSON(t *testing.T) {
	submit := MiniGameResultSubmit{
		Rankings: []RankingEntry{
			{PlayerID: "player-001", Rank: 1},
			{PlayerID: "player-002", Rank: 2},
		},
	}

	jsonBytes, err := json.Marshal(submit)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var parsed MiniGameResultSubmit
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if len(parsed.Rankings) != 2 {
		t.Errorf("len(parsed.Rankings) = %d, want 2", len(parsed.Rankings))
	}
}