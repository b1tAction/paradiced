// Package nakama provides Nakama match handler implementation for Paradiced.
// This package implements the authoritative server logic using Nakama SDK.
package nakama

import (
	"math/rand"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/engine"
	"github.com/b1tAction/paradiced/internal/engine/hsm"
	"github.com/b1tAction/paradiced/internal/gamemap"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
	pkgnet "github.com/b1tAction/paradiced/pkg/net"
	"github.com/b1tAction/paradiced/pkg/rng"
	"github.com/heroiclabs/nakama-common/runtime"
)

// NakamaMatchHandler implements the authoritative match handler for Paradiced.
// It manages game state, HSM execution, and client communication.
// HSM is the single source of truth - Game is accessed via hsm.GetGame().
type NakamaMatchHandler struct {
	// Core game components
	hsm       *hsm.HSM          // Hierarchical State Machine (holds Game reference)
	mapEngine *gamemap.MapEngine // Map engine for path calculation
	diceMgr   *rng.DiceManager  // Dice manager for rolling

	// Message dispatcher
	dispatcher DispatcherAdapter // Dispatcher for sending messages to clients

	// Logger for debug/info/error output
	logger runtime.Logger // Nakama runtime logger

	// Match identification
	matchID string // Nakama match ID

	// Player tracking
	players         map[string]*core.Player // userID -> Player mapping
	playerList      []string                // Ordered player list for turn sequence
	disconnected    map[string]bool         // userID -> disconnected status

	// Configuration
	maxPlayers  int    // Maximum players (default: 4)
	mapLength   int    // Map length (default: 100)
	randomSeed  int64  // Random seed for reproducibility

	// Decision tracking (prevent duplicate sends)
	lastDecisionID string // Last sent decision ID to prevent duplicate sends
}

// NewNakamaMatchHandler creates a new match handler with configuration.
func NewNakamaMatchHandler(matchID string, seed int64, maxPlayers int, mapLength int) *NakamaMatchHandler {
	if maxPlayers <= 0 {
		maxPlayers = 4 // Default 4 players
	}
	if mapLength <= 0 {
		mapLength = 100 // Default 100 cells
	}

	// Create RNG for dice manager
	var rngInst *rand.Rand
	if seed != 0 {
		rngInst = rand.New(rand.NewSource(seed))
	} else {
		rngInst = rand.New(rand.NewSource(0)) // Will be replaced by game's RNG
	}

	return &NakamaMatchHandler{
		matchID:     matchID,
		maxPlayers:  maxPlayers,
		mapLength:   mapLength,
		randomSeed:  seed,
		players:     make(map[string]*core.Player),
		playerList:  make([]string, 0),
		disconnected: make(map[string]bool),
		diceMgr:     rng.NewDiceManager(rngInst),
		dispatcher:  nil, // Set via WithDispatcher or during MatchInit
	}
}

// WithDispatcher sets the dispatcher for message sending.
func (h *NakamaMatchHandler) WithDispatcher(dispatcher DispatcherAdapter) *NakamaMatchHandler {
	h.dispatcher = dispatcher
	return h
}

// GetDispatcher returns the current dispatcher.
func (h *NakamaMatchHandler) GetDispatcher() DispatcherAdapter {
	return h.dispatcher
}

// WithLogger sets the logger for debug output.
func (h *NakamaMatchHandler) WithLogger(logger runtime.Logger) *NakamaMatchHandler {
	h.logger = logger
	return h
}

// logDebug logs a debug message if logger is available.
func (h *NakamaMatchHandler) logDebug(msg string, keysAndValues ...interface{}) {
	if h.logger != nil {
		h.logger.Debug(msg, keysAndValues...)
	}
}

// logInfo logs an info message if logger is available.
func (h *NakamaMatchHandler) logInfo(msg string, keysAndValues ...interface{}) {
	if h.logger != nil {
		h.logger.Info(msg, keysAndValues...)
	}
}

// logWarn logs a warn message if logger is available.
func (h *NakamaMatchHandler) logWarn(msg string, keysAndValues ...interface{}) {
	if h.logger != nil {
		h.logger.Warn(msg, keysAndValues...)
	}
}

// logError logs an error message if logger is available.
func (h *NakamaMatchHandler) logError(msg string, keysAndValues ...interface{}) {
	if h.logger != nil {
		h.logger.Error(msg, keysAndValues...)
	}
}

// initializeGame creates the Game instance and HSM.
// Called during MatchInit after all players have joined.
func (h *NakamaMatchHandler) initializeGame() error {
	// Create Game instance
	gameID := id.NewGameID()
	game := engine.NewGame(gameID, h.randomSeed)

	// Create HSM with game reference (HSM is single source of truth)
	h.hsm = hsm.NewHSM(game)

	// Create MapEngine
	h.mapEngine = gamemap.NewMapEngine(h.mapLength)
	h.mapEngine.GenerateLinearMap(nil) // Default map with checkpoints

	// Register all states
	if err := hsm.RegisterGlobalStates(h.hsm); err != nil {
		return err
	}
	if err := hsm.RegisterTurnStates(h.hsm); err != nil {
		return err
	}
	if err := hsm.RegisterInterruptStates(h.hsm); err != nil {
		return err
	}

	// Add players to game (access via HSM)
	game = h.hsm.GetGame()
	for _, playerID := range h.playerList {
		player := h.players[playerID]
		game.AddPlayer(player)

		// Subscribe player buffs to EventBus
		for _, buff := range player.ActiveBuffs {
			game.SubscribeBuff(player, buff)
		}
	}

	return nil
}

// GetPlayer returns the player by userID.
func (h *NakamaMatchHandler) GetPlayer(userID string) *core.Player {
	return h.players[userID]
}

// GetRound returns the current round number (from HSM).
func (h *NakamaMatchHandler) GetRound() int {
	if h.hsm == nil {
		return 0
	}
	return h.hsm.GetRound()
}

// GetTurn returns the current turn index (from HSM).
func (h *NakamaMatchHandler) GetTurn() int {
	if h.hsm == nil {
		return 0
	}
	return h.hsm.GetTurn()
}

// GetMatchID returns the match ID.
func (h *NakamaMatchHandler) GetMatchID() string {
	return h.matchID
}

// sendActionRejectedWithCode sends an action rejection with a standardized error code.
// This is the preferred method for sending action rejection responses.
func (h *NakamaMatchHandler) sendActionRejectedWithCode(playerID string, opCode pkgnet.OpCode, errCode constants.ErrorCode, message string) error {
	h.logWarn("Action rejected with error code",
		"player_id", playerID,
		"op_code", opCode,
		"error_code", errCode,
		"reason", errCode.ToReason(),
		"message", message)

	rejected := pkgnet.ActionRejected{
		OpCode:    opCode,
		ErrorCode: errCode,
		Reason:    errCode.ToReason(),
		Message:   message,
	}

	adapter := NewNakamaBroadcastAdapter(h)
	return adapter.SendActionRejected(playerID, &rejected)
}