# Paradiced - Paradise Dice Game Backend

《派乐代》回合制派对游戏后端逻辑，使用 Go 语言实现，基于 Nakama 权威服务器框架架构与领域驱动设计 (DDD)。

## Project Overview

This is a turn-based party game backend similar to Mario Party. Players from four factions compete to reach the end of the map and defeat the boss.

### Core Features
- Four divine beast factions (青龙/朱雀/白虎/玄武) with unique passive skills
- Map system with various cell types (Normal/Fragile/Fog/Checkpoint/Boss)
- Event system with good/neutral/bad events influenced by luck
- Buff/Item system with multi-phase trigger support
- EventBus for unified event dispatch mechanism
- GameLog for client animation playback

## Architecture

### Package Structure

```
├── internal/
│   ├── core/           # Core data structures (Player, Buff, Item, GameEvent)
│   │   ├── buff.go     # Buff struct + BuffDefinition (static metadata)
│   │   ├── item.go     # Item struct + ItemDefinition (static metadata)
│   │   ├── game_event.go # GameEvent struct + EventDefinition (static metadata)
│   │   ├── player.go   # Player struct (HP/LP/Buffs/Items/Metadata)
│   │   └── faction.go  # Faction helper functions
│   ├── event/          # EventBus system (Bus, Decision, Context)
│   ├── engine/         # Game engine (Game, Registry, Handlers)
│   │   ├── buff_registry.go # Buff Registry + HandlerConfig + handlers
│   │   ├── item_registry.go # Item Registry + HandlerConfig + handlers
│   │   ├── event_registry.go # Event Registry + HandlerConfig + handlers
│   │   ├── boss_registry.go  # Boss Registry + BossDefinition + SkillHandlers
│   │   ├── action/     # Action system (DamageAction, HealAction, BossDamageAction, etc.)
│   │   └── hsm/        # Hierarchical State Machine
│   ├── gamemap/        # Map system (Cell, MapEngine, PathResult)
│   └── net/            # Sync data builder (Builder, test helper)
├── pkg/
│   ├── constants/      # Unified enum types (BuffType, EventType, ItemType, Phase, etc.) + Definition structs
│   ├── gamelog/        # Unified game log system for client playback
│   ├── id/             # Typed ID wrapper system (PlayerID, BuffID, ItemID, etc.)
│   ├── net/            # Network protocol layer (OpCode, Message, StateSync, MatchHandler, Builder interface)
│   ├── protocol/       # Public interfaces (Game, MapEngine) - for testing mocks
│   ├── resource/       # YAML/JSON resource loading (DefinitionsSet, MapConfig, go:embed)
│   ├── rng/            # Random number engine (WeightedPool, LuckModifier, DiceManager)
│   └── util/           # Utilities (Metadata with JSON serialization)
└── doc/
    ├── internal/       # Internal package documentation
    ├── metadata/       # Metadata contract documentation
    └── background.md   # Game design document (Chinese)
```

### Key Components

#### Constants Layer (`pkg/constants`)
- **BuffType**: Buff identifiers with IsPositive/IsNegative/IsBoss/IsHidden/IsFaction/IsDraw classification
- **EventType**: Event identifiers for random events
- **ItemType**: Item identifiers for consumables
- **Phase**: Trigger timing (HSM: BeforeTurn/OnLand/AfterTurn/PreMove; Action: PreDamage/PreEvent/PreRespawn/PostBuffApplied/PreBuffRemoved)
- **Faction**: Player faction type (青龙/朱雀/白虎/玄武)
- **CellType**: Map cell type (Normal, Fragile, Fog, Checkpoint, Boss, Event)
- **DrawType**: Cell draw type (none/event/item) - specifies what to draw when landing on a cell
- **StateID**: HSM state identifier (Global/Turn/Interrupt layers)
- **EntryType**: GameLog entry type (action, state, mini_game, boss, decision)
- **ActionSource**: Action source identifier (Buff/Item/Event/Faction/System/Boss)
- **Evaluation**: 0-100 scoring system (Bad ≤40, Neutral 41-65, Good >65)
- **ErrorCode**: Error codes for client-server communication (0=OK, 1xxx=validation, 2xxx=game logic, 3xxx=system, 4xxx=not found)
- **BossType**: Boss entity type (beast)
- **BossSkillType**: Boss skill type (thunder/curse/lost/rest)
- **BossAttackType**: Boss attack type (normal/crit/skill)
- **BossPlayerUUID**: Fixed UUID for Boss special player
- All enums use string type with snake_case values for JSON compatibility
- **Player**: Interface for player operations (Reader/Writer/Lite variants)
- **Game**: Interface for game state access, includes GetGameLog()
- **MapEngine**: Interface for map operations
- **Faction**: Player faction type (青龙/朱雀/白虎/玄武)

