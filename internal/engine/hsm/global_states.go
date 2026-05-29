package hsm

import (
	"fmt"
	"sort"
	"time"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/engine"
	"github.com/b1tAction/paradiced/internal/engine/minigame"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/errors"
	pkgnet "github.com/b1tAction/paradiced/pkg/net"
	"github.com/b1tAction/paradiced/pkg/protocol"
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
// Generates map, assigns factions. Faction buffs are initialized later in
// WaitingForHostState.Exit() so that late-joining players and faction changes
// are correctly handled.

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

	game.DebugLog.Info("HSM.MatchInitState.Enter", "players", len(game.Players))

	// Map is already initialized by the caller (Nakama handler loads from
	// pkg/resource/default.json via BuildMapEngine). MatchInitState should
	// NOT regenerate the map, as that would overwrite DrawType, ProbGood,
	// ProbNeutral, ProbBad, EventID and other per-cell configuration.

	// Faction buffs are NOT initialized here — they are deferred to
	// WaitingForHostState.Exit() so that late-joining players and
	// faction changes during the waiting room are correctly reflected.

	// Broadcast initial state sync
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
	// Initialize faction-specific buffs for all players now that factions
	// are finalized (late joiners and faction changes are settled).
	// Uses ApplyBuffToPlayer for complete lifecycle (AddBuff + Subscribe).
	game := ctx.GetGame()
	if game != nil {
		for _, player := range game.Players {
			game.InitializePlayerFactionBuffs(player)
			// Initialize achievement handlers via EventBus (PhasePreAction subscriptions)
			game.InitializePlayerAchievements(player)
		}
	}

	// Broadcast StateSync so clients update faction and buff data
	// before entering MiniGame (MiniGameStart does not carry player data).
	// Without this, the host's client retains the initial faction (qing_long)
	// from MatchInit StateSync until the first TurnUpkeep StateSync arrives.
	if ctx.Broadcast != nil && ctx.Builder != nil {
		stateSync := ctx.Builder.BuildStateSync()
		ctx.Broadcast.BroadcastStateSync(stateSync)
	}

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
	gameType        constants.MiniGameType            // Current round mini-game type (server-selected)
	mode            constants.MiniGameMode            // Frontend-driven or RPC-driven
	gameData        map[string]map[string]interface{} // playerID -> game_data (frontend mode storage)
	rankCalculator  minigame.RankCalculator
	provider        protocol.OnlineMiniGameProvider // Online mini-game service provider (nil for frontend-only)
	connection      *pkgnet.MiniGameConn            // Connection info for online mode (nil for frontend mode)
	roomCreatedAt   time.Time                       // Timestamp when online room was created, for timeout check
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

// WithProvider sets the online mini-game provider for RPC mode.
// When set, online MiniGameTypes will use CreateRoom to establish
// an external game session instead of frontend-driven mode.
func (s *RoundMiniGameState) WithProvider(provider protocol.OnlineMiniGameProvider) *RoundMiniGameState {
	s.provider = provider
	return s
}

// GetGameType returns the current round's mini-game type.
func (s *RoundMiniGameState) GetGameType() constants.MiniGameType {
	return s.gameType
}

// GetMode returns the current mini-game execution mode.
func (s *RoundMiniGameState) GetMode() constants.MiniGameMode {
	return s.mode
}

// GetResultsReceived returns the count of received mini-game results.
func (s *RoundMiniGameState) GetResultsReceived() int {
	return s.resultsReceived
}

// GetTotalPlayers returns the total number of players participating in mini-game.
func (s *RoundMiniGameState) GetTotalPlayers() int {
	return s.totalPlayers
}

// GetConnection returns the online mini-game connection info (nil for frontend mode).
func (s *RoundMiniGameState) GetConnection() *pkgnet.MiniGameConn {
	return s.connection
}

// GetProvider returns the online mini-game provider (nil for frontend-only).
func (s *RoundMiniGameState) GetProvider() protocol.OnlineMiniGameProvider {
	return s.provider
}

// GetRoomCreatedAt returns the timestamp when the online room was created.
func (s *RoundMiniGameState) GetRoomCreatedAt() time.Time {
	return s.roomCreatedAt
}

func (s *RoundMiniGameState) Enter(ctx *StateContext) {
	// Start mini-game phase
	game := ctx.GetGame()
	if game != nil && game.RoundData != nil {
		game.RoundData.Clear()
	}

	// Count non-Boss players (Boss doesn't participate in MiniGame)
	nonBossPlayers := 0
	playerIDs := make([]string, 0)
	for _, p := range game.Players {
		if !p.ID.IsBoss() {
			nonBossPlayers++
			playerIDs = append(playerIDs, p.ID.UUID())
		}
	}
	s.totalPlayers = nonBossPlayers
	s.resultsReceived = 0
	s.gameData = make(map[string]map[string]interface{}, nonBossPlayers)

	// Select mini-game type using game RNG for deterministic replay unless a
	// debug trigger explicitly forced a valid type.
	forcedType := constants.ParseMiniGameType(ctx.GetStringOrDefault(KeyForcedMiniGameType, ""))
	if forcedType != constants.MiniGameTypeNone {
		s.gameType = forcedType
		if forcedType.IsOnline() && s.provider == nil {
			game.DebugLog.Warn("HSM.RoundMiniGameState.Enter.forced_online_without_provider", "game_type", forcedType)
		}
	} else {
		s.gameType = minigame.SelectMiniGameTypeWithProvider(game.RNG, s.provider != nil)
	}

	// 限制人数：如果抽到的是信任博弈且玩家人数少于或等于2人，并且没有通过 Debug 调试面板强制选择，则重新抽取其他游戏
	if s.gameType == constants.MiniGameTypeTrustDilemma && s.totalPlayers <= 2 && forcedType != constants.MiniGameTypeTrustDilemma {
		pool := make([]constants.MiniGameType, 0)
		for _, mt := range constants.AllMiniGameTypes {
			if mt != constants.MiniGameTypeTrustDilemma && (!mt.IsOnline() || s.provider != nil) {
				pool = append(pool, mt)
			}
		}
		if len(pool) > 0 {
			idx := game.RNG.Intn(len(pool))
			s.gameType = pool[idx]
		} else {
			s.gameType = constants.MiniGameTypeNone
		}
	}

	game.DebugLog.Info("HSM.RoundMiniGameState.Enter.selected", "game_type", s.gameType, "forced_game_type", forcedType, "has_provider", s.provider != nil, "total_players", s.totalPlayers)

	// No eligible mini-game available: skip mini-game phase entirely.
	if s.gameType == constants.MiniGameTypeNone {
		game.DebugLog.Warn("HSM.RoundMiniGameState.Enter.no_game_available", "reason", "no_eligible_mini_game_type")
		ctx.SetBool(KeyMiniGameStarted, false)
		return
	}

	// Mode determination based on game type and provider availability
	if s.gameType.IsOnline() && s.provider != nil {
		s.mode = constants.MiniGameModeRPC
		game.DebugLog.Info("HSM.RoundMiniGameState.Enter.rpc_mode", "game_type", s.gameType, "mode", s.mode)
		// Create room on mini-game service
		conn, err := s.provider.CreateRoom(s.gameType, playerIDs)
		if err != nil {
			// Room creation failed - try fallback to frontend-compatible type.
			// This is a recoverable condition, NOT a fatal error.
			// Do NOT set ctx.Error to avoid killing the game.
			game.DebugLog.Warn("HSM.RoundMiniGameState.Enter.room_creation_failed", "game_type", s.gameType, "error", err.Error(), "fallback", "frontend_mode")
			frontendPool := minigame.FrontendMiniGamePool()
			if len(frontendPool) > 0 {
				s.gameType = minigame.SelectFromPool(game.RNG, frontendPool)
				s.mode = constants.MiniGameModeFrontend
				s.connection = nil
				game.DebugLog.Info("HSM.RoundMiniGameState.Enter.fallback_success", "fallback_game_type", s.gameType, "mode", s.mode)
			} else {
				// No frontend types available, skip mini-game entirely
				game.DebugLog.Warn("HSM.RoundMiniGameState.Enter.no_frontend_fallback", "reason", "empty_frontend_pool")
				s.gameType = constants.MiniGameTypeNone
				ctx.SetBool(KeyMiniGameStarted, false)
				return
			}
		} else {
			s.connection = conn
			s.roomCreatedAt = time.Now()
			game.DebugLog.Info("HSM.RoundMiniGameState.Enter.room_created", "game_type", s.gameType, "room_id", conn.RoomID)
		}
	} else {
		s.mode = constants.MiniGameModeFrontend
		s.connection = nil
		game.DebugLog.Info("HSM.RoundMiniGameState.Enter.frontend_mode", "game_type", s.gameType, "mode", s.mode)
	}

	ctx.SetBool(KeyMiniGameStarted, true)
	ctx.SetBool(KeyWaitingForResults, true)

	// Broadcast MiniGameStart to all clients (excluding Boss)
	if ctx.Broadcast != nil {
		start := &pkgnet.MiniGameStart{
			GameType:   string(s.gameType),
			Players:    playerIDs,
			Connection: s.connection, // nil for frontend mode, populated for RPC mode
		}
		ctx.Broadcast.BroadcastMiniGameStart(start)
	}
}

func (s *RoundMiniGameState) Update(ctx *StateContext) StateID {
	// No mini-game available: skip to RoundPrep
	if s.gameType == constants.MiniGameTypeNone {
		return StateRoundPrep
	}
	// Check if all results received
	// In actual implementation, this would check for incoming messages
	if s.resultsReceived >= s.totalPlayers {
		return StateRoundPrep
	}
	return StateNone // Stay waiting
}

func (s *RoundMiniGameState) Exit(ctx *StateContext) {
	// Track rounds won stat and add mini-game score for all players
	game := ctx.GetGame()
	round := ctx.GetRound()
	if game != nil {
		totalNonBossPlayers := 0
		for _, p := range game.Players {
			if p.ID.IsBoss() {
				continue
			}
			totalNonBossPlayers++
		}

		for _, p := range game.Players {
			if p.ID.IsBoss() {
				continue
			}
			rank := ctx.GetMiniGameRank(p.ID.UUID())
			if rank == 1 {
				p.IncrementRoundsWon()
			}

			// Add mini-game ranking score
			if rank > 0 && totalNonBossPlayers >= 2 {
				score := constants.MiniGameRankToScore(rank, totalNonBossPlayers)
				reason := "小游戏第" + fmt.Sprintf("%d", rank) + "名"
				game.AddScoreToPlayer(p, constants.ScoreCategoryMiniGame, score, reason, round)
			}

			// mini_game_winner_three achievement: won mini-game rank 1 for 3+ rounds
			if !p.HasAchievement(constants.AchievementMiniGameWinnerThree) && p.GetRoundsWon() >= 3 {
				game.GrantAchievementToPlayer(p, constants.AchievementMiniGameWinnerThree)
				def := engine.GlobalAchievementRegistry.GetDefinition(constants.AchievementMiniGameWinnerThree)
				if def != nil {
					game.AddScoreToPlayer(p, constants.ScoreCategoryAchievement, def.Points, def.Name, 0)
				}
			}
		}
	}

	// Broadcast final mini-game rankings before leaving mini-game phase.
	if ctx.Broadcast != nil {
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

				// Include game_data submitted by this player for client rendering.
				var gameData map[string]interface{}
				if gd, ok := s.gameData[p.ID.UUID()]; ok {
					gameData = gd
				}

				rankings = append(rankings, pkgnet.RankingEntry{
					PlayerID:    p.ID.UUID(),
					DisplayName: p.ID.UUID(), // TODO: use actual display name from Nakama username
					Rank:        rank,
					GameData:    gameData,
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

// OnMiniGameGameData stores game_data for RPC mode ranking rendering.
// Called by MatchLoop after OnMiniGameResult when game_data is present.
// This data is used in Exit() to populate RankingEntry.GameData,
// enabling identical ranking rendering for both Frontend and RPC modes.
func (s *RoundMiniGameState) OnMiniGameGameData(playerID string, gameData map[string]interface{}) {
	s.gameData[playerID] = gameData
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
	game := ctx.GetGame()
	game.DebugLog.Info("HSM.RoundPrepState.Enter", "round", ctx.GetRound(), "players", len(game.Players))
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
	game.DebugLog.Info("HSM.TurnLoopState.Enter", "round", ctx.GetRound(), "players", len(game.Players))
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
}

// NewGameOverState creates a new GameOver state.
func NewGameOverState() *GameOverState {
	return &GameOverState{
		BaseGlobalState: BaseGlobalState{id: StateGameOver},
	}
}

func (s *GameOverState) Enter(ctx *StateContext) {
	// Score-based ranking settlement: detect HSM-direct achievements, rank by total score
	game := ctx.GetGame()
	ctx.SetBool(KeyGameOver, true)

	// Step 1: Detect HSM-direct achievements (survivor, luck_master)
	engine.EnsureAchievementRegistryInitialized()
	for _, p := range game.Players {
		if p.ID.IsBoss() {
			continue
		}
		// survivor achievement: never died during the game
		if !p.HasAchievement(constants.AchievementSurvivor) && p.GetDeathCount() == 0 {
			game.GrantAchievementToPlayer(p, constants.AchievementSurvivor)
			def := engine.GlobalAchievementRegistry.GetDefinition(constants.AchievementSurvivor)
			if def != nil {
				game.AddScoreToPlayer(p, constants.ScoreCategoryAchievement, def.Points, def.Name, 0)
			}
		}
		// luck_master achievement: LP at maximum when game ends
		if !p.HasAchievement(constants.AchievementLuckMaster) && p.LP == p.MaxLP {
			game.GrantAchievementToPlayer(p, constants.AchievementLuckMaster)
			def := engine.GlobalAchievementRegistry.GetDefinition(constants.AchievementLuckMaster)
			if def != nil {
				game.AddScoreToPlayer(p, constants.ScoreCategoryAchievement, def.Points, def.Name, 0)
			}
		}
	}

	// Step 2: Build rankings from all non-Boss players, sorted by total score descending
	rankings := make([]pkgnet.PlayerRanking, 0)
	for _, p := range game.Players {
		if p.ID.IsBoss() {
			continue
		}
		rankings = append(rankings, pkgnet.PlayerRanking{
			PlayerID:         p.ID.UUID(),
			DisplayName:      p.Metadata.GetStringOrDefault("display_name", p.ID.UUID()),
			TotalScore:       p.GetTotalScore(),
			MiniGameScore:    p.GetScoreByCategory(constants.ScoreCategoryMiniGame),
			BossScore:        p.GetScoreByCategory(constants.ScoreCategoryBoss),
			ItemScore:        p.GetScoreByCategory(constants.ScoreCategoryItem),
			AchievementScore: p.GetScoreByCategory(constants.ScoreCategoryAchievement),
			Achievements:     achievementTypesToStrings(p.GetAchievements()),
			ScoreReasons:     p.GetScoreReasons(),
		})
	}
	sort.SliceStable(rankings, func(i, j int) bool {
		if rankings[i].TotalScore != rankings[j].TotalScore {
			return rankings[i].TotalScore > rankings[j].TotalScore
		}
		return rankings[i].PlayerID < rankings[j].PlayerID
	})
	// Assign rank positions (1 = champion)
	for i := range rankings {
		rankings[i].Rank = i + 1
	}

	game.DebugLog.Info("HSM.GameOverState.Enter", "champion_id", rankings[0].PlayerID, "champion_score", rankings[0].TotalScore)

	// Step 3: Build stats for all players (including Boss)
	stats := make([]pkgnet.PlayerStats, 0)
	for _, p := range game.Players {
		scoreBreakdown := map[string]int{
			string(constants.ScoreCategoryMiniGame):    p.GetScoreByCategory(constants.ScoreCategoryMiniGame),
			string(constants.ScoreCategoryBoss):        p.GetScoreByCategory(constants.ScoreCategoryBoss),
			string(constants.ScoreCategoryItem):        p.GetScoreByCategory(constants.ScoreCategoryItem),
			string(constants.ScoreCategoryAchievement): p.GetScoreByCategory(constants.ScoreCategoryAchievement),
		}
		stats = append(stats, pkgnet.PlayerStats{
			PlayerID:        p.ID.UUID(),
			DisplayName:     p.Metadata.GetStringOrDefault("display_name", p.ID.UUID()),
			RoundsWon:       p.GetRoundsWon(),
			EventsDrawn:     p.GetEventsDrawn(),
			ItemsUsed:       p.GetItemsUsed(),
			BossDamageDealt: p.GetBossDamageDealt(),
			Achievements:    achievementTypesToStrings(p.GetAchievements()),
			TotalScore:      p.GetTotalScore(),
			ScoreBreakdown:  scoreBreakdown,
		})
	}

	// Step 4: Broadcast GameOver with rankings + stats
	if ctx.Broadcast != nil {
		over := &pkgnet.GameOver{
			Rankings: rankings,
			Stats:    stats,
		}
		ctx.Broadcast.BroadcastGameOver(over)
	}

	// Stop HSM after broadcasting GameOver.
	// This sets hsm.running=false, causing MatchLoop to return nil on next tick,
	// which triggers Nakama to call MatchTerminate → MatchStop → clear player data.
	ctx.HSM.Stop(ctx)
}

func (s *GameOverState) Update(ctx *StateContext) StateID {
	// Terminal state, no transitions
	return StateNone
}

func (s *GameOverState) Exit(ctx *StateContext) {
	// Final cleanup
}

// CanTransitionTo - GameOver is terminal, cannot transition.
func (s *GameOverState) CanTransitionTo(target StateID) bool {
	return false // Terminal state
}

// ========== Helper Functions ==========

// achievementTypesToStrings converts AchievementType slice to string slice for protocol.
func achievementTypesToStrings(types []constants.AchievementType) []string {
	result := make([]string, len(types))
	for i, t := range types {
		result[i] = string(t)
	}
	return result
}

// ========== Factory for Global States ==========

// GlobalStateFactory creates global layer states.
// If provider is set, RoundMiniGameState will use it for online mini-game room creation.
type GlobalStateFactory struct {
	provider protocol.OnlineMiniGameProvider
}

// NewGlobalStateFactory creates a factory with optional online mini-game provider.
func NewGlobalStateFactory(provider protocol.OnlineMiniGameProvider) *GlobalStateFactory {
	return &GlobalStateFactory{provider: provider}
}

// CreateState creates a global state by ID.
func (f *GlobalStateFactory) CreateState(id StateID) State {
	switch id {
	case StateMatchInit:
		return NewMatchInitState()
	case StateWaitingForHost:
		return NewWaitingForHostState()
	case StateRoundMiniGame:
		state := NewRoundMiniGameState()
		if f.provider != nil {
			state.WithProvider(f.provider)
		}
		return state
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

// RegisterGlobalStatesWithProvider registers all global states with HSM,
// injecting the online mini-game provider into RoundMiniGameState.
func RegisterGlobalStatesWithProvider(hsm *HSM, provider protocol.OnlineMiniGameProvider) error {
	factory := NewGlobalStateFactory(provider)
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
