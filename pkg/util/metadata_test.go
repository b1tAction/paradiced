package util

import (
	"testing"
)

// ========== 初始化测试 ==========

func TestNewMetadata(t *testing.T) {
	m := NewMetadata()
	if m == nil {
		t.Fatal("NewMetadata should return non-nil")
	}
	if m.values == nil {
		t.Error("values map should be initialized")
	}
	if m.Size() != 0 {
		t.Errorf("new metadata size = %d, expected 0", m.Size())
	}
}

// ========== Set/Get 基本操作测试 ==========

func TestSetAndGet(t *testing.T) {
	m := NewMetadata()

	// 设置并获取任意类型
	m.Set("key1", 100)
	m.Set("key2", "hello")
	m.Set("key3", true)

	val1, ok1 := m.Get("key1")
	if !ok1 {
		t.Error("key1 should exist")
	}
	if val1 != 100 {
		t.Errorf("key1 = %v, expected 100", val1)
	}

	val2, ok2 := m.Get("key2")
	if !ok2 {
		t.Error("key2 should exist")
	}
	if val2 != "hello" {
		t.Errorf("key2 = %v, expected hello", val2)
	}

	// 获取不存在的键
	val, ok := m.Get("nonexistent")
	if ok {
		t.Error("nonexistent key should not exist")
	}
	if val != nil {
		t.Errorf("nonexistent key value = %v, expected nil", val)
	}
}

func TestSetOverwrite(t *testing.T) {
	m := NewMetadata()

	m.Set("key", 100)
	m.Set("key", 200)

	if m.GetInt("key") != 200 {
		t.Errorf("overwritten key = %d, expected 200", m.GetInt("key"))
	}
}

// ========== GetInt/SetInt 测试 ==========

func TestGetInt(t *testing.T) {
	m := NewMetadata()

	// 设置并获取整型
	m.SetInt("count", 42)
	if m.GetInt("count") != 42 {
		t.Errorf("GetInt = %d, expected 42", m.GetInt("count"))
	}

	// 获取不存在的键，返回 0
	if m.GetInt("nonexistent") != 0 {
		t.Errorf("GetInt nonexistent = %d, expected 0", m.GetInt("nonexistent"))
	}

	// 类型不匹配时返回 0
	m.Set("string_key", "not an int")
	if m.GetInt("string_key") != 0 {
		t.Errorf("GetInt with wrong type = %d, expected 0", m.GetInt("string_key"))
	}
}

func TestGetIntOrDefault(t *testing.T) {
	m := NewMetadata()

	// 存在的键返回实际值
	m.SetInt("existing", 10)
	if m.GetIntOrDefault("existing", 5) != 10 {
		t.Errorf("GetIntOrDefault existing = %d, expected 10", m.GetIntOrDefault("existing", 5))
	}

	// 不存在的键返回默认值
	if m.GetIntOrDefault("nonexistent", 5) != 5 {
		t.Errorf("GetIntOrDefault nonexistent = %d, expected 5", m.GetIntOrDefault("nonexistent", 5))
	}
}

// ========== GetBool/SetBool 测试 ==========

func TestGetBool(t *testing.T) {
	m := NewMetadata()

	m.SetBool("flag_true", true)
	m.SetBool("flag_false", false)

	if !m.GetBool("flag_true") {
		t.Error("GetBool flag_true should be true")
	}
	if m.GetBool("flag_false") {
		t.Error("GetBool flag_false should be false")
	}

	// 不存在的键返回 false
	if m.GetBool("nonexistent") {
		t.Error("GetBool nonexistent should be false")
	}

	// 类型不匹配返回 false
	m.Set("int_key", 100)
	if m.GetBool("int_key") {
		t.Error("GetBool with wrong type should be false")
	}
}

// ========== GetString/SetString 测试 ==========