#### Action System (`internal/engine/action`)
- **ActionType**: String type with snake_case naming (damage, heal, move, etc.)
- **Action interface**: Core interface with PreTriggerPhase/PostTriggerPhase/TargetPlayer
- **Action implementations**: Concrete implementations (DamageAction, HealAction, RespawnAction, FellDownAction, DrawItemAction, DrawEventAction, DrawBuffAction, AddItemAction, RemoveItemAction, DiceUpgradeAction, BossDamageAction, BossAttackAction, BossSkillAction, etc.)
- **ActionContext**: Execution context with EventBus, GameLog, pools (EventPool/ItemPool/BuffPool), and lifecycle callbacks (OnAddBuff/OnRemoveBuff/OnAddItem/OnRemoveItem)
- **DerivedAction pattern**: Event/Item handlers push concrete Actions via ctx.AddDerivedAction() instead of setting flags

#### Protocol Layer (`pkg/net`)
- **Builder interface**: Abstract interface for building protocol sync messages (implemented in internal/net)
- **BroadcastAdapter**: Broadcast abstraction for client communication
- **OpCode**: Message operation codes (Server→Client: 1-99, Client→Server: 100+)
  - Server→Client: OpStateSync(1), OpDecisionRequest(3), OpAvailable(4), OpMiniGameStart(5), OpMiniGameResult(6), OpGameOver(7), OpFullSync(8), OpActionRejected(9), OpWaitingSync(10), OpStartGameAck(11)
  - Client→Server: OpRollDice(100), OpUseItem(101), OpUseSkill(102), OpUserChoice(103), OpStartGame(105), OpRoundReady(106), OpMiniGameDataSubmit(107)
- **StateSync**: Complete state synchronization structure
- **Decision**: User confirmation request structure
- **ActionRejected**: Action rejection notification with ErrorCode for client-side error handling

#### Nakama Protocol Layer (`internal/nakama`)
- **NakamaMatchHandler**: Main match handler with HSM integration
- **Logger**: Structured logging helper for request/response/rejection tracking (nil-safe)
- **ErrorCode System**: Standardized error codes (pkg/constants.ErrorCode) for client feedback
- **Message Handlers**: Handle client requests (roll dice, use item, use skill, etc.) with validation and error reporting

#### CLI Tool (`internal/cli`)
- **playtest run**: Run single automated playtest (2-4 players)
- **playtest soak**: Run multiple rounds for stability testing
- **AutoPlayPlayer**: Automated player with default strategies (roll dice, choose first option)
- **Test reports**: JSON output with success rate, duration, message count

#### GameLog System (`pkg/gamelog`)
- **EntryType**: Alias to constants.EntryType - log entry types (action, state, mini_game, boss, decision)
- **LogEntry**: Single event with util.Metadata for type-safe metadata
- **TurnSegment**: Turn-based log grouping for client playback
- **GameLog**: Global log manager with StartTurn/EndTurn/AddEntry methods

#### EventBus System (`pkg/event`)
- **Phase**: Alias to constants.Phase - trigger timing enumeration
- **EventBus**: Manages Buff/Item subscriptions and triggers
- **Decision**: User confirmation mechanism
- **Context**: Execution context with Metadata embedding and DerivedActions

