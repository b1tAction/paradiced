// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/b1tAction/paradiced/internal/engine/hsm"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
	"github.com/b1tAction/paradiced/pkg/util"
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
	handler      *NakamaMatchHandler
	logger       runtime.Logger
	db           *sql.DB
	nk           runtime.NakamaModule
	joinMetadata map[string]*util.Metadata
}

// NewNakamaMatchHandlerAdapter creates a new adapter with optional config.
func NewNakamaMatchHandlerAdapter() *NakamaMatchHandlerAdapter {
	return &NakamaMatchHandlerAdapter{
		joinMetadata: make(map[string]*util.Metadata),
	}
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
	if mp, ok := params["maxPlayers"].(int); ok {
		maxPlayers = mp
	}
	if mp, ok := params["maxPlayers"].(float64); ok {
		maxPlayers = int(mp)
	}
	if ml, ok := params["map_length"].(int); ok {
		mapLength = ml
	}
	if ml, ok := params["map_length"].(float64); ok {
		mapLength = int(ml)
	}
	lobbyName := ""
	if name, ok := params["lobby_name"].(string); ok {
		lobbyName = name
	}
	if name, ok := params["lobbyName"].(string); ok && lobbyName == "" {
		lobbyName = name
	}

	// Create handler
	a.handler = NewNakamaMatchHandler(matchID, seed, maxPlayers, mapLength)
	a.handler.WithLogger(logger)
	a.handler.lobbyName = lobbyName

	// Build match label (JSON format for match queries)
	label := matchLabel{
		MaxPlayers:      maxPlayers,
		Game:            "paradiced",
		Status:          "waiting",
		HostDisplayName: "",
		LobbyName:       lobbyName,
	}
	labelJSON, _ := json.Marshal(label)

	logger.Info("Paradiced match initialized: match_id=%s, seed=%d, max_players=%d, map_length=%d", matchID, seed, maxPlayers, mapLength)

	// Check if this match was created by matchmaker (entries are passed in params)
	if entriesRaw, ok := params["entries"]; ok {
		if entries, ok := entriesRaw.([]runtime.MatchmakerEntry); ok {
			logger.Info("Matchmaker-created match: adding %d players", len(entries))
			// Add players from matchmaker entries before initializing game
			for _, entry := range entries {
				userID := entry.GetPresence().GetUserId()
				// Extract faction from entry properties (if available)
				faction := getFactionFromProperties(entry.GetProperties())
				// Extract display_name from entry properties, fallback to userID
				displayName := getDisplayNameFromProperties(entry.GetProperties(), userID)
				a.handler.addPlayer(userID, faction, displayName)
				// Mark player as disconnected until they actually join
				a.handler.disconnected[userID] = true
				logger.Debug("Added player from matchmaker: user_id=%s, display_name=%s (marked as disconnected)", userID, displayName)
			}
		}
	}

	// For matchmaker-created matches we only pre-add players here.
	// Actual match init is deferred until MatchJoin so initial protocol messages
	// are delivered through a live dispatcher with real presences.

	// Return state, tick rate (10 = 100ms per tick), label
	return a.handler, 10, string(labelJSON)
}

// MatchJoinAttempt implements runtime.Match.MatchJoinAttempt.
// Called when a player attempts to join.
func (a *NakamaMatchHandlerAdapter) MatchJoinAttempt(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, presence runtime.Presence, metadata map[string]string) (interface{}, bool, string) {
	handler := state.(*NakamaMatchHandler)
	userID := presence.GetUserId()

	// Log join attempt for debugging
	logger.Info("MatchJoinAttempt: user_id=%s, hsm_nil=%v, player_list_len=%d", userID, handler.hsm == nil, len(handler.playerList))

	// Existing player (connected or reconnecting) is always allowed.
	if handler.players[userID] != nil {
		a.joinMetadata[userID] = newJoinMetadata(metadata)
		return state, true, ""
	}

	// Check if match is full
	if len(handler.playerList) >= handler.maxPlayers {
		return state, false, "match is full"
	}

	// Check if match is already running (but allow join during WaitingForHost state)
	if handler.hsm != nil && handler.hsm.IsRunning() {
		// Allow join during WaitingForHost state (manual start mode)
		if handler.hsm.GetGlobalStateID() != hsm.StateWaitingForHost {
			return state, false, "game already in progress"
		}
		// WaitingForHost state - allow more players to join
	}

	a.joinMetadata[userID] = newJoinMetadata(metadata)

	// Allow join
	logger.Debug("Player join attempt accepted: user_id=%s", userID)
	return state, true, ""
}

// MatchJoin implements runtime.Match.MatchJoin.
// Called when players successfully join.
func (a *NakamaMatchHandlerAdapter) MatchJoin(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, presences []runtime.Presence) interface{} {
	handler := state.(*NakamaMatchHandler)

	// Log join info
	logger.Info("MatchJoin called: user_ids=%v, hsm_nil=%v", func() []string {
		ids := make([]string, len(presences))
		for i, p := range presences {
			ids[i] = p.GetUserId()
		}
		return ids
	}(), handler.hsm == nil)

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
		metadata := a.joinMetadata[userID]
		logger.Debug("Player joined: user_id=%s", userID)
		handler.HandlePresenceJoin(userID, metadata)
		realDispatcher.UpdatePresence(p)
		delete(a.joinMetadata, userID)
	}

	// For matchmaker-created matches, players were pre-added during MatchInit and
	// HandlePresenceJoin only marks them connected. Initialize here once we have
	// a live dispatcher/presence so startup sync/action prompts are deliverable.
	if handler.hsm == nil && len(handler.playerList) > 0 {
		logger.Info("Initializing deferred match game after player join...")
		if err := handler.MatchInit(); err != nil {
			logger.Error("Failed to initialize deferred match game: %v", err)
		} else {
			logger.Info("Deferred match game initialized successfully")
		}
	}

	// Update match label with current status and host display name
	dispatcher.MatchLabelUpdate(a.buildCurrentLabel(handler))

	return state
}

