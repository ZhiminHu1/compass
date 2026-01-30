package pubsub

import (
	"context"
	"sync"
)

const bufferSize = 64

// Broker 实现了基于内存的发布者/订阅者模型。
// 它使用泛型 T 来保证事件数据载荷的类型安全。
type Broker[T any] struct {
	subs      map[chan Event[T]]struct{} // 活跃订阅者的集合，键为事件通道
	mu        sync.RWMutex               // 读写锁，保护 subs 映射的并发访问
	done      chan struct{}              // 关闭信号通道，用于停止所有操作
	subCount  int                        // 当前订阅者数量（统计用途）
	maxEvents int                        // 最大事件限制（可用于背压或限制）
}

// NewBroker 创建并返回一个新的具有默认设置的 Broker。
func NewBroker[T any]() *Broker[T] {
	return NewBrokerWithOptions[T](bufferSize, 1000)
}

// NewBrokerWithOptions 创建一个带有自定义通道缓冲区大小和最大事件数限制的 Broker。
func NewBrokerWithOptions[T any](channelBufferSize, maxEvents int) *Broker[T] {
	b := &Broker[T]{
		subs:      make(map[chan Event[T]]struct{}),
		done:      make(chan struct{}),
		subCount:  0,
		maxEvents: maxEvents,
	}
	return b
}

// Shutdown 优雅地关闭 Broker，停止处理新请求并通知所有订阅者。
func (b *Broker[T]) Shutdown() {
	select {
	case <-b.done: // 已经关闭，直接返回
		return
	default:
		close(b.done)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// 关闭所有订阅者的通道并从 map 中移除
	for ch := range b.subs {
		delete(b.subs, ch)
		close(ch)
	}

	b.subCount = 0
}

// Subscribe 注册一个订阅者并返回一个接收事件的通道。
// 该通道会在 ctx.Done() 信号触发或 Broker 关闭时自动注销并关闭。
func (b *Broker[T]) Subscribe(ctx context.Context) <-chan Event[T] {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 如果 Broker 已关闭，返回一个立即关闭的通道
	select {
	case <-b.done:
		ch := make(chan Event[T])
		close(ch)
		return ch
	default:
	}

	sub := make(chan Event[T], bufferSize)
	b.subs[sub] = struct{}{}
	b.subCount++

	// 启动后台协程监听上下文状态以便自动清理
	go func() {
		<-ctx.Done()

		b.mu.Lock()
		defer b.mu.Unlock()

		// 检查 Broker 是否已关闭，避免重复关闭通道
		select {
		case <-b.done:
			return
		default:
		}

		if _, ok := b.subs[sub]; ok {
			delete(b.subs, sub)
			close(sub)
			b.subCount--
		}
	}()

	return sub
}

// GetSubscriberCount 返回当前活跃的订阅者数量。
func (b *Broker[T]) GetSubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.subCount
}

// Publish 将一个事件分发给所有活跃的订阅者。
// 该操作是非阻塞的：如果订阅者的缓冲区已满，该订阅者将跳过当前事件。
func (b *Broker[T]) Publish(t EventType, payload T) {
	b.mu.RLock()
	// 如果 Broker 已关闭，直接放弃分发
	select {
	case <-b.done:
		b.mu.RUnlock()
		return
	default:
	}

	// 复制一份订阅者切片，以缩短持有读锁的时间
	subscribers := make([]chan Event[T], 0, len(b.subs))
	for sub := range b.subs {
		subscribers = append(subscribers, sub)
	}
	b.mu.RUnlock()

	event := Event[T]{Type: t, Payload: payload}

	// 循环发送，使用 select 默认分支保证非阻塞
	for _, sub := range subscribers {
		select {
		case sub <- event:
		default:
			// 如果通道已满，则消息无法在不阻塞的情况下发送，直接忽略
		}
	}
}
