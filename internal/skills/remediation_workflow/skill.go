package remediation_workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/sysguard/sysguard/internal/monitor"
	"github.com/sysguard/sysguard/internal/rag"
	"github.com/sysguard/sysguard/internal/skills"
)

// RemediationWorkflowSkill 修复工作流 Skill
type RemediationWorkflowSkill struct {
	historyKB *rag.HistoryKnowledgeBase
	sopKB      *rag.KnowledgeBase
	skillReg   *skills.SkillRegistry
	version    string
}

// ProblemAnalysisResult 问题分析结果
type ProblemAnalysisResult struct {
	ProblemType     string
	Description    string
	RootCause      string
	Recommendations []string
	Summary        string
	Timestamp      time.Time
}

// RemediationPlan 修复计划
type RemediationPlan struct {
	ProblemAnalysis *ProblemAnalysisResult
	HistoricalRef  []*rag.HistoryRecord
	Steps         []PlanStep
	EstimatedTime time.Duration
	Priority      string
}

// PlanStep 计划步骤
type PlanStep struct {
	StepNumber    int
	SkillName     string
	Action        string
	Parameters    map[string]interface{}
	ExpectedResult string
}

// NewRemediationWorkflowSkill 创建新的修复工作流 Skill
func NewRemediationWorkflowSkill(
	historyKB *rag.HistoryKnowledgeBase,
	sopKB *rag.KnowledgeBase,
	skillReg *skills.SkillRegistry,
) *RemediationWorkflowSkill {
	return &RemediationWorkflowSkill{
		historyKB: historyKB,
		sopKB:      sopKB,
		skillReg:   skillReg,
		version:    "1.0.0",
	}
}

// Name 返回 Skill 名称
func (s *RemediationWorkflowSkill) Name() string {
	return "remediation_workflow"
}

// Description 返回 Skill 描述
func (s *RemediationWorkflowSkill) Description() string {
	return "执行规范的三步修复流程：问题分析、计划执行、文档生成"
}

// Execute 执行修复工作流
func (s *RemediationWorkflowSkill) Execute(ctx context.Context, input *skills.SkillInput) (*skills.SkillOutput, error) {
	startTime := time.Now()

	// 获取异常信息
	anomaly, ok := input.Params["anomaly"].(monitor.Anomaly)
	if !ok {
		return nil, fmt.Errorf("anomaly parameter is required")
	}

	// Step 1: 问题分析
	analysis, err := s.analyzeProblem(ctx, anomaly)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze problem: %w", err)
	}

	// Step 2: 检索相似历史记录
	similarRecords := s.historyKB.SearchSimilarRecords(
		ctx,
		analysis.ProblemType,
		analysis.Description,
		5,
	)

	// Step 3: 制定修复计划
	plan := s.createRemediationPlan(ctx, analysis, similarRecords)

	// Step 4: 执行修复计划
	executionResult, err := s.executePlan(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("failed to execute plan: %w", err)
	}

	// Step 5: 生成历史记录（如果是首次处理或处理成功）
	if len(similarRecords) == 0 && executionResult.Success {
		record := s.createHistoryRecord(anomaly, analysis, plan, executionResult)
		if err := s.historyKB.AddRecord(record); err != nil {
			// 记录失败但不影响主流程
			fmt.Printf("Warning: Failed to save history record: %v\n", err)
		}

		// 生成文档
		docGen := NewDocumentGenerator(s.historyKB.GetHistoryPath())
		docContent := docGen.generateDocument(anomaly, analysis, plan, executionResult)
		docPath, err := docGen.saveDocument(docContent, anomaly)
		if err != nil {
			fmt.Printf("Warning: Failed to save document: %v\n", err)
		} else {
			fmt.Printf("Document saved to: %s\n", docPath)
		}
	}

	duration := time.Since(startTime).Milliseconds()

	return &skills.SkillOutput{
		Success: executionResult.Success,
		Message: fmt.Sprintf("Remediation workflow completed. Success: %v", executionResult.Success),
		Data: map[string]interface{}{
			"analysis":         analysis,
			"similar_records":  similarRecords,
			"plan":            plan,
			"execution_result": executionResult,
			"history_saved":    len(similarRecords) == 0 && executionResult.Success,
		},
		ToolsUsed: []string{"problem_analyzer", "plan_executor", "document_generator"},
		Duration:  duration,
	}, nil
}

// Tools 返回该 Skill 使用的工具集
func (s *RemediationWorkflowSkill) Tools() []skills.Tool {
	return []skills.Tool{
		&ProblemAnalyzer{},
		NewPlanExecutor(s.skillReg),
		&DocumentGenerator{},
	}
}

