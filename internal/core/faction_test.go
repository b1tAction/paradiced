package core

import (
	"testing"

	"github.com/b1tAction/paradiced/pkg/constants"
)

func TestGetFactionNames(t *testing.T) {
	names := GetFactionNames()
	if len(names) != 4 {
		t.Errorf("GetFactionNames() should return 4 factions, got %d", len(names))
	}
	if names[constants.FactionQingLong] != "青龙" {
		t.Errorf("FactionQingLong name = %s, expected 龍龙", names[constants.FactionQingLong])
	}
	if names[constants.FactionZhuQue] != "朱雀" {
		t.Errorf("FactionZhuQue name = %s, expected 朱雀", names[constants.FactionZhuQue])
	}
	if names[constants.FactionBaiHu] != "白虎" {
		t.Errorf("FactionBaiHu name = %s, expected 白虎", names[constants.FactionBaiHu])
	}
	if names[constants.FactionXuanWu] != "玄武" {
		t.Errorf("FactionXuanWu name = %s, expected 玄武", names[constants.FactionXuanWu])
	}
}

func TestGetFactionSkillName(t *testing.T) {
	tests := []struct {
		faction  constants.Faction
		expected string
	}{
		{constants.FactionQingLong, "行迹"},
		{constants.FactionZhuQue, "离火"},
		{constants.FactionBaiHu, "劫运"},
		{constants.FactionXuanWu, "镇厄"},
	}
	for _, tt := range tests {
		result := GetFactionSkillName(tt.faction)
		if result != tt.expected {
			t.Errorf("GetFactionSkillName(%s) = %s, expected %s", tt.faction, result, tt.expected)
		}
	}
}

func TestGetFactionSkillDesc(t *testing.T) {
	// Verify that each faction returns a non-empty description
	for _, faction := range constants.AllFactions() {
		desc := GetFactionSkillDesc(faction)
		if desc == "" {
			t.Errorf("GetFactionSkillDesc(%s) should return non-empty description", faction)
		}
	}
}