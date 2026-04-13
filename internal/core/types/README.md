# internal/core/types - Shared Basic Types

共享基础类型包，无外部依赖，可被所有子包导入。

## 概述

types 包定义了 Buff/Event/Item 共用的基础类型，包括评分系统和特殊效果枚举。

## 数据类型

### Evaluation (评分系统)

```go
type Evaluation int  // 0-100

// 分类边界
EvaluationBadThreshold     = 40   // 恶性上限（≤40）
EvaluationNeutralThreshold = 65   // 中性上限（≤65）
// Evaluation > 65 为良性

// 预定义常量
EvaluationVeryBad   = 10   // 极恶性
EvaluationBad       = 25   // 恶性
EvaluationMildBad   = 35   // 轻度恶性
EvaluationNeutral   = 50   // 中性
EvaluationMixed     = 55   // 混合效果
EvaluationMildGood  = 70   // 轻度良性
EvaluationGood      = 80   // 良性
EvaluationVeryGood  = 90   // 极良性
EvaluationExcellent = 100  // 最佳
```

### SpecialEffect (特殊效果枚举)

```go
type SpecialEffect int

// Buff 特殊效果
SpecialImmune        // 隐匿：免疫伤害/事件
SpecialReverse       // 迷途：反向移动
SpecialImmunePoison  // 辟邪：免疫毒瘴
SpecialBadEvent      // 毒瘴：每回合恶性事件
SpecialZhuQuePassive // 离火：朱雀被动

// Item 特殊效果
SpecialTeleport      // 任意门：传送
SpecialDiceSwap      // 骰子交换
SpecialDiceUpgrade   // 骰子升级
SpecialGiveLost      // 反方向的钟

// Event 特殊效果
SpecialDrawItem      // 圣遗物：抽奖
SpecialLoseItem      // 盗贼：丢失道具
SpecialSwapPosition  // 交换：位置互换
SpecialRandomBuff    // 尝一口：随机Buff
```

## 使用示例

```go
import "github.com/b1tAction/Fated/internal/core/types"

// 判断评分类型
eval := types.EvaluationGood
if eval.IsGood() {
    // 良性效果
}

// 判断特殊效果类型
se := types.SpecialImmune
if se.IsBuffEffect() {
    // Buff 特殊效果
}
```

## 测试

```bash
go test ./internal/core/types/...
```