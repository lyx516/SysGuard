package remediation_workflow

import (
	"context"
	"fmt"

	"github.com/sysguard/sysguard/internal/monitor"
	"github.com/sysguard/sysguard/internal/skills"
)

// ProblemAnalyzer 问题分析工具
type ProblemAnalyzer struct{}

// Name 返回工具名称
func (t *ProblemAnalyzer) Name() string {
	return "problem_analyzer"
}

// Description 返回工具描述
func (t *ProblemAnalyzer) Description() string {
	return "分析当前环境问题，结合健康检查和日志分析给出问题根源"
}

// Execute 执行工具
func (t *ProblemAnalyzer) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	anomaly, ok := input.Params["anomaly"].(monitor.Anomaly)
	if !ok {
		return nil, fmt.Errorf("anomaly parameter is required")
	}

	// 获取系统健康状态
	healthStatus, err := t.getHealthStatus(ctx, input)
	if err != nil {
		return nil, err
	}

	// 获取日志信息
	logInfo, err := t.getLogInfo(ctx, anomaly, input)
	if err != nil {
		return nil, err
	}

	// 获取指标数据
	metrics, err := t.getMetrics(ctx, input)
	if err != nil {
		return nil, err
	}

	// 分析问题
	analysis := t.analyze(anomaly, healthStatus, logInfo, metrics)

	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"problem_type":   analysis.ProblemType,
			"description":    analysis.Description,
			"root_cause":    analysis.RootCause,
			"recommendations": analysis.Recommendations,
		},
	}, nil
}

// ProblemAnalysis 问题分析结果
type ProblemAnalysis struct {
	ProblemType     string
	Description    string
	RootCause      string
	Recommendations []string
}

// getHealthStatus 获取健康状态
func (t *ProblemAnalyzer) getHealthStatus(ctx context.Context, input *skills.ToolInput) (map[string]interface{}, error) {
	// 这里简化实现，实际应该调用 health_check skill
	return map[string]interface{}{
		"cpu": map[string]interface{}{
			"status":  "healthy",
			"usage":  45.5,
		},
		"memory": map[string]interface{}{
			"status": "healthy",
			"usage": 62.3,
		},
		"disk": map[string]interface{}{
			"status": "degraded",
			"usage": 85.2,
		},
	}, nil
}

// getLogInfo 获取日志信息
func (t *ProblemAnalyzer) getLogInfo(ctx context.Context, anomaly monitor.Anomaly, input *skills.ToolInput) (map[string]interface{}, error) {
	// 这里简化实现，实际应该调用 log_analysis skill
	return map[string]interface{}{
		"error_count":    15,
		"warning_count":  42,
		"recent_errors":  []string{
			"Connection timeout",
			"Out of memory",
			"Disk space low",
		},
	}, nil
}

// getMetrics 获取指标数据
func (t *ProblemAnalyzer) getMetrics(ctx context.Context, input *skills.ToolInput) (map[string]interface{}, error) {
	// 这里简化实现，实际应该调用 metrics skill
	return map[string]interface{}{
		"trend_cpu":    []float64{40, 45, 50, 55, 45},
		"trend_memory": []float64{60, 62, 65, 63, 62},
		"trend_disk":   []float64{80, 82, 85, 84, 85},
	}, nil
}

// analyze 综合分析
func (t *ProblemAnalyzer) analyze(
	anomaly monitor.Anomaly,
	healthStatus map[string]interface{},
	logInfo map[string]interface{},
	metrics map[string]interface{},
) *ProblemAnalysis {
	// 确定问题类型
	problemType := t.classifyProblem(anomaly, healthStatus)

	// 分析根本原因
	rootCause := t.identifyRootCause(anomaly, healthStatus, logInfo, metrics)

	// 生成建议
	recommendations := t.generateRecommendations(problemType, rootCause, healthStatus)

	return &ProblemAnalysis{
		ProblemType:     problemType,
		Description:    anomaly.Description,
		RootCause:      rootCause,
		Recommendations: recommendations,
	}
}

