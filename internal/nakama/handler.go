// Package nakama provides Nakama match handler implementation for Paradiced.
// This package implements the authoritative server logic using Nakama SDK.
package nakama

import (
	"math/rand"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/engine"
	"github.com/b1tAction/paradiced/internal/engine/hsm"
	"github.com/b1tAction/paradiced/internal/gamemap"
	"github.com/b1tAction/paradiced/pkg/id"
	"github.com/b1tAction/paradiced/pkg/net"
	"github.com/b1tAction/paradiced/pkg/rng"
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

// sendActionRejected sends an action rejection notification to a player.
// Returns the original error to allow easy integration in handler functions.
func (h *NakamaMatchHandler) sendActionRejected(playerID string, rejected net.ActionRejected) error {
	adapter := NewNakamaBroadcastAdapter(h)
	return adapter.SendActionRejected(playerID, &rejected)
}