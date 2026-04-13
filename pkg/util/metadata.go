package util

// Metadata 动态属性袋，负责键值对的存储与类型安全的读取
// 通过匿名嵌入可以让任何结构体获得类型安全的键值存取能力
type Metadata struct {
	values map[string]interface{}
}

// NewMetadata 初始化属性袋
func NewMetadata() *Metadata {
	return &Metadata{
		values: make(map[string]interface{}),
	}
}

// Set 链式写入任意类型值
func (m *Metadata) Set(key string, value interface{}) *Metadata {
	m.values[key] = value
	return m
}

// Get 安全获取原始值
func (m *Metadata) Get(key string) (interface{}, bool) {
	val, ok := m.values[key]
	return val, ok
}

// GetInt 安全获取整型，不存在或类型不匹配返回 0
func (m *Metadata) GetInt(key string) int {
	if val, ok := m.values[key].(int); ok {
		return val
	}
	return 0
}

// GetIntOrDefault 安全获取整型，带默认值
func (m *Metadata) GetIntOrDefault(key string, defaultValue int) int {
	if val, ok := m.values[key].(int); ok {
		return val
	}
	return defaultValue
}

// SetInt 设置整型值，链式调用
func (m *Metadata) SetInt(key string, value int) *Metadata {
	m.values[key] = value
	return m
}

// GetBool 安全获取布尔值，不存在或类型不匹配返回 false
func (m *Metadata) GetBool(key string) bool {
	if val, ok := m.values[key].(bool); ok {
		return val
	}
	return false
}

// SetBool 设置布尔值，链式调用
func (m *Metadata) SetBool(key string, value bool) *Metadata {
	m.values[key] = value
	return m
}

// GetString 安全获取字符串，不存在或类型不匹配返回空字符串
func (m *Metadata) GetString(key string) string {
	if val, ok := m.values[key].(string); ok {
		return val
	}
	return ""
}

// SetString 设置字符串值，链式调用
func (m *Metadata) SetString(key string, value string) *Metadata {
	m.values[key] = value
	return m
}

// GetFloat64 安全获取浮点数，不存在或类型不匹配返回 0
func (m *Metadata) GetFloat64(key string) float64 {
	if val, ok := m.values[key].(float64); ok {
		return val
	}
	return 0
}

// SetFloat64 设置浮点数，链式调用
func (m *Metadata) SetFloat64(key string, value float64) *Metadata {
	m.values[key] = value
	return m
}

// HasKey 检查是否存在某个键
func (m *Metadata) HasKey(key string) bool {
	_, ok := m.values[key]
	return ok
}

// Delete 删除某个键
func (m *Metadata) Delete(key string) {
	delete(m.values, key)
}

// Clear 清空所有键
func (m *Metadata) Clear() {
	m.values = make(map[string]interface{})
}

// Size 返回键的数量
func (m *Metadata) Size() int {
	return len(m.values)
}

// Keys 返回所有键名
func (m *Metadata) Keys() []string {
	keys := make([]string, 0, len(m.values))
	for k := range m.values {
		keys = append(keys, k)
	}
	return keys
}

// Clone 克隆 Metadata，返回独立副本
func (m *Metadata) Clone() *Metadata {
	newValues := make(map[string]interface{}, len(m.values))
	for k, v := range m.values {
		newValues[k] = v
	}
	return &Metadata{values: newValues}
}

// IncrementInt 整型值递增，返回递增后的值
// 如果键不存在，从 0 开始递增
func (m *Metadata) IncrementInt(key string, delta int) int {
	current := m.GetInt(key)
	newValue := current + delta
	m.SetInt(key, newValue)
	return newValue
}

// DecrementInt 整型值递减，返回递减后的值
// 如果键不存在，从 0 开始递减
func (m *Metadata) DecrementInt(key string, delta int) int {
	return m.IncrementInt(key, -delta)
}

// Merge 合并另一个 Metadata 的值到当前 Metadata
// 相同的键会被覆盖
func (m *Metadata) Merge(other *Metadata) *Metadata {
	if other == nil {
		return m
	}
	for k, v := range other.values {
		m.values[k] = v
	}
	return m
}

// ToMap 返回底层 map 的副本（只读用途）
func (m *Metadata) ToMap() map[string]interface{} {
	result := make(map[string]interface{}, len(m.values))
	for k, v := range m.values {
		result[k] = v
	}
	return result
}