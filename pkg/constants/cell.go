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
)

// IsValid checks if CellType is valid.
func (ct CellType) IsValid() bool {
	return ct != CellTypeNone && ct != ""
}

// IsSpecial checks if the cell has special behavior.
func (ct CellType) IsSpecial() bool {
	return ct == CellTypeFragile || ct == CellTypeFog ||
		ct == CellTypeCheckpoint || ct == CellTypeBoss
}

// ParseCellType converts a string to CellType.
// Returns CellTypeNone if the string is not a valid cell type.
func ParseCellType(s string) CellType {
	switch s {
	case "normal":
		return CellTypeNormal
	case "fragile":
		return CellTypeFragile
	case "fog":
		return CellTypeFog
	case "checkpoint":
		return CellTypeCheckpoint
	case "boss":
		return CellTypeBoss
	default:
		return CellTypeNone
	}
}