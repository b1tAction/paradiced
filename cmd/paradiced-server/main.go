// Package main provides Nakama plugin entry point for Paradiced.
// This module is loaded by Nakama server at startup as a shared object (.so).
package main

import (
	"context"
	"database/sql"
	"log"

	"github.com/b1tAction/paradiced/internal/nakama"
	"github.com/heroiclabs/nakama-common/runtime"
)

// InitModule is the entry point called by Nakama when the module is loaded.
// This function registers all match handlers and hooks.
func InitModule(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, initializer runtime.Initializer) error {
	// Register Paradiced match handler
	err := initializer.RegisterMatch("paradiced_match", func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule) (runtime.Match, error) {
		// Create a new match handler adapter
		return nakama.NewNakamaMatchHandlerAdapter(), nil
	})

	if err != nil {
		logger.Error("Failed to register paradiced match handler: %v", err)
		return err
	}

	logger.Info("Paradiced match handler registered successfully")
	log.Printf("[Paradiced] Module initialized - match handler registered")

	return nil
}