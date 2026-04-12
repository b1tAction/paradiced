// Package engine provides game engine logic for the Fated game.
package engine

import (
	"github.com/b1tAction/Fated/internal/core"
	"github.com/b1tAction/Fated/pkg/event"
)

// EventHandler 是高度定制化的 Buff 效果处理函数
// 通过策略模式，每个 Buff 可以有自己的专属处理逻辑
// 参数:
//   - phase: 当前触发的 Phase
//   - ctx: 事件上下文，包含 Player、Data 等信息
type EventHandler func(phase event.Phase, ctx *event.Context)

// BuffHandlers 是 Buff 处理策略注册表
// 将 BuffType 映射到其定制化的 EventHandler
// 如果 Buff 没有注册定制处理器，则使用默认的数值处理器
var BuffHandlers = map[core.BuffType]EventHandler{
	core.BuffTypeFire: handleZhuQueFire, // 朱雀离火：每4回合LP+1
	// 更多定制处理器可以在这里注册
	// 例如：
	// core.BuffTypeHidden: handleHiddenImmunity,    // 隐匿：免疫伤害
	// core.BuffTypeLost: handleLostReverse,         // 迈途：反向移动
}

// HasCustomHandler 检查 Buff 是否有定制处理器
func HasCustomHandler(buffType core.BuffType) bool {
	_, ok := BuffHandlers[buffType]
	return ok
}

// GetHandler 获取 Buff 的定制处理器
func GetHandler(buffType core.BuffType) EventHandler {
	if handler, ok := BuffHandlers[buffType]; ok {
		return handler
	}
	return nil
}

// ========== 定制处理器实现 ==========

// handleZhuQueFire 朱雀离火 Buff 的定制处理器
// 效果：每4回合 LP+1
// 这是离火 Buff 的特殊逻辑，需要计数器来追踪回合
func handleZhuQueFire(phase event.Phase, ctx *event.Context) {
	player, ok := ctx.Player.(*core.Player)
	if !ok {
		return
	}

	// 只在 BeforeTurn Phase 执行
	if phase != event.PhaseBeforeTurn {
		return
	}

	// 离火计数器递增
	player.FireCounter++

	// 每4回合增加1点幸运值
	if player.FireCounter >= 4 {
		player.ModifyLP(1)
		player.FireCounter = 0
	}
}

// ========== 默认处理器 ==========

// executeDefaultBuffAction 执行默认的 Buff 数值效果
// 用于没有定制处理器的 Buff，根据 HPPerTurn/LPPerTurn 修改数值
func executeDefaultBuffAction(def *core.BuffDefinition, player *core.Player) {
	// 修改 HP
	if def.HPPerTurn != 0 {
		if def.HPPerTurn > 0 {
			player.Heal(def.HPPerTurn)
		} else {
			player.ApplyDamage(-def.HPPerTurn) // HPPerTurn 为负数表示扣血
		}
	}

	// 修改 LP
	if def.LPPerTurn != 0 {
		player.ModifyLP(def.LPPerTurn)
	}
}

// ========== 注册新处理器 ==========

// RegisterBuffHandler 注册新的 Buff 处理器
// 允许外部扩展 BuffHandlers 注册表
func RegisterBuffHandler(buffType core.BuffType, handler EventHandler) {
	BuffHandlers[buffType] = handler
}