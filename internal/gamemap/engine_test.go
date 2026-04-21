package gamemap

import (
	"encoding/json"
	"testing"

	"github.com/b1tAction/paradiced/pkg/constants"
)

// ========== CellType Tests ==========

func TestCellTypeStringValues(t *testing.T) {
	tests := []struct {
		ct       constants.CellType
		expected string
	}{
		{constants.CellTypeNormal, "normal"},
		{constants.CellTypeFragile, "fragile"},
		{constants.CellTypeFog, "fog"},
		{constants.CellTypeCheckpoint, "checkpoint"},
		{constants.CellTypeBoss, "boss"},
		{constants.CellTypeEvent, "event"},
	}

	for _, tt := range tests {
		if string(tt.ct) != tt.expected {
			t.Errorf("CellType(%s) = %s, expected %s", tt.ct, string(tt.ct), tt.expected)
		}
	}
}

func TestCellTypeIsValid(t *testing.T) {
	validTypes := []constants.CellType{constants.CellTypeNormal, constants.CellTypeFragile, constants.CellTypeFog, constants.CellTypeCheckpoint, constants.CellTypeBoss, constants.CellTypeEvent}
	for _, ct := range validTypes {
		if !ct.IsValid() {
			t.Errorf("constants.CellType(%s).IsValid() should be true", ct)
		}
	}

	invalidTypes := []constants.CellType{constants.CellType("invalid"), constants.CellType("unknown"), constants.CellType("")}
	for _, ct := range invalidTypes {
		if ct.IsValid() {
			t.Errorf("constants.CellType(%s).IsValid() should be false", ct)
		}
	}
}

// ========== MapCell Tests ==========

func TestNewMapCell(t *testing.T) {
	cell := NewMapCell(5, constants.CellTypeFragile)
	if cell.Index != 5 {
		t.Errorf("cell.Index = %d, expected 5", cell.Index)
	}
	if cell.CellType != constants.CellTypeFragile {
		t.Errorf("cell.CellType = %s, expected %s", cell.CellType, constants.CellTypeFragile)
	}
	if cell.IsBroken {
		t.Error("cell.IsBroken should be false initially")
	}
	if cell.EventID != "" {
		t.Error("cell.EventID should be empty initially")
	}
}

// ========== MapEngine Tests ==========

func TestNewMapEngine(t *testing.T) {
	// 正常创建
	engine := NewMapEngine(50)
	if engine.Length != 50 {
		t.Errorf("engine.Length = %d, expected 50", engine.Length)
	}
	if len(engine.Cells) != 50 {
		t.Errorf("len(engine.Cells) = %d, expected 50", len(engine.Cells))
	}
	if engine.StartIndex != 0 {
		t.Errorf("engine.StartIndex = %d, expected 0", engine.StartIndex)
	}
	if engine.EndIndex != 49 {
		t.Errorf("engine.EndIndex = %d, expected 49", engine.EndIndex)
	}

	// 检查所有格子初始为 Normal
	for i, cell := range engine.Cells {
		if cell.CellType != constants.CellTypeNormal {
			t.Errorf("cell[%d].CellType = %s, expected Normal", i, cell.CellType)
		}
		if cell.Index != i {
			t.Errorf("cell[%d].Index = %d, expected %d", i, cell.Index, i)
		}
	}

	// 长度为 0 时自动修正为 1
	engine2 := NewMapEngine(0)
	if engine2.Length != 1 {
		t.Errorf("engine2.Length = %d, expected 1", engine2.Length)
	}
}

