# 前后端协议交接文档

本文档详细描述服务器与客户端之间的所有交互协议，包括消息格式、字段定义和 Action 类型。

---

## 一、消息操作码

### Server → Client (1-99)

| OpCode | 名称 | 数据类型 | 触发时机 |
|--------|------|----------|----------|
| 1 | `OpStateSync` | StateSync | 进入新全局状态时 |
| 2 | `OpTurnSync` | TurnSync | 回合效果执行后 |
| ~~3~~ | ~~`OpDecisionRequest`~~ | ~~Decision~~ | ~~需要玩家决策时~~ |
| 4 | `OpAvailable` | Available | 进入 MainAction 状态 |
| 5 | `OpMiniGameStart` | MiniGameStart | 小游戏阶段开始 |
| 6 | `OpMiniGameResult` | MiniGameResult | 小游戏结束广播排名 |
| 7 | `OpGameOver` | GameOver | 游戏结束 |
| 8 | `OpFullSync` | FullSync | 玩家断线重连 |
| 9 | `OpActionRejected` | ActionRejected | 玩家操作被拒绝 |
| 10 | `OpWaitingSync` | WaitingSync | 等待房间玩家变更 |

### Client → Server (100+)

| OpCode | 名称 | 数据类型 | 触发时机 |
|--------|------|----------|----------|
| 100 | `OpRollDice` | RollDice | 玩家请求投骰子 |
| 101 | `OpUseItem` | UseItem | 玩家使用道具 |
| 102 | `OpUseSkill` | UseSkill | 玩家使用阵营技能 |
| ~~103~~ | ~~`OpUserChoice`~~ | ~~UserChoice~~ | ~~玩家回复决策选择~~ |
| 104 | `OpMiniGameResultSubmit` | MiniGameResultSubmit | 小游戏排名提交 |
| 105 | `OpStartGame` | StartGame | 主机请求开始游戏 |

---

## 二、Server → Client 数据结构

### 2.1 StateSync（状态同步）

**用途**：广播当前游戏状态，用于客户端 UI 更新和断线重连。

```typescript
interface StateSync {
    global_state: string;       // 全局状态标识
    turn_state: string;         // 回合状态标识（空表示未在回合循环）
    current_player_id: string;  // 当前行动玩家 UUID
    round: number;              // 当前回合数
    turn: number;               // 当前回合索引 (0-3)
    paused: boolean;            // 是否等待决策
    players: Player[];          // 所有玩家状态
}
```

**global_state 可选值**：

| 值 | 含义 | 客户端行为 |
|----|------|-----------|
| `match_init` | 游戏初始化 | 显示加载界面 |
| `waiting_for_host` | 等待主机开始 | 显示等待房间 UI |
| `round_mini_game` | 小游戏阶段 | 显示小游戏界面 |
| `round_prep` | 回合准备 | 显示骰子分配结果 |
| `turn_loop` | 回合循环 | 显示回合 UI |
| `boss_battle` | Boss 战 | 显示 Boss 战界面 |
| `game_over` | 游戏结束 | 显示结算界面 |

**turn_state 可选值**：

| 值 | 含义 |
|----|------|
| `turn_upkeep` | 回合开始维护 |
| `main_action` | 主行动阶段 |
| `turn_moving` | 移动中 |
| `turn_landed` | 已落地 |
| `turn_event` | 事件阶段 |
| `turn_end` | 回合结束 |

---

### 2.2 TurnSync（回合同步）

**用途**：发送回合内所有效果，客户端按顺序渲染动画。

```typescript
interface TurnSync {
    round: number;              // 当前回合数
    turn: number;               // 当前回合索引
    current_player_id: string;  // 回合玩家 UUID
    entries: LogEntry[];        // 效果列表（顺序渲染）
}
```

**客户端渲染逻辑**：

```typescript
for (const entry of turnSync.entries) {
    switch (entry.action_type) {
        case "damage":
            playDamageAnimation(entry.target, entry.metadata.hp_change);
            break;
        case "move":
            playMoveAnimation(entry.target, entry.metadata.path);
            break;
        // ...
    }
}
```

