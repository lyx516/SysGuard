package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// HistoryRecord 历史处理记录
type HistoryRecord struct {
	ID          string            `json:"id"`
	ProblemType string            `json:"problem_type"`
	Description string            `json:"description"`
	RootCause   string            `json:"root_cause"`
	Solution    string            `json:"solution"`
	Steps       []string          `json:"steps"`
	Success     bool              `json:"success"`
	Timestamp   time.Time         `json:"timestamp"`
	Metadata    map[string]string `json:"metadata"`
}

// HistoryKnowledgeBase 历史文档知识库
type HistoryKnowledgeBase struct {
	records        map[string]*HistoryRecord
	recordsByType  map[string][]*HistoryRecord
	mu             sync.RWMutex
	historyPath    string
	maxRecords     int
}

// NewHistoryKnowledgeBase 创建新的历史知识库
func NewHistoryKnowledgeBase(historyPath string, maxRecords int) (*HistoryKnowledgeBase, error) {
	hkb := &HistoryKnowledgeBase{
		records:       make(map[string]*HistoryRecord),
		recordsByType: make(map[string][]*HistoryRecord),
		historyPath:   historyPath,
		maxRecords:    maxRecords,
	}

	// 创建历史目录
	if err := os.MkdirAll(historyPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create history directory: %w", err)
	}

	// 加载历史记录
	if err := hkb.loadRecords(); err != nil {
		return nil, fmt.Errorf("failed to load history records: %w", err)
	}

	return hkb, nil
}

// loadRecords 加载历史记录
func (hkb *HistoryKnowledgeBase) loadRecords() error {
	files, err := filepath.Glob(filepath.Join(hkb.historyPath, "*.json"))
	if err != nil {
		return err
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var record HistoryRecord
		if err := json.Unmarshal(content, &record); err != nil {
			continue
		}

		hkb.records[record.ID] = &record
		hkb.recordsByType[record.ProblemType] = append(
			hkb.recordsByType[record.ProblemType],
			&record,
		)
	}

	return nil
}

// AddRecord 添加历史记录
func (hkb *HistoryKnowledgeBase) AddRecord(record *HistoryRecord) error {
	hkb.mu.Lock()
	defer hkb.mu.Unlock()

	// 设置 ID 和时间戳
	if record.ID == "" {
		record.ID = fmt.Sprintf("HIST-%d", time.Now().UnixNano())
	}
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}

	// 检查记录数量限制
	if len(hkb.records) >= hkb.maxRecords {
		hkb.evictOldestRecord()
	}

	// 保存到文件
	filePath := filepath.Join(hkb.historyPath, fmt.Sprintf("%s.json", record.ID))
	content, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return err
	}

	// 添加到内存
	hkb.records[record.ID] = record
	hkb.recordsByType[record.ProblemType] = append(
		hkb.recordsByType[record.ProblemType],
		record,
	)

	return nil
}

// GetRecord 获取指定记录
func (hkb *HistoryKnowledgeBase) GetRecord(id string) (*HistoryRecord, bool) {
	hkb.mu.RLock()
	defer hkb.mu.RUnlock()

	record, ok := hkb.records[id]
	return record, ok
}

// SearchSimilarRecords 搜索相似的历史记录
func (hkb *HistoryKnowledgeBase) SearchSimilarRecords(
	ctx context.Context,
	problemType string,
	description string,
	limit int,
) []*HistoryRecord {
	hkb.mu.RLock()
	defer hkb.mu.RUnlock()

	// 按问题类型筛选
	records, ok := hkb.recordsByType[problemType]
	if !ok {
		return []*HistoryRecord{}
	}

	// 计算相似度并排序
	type scoredRecord struct {
		record *HistoryRecord
		score  float64
	}

	scored := make([]scoredRecord, 0)
	problemWords := extractKeywords(description)

	for _, record := range records {
		score := hkb.calculateSimilarity(problemWords, record)
		scored = append(scored, scoredRecord{
			record: record,
			score:  score,
		})
	}

	// 按相似度排序
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[i].score < scored[j].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// 返回前 N 个
	if limit > len(scored) {
		limit = len(scored)
	}

	result := make([]*HistoryRecord, limit)
	for i := 0; i < limit; i++ {
		result[i] = scored[i].record
	}

	return result
}

