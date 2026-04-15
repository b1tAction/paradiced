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
	"github.com/b1tAction/paradiced/pkg/rng"
)

// NakamaMatchHandler implements the authoritative match handler for Paradiced.
// It manages game state, HSM execution, and client communication.
type NakamaMatchHandler struct {
	// Core game components
	game      *engine.Game      // Game instance with EventBus and GameLog
	hsm       *hsm.HSM          // Hierarchical State Machine
	mapEngine *gamemap.MapEngine // Map engine for path calculation
	diceMgr   *rng.DiceManager  // Dice manager for rolling

	// Message dispatcher
	dispatcher DispatcherAdapter // Dispatcher for sending messages to clients

	// Match identification
	matchID string // Nakama match ID

	// Player tracking
	players    map[string]*core.Player // userID -> Player mapping
	playerList []string                // Ordered player list for turn sequence

	// Configuration
	maxPlayers  int    // Maximum players (default: 4)
	mapLength   int    // Map length (default: 100)
	randomSeed  int64  // Random seed for reproducibility
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
		matchID:    matchID,
		maxPlayers: maxPlayers,
		mapLength:  mapLength,
		randomSeed: seed,
		players:    make(map[string]*core.Player),
		playerList: make([]string, 0),
		diceMgr:    rng.NewDiceManager(rngInst),
		dispatcher: nil, // Set via WithDispatcher or during MatchInit
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
	h.game = engine.NewGame(gameID, h.randomSeed)

	// Create HSM with game reference
	h.hsm = hsm.NewHSM(h.game)

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

	// Add players to game
	for _, playerID := range h.playerList {
		player := h.players[playerID]
		h.game.AddPlayer(player)

		// Subscribe player buffs to EventBus
		for _, buff := range player.ActiveBuffs {
			h.game.SubscribeBuff(player, buff)
		}
	}

	return nil
}

// GetPlayer returns the player by userID.
func (h *NakamaMatchHandler) GetPlayer(userID string) *core.Player {
	return h.players[userID]
}

// GetGameState returns the current game state.
func (h *NakamaMatchHandler) GetGameState() *engine.GameState {
	if h.game == nil {
		return nil
	}
	return h.game.State
}

// GetMatchID returns the match ID.
func (h *NakamaMatchHandler) GetMatchID() string {
	return h.matchID
}