---

### 2.3 Player（玩家状态）

**用途**：玩家完整状态快照。

```typescript
interface Player {
    player_id: string;     // 游戏内部 ID（UUID）
    client_id: string;     // 客户端标识（用于自识别）
    faction: string;       // 阵营（snake_case）
    position: number;      // 地图位置
    hp: number;            // 当前 HP
    lp: number;            // 当前 LP (0-8)
    buffs: Buff[];         // 激活的 Buff
    items: Item[];         // 拥有的道具
    charge: number;        // 阵营充能数（青龙/玄武）
    fire_counter: number;  // 朱雀火计数
    is_dead: boolean;      // 是否死亡
    skip_turn: boolean;    // 是否跳过回合
}
```

**faction 可选值**：

| 值 | 中文名 | 技能 |
|----|--------|------|
| `qing_long` | 青龙 | 行迹：每5回合充能，忽略负面地形 |
| `zhu_que` | 朱雀 | 离火：每4回合 LP+1 |
| `bai_hu` | 白虎 | 劫运：超越他人时偷取 Buff |
| `xuan_wu` | 玄武 | 镇厄：每5回合充能，取消坏事件 |

---

### 2.4 Buff（状态效果）

```typescript
interface Buff {
    type: string;     // Buff 类型标识
    name: string;     // 中文显示名
    duration: number; // 持续回合（-1 表示永久）
}
```

**type 可选值**：

| type | name | 效果 |
|------|------|------|
| `divine` | 神眷 | 每回合 LP+1 |
| `curse` | 诅咒 | 每回合 LP-1 |
| `lost` | 迷途 | 移动方向反转 |
| `hidden` | 隐匿 | 免疫伤害和事件 |
| `rain` | 甘霖 | 每2回合 HP+1 |
| `corrupt` | 腐化 | 每2回合 HP-1 |
| `exorcism` | 辟邪 | 免疫毒瘴事件 |
| `poison` | 毒瘴 | 每回合触发坏事件 |
| `fire` | 离火 | 朱雀被动（永久） |

---

### 2.5 Item（道具）

```typescript
interface Item {
    id: string;   // 道具实例 UUID
    type: string; // 道具类型标识
    name: string; // 中文显示名
}
```

**type 可选值**：

| type | name | 效果 |
|------|------|------|
| `reverse_clock` | 反方向的钟 | 回退到上一回合状态 |
| `any_door` | 任意门 | 传送到指定位置 |
| `dice_swap` | 骰子交换 | 与其他玩家交换骰子类型 |
| `dice_upgrade` | 骰子升级卡 | 升级骰子类型 |

---

### 2.6 Available（可用操作）

**用途**：通知当前玩家可执行的操作。

```typescript
interface Available {
    items: Item[];        // 可用道具列表
    // can_use_skill: boolean; // 阵营技能是否可用
    dice_type: string;    // 当前骰子类型
}
```

**dice_type 可选值**：

| 值 | 获得条件 | 权重分布 |
|----|----------|----------|
| `gold` | 小游戏第1名 | 高数值概率高 (5-6: 70%) |
| `silver` | 小游戏第2名 | 中等 (5-6: 50%) |
| `copper` | 小游戏第3名 | 偏低 (5-6: 40%) |
| `wood` | 小游戏第4名/默认 | 均匀 (16.67% each) |

---

### ~~2.7 Decision（决策请求）~~

**用途**：请求玩家选择。

```typescript
interface Decision {
    id: string;          // 决策 ID（回复时需匹配）
    prompt: string;      // 提示文本
    context: string;     // 来源标识
    options: Option[];   // 选项列表
    timeout: number;     // 超时秒数
    default: number;     // 默认选项索引（超时使用）
}

interface Option {
    id: string;          // 选项 ID
    label: string;       // 显示文本
    effect?: string;     // 效果预览（可选）
}
```

---

### 2.8 MiniGameStart（小游戏开始）

```typescript
interface MiniGameStart {
    game_type: string;  // 小游戏类型标识
    players: string[];  // 参赛玩家 ID 列表
}
```

**game_type 可选值**：

