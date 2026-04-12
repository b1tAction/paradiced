# internal/gamemap - Map System

地图系统包，管理游戏地图的生成和路径计算。

## 功能

### Cell 类型

- `Normal`: 普通格子
- `Fragile`: 易碎格子（踩过后会碎掉）
- `Fog`: 迷雾格子（激活后影响区域内玩家）
- `Checkpoint`: 检查点（死亡后回城位置）
- `Boss`: Boss 格子（终点）

### MapEngine

- 地图生成
- 路径计算（考虑 Fragile/Fog）
- 检查点管理
- 导入/导出

## 文件结构

```
internal/gamemap/
├── cell.go   # Cell 类型定义
└── engine.go # MapEngine 地图引擎
```

## 使用示例

```go
// 创建地图
mapEngine := gamemap.NewMapEngine(100)

// 配置特定格子
configs := map[int]gamemap.CellType{
    20: gamemap.CellTypeFragile,
    50: gamemap.CellTypeCheckpoint,
    99: gamemap.CellTypeBoss,
}
mapEngine.GenerateLinearMap(configs)

// 计算路径
newPos, fell, err := mapEngine.CalculatePath(10, 5, player)

// 获取检查点
checkpoint := mapEngine.GetLastCheckpoint(60)
```

## 注意

包名为 `gamemap` 而不是 `map`，因为 `map` 是 Go 保留关键字。