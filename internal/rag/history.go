package rag

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// HistoryRecord represents a historical remediation record
type HistoryRecord struct {
	ID          string
	ProblemType string
	Description string
	RootCause   string
	Solution    string
	Steps       []string
	Success     bool
	Timestamp   time.Time
	Metadata    map[string]string
}

// HistoryKnowledgeBase manages historical remediation records
type HistoryKnowledgeBase struct {
	records map[string]*HistoryRecord
	mu      sync.RWMutex
	storage string
}

// NewHistoryKnowledgeBase creates a new history knowledge base
func NewHistoryKnowledgeBase(storagePath string) *HistoryKnowledgeBase {
	return &HistoryKnowledgeBase{
		records: make(map[string]*HistoryRecord),
		storage: storagePath,
	}
}

// AddRecord adds a new history record
func (h *HistoryKnowledgeBase) AddRecord(ctx context.Context, record *HistoryRecord) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if record.ID == "" {
		record.ID = fmt.Sprintf("rec-%d", time.Now().UnixNano())
	}

	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}

	h.records[record.ID] = record
	return nil
}

// SearchSimilarRecords searches for similar historical problems
func (h *HistoryKnowledgeBase) SearchSimilarRecords(ctx context.Context, description string, threshold float64) ([]*HistoryRecord, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var similar []*HistoryRecord
	for _, record := range h.records {
		similarity := h.calculateSimilarity(description, record.Description)
		if similarity >= threshold {
			similar = append(similar, record)
		}
	}

	return similar, nil
}

// calculateSimilarity calculates similarity between two descriptions
func (h *HistoryKnowledgeBase) calculateSimilarity(desc1, desc2 string) float64 {
	// Simple word overlap similarity
	words1 := []string{}
	words2 := []string{}
	wordSet := make(map[string]bool)
	for _, word := range words1 {
		wordSet[word] = true
	}

	matches := 0
	for _, word := range words2 {
		if wordSet[word] {
			matches++
		}
	}

	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	return float64(matches*2) / float64(len(words1)+len(words2))
}

// ListAll returns all records
func (h *HistoryKnowledgeBase) ListAll(ctx context.Context) ([]*HistoryRecord, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	records := make([]*HistoryRecord, 0, len(h.records))
	for _, record := range h.records {
		records = append(records, record)
	}

	return records, nil
}
