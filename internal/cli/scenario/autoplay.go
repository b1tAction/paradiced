// Package scenario provides automated play scenarios for testing.
package scenario

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/b1tAction/paradiced/internal/cli/model"
	"github.com/b1tAction/paradiced/internal/cli/nakama"
)

// ScenarioConfig holds configuration for a play scenario.
type ScenarioConfig struct {
	PlayersCount int
	MatchName    string
	MaxTurns     int
	TimeoutSec   int
}

// Result holds the result of a scenario run.
type Result struct {
	Success          bool          `json:"success"`
	FailureReason    string        `json:"failure_reason,omitempty"`
	Duration         time.Duration `json:"duration"`
	MessagesReceived int           `json:"messages_received"`
	TurnsCompleted   int           `json:"turns_completed"`
	GlobalState      string        `json:"global_state"`
}

// RunAutoPlay runs an autoplay scenario with the given configuration.
// This function uses matchmaker to create an authoritative match.
// Note: Requires at least 2 players for matchmaker to work.
func RunAutoPlay(ctx context.Context, client *nakama.Client, config ScenarioConfig, logger *nakama.Logger) (Result, error) {
	startTime := time.Now()

	result := Result{
		Success: false,
	}

	// Ensure minimum 2 players for matchmaker
	playersCount := config.PlayersCount
	if playersCount < 2 {
		playersCount = 2
		logger.Info("玩家数量不足 2 人，已自动调整为 2 人进行测试")
	}

	logger.Info("开始对局测试", "players", playersCount, "max_turns", config.MaxTurns)

	// Create socket clients for each player
	socketClients := make([]*nakama.SocketClient, playersCount)
	players := make([]*AutoPlayPlayer, playersCount)

	// Cleanup on exit
	defer func() {
		for _, sc := range socketClients {
			if sc != nil {
				sc.Close()
			}
		}
	}()

	// Channel to receive match ID from matchmaker
	matchChan := make(chan string, playersCount)
	tokenChan := make(chan string, playersCount)

	// Create and authenticate all players
	for i := 0; i < playersCount; i++ {
		var err error
		playerID := fmt.Sprintf("cli_bot_%02d", i+1)

		// Create socket client
		socketClients[i], err = nakama.NewSocketClient(client)
		if err != nil {
			return result, fmt.Errorf("创建 socket 客户端 %d 失败：%w", i+1, err)
		}

		// Authenticate
		session, err := client.Authenticate(ctx, playerID)
		if err != nil {
			return result, fmt.Errorf("认证玩家 %d 失败：%w", i+1, err)
		}

		// Connect socket
		if err := socketClients[i].Connect(ctx, session); err != nil {
			return result, fmt.Errorf("连接 socket %d 失败：%w", i+1, err)
		}

		// Create autoplay player
		players[i] = NewAutoPlayPlayer(socketClients[i], playerID, logger)

		// Set matchmaker handler BEFORE adding to matchmaker
		socketClients[i].SetMatchmakerMatchedHandler(func(ctx context.Context, msg *nakama.MatchmakerMatchedMsg) {
			// Matchmaker can return either a match_id or a token
			// If match_id is empty, use token to join the match
			matchID := msg.GetMatchId()
			token := msg.GetToken()

			if matchID != "" {
				logger.Info("匹配器匹配成功，返回 match_id", "match_id", matchID)
				select {
				case matchChan <- matchID:
				default:
				}
			} else if token != "" {
				logger.Info("匹配器匹配成功，返回 token", "token", token)
				select {
				case tokenChan <- token:
				default:
				}
			} else {
				logger.Warn("匹配器匹配成功，但未返回 match_id 或 token")
			}
		})
	}

	logger.Info("所有玩家已连接，开始匹配...")

	// Add all players to matchmaker
	// Query: match players who want to join "paradiced" match
	// Note: This creates a realtime match, not authoritative match
	// For authoritative match, we need to use matchmaker with proper configuration
	query := "properties.match:paradiced"
	props := map[string]string{"match": "paradiced"}

	tickets := make([]*nakama.MatchmakerTicketMsg, playersCount)
	for i := 0; i < playersCount; i++ {
		var addErr error
		// Use min=max=playersCount to ensure all bots are matched together
		tickets[i], addErr = socketClients[i].AddMatchmaker(ctx, query, playersCount, playersCount, props, nil)
		if addErr != nil {
			return result, fmt.Errorf("添加玩家 %d 到匹配器失败：%w", i+1, addErr)
		}
		logger.Info("玩家已添加到匹配器", "player", i+1, "ticket", tickets[i].Ticket)

		// Small delay between adding players
		if i < playersCount-1 {
			time.Sleep(200 * time.Millisecond)
		}
	}

	// Wait for all matchmaker matched events
	matchIDs := make(map[string]int)
	tokens := make(map[string]int)
	matchTimeout := time.After(30 * time.Second)

	// Wait for match_id or token from all players
	for i := 0; i < playersCount; i++ {
		select {
		case matchID := <-matchChan:
			matchIDs[matchID]++
			logger.Info("收到匹配 ID", "match_id", matchID, "count", i+1)
		case token := <-tokenChan:
			tokens[token]++
			logger.Info("收到 Token", "token", token, "count", i+1)
		case <-matchTimeout:
			return result, fmt.Errorf("匹配器超时 (收到 %d/%d 响应)", len(matchIDs)+len(tokens), playersCount)
		}
	}

	// Get final match ID or join with token
	var finalMatchID string
	if len(matchIDs) > 0 {
		// Received match_id directly - still need to join the match
		for mid := range matchIDs {
			finalMatchID = mid
		}
		logger.Info("使用匹配 ID 加入匹配", "match_id", finalMatchID)
		// All players need to join the match
		for i := 0; i < playersCount; i++ {
			err := socketClients[i].JoinMatch(ctx, finalMatchID)
			if err != nil {
				return result, fmt.Errorf("玩家 %d 加入匹配失败：%w", i+1, err)
			}
		}
	} else if len(tokens) > 0 {
		// Received tokens - all players should have the same token
		// Join match with token
		for token, count := range tokens {
			if count == playersCount {
				// All players have the same token, join the match
				logger.Info("所有玩家有相同 token，加入匹配", "token", token)
				// All players need to join the match for authoritative match to start
				for i := 0; i < playersCount; i++ {
					err := socketClients[i].JoinMatch(ctx, token)
					if err != nil {
						return result, fmt.Errorf("玩家 %d 加入匹配失败：%w", i+1, err)
					}
				}
				// The match_id will be set by JoinMatch (from last player)
				// For now, we'll use the token as the match ID placeholder
				finalMatchID = token
				break
			}
		}
		if finalMatchID == "" {
			return result, fmt.Errorf("玩家 token 不一致，无法加入匹配")
		}
	}

	if finalMatchID == "" {
		return result, fmt.Errorf("未收到匹配 ID 或 token")
	}

	// Set match ID for all socket clients (use the actual match ID from first join)
	for i := 0; i < playersCount; i++ {
		socketClients[i].SetMatchID(finalMatchID)
	}

	logger.Info("所有玩家已加入匹配", "match_id", finalMatchID, "players", playersCount)

	// Start listening for messages for all players
	mainPlayer := players[0]
	for i := 0; i < playersCount; i++ {
		go players[i].Listen(ctx)
	}

	// Wait for game to complete or timeout
	timeout := time.Duration(config.TimeoutSec) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			result.FailureReason = "timeout"
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("超时 (%.1f 秒)", timeout.Seconds())

		case <-mainPlayer.GameOverChan():
			result.Success = true
			result.Duration = time.Since(startTime)
			result.MessagesReceived = mainPlayer.MessagesReceived()
			result.TurnsCompleted = mainPlayer.TurnsCompleted()
			result.GlobalState = "game_over"
			return result, nil

		case <-ticker.C:
			if err := mainPlayer.LastError(); err != nil {
				result.FailureReason = err.Error()
				result.Duration = time.Since(startTime)
				return result, err
			}
			if mainPlayer.GlobalState() != "" {
				result.GlobalState = mainPlayer.GlobalState()
			}
		}
	}
}

