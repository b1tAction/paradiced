package event

import (
	"github.com/b1tAction/Fated/pkg/util"
)

// Context 执行上下文，包含Phase触发时的所有相关信息
// 注意：为了避免循环依赖，Player和GameEvent使用interface{}类型
// 实际使用时由调用方传入具体类型
type Context struct {
	Player     interface{} `json:"player"`     // 触发Phase的玩家（具体类型由调用方决定）
	GameEvent  interface{} `json:"game_event"` // 相关的游戏事件（可选）
	GameState  *GameState  `json:"game_state"` // 游戏状态（可选）
	Choice     int         `json:"choice"`     // 用户选择的选项索引
	*util.Metadata          `json:"metadata"`   // 类型安全的动态数据容器（替代原 Data interface{}）
}

// GameState 游戏状态快照（简化版，后续可扩展）
type GameState struct {
	Round        int           `json:"round"`         // 当前轮次
	Turn         int           `json:"turn"`          // 当前回合
	CurrentPhase Phase         `json:"current_phase"` // 当前Phase
	AllPlayers   []interface{} `json:"all_players"`   // 所有玩家
}

// NewContext 创建新的上下文
func NewContext(player interface{}) *Context {
	return &Context{
		Player:   player,
		Metadata: util.NewMetadata(),
	}
}

// WithEvent 设置游戏事件
func (c *Context) WithEvent(event interface{}) *Context {
	c.GameEvent = event
	return c
}

// WithState 设置游戏状态
func (c *Context) WithState(state *GameState) *Context {
	c.GameState = state
	return c
}

// WithData 设置额外数据（向后兼容方法，内部使用 Metadata）
// Deprecated: 建议直接使用 Set(key, value) 或类型特定的 SetInt/SetString 等
func (c *Context) WithData(data interface{}) *Context {
	c.Set("data", data)
	return c
}

// GetData 获取额外数据（向后兼容方法）
// Deprecated: 建议根据数据类型使用 GetInt/GetString/Get 等
func (c *Context) GetData() interface{} {
	if val, ok := c.Get("data"); ok {
		return val
	}
	return nil
}

// WithChoice 设置用户选择
func (c *Context) WithChoice(choice int) *Context {
	c.Choice = choice
	return c
}

// Clone 克隆上下文（用于测试或需要独立副本的场景）
func (c *Context) Clone() *Context {
	return &Context{
		Player:     c.Player,
		GameEvent:  c.GameEvent,
		GameState:  c.GameState,
		Choice:     c.Choice,
		Metadata:   c.Metadata.Clone(),
	}
}