// Metadata 返回 Skill 元数据
func (s *RemediationWorkflowSkill) Metadata() *skills.SkillMetadata {
	return &skills.SkillMetadata{
		Version:     s.version,
		Category:    "workflow",
		Tags:        []string{"remediation", "automation", "history", "sop"},
		Author:      "SysGuard Team",
		Permissions: []string{"read:history", "write:history", "execute:remediation"},
	}
}

// analyzeProblem 分析问题
func (s *RemediationWorkflowSkill) analyzeProblem(ctx context.Context, anomaly monitor.Anomaly) (*ProblemAnalysisResult, error) {
	// 1. 调用 health_check skill 获取当前系统状态
	healthSkill, ok := s.skillReg.Get("health_check")
	if !ok {
		return nil, fmt.Errorf("health_check skill not found")
	}

	healthOutput, err := healthSkill.Execute(ctx, &skills.SkillInput{
		Params: map[string]interface{}{
			"check_type": "full",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check health: %w", err)
	}

	// 2. 分析异常信息，确定问题类型
	problemType := s.classifyProblem(anomaly, healthOutput)

	// 3. 生成分析结果
	result := &ProblemAnalysisResult{
		ProblemType:  problemType,
		Description: fmt.Sprintf("%s (Severity: %s, Source: %s)",
			anomaly.Description, anomaly.Severity, anomaly.Source),
		RootCause:   s.inferRootCause(anomaly, healthOutput),
		Summary:     s.generateAnalysisSummary(anomaly, healthOutput),
		Timestamp:   time.Now(),
	}

	return result, nil
}

// classifyProblem 分类问题
func (s *RemediationWorkflowSkill) classifyProblem(
	anomaly monitor.Anomaly,
	healthOutput *skills.SkillOutput,
) string {
	// 根据异常描述和健康检查结果分类
	desc := anomaly.Description

	switch {
	case containsAny(desc, []string{"CPU", "memory", "disk", "network"}):
		return "system_resource"
	case containsAny(desc, []string{"service", "daemon", "process"}):
		return "service_failure"
	case containsAny(desc, []string{"database", "DB", "connection"}):
		return "database_issue"
	case containsAny(desc, []string{"container", "pod", "docker", "kubernetes"}):
		return "container_issue"
	default:
		return "general_issue"
	}
}

// inferRootCause 推断问题根源
func (s *RemediationWorkflowSkill) inferRootCause(
	anomaly monitor.Anomaly,
	healthOutput *skills.SkillOutput,
) string {
	// 结合异常和健康检查结果推断根源
	desc := anomaly.Description

	if healthData, ok := healthOutput.Data["components"].(map[string]interface{}); ok {
		// 检查具体组件状态
		for name, component := range healthData {
			if compMap, ok := component.(map[string]interface{}); ok {
				if status, ok := compMap["status"].(string); ok {
					if status == "down" || status == "degraded" {
						return fmt.Sprintf("Component '%s' is %s", name, status)
					}
				}
			}
		}
	}

	// 默认根源
	return fmt.Sprintf("Detected anomaly: %s", desc)
}

// generateAnalysisSummary 生成分析摘要
func (s *RemediationWorkflowSkill) generateAnalysisSummary(
	anomaly monitor.Anomaly,
	healthOutput *skills.SkillOutput,
) string {
	return fmt.Sprintf("Anomaly detected at %s with severity %s. "+
		"System health check completed. "+
		"Ready to proceed with remediation.",
		anomaly.Timestamp.Format(time.RFC3339),
		anomaly.Severity)
}

// createRemediationPlan 创建修复计划
func (s *RemediationWorkflowSkill) createRemediationPlan(
	ctx context.Context,
	analysis *ProblemAnalysisResult,
	similarRecords []*rag.HistoryRecord,
) *RemediationPlan {
	plan := &RemediationPlan{
		ProblemAnalysis: analysis,
		HistoricalRef:  similarRecords,
		Steps:         make([]PlanStep, 0),
		Priority:      determinePriority(analysis),
	}

	// 如果有相似历史记录，参考它们
	if len(similarRecords) > 0 {
		plan = s.refinePlanWithHistory(plan, similarRecords)
	} else {
		// 否则基于问题类型生成标准计划
		plan = s.generateStandardPlan(plan)
	}

	// 估算时间
	plan.EstimatedTime = s.estimateExecutionTime(plan)

	return plan
}

// refinePlanWithHistory 基于历史记录优化计划
func (s *RemediationWorkflowSkill) refinePlanWithHistory(
	plan *RemediationPlan,
	records []*rag.HistoryRecord,
) *RemediationPlan {
	// 从成功的历史记录中学习
	for _, record := range records {
		if record.Success && len(record.Steps) > 0 {
			plan.Steps = append(plan.Steps, PlanStep{
				StepNumber: len(plan.Steps) + 1,
				SkillName:  "history_reference",
				Action:     "Reference historical solution",
				Parameters: map[string]interface{}{
					"record_id":  record.ID,
					"solution":   record.Solution,
					"steps":      record.Steps,
				},
				ExpectedResult: "Apply proven solution from history",
			})
			break
		}
	}

	return plan
}

// generateStandardPlan 生成标准计划
func (s *RemediationWorkflowSkill) generateStandardPlan(plan *RemediationPlan) *RemediationPlan {
	switch plan.ProblemAnalysis.ProblemType {
	case "system_resource":
		plan.Steps = append(plan.Steps, PlanStep{
			StepNumber: 1,
			SkillName:  "metrics",
			Action:     "collect_metrics",
			Parameters: map[string]interface{}{
				"action": "collect",
				"types":  []string{"cpu", "memory", "disk"},
			},
			ExpectedResult: "Resource metrics collected",
		})
	case "service_failure":
		plan.Steps = append(plan.Steps, PlanStep{
			StepNumber: 1,
			SkillName:  "service_management",
			Action:     "restart_service",
			Parameters: map[string]interface{}{
				"action":  "restart",
				"service": "affected_service",
			},
			ExpectedResult: "Service restarted successfully",
		})
	case "container_issue":
		plan.Steps = append(plan.Steps, PlanStep{
			StepNumber: 1,
			SkillName:  "container_management",
			Action:     "restart_container",
			Parameters: map[string]interface{}{
				"action":    "restart",
				"type":      "docker",
				"container": "affected_container",
			},
			ExpectedResult: "Container restarted successfully",
		})
	default:
		// 通用计划
		plan.Steps = append(plan.Steps, PlanStep{
			StepNumber: 1,
			SkillName:  "health_check",
			Action:     "check_health",
			Parameters: map[string]interface{}{
				"check_type": "full",
			},
			ExpectedResult: "System health verified",
		})
	}

	return plan
}

// estimateExecutionTime 估算执行时间
func (s *RemediationWorkflowSkill) estimateExecutionTime(plan *RemediationPlan) time.Duration {
	// 每个步骤估算 2 分钟
	baseTime := time.Duration(len(plan.Steps)) * 2 * time.Minute

	// 根据优先级调整
	switch plan.Priority {
	case "critical":
		baseTime *= 2
	case "high":
		baseTime *= 1.5
	}

	return baseTime
}

// executePlan 执行修复计划
func (s *RemediationWorkflowSkill) executePlan(ctx context.Context, plan *RemediationPlan) (*skills.SkillOutput, error) {
	executor := NewPlanExecutor(s.skillReg)
	output, err := executor.Execute(ctx, &skills.ToolInput{
		Params: map[string]interface{}{
			"plan": plan,
		},
	})
	if err != nil {
		return nil, err
	}

	return output
}

// createHistoryRecord 创建历史记录
func (s *RemediationWorkflowSkill) createHistoryRecord(
	anomaly monitor.Anomaly,
	analysis *ProblemAnalysisResult,
	plan *RemediationPlan,
	executionResult *skills.SkillOutput,
) *rag.HistoryRecord {
	steps := make([]string, 0)
	for _, step := range plan.Steps {
		steps = append(steps, fmt.Sprintf("%s: %s", step.SkillName, step.Action))
	}

	return &rag.HistoryRecord{
		ProblemType: analysis.ProblemType,
		Description: analysis.Description,
		RootCause:   analysis.RootCause,
		Solution:    fmt.Sprintf("Executed %d steps based on analysis", len(plan.Steps)),
		Steps:       steps,
		Success:     executionResult.Success,
		Timestamp:   time.Now(),
		Metadata: map[string]string{
			"anomaly_severity": anomaly.Severity,
			"anomaly_source":  anomaly.Source,
			"priority":         plan.Priority,
			"estimated_time":  plan.EstimatedTime.String(),
		},
	}
}

// determinePriority 确定优先级
func determinePriority(analysis *ProblemAnalysisResult) string {
	// 根据问题类型和描述确定优先级
	// 这里简化实现，实际可以更复杂
	return "medium"
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

	// 简化实现
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
