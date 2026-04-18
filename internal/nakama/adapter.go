// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/b1tAction/paradiced/pkg/id"
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
}

// NewNakamaMatchHandlerAdapter creates a new adapter with optional config.
func NewNakamaMatchHandlerAdapter() *NakamaMatchHandlerAdapter {
	return &NakamaMatchHandlerAdapter{}
}

// MatchInit implements runtime.Match.MatchInit.
// Called when match is created. Returns initial state.
func (a *NakamaMatchHandlerAdapter) MatchInit(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, params map[string]interface{}) (interface{}, int, string) {
	a.logger = logger
	a.db = db
	a.nk = nk

	// Generate match ID and seed
	matchID := id.NewGameID().UUID()
	seed := time.Now().UnixNano()
	if seedParam, ok := params["seed"].(int64); ok {
		seed = seedParam
	}

	// Extract config from params
	maxPlayers := 4
	mapLength := 100
	if mp, ok := params["max_players"].(int); ok {
		maxPlayers = mp
	}
	if mp, ok := params["max_players"].(float64); ok {
		maxPlayers = int(mp)
	}
	if ml, ok := params["map_length"].(int); ok {
		mapLength = ml
	}
	if ml, ok := params["map_length"].(float64); ok {
		mapLength = int(ml)
	}

	// Create handler
	a.handler = NewNakamaMatchHandler(matchID, seed, maxPlayers, mapLength)

	// Build match label (JSON format for match queries)
	label := map[string]interface{}{
		"max_players": maxPlayers,
		"game":        "paradiced",
	}
	labelJSON, _ := json.Marshal(label)

	logger.Info("Paradiced match initialized: match_id=%s, seed=%d, max_players=%d, map_length=%d", matchID, seed, maxPlayers, mapLength)

	// Return state, tick rate (10 = 100ms per tick), label
	return a.handler, 10, string(labelJSON)
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

	// Check if match is already running (4 players joined and game started)
	if handler.hsm != nil && handler.hsm.IsRunning() {
		// Allow rejoin for disconnected players
		if handler.players[presence.GetUserId()] != nil && handler.disconnected[presence.GetUserId()] {
			return state, true, "rejoin allowed"
		}
		// Reject new players after game started
		if handler.players[presence.GetUserId()] == nil {
			return state, false, "game already in progress"
		}
	}

	// Allow join
	logger.Debug("Player join attempt accepted: user_id=%s", presence.GetUserId())
	return state, true, ""
}

// MatchJoin implements runtime.Match.MatchJoin.
// Called when players successfully join.
func (a *NakamaMatchHandlerAdapter) MatchJoin(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, presences []runtime.Presence) interface{} {
	handler := state.(*NakamaMatchHandler)

	// Build presence list for dispatcher
	presenceList := make([]runtime.Presence, 0)
	for _, id := range handler.playerList {
		if !handler.disconnected[id] {
			rd, ok := handler.dispatcher.(*RealDispatcherAdapter)
			if ok && rd != nil {
				p := rd.GetPresence(id)
				if p != nil {
					presenceList = append(presenceList, p)
				}
			}
		}
	}
	// Add new presences
	presenceList = append(presenceList, presences...)

	// Update dispatcher
	realDispatcher := NewRealDispatcherAdapter(ctx, dispatcher, presenceList)
	handler.WithDispatcher(realDispatcher)

	// Handle each joining presence
	for _, p := range presences {
		userID := p.GetUserId()
		// Extract faction from metadata
		metadata := make(map[string]string)
		// Note: Nakama join metadata can be passed via presence properties
		// For now, we use empty metadata and default faction
		logger.Debug("Player joined: user_id=%s", userID)
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
		logger.Debug("Player left: user_id=%s", p.GetUserId())
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
			rd, ok := handler.dispatcher.(*RealDispatcherAdapter)
			if ok && rd != nil {
				p := rd.GetPresence(id)
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
		opCode := msg.GetOpCode()
		logger.Debug("Received message: user_id=%s, op_code=%d, data_len=%d", userID, opCode, len(data))
		handler.HandleMessage(userID, data)
	}

	// Run match loop (delta time not used, tick rate controls timing)
	handler.MatchLoop(0)

	// If HSM stopped, return nil to end match
	if handler.hsm != nil && !handler.hsm.IsRunning() {
		logger.Info("Match ended: match_id=%s", handler.matchID)
		return nil
	}

	return state
}

// MatchTerminate implements runtime.Match.MatchTerminate.
// Called when match is shutting down.
func (a *NakamaMatchHandlerAdapter) MatchTerminate(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, graceSeconds int) interface{} {
	handler := state.(*NakamaMatchHandler)
	logger.Info("Match terminating: match_id=%s, grace_seconds=%d", handler.matchID, graceSeconds)
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
			a.logger.Debug("Received pause signal")
		case "resume":
			// Resume match logic if needed
			a.logger.Debug("Received resume signal")
		}
	}

	return state, "" // Return state and empty response
}

// matchLabel returns JSON label for match queries.
type matchLabel struct {
	MaxPlayers int    `json:"max_players"`
	Game       string `json:"game"`
	Status     string `json:"status"`
}