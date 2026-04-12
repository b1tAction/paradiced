package event

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// newID 生成唯一ID（包内共享）
func newID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Nanosecond())
}

// Subscription 订阅信息
type Subscription struct {
	ID         string    `json:"id"`          // 订阅ID，用于取消
	OwnerID    string    `json:"owner_id"`    // 玩家ID
	SourceID   string    `json:"source_id"`   // Buff/道具ID
	SourceType string    `json:"source_type"` // 来源类型 "buff" / "item"
	Priority   int       `json:"priority"`    // 执行优先级（高先执行）
	Decision   *Decision `json:"decision"`    // 预绑定的Decision
	Phase      Phase     `json:"phase"`       // 订阅的Phase
}

// EventBus 事件总线
// 每局游戏实例一个EventBus，管理Buff/道具的订阅和触发
type EventBus struct {
	subscriptions map[Phase][]*Subscription
	mutex         sync.RWMutex
	GameID        string `json:"game_id"` // 所属游戏实例ID
}

// NewEventBus 创建新的事件总线
func NewEventBus(gameID string) *EventBus {
	return &EventBus{
		subscriptions: make(map[Phase][]*Subscription),
		GameID:        gameID,
	}
}

// Subscribe 订阅某个Phase
// 返回订阅ID，用于取消订阅
func (bus *EventBus) Subscribe(phase Phase, ownerID, sourceID, sourceType string, decision *Decision) string {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	subID := newID()
	sub := &Subscription{
		ID:         subID,
		OwnerID:    ownerID,
		SourceID:   sourceID,
		SourceType: sourceType,
		Priority:   decision.Priority,
		Decision:   decision.WithSource(sourceID, sourceType),
		Phase:      phase,
	}

	bus.subscriptions[phase] = append(bus.subscriptions[phase], sub)
	return subID
}

// Unsubscribe 取消订阅
func (bus *EventBus) Unsubscribe(subID string) bool {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	for phase, subs := range bus.subscriptions {
		for i, sub := range subs {
			if sub.ID == subID {
				bus.subscriptions[phase] = append(subs[:i], subs[i+1:]...)
				return true
			}
		}
	}
	return false
}

// UnsubscribeBySource 按来源ID取消所有订阅
// 用于移除Buff/道具时批量取消订阅
func (bus *EventBus) UnsubscribeBySource(sourceID string) int {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	count := 0
	for phase, subs := range bus.subscriptions {
		newSubs := make([]*Subscription, 0, len(subs))
		for _, sub := range subs {
			if sub.SourceID == sourceID {
				count++
			} else {
				newSubs = append(newSubs, sub)
			}
		}
		bus.subscriptions[phase] = newSubs
	}
	return count
}

// UnsubscribeByOwner 按玩家ID取消所有订阅
// 用于玩家离开游戏时批量取消订阅
func (bus *EventBus) UnsubscribeByOwner(ownerID string) int {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	count := 0
	for phase, subs := range bus.subscriptions {
		newSubs := make([]*Subscription, 0, len(subs))
		for _, sub := range subs {
			if sub.OwnerID == ownerID {
				count++
			} else {
				newSubs = append(newSubs, sub)
			}
		}
		bus.subscriptions[phase] = newSubs
	}
	return count
}

// Publish 发布Phase事件
// 返回需要用户确认的Decision列表
func (bus *EventBus) Publish(phase Phase, ownerID string, ctx *Context) []*Decision {
	bus.mutex.RLock()
	subs := bus.subscriptions[phase]
	bus.mutex.RUnlock()

	// 过滤出该玩家的订阅
	ownerSubs := make([]*Subscription, 0)
	for _, sub := range subs {
		if sub.OwnerID == ownerID {
			ownerSubs = append(ownerSubs, sub)
		}
	}

	// 按Priority排序（高优先级先执行）
	sort.Slice(ownerSubs, func(i, j int) bool {
		return ownerSubs[i].Priority > ownerSubs[j].Priority
	})

	// 执行不需要确认的Decision，收集需要确认的
	decisions := make([]*Decision, 0)
	for _, sub := range ownerSubs {
		if sub.Decision.ShouldAsk() {
			// 需要用户确认，加入等待列表
			decisions = append(decisions, sub.Decision)
		} else {
			// 不需要确认，直接执行默认选项
			sub.Decision.Execute(sub.Decision.Default, ctx)
		}
	}

	return decisions
}

// GetSubscriptions 获取某个Phase的所有订阅（用于调试）
func (bus *EventBus) GetSubscriptions(phase Phase) []*Subscription {
	bus.mutex.RLock()
	defer bus.mutex.RUnlock()
	return bus.subscriptions[phase]
}

// GetSubscriptionCount 获取订阅总数（用于调试）
func (bus *EventBus) GetSubscriptionCount() int {
	bus.mutex.RLock()
	defer bus.mutex.RUnlock()
	count := 0
	for _, subs := range bus.subscriptions {
		count += len(subs)
	}
	return count
}

// Clear 清空所有订阅
func (bus *EventBus) Clear() {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()
	bus.subscriptions = make(map[Phase][]*Subscription)
}