func TestGenerateLinearMap(t *testing.T) {
	engine := NewMapEngine(50)

	// 配置特定格子类型
	configs := map[int]constants.CellType{
		10: constants.CellTypeFragile,
		15: constants.CellTypeFog,
		20: constants.CellTypeCheckpoint,
		49: constants.CellTypeBoss,
	}

	err := engine.GenerateLinearMap(configs)
	if err != nil {
		t.Fatalf("GenerateLinearMap failed: %v", err)
	}

	// 验证配置的格子类型
	if engine.Cells[10].CellType != constants.CellTypeFragile {
		t.Error("cell 10 should be Fragile")
	}
	if engine.Cells[15].CellType != constants.CellTypeFog {
		t.Error("cell 15 should be Fog")
	}
	if engine.Cells[20].CellType != constants.CellTypeCheckpoint {
		t.Error("cell 20 should be Checkpoint")
	}
	if engine.Cells[49].CellType != constants.CellTypeBoss {
		t.Error("cell 49 should be Boss")
	}

	// 验证其他格子为 Normal
	if engine.Cells[0].CellType != constants.CellTypeNormal {
		t.Error("cell 0 should be Normal")
	}
	if engine.Cells[25].CellType != constants.CellTypeNormal {
		t.Error("cell 25 should be Normal")
	}

	// 测试无效格子类型
	invalidConfigs := map[int]constants.CellType{
		5: constants.CellType("invalid"),
	}
	err = engine.GenerateLinearMap(invalidConfigs)
	if err == nil {
		t.Error("GenerateLinearMap should return error for invalid cell type")
	}
}

func TestGetCell(t *testing.T) {
	engine := NewMapEngine(10)

	// 正常获取
	cell, err := engine.GetCell(5)
	if err != nil {
		t.Fatalf("GetCell(5) failed: %v", err)
	}
	if cell.Index != 5 {
		t.Errorf("cell.Index = %d, expected 5", cell.Index)
	}

	// 越界测试
	_, err = engine.GetCell(-1)
	if err == nil {
		t.Error("GetCell(-1) should return error")
	}
	_, err = engine.GetCell(100)
	if err == nil {
		t.Error("GetCell(100) should return error")
	}
}

func TestSetCellType(t *testing.T) {
	engine := NewMapEngine(10)

	err := engine.SetCellType(5, constants.CellTypeFragile)
	if err != nil {
		t.Fatalf("SetCellType failed: %v", err)
	}
	if engine.Cells[5].CellType != constants.CellTypeFragile {
		t.Error("cell 5 should be Fragile after SetCellType")
	}

	// 越界测试
	err = engine.SetCellType(-1, constants.CellTypeNormal)
	if err == nil {
		t.Error("SetCellType(-1) should return error")
	}

	// 无效类型测试
	err = engine.SetCellType(5, constants.CellType("invalid"))
	if err == nil {
		t.Error("SetCellType with invalid type should return error")
	}
}

func TestBreakFragile(t *testing.T) {
	engine := NewMapEngine(10)
	engine.SetCellType(5, constants.CellTypeFragile)

	// 正常破碎
	err := engine.BreakFragile(5)
	if err != nil {
		t.Fatalf("BreakFragile failed: %v", err)
	}
	if !engine.Cells[5].IsBroken {
		t.Error("cell 5 should be broken")
	}

	// 重复破碎（应该成功，已经 broken）
	err = engine.BreakFragile(5)
	if err != nil {
		t.Error("BreakFragile on already broken cell should succeed")
	}

	// 非 Fragile 格子破碎
	engine.SetCellType(3, constants.CellTypeNormal)
	err = engine.BreakFragile(3)
	if err == nil {
		t.Error("BreakFragile on Normal cell should return error")
	}
}

func TestActivateFog(t *testing.T) {
	engine := NewMapEngine(10)
	engine.SetCellType(5, constants.CellTypeFog)

	err := engine.ActivateFog(5)
	if err != nil {
		t.Fatalf("ActivateFog failed: %v", err)
	}
	if !engine.Cells[5].FogActive {
		t.Error("cell 5 should have fog active")
	}

	// 非 Fog 格子激活
	engine.SetCellType(3, constants.CellTypeNormal)
	err = engine.ActivateFog(3)
	if err == nil {
		t.Error("ActivateFog on Normal cell should return error")
	}
}

// ========== CalculatePath Tests ==========

