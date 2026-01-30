package agent

import (
	"context"
	"sync"

	"github.com/cloudwego/eino/adk"
)

// ConversationStore 对话存储接口
type ConversationStore interface {
	// Add 添加一条消息到存储
	Add(ctx context.Context, msg adk.Message) error
	// List 获取所有消息历史
	List(ctx context.Context) ([]adk.Message, error)
	// Clear 清空消息历史
	Clear(ctx context.Context) error
}

// MemoryStore 内存实现的对话存储
type MemoryStore struct {
	mu   sync.RWMutex
	msgs []adk.Message
}

// NewMemoryStore 创建一个新的内存存储
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		msgs: make([]adk.Message, 0),
	}
}

// Add 添加一条消息
func (s *MemoryStore) Add(ctx context.Context, msg adk.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgs = append(s.msgs, msg)
	return nil
}

// List 获取所有消息
func (s *MemoryStore) List(ctx context.Context) ([]adk.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// 返回副本，避免外部修改
	result := make([]adk.Message, len(s.msgs))
	copy(result, s.msgs)
	return result, nil
}

// Clear 清空所有消息
func (s *MemoryStore) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgs = nil
	return nil
}
