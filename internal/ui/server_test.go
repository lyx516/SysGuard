package ui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sysguard/sysguard/internal/config"
	"github.com/sysguard/sysguard/internal/monitor"
	"github.com/sysguard/sysguard/internal/orchestration"
)

func TestServerExposesDashboardResourceEndpoints(t *testing.T) {
	t.Parallel()

	collector := NewCollector(config.Default(), nil, nil, nil)
	server := NewServer(":0", collector)

	cases := []struct {
		path string
		key  string
	}{
		{path: "/api/tools", key: "recent"},
		{path: "/api/logs", key: "recent"},
		{path: "/api/history", key: "recent"},
		{path: "/api/runs", key: "recent"},
		{path: "/api/documents", key: "items"},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			res := httptest.NewRecorder()
			server.mux.ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 body=%s", res.Code, res.Body.String())
			}
			var body map[string]interface{}
			if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
				t.Fatalf("response is not JSON: %v", err)
			}
			if _, ok := body[tc.key]; !ok {
				t.Fatalf("response missing key %q: %#v", tc.key, body)
			}
		})
	}
}

func TestServerExposesRunRecords(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := config.Default()
	cfg.Storage.RunsPath = dir + "/runs.json"
	store, err := orchestration.NewRunStore(cfg.Storage.RunsPath)
	if err != nil {
		t.Fatalf("new run store: %v", err)
	}
	state := orchestration.NewState(orchestration.TriggerManualCheck)
	state.Branch = orchestration.BranchAlertOnly
	state.CompletedAt = state.StartedAt.Add(time.Second)
	if err := store.Upsert(context.Background(), state, orchestration.RunStatusCompleted); err != nil {
		t.Fatalf("upsert run: %v", err)
	}

	collector := NewCollector(cfg, nil, nil, nil)
	server := NewServer(":0", collector)

	req := httptest.NewRequest(http.MethodGet, "/api/runs/"+state.RunID, nil)
	res := httptest.NewRecorder()
	server.mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", res.Code, res.Body.String())
	}
	var body orchestration.RunRecord
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not run JSON: %v", err)
	}
	if body.RunID != state.RunID || body.Status != orchestration.RunStatusCompleted {
		t.Fatalf("unexpected run body: %#v", body)
	}
}

func TestServerRequiresBearerTokenForAPIWhenConfigured(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	cfg.UI.AuthToken = "secret-token"
	collector := NewCollector(cfg, nil, nil, nil)
	server := NewServer(":0", collector)

	req := httptest.NewRequest(http.MethodGet, "/api/snapshot", nil)
	res := httptest.NewRecorder()
	server.mux.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status without token = %d, want 401", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/snapshot", nil)
	req.Header.Set("Authorization", "secret-token")
	res = httptest.NewRecorder()
	server.mux.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status with bare token = %d, want 401", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/snapshot", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	res = httptest.NewRecorder()
	server.mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status with bearer token = %d, want 200 body=%s", res.Code, res.Body.String())
	}
}

func TestServerRejectsPostOnReadOnlyEndpoints(t *testing.T) {
	t.Parallel()

	collector := NewCollector(config.Default(), nil, nil, nil)
	server := NewServer(":0", collector)

	req := httptest.NewRequest(http.MethodPost, "/api/logs", nil)
	res := httptest.NewRecorder()
	server.mux.ServeHTTP(res, req)
	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", res.Code)
	}
}

type checkTriggeringMonitor struct {
	checks   int
	notifies int
}

func (m *checkTriggeringMonitor) CheckHealth(ctx context.Context) (*monitor.HealthReport, error) {
	m.checks++
	return &monitor.HealthReport{
		Timestamp: time.Now().UTC(),
		IsHealthy: false,
		Score:     40,
		Components: map[string]monitor.ComponentStatus{
			"services": {
				Name:    "services",
				Status:  "down",
				Message: "service down",
				Metrics: map[string]interface{}{"failed_service": "nginx"},
			},
		},
	}, nil
}

func (m *checkTriggeringMonitor) BuildAnomaly(report *monitor.HealthReport) monitor.Anomaly {
	return monitor.Anomaly{
		Timestamp:   report.Timestamp,
		Severity:    "critical",
		Description: "service down",
		Source:      "monitor",
	}
}

func (m *checkTriggeringMonitor) NotifyAnomaly(ctx context.Context, anomaly monitor.Anomaly) error {
	m.notifies++
	return nil
}

func TestServerCheckEndpointTriggersImmediateHealthCheckAndAnomalyFlow(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	mon := &checkTriggeringMonitor{}
	collector := NewCollector(cfg, mon, nil, nil)
	server := NewServer(":0", collector)

	req := httptest.NewRequest(http.MethodPost, "/api/check", nil)
	res := httptest.NewRecorder()
	server.mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", res.Code, res.Body.String())
	}
	if mon.checks == 0 {
		t.Fatal("expected /api/check to run health checks immediately")
	}
	if mon.notifies != 1 {
		t.Fatalf("notify count = %d, want 1", mon.notifies)
	}
}
