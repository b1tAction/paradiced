# Fated - Fate Dice Game Backend

《命运骰子》回合制派对游戏后端逻辑，使用 Go 语言实现，基于 Nakama 权威服务器框架架构与领域驱动设计 (DDD)。

## Project Overview

This is a turn-based party game backend similar to Mario Party. Players from four factions compete to reach the end of the map and defeat the boss.

### Core Features
- Four divine beast factions (青龙/朱雀/白虎/玄武) with unique passive skills
- Map system with various cell types (Normal/Fragile/Fog/Checkpoint/Boss)
- Event system with good/neutral/bad events influenced by luck
- Buff/Item system with multi-phase trigger support
- EventBus for unified event dispatch mechanism

## Architecture

### Package Structure

```
├── internal/
│   ├── core/           # Core data structures (Player, Buff, Item, Event, Faction)
│   │   ├── types/      # Shared basic types (Evaluation, SpecialEffect)
│   │   ├── buff/       # Buff system with multi-phase support
│   │   ├── event/      # Event system with evaluation scores
│   │   └── item/       # Item system with phase-specific triggers
│   ├── engine/         # Game engine (Game, Handlers)
│   │   ├── action/     # Action system (DamageAction, HealAction, etc.)
│   │   └── hsm/        # Hierarchical State Machine (removed old StateMachine)
│   └── gamemap/        # Map system (Cell, MapEngine, PathResult)
├── pkg/
│   ├── action/         # Action interface layer (ActionType, Action interface)
│   ├── event/          # EventBus system (Phase, Bus, Decision, Context)
│   ├── protocol/       # Public interfaces (Player, Game, MapEngine, Faction)
│   ├── rng/            # Random number engine (WeightedPool, LuckModifier)
│   └── util/           # Utilities (Metadata)
└── doc/
    ├── internal/       # Internal package documentation
    └── background.md   # Game design document (Chinese)
```

### Key Components

#### Protocol Layer (`pkg/protocol`)
- **Player**: Interface for player operations (Reader/Writer/Lite variants)
- **Game**: Interface for game state access
- **MapEngine**: Interface for map operations
- **Faction**: Player faction type (青龙/朱雀/白虎/玄武)

#### Action System (`pkg/action` + `internal/engine/action`)
- **ActionType**: Enum for action types (Damage, Heal, Move, AddBuff, etc.)
- **Action interface**: Core interface with PreTriggerPhase/PostTriggerPhase
- **ExecutableAction**: Concrete implementations (DamageAction, HealAction, etc.)
- **ActionContext**: Execution context with EventBus integration
- **TurnEventLog**: Event log for client animation playback

#### EventBus System (`pkg/event`)
- **Phase**: Trigger timing enumeration (HSM: BeforeTurn/OnLand/AfterTurn; Action: PreDamage/PreEvent/PreMove/OnBuffApplied/OnBuffRemoved)
- **EventBus**: Manages Buff/Item subscriptions and triggers
- **Decision**: User confirmation mechanism
- **Context**: Execution context with Metadata embedding

#### Core Data Structures (`internal/core`)
- **Player**: User entity with HP/LP/Buffs/Items/Metadata (implements protocol.Player)
- **Buff**: Status effects with multi-phase support
- **Item**: Consumable items with phase-specific triggers
- **Event**: Random events with evaluation scores
- **Evaluation**: 0-100 scoring system (Bad ≤40, Neutral 41-65, Good >65)

#### Game Engine (`internal/engine`)
- **Game**: Game instance managing EventBus and players
- **Handlers**: Custom Buff effect handlers (strategy pattern)
- **Action Integration**: All effects use Action system

#### Map System (`internal/gamemap`)
- **MapEngine**: Linear map generation and path calculation
- **CellType**: Normal, Fragile, Fog, Checkpoint, Boss
- **PathResult**: Movement calculation with Fragile/Fog handling

#### RNG Engine (`pkg/rng`)
- **WeightedPool**: Weighted random draw
- **LuckModifier**: Luck-based weight adjustment
- **EventPool/ItemPool**: Predefined pools for game content

## Game Flow

### Round Pipeline
1. **MatchInit**: Generate map, assign factions
2. **MiniGame**: All players compete
3. **RoundPreparation**: Award dice based on mini-game ranking

### Player Turn Pipeline

**Phase Design Principle: Who produces timing, who publishes Phase**

1. **BeforeTurn** (HSM publishes): Trigger BeforeTurn phase effects (神眷/诅咒 LP±1, 离火 every 4 turns)
2. **MainAction**: Player can use items or faction skills
3. **PreMove** (Action publishes): MoveAction triggers PreMove phase for 迷途 interception
4. **OnLand** (HSM publishes): Trigger landing events
5. **PreEvent** (Action publishes): DrawEventAction triggers PreEvent for 辟邪/玄武
6. **AfterTurn** (HSM publishes): Tick Buff durations, trigger AfterTurn effects

## Development Guidelines

### Coding Standards
- Use English comments for code (Chinese reserved for game-specific terms like Buff names, events)
- Documentation files in Chinese
- Follow TDD principles
- No external dependencies in core packages

### Testing
Run tests with:
```bash
go test ./...
```

### Commit Convention
- `feat(scope): description` - New feature
- `refactor(scope): description` - Code refactoring
- `fix(scope): description` - Bug fix
- `docs(scope): description` - Documentation update
- `chore(scope): description` - Maintenance tasks

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
| 神眷 (Divine) | BeforeTurn | LP+1 per turn |
| 诅咒 (Curse) | BeforeTurn | LP-1 per turn |
| 迷途 (Lost) | PreMove | Reverse movement direction |
| 隐匿 (Hidden) | PreDamage | Immunity to damage/events |
| 甘霖 (Rain) | AfterTurn | HP+1 every 2 turns |
| 腐化 (Corrupt) | AfterTurn | HP-1 every 2 turns |
| 辟邪 (Exorcism) | PreEvent | Immune to poison |
| 毒瘴 (Poison) | BeforeTurn | Bad event each turn |
| 离火 (Fire) | BeforeTurn | ZhuQue passive, LP+1 every 4 turns |

## Related Documentation

- [doc/background.md](doc/background.md) - Game design and rules (Chinese)
- [doc/internal/event_bus_system.md](doc/internal/event_bus_system.md) - EventBus documentation (Chinese)
- [doc/internal/core.md](doc/internal/core.md) - Core structures (Chinese)
- [doc/internal/rng_engine.md](doc/internal/rng_engine.md) - RNG engine (Chinese)
- [doc/internal/metadata.md](doc/internal/metadata.md) - Metadata utility (Chinese)
- [doc/internal/gamemap.md](doc/internal/gamemap.md) - Map system (Chinese)
- [pkg/protocol/README.md](pkg/protocol/README.md) - Protocol interface layer
- [pkg/action/README.md](pkg/action/README.md) - Action interface layer
- [internal/engine/action/README.md](internal/engine/action/README.md) - Action implementation