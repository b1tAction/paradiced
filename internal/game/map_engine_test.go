package game

import (
	"encoding/json"
	"testing"
)

// ========== CellType Tests ==========

func TestCellTypeString(t *testing.T) {
	tests := []struct {
		ct       CellType
		expected string
	}{
		{CellTypeNormal, "Normal"},
		{CellTypeFragile, "Fragile"},
		{CellTypeFog, "Fog"},
		{CellTypeCheckpoint, "Checkpoint"},
		{CellTypeBoss, "Boss"},
		{CellType(999), "Unknown"},
	}

	for _, tt := range tests {
		result := tt.ct.String()
		if result != tt.expected {
			t.Errorf("CellType(%d).String() = %s, expected %s", tt.ct, result, tt.expected)
		}
	}
}

func TestCellTypeIsValid(t *testing.T) {
	validTypes := []CellType{CellTypeNormal, CellTypeFragile, CellTypeFog, CellTypeCheckpoint, CellTypeBoss}
	for _, ct := range validTypes {
		if !ct.IsValid() {
			t.Errorf("CellType(%d).IsValid() should be true", ct)
		}
	}

	invalidTypes := []CellType{CellType(-1), CellType(100)}
	for _, ct := range invalidTypes {
		if ct.IsValid() {
			t.Errorf("CellType(%d).IsValid() should be false", ct)
		}
	}
}

// ========== MapCell Tests ==========

