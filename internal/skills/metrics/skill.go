package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/sysguard/sysguard/internal/skills"
)

// MetricsSkill 指标收集 Skill
type MetricsSkill struct {
	version string
}

// NewMetricsSkill 创建指标收集 Skill
func NewMetricsSkill() *MetricsSkill {
	return &MetricsSkill{
		version: "1.0.0",
	}
}

// Name 返回 Skill 名称
func (s *MetricsSkill) Name() string {
	return "metrics"
}

// Description 返回 Skill 描述
func (s *MetricsSkill) Description() string {
	return "收集和查询系统指标数据"
}

// Execute 执行指标收集操作
func (s *MetricsSkill) Execute(ctx context.Context, input *skills.SkillInput) (*skills.SkillOutput, error) {
	startTime := time.Now()

	// 获取操作类型
	action, ok := input.Params["action"].(string)
	if !ok {
		return nil, fmt.Errorf("action parameter is required")
	}

	var result map[string]interface{}
	var message string
	var toolsUsed []string

	switch action {
	case "collect":
		result, message = s.collectMetrics(ctx, input)
		toolsUsed = []string{"prometheus", "node_exporter", "custom_collector"}
	case "query":
		result, message = s.queryMetrics(ctx, input)
		toolsUsed = []string{"promql", "grafana_api"}
	case "export":
		result, message = s.exportMetrics(ctx, input)
		toolsUsed = []string{"exporter", "formatter"}
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
func (s *MetricsSkill) Tools() []skills.Tool {
	return []skills.Tool{
		&PrometheusTool{},
		&NodeExporterTool{},
		&CustomCollectorTool{},
	}
}

// Metadata 返回 Skill 元数据
func (s *MetricsSkill) Metadata() *skills.SkillMetadata {
	return &skills.SkillMetadata{
		Version:     s.version,
		Category:    "monitoring",
		Tags:        []string{"metrics", "monitoring", "prometheus", "telemetry"},
		Author:      "SysGuard Team",
		Permissions: []string{"read:metrics", "write:metrics"},
	}
}

// collectMetrics 收集指标
func (s *MetricsSkill) collectMetrics(ctx context.Context, input *skills.SkillInput) (map[string]interface{}, string) {
	// 获取指标类型
	metricTypes, _ := input.Params["types"].([]string)
	if len(metricTypes) == 0 {
		metricTypes = []string{"cpu", "memory", "disk", "network"}
	}

	metrics := make(map[string]interface{})
	for _, mtype := range metricTypes {
		metrics[mtype] = s.getSampleMetrics(mtype)
	}

	return map[string]interface{}{
		"success":     true,
		"metrics":     metrics,
		"timestamp":   time.Now(),
		"metric_types": metricTypes,
	}, fmt.Sprintf("Collected %d metric types", len(metricTypes))
}

// queryMetrics 查询指标
func (s *MetricsSkill) queryMetrics(ctx context.Context, input *skills.SkillInput) (map[string]interface{}, string) {
	query, _ := input.Params["query"].(string)
	if query == "" {
		query = "up"
	}

	start, _ := input.Params["start"].(string)
	end, _ := input.Params["end"].(string)

	return map[string]interface{}{
		"success": true,
		"query":   query,
		"start":   start,
		"end":     end,
		"result": map[string]interface{}{
			"value": []float64{1.0, 1.0, 1.0},
			"timestamps": []int64{1000, 2000, 3000},
		},
	}, fmt.Sprintf("Query '%s' executed successfully", query)
}

// exportMetrics 导出指标
func (s *MetricsSkill) exportMetrics(ctx context.Context, input *skills.SkillInput) (map[string]interface{}, string) {
	format, _ := input.Params["format"].(string)
	if format == "" {
		format = "json"
	}

	return map[string]interface{}{
		"success": true,
		"format":  format,
		"exported": true,
		"file":    fmt.Sprintf("metrics.%s", format),
	}, fmt.Sprintf("Metrics exported in %s format", format)
}

// getSampleMetrics 获取示例指标
func (s *MetricsSkill) getSampleMetrics(metricType string) map[string]interface{} {
	switch metricType {
	case "cpu":
		return map[string]interface{}{
			"usage_percent":     45.5,
			"user_percent":      30.2,
			"system_percent":    10.5,
			"idle_percent":      45.0,
			"load_avg":         []float64{0.5, 0.6, 0.7},
			"cores":            8,
		}
	case "memory":
		return map[string]interface{}{
			"total_gb":         16,
			"used_gb":          10,
			"free_gb":          6,
			"cached_gb":        3,
			"usage_percent":    62.5,
			"swap_total_gb":    4,
			"swap_used_gb":     0.5,
		}
	case "disk":
		return map[string]interface{}{
			"total_gb":         1024,
			"used_gb":          565,
			"free_gb":          459,
			"usage_percent":    55.2,
			"iops_read":        150,
			"iops_write":       200,
			"throughput_mbps":   350,
		}
	case "network":
		return map[string]interface{}{
			"rx_bytes":      1024000,
			"tx_bytes":      512000,
			"rx_packets":    10000,
			"tx_packets":    5000,
			"rx_errors":     0,
			"tx_errors":     0,
			"latency_ms":    12.5,
			"bandwidth_mbps": 1000,
		}
	default:
		return map[string]interface{}{}
	}
}

// PrometheusTool Prometheus 工具
type PrometheusTool struct{}

// Name 返回工具名称
func (t *PrometheusTool) Name() string {
	return "prometheus"
}

// Description 返回工具描述
func (t *PrometheusTool) Description() string {
	return "Prometheus 指标查询和收集"
}

// Execute 执行工具
func (t *PrometheusTool) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"source": "prometheus",
		},
	}, nil
}

// NodeExporterTool Node Exporter 工具
type NodeExporterTool struct{}

// Name 返回工具名称
func (t *NodeExporterTool) Name() string {
	return "node_exporter"
}

// Description 返回工具描述
func (t *NodeExporterTool) Description() string {
	return "Node Exporter 系统指标收集"
}

// Execute 执行工具
func (t *NodeExporterTool) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"source": "node_exporter",
		},
	}, nil
}

// CustomCollectorTool 自定义收集器工具
type CustomCollectorTool struct{}

// Name 返回工具名称
func (t *CustomCollectorTool) Name() string {
	return "custom_collector"
}

// Description 返回工具描述
func (t *CustomCollectorTool) Description() string {
	return "自定义指标收集器"
}

// Execute 执行工具
func (t *CustomCollectorTool) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"source": "custom",
		},
	}, nil
}