#### ID System (`pkg/id`)
- **PlayerID**: Player unique identifier (prefix: "player")
- **BuffID**: Buff instance identifier (prefix: "buff")
- **ItemID**: Item instance identifier (prefix: "item")
- **GameID**: Game instance identifier (prefix: "game")
- **SubscriptionID**: EventBus subscription identifier (prefix: "sub")
- **DecisionID**: Decision identifier (prefix: "dec")
- ID.String() returns prefix+UUID for debugging (e.g., "player-{uuid}")
- ID.UUID() returns pure UUID for protocol transmission
- JSON serialization outputs pure UUID (type recognized by field name)

#### Core Data Structures (`internal/core`)
- **Player**: User entity with HP/LP/Buffs/Items/Metadata (implements protocol.Player)
- **Buff**: Status effects with multi-phase support (uses constants.BuffType)
- **Item**: Consumable items with phase-specific triggers (uses constants.ItemType)
- **Event**: Random events with evaluation scores (uses constants.EventType, constants.Evaluation)

#### Game Engine (`internal/engine`)
- **Game**: Game instance managing EventBus, players, GameLog, and pools (EventPool/ItemPool/BuffPool)
- **Handlers**: Custom Buff/Item/Event effect handlers (strategy pattern, DerivedAction pattern)
- **Action Integration**: All effects use Action system and record to GameLog
- **Item Lifecycle**: ApplyItemToPlayer/RemoveItemFromPlayer manage EventBus subscription + data layer

#### HSM System (`internal/engine/hsm`)
- **HSM**: Hierarchical State Machine with three layers
- **Global States**: MatchInit, RoundMiniGame, RoundPrep, TurnLoop, GameOver
- **Turn States**: TurnUpkeep, MainAction, TurnMoving, TurnCheckpoint, TurnLanded, TurnDraw, TurnBossBattle, TurnEnd
- **Interrupt States**: WaitDecision for user input
- **TurnDrawState**: Unified draw state that handles both Event and Item draws based on cell's DrawType configuration. Entered from TurnLanded when cell has a valid DrawType (event/item) and prob settings.
- **decisionStateResetter**: Interface for states that cache pending decisions, called after decision resolution
- **OnUseItem**: Handler execution + item consumption via RemoveItemAction

#### Map System (`internal/gamemap`)
- **MapEngine**: Linear map generation and path calculation
- **CellType**: Alias to constants.CellType - Normal, Fragile, Fog, Checkpoint, Boss, Event
- **PathResult**: Movement calculation with Fragile/Fog handling

#### RNG Engine (`pkg/rng`)
- **WeightedPool**: Weighted random draw
- **LuckModifier**: Luck-based weight adjustment
- **EventPool/ItemPool/BuffPool**: Predefined pools for game content
- **DrawEngine**: Probability-based drawing with `DrawWithProb` method that supports weighted pool selection (Good/Neutral/Bad) and fallback to all items when total probability < 1.0
- **Boss Attack Calculation**: CalcBossAttackType, SelectBossTarget, CalcPlayerCrit for Boss battle RNG

## Game Flow

### Round Pipeline
1. **MatchInit**: Generate map, assign factions
2. **MiniGame**: All players compete
3. **RoundPreparation**: Award dice based on mini-game ranking

### Player Turn Pipeline

**Phase Design Principle: Who produces timing, who publishes Phase**

