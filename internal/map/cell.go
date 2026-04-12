package map

// CellType 定义地图格子的类型
type CellType int

const (
	CellTypeNormal CellType = iota // 普通格子
	CellTypeFragile                // 易碎格子
	CellTypeFog                    // 迷雾格子
	CellTypeCheckpoint             // 检查点
	CellTypeBoss                   // Boss格子（终点）
)

// String 返回格子类型的字符串表示
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

// IsValid 检查格子类型是否有效
func (ct CellType) IsValid() bool {
	return ct >= CellTypeNormal && ct <= CellTypeBoss
}