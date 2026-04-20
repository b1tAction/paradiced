// Package main provides Nakama plugin entry point for Paradiced.
// This module is loaded by Nakama server at startup as a shared object (.so).
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"

	"github.com/b1tAction/paradiced/internal/nakama"
	"github.com/heroiclabs/nakama-common/runtime"
)

// InitModule is the entry point called by Nakama when the module is loaded.
// This function registers all match handlers and hooks.
func InitModule(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, initializer runtime.Initializer) error {
	// Register RPC for creating authoritative rooms (used by pdcli host)
	// Clients must call this RPC instead of socket.CreateMatch to create authoritative matches
	err := initializer.RegisterRpc("create_authoritative_room", func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		// Parse optional params from payload (JSON format)
		params := make(map[string]interface{})
		if payload != "" {
			// Could parse max_players, map_length etc. from payload if needed
			logger.Debug("Create room RPC payload: %s", payload)
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
	log.Printf("[Paradiced] Module initialized - RPC, match handler and matchmaker callback registered")

	return nil
}