func TestNewMapCell(t *testing.T) {
	cell := NewMapCell(5, CellTypeFragile)
	if cell.Index != 5 {
		t.Errorf("cell.Index = %d, expected 5", cell.Index)
	}
	if cell.CellType != CellTypeFragile {
		t.Errorf("cell.CellType = %d, expected %d", cell.CellType, CellTypeFragile)
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
		if cell.CellType != CellTypeNormal {
			t.Errorf("cell[%d].CellType = %d, expected Normal", i, cell.CellType)
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
	configs := map[int]CellType{
		10: CellTypeFragile,
		15: CellTypeFog,
		20: CellTypeCheckpoint,
		49: CellTypeBoss,
	}

	err := engine.GenerateLinearMap(configs)
	if err != nil {
		t.Fatalf("GenerateLinearMap failed: %v", err)
	}

	// 验证配置的格子类型
	if engine.Cells[10].CellType != CellTypeFragile {
		t.Error("cell 10 should be Fragile")
	}
	if engine.Cells[15].CellType != CellTypeFog {
		t.Error("cell 15 should be Fog")
	}
	if engine.Cells[20].CellType != CellTypeCheckpoint {
		t.Error("cell 20 should be Checkpoint")
	}
	if engine.Cells[49].CellType != CellTypeBoss {
		t.Error("cell 49 should be Boss")
	}

	// 验证其他格子为 Normal
	if engine.Cells[0].CellType != CellTypeNormal {
		t.Error("cell 0 should be Normal")
	}
	if engine.Cells[25].CellType != CellTypeNormal {
		t.Error("cell 25 should be Normal")
	}

	// 测试无效格子类型
	invalidConfigs := map[int]CellType{
		5: CellType(999),
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

	err := engine.SetCellType(5, CellTypeFragile)
	if err != nil {
		t.Fatalf("SetCellType failed: %v", err)
	}
	if engine.Cells[5].CellType != CellTypeFragile {
		t.Error("cell 5 should be Fragile after SetCellType")
	}

	// 越界测试
	err = engine.SetCellType(-1, CellTypeNormal)
	if err == nil {
		t.Error("SetCellType(-1) should return error")
	}

	// 无效类型测试
	err = engine.SetCellType(5, CellType(999))
	if err == nil {
		t.Error("SetCellType with invalid type should return error")
	}
}

func TestBreakFragile(t *testing.T) {
	engine := NewMapEngine(10)
	engine.SetCellType(5, CellTypeFragile)

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
	engine.SetCellType(3, CellTypeNormal)
	err = engine.BreakFragile(3)
	if err == nil {
		t.Error("BreakFragile on Normal cell should return error")
	}
}

func TestActivateFog(t *testing.T) {
	engine := NewMapEngine(10)
	engine.SetCellType(5, CellTypeFog)

	err := engine.ActivateFog(5)
	if err != nil {
		t.Fatalf("ActivateFog failed: %v", err)
	}
	if !engine.Cells[5].FogActive {
		t.Error("cell 5 should have fog active")
	}

	// 非 Fog 格子激活
	engine.SetCellType(3, CellTypeNormal)
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

	// 验证路径
	expectedPath := []int{0, 1, 2, 3, 4, 5}
	for i, idx := range result.Path {
		if idx != expectedPath[i] {
			t.Errorf("Path[%d] = %d, expected %d", i, idx, expectedPath[i])
		}
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

// ========== Fragile Tests (New Logic) ==========

// 测试：经过未碎 Fragile，最终落点不是 Fragile（不中断，继续移动）
func TestCalculatePathPassUnbrokenFragile(t *testing.T) {
	engine := NewMapEngine(50)
	engine.SetCellType(5, CellTypeFragile) // 位置5是未碎Fragile

	// 从位置0移动10步，目标位置10，经过位置5的Fragile
	result, err := engine.CalculatePath(0, 10)
	if err != nil {
		t.Fatalf("CalculatePath failed: %v", err)
	}

	// 应该正常到达位置10（不中断）
	if result.TargetIndex != 10 {
		t.Errorf("TargetIndex = %d, expected 10", result.TargetIndex)
	}
	if result.Interrupted {
		t.Error("should not be interrupted (passing fragile, not landing on it)")
	}
	if result.FellDown {
		t.Error("should not fall down (passing fragile, not landing on it)")
	}

	// 经过路径中的 Fragile 应该碎裂
	if !engine.Cells[5].IsBroken {
		t.Error("fragile cell at position 5 should be broken after passing")
	}
	if len(result.BrokenFragiles) != 1 {
		t.Errorf("BrokenFragiles count = %d, expected 1", len(result.BrokenFragiles))
	}
	if result.BrokenFragiles[0] != 5 {
		t.Errorf("BrokenFragiles[0] = %d, expected 5", result.BrokenFragiles[0])
	}

	// 验证路径长度（包含起点和终点）
	if len(result.Path) != 11 {
		t.Errorf("Path length = %d, expected 11", len(result.Path))
	}
}

// 测试：最终落点恰好是未碎 Fragile（碎裂+排落）
func TestCalculatePathLandOnUnbrokenFragile(t *testing.T) {
	engine := NewMapEngine(50)
	engine.SetCellType(10, CellTypeFragile) // 位置10是未碎Fragile

	// 从位置0移动10步，目标恰好是位置10的Fragile
	result, err := engine.CalculatePath(0, 10)
	if err != nil {
		t.Fatalf("CalculatePath failed: %v", err)
	}

	// 应该在位置10中断并排落
	if result.TargetIndex != 10 {
		t.Errorf("TargetIndex = %d, expected 10", result.TargetIndex)
	}
	if !result.Interrupted {
		t.Error("should be interrupted (landing on fragile)")
	}
	if !result.FellDown {
		t.Error("should fall down (landing on fragile)")
	}

	// 落点的 Fragile 应该碎裂
	if !engine.Cells[10].IsBroken {
		t.Error("fragile cell should be broken after landing")
	}
	if len(result.BrokenFragiles) != 1 {
		t.Errorf("BrokenFragiles count = %d, expected 1", len(result.BrokenFragiles))
	}
	if result.BrokenFragiles[0] != 10 {
		t.Errorf("BrokenFragiles[0] = %d, expected 10", result.BrokenFragiles[0])
	}

	// 验证路径长度
	if len(result.Path) != 11 {
		t.Errorf("Path length = %d, expected 11", len(result.Path))
	}
}

// 测试：最终落点恰好是已碎 Fragile（停在上一格）
func TestCalculatePathLandOnBrokenFragile(t *testing.T) {
	engine := NewMapEngine(50)
	engine.SetCellType(10, CellTypeFragile)
	engine.Cells[10].IsBroken = true // 位置10是已碎Fragile

	// 从位置0移动10步，目标恰好是位置10的已碎Fragile
	result, err := engine.CalculatePath(0, 10)
	if err != nil {
		t.Fatalf("CalculatePath failed: %v", err)
	}

	// 应该停在上一格（位置9）
	if result.TargetIndex != 9 {
		t.Errorf("TargetIndex = %d, expected 9 (stopped before broken fragile)", result.TargetIndex)
	}
	if result.Interrupted {
		t.Error("should not be interrupted (just stopped, not fell)")
	}
	if result.FellDown {
		t.Error("should not fall down (broken fragile)")
	}

	// 路径不包含无法到达的格子
	if len(result.Path) != 10 {
		t.Errorf("Path length = %d, expected 10 (without broken fragile)", len(result.Path))
	}

	// 验证最后一个格子是位置9
	if result.Path[len(result.Path)-1] != 9 {
		t.Errorf("last path position = %d, expected 9", result.Path[len(result.Path)-1])
	}
}

// 测试：经过已碎 Fragile，最终落点不是 Fragile（正常移动）
func TestCalculatePathPassBrokenFragile(t *testing.T) {
	engine := NewMapEngine(50)
	engine.SetCellType(5, CellTypeFragile)
	engine.Cells[5].IsBroken = true // 位置5是已碎Fragile

	// 从位置0移动10步，经过已碎的Fragile，到达位置10
	result, err := engine.CalculatePath(0, 10)
	if err != nil {
		t.Fatalf("CalculatePath failed: %v", err)
	}

	// 应该正常到达位置10
	if result.TargetIndex != 10 {
		t.Errorf("TargetIndex = %d, expected 10", result.TargetIndex)
	}
	if result.Interrupted {
		t.Error("should not be interrupted (fragile already broken)")
	}
	if result.FellDown {
		t.Error("should not fall down (fragile already broken)")
	}
	if len(result.BrokenFragiles) != 0 {
		t.Errorf("BrokenFragiles count = %d, expected 0 (already broken)", len(result.BrokenFragiles))
	}
}

// 测试：路径中经过多个未碎 Fragile，最终落点是其中一个
func TestCalculatePathMultipleFragiles(t *testing.T) {
	engine := NewMapEngine(50)
	engine.SetCellType(5, CellTypeFragile)
	engine.SetCellType(10, CellTypeFragile) // 最终落点

	// 从位置0移动10步，经过位置5，最终落在位置10
	result, err := engine.CalculatePath(0, 10)
	if err != nil {
		t.Fatalf("CalculatePath failed: %v", err)
	}

	// 应该在位置10中断并排落
	if result.TargetIndex != 10 {
		t.Errorf("TargetIndex = %d, expected 10", result.TargetIndex)
	}
	if !result.Interrupted {
		t.Error("should be interrupted")
	}
	if !result.FellDown {
		t.Error("should fall down")
	}

	// 位置5的Fragile也应该碎裂（经过时）
	if !engine.Cells[5].IsBroken {
		t.Error("fragile cell at 5 should be broken (passed through)")
	}
	// 位置10的Fragile碎裂（落点）
	if !engine.Cells[10].IsBroken {
		t.Error("fragile cell at 10 should be broken (landing)")
	}
}

// 测试：路径中经过多个未碎 Fragile，最终落点不是 Fragile
func TestCalculatePathPassMultipleFragiles(t *testing.T) {
	engine := NewMapEngine(50)
	engine.SetCellType(5, CellTypeFragile)
	engine.SetCellType(10, CellTypeFragile)

	// 从位置0移动15步，经过位置5和10，最终落在位置15
	result, err := engine.CalculatePath(0, 15)
	if err != nil {
		t.Fatalf("CalculatePath failed: %v", err)
	}

	// 应该正常到达位置15
	if result.TargetIndex != 15 {
		t.Errorf("TargetIndex = %d, expected 15", result.TargetIndex)
	}
	if result.Interrupted {
		t.Error("should not be interrupted")
	}
	if result.FellDown {
		t.Error("should not fall down")
	}

	// 两个 Fragile 都应该碎裂
	if !engine.Cells[5].IsBroken {
		t.Error("fragile cell at 5 should be broken")
	}
	if !engine.Cells[10].IsBroken {
		t.Error("fragile cell at 10 should be broken")
	}
	if len(result.BrokenFragiles) != 2 {
		t.Errorf("BrokenFragiles count = %d, expected 2", len(result.BrokenFragiles))
	}
}

// 测试：从 Fragile 格子开始移动
func TestCalculatePathStartOnFragile(t *testing.T) {
	engine := NewMapEngine(50)
	engine.SetCellType(5, CellTypeFragile)

	// 从位置5（Fragile）开始移动
	result, err := engine.CalculatePath(5, 5)
	if err != nil {
		t.Fatalf("CalculatePath failed: %v", err)
	}

	// 正常移动到位置10
	if result.TargetIndex != 10 {
		t.Errorf("TargetIndex = %d, expected 10", result.TargetIndex)
	}
	if result.Interrupted {
		t.Error("should not be interrupted (starting on fragile)")
	}
	if result.FellDown {
		t.Error("should not fall down (starting on fragile)")
	}

	// 起始位置的 Fragile 不应该碎裂（已经在上面，不是"经过"）
	if engine.Cells[5].IsBroken {
		t.Error("starting fragile cell should not be broken")
	}
}

// 测试：从已碎 Fragile 格子开始移动
func TestCalculatePathStartOnBrokenFragile(t *testing.T) {
	engine := NewMapEngine(50)
	engine.SetCellType(5, CellTypeFragile)
	engine.Cells[5].IsBroken = true

	// 从位置5（已碎Fragile）开始移动1步
	result, err := engine.CalculatePath(5, 1)
	if err != nil {
		t.Fatalf("CalculatePath failed: %v", err)
	}

	// 应该正常移动到位置6
	if result.TargetIndex != 6 {
		t.Errorf("TargetIndex = %d, expected 6", result.TargetIndex)
	}
	if result.Interrupted {
		t.Error("should not be interrupted")
	}
}

// 测试：步数为0（原地不动）
func TestCalculatePathZeroSteps(t *testing.T) {
	engine := NewMapEngine(50)
	engine.SetCellType(5, CellTypeFragile)

	// 从位置0移动0步
	result, err := engine.CalculatePath(0, 0)
	if err != nil {
		t.Fatalf("CalculatePath failed: %v", err)
	}

	// 应该停在位置0
	if result.TargetIndex != 0 {
		t.Errorf("TargetIndex = %d, expected 0", result.TargetIndex)
	}
	if len(result.Path) != 1 {
		t.Errorf("Path length = %d, expected 1", len(result.Path))
	}
}

// 测试：最终落点是已碎 Fragile，但起点就在前一格
func TestCalculatePathLandOnBrokenFragileAdjacent(t *testing.T) {
	engine := NewMapEngine(50)
	engine.SetCellType(6, CellTypeFragile)
	engine.Cells[6].IsBroken = true

	// 从位置5移动1步，目标是已碎的位置6
	result, err := engine.CalculatePath(5, 1)
	if err != nil {
		t.Fatalf("CalculatePath failed: %v", err)
	}

	// 应该停在位置5
	if result.TargetIndex != 5 {
		t.Errorf("TargetIndex = %d, expected 5", result.TargetIndex)
	}
	if len(result.Path) != 1 {
		t.Errorf("Path length = %d, expected 1", len(result.Path))
	}
}

func TestCalculatePathFogActivation(t *testing.T) {
	engine := NewMapEngine(50)
	engine.SetCellType(5, CellTypeFog)

	_, err := engine.CalculatePath(0, 10)
	if err != nil {
		t.Fatalf("CalculatePath failed: %v", err)
	}

	// 迷雾应该被激活
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

	// 步数为负（用户修改：允许负步数表示反向移动）
	_, err = engine.CalculatePath(0, -1)
	// 负步数现在不返回错误，而是正常处理
	// if err == nil {
	// 	t.Error("CalculatePath(0, -1) should return error")
	// }
}

// ========== Export/Import Tests ==========

func TestMapExportImport(t *testing.T) {
	original := NewMapEngine(50)
	original.SetCellType(10, CellTypeFragile)
	original.SetCellType(15, CellTypeFog)
	original.SetCellType(20, CellTypeCheckpoint)
	original.Cells[10].IsBroken = true

	// 导出
	data, err := original.Export()
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// 导入到新引擎
	loaded, err := LoadMap(data)
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}

	// 验证数据一致性
	if loaded.Length != original.Length {
		t.Errorf("loaded.Length = %d, expected %d", loaded.Length, original.Length)
	}
	if loaded.Cells[10].CellType != CellTypeFragile {
		t.Error("loaded cell 10 should be Fragile")
	}
	if !loaded.Cells[10].IsBroken {
		t.Error("loaded cell 10 should be broken")
	}
	if loaded.Cells[15].CellType != CellTypeFog {
		t.Error("loaded cell 15 should be Fog")
	}
	if loaded.Cells[20].CellType != CellTypeCheckpoint {
		t.Error("loaded cell 20 should be Checkpoint")
	}
}

func TestExportImportJSON(t *testing.T) {
	engine := NewMapEngine(10)
	engine.SetCellType(5, CellTypeFragile)

	// 导出
	data, err := engine.Export()
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// 验证 JSON 格式
	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if raw["length"] != float64(10) {
		t.Errorf("JSON length = %v, expected 10", raw["length"])
	}

	// 导入到同一引擎
	newEngine := NewMapEngine(1)
	err = newEngine.Import(data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if newEngine.Length != 10 {
		t.Errorf("newEngine.Length = %d, expected 10", newEngine.Length)
	}
	if newEngine.Cells[5].CellType != CellTypeFragile {
		t.Error("imported cell 5 should be Fragile")
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

	// 格子数量不匹配
	data, _ = json.Marshal(&MapEngine{Length: 10, Cells: make([]*MapCell, 5)})
	_, err = LoadMap(data)
	if err == nil {
		t.Error("LoadMap with mismatched cells count should return error")
	}
}

// ========== Helper Methods Tests ==========

func TestGetCellsByType(t *testing.T) {
	engine := NewMapEngine(50)
	engine.SetCellType(10, CellTypeFragile)
	engine.SetCellType(20, CellTypeFragile)
	engine.SetCellType(15, CellTypeFog)
	engine.SetCellType(25, CellTypeCheckpoint)

	fragiles := engine.GetCellsByType(CellTypeFragile)
	if len(fragiles) != 2 {
		t.Errorf("fragiles count = %d, expected 2", len(fragiles))
	}

	fogs := engine.GetCellsByType(CellTypeFog)
	if len(fogs) != 1 {
		t.Errorf("fogs count = %d, expected 1", len(fogs))
	}

	checkpoints := engine.GetCellsByType(CellTypeCheckpoint)
	if len(checkpoints) != 1 {
		t.Errorf("checkpoints count = %d, expected 1", len(checkpoints))
	}

 normals := engine.GetCellsByType(CellTypeNormal)
	if len(normals) != 46 {
		t.Errorf("normals count = %d, expected 46", len(normals))
	}
}

func TestGetLastCheckpoint(t *testing.T) {
	engine := NewMapEngine(50)
	engine.SetCellType(10, CellTypeCheckpoint)
	engine.SetCellType(20, CellTypeCheckpoint)
	engine.SetCellType(30, CellTypeCheckpoint)

	// 位置 25，最近的检查点是 20
	cp := engine.GetLastCheckpoint(25)
	if cp != 20 {
		t.Errorf("last checkpoint before 25 = %d, expected 20", cp)
	}

	// 位置 15，最近的检查点是 10
	cp = engine.GetLastCheckpoint(15)
	if cp != 10 {
		t.Errorf("last checkpoint before 15 = %d, expected 10", cp)
	}

	// 位置 5，没有检查点，返回 0
	cp = engine.GetLastCheckpoint(5)
	if cp != 0 {
		t.Errorf("last checkpoint before 5 = %d, expected 0", cp)
	}

	// 位置 50（超过地图长度），最近的检查点是 30
	cp = engine.GetLastCheckpoint(50)
	if cp != 30 {
		t.Errorf("last checkpoint before 50 = %d, expected 30", cp)
	}
}

func TestClone(t *testing.T) {
	original := NewMapEngine(50)
	original.SetCellType(10, CellTypeFragile)
	original.Cells[10].IsBroken = true

	cloned := original.Clone()

	// 修改克隆版本，不影响原版本
	cloned.Cells[10].IsBroken = false
	cloned.SetCellType(15, CellTypeFog)

	if !original.Cells[10].IsBroken {
		t.Error("original cell 10 should still be broken")
	}
	if original.Cells[15].CellType == CellTypeFog {
		t.Error("original cell 15 should not be affected")
	}

	// 验证克隆版本
	if cloned.Cells[10].IsBroken {
		t.Error("cloned cell 10 should be false after modification")
	}
	if cloned.Cells[15].CellType != CellTypeFog {
		t.Error("cloned cell 15 should be Fog")
	}
}