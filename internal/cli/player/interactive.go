// Package player provides interactive player functionality for CLI.
package player

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
// Same as scenario.SocketClientAdapter but defined here for independence.
type SocketClientAdapter interface {
	MessageChan() <-chan *nakama.SocketMessage
	SendMessage(ctx context.Context, opCode int64, data any) error
	Close() error
}

// InteractivePlayer represents a human-controlled player.
type InteractivePlayer struct {
	socket       SocketClientAdapter
	userID       string // Nakama UserID (UUID format)
	playerID     string // Game PlayerID, extracted from StateSync
	logger       *nakama.Logger
	msgChan      <-chan *nakama.SocketMessage
	gameOverChan chan struct{}

	mu               sync.RWMutex
	messagesReceived int
	globalState      string
	currentPlayerID  string
	stateSync        *model.StateSync // current game state for viewing

	uiAdapter PlayerUIAdapter // UI interface
}

// NewInteractivePlayer creates a new interactive player.
func NewInteractivePlayer(socket SocketClientAdapter, userID string, uiAdapter PlayerUIAdapter, logger *nakama.Logger) *InteractivePlayer {
	return &InteractivePlayer{
		socket:       socket,
		userID:       userID,
		logger:       logger,
		msgChan:      socket.MessageChan(),
		gameOverChan: make(chan struct{}, 1),
		uiAdapter:    uiAdapter,
	}
}

// GameOverChan returns the channel that receives when game is over.
func (p *InteractivePlayer) GameOverChan() <-chan struct{} {
	return p.gameOverChan
}

// MessagesReceived returns the number of messages received.
func (p *InteractivePlayer) MessagesReceived() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.messagesReceived
}

// GlobalState returns the current global state.
func (p *InteractivePlayer) GlobalState() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.globalState
}

// GetCurrentState returns the current game state.
func (p *InteractivePlayer) GetCurrentState() *model.StateSync {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stateSync
}

// GetMyPlayerID returns the current player's game ID.
func (p *InteractivePlayer) GetMyPlayerID() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.playerID
}

// GetUserID returns the Nakama user ID.
func (p *InteractivePlayer) GetUserID() string {
	return p.userID
}

