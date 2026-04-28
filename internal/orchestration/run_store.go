package orchestration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const defaultRunStoreLimit = 200

type RunStatus string

const (
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
)

type RunRecord struct {
	RunID          string    `json:"run_id"`
	Trigger        Trigger   `json:"trigger"`
	Branch         Branch    `json:"branch"`
	Status         RunStatus `json:"status"`
	StartedAt      time.Time `json:"started_at"`
	CompletedAt    time.Time `json:"completed_at,omitempty"`
	DurationMillis int64     `json:"duration_millis,omitempty"`
	Healthy        bool      `json:"healthy"`
	HealthScore    float64   `json:"health_score"`
	Anomaly        string    `json:"anomaly,omitempty"`
	Severity       string    `json:"severity,omitempty"`
	AgentFinal     string    `json:"agent_final,omitempty"`
	AgentError     string    `json:"agent_error,omitempty"`
	Tools          []string  `json:"tools,omitempty"`
	Verified       bool      `json:"verified"`
	Verification   string    `json:"verification,omitempty"`
	HistoryWritten bool      `json:"history_written"`
	PersistError   string    `json:"persist_error,omitempty"`
}

type RunStore struct {
	path  string
	limit int
	mu    sync.RWMutex
	runs  map[string]RunRecord
}

func NewRunStore(path string) (*RunStore, error) {
	store := &RunStore{
		path:  path,
		limit: defaultRunStoreLimit,
		runs:  make(map[string]RunRecord),
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *RunStore) Upsert(ctx context.Context, state *State, status RunStatus) error {
	if s == nil || s.path == "" || state == nil {
		return nil
	}
	record := NewRunRecord(state, status)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[record.RunID] = record
	s.pruneLocked()
	return s.saveLocked()
}

func (s *RunStore) List(ctx context.Context, limit int) ([]RunRecord, error) {
	if s == nil {
		return nil, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	records := make([]RunRecord, 0, len(s.runs))
	for _, record := range s.runs {
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].StartedAt.After(records[j].StartedAt)
	})
	if limit > 0 && len(records) > limit {
		records = records[:limit]
	}
	return records, nil
}

func (s *RunStore) Get(ctx context.Context, runID string) (RunRecord, bool, error) {
	if s == nil {
		return RunRecord{}, false, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.runs[runID]
	return record, ok, nil
}

func NewRunRecord(state *State, status RunStatus) RunRecord {
	record := RunRecord{
		RunID:          state.RunID,
		Trigger:        state.Trigger,
		Branch:         state.Branch,
		Status:         status,
		StartedAt:      state.StartedAt,
		CompletedAt:    state.CompletedAt,
		Tools:          append([]string(nil), state.Agent.Tools...),
		AgentFinal:     state.Agent.Final,
		AgentError:     state.Agent.Error,
		Verified:       state.Verification.Attempted,
		Verification:   state.Verification.Message,
		HistoryWritten: state.Persistence.HistoryWritten,
		PersistError:   state.Persistence.Error,
	}
	if state.Report != nil {
		record.Healthy = state.Report.IsHealthy
		record.HealthScore = state.Report.Score
	}
	if state.Anomaly != nil {
		record.Anomaly = state.Anomaly.Description
		record.Severity = state.Anomaly.Severity
	}
	if !record.StartedAt.IsZero() && !record.CompletedAt.IsZero() {
		record.DurationMillis = record.CompletedAt.Sub(record.StartedAt).Milliseconds()
	}
	if record.Status == RunStatusCompleted && record.AgentError != "" {
		record.Status = RunStatusFailed
	}
	return record
}

func (s *RunStore) load() error {
	if s.path == "" {
		return nil
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}
	var records []RunRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return err
	}
	for _, record := range records {
		if record.RunID != "" {
			s.runs[record.RunID] = record
		}
	}
	return nil
}

func (s *RunStore) pruneLocked() {
	records := make([]RunRecord, 0, len(s.runs))
	for _, record := range s.runs {
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].StartedAt.After(records[j].StartedAt)
	})
	for i := s.limit; i < len(records); i++ {
		delete(s.runs, records[i].RunID)
	}
}

func (s *RunStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	records := make([]RunRecord, 0, len(s.runs))
	for _, record := range s.runs {
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].StartedAt.After(records[j].StartedAt)
	})
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".runs-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(append(data, '\n')); err != nil {
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
	if err := os.Rename(tmpPath, s.path); err != nil {
		return err
	}
	return os.Chmod(s.path, 0o600)
}