func newJoinMetadata(raw map[string]string) *util.Metadata {
	metadata := util.NewMetadata()
	for k, v := range raw {
		metadata.SetString(k, v)
	}
	return metadata
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

	// Check if all players disconnected - return nil to end match
	allDisconnected := true
	for _, id := range handler.playerList {
		if !handler.disconnected[id] {
			allDisconnected = false
			break
		}
	}

	if allDisconnected {
		logger.Info("All players disconnected, ending match: match_id=%s", handler.matchID)
		return nil // Return nil to terminate match in Nakama
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
		if err := handler.HandleMessageWithOp(userID, opCode, data); err != nil {
			logger.Error("HandleMessageWithOp error: user_id=%s, op_code=%d, error=%v", userID, opCode, err)
		}
	}

	// Run match loop (delta time not used, tick rate controls timing)
	handler.MatchLoop(0)

	// Update match label to reflect current game status
	dispatcher.MatchLabelUpdate(a.buildCurrentLabel(handler))

	// If HSM stopped, return nil to end match
	if handler.hsm != nil && !handler.hsm.IsRunning() {
		logger.Info("Match ended: match_id=%s", handler.matchID)
		return nil
	}

	return state
}

// storageDeleter is a minimal interface for deleting storage records.
// Extracted from runtime.NakamaModule to enable testing without mocking the full interface.
type storageDeleter interface {
	StorageDelete(ctx context.Context, deletes []*runtime.StorageDelete) error
}

// MatchTerminate implements runtime.Match.MatchTerminate.
// Called when match is shutting down.
// Cleans up Paradiced-specific storage data for all players before stopping the match.
func (a *NakamaMatchHandlerAdapter) MatchTerminate(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, graceSeconds int) interface{} {
	handler := state.(*NakamaMatchHandler)
	logger.Info("Match terminating: match_id=%s, grace_seconds=%d", handler.matchID, graceSeconds)

	// Clean up Paradiced-specific storage records for all players
	cleanupPlayerStorage(ctx, nk, handler, logger)

	handler.MatchStop()
	return nil // Return nil to indicate match ended
}

// cleanupPlayerStorage deletes Paradiced-specific storage records for all players in the match.
func cleanupPlayerStorage(ctx context.Context, deleter storageDeleter, handler *NakamaMatchHandler, logger runtime.Logger) {
	if deleter == nil || len(handler.players) == 0 {
		return
	}

	storageKeys := []string{"paradiced_match_result", "paradiced_stats"}
	for userID := range handler.players {
		deletes := make([]*runtime.StorageDelete, 0, len(storageKeys))
		for _, key := range storageKeys {
			deletes = append(deletes, &runtime.StorageDelete{
				Collection: "paradiced",
				Key:        key,
				UserID:     userID,
			})
		}
		if err := deleter.StorageDelete(ctx, deletes); err != nil {
			logger.Error("Failed to delete storage for user: user_id=%s, error=%v", userID, err)
		}
	}
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
	MaxPlayers      int    `json:"max_players"`
	Game            string `json:"game"`
	Status          string `json:"status"`
	HostDisplayName string `json:"host_display_name"`
	LobbyName       string `json:"lobby_name"`
}

// buildCurrentLabel builds the current match label JSON from handler state.
// Used for dynamic label updates via MatchDispatcher.MatchLabelUpdate.
func (a *NakamaMatchHandlerAdapter) buildCurrentLabel(handler *NakamaMatchHandler) string {
	status := "waiting"
	if handler.hsm != nil && handler.hsm.IsRunning() {
		switch handler.hsm.GetGlobalStateID() {
		case hsm.StateWaitingForHost:
			status = "waiting"
		case hsm.StateGameOver:
			status = "finished"
		default:
			status = "in_progress"
		}
	}

	hostDN := ""
	if h, ok := handler.players[handler.hostUserID]; ok && h.Metadata != nil {
		hostDN = h.Metadata.GetStringOrDefault("display_name", handler.hostUserID)
	}

	l := matchLabel{
		MaxPlayers:      handler.maxPlayers,
		Game:            "paradiced",
		Status:          status,
		HostDisplayName: hostDN,
		LobbyName:       handler.lobbyName,
	}
	b, _ := json.Marshal(l)
	return string(b)
}

// getFactionFromProperties extracts faction from matchmaker entry properties.
// Returns default faction (QingLong) if not specified.
func getFactionFromProperties(props map[string]interface{}) constants.Faction {
	if props == nil {
		return constants.FactionQingLong
	}

	// Try string property first
	if factionStr, ok := props["faction"].(string); ok {
		parsed := constants.ParseFaction(factionStr)
		if parsed.IsValid() {
			return parsed
		}
		return constants.FactionQingLong
	}

	// Try float64 (JSON number) - should not happen for string but handle just in case
	if factionNum, ok := props["faction"].(float64); ok {
		// This would be unusual, but handle it by returning default
		_ = factionNum
	}

	return constants.FactionQingLong
}

// getDisplayNameFromProperties extracts display_name from matchmaker entry properties.
// Returns fallback userID if not specified.
func getDisplayNameFromProperties(props map[string]interface{}, fallbackUserID string) string {
	if props == nil {
		return fallbackUserID
	}

	// Try string property
	if displayName, ok := props["display_name"].(string); ok && displayName != "" {
		return displayName
	}

	return fallbackUserID
}
