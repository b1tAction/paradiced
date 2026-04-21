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