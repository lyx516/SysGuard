package remediation_workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sysguard/sysguard/internal/monitor"
	"github.com/sysguard/sysguard/internal/skills"
)

// DocumentGenerator 文档生成工具
type DocumentGenerator struct {
	historyPath string
}

// NewDocumentGenerator 创建新的文档生成器
func NewDocumentGenerator(historyPath string) *DocumentGenerator {
	return &DocumentGenerator{
		historyPath: historyPath,
	}
}

// Name 返回工具名称
func (t *DocumentGenerator) Name() string {
	return "document_generator"
}

// Description 返回工具描述
func (t *DocumentGenerator) Description() string {
	return "生成处理文档，记录问题描述、分析过程、执行步骤和结果"
}

// Execute 执行工具
func (t *DocumentGenerator) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	anomaly, ok := input.Params["anomaly"].(monitor.Anomaly)
	if !ok {
		return nil, fmt.Errorf("anomaly parameter is required")
	}

	analysis, ok := input.Params["analysis"].(*ProblemAnalysisResult)
	if !ok {
		return nil, fmt.Errorf("analysis parameter is required")
	}

	plan, ok := input.Params["plan"].(*RemediationPlan)
	if !ok {
		return nil, fmt.Errorf("plan parameter is required")
	}

	executionResult, ok := input.Params["execution_result"].(*skills.SkillOutput)
	if !ok {
		return nil, fmt.Errorf("execution_result parameter is required")
	}

	// 生成文档
	docContent := t.generateDocument(anomaly, analysis, plan, executionResult)

	// 保存文档
	docPath, err := t.saveDocument(docContent, anomaly)
	if err != nil {
		return nil, err
	}

	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"document_path": docPath,
			"content":       docContent,
		},
	}, nil
}

// generateDocument 生成文档内容
func (t *DocumentGenerator) generateDocument(
	anomaly monitor.Anomaly,
	analysis *ProblemAnalysisResult,
	plan *RemediationPlan,
	executionResult *skills.SkillOutput,
) string {
	var sb strings.Builder

	// 标题
	sb.WriteString(fmt.Sprintf("# 系统异常处理记录\n\n"))
	sb.WriteString(fmt.Sprintf("**生成时间**: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// 1. 异常信息
	sb.WriteString("## 1. 异常信息\n\n")
	sb.WriteString(fmt.Sprintf("- **时间**: %s\n", anomaly.Timestamp.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("- **严重程度**: %s\n", anomaly.Severity))
	sb.WriteString(fmt.Sprintf("- **来源**: %s\n", anomaly.Source))
	sb.WriteString(fmt.Sprintf("- **描述**: %s\n\n", anomaly.Description))

	// 2. 问题分析
	sb.WriteString("## 2. 问题分析\n\n")
	sb.WriteString(fmt.Sprintf("### 问题类型\n%s\n\n", analysis.ProblemType))
	sb.WriteString(fmt.Sprintf("### 问题描述\n%s\n\n", analysis.Description))
	sb.WriteString(fmt.Sprintf("### 根本原因\n%s\n\n", analysis.RootCause))

	// 推荐建议
	if len(analysis.Recommendations) > 0 {
		sb.WriteString("### 建议措施\n\n")
		for i, rec := range analysis.Recommendations {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, rec))
		}
		sb.WriteString("\n")
	}

	// 3. 修复计划
	sb.WriteString("## 3. 修复计划\n\n")
	sb.WriteString(fmt.Sprintf("### 计划优先级\n%s\n\n", plan.Priority))
	sb.WriteString(fmt.Sprintf("### 预估时间\n%s\n\n", plan.EstimatedTime.String()))

	// 历史参考
	if len(plan.HistoricalRef) > 0 {
		sb.WriteString("### 历史参考\n\n")
		sb.WriteString(fmt.Sprintf("找到 %d 条相似的历史记录：\n\n", len(plan.HistoricalRef)))
		for i, ref := range plan.HistoricalRef {
			sb.WriteString(fmt.Sprintf("**记录 %d** (ID: %s)\n", i+1, ref.ID))
			sb.WriteString(fmt.Sprintf("- 时间: %s\n", ref.Timestamp.Format("2006-01-02 15:04:05")))
			sb.WriteString(fmt.Sprintf("- 成功: %v\n", ref.Success))
			if ref.Success && len(ref.Steps) > 0 {
				sb.WriteString("- 执行步骤:\n")
				for _, step := range ref.Steps {
					sb.WriteString(fmt.Sprintf("  - %s\n", step))
				}
			}
			sb.WriteString("\n")
		}
	}

	// 执行步骤
	sb.WriteString("### 执行步骤\n\n")
	for _, step := range plan.Steps {
		sb.WriteString(fmt.Sprintf("#### 步骤 %d: %s\n", step.StepNumber, step.SkillName))
		sb.WriteString(fmt.Sprintf("- **操作**: %s\n", step.Action))
		sb.WriteString(fmt.Sprintf("- **预期结果**: %s\n", step.ExpectedResult))

		if len(step.Parameters) > 0 {
			sb.WriteString("- **参数**:\n")
			for key, value := range step.Parameters {
				sb.WriteString(fmt.Sprintf("  - %s: %v\n", key, value))
			}
		}
		sb.WriteString("\n")
	}

	// 4. 执行结果
	sb.WriteString("## 4. 执行结果\n\n")
	sb.WriteString(fmt.Sprintf("- **总体状态**: %v\n", executionResult.Success))
	sb.WriteString(fmt.Sprintf("- **消息**: %s\n", executionResult.Message))

	if len(executionResult.Data) > 0 {
		sb.WriteString("\n### 详细数据\n\n")
		for key, value := range executionResult.Data {
			sb.WriteString(fmt.Sprintf("- **%s**: %v\n", key, value))
		}
		sb.WriteString("\n")
	}

	// 错误信息
	if len(executionResult.Errors) > 0 {
		sb.WriteString("### 错误信息\n\n")
		for i, err := range executionResult.Errors {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, err))
		}
		sb.WriteString("\n")
	}

	// 使用的工具
	if len(executionResult.ToolsUsed) > 0 {
		sb.WriteString("### 使用的工具\n\n")
		for _, tool := range executionResult.ToolsUsed {
			sb.WriteString(fmt.Sprintf("- %s\n", tool))
		}
		sb.WriteString("\n")
	}

	// 5. 总结
	sb.WriteString("## 5. 总结\n\n")
	if executionResult.Success {
		sb.WriteString("✅ **修复成功**\n\n")
		sb.WriteString("本次异常已成功修复。系统已恢复正常运行。\n\n")
	} else {
		sb.WriteString("❌ **修复失败**\n\n")
		sb.WriteString("本次异常修复未完全成功。需要进一步调查和处理。\n\n")
	}

	// 处理时长
	if executionResult.Duration > 0 {
		duration := time.Duration(executionResult.Duration) * time.Millisecond
		sb.WriteString(fmt.Sprintf("**处理时长**: %s\n", duration.String()))
	}

	// 签名
	sb.WriteString("\n---\n")
	sb.WriteString("*本文档由 SysGuard 自动生成*")

	return sb.String()
}

