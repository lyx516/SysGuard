package rag

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHistoryKnowledgeBasePersistsRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	ctx := context.Background()

	historyKB, err := NewHistoryKnowledgeBase(path)
	if err != nil {
		t.Fatalf("new history kb: %v", err)
	}

	record := &HistoryRecord{
		Description: "nginx service down",
		Solution:    "restart nginx",
		Steps:       []string{"systemctl restart nginx"},
		Success:     true,
		Timestamp:   time.Now().UTC(),
	}
	if err := historyKB.AddRecord(ctx, record); err != nil {
		t.Fatalf("add record: %v", err)
	}

	reloaded, err := NewHistoryKnowledgeBase(path)
	if err != nil {
		t.Fatalf("reload history kb: %v", err)
	}

	matches, err := reloaded.SearchSimilarRecords(ctx, "nginx is down", 0.2)
	if err != nil {
		t.Fatalf("search similar: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("expected at least one similar record")
	}
}

func TestHistoryKnowledgeBaseWritesPrivateHistoryFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	historyKB, err := NewHistoryKnowledgeBase(path)
	if err != nil {
		t.Fatalf("new history kb: %v", err)
	}

	if err := historyKB.AddRecord(context.Background(), &HistoryRecord{
		Description: "disk full",
		Solution:    "clean logs",
		Success:     false,
		Timestamp:   time.Now().UTC(),
	}); err != nil {
		t.Fatalf("add record: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat history file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("history file permission = %o, want 600", got)
	}
}
