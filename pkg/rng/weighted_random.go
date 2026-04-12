package rng

import (
	"errors"
	"math/rand"
	"sort"
	"time"
)

// WeightedItem 带权重的抽卡项
type WeightedItem struct {
	ID     string      `json:"id"`     // 唯一标识
	Type   string      `json:"type"`   // 类型分类
	Weight int         `json:"weight"` // 权重值
	Data   interface{} `json:"data"`   // 附加数据
}

// WeightedPool 带权重的抽卡池
type WeightedPool struct {
	Items       []WeightedItem `json:"items"`       // 抽卡项列表
	TotalWeight int            `json:"total_weight"` // 总权重
	rng         *rand.Rand     `json:"-"`           // 随机数生成器
}

// NewWeightedPool 创建新的抽卡池
func NewWeightedPool() *WeightedPool {
	return &WeightedPool{
		Items:       make([]WeightedItem, 0),
		TotalWeight: 0,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// NewWeightedPoolWithSeed 创建带指定种子随机源的抽卡池（用于测试）
func NewWeightedPoolWithSeed(seed int64) *WeightedPool {
	return &WeightedPool{
		Items:       make([]WeightedItem, 0),
		TotalWeight: 0,
		rng:         rand.New(rand.NewSource(seed)),
	}
}

// AddItem 添加抽卡项
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

// RemoveItem 移除抽卡项
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

// GetItem 获取指定 ID 的项
func (p *WeightedPool) GetItem(id string) *WeightedItem {
	for _, item := range p.Items {
		if item.ID == id {
			return &item
		}
	}
	return nil
}

// GetItemsByType 获取指定类型的所有项
func (p *WeightedPool) GetItemsByType(itemType string) []WeightedItem {
	var result []WeightedItem
	for _, item := range p.Items {
		if item.Type == itemType {
			result = append(result, item)
		}
	}
	return result
}

// Draw 执行一次抽卡
func (p *WeightedPool) Draw() (*WeightedItem, error) {
	if len(p.Items) == 0 {
		return nil, errors.New("pool is empty")
	}
	if p.TotalWeight <= 0 {
		return nil, errors.New("total weight must be positive")
	}

	// 生成随机权重值
	r := p.rng.Intn(p.TotalWeight)

	// 找到对应的项
 cumulative := 0
	for _, item := range p.Items {
	 cumulative += item.Weight
		if r < cumulative {
			return &item, nil
		}
	}

	// 理论上不应该到达这里
	return &p.Items[len(p.Items)-1], nil
}

// DrawWithType 按类型抽卡
// 只在指定类型的项中进行抽取
func (p *WeightedPool) DrawWithType(itemType string) (*WeightedItem, error) {
	items := p.GetItemsByType(itemType)
	if len(items) == 0 {
		return nil, errors.New("no items of specified type")
	}

	// 计算该类型的总权重
	totalWeight := 0
	for _, item := range items {
		totalWeight += item.Weight
	}

	// 在该类型范围内抽取
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

// DrawMultiple 执行多次抽卡（不重复）
func (p *WeightedPool) DrawMultiple(count int) ([]WeightedItem, error) {
	if count > len(p.Items) {
		return nil, errors.New("count exceeds pool size")
	}
	if count <= 0 {
		return nil, errors.New("count must be positive")
	}

	// 创建临时池用于抽取
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

// DrawMultipleWithType 按类型多次抽卡（不重复）
func (p *WeightedPool) DrawMultipleWithType(itemType string, count int) ([]WeightedItem, error) {
	items := p.GetItemsByType(itemType)
	if count > len(items) {
		return nil, errors.New("count exceeds type pool size")
	}
	if count <= 0 {
		return nil, errors.New("count must be positive")
	}

	// 创建临时池
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

// ========== 幸运值调节系统 ==========

// LuckModifier 幸运值调节配置
type LuckModifier struct {
	BaseWeight    int     `json:"base_weight"`    // 基础权重
	LuckFactor    float64 `json:"luck_factor"`    // 幸运值影响因子
	MinWeight     int     `json:"min_weight"`     // 最小权重
	MaxWeight     int     `json:"max_weight"`     // 最大权重
	GoodType      string  `json:"good_type"`      // 好事件类型
	BadType       string  `json:"bad_type"`       // 坏事件类型
}

// NewLuckModifier 创建幸运值调节器
func NewLuckModifier(baseWeight int, luckFactor float64) *LuckModifier {
	return &LuckModifier{
		BaseWeight: baseWeight,
		LuckFactor: luckFactor,
		MinWeight:  1,
		MaxWeight:  100,
	}
}

// CalculateWeight 根据幸运值计算调整后的权重
// luck: 玩家幸运值 (0~8)
// isGoodEvent: 是否为良性事件
func (lm *LuckModifier) CalculateWeight(luck int, isGoodEvent bool) int {
	adjusted := float64(lm.BaseWeight)

	if isGoodEvent {
		// 幸运值增加良性事件权重
		adjusted += adjusted * lm.LuckFactor * float64(luck)
	} else {
		// 幸运值减少恶性事件权重
		adjusted -= adjusted * lm.LuckFactor * float64(luck)
	}

	// 限制范围
	result := int(adjusted)
	if result < lm.MinWeight {
		result = lm.MinWeight
	}
	if result > lm.MaxWeight {
		result = lm.MaxWeight
	}

	return result
}

// LuckAdjustedPool 幸运值调节后的抽卡池
type LuckAdjustedPool struct {
	BasePool     *WeightedPool `json:"base_pool"`     // 基础抽卡池
	LuckModifier *LuckModifier `json:"luck_modifier"` // 幸运值调节器
}

// NewLuckAdjustedPool 创建幸运值调节抽卡池
func NewLuckAdjustedPool(basePool *WeightedPool, modifier *LuckModifier) *LuckAdjustedPool {
	return &LuckAdjustedPool{
		BasePool:     basePool,
		LuckModifier: modifier,
	}
}

// DrawWithLuck 根据幸运值抽卡
func (lap *LuckAdjustedPool) DrawWithLuck(luck int) (*WeightedItem, error) {
	if lap.BasePool == nil {
		return nil, errors.New("base pool is nil")
	}

	// 创建调节后的临时池
	adjustedPool := NewWeightedPoolWithSeed(lap.BasePool.rng.Int63())
	for _, item := range lap.BasePool.Items {
		isGood := item.Type == lap.LuckModifier.GoodType
		adjustedWeight := lap.LuckModifier.CalculateWeight(luck, isGood)
		adjustedPool.AddItem(item.ID, item.Type, adjustedWeight, item.Data)
	}

	return adjustedPool.Draw()
}

// ========== 抽卡概率配置 ==========

// ProbabilityConfig 抽卡概率配置
type ProbabilityConfig struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Probabilities map[string]float64 `json:"probabilities"` // ID -> 概率百分比
}

// NewProbabilityConfig 创建概率配置
func NewProbabilityConfig(id string, name string) *ProbabilityConfig {
	return &ProbabilityConfig{
		ID:           id,
		Name:         name,
		Probabilities: make(map[string]float64),
	}
}

// SetProbability 设置概率
func (pc *ProbabilityConfig) SetProbability(itemID string, probability float64) error {
	if probability < 0 || probability > 100 {
		return errors.New("probability must be between 0 and 100")
	}
	pc.Probabilities[itemID] = probability
	return nil
}

// Validate 验证概率总和是否为100%
func (pc *ProbabilityConfig) Validate() bool {
	total := 0.0
	for _, p := range pc.Probabilities {
		total += p
	}
	// 允许一定误差
	return total >= 99.9 && total <= 100.1
}

// ToWeightedPool 转换为权重抽卡池
func (pc *ProbabilityConfig) ToWeightedPool() (*WeightedPool, error) {
	if !pc.Validate() {
		return nil, errors.New("probabilities do not sum to 100%")
	}

	pool := NewWeightedPool()
	for itemID, probability := range pc.Probabilities {
		// 概率转换为权重（乘以100）
		weight := int(probability * 10)
		if weight <= 0 {
			weight = 1
		}
		pool.AddItem(itemID, "", weight, nil)
	}

	return pool, nil
}

// ========== 抽卡结果统计 ==========

// DrawStatistics 抽卡统计
type DrawStatistics struct {
	TotalDraws    int            `json:"total_draws"`    // 总抽卡次数
	ItemCounts    map[string]int `json:"item_counts"`    // 各项抽中次数
	TypeCounts    map[string]int `json:"type_counts"`    // 各类型抽中次数
}

// NewDrawStatistics 创建统计对象
func NewDrawStatistics() *DrawStatistics {
	return &DrawStatistics{
		TotalDraws: 0,
		ItemCounts: make(map[string]int),
		TypeCounts: make(map[string]int),
	}
}

// Record 记录一次抽卡
func (ds *DrawStatistics) Record(item *WeightedItem) {
	ds.TotalDraws++
	ds.ItemCounts[item.ID]++
	ds.TypeCounts[item.Type]++
}

// GetProbability 计算某项的实际抽中概率
func (ds *DrawStatistics) GetProbability(itemID string) float64 {
	if ds.TotalDraws == 0 {
		return 0
	}
	return float64(ds.ItemCounts[itemID]) / float64(ds.TotalDraws) * 100
}

// GetTopItems 获取抽中最多的项
func (ds *DrawStatistics) GetTopItems(limit int) []string {
	// 按次数排序
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

// ========== 预定义抽卡池 ==========

// EventPool 游戏事件抽卡池
type EventPool struct {
	GoodPool *WeightedPool `json:"good_pool"` // 良性事件池
	BadPool  *WeightedPool `json:"bad_pool"`  // 恶性事件池
	NeutralPool *WeightedPool `json:"neutral_pool"` // 中性事件池
}

// NewEventPool 创建事件抽卡池
func NewEventPool() *EventPool {
	return &EventPool{
		GoodPool:    NewWeightedPool(),
		BadPool:     NewWeightedPool(),
		NeutralPool: NewWeightedPool(),
	}
}

// AddGoodEvent 添加良性事件
func (ep *EventPool) AddGoodEvent(id string, weight int, data interface{}) error {
	return ep.GoodPool.AddItem(id, "good", weight, data)
}

// AddBadEvent 添加恶性事件
func (ep *EventPool) AddBadEvent(id string, weight int, data interface{}) error {
	return ep.BadPool.AddItem(id, "bad", weight, data)
}

// AddNeutralEvent 添加中性事件
func (ep *EventPool) AddNeutralEvent(id string, weight int, data interface{}) error {
	return ep.NeutralPool.AddItem(id, "neutral", weight, data)
}

// DrawEvent 抽取事件
// luck: 幸运值，影响好/坏事件概率
func (ep *EventPool) DrawEvent(luck int, rng *rand.Rand) (*WeightedItem, string, error) {
	// 计算好/坏/中性事件的整体概率
	// 基础概率：好30%，坏30%，中性40%
	// 幸运值影响：每点幸运值增加好事件概率5%，减少坏事件概率5%

	goodProb := 30 + luck*5
	badProb := 30 - luck*5
	neutralProb := 40

	// 限制概率范围
	if goodProb > 70 {
		goodProb = 70
	}
	if badProb < 10 {
		badProb = 10
	}

	// 创建类型选择池
	typePool := NewWeightedPoolWithSeed(rng.Int63())
	typePool.AddItem("good", "type", goodProb, nil)
	typePool.AddItem("bad", "type", badProb, nil)
	typePool.AddItem("neutral", "type", neutralProb, nil)

	// 选择事件类型
	typeItem, err := typePool.Draw()
	if err != nil {
		return nil, "", err
	}

	// 从对应池中抽取
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

// ItemPool 道具抽卡池
type ItemPool struct {
	CommonPool   *WeightedPool `json:"common_pool"`   // 普通道具池
	RarePool     *WeightedPool `json:"rare_pool"`     // 稀有道具池
	EpicPool     *WeightedPool `json:"epic_pool"`     // 史诗道具池
}

// NewItemPool 创建道具抽卡池
func NewItemPool() *ItemPool {
	return &ItemPool{
		CommonPool: NewWeightedPool(),
		RarePool:   NewWeightedPool(),
		EpicPool:   NewWeightedPool(),
	}
}

// AddCommonItem 添加普通道具
func (ip *ItemPool) AddCommonItem(id string, weight int, data interface{}) error {
	return ip.CommonPool.AddItem(id, "common", weight, data)
}

// AddRareItem 添加稀有道具
func (ip *ItemPool) AddRareItem(id string, weight int, data interface{}) error {
	return ip.RarePool.AddItem(id, "rare", weight, data)
}

// AddEpicItem 添加史诗道具
func (ip *ItemPool) AddEpicItem(id string, weight int, data interface{}) error {
	return ip.EpicPool.AddItem(id, "epic", weight, data)
}

// DrawItem 抽取道具
// luck: 幸运值，影响稀有度概率
func (ip *ItemPool) DrawItem(luck int, rng *rand.Rand) (*WeightedItem, string, error) {
	// 基础概率：普通70%，稀有20%，史诗10%
	// 幸运值影响：每点幸运值增加稀有/史诗概率各2%

	commonProb := 70 - luck*4
	rareProb := 20 + luck*2
	epicProb := 10 + luck*2

	// 限制概率范围
	if commonProb < 50 {
		commonProb = 50
	}
	if rareProb > 30 {
		rareProb = 30
	}
	if epicProb > 20 {
		epicProb = 20
	}

	// 创建稀有度选择池
	rarityPool := NewWeightedPoolWithSeed(rng.Int63())
	rarityPool.AddItem("common", "rarity", commonProb, nil)
	rarityPool.AddItem("rare", "rarity", rareProb, nil)
	rarityPool.AddItem("epic", "rarity", epicProb, nil)

	// 选择稀有度
	rarityItem, err := rarityPool.Draw()
	if err != nil {
		return nil, "", err
	}

	// 从对应池中抽取
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