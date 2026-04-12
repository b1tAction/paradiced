// Package core provides core data structures for the Fated game.
// This package has no external dependencies and can be used independently.
package core

import "fmt"

// Evaluation 属性评分（0~100）
// 越低越坏，越高越好
// 0~40: 恶性（Bad）
// 41~65: 中性（Neutral）
// 66~100: 良性（Good）
type Evaluation int

const (
	// Evaluation 范围常量
	EvaluationMin    Evaluation = 0   // 最低评分
	EvaluationMax    Evaluation = 100 // 最高评分

	// 分类边界
	EvaluationBadThreshold     Evaluation = 40  // 恶性上限（≤40）
	EvaluationNeutralThreshold Evaluation = 65  // 中性上限（≤65）
	// Evaluation > 65 为良性
)

// 预定义的 Evaluation 常量（常用评分）
const (
	// 恶性评分（0~40）
	EvaluationVeryBad   Evaluation = 10  // 极恶（如雷劫）
	EvaluationBad       Evaluation = 25  // 较恶（如诅咒）
	EvaluationMildBad   Evaluation = 35  // 轻恶（如蚊虫叮咬）

	// 中性评分（41~65）
	EvaluationNeutral   Evaluation = 50  // 标准中性（如交换）
	EvaluationMixed     Evaluation = 55  // 混合效果（如尝一口）

	// 良性评分（66~100）
	EvaluationMildGood  Evaluation = 70  // 轻良（如草药）
	EvaluationGood      Evaluation = 80  // 较良（如奶茶）
	EvaluationVeryGood  Evaluation = 90  // 极良（如神眷）
	EvaluationExcellent Evaluation = 100 // 最佳
)

// IsValid 检查评分是否在有效范围内
func (e Evaluation) IsValid() bool {
	return e >= EvaluationMin && e <= EvaluationMax
}

// GetCategory 获取评分类别
func (e Evaluation) GetCategory() string {
	if e <= EvaluationBadThreshold {
		return "Bad"
	} else if e <= EvaluationNeutralThreshold {
		return "Neutral"
	}
	return "Good"
}

// IsGood 判断是否为良性
func (e Evaluation) IsGood() bool {
	return e > EvaluationNeutralThreshold
}

// IsNeutral 判断是否为中性
func (e Evaluation) IsNeutral() bool {
	return e > EvaluationBadThreshold && e <= EvaluationNeutralThreshold
}

// IsBad 判断是否为恶性
func (e Evaluation) IsBad() bool {
	return e <= EvaluationBadThreshold
}

// String 返回评分描述
func (e Evaluation) String() string {
	return fmt.Sprintf("Evaluation(%d): %s", e, e.GetCategory())
}

// Compare 比较两个评分
// 返回 1 表示当前评分更好，-1 表示更差，0 表示相同
func (e Evaluation) Compare(other Evaluation) int {
	if e > other {
		return 1
	} else if e < other {
		return -1
	}
	return 0
}