package game

import (
	"encoding/json"
	"errors"
)

// MapCell 定义单个地图格子的结构
type MapCell struct {
	Index     int      `json:"index"`     // 坐标序号（0~N）
	CellType  CellType `json:"cell_type"` // 格子类型
	IsBroken  bool     `json:"is_broken"` // 是否已被踩碎（仅对 Fragile 有效）
	EventID   string   `json:"event_id"`  // 格子关联的事件ID（可选）
	FogActive bool     `json:"fog_active"` // 迷雾是否已激活（仅对 Fog 有效）
}

// NewMapCell 创建新的地图格子
func NewMapCell(index int, cellType CellType) *MapCell {
	return &MapCell{
		Index:     index,
		CellType:  cellType,
		IsBroken:  false,
		EventID:   "",
		FogActive: false,
	}
}

// MapEngine 地图引擎，管理整个地图
type MapEngine struct {
	Cells      []*MapCell `json:"cells"`      // 地图格子数组
	Length     int        `json:"length"`     // 地图总长度
	StartIndex int        `json:"start_index"` // 起点索引（默认0）
	EndIndex   int        `json:"end_index"`   // 终点索引
}

// NewMapEngine 创建新的地图引擎
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

// GenerateLinearMap 生成线性地图，可指定特定格子的类型
// cellConfigs: map[index]cellType 用于指定特定位置的格子类型
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

// GetCell 获取指定位置的格子
func (m *MapEngine) GetCell(index int) (*MapCell, error) {
	if index < 0 || index >= m.Length {
		return nil, errors.New("index out of bounds")
	}
	return m.Cells[index], nil
}

// SetCellType 设置指定位置格子的类型
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

// BreakFragile 破碎易碎格子
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

// ActivateFog 激活迷雾格子
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

// PathResult 移动路径计算结果
type PathResult struct {
	StartIndex  int   `json:"start_index"`  // 起始位置
	TargetIndex int   `json:"target_index"` // 目标位置（实际到达位置）
	Path        []int `json:"path"`         // 经过的格子索引列表
	Interrupted bool  `json:"interrupted"`  // 是否被中断（如掉落 fragile）
	FellDown    bool  `json:"fell_down"`    // 是否掉落（踩碎 fragile）
	ReachedEnd  bool  `json:"reached_end"`  // 是否到达终点
}

// CalculatePath 计算移动路径
// start: 起始位置，steps: 骰子步数
// 返回实际移动路径，处理途经 Fragile 块掉落中止移动的逻辑
func (m *MapEngine) CalculatePath(start int, steps int) (*PathResult, error) {
	if start < 0 || start >= m.Length {
		return nil, errors.New("start index out of bounds")
	}
	if steps < 0 {
		return nil, errors.New("steps cannot be negative")
	}

	result := &PathResult{
		StartIndex:  start,
		TargetIndex: start,
		Path:        []int{start},
		Interrupted: false,
		FellDown:    false,
		ReachedEnd:  false,
	}

	// 计算目标位置（不超过地图终点）
	target := start + steps
	if target >= m.Length {
		target = m.Length - 1
		result.ReachedEnd = true
	}

	// 逐格移动，检查 Fragile 块
	for i := start + 1; i <= target; i++ {
		cell, err := m.GetCell(i)
		if err != nil {
			break
		}

		result.Path = append(result.Path, i)
		result.TargetIndex = i

		// 检查 Fragile 块
		if cell.CellType == CellTypeFragile {
			if !cell.IsBroken {
				// 未碎的 Fragile 块，踩上去会碎掉并掉落
				cell.IsBroken = true
				result.Interrupted = true
				result.FellDown = true
				break
			}
			// 已碎的 Fragile 块，跳过去（不影响移动）
		}

		// 检查迷雾区域
		if cell.CellType == CellTypeFog {
			cell.FogActive = true
		}

		// 到达终点
		if i == m.Length-1 {
			result.ReachedEnd = true
			break
		}
	}

	return result, nil
}

// Export 导出地图数据为 JSON
func (m *MapEngine) Export() ([]byte, error) {
	return json.Marshal(m)
}

// Import 从 JSON 导入地图数据
func (m *MapEngine) Import(data []byte) error {
	return json.Unmarshal(data, m)
}

// LoadMap 从 JSON 加载地图（创建新的 MapEngine）
func LoadMap(data []byte) (*MapEngine, error) {
	var engine MapEngine
	if err := json.Unmarshal(data, &engine); err != nil {
		return nil, err
	}
	// 验证数据有效性
	if engine.Length < 1 {
		return nil, errors.New("invalid map length")
	}
	if len(engine.Cells) != engine.Length {
		return nil, errors.New("cells count mismatch")
	}
	return &engine, nil
}

// GetCellsByType 获取指定类型的所有格子
func (m *MapEngine) GetCellsByType(cellType CellType) []*MapCell {
	var result []*MapCell
	for _, cell := range m.Cells {
		if cell.CellType == cellType {
			result = append(result, cell)
		}
	}
	return result
}

// GetLastCheckpoint 获取指定位置之前最近的检查点
func (m *MapEngine) GetLastCheckpoint(position int) int {
	lastCheckpoint := 0
	for i := 0; i < position && i < m.Length; i++ {
		if m.Cells[i].CellType == CellTypeCheckpoint {
			lastCheckpoint = i
		}
	}
	return lastCheckpoint
}

// Clone 克隆地图引擎
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