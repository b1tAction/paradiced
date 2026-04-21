// Package net provides network message protocol definitions for client-server communication.
package net

// Builder interface defines methods for constructing protocol sync messages.
// Implemented by internal/net.Builder, used by HSM states to build client messages.
// This interface avoids import cycle between internal/engine/hsm and internal/net.
type Builder interface {
	// BuildStateSync builds a complete state sync message.
	BuildStateSync() *StateSync

	// BuildTurnSync builds a turn sync message with log entries.
	BuildTurnSync() *TurnSync

	// BuildAvailable builds available actions for current player.
	BuildAvailable() *Available

	// BuildMapInfo builds map info from MapEngine data.
	BuildMapInfo() *MapInfo

	// SetDiceType sets the current player's dice type (string format: "gold", "silver", "copper", "wood").
	SetDiceType(diceType string)
}