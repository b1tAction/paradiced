# PlayerSystem (玩家系统) 实现文档

## 概述

PlayerSystem 是《命运骰子》游戏的玩家管理模块，负责玩家数据结构、阵营系统、Buff/道具管理和数值逻辑。

## 文件结构

```
internal/game/
├── faction.go        # 阵营定义（~100行）
├── buff.go           # Evaluation 系统 + Buff 系统（~360行）
├── event.go          # Event 系统（~290行）
├── item.go           # Item 系统（~200行）
├── player.go         # 玩家实现（~480行）
├── buff_test.go      # Buff 和 Evaluation 单元测试
├── item_test.go      # Item 单元测试
├── event_test.go     # Event 单元测试
└── player_test.go    # Player 单元测试（~600行）
```

类型定义已分散到各自领域文件中：
- `buff.go`: Evaluation, BuffType, Buff, BuffDefinition, BuffRegistry
- `event.go`: EventType, EventDefinition, EventRegistry
- `item.go`: ItemType, Item, ItemDefinition, ItemRegistry
- `faction.go`: Faction

## 数据结构

### Faction (阵营枚举)

```go
type Faction int

const (
    FactionQingLong Faction = iota // 青龙（东方）- 行迹
    FactionZhuQue                  // 朱雀（南方）- 离火
    FactionBaiHu                   // 白虎（西方）- 劫运
    FactionXuanWu                  // 玄武（北方）- 镇厄
)
```

### PlayerConfig (玩家配置)

```go
type PlayerConfig struct {
    UserID   string  // 玩家UUID
    Faction  Faction // 阵营
    MaxHP    int     // 最大血量
    MaxLP    int     // 最大幸运值
    StartPos int     // 起始位置
}

// 默认配置
var DefaultPlayerConfig = PlayerConfig{
    MaxHP:    6,   // 默认血量
    MaxLP:    10,  // 默认幸运值上限
    StartPos: 0,   // 起点位置
}
```

### Player (玩家结构体)

```go
type Player struct {
    UserID      string   `json:"user_id"`      // 玩家UUID
    Faction     Faction  `json:"faction"`      // 阵营
    Position    int      `json:"position"`     // 当前位置
    HP          int      `json:"hp"`           // 血量
    LP          int      `json:"lp"`           // 幸运值（影响随机事件）
    Inventory   []*Item  `json:"inventory"`    // 道具栏
    ActiveBuffs []*Buff  `json:"active_buffs"` // 持续状态
    IsDead      bool     `json:"is_dead"`      // 是否死亡
    SkipTurn    bool     `json:"skip_turn"`    // 是否跳过回合（冰冻/晕眩）
    ChargeCount int      `json:"charge_count"` // 充能计数（青龙/玄武）
}
```

### BuffType / ItemType

Buff 和 Item 的详细定义参见 `doc/internal/event_system.md`。

## 核心功能

### 1. 数值逻辑

#### ApplyDamage (扣血)
```go
isDead, respawnPos, err := player.ApplyDamage(amount, engine)
```

**逻辑流程**：
1. 检查伤害值有效性（不能为负数）
2. 检查隐匿状态（Hidden Buff 免疫伤害）
3. 扣减 HP
4. HP ≤ 0 时触发死亡，回城到最近检查点
5. 返回死亡状态和回城位置

#### Heal / ModifyLP
```go
player.Heal(amount)      // 回血
player.ModifyLP(amount)  // 修改幸运值（范围限制 0~8）
```

**幸运值范围**：LP 限制在 `[0, 8]` 区间内。

### 2. 移动逻辑

```go
player.Move(newPosition, maxLength)  // 移动到指定位置
player.Respawn(respawnPos)           // 复活回城
```

**复活回城**：
- 重置 HP 到默认值（DefaultPlayerConfig.MaxHP = 6）
- 移动到检查点位置
- 清除 IsDead 和 SkipTurn 状态

### 3. Buff 管理

```go
player.AddBuff(buff)              // 添加 Buff（隐匿状态下免疫负面 Buff）
player.RemoveBuff(buffType)       // 移除指定类型 Buff
player.HasBuff(buffType)          // 检查是否有 Buff
player.GetBuff(buffType)          // 获取 Buff 实例
player.TickBuffs()                // 更新持续时间，返回失效的 Buff
player.ClearNegativeBuffs()       // 清除所有负面 Buff
```

**特殊规则**：
- `Duration == -1` 表示永久 Buff（如朱雀离火）
- 隐匿状态下免疫负面 Buff（AddBuff 时判断，正面 Buff 可以添加）
- 隐匿状态下免疫伤害（ApplyDamage 时判断）
- TickBuffs 在回合结束时调用

### 4. 道具管理

