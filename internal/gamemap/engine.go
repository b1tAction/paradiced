package gamemap

import (
	"encoding/json"
	"errors"
)

// MapCell defines single map cell structure.
type MapCell struct {
	Index     int      `json:"index"`      // Coordinate index (0~N)
	CellType  CellType `json:"cell_type"`  // Cell type
	IsBroken  bool     `json:"is_broken"`  // Whether broken (only for Fragile)
	EventID   string   `json:"event_id"`   // Associated event ID (optional)
	FogActive bool     `json:"fog_active"` // Whether fog activated (only for Fog)
}

// NewMapCell creates a new map cell.
func NewMapCell(index int, cellType CellType) *MapCell {
	return &MapCell{
		Index:     index,
		CellType:  cellType,
		IsBroken:  false,
		EventID:   "",
		FogActive: false,
	}
}

// MapEngine is the map engine, managing the entire map.
type MapEngine struct {
	Cells      []*MapCell `json:"cells"`       // Map cell array
	Length     int        `json:"length"`      // Total map length
	StartIndex int        `json:"start_index"` // Start index (default 0)
	EndIndex   int        `json:"end_index"`   // End index
}

// NewMapEngine creates a new map engine.
func NewMapEngine(length int) *MapEngine {
	if length < 1 {
		length = 1
	}
	cells := make([]*MapCell, length)
	for i := 0; i < length; i++ {
		cells[i] = NewMapCell(i, CellTypeNormal)
	}
	return &MapEngine{
		Cells:      cells,
		Length:     length,
		StartIndex: 0,
		EndIndex:   length - 1,
	}
}

// GenerateLinearMap generates a linear map with specified cell types.
// cellConfigs: map[index]cellType for specifying cell types at specific positions.
func (m *MapEngine) GenerateLinearMap(cellConfigs map[int]CellType) error {
	for i := 0; i < m.Length; i++ {
		cellType := CellTypeNormal
		if ct, ok := cellConfigs[i]; ok {
			if !ct.IsValid() {
				return errors.New("invalid cell type")
			}
			cellType = ct
		}
		m.Cells[i] = NewMapCell(i, cellType)
	}
	return nil
}

// GetCell returns the cell at specified position.
func (m *MapEngine) GetCell(index int) (*MapCell, error) {
	if index < 0 || index >= m.Length {
		return nil, errors.New("index out of bounds")
	}
	return m.Cells[index], nil
}

// SetCellType sets cell type at specified position.
func (m *MapEngine) SetCellType(index int, cellType CellType) error {
	if index < 0 || index >= m.Length {
		return errors.New("index out of bounds")
	}
	if !cellType.IsValid() {
		return errors.New("invalid cell type")
	}
	m.Cells[index].CellType = cellType
	return nil
}

// BreakFragile breaks a fragile cell.
func (m *MapEngine) BreakFragile(index int) error {
	cell, err := m.GetCell(index)
	if err != nil {
		return err
	}
	if cell.CellType != CellTypeFragile {
		return errors.New("cell is not fragile")
	}
	cell.IsBroken = true
	return nil
}

// ActivateFog activates a fog cell.
func (m *MapEngine) ActivateFog(index int) error {
	cell, err := m.GetCell(index)
	if err != nil {
		return err
	}
	if cell.CellType != CellTypeFog {
		return errors.New("cell is not fog")
	}
	cell.FogActive = true
	return nil
}

// PathResult represents movement path calculation result.
type PathResult struct {
	StartIndex     int   `json:"start_index"`      // Start position
	TargetIndex    int   `json:"target_index"`     // Target position (actual arrival)
	OriginalTarget int   `json:"original_target"`  // Original target (dice steps calculation)
	Path           []int `json:"path"`             // List of passed cell indices
	Interrupted    bool  `json:"interrupted"`      // Whether interrupted alias for FellDown, saved for other interrupt behavs
	FellDown       bool  `json:"fell_down"`        // Whether fell (final landing on unbroken fragile)
	BrokenFragiles []int `json:"broken_fragiles"`  // Indices of Fragile cells broken in this move
	ReachedEnd     bool  `json:"reached_end"`      // Whether reached end point
}

// GetTargetIndex returns the target position.
func (r *PathResult) GetTargetIndex() int {
	return r.TargetIndex
}

// GetPath returns the path of visited cells.
func (r *PathResult) GetPath() []int {
	return r.Path
}

