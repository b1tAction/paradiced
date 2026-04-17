package hsm

import (
	"github.com/b1tAction/paradiced/internal/engine"
	"github.com/b1tAction/paradiced/internal/gamemap"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/event"
	"github.com/b1tAction/paradiced/pkg/gamelog"
	"github.com/b1tAction/paradiced/pkg/id"
	"github.com/b1tAction/paradiced/pkg/protocol"
)

// ========== EventBus Adapter ==========

// EventBusAdapter isolates HSM from pkg/event package details.
// This interface allows HSM to interact with EventBus without direct coupling
// to the concrete implementation, enabling easier testing and future changes.
type EventBusAdapter interface {
	// Publish publishes a Phase event and returns Decisions that need user confirmation.
	Publish(phase constants.Phase, playerID string, ctx *event.Context) []*event.Decision

	// Subscribe subscribes to a Phase with a pre-bound Decision.
	Subscribe(phase constants.Phase, ownerID id.PlayerID, sourceID, sourceType string, decision *event.Decision) id.SubscriptionID

	// Unsubscribe removes a subscription by ID.
	Unsubscribe(subID id.SubscriptionID) bool

	// UnsubscribeBySource removes all subscriptions by source ID (e.g., Buff/Item removal).
	UnsubscribeBySource(sourceID string) int

	// UnsubscribeByOwner removes all subscriptions by player ID (e.g., player leaving).
	UnsubscribeByOwner(ownerID string) int

	// GetSubscriptionCount returns total subscription count (for debugging).
	GetSubscriptionCount() int

	// Clear removes all subscriptions.
	Clear()
}

// EventBusWrapper wraps event.EventBus to implement EventBusAdapter.
type EventBusWrapper struct {
	bus *event.EventBus
}

// NewEventBusWrapper creates a new EventBusWrapper.
func NewEventBusWrapper(bus *event.EventBus) EventBusAdapter {
	return &EventBusWrapper{bus: bus}
}

// Publish publishes a Phase event.
func (w *EventBusWrapper) Publish(phase constants.Phase, playerID string, ctx *event.Context) []*event.Decision {
	return w.bus.Publish(phase, playerID, ctx)
}

// Subscribe subscribes to a Phase.
func (w *EventBusWrapper) Subscribe(phase constants.Phase, ownerID id.PlayerID, sourceID, sourceType string, decision *event.Decision) id.SubscriptionID {
	return w.bus.Subscribe(phase, ownerID, sourceID, sourceType, decision)
}

// Unsubscribe removes a subscription by ID.
func (w *EventBusWrapper) Unsubscribe(subID id.SubscriptionID) bool {
	return w.bus.Unsubscribe(subID)
}

// UnsubscribeBySource removes all subscriptions by source ID.
func (w *EventBusWrapper) UnsubscribeBySource(sourceID string) int {
	return w.bus.UnsubscribeBySource(sourceID)
}

// UnsubscribeByOwner removes all subscriptions by player ID (UUID string).
func (w *EventBusWrapper) UnsubscribeByOwner(ownerID string) int {
	return w.bus.UnsubscribeByOwner(ownerID)
}

// GetSubscriptionCount returns total subscription count.
func (w *EventBusWrapper) GetSubscriptionCount() int {
	return w.bus.GetSubscriptionCount()
}

// Clear removes all subscriptions.
func (w *EventBusWrapper) Clear() {
	w.bus.Clear()
}

// ========== Game Adapter (protocol.Game) ==========

// GameWrapper wraps engine.Game to implement protocol.Game interface.
// Used by ActionContext to access game state.
type GameWrapper struct {
	game *engine.Game
}

// NewGameWrapper creates a new GameWrapper.
func NewGameWrapper(game *engine.Game) protocol.Game {
	return &GameWrapper{game: game}
}

// GetCurrentPlayer returns the current player as interface{}.
func (w *GameWrapper) GetCurrentPlayer() interface{} {
	return w.game.GetCurrentPlayer()
}

// GetPlayer returns a player by ID as interface{}.
func (w *GameWrapper) GetPlayer(playerID id.PlayerID) interface{} {
	return w.game.GetPlayer(playerID)
}

// GetPlayers returns all players as []interface{}.
func (w *GameWrapper) GetPlayers() []interface{} {
	players := w.game.GetPlayers()
	result := make([]interface{}, len(players))
	for i, p := range players {
		result[i] = p
	}
	return result
}

// GetGameLog returns the global game log.
func (w *GameWrapper) GetGameLog() *gamelog.GameLog {
	return w.game.GetGameLog()
}

// ========== MapEngine Adapter ==========

// MapEngineAdapter isolates HSM from internal/gamemap package.
// This interface allows HSM to interact with MapEngine without direct coupling,
// enabling easier testing and potential future changes to map implementation.
type MapEngineAdapter interface {
	// GetLength returns the total map length.
	GetLength() int

	// GetCell returns the cell at specified position.
	GetCell(pos int) (*gamemap.MapCell, error)

	// CalculatePath calculates movement path from start position with given steps.
	CalculatePath(startPos int, steps int) (*gamemap.PathResult, error)

	// GetLastCheckpoint returns the last checkpoint before specified position.
	GetLastCheckpoint(pos int) int

	// SetCellType sets cell type at specified position.
	SetCellType(pos int, cellType gamemap.CellType) error

	// ActivateFog activates a fog cell at specified position.
	ActivateFog(pos int) error

	// IsFogActivated checks if fog is activated at specified position.
	// Returns false if cell is not a fog cell or fog is not activated.
	IsFogActivated(pos int) bool
}