| 值 | 含义 |
|----|------|
| `dice_race` | 骰子竞速小游戏 |

---

### 2.9 MiniGameResult（小游戏结果）

```typescript
interface MiniGameResult {
    rankings: RankingEntry[]; // 排名列表
}

interface RankingEntry {
    player_id: string; // 玩家 ID
    rank: number;      // 排名 (1-4)
}
```

---

### 2.10 GameOver（游戏结束）

```typescript
interface GameOver {
    winner_id: string;     // 胜利玩家 ID
    stats: PlayerStats[];  // 统计数据
}

interface PlayerStats {
    player_id: string;    // 玩家 ID
    rounds_won: number;   // 小游戏第一名次数
    events_drawn: number; // 抽取事件次数
    items_used: number;   // 使用道具次数
}
```

---

### 2.11 ActionRejected（操作拒绝）

**用途**：通知客户端操作被拒绝，附带错误码。

```typescript
interface ActionRejected {
    op_code: number;      // 被拒绝的操作码
    error_code: string;   // 错误码（见错误码表）
    reason: string;       // 拒绝原因标识
    message: string;      // 人类可读消息
}
```

**错误码分类**：

| 范围 | 分类 |
|------|------|
| `OK` (0) | 成功 |
| `1001-1999` | 验证错误 |
| `2001-2999` | 游戏逻辑错误 |
| `3001-3999` | 系统错误 |
| `4001-4999` | 未找到错误 |

**常见错误码**：

| error_code | 含义 | reason |
|------------|------|--------|
| `not_current_turn` | 非当前回合玩家 | `not_current_player` |
| `invalid_state` | 无效状态 | `invalid_state` |
| `player_not_found` | 玩家未找到 | `player_not_found` |
| `item_not_found` | 道具未找到 | `item_not_found` |
| `skill_not_ready` | 技能未充能 | `skill_not_ready` |
| `condition_not_met` | 条件未满足 | `condition_not_met` |

---

### 2.12 WaitingSync（等待房间状态）

**用途**：广播等待房间状态（仅发送给主机）。

```typescript
interface WaitingSync {
    match_id: string;        // 比赛 ID
    host_user_id: string;    // 主机用户 ID
    players: WaitingPlayer[]; // 已加入玩家
    player_count: number;    // 玩家数量
    min_players: number;     // 最小玩家数 (2)
    max_players: number;     // 最大玩家数 (4)
    can_start: boolean;      // 是否可以开始
    message: string;         // 状态消息
}

interface WaitingPlayer {
    user_id: string;  // 用户 ID
    faction: string;  // 选择阵营
    is_host: boolean; // 是否主机
}
```

---

## 三、Client → Server 数据结构

### 3.1 RollDice（投骰子）

```typescript
interface RollDice {
    // 空结构，服务器根据玩家骰子类型计算结果
}
```

---

### 3.2 UseItem（使用道具）

```typescript
interface UseItem {
    item_id: string;          // 道具实例 ID
    target_id?: string;       // 目标玩家 ID（可选）
}
```

---

### 3.3 UseSkill（使用阵营技能）

```typescript
interface UseSkill {
    // 空结构，服务器检查玩家阵营和充能状态
}
```

---

### 3.4 UserChoice（决策回复）

```typescript
interface UserChoice {
    decision_id: string; // 对应 Decision.id
    choice: number;      // 选项索引 (0-based)
}
```

---

### 3.5 MiniGameResultSubmit（小游戏排名提交）

```typescript
interface MiniGameResultSubmit {
    rank: number; // 玩家自己的排名 (1-4)
}
```

---

### 3.6 StartGame（开始游戏）

```typescript
interface StartGame {
    // 空结构，服务器验证主机身份和最小玩家数
}
```

---

## 四、Action 类型与 Metadata 字段

客户端根据 `LogEntry.action_type` 判断应解析哪些 metadata 字段。

### 4.1 damage（伤害）

| 字段 | 类型 | 必填 | 用途 |
|------|------|------|------|
| `hp_change` | int | 是 | HP 变化值（负数） |
| `blocked_by` | string | 否 | 阻挡来源 Buff 名称 |
| `piercing` | bool | 否 | 是否穿透防御 |

