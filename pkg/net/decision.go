package net

// Decision represents a decision request sent to client.
// The client must respond with a choice within the timeout.
type Decision struct {
	// ID is the unique decision identifier for matching responses.
	ID string `json:"id"`

	// Prompt is the decision prompt text to display.
	Prompt string `json:"prompt"`

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
// Empty structure - server checks player's faction and charge status.
type UseSkill struct{}

// UserChoice represents a decision choice response.
type UserChoice struct {
	// DecisionID matches the Decision.ID being responded to.
	DecisionID string `json:"decision_id"`

	// Choice is the selected option index (0-based).
	Choice int `json:"choice"`
}

// MiniGameResult represents mini-game ranking submission.
type MiniGameResult struct {
	// Rank is the player's ranking (1-4 for 4 players).
	Rank int `json:"rank"`
}