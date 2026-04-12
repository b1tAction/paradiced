package engine

import (
	"testing"

	"github.com/b1tAction/Fated/internal/core"
	"github.com/b1tAction/Fated/pkg/event"
)

// ========== EventHandler Tests ==========

func TestHasCustomHandler(t *testing.T) {
	// 离火有定制处理器
	if !HasCustomHandler(core.BuffTypeFire) {
		t.Error("BuffTypeFire should have custom handler")
	}

	// 诅咒没有定制处理器（使用默认）
	if HasCustomHandler(core.BuffTypeCurse) {
		t.Error("BuffTypeCurse should not have custom handler")
	}

	// 隐匿没有定制处理器
	if HasCustomHandler(core.BuffTypeHidden) {
		t.Error("BuffTypeHidden should not have custom handler")
	}
}

func TestGetHandler(t *testing.T) {
	// 获取离火处理器
	handler := GetHandler(core.BuffTypeFire)
	if handler == nil {
		t.Error("GetHandler(Fire) should return handler")
	}

	// 获取诅咒处理器（无）
	handler = GetHandler(core.BuffTypeCurse)
	if handler != nil {
		t.Error("GetHandler(Curse) should return nil")
	}
}

func TestHandleZhuQueFire(t *testing.T) {
	// 创建朱雀玩家
	player := core.NewPlayer(core.PlayerConfig{
		UserID:  "player-001",
		Faction: core.FactionZhuQue,
		MaxLP:   5,
	})
	initialLP := player.LP

	// 创建上下文
	ctx := event.NewContext(player)

	// 执行离火处理器（BeforeTurn Phase）
	handleZhuQueFire(event.PhaseBeforeTurn, ctx)

	// 第一次执行，计数器应该为1，LP不变
	if player.FireCounter != 1 {
		t.Errorf("FireCounter = %d, expected 1", player.FireCounter)
	}
	if player.LP != initialLP {
		t.Errorf("LP should not change after first trigger")
	}

	// 执行3次（达到4次触发）
	for i := 0; i < 3; i++ {
		handleZhuQueFire(event.PhaseBeforeTurn, ctx)
	}

	// 4次后，LP应该+1，计数器归零
	if player.LP != initialLP+1 {
		t.Errorf("LP = %d, expected %d (initial+1)", player.LP, initialLP+1)
	}
	if player.FireCounter != 0 {
		t.Errorf("FireCounter = %d, expected 0 (reset)", player.FireCounter)
	}
}

func TestHandleZhuQueFireNonBeforeTurnPhase(t *testing.T) {
	// 离火只在 BeforeTurn Phase 执行
	player := core.NewPlayer(core.PlayerConfig{
		UserID:  "player-001",
		Faction: core.FactionZhuQue,
		MaxLP:   5,
	})
	ctx := event.NewContext(player)

	// 在其他 Phase 执行应该无效
	handleZhuQueFire(event.PhaseAfterTurn, ctx)
	if player.FireCounter != 0 {
		t.Errorf("FireCounter should be 0 when not BeforeTurn phase")
	}
	if player.LP != 5 {
		t.Errorf("LP should not change when not BeforeTurn phase")
	}
}

func TestExecuteDefaultBuffAction(t *testing.T) {
	// 测试默认处理器的 HP/LP 修改
	tests := []struct {
		name     string
		def      *core.BuffDefinition
		initHP   int
		initLP   int
		expectedHP int
		expectedLP int
	}{
		{
			name:     "诅咒 LP-1",
			def:      core.BuffTypeCurse.GetBuffDefinition(),
			initHP:   6,
			initLP:   5,
			expectedHP: 6,
			expectedLP: 4,
		},
		{
			name:     "神眷 LP+1",
			def:      core.BuffTypeDivine.GetBuffDefinition(),
			initHP:   6,
			initLP:   5,
			expectedHP: 6,
			expectedLP: 6,
		},
		{
			name:     "甘霖 HP+1",
			def:      core.BuffTypeRain.GetBuffDefinition(),
			initHP:   6,
			initLP:   5,
			expectedHP: 7,
			expectedLP: 5,
		},
		{
			name:     "腐化 HP-1",
			def:      core.BuffTypeCorrupt.GetBuffDefinition(),
			initHP:   6,
			initLP:   5,
			expectedHP: 5,
			expectedLP: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player := core.NewPlayer(core.PlayerConfig{
				UserID: "player-001",
				MaxHP:  tt.initHP,
				MaxLP:  tt.initLP,
			})
			player.HP = tt.initHP
			player.LP = tt.initLP

			executeDefaultBuffAction(tt.def, player)

			if player.HP != tt.expectedHP {
				t.Errorf("HP = %d, expected %d", player.HP, tt.expectedHP)
			}
			if player.LP != tt.expectedLP {
				t.Errorf("LP = %d, expected %d", player.LP, tt.expectedLP)
			}
		})
	}
}

func TestRegisterBuffHandler(t *testing.T) {
	// 测试注册新的处理器
	executed := false
	customHandler := func(phase event.Phase, ctx *event.Context) {
		executed = true
	}

	// 注册一个自定义处理器（使用一个未注册的 BuffType）
	RegisterBuffHandler(core.BuffTypeHidden, customHandler)

	// 验证已注册
	if !HasCustomHandler(core.BuffTypeHidden) {
		t.Error("Hidden should have custom handler after registration")
	}

	// 执行
	ctx := event.NewContext(nil)
	GetHandler(core.BuffTypeHidden)(event.PhasePreDamage, ctx)

	if !executed {
		t.Error("Custom handler should be executed")
	}

	// 清理（恢复原状态）
	delete(BuffHandlers, core.BuffTypeHidden)
}

func TestExecuteDefaultBuffActionWithHPDamage(t *testing.T) {
	// 测试 HPPerTurn 为负数时调用 ApplyDamage
	player := core.NewPlayer(core.PlayerConfig{
		UserID: "player-001",
		MaxHP:  6,
	})
	player.HP = 6

	def := core.BuffTypeCorrupt.GetBuffDefinition()
	def.HPPerTurn = -2 // 每回合扣2血

	executeDefaultBuffAction(def, player)

	if player.HP != 4 {
		t.Errorf("HP = %d, expected 4 (扣2血)", player.HP)
	}
}

func TestExecuteDefaultBuffActionWithHealing(t *testing.T) {
	// 测试 HPPerTurn 为正数时调用 Heal
	player := core.NewPlayer(core.PlayerConfig{
		UserID: "player-001",
		MaxHP:  6,
	})
	player.HP = 6

	def := core.BuffTypeRain.GetBuffDefinition()
	def.HPPerTurn = 2 // 每回合加2血

	executeDefaultBuffAction(def, player)

	if player.HP != 8 {
		t.Errorf("HP = %d, expected 8 (加2血)", player.HP)
	}
}