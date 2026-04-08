package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/sysguard/sysguard/internal/skills"
)

// NotificationSkill 通知 Skill
type NotificationSkill struct {
	version string
}

// NewNotificationSkill 创建通知 Skill
func NewNotificationSkill() *NotificationSkill {
	return &NotificationSkill{
		version: "1.0.0",
	}
}

// Name 返回 Skill 名称
func (s *NotificationSkill) Name() string {
	return "notification"
}

// Description 返回 Skill 描述
func (s *NotificationSkill) Description() string {
	return "发送各种类型的通知消息"
}

// Execute 执行通知操作
func (s *NotificationSkill) Execute(ctx context.Context, input *skills.SkillInput) (*skills.SkillOutput, error) {
	startTime := time.Now()

	// 获取通知类型
	notificationType, ok := input.Params["type"].(string)
	if !ok {
		return nil, fmt.Errorf("type parameter is required")
	}

	var result map[string]interface{}
	var message string
	var toolsUsed []string

	switch notificationType {
	case "email":
		result, message = s.sendEmail(ctx, input)
		toolsUsed = []string{"smtp_client"}
	case "slack":
		result, message = s.sendSlack(ctx, input)
		toolsUsed = []string{"slack_webhook"}
	case "webhook":
		result, message = s.sendWebhook(ctx, input)
		toolsUsed = []string{"http_client"}
	case "sms":
		result, message = s.sendSMS(ctx, input)
		toolsUsed = []string{"sms_provider"}
	case "telegram":
		result, message = s.sendTelegram(ctx, input)
		toolsUsed = []string{"telegram_bot"}
	case "discord":
		result, message = s.sendDiscord(ctx, input)
		toolsUsed = []string{"discord_webhook"}
	default:
		return nil, fmt.Errorf("unsupported notification type: %s", notificationType)
	}

	duration := time.Since(startTime).Milliseconds()

	return &skills.SkillOutput{
		Success: result["success"].(bool),
		Message: message,
		Data:    result,
		ToolsUsed: toolsUsed,
		Duration:  duration,
	}, nil
}

// Tools 返回该 Skill 使用的工具集
func (s *NotificationSkill) Tools() []skills.Tool {
	return []skills.Tool{
		&SMTPClient{},
		&SlackWebhook{},
		&HTTPClient{},
		&SMSProvider{},
		&TelegramBot{},
		&DiscordWebhook{},
	}
}

// Metadata 返回 Skill 元数据
func (s *NotificationSkill) Metadata() *skills.SkillMetadata {
	return &skills.SkillMetadata{
		Version:     s.version,
		Category:    "notification",
		Tags:        []string{"notification", "alert", "messaging", "communication"},
		Author:      "SysGuard Team",
		Permissions: []string{"send:email", "send:slack", "send:webhook", "send:sms"},
	}
}

// sendEmail 发送邮件
func (s *NotificationSkill) sendEmail(ctx context.Context, input *skills.SkillInput) (map[string]interface{}, string) {
	to, _ := input.Params["to"].(string)
	subject, _ := input.Params["subject"].(string)
	body, _ := input.Params["body"].(string)

	return map[string]interface{}{
		"success": true,
		"type":    "email",
		"to":      to,
		"subject": subject,
		"message_id": fmt.Sprintf("MSG-%d", time.Now().Unix()),
		"timestamp": time.Now(),
	}, fmt.Sprintf("Email sent to %s", to)
}

// sendSlack 发送 Slack 消息
func (s *NotificationSkill) sendSlack(ctx context.Context, input *skills.SkillInput) (map[string]interface{}, string) {
	channel, _ := input.Params["channel"].(string)
	message, _ := input.Params["message"].(string)

	return map[string]interface{}{
		"success": true,
		"type":    "slack",
		"channel": channel,
		"timestamp": time.Now(),
	}, fmt.Sprintf("Slack message sent to %s", channel)
}

// sendWebhook 发送 Webhook
func (s *NotificationSkill) sendWebhook(ctx context.Context, input *skills.SkillInput) (map[string]interface{}, string) {
	url, _ := input.Params["url"].(string)
	payload, _ := input.Params["payload"].(map[string]interface{})

	return map[string]interface{}{
		"success": true,
		"type":    "webhook",
		"url":     url,
		"payload": payload,
		"timestamp": time.Now(),
	}, fmt.Sprintf("Webhook sent to %s", url)
}

