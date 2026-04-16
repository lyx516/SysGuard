package workflow

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

// LogAnalysisGraph 日志分析工作流图
type LogAnalysisGraph struct {
	chunkSize int // 分块大小（行数）
	keywords  []string
}

// NewLogAnalysisGraph 创建新的日志分析图
func NewLogAnalysisGraph(chunkSize int, keywords []string) *LogAnalysisGraph {
	return &LogAnalysisGraph{
		chunkSize: chunkSize,
		keywords:  keywords,
	}
}

// Analyze 分析日志文件
func (g *LogAnalysisGraph) Analyze(ctx context.Context, filePath string) (*AnalysisResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	result := &AnalysisResult{
		FilePath: filePath,
		Chunks:   make([]*LogChunk, 0),
	}

	// 分块读取日志
	chunk, err := g.readChunk(file)
	chunkIndex := 0
	for err == nil {
		// 过滤关键词
		filtered := g.filterByKeywords(chunk)

		if len(filtered) > 0 {
			result.Chunks = append(result.Chunks, &LogChunk{
				Index: chunkIndex,
				Lines: filtered,
				Count: len(filtered),
			})
			result.Total += len(filtered)
		}

		chunk, err = g.readChunk(file)
		chunkIndex++
	}

	if err != io.EOF {
		return nil, fmt.Errorf("failed to read chunk: %w", err)
	}

	return result, nil
}

// readChunk 读取一个分块的日志
func (g *LogAnalysisGraph) readChunk(file *os.File) ([]string, error) {
	lines := make([]string, 0, g.chunkSize)
	for i := 0; i < g.chunkSize; i++ {
		line, err := g.readLine(file)
		if err != nil {
			if err == io.EOF && len(lines) > 0 {
				return lines, nil
			}
			return lines, err
		}
		lines = append(lines, line)
	}
	return lines, nil
}

// readLine 读取一行
func (g *LogAnalysisGraph) readLine(file *os.File) (string, error) {
	var line []byte
	for {
		buf := make([]byte, 1)
		n, err := file.Read(buf)
		if err != nil {
			return "", err
		}
		if n == 0 {
			break
		}

		if buf[0] == '\n' {
			break
		}

		line = append(line, buf[0])
	}
	return string(line), nil
}

// filterByKeywords 根据关键词过滤日志
func (g *LogAnalysisGraph) filterByKeywords(lines []string) []string {
	if len(g.keywords) == 0 {
		return lines
	}

	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		for _, keyword := range g.keywords {
			if strings.Contains(strings.ToLower(line), strings.ToLower(keyword)) {
				filtered = append(filtered, line)
				break
			}
		}
	}
	return filtered
}

// AnalysisResult 分析结果
type AnalysisResult struct {
	FilePath string
	Chunks   []*LogChunk
	Total    int
}

// LogChunk 日志分块
type LogChunk struct {
	Index int
	Lines []string
	Count int
}

// GetSummary 获取分析摘要
func (ar *AnalysisResult) GetSummary() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("File: %s\n", ar.FilePath))
	sb.WriteString(fmt.Sprintf("Total chunks: %d\n", len(ar.Chunks)))
	sb.WriteString(fmt.Sprintf("Total filtered lines: %d\n", ar.Total))
	return sb.String()
}
