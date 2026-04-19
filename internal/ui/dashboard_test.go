package ui

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sysguard/sysguard/internal/config"
	"github.com/sysguard/sysguard/internal/observability"
	"github.com/sysguard/sysguard/internal/rag"
)

func TestCollectorBuildsOperationsDashboardSnapshot(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tracePath := filepath.Join(dir, "trace.log")
	logPath := filepath.Join(dir, "sysguard.log")
	historyPath := filepath.Join(dir, "history.json")

	obs, err := observability.NewGlobalCallback(true, tracePath)
	if err != nil {
		t.Fatalf("NewGlobalCallback() error = %v", err)
	}
	inspectorID := obs.OnCallbackStarted("Inspector.healthCheck")
	obs.OnCallbackCompleted(inspectorID, map[string]interface{}{"score": 96.5})
	remediatorID := obs.OnCallbackStarted("Remediator.remediate")
	obs.OnCallbackError(remediatorID, assertErr("restart failed"))

	if err := os.WriteFile(logPath, []byte(
		"2026/04/19 09:00:00 Inspector: Health check completed - score=96.50 healthy=true\n"+
			"2026/04/19 09:01:00 Remediator: Command succeeded cmd=\"systemctl restart nginx\" exit=0 duration=1s\n"+
			"2026/04/19 09:02:00 ERROR failed to notify anomaly\n",
	), 0o644); err != nil {
		t.Fatalf("WriteFile(log) error = %v", err)
	}

	historyKB, err := rag.NewHistoryKnowledgeBase(historyPath)
	if err != nil {
		t.Fatalf("NewHistoryKnowledgeBase() error = %v", err)
	}
	if err := historyKB.AddRecord(context.Background(), &rag.HistoryRecord{
		ID:          "rec-1",
		ProblemType: "monitor",
		Description: "nginx down",
		Solution:    "Recover service nginx",
		Steps:       []string{"journalctl -u nginx -n 100 --no-pager", "systemctl restart nginx"},
		Success:     true,
		Timestamp:   time.Date(2026, 4, 19, 9, 3, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("AddRecord() error = %v", err)
	}

	cfg := config.Default()
	cfg.Storage.LogPath = logPath
	cfg.Observability.TraceLogPath = tracePath
	cfg.Storage.HistoryPath = historyPath
	cfg.Services = []string{"nginx", "redis"}

	collector := NewCollector(cfg, nil, obs, historyKB)
	snapshot, err := collector.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}

	if snapshot.GeneratedAt.IsZero() {
		t.Fatal("snapshot GeneratedAt should be set")
	}
	if snapshot.System.ManagedServices != 2 {
		t.Fatalf("ManagedServices = %d, want 2", snapshot.System.ManagedServices)
	}
	if len(snapshot.Agents) != 3 {
		t.Fatalf("len(Agents) = %d, want 3", len(snapshot.Agents))
	}
	if snapshot.AgentByName("Inspector").Status != "healthy" {
		t.Fatalf("Inspector status = %q, want healthy", snapshot.AgentByName("Inspector").Status)
	}
	if snapshot.AgentByName("Remediator").Status != "error" {
		t.Fatalf("Remediator status = %q, want error", snapshot.AgentByName("Remediator").Status)
	}
	if snapshot.Tools.Total != 2 || snapshot.Tools.Errors != 1 {
		t.Fatalf("tool summary = total %d errors %d, want total 2 errors 1", snapshot.Tools.Total, snapshot.Tools.Errors)
	}
	if snapshot.Logs.Total != 3 || snapshot.Logs.Errors != 1 {
		t.Fatalf("log summary = total %d errors %d, want total 3 errors 1", snapshot.Logs.Total, snapshot.Logs.Errors)
	}
	if snapshot.History.Total != 1 || snapshot.History.Success != 1 {
		t.Fatalf("history summary = total %d success %d, want total 1 success 1", snapshot.History.Total, snapshot.History.Success)
	}
	if len(snapshot.Timeline) == 0 {
		t.Fatal("timeline should include callback, log, or history events")
	}
}

func TestCollectorBuildsToolSummaryFromTraceLog(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tracePath := filepath.Join(dir, "trace.log")
	if err := os.WriteFile(tracePath, []byte(
		`{"timestamp":"2026-04-19T09:00:00Z","type":"callback_started","payload":{"id":"Inspector.healthCheck-1","name":"Inspector.healthCheck"}}`+"\n"+
			`{"timestamp":"2026-04-19T09:00:01Z","type":"callback_completed","payload":{"id":"Inspector.healthCheck-1","data":{"score":92.4}}}`+"\n"+
			`{"timestamp":"2026-04-19T09:01:00Z","type":"callback_started","payload":{"id":"Remediator.remediate-2","name":"Remediator.remediate"}}`+"\n"+
			`{"timestamp":"2026-04-19T09:01:02Z","type":"callback_error","payload":{"id":"Remediator.remediate-2","error":"command denied"}}`+"\n",
	), 0o644); err != nil {
		t.Fatalf("WriteFile(trace) error = %v", err)
	}

	cfg := config.Default()
	cfg.Observability.TraceLogPath = tracePath

	collector := NewCollector(cfg, nil, nil, nil)
	snapshot, err := collector.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}

	if snapshot.Tools.Total != 2 || snapshot.Tools.Errors != 1 {
		t.Fatalf("tool summary = total %d errors %d, want total 2 errors 1", snapshot.Tools.Total, snapshot.Tools.Errors)
	}
	if snapshot.AgentByName("Inspector").Runs != 1 {
		t.Fatalf("Inspector runs = %d, want 1", snapshot.AgentByName("Inspector").Runs)
	}
	if snapshot.AgentByName("Remediator").Status != "error" {
		t.Fatalf("Remediator status = %q, want error", snapshot.AgentByName("Remediator").Status)
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("Marshal(snapshot) error = %v", err)
	}
	if string(data) == "null" || !contains(string(data), `"recent":[]`) {
		t.Fatalf("snapshot JSON should encode empty recent lists as arrays: %s", data)
	}
}

