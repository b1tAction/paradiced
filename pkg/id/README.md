# pkg/id - Typed ID Wrapper System

提供类型安全的 ID 包装系统，基于 UUID v7 实现。

## 设计目标

- **类型安全**：编译时区分不同 ID 类型（PlayerID vs BuffID）
- **领域建模**：每个 ID 类型有明确的语义
- **验证能力**：可验证 ID 格式正确性
- **JSON 兼容**：保持协议层序列化兼容
- **调试友好**：内部前缀便于日志识别

## ID 类型一览

| 类型 | 前缀 | 用途 |
|------|------|------|
| PlayerID | "player" | 玩家唯一标识 |
| BuffID | "buff" | Buff 实例标识 |
| ItemID | "item" | Item 实例标识 |
| GameID | "game" | 游戏实例标识 |
| SubscriptionID | "sub" | EventBus 订阅标识 |
| DecisionID | "dec" | Decision 决策标识 |

## 核心结构

```go
type ID struct {
    prefix string    // 内部前缀（调试用）
    uuid   uuid.UUID // UUID v7
}

// PlayerID 等类型通过嵌入 ID 实现
type PlayerID struct { ID }
type BuffID struct { ID }
type ItemID struct { ID }
type GameID struct { ID }
type SubscriptionID struct { ID }
type DecisionID struct { ID }
```

## 序列化规则

**关键设计**：内部调试使用前缀格式，协议传输使用纯 UUID。

```go
// String() - 用于日志/调试
func (id ID) String() string {
    return id.prefix + "-" + id.uuid.String()  // "player-019d8c44-..."
}

// UUID() - 用于协议传输
func (id ID) UUID() string {
    return id.uuid.String()  // "019d8c44-..."
}

// MarshalJSON() - 序列化为纯 UUID
func (id ID) MarshalJSON() ([]byte, error) {
    return []byte(`"` + id.uuid.String() + `"`), nil
}
```

**协议示例**：
```json
// 前端传入纯 UUID，后端根据字段名解析为对应类型
{ "item_id": "019d8c44-b9ea-7931-9712-f46b71a35374" }

// 后端输出纯 UUID 给前端
{ "id": "019d8c44-b9ea-7931-9712-f46b71a35374", "type": "curse" }
```

**类型识别规则**：
- API 字段名决定类型（`item_id` → ItemID, `player_id` → PlayerID）
- UnmarshalJSON 不解析 prefix，由类型自身设置
- 内部调试用 String() 带 prefix 识别

## 使用示例

```go
// 创建新 ID
playerID := id.NewPlayerID()
buffID := id.NewBuffID()

// 解析 ID
parsed, err := id.ParsePlayerID("019d8c44-b9ea-7931-9712-f46b71a35374")

// 协议传输
json.Marshal(playerID)  // 输出纯 UUID 字符串

// 调试日志
log.Printf("Player: %s", playerID.String())  // "player-019d8c44-..."

// 类型安全（编译时检查）
func GetPlayer(id id.PlayerID) *Player  // 只接受 PlayerID
func GetBuff(id id.BuffID) *Buff        // 只接受 BuffID

// 测试辅助函数
// TestUUID 生成用于测试的合法 UUID
testID := id.TestUUID(1)  // "00000000-0000-0000-0000-000000000001"
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

- ID 生成和解析
- JSON 序列化/反序列化
- UUID 格式验证
- 前缀调试输出
- 类型比较和零值检查