func TestCalculatePathNormal(t *testing.T) {
	engine := NewMapEngine(50)

	// 正常移动
	result, err := engine.CalculatePath(0, 5)
	if err != nil {
		t.Fatalf("CalculatePath failed: %v", err)
	}

	if result.StartIndex != 0 {
		t.Errorf("StartIndex = %d, expected 0", result.StartIndex)
	}
	if result.TargetIndex != 5 {
		t.Errorf("TargetIndex = %d, expected 5", result.TargetIndex)
	}
	if result.OriginalTarget != 5 {
		t.Errorf("OriginalTarget = %d, expected 5", result.OriginalTarget)
	}
	if len(result.Path) != 6 {
		t.Errorf("Path length = %d, expected 6", len(result.Path))
	}
	if result.Interrupted {
		t.Error("should not be interrupted")
	}
	if result.FellDown {
		t.Error("should not fall down")
	}
	if result.ReachedEnd {
		t.Error("should not reach end")
	}
	if len(result.BrokenFragiles) != 0 {
		t.Errorf("BrokenFragiles count = %d, expected 0", len(result.BrokenFragiles))
	}
}

func TestCalculatePathReachEnd(t *testing.T) {
	engine := NewMapEngine(10)

	// 移动超过终点
	result, err := engine.CalculatePath(5, 10)
	if err != nil {
		t.Fatalf("CalculatePath failed: %v", err)
	}

	if result.TargetIndex != 9 {
		t.Errorf("TargetIndex = %d, expected 9 (end)", result.TargetIndex)
	}
	if result.OriginalTarget != 15 {
		t.Errorf("OriginalTarget = %d, expected 15", result.OriginalTarget)
	}
	if !result.ReachedEnd {
		t.Error("should reach end")
	}
}

// ========== Fragile Tests ==========

func TestCalculatePathPassUnbrokenFragile(t *testing.T) {
	engine := NewMapEngine(50)
	engine.SetCellType(5, constants.CellTypeFragile)

	result, err := engine.CalculatePath(0, 10)
	if err != nil {
		t.Fatalf("CalculatePath failed: %v", err)
	}

	if result.TargetIndex != 10 {
		t.Errorf("TargetIndex = %d, expected 10", result.TargetIndex)
	}
	if result.Interrupted {
		t.Error("should not be interrupted")
	}
	if result.FellDown {
		t.Error("should not fall down")
	}

	if !engine.Cells[5].IsBroken {
		t.Error("fragile cell at position 5 should be broken after passing")
	}
	if len(result.BrokenFragiles) != 1 {
		t.Errorf("BrokenFragiles count = %d, expected 1", len(result.BrokenFragiles))
	}
}

func TestCalculatePathLandOnUnbrokenFragile(t *testing.T) {
	engine := NewMapEngine(50)
	engine.SetCellType(10, constants.CellTypeFragile)

	result, err := engine.CalculatePath(0, 10)
	if err != nil {
		t.Fatalf("CalculatePath failed: %v", err)
	}

	if result.TargetIndex != 10 {
		t.Errorf("TargetIndex = %d, expected 10", result.TargetIndex)
	}
	if !result.Interrupted {
		t.Error("should be interrupted")
	}
	if !result.FellDown {
		t.Error("should fall down")
	}

	if !engine.Cells[10].IsBroken {
		t.Error("fragile cell should be broken after landing")
	}
}

func TestCalculatePathLandOnBrokenFragile(t *testing.T) {
	engine := NewMapEngine(50)
	engine.SetCellType(10, constants.CellTypeFragile)
	engine.Cells[10].IsBroken = true

	result, err := engine.CalculatePath(0, 10)
	if err != nil {
		t.Fatalf("CalculatePath failed: %v", err)
	}

	if result.TargetIndex != 9 {
		t.Errorf("TargetIndex = %d, expected 9", result.TargetIndex)
	}
	if result.Interrupted {
		t.Error("should not be interrupted")
	}
	if result.FellDown {
		t.Error("should not fall down")
	}
}

func TestCalculatePathFogActivation(t *testing.T) {
	engine := NewMapEngine(50)
	engine.SetCellType(5, constants.CellTypeFog)

	_, err := engine.CalculatePath(0, 10)
	if err != nil {
		t.Fatalf("CalculatePath failed: %v", err)
	}

	if !engine.Cells[5].FogActive {
		t.Error("fog cell should be activated after passing")
	}
}

