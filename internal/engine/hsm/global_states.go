package hsm

import (
	"sort"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/engine/minigame"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/errors"
	"github.com/b1tAction/paradiced/pkg/id"
	pkgnet "github.com/b1tAction/paradiced/pkg/net"
	"github.com/b1tAction/paradiced/pkg/rng"
)

// ========== Global States (Layer 1) ==========

// BaseGlobalState provides common functionality for global layer states.
type BaseGlobalState struct {
	id StateID
}

// ID returns the state identifier.
func (s *BaseGlobalState) ID() StateID {
	return s.id
}

// CanTransitionTo defines valid transition rules for global states.
func (s *BaseGlobalState) CanTransitionTo(target StateID) bool {
	// Global states can transition to any global state
	return target.IsGlobalState()
}

// MatchInitState - Match Initialization State
// Generates map, assigns factions, initializes buffs.

type MatchInitState struct {
	BaseGlobalState
}

// NewMatchInitState creates a new MatchInit state.
func NewMatchInitState() *MatchInitState {
	return &MatchInitState{
		BaseGlobalState: BaseGlobalState{id: StateMatchInit},
	}
}

func (s *MatchInitState) Enter(ctx *StateContext) {
	game := ctx.GetGame()

	if game == nil {
		ctx.Error = errors.WrapHSMError(
			errors.NewInternalError("HSM", "MatchInit", nil),
			"MatchInit", 1, "Enter", "game instance is nil")
		return
	}

	// Map is already initialized by the caller (Nakama handler loads from
	// pkg/resource/default.json via BuildMapEngine). MatchInitState should
	// NOT regenerate the map, as that would overwrite DrawType, ProbGood,
	// ProbNeutral, ProbBad, EventID and other per-cell configuration.

	// 1. Initialize faction-specific buffs for all players
	// Uses ApplyBuffToPlayer for complete lifecycle (AddBuff + Subscribe)
	for _, player := range game.Players {
		game.InitializePlayerFactionBuffs(player)
	}

	// 3. Broadcast initial state sync
	if ctx.Broadcast != nil && ctx.Builder != nil {
		stateSync := ctx.Builder.BuildStateSync()
		ctx.Broadcast.BroadcastStateSync(stateSync)
	}

	ctx.SetBool(KeyInitialized, true)
}

func (s *MatchInitState) Update(ctx *StateContext) StateID {
	// Transition to WaitingForHost state (manual start mode)
	// Host can start game with 2-4 players
	return StateWaitingForHost
}

func (s *MatchInitState) Exit(ctx *StateContext) {
	// Cleanup initialization resources
	ctx.Delete("initialized")
}

// WaitingForHostState - Wait for Host to Start State
// Host can manually start game with 2-4 players.

type WaitingForHostState struct {
	BaseGlobalState
	startRequested bool
}

// KeyStartRequested is used in StateContext to signal game start.
const KeyStartRequested = "start_requested"

// NewWaitingForHostState creates a new WaitingForHost state.
func NewWaitingForHostState() *WaitingForHostState {
	return &WaitingForHostState{
		BaseGlobalState: BaseGlobalState{id: StateWaitingForHost},
		startRequested:  false,
	}
}

func (s *WaitingForHostState) Enter(ctx *StateContext) {
	// Nothing to do on enter - waiting is handled by Nakama match handler
	// which broadcasts WaitingSync to host when players join/leave
	ctx.SetBool("waiting_for_host", true)
}

func (s *WaitingForHostState) Update(ctx *StateContext) StateID {
	// Check if start signal was received via HandleStartGame
	if ctx.GetBoolOrDefault(KeyStartRequested, false) {
		// Start game - transition to RoundMiniGame
		return StateRoundMiniGame
	}

	// Stay in waiting state
	return StateNone
}

func (s *WaitingForHostState) Exit(ctx *StateContext) {
	// Cleanup waiting state markers
	ctx.Delete("waiting_for_host")
	ctx.Delete(KeyStartRequested)
}

// RoundMiniGameState - Mini-Game Phase State
// Waits for all players to submit mini-game data (game_data), then calculates rankings.

type RoundMiniGameState struct {
	BaseGlobalState
	resultsReceived int
	totalPlayers    int
	gameType        constants.MiniGameType                      // Current round mini-game type (server-selected)
	mode            constants.MiniGameMode                      // Frontend-driven or RPC-driven
	gameData        map[string]map[string]interface{}           // playerID -> game_data (frontend mode storage)
	rankCalculator  minigame.RankCalculator
}

