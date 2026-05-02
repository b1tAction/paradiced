package hsm

import (
	"fmt"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/engine"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/internal/gamemap"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/gamelog"
	"github.com/b1tAction/paradiced/pkg/id"
	pkgnet "github.com/b1tAction/paradiced/pkg/net"
	"github.com/b1tAction/paradiced/pkg/rng"
)

// ========== GameTestHarness ==========

// HarnessConfig configures the GameTestHarness setup.
type HarnessConfig struct {
	// Seed for deterministic RNG (0 = auto from time).
	Seed int64

	// PlayerCount is the number of players (2-4, default 4).
	PlayerCount int

	// MapLength is the total cell count (default 100).
	MapLength int

	// CellTypeOverrides specifies custom cell types at specific indices.
	// e.g., {25: constants.CellTypeCheckpoint, 30: constants.CellTypeFragile}
	CellTypeOverrides map[int]constants.CellType

	// Factions assigns factions to players by index.
	// If nil, defaults are assigned: QingLong, ZhuQue, BaiHu, XuanWu.
	Factions []constants.Faction

	// InitialHP sets starting HP for all players (default 6).
	InitialHP int

	// MaxHP sets maximum HP for all players (default 8).
	MaxHP int

	// InitialLP sets starting LP for all players (default 3).
	InitialLP int

	// MaxLP sets maximum LP for all players (default 8).
	MaxLP int

	// InitialPosition sets starting position for all players (default 0).
	InitialPosition int
}

// DefaultHarnessConfig returns default configuration for 4-player game.
func DefaultHarnessConfig() *HarnessConfig {
	return &HarnessConfig{
		Seed:             42,
		PlayerCount:      4,
		MapLength:        100,
		CellTypeOverrides: nil,
		Factions:         []constants.Faction{
			constants.FactionQingLong,
			constants.FactionZhuQue,
			constants.FactionBaiHu,
			constants.FactionXuanWu,
		},
		InitialHP:        6,
		MaxHP:            8,
		InitialLP:        3,
		MaxLP:            8,
		InitialPosition:  0,
	}
}

// GameTestHarness sets up a complete game environment for integration testing.
// It creates Game+HSM+MapEngine with MockBroadcastAdapter and provides
// methods to drive the game through rounds with simulated player actions.
type GameTestHarness struct {
	// Core game components
	Game      *engine.Game
	HSM       *HSM
	MapEngine *gamemap.MapEngine

	// Protocol mock
	Broadcast *pkgnet.MockBroadcastAdapter
	Builder   *builderAdapter // wraps internal/net.Builder for test use

	// Players
	Players []*core.Player
}

// builderAdapter wraps internal/net.Builder to provide pkg/net.Builder interface.
type builderAdapter struct {
	hsmInst *HSM
}