```go
player.AddItem(item)              // 添加道具
player.RemoveItem(itemID)         // 移除道具
player.GetItem(itemID)            // 获取道具
player.HasItem(itemType)          // 检查是否有指定类型道具
```

### 5. 阵营被动技能

| 阵营 | 技能名 | 效果 | 触发条件 |
|------|--------|------|----------|
| 青龙 | 行迹 | 无视负面地形（迷雾/Fragile） | 每5回合获得充能，主动使用 |
| 朱雀 | 离火 | 每4回合 LP+1，最高8 | 自动生效（永久 Buff） |
| 白虎 | 劫运 | 反超他人时偷取一个 Buff | 反超事件触发时 |
| 玄武 | 镇厄 | 抵消一次恶性事件影响 | 每5回合获得充能，恶性事件前触发 |

```go
player.TriggerFactionSkill(event)  // 触发阵营被动
player.UpdateCharge()              // 更新充能计数（回合结束时调用）
```

**充能机制**：
- 青龙/玄武：每回合 `UpdateCharge()` 增加 ChargeCount，满5触发充能获得
- 朱雀：创建时自动获得离火 Buff（Duration = -1）
- 白虎：反超事件时自动触发 Buff 偷取

### 6. 游戏事件系统

```go
type EventPhase string

const (
    EventPreBadEvent EventPhase = "PreBadEvent" // 触发恶性事件前
    EventOnOvertake  EventPhase = "OnOvertake"  // 发生反超后
    EventPreDamage   EventPhase = "PreDamage"   // 扣血前
    EventOnMove      EventPhase = "OnMove"      // 每次移动一格时
)

type GameEvent struct {
    Type     EventPhase  `json:"type"`      // 事件类型
    Source   *Player     `json:"source"`    // 触发事件的玩家
    Target   *Player     `json:"target"`    // 目标玩家
    Payload  interface{} `json:"payload"`   // 事件数据
    IsCancel bool        `json:"is_cancel"` // 是否被取消/拦截
}
```

```go
player.DispatchEvent(event)  // 分发事件到玩家的 Hooks
```

**事件处理流程**：
1. 检查隐匿状态 → 取消事件
2. 触发阵营被动技能（玄武抵消恶性事件）
3. 触发道具 Hook（待扩展）

## FactionSkill 接口

```go
type FactionSkill interface {
    CanActivate(player *Player) bool
    Activate(player *Player, event *GameEvent) bool
    GetCharge() int
}

// 具体实现
type QingLongPassive struct { Charge int }
type ZhuQuePassive struct{}
type BaiHuPassive struct{}
type XuanWuPassive struct { Charge int }
```

## 与 MapEngine 协作

```go
engine := NewMapEngine(50)
engine.SetCellType(10, CellTypeCheckpoint)

player := NewPlayer(PlayerConfig{
    UserID:  "player-001",
    Faction: FactionQingLong,
})

// 扣血致死时自动获取最近检查点
isDead, respawnPos, _ := player.ApplyDamage(15, engine)
if isDead {
    player.Respawn(respawnPos)  // 回城到检查点 10
}
```

## 辅助方法

```go
player.Clone()        // 克隆玩家（用于测试）
player.String()       // 返回玩家信息字符串
player.IsAlive()      // 检查玩家是否存活（HP>0 && !IsDead）
player.CanAct()       // 检查玩家是否可以行动（IsAlive && !SkipTurn）
```

## 测试覆盖

测试文件：`internal/game/player_test.go`

| 测试类 | 覆盖内容 |
|--------|----------|
| FactionTest | 阵营名称转换、有效性验证 |
| PlayerTest | 创建玩家、默认配置、朱雀初始Buff |
| HP/LPTest | 扣血/回血/死亡/回城、隐匿免疫、LP范围限制 |
| MovementTest | 移动位置、终点限制、复活回城 |
| BuffMgmtTest | 添加/移除/查询、隐匿免疫负面Buff、Tick更新 |
| ItemMgmtTest | 添加/移除/查询道具 |
| FactionSkillTest | 青龙充能、朱雀离火、白虎偷取、玄武抵消 |
| EventSystemTest | 隐匿取消事件 |
| HelperTest | 克隆、字符串表示、存活状态、行动状态 |

**分离的测试文件**：
- `buff_test.go`: Evaluation 和 Buff 相关测试
- `item_test.go`: Item 相关测试
- `event_test.go`: Event 相关测试

## 后续扩展

1. **道具 Hook 系统**：护盾类道具监听 PreDamage 事件
2. **回合状态机集成**：State_Turn_Upkeep 调用 CanAct()，State_Turn_End 调用 TickBuffs()
3. **Buff 效果结算**：回合结束时根据 Buff 类型修改 HP/LP