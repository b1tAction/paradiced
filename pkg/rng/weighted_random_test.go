package rng

import (
	"math/rand"
	"testing"
	"time"
)

// ========== WeightedPool Tests ==========

func TestNewWeightedPool(t *testing.T) {
	pool := NewWeightedPool()
	if pool == nil {
		t.Fatal("pool should not be nil")
	}
	if len(pool.Items) != 0 {
		t.Errorf("pool.Items length = %d, expected 0", len(pool.Items))
	}
	if pool.TotalWeight != 0 {
		t.Errorf("pool.TotalWeight = %d, expected 0", pool.TotalWeight)
	}
}

func TestNewWeightedPoolWithSeed(t *testing.T) {
	pool := NewWeightedPoolWithSeed(42)
	if pool == nil {
		t.Fatal("pool should not be nil")
	}
	// 相同种子应该产生相同结果
	pool2 := NewWeightedPoolWithSeed(42)
	if pool.rng.Intn(100) != pool2.rng.Intn(100) {
		t.Error("same seed should produce same random numbers")
	}
}

func TestAddItem(t *testing.T) {
	pool := NewWeightedPool()

	err := pool.AddItem("item1", "type1", 10, "data1")
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}
	if len(pool.Items) != 1 {
		t.Errorf("pool.Items length = %d, expected 1", len(pool.Items))
	}
	if pool.TotalWeight != 10 {
		t.Errorf("pool.TotalWeight = %d, expected 10", pool.TotalWeight)
	}

	// 添加多个项
	pool.AddItem("item2", "type1", 20, "data2")
	pool.AddItem("item3", "type2", 30, "data3")
	if pool.TotalWeight != 60 {
		t.Errorf("pool.TotalWeight = %d, expected 60", pool.TotalWeight)
	}

	// 无效权重
	err = pool.AddItem("item4", "type", 0, nil)
	if err == nil {
		t.Error("AddItem with zero weight should return error")
	}
	err = pool.AddItem("item4", "type", -1, nil)
	if err == nil {
		t.Error("AddItem with negative weight should return error")
	}

	// 空 ID
	err = pool.AddItem("", "type", 10, nil)
	if err == nil {
		t.Error("AddItem with empty ID should return error")
	}
}

func TestRemoveItem(t *testing.T) {
	pool := NewWeightedPool()
	pool.AddItem("item1", "type1", 10, nil)
	pool.AddItem("item2", "type1", 20, nil)

	removed := pool.RemoveItem("item1")
	if !removed {
		t.Error("RemoveItem should return true for existing item")
	}
	if len(pool.Items) != 1 {
		t.Errorf("pool.Items length = %d, expected 1", len(pool.Items))
	}
	if pool.TotalWeight != 20 {
		t.Errorf("pool.TotalWeight = %d, expected 20", pool.TotalWeight)
	}

	// 移除不存在的项
	removed = pool.RemoveItem("nonexistent")
	if removed {
		t.Error("RemoveItem should return false for non-existent item")
	}
}

func TestGetItem(t *testing.T) {
	pool := NewWeightedPool()
	pool.AddItem("item1", "type1", 10, "data1")

	item := pool.GetItem("item1")
	if item == nil {
		t.Fatal("GetItem should return item")
	}
	if item.ID != "item1" {
		t.Errorf("item.ID = %s, expected item1", item.ID)
	}
	if item.Weight != 10 {
		t.Errorf("item.Weight = %d, expected 10", item.Weight)
	}

	// 获取不存在的项
	item = pool.GetItem("nonexistent")
	if item != nil {
		t.Error("GetItem non-existent should return nil")
	}
}

