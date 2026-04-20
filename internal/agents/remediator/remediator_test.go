package remediator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sysguard/sysguard/internal/config"
	"github.com/sysguard/sysguard/internal/monitor"
	"github.com/sysguard/sysguard/internal/observability"
	"github.com/sysguard/sysguard/internal/rag"
	"github.com/sysguard/sysguard/internal/security"
)

func TestRemediateDryRunRecordsPlanWithoutExecutingCommands(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	docsPath := filepath.Join(dir, "docs")
	if err := os.MkdirAll(docsPath, 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(docsPath, "restart.md"), []byte("# Restart Demo\n\nmissing service failed\n\n```bash\nsysguard-command-that-should-not-run\n```\n"), 0o644); err != nil {
		t.Fatalf("write sop: %v", err)
	}

	cfg := config.Default()
	cfg.Agents.Remediator.DryRun = true
	cfg.Observability.EnableTracing = false
	cfg.Storage.HistoryPath = filepath.Join(dir, "history.json")

	kb, err := rag.NewKnowledgeBase(context.Background(), docsPath)
	if err != nil {
		t.Fatalf("new knowledge base: %v", err)
	}
	historyKB, err := rag.NewHistoryKnowledgeBase(cfg.Storage.HistoryPath)
	if err != nil {
		t.Fatalf("new history: %v", err)
	}
	obs, err := observability.NewGlobalCallback(false, "")
	if err != nil {
		t.Fatalf("new callback: %v", err)
	}
	remediator := NewRemediator(cfg, kb, historyKB, security.NewCommandInterceptor(nil), obs)

	err = remediator.Remediate(context.Background(), monitor.Anomaly{
		Severity:    "warning",
		Description: "missing service failed",
		Source:      "monitor",
		Metadata:    map[string]string{},
	})
	if err != nil {
		t.Fatalf("dry-run remediation should not execute missing command: %v", err)
	}

	records, err := historyKB.ListAll(context.Background())
	if err != nil {
		t.Fatalf("list history: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("history records = %d, want 1", len(records))
	}
	if records[0].Success {
		t.Fatal("dry-run remediation must not be recorded as a successful repair")
	}
	if records[0].Metadata["dry_run"] != "true" {
		t.Fatalf("dry-run metadata = %#v, want dry_run=true", records[0].Metadata)
	}
}

func TestRemediateVerificationFailureRecordsFailedRepair(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	docsPath := filepath.Join(dir, "docs")
	if err := os.MkdirAll(docsPath, 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(docsPath, "verify.md"), []byte("# Verify Demo\n\nservice failed\n\n```bash\ntrue\n```\n"), 0o644); err != nil {
		t.Fatalf("write sop: %v", err)
	}

	cfg := config.Default()
	cfg.Agents.Remediator.DryRun = false
	cfg.Agents.Remediator.VerifyAfterRemediation = true
	cfg.Observability.EnableTracing = false
	cfg.Storage.HistoryPath = filepath.Join(dir, "history.json")

	kb, err := rag.NewKnowledgeBase(context.Background(), docsPath)
	if err != nil {
		t.Fatalf("new knowledge base: %v", err)
	}
	historyKB, err := rag.NewHistoryKnowledgeBase(cfg.Storage.HistoryPath)
	if err != nil {
		t.Fatalf("new history: %v", err)
	}
	obs, err := observability.NewGlobalCallback(false, "")
	if err != nil {
		t.Fatalf("new callback: %v", err)
	}
	remediator := NewRemediator(cfg, kb, historyKB, security.NewCommandInterceptor(nil), obs)
	remediator.SetVerifier(func(context.Context, monitor.Anomaly) error {
		return errors.New("service still unhealthy")
	})

	err = remediator.Remediate(context.Background(), monitor.Anomaly{
		Severity:    "warning",
		Description: "service failed",
		Source:      "monitor",
		Metadata:    map[string]string{},
	})
	if err == nil || !strings.Contains(err.Error(), "verification failed") {
		t.Fatalf("expected verification failure, got %v", err)
	}

	records, err := historyKB.ListAll(context.Background())
	if err != nil {
		t.Fatalf("list history: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("history records = %d, want 1", len(records))
	}
	if records[0].Success {
		t.Fatal("verification failure must not be recorded as a successful repair")
	}
	if records[0].Metadata["verify_error"] != "service still unhealthy" {
		t.Fatalf("verification metadata = %#v, want verify_error", records[0].Metadata)
	}
}
