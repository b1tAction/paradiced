// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/b1tAction/paradiced/internal/engine/minigame"
	"github.com/b1tAction/paradiced/pkg/constants"
	pkgnet "github.com/b1tAction/paradiced/pkg/net"
	"github.com/heroiclabs/nakama-common/runtime"
)

// minigameResultPayload is the JSON payload received from Colyseus
// when an online mini-game completes.
type minigameResultPayload struct {
	MatchID  string `json:"match_id"`   // Nakama match ID
	RoomID   string `json:"room_id"`    // Colyseus room ID
	GameType string `json:"game_type"`  // Mini-game type (e.g. "dilemma_race")
	Secret   string `json:"secret"`     // Shared secret for authentication
	Rankings []struct {
		PlayerID string                 `json:"player_id"` // Nakama player UUID
		Rank     int                    `json:"rank"`      // Player ranking (1-4)
		GameData map[string]interface{} `json:"game_data,omitempty"` // Mini-game data for ranking rendering
	} `json:"rankings"`
}

// RegisterMiniGameResultRPC registers the RPC endpoint that receives
// mini-game results from external game services (e.g. Colyseus).
// When Colyseus finishes a game, it calls this endpoint with rankings.
// The RPC handler then sends a MatchSignal to the target match,
// which stores the rankings for MatchLoop to consume.
//
// The shared secret is validated to prevent unauthorized result submissions.
func RegisterMiniGameResultRPC(initializer runtime.Initializer, secret string) error {
	return initializer.RegisterRpc("minigame_result_callback", func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if payload == "" {
			logger.Error("MiniGame result RPC: empty payload")
			return "", fmt.Errorf("empty payload")
		}

		var req minigameResultPayload
		if err := json.Unmarshal([]byte(payload), &req); err != nil {
			logger.Error("MiniGame result RPC: failed to parse payload: %v", err)
			return "", fmt.Errorf("invalid payload: %w", err)
		}

		// Validate shared secret
		if req.Secret != secret {
			logger.Error("MiniGame result RPC: invalid secret from match_id=%s, room_id=%s", req.MatchID, req.RoomID)
			return "", fmt.Errorf("invalid secret")
		}

		// Validate match ID
		if req.MatchID == "" {
			logger.Error("MiniGame result RPC: missing match_id")
			return "", fmt.Errorf("missing match_id")
		}

		// Validate rankings
		if len(req.Rankings) == 0 {
			logger.Error("MiniGame result RPC: empty rankings from match_id=%s", req.MatchID)
			return "", fmt.Errorf("empty rankings")
		}

		logger.Info("MiniGame result RPC: received results for match_id=%s, room_id=%s, game_type=%s, rankings_count=%d",
			req.MatchID, req.RoomID, req.GameType, len(req.Rankings))

		// Build signal data to send to the running match
		signalData, err := json.Marshal(map[string]interface{}{
			"type":      "minigame_result",
			"match_id":  req.MatchID,
			"room_id":   req.RoomID,
			"game_type": req.GameType,
			"rankings":  req.Rankings,
		})
		if err != nil {
			logger.Error("MiniGame result RPC: failed to marshal signal data: %v", err)
			return "", fmt.Errorf("failed to marshal signal: %w", err)
		}

		// Send signal to the target match via Nakama API
		result, err := nk.MatchSignal(ctx, req.MatchID, string(signalData))
		if err != nil {
			logger.Error("MiniGame result RPC: failed to send MatchSignal to match_id=%s: %v", req.MatchID, err)
			return "", fmt.Errorf("failed to send match signal: %w", err)
		}

		logger.Info("MiniGame result RPC: signal sent successfully, result=%s", result)

		// Return success response
		response, _ := json.Marshal(map[string]interface{}{
			"success":   true,
			"match_id":  req.MatchID,
			"room_id":   req.RoomID,
		})
		return string(response), nil
	})
}

// RegisterMiniGameRequestRPC registers the minigame_request RPC endpoint.
// This is a debug/testing endpoint that supports two modes:
//   - "judge": offline rank calculation — directly computes rankings from
//     submitted game_data without requiring a running match.
//   - "online": trigger an online mini-game session — sends a MatchSignal
//     to the target match to force a RoundMiniGame state transition.
func RegisterMiniGameRequestRPC(initializer runtime.Initializer) error {
	return initializer.RegisterRpc("minigame_request", func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if payload == "" {
			logger.Error("MiniGameRequest RPC: empty payload")
			return "", fmt.Errorf("empty payload")
		}

		var req pkgnet.MiniGameRequestPayload
		if err := json.Unmarshal([]byte(payload), &req); err != nil {
			logger.Error("MiniGameRequest RPC: failed to parse payload: %v", err)
			return "", fmt.Errorf("invalid payload: %w", err)
		}

		switch req.Mode {
		case "judge":
			return handleJudgeMode(ctx, logger, req)
		case "online":
			return handleOnlineMode(ctx, logger, nk, req)
		default:
			logger.Error("MiniGameRequest RPC: unknown mode: %s", req.Mode)
			return "", fmt.Errorf("unknown mode: %s (expected 'judge' or 'online')", req.Mode)
		}
	})
}

