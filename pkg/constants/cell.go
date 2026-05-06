// Package constants provides unified enum type definitions.
package constants

// CellType defines map cell type.
type CellType string

// CellType constants - snake_case values for JSON serialization.
const (
	CellTypeNone     CellType = "none"
	CellTypeNormal   CellType = "normal"    // Normal cell
	CellTypeFragile  CellType = "fragile"   // Fragile cell (fall through)
	CellTypeFog      CellType = "fog"       // Fog cell (poison area)
	CellTypeCheckpoint CellType = "checkpoint" // Checkpoint (respawn point)
	CellTypeBoss     CellType = "boss"      // Boss cell (end point)
	CellTypeEvent    CellType = "event"     // Event cell (triggers bound event)
)

// IsValid checks if CellType is valid (one of the known cell types).
func (ct CellType) IsValid() bool {
	switch ct {
	case CellTypeNormal, CellTypeFragile, CellTypeFog,
		CellTypeCheckpoint, CellTypeBoss, CellTypeEvent:
		return true
	default:
		return false
	}
}

// IsSpecial checks if the cell has special behavior.
func (ct CellType) IsSpecial() bool {
	return ct == CellTypeFragile || ct == CellTypeFog ||
		ct == CellTypeCheckpoint || ct == CellTypeBoss || ct == CellTypeEvent
}

// ParseCellType converts a string to CellType.
// Returns CellTypeNone if the string is not a valid cell type.
// Supports both snake_case ("normal") and PascalCase ("Normal") formats.
func ParseCellType(s string) CellType {
	switch s {
	case "normal", "Normal":
		return CellTypeNormal
	case "fragile", "Fragile":
		return CellTypeFragile
	case "fog", "Fog":
		return CellTypeFog
	case "checkpoint", "Checkpoint":
		return CellTypeCheckpoint
	case "boss", "Boss":
		return CellTypeBoss
	case "event", "Event":
		return CellTypeEvent
	default:
		return CellTypeNone
	}
}

// ========== DrawType ==========

// DrawType defines what type of draw to perform on a cell.
type DrawType string

// DrawType constants.
const (
	DrawTypeNone  DrawType = "none"  // No draw
	DrawTypeEvent DrawType = "event" // Draw event
	DrawTypeItem  DrawType = "item"  // Draw item
	DrawTypeBuff  DrawType = "buff"  // Draw buff
)

// IsValid checks if DrawType is valid.
func (dt DrawType) IsValid() bool {
	switch dt {
	case DrawTypeNone, DrawTypeEvent, DrawTypeItem, DrawTypeBuff:
		return true
	default:
		return false
	}
}

// ParseDrawType converts a string to DrawType.
// Returns DrawTypeNone if the string is not valid.
func ParseDrawType(s string) DrawType {
	switch s {
	case "none", "None":
		return DrawTypeNone
	case "event", "Event":
		return DrawTypeEvent
	case "item", "Item":
		return DrawTypeItem
	case "buff", "Buff":
		return DrawTypeBuff
	default:
		return DrawTypeNone
	}
}