// NewRoundMiniGameState creates a new RoundMiniGame state with default frontend-driven mode.
func NewRoundMiniGameState() *RoundMiniGameState {
	return &RoundMiniGameState{
		BaseGlobalState: BaseGlobalState{id: StateRoundMiniGame},
		resultsReceived: 0,
		totalPlayers:    0,
		gameType:        constants.MiniGameTypeDiceRace,
		mode:            constants.MiniGameModeFrontend,
		gameData:        make(map[string]map[string]interface{}),
		rankCalculator:  minigame.NewDefaultRankCalculator(),
	}
}

// WithMode sets the mini-game mode (frontend-driven or RPC-driven).
func (s *RoundMiniGameState) WithMode(mode constants.MiniGameMode) *RoundMiniGameState {
	s.mode = mode
	return s
}

// WithRankCalculator sets a custom rank calculator.
func (s *RoundMiniGameState) WithRankCalculator(calc minigame.RankCalculator) *RoundMiniGameState {
	s.rankCalculator = calc
	return s
}

// GetGameType returns the current round's mini-game type.
func (s *RoundMiniGameState) GetGameType() constants.MiniGameType {
	return s.gameType
}

func (s *RoundMiniGameState) Enter(ctx *StateContext) {
	// Start mini-game phase
	game := ctx.GetGame()
	// Clear round-level persistent data for new round
	if game != nil && game.RoundData != nil {
		game.RoundData.Clear()
	}

	// Count non-Boss players (Boss doesn't participate in MiniGame)
	nonBossPlayers := 0
	for _, p := range game.Players {
		if !p.ID.IsBoss() {
			nonBossPlayers++
		}
	}
	s.totalPlayers = nonBossPlayers
	s.resultsReceived = 0
	s.gameData = make(map[string]map[string]interface{}, nonBossPlayers)

	// Select mini-game type using game RNG for deterministic replay
	s.gameType = minigame.SelectMiniGameType(game.RNG)

	ctx.SetBool(KeyMiniGameStarted, true)
	ctx.SetBool(KeyWaitingForResults, true)

	// Broadcast MiniGameStart to all clients (excluding Boss)
	if ctx.Broadcast != nil {
		playerIDs := make([]string, 0, nonBossPlayers)
		for _, p := range game.Players {
			if !p.ID.IsBoss() {
				playerIDs = append(playerIDs, p.ID.UUID())
			}
		}
		start := &pkgnet.MiniGameStart{
			GameType: string(s.gameType),
			Players:  playerIDs,
		}
		ctx.Broadcast.BroadcastMiniGameStart(start)
	}
}

func (s *RoundMiniGameState) Update(ctx *StateContext) StateID {
	// Check if all results received
	// In actual implementation, this would check for incoming messages
	if s.resultsReceived >= s.totalPlayers {
		return StateRoundPrep
	}
	return StateNone // Stay waiting
}

func (s *RoundMiniGameState) Exit(ctx *StateContext) {
	// Broadcast final mini-game rankings before leaving mini-game phase.
	if ctx.Broadcast != nil {
		game := ctx.GetGame()
		if game != nil {
			rankings := make([]pkgnet.RankingEntry, 0, len(game.Players))

			for idx, p := range game.Players {
				// Skip Boss player - Boss doesn't participate in MiniGame
				if p.ID.IsBoss() {
					continue
				}
				rank := ctx.GetMiniGameRank(p.ID.UUID())
				if rank <= 0 {
					// Deterministic fallback for players without submission.
					rank = len(game.Players) + idx + 1
				}
				rankings = append(rankings, pkgnet.RankingEntry{
					PlayerID: p.ID.UUID(),
					Rank:     rank,
				})
			}

			sort.SliceStable(rankings, func(i, j int) bool {
				if rankings[i].Rank == rankings[j].Rank {
					return rankings[i].PlayerID < rankings[j].PlayerID
				}
				return rankings[i].Rank < rankings[j].Rank
			})

			_ = ctx.Broadcast.BroadcastMiniGameResult(&pkgnet.MiniGameResult{Rankings: rankings})
		}
	}

	ctx.SetBool(KeyMiniGameStarted, false)
	ctx.SetBool(KeyWaitingForResults, false)
}