// handleJudgeMode computes offline mini-game rankings from submitted game_data.
// This is purely a calculation — no match state is affected.
func handleJudgeMode(ctx context.Context, logger runtime.Logger, req pkgnet.MiniGameRequestPayload) (string, error) {
	if len(req.Submissions) == 0 {
		logger.Error("MiniGameRequest RPC (judge): no submissions provided")
		return "", fmt.Errorf("no submissions provided for judge mode")
	}

	gameType := constants.MiniGameType(req.GameType)

	// Build submissions map for RankCalculator: player_id -> game_data
	submissions := make(map[string]map[string]any, len(req.Submissions))
	displayNames := make(map[string]string, len(req.Submissions))
	for _, s := range req.Submissions {
		submissions[s.PlayerID] = s.GameData
		displayNames[s.PlayerID] = s.DisplayName
	}

	// Calculate rankings
	calc := minigame.NewDefaultRankCalculator()
	ranks := calc.Calculate(gameType, submissions)

	// Build ranking entries sorted by rank
	rankingEntries := make([]pkgnet.MiniGameJudgeRankingEntry, 0, len(ranks))
	for playerID, rank := range ranks {
		entry := pkgnet.MiniGameJudgeRankingEntry{
			PlayerID:    playerID,
			DisplayName: displayNames[playerID],
			Rank:        rank,
			GameData:    submissions[playerID],
		}
		rankingEntries = append(rankingEntries, entry)
	}
	sort.Slice(rankingEntries, func(i, j int) bool {
		return rankingEntries[i].Rank < rankingEntries[j].Rank
	})

	// Build dice assignments: rank -> dice type
	// Matches the server's standard dice assignment logic:
	//   rank 1 = gold, rank 2 = silver, rank 3 = copper, rank 4 = wood
	diceAssignments := make(map[string]string, len(ranks))
	for playerID, rank := range ranks {
		switch rank {
		case 1:
			diceAssignments[playerID] = "gold"
		case 2:
			diceAssignments[playerID] = "silver"
		case 3:
			diceAssignments[playerID] = "copper"
		default:
			diceAssignments[playerID] = "wood"
		}
	}

	response := pkgnet.MiniGameJudgeResponse{
		Rankings:        rankingEntries,
		DiceAssignments: diceAssignments,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		logger.Error("MiniGameRequest RPC (judge): failed to marshal response: %v", err)
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	logger.Info("MiniGameRequest RPC (judge): calculated rankings for game_type=%s, players=%d",
		req.GameType, len(req.Submissions))

	return string(responseBytes), nil
}

// handleOnlineMode triggers an online mini-game session by sending a
// MatchSignal to the target match. The signal forces the match to
// transition to RoundMiniGame state with the specified game_type.
func handleOnlineMode(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, req pkgnet.MiniGameRequestPayload) (string, error) {
	if req.MatchID == "" {
		logger.Error("MiniGameRequest RPC (online): missing match_id")
		return "", fmt.Errorf("missing match_id for online mode")
	}

	// Build signal data to trigger mini-game start
	signalData, err := json.Marshal(map[string]interface{}{
		"type":      "trigger_minigame",
		"match_id":  req.MatchID,
		"game_type": req.GameType,
	})
	if err != nil {
		logger.Error("MiniGameRequest RPC (online): failed to marshal signal data: %v", err)
		return "", fmt.Errorf("failed to marshal signal: %w", err)
	}

	// Send signal to the target match via Nakama API
	result, err := nk.MatchSignal(ctx, req.MatchID, string(signalData))
	if err != nil {
		logger.Error("MiniGameRequest RPC (online): failed to send MatchSignal to match_id=%s: %v", req.MatchID, err)
		return "", fmt.Errorf("failed to send match signal: %w", err)
	}

	logger.Info("MiniGameRequest RPC (online): trigger signal sent to match_id=%s, game_type=%s, result=%s",
		req.MatchID, req.GameType, result)

	response := pkgnet.MiniGameOnlineResponse{
		Success: true,
		MatchID: req.MatchID,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		logger.Error("MiniGameRequest RPC (online): failed to marshal response: %v", err)
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(responseBytes), nil
}