// Listen starts listening for messages and handling them.
func (p *InteractivePlayer) Listen(ctx context.Context) {
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

// Play starts listening and handling messages.
func (p *InteractivePlayer) Play(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	p.Listen(ctx)
}

// handleMessage processes an incoming message.
func (p *InteractivePlayer) handleMessage(ctx context.Context, msg *nakama.SocketMessage) {
	p.mu.Lock()
	p.messagesReceived++
	p.mu.Unlock()

	// Parse opCode if not set (standalone mode)
	opCode := msg.OpCode
	if opCode == 0 {
		var base struct {
			OpCode int64 `json:"op_code"`
		}
		if err := json.Unmarshal(msg.Data, &base); err == nil {
			opCode = base.OpCode
		}
	}

	// Handlers that require human input should not use a short timeout.
	// Otherwise input wait may exceed timeout and fail sending requests.
	handlerCtx := ctx
	cancel := func() {}
	if !requiresUserInput(opCode) {
		handlerCtx, cancel = context.WithTimeout(ctx, 30*time.Second)
	}
	defer cancel()

	switch opCode {
	case nakama.OpStateSync:
		p.handleStateSync(handlerCtx, msg.Data)
	case nakama.OpAvailable:
		p.handleAvailable(handlerCtx, msg.Data)
	case nakama.OpMiniGameStart:
		p.handleMiniGameStart(handlerCtx, msg.Data)
	case nakama.OpMiniGameResult:
		p.handleMiniGameResult(handlerCtx, msg.Data)
	case nakama.OpFullSync:
		p.handleFullSync(handlerCtx, msg.Data)
	case nakama.OpGameOver:
		p.handleGameOver(handlerCtx, msg.Data)
	case nakama.OpActionRejected:
		p.handleActionRejected(handlerCtx, msg.Data)
	case nakama.OpWaitingSync:
		p.handleWaitingSync(handlerCtx, msg.Data)
	case nakama.OpStartGameAck:
		p.handleStartGameAck(handlerCtx, msg.Data)
	default:
		p.logger.Debug("Received unknown message", "op_code", opCode)
	}
}

func requiresUserInput(opCode int64) bool {
	switch opCode {
	case nakama.OpAvailable, nakama.OpMiniGameStart, nakama.OpWaitingSync:
		return true
	default:
		return false
	}
}

func (p *InteractivePlayer) handleStateSync(ctx context.Context, data []byte) {
	var stateSync model.StateSync
	if err := json.Unmarshal(data, &stateSync); err != nil {
		p.logger.Error("Failed to parse StateSync", "error", err)
		p.uiAdapter.OnError(err)
		return
	}

	p.mu.Lock()
	p.globalState = stateSync.GlobalState
	p.currentPlayerID = stateSync.CurrentPlayerID
	p.stateSync = &stateSync

	// Find own PlayerID from Players array
	// PlayerID now equals the frontend userID, so we can directly match
	for _, player := range stateSync.Players {
		// Skip Boss player - it's not a human player
		if player.IsBoss {
			continue
		}
		if player.PlayerID == p.userID {
			p.playerID = player.PlayerID
			p.logger.Debug("Found own player", "player_id", p.playerID, "user_id", p.userID)
			break
		}
	}
	p.mu.Unlock()

	// When in RoundEndWait state, auto-send RoundReady signal
	// (client would send this after finishing rendering; CLI sends immediately)
	if stateSync.GlobalState == "RoundEndWait" {
		p.logger.Info("Sending RoundReady (round end wait)")
		if err := p.socket.SendMessage(ctx, nakama.OpRoundReady, model.RoundReady{}); err != nil {
			p.logger.Error("Failed to send RoundReady", "error", err)
			p.uiAdapter.OnError(err)
		}
	}

	// Notify UI
	p.uiAdapter.OnStateSync(ctx, &stateSync)

	p.logger.Debug("Received state sync",
		"global", stateSync.GlobalState,
		"turn", stateSync.TurnState,
		"round", stateSync.Round,
		"current_player_id", stateSync.CurrentPlayerID,
		"players", len(stateSync.Players),
		"entries", len(stateSync.Entries))
}

func (p *InteractivePlayer) handleAvailable(ctx context.Context, data []byte) {
	var available model.Available
	if err := json.Unmarshal(data, &available); err != nil {
		p.logger.Error("Failed to parse Available", "error", err)
		p.uiAdapter.OnError(err)
		return
	}

	p.logger.Debug("Received available actions",
		"items", len(available.Items),
		"can_use_skill", available.CanUseSkill,
		"dice_type", available.DiceType)

	// Get user action from UI
	action := p.uiAdapter.OnAvailable(ctx, &available)

	// Handle the action
	for {
		if !action.IsValid() {
			p.logger.Warn("Invalid action received")
			action = p.uiAdapter.OnAvailable(ctx, &available)
			continue
		}

		// If view status, display and re-prompt
		if action.Type == ActionViewStatus {
			p.displayDetailedStatus()
			action = p.uiAdapter.OnAvailable(ctx, &available)
			continue
		}

		// If server action, send it
		if action.IsServerAction() {
			p.sendAction(ctx, action)
			return
		}

		// Unknown action type, re-prompt
		action = p.uiAdapter.OnAvailable(ctx, &available)
	}
}

func (p *InteractivePlayer) handleMiniGameStart(ctx context.Context, data []byte) {
	var start model.MiniGameStart
	if err := json.Unmarshal(data, &start); err != nil {
		p.logger.Error("Failed to parse MiniGameStart", "error", err)
		p.uiAdapter.OnError(err)
		return
	}

	// Entering mini-game phase: guard against any stale WaitingSync prompts.
	p.mu.Lock()
	p.globalState = "RoundMiniGame"
	p.mu.Unlock()

	p.logger.Debug("Received mini-game start", "game_type", start.GameType, "players", len(start.Players))

	// Get game_data from user (interactive mode lets user choose score/time)
	gameData := p.uiAdapter.OnMiniGameStart(ctx, &start)

	// Submit game_data
	submit := model.MiniGameDataSubmit{
		GameType: start.GameType,
		GameData: gameData,
	}
	if err := p.socket.SendMessage(ctx, nakama.OpMiniGameDataSubmit, submit); err != nil {
		p.logger.Error("Failed to send MiniGameDataSubmit", "error", err)
		p.uiAdapter.OnError(err)
		return
	}

	p.logger.Debug("Submitted mini-game data", "game_type", submit.GameType)
}

func (p *InteractivePlayer) handleMiniGameResult(ctx context.Context, data []byte) {
	var result model.MiniGameResult
	if err := json.Unmarshal(data, &result); err != nil {
		p.logger.Error("Failed to parse MiniGameResult", "error", err)
		p.uiAdapter.OnError(err)
		return
	}

	p.mu.Lock()
	// Mini-game result received: local phase is complete; next sync will enter turn flow.
	if p.globalState == "RoundMiniGame" {
		p.globalState = "RoundPrep"
	}
	p.mu.Unlock()

	p.uiAdapter.OnMiniGameResult(ctx, &result)

	p.logger.Debug("Received mini-game result")
}

func (p *InteractivePlayer) handleGameOver(ctx context.Context, data []byte) {
	var gameOver model.GameOver
	if err := json.Unmarshal(data, &gameOver); err != nil {
		p.logger.Error("Failed to parse GameOver", "error", err)
		p.uiAdapter.OnError(err)
		return
	}

	p.uiAdapter.OnGameOver(ctx, &gameOver)

	p.logger.Debug("Game over", "winner", gameOver.WinnerID)

	// Signal game over
	select {
	case p.gameOverChan <- struct{}{}:
	default:
	}
}

func (p *InteractivePlayer) handleFullSync(ctx context.Context, data []byte) {
	var stateSync model.StateSync
	if err := json.Unmarshal(data, &stateSync); err != nil {
		p.logger.Error("Failed to parse FullSync", "error", err)
		p.uiAdapter.OnError(err)
		return
	}

	p.mu.Lock()
	p.globalState = stateSync.GlobalState
	p.stateSync = &stateSync
	p.mu.Unlock()

	// When in RoundEndWait state, auto-send RoundReady signal (same as handleStateSync)
	if stateSync.GlobalState == "RoundEndWait" {
		p.logger.Info("Sending RoundReady (round end wait, from FullSync)")
		if err := p.socket.SendMessage(ctx, nakama.OpRoundReady, model.RoundReady{}); err != nil {
			p.logger.Error("Failed to send RoundReady", "error", err)
			p.uiAdapter.OnError(err)
		}
	}

	p.uiAdapter.OnFullSync(ctx, &stateSync)

	p.logger.Debug("Received full sync (reconnection)",
		"global", stateSync.GlobalState,
		"players", len(stateSync.Players),
		"entries", len(stateSync.Entries))
}


func (p *InteractivePlayer) handleActionRejected(ctx context.Context, data []byte) {
	var rejected model.ActionRejected
	if err := json.Unmarshal(data, &rejected); err != nil {
		p.logger.Error("Failed to parse ActionRejected", "error", err)
		p.uiAdapter.OnError(err)
		return
	}

	p.uiAdapter.OnActionRejected(ctx, &rejected)

	p.logger.Warn("Action rejected",
		"op_code", rejected.OpCode,
		"reason", rejected.Reason,
		"message", rejected.Message)
}

func (p *InteractivePlayer) handleWaitingSync(ctx context.Context, data []byte) {
	var waiting model.WaitingSync
	if err := json.Unmarshal(data, &waiting); err != nil {
		p.logger.Error("Failed to parse WaitingSync", "error", err)
		p.uiAdapter.OnError(err)
		return
	}

	// Ignore waiting-room prompts once we have left waiting states.
	p.mu.RLock()
	state := p.globalState
	p.mu.RUnlock()
	if state != "" && state != "MatchInit" && state != "WaitingForHost" {
		p.logger.Debug("Ignoring waiting sync outside waiting states",
			"global_state", state,
			"player_count", waiting.PlayerCount,
			"can_start", waiting.CanStart)
		return
	}

	p.logger.Debug("Received waiting sync",
		"match_id", waiting.MatchID,
		"host_user_id", waiting.HostUserID,
		"player_count", waiting.PlayerCount,
		"can_start", waiting.CanStart)

	// Get user action from UI
	if p.uiAdapter.OnWaitingSync(ctx, &waiting) {
		// User wants to start the game
		p.sendStartGame(ctx)
	}
}

func (p *InteractivePlayer) handleStartGameAck(ctx context.Context, data []byte) {
	var ack model.StartGameAck
	if err := json.Unmarshal(data, &ack); err != nil {
		p.logger.Error("Failed to parse StartGameAck", "error", err)
		p.uiAdapter.OnError(err)
		return
	}

	p.logger.Debug("Received start game ack",
		"map_length", ack.MapConfig.Length,
		"cells", len(ack.MapConfig.Cells))

	p.uiAdapter.OnStartGameAck(ctx, &ack)
}

// ========== Action Sending ==========

func (p *InteractivePlayer) sendAction(ctx context.Context, action PlayerAction) {
	switch action.Type {
	case ActionRollDice:
		p.sendRollDice(ctx)
	case ActionUseItem:
		p.sendUseItem(ctx, action)
	default:
		p.logger.Warn("Unknown action type", "type", action.Type)
	}
}

func (p *InteractivePlayer) sendRollDice(ctx context.Context) {
	p.logger.Debug("Sending RollDice")

	if err := p.socket.SendMessage(ctx, nakama.OpRollDice, model.RollDice{}); err != nil {
		p.logger.Error("Failed to send RollDice", "error", err)
		p.uiAdapter.OnError(err)
	}
}

func (p *InteractivePlayer) sendUseItem(ctx context.Context, action PlayerAction) {
	p.logger.Debug("Sending UseItem", "item_id", action.ItemID, "target_id", action.TargetID)

	useItem := model.UseItem{
		ItemID:   action.ItemID,
		TargetID: action.TargetID,
	}

	if err := p.socket.SendMessage(ctx, nakama.OpUseItem, useItem); err != nil {
		p.logger.Error("Failed to send UseItem", "error", err)
		p.uiAdapter.OnError(err)
	}
}

func (p *InteractivePlayer) sendStartGame(ctx context.Context) {
	p.logger.Debug("Sending StartGame")

	// StartGame has empty body - just the opcode matters
	if err := p.socket.SendMessage(ctx, nakama.OpStartGame, nil); err != nil {
		p.logger.Error("Failed to send StartGame", "error", err)
		p.uiAdapter.OnError(err)
	}
}

// RequestStartGame sends host start-game request.
func (p *InteractivePlayer) RequestStartGame(ctx context.Context) {
	p.sendStartGame(ctx)
}

// ========== Helper Methods ==========

// DisplayDetailedStatus displays detailed game status (public method for commands).
func (p *InteractivePlayer) DisplayDetailedStatus() {
	p.displayDetailedStatus()
}

func (p *InteractivePlayer) displayDetailedStatus() {
	p.mu.RLock()
	state := p.stateSync
	p.mu.RUnlock()

	if state == nil {
		fmt.Println("No game state received yet")
		return
	}

	fmt.Println()
	fmt.Println("========== Detailed Status ==========")
	fmt.Printf("Global State: %s\n", state.GlobalState)
	fmt.Printf("Turn State: %s\n", state.TurnState)
	fmt.Printf("Round: %d | Turn: %d\n", state.Round, state.Turn)
	fmt.Printf("Paused: %v\n", state.Paused)
	fmt.Println()

	fmt.Println("My Status:")
	for _, player := range state.Players {
		if player.PlayerID == p.userID {
			fmt.Printf("  PlayerID: %s\n", player.PlayerID)
			fmt.Printf("  Faction: %s\n", player.Faction)
			fmt.Printf("  Position: %d\n", player.Position)
			fmt.Printf("  HP: %d | LP: %d\n", player.HP, player.LP)
			fmt.Printf("  Charge: %d\n", player.Charge)
			fmt.Printf("  Dead: %v | SkipTurn: %v\n", player.IsDead, player.SkipTurn)

			if len(player.Buffs) > 0 {
				fmt.Println("  Buffs:")
				for _, buff := range player.Buffs {
					fmt.Printf("	- %s (%s): %d turns remaining\n", buff.Name, buff.Type, buff.Duration)
				}
			}

			if len(player.Items) > 0 {
				fmt.Println("  Items:")
				for _, item := range player.Items {
					fmt.Printf("	- %s (%s): ID=%s\n", item.Name, item.Type, item.ID)
				}
			}
			break
		}
	}

	fmt.Println()
	fmt.Println("Other Players:")
	for _, player := range state.Players {
		if player.PlayerID != p.userID {
			isCurrent := player.PlayerID == state.CurrentPlayerID
			currentMark := ""
			if isCurrent {
				currentMark = " [Current Turn]"
			}
			fmt.Printf("  %s (%s)%s: Pos=%d, HP=%d, LP=%d\n",
				player.PlayerID, player.Faction, currentMark, player.Position, player.HP, player.LP)
		}
	}

	fmt.Println("====================================")
	fmt.Println()
}
