package event

// Phase 定义游戏中的触发时机
// 用于Buff、道具、阵营被动等效果的触发阶段分类
type Phase int

const (
	PhaseBeforeTurn Phase = iota // 回合开始前（神眷、诅咒 LP±1，离火每4回合LP+1）
	PhaseOnMove                  // 移动时（迷途反向）
	PhaseOnLand                  // 落地后（任意门、落点事件）
	PhasePreEvent                // 事件触发前（辟邪、玄武、护盾道具）
	PhasePreDamage               // 受伤前（隐匿、护盾）
	PhaseAfterTurn               // 回合结束后（甘霖/腐化 HP±1，TickDuration）
	PhaseAnyTime                 // 任何时候可用（道具主动使用）
	// 事件驱动型 Phase - Buff 生命周期事件
	PhaseOnBuffApplied           // 当任意 Buff 被挂载到玩家身上时触发
	PhaseOnBuffRemoved           // 当任意 Buff 从玩家身上被移除/失效时触发
)

// String 返回Phase的字符串表示
func (p Phase) String() string {
	names := map[Phase]string{
		PhaseBeforeTurn:     "BeforeTurn",
		PhaseOnMove:         "OnMove",
		PhaseOnLand:         "OnLand",
		PhasePreEvent:       "PreEvent",
		PhasePreDamage:      "PreDamage",
		PhaseAfterTurn:      "AfterTurn",
		PhaseAnyTime:        "AnyTime",
		PhaseOnBuffApplied:  "OnBuffApplied",
		PhaseOnBuffRemoved:  "OnBuffRemoved",
	}
	if name, ok := names[p]; ok {
		return name
	}
	return "Unknown"
}

// IsValid 检查Phase是否有效
func (p Phase) IsValid() bool {
	return p >= PhaseBeforeTurn && p <= PhaseOnBuffRemoved
}

// NeedsSubscription 判断该Phase是否需要订阅EventBus
// AnyTime类型不订阅，需要主动触发
// PhaseOnBuffApplied 和 PhaseOnBuffRemoved 需要订阅以监听 Buff 生命周期事件
func (p Phase) NeedsSubscription() bool {
	return p != PhaseAnyTime
}