// calculateSimilarity 计算相似度
func (hkb *HistoryKnowledgeBase) calculateSimilarity(
	problemWords []string,
	record *HistoryRecord,
) float64 {
	recordWords := extractKeywords(record.Description + " " + record.RootCause)

	// 计算 Jaccard 相似度
	intersection := 0
	union := make(map[string]bool)

	for _, word := range problemWords {
		union[word] = true
	}

	for _, word := range recordWords {
		union[word] = true
		for _, pw := range problemWords {
			if word == pw {
				intersection++
				break
			}
		}
	}

	if len(union) == 0 {
		return 0
	}

	return float64(intersection) / float64(len(union))
}

// extractKeywords 提取关键词
func extractKeywords(text string) []string {
	words := strings.Fields(strings.ToLower(text))
	keywords := make(map[string]bool)

	for _, word := range words {
		// 过滤短词和停用词
		if len(word) <= 2 {
			continue
		}

		// 简单停用词过滤
		stopWords := map[string]bool{
			"the":   true,
			"is":     true,
			"are":    true,
			"was":    true,
			"were":   true,
			"be":     true,
			"been":   true,
			"being":  true,
			"have":   true,
			"has":    true,
			"had":    true,
			"do":     true,
			"does":   true,
			"did":    true,
			"will":   true,
			"would":  true,
			"should": true,
			"can":    true,
			"could":  true,
			"may":    true,
			"might":  true,
			"的":     true,
			"了":     true,
			"在":     true,
			"是":     true,
		}

		if !stopWords[word] {
			keywords[word] = true
		}
	}

	result := make([]string, 0, len(keywords))
	for word := range keywords {
		result = append(result, word)
	}

	return result
}

// evictOldestRecord 淘汰最旧的记录
func (hkb *HistoryKnowledgeBase) evictOldestRecord() {
	if len(hkb.records) == 0 {
		return
	}

	var oldestID string
	var oldestTime time.Time

	for id, record := range hkb.records {
		if oldestID == "" || record.Timestamp.Before(oldestTime) {
			oldestID = id
			oldestTime = record.Timestamp
		}
	}

	if oldestID != "" {
		hkb.deleteRecord(oldestID)
	}
}

// deleteRecord 删除记录
func (hkb *HistoryKnowledgeBase) deleteRecord(id string) {
	record, ok := hkb.records[id]
	if !ok {
		return
	}

	// 删除文件
	filePath := filepath.Join(hkb.historyPath, fmt.Sprintf("%s.json", id))
	os.Remove(filePath)

	// 从内存删除
	delete(hkb.records, id)

	// 从类型索引删除
	records := hkb.recordsByType[record.ProblemType]
	for i, r := range records {
		if r.ID == id {
			hkb.recordsByType[record.ProblemType] = append(
				records[:i],
				records[i+1:]...,
			)
			break
		}
	}
}

// GetRecordCount 获取记录数量
func (hkb *HistoryKnowledgeBase) GetRecordCount() int {
	hkb.mu.RLock()
	defer hkb.mu.RUnlock()
	return len(hkb.records)
}

// GetRecordsByType 按类型获取记录
func (hkb *HistoryKnowledgeBase) GetRecordsByType(problemType string) []*HistoryRecord {
	hkb.mu.RLock()
	defer hkb.mu.RUnlock()

	records, ok := hkb.recordsByType[problemType]
	if !ok {
		return []*HistoryRecord{}
	}

	result := make([]*HistoryRecord, len(records))
	copy(result, records)
	return result
}

// ListAllRecords 列出所有记录
func (hkb *HistoryKnowledgeBase) ListAllRecords() []*HistoryRecord {
	hkb.mu.RLock()
	defer hkb.mu.RUnlock()

	result := make([]*HistoryRecord, 0, len(hkb.records))
	for _, record := range hkb.records {
		result = append(result, record)
	}

	return result
}

// UpdateRecord 更新记录
func (hkb *HistoryKnowledgeBase) UpdateRecord(record *HistoryRecord) error {
	hkb.mu.Lock()
	defer hkb.mu.Unlock()

	if _, ok := hkb.records[record.ID]; !ok {
		return fmt.Errorf("record not found: %s", record.ID)
	}

	// 保存到文件
	filePath := filepath.Join(hkb.historyPath, fmt.Sprintf("%s.json", record.ID))
	content, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return err
	}

	// 更新内存
	hkb.records[record.ID] = record

	return nil
}

// GetHistoryPath 获取历史记录路径
func (hkb *HistoryKnowledgeBase) GetHistoryPath() string {
	return hkb.historyPath
}
