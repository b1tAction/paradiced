// Package scenario provides automated play scenarios for testing.
package scenario

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/b1tAction/paradiced/internal/cli/model"
	"github.com/b1tAction/paradiced/internal/cli/nakama"
)

// SocketClientAdapter is an interface for socket clients.
type SocketClientAdapter interface {
	MessageChan() <-chan *nakama.SocketMessage
	SendMessage(ctx context.Context, opCode int64, data any) error
	Close() error
}

// ScenarioConfig holds configuration for a play scenario.
type ScenarioConfig struct {
	PlayersCount int
	MatchName    string
	MaxTurns     int
	TimeoutSec   int
	Mode         string // "nakama" or "standalone"
}

// Result holds the result of a scenario run.
type Result struct {
	Success          bool          `json:"success"`
	FailureReason    string        `json:"failure_reason,omitempty"`
	Duration         time.Duration `json:"duration"`
	MessagesReceived int           `json:"messages_received"`
	TurnsCompleted   int           `json:"turns_completed"`
	GlobalState      string        `json:"global_state"`
	Rejections       int           `json:"rejections"`
	LastError        string        `json:"last_error,omitempty"`
}

// RunAutoPlay runs an autoplay scenario with the given configuration.
// Supports both Nakama and standalone server modes.
func RunAutoPlay(ctx context.Context, client nakama.IClient, config ScenarioConfig, logger *nakama.Logger) (Result, error) {
	if config.Mode == "standalone" {
		return runStandalonePlay(ctx, client, config, logger)
	}
	return runNakamaPlay(ctx, client.(*nakama.Client), config, logger)
}