1. **BeforeTurn** (HSM publishes): Trigger BeforeTurn phase effects (神眷/诅咒 LP±1, 离火 every 4 turns). Poison buff draws bad event (100% bad probability). Boss player skips.
2. **MainAction**: Player can use items or faction skills. OnUseItem executes handler + consumes item via RemoveItemAction. If on Boss cell → TurnBossBattle.
3. **PreMove** (HSM publishes): TurnMovingState publishes PreMove, 迷途 handler modifies Steps via StepsModifier interface
4. **OnLand** (HSM publishes): Trigger landing events
5. **TurnDraw** (HSM state): When landing on a cell with DrawType (event/item), enter TurnDraw state to perform probability-based draw
6. **TurnBossBattle** (HSM state): When player is on Boss cell, enter TurnBossBattle for player attack; Boss counter-attack on Boss's turn
7. **PreEvent** (Action publishes): DrawEventAction triggers PreEvent for 辟邪/玄武
8. **AfterTurn** (HSM publishes): Tick Buff durations, trigger AfterTurn effects. Boss player and Boss-defeated turn skip.

## Development Guidelines

### Coding Standards
- Use English comments for code (Chinese reserved for game-specific terms like Buff names, events)
- Documentation files in Chinese
- Follow TDD principles
- No external dependencies in core packages
- **禁止使用类型别名**：不使用 `type Faction = constants.Faction`、`type BuffType = constants.BuffType` 等别名写法。应直接使用完整路径如 `constants.Faction`、`constants.BuffType`、`constants.EntryType` 等。
  - 类型别名削弱类型安全，增加维护成本
  - 所有枚举类型应通过 `constants.Type` 格式引用，例如：`constants.FactionQingLong`、`constants.BuffTypeCurse`
  - 不要在 internal/core 或其他包中重导出 constants 的常量

### Environment
- **GOMODCACHE**: `${workdir}/.gomodcache` (本地模块缓存位置)
- 运行测试和构建时需设置：`GOMODCACHE=${workdir}/.gomodcache go test ./...`

### Nakama Development (CLI 与 Nakama 协议层对接)

#### Docker Compose 启动服务

项目使用 `docker-compose.yml` 启动 Nakama 服务器和 CockroachDB：

```bash
# 启动所有服务（CockroachDB + Nakama）
make docker-up

# 停止服务
make docker-down

# 停止并清除所有数据（包括 volume）
make docker-clean
```

#### 插件开发流程

1. **首次启动 Nakama**：
```bash
# 创建 modules 目录
make prepare-modules

# 启动 Nakama（会自动运行数据库迁移）
make docker-up
```

2. **构建并加载插件**：
```bash
# 构建插件为共享对象
make build-plugin

# 重启 Nakama 容器以加载新插件
docker-compose restart nakama
```

3. **开发循环（修改代码后）**：
```bash
# 一键重建插件并重启 Nakama
make rebuild
```

4. **查看日志**：
```bash
# 查看 Nakama 日志(推荐、实时输出)
cat ./logs/nakama.log
```

```bash
# 查看 Nakama 日志
make docker-logs

# 查看 CockroachDB 日志
make docker-logs-db

# 查看全部日志
make docker-logs-all
```

5. **访问管理界面**：
- Nakama Console: http://localhost:7351 (默认账号/密码：admin/admin)
- CockroachDB Admin: http://localhost:8080

#### 重要注意事项

- **修改插件后必须重启 Nakama 容器**：Nakama 在启动时加载插件，修改 `modules/*.so` 后需要 `docker-compose restart nakama`
- **首次启动会自动迁移数据库**：`nakama migrate up` 命令在 entrypoint 中执行
- **插件构建使用 Nakama PluginBuilder 镜像**：确保本地 Docker 可访问 `heroiclabs/nakama-pluginbuilder:3.22.0`
- **挂载点**：`./modules:/nakama/modules` 和 `./config.yml:/nakama/data/config.yml`

### Testing

Run tests with: ${workdir}需要替换为当前目录路径：
```bash
GOMODCACHE=${workdir}/.gomodcache go test ./...
```

### Commit Convention
Git Commit信息必须使用英文提交

- `feat(scope): description` - New feature
- `refactor(scope): description` - Code refactoring
- `fix(scope): description` - Bug fix
- `docs(scope): description` - Documentation update
- `chore(scope): description` - Maintenance tasks

### Git Operations

