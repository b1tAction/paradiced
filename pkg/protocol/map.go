package protocol

// PathResult defines the interface for movement path results.
// Note: Uses interface{} to avoid importing gamemap package.
type PathResult interface {
	GetTargetIndex() int
	GetPath() []int
}

// MapEngine defines the interface for map operations.
// Used by MoveAction to calculate movement paths.
// Note: Returns PathResult interface, concrete implementations return their types.
type MapEngine interface {
	CalculatePath(startPos int, steps int) (PathResult, error)
}