**客户端渲染**：显示伤害数值动画，如有 `blocked_by` 显示阻挡提示，`piercing` 显示穿透图标。

---

### 4.2 heal（治疗）

| 字段 | 类型 | 必填 | 用途 |
|------|------|------|------|
| `hp_change` | int | 是 | HP 变化值（正数） |

**客户端渲染**：显示恢复数值动画。

---

### 4.3 modify_lp（LP 变化）

| 字段 | 类型 | 必填 | 用途 |
|------|------|------|------|
| `lp_change` | int | 是 | LP 变化值 |

**客户端渲染**：显示 LP 变化动画（神眷/诅咒效果）。

---

### 4.4 move（移动）

| 字段 | 类型 | 必填 | 用途 |
|------|------|------|------|
| `steps` | int | 是 | 移动步数 |
| `start_pos` | int | 是 | 起点位置 |
| `end_pos` | int | 是 | 终点位置 |
| `path` | []int | 是 | 移动路径（格子索引列表） |

**客户端渲染**：按 `path` 顺序播放移动动画。

**注意**：`path` 从 JSON 反序列化后可能是 `[]any`，需转换为 `[]int`。

---

### 4.5 add_buff（添加 Buff）

| 字段 | 类型 | 必填 | 用途 |
|------|------|------|------|
| `buff_type` | string | 是 | Buff 类型标识 |
| `duration` | int | 是 | 持续回合数（-1 永久） |

**客户端渲染**：显示获得 Buff 动画和持续时间。

---

### 4.6 remove_buff（移除 Buff）

| 字段 | 类型 | 必填 | 用途 |
|------|------|------|------|
| `buff_type` | string | 是 | Buff 类型标识 |

**客户端渲染**：显示移除 Buff 动画。

---

### 4.7 teleport（传送）

| 字段 | 类型 | 必填 | 用途 |
|------|------|------|------|
| `from_pos` | int | 是 | 传送起点 |
| `to_pos` | int | 是 | 传送终点 |

**客户端渲染**：播放传送动画（任意门道具）。

---

### 4.8 steal_buff（偷取 Buff）

| 字段 | 类型 | 必填 | 用途 |
|------|------|------|------|
| `stolen_by` | string | 是 | 偷取者玩家 ID |
| `buff_type` | string | 是 | 被偷 Buff 类型 |

**客户端渲染**：显示白虎劫运动画。

---

### 4.9 fell_down（落坑）

| 字段 | 类型 | 必填 | 用途 |
|------|------|------|------|
| `position` | int | 是 | 落坑位置 |
| `hp_change` | int | 是 | 坠落伤害（负数） |

**客户端渲染**：播放坠落动画和伤害数值。

---

### 4.10 respawn（重生）

| 字段 | 类型 | 必填 | 用途 |
|------|------|------|------|
| `checkpoint_pos` | int | 是 | 重生检查点位置 |

**客户端渲染**：播放重生动画。

---

### 4.11 draw_event（抽取事件）

| 字段 | 类型 | 必填 | 用途 |
|------|------|------|------|
| `event_type` | string | 是 | 事件类型标识 |
| `event_name` | string | 是 | 事件中文名 |

**客户端渲染**：显示事件卡片。

---

### 4.12 dice_roll（骰子结果）

| 字段 | 类型 | 必填 | 用途 |
|------|------|------|------|
| `dice_type` | string | 是 | 骰子类型 |
| `dice_steps` | int | 是 | 骰子结果 |

**客户端渲染**：显示骰子动画和数值。

---

### 4.13 state（状态转换）

| 字段 | 类型 | 必填 | 用途 |
|------|------|------|------|
| `from` | string | 是 | 状态转换起点 |
| `to` | string | 是 | 状态转换终点 |

**客户端渲染**：通常用于内部日志，不显示动画。

---

## 五、LogEntry 基础结构

```typescript
interface LogEntry {
    timestamp: string;     // 时间戳
    type: string;          // "action" 或 "state"
    action_type?: string;  // Action 类型（type=action 时）
    target?: string;       // 目标玩家 ID
    source?: string;       // 效果来源标识
    metadata?: object;     // 类型特定字段
}
```

