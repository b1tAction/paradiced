// Package test provides testing utilities for the net protocol layer.
package test

import (
	"time"

	pkgnet "github.com/b1tAction/fated/pkg/net"
	internalnet "github.com/b1tAction/fated/internal/net"
	"github.com/b1tAction/fated/internal/engine"
	"github.com/b1tAction/fated/internal/engine/hsm"
	"github.com/b1tAction/fated/pkg/id"
	"github.com/b1tAction/fated/pkg/rng"
)

// TestHelper provides testing utilities for simulating game flow
// and capturing broadcast messages.
type TestHelper struct {
	game        *engine.Game
	hsm         *hsm.HSM
	builder     *internalnet.Builder
	diceMgr     *rng.DiceManager

	// Message capture
	broadcasts  []pkgnet.Message
	playerMsgs  map[string][]pkgnet.Message
}

// NewTestHelper creates a new test helper with a random seed.
func NewTestHelper(seed int64) *TestHelper {
	game := engine.NewGame(id.NewGameID(), seed)
	hsmInstance := hsm.NewHSM(game)

	// Register states (simplified for testing)
	hsm.RegisterGlobalStates(hsmInstance)
	hsm.RegisterTurnStates(hsmInstance)
	hsm.RegisterInterruptStates(hsmInstance)

	return &TestHelper{
		game:       game,
		hsm:        hsmInstance,
		builder:    internalnet.NewBuilder(hsmInstance, game),
		diceMgr:    rng.NewDiceManager(game.RNG),
		broadcasts: make([]pkgnet.Message, 0),
		playerMsgs: make(map[string][]pkgnet.Message),
	}
}

// SimulateRollDice simulates a dice roll and returns the result.
func (h *TestHelper) SimulateRollDice(diceType rng.DiceType) int {
	dice := rng.NewDice(diceType, h.game.RNG)
	return dice.Roll()
}

// SimulateRollPlayerDice rolls a player's special dice.
func (h *TestHelper) SimulateRollPlayerDice(playerID string) int {
	return h.diceMgr.RollSpecialDice(playerID)
}

// CaptureBroadcast captures a broadcast message for later retrieval.
func (h *TestHelper) CaptureBroadcast(op pkgnet.OpCode, data interface{}) error {
	msg, err := pkgnet.NewMessage(op, data)
	if err != nil {
		return err
	}
	h.broadcasts = append(h.broadcasts, *msg)
	return nil
}

// CaptureSendToPlayer captures a message sent to a specific player.
func (h *TestHelper) CaptureSendToPlayer(playerID string, op pkgnet.OpCode, data interface{}) error {
	msg, err := pkgnet.NewMessage(op, data)
	if err != nil {
		return err
	}
	h.playerMsgs[playerID] = append(h.playerMsgs[playerID], *msg)
	return nil
}

// GetBroadcasts returns all captured broadcast messages.
func (h *TestHelper) GetBroadcasts() []pkgnet.Message {
	return h.broadcasts
}

// GetPlayerMessages returns all messages sent to a specific player.
func (h *TestHelper) GetPlayerMessages(playerID string) []pkgnet.Message {
	return h.playerMsgs[playerID]
}

// ClearMessages clears all captured messages.
func (h *TestHelper) ClearMessages() {
	h.broadcasts = make([]pkgnet.Message, 0)
	h.playerMsgs = make(map[string][]pkgnet.Message)
}

// BuildStateSync builds current state sync using the builder.
func (h *TestHelper) BuildStateSync() *pkgnet.StateSync {
	return h.builder.BuildStateSync()
}

// BuildAvailable builds available actions for a player.
func (h *TestHelper) BuildAvailable(diceType rng.DiceType) *pkgnet.Available {
	h.builder.SetDiceType(diceType)
	if player := h.hsm.GetTurnPlayer(); player != nil {
		return h.builder.BuildAvailable(player)
	}
	return nil
}

// BuildActionSyncBatch builds all ActionSync from current turn entries.
func (h *TestHelper) BuildActionSyncBatch() []*pkgnet.ActionSync {
	return h.builder.BuildActionSyncBatch()
}

// GetGame returns the game instance for direct manipulation.
func (h *TestHelper) GetGame() *engine.Game {
	return h.game
}

// GetHSM returns the HSM instance for direct manipulation.
func (h *TestHelper) GetHSM() *hsm.HSM {
	return h.hsm
}

// GetBuilder returns the builder instance.
func (h *TestHelper) GetBuilder() *internalnet.Builder {
	return h.builder
}

// SetDiceType sets the current player's dice type.
func (h *TestHelper) SetDiceType(diceType rng.DiceType) {
	h.builder.SetDiceType(diceType)
}

// SimulateStateTransition simulates entering a new state and captures broadcasts.
func (h *TestHelper) SimulateStateTransition(globalState, turnState string) error {
	// Build and capture state sync
	stateSync := h.builder.BuildStateSync()
	stateSync.GlobalState = globalState
	stateSync.TurnState = turnState

	return h.CaptureBroadcast(pkgnet.OpStateSync, stateSync)
}

// SimulateActionExecution simulates action execution and captures broadcasts.
func (h *TestHelper) SimulateActionExecution(actionType, target string, delta int, source string) error {
	actionSync := &pkgnet.ActionSync{
		ActionType: actionType,
		Target:     target,
		Delta:      delta,
		Source:     source,
		Metadata:   make(map[string]interface{}),
	}
	return h.CaptureBroadcast(pkgnet.OpActionSync, actionSync)
}

// WaitForDecision simulates sending a decision request to a player.
func (h *TestHelper) WaitForDecision(playerID string, decision *pkgnet.Decision) error {
	return h.CaptureSendToPlayer(playerID, pkgnet.OpDecisionRequest, decision)
}

// AdvanceTurn advances the game to the next player's turn.
func (h *TestHelper) AdvanceTurn() {
	h.game.NextTurn()
	h.hsm.SetTurnPlayer(h.game.GetCurrentPlayer())
}

// Now returns current time for test assertions.
func (h *TestHelper) Now() time.Time {
	return time.Now()
}