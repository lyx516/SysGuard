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
	chunks    []*EvidenceChunk
}

// Document 文档结构
type Document struct {
	ID       string
	Title    string
	Content  string
	Runbook  RunbookMetadata
	Metadata map[string]string
}

type RunbookMetadata struct {
	ID                string   `json:"id,omitempty"`
	RiskLevel         string   `json:"risk_level,omitempty"`
	RequiredApproval  bool     `json:"required_approval,omitempty"`
	Signals           []string `json:"signals,omitempty"`
	DiagnosisSteps    []string `json:"diagnosis_steps,omitempty"`
	ExecutionSteps    []string `json:"execution_steps,omitempty"`
	VerificationSteps []string `json:"verification_steps,omitempty"`
	RollbackSteps     []string `json:"rollback_steps,omitempty"`
}

type Citation struct {
	DocumentID string `json:"document_id"`
	Title      string `json:"title"`
	Path       string `json:"path"`
	ChunkID    string `json:"chunk_id"`
}

type EvidenceChunk struct {
	ID        string             `json:"id"`
	Content   string             `json:"content"`
	Score     float64            `json:"score"`
	Citation  Citation           `json:"citation"`
	Runbook   RunbookMetadata    `json:"runbook,omitempty"`
	Embedding map[string]float64 `json:"-"`
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

		body, runbook := splitFrontMatter(string(content))
		doc := &Document{
			ID:      filepath.Base(path),
			Title:   extractTitle(body),
			Content: body,
			Runbook: runbook,
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
	kb.index = make(map[string][]string)
	kb.chunks = nil
	for id, doc := range kb.documents {
		words := extractWords(doc.Content)
		for _, word := range words {
			kb.index[word] = append(kb.index[word], id)
		}
		for chunkIndex, content := range chunkDocument(doc.Content, 700) {
			chunkID := fmt.Sprintf("%s#chunk-%d", id, chunkIndex+1)
			kb.chunks = append(kb.chunks, &EvidenceChunk{
				ID:      chunkID,
				Content: content,
				Citation: Citation{
					DocumentID: id,
					Title:      doc.Title,
					Path:       doc.Metadata["path"],
					ChunkID:    chunkID,
				},
				Runbook:   doc.Runbook,
				Embedding: sparseEmbedding(content),
			})
		}
	}
}

func splitFrontMatter(content string) (string, RunbookMetadata) {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "---\n") {
		return content, RunbookMetadata{}
	}
	rest := strings.TrimPrefix(trimmed, "---\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return content, RunbookMetadata{}
	}
	metaBlock := rest[:end]
	body := strings.TrimSpace(rest[end+len("\n---"):])
	return body, parseRunbookFrontMatter(metaBlock)
}

func parseRunbookFrontMatter(block string) RunbookMetadata {
	var meta RunbookMetadata
	var currentList string
	for _, raw := range strings.Split(block, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "- ") {
			item := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			switch currentList {
			case "signals":
				meta.Signals = append(meta.Signals, item)
			case "diagnosis_steps":
				meta.DiagnosisSteps = append(meta.DiagnosisSteps, item)
			case "execution_steps":
				meta.ExecutionSteps = append(meta.ExecutionSteps, item)
			case "verification_steps":
				meta.VerificationSteps = append(meta.VerificationSteps, item)
			case "rollback_steps":
				meta.RollbackSteps = append(meta.RollbackSteps, item)
			}
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		currentList = ""
		switch key {
		case "id":
			meta.ID = value
		case "risk_level":
			meta.RiskLevel = value
		case "required_approval":
			meta.RequiredApproval = value == "true"
		case "signals", "diagnosis_steps", "execution_steps", "verification_steps", "rollback_steps":
			currentList = key
		}
	}
	return meta
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

func (kb *KnowledgeBase) RetrieveEvidence(ctx context.Context, query string, limit int) ([]EvidenceChunk, error) {
	if limit <= 0 {
		limit = 5
	}
	queryEmbedding := sparseEmbedding(query)
	scored := make([]EvidenceChunk, 0, len(kb.chunks))
	for _, chunk := range kb.chunks {
		copyChunk := *chunk
		copyChunk.Score = cosineSimilarity(queryEmbedding, chunk.Embedding)
		if copyChunk.Score > 0 {
			copyChunk.Embedding = nil
			scored = append(scored, copyChunk)
		}
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].Score == scored[j].Score {
			return scored[i].Citation.DocumentID < scored[j].Citation.DocumentID
		}
		return scored[i].Score > scored[j].Score
	})
	if len(scored) > limit {
		scored = scored[:limit]
	}
	return scored, nil
}

// AddDocument 添加文档到知识库
func (kb *KnowledgeBase) AddDocument(id, title, content string) {
	doc := &Document{
		ID:       id,
		Title:    title,
		Content:  content,
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

func chunkDocument(content string, maxChars int) []string {
	paragraphs := strings.Split(content, "\n\n")
	chunks := make([]string, 0)
	var current strings.Builder
	for _, paragraph := range paragraphs {
		trimmed := strings.TrimSpace(paragraph)
		if trimmed == "" {
			continue
		}
		if current.Len() > 0 && current.Len()+len(trimmed)+2 > maxChars {
			chunks = append(chunks, current.String())
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(trimmed)
	}
	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}
	return chunks
}

func sparseEmbedding(content string) map[string]float64 {
	embedding := make(map[string]float64)
	for _, word := range extractWords(content) {
		normalized := strings.Trim(word, ".,:;()[]{}#`\"'")
		if len(normalized) > 2 {
			embedding[normalized]++
		}
	}
	return embedding
}

func cosineSimilarity(a, b map[string]float64) float64 {
	var dot, normA, normB float64
	for key, av := range a {
		normA += av * av
		if bv, ok := b[key]; ok {
			dot += av * bv
		}
	}
	for _, bv := range b {
		normB += bv * bv
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (sqrt(normA) * sqrt(normB))
}

func sqrt(v float64) float64 {
	if v == 0 {
		return 0
	}
	x := v
	for i := 0; i < 12; i++ {
		x = 0.5 * (x + v/x)
	}
	return x
}
