package protocol

import "github.com/b1tAction/paradiced/pkg/constants"

// PathResult defines the interface for movement path results.
// Note: Uses interface{} to avoid importing gamemap package.
type PathResult interface {
	GetTargetIndex() int
	GetPath() []int
}

// Cell defines the interface for map cell.
// Used by HSM to access cell properties.
type Cell interface {
	GetPosition() int
	GetType() constants.CellType
	IsFogActive() bool
}

// MapEngine defines the interface for map operations.
// Used by MoveAction to calculate movement paths.
// Extended with methods needed by HSM.
type MapEngine interface {
	// CalculatePath calculates movement path from start position with given steps.
	CalculatePath(startPos int, steps int) (PathResult, error)

	// GetLength returns the total map length.
	GetLength() int

	// GetCell returns the cell at specified position.
	GetCell(pos int) (Cell, error)

	// GetLastCheckpoint returns the last checkpoint before specified position.
	GetLastCheckpoint(pos int) int

	// SetCellType sets cell type at specified position.
	SetCellType(pos int, cellType constants.CellType) error

	// ActivateFog activates a fog cell at specified position.
	ActivateFog(pos int) error

	// IsFogActivated checks if fog is activated at specified position.
	IsFogActivated(pos int) bool
}