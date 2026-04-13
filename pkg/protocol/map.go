package protocol

// PathResult defines the interface for movement path results.
type PathResult interface {
	GetTargetIndex() int
	GetPath() []int
}

// MapEngine defines the interface for map operations.
// Used by MoveAction to calculate movement paths.
type MapEngine interface {
	CalculatePath(startPos int, steps int) (PathResult, error)
}