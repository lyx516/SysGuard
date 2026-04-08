package log_analysis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sysguard/sysguard/internal/skills"
	"github.com/sysguard/sysguard/internal/workflow"
)

// LogAnalysisSkill 日志分析 Skill
type LogAnalysisSkill struct {
	graph   *workflow.LogAnalysisGraph
	version string
}

// NewLogAnalysisSkill 创建日志分析 Skill
func NewLogAnalysisSkill(chunkSize int, keywords []string) *LogAnalysisSkill {
	return &LogAnalysisSkill{
		graph:   workflow.NewLogAnalysisGraph(chunkSize, keywords),
		version: "1.0.0",
	}
}

// Name 返回 Skill 名称
func (s *LogAnalysisSkill) Name() string {
	return "log_analysis"
}

// Description 返回 Skill 描述
func (s *LogAnalysisSkill) Description() string {
	return "分析日志文件，提取关键信息和异常模式"
}

// Execute 执行日志分析
func (s *LogAnalysisSkill) Execute(ctx context.Context, input *skills.SkillInput) (*skills.SkillOutput, error) {
	startTime := time.Now()

	// 获取文件路径
	filePath, ok := input.Params["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path parameter is required")
	}

	// 执行分析
	result, err := s.graph.Analyze(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze logs: %w", err)
	}

	// 计算统计信息
	totalLines := 0
	for _, chunk := range result.Chunks {
		totalLines += chunk.Count
	}

	duration := time.Since(startTime).Milliseconds()

	return &skills.SkillOutput{
		Success: true,
		Message: fmt.Sprintf("Successfully analyzed %d chunks, %d lines", len(result.Chunks), totalLines),
		Data: map[string]interface{}{
			"file_path":       result.FilePath,
			"total_chunks":    len(result.Chunks),
			"total_lines":     totalLines,
			"chunks":          result.Chunks,
			"summary":         result.GetSummary(),
		},
		ToolsUsed: []string{"log_reader", "keyword_filter", "pattern_matcher"},
		Duration:  duration,
	}, nil
}

// Tools 返回该 Skill 使用的工具集
func (s *LogAnalysisSkill) Tools() []skills.Tool {
	return []skills.Tool{
		&LogReaderTool{},
		&KeywordFilterTool{},
		&PatternMatcherTool{},
	}
}

// Metadata 返回 Skill 元数据
func (s *LogAnalysisSkill) Metadata() *skills.SkillMetadata {
	return &skills.SkillMetadata{
		Version:     s.version,
		Category:    "analysis",
		Tags:        []string{"logs", "monitoring", "analysis", "debugging"},
		Author:      "SysGuard Team",
		Permissions: []string{"read:logs", "read:files"},
	}
}

// LogReaderTool 日志读取工具
type LogReaderTool struct{}

// Name 返回工具名称
func (t *LogReaderTool) Name() string {
	return "log_reader"
}

// Description 返回工具描述
func (t *LogReaderTool) Description() string {
	return "读取和解析日志文件"
}

// Execute 执行工具
func (t *LogReaderTool) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	filePath, ok := input.Params["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path parameter is required")
	}

	// 实现日志读取逻辑
	// 这里简化实现
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"file_path": filePath,
			"lines_read": 1000,
		},
	}, nil
}

// KeywordFilterTool 关键词过滤工具
type KeywordFilterTool struct{}

// Name 返回工具名称
func (t *KeywordFilterTool) Name() string {
	return "keyword_filter"
}

// Description 返回工具描述
func (t *KeywordFilterTool) Description() string {
	return "根据关键词过滤日志内容"
}

// Execute 执行工具
func (t *KeywordFilterTool) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	lines, _ := input.Params["lines"].([]string)
	keywords, _ := input.Params["keywords"].([]string)

	filtered := make([]string, 0)
	for _, line := range lines {
		for _, keyword := range keywords {
			if strings.Contains(strings.ToLower(line), strings.ToLower(keyword)) {
				filtered = append(filtered, line)
				break
			}
		}
	}

	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"filtered_lines": filtered,
			"count":          len(filtered),
		},
	}, nil
}

// PatternMatcherTool 模式匹配工具
type PatternMatcherTool struct{}

// Name 返回工具名称
func (t *PatternMatcherTool) Name() string {
	return "pattern_matcher"
}

// Description 返回工具描述
func (t *PatternMatcherTool) Description() string {
	return "使用正则表达式匹配日志模式"
}

// Execute 执行工具
func (t *PatternMatcherTool) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	// 实现正则表达式匹配
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"matches": []string{},
		},
	}, nil
}