**source 常见值**：

| 值 | 含义 |
|----|------|
| `Buff_Divine` | 神眷 Buff |
| `Buff_Curse` | 诅咒 Buff |
| `Buff_Fire` | 离火 Buff |
| `Cell_Fragile` | 易碎格子 |
| `Cell_Fog` | 迷雾格子 |
| `Item_AnyDoor` | 任意门道具 |
| `Faction_BaiHu` | 白虎阵营技能 |
| `DiceRoll` | 骰子投掷 |

---

## 六、消息发送时机详解

本节详细说明每个消息的精确发送时机，帮助前端开发者正确处理消息流。

### 6.1 Server → Client 消息时机

#### OpWaitingSync (10)

**发送时机**：
1. **玩家加入等待房间**：当新玩家连接到 match 且当前状态为 `WaitingForHost` 时
2. **玩家离开等待房间**：当玩家断开连接且当前状态为 `WaitingForHost` 时

**接收对象**：广播给所有已连接玩家

**触发条件**：`hsm.GetGlobalStateID() == StateWaitingForHost`

**相关代码**：`internal/nakama/presence.go` - `HandlePresenceJoin/HandlePresenceLeave`

---

#### OpStateSync (1)

**发送时机**：

| 状态 | 发送时机 | 目的 |
|------|----------|------|
| `MatchInit.Enter` | 游戏初始化完成（生成地图、分配阵营、初始化 Buff）后 | 广播初始游戏状态 |
| `TurnUpkeep.Enter` | 进入回合维护阶段时 | 更新回合玩家状态 |
| `MainAction.Enter` | 进入主行动阶段时 | 更新当前行动玩家状态 |
| `TurnEnd.Enter` | 回合结束时 | 广播回合结束后的最终状态 |

**接收对象**：广播给所有玩家

**客户端行为**：更新 UI 显示当前全局状态、回合状态、玩家列表

---

#### OpAvailable (4)

**发送时机**：`MainAction.Enter` 时，紧随 `OpStateSync` 之后发送

**接收对象**：仅发送给当前回合玩家（`current_player_id`）

**触发条件**：进入 `MainAction` 状态时

**客户端行为**：
1. 显示骰子类型（`dice_type`）
2. 显示可用道具列表（`items`）
3. 启用投骰子按钮
4. 启用道具使用按钮（如果有道具）

---

#### OpMiniGameStart (5)

**发送时机**：`RoundMiniGame.Enter` 时

**接收对象**：广播给所有玩家

**触发条件**：进入小游戏阶段（每回合开始）

**客户端行为**：
1. 切换到小游戏界面
2. 根据 `game_type` 加载对应小游戏
3. 显示参赛玩家列表

---

#### OpMiniGameResult (6)

**发送时机**：所有玩家提交 `MiniGameResultSubmit` 后，退出 `RoundMiniGame` 时

**接收对象**：广播给所有玩家

**触发条件**：小游戏结束，准备进入 `RoundPrep`

**客户端行为**：
1. 显示排名结果
2. 根据排名显示骰子分配（第1名=金骰，第2名=银骰...）

---

#### OpTurnSync (2)

**发送时机**：

| 状态 | 发送时机 | 包含内容 |
|------|----------|----------|
| `TurnEnd.Enter` | 回合结束时 | 该回合所有 Action 效果 |

**注意**：`TurnSync` 在 `TurnEnd.Enter` 时发送，包含整个回合的 `LogEntry` 列表

**接收对象**：广播给所有玩家

**客户端行为**：
1. 按顺序遍历 `entries` 数组
2. 根据 `action_type` 播放对应动画
3. 动画完成后等待下一个 `StateSync`

---

#### OpGameOver (7)

**发送时机**：有玩家到达终点并击败 Boss 时

**接收对象**：广播给所有玩家

**触发条件**：游戏结束条件达成

**客户端行为**：
1. 显示胜利玩家
2. 显示统计数据
3. 显示结算界面

---

