package pubsub

import "context"

const (
	// CreatedEvent 资源创建事件
	CreatedEvent EventType = "created"
	// UpdatedEvent 资源更新事件
	UpdatedEvent EventType = "updated"
	// DeletedEvent 资源删除事件
	DeletedEvent EventType = "deleted"
	// FinishedEvent 资源结束事件
	FinishedEvent EventType = "finished"
)

// Subscriber 订阅者接口，定义了获取事件通道的方法
type Subscriber[T any] interface {
	// Subscribe 返回一个只读的事件通道，并在 context 结束时自动关闭
	Subscribe(context.Context) <-chan Event[T]
}

type (
	// EventType 标识事件的类型
	EventType string

	// Event 代表资源生命周期中的一个事件
	Event[T any] struct {
		Type    EventType // 事件类型
		Payload T         // 事件携带的具体数据载荷
	}

	// Publisher 发布者接口，定义了发布事件的方法
	Publisher[T any] interface {
		// Publish 将指定类型和载荷的事件发布给所有订阅者
		Publish(EventType, T)
	}
)