func TestGetItemsByType(t *testing.T) {
	pool := NewWeightedPool()
	pool.AddItem("item1", "typeA", 10, nil)
	pool.AddItem("item2", "typeA", 20, nil)
	pool.AddItem("item3", "typeB", 30, nil)

	items := pool.GetItemsByType("typeA")
	if len(items) != 2 {
		t.Errorf("typeA items count = %d, expected 2", len(items))
	}

	items = pool.GetItemsByType("typeB")
	if len(items) != 1 {
		t.Errorf("typeB items count = %d, expected 1", len(items))
	}

	items = pool.GetItemsByType("typeC")
	if len(items) != 0 {
		t.Errorf("typeC items count = %d, expected 0", len(items))
	}
}

func TestDraw(t *testing.T) {
	pool := NewWeightedPoolWithSeed(42)

	// 空池抽卡
	_, err := pool.Draw()
	if err == nil {
		t.Error("Draw from empty pool should return error")
	}

	// 添加项
	pool.AddItem("item1", "type1", 50, nil)
	pool.AddItem("item2", "type1", 50, nil)

	item, err := pool.Draw()
	if err != nil {
		t.Fatalf("Draw failed: %v", err)
	}
	if item == nil {
		t.Fatal("Draw should return item")
	}
	if item.ID == "" {
		t.Error("Draw item should have ID")
	}
}

func TestDrawDistribution(t *testing.T) {
	pool := NewWeightedPoolWithSeed(42)
	pool.AddItem("item1", "type1", 10, nil) // 10% 概率
	pool.AddItem("item2", "type2", 30, nil) // 30% 概率
	pool.AddItem("item3", "type3", 60, nil) // 60% 概率

	// 进行大量抽卡，验证分布
	stats := NewDrawStatistics()
	iterations := 10000

	for i := 0; i < iterations; i++ {
		item, err := pool.Draw()
		if err != nil {
			t.Fatalf("Draw failed: %v", err)
		}
		stats.Record(item)
	}

	// 验证概率分布（允许5%误差）
	prob1 := stats.GetProbability("item1")
	if prob1 < 7 || prob1 > 13 {
		t.Errorf("item1 probability = %.2f%%, expected ~10%%", prob1)
	}

	prob2 := stats.GetProbability("item2")
	if prob2 < 27 || prob2 > 33 {
		t.Errorf("item2 probability = %.2f%%, expected ~30%%", prob2)
	}

	prob3 := stats.GetProbability("item3")
	if prob3 < 57 || prob3 > 63 {
		t.Errorf("item3 probability = %.2f%%, expected ~60%%", prob3)
	}
}

func TestDrawWithType(t *testing.T) {
	pool := NewWeightedPoolWithSeed(42)
	pool.AddItem("item1", "typeA", 10, nil)
	pool.AddItem("item2", "typeA", 20, nil)
	pool.AddItem("item3", "typeB", 30, nil)

	// 按类型抽卡
	item, err := pool.DrawWithType("typeA")
	if err != nil {
		t.Fatalf("DrawWithType failed: %v", err)
	}
	if item.Type != "typeA" {
		t.Errorf("item.Type = %s, expected typeA", item.Type)
	}

	// 不存在的类型
	_, err = pool.DrawWithType("typeC")
	if err == nil {
		t.Error("DrawWithType non-existent type should return error")
	}
}

