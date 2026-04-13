package rng

import (
	"errors"
	"math/rand"
	"sort"
	"time"
)

// WeightedItem represents a weighted draw item.
type WeightedItem struct {
	ID     string      `json:"id"`     // Unique identifier
	Type   string      `json:"type"`   // Type category
	Weight int         `json:"weight"` // Weight value
	Data   interface{} `json:"data"`   // Additional data
}

// WeightedPool represents a weighted draw pool.
type WeightedPool struct {
	Items       []WeightedItem `json:"items"`       // Draw items list
	TotalWeight int            `json:"total_weight"` // Total weight
	rng         *rand.Rand     `json:"-"`           // Random number generator
}

// NewWeightedPool creates a new draw pool.
func NewWeightedPool() *WeightedPool {
	return &WeightedPool{
		Items:       make([]WeightedItem, 0),
		TotalWeight: 0,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// NewWeightedPoolWithSeed creates a draw pool with specified seed (for testing).
func NewWeightedPoolWithSeed(seed int64) *WeightedPool {
	return &WeightedPool{
		Items:       make([]WeightedItem, 0),
		TotalWeight: 0,
		rng:         rand.New(rand.NewSource(seed)),
	}
}

// AddItem adds a draw item.
func (p *WeightedPool) AddItem(id string, itemType string, weight int, data interface{}) error {
	if weight <= 0 {
		return errors.New("weight must be positive")
	}
	if id == "" {
		return errors.New("id cannot be empty")
	}

	item := WeightedItem{
		ID:     id,
		Type:   itemType,
		Weight: weight,
		Data:   data,
	}
	p.Items = append(p.Items, item)
	p.TotalWeight += weight
	return nil
}

// RemoveItem removes a draw item.
func (p *WeightedPool) RemoveItem(id string) bool {
	for i, item := range p.Items {
		if item.ID == id {
			p.TotalWeight -= item.Weight
			p.Items = append(p.Items[:i], p.Items[i+1:]...)
			return true
		}
	}
	return false
}

// GetItem returns the item with specified ID.
func (p *WeightedPool) GetItem(id string) *WeightedItem {
	for _, item := range p.Items {
		if item.ID == id {
			return &item
		}
	}
	return nil
}

// GetItemsByType returns all items of specified type.
func (p *WeightedPool) GetItemsByType(itemType string) []WeightedItem {
	var result []WeightedItem
	for _, item := range p.Items {
		if item.Type == itemType {
			result = append(result, item)
		}
	}
	return result
}

// Draw performs a single draw.
func (p *WeightedPool) Draw() (*WeightedItem, error) {
	if len(p.Items) == 0 {
		return nil, errors.New("pool is empty")
	}
	if p.TotalWeight <= 0 {
		return nil, errors.New("total weight must be positive")
	}

	// Generate random weight value
	r := p.rng.Intn(p.TotalWeight)

	// Find corresponding item
	cumulative := 0
	for _, item := range p.Items {
		cumulative += item.Weight
		if r < cumulative {
			return &item, nil
		}
	}

	// Should not reach here theoretically
	return &p.Items[len(p.Items)-1], nil
}

// DrawWithType draws by type.
// Only draws from items of specified type.
func (p *WeightedPool) DrawWithType(itemType string) (*WeightedItem, error) {
	items := p.GetItemsByType(itemType)
	if len(items) == 0 {
		return nil, errors.New("no items of specified type")
	}

	// Calculate total weight for this type
	totalWeight := 0
	for _, item := range items {
		totalWeight += item.Weight
	}

	// Draw within this type range
	r := p.rng.Intn(totalWeight)
	cumulative := 0
	for _, item := range items {
		cumulative += item.Weight
		if r < cumulative {
			return &item, nil
		}
	}

	return &items[len(items)-1], nil
}

// DrawMultiple performs multiple draws (no repeat).
func (p *WeightedPool) DrawMultiple(count int) ([]WeightedItem, error) {
	if count > len(p.Items) {
		return nil, errors.New("count exceeds pool size")
	}
	if count <= 0 {
		return nil, errors.New("count must be positive")
	}

	// Create temporary pool for drawing
	tempPool := &WeightedPool{
		Items:       make([]WeightedItem, len(p.Items)),
		TotalWeight: p.TotalWeight,
		rng:         p.rng,
	}
	copy(tempPool.Items, p.Items)

	result := make([]WeightedItem, 0, count)
	for i := 0; i < count; i++ {
		item, err := tempPool.Draw()
		if err != nil {
			return nil, err
		}
		result = append(result, *item)
		tempPool.RemoveItem(item.ID)
	}

	return result, nil
}

// DrawMultipleWithType draws multiple by type (no repeat).
func (p *WeightedPool) DrawMultipleWithType(itemType string, count int) ([]WeightedItem, error) {
	items := p.GetItemsByType(itemType)
	if count > len(items) {
		return nil, errors.New("count exceeds type pool size")
	}
	if count <= 0 {
		return nil, errors.New("count must be positive")
	}

	// Create temporary pool
	tempPool := &WeightedPool{
		Items:       items,
		TotalWeight: 0,
		rng:         p.rng,
	}
	for _, item := range items {
		tempPool.TotalWeight += item.Weight
	}

	result := make([]WeightedItem, 0, count)
	for i := 0; i < count; i++ {
		item, err := tempPool.Draw()
		if err != nil {
			return nil, err
		}
		result = append(result, *item)
		tempPool.RemoveItem(item.ID)
	}

	return result, nil
}

// ========== Luck Adjustment System ==========

// LuckModifier represents luck adjustment configuration.
type LuckModifier struct {
	BaseWeight int     `json:"base_weight"`    // Base weight
	LuckFactor float64 `json:"luck_factor"`    // Luck influence factor
	MinWeight  int     `json:"min_weight"`     // Minimum weight
	MaxWeight  int     `json:"max_weight"`     // Maximum weight
	GoodType   string  `json:"good_type"`      // Good event type
	BadType    string  `json:"bad_type"`       // Bad event type
}

// NewLuckModifier creates a luck modifier.
func NewLuckModifier(baseWeight int, luckFactor float64) *LuckModifier {
	return &LuckModifier{
		BaseWeight: baseWeight,
		LuckFactor: luckFactor,
		MinWeight:  1,
		MaxWeight:  100,
	}
}

// CalculateWeight calculates adjusted weight based on luck.
// luck: player luck value (0~8)
// isGoodEvent: whether it's a good event
func (lm *LuckModifier) CalculateWeight(luck int, isGoodEvent bool) int {
	adjusted := float64(lm.BaseWeight)

	if isGoodEvent {
		// Luck increases good event weight
		adjusted += adjusted * lm.LuckFactor * float64(luck)
	} else {
		// Luck decreases bad event weight
		adjusted -= adjusted * lm.LuckFactor * float64(luck)
	}

	// Limit range
	result := int(adjusted)
	if result < lm.MinWeight {
		result = lm.MinWeight
	}
	if result > lm.MaxWeight {
		result = lm.MaxWeight
	}

	return result
}

// LuckAdjustedPool represents luck-adjusted draw pool.
type LuckAdjustedPool struct {
	BasePool     *WeightedPool `json:"base_pool"`     // Base draw pool
	LuckModifier *LuckModifier `json:"luck_modifier"` // Luck modifier
}

// NewLuckAdjustedPool creates a luck-adjusted draw pool.
func NewLuckAdjustedPool(basePool *WeightedPool, modifier *LuckModifier) *LuckAdjustedPool {
	return &LuckAdjustedPool{
		BasePool:     basePool,
		LuckModifier: modifier,
	}
}

// DrawWithLuck draws based on luck value.
func (lap *LuckAdjustedPool) DrawWithLuck(luck int) (*WeightedItem, error) {
	if lap.BasePool == nil {
		return nil, errors.New("base pool is nil")
	}

	// Create adjusted temporary pool
	adjustedPool := NewWeightedPoolWithSeed(lap.BasePool.rng.Int63())
	for _, item := range lap.BasePool.Items {
		isGood := item.Type == lap.LuckModifier.GoodType
		adjustedWeight := lap.LuckModifier.CalculateWeight(luck, isGood)
		adjustedPool.AddItem(item.ID, item.Type, adjustedWeight, item.Data)
	}

	return adjustedPool.Draw()
}

// ========== Draw Probability Configuration ==========

// ProbabilityConfig represents draw probability configuration.
type ProbabilityConfig struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Probabilities map[string]float64 `json:"probabilities"` // ID -> probability percentage
}

// NewProbabilityConfig creates a probability configuration.
func NewProbabilityConfig(id string, name string) *ProbabilityConfig {
	return &ProbabilityConfig{
		ID:           id,
		Name:         name,
		Probabilities: make(map[string]float64),
	}
}

// SetProbability sets probability.
func (pc *ProbabilityConfig) SetProbability(itemID string, probability float64) error {
	if probability < 0 || probability > 100 {
		return errors.New("probability must be between 0 and 100")
	}
	pc.Probabilities[itemID] = probability
	return nil
}

// Validate validates if probability sum is 100%.
func (pc *ProbabilityConfig) Validate() bool {
	total := 0.0
	for _, p := range pc.Probabilities {
		total += p
	}
	// Allow some error margin
	return total >= 99.9 && total <= 100.1
}

// ToWeightedPool converts to weighted draw pool.
func (pc *ProbabilityConfig) ToWeightedPool() (*WeightedPool, error) {
	if !pc.Validate() {
		return nil, errors.New("probabilities do not sum to 100%")
	}

	pool := NewWeightedPool()
	for itemID, probability := range pc.Probabilities {
		// Convert probability to weight (multiply by 100)
		weight := int(probability * 10)
		if weight <= 0 {
			weight = 1
		}
		pool.AddItem(itemID, "", weight, nil)
	}

	return pool, nil
}

// ========== Draw Statistics ==========

// DrawStatistics represents draw statistics.
type DrawStatistics struct {
	TotalDraws int            `json:"total_draws"` // Total draw count
	ItemCounts map[string]int `json:"item_counts"` // Item draw counts
	TypeCounts map[string]int `json:"type_counts"` // Type draw counts
}

// NewDrawStatistics creates a statistics object.
func NewDrawStatistics() *DrawStatistics {
	return &DrawStatistics{
		TotalDraws: 0,
		ItemCounts: make(map[string]int),
		TypeCounts: make(map[string]int),
	}
}

// Record records a draw.
func (ds *DrawStatistics) Record(item *WeightedItem) {
	ds.TotalDraws++
	ds.ItemCounts[item.ID]++
	ds.TypeCounts[item.Type]++
}

// GetProbability calculates actual draw probability for an item.
func (ds *DrawStatistics) GetProbability(itemID string) float64 {
	if ds.TotalDraws == 0 {
		return 0
	}
	return float64(ds.ItemCounts[itemID]) / float64(ds.TotalDraws) * 100
}

// GetTopItems returns top drawn items.
func (ds *DrawStatistics) GetTopItems(limit int) []string {
	// Sort by count
	items := make([]struct {
		id    string
		count int
	}, 0, len(ds.ItemCounts))

	for id, count := range ds.ItemCounts {
		items = append(items, struct {
			id    string
			count int
		}{id, count})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].count > items[j].count
	})

	result := make([]string, 0, limit)
	for i := 0; i < limit && i < len(items); i++ {
		result = append(result, items[i].id)
	}

	return result
}