func TestCollectorIncludesTraceDetailsAndDocumentLibrary(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tracePath := filepath.Join(dir, "trace.log")
	docsPath := filepath.Join(dir, "docs", "sop")
	skillsPath := filepath.Join(dir, "skills")
	if err := os.MkdirAll(docsPath, 0o755); err != nil {
		t.Fatalf("MkdirAll(docs) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(skillsPath, "health-check"), 0o755); err != nil {
		t.Fatalf("MkdirAll(skills) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(docsPath, "restart-nginx.md"), []byte("# Restart Nginx\n\nUse this SOP when nginx is down.\n\n```bash\nsystemctl restart nginx\n```\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(sop) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillsPath, "health-check", "SKILL.md"), []byte("---\nname: health-check\n---\n# Health Check\n\nCollect host metrics.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(skill) error = %v", err)
	}
	if err := os.WriteFile(tracePath, []byte(
		`{"timestamp":"2026-04-19T09:00:00Z","type":"callback_started","payload":{"id":"Inspector.healthCheck-1","name":"Inspector.healthCheck"}}`+"\n"+
			`{"timestamp":"2026-04-19T09:00:01Z","type":"callback_completed","payload":{"id":"Inspector.healthCheck-1","data":{"score":92.4,"component":"cpu"}}}`+"\n",
	), 0o644); err != nil {
		t.Fatalf("WriteFile(trace) error = %v", err)
	}

	cfg := config.Default()
	cfg.Observability.TraceLogPath = tracePath
	cfg.KnowledgeBase.DocsPath = docsPath

	collector := NewCollector(cfg, nil, nil, nil)
	collector.skillsPath = skillsPath
	snapshot, err := collector.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}

	if len(snapshot.Tools.Recent) != 1 {
		t.Fatalf("len(Tools.Recent) = %d, want 1", len(snapshot.Tools.Recent))
	}
	if len(snapshot.Tools.Recent[0].Events) != 2 {
		t.Fatalf("tool events = %d, want 2", len(snapshot.Tools.Recent[0].Events))
	}
	if snapshot.Tools.Recent[0].Data["score"] != float64(92.4) {
		t.Fatalf("tool data score = %#v, want 92.4", snapshot.Tools.Recent[0].Data["score"])
	}
	if snapshot.Documents.Total != 2 {
		t.Fatalf("documents total = %d, want 2", snapshot.Documents.Total)
	}
	if snapshot.Documents.ByKind["sop"] != 1 || snapshot.Documents.ByKind["skill"] != 1 {
		t.Fatalf("documents by kind = %#v, want sop=1 skill=1", snapshot.Documents.ByKind)
	}
	if snapshot.Documents.Items[0].ID == snapshot.Documents.Items[1].ID {
		t.Fatalf("document IDs should be unique: %#v", snapshot.Documents.Items)
	}
	if snapshot.Documents.Items[0].Title == "" || snapshot.Documents.Items[0].Preview == "" {
		t.Fatalf("document title and preview should be populated: %#v", snapshot.Documents.Items[0])
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }

func contains(input, needle string) bool {
	return strings.Contains(input, needle)
}
