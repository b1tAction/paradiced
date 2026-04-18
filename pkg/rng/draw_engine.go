// Package rng provides random draw engine for the Paradiced game.
// This package has no internal dependencies - provides weighted random draw.
package rng

import (
	"math/rand"

	"github.com/b1tAction/paradiced/pkg/constants"
)

// PoolType represents the pool category for drawing.
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

// ========== Pool-based Draw ==========

// DrawFromPool draws from a given EvaluatedItemPool based on LP.
// Returns the selected item's Type string (empty if no selection).
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