// ========== Predefined Draw Pools ==========

// EventPool represents game event draw pool.
type EventPool struct {
	GoodPool    *WeightedPool `json:"good_pool"`    // Good event pool
	BadPool     *WeightedPool `json:"bad_pool"`     // Bad event pool
	NeutralPool *WeightedPool `json:"neutral_pool"` // Neutral event pool
}

// NewEventPool creates an event draw pool.
func NewEventPool() *EventPool {
	return &EventPool{
		GoodPool:    NewWeightedPool(),
		BadPool:     NewWeightedPool(),
		NeutralPool: NewWeightedPool(),
	}
}

// AddGoodEvent adds a good event.
func (ep *EventPool) AddGoodEvent(id string, weight int, data interface{}) error {
	return ep.GoodPool.AddItem(id, "good", weight, data)
}

// AddBadEvent adds a bad event.
func (ep *EventPool) AddBadEvent(id string, weight int, data interface{}) error {
	return ep.BadPool.AddItem(id, "bad", weight, data)
}

// AddNeutralEvent adds a neutral event.
func (ep *EventPool) AddNeutralEvent(id string, weight int, data interface{}) error {
	return ep.NeutralPool.AddItem(id, "neutral", weight, data)
}

// DrawEvent draws an event.
// luck: luck value, affects good/bad event probability
func (ep *EventPool) DrawEvent(luck int, rng *rand.Rand) (*WeightedItem, string, error) {
	// Calculate overall probability for good/bad/neutral events
	// Base probability: good 30%, bad 30%, neutral 40%
	// Luck influence: each luck point increases good event 5%, decreases bad event 5%

	goodProb := 30 + luck*5
	badProb := 30 - luck*5
	neutralProb := 40

	// Limit probability range
	if goodProb > 70 {
		goodProb = 70
	}
	if badProb < 10 {
		badProb = 10
	}

	// Create type selection pool
	typePool := NewWeightedPoolWithSeed(rng.Int63())
	typePool.AddItem("good", "type", goodProb, nil)
	typePool.AddItem("bad", "type", badProb, nil)
	typePool.AddItem("neutral", "type", neutralProb, nil)

	// Select event type
	typeItem, err := typePool.Draw()
	if err != nil {
		return nil, "", err
	}

	// Draw from corresponding pool
	switch typeItem.ID {
	case "good":
		item, err := ep.GoodPool.Draw()
		return item, "good", err
	case "bad":
		item, err := ep.BadPool.Draw()
		return item, "bad", err
	case "neutral":
		item, err := ep.NeutralPool.Draw()
		return item, "neutral", err
	}

	return nil, "", errors.New("unknown event type")
}

