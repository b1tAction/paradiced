package gamemap

// CellType defines map cell types.
type CellType int

const (
	CellTypeNormal CellType = iota // Normal cell
	CellTypeFragile                // Fragile cell
	CellTypeFog                    // Fog cell
	CellTypeCheckpoint             // Checkpoint
	CellTypeBoss                   // Boss cell (end point)
)

// String returns the string representation of cell type.
func (ct CellType) String() string {
	names := map[CellType]string{
		CellTypeNormal:     "Normal",
		CellTypeFragile:    "Fragile",
		CellTypeFog:        "Fog",
		CellTypeCheckpoint: "Checkpoint",
		CellTypeBoss:       "Boss",
	}
	if name, ok := names[ct]; ok {
		return name
	}
	return "Unknown"
}

// IsValid checks if cell type is valid.
func (ct CellType) IsValid() bool {
	return ct >= CellTypeNormal && ct <= CellTypeBoss
}