// AutoPlayPlayer represents an automated player.
type AutoPlayPlayer struct {
	socket           *nakama.SocketClient
	userID           string
	logger           *nakama.Logger
	msgChan          <-chan *nakama.SocketMessage
	gameOverChan     chan struct{}
	mu               sync.RWMutex
	messagesReceived int
	turnsCompleted   int
	globalState      string
	currentDecision  *model.Decision
	lastErr          error
}

// NewAutoPlayPlayer creates a new autoplay player.
func NewAutoPlayPlayer(socket *nakama.SocketClient, userID string, logger *nakama.Logger) *AutoPlayPlayer {
	return &AutoPlayPlayer{
		socket:       socket,
		userID:       userID,
		logger:       logger,
		msgChan:      socket.MessageChan(),
		gameOverChan: make(chan struct{}, 1),
	}
}

// Listen starts listening for messages and handling them.
func (p *AutoPlayPlayer) Listen(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-p.msgChan:
			if !ok {
				return
			}
			p.handleMessage(ctx, msg)
		}
	}
}

// handleMessage processes an incoming message.
func (p *AutoPlayPlayer) handleMessage(ctx context.Context, msg *nakama.SocketMessage) {
	p.mu.Lock()
	p.messagesReceived++
	p.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	switch msg.OpCode {
	case nakama.OpStateSync:
		p.handleStateSync(ctx, msg.Data)
	case nakama.OpAvailable:
		p.handleAvailable(ctx, msg.Data)
	case nakama.OpDecisionRequest:
		p.handleDecisionRequest(ctx, msg.Data)
	case nakama.OpMiniGameStart:
		p.handleMiniGameStart(ctx, msg.Data)
	case nakama.OpGameOver:
		p.handleGameOver(ctx, msg.Data)
	case nakama.OpTurnSync:
		p.mu.Lock()
		p.turnsCompleted++
		p.mu.Unlock()
		p.logger.Debug("收到 TurnSync")
	case nakama.OpActionRejected:
		p.handleActionRejected(ctx, msg.Data)
	default:
		p.logger.Debug("收到未知消息", "op_code", msg.OpCode)
	}
}