func TestDrawMultiple(t *testing.T) {
	pool := NewWeightedPoolWithSeed(42)
	pool.AddItem("item1", "type1", 10, nil)
	pool.AddItem("item2", "type1", 20, nil)
	pool.AddItem("item3", "type2", 30, nil)

	// 抽取多个（不重复）
	items, err := pool.DrawMultiple(3)
	if err != nil {
		t.Fatalf("DrawMultiple failed: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("items count = %d, expected 3", len(items))
	}

	// 验证不重复
	ids := make(map[string]bool)
	for _, item := range items {
		if ids[item.ID] {
			t.Errorf("item %s is duplicated", item.ID)
		}
		ids[item.ID] = true
	}

	// 抽取超过池大小
	_, err = pool.DrawMultiple(5)
	if err == nil {
		t.Error("DrawMultiple exceeding pool size should return error")
	}

	// 抽取0个
	_, err = pool.DrawMultiple(0)
	if err == nil {
		t.Error("DrawMultiple with count 0 should return error")
	}
}

func TestDrawMultipleWithType(t *testing.T) {
	pool := NewWeightedPoolWithSeed(42)
	pool.AddItem("item1", "typeA", 10, nil)
	pool.AddItem("item2", "typeA", 20, nil)
	pool.AddItem("item3", "typeB", 30, nil)

	// 按类型抽取多个
	items, err := pool.DrawMultipleWithType("typeA", 2)
	if err != nil {
		t.Fatalf("DrawMultipleWithType failed: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("items count = %d, expected 2", len(items))
	}

	for _, item := range items {
		if item.Type != "typeA" {
			t.Errorf("item.Type = %s, expected typeA", item.Type)
		}
	}

	// 抽取超过类型池大小
	_, err = pool.DrawMultipleWithType("typeA", 3)
	if err == nil {
		t.Error("DrawMultipleWithType exceeding type pool size should return error")
	}
}

// ========== LuckModifier Tests ==========

func TestNewLuckModifier(t *testing.T) {
	modifier := NewLuckModifier(50, 0.1)
	if modifier.BaseWeight != 50 {
		t.Errorf("BaseWeight = %d, expected 50", modifier.BaseWeight)
	}
	if modifier.LuckFactor != 0.1 {
		t.Errorf("LuckFactor = %.2f, expected 0.1", modifier.LuckFactor)
	}
}

func TestCalculateWeight(t *testing.T) {
	modifier := NewLuckModifier(50, 0.1)

	// luck=0，良性事件，权重不变
	weight := modifier.CalculateWeight(0, true)
	if weight != 50 {
		t.Errorf("weight = %d, expected 50", weight)
	}

	// luck=5，良性事件，权重增加
	weight = modifier.CalculateWeight(5, true)
	expected := int(50.0 + 50.0*0.1*5) // 50 + 25 = 75
	if weight != expected {
		t.Errorf("weight = %d, expected %d", weight, expected)
	}

	// luck=5，恶性事件，权重减少
	weight = modifier.CalculateWeight(5, false)
	expected = int(50.0 - 50.0*0.1*5) // 50 - 25 = 25
	if weight != expected {
		t.Errorf("weight = %d, expected %d", weight, expected)
	}

	// luck=10，恶性事件，权重不应该低于最小值
	modifier.MaxWeight = 100
	modifier.MinWeight = 1
	weight = modifier.CalculateWeight(100, false)
	if weight < modifier.MinWeight {
		t.Errorf("weight = %d, should not be below min %d", weight, modifier.MinWeight)
	}

	// luck=10，良性事件，权重不应该超过最大值
	weight = modifier.CalculateWeight(100, true)
	if weight > modifier.MaxWeight {
		t.Errorf("weight = %d, should not exceed max %d", weight, modifier.MaxWeight)
	}
}

func TestLuckAdjustedPool(t *testing.T) {
	basePool := NewWeightedPoolWithSeed(42)
	basePool.AddItem("good1", "good", 50, nil)
	basePool.AddItem("bad1", "bad", 50, nil)

	modifier := NewLuckModifier(50, 0.1)
	modifier.GoodType = "good"
	modifier.BadType = "bad"

	adjustedPool := NewLuckAdjustedPool(basePool, modifier)

	// luck=0 时抽卡
	item, err := adjustedPool.DrawWithLuck(0)
	if err != nil {
		t.Fatalf("DrawWithLuck failed: %v", err)
	}
	if item == nil {
		t.Fatal("DrawWithLuck should return item")
	}

	// luck=5 时抽卡（应该倾向良性事件）
	stats := NewDrawStatistics()
	iterations := 1000
	for i := 0; i < iterations; i++ {
		// 重置池
		pool := NewLuckAdjustedPool(NewWeightedPoolWithSeed(time.Now().UnixNano()+int64(i)), modifier)
		pool.BasePool.AddItem("good1", "good", 50, nil)
		pool.BasePool.AddItem("bad1", "bad", 50, nil)
		item, _ := pool.DrawWithLuck(5)
		stats.Record(item)
	}

	// 验证良性事件概率增加
	goodProb := stats.GetProbability("good1")
	if goodProb < 50 {
		t.Errorf("with luck=5, good event probability = %.2f%%, should be higher than 50%%", goodProb)
	}

	// nil base pool
	nilPool := NewLuckAdjustedPool(nil, modifier)
	_, err = nilPool.DrawWithLuck(0)
	if err == nil {
		t.Error("DrawWithLuck with nil base pool should return error")
	}
}

// ========== ProbabilityConfig Tests ==========

func TestNewProbabilityConfig(t *testing.T) {
	config := NewProbabilityConfig("config1", "Test Config")
	if config.ID != "config1" {
		t.Errorf("config.ID = %s, expected config1", config.ID)
	}
	if config.Name != "Test Config" {
		t.Errorf("config.Name = %s, expected Test Config", config.Name)
	}
}

func TestSetProbability(t *testing.T) {
	config := NewProbabilityConfig("config1", "Test")

	err := config.SetProbability("item1", 30.0)
	if err != nil {
		t.Fatalf("SetProbability failed: %v", err)
	}
	if config.Probabilities["item1"] != 30.0 {
		t.Errorf("probability = %.2f, expected 30.0", config.Probabilities["item1"])
	}

	// 无效概率
	err = config.SetProbability("item2", -10.0)
	if err == nil {
		t.Error("SetProbability with negative value should return error")
	}
	err = config.SetProbability("item2", 150.0)
	if err == nil {
		t.Error("SetProbability with value > 100 should return error")
	}
}

func TestProbabilityConfigValidate(t *testing.T) {
	config := NewProbabilityConfig("config1", "Test")
	config.SetProbability("item1", 50.0)
	config.SetProbability("item2", 50.0)

	if !config.Validate() {
		t.Error("config with 100% total should be valid")
	}

	// 不等于100%
	config.SetProbability("item3", 10.0)
	if config.Validate() {
		t.Error("config with 110% total should not be valid")
	}
}

func TestProbabilityConfigToWeightedPool(t *testing.T) {
	config := NewProbabilityConfig("config1", "Test")
	config.SetProbability("item1", 50.0)
	config.SetProbability("item2", 50.0)

	pool, err := config.ToWeightedPool()
	if err != nil {
		t.Fatalf("ToWeightedPool failed: %v", err)
	}
	if pool.TotalWeight != 1000 { // 50*10 + 50*10
		t.Errorf("pool.TotalWeight = %d, expected 1000", pool.TotalWeight)
	}

	// 无效配置
	config.SetProbability("item3", 10.0)
	_, err = config.ToWeightedPool()
	if err == nil {
		t.Error("ToWeightedPool with invalid config should return error")
	}
}

// ========== DrawStatistics Tests ==========

func TestNewDrawStatistics(t *testing.T) {
	stats := NewDrawStatistics()
	if stats.TotalDraws != 0 {
		t.Errorf("TotalDraws = %d, expected 0", stats.TotalDraws)
	}
	if len(stats.ItemCounts) != 0 {
		t.Error("ItemCounts should be empty initially")
	}
}

func TestDrawStatisticsRecord(t *testing.T) {
	stats := NewDrawStatistics()

	item1 := &WeightedItem{ID: "item1", Type: "typeA"}
	item2 := &WeightedItem{ID: "item2", Type: "typeA"}
	item3 := &WeightedItem{ID: "item3", Type: "typeB"}

	stats.Record(item1)
	stats.Record(item1)
	stats.Record(item2)
	stats.Record(item3)

	if stats.TotalDraws != 4 {
		t.Errorf("TotalDraws = %d, expected 4", stats.TotalDraws)
	}
	if stats.ItemCounts["item1"] != 2 {
		t.Errorf("item1 count = %d, expected 2", stats.ItemCounts["item1"])
	}
	if stats.TypeCounts["typeA"] != 3 {
		t.Errorf("typeA count = %d, expected 3", stats.TypeCounts["typeA"])
	}
}

func TestDrawStatisticsGetProbability(t *testing.T) {
	stats := NewDrawStatistics()

	for i := 0; i < 100; i++ {
		stats.Record(&WeightedItem{ID: "item1", Type: "type"})
	}
	for i := 0; i < 50; i++ {
		stats.Record(&WeightedItem{ID: "item2", Type: "type"})
	}

	prob1 := stats.GetProbability("item1")
	// 100/150 * 100 = 66.66666...，允许误差
	if prob1 < 66.0 || prob1 > 67.0 {
		t.Errorf("item1 probability = %.2f%%, expected ~66.67%%", prob1)
	}

	// 无抽卡时
	emptyStats := NewDrawStatistics()
	prob := emptyStats.GetProbability("item1")
	if prob != 0 {
		t.Errorf("probability with no draws = %.2f%%, expected 0", prob)
	}
}

func TestDrawStatisticsGetTopItems(t *testing.T) {
	stats := NewDrawStatistics()

	// item1: 10次, item2: 5次, item3: 3次
	for i := 0; i < 10; i++ {
		stats.Record(&WeightedItem{ID: "item1", Type: "type"})
	}
	for i := 0; i < 5; i++ {
		stats.Record(&WeightedItem{ID: "item2", Type: "type"})
	}
	for i := 0; i < 3; i++ {
		stats.Record(&WeightedItem{ID: "item3", Type: "type"})
	}

	top := stats.GetTopItems(2)
	if len(top) != 2 {
		t.Errorf("top items count = %d, expected 2", len(top))
	}
	if top[0] != "item1" {
		t.Errorf("top[0] = %s, expected item1", top[0])
	}
	if top[1] != "item2" {
		t.Errorf("top[1] = %s, expected item2", top[1])
	}
}

// ========== EventPool Tests ==========

func TestNewEventPool(t *testing.T) {
	pool := NewEventPool()
	if pool.GoodPool == nil {
		t.Error("GoodPool should not be nil")
	}
	if pool.BadPool == nil {
		t.Error("BadPool should not be nil")
	}
	if pool.NeutralPool == nil {
		t.Error("NeutralPool should not be nil")
	}
}

func TestEventPoolAddEvent(t *testing.T) {
	pool := NewEventPool()

	err := pool.AddGoodEvent("hp_plus", 10, map[string]int{"hp": 1})
	if err != nil {
		t.Fatalf("AddGoodEvent failed: %v", err)
	}
	if len(pool.GoodPool.Items) != 1 {
		t.Errorf("GoodPool items count = %d, expected 1", len(pool.GoodPool.Items))
	}

	err = pool.AddBadEvent("hp_minus", 10, map[string]int{"hp": -1})
	if err != nil {
		t.Fatalf("AddBadEvent failed: %v", err)
	}

	err = pool.AddNeutralEvent("exchange", 10, nil)
	if err != nil {
		t.Fatalf("AddNeutralEvent failed: %v", err)
	}
}

func TestEventPoolDrawEvent(t *testing.T) {
	pool := NewEventPool()
	pool.AddGoodEvent("hp_plus", 50, nil)
	pool.AddBadEvent("hp_minus", 50, nil)
	pool.AddNeutralEvent("exchange", 50, nil)

	rng := rand.New(rand.NewSource(42))

	// luck=0 时抽卡
	item, eventType, err := pool.DrawEvent(0, rng)
	if err != nil {
		t.Fatalf("DrawEvent failed: %v", err)
	}
	if item == nil {
		t.Fatal("DrawEvent should return item")
	}
	if eventType == "" {
		t.Error("DrawEvent should return event type")
	}

	// luck=5 时，应该倾向良性事件
	stats := NewDrawStatistics()
	iterations := 1000
	for i := 0; i < iterations; i++ {
		testPool := NewEventPool()
		testPool.AddGoodEvent("good1", 50, nil)
		testPool.AddBadEvent("bad1", 50, nil)
		testPool.AddNeutralEvent("neutral1", 50, nil)
		item, eventType, _ = testPool.DrawEvent(5, rand.New(rand.NewSource(time.Now().UnixNano()+int64(i))))
		stats.Record(&WeightedItem{ID: eventType, Type: eventType})
	}

	// 验证良性事件概率增加
	goodProb := stats.GetProbability("good")
	if goodProb < 30 {
		t.Errorf("with luck=5, good event probability = %.2f%%, should be higher", goodProb)
	}

	// 验证恶性事件概率减少
	badProb := stats.GetProbability("bad")
	if badProb > 30 {
		t.Errorf("with luck=5, bad event probability = %.2f%%, should be lower", badProb)
	}
}

// ========== ItemPool Tests ==========

func TestNewItemPool(t *testing.T) {
	pool := NewItemPool()
	if pool.CommonPool == nil {
		t.Error("CommonPool should not be nil")
	}
	if pool.RarePool == nil {
		t.Error("RarePool should not be nil")
	}
	if pool.EpicPool == nil {
		t.Error("EpicPool should not be nil")
	}
}

func TestItemPoolAddItem(t *testing.T) {
	pool := NewItemPool()

	err := pool.AddCommonItem("item1", 10, nil)
	if err != nil {
		t.Fatalf("AddCommonItem failed: %v", err)
	}

	err = pool.AddRareItem("item2", 10, nil)
	if err != nil {
		t.Fatalf("AddRareItem failed: %v", err)
	}

	err = pool.AddEpicItem("item3", 10, nil)
	if err != nil {
		t.Fatalf("AddEpicItem failed: %v", err)
	}
}

func TestItemPoolDrawItem(t *testing.T) {
	pool := NewItemPool()
	pool.AddCommonItem("common1", 50, nil)
	pool.AddRareItem("rare1", 50, nil)
	pool.AddEpicItem("epic1", 50, nil)

	rng := rand.New(rand.NewSource(42))

	// luck=0 时抽卡
	item, rarity, err := pool.DrawItem(0, rng)
	if err != nil {
		t.Fatalf("DrawItem failed: %v", err)
	}
	if item == nil {
		t.Fatal("DrawItem should return item")
	}
	if rarity == "" {
		t.Error("DrawItem should return rarity")
	}

	// luck=5 时，应该增加稀有/史诗概率
	stats := NewDrawStatistics()
	iterations := 1000
	for i := 0; i < iterations; i++ {
		testPool := NewItemPool()
		testPool.AddCommonItem("c1", 50, nil)
		testPool.AddRareItem("r1", 50, nil)
		testPool.AddEpicItem("e1", 50, nil)
		_, rarity, _ = testPool.DrawItem(5, rand.New(rand.NewSource(time.Now().UnixNano()+int64(i))))
		stats.Record(&WeightedItem{ID: rarity, Type: rarity})
	}

	// 验证史诗概率增加
	epicProb := stats.GetProbability("epic")
	if epicProb < 10 {
		t.Errorf("with luck=5, epic probability = %.2f%%, should be higher than 10%%", epicProb)
	}

	// 验证普通概率减少
	commonProb := stats.GetProbability("common")
	if commonProb > 70 {
		t.Errorf("with luck=5, common probability = %.2f%%, should be lower than 70%%", commonProb)
	}
}