package event

import (
	"sort"
	"sync"

	"github.com/b1tAction/fated/pkg/constants"
	"github.com/b1tAction/fated/pkg/id"
)

// Subscription represents subscription information.
type Subscription struct {
	ID         id.SubscriptionID  `json:"id"`          // Subscription ID, used for unsubscribe
	OwnerID    string             `json:"owner_id"`    // Player ID (UUID string)
	SourceID   string             `json:"source_id"`   // Buff/Item ID (UUID string)
	SourceType string             `json:"source_type"` // Source type "buff" / "item"
	Priority   int                `json:"priority"`    // Execution priority (higher executes first)
	Decision   *Decision          `json:"decision"`    // Pre-bound Decision
	Phase      constants.Phase    `json:"phase"`       // Subscribed Phase
}

// EventBus is the event bus.
// Each game instance has one EventBus, managing Buff/Item subscriptions and triggers.
type EventBus struct {
	subscriptions map[constants.Phase][]*Subscription
	mutex         sync.RWMutex
	GameID        string `json:"game_id"` // Owning game instance ID (UUID string)
}

// NewEventBus creates a new event bus.
func NewEventBus(gameID string) *EventBus {
	return &EventBus{
		subscriptions: make(map[constants.Phase][]*Subscription),
		GameID:        gameID,
	}
}

// Subscribe subscribes to a Phase.
// Returns subscription ID for unsubscribing.
func (bus *EventBus) Subscribe(phase constants.Phase, ownerID id.PlayerID, sourceID, sourceType string, decision *Decision) id.SubscriptionID {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	subID := id.NewSubscriptionID()
	sub := &Subscription{
		ID:         subID,
		OwnerID:    ownerID.UUID(),
		SourceID:   sourceID,
		SourceType: sourceType,
		Priority:   decision.Priority,
		Decision:   decision.WithSource(sourceID, sourceType),
		Phase:      phase,
	}

	bus.subscriptions[phase] = append(bus.subscriptions[phase], sub)
	return subID
}

// Unsubscribe unsubscribes by ID.
func (bus *EventBus) Unsubscribe(subID id.SubscriptionID) bool {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	for phase, subs := range bus.subscriptions {
		for i, sub := range subs {
			if sub.ID.Equal(subID.ID) {
				bus.subscriptions[phase] = append(subs[:i], subs[i+1:]...)
				return true
			}
		}
	}
	return false
}

// UnsubscribeBySource unsubscribes all subscriptions by source ID.
// Used when removing Buff/Item to batch unsubscribe.
func (bus *EventBus) UnsubscribeBySource(sourceID string) int {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	count := 0
	for phase, subs := range bus.subscriptions {
		newSubs := make([]*Subscription, 0, len(subs))
		for _, sub := range subs {
			if sub.SourceID == sourceID {
				count++
			} else {
				newSubs = append(newSubs, sub)
			}
		}
		bus.subscriptions[phase] = newSubs
	}
	return count
}

// UnsubscribeByOwner unsubscribes all subscriptions by player ID (UUID string).
// Used when player leaves game to batch unsubscribe.
func (bus *EventBus) UnsubscribeByOwner(ownerID string) int {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	count := 0
	for phase, subs := range bus.subscriptions {
		newSubs := make([]*Subscription, 0, len(subs))
		for _, sub := range subs {
			if sub.OwnerID == ownerID {
				count++
			} else {
				newSubs = append(newSubs, sub)
			}
		}
		bus.subscriptions[phase] = newSubs
	}
	return count
}

// Publish publishes a Phase event.
// Returns list of Decisions that need user confirmation.
func (bus *EventBus) Publish(phase constants.Phase, ownerID string, ctx *Context) []*Decision {
	bus.mutex.RLock()
	subs := bus.subscriptions[phase]
	bus.mutex.RUnlock()

	// Filter subscriptions for this player
	ownerSubs := make([]*Subscription, 0)
	for _, sub := range subs {
		if sub.OwnerID == ownerID {
			ownerSubs = append(ownerSubs, sub)
		}
	}

	// Sort by Priority (higher priority executes first)
	sort.Slice(ownerSubs, func(i, j int) bool {
		return ownerSubs[i].Priority > ownerSubs[j].Priority
	})

	// Execute Decisions that don't need confirmation, collect those that do
	decisions := make([]*Decision, 0)
	for _, sub := range ownerSubs {
		if sub.Decision.ShouldAsk() {
			// Needs user confirmation, add to waiting list
			decisions = append(decisions, sub.Decision)
		} else {
			// No confirmation needed, execute default option directly
			sub.Decision.Execute(sub.Decision.Default, ctx)
		}
	}

	return decisions
}

// GetSubscriptions returns all subscriptions for a Phase (for debugging).
func (bus *EventBus) GetSubscriptions(phase constants.Phase) []*Subscription {
	bus.mutex.RLock()
	defer bus.mutex.RUnlock()
	return bus.subscriptions[phase]
}

// GetSubscriptionCount returns total subscription count (for debugging).
func (bus *EventBus) GetSubscriptionCount() int {
	bus.mutex.RLock()
	defer bus.mutex.RUnlock()
	count := 0
	for _, subs := range bus.subscriptions {
		count += len(subs)
	}
	return count
}

// Clear clears all subscriptions.
func (bus *EventBus) Clear() {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()
	bus.subscriptions = make(map[constants.Phase][]*Subscription)
}