func (p *AutoPlayPlayer) handleStateSync(ctx context.Context, data []byte) {
	var stateSync model.StateSync
	if err := json.Unmarshal(data, &stateSync); err != nil {
		p.logger.Error("解析 StateSync 失败", "error", err)
		return
	}

	p.mu.Lock()
	p.globalState = stateSync.GlobalState
	p.mu.Unlock()

	p.logger.Info("收到状态同步",
		"global", stateSync.GlobalState,
		"turn", stateSync.TurnState,
		"round", stateSync.Round,
		"players", len(stateSync.Players))
}

func (p *AutoPlayPlayer) handleAvailable(ctx context.Context, data []byte) {
	var available model.Available
	if err := json.Unmarshal(data, &available); err != nil {
		p.logger.Error("解析 Available 失败", "error", err)
		return
	}

	p.logger.Info("收到可用动作",
		"items", len(available.Items),
		"can_use_skill", available.CanUseSkill,
		"dice_type", available.DiceType)

	// Auto strategy: always roll dice
	p.logger.Info("自动策略：掷骰子")
	if err := p.socket.SendMessage(ctx, nakama.OpRollDice, model.RollDice{}); err != nil {
		p.logger.Error("发送 RollDice 失败", "error", err)
		p.mu.Lock()
		p.lastErr = err
		p.mu.Unlock()
	}
}