// ItemPool represents item draw pool.
type ItemPool struct {
	CommonPool *WeightedPool `json:"common_pool"` // Common item pool
	RarePool   *WeightedPool `json:"rare_pool"`   // Rare item pool
	EpicPool   *WeightedPool `json:"epic_pool"`   // Epic item pool
}

// NewItemPool creates an item draw pool.
func NewItemPool() *ItemPool {
	return &ItemPool{
		CommonPool: NewWeightedPool(),
		RarePool:   NewWeightedPool(),
		EpicPool:   NewWeightedPool(),
	}
}

// AddCommonItem adds a common item.
func (ip *ItemPool) AddCommonItem(id string, weight int, data interface{}) error {
	return ip.CommonPool.AddItem(id, "common", weight, data)
}

// AddRareItem adds a rare item.
func (ip *ItemPool) AddRareItem(id string, weight int, data interface{}) error {
	return ip.RarePool.AddItem(id, "rare", weight, data)
}

// AddEpicItem adds an epic item.
func (ip *ItemPool) AddEpicItem(id string, weight int, data interface{}) error {
	return ip.EpicPool.AddItem(id, "epic", weight, data)
}

// DrawItem draws an item.
// luck: luck value, affects rarity probability
func (ip *ItemPool) DrawItem(luck int, rng *rand.Rand) (*WeightedItem, string, error) {
	// Base probability: common 70%, rare 20%, epic 10%
	// Luck influence: each luck point increases rare/epic 2%

	commonProb := 70 - luck*4
	rareProb := 20 + luck*2
	epicProb := 10 + luck*2

	// Limit probability range
	if commonProb < 50 {
		commonProb = 50
	}
	if rareProb > 30 {
		rareProb = 30
	}
	if epicProb > 20 {
		epicProb = 20
	}

	// Create rarity selection pool
	rarityPool := NewWeightedPoolWithSeed(rng.Int63())
	rarityPool.AddItem("common", "rarity", commonProb, nil)
	rarityPool.AddItem("rare", "rarity", rareProb, nil)
	rarityPool.AddItem("epic", "rarity", epicProb, nil)

	// Select rarity
	rarityItem, err := rarityPool.Draw()
	if err != nil {
		return nil, "", err
	}

	// Draw from corresponding pool
	switch rarityItem.ID {
	case "common":
		item, err := ip.CommonPool.Draw()
		return item, "common", err
	case "rare":
		item, err := ip.RarePool.Draw()
		return item, "rare", err
	case "epic":
		item, err := ip.EpicPool.Draw()
		return item, "epic", err
	}

	return nil, "", errors.New("unknown rarity type")
}