// MapEngineWrapper wraps gamemap.MapEngine to implement MapEngineAdapter.
type MapEngineWrapper struct {
	engine *gamemap.MapEngine
}

// NewMapEngineWrapper creates a new MapEngineWrapper.
func NewMapEngineWrapper(engine *gamemap.MapEngine) MapEngineAdapter {
	return &MapEngineWrapper{engine: engine}
}

// GetLength returns the total map length.
func (w *MapEngineWrapper) GetLength() int {
	return w.engine.Length
}

// GetCell returns the cell at specified position.
func (w *MapEngineWrapper) GetCell(pos int) (*gamemap.MapCell, error) {
	return w.engine.GetCell(pos)
}

// CalculatePath calculates movement path.
func (w *MapEngineWrapper) CalculatePath(startPos int, steps int) (*gamemap.PathResult, error) {
	return w.engine.CalculatePath(startPos, steps)
}

// GetLastCheckpoint returns the last checkpoint before specified position.
func (w *MapEngineWrapper) GetLastCheckpoint(pos int) int {
	return w.engine.GetLastCheckpoint(pos)
}

// SetCellType sets cell type at specified position.
func (w *MapEngineWrapper) SetCellType(pos int, cellType gamemap.CellType) error {
	return w.engine.SetCellType(pos, cellType)
}

// ActivateFog activates a fog cell.
func (w *MapEngineWrapper) ActivateFog(pos int) error {
	return w.engine.ActivateFog(pos)
}

// IsFogActivated checks if fog is activated at specified position.
func (w *MapEngineWrapper) IsFogActivated(pos int) bool {
	cell, err := w.engine.GetCell(pos)
	if err != nil {
		return false
	}
	return cell.CellType == gamemap.CellTypeFog && cell.FogActive
}

// PathResultWrapper wraps gamemap.PathResult to implement protocol.PathResult.
type PathResultWrapper struct {
	result *gamemap.PathResult
}

// GetTargetIndex returns the target position.
func (w *PathResultWrapper) GetTargetIndex() int {
	return w.result.TargetIndex
}

// GetPath returns the path of visited cells.
func (w *PathResultWrapper) GetPath() []int {
	return w.result.Path
}

// CellWrapper wraps gamemap.MapCell to implement protocol.Cell.
type CellWrapper struct {
	cell *gamemap.MapCell
}

// GetPosition returns the cell position.
func (w *CellWrapper) GetPosition() int {
	return w.cell.Index
}

// GetType returns the cell type.
func (w *CellWrapper) GetType() constants.CellType {
	return constants.CellType(w.cell.CellType.String())
}

// IsFogActive returns whether fog is active.
func (w *CellWrapper) IsFogActive() bool {
	return w.cell.FogActive
}

// ========== Protocol MapEngine Adapter ==========

// ProtocolMapEngineWrapper wraps MapEngineAdapter to implement protocol.MapEngine.
// Used by ActionContext which expects protocol.MapEngine interface.
type ProtocolMapEngineWrapper struct {
	adapter MapEngineAdapter
}

// NewProtocolMapEngineWrapper creates a wrapper that implements protocol.MapEngine.
func NewProtocolMapEngineWrapper(adapter MapEngineAdapter) protocol.MapEngine {
	return &ProtocolMapEngineWrapper{adapter: adapter}
}

// CalculatePath returns protocol.PathResult interface.
func (w *ProtocolMapEngineWrapper) CalculatePath(startPos int, steps int) (protocol.PathResult, error) {
	result, err := w.adapter.CalculatePath(startPos, steps)
	if err != nil {
		return nil, err
	}
	return &PathResultWrapper{result: result}, nil
}

// GetLength returns the total map length.
func (w *ProtocolMapEngineWrapper) GetLength() int {
	return w.adapter.GetLength()
}

// GetCell returns the cell at specified position.
func (w *ProtocolMapEngineWrapper) GetCell(pos int) (protocol.Cell, error) {
	cell, err := w.adapter.GetCell(pos)
	if err != nil {
		return nil, err
	}
	return &CellWrapper{cell: cell}, nil
}

// GetLastCheckpoint returns the last checkpoint before specified position.
func (w *ProtocolMapEngineWrapper) GetLastCheckpoint(pos int) int {
	return w.adapter.GetLastCheckpoint(pos)
}

// SetCellType sets cell type at specified position.
func (w *ProtocolMapEngineWrapper) SetCellType(pos int, cellType constants.CellType) error {
	// Convert constants.CellType to gamemap.CellType
	gamemapCellType := gamemapCellTypeFromConstants(cellType)
	return w.adapter.SetCellType(pos, gamemapCellType)
}

// ActivateFog activates a fog cell at specified position.
func (w *ProtocolMapEngineWrapper) ActivateFog(pos int) error {
	return w.adapter.ActivateFog(pos)
}

// IsFogActivated checks if fog is activated at specified position.
func (w *ProtocolMapEngineWrapper) IsFogActivated(pos int) bool {
	return w.adapter.IsFogActivated(pos)
}

// gamemapCellTypeFromConstants converts constants.CellType to gamemap.CellType.
func gamemapCellTypeFromConstants(ct constants.CellType) gamemap.CellType {
	switch ct {
	case constants.CellTypeNormal:
		return gamemap.CellTypeNormal
	case constants.CellTypeFragile:
		return gamemap.CellTypeFragile
	case constants.CellTypeFog:
		return gamemap.CellTypeFog
	case constants.CellTypeCheckpoint:
		return gamemap.CellTypeCheckpoint
	case constants.CellTypeBoss:
		return gamemap.CellTypeBoss
	default:
		return gamemap.CellTypeNormal
	}
}
