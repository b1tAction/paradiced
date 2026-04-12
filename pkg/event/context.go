package event

// Context 执行上下文，包含Phase触发时的所有相关信息
// 注意：为了避免循环依赖，Player和GameEvent使用interface{}类型
// 实际使用时由调用方传入具体类型
type Context struct {
	Player     interface{} `json:"player"`     // 触发Phase的玩家（具体类型由调用方决定）
	GameEvent  interface{} `json:"game_event"` // 相关的游戏事件（可选）
	GameState  *GameState  `json:"game_state"` // 游戏状态（可选）
	Choice     int         `json:"choice"`     // 用户选择的选项索引
	Data       interface{} `json:"data"`       // 额外数据（如伤害值、事件类型等）
}

// GameState 游戏状态快照（简化版，后续可扩展）
type GameState struct {
	Round        int         `json:"round"`         // 当前轮次
	Turn         int         `json:"turn"`          // 当前回合
	CurrentPhase Phase       `json:"current_phase"` // 当前Phase
	AllPlayers   []interface{} `json:"all_players"`  // 所有玩家
}

// NewContext 创建新的上下文
func NewContext(player interface{}) *Context {
	return &Context{
		Player: player,
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

// WithData 设置额外数据
func (c *Context) WithData(data interface{}) *Context {
	c.Data = data
	return c
}

// WithChoice 设置用户选择
func (c *Context) WithChoice(choice int) *Context {
	c.Choice = choice
	return c
}