func TestGetString(t *testing.T) {
	m := NewMetadata()

	m.SetString("name", "test")
	if m.GetString("name") != "test" {
		t.Errorf("GetString = %s, expected test", m.GetString("name"))
	}

	// 不存在的键返回空字符串
	if m.GetString("nonexistent") != "" {
		t.Errorf("GetString nonexistent = %s, expected empty", m.GetString("nonexistent"))
	}

	// 类型不匹配返回空字符串
	m.Set("int_key", 100)
	if m.GetString("int_key") != "" {
		t.Errorf("GetString with wrong type = %s, expected empty", m.GetString("int_key"))
	}
}

// ========== GetFloat64/SetFloat64 测试 ==========

func TestGetFloat64(t *testing.T) {
	m := NewMetadata()

	m.SetFloat64("ratio", 3.14)
	if m.GetFloat64("ratio") != 3.14 {
		t.Errorf("GetFloat64 = %f, expected 3.14", m.GetFloat64("ratio"))
	}

	// 不存在的键返回 0
	if m.GetFloat64("nonexistent") != 0 {
		t.Errorf("GetFloat64 nonexistent = %f, expected 0", m.GetFloat64("nonexistent"))
	}
}

// ========== HasKey/Delete/Clear 测试 ==========

func TestHasKey(t *testing.T) {
	m := NewMetadata()

	m.Set("existing", 100)

	if !m.HasKey("existing") {
		t.Error("existing key should be found")
	}
	if m.HasKey("nonexistent") {
		t.Error("nonexistent key should not be found")
	}
}

func TestDelete(t *testing.T) {
	m := NewMetadata()

	m.Set("key", 100)
	if !m.HasKey("key") {
		t.Fatal("key should exist before delete")
	}

	m.Delete("key")
	if m.HasKey("key") {
		t.Error("key should not exist after delete")
	}

	// 删除不存在的键不会 panic
	m.Delete("nonexistent")
}

func TestClear(t *testing.T) {
	m := NewMetadata()

	m.Set("key1", 100)
	m.Set("key2", 200)
	m.Set("key3", 300)

	if m.Size() != 3 {
		t.Fatalf("size = %d, expected 3 before clear", m.Size())
	}

	m.Clear()

	if m.Size() != 0 {
		t.Errorf("size = %d, expected 0 after clear", m.Size())
	}
	if m.HasKey("key1") {
		t.Error("key1 should not exist after clear")
	}
}

// ========== Keys 测试 ==========

func TestKeys(t *testing.T) {
	m := NewMetadata()

	m.Set("a", 1)
	m.Set("b", 2)
	m.Set("c", 3)

	keys := m.Keys()
	if len(keys) != 3 {
		t.Errorf("keys count = %d, expected 3", len(keys))
	}

	// 验证所有键都在返回列表中
	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}
	for expected := range []string{"a", "b", "c"} {
		if !keySet[keys[expected]] {
			t.Errorf("key %s not found in Keys()", keys[expected])
		}
	}
}

// ========== Clone 测试 ==========

func TestClone(t *testing.T) {
	m := NewMetadata()
	m.SetInt("count", 10)
	m.SetString("name", "original")

	cloned := m.Clone()

	// 克隆副本独立
	cloned.SetInt("count", 20)
	cloned.SetString("name", "cloned")

	// 原版不受影响
	if m.GetInt("count") != 10 {
		t.Errorf("original count = %d, expected 10", m.GetInt("count"))
	}
	if m.GetString("name") != "original" {
		t.Errorf("original name = %s, expected original", m.GetString("name"))
	}

	// 克隆副本值正确
	if cloned.GetInt("count") != 20 {
		t.Errorf("cloned count = %d, expected 20", cloned.GetInt("count"))
	}
	if cloned.GetString("name") != "cloned" {
		t.Errorf("cloned name = %s, expected cloned", cloned.GetString("name"))
	}
}

func TestCloneNil(t *testing.T) {
	// Clone 空 Metadata
	m := NewMetadata()
	cloned := m.Clone()

	if cloned == nil {
		t.Error("Clone should return non-nil")
	}
	if cloned.Size() != 0 {
		t.Errorf("empty clone size = %d, expected 0", cloned.Size())
	}
}

