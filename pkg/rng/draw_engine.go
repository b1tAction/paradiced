// Package rng provides random draw engine for the Paradiced game.
// This package has no internal dependencies - provides weighted random draw.
package rng

import (
	"math/rand"

	"github.com/b1tAction/paradiced/pkg/constants"
)

// ========== PoolType (Legacy) ==========

// PoolType represents the pool category for drawing.
// Used internally for weight calculation.
type PoolType int

const (
	PoolTypeGood    PoolType = iota // Good pool (Evaluation > 65)
	PoolTypeNeutral                 // Neutral pool (Evaluation 41-65)
	PoolTypeBad                     // Bad pool (Evaluation <= 40)
)

func (pt PoolType) String() string {
	switch pt {
	case PoolTypeGood:
		return "Good"
	case PoolTypeNeutral:
		return "Neutral"
	case PoolTypeBad:
		return "Bad"
	default:
		return "Unknown"
	}
}

// EvaluatedItem represents an item with its Evaluation for weighted draw.
// Type uses string to accommodate EventType/BuffType/ItemType (all are string aliases).
type EvaluatedItem struct {
	Type string            // Type identifier (snake_case)
	Eval constants.Evaluation
}

// EvaluatedItemPool represents a pool of items with their evaluations.
// Deprecated: Use []EvaluatedItem directly.
type EvaluatedItemPool struct {
	Items []EvaluatedItem
}

// DrawEngine is the random draw engine.
type DrawEngine struct {
	rng *rand.Rand
}

// NewDrawEngine creates a new draw engine.
func NewDrawEngine(rng *rand.Rand) *DrawEngine {
	return &DrawEngine{rng: rng}
}

// ========== LP Influence Constants ==========

const (
	lpMin        = 0
	lpMax        = 8
	lpEvalFactor = 0.1
)

// ========== Weighted Draw ==========

type weightedItem struct {
	item   EvaluatedItem
	weight float64
}

func (e *DrawEngine) weightedDraw(items []weightedItem) *EvaluatedItem {
	if len(items) == 0 {
		return nil
	}

	totalWeight := 0.0
	for _, item := range items {
		totalWeight += item.weight
	}

	if totalWeight <= 0 {
		return &items[e.rng.Intn(len(items))].item
	}

	r := e.rng.Float64() * totalWeight
	cumulative := 0.0
	for _, item := range items {
		cumulative += item.weight
		if r < cumulative {
			return &item.item
		}
	}

	return &items[len(items)-1].item
}

// ========== Weight Calculation ==========

func calculateGoodWeight(eval constants.Evaluation, lp int) float64 {
	baseWeight := float64(eval) / 100.0
	lpFactorTotal := lpEvalFactor * float64(lp) * baseWeight
	return baseWeight * (1 + lpFactorTotal)
}

func calculateBadWeight(eval constants.Evaluation, lp int) float64 {
	baseWeight := float64(eval) / float64(constants.EvaluationBadThreshold)
	lpFactorTotal := lpEvalFactor * float64(lp) * baseWeight
	return baseWeight * (1 + lpFactorTotal)
}

func calculateNeutralWeight(eval constants.Evaluation) float64 {
	return float64(eval) / 100.0
}

// ========== Pool-based Draw (Legacy) ==========

// DrawFromPool draws from a given EvaluatedItemPool based on LP.
// Returns the selected item's Type string (empty if no selection).
// Deprecated: Use DrawFromCategory instead.
func (e *DrawEngine) DrawFromPool(pool *EvaluatedItemPool, poolType PoolType, lp int) string {
	if pool == nil || len(pool.Items) == 0 {
		return ""
	}

	// Clamp LP
	if lp < lpMin {
		lp = lpMin
	}
	if lp > lpMax {
		lp = lpMax
	}

	// Build weighted items
	items := make([]weightedItem, 0, len(pool.Items))
	for _, item := range pool.Items {
		weight := 0.0
		switch poolType {
		case PoolTypeGood:
			weight = calculateGoodWeight(item.Eval, lp)
		case PoolTypeBad:
			weight = calculateBadWeight(item.Eval, lp)
		case PoolTypeNeutral:
			weight = calculateNeutralWeight(item.Eval)
		default:
			weight = calculateGoodWeight(item.Eval, lp)
		}
		items = append(items, weightedItem{item: item, weight: weight})
	}

	selected := e.weightedDraw(items)
	if selected == nil {
		return ""
	}
	return selected.Type
}

// ========== Category-based Draw (New) ==========

// CategoryDrawResult represents the result of a category-based draw.
type CategoryDrawResult struct {
	Item    *EvaluatedItem // Selected item (nil if failed)
}