// runNakamaPlay runs autoplay using Nakama server.
func runNakamaPlay(ctx context.Context, client *nakama.Client, config ScenarioConfig, logger *nakama.Logger) (Result, error) {
	startTime := time.Now()

	result := Result{
		Success: false,
	}

	// Ensure minimum 2 players for matchmaker
	playersCount := config.PlayersCount
	if playersCount < 2 {
		playersCount = 2
		logger.Info("Player count less than 2, adjusted to 2 for testing")
	}

	logger.Info("Starting playtest", "players", playersCount, "max_turns", config.MaxTurns)

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
			return result, fmt.Errorf("failed to create socket client %d: %w", i+1, err)
		}

		// Authenticate
		session, err := client.Authenticate(ctx, playerID)
		if err != nil {
			return result, fmt.Errorf("failed to authenticate player %d: %w", i+1, err)
		}

		// Connect socket
		if err := socketClients[i].Connect(ctx, session); err != nil {
			return result, fmt.Errorf("failed to connect socket %d: %w", i+1, err)
		}

		// Create autoplay player with session UserID (UUID format) for turn matching
		players[i] = NewAutoPlayPlayer(socketClients[i], session.UserID, logger)

		// Set matchmaker handler BEFORE adding to matchmaker
		socketClients[i].SetMatchmakerMatchedHandler(func(ctx context.Context, msg *nakama.MatchmakerMatchedMsg) {
			// Matchmaker can return either a match_id or a token
			// If match_id is empty, use token to join the match
			matchID := msg.GetMatchId()
			token := msg.GetToken()

			if matchID != "" {
				logger.Info("Matchmaker matched, received match_id", "match_id", matchID)
				select {
				case matchChan <- matchID:
				default:
				}
			} else if token != "" {
				logger.Info("Matchmaker matched, received token", "token", token)
				select {
				case tokenChan <- token:
				default:
				}
			} else {
				logger.Warn("Matchmaker matched but no match_id or token returned")
			}
		})
	}

	logger.Info("All players connected, starting matchmaking...")

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
			return result, fmt.Errorf("failed to add player %d to matchmaker: %w", i+1, addErr)
		}
		logger.Info("Player added to matchmaker", "player", i+1, "ticket", tickets[i].Ticket)

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
			logger.Info("Received match ID", "match_id", matchID, "count", i+1)
		case token := <-tokenChan:
			tokens[token]++
			logger.Info("Received token", "token", token, "count", i+1)
		case <-matchTimeout:
			return result, fmt.Errorf("matchmaker timeout (received %d/%d responses)", len(matchIDs)+len(tokens), playersCount)
		}
	}

	// Get final match ID or join with token
	var finalMatchID string
	if len(matchIDs) > 0 {
		// Received match_id directly - still need to join the match
		for mid := range matchIDs {
			finalMatchID = mid
		}
		logger.Info("Joining match with match ID", "match_id", finalMatchID)
		// All players need to join the match
		for i := 0; i < playersCount; i++ {
			// Pass metadata with display_name for autoplay testing
			metadata := map[string]string{
				"display_name": fmt.Sprintf("player-%d", i+1),
			}
			err := socketClients[i].JoinMatch(ctx, finalMatchID, metadata)
			if err != nil {
				return result, fmt.Errorf("player %d failed to join match: %w", i+1, err)
			}
		}
	} else if len(tokens) > 0 {
		// Received tokens - all players should have the same token
		// Join match with token
		for token, count := range tokens {
			if count == playersCount {
				// All players have the same token, join the match
				logger.Info("All players have same token, joining match", "token", token)
				// All players need to join the match for authoritative match to start
				for i := 0; i < playersCount; i++ {
					// Pass metadata with display_name for autoplay testing
					metadata := map[string]string{
						"display_name": fmt.Sprintf("player-%d", i+1),
					}
					err := socketClients[i].JoinMatch(ctx, token, metadata)
					if err != nil {
						return result, fmt.Errorf("player %d failed to join match: %w", i+1, err)
					}
				}
				// The match_id will be set by JoinMatch (from last player)
				// For now, we'll use the token as the match ID placeholder
				finalMatchID = token
				break
			}
		}
		if finalMatchID == "" {
			return result, fmt.Errorf("player tokens inconsistent, cannot join match")
		}
	}

	if finalMatchID == "" {
		return result, fmt.Errorf("no match ID or token received")
	}

	// Set match ID for all socket clients (use the actual match ID from first join)
	for i := 0; i < playersCount; i++ {
		socketClients[i].SetMatchID(finalMatchID)
	}

	logger.Info("All players joined match", "match_id", finalMatchID, "players", playersCount)

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
			return result, fmt.Errorf("timeout (%.1f seconds)", timeout.Seconds())

		case <-mainPlayer.GameOverChan():
			result.Success = true
			result.Duration = time.Since(startTime)
			result.MessagesReceived = mainPlayer.MessagesReceived()
			result.TurnsCompleted = mainPlayer.TurnsCompleted()
			result.GlobalState = "game_over"
			result.Rejections = mainPlayer.Rejections()
			if rejection := mainPlayer.LastRejection(); rejection != nil {
				result.LastError = fmt.Sprintf("ActionRejected: op=%d, reason=%s", rejection.OpCode, rejection.Reason)
			}
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
	socket           SocketClientAdapter
	userID           string // Nakama UserID (UUID format) for identifying self in Players array
	playerID         string // Game PlayerID (UUID format), extracted from StateSync
	logger           *nakama.Logger
	msgChan          <-chan *nakama.SocketMessage
	gameOverChan     chan struct{}
	mu               sync.RWMutex
	messagesReceived int
	turnsCompleted   int
	globalState      string
	currentPlayerID  string // Current player ID from StateSync (PlayerID format)
	currentDecision  *model.Decision
	lastErr          error
	rejections       int
	lastRejection    *model.ActionRejected
}

// NewAutoPlayPlayer creates a new autoplay player.
// The playerID will be extracted from StateSync.Players array by matching UserID.
func NewAutoPlayPlayer(socket *nakama.SocketClient, userID string, logger *nakama.Logger) *AutoPlayPlayer {
	return &AutoPlayPlayer{
		socket:       socket,
		userID:       userID,
		logger:       logger,
		msgChan:      socket.MessageChan(),
		gameOverChan: make(chan struct{}, 1),
	}
}

