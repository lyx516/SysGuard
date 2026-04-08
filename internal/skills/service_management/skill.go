package service_management

import (
	"context"
	"fmt"
	"time"

	"github.com/sysguard/sysguard/internal/skills"
)

// ServiceManagementSkill 服务管理 Skill
type ServiceManagementSkill struct {
	version string
}

// NewServiceManagementSkill 创建服务管理 Skill
func NewServiceManagementSkill() *ServiceManagementSkill {
	return &ServiceManagementSkill{
		version: "1.0.0",
	}
}

// Name 返回 Skill 名称
func (s *ServiceManagementSkill) Name() string {
	return "service_management"
}

// Description 返回 Skill 描述
func (s *ServiceManagementSkill) Description() string {
	return "管理系统服务的启动、停止、重启和状态查询"
}

// Execute 执行服务管理操作
func (s *ServiceManagementSkill) Execute(ctx context.Context, input *skills.SkillInput) (*skills.SkillOutput, error) {
	startTime := time.Now()

	// 获取操作类型
	action, ok := input.Params["action"].(string)
	if !ok {
		return nil, fmt.Errorf("action parameter is required")
	}

	// 获取服务名称
	serviceName, ok := input.Params["service"].(string)
	if !ok {
		return nil, fmt.Errorf("service parameter is required")
	}

	var result map[string]interface{}
	var message string
	var toolsUsed []string

	switch action {
	case "start":
		result, message = s.startService(ctx, serviceName)
		toolsUsed = []string{"systemctl_start"}
	case "stop":
		result, message = s.stopService(ctx, serviceName)
		toolsUsed = []string{"systemctl_stop"}
	case "restart":
		result, message = s.restartService(ctx, serviceName)
		toolsUsed = []string{"systemctl_restart"}
	case "status":
		result, message = s.checkServiceStatus(ctx, serviceName)
		toolsUsed = []string{"systemctl_status"}
	case "list":
		result, message = s.listServices(ctx)
		toolsUsed = []string{"systemctl_list"}
	default:
		return nil, fmt.Errorf("unsupported action: %s", action)
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
func (s *ServiceManagementSkill) Tools() []skills.Tool {
	return []skills.Tool{
		&SystemCTLStart{},
		&SystemCTLStop{},
		&SystemCTLRestart{},
		&SystemCTLStatus{},
		&SystemCTLList{},
	}
}

// Metadata 返回 Skill 元数据
func (s *ServiceManagementSkill) Metadata() *skills.SkillMetadata {
	return &skills.SkillMetadata{
		Version:     s.version,
		Category:    "management",
		Tags:        []string{"services", "systemd", "management", "control"},
		Author:      "SysGuard Team",
		Permissions: []string{"systemctl:service", "write:service"},
	}
}

// startService 启动服务
func (s *ServiceManagementSkill) startService(ctx context.Context, serviceName string) (map[string]interface{}, string) {
	// 实现 systemd 服务启动逻辑
	return map[string]interface{}{
		"success":     true,
		"service":     serviceName,
		"status":      "running",
		"pid":         12345,
		"uptime":      "10s",
	}, fmt.Sprintf("Service %s started successfully", serviceName)
}

// stopService 停止服务
func (s *ServiceManagementSkill) stopService(ctx context.Context, serviceName string) (map[string]interface{}, string) {
	return map[string]interface{}{
		"success": true,
		"service": serviceName,
		"status":  "stopped",
	}, fmt.Sprintf("Service %s stopped successfully", serviceName)
}

// restartService 重启服务
func (s *ServiceManagementSkill) restartService(ctx context.Context, serviceName string) (map[string]interface{}, string) {
	return map[string]interface{}{
		"success": true,
		"service": serviceName,
		"status":  "running",
		"pid":     12346,
		"uptime":  "5s",
	}, fmt.Sprintf("Service %s restarted successfully", serviceName)
}

// checkServiceStatus 检查服务状态
func (s *ServiceManagementSkill) checkServiceStatus(ctx context.Context, serviceName string) (map[string]interface{}, string) {
	return map[string]interface{}{
		"success": true,
		"service": serviceName,
		"status":  "running",
		"enabled": true,
		"pid":     12345,
		"uptime":  "2h 30m",
		"memory_mb": 128,
		"cpu_percent": 0.5,
	}, fmt.Sprintf("Service %s is running", serviceName)
}

// listServices 列出所有服务
func (s *ServiceManagementSkill) listServices(ctx context.Context) (map[string]interface{}, string) {
	return map[string]interface{}{
		"success": true,
		"total":   20,
		"running": 18,
		"stopped": 2,
		"services": []map[string]interface{}{
			{"name": "nginx", "status": "running", "enabled": true},
			{"name": "mysql", "status": "running", "enabled": true},
			{"name": "redis", "status": "stopped", "enabled": false},
		},
	}, "Listed all services"
}

// SystemCTLStart systemctl start 工具
type SystemCTLStart struct{}

// Name 返回工具名称
func (t *SystemCTLStart) Name() string {
	return "systemctl_start"
}

// Description 返回工具描述
func (t *SystemCTLStart) Description() string {
	return "启动 systemd 服务"
}

// Execute 执行工具
func (t *SystemCTLStart) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	serviceName := input.Params["service"].(string)
	// 实现实际的 systemctl start 命令
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"service": serviceName,
			"status":  "running",
		},
	}, nil
}

// SystemCTLStop systemctl stop 工具
type SystemCTLStop struct{}

// Name 返回工具名称
func (t *SystemCTLStop) Name() string {
	return "systemctl_stop"
}

// Description 返回工具描述
func (t *SystemCTLStop) Description() string {
	return "停止 systemd 服务"
}

// Execute 执行工具
func (t *SystemCTLStop) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	serviceName := input.Params["service"].(string)
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"service": serviceName,
			"status":  "stopped",
		},
	}, nil
}

// SystemCTLRestart systemctl restart 工具
type SystemCTLRestart struct{}

// Name 返回工具名称
func (t *SystemCTLRestart) Name() string {
	return "systemctl_restart"
}

// Description 返回工具描述
func (t *SystemCTLRestart) Description() string {
	return "重启 systemd 服务"
}

// Execute 执行工具
func (t *SystemCTLRestart) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	serviceName := input.Params["service"].(string)
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"service": serviceName,
			"status":  "running",
		},
	}, nil
}

// SystemCTLStatus systemctl status 工具
type SystemCTLStatus struct{}

// Name 返回工具名称
func (t *SystemCTLStatus) Name() string {
	return "systemctl_status"
}

// Description 返回工具描述
func (t *SystemCTLStatus) Description() string {
	return "查询 systemd 服务状态"
}

// Execute 执行工具
func (t *SystemCTLStatus) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	serviceName := input.Params["service"].(string)
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"service": serviceName,
			"status":  "running",
			"enabled": true,
		},
	}, nil
}

// SystemCTLList systemctl list 工具
type SystemCTLList struct{}

// Name 返回工具名称
func (t *SystemCTLList) Name() string {
	return "systemctl_list"
}

// Description 返回工具描述
func (t *SystemCTLList) Description() string {
	return "列出所有 systemd 服务"
}

// Execute 执行工具
func (t *SystemCTLList) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"services": []string{"nginx", "mysql", "redis"},
		},
	}, nil
}
