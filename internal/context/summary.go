package context

import (
	"context"
	"fmt"
	"strings"
)

// SummaryManager 递归摘要管理器，处理长对话以降低 Token 消耗
type SummaryManager struct {
	maxTokens      int
	summaryBuffer  []Message
	currentMessages []Message
}

// Message 对话消息
type Message struct {
	Role    string // "user", "assistant", "system"
	Content string
}

// NewSummaryManager 创建新的摘要管理器
func NewSummaryManager(maxTokens int) *SummaryManager {
	return &SummaryManager{
		maxTokens:      maxTokens,
		summaryBuffer:  make([]Message, 0),
		currentMessages: make([]Message, 0),
	}
}

// AddMessage 添加消息到上下文
func (sm *SummaryManager) AddMessage(role, content string) {
	message := Message{
		Role:    role,
		Content: content,
	}

	sm.currentMessages = append(sm.currentMessages, message)

	// 检查是否需要摘要
	if sm.estimateTokens() > sm.maxTokens {
		sm.summarize()
	}
}

// GetContext 获取当前上下文（包括摘要和最新消息）
func (sm *SummaryManager) GetContext() []Message {
	context := make([]Message, 0, len(sm.summaryBuffer)+len(sm.currentMessages))
	context = append(context, sm.summaryBuffer...)
	context = append(context, sm.currentMessages...)
	return context
}

// summarize 执行摘要
func (sm *SummaryManager) summarize() {
	if len(sm.currentMessages) == 0 {
		return
	}

	// 创建摘要
	summary := sm.createSummary()

	// 将摘要添加到摘要缓冲区
	sm.summaryBuffer = append(sm.summaryBuffer, Message{
		Role:    "system",
		Content: summary,
	})

	// 清空当前消息，保留最近几条
	if len(sm.currentMessages) > 5 {
		sm.currentMessages = sm.currentMessages[len(sm.currentMessages)-5:]
	}
}

// createSummary 创建摘要
func (sm *SummaryManager) createSummary() string {
	var sb strings.Builder

	sb.WriteString("=== Conversation Summary ===\n")
	sb.WriteString(fmt.Sprintf("Total messages summarized: %d\n", len(sm.currentMessages)))

	// 提取关键信息
	userMessages := 0
	assistantMessages := 0

	for _, msg := range sm.currentMessages {
		switch msg.Role {
		case "user":
			userMessages++
		case "assistant":
			assistantMessages++
		}

		// 提取重要内容
		if sm.isImportant(msg) {
			sb.WriteString(fmt.Sprintf("[%s]: %s\n", msg.Role, sm.truncate(msg.Content, 100)))
		}
	}

	sb.WriteString(fmt.Sprintf("Statistics: %d user messages, %d assistant messages\n", userMessages, assistantMessages))

	return sb.String()
}

// isImportant 判断消息是否重要
func (sm *SummaryManager) isImportant(msg Message) bool {
	keywords := []string{
		"error", "failed", "critical", "warning",
		"fix", "resolve", "issue", "problem",
		"remediate", "anomaly", "incident",
	}

	content := strings.ToLower(msg.Content)
	for _, keyword := range keywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}

	return false
}

// truncate 截断文本
func (sm *SummaryManager) truncate(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// estimateTokens 估算 Token 数量
func (sm *SummaryManager) estimateTokens() int {
	total := 0
	for _, msg := range sm.currentMessages {
		// 粗略估算：1 token ≈ 4 字符
		total += len(msg.Content) / 4
	}
	return total
}

// Reset 重置摘要管理器
func (sm *SummaryManager) Reset() {
	sm.summaryBuffer = make([]Message, 0)
	sm.currentMessages = make([]Message, 0)
}

// GetSummary 获取摘要
func (sm *SummaryManager) GetSummary() string {
	if len(sm.summaryBuffer) == 0 {
		return "No summary available"
	}
	return sm.summaryBuffer[len(sm.summaryBuffer)-1].Content
}