// ========== IncrementInt/DecrementInt 测试 ==========

func TestIncrementInt(t *testing.T) {
	m := NewMetadata()

	// 从 0 开始递增
	result := m.IncrementInt("counter", 1)
	if result != 1 {
		t.Errorf("first increment = %d, expected 1", result)
	}
	if m.GetInt("counter") != 1 {
		t.Errorf("counter = %d, expected 1", m.GetInt("counter"))
	}

	// 继续递增
	result = m.IncrementInt("counter", 5)
	if result != 6 {
		t.Errorf("second increment = %d, expected 6", result)
	}

	// 递增负数（实际上是递减）
	result = m.IncrementInt("counter", -2)
	if result != 4 {
		t.Errorf("increment with negative = %d, expected 4", result)
	}
}

func TestDecrementInt(t *testing.T) {
	m := NewMetadata()

	m.SetInt("counter", 10)
	result := m.DecrementInt("counter", 3)
	if result != 7 {
		t.Errorf("decrement = %d, expected 7", result)
	}
}

// ========== Merge 测试 ==========

func TestMerge(t *testing.T) {
	m1 := NewMetadata()
	m1.SetInt("a", 1)
	m1.SetInt("b", 2)

	m2 := NewMetadata()
	m2.SetInt("b", 20) // 相同键，会被覆盖
	m2.SetInt("c", 3)

	m1.Merge(m2)

	if m1.GetInt("a") != 1 {
		t.Errorf("a = %d, expected 1", m1.GetInt("a"))
	}
	if m1.GetInt("b") != 20 {
		t.Errorf("b = %d, expected 20 (overwritten)", m1.GetInt("b"))
	}
	if m1.GetInt("c") != 3 {
		t.Errorf("c = %d, expected 3", m1.GetInt("c"))
	}
}

func TestMergeNil(t *testing.T) {
	m := NewMetadata()
	m.SetInt("key", 100)

	// 合并 nil 不应 panic
	m.Merge(nil)

	if m.GetInt("key") != 100 {
		t.Errorf("key = %d, expected 100 (unchanged)", m.GetInt("key"))
	}
}

// ========== ToMap 测试 ==========

func TestToMap(t *testing.T) {
	m := NewMetadata()
	m.SetInt("a", 1)
	m.SetString("b", "hello")

	mapCopy := m.ToMap()

	if mapCopy["a"] != 1 {
		t.Errorf("mapCopy[a] = %v, expected 1", mapCopy["a"])
	}
	if mapCopy["b"] != "hello" {
		t.Errorf("mapCopy[b] = %v, expected hello", mapCopy["b"])
	}

	// 修改副本不影响原版
	mapCopy["a"] = 100
	if m.GetInt("a") != 1 {
		t.Errorf("original a = %d, expected 1", m.GetInt("a"))
	}
}

// ========== 链式调用测试 ==========

func TestChainedCalls(t *testing.T) {
	m := NewMetadata()

	// 链式设置多个值
	m.SetInt("int", 10).
		SetString("string", "hello").
		SetBool("bool", true)

	if m.GetInt("int") != 10 {
		t.Errorf("int = %d, expected 10", m.GetInt("int"))
	}
	if m.GetString("string") != "hello" {
		t.Errorf("string = %s, expected hello", m.GetString("string"))
	}
	if !m.GetBool("bool") {
		t.Error("bool should be true")
	}
}

// ========== Size 测试 ==========

func TestSize(t *testing.T) {
	m := NewMetadata()

	if m.Size() != 0 {
		t.Errorf("initial size = %d, expected 0", m.Size())
	}

	m.Set("a", 1)
	m.Set("b", 2)

	if m.Size() != 2 {
		t.Errorf("size after set = %d, expected 2", m.Size())
	}

	m.Delete("a")

	if m.Size() != 1 {
		t.Errorf("size after delete = %d, expected 1", m.Size())
	}
}