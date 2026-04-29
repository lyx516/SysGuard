package orchestration

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/sysguard/sysguard/internal/monitor"
)

func TestRunStorePersistsAndReloadsLatestRunRecords(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "runs.json")
	store, err := NewRunStore(path)
	if err != nil {
		t.Fatalf("new run store: %v", err)
	}
	state := NewState(TriggerManualCheck)
	state.Report = &monitor.HealthReport{IsHealthy: false, Score: 42}
	state.Anomaly = &monitor.Anomaly{Severity: "critical", Description: "nginx down"}

	if err := store.Upsert(context.Background(), state, RunStatusRunning); err != nil {
		t.Fatalf("upsert running: %v", err)
	}
	state.Branch = BranchAI
	state.CompletedAt = state.StartedAt.Add(150 * time.Millisecond)
	state.Persistence.HistoryWritten = true
	if err := store.Upsert(context.Background(), state, RunStatusCompleted); err != nil {
		t.Fatalf("upsert completed: %v", err)
	}

	reloaded, err := NewRunStore(path)
	if err != nil {
		t.Fatalf("reload run store: %v", err)
	}
	records, err := reloaded.List(context.Background(), 10)
	if err != nil {
		t.Fatalf("list records: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("records = %d, want 1", len(records))
	}
	record := records[0]
	if record.RunID != state.RunID || record.Status != RunStatusCompleted || record.Branch != BranchAI {
		t.Fatalf("unexpected record: %#v", record)
	}
	if record.Anomaly != "nginx down" || record.Severity != "critical" || record.DurationMillis != 150 {
		t.Fatalf("record missing run details: %#v", record)
	}
}

func TestRunStoreMarksCompletedRunWithAgentErrorAsFailed(t *testing.T) {
	t.Parallel()

	state := NewState(TriggerPeriodic)
	state.Agent.Error = "model timeout"
	record := NewRunRecord(state, RunStatusCompleted)
	if record.Status != RunStatusFailed {
		t.Fatalf("status = %q, want failed", record.Status)
	}
}
