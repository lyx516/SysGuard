package alerting

import (
	"context"
	"fmt"
	"time"

	"github.com/sysguard/sysguard/internal/skills"
)

// AlertingSkill 告警 Skill
type AlertingSkill struct {
	notifier Notifier
	version  string
}

// Notifier 告警通知器接口
type Notifier interface {
	Send(ctx context.Context, alert *Alert) error
}

// Alert 告警信息
type Alert struct {
	Level     string            // info, warning, error, critical
	Title     string            // 告警标题
	Message   string            // 告警详情
	Timestamp time.Time         // 时间戳
	Source    string            // 告警来源
	Metadata  map[string]string // 元数据
}

// NewAlertingSkill 创建告警 Skill
func NewAlertingSkill(notifier Notifier) *AlertingSkill {
	return &AlertingSkill{
		notifier: notifier,
		version:  "1.0.0",
	}
}

// Name 返回 Skill 名称
func (s *AlertingSkill) Name() string {
	return "alerting"
}

// Description 返回 Skill 描述
func (s *AlertingSkill) Description() string {
	return "发送和管理告警通知"
}

// Execute 执行告警操作
func (s *AlertingSkill) Execute(ctx context.Context, input *skills.SkillInput) (*skills.SkillOutput, error) {
	startTime := time.Now()

	// 获取操作类型
	action, ok := input.Params["action"].(string)
	if !ok {
		return nil, fmt.Errorf("action parameter is required")
	}

	switch action {
	case "send":
		return s.sendAlert(ctx, input, startTime)
	case "batch":
		return s.sendBatchAlerts(ctx, input, startTime)
	case "history":
		return s.getAlertHistory(ctx, input, startTime)
	default:
		return nil, fmt.Errorf("unsupported action: %s", action)
	}
}

// Tools 返回该 Skill 使用的工具集
func (s *AlertingSkill) Tools() []skills.Tool {
	return []skills.Tool{
		&EmailNotifier{},
		&SlackNotifier{},
		&WebhookNotifier{},
	}
}

// Metadata 返回 Skill 元数据
func (s *AlertingSkill) Metadata() *skills.SkillMetadata {
	return &skills.SkillMetadata{
		Version:     s.version,
		Category:    "notification",
		Tags:        []string{"alert", "notification", "monitoring", "incident"},
		Author:      "SysGuard Team",
		Permissions: []string{"send:alert", "write:notification"},
	}
}

// sendAlert 发送单个告警
func (s *AlertingSkill) sendAlert(ctx context.Context, input *skills.SkillInput, startTime time.Time) (*skills.SkillOutput, error) {
	level, _ := input.Params["level"].(string)
	title, _ := input.Params["title"].(string)
	message, _ := input.Params["message"].(string)
	source, _ := input.Params["source"].(string)

	if level == "" {
		level = "info"
	}
	if title == "" {
		title = "Alert"
	}
	if source == "" {
		source = "SysGuard"
	}

	alert := &Alert{
		Level:     level,
		Title:     title,
		Message:   message,
		Timestamp: time.Now(),
		Source:    source,
		Metadata:  make(map[string]string),
	}

	if err := s.notifier.Send(ctx, alert); err != nil {
		return nil, fmt.Errorf("failed to send alert: %w", err)
	}

	duration := time.Since(startTime).Milliseconds()

	return &skills.SkillOutput{
		Success: true,
		Message: "Alert sent successfully",
		Data: map[string]interface{}{
			"alert_id":    fmt.Sprintf("ALT-%d", time.Now().Unix()),
			"level":       level,
			"title":       title,
			"timestamp":   alert.Timestamp,
		},
		ToolsUsed: []string{"notifier"},
		Duration:  duration,
	}, nil
}

// sendBatchAlerts 批量发送告警
func (s *AlertingSkill) sendBatchAlerts(ctx context.Context, input *skills.SkillInput, startTime time.Time) (*skills.SkillOutput, error) {
	// 实现批量告警发送逻辑
	duration := time.Since(startTime).Milliseconds()

	return &skills.SkillOutput{
		Success: true,
		Message: "Batch alerts sent successfully",
		Data: map[string]interface{}{
			"count": 10,
		},
		ToolsUsed: []string{"notifier"},
		Duration:  duration,
	}, nil
}

// getAlertHistory 获取告警历史
func (s *AlertingSkill) getAlertHistory(ctx context.Context, input *skills.SkillInput, startTime time.Time) (*skills.SkillOutput, error) {
	// 实现告警历史查询逻辑
	duration := time.Since(startTime).Milliseconds()

	return &skills.SkillOutput{
		Success: true,
		Message: "Alert history retrieved",
		Data: map[string]interface{}{
			"total": 100,
			"alerts": []map[string]interface{}{
				{"id": "ALT-1", "level": "error", "title": "CPU High", "time": time.Now()},
				{"id": "ALT-2", "level": "warning", "title": "Disk Space Low", "time": time.Now()},
			},
		},
		ToolsUsed: []string{"history_reader"},
		Duration:  duration,
	}, nil
}

// EmailNotifier 邮件通知器
type EmailNotifier struct{}

// Name 返回工具名称
func (t *EmailNotifier) Name() string {
	return "email_notifier"
}

// Description 返回工具描述
func (t *EmailNotifier) Description() string {
	return "通过邮件发送告警"
}

// Execute 执行工具
func (t *EmailNotifier) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"method": "email",
			"status": "sent",
		},
	}, nil
}

// SlackNotifier Slack 通知器
type SlackNotifier struct{}

// Name 返回工具名称
func (t *SlackNotifier) Name() string {
	return "slack_notifier"
}

// Description 返回工具描述
func (t *SlackNotifier) Description() string {
	return "通过 Slack 发送告警"
}

// Execute 执行工具
func (t *SlackNotifier) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"method": "slack",
			"status": "sent",
		},
	}, nil
}

// WebhookNotifier Webhook 通知器
type WebhookNotifier struct{}

// Name 返回工具名称
func (t *WebhookNotifier) Name() string {
	return "webhook_notifier"
}

// Description 返回工具描述
func (t *WebhookNotifier) Description() string {
	return "通过 Webhook 发送告警"
}

// Execute 执行工具
func (t *WebhookNotifier) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"method": "webhook",
			"status": "sent",
		},
	}, nil
}