func TestCalculatePathErrors(t *testing.T) {
	engine := NewMapEngine(10)

	// 起始位置越界
	_, err := engine.CalculatePath(-1, 5)
	if err == nil {
		t.Error("CalculatePath(-1, 5) should return error")
	}

	_, err = engine.CalculatePath(100, 5)
	if err == nil {
		t.Error("CalculatePath(100, 5) should return error")
	}
}

// ========== Reverse Movement Tests ==========

func TestCalculatePathReverseMovement(t *testing.T) {
	engine := NewMapEngine(50)

	// Reverse 3 steps from position 10 → 7
	result, err := engine.CalculatePath(10, -3)
	if err != nil {
		t.Fatalf("CalculatePath(10, -3) error: %v", err)
	}
	if result.TargetIndex != 7 {
		t.Errorf("TargetIndex = %d, want 7", result.TargetIndex)
	}
	if result.StartIndex != 10 {
		t.Errorf("StartIndex = %d, want 10", result.StartIndex)
	}
	if result.OriginalTarget != 7 {
		t.Errorf("OriginalTarget = %d, want 7", result.OriginalTarget)
	}
	// Path should be descending: [10, 9, 8, 7]
	expectedPath := []int{10, 9, 8, 7}
	if len(result.Path) != len(expectedPath) {
		t.Fatalf("Path length = %d, want %d", len(result.Path), len(expectedPath))
	}
	for i, v := range result.Path {
		if v != expectedPath[i] {
			t.Errorf("Path[%d] = %d, want %d", i, v, expectedPath[i])
		}
	}
	if result.ReachedEnd {
		t.Error("ReachedEnd should be false for reverse movement")
	}
}

func TestCalculatePathReverseBoundary(t *testing.T) {
	engine := NewMapEngine(50)

	// Reverse movement that would go below 0 → target clamped to 0
	result, err := engine.CalculatePath(3, -10)
	if err != nil {
		t.Fatalf("CalculatePath(3, -10) error: %v", err)
	}
	if result.TargetIndex != 0 {
		t.Errorf("TargetIndex = %d, want 0 (clamped)", result.TargetIndex)
	}
	// Path should be [3, 2, 1, 0]
	expectedPath := []int{3, 2, 1, 0}
	if len(result.Path) != len(expectedPath) {
		t.Fatalf("Path length = %d, want %d", len(result.Path), len(expectedPath))
	}
}

func TestCalculatePathReverseFogActivation(t *testing.T) {
	engine := NewMapEngine(20)
	engine.SetCellType(8, constants.CellTypeFog)
	engine.SetCellType(9, constants.CellTypeFog)

	// Reverse movement from 10 → 7, passing Fog cells at 9 and 8
	result, err := engine.CalculatePath(10, -3)
	if err != nil {
		t.Fatalf("CalculatePath(10, -3) error: %v", err)
	}
	if result.TargetIndex != 7 {
		t.Errorf("TargetIndex = %d, want 7", result.TargetIndex)
	}

	// Check fog cells are activated
	fogCell8, _ := engine.GetCell(8)
	fogCell9, _ := engine.GetCell(9)
	if !fogCell8.FogActive {
		t.Error("Fog cell at 8 should be activated when passing")
	}
	if !fogCell9.FogActive {
		t.Error("Fog cell at 9 should be activated when passing")
	}
}

func TestCalculatePathReverseFragileLanding(t *testing.T) {
	engine := NewMapEngine(20)
	engine.SetCellType(7, constants.CellTypeFragile)

	// Reverse movement landing on unbroken Fragile at position 7
	result, err := engine.CalculatePath(10, -3)
	if err != nil {
		t.Fatalf("CalculatePath(10, -3) error: %v", err)
	}
	if !result.FellDown {
		t.Error("FellDown should be true when landing on unbroken Fragile")
	}
	if !result.Interrupted {
		t.Error("Interrupted should be true when FellDown")
	}
	if result.TargetIndex != 7 {
		t.Errorf("TargetIndex = %d, want 7 (fell down at fragile)", result.TargetIndex)
	}

	// Verify fragile cell is now broken
	fragileCell, _ := engine.GetCell(7)
	if !fragileCell.IsBroken {
		t.Error("Fragile cell at 7 should be broken after landing")
	}
}

