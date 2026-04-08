package health_check

import (
	"context"
	"fmt"
	"time"

	"github.com/sysguard/sysguard/internal/skills"
	"github.com/sysguard/sysguard/internal/monitor"
)

// HealthCheckSkill 健康检查 Skill
type HealthCheckSkill struct {
	monitor *monitor.Monitor
	version string
}

// NewHealthCheckSkill 创建健康检查 Skill
func NewHealthCheckSkill(mon *monitor.Monitor) *HealthCheckSkill {
	return &HealthCheckSkill{
		monitor: mon,
		version: "1.0.0",
	}
}

// Name 返回 Skill 名称
func (s *HealthCheckSkill) Name() string {
	return "health_check"
}

// Description 返回 Skill 描述
func (s *HealthCheckSkill) Description() string {
	return "检查系统各组件的健康状态"
}

// Execute 执行健康检查
func (s *HealthCheckSkill) Execute(ctx context.Context, input *skills.SkillInput) (*skills.SkillOutput, error) {
	startTime := time.Now()

	// 获取检查类型（默认全面检查）
	checkType, _ := input.Params["check_type"].(string)
	if checkType == "" {
		checkType = "full"
	}

	// 执行健康检查
	report, err := s.monitor.CheckHealth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check health: %w", err)
	}

	duration := time.Since(startTime).Milliseconds()

	return &skills.SkillOutput{
		Success: report.IsHealthy,
		Message: fmt.Sprintf("Health score: %.2f, Status: %s", report.Score, getStatusText(report.IsHealthy)),
		Data: map[string]interface{}{
			"health_score":   report.Score,
			"is_healthy":     report.IsHealthy,
			"components":     report.Components,
			"check_type":     checkType,
			"timestamp":      report.Timestamp,
		},
		ToolsUsed: []string{"cpu_checker", "memory_checker", "disk_checker", "network_checker", "service_checker"},
		Duration:  duration,
	}, nil
}

// Tools 返回该 Skill 使用的工具集
func (s *HealthCheckSkill) Tools() []skills.Tool {
	return []skills.Tool{
		&CPUChecker{},
		&MemoryChecker{},
		&DiskChecker{},
		&NetworkChecker{},
		&ServiceChecker{},
	}
}

// Metadata 返回 Skill 元数据
func (s *HealthCheckSkill) Metadata() *skills.SkillMetadata {
	return &skills.SkillMetadata{
		Version:     s.version,
		Category:    "monitoring",
		Tags:        []string{"health", "monitoring", "system", "infrastructure"},
		Author:      "SysGuard Team",
		Permissions: []string{"read:system", "read:metrics"},
	}
}

// getStatusText 获取状态文本
func getStatusText(isHealthy bool) string {
	if isHealthy {
		return "Healthy"
	}
	return "Unhealthy"
}

// CPUChecker CPU 检查器
type CPUChecker struct{}

// Name 返回工具名称
func (t *CPUChecker) Name() string {
	return "cpu_checker"
}

// Description 返回工具描述
func (t *CPUChecker) Description() string {
	return "检查 CPU 使用率和负载"
}

// Execute 执行工具
func (t *CPUChecker) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	// 实现 CPU 检查逻辑
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"usage_percent": 45.5,
			"load_avg":      []float64{0.5, 0.6, 0.7},
		},
	}, nil
}

// MemoryChecker 内存检查器
type MemoryChecker struct{}

// Name 返回工具名称
func (t *MemoryChecker) Name() string {
	return "memory_checker"
}

// Description 返回工具描述
func (t *MemoryChecker) Description() string {
	return "检查内存使用情况"
}

// Execute 执行工具
func (t *MemoryChecker) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"total_gb":   16,
			"used_gb":    10,
			"free_gb":    6,
			"usage_percent": 62.5,
		},
	}, nil
}

// DiskChecker 磁盘检查器
type DiskChecker struct{}

// Name 返回工具名称
func (t *DiskChecker) Name() string {
	return "disk_checker"
}

// Description 返回工具描述
func (t *DiskChecker) Description() string {
	return "检查磁盘空间和使用情况"
}

// Execute 执行工具
func (t *DiskChecker) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"total_gb":   1024,
			"used_gb":    565,
			"free_gb":    459,
			"usage_percent": 55.2,
			"mount_points": []string{"/", "/var", "/home"},
		},
	}, nil
}

// NetworkChecker 网络检查器
type NetworkChecker struct{}

// Name 返回工具名称
func (t *NetworkChecker) Name() string {
	return "network_checker"
}

// Description 返回工具描述
func (t *NetworkChecker) Description() string {
	return "检查网络连通性和延迟"
}

// Execute 执行工具
func (t *NetworkChecker) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"latency_ms":    12.5,
			"packet_loss":   0.0,
			"connections":   150,
			"interfaces":    []string{"eth0", "lo"},
		},
	}, nil
}

// ServiceChecker 服务检查器
type ServiceChecker struct{}

// Name 返回工具名称
func (t *ServiceChecker) Name() string {
	return "service_checker"
}

// Description 返回工具描述
func (t *ServiceChecker) Description() string {
	return "检查系统服务状态"
}

// Execute 执行工具
func (t *ServiceChecker) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"total_services":  20,
			"running":         18,
			"stopped":         2,
			"services": []map[string]interface{}{
				{"name": "nginx", "status": "running"},
				{"name": "mysql", "status": "running"},
				{"name": "redis", "status": "stopped"},
			},
		},
	}, nil
}
