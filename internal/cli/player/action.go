// Package player provides interactive player functionality for CLI.
package player

// ActionType represents the type of action a player can take.
type ActionType int

const (
	// ActionRollDice indicates the player wants to roll dice.
	ActionRollDice ActionType = 0

	// ActionUseItem indicates the player wants to use an item.
	ActionUseItem ActionType = 1

	// ActionViewStatus indicates the player wants to view game status.
	// This is a local action, not sent to server.
	ActionViewStatus ActionType = 2

	// ActionNone indicates no action (invalid input).
	ActionNone ActionType = -1
)

// String returns the string representation of ActionType.
func (a ActionType) String() string {
	switch a {
	case ActionRollDice:
		return "roll_dice"
	case ActionUseItem:
		return "use_item"
	case ActionViewStatus:
		return "view_status"
	default:
		return "none"
	}
}

// PlayerAction represents a player's chosen action.
type PlayerAction struct {
	// Type is the action type.
	Type ActionType

	// ItemID is the item ID to use (only for ActionUseItem).
	ItemID string

	// TargetID is the target player ID (optional for some items).
	TargetID string
}

// NewRollDiceAction creates a roll dice action.
func NewRollDiceAction() PlayerAction {
	return PlayerAction{
		Type: ActionRollDice,
	}
}

// NewUseItemAction creates a use item action.
func NewUseItemAction(itemID string) PlayerAction {
	return PlayerAction{
		Type:   ActionUseItem,
		ItemID: itemID,
	}
}

// NewUseItemWithTargetAction creates a use item action with target.
func NewUseItemWithTargetAction(itemID, targetID string) PlayerAction {
	return PlayerAction{
		Type:     ActionUseItem,
		ItemID:   itemID,
		TargetID: targetID,
	}
}

// NewViewStatusAction creates a view status action.
func NewViewStatusAction() PlayerAction {
	return PlayerAction{
		Type: ActionViewStatus,
	}
}

// IsValid returns true if the action is valid (not ActionNone).
func (a PlayerAction) IsValid() bool {
	return a.Type != ActionNone
}

// IsServerAction returns true if the action should be sent to server.
func (a PlayerAction) IsServerAction() bool {
	return a.Type == ActionRollDice || a.Type == ActionUseItem
}