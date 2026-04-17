package protocol

import (
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
)

// Decision represents a user decision request.
// Imported from pkg/event for interface compatibility.
type Decision interface {
	GetID() id.DecisionID
	GetPrompt() string
	GetOptions() []DecisionOption
	GetDefault() int
	ShouldAsk() bool
	Execute(choice int, ctx interface{})
}

// DecisionOption represents an option in a decision.
type DecisionOption interface {
	GetID() string
	GetLabel() string
	GetEffect() string
}

// EventBus defines the event subscription system interface.
// Used by HSM to interact with EventBus without direct coupling.
type EventBus interface {
	// Publish publishes a Phase event and returns Decisions that need user confirmation.
	Publish(phase constants.Phase, playerID string, ctx interface{}) []Decision

	// Subscribe subscribes to a Phase with a pre-bound Decision.
	Subscribe(phase constants.Phase, ownerID id.PlayerID, sourceID, sourceType string, decision Decision) id.SubscriptionID

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