// NewGameTestHarness creates a fully initialized test harness.
// Sets up Game, HSM, MapEngine, players with factions, mock broadcast,
// and registers all HSM states.
func NewGameTestHarness(config *HarnessConfig) *GameTestHarness {
	if config == nil {
		config = DefaultHarnessConfig()
	}

	// Validate config
	if config.PlayerCount < 2 {
		config.PlayerCount = 2
	}
	if config.PlayerCount > 4 {
		config.PlayerCount = 4
	}
	if config.InitialHP <= 0 {
		config.InitialHP = 6
	}
	if config.MaxHP <= 0 {
		config.MaxHP = 8
	}
	if config.InitialLP <= 0 {
		config.InitialLP = 3
	}
	if config.MaxLP <= 0 {
		config.MaxLP = 8
	}

	// Create Game instance
	gameID := id.NewGameID()
	game := engine.NewGame(gameID, config.Seed)

	// Initialize pools from Registry definitions
	game.EventPool = engine.BuildEventPool()
	game.ItemPool = engine.BuildItemPool()
	game.BuffPool = engine.BuildBuffPool()

	// Create HSM
	hsmInst := NewHSM(game)

	// Create MapEngine
	mapEngine := gamemap.NewMapEngine(config.MapLength)
	mapEngine.GenerateLinearMap(config.CellTypeOverrides)

	// Set map engine in HSM
	hsmInst.SetMapEngine(mapEngine)

	// Register all states
	RegisterGlobalStates(hsmInst)
	RegisterTurnStates(hsmInst)
	RegisterInterruptStates(hsmInst)

	// Create mock broadcast adapter
	broadcast := pkgnet.NewMockBroadcastAdapter()

	// Create players with factions
	factions := config.Factions
	if len(factions) < config.PlayerCount {
		// Fill remaining with default factions
		defaults := []constants.Faction{
			constants.FactionQingLong,
			constants.FactionZhuQue,
			constants.FactionBaiHu,
			constants.FactionXuanWu,
		}
		for i := len(factions); i < config.PlayerCount; i++ {
			factions = append(factions, defaults[i%len(defaults)])
		}
	}

	players := make([]*core.Player, config.PlayerCount)
	for i := 0; i < config.PlayerCount; i++ {
		player := core.NewPlayer(core.PlayerConfig{
			ID:      id.NewPlayerID(),
			InitHP:  config.InitialHP,
			InitLP:  config.InitialLP,
			MaxHP:   config.MaxHP,
			MaxLP:   config.MaxLP,
			Faction: factions[i],
			StartPos: config.InitialPosition,
		})
		players[i] = player
		game.AddPlayer(player)
	}

	// Initialize faction buffs for all players
	for _, player := range players {
		game.InitializePlayerFactionBuffs(player)
	}

	// Start HSM (required for TransitionTo to work)
	startCtx := NewStateContext().WithHSM(hsmInst).WithBroadcast(broadcast).WithBuilder(&builderAdapter{hsmInst: hsmInst})
	hsmInst.Start(StateMatchInit, startCtx)

	return &GameTestHarness{
		Game:      game,
		HSM:       hsmInst,
		MapEngine: mapEngine,
		Broadcast: broadcast,
		Builder:   &builderAdapter{hsmInst: hsmInst},
		Players:   players,
	}
}

// ========== State Context Helpers ==========

// newCtx creates a fresh StateContext with HSM, broadcast, and builder.
func (h *GameTestHarness) newCtx(player *core.Player) *StateContext {
	return NewStateContext().
		WithHSM(h.HSM).
		WithPlayer(player).
		WithBroadcast(h.Broadcast).
		WithBuilder(h.Builder)
}

// ========== Player Turn Execution ==========

// RunPlayerTurn executes a single player's complete turn through HSM:
// TurnLoop → TurnUpkeep → (BeforeTurn phase) → MainAction → (dice roll) →
// TurnMoving → TurnLanded → TurnDraw(if applicable) → TurnEnd
//
// diceSteps is the number of steps to move (simulated dice roll).
// Returns error if any state execution fails.
//
// This method assumes HSM is already in an appropriate state for turn execution.
// If not in TurnLoop, it will transition there first.
func (h *GameTestHarness) RunPlayerTurn(playerIndex int, diceSteps int) error {
	player := h.Players[playerIndex]

	// Ensure HSM is in TurnLoop global state
	if h.HSM.GetGlobalStateID() != StateTurnLoop {
		// Enter TurnLoop state
		loopCtx := h.newCtx(nil)
		err := h.HSM.TransitionTo(StateTurnLoop, loopCtx)
		if err != nil {
			return err
		}
		if loopCtx.Error != nil {
			return loopCtx.Error
		}
	}

	// Set current turn player in HSM
	h.HSM.SetTurnPlayer(player)

	// Create context for the turn
	ctx := h.newCtx(player)
	ctx.SetInt(KeyDiceSteps, diceSteps)

	// Set dice type (default to copper for tests)
	ctx.SetDiceType(player.ID.UUID(), rng.DiceTypeCopper)

	// === TurnUpkeep ===
	err := h.HSM.TransitionTo(StateTurnUpkeep, ctx)
	if err != nil {
		return err
	}
	if ctx.Error != nil {
		return ctx.Error
	}

	// Check if decisions were produced (BeforeTurn phase)
	if h.HSM.IsWaiting() {
		// Auto-resolve decision (choose option 0)
		err = h.HSM.OnUserChoice(0, ctx)
		if err != nil {
			return err
		}
	}

	// === MainAction ===
	// After TurnUpkeep auto-transitions through Update(),
	// HSM should be in MainAction state (or waiting for decision).
	// We need to roll the dice to trigger state transition.
	// The HSM's TransitionTo will handle auto-proceeding through states.

	currentTurnState := h.HSM.GetTurnStateID()

	if currentTurnState == StateMainAction {
		// Roll dice to trigger movement
		err = h.HSM.OnRollDice(ctx)
		if err != nil {
			return err
		}
		if ctx.Error != nil {
			return ctx.Error
		}
	}

	// Handle any remaining decisions during the turn
	if h.HSM.IsWaiting() {
		decisionCtx := h.newCtx(player)
		err = h.HSM.OnUserChoice(0, decisionCtx)
		if err != nil {
			return err
		}
	}

	return nil
}

