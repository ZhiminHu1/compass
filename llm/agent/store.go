package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
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
	mu              sync.RWMutex
	msgs            []adk.Message
	maxMessages     int // 最大保留消息数
	maxToolResponse int // 工具响应最大长度（字符数）
}

// NewMemoryStore 创建一个新的内存存储
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		msgs:            make([]adk.Message, 0),
		maxMessages:     20,   // 默认保留最近20条消息
		maxToolResponse: 2000, // 工具响应最大2000字符
	}
}

// Add 添加一条消息（带滑动窗口和工具结果压缩）
func (s *MemoryStore) Add(ctx context.Context, msg adk.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// 压缩工具响应！
	if msg.Role == schema.Tool {
		msg = s.compressToolResponse(msg)
	}

	// 添加压缩后的消息
	s.msgs = append(s.msgs, msg)

	// 滑动窗口：超过限制时删除最旧的消息
	if len(s.msgs) > s.maxMessages {
		s.msgs = s.msgs[len(s.msgs)-s.maxMessages:]
	}

	return nil
}

// compressToolResponse 压缩工具响应消息
func (s *MemoryStore) compressToolResponse(msg adk.Message) adk.Message {
	// 如果内容不大，直接返回
	if len(msg.Content) <= s.maxToolResponse {
		return msg
	}

	// 保存原始长度
	originalLen := len(msg.Content)

	// 智能截断：尝试在句号、换行符处截断
	truncated := msg.Content[:s.maxToolResponse]

	// 寻找合适的截断点
	breakPoints := []string{"。\n", ".\n", "。", ". ", "\n\n", "\n"}
	cutoff := s.maxToolResponse

	for _, bp := range breakPoints {
		if idx := findLastIndex(truncated, bp); idx > s.maxToolResponse/2 {
			cutoff = idx + len(bp)
			break
		}
	}

	// 创建压缩后的内容
	compressed := msg.Content[:cutoff]
	compressed += fmt.Sprintf(
		"\n\n[Content truncated: original %d chars (%d tokens) -> %d chars (%d tokens), saved %.1f%%]",
		originalLen,
		originalLen/3,
		cutoff,
		cutoff/3,
		float64(originalLen-cutoff)/float64(originalLen)*100,
	)

	// 返回压缩后的消息
	return &schema.Message{
		Role:    msg.Role,
		Content: compressed,
	}
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

// findLastIndex 查找最后一个匹配的位置
func findLastIndex(s, substr string) int {
	idx := -1
	pos := 0
	for {
		i := indexOf(s[pos:], substr)
		if i == -1 {
			break
		}
		idx = pos + i
		pos = idx + len(substr)
	}
	return idx
}

// indexOf 查找子串位置
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
