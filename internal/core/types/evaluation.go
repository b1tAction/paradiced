// Package types provides shared basic types for core packages.
// This package has no dependencies and can be imported by all subpackages.
package types

import "fmt"

// Evaluation represents attribute score (0~100).
// Lower is worse, higher is better.
// 0~40: Bad (恶性)
// 41~65: Neutral (中性)
// 66~100: Good (良性)
type Evaluation int

const (
	// Evaluation range constants
	EvaluationMin    Evaluation = 0   // Minimum score
	EvaluationMax    Evaluation = 100 // Maximum score

	// Category thresholds
	EvaluationBadThreshold     Evaluation = 40  // Bad upper bound (≤40)
	EvaluationNeutralThreshold Evaluation = 65  // Neutral upper bound (≤65)
	// Evaluation > 65 is Good
)

// Predefined Evaluation constants (common scores).
const (
	// Bad scores (0~40)
	EvaluationVeryBad   Evaluation = 10  // Very bad
	EvaluationBad       Evaluation = 25  // Bad
	EvaluationMildBad   Evaluation = 35  // Mild bad

	// Neutral scores (41~65)
	EvaluationNeutral   Evaluation = 50  // Standard neutral
	EvaluationMixed     Evaluation = 55  // Mixed effect

	// Good scores (66~100)
	EvaluationMildGood  Evaluation = 70  // Mild good
	EvaluationGood      Evaluation = 80  // Good
	EvaluationVeryGood  Evaluation = 90  // Very good
	EvaluationExcellent Evaluation = 100 // Best
)

// IsValid checks if the score is in valid range.
func (e Evaluation) IsValid() bool {
	return e >= EvaluationMin && e <= EvaluationMax
}

// GetCategory returns the evaluation category.
func (e Evaluation) GetCategory() string {
	if e <= EvaluationBadThreshold {
		return "Bad"
	} else if e <= EvaluationNeutralThreshold {
		return "Neutral"
	}
	return "Good"
}

// IsGood checks if the evaluation is good.
func (e Evaluation) IsGood() bool {
	return e > EvaluationNeutralThreshold
}

// IsNeutral checks if the evaluation is neutral.
func (e Evaluation) IsNeutral() bool {
	return e > EvaluationBadThreshold && e <= EvaluationNeutralThreshold
}

// IsBad checks if the evaluation is bad.
func (e Evaluation) IsBad() bool {
	return e <= EvaluationBadThreshold
}

// String returns the evaluation description.
func (e Evaluation) String() string {
	return fmt.Sprintf("Evaluation(%d): %s", e, e.GetCategory())
}

// Compare compares two evaluations.
// Returns 1 if current is better, -1 if worse, 0 if equal.
func (e Evaluation) Compare(other Evaluation) int {
	if e > other {
		return 1
	} else if e < other {
		return -1
	}
	return 0
}