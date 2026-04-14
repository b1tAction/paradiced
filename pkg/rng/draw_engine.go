// Package rng provides random draw engine for the Fated game.
package rng

import (
	"math/rand"

	"github.com/b1tAction/fated/internal/core/buff"
	"github.com/b1tAction/fated/internal/core/event"
	"github.com/b1tAction/fated/internal/core/item"
	"github.com/b1tAction/fated/pkg/constants"
)

// PoolType represents the pool category for drawing.
// Used for Event/Buff draws to specify which pool to draw from.
// The pool type is determined by game logic (e.g., a bad event needs to draw from Bad pool).
type PoolType int

const (
	// PoolTypeGood draws from Good pool (Evaluation > 65).
	// LP affects weight: higher LP favors higher Evaluation items.
	PoolTypeGood PoolType = iota
	// PoolTypeNeutral draws from Neutral pool (Evaluation 41-65).
	// LP has minimal effect on Neutral draws.
	PoolTypeNeutral
	// PoolTypeBad draws from Bad pool (Evaluation <= 40).
	// LP affects weight: higher LP favors higher Evaluation (less bad outcomes).
	PoolTypeBad
)

// String returns the pool type name.
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

// ToCategoryString returns the category string for registry lookup.
func (pt PoolType) ToCategoryString() string {
	return pt.String()
}

// DrawEngine is the core random draw engine for the game.
// It uses a single rand.Rand instance provided by the Game.
type DrawEngine struct {
	rng *rand.Rand
}

// NewDrawEngine creates a new draw engine with the provided random source.
func NewDrawEngine(rng *rand.Rand) *DrawEngine {
	return &DrawEngine{rng: rng}
}

// ========== LP Influence Constants ==========

const (
	// LP range
	lpMin = 0
	lpMax = 8

	// LP influence factor on Evaluation weight
	lpEvalFactor = 0.1 // each LP adds 0.1 × baseWeight to weight multiplier
)

// ========== Weighted Draw Helpers ==========

// weightedItem represents an item with its weighted probability.
type weightedItem struct {
	id     interface{}
	eval   constants.Evaluation
	weight float64
}

// weightedDraw performs a weighted random draw from the given items.
// Returns the selected item's id.
func (e *DrawEngine) weightedDraw(items []weightedItem) interface{} {
	if len(items) == 0 {
		return nil
	}

	// Calculate total weight
	totalWeight := 0.0
	for _, item := range items {
		totalWeight += item.weight
	}

	if totalWeight <= 0 {
		// Equal weight fallback
		return items[e.rng.Intn(len(items))].id
	}

	// Draw
	r := e.rng.Float64() * totalWeight
	cumulative := 0.0
	for _, item := range items {
		cumulative += item.weight
		if r < cumulative {
			return item.id
		}
	}

	// Fallback to last item
	return items[len(items)-1].id
}

// calculateGoodWeight calculates weight for Good pool items.
// Higher LP increases weight more for higher Evaluation items.
// Formula: weight = baseWeight × (1 + lpFactor × LP × baseWeight)
func calculateGoodWeight(eval constants.Evaluation, lp int) float64 {
	// Base weight from Evaluation (normalized to 0-1)
	baseWeight := float64(eval) / 100.0

	// LP bonus scales with Evaluation: higher LP favors higher Evaluation
	lpFactorTotal := lpEvalFactor * float64(lp) * baseWeight

	return baseWeight * (1 + lpFactorTotal)
}

// calculateBadWeight calculates weight for Bad pool items.
// Higher LP increases weight more for higher Evaluation items (less bad outcomes).
// Formula: weight = baseWeight × (1 + lpFactor × LP × baseWeight)
func calculateBadWeight(eval constants.Evaluation, lp int) float64 {
	// Base weight from Evaluation relative to Bad threshold
	// Higher Evaluation (closer to 40) = less bad = higher weight
	baseWeight := float64(eval) / float64(constants.EvaluationBadThreshold)

	// LP bonus scales with base weight: higher LP favors less bad outcomes
	lpFactorTotal := lpEvalFactor * float64(lp) * baseWeight

	return baseWeight * (1 + lpFactorTotal)
}

// calculateNeutralWeight calculates weight for Neutral pool items.
// LP has minimal effect on Neutral draws.
func calculateNeutralWeight(eval constants.Evaluation) float64 {
	// Simple base weight from Evaluation
	return float64(eval) / 100.0
}

// ========== Event Draw ==========

