// Package main provides Nakama plugin entry point for Paradiced.
// This module is loaded by Nakama server at startup as a shared object (.so).
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/b1tAction/paradiced/internal/nakama"
	"github.com/heroiclabs/nakama-common/runtime"
)

// Device ID prefix used by ParaDiced web client for username-based auth.
const deviceIDPrefix = "paradiced_"

// Stale account threshold: accounts inactive for more than 7 days will be cleaned up.
const staleThresholdDays = 7

// InitModule is the entry point called by Nakama when the module is loaded.
// This function registers all match handlers and hooks.
func InitModule(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, initializer runtime.Initializer) error {
	// Register RPC for creating authoritative rooms (used by pdcli host)
	// Clients must call this RPC instead of socket.CreateMatch to create authoritative matches
	err := initializer.RegisterRpc("create_authoritative_room", func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		// Parse optional params from payload (JSON format)
		params := make(map[string]interface{})
		if payload != "" {
			if err := json.Unmarshal([]byte(payload), &params); err != nil {
				logger.Error("Failed to parse create room payload: %v", err)
			} else {
				logger.Debug("Create room RPC payload: %s", payload)
			}
		}

		// Server-side MatchCreate triggers MatchInit lifecycle
		// Use the registered handler name "paradiced_match_*" - Nakama will match the wildcard
		matchID, err := nk.MatchCreate(ctx, "paradiced_match_*", params)
		if err != nil {
			logger.Error("Failed to create authoritative room: %v", err)
			return "", err
		}

		logger.Info("Authoritative room created via RPC: match_id=%s", matchID)

		// JSON-encode the matchID for proper unmarshaling by nakama-go client
		response, err := json.Marshal(matchID)
		if err != nil {
			logger.Error("Failed to JSON encode matchID: %v", err)
			return "", err
		}
		return string(response), nil
	})

	if err != nil {
		logger.Error("Failed to register create_authoritative_room RPC: %v", err)
		return err
	}

	// Register RPC for cleaning up stale device-authenticated accounts.
	// This can be called via HTTP API or scheduled with external cron.
	// Payload (optional JSON): { "threshold_days": 7 }
	err = initializer.RegisterRpc("cleanup_stale_accounts", func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		thresholdDays := staleThresholdDays

		// Parse optional threshold override from payload
		if payload != "" {
			var params struct {
				ThresholdDays int `json:"threshold_days"`
			}
			if err := json.Unmarshal([]byte(payload), &params); err == nil && params.ThresholdDays > 0 {
				thresholdDays = params.ThresholdDays
			}
		}

		logger.Info("Starting stale account cleanup, threshold_days=%d", thresholdDays)

		thresholdTime := time.Now().UTC().AddDate(0, 0, -thresholdDays)

		// Query user IDs of device-authenticated accounts with paradiced_ prefix
		// that have not been updated since the threshold time.
		// Nakama stores device links in the user_device table.
		query := `
			SELECT DISTINCT u.id::TEXT
			FROM users u
			INNER JOIN user_device d ON u.id = d.user_id
			WHERE d.id LIKE $1
			AND u.update_time < $2
		`

		rows, err := db.QueryContext(ctx, query, deviceIDPrefix+"%", thresholdTime)
		if err != nil {
			logger.Error("Failed to query stale accounts: %v", err)
			return "", fmt.Errorf("query failed: %w", err)
		}
		defer rows.Close()

		var staleUserIDs []string
		for rows.Next() {
			var userID string
			if err := rows.Scan(&userID); err != nil {
				logger.Error("Failed to scan stale account row: %v", err)
				continue
			}
			staleUserIDs = append(staleUserIDs, userID)
		}

		if len(staleUserIDs) == 0 {
			logger.Info("No stale accounts found")
			result := map[string]interface{}{
				"deleted_count":  0,
				"threshold_days": thresholdDays,
			}
			response, _ := json.Marshal(result)
			return string(response), nil
		}

		logger.Info("Found %d stale accounts to delete", len(staleUserIDs))

		// Delete each stale account using Nakama API
		deletedCount := 0
		for _, userID := range staleUserIDs {
			if err := nk.AccountDeleteId(ctx, userID, false); err != nil {
				logger.Error("Failed to delete stale account %s: %v", userID, err)
				continue
			}
			deletedCount++
			logger.Debug("Deleted stale account: %s", userID)
		}

		logger.Info("Stale account cleanup completed: deleted=%d, total=%d", deletedCount, len(staleUserIDs))

		result := map[string]interface{}{
			"deleted_count":  deletedCount,
			"total_stale":    len(staleUserIDs),
			"threshold_days": thresholdDays,
		}
		response, _ := json.Marshal(result)
		return string(response), nil
	})

	if err != nil {
		logger.Error("Failed to register cleanup_stale_accounts RPC: %v", err)
		return err
	}

	// Register matchmaker matched callback to create authoritative matches
	err = initializer.RegisterMatchmakerMatched(func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, entries []runtime.MatchmakerEntry) (string, error) {
		// Create authoritative match with matched players
		// Use the registered handler name "paradiced_match_*"
		matchID, err := nk.MatchCreate(ctx, "paradiced_match_*", map[string]interface{}{
			"entries": entries,
		})
		if err != nil {
			logger.Error("Failed to create authoritative match: %v", err)
			return "", err
		}
		logger.Info("Authoritative match created via matchmaker: match_id=%s, players=%d", matchID, len(entries))
		return matchID, nil
	})

	if err != nil {
		logger.Error("Failed to register matchmaker matched callback: %v", err)
		return err
	}

	// Register Paradiced match handler with wildcard pattern
	// This allows creating matches with unique names (e.g., paradiced_match_abc123)
	// while using the same handler. Each unique name triggers a fresh MatchInit call.
	err = initializer.RegisterMatch("paradiced_match_*", func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule) (runtime.Match, error) {
		// Create a new match handler adapter
		return nakama.NewNakamaMatchHandlerAdapter(), nil
	})

	if err != nil {
		logger.Error("Failed to register paradiced match handler: %v", err)
		return err
	}

	logger.Info("Paradiced match handler registered successfully")
	log.Printf("[Paradiced] Module initialized - RPC, match handler, matchmaker callback, and cleanup RPC registered")

	return nil
}
