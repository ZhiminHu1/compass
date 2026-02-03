package renderer

import (
	"fmt"
	"strings"
	"time"
)

// Truncate 截断字符串到指定长度，添加省略号
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	// 简单处理：按字节截断，保留部分中文字符
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}

// FormatBytes 格式化字节数为人类可读格式
func FormatBytes(bytes int) string {
	const unit = 1024
	b := int64(bytes)
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// ShortenURL 缩短URL显示
func ShortenURL(url string) string {
	// 移除协议前缀
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "www.")

	// 如果仍然太长，截断
	if len(url) > 40 {
		return Truncate(url, 40)
	}
	return url
}

// FormatDuration 格式化时间间隔（接收毫秒数）
func FormatDuration(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	if d < time.Millisecond {
		return fmt.Sprintf("%dμs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}
