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

func TestRetrieveEvidenceIncludesRunbookMetadataFromFrontMatter(t *testing.T) {
	dir := t.TempDir()
	content := `---
id: service-down
risk_level: privileged
required_approval: true
signals:
  - service down
diagnosis_steps:
  - check service status
verification_steps:
  - run health check
rollback_steps:
  - restore previous configuration
steps:
  - id: diagnose-status
    title: Check service status
    type: diagnosis
    intent: Confirm whether the service manager reports the service as failed.
    tool: service-management
    action: status
    preconditions:
      - service name is known
    risks:
      - status output may include sensitive environment values
    verification:
      - status command completed successfully
    rollback:
      - no rollback required for read-only diagnosis
  - id: restart-service
    title: Restart service after approval
    type: execution
    intent: Restore service availability with a controlled restart.
    tool: service-management
    action: restart
    requires_approval: true
    preconditions:
      - diagnosis confirms the service is down
      - approval has been granted
    risks:
      - active connections may be interrupted
    verification:
      - run health check
      - inspect recent service logs
    rollback:
      - restore previous configuration
      - restart service again after rollback
---
# Service Down

When a service is down, inspect status and logs before privileged action.
`
	if err := os.WriteFile(filepath.Join(dir, "service-down.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write sop: %v", err)
	}

	kb, err := NewKnowledgeBase(context.Background(), dir)
	if err != nil {
		t.Fatalf("new knowledge base: %v", err)
	}
	evidence, err := kb.RetrieveEvidence(context.Background(), "service down logs status", 1)
	if err != nil {
		t.Fatalf("retrieve evidence: %v", err)
	}
	if len(evidence) != 1 {
		t.Fatalf("evidence count = %d, want 1", len(evidence))
	}
	meta := evidence[0].Runbook
	if meta.ID != "service-down" || meta.RiskLevel != "privileged" || !meta.RequiredApproval {
		t.Fatalf("unexpected runbook metadata: %#v", meta)
	}
	if len(meta.Signals) != 1 || len(meta.DiagnosisSteps) != 1 || len(meta.VerificationSteps) != 1 || len(meta.RollbackSteps) != 1 {
		t.Fatalf("front matter lists were not parsed: %#v", meta)
	}
	if len(meta.Steps) != 2 {
		t.Fatalf("structured steps count = %d, want 2: %#v", len(meta.Steps), meta.Steps)
	}
	step := meta.Steps[1]
	if step.ID != "restart-service" || step.Type != "execution" || !step.RequiresApproval {
		t.Fatalf("unexpected structured step metadata: %#v", step)
	}
	if len(step.Preconditions) != 2 || len(step.Risks) != 1 || len(step.Verification) != 2 || len(step.Rollback) != 2 {
		t.Fatalf("structured step guardrails were not parsed: %#v", step)
	}
}
