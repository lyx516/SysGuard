package rag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// KnowledgeBase 知识库，存储和检索运维手册和 SOP
type KnowledgeBase struct {
	documents map[string]*Document
	index     map[string][]string // 关键词到文档ID的映射
}

// Document 文档结构
type Document struct {
	ID       string
	Title    string
	Content  string
	Metadata map[string]string
}

// NewKnowledgeBase 创建新的知识库
func NewKnowledgeBase(ctx context.Context, docPath string) (*KnowledgeBase, error) {
	kb := &KnowledgeBase{
		documents: make(map[string]*Document),
		index:     make(map[string][]string),
	}

	// 加载文档
	if err := kb.loadDocuments(docPath); err != nil {
		return nil, fmt.Errorf("failed to load documents: %w", err)
	}

	// 建立索引
	kb.buildIndex()

	return kb, nil
}

// loadDocuments 从指定路径加载 Markdown 文档
func (kb *KnowledgeBase) loadDocuments(docPath string) error {
	return filepath.Walk(docPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		doc := &Document{
			ID:      filepath.Base(path),
			Title:   extractTitle(string(content)),
			Content: string(content),
			Metadata: map[string]string{
				"path": path,
			},
		}

		kb.documents[doc.ID] = doc
		return nil
	})
}

// buildIndex 构建倒排索引
func (kb *KnowledgeBase) buildIndex() {
	for id, doc := range kb.documents {
		words := extractWords(doc.Content)
		for _, word := range words {
			kb.index[word] = append(kb.index[word], id)
		}
	}
}

// Retrieve 根据查询检索相关文档
func (kb *KnowledgeBase) Retrieve(ctx context.Context, query string) ([]string, error) {
	queryWords := extractWords(query)

	// 计算文档相关性分数
	scores := make(map[string]int)
	for _, word := range queryWords {
		if docIDs, ok := kb.index[word]; ok {
			for _, docID := range docIDs {
				scores[docID]++
			}
		}
	}

	// 返回最相关的文档
	type scoredDoc struct {
		id    string
		score int
	}
	ranked := make([]scoredDoc, 0, len(scores))
	for id, score := range scores {
		ranked = append(ranked, scoredDoc{id: id, score: score})
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})

	results := make([]string, 0, len(ranked))
	for _, item := range ranked {
		id := item.id
		if doc, ok := kb.documents[id]; ok {
			results = append(results, doc.Content)
		}
	}

	return results, nil
}

// AddDocument 添加文档到知识库
func (kb *KnowledgeBase) AddDocument(id, title, content string) {
	doc := &Document{
		ID:      id,
		Title:   title,
		Content: content,
		Metadata: make(map[string]string),
	}

	kb.documents[id] = doc
	kb.buildIndex()
}

// extractTitle 从 Markdown 内容中提取标题
func extractTitle(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[2:])
		}
	}
	return "Untitled"
}

// extractWords 提取内容中的关键词
func extractWords(content string) []string {
	// 简化实现，实际应该使用更复杂的分词算法
	words := strings.Fields(content)
	var result []string
	for _, word := range words {
		// 过滤掉短词和常见停用词
		if len(word) > 2 {
			result = append(result, strings.ToLower(word))
		}
	}
	return result
}