// CalculatePath calculates movement path.
// start: start position, steps: dice steps.
// Returns actual movement path, handles Fragile cell logic:
// 1. Passing unbroken Fragile → Fragile breaks, player continues moving
// 2. Final landing on unbroken Fragile → breaks + falls (interrupted)
// 3. Final landing on broken Fragile → cannot reach, stops at previous cell
func (m *MapEngine) CalculatePath(start int, steps int) (*PathResult, error) {
	if start < 0 || start >= m.Length {
		return nil, errors.New("start index out of bounds")
	}
	// if steps < 0 {
	// 	return nil, errors.New("steps cannot be negative")
	// }

	result := &PathResult{
		StartIndex:     start,
		TargetIndex:    start,
		OriginalTarget: start + steps, // Original target (dice steps calculation, may exceed end)
		Path:           []int{start},
		Interrupted:    false,
		FellDown:       false,
		BrokenFragiles: []int{},
		ReachedEnd:     false,
	}

	// Calculate target position (not exceeding map end)
	target := start + steps
	if target >= m.Length {
		target = m.Length - 1
		result.ReachedEnd = true
	}

	// Move cell by cell, record path and activate fog
	for i := start + 1; i <= target; i++ {
		cell, err := m.GetCell(i)
		if err != nil {
			break
		}

		// Check fog area (activate when passing)
		if cell.CellType == CellTypeFog {
			cell.FogActive = true
		}

		// Record path
		result.Path = append(result.Path, i)
	}

	// Check Fragile status at final landing point
	finalCell, err := m.GetCell(target)
	if err != nil {
		result.TargetIndex = start
		return result, nil
	}

	// Handle Fragile cell logic at final landing point
	if finalCell.CellType == CellTypeFragile {
		if !finalCell.IsBroken {
			// Final landing on unbroken Fragile → breaks + falls
			finalCell.IsBroken = true
			result.BrokenFragiles = append(result.BrokenFragiles, target)
			result.TargetIndex = target
			result.Interrupted = true
			result.FellDown = true

			// Check other Fragile cells in path (not final landing), mark as broken
			for i := start + 1; i < target; i++ {
				cell, _ := m.GetCell(i)
				if cell != nil && cell.CellType == CellTypeFragile && !cell.IsBroken {
					cell.IsBroken = true
					result.BrokenFragiles = append(result.BrokenFragiles, i)
				}
			}

			return result, nil
		} else {
			// Final landing on broken Fragile → cannot reach, stop at previous cell
			if target > start {
				result.TargetIndex = target - 1
				// Remove last cell from path (cannot reach)
				result.Path = result.Path[:len(result.Path)-1]
			} else {
				result.TargetIndex = start
			}

			// Check other Fragile cells in path (not final landing), mark as broken
			for i := start + 1; i < target; i++ {
				cell, _ := m.GetCell(i)
				if cell != nil && cell.CellType == CellTypeFragile && !cell.IsBroken {
					cell.IsBroken = true
					result.BrokenFragiles = append(result.BrokenFragiles, i)
				}
			}

			return result, nil
		}
	}

	// Normal case: final landing is not Fragile
	result.TargetIndex = target

	// Check Fragile cells in path (not final landing), mark as broken
	for i := start + 1; i <= target; i++ {
		cell, _ := m.GetCell(i)
		if cell != nil && cell.CellType == CellTypeFragile && !cell.IsBroken {
			cell.IsBroken = true
			result.BrokenFragiles = append(result.BrokenFragiles, i)
		}
	}

	return result, nil
}

// Export exports map data as JSON.
func (m *MapEngine) Export() ([]byte, error) {
	return json.Marshal(m)
}

// Import imports map data from JSON.
func (m *MapEngine) Import(data []byte) error {
	return json.Unmarshal(data, m)
}

// LoadMap loads map from JSON (creates new MapEngine).
func LoadMap(data []byte) (*MapEngine, error) {
	var engine MapEngine
	if err := json.Unmarshal(data, &engine); err != nil {
		return nil, err
	}
	// Validate data validity
	if engine.Length < 1 {
		return nil, errors.New("invalid map length")
	}
	if len(engine.Cells) != engine.Length {
		return nil, errors.New("cells count mismatch")
	}
	return &engine, nil
}

// GetCellsByType returns all cells of specified type.
func (m *MapEngine) GetCellsByType(cellType CellType) []*MapCell {
	var result []*MapCell
	for _, cell := range m.Cells {
		if cell.CellType == cellType {
			result = append(result, cell)
		}
	}
	return result
}

// GetLastCheckpoint returns the last checkpoint before specified position.
func (m *MapEngine) GetLastCheckpoint(position int) int {
	lastCheckpoint := 0
	for i := 0; i < position && i < m.Length; i++ {
		if m.Cells[i].CellType == CellTypeCheckpoint {
			lastCheckpoint = i
		}
	}
	return lastCheckpoint
}

// Clone clones the map engine.
func (m *MapEngine) Clone() *MapEngine {
	cells := make([]*MapCell, m.Length)
	for i, cell := range m.Cells {
		cells[i] = &MapCell{
			Index:     cell.Index,
			CellType:  cell.CellType,
			IsBroken:  cell.IsBroken,
			EventID:   cell.EventID,
			FogActive: cell.FogActive,
		}
	}
	return &MapEngine{
		Cells:      cells,
		Length:     m.Length,
		StartIndex: m.StartIndex,
		EndIndex:   m.EndIndex,
	}
}