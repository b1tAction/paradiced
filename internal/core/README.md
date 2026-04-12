# internal/core - Core Data Structures

核心数据结构包，定义游戏的基础类型和数据结构。

## 功能

此包无外部依赖（仅依赖 pkg/event），可独立使用。

### 数据类型

- **Evaluation**: 0-100 评分系统
  - 0-40: 恶性（Bad）
  - 41-65: 中性（Neutral）
  - 66-100: 良性（Good）

- **Faction**: 四阵营（青龙/朱雀/白虎/玄武）

- **Buff**: Buff 类型、实例、定义、注册表

- **Item**: 道具类型、实例、定义、注册表

- **Event**: 事件类型、定义、注册表

- **Player**: 玩家结构体
  - HP/LP 管理
  - Buff/道具管理
  - 移动逻辑

## 文件结构

```
internal/core/
├── evaluation.go  # 评分系统
├── faction.go     # 阵营定义
├── buff.go        # Buff 系统
├── item.go        # Item 系统
├── event.go       # Event 系统
└── player.go      # Player 结构
```

## 使用示例

```go
// 创建玩家
player := core.NewPlayer(core.PlayerConfig{
    UserID:  "player-001",
    Faction: core.FactionZhuQue,
})

// 添加 Buff
buff := core.NewBuff(core.BuffTypeCurse, 3)
player.AddBuff(buff)

// 获取 Buff 定义
def := core.BuffTypeCurse.GetBuffDefinition()
fmt.Println(def.Name)    // "诅咒"
fmt.Println(def.Eval)    // 25 (Bad)
fmt.Println(def.Phase)   // BeforeTurn
```

## 与其他包的关系

- `pkg/event`: Phase 类型依赖
- `internal/engine`: 使用 core 的数据类型
- `internal/gamemap`: 无依赖