#### OpFullSync (8)

**发送时机**：
1. 玩家断线重连时
2. 新玩家中途加入正在进行的游戏时

**接收对象**：仅发送给重连/新加入的玩家

**触发条件**：`HandlePresenceJoin` 且游戏已在进行中（`hsm.IsRunning()`）

**客户端行为**：
1. 从 `state` 恢复完整游戏状态
2. 从 `turn` 恢复当前回合效果
3. 同步显示当前界面

---

#### OpActionRejected (9)

**发送时机**：客户端发送的操作被服务器拒绝时

**接收对象**：仅发送给发送操作的玩家

**触发条件**：
- 非当前回合玩家发送操作
- 无效状态时发送操作
- 道具不存在
- 技能未充能
- 条件未满足

**客户端行为**：
1. 根据 `error_code` 显示错误提示
2. 禁用相关按钮或重置状态

---

### 6.2 Client → Server 消息时机

#### OpStartGame (105)

**发送时机**：主机在等待房间点击"开始游戏"按钮时

**前置条件**：
1. 发送者必须是主机（`host_user_id`）
2. 玩家数量 >= 2

**服务器响应**：
- 成功：进入 `MatchInit` → `WaitingForHost` → `RoundMiniGame`
- 失败：发送 `OpActionRejected`

---

#### OpMiniGameResultSubmit (104)

**发送时机**：小游戏结束后，客户端计算出排名时

**前置条件**：
1. 当前状态为 `RoundMiniGame`
2. 小游戏已完成

**服务器响应**：
- 成功：等待所有玩家提交后进入 `RoundPrep`
- 失败：发送 `OpActionRejected`

---

#### OpRollDice (100)

**发送时机**：当前回合玩家在 `MainAction` 阶段点击投骰子按钮时

**前置条件**：
1. 发送者必须是当前回合玩家（`current_player_id`）
2. 当前状态为 `MainAction`

**服务器响应**：
- 成功：进入 `TurnMoving` → 执行移动
- 失败：发送 `OpActionRejected`（`not_current_player`）

---

#### OpUseItem (101)

**发送时机**：当前回合玩家在 `MainAction` 阶段选择使用道具时

**前置条件**：
1. 发送者必须是当前回合玩家
2. 当前状态为 `MainAction`
3. 玩家拥有该道具

**服务器响应**：
- 成功：执行道具效果，可能触发衍生 Action
- 失败：发送 `OpActionRejected`（`item_not_found`）

---

#### OpUseSkill (102)

**发送时机**：当前回合玩家使用阵营技能时

**前置条件**：
1. 发送者必须是当前回合玩家
2. 玩家阵营为青龙或玄武
3. `charge >= 1`（充能已满）

**服务器响应**：
- 成功：执行阵营技能，消耗充能
- 失败：发送 `OpActionRejected`（`skill_not_ready`）

---

### 6.3 消息时序图

#### 游戏启动流程

```
[Client Host]              [Server]                  [All Clients]
     |                        |                          |
     |--- OpStartGame ------->|                          |
     |                        |--- OpStateSync --------->| (MatchInit完成)
     |                        |                          |
     |                        |<== 状态转换 ==>           |
     |                        |                          |
     |                        |--- OpMiniGameStart ----->| (RoundMiniGame)
     |                        |                          |
```

#### 小游戏流程

```
[All Clients]              [Server]
     |                        |
     |<== 小游戏执行 ==>       |
     |                        |
     |--- OpMiniGameResultSubmit -->| (每个玩家发送)
     |                        |
     |<== 等待所有玩家提交 ==>  |
     |                        |
     |<-- OpMiniGameResult ---| (广播排名)
     |                        |
     |<-- OpStateSync --------| (RoundPrep完成)
     |                        |
```

#### 回合流程

