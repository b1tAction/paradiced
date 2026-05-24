// Package event provides EventBus and Context for the Paradiced game.
// This package depends on internal/core for Player type.
package event

import (
	"sort"
	"sync"

	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/gamelog"
	"github.com/b1tAction/paradiced/pkg/id"
)

// Subscription represents subscription information.
type Subscription struct {
	ID         id.SubscriptionID  `json:"id"`
	OwnerID    string             `json:"owner_id"`
	SourceID   string             `json:"source_id"`
	SourceType string             `json:"source_type"`
	Priority   int                `json:"priority"`
	Decision   *Decision          `json:"decision"`
	Phase      constants.Phase    `json:"phase"`
}

// EventBus is the event bus.
type EventBus struct {
	subscriptions map[constants.Phase][]*Subscription
	mutex         sync.RWMutex
	GameID        string             `json:"game_id"`
	DebugLog      *gamelog.GameLogger `json:"-"` // Debug logger (nil-safe)
}

// NewEventBus creates a new event bus.
func NewEventBus(gameID string) *EventBus {
	return &EventBus{
		subscriptions: make(map[constants.Phase][]*Subscription),
		GameID:        gameID,
	}
}

// Subscribe subscribes to a Phase.
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

	bus.DebugLog.Debug("EventBus.Subscribe", "phase", phase, "owner_id", ownerID.UUID(), "source_id", sourceID, "source_type", sourceType, "priority", decision.Priority)
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
	bus.DebugLog.Debug("EventBus.UnsubscribeBySource", "source_id", sourceID, "count", count)
	return count
}

// UnsubscribeByOwner unsubscribes all subscriptions by player ID.
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

// GetSubscriptions returns all subscriptions for a Phase (for debugging).
func (bus *EventBus) GetSubscriptions(phase constants.Phase) []*Subscription {
	bus.mutex.RLock()
	defer bus.mutex.RUnlock()
	return bus.subscriptions[phase]
}

// Publish publishes a Phase event.
// Errors from handlers are collected in ctx.Errors.
// Use ctx.HasError() or ctx.FirstError() to check for errors after Publish.
func (bus *EventBus) Publish(phase constants.Phase, ownerID string, ctx *Context) []*Decision {
	bus.mutex.RLock()
	subs := bus.subscriptions[phase]
	bus.mutex.RUnlock()

	ownerSubs := make([]*Subscription, 0)
	for _, sub := range subs {
		if sub.OwnerID == ownerID {
			ownerSubs = append(ownerSubs, sub)
		}
	}

	sort.Slice(ownerSubs, func(i, j int) bool {
		return ownerSubs[i].Priority > ownerSubs[j].Priority
	})

	decisions := make([]*Decision, 0)
	for _, sub := range ownerSubs {
		if sub.Decision.ShouldAsk() {
			decisions = append(decisions, sub.Decision)
		} else {
			// Execute stores errors in ctx.Errors
			sub.Decision.Execute(sub.Decision.Default, ctx)
		}
	}

	bus.DebugLog.Debug("EventBus.Publish", "phase", phase, "owner_id", ownerID, "total_subs", len(subs), "owner_subs", len(ownerSubs), "decisions", len(decisions), "auto_exec", len(ownerSubs)-len(decisions))
	return decisions
}

// GetSubscriptionCount returns total subscription count.
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