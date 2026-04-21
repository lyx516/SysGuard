package rag

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRetrieveEvidenceReturnsRankedChunksWithCitations(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "nginx.md"), []byte("# Nginx SOP\n\nWhen nginx is down, inspect logs first.\n\n```bash\njournalctl -u nginx -n 100 --no-pager\n```\n\nThen restart nginx only after approval.\n"), 0o644); err != nil {
		t.Fatalf("write nginx sop: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "disk.md"), []byte("# Disk SOP\n\nWhen disk usage is high, identify large logs and rotate them.\n"), 0o644); err != nil {
		t.Fatalf("write disk sop: %v", err)
	}

	kb, err := NewKnowledgeBase(context.Background(), dir)
	if err != nil {
		t.Fatalf("new knowledge base: %v", err)
	}
	evidence, err := kb.RetrieveEvidence(context.Background(), "nginx service down restart logs", 2)
	if err != nil {
		t.Fatalf("retrieve evidence: %v", err)
	}
	if len(evidence) == 0 {
		t.Fatal("expected evidence")
	}
	if evidence[0].Citation.DocumentID != "nginx.md" {
		t.Fatalf("top citation = %#v, want nginx.md", evidence[0].Citation)
	}
	if evidence[0].Citation.Path == "" || evidence[0].Citation.ChunkID == "" {
		t.Fatalf("citation missing path/chunk id: %#v", evidence[0].Citation)
	}
	if !strings.Contains(evidence[0].Content, "nginx") {
		t.Fatalf("top evidence not nginx-related: %q", evidence[0].Content)
	}
}