// RunPlayerTurnWithBuff runs a player turn with an additional buff applied.
// The buff is added before the turn starts and properly subscribed.
func (h *GameTestHarness) RunPlayerTurnWithBuff(playerIndex int, diceSteps int, buffType constants.BuffType, duration int) error {
	player := h.Players[playerIndex]
	h.AddBuffToPlayer(player, buffType, duration)
	return h.RunPlayerTurn(playerIndex, diceSteps)
}

// ========== Full Round Execution ==========

// RunFullRound drives HSM through a complete round for all players.
// miniGameRanks maps player index to mini-game rank (1=best, 4=worst).
// Returns error if any state execution fails.
//
// Flow: RoundMiniGame → submit ranks → RoundPrep → TurnLoop →
// (for each player: TurnUpkeep → decisions → MainAction → dice → Moving → Landed → End)
func (h *GameTestHarness) RunFullRound(miniGameRanks map[int]int) error {
	// HSM is already started and in WaitingForHost state (from harness creation).
	// Signal game start.
	hostCtx := h.newCtx(nil)
	hostCtx.SetBool(KeyStartRequested, true)
	_, err := h.HSM.Update(hostCtx)
	if err != nil {
		return fmt.Errorf("Update(hostCtx) failed: %v", err)
	}

	// After Update, HSM should be in RoundMiniGame (auto-transitioned via WaitingForHost.Update)
	if h.HSM.GetGlobalStateID() != StateRoundMiniGame {
		mgCtx := h.newCtx(nil)
		err = h.HSM.TransitionTo(StateRoundMiniGame, mgCtx)
		if err != nil {
			return err
		}
		if mgCtx.Error != nil {
			return mgCtx.Error
		}
	}

	// Submit mini-game results
	mgState := h.HSM.GetGlobalState()
	roundMiniGame, ok := mgState.(*RoundMiniGameState)
	if !ok {
		return fmt.Errorf("RunFullRound: expected RoundMiniGameState, got %T", mgState)
	}
	for i, player := range h.Players {
		rank := miniGameRanks[i]
		if rank <= 0 {
			rank = i + 1
		}
		mgCtx := h.newCtx(player)
		roundMiniGame.OnMiniGameResult(mgCtx, player.ID.UUID(), rank)
	}

	// === RoundPrep → TurnLoop → TurnUpkeep ===
	// TransitionTo(StateRoundPrep) auto-proceeds to TurnLoop → TurnUpkeep.
	// TurnUpkeepState.Enter publishes PhaseBeforeTurn which may produce decisions.
	// If decisions exist, HSM enters WaitDecision (paused).
	prepCtx := h.newCtx(nil)
	err = h.HSM.TransitionTo(StateRoundPrep, prepCtx)
	if err != nil {
		return err
	}
	if prepCtx.Error != nil {
		return prepCtx.Error
	}

	// Check if HSM is paused (waiting for decision from BeforeTurn)
	if h.HSM.IsWaiting() {
		// Resolve all pending decisions for first player
		err = h.HSM.OnUserChoice(0, h.newCtx(h.Players[0]))
		if err != nil {
			return err
		}
	}

	// After resolving decisions, TurnUpkeepState.Update should auto-proceed
	// to MainAction. But HSM auto-proceed only happens inside TransitionTo.
	// We need to check the current turn state and drive it.

	// At this point, HSM should be in TurnLoop with a turn state.
	// Run all player turns
	for i := range h.Players {
		// The first player's turn may already have started (from auto-proceed).
		// Subsequent players need to be started via advanceTurn + RunSingleTurnInLoop.
		if i == 0 {
			// First player's TurnUpkeep may already be done.
			// Check if we're in MainAction state.
			if h.HSM.GetTurnStateID() == StateMainAction {
				err = h.HSM.OnRollDice(h.newCtx(h.Players[0]))
				if err != nil {
					return err
				}
			} else if h.HSM.GetTurnStateID() == StateNone {
				// No turn started - start manually
				err = h.RunSingleTurnInLoop(0, 3)
				if err != nil {
					return err
				}
			}
		} else {
			// Subsequent players
			err = h.RunSingleTurnInLoop(i, 3)
			if err != nil {
				return err
			}
		}

		// Advance to next player's turn (if not last player)
		if i < len(h.Players)-1 {
			h.advanceTurn()
		}
	}

	return nil
}

