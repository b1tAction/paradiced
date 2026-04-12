# PlayerSystem (玩家系统) 实现文档

## 概述

PlayerSystem 是《命运骰子》游戏的玩家管理模块，负责玩家数据结构、阵营系统、Buff/道具管理和数值逻辑。

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

### BuffType (Buff类型枚举)

| 类型 | 属性 | 效果 | 持续回合 |
|------|------|------|----------|
| Curse | 负面 | 每回合 LP-1 | 3 |
| Lost | 负面 | 反方向移动 | 1 |
| Corrupt | 负面 | 每2回合 HP-1 | 4 |
| Poison | 负面 | 每回合受恶性随机事件 | 3 |
| Divine | 正面 | 每回合 LP+1 | 3 |
| Hidden | 正面 | 免疫任意事件/BUFF/道具 | 3 |
| Rain | 正面 | 每2回合 HP+1 | 4 |
| Exorcism | 正面 | 无视毒瘴buff | 5 |
| Fire | 正面 | 朱雀阵营增益，每4回合 LP+1 | 永久 |

### ItemType (道具类型枚举)

| 类型 | 效果 |
|------|------|
| ReverseClock | 给予指定玩家迷途buff |
| AnyDoor | 去到30格内指定玩家身边 |
| DiceSwap | 骰子交换 |
| DiceUpgrade | 骰子升级卡 |

### Player (玩家结构体)

```go
type Player struct {
    UserID      string   // 玩家UUID
    Faction     Faction  // 阵营
    Position    int      // 当前位置
    HP          int      // 血量
    LP          int      // 幸运值（影响随机事件）
    Inventory   []*Item  // 道具栏
    ActiveBuffs []*Buff  // 持续状态
    IsDead      bool     // 是否死亡
    SkipTurn    bool     // 是否跳过回合
    ChargeCount int      // 充能计数（青龙/玄武）
}
```

## 核心功能

### 1. 数值逻辑

#### ApplyDamage (扣血)
```go
isDead, respawnPos, err := player.ApplyDamage(amount, engine)
```

**逻辑流程**：
1. 检查隐匿状态（Hidden Buff 免疫伤害）
2. 扣减 HP
3. HP ≤ 0 时触发死亡，回城到最近检查点
4. 返回死亡状态和回城位置

#### Heal / ModifyLP
```go
player.Heal(amount)      // 回血
player.ModifyLP(amount)  // 修改幸运值（范围限制 0~8）
```

### 2. 移动逻辑

```go
player.Move(newPosition, maxLength)  // 移动到指定位置
player.Respawn(respawnPos)           // 复活回城
```

**复活回城**：
- 重置 HP 到初始值
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
- 隐匿状态下免疫负面 Buff
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
player.UpdateCharge()              // 更新充能计数
```

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
    Type     EventPhase
    Source   *Player
    Target   *Player
    Payload  interface{}
    IsCancel bool  // 是否被取消/拦截
}
```

```go
player.DispatchEvent(event)  // 分发事件到玩家的 Hooks
```

**事件处理流程**：
1. 检查隐匿状态 → 取消事件
2. 触发阵营被动技能（玄武抵消恶性事件）
3. 触发道具 Hook（待扩展）

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

## 测试覆盖

测试文件：`internal/game/player_test.go`

| 测试类 | 覆盖内容 |
|--------|----------|
| FactionTest | 阵营名称转换、有效性验证 |
| BuffTypeTest | Buff名称、正面/负面分类 |
| BuffTest | 创建、激活状态、持续时间更新 |
| ItemTest | 道具创建、类型名称 |
| PlayerTest | 创建玩家、默认配置、朱雀初始Buff |
| HP/LPTest | 扣血/回血/死亡/回城、隐匿免疫、LP范围限制 |
| MovementTest | 移动位置、终点限制、复活回城 |
| BuffMgmtTest | 添加/移除/查询、隐匿免疫负面Buff、Tick更新 |
| ItemMgmtTest | 添加/移除/查询道具 |
| FactionSkillTest | 青龙充能、朱雀离火、白虎偷取、玄武抵消 |
| EventSystemTest | 隐匿取消事件 |
| HelperTest | 克隆、字符串表示、存活状态、行动状态 |

## 后续扩展

1. **道具 Hook 系统**：护盾类道具监听 PreDamage 事件
2. **回合状态机集成**：State_Turn_Upkeep 调用 CanAct()，State_Turn_End 调用 TickBuffs()
3. **Buff 效果结算**：回合结束时根据 Buff 类型修改 HP/LP

## 文件结构

```
internal/game/
├── types.go          # 类型定义 (Faction, BuffType, ItemType) (150行)
├── player.go         # 玩家实现 (330行)
└── player_test.go    # 单元测试 (480行)
```