// OnMiniGameDataSubmit handles client mini-game data submission (frontend-driven mode).
// Stores game_data, verifies game_type match, and calculates rankings when all submissions received.
// Returns true if all players have submitted and rankings were calculated.
func (s *RoundMiniGameState) OnMiniGameDataSubmit(ctx *StateContext, playerID string, gameType constants.MiniGameType, gameData map[string]interface{}) bool {
	if s.mode != constants.MiniGameModeFrontend {
		return false // Not applicable for RPC mode
	}

	// Verify game_type matches current round's selection
	if gameType != s.gameType {
		return false // Game type mismatch, will be rejected by handler
	}

	s.gameData[playerID] = gameData
	s.resultsReceived++

	// When all submissions received, calculate rankings
	if s.resultsReceived >= s.totalPlayers {
		ranks := s.rankCalculator.Calculate(s.gameType, s.gameData)
		for pid, rank := range ranks {
			ctx.SetMiniGameRank(pid, rank)
		}
		return true
	}
	return false
}

// OnMiniGameResult handles direct rank assignment (internal/RPC mode).
// Used by RPC report handler to directly set pre-calculated rankings.
func (s *RoundMiniGameState) OnMiniGameResult(ctx *StateContext, playerID string, rank int) {
	s.resultsReceived++
	ctx.SetMiniGameRank(playerID, rank)
}

// RoundPrepState - Round Preparation State
// Assigns dice types based on mini-game rankings.

type RoundPrepState struct {
	BaseGlobalState
	diceAssignments map[string]int // playerID -> diceType (1=gold, 2=silver, 3=copper, 4=wood)
}

// NewRoundPrepState creates a new RoundPrep state.
func NewRoundPrepState() *RoundPrepState {
	return &RoundPrepState{
		BaseGlobalState: BaseGlobalState{id: StateRoundPrep},
		diceAssignments: make(map[string]int),
	}
}

func (s *RoundPrepState) Enter(ctx *StateContext) {
	// Assign dice based on mini-game rankings
	// Rank 1 -> Gold dice (weighted toward high numbers)
	// Rank 2 -> Silver dice
	// Rank 3 -> Copper dice
	// Rank 4 -> Wood dice (uniform distribution)

	game := ctx.GetGame()
	players := game.Players

	// Reorder players by mini-game rank (lower rank goes first).
	// Stable sort preserves previous relative order for ties/unset ranks.
	type rankedPlayer struct {
		player *core.Player
		rank   int
		index  int
	}
	ranked := make([]rankedPlayer, len(players))
	for i, p := range players {
		rank := ctx.GetMiniGameRank(p.ID.UUID())
		if rank <= 0 {
			// Unset rank treated as lowest priority.
			rank = len(players) + 100
		}
		ranked[i] = rankedPlayer{player: p, rank: rank, index: i}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].rank == ranked[j].rank {
			return ranked[i].index < ranked[j].index
		}
		return ranked[i].rank < ranked[j].rank
	})
	for i := range ranked {
		players[i] = ranked[i].player
	}
	for _, player := range players {
		// Boss does not participate in dice assignment
		if player.ID.IsBoss() {
			continue
		}

		// Default assignment based on position (will be updated by mini-game results)
		rank := len(players) // Default to lowest rank
		playerRank := ctx.GetMiniGameRank(player.ID.UUID())
		if playerRank > 0 {
			rank = playerRank
		}
		s.diceAssignments[player.ID.UUID()] = rank
		// Store dice type as rng.DiceType for context
		ctx.SetDiceType(player.ID.UUID(), rng.RankToDiceType(rank))
	}

	// Note: round counter is NOT incremented here
	// round=1 is the first round, round++ happens when round completes
	// Use HSM's round counter via context
	_ = ctx.GetRound() // Current round (1 for first round)
}

func (s *RoundPrepState) Update(ctx *StateContext) StateID {
	// Auto-transition to TurnLoop after preparation
	return StateTurnLoop
}

func (s *RoundPrepState) Exit(ctx *StateContext) {
	s.diceAssignments = make(map[string]int)
}

// TurnLoopState - Turn Loop State
// Iterates through player turns until end condition.

type TurnLoopState struct {
	BaseGlobalState
	currentPlayerIndex int
	turnsCompleted     int
	pendingTurnStart   bool // Flag to start first player turn
}

// NewTurnLoopState creates a new TurnLoop state.
func NewTurnLoopState() *TurnLoopState {
	return &TurnLoopState{
		BaseGlobalState:    BaseGlobalState{id: StateTurnLoop},
		currentPlayerIndex: 0,
		turnsCompleted:     0,
		pendingTurnStart:   false,
	}
}

func (s *TurnLoopState) Enter(ctx *StateContext) {
	game := ctx.GetGame()
	players := game.Players

	// Reset state
	s.currentPlayerIndex = 0
	s.turnsCompleted = 0

	if len(players) > 0 {
		// Set first player as current turn player
		ctx.SetTurn(0)
		ctx.HSM.SetTurnPlayer(players[0])

		// Mark that we need to start first turn (handled in Update)
		s.pendingTurnStart = true
	}

	ctx.SetBool(KeyTurnLoopActive, true)
}