func TestCalculatePathZeroSteps(t *testing.T) {
	engine := NewMapEngine(50)

	// Zero steps: player stays at current position
	result, err := engine.CalculatePath(10, 0)
	if err != nil {
		t.Fatalf("CalculatePath(10, 0) error: %v", err)
	}
	if result.TargetIndex != 10 {
		t.Errorf("TargetIndex = %d, want 10", result.TargetIndex)
	}
	if len(result.Path) != 1 {
		t.Errorf("Path length = %d, want 1 (only start position)", len(result.Path))
	}
}

// ========== Export/Import Tests ==========

func TestMapExportImport(t *testing.T) {
	original := NewMapEngine(50)
	original.SetCellType(10, constants.CellTypeFragile)
	original.SetCellType(15, constants.CellTypeFog)
	original.SetCellType(20, constants.CellTypeCheckpoint)
	original.Cells[10].IsBroken = true

	data, err := original.Export()
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	loaded, err := LoadMap(data)
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}

	if loaded.Length != original.Length {
		t.Errorf("loaded.Length = %d, expected %d", loaded.Length, original.Length)
	}
	if loaded.Cells[10].CellType != constants.CellTypeFragile {
		t.Error("loaded cell 10 should be Fragile")
	}
	if !loaded.Cells[10].IsBroken {
		t.Error("loaded cell 10 should be broken")
	}
}

func TestLoadMapInvalidData(t *testing.T) {
	// 无效 JSON
	_, err := LoadMap([]byte("invalid json"))
	if err == nil {
		t.Error("LoadMap with invalid JSON should return error")
	}

	// 长度为 0
	data, _ := json.Marshal(&MapEngine{Length: 0})
	_, err = LoadMap(data)
	if err == nil {
		t.Error("LoadMap with length 0 should return error")
	}
}

// ========== Helper Methods Tests ==========

func TestGetCellsByType(t *testing.T) {
	engine := NewMapEngine(50)
	engine.SetCellType(10, constants.CellTypeFragile)
	engine.SetCellType(20, constants.CellTypeFragile)
	engine.SetCellType(15, constants.CellTypeFog)
	engine.SetCellType(25, constants.CellTypeCheckpoint)

	fragiles := engine.GetCellsByType(constants.CellTypeFragile)
	if len(fragiles) != 2 {
		t.Errorf("fragiles count = %d, expected 2", len(fragiles))
	}

	fogs := engine.GetCellsByType(constants.CellTypeFog)
	if len(fogs) != 1 {
		t.Errorf("fogs count = %d, expected 1", len(fogs))
	}
}

func TestGetLastCheckpoint(t *testing.T) {
	engine := NewMapEngine(50)
	engine.SetCellType(10, constants.CellTypeCheckpoint)
	engine.SetCellType(20, constants.CellTypeCheckpoint)
	engine.SetCellType(30, constants.CellTypeCheckpoint)

	cp := engine.GetLastCheckpoint(25)
	if cp != 20 {
		t.Errorf("last checkpoint before 25 = %d, expected 20", cp)
	}

	cp = engine.GetLastCheckpoint(15)
	if cp != 10 {
		t.Errorf("last checkpoint before 15 = %d, expected 10", cp)
	}

	cp = engine.GetLastCheckpoint(5)
	if cp != 0 {
		t.Errorf("last checkpoint before 5 = %d, expected 0", cp)
	}
}

func TestClone(t *testing.T) {
	original := NewMapEngine(50)
	original.SetCellType(10, constants.CellTypeFragile)
	original.Cells[10].IsBroken = true

	cloned := original.Clone()

	cloned.Cells[10].IsBroken = false
	cloned.SetCellType(15, constants.CellTypeFog)

	if !original.Cells[10].IsBroken {
		t.Error("original cell 10 should still be broken")
	}
	if original.Cells[15].CellType == constants.CellTypeFog {
		t.Error("original cell 15 should not be affected")
	}
}