- **禁止使用 `git add .` 或 `git add -A`**：每次提交必须明确指定具体文件
- 正确做法：`git add file1.go file2.go && git commit -m "..."`
- 先 `git status` 查看改动，按改动方向分批提交

## Metadata Contracts

**重要**：项目中多个类型嵌入 `util.Metadata`，所有字段使用必须遵循契约文档。

### 契约文档位置

契约文档按类划分，位于 `doc/metadata/` 目录：

| 文件 | 类型 | 可见性 | 说明 |
|------|------|--------|------|
| [doc/metadata/logentry.md](doc/metadata/logentry.md) | `gamelog.LogEntry.Metadata` | **客户端可见** | Action 效果详情，客户端渲染关键 |
| [doc/metadata/player.md](doc/metadata/player.md) | `core.Player.Metadata` | **客户端可见** | 玩家动态属性（阵营特定） |
| [doc/metadata/event_context.md](doc/metadata/event_context.md) | `event.Context.Metadata` | 内部 | EventBus Handler 通信 |
| [doc/metadata/hsm_context.md](doc/metadata/hsm_context.md) | `hsm.StateContext.Metadata` | 内部 | HSM 状态机通信 |
| [doc/metadata/action_context.md](doc/metadata/action_context.md) | `action.ActionContext.Metadata` | 内部 | Action 执行上下文 |
| [doc/metadata/buff.md](doc/metadata/buff.md) | `core.Buff.Metadata` | 内部 | Buff 实例动态属性 |
| [doc/metadata/round_data.md](doc/metadata/round_data.md) | `hsm.StateContext.RoundData` | 内部 | 回合周期性数据 |

### 新增 Metadata 字段时

1. 确定字段归属（LogEntry/Player/Context/StateContext/ActionContext/Buff/RoundData）
2. 在对应契约文档更新表格
3. 若客户端可见，同步更新 TypeScript 类型定义

## Faction System

| Faction | Skill | Description |
|---------|-------|-------------|
| 青龙 (QingLong) | 行迹 | Every 5 turns gain charge, ignore negative terrain for 1 turn |
| 朱雀 (ZhuQue) | 离火 | Every 4 turns LP+1 (max 8) |
| 白虎 (BaiHu) | 劫运 | When overtaking others, steal random Buff |
| 玄武 (XuanWu) | 鎮厄 | Every 5 turns gain charge, cancel one bad event |

## Buff System

| Buff | Phases | Effect |
|------|--------|--------|
| 神眷 (Divine) | PostBuffApplied, PreBuffRemoved | LP+1 on application, LP-1 on removal |
| 诅咒 (Curse) | PostBuffApplied, PreBuffRemoved | LP-1 on application, LP+1 on removal |
| 迷途 (Lost) | PreMove (HSM发布) | Reverse movement direction (Steps → -Steps, anti double-flip) |
| 隐匿 (Hidden) | PreBuffApplied | Immunity to events/buffs; blocks non-positive, non-Boss buffs (IsBoss/IsPositive bypass). Does NOT block damage. |
| 甘霖 (Rain) | AfterTurn | HP+1 every 2 turns |
| 腐化 (Corrupt) | AfterTurn | HP-1 every 2 turns |
| 辟邪 (Exorcism) | PreEvent | Immune to poison |
| 毒瘴 (Poison) | BeforeTurn | Bad event each turn |
| 离火 (Fire) | BeforeTurn | ZhuQue passive, LP+1 every 4 turns (IsFaction=true, not in draw pool) |
| 死亡标记 (DeathMark) | PreAction (Hidden) | Block all actions for dead players (exempt: RespawnAction, RemoveBuffAction(DeathMark)) |
| 反刺 (Thorns) | PreDamage (Boss self) | Boss reflect: 30% damage back as derived BossAttackAction (BossAttackAction derives DamageAction) |

## Item System

| Item | Phase | Effect |
|------|-------|--------|
| 反方向的钟 (ReverseClock) | AnyTime | Give target player Lost buff |
| 任意门 (AnyDoor) | ItemUsed | Teleport to target player's position (NeedConfirm=false, frontend sends targetID) |
| 骰子升级卡 (DiceUpgrade) | ItemUsed | Upgrade dice type (Wood→Copper→Silver→Gold) |

