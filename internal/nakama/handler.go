// Package nakama provides Nakama match handler implementation for Paradiced.
// This package implements the authoritative server logic using Nakama SDK.
package nakama

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/engine"
	"github.com/b1tAction/paradiced/internal/engine/hsm"
	"github.com/b1tAction/paradiced/internal/gamemap"
	"github.com/b1tAction/paradiced/pkg/constants"
	pkgerrors "github.com/b1tAction/paradiced/pkg/errors"
	"github.com/b1tAction/paradiced/pkg/id"
	pkgnet "github.com/b1tAction/paradiced/pkg/net"
	"github.com/b1tAction/paradiced/pkg/protocol"
	"github.com/b1tAction/paradiced/pkg/resource"
	"github.com/b1tAction/paradiced/pkg/rng"
	"github.com/heroiclabs/nakama-common/runtime"
)

// NakamaMatchHandler implements the authoritative match handler for Paradiced.
// It manages game state, HSM execution, and client communication.
// HSM is the single source of truth - Game is accessed via hsm.GetGame().
type NakamaMatchHandler struct {
	// Core game components
	hsm       *hsm.HSM           // Hierarchical State Machine (holds Game reference)
	mapEngine *gamemap.MapEngine // Map engine for path calculation
	diceMgr   *rng.DiceManager   // Dice manager for rolling

	// Message dispatcher
	dispatcher DispatcherAdapter // Dispatcher for sending messages to clients

	// Logger for debug/info/error output
	logger runtime.Logger // Nakama runtime logger

	// Match identification
	matchID string // Nakama match ID

	// Player tracking
	players      map[string]*core.Player // userID -> Player mapping
	playerList   []string                // Ordered player list for turn sequence
	disconnected map[string]bool         // userID -> disconnected status

	// Host tracking
	hostUserID string // User ID of the host (first player to join)

	// Start game signal (persisted across MatchLoop ticks)
	startRequested bool // Set by HandleStartGame, checked in WaitingForHost state

	// Map configuration (stored for StartGameAck broadcast)
	mapConfig *pkgnet.MapConfig

	// Online mini-game integration
	provider               protocol.OnlineMiniGameProvider // Colyseus provider for online mini-games (nil for frontend-only)
	pendingMiniGameResults map[string]int                 // playerID -> rank, populated by MatchSignal, consumed by MatchLoop

	// Debug mini-game trigger (populated by MatchSignal, consumed by MatchLoop)
	pendingTriggerMinigame string // Game type to force trigger, empty = no trigger pending

	// Configuration
	maxPlayers int    // Maximum players (default: 4)
	mapLength  int    // Map length (default: 100)
	randomSeed int64  // Random seed for reproducibility
	lobbyName  string // Human-readable lobby name for match listing

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
		matchID:      matchID,
		maxPlayers:   maxPlayers,
		mapLength:    mapLength,
		randomSeed:   seed,
		players:      make(map[string]*core.Player),
		playerList:   make([]string, 0),
		disconnected: make(map[string]bool),
		diceMgr:      rng.NewDiceManager(rngInst),
		dispatcher:   nil, // Set via WithDispatcher or during MatchInit
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

// WithProvider sets the online mini-game provider for RPC mode mini-games.
func (h *NakamaMatchHandler) WithProvider(provider protocol.OnlineMiniGameProvider) *NakamaMatchHandler {
	h.provider = provider
	return h
}

// logDebug logs a debug message if logger is available.
// Uses WithFields for structured data to avoid Printf-style %!(EXTRA) errors.
func (h *NakamaMatchHandler) logDebug(msg string, keysAndValues ...interface{}) {
	if h.logger != nil {
		fields := kvToFields(keysAndValues)
		h.logger.WithFields(fields).Debug(msg)
	}
}

// logInfo logs an info message if logger is available.
func (h *NakamaMatchHandler) logInfo(msg string, keysAndValues ...interface{}) {
	if h.logger != nil {
		fields := kvToFields(keysAndValues)
		h.logger.WithFields(fields).Info(msg)
	}
}

// logWarn logs a warn message if logger is available.
func (h *NakamaMatchHandler) logWarn(msg string, keysAndValues ...interface{}) {
	if h.logger != nil {
		fields := kvToFields(keysAndValues)
		h.logger.WithFields(fields).Warn(msg)
	}
}

// logError logs an error message if logger is available.
func (h *NakamaMatchHandler) logError(msg string, keysAndValues ...interface{}) {
	if h.logger != nil {
		fields := kvToFields(keysAndValues)
		h.logger.WithFields(fields).Error(msg)
	}
}

// formatKVs formats key-value pairs into a single string for runtime.Logger Printf-style calls.
// runtime.Logger's Info/Warn/Error/Debug use Printf format, so kv pairs must be
// pre-formatted into the msg string to avoid %!(EXTRA ...) errors.
func formatKVs(msg string, kv []interface{}) string {
	if len(kv) == 0 {
		return msg
	}
	pairs := make([]string, 0, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		pairs = append(pairs, fmt.Sprintf("%s=%v", kv[i], kv[i+1]))
	}
	if len(kv)%2 == 1 {
		pairs = append(pairs, fmt.Sprintf("extra=%v", kv[len(kv)-1]))
	}
	return msg + " " + strings.Join(pairs, " ")
}

// kvToFields converts alternating key-value pairs into a map for runtime.Logger.WithFields.
func kvToFields(kv []interface{}) map[string]interface{} {
	fields := make(map[string]interface{}, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if ok {
			fields[key] = kv[i+1]
		}
	}
	return fields
}

// initializeGame creates the Game instance and HSM.
// Called during MatchInit after all players have joined.
func (h *NakamaMatchHandler) initializeGame() error {
	// Create Game instance
	gameID := id.NewGameID()
	game := engine.NewGame(gameID, h.randomSeed)

	// Bridge Game's DebugLog to Nakama runtime.Logger.
	// runtime.Logger uses Printf-style (format string + format args), not structured key-value.
	// We must format kv pairs into the msg string before passing to runtime.Logger,
	// otherwise extra args produce %!(EXTRA ...) formatting errors.
	if h.logger != nil {
		game.DebugLog.WithWriter(func(msg string, kv ...interface{}) {
			formattedMsg := formatKVs(msg, kv)
			// Route to appropriate runtime.Logger level based on GameLogger prefix.
			// config.yml level:"info" allows Info+ to pass through naturally.
			if len(msg) >= 5 && msg[:5] == "DEBUG" {
				h.logger.Debug(formattedMsg)
			} else if len(msg) >= 4 && msg[:4] == "INFO" {
				h.logger.Info(formattedMsg)
			} else if len(msg) >= 4 && msg[:4] == "WARN" {
				h.logger.Warn(formattedMsg)
			} else {
				h.logger.Error(formattedMsg)
			}
		})
	}

	// Share Game's DebugLog with EventBus
	game.Bus.DebugLog = game.DebugLog

	// Initialize event, item, and buff pools from Registry definitions
	game.EventPool = engine.BuildEventPool()
	game.ItemPool = engine.BuildItemPool()
	game.BuffPool = engine.BuildBuffPool()

	// Create HSM with game reference (HSM is single source of truth)
	h.hsm = hsm.NewHSM(game)

	// Load map configuration from embedded resource (pkg/resource/default.json)
	mapConfig, err := resource.LoadDefault()
	if err != nil {
		return pkgerrors.Wrap(err, "NakamaHandler", "initializeGame.LoadDefault")
	}

	// Store map config for StartGameAck broadcast
	h.mapConfig = mapConfig

	// Build MapEngine from loaded configuration
	h.mapEngine = resource.BuildMapEngineFromConfig(mapConfig)

	// Register all states (inject online mini-game provider if configured)
	if h.provider != nil {
		if err := hsm.RegisterGlobalStatesWithProvider(h.hsm, h.provider); err != nil {
			return pkgerrors.Wrap(err, "NakamaHandler", "initializeGame.RegisterGlobalStatesWithProvider")
		}
	} else {
		if err := hsm.RegisterGlobalStates(h.hsm); err != nil {
			return pkgerrors.Wrap(err, "NakamaHandler", "initializeGame.RegisterGlobalStates")
		}
	}
	if err := hsm.RegisterTurnStates(h.hsm); err != nil {
		return pkgerrors.Wrap(err, "NakamaHandler", "initializeGame.RegisterTurnStates")
	}
	if err := hsm.RegisterInterruptStates(h.hsm); err != nil {
		return pkgerrors.Wrap(err, "NakamaHandler", "initializeGame.RegisterInterruptStates")
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

	// Initialize Boss player (always at end of Players list)
	if h.mapEngine != nil {
		mapEndIndex := h.mapEngine.Length - 1
		game.InitializeBoss(mapEndIndex)
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