```
[Current Player]           [Server]                  [All Clients]
     |                        |                          |
     |                        |--- OpStateSync --------->| (TurnUpkeep)
     |                        |                          |
     |                        |--- OpAvailable --------->| (仅发给当前玩家)
     |                        |                          |
     |<== 等待玩家操作 ==>     |                          |
     |                        |                          |
     |--- OpRollDice -------->|                          |
     |                        |<== 执行移动 ==>           |
     |                        |                          |
     |                        |<== 状态转换 ==>           |
     |                        |--- OpTurnSync ---------->| (TurnEnd)
     |                        |--- OpStateSync --------->| (回合结束状态)
     |                        |                          |
     |<== 下一个玩家回合 ==>   |                          |
```

#### 断线重连流程

```
[Reconnecting Player]      [Server]
     |                        |
     |<== 重新连接 ==>         |
     |                        |
     |<-- OpFullSync ---------| (恢复完整状态)
     |                        |
     |<== 同步显示 ==>         |
     |                        |
```

---

### 6.4 客户端状态对应表

| global_state | 客户端应显示的界面 |
|---------------|-------------------|
| `waiting_for_host` | 等待房间界面，显示玩家列表、等待主机开始 |
| `round_mini_game` | 小游戏界面，根据 `game_type` 加载 |
| `round_prep` | 回合准备界面，显示骰子分配结果 |
| `turn_loop` | 回合循环界面，显示地图和玩家位置 |
| `game_over` | 结算界面，显示胜利者和统计数据 |

| turn_state | 客户端应显示的 UI 元素 |
|------------|----------------------|
| `turn_upkeep` | 回合开始动画（LP 变化等） |
| `main_action` | 操作面板（投骰子、道具、技能按钮），等待当前玩家操作 |
| `turn_moving` | 移动动画播放中 |
| `turn_landed` | 落地效果动画 |
| `turn_event` | 事件卡片显示 |
| `turn_end` | 回合结束动画（Buff 消耗、甘霖/腐化触发） |

---

## 七、游戏流程图

```
┌─────────────────────────────────────────────────────────────┐
│                      WaitingForHost                          │
│  主机等待玩家加入，广播 WaitingSync                            │
│  主机发送 OpStartGame 开始游戏                                │
└───────────────────────┬─────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│                      MatchInit                               │
│  生成地图、分配阵营、初始化 Buff                                │
│  广播 StateSync                                              │
└───────────────────────┬─────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│                    RoundMiniGame                             │
│  广播 MiniGameStart                                          │
│  等待所有玩家提交 MiniGameResultSubmit                         │
│  清空 RoundData                                              │
└───────────────────────┬─────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│                      RoundPrep                               │
│  根据 rank 分配骰子类型                                        │
│  根据 rank 排序玩家行动顺序                                    │
│  广播 StateSync                                              │
└───────────────────────┬─────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│                      TurnLoop                                │
│  循环每个玩家回合                                              │
└───────────────────────┬─────────────────────────────────────┘
                        ↓
        ┌───────────────┴───────────────┐
        ↓                               ↓
┌───────────────────┐           ┌───────────────────┐
│    TurnUpkeep     │           │    BossBattle     │
│  BeforeTurn Phase │           │  Boss 战斗        │
│  LP±1 效果触发     │           │  广播 GameOver    │
└─────────┬─────────┘           └───────────────────┘
          ↓
┌───────────────────┐
│    MainAction     │
│  发送 Available   │
│  等待 RollDice    │
│  或 UseItem       │
│  或 UseSkill      │
└─────────┬─────────┘
          ↓
┌───────────────────┐
│    TurnMoving     │
│  执行 MoveAction  │
│  广播 TurnSync    │
│  检查 Fragile     │
└─────────┬─────────┘
          ↓
┌───────────────────┐
│    TurnLanded     │
│  OnLand Phase     │
│  检查到达终点      │
└─────────┬─────────┘
          ↓
┌───────────────────┐
│    TurnEvent      │
│  PreEvent Phase   │
│  抽取随机事件      │
│  广播 TurnSync    │
└─────────┬─────────┘
          ↓
┌───────────────────┐
│    TurnEnd        │
│  AfterTurn Phase  │
│  Buff Duration-1  │
│  甘霖/腐化触发     │
└─────────┬─────────┘
          ↓
    (返回 TurnLoop)
```

---

## 八、客户端 TypeScript 类型汇总