func (p *AutoPlayPlayer) handleDecisionRequest(ctx context.Context, data []byte) {
	var decision model.Decision
	if err := json.Unmarshal(data, &decision); err != nil {
		p.logger.Error("解析 DecisionRequest 失败", "error", err)
		return
	}

	p.logger.Info("收到决策请求",
		"id", decision.ID,
		"prompt", decision.Prompt,
		"options", len(decision.Options))

	// Auto strategy: always choose first option (index 0)
	p.mu.Lock()
	p.currentDecision = &decision
	p.mu.Unlock()

	p.logger.Info("自动策略：选择选项 0")
	userChoice := model.UserChoice{
		DecisionID: decision.ID,
		Choice:     0,
	}

	if err := p.socket.SendMessage(ctx, nakama.OpUserChoice, userChoice); err != nil {
		p.logger.Error("发送 UserChoice 失败", "error", err)
		p.mu.Lock()
		p.lastErr = err
		p.mu.Unlock()
	}
}

func (p *AutoPlayPlayer) handleMiniGameStart(ctx context.Context, data []byte) {
	var start model.MiniGameStart
	if err := json.Unmarshal(data, &start); err != nil {
		p.logger.Error("解析 MiniGameStart 失败", "error", err)
		return
	}

	p.logger.Info("收到小游戏开始", "game_type", start.GameType, "players", len(start.Players))

	// 策略：根据玩家索引分配唯一排名，确保排名连续且不重复
	// 例如：4 个玩家时，排名分别为 1,2,3,4
	maxRank := len(start.Players)
	if maxRank <= 0 {
		maxRank = 2
	}

	// 查找当前玩家在 players 列表中的索引
	playerIndex := 0
	for i, playerID := range start.Players {
		if playerID == p.userID {
			playerIndex = i
			break
		}
	}

	// 使用索引 +1 作为排名（索引 0 对应排名 1）
	// 为了增加随机性，可以预先设置随机种子
	rand.Seed(time.Now().UnixNano() + int64(playerIndex*1000))
	ranks := rand.Perm(maxRank)
	myRank := ranks[playerIndex] + 1

	submit := model.MiniGameResultSubmit{Rank: myRank}
	if err := p.socket.SendMessage(ctx, nakama.OpMiniGameResultSubmit, submit); err != nil {
		p.logger.Error("发送 MiniGameResultSubmit 失败", "error", err)
		p.mu.Lock()
		p.lastErr = err
		p.mu.Unlock()
		return
	}

	p.logger.Info("已提交小游戏排名", "rank", submit.Rank, "player_index", playerIndex)
}

func (p *AutoPlayPlayer) handleGameOver(ctx context.Context, data []byte) {
	var gameOver model.GameOver
	if err := json.Unmarshal(data, &gameOver); err != nil {
		p.logger.Error("解析 GameOver 失败", "error", err)
		return
	}

	p.logger.Info("游戏结束", "winner", gameOver.WinnerID)

	// Print stats
	for _, stat := range gameOver.Stats {
		p.logger.Info("玩家统计",
			"user_id", stat.UserID,
			"rounds_won", stat.RoundsWon,
			"events_drawn", stat.EventsDrawn,
			"items_used", stat.ItemsUsed)
	}

	// Signal game over
	select {
	case p.gameOverChan <- struct{}{}:
	default:
	}
}

func (p *AutoPlayPlayer) handleActionRejected(ctx context.Context, data []byte) {
	var rejected model.ActionRejected
	if err := json.Unmarshal(data, &rejected); err != nil {
		p.logger.Error("解析 ActionRejected 失败", "error", err)
		return
	}

	p.logger.Warn("动作被拒绝",
		"op_code", rejected.OpCode,
		"reason", rejected.Reason,
		"message", rejected.Message)
}

// GameOverChan returns the channel that receives when game is over.
func (p *AutoPlayPlayer) GameOverChan() <-chan struct{} {
	return p.gameOverChan
}

// MessagesReceived returns the number of messages received.
func (p *AutoPlayPlayer) MessagesReceived() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.messagesReceived
}

// TurnsCompleted returns the number of turns completed.
func (p *AutoPlayPlayer) TurnsCompleted() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.turnsCompleted
}

// GlobalState returns the current global state.
func (p *AutoPlayPlayer) GlobalState() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.globalState
}

// LastError returns the last error encountered.
func (p *AutoPlayPlayer) LastError() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastErr
}
