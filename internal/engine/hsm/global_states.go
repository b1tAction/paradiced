package hsm

import (
	"github.com/b1tAction/Fated/internal/core"
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
	ctx.SetMetadata("initialized", true)
}

func (s *MatchInitState) Update(ctx *StateContext) StateID {
	// Auto-transition to MiniGame after initialization
	return StateRoundMiniGame
}

func (s *MatchInitState) Exit(ctx *StateContext) {
	// Cleanup initialization resources
	ctx.Metadata = nil
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
	// Broadcast MiniGameStart to all clients
	s.totalPlayers = len(ctx.Game.Players) // Direct access to Game.Players
	s.resultsReceived = 0

	ctx.SetMetadata("mini_game_started", true)
	ctx.SetMetadata("waiting_for_results", true)
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
	ctx.SetMetadata("mini_game_started", false)
	ctx.SetMetadata("waiting_for_results", false)
}

// OnMiniGameResult handles mini-game result submission.
func (s *RoundMiniGameState) OnMiniGameResult(ctx *StateContext, playerID string, rank int) {
	s.resultsReceived++
	ctx.SetMetadata("result_"+playerID, rank)
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
	// Rank 1 -> Gold dice (1-10)
	// Rank 2 -> Silver dice (1-7)
	// Rank 3 -> Copper dice (1-5)
	// Rank 4 -> Wood dice (1-3)

	players := ctx.Game.Players // Direct access
	for _, player := range players {
		// Default assignment based on position (will be updated by mini-game results)
		rank := len(players) // Default to lowest rank
		if r, ok := ctx.GetMetadata("result_"+player.UserID).(int); ok {
			rank = r
		}
		s.diceAssignments[player.UserID] = rank
		ctx.SetMetadata("dice_"+player.UserID, getDiceType(rank))
	}

	// Increment round counter
	ctx.Game.State.Round++ // Direct access

	ctx.Success = true
}

func (s *RoundPrepState) Update(ctx *StateContext) StateID {
	// Auto-transition to TurnLoop after preparation
	return StateTurnLoop
}

func (s *RoundPrepState) Exit(ctx *StateContext) {
	s.diceAssignments = make(map[string]int)
}

// getDiceType returns dice type based on rank.
func getDiceType(rank int) string {
	switch rank {
	case 1:
		return "gold" // 1-10
	case 2:
		return "silver" // 1-7
	case 3:
		return "copper" // 1-5
	default:
		return "wood" // 1-3
	}
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
		BaseGlobalState:   BaseGlobalState{id: StateTurnLoop},
		currentPlayerIndex: 0,
		turnsCompleted:     0,
		reachedEnd:         false,
	}
}

func (s *TurnLoopState) Enter(ctx *StateContext) {
	// Initialize turn queue
	// First player starts their turn
	players := ctx.Game.Players // Direct access
	if len(players) > 0 {
		ctx.Game.State.Turn = 0 // Direct access
	}

	ctx.SetMetadata("turn_loop_active", true)
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
	ctx.SetMetadata("turn_loop_active", false)
	s.turnsCompleted = 0
	s.currentPlayerIndex = 0
}

// StartPlayerTurn initiates a player's turn (called by external controller).
func (s *TurnLoopState) StartPlayerTurn(ctx *StateContext) StateID {
	players := ctx.Game.Players // Direct access
	if s.currentPlayerIndex >= len(players) {
		// All players completed, next round
		s.currentPlayerIndex = 0
		s.turnsCompleted = 0
		return StateRoundMiniGame
	}

	// Set current player
	ctx.Game.State.Turn = s.currentPlayerIndex // Direct access

	// Transition to TurnUpkeep (first turn state)
	return StateTurnUpkeep
}

// OnTurnComplete handles turn completion.
func (s *TurnLoopState) OnTurnComplete(ctx *StateContext) {
	s.turnsCompleted++
	s.currentPlayerIndex++

	// Check if player reached end
	if reachedEnd, ok := ctx.GetMetadata("reached_end").(bool); ok && reachedEnd {
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
	triggerPlayer *core.Player // Direct type
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
	if playerID, ok := ctx.GetMetadata("boss_trigger_player").(string); ok {
		s.triggerPlayer = ctx.Game.GetPlayer(playerID) // Returns *core.Player directly
	}

	ctx.SetMetadata("boss_battle_active", true)
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
	ctx.SetMetadata("boss_battle_active", false)
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
	winner *core.Player // Direct type
}

// NewGameOverState creates a new GameOver state.
func NewGameOverState() *GameOverState {
	return &GameOverState{
		BaseGlobalState: BaseGlobalState{id: StateGameOver},
	}
}

func (s *GameOverState) Enter(ctx *StateContext) {
	// Broadcast winner
	// Perform final data settlement
	// Cleanup resources

	if winnerID, ok := ctx.GetMetadata("winner_id").(string); ok {
		s.winner = ctx.Game.GetPlayer(winnerID) // Returns *core.Player directly
	}

	ctx.Success = true
	ctx.SetMetadata("game_over", true)
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