package pubsub

import (
	"context"
	"testing"
	"time"
)

// TestBrokerFlow 演示了基本的订阅和发布流程
func TestBrokerFlow(t *testing.T) {
	// 1. 初始化一个传递 string 类型数据的 Broker
	broker := NewBroker[string]()
	defer broker.Shutdown()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. 订阅消息流
	// Subscribe 返回一个 <-chan Event[string]
	events := broker.Subscribe(ctx)

	// 3. 异步模拟订阅者处理逻辑
	received := make(chan string, 1)
	go func() {
		for event := range events {
			if event.Type == CreatedEvent {
				received <- event.Payload
			}
		}
	}()

	// 4. 发布消息
	const testMsg = "hello pubsub"
	broker.Publish(CreatedEvent, testMsg)

	// 5. 验证是否接收成功
	select {
	case msg := <-received:
		if msg != testMsg {
			t.Errorf("期望得到 %s, 实际得到 %s", testMsg, msg)
		}
	case <-time.After(1 * time.Second):
		t.Error("接收消息超时")
	}
}

// TestAutoUnsubscribe 演示了基于 Context 的自动退订机制
func TestAutoUnsubscribe(t *testing.T) {
	broker := NewBroker[int]()

	ctx, cancel := context.WithCancel(context.Background())

	// 订阅
	_ = broker.Subscribe(ctx)
	if broker.GetSubscriberCount() != 1 {
		t.Errorf("期望订阅者数量为 1, 实际为 %d", broker.GetSubscriberCount())
	}

	// 取消 Context
	cancel()

	// 给一点点时间让后台清理协程运行
	time.Sleep(10 * time.Millisecond)

	// 验证订阅者数量是否归零
	if broker.GetSubscriberCount() != 0 {
		t.Errorf("Context 取消后订阅者未自动清理，当前数量: %d", broker.GetSubscriberCount())
	}
}

// TestNonBlockingPublish 演示了背压处理（非阻塞发送）
// 当订阅者处理太慢时，Broker 不会被阻塞，而是会丢失该订阅者的消息以保证系统通畅
func TestNonBlockingPublish(t *testing.T) {
	// 创建一个缓冲区非常小的 Broker
	// bufferSize 在代码中固定为 64，我们发送多于 64 条消息
	broker := NewBroker[int]()

	ctx := context.Background()
	// 订阅者 1：处理非常慢
	_ = broker.Subscribe(ctx)

	// 发布大量消息
	for i := 0; i < 100; i++ {
		// 即使订阅者通道满了，这里也不会阻塞
		broker.Publish(CreatedEvent, i)
	}

	// 如果能运行到这里，说明 Publish 是非阻塞的
	t.Log("Publish 成功通过了慢订阅者的背压测试")
}

// TestBrokerShutdown 演示了安全关闭
func TestBrokerShutdown(t *testing.T) {
	broker := NewBroker[string]()
	ctx := context.Background()

	events := broker.Subscribe(ctx)

	// 关闭 Broker
	broker.Shutdown()

	// 验证通道是否已关闭
	select {
	case _, ok := <-events:
		if ok {
			t.Error("Broker 关闭后，订阅通道仍未关闭")
		}
	case <-time.After(1 * time.Second):
		t.Error("Broker 关闭后，订阅通道关闭超时")
	}
}
