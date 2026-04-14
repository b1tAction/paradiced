# pkg/id - Typed ID Wrapper System

提供类型安全的ID包装系统，基于UUID v7实现。

## 设计目标

- **类型安全**：编译时区分不同ID类型（PlayerID vs BuffID）
- **领域建模**：每个ID类型有明确的语义
- **验证能力**：可验证ID格式正确性
- **JSON兼容**：保持协议层序列化兼容
- **调试友好**：内部前缀便于日志识别

## ID类型一览

| 类型 | 前缀 | 用途 |
|------|------|------|
| PlayerID | "player" | 玩家唯一标识 |
| BuffID | "buff" | Buff实例标识 |
| ItemID | "item" | Item实例标识 |
| GameID | "game" | 游戏实例标识 |
| SubscriptionID | "sub" | EventBus订阅标识 |
| DecisionID | "dec" | Decision决策标识 |

## 核心结构

```go
type ID struct {
    prefix string    // 内部前缀（调试用）
    uuid   uuid.UUID // UUID v7
}

// PlayerID 等类型通过嵌入ID实现
type PlayerID struct { ID }
type BuffID struct { ID }
type ItemID struct { ID }
type GameID struct { ID }
type SubscriptionID struct { ID }
type DecisionID struct { ID }
```

## 序列化规则

**关键设计**：内部调试使用前缀格式，协议传输使用纯UUID。

```go
// String() - 用于日志/调试
func (id ID) String() string {
    return id.prefix + "-" + id.uuid.String()  // "player-019d8c44-..."
}

// UUID() - 用于协议传输
func (id ID) UUID() string {
    return id.uuid.String()  // "019d8c44-..."
}

// MarshalJSON() - 序列化为纯UUID
func (id ID) MarshalJSON() ([]byte, error) {
    return []byte(`"` + id.uuid.String() + `"`), nil
}
```

**协议示例**：
```json
// 前端传入纯UUID，后端根据字段名解析为对应类型
{ "item_id": "019d8c44-b9ea-7931-9712-f46b71a35374" }

// 后端输出纯UUID给前端
{ "id": "019d8c44-b9ea-7931-9712-f46b71a35374", "type": "curse" }
```

**类型识别规则**：
- API字段名决定类型（`item_id` → ItemID, `player_id` → PlayerID）
- UnmarshalJSON不解析prefix，由类型自身设置
- 内部调试用String()带prefix识别

## 使用示例

```go
// 创建新ID
playerID := id.NewPlayerID()
buffID := id.NewBuffID()

// 解析ID
parsed, err := id.ParsePlayerID("019d8c44-b9ea-7931-9712-f46b71a35374")

// 协议传输
json.Marshal(playerID)  // 输出纯UUID字符串

// 调试日志
log.Printf("Player: %s", playerID.String())  // "player-019d8c44-..."

// 类型安全（编译时检查）
func GetPlayer(id id.PlayerID) *Player  // 只接受PlayerID
func GetBuff(id id.BuffID) *Buff        // 只接受BuffID
```

## 设计限制

**本项目禁止使用类型别名**：

```go
// ❌ 禁止的写法
type PlayerID = id.PlayerID

// ✅ 正确的写法 - 直接使用
func (p *Player) GetID() id.PlayerID {
    return p.ID
}
```

## 测试覆盖

- ID生成和解析
- JSON序列化/反序列化
- UUID格式验证
- 前缀调试输出
- 类型比较和零值检查