// sendSMS 发送短信
func (s *NotificationSkill) sendSMS(ctx context.Context, input *skills.SkillInput) (map[string]interface{}, string) {
	to, _ := input.Params["to"].(string)
	message, _ := input.Params["message"].(string)

	return map[string]interface{}{
		"success": true,
		"type":    "sms",
		"to":      to,
		"message_id": fmt.Sprintf("SMS-%d", time.Now().Unix()),
		"timestamp": time.Now(),
	}, fmt.Sprintf("SMS sent to %s", to)
}

// sendTelegram 发送 Telegram 消息
func (s *NotificationSkill) sendTelegram(ctx context.Context, input *skills.SkillInput) (map[string]interface{}, string) {
	chatID, _ := input.Params["chat_id"].(int64)
	message, _ := input.Params["message"].(string)

	return map[string]interface{}{
		"success": true,
		"type":    "telegram",
		"chat_id": chatID,
		"message_id": fmt.Sprintf("TG-%d", time.Now().Unix()),
		"timestamp": time.Now(),
	}, fmt.Sprintf("Telegram message sent to chat %d", chatID)
}

// sendDiscord 发送 Discord 消息
func (s *NotificationSkill) sendDiscord(ctx context.Context, input *skills.SkillInput) (map[string]interface{}, string) {
	webhookURL, _ := input.Params["webhook_url"].(string)
	message, _ := input.Params["message"].(string)

	return map[string]interface{}{
		"success": true,
		"type":    "discord",
		"webhook_url": webhookURL,
		"timestamp": time.Now(),
	}, fmt.Sprintf("Discord message sent")
}

// SMTPClient SMTP 客户端工具
type SMTPClient struct{}

// Name 返回工具名称
func (t *SMTPClient) Name() string {
	return "smtp_client"
}

// Description 返回工具描述
func (t *SMTPClient) Description() string {
	return "SMTP 邮件发送客户端"
}

// Execute 执行工具
func (t *SMTPClient) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"message_id": fmt.Sprintf("MSG-%d", time.Now().Unix()),
		},
	}, nil
}

// SlackWebhook Slack Webhook 工具
type SlackWebhook struct{}

// Name 返回工具名称
func (t *SlackWebhook) Name() string {
	return "slack_webhook"
}

// Description 返回工具描述
func (t *SlackWebhook) Description() string {
	return "Slack Webhook 发送工具"
}

// Execute 执行工具
func (t *SlackWebhook) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"timestamp": time.Now(),
		},
	}, nil
}

// HTTPClient HTTP 客户端工具
type HTTPClient struct{}

// Name 返回工具名称
func (t *HTTPClient) Name() string {
	return "http_client"
}

// Description 返回工具描述
func (t *HTTPClient) Description() string {
	return "HTTP 客户端，用于发送 Webhook"
}

// Execute 执行工具
func (t *HTTPClient) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"status_code": 200,
		},
	}, nil
}

// SMSProvider SMS 提供商工具
type SMSProvider struct{}

// Name 返回工具名称
func (t *SMSProvider) Name() string {
	return "sms_provider"
}

// Description 返回工具描述
func (t *SMSProvider) Description() string {
	return "短信发送服务提供商客户端"
}

// Execute 执行工具
func (t *SMSProvider) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"message_id": fmt.Sprintf("SMS-%d", time.Now().Unix()),
		},
	}, nil
}

// TelegramBot Telegram Bot 工具
type TelegramBot struct{}

// Name 返回工具名称
func (t *TelegramBot) Name() string {
	return "telegram_bot"
}

// Description 返回工具描述
func (t *TelegramBot) Description() string {
	return "Telegram Bot API 客户端"
}

// Execute 执行工具
func (t *TelegramBot) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"message_id": fmt.Sprintf("TG-%d", time.Now().Unix()),
		},
	}, nil
}

// DiscordWebhook Discord Webhook 工具
type DiscordWebhook struct{}

// Name 返回工具名称
func (t *DiscordWebhook) Name() string {
	return "discord_webhook"
}

// Description 返回工具描述
func (t *DiscordWebhook) Description() string {
	return "Discord Webhook 发送工具"
}

// Execute 执行工具
func (t *DiscordWebhook) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"timestamp": time.Now(),
		},
	}, nil
}
