package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
func NewHistoryKnowledgeBase(storagePath string) (*HistoryKnowledgeBase, error) {
	h := &HistoryKnowledgeBase{
		records: make(map[string]*HistoryRecord),
		storage: storagePath,
	}
	if err := h.load(); err != nil {
		return nil, err
	}
	return h, nil
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
	return h.save()
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
	sort.Slice(similar, func(i, j int) bool {
		return similar[i].Timestamp.After(similar[j].Timestamp)
	})

	return similar, nil
}

// calculateSimilarity calculates similarity between two descriptions
func (h *HistoryKnowledgeBase) calculateSimilarity(desc1, desc2 string) float64 {
	words1 := tokenize(desc1)
	words2 := tokenize(desc2)
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

func (h *HistoryKnowledgeBase) load() error {
	if h.storage == "" {
		return nil
	}
	data, err := os.ReadFile(h.storage)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}

	var records []*HistoryRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return err
	}
	for _, record := range records {
		h.records[record.ID] = record
	}
	return nil
}

func (h *HistoryKnowledgeBase) save() error {
	if h.storage == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(h.storage), 0o755); err != nil {
		return err
	}

	records := make([]*HistoryRecord, 0, len(h.records))
	for _, record := range h.records {
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Timestamp.After(records[j].Timestamp)
	})

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(h.storage)
	tmp, err := os.CreateTemp(dir, ".history-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, h.storage); err != nil {
		return err
	}
	return os.Chmod(h.storage, 0o600)
}

func tokenize(input string) []string {
	normalized := strings.ToLower(strings.TrimSpace(input))
	normalized = strings.NewReplacer(
		",", " ",
		".", " ",
		":", " ",
		";", " ",
		"/", " ",
		"_", " ",
		"-", " ",
	).Replace(normalized)
	fields := strings.Fields(normalized)
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		if len(field) >= 3 {
			result = append(result, field)
		}
	}
	return result
}
