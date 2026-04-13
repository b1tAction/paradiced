package hsm

import (
	"github.com/b1tAction/Fated/internal/gamemap"
	"github.com/b1tAction/Fated/pkg/event"
)

// EventBusAdapter isolates HSM from pkg/event package details.
// This interface allows HSM to interact with EventBus without direct coupling
// to the concrete implementation, enabling easier testing and future changes.
type EventBusAdapter interface {
	// Publish publishes a Phase event and returns Decisions that need user confirmation.
	Publish(phase event.Phase, playerID string, ctx *event.Context) []*event.Decision

	// Subscribe subscribes to a Phase with a pre-bound Decision.
	Subscribe(phase event.Phase, ownerID, sourceID, sourceType string, decision *event.Decision) string

	// Unsubscribe removes a subscription by ID.
	Unsubscribe(subID string) bool

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
func (w *EventBusWrapper) Publish(phase event.Phase, playerID string, ctx *event.Context) []*event.Decision {
	return w.bus.Publish(phase, playerID, ctx)
}

// Subscribe subscribes to a Phase.
func (w *EventBusWrapper) Subscribe(phase event.Phase, ownerID, sourceID, sourceType string, decision *event.Decision) string {
	return w.bus.Subscribe(phase, ownerID, sourceID, sourceType, decision)
}

// Unsubscribe removes a subscription by ID.
func (w *EventBusWrapper) Unsubscribe(subID string) bool {
	return w.bus.Unsubscribe(subID)
}

// UnsubscribeBySource removes all subscriptions by source ID.
func (w *EventBusWrapper) UnsubscribeBySource(sourceID string) int {
	return w.bus.UnsubscribeBySource(sourceID)
}

// UnsubscribeByOwner removes all subscriptions by player ID.
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