// DrawWithProb draws from items based on probability weights.
// Parameters:
//   - items: all available items (will be filtered by category or used as-is)
//   - probGood, probNeutral, probBad: probability weights for selecting pool
//   - lp: player's luck points for weight calculation
//
// The method works as follows:
// 1. If total probability (probGood+probNeutral+probBad) < 1.0:
//    - With probability (1.0 - total), draw from ALL items (no filtering)
//    - With remaining probability, proceed to step 2
// 2. Select a pool (Good/Neutral/Bad) based on normalized probability weights
// 3. Filter items by selected pool and draw using LP-based weights
//
// Returns CategoryDrawResult with selected item.
func (e *DrawEngine) DrawWithProb(
	items []*EvaluatedItem,
	probGood, probNeutral, probBad float64,
	lp int,
) *CategoryDrawResult {
	if len(items) == 0 {
		return &CategoryDrawResult{Item: nil}
	}

	// Clamp LP
	if lp < lpMin {
		lp = lpMin
	}
	if lp > lpMax {
		lp = lpMax
	}

	total := probGood + probNeutral + probBad

	// If total < 1.0, there's a chance to draw from ALL items (no filtering)
	if total < 1.0 {
		remainingProb := 1.0 - total
		if e.rng.Float64() < remainingProb {
			// Draw from all items (no pool filtering)
			return e.drawFromAllItems(items, lp)
		}
		// Otherwise, proceed with pool-based draw (renormalize)
		// If total is 0, we'll handle it in selectPoolByProbability
	}

	// If total is 0, use equal distribution (handled by selectPoolByProbability)
	if total <= 0 {
		return e.drawFromAllItems(items, lp)
	}

	// Select pool based on probability weights (normalized)
	selectedPool := e.selectPoolByProbability(probGood, probNeutral, probBad)

	// Filter items by selected pool
	var filtered []*EvaluatedItem
	switch selectedPool {
	case PoolTypeGood:
		for _, item := range items {
			if item.Eval.IsGood() {
				filtered = append(filtered, item)
			}
		}
	case PoolTypeBad:
		for _, item := range items {
			if item.Eval.IsBad() {
				filtered = append(filtered, item)
			}
		}
	case PoolTypeNeutral:
		for _, item := range items {
			if item.Eval.IsNeutral() {
				filtered = append(filtered, item)
			}
		}
	}

	if len(filtered) == 0 {
		// Fallback: use all items if filtered is empty
		return e.drawFromAllItems(items, lp)
	}

	// Build weighted items based on selected pool
	weightedItems := make([]weightedItem, 0, len(filtered))
	for _, item := range filtered {
		weight := calculateWeight(item.Eval, lp, selectedPool)
		weightedItems = append(weightedItems, weightedItem{item: *item, weight: weight})
	}

	selected := e.weightedDraw(weightedItems)
	return &CategoryDrawResult{Item: selected}
}

// drawFromAllItems draws from all items without pool filtering.
// Uses LP-based weights with averaged weight calculation.
func (e *DrawEngine) drawFromAllItems(items []*EvaluatedItem, lp int) *CategoryDrawResult {
	if len(items) == 0 {
		return &CategoryDrawResult{Item: nil}
	}

	// Build weighted items using average weight across all pools
	weightedItems := make([]weightedItem, 0, len(items))
	for _, item := range items {
		// Use average weight from all three pools
		weightGood := calculateGoodWeight(item.Eval, lp)
		weightBad := calculateBadWeight(item.Eval, lp)
		weightNeutral := calculateNeutralWeight(item.Eval)
		weight := (weightGood + weightBad + weightNeutral) / 3.0
		weightedItems = append(weightedItems, weightedItem{item: *item, weight: weight})
	}

	selected := e.weightedDraw(weightedItems)
	return &CategoryDrawResult{Item: selected}
}

// selectPoolByProbability selects a pool based on probability weights.
// Parameters:
//   - probGood, probNeutral, probBad: probability weights for each pool
//
// If all probabilities are 0, uses equal distribution (1/3 each).
// If sum < 1.0, remaining probability is distributed proportionally.
func (e *DrawEngine) selectPoolByProbability(probGood, probNeutral, probBad float64) PoolType {
	total := probGood + probNeutral + probBad

	// If all probabilities are 0, use equal distribution
	if total <= 0 {
		r := e.rng.Float64()
		if r < 0.33 {
			return PoolTypeGood
		} else if r < 0.66 {
			return PoolTypeNeutral
		}
		return PoolTypeBad
	}

	// Normalize probabilities
	normGood := probGood / total
	normNeutral := probNeutral / total

	r := e.rng.Float64()
	if r < normGood {
		return PoolTypeGood
	} else if r < normGood+normNeutral {
		return PoolTypeNeutral
	}
	return PoolTypeBad
}

// calculateWeight calculates weight based on evaluation and pool type.
func calculateWeight(eval constants.Evaluation, lp int, poolType PoolType) float64 {
	switch poolType {
	case PoolTypeGood:
		return calculateGoodWeight(eval, lp)
	case PoolTypeBad:
		return calculateBadWeight(eval, lp)
	case PoolTypeNeutral:
		return calculateNeutralWeight(eval)
	default:
		return calculateGoodWeight(eval, lp)
	}
}

// ========== Utility Methods ==========

func (e *DrawEngine) GetRNG() *rand.Rand {
	return e.rng
}

func (e *DrawEngine) Intn(n int) int {
	return e.rng.Intn(n)
}

func (e *DrawEngine) Float64() float64 {
	return e.rng.Float64()
}