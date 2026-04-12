# MapEngine (地图引擎) 实现文档

## 概述

MapEngine 是《命运骰子》游戏的核心地图管理模块，负责线性地图的生成、格子状态管理和移动路径计算。

## 数据结构

### CellType (格子类型枚举)

```go
type CellType int

const (
    CellTypeNormal     CellType = iota // 普通格子
    CellTypeFragile                    // 易碎格子
    CellTypeFog                        // 迷雾格子
    CellTypeCheckpoint                 // 检查点
    CellTypeBoss                       // Boss格子（终点）
)
```

| 类型 | 说明 | 特殊行为 |
|------|------|----------|
| Normal | 普通格子 | 无特殊效果 |
| Fragile | 易碎格子 | 首次踩上去会碎裂并掉落，后续玩家可跳过 |
| Fog | 迷雾格子 | 首位玩家经过时激活迷雾区域，区域内玩家受毒瘴buff影响 |
| Checkpoint | 检查点 | 玩家死亡后回城的位置，可设置宝箱道具刷新 |
| Boss | Boss格子 | 地图终点，击败Boss获胜 |

### MapCell (地图格子)

```go
type MapCell struct {
    Index     int      // 坐标序号（0~N）
    CellType  CellType // 格子类型
    IsBroken  bool     // 是否已被踩碎（仅 Fragile 有效）
    EventID   string   // 格子关联的事件ID
    FogActive bool     // 迷雾是否已激活（仅 Fog 有效）
}
```

### MapEngine (地图引擎)

```go
type MapEngine struct {
    Cells      []*MapCell // 地图格子数组
    Length     int        // 地图总长度
    StartIndex int        // 起点索引（默认0）
    EndIndex   int        // 终点索引
}
```

## 核心功能

### 1. 地图生成

```go
// 创建长度为50的线性地图
engine := NewMapEngine(50)

// 配置特定格子类型
configs := map[int]CellType{
    10: CellTypeFragile,    // 位置10为易碎格子
    15: CellTypeFog,        // 位置15为迷雾
    20: CellTypeCheckpoint, // 位置20为检查点
    49: CellTypeBoss,       // 位置49为终点Boss
}
engine.GenerateLinearMap(configs)
```

### 2. 路径计算 (CalculatePath)

**核心逻辑**：
- 逐格计算移动路径
- 首次踩上未碎 Fragile 格子 → 立即中断，格子碎裂，玩家掉落
- 经过已碎 Fragile 格子 → 正常跳过，不影响移动
- 经过 Fog 格子 → 自动激活迷雾区域
- 到达终点 → 设置 ReachedEnd 标志

```go
result, err := engine.CalculatePath(startPosition, diceSteps)

// PathResult 结构
type PathResult struct {
    StartIndex  int   // 起始位置
    TargetIndex int   // 实际到达位置
    Path        []int // 经过的格子索引列表
    Interrupted bool  // 是否被中断
    FellDown    bool  // 是否掉落
    ReachedEnd  bool  // 是否到达终点
}
```

**特殊场景处理**：
| 场景 | 处理方式 |
|------|----------|
| 从 Fragile 格子开始移动 | 不触发掉落（已在上面） |
| 目标恰好是未碎 Fragile | 到达后碎裂并掉落 |
| 目标恰好是已碎 Fragile | 无法到达，停在上一格 |
| 移动超过终点 | 自动停在终点 |

### 3. 地图导入/导出

```go
// 导出为 JSON
data, err := engine.Export()

// 从 JSON 加载
loaded, err := LoadMap(data)

// 导入到现有引擎
engine.Import(data)
```

JSON 格式示例：
```json
{
  "cells": [
    {"index": 0, "cell_type": 0, "is_broken": false, "event_id": "", "fog_active": false},
    {"index": 10, "cell_type": 1, "is_broken": true, "event_id": "", "fog_active": false}
  ],
  "length": 50,
  "start_index": 0,
  "end_index": 49
}
```

### 4. 辅助方法

| 方法 | 功能 |
|------|------|
| `GetCell(index)` | 获取指定位置的格子 |
| `SetCellType(index, type)` | 设置格子类型 |
| `BreakFragile(index)` | 破碎易碎格子 |
| `ActivateFog(index)` | 激活迷雾格子 |
| `GetCellsByType(type)` | 获取指定类型的所有格子 |
| `GetLastCheckpoint(position)` | 获取最近的检查点位置 |
| `Clone()` | 克隆地图引擎 |

## 测试覆盖

测试文件：`internal/game/map_engine_test.go`

| 测试类 | 覆盖内容 |
|--------|----------|
| CellTypeTest | 类型字符串转换、有效性验证 |
| MapCellTest | 格子创建、初始状态验证 |
| MapEngineTest | 地图生成、长度边界、格子类型配置 |
| CalculatePathTest | 正常移动、终点到达、Fragile中断/跳过、迷雾激活 |
| ExportImportTest | JSON序列化、数据一致性、异常处理 |
| HelperMethodTest | 类型筛选、检查点查找、克隆功能 |

## 后续扩展

1. **捷径机制**：传送阵实现（先发玩家经过后生成）
2. **地图编辑器**：可视化配置格子分布
3. **动态事件**：格子关联事件ID的触发逻辑

## 文件结构

```
internal/game/
├── cell.go           # 格子类型定义 (28行)
├── map_engine.go     # 地图引擎实现 (185行)
└── map_engine_test.go # 单元测试 (415行)
```