func (s *TurnLoopState) Update(ctx *StateContext) StateID {
	// Check for end conditions:
	// 1. Boss defeated -> GameOver
	// 2. All players completed turns -> Back to MiniGame (next round)

	// Check if Boss was defeated during a turn
	// Check both StateContext (same-tick) and Game.RoundData (cross-tick persistence)
	if ctx.GetBoolOrDefault(KeyBossDefeated, false) {
		return StateGameOver
	}
	game := ctx.GetGame()
	if game != nil && game.RoundData != nil && game.RoundData.GetBoolOrDefault(KeyBossDefeated, false) {
		// Copy from RoundData to StateContext for downstream states
		winnerID := game.RoundData.GetStringOrDefault(KeyBossDefeatedBy, "")
		ctx.SetBool(KeyBossDefeated, true)
		ctx.SetString(KeyBossDefeatedBy, winnerID)
		ctx.SetString(KeyWinner, winnerID)
		return StateGameOver
	}

	// Auto-start first player turn if pending
	if s.pendingTurnStart {
		s.pendingTurnStart = false
		return s.StartPlayerTurn(ctx)
	}

	// Stay in TurnLoop, wait for turn completion
	// TurnEndState.Update() returns StateNone, external controller (MatchLoop)
	// calls OnTurnComplete and StartPlayerTurn
	return StateNone
}

func (s *TurnLoopState) Exit(ctx *StateContext) {
	ctx.SetBool(KeyTurnLoopActive, false)
	s.turnsCompleted = 0
	s.currentPlayerIndex = 0
}

// CanTransitionTo defines valid transitions from TurnLoop.
func (s *TurnLoopState) CanTransitionTo(target StateID) bool {
	// TurnLoop can transition to:
	// - GameOver (when Boss is defeated)
	// - RoundEndWait (when round completes, wait for clients)
	// - Turn states (when starting player turn)
	return target == StateGameOver ||
		target == StateRoundEndWait ||
		target.IsTurnState()
}

// StartPlayerTurn initiates a player's turn (called by external controller).
func (s *TurnLoopState) StartPlayerTurn(ctx *StateContext) StateID {
	game := ctx.GetGame()
	players := game.Players
	if s.currentPlayerIndex >= len(players) {
		// All players completed, next round
		s.currentPlayerIndex = 0
		s.turnsCompleted = 0

		// Increment round counter for next round
		ctx.IncrementRound()

		return StateRoundEndWait
	}

	// Set current player turn index via HSM
	ctx.SetTurn(s.currentPlayerIndex)

	// Set current turn player in HSM
	ctx.HSM.SetTurnPlayer(players[s.currentPlayerIndex])

	// Transition to TurnUpkeep (first turn state)
	return StateTurnUpkeep
}

// OnTurnComplete handles turn completion.
func (s *TurnLoopState) OnTurnComplete(ctx *StateContext) {
	s.turnsCompleted++
	s.currentPlayerIndex++

	// Check if Boss was defeated during this turn
	// Check both StateContext and Game.RoundData
	bossDefeated := ctx.GetBoolOrDefault(KeyBossDefeated, false)
	if !bossDefeated {
		game := ctx.GetGame()
		if game != nil && game.RoundData != nil {
			bossDefeated = game.RoundData.GetBoolOrDefault(KeyBossDefeated, false)
		}
	}

	if bossDefeated {
		// Get winner from RoundData (more reliable than StateContext for cross-tick)
		game := ctx.GetGame()
		winnerID := ctx.GetStringOrDefault(KeyBossDefeatedBy, "")
		if winnerID == "" && game != nil && game.RoundData != nil {
			winnerID = game.RoundData.GetStringOrDefault(KeyBossDefeatedBy, "")
		}
		if winnerID != "" {
			ctx.SetString(KeyWinner, winnerID)
		}
	}
}

// ========== RoundEndWaitState ==========
// Waits for all clients to signal OpRoundReady before transitioning to RoundMiniGame.
// This gives clients time to finish rendering the last turn's animations.

type RoundEndWaitState struct {
	BaseGlobalState
	readyReceived int
	totalPlayers  int
}

// NewRoundEndWaitState creates a new RoundEndWait state.
func NewRoundEndWaitState() *RoundEndWaitState {
	return &RoundEndWaitState{
		BaseGlobalState: BaseGlobalState{id: StateRoundEndWait},
	}
}