// RunSingleTurnInLoop executes a single player turn when HSM is already
// in TurnLoop global state. Unlike RunPlayerTurn, this does NOT transition
// to TurnLoop first - it assumes we're already there.
func (h *GameTestHarness) RunSingleTurnInLoop(playerIndex int, diceSteps int) error {
	player := h.Players[playerIndex]
	h.HSM.SetTurnPlayer(player)

	ctx := h.newCtx(player)
	ctx.SetInt(KeyDiceSteps, diceSteps)
	ctx.SetDiceType(player.ID.UUID(), rng.DiceTypeCopper)

	// === TurnUpkeep ===
	err := h.HSM.TransitionTo(StateTurnUpkeep, ctx)
	if err != nil {
		return err
	}
	if ctx.Error != nil {
		return ctx.Error
	}

	// Handle BeforeTurn decisions
	if h.HSM.IsWaiting() {
		err = h.HSM.OnUserChoice(0, ctx)
		if err != nil {
			return err
		}
	}

	// === MainAction ===
	currentTurnState := h.HSM.GetTurnStateID()

	if currentTurnState == StateMainAction {
		err = h.HSM.OnRollDice(ctx)
		if err != nil {
			return err
		}
		if ctx.Error != nil {
			return ctx.Error
		}
	}

	// Handle remaining decisions
	if h.HSM.IsWaiting() {
		decCtx := h.newCtx(player)
		err = h.HSM.OnUserChoice(0, decCtx)
		if err != nil {
			return err
		}
	}

	return nil
}

// advanceTurn simulates the MatchLoop TurnEnd→NextPlayer transition.
// Gets the TurnLoopState and calls OnTurnComplete + StartPlayerTurn.
func (h *GameTestHarness) advanceTurn() {
	globalState := h.HSM.GetGlobalState()
	turnLoopState, ok := globalState.(*TurnLoopState)
	if !ok {
		return
	}

	ctx := h.newCtx(nil)
	turnLoopState.OnTurnComplete(ctx)

	nextState := turnLoopState.StartPlayerTurn(ctx)
	if nextState != StateNone {
		h.HSM.TransitionTo(nextState, ctx)
	}
}

// ========== Buff/Item Manipulation ==========