**Item Lifecycle**: Items follow complete lifecycle through Action system:
- **AddItemAction** → OnAddItem callback → game.ApplyItemToPlayer (data + EventBus subscription)
- **RemoveItemAction** → OnRemoveItem callback → game.RemoveItemFromPlayer (unsubscribe + data removal)
- **Consumption**: OnUseItem handler execution followed by RemoveItemAction for item consumption

**DerivedAction pattern**: Event/Item handlers push concrete Actions (DrawItemAction, DrawBuffAction, TeleportAction, RemoveItemAction, DiceUpgradeAction) instead of setting flags.

## Boss System

| Property | Value |
|----------|-------|
| BossType | Beast (凶兽) |
| Boss HP | 50 |
| Boss LP | 0 (no LP) |
| Boss UUID | `beeeeeef-beef-beef-beef-beeeeeeeeeef` |
| Boss Position | Map end (Boss cell) |

### Boss Skills (equal weight random draw)

| Skill | Type | Effect |
|-------|------|--------|
| Thunder (天雷) | AOE damage | All Boss-cell players take 3 damage |
| Curse (诅咒) | Debuff | All Boss-cell players get Curse buff |
| Thorns (反刺) | Self-buff | Boss gains Thorns buff (2 turns): reflect 30% damage back |
| Rest (息) | Heal | Boss heals 20 HP |

### Boss Attack Mechanics

- Boss crit/skill probability: `0.25 + 0.05 × (8 - avgLP) + 0.30 × (maxHP - currentHP) / maxHP` (30/70 crit/skill split)
- Boss target selection: LP-weighted, lower LP = higher chance of being targeted
- Player crit rate: based on dice quality (Gold:30%, Silver:20%, Copper:10%, Wood:5%)

## Related Documentation

- [doc/background.md](doc/background.md) - Game design and rules (Chinese)
- [doc/internal/event_bus_system.md](doc/internal/event_bus_system.md) - EventBus documentation (Chinese)
- [doc/internal/core.md](doc/internal/core.md) - Core structures (Chinese)
- [doc/internal/yaml_definitions.md](doc/internal/yaml_definitions.md) - YAML definition system (Chinese)
- [doc/internal/rng_engine.md](doc/internal/rng_engine.md) - RNG engine (Chinese)
- [doc/internal/metadata.md](doc/internal/metadata.md) - Metadata utility usage (Chinese)
- [doc/metadata/README.md](doc/metadata/README.md) - Metadata contracts (Chinese)
- [doc/internal/gamemap.md](doc/internal/gamemap.md) - Map system (Chinese)
- [doc/internal/nakama.md](doc/internal/nakama.md) - Nakama integration (Chinese)
- [doc/protocol_hsm_interaction.md](doc/protocol_hsm_interaction.md) - Protocol-HSM interaction flow (Chinese)
- [pkg/constants/README.md](pkg/constants/README.md) - Unified enum types
- [pkg/net/README.md](pkg/net/README.md) - Network protocol layer and Builder interface
- [pkg/protocol/README.md](pkg/protocol/README.md) - Protocol interface layer
- [pkg/id/README.md](pkg/id/README.md) - Typed ID wrapper system
- [pkg/gamelog/README.md](pkg/gamelog/README.md) - GameLog system
- [pkg/rng/README.md](pkg/rng/README.md) - RNG engine and dice system
- [internal/engine/hsm/README.md](internal/engine/hsm/README.md) - HSM state machine
- [internal/engine/action/README.md](internal/engine/action/README.md) - Action implementation
- [internal/net/README.md](internal/net/README.md) - Sync data builder
- [internal/nakama/README.md](internal/nakama/README.md) - Nakama Match Handler
- [internal/cli/README.md](internal/cli/README.md) - CLI testing tool