// saveDocument 保存文档
func (t *DocumentGenerator) saveDocument(content string, anomaly monitor.Anomaly) (string, error) {
	// 创建文档目录
	docsPath := filepath.Join(t.historyPath, "documents")
	if err := os.MkdirAll(docsPath, 0755); err != nil {
		return "", err
	}

	// 生成文件名
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("remediation-%s-%s.md", timestamp, anomaly.Severity)
	filePath := filepath.Join(docsPath, filename)

	// 写入文件
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return "", err
	}

	return filePath, nil
}

// GenerateSummary 生成摘要
func (t *DocumentGenerator) GenerateSummary(anomaly monitor.Anomaly, analysis *ProblemAnalysisResult) string {
	return fmt.Sprintf(
		"异常类型: %s | 严重程度: %s | 根本原因: %s",
		analysis.ProblemType,
		anomaly.Severity,
		analysis.RootCause,
	)
}

// FormatTime 格式化时间
func (t *DocumentGenerator) FormatTime(timeVal time.Time) string {
	return timeVal.Format("2006-01-02 15:04:05")
}

// FormatDuration 格式化时长
func (t *DocumentGenerator) FormatDuration(dur time.Duration) string {
	return dur.String()
}

// CreateDocumentIndex 创建文档索引
func (t *DocumentGenerator) CreateDocumentIndex(docsPath string) error {
	files, err := filepath.Glob(filepath.Join(docsPath, "*.md"))
	if err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("# 异常处理文档索引\n\n")
	sb.WriteString(fmt.Sprintf("**生成时间**: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("**文档数量**: %d\n\n", len(files)))

	sb.WriteString("| 时间 | 严重程度 | 文件名 |\n")
	sb.WriteString("|------|----------|--------|\n")

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		filename := filepath.Base(file)
		sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n",
			info.ModTime().Format("2006-01-02 15:04:05"),
			t.extractSeverity(filename),
			filename,
		))
	}

	indexPath := filepath.Join(docsPath, "index.md")
	return os.WriteFile(indexPath, []byte(sb.String()), 0644)
}

// extractSeverity 从文件名提取严重程度
func (t *DocumentGenerator) extractSeverity(filename string) string {
	severities := []string{"critical", "error", "warning", "info"}
	for _, sev := range severities {
		if contains(filename, sev) {
			return sev
		}
	}
	return "unknown"
}

// contains 检查字符串包含
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
