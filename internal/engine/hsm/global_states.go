package hsm

import (
	"github.com/b1tAction/paradiced/internal/core"
	pkgnet "github.com/b1tAction/paradiced/pkg/net"
	"github.com/b1tAction/paradiced/pkg/id"
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
	// Initialize match:
	// 1. Generate map
	// 2. Assign player factions
	// 3. ZhuQue players get Fire buff
	// 4. Initialize EventBus subscriptions

	ctx.Success = true
	ctx.SetBool(KeyInitialized, true)
}

func (s *MatchInitState) Update(ctx *StateContext) StateID {
	// Auto-transition to MiniGame after initialization
	return StateRoundMiniGame
}

func (s *MatchInitState) Exit(ctx *StateContext) {
	// Cleanup initialization resources
	ctx.Delete("initialized")
}

// RoundMiniGameState - Mini-Game Phase State
// Waits for all players to submit mini-game rankings.

type RoundMiniGameState struct {
	BaseGlobalState
	resultsReceived int
	totalPlayers    int
}

// NewRoundMiniGameState creates a new RoundMiniGame state.
func NewRoundMiniGameState() *RoundMiniGameState {
	return &RoundMiniGameState{
		BaseGlobalState: BaseGlobalState{id: StateRoundMiniGame},
		resultsReceived: 0,
		totalPlayers:    0,
	}
}

func (s *RoundMiniGameState) Enter(ctx *StateContext) {
	// Start mini-game phase
	game := ctx.GetGame()
	s.totalPlayers = len(game.Players)
	s.resultsReceived = 0

	ctx.SetBool(KeyMiniGameStarted, true)
	ctx.SetBool(KeyWaitingForResults, true)

	// Broadcast MiniGameStart to all clients
	if ctx.Broadcast != nil {
		playerIDs := make([]string, len(game.Players))
		for i, p := range game.Players {
			playerIDs[i] = p.ID.UUID()
		}
		start := &pkgnet.MiniGameStart{
			GameType: "dice_race",
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
	ctx.SetBool(KeyMiniGameStarted, false)
	ctx.SetBool(KeyWaitingForResults, false)
}

// OnMiniGameResult handles mini-game result submission.
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
	for _, player := range players {
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

	// Increment round counter
	game.State.Round++

	ctx.Success = true
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
	reachedEnd         bool
}

// NewTurnLoopState creates a new TurnLoop state.
func NewTurnLoopState() *TurnLoopState {
	return &TurnLoopState{
		BaseGlobalState:    BaseGlobalState{id: StateTurnLoop},
		currentPlayerIndex: 0,
		turnsCompleted:     0,
		reachedEnd:         false,
	}
}

func (s *TurnLoopState) Enter(ctx *StateContext) {
	// Initialize turn queue
	// First player starts their turn
	game := ctx.GetGame()
	players := game.Players
	if len(players) > 0 {
		game.State.Turn = 0
	}

	ctx.SetBool(KeyTurnLoopActive, true)
}

func (s *TurnLoopState) Update(ctx *StateContext) StateID {
	// Check for end conditions:
	// 1. Player reached boss cell -> BossBattle
	// 2. All players completed turns -> Back to MiniGame (next round)

	if s.reachedEnd {
		return StateBossBattle
	}

	// Check if turn state should be entered
	// If not in turn state, enter first turn state
	// This will be controlled by external game loop calling TurnLoop.StartPlayerTurn()

	return StateNone // Stay in TurnLoop, wait for turn completion
}

func (s *TurnLoopState) Exit(ctx *StateContext) {
	ctx.SetBool(KeyTurnLoopActive, false)
	s.turnsCompleted = 0
	s.currentPlayerIndex = 0
}

// StartPlayerTurn initiates a player's turn (called by external controller).
func (s *TurnLoopState) StartPlayerTurn(ctx *StateContext) StateID {
	game := ctx.GetGame()
	players := game.Players
	if s.currentPlayerIndex >= len(players) {
		// All players completed, next round
		s.currentPlayerIndex = 0
		s.turnsCompleted = 0
		return StateRoundMiniGame
	}

	// Set current player
	game.State.Turn = s.currentPlayerIndex

	// Transition to TurnUpkeep (first turn state)
	return StateTurnUpkeep
}

// OnTurnComplete handles turn completion.
func (s *TurnLoopState) OnTurnComplete(ctx *StateContext) {
	s.turnsCompleted++
	s.currentPlayerIndex++

	// Check if player reached end
	if ctx.HasReachedEnd() {
		s.reachedEnd = true
	}
}

// OnPlayerReachedEnd marks that a player reached the boss cell.
func (s *TurnLoopState) OnPlayerReachedEnd() {
	s.reachedEnd = true
}

// CanTransitionTo defines valid transitions from TurnLoop.
func (s *TurnLoopState) CanTransitionTo(target StateID) bool {
	// TurnLoop can transition to:
	// - BossBattle (when player reaches end)
	// - RoundMiniGame (when round completes)
	// - Turn states (when starting player turn)
	return target == StateBossBattle ||
		target == StateRoundMiniGame ||
		target.IsTurnState()
}

// BossBattleState - Boss Battle State
// Handles end-game boss encounter.

type BossBattleState struct {
	BaseGlobalState
	triggerPlayer *core.Player
	bossDefeated  bool
}

// NewBossBattleState creates a new BossBattle state.
func NewBossBattleState() *BossBattleState {
	return &BossBattleState{
		BaseGlobalState: BaseGlobalState{id: StateBossBattle},
		bossDefeated:    false,
	}
}

func (s *BossBattleState) Enter(ctx *StateContext) {
	// Get the player who triggered boss battle
	// This would be set by TurnLoop before transitioning
	triggerID := ctx.GetStringOrDefault(KeyBossTrigger, "")
	if triggerID != "" {
		parsedID, err := id.ParsePlayerID(triggerID)
		if err == nil {
			s.triggerPlayer = ctx.GetGame().GetPlayer(parsedID)
		}
	}

	ctx.SetBool(KeyBossBattleActive, true)
}

func (s *BossBattleState) Update(ctx *StateContext) StateID {
	// Wait for boss battle result
	// In actual implementation, this would check for battle outcome

	if s.bossDefeated {
		return StateGameOver
	}

	return StateNone // Stay in boss battle
}

func (s *BossBattleState) Exit(ctx *StateContext) {
	ctx.SetBool(KeyBossBattleActive, false)
	s.triggerPlayer = nil
}

// OnBossDefeated marks boss as defeated.
func (s *BossBattleState) OnBossDefeated() {
	s.bossDefeated = true
}

// GameOverState - Game Over State
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

	ctx.Success = true
	ctx.SetBool(KeyGameOver, true)

	// Broadcast GameOver to all clients
	if ctx.Broadcast != nil {
		stats := make([]pkgnet.PlayerStats, len(game.Players))
		for i, p := range game.Players {
			stats[i] = pkgnet.PlayerStats{
				UserID:      p.ID.UUID(),
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
	case StateRoundMiniGame:
		return NewRoundMiniGameState()
	case StateRoundPrep:
		return NewRoundPrepState()
	case StateTurnLoop:
		return NewTurnLoopState()
	case StateBossBattle:
		return NewBossBattleState()
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
		factory.CreateState(StateRoundMiniGame),
		factory.CreateState(StateRoundPrep),
		factory.CreateState(StateTurnLoop),
		factory.CreateState(StateBossBattle),
		factory.CreateState(StateGameOver),
	}
	return hsm.RegisterStates(states)
}
