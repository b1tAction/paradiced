// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

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
		PlayerID string `json:"player_id"` // Nakama player UUID
		Rank     int    `json:"rank"`      // Player ranking (1-4)
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