```typescript
// ========== OpCode 枚举 ==========

enum OpCode {
    // Server → Client
    StateSync = 1,
    TurnSync = 2,
    DecisionRequest = 3,
    Available = 4,
    MiniGameStart = 5,
    MiniGameResult = 6,
    GameOver = 7,
    FullSync = 8,
    ActionRejected = 9,
    WaitingSync = 10,

    // Client → Server
    RollDice = 100,
    UseItem = 101,
    UseSkill = 102,
    UserChoice = 103,
    MiniGameResultSubmit = 104,
    StartGame = 105,
}

// ========== 数据结构 ==========

interface StateSync {
    global_state: string;
    turn_state: string;
    current_player_id: string;
    round: number;
    turn: number;
    paused: boolean;
    players: Player[];
}

interface TurnSync {
    round: number;
    turn: number;
    current_player_id: string;
    entries: LogEntry[];
}

interface Player {
    player_id: string;
    client_id: string;
    faction: string;
    position: number;
    hp: number;
    lp: number;
    buffs: Buff[];
    items: Item[];
    charge: number;
    fire_counter: number;
    is_dead: boolean;
    skip_turn: boolean;
}

interface Buff {
    type: string;
    name: string;
    duration: number;
}

interface Item {
    id: string;
    type: string;
    name: string;
}

interface Available {
    items: Item[];
    can_use_skill: boolean;
    dice_type: string;
}

interface Decision {
    id: string;
    prompt: string;
    context: string;
    options: Option[];
    timeout: number;
    default: number;
}

interface Option {
    id: string;
    label: string;
    effect?: string;
}

interface MiniGameStart {
    game_type: string;
    players: string[];
}

interface MiniGameResult {
    rankings: RankingEntry[];
}

interface RankingEntry {
    player_id: string;
    rank: number;
}

interface GameOver {
    winner_id: string;
    stats: PlayerStats[];
}

interface PlayerStats {
    player_id: string;
    rounds_won: number;
    events_drawn: number;
    items_used: number;
}

interface ActionRejected {
    op_code: number;
    error_code: string;
    reason: string;
    message: string;
}

interface WaitingSync {
    match_id: string;
    host_user_id: string;
    players: WaitingPlayer[];
    player_count: number;
    min_players: number;
    max_players: number;
    can_start: boolean;
    message: string;
}

interface WaitingPlayer {
    user_id: string;
    faction: string;
    is_host: boolean;
}

interface FullSync {
    state: StateSync;
    turn: TurnSync;
}

// ========== LogEntry ==========

interface LogEntry {
    timestamp: string;
    type: string;
    action_type?: string;
    target?: string;
    source?: string;
    metadata?: {
        hp_change?: number;
        lp_change?: number;
        duration?: number;
        steps?: number;
        start_pos?: number;
        end_pos?: number;
        path?: number[];
        blocked_by?: string;
        piercing?: boolean;
        buff_type?: string;
        event_type?: string;
        event_name?: string;
        from_pos?: number;
        to_pos?: number;
        position?: number;
        checkpoint_pos?: number;
        stolen_by?: string;
        dice_type?: string;
        dice_steps?: number;
        from?: string;
        to?: string;
    };
}

// ========== Client Request ==========

interface RollDice {}

interface UseItem {
    item_id: string;
    target_id?: string;
}

interface UseSkill {}

interface UserChoice {
    decision_id: string;
    choice: number;
}

interface MiniGameResultSubmit {
    rank: number;
}

interface StartGame {}
```

---

## 九、相关文档

- [pkg/net/README.md](../pkg/net/README.md) - 网络协议层详细说明
- [doc/metadata/logentry.md](../doc/metadata/logentry.md) - LogEntry.Metadata 契约
- [doc/metadata/player.md](../doc/metadata/player.md) - Player.Metadata 契约
- [internal/engine/action/README.md](../internal/engine/action/README.md) - Action 实现
- [internal/engine/hsm/README.md](../internal/engine/hsm/README.md) - HSM 状态机
- [doc/protocol_hsm_interaction.md](../doc/protocol_hsm_interaction.md) - 协议与 HSM 交互流程