func (s *RoundEndWaitState) Enter(ctx *StateContext) {
	game := ctx.GetGame()

	// Count non-Boss players
	nonBossCount := 0
	for _, p := range game.Players {
		if !p.ID.IsBoss() {
			nonBossCount++
		}
	}
	s.totalPlayers = nonBossCount
	s.readyReceived = 0

	ctx.SetBool(KeyRoundEndWaiting, true)

	// Broadcast StateSync showing "round_end_wait" global state
	if ctx.Broadcast != nil && ctx.Builder != nil {
		stateSync := ctx.Builder.BuildStateSync()
		ctx.Broadcast.BroadcastStateSync(stateSync)
	}
}

func (s *RoundEndWaitState) Update(ctx *StateContext) StateID {
	// Check if all clients have signaled ready
	if s.readyReceived >= s.totalPlayers {
		return StateRoundMiniGame
	}
	return StateNone // Stay waiting
}

func (s *RoundEndWaitState) Exit(ctx *StateContext) {
	ctx.SetBool(KeyRoundEndWaiting, false)
}

// CanTransitionTo defines valid transitions from RoundEndWait.
func (s *RoundEndWaitState) CanTransitionTo(target StateID) bool {
	return target == StateRoundMiniGame || target == StateGameOver
}

// OnRoundReady handles client's round-ready signal.
func (s *RoundEndWaitState) OnRoundReady(ctx *StateContext, playerID string) {
	s.readyReceived++
}

// ========== GameOverState ==========
// Final state, broadcasts winner and performs cleanup.

type GameOverState struct {
	BaseGlobalState
	winner *core.Player
}

// NewGameOverState creates a new GameOver state.
func NewGameOverState() *GameOverState {
	return &GameOverState{
		BaseGlobalState: BaseGlobalState{id: StateGameOver},
	}
}

func (s *GameOverState) Enter(ctx *StateContext) {
	// Broadcast winner and perform final data settlement
	game := ctx.GetGame()

	winnerID := ctx.GetStringOrDefault(KeyWinner, "")
	if winnerID != "" {
		parsedID, err := id.ParsePlayerID(winnerID)
		if err == nil {
			s.winner = game.GetPlayer(parsedID)
		}
	}

	ctx.SetBool(KeyGameOver, true)

	// Broadcast GameOver to all clients
	if ctx.Broadcast != nil {
		stats := make([]pkgnet.PlayerStats, len(game.Players))
		for i, p := range game.Players {
			stats[i] = pkgnet.PlayerStats{
				PlayerID:    p.ID.UUID(),
				RoundsWon:   0, // TODO: track rounds won
				EventsDrawn: 0, // TODO: track events drawn
				ItemsUsed:   0, // TODO: track items used
			}
		}
		over := &pkgnet.GameOver{
			WinnerID: winnerID,
			Stats:    stats,
		}
		ctx.Broadcast.BroadcastGameOver(over)
	}
}

func (s *GameOverState) Update(ctx *StateContext) StateID {
	// Terminal state, no transitions
	return StateNone
}

func (s *GameOverState) Exit(ctx *StateContext) {
	// Final cleanup
	s.winner = nil
}

// CanTransitionTo - GameOver is terminal, cannot transition.
func (s *GameOverState) CanTransitionTo(target StateID) bool {
	return false // Terminal state
}

// ========== Factory for Global States ==========

// GlobalStateFactory creates global layer states.
type GlobalStateFactory struct{}

// CreateState creates a global state by ID.
func (f *GlobalStateFactory) CreateState(id StateID) State {
	switch id {
	case StateMatchInit:
		return NewMatchInitState()
	case StateWaitingForHost:
		return NewWaitingForHostState()
	case StateRoundMiniGame:
		return NewRoundMiniGameState()
	case StateRoundPrep:
		return NewRoundPrepState()
	case StateTurnLoop:
		return NewTurnLoopState()
	case StateRoundEndWait:
		return NewRoundEndWaitState()
	case StateGameOver:
		return NewGameOverState()
	default:
		return nil
	}
}

// RegisterGlobalStates registers all global states with HSM.
func RegisterGlobalStates(hsm *HSM) error {
	factory := &GlobalStateFactory{}
	states := []State{
		factory.CreateState(StateMatchInit),
		factory.CreateState(StateWaitingForHost),
		factory.CreateState(StateRoundMiniGame),
		factory.CreateState(StateRoundPrep),
		factory.CreateState(StateTurnLoop),
		factory.CreateState(StateRoundEndWait),
		factory.CreateState(StateGameOver),
	}
	return hsm.RegisterStates(states)
}