// ========== Attribute-based Draw Pool ==========

// AttributeType represents attribute type (for pool classification).
type AttributeType string

const (
	AttributeGood    AttributeType = "good"    // Good attribute
	AttributeNeutral AttributeType = "neutral" // Neutral attribute
	AttributeBad     AttributeType = "bad"     // Bad attribute
)

// AttributeBasedPool represents attribute-based draw pool.
// Supports storage and drawing by attribute classification.
type AttributeBasedPool struct {
	GoodPool    *WeightedPool `json:"good_pool"`    // Good pool
	NeutralPool *WeightedPool `json:"neutral_pool"` // Neutral pool
	BadPool     *WeightedPool `json:"bad_pool"`     // Bad pool
	rng         *rand.Rand    `json:"-"`            // Random number generator
}

// NewAttributeBasedPool creates an attribute-based draw pool.
func NewAttributeBasedPool() *AttributeBasedPool {
	return &AttributeBasedPool{
		GoodPool:    NewWeightedPool(),
		NeutralPool: NewWeightedPool(),
		BadPool:     NewWeightedPool(),
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// NewAttributeBasedPoolWithSeed creates an attribute-based draw pool with specified seed (for testing).
func NewAttributeBasedPoolWithSeed(seed int64) *AttributeBasedPool {
	return &AttributeBasedPool{
		GoodPool:    NewWeightedPoolWithSeed(seed),
		NeutralPool: NewWeightedPoolWithSeed(seed + 1),
		BadPool:     NewWeightedPoolWithSeed(seed + 2),
		rng:         rand.New(rand.NewSource(seed)),
	}
}

// AddItem adds an item to specified attribute pool.
func (abp *AttributeBasedPool) AddItem(id string, attr AttributeType, weight int, data interface{}) error {
	switch attr {
	case AttributeGood:
		return abp.GoodPool.AddItem(id, string(attr), weight, data)
	case AttributeNeutral:
		return abp.NeutralPool.AddItem(id, string(attr), weight, data)
	case AttributeBad:
		return abp.BadPool.AddItem(id, string(attr), weight, data)
	}
	return errors.New("unknown attribute type")
}

// AddGoodItem adds a good item.
func (abp *AttributeBasedPool) AddGoodItem(id string, weight int, data interface{}) error {
	return abp.AddItem(id, AttributeGood, weight, data)
}

// AddNeutralItem adds a neutral item.
func (abp *AttributeBasedPool) AddNeutralItem(id string, weight int, data interface{}) error {
	return abp.AddItem(id, AttributeNeutral, weight, data)
}

// AddBadItem adds a bad item.
func (abp *AttributeBasedPool) AddBadItem(id string, weight int, data interface{}) error {
	return abp.AddItem(id, AttributeBad, weight, data)
}

// GetPoolByAttribute returns the pool for specified attribute.
func (abp *AttributeBasedPool) GetPoolByAttribute(attr AttributeType) *WeightedPool {
	switch attr {
	case AttributeGood:
		return abp.GoodPool
	case AttributeNeutral:
		return abp.NeutralPool
	case AttributeBad:
		return abp.BadPool
	}
	return nil
}

// DrawFromAttribute draws from specified attribute pool.
func (abp *AttributeBasedPool) DrawFromAttribute(attr AttributeType) (*WeightedItem, error) {
	pool := abp.GetPoolByAttribute(attr)
	if pool == nil {
		return nil, errors.New("unknown attribute type")
	}
	return pool.Draw()
}

// DrawWithLuck draws based on luck value.
// luck: luck value, affects good/bad attribute probability distribution
// Base probability: good 30%, bad 30%, neutral 40%
// Each luck point increases good 5%, decreases bad 5%
func (abp *AttributeBasedPool) DrawWithLuck(luck int) (*WeightedItem, AttributeType, error) {
	// Calculate attribute selection probability
	goodProb := 30 + luck*5
	badProb := 30 - luck*5
	neutralProb := 40

	// Limit probability range
	if goodProb > 70 {
		goodProb = 70
	}
	if badProb < 10 {
		badProb = 10
	}

	// Create attribute selection pool
	attrPool := NewWeightedPoolWithSeed(abp.rng.Int63())
	attrPool.AddItem("good", "attr", goodProb, nil)
	attrPool.AddItem("bad", "attr", badProb, nil)
	attrPool.AddItem("neutral", "attr", neutralProb, nil)

	// Select attribute
	attrItem, err := attrPool.Draw()
	if err != nil {
		return nil, "", err
	}

	attr := AttributeType(attrItem.ID)
	item, err := abp.DrawFromAttribute(attr)
	return item, attr, err
}

// DrawWithLuckAndSeed draws based on luck value and specified seed (for testing).
func (abp *AttributeBasedPool) DrawWithLuckAndSeed(luck int, seed int64) (*WeightedItem, AttributeType, error) {
	// Calculate attribute selection probability
	goodProb := 30 + luck*5
	badProb := 30 - luck*5
	neutralProb := 40

	// Limit probability range
	if goodProb > 70 {
		goodProb = 70
	}
	if badProb < 10 {
		badProb = 10
	}

	// Create attribute selection pool
	attrPool := NewWeightedPoolWithSeed(seed)
	attrPool.AddItem("good", "attr", goodProb, nil)
	attrPool.AddItem("bad", "attr", badProb, nil)
	attrPool.AddItem("neutral", "attr", neutralProb, nil)

	// Select attribute
	attrItem, err := attrPool.Draw()
	if err != nil {
		return nil, "", err
	}

	attr := AttributeType(attrItem.ID)
	pool := abp.GetPoolByAttribute(attr)
	if pool == nil {
		return nil, "", errors.New("pool not found for attribute")
	}

	item, err := pool.Draw()
	return item, attr, err
}

// GetTotalWeight returns total weight of all pools.
func (abp *AttributeBasedPool) GetTotalWeight() int {
	return abp.GoodPool.TotalWeight + abp.NeutralPool.TotalWeight + abp.BadPool.TotalWeight
}

// GetAttributeWeights returns weights of each attribute pool.
func (abp *AttributeBasedPool) GetAttributeWeights() map[AttributeType]int {
	return map[AttributeType]int{
		AttributeGood:    abp.GoodPool.TotalWeight,
		AttributeNeutral: abp.NeutralPool.TotalWeight,
		AttributeBad:     abp.BadPool.TotalWeight,
	}
}

// Clear clears all pools.
func (abp *AttributeBasedPool) Clear() {
	abp.GoodPool = NewWeightedPool()
	abp.NeutralPool = NewWeightedPool()
	abp.BadPool = NewWeightedPool()
}