// DrawEvent draws an Event from the specified pool based on LP.
//
// poolType: the pool to draw from (Good/Neutral/Bad), determined by game logic
// lp: player's luck points (0-8), affects Evaluation weight within the pool
//
// Returns the drawn EventType.
func (e *DrawEngine) DrawEvent(poolType PoolType, lp int) constants.EventType {
	// Clamp LP
	if lp < lpMin {
		lp = lpMin
	}
	if lp > lpMax {
		lp = lpMax
	}

	// Get event types from registry for the specified pool
	eventTypes := event.GetEventTypesByCategory(poolType.ToCategoryString())
	if len(eventTypes) == 0 {
		// Fallback to all events
		eventTypes = event.GetAllEventTypes()
	}

	// Build weighted items
	items := make([]weightedItem, 0, len(eventTypes))
	for _, et := range eventTypes {
		eval := event.GetEventEvaluation(et)
		weight := 0.0

		switch poolType {
		case PoolTypeGood:
			weight = calculateGoodWeight(eval, lp)
		case PoolTypeBad:
			weight = calculateBadWeight(eval, lp)
		case PoolTypeNeutral:
			weight = calculateNeutralWeight(eval)
		default:
			weight = calculateGoodWeight(eval, lp)
		}

		items = append(items, weightedItem{
			id:     et,
			eval:   eval,
			weight: weight,
		})
	}

	// Draw
	selected := e.weightedDraw(items)
	if selected == nil {
		return constants.EventTypeNone
	}

	return selected.(constants.EventType)
}

// ========== Buff Draw ==========

// DrawBuff draws a Buff from the specified pool based on LP.
//
// poolType: the pool to draw from (Good/Neutral/Bad), determined by game logic
// lp: player's luck points (0-8), affects Evaluation weight within the pool
//
// Returns the drawn BuffType.
func (e *DrawEngine) DrawBuff(poolType PoolType, lp int) constants.BuffType {
	// Clamp LP
	if lp < lpMin {
		lp = lpMin
	}
	if lp > lpMax {
		lp = lpMax
	}

	// Get buff types from registry for the specified pool
	buffTypes := buff.GetBuffTypesByCategory(poolType.ToCategoryString())
	if len(buffTypes) == 0 {
		// Fallback to all buffs
		buffTypes = buff.GetAllBuffTypes()
	}

	// Build weighted items
	items := make([]weightedItem, 0, len(buffTypes))
	for _, bt := range buffTypes {
		eval := buff.GetBuffEvaluation(bt)
		weight := 0.0

		switch poolType {
		case PoolTypeGood:
			weight = calculateGoodWeight(eval, lp)
		case PoolTypeBad:
			weight = calculateBadWeight(eval, lp)
		case PoolTypeNeutral:
			weight = calculateNeutralWeight(eval)
		default:
			weight = calculateGoodWeight(eval, lp)
		}

		items = append(items, weightedItem{
			id:     bt,
			eval:   eval,
			weight: weight,
		})
	}

	// Draw
	selected := e.weightedDraw(items)
	if selected == nil {
		return constants.BuffTypeNone
	}

	return selected.(constants.BuffType)
}

// ========== Item Draw ==========

// DrawItem draws an Item based on LP.
// Items are always Good (players actively use them).
// LP affects Evaluation weight - higher LP favors better items.
//
// lp: player's luck points (0-8), affects draw outcome
//
// Returns the drawn ItemType.
func (e *DrawEngine) DrawItem(lp int) constants.ItemType {
	// Clamp LP
	if lp < lpMin {
		lp = lpMin
	}
	if lp > lpMax {
		lp = lpMax
	}

	// Get all item types (all are Good)
	itemTypes := item.GetAllItemTypes()
	if len(itemTypes) == 0 {
		return constants.ItemTypeNone
	}

	// Build weighted items with Good weight calculation
	items := make([]weightedItem, 0, len(itemTypes))
	for _, it := range itemTypes {
		eval := item.GetItemEvaluation(it)
		weight := calculateGoodWeight(eval, lp)

		items = append(items, weightedItem{
			id:     it,
			eval:   eval,
			weight: weight,
		})
	}

	// Draw
	selected := e.weightedDraw(items)
	if selected == nil {
		return constants.ItemTypeNone
	}

	return selected.(constants.ItemType)
}

// ========== Utility Methods ==========

// GetRNG returns the underlying rand.Rand instance.
// Useful for other random operations in the game.
func (e *DrawEngine) GetRNG() *rand.Rand {
	return e.rng
}

// Intn returns a random integer in range [0, n).
func (e *DrawEngine) Intn(n int) int {
	return e.rng.Intn(n)
}

// Float64 returns a random float64 in range [0.0, 1.0).
func (e *DrawEngine) Float64() float64 {
	return e.rng.Float64()
}