// AddBuffToPlayer adds a buff to a player and subscribes it to EventBus.
func (h *GameTestHarness) AddBuffToPlayer(player *core.Player, buffType constants.BuffType, duration int) {
	buff := core.NewBuff(buffType, duration)
	h.Game.ApplyBuffToPlayer(player, buff)
}

// AddItemToPlayer adds an item to a player and subscribes it to EventBus.
func (h *GameTestHarness) AddItemToPlayer(player *core.Player, itemType constants.ItemType) {
	item := core.NewItem(itemType)
	player.AddItem(item)
	h.Game.SubscribeItem(player, item)
}

// KillPlayer sets a player to dead state for testing death/respawn scenarios.
func (h *GameTestHarness) KillPlayer(player *core.Player) {
	player.IsDead = true
	player.HP = 0
}

// ========== State Verification ==========

// VerifyPlayerHP checks if a player's HP matches the expected value.
func (h *GameTestHarness) VerifyPlayerHP(player *core.Player, expected int) bool {
	return player.HP == expected
}

// VerifyPlayerLP checks if a player's LP matches the expected value.
func (h *GameTestHarness) VerifyPlayerLP(player *core.Player, expected int) bool {
	return player.LP == expected
}

// VerifyPlayerPosition checks if a player's position matches the expected value.
func (h *GameTestHarness) VerifyPlayerPosition(player *core.Player, expected int) bool {
	return player.Position == expected
}

// VerifyBuffOnPlayer checks if a player has a specific buff type active.
func (h *GameTestHarness) VerifyBuffOnPlayer(player *core.Player, buffType constants.BuffType) bool {
	return player.HasBuff(buffType)
}

// VerifyBuffNotOnPlayer checks if a player does NOT have a specific buff type active.
func (h *GameTestHarness) VerifyBuffNotOnPlayer(player *core.Player, buffType constants.BuffType) bool {
	return !player.HasBuff(buffType)
}

// ========== GameLog Access ==========

// GetGameLogEntries returns all entries from the current turn.
func (h *GameTestHarness) GetGameLogEntries() []gamelog.LogEntry {
	return h.Game.Log.GetCurrentTurnEntries()
}

// GetGameLogSegments returns all completed turn segments.
func (h *GameTestHarness) GetGameLogSegments() []*gamelog.TurnSegment {
	return h.Game.Log.GetTurnSegments()
}

// ========== builderAdapter implements pkg/net.Builder interface ==========

func (b *builderAdapter) BuildStateSync() *pkgnet.StateSync {
	return &pkgnet.StateSync{
		GlobalState: b.hsmInst.GetGlobalStateID().String(),
		TurnState:   b.hsmInst.GetTurnStateID().String(),
		Round:       b.hsmInst.GetRound(),
		Turn:        b.hsmInst.GetTurn(),
		Paused:      b.hsmInst.IsPaused(),
	}
}

func (b *builderAdapter) BuildFullSyncStateSync() *pkgnet.StateSync {
	return &pkgnet.StateSync{
		GlobalState:     b.hsmInst.GetGlobalStateID().String(),
		TurnState:       b.hsmInst.GetTurnStateID().String(),
		CurrentPlayerID: "",
		Round:           b.hsmInst.GetRound(),
		Turn:            b.hsmInst.GetTurn(),
		Paused:          b.hsmInst.IsPaused(),
	}
}

func (b *builderAdapter) BuildAvailable() *pkgnet.Available {
	return &pkgnet.Available{}
}

func (b *builderAdapter) BuildDecisionFromEvent(decision *event.Decision) *pkgnet.Decision {
	return &pkgnet.Decision{
		ID:     decision.ID.UUID(),
		Prompt: decision.Prompt,
	}
}

func (b *builderAdapter) SetDiceType(_ string) {}

func (b *builderAdapter) BuildMapInfo() *pkgnet.MapInfo {
	return &pkgnet.MapInfo{}
}