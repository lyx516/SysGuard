package orchestration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/sysguard/sysguard/internal/config"
	"github.com/sysguard/sysguard/internal/monitor"
	"github.com/sysguard/sysguard/internal/rag"
)

func TestRouteModeSuppressesRepeatedAnomalyWithinCooldown(t *testing.T) {
	cfg := config.Default()
	cfg.AI.Enabled = true
	cfg.Orchestration.AnomalyCooldown = time.Minute
	anomaly := monitor.Anomaly{
		Source:      "monitor",
		Severity:    "critical",
		Description: "service down",
		Metadata:    map[string]string{"service_name": "demo"},
	}
	runtime := &Runtime{
		cfg: cfg,
		lastHandled: map[string]time.Time{
			anomalySignature(anomaly): time.Now().UTC(),
		},
	}
	state := NewState(TriggerPeriodic)
	state.Anomaly = &anomaly
	state.Report = &monitor.HealthReport{IsHealthy: false}

	next, err := runtime.routeMode(context.Background(), state)
	if err != nil {
		t.Fatalf("routeMode() error = %v", err)
	}
	if next.Branch != BranchSuppressed {
		t.Fatalf("routeMode() branch = %s, want %s", next.Branch, BranchSuppressed)
	}
	if !next.Suppressed {
		t.Fatalf("routeMode() did not mark state suppressed")
	}
}

func TestRouteModeAlertOnlyWhenAIDisabled(t *testing.T) {
	cfg := config.Default()
	cfg.AI.Enabled = false
	cfg.Orchestration.AnomalyCooldown = time.Minute
	runtime := &Runtime{cfg: cfg, lastHandled: map[string]time.Time{}}
	state := NewState(TriggerManualCheck)
	state.Report = &monitor.HealthReport{IsHealthy: false}
	anomaly := monitor.Anomaly{Source: "monitor", Severity: "warning", Description: "disk high"}
	state.Anomaly = &anomaly

	next, err := runtime.routeMode(context.Background(), state)
	if err != nil {
		t.Fatalf("routeMode() error = %v", err)
	}
	if next.Branch != BranchAlertOnly {
		t.Fatalf("routeMode() branch = %s, want %s", next.Branch, BranchAlertOnly)
	}
}

func TestRunStatePersistsFailedAgentRun(t *testing.T) {
	ctx := context.Background()
	historyKB, err := rag.NewHistoryKnowledgeBase(t.TempDir() + "/history.json")
	if err != nil {
		t.Fatalf("NewHistoryKnowledgeBase() error = %v", err)
	}
	cfg := config.Default()
	runtime := &Runtime{
		cfg:       cfg,
		graph:     failingGraph{err: fmt.Errorf("agent failed")},
		historyKB: historyKB,
	}
	state := NewState(TriggerPeriodic)
	state.Branch = BranchAI
	state.Anomaly = &monitor.Anomaly{
		Source:      "monitor",
		Severity:    "critical",
		Description: "service down",
		Metadata:    map[string]string{"service_name": "demo"},
	}

	out, err := runtime.RunState(ctx, state)
	if err == nil {
		t.Fatalf("RunState() error = nil, want graph error")
	}
	if out == nil || !out.Persistence.HistoryWritten {
		t.Fatalf("RunState() did not mark failed run as persisted: %#v", out)
	}
	records, err := historyKB.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("history record count = %d, want 1", len(records))
	}
	if records[0].Success {
		t.Fatalf("history record Success = true, want false")
	}
	if records[0].Metadata["graph_error"] != "agent failed" {
		t.Fatalf("history graph_error = %q, want agent failed", records[0].Metadata["graph_error"])
	}
}

type failingGraph struct {
	err error
}

var _ compose.Runnable[*State, *State] = failingGraph{}

func (f failingGraph) Invoke(ctx context.Context, input *State, opts ...compose.Option) (*State, error) {
	input.Agent.Error = f.err.Error()
	return input, f.err
}

func (f failingGraph) Stream(ctx context.Context, input *State, opts ...compose.Option) (*schema.StreamReader[*State], error) {
	return nil, f.err
}

func (f failingGraph) Collect(ctx context.Context, input *schema.StreamReader[*State], opts ...compose.Option) (*State, error) {
	return nil, f.err
}

func (f failingGraph) Transform(ctx context.Context, input *schema.StreamReader[*State], opts ...compose.Option) (*schema.StreamReader[*State], error) {
	return nil, f.err
}
