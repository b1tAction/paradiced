// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/heroiclabs/nakama-common/runtime"
)

// NakamaMatchHandlerAdapter implements runtime.Match interface.
// This adapter wraps NakamaMatchHandler to work with Nakama's match system.
//
// Usage in InitModule:
//
//	func InitModule(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, initializer runtime.Initializer) error {
//	    return initializer.RegisterMatch("paradiced_match", func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule) (runtime.Match, error) {
//	        return &NakamaMatchHandlerAdapter{}, nil
//	    })
//	}
type NakamaMatchHandlerAdapter struct {
	handler   *NakamaMatchHandler
	logger    runtime.Logger
	db        *sql.DB
	nk        runtime.NakamaModule
	matchID   string
	seed      int64
}

// NewNakamaMatchHandlerAdapter creates a new adapter.
func NewNakamaMatchHandlerAdapter(matchID string, seed int64) *NakamaMatchHandlerAdapter {
	return &NakamaMatchHandlerAdapter{
		matchID: matchID,
		seed:    seed,
	}
}

// MatchInit implements runtime.Match.MatchInit.
// Called when match is created. Returns initial state.
func (a *NakamaMatchHandlerAdapter) MatchInit(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, params map[string]interface{}) (interface{}, int, string) {
	a.logger = logger
	a.db = db
	a.nk = nk

	// Extract config from params
	maxPlayers := 4
	mapLength := 100
	if mp, ok := params["max_players"].(int); ok {
		maxPlayers = mp
	}
	if ml, ok := params["map_length"].(int); ok {
		mapLength = ml
	}

	// Create handler
	a.handler = NewNakamaMatchHandler(a.matchID, a.seed, maxPlayers, mapLength)

	// Return state, tick rate (10 = 100ms), empty label
	return a.handler, 10, ""
}

// MatchJoinAttempt implements runtime.Match.MatchJoinAttempt.
// Called when a player attempts to join.
func (a *NakamaMatchHandlerAdapter) MatchJoinAttempt(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, presence runtime.Presence, metadata map[string]string) (interface{}, bool, string) {
	handler := state.(*NakamaMatchHandler)

	// Check if match is full
	if len(handler.playerList) >= handler.maxPlayers {
		// Reject join
		return state, false, "match is full"
	}

	// Allow join
	return state, true, ""
}

// MatchJoin implements runtime.Match.MatchJoin.
// Called when players successfully join.
func (a *NakamaMatchHandlerAdapter) MatchJoin(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, presences []runtime.Presence) interface{} {
	handler := state.(*NakamaMatchHandler)

	// Update dispatcher
	realDispatcher := NewRealDispatcherAdapter(ctx, dispatcher, presences)
	handler.WithDispatcher(realDispatcher)

	// Handle each joining presence
	for _, p := range presences {
		userID := p.GetUserId()
		// Extract metadata from presence if available
		metadata := make(map[string]string)
		// Note: Nakama presence metadata handling may differ
		handler.HandlePresenceJoin(userID, metadata)
		realDispatcher.UpdatePresence(p)
	}

	return state
}

// MatchLeave implements runtime.Match.MatchLeave.
// Called when players leave/disconnect.
func (a *NakamaMatchHandlerAdapter) MatchLeave(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, presences []runtime.Presence) interface{} {
	handler := state.(*NakamaMatchHandler)

	// Update dispatcher
	realDispatcher, ok := handler.dispatcher.(*RealDispatcherAdapter)
	if ok {
		for _, p := range presences {
			realDispatcher.RemovePresence(p.GetUserId())
		}
	}

	// Handle each leaving presence
	for _, p := range presences {
		handler.HandlePresenceLeave(p.GetUserId())
	}

	return state
}

// MatchLoop implements runtime.Match.MatchLoop.
// Called every tick (based on tick rate from MatchInit).
func (a *NakamaMatchHandlerAdapter) MatchLoop(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, messages []runtime.MatchData) interface{} {
	handler := state.(*NakamaMatchHandler)

	// Update dispatcher with current presences
	presences := make([]runtime.Presence, 0)
	for _, id := range handler.playerList {
		if !handler.disconnected[id] {
			realDispatcher, ok := handler.dispatcher.(*RealDispatcherAdapter)
			if ok && realDispatcher != nil {
				p := realDispatcher.GetPresence(id)
				if p != nil {
					presences = append(presences, p)
				}
			}
		}
	}

	realDispatcher := NewRealDispatcherAdapter(ctx, dispatcher, presences)
	handler.WithDispatcher(realDispatcher)

	// Process incoming messages
	for _, msg := range messages {
		userID := msg.GetUserId() // MatchData extends Presence
		data := msg.GetData()
		handler.HandleMessage(userID, data)
	}

	// Run match loop
	handler.MatchLoop(0) // delta time not used, tick rate controls timing

	return state
}

// MatchTerminate implements runtime.Match.MatchTerminate.
// Called when match is shutting down.
func (a *NakamaMatchHandlerAdapter) MatchTerminate(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, graceSeconds int) interface{} {
	handler := state.(*NakamaMatchHandler)
	handler.MatchStop()
	return nil // Return nil to indicate match ended
}

// MatchSignal implements runtime.Match.MatchSignal.
// Called when a signal is received (e.g., from server admin).
func (a *NakamaMatchHandlerAdapter) MatchSignal(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, data string) (interface{}, string) {
	// Handle signal data
	var signal struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(data), &signal); err == nil {
		switch signal.Type {
		case "pause":
			// Pause match logic if needed
		case "resume":
			// Resume match logic if needed
		}
	}

	return state, "" // Return state and empty response
}