// NewAutoPlayPlayerStandalone creates a new autoplay player for standalone mode.
func NewAutoPlayPlayerStandalone(socket SocketClientAdapter, userID string, logger *nakama.Logger) *AutoPlayPlayer {
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

// Play starts listening and handling messages for a player.
func (p *AutoPlayPlayer) Play(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	p.Listen(ctx)
}

// handleMessage processes an incoming message.
func (p *AutoPlayPlayer) handleMessage(ctx context.Context, msg *nakama.SocketMessage) {
	p.mu.Lock()
	p.messagesReceived++
	p.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// For standalone mode, OpCode may be 0 - parse from data
	opCode := msg.OpCode
	if opCode == 0 {
		// Try to parse op_code from JSON data
		var base struct {
			OpCode int64 `json:"op_code"`
		}
		if err := json.Unmarshal(msg.Data, &base); err == nil {
			opCode = base.OpCode
		}
	}

	switch opCode {
	case nakama.OpStateSync:
		p.handleStateSync(ctx, msg.Data)
	case nakama.OpAvailable:
		p.handleAvailable(ctx, msg.Data)
	case nakama.OpDecisionRequest:
		p.handleDecisionRequest(ctx, msg.Data)
	case nakama.OpMiniGameStart:
		p.handleMiniGameStart(ctx, msg.Data)
	case nakama.OpMiniGameResult:
		p.handleMiniGameResult(ctx, msg.Data)
	case nakama.OpFullSync:
		p.handleFullSync(ctx, msg.Data)
	case nakama.OpGameOver:
		p.handleGameOver(ctx, msg.Data)
	case nakama.OpTurnSync:
		p.mu.Lock()
		p.turnsCompleted++
		p.mu.Unlock()
		p.logger.Debug("Received TurnSync")
	case nakama.OpActionRejected:
		p.handleActionRejected(ctx, msg.Data)
	default:
		p.logger.Debug("Received unknown message", "op_code", opCode)
	}
}

func (p *AutoPlayPlayer) handleStateSync(ctx context.Context, data []byte) {
	var stateSync model.StateSync
	if err := json.Unmarshal(data, &stateSync); err != nil {
		p.logger.Error("Failed to parse StateSync", "error", err)
		return
	}

	p.mu.Lock()
	p.globalState = stateSync.GlobalState
	p.currentPlayerID = stateSync.CurrentPlayerID

	// Find own PlayerID from Players array by matching ClientID
	// ClientID is injected by NakamaBroadcastAdapter for client-side turn matching
	for _, player := range stateSync.Players {
		if player.ClientID == p.userID {
			// Found own player, store PlayerID for turn checking
			p.playerID = player.PlayerID
			p.logger.Info("Found own player", "player_id", p.playerID, "client_id", p.userID)
			break
		}
	}
	p.mu.Unlock()

	p.logger.Info("Received state sync",
		"global", stateSync.GlobalState,
		"turn", stateSync.TurnState,
		"round", stateSync.Round,
		"current_player_id", stateSync.CurrentPlayerID,
		"players", len(stateSync.Players),
		"my_player_id", p.playerID)
}

func (p *AutoPlayPlayer) handleAvailable(ctx context.Context, data []byte) {
	var available model.Available
	if err := json.Unmarshal(data, &available); err != nil {
		p.logger.Error("Failed to parse Available", "error", err)
		return
	}

	p.mu.RLock()
	currentPlayerID := p.currentPlayerID
	myPlayerID := p.playerID
	p.mu.RUnlock()

	// Available is a server-targeted prompt for the current player.
	// Do not perform local turn gating here, to avoid race-induced false negatives.
	if currentPlayerID != "" && myPlayerID != "" && currentPlayerID != myPlayerID {
		p.logger.Debug("Turn mismatch on Available, still proceeding",
			"my_player_id", myPlayerID,
			"current_player_id", currentPlayerID)
	}

	p.logger.Info("Received available actions",
		"items", len(available.Items),
		"can_use_skill", available.CanUseSkill,
		"dice_type", available.DiceType)

	// Auto strategy:
	// 1. If has items, use first item (for testing item system)
	// 2. If can use skill, use skill (for testing faction skill system)
	// 3. Otherwise, roll dice

	if len(available.Items) > 0 {
		// Use first item
		item := available.Items[0]
		p.logger.Info("Auto strategy: use item", "item_id", item.ID, "item_type", item.Type)
		useItem := model.UseItem{
			ItemID:   item.ID,
			TargetID: "", // No target for now
		}
		if err := p.socket.SendMessage(ctx, nakama.OpUseItem, useItem); err != nil {
			p.logger.Error("Failed to send UseItem", "error", err)
			p.mu.Lock()
			p.lastErr = err
			p.mu.Unlock()
		}
		return
	}

	if available.CanUseSkill {
		// Use faction skill
		p.logger.Info("Auto strategy: use faction skill")
		if err := p.socket.SendMessage(ctx, nakama.OpUseSkill, model.UseSkill{}); err != nil {
			p.logger.Error("Failed to send UseSkill", "error", err)
			p.mu.Lock()
			p.lastErr = err
			p.mu.Unlock()
		}
		return
	}

	// Default: roll dice
	p.logger.Info("Auto strategy: roll dice")
	p.logger.Debug("Sending RollDice message", "ctx", ctx)
	if err := p.socket.SendMessage(ctx, nakama.OpRollDice, model.RollDice{}); err != nil {
		p.logger.Error("Failed to send RollDice", "error", err)
		p.mu.Lock()
		p.lastErr = err
		p.mu.Unlock()
	} else {
		p.logger.Info("RollDice message sent successfully")
	}
}

func (p *AutoPlayPlayer) handleDecisionRequest(ctx context.Context, data []byte) {
	var decision model.Decision
	if err := json.Unmarshal(data, &decision); err != nil {
		p.logger.Error("Failed to parse DecisionRequest", "error", err)
		return
	}

	p.logger.Info("Received decision request",
		"id", decision.ID,
		"prompt", decision.Prompt,
		"options", len(decision.Options))

	// Auto strategy: always choose first option (index 0)
	p.mu.Lock()
	p.currentDecision = &decision
	p.mu.Unlock()

	p.logger.Info("Auto strategy: choose option 0")
	userChoice := model.UserChoice{
		DecisionID: decision.ID,
		Choice:     0,
	}

	if err := p.socket.SendMessage(ctx, nakama.OpUserChoice, userChoice); err != nil {
		p.logger.Error("Failed to send UserChoice", "error", err)
		p.mu.Lock()
		p.lastErr = err
		p.mu.Unlock()
	}
}

func (p *AutoPlayPlayer) handleMiniGameStart(ctx context.Context, data []byte) {
	var start model.MiniGameStart
	if err := json.Unmarshal(data, &start); err != nil {
		p.logger.Error("Failed to parse MiniGameStart", "error", err)
		return
	}

	p.logger.Info("Received mini-game start", "game_type", start.GameType, "players", len(start.Players))

	// Strategy: Assign unique rank based on player index to ensure consecutive and non-duplicate rankings
	// Example: For 4 players, rankings are 1, 2, 3, 4
	maxRank := len(start.Players)
	if maxRank <= 0 {
		maxRank = 2
	}

	// Find current player's index in the players list using userID
	// MiniGameStart.Players contains Nakama UserIDs
	playerIndex := 0
	for i, uid := range start.Players {
		if uid == p.userID {
			playerIndex = i
			break
		}
	}

	// Simple strategy: use index + 1 as rank (index 0 -> rank 1, index 1 -> rank 2, etc.)
	// This ensures all players submit different ranks
	myRank := playerIndex + 1
	if myRank > maxRank {
		myRank = 1 // Wrap around if index exceeds player count
	}

	submit := model.MiniGameResultSubmit{Rank: myRank}
	if err := p.socket.SendMessage(ctx, nakama.OpMiniGameResultSubmit, submit); err != nil {
		p.logger.Error("Failed to send MiniGameResultSubmit", "error", err)
		p.mu.Lock()
		p.lastErr = err
		p.mu.Unlock()
		return
	}

	p.logger.Info("Submitted mini-game rank", "rank", submit.Rank, "player_index", playerIndex)
}

func (p *AutoPlayPlayer) handleGameOver(ctx context.Context, data []byte) {
	var gameOver model.GameOver
	if err := json.Unmarshal(data, &gameOver); err != nil {
		p.logger.Error("Failed to parse GameOver", "error", err)
		return
	}

	p.logger.Info("Game over", "winner", gameOver.WinnerID)

	// Print stats
	for _, stat := range gameOver.Stats {
		p.logger.Info("Player stats",
			"player_id", stat.PlayerID,
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

func (p *AutoPlayPlayer) handleMiniGameResult(ctx context.Context, data []byte) {
	var result model.MiniGameResult
	if err := json.Unmarshal(data, &result); err != nil {
		p.logger.Error("Failed to parse MiniGameResult", "error", err)
		return
	}

	p.logger.Info("Received mini-game result")
	for _, entry := range result.Rankings {
		// Rankings use PlayerID (game internal ID), need to match with playerID
		if entry.PlayerID == p.playerID {
			p.logger.Info("My mini-game rank", "rank", entry.Rank)
		}
	}
}

func (p *AutoPlayPlayer) handleFullSync(ctx context.Context, data []byte) {
	var stateSync model.StateSync
	if err := json.Unmarshal(data, &stateSync); err != nil {
		p.logger.Error("Failed to parse FullSync", "error", err)
		return
	}

	p.mu.Lock()
	p.globalState = stateSync.GlobalState
	p.mu.Unlock()

	p.logger.Info("Received full sync (reconnection)",
		"global", stateSync.GlobalState,
		"turn", stateSync.TurnState,
		"round", stateSync.Round,
		"players", len(stateSync.Players))
}

func (p *AutoPlayPlayer) handleActionRejected(ctx context.Context, data []byte) {
	var rejected model.ActionRejected
	if err := json.Unmarshal(data, &rejected); err != nil {
		p.logger.Error("Failed to parse ActionRejected", "error", err)
		return
	}

	// Track rejection statistics
	p.mu.Lock()
	p.rejections++
	p.lastRejection = &rejected
	p.mu.Unlock()

	p.logger.Warn("Action rejected",
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

// Rejections returns the number of action rejections.
func (p *AutoPlayPlayer) Rejections() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.rejections
}

// LastRejection returns the last action rejection details.
func (p *AutoPlayPlayer) LastRejection() *model.ActionRejected {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastRejection
}

// runStandalonePlay runs autoplay using standalone WebSocket server.
func runStandalonePlay(ctx context.Context, client nakama.IClient, config ScenarioConfig, logger *nakama.Logger) (Result, error) {
	startTime := time.Now()

	result := Result{
		Success: false,
	}

	playersCount := config.PlayersCount
	if playersCount < 2 {
		playersCount = 2
		logger.Info("Player count less than 2, adjusted to 2 for testing")
	}

	logger.Info("Starting standalone server test", "players", playersCount, "max_turns", config.MaxTurns)

	// Create socket clients for each player
	socketClients := make([]nakama.ISocketClient, playersCount)
	players := make([]*AutoPlayPlayer, playersCount)

	// Cleanup on exit
	defer func() {
		for _, sc := range socketClients {
			if sc != nil {
				sc.Close()
			}
		}
	}()

	// Create and authenticate all players
	for i := 0; i < playersCount; i++ {
		var err error
		playerID := fmt.Sprintf("cli_bot_%02d", i+1)

		// Authenticate
		session, err := client.Authenticate(ctx, playerID)
		if err != nil {
			return result, fmt.Errorf("failed to authenticate player %d: %w", i+1, err)
		}

		// Create socket client
		socketClients[i], err = client.CreateSocketClient()
		if err != nil {
			return result, fmt.Errorf("failed to create socket client %d: %w", i+1, err)
		}

		// Connect socket
		if err := socketClients[i].Connect(ctx, session); err != nil {
			return result, fmt.Errorf("failed to connect socket %d: %w", i+1, err)
		}

		// Create autoplay player with playerID as userID for standalone mode
		players[i] = NewAutoPlayPlayerStandalone(socketClients[i], playerID, logger)
	}

	logger.Info("All players connected, waiting for game to start...")

	// Wait a bit for all players to connect and game to initialize
	time.Sleep(2 * time.Second)

	// Start autoplay for all players
	var wg sync.WaitGroup
	for i, player := range players {
		wg.Add(1)
		go func(idx int, p *AutoPlayPlayer) {
			defer wg.Done()
			p.Play(ctx, &wg)
		}(i, player)
	}

	// Wait for completion or timeout
	timeout := time.After(time.Duration(config.TimeoutSec) * time.Second)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All players finished
		result.Success = true
	case <-timeout:
		// Timeout - check if at least some progress was made
		totalMsgs := 0
		for _, p := range players {
			totalMsgs += p.MessagesReceived()
		}
		if totalMsgs > 0 {
			result.Success = true // Partial success
		}
		result.FailureReason = fmt.Sprintf("timeout (%.1f seconds)", float64(config.TimeoutSec))
	}

	// Aggregate results
	result.Duration = time.Since(startTime)
	result.MessagesReceived = 0
	result.TurnsCompleted = 0
	for _, p := range players {
		result.MessagesReceived += p.MessagesReceived()
		result.TurnsCompleted = max(result.TurnsCompleted, p.TurnsCompleted())
		result.GlobalState = p.GlobalState()
	}

	return result, nil
}
