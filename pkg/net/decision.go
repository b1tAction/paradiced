// Package net provides network message protocol definitions for client-server communication.
package net

// Decision represents a decision request sent to client.
// The client must respond with a choice within the timeout.
type Decision struct {
	// ID is the unique decision identifier for matching responses.
	ID string `json:"id"`

	// Prompt is the decision prompt text to display.
	Prompt string `json:"prompt"`

	// Context is the source identifier (buff ID, item ID, event ID).
	// Example: "Buff_Divine", "Item_AnyDoor", "Event_Exchange"
	Context string `json:"context"`

	// Options contains available choices.
	Options []Option `json:"options"`

	// Timeout is the maximum wait time in seconds.
	// If no response received, the default option is applied.
	Timeout int `json:"timeout"`

	// Default is the index of the default option for timeout handling.
	Default int `json:"default"`
}

// Option represents a decision choice option.
type Option struct {
	// ID is the option identifier.
	ID string `json:"id"`

	// Label is the display text for this option.
	Label string `json:"label"`

	// Effect is the effect preview text for client UI.
	// Example: "HP+1", "LP-1", "获得神眷Buff"
	Effect string `json:"effect,omitempty"`
}

// ========== Client -> Server Request Structures ==========

// RollDice represents a dice roll request.
// Empty structure - server calculates result based on player's dice type.
type RollDice struct{}

// UseItem represents an item usage request.
type UseItem struct {
	// ItemID is the item instance ID to use.
	ItemID string `json:"item_id"`

	// TargetID is optional target player ID for targeted items.
	TargetID string `json:"target_id,omitempty"`
}

// UseSkill represents a faction skill activation request.
// BaiHu faction requires TargetID to specify the player targeted by 劫运(RobLuck).
type UseSkill struct {
	// TargetID is optional target player ID for targeted faction skills (BaiHu 劫运).
	TargetID string `json:"target_id,omitempty"`
}

// UserChoice represents a decision choice response.
type UserChoice struct {
	// DecisionID matches the Decision.ID being responded to.
	DecisionID string `json:"decision_id"`

	// Choice is the selected option index (0-based).
	Choice int `json:"choice"`
}

// MiniGameResultSubmit represents mini-game ranking submission from client.
// Note: In authoritative server mode, server should calculate rankings.
// This is kept for potential client-side mini-game implementations.
type MiniGameResultSubmit struct {
	// Rankings contains player rankings submitted by mini-game system.
	Rankings []RankingEntry `json:"rankings"`
}