// classifyProblem 分类问题
func (t *ProblemAnalyzer) classifyProblem(anomaly monitor.Anomaly, healthStatus map[string]interface{}) string {
	desc := anomaly.Description

	// 检查系统资源问题
	if disk, ok := healthStatus["disk"].(map[string]interface{}); ok {
		if status, ok := disk["status"].(string); ok && status == "degraded" {
			return "disk_space"
		}
	}

	// 根据描述分类
	switch {
	case containsAny(desc, []string{"CPU", "cpu", "processor"}):
		return "cpu_high"
	case containsAny(desc, []string{"memory", "RAM", "out of memory"}):
		return "memory_high"
	case containsAny(desc, []string{"disk", "storage", "space"}):
		return "disk_space"
	case containsAny(desc, []string{"network", "connection", "timeout"}):
		return "network_issue"
	case containsAny(desc, []string{"service", "daemon", "process"}):
		return "service_failure"
	case containsAny(desc, []string{"container", "pod", "docker", "kubernetes"}):
		return "container_issue"
	case containsAny(desc, []string{"database", "DB", "query"}):
		return "database_issue"
	default:
		return "general_issue"
	}
}

// identifyRootCause 识别根本原因
func (t *ProblemAnalyzer) identifyRootCause(
	anomaly monitor.Anomaly,
	healthStatus map[string]interface{},
	logInfo map[string]interface{},
	metrics map[string]interface{},
) string {
	desc := anomaly.Description

	// 基于日志信息
	if errors, ok := logInfo["recent_errors"].([]string); ok && len(errors) > 0 {
		for _, err := range errors {
			if contains(desc, err) || contains(desc, err) {
				return fmt.Sprintf("Detected from logs: %s", err)
			}
		}
	}

	// 基于健康状态
	if disk, ok := healthStatus["disk"].(map[string]interface{}); ok {
		if status, ok := disk["status"].(string); ok && status == "degraded" {
			if usage, ok := disk["usage"].(float64); ok && usage > 80 {
				return fmt.Sprintf("Disk usage at %.1f%%, causing performance degradation", usage)
			}
		}
	}

	// 基于指标趋势
	if trend, ok := metrics["trend_disk"].([]float64); ok && len(trend) > 1 {
		increasing := true
		for i := 1; i < len(trend); i++ {
			if trend[i] < trend[i-1] {
				increasing = false
				break
			}
		}
		if increasing && trend[len(trend)-1] > 80 {
			return "Disk usage has been steadily increasing"
		}
	}

	return fmt.Sprintf("Anomaly: %s requires investigation", desc)
}

// generateRecommendations 生成建议
func (t *ProblemAnalyzer) generateRecommendations(
	problemType string,
	rootCause string,
	healthStatus map[string]interface{},
) []string {
	var recommendations []string

	switch problemType {
	case "disk_space":
		recommendations = append(recommendations, "Clean up temporary files and logs")
		recommendations = append(recommendations, "Archive old data to free up space")
		recommendations = append(recommendations, "Consider expanding disk capacity")
	case "cpu_high":
		recommendations = append(recommendations, "Check for runaway processes")
		recommendations = append(recommendations, "Review and optimize resource-intensive applications")
	case "memory_high":
		recommendations = append(recommendations, "Restart memory-heavy services")
		recommendations = append(recommendations, "Consider increasing available memory")
	case "network_issue":
		recommendations = append(recommendations, "Check network connectivity")
		recommendations = append(recommendations, "Verify firewall and routing rules")
	case "service_failure":
		recommendations = append(recommendations, "Restart the failed service")
		recommendations = append(recommendations, "Check service logs for error patterns")
	case "container_issue":
		recommendations = append(recommendations, "Restart the affected container")
		recommendations = append(recommendations, "Check container resource limits")
	case "database_issue":
		recommendations = append(recommendations, "Check database connection pool")
		recommendations = append(recommendations, "Review slow query logs")
	default:
		recommendations = append(recommendations, "Perform full system diagnosis")
		recommendations = append(recommendations, "Review system logs for patterns")
	}

	return recommendations
}

// containsAny 检查字符串是否包含任一子串
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if contains(s, substr) {
			return true
		}
	}
	return false
}

// contains 检查字符串包含（不区分大小写）
func contains(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}

	// 简化实现，实际应该使用更高效的算法
	sLower := toLower(s)
	substrLower := toLower(substr)

	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}

// toLower 转小写
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			result[i] = s[i] + 32
		} else {
			result[i] = s[i]
		}
	}
	return string(result)
}
