package orchestration

import (
	"context"
	"strings"
	"time"

	"github.com/sysguard/sysguard/internal/rag"
)

func (r *Runtime) persistResult(ctx context.Context, state *State) (*State, error) {
	if state.Anomaly == nil {
		return state, nil
	}
	if state.Branch == BranchSuppressed || state.Branch == BranchHealthy {
		return state, nil
	}
	if r.historyKB == nil {
		return state, nil
	}

	solution := state.Agent.Final
	if solution == "" {
		solution = string(state.Branch)
	}
	metadata := make(map[string]string, len(state.Anomaly.Metadata)+4)
	for k, v := range state.Anomaly.Metadata {
		metadata[k] = v
	}
	metadata["mode"] = string(state.Branch)
	metadata["trigger"] = string(state.Trigger)
	metadata["run_id"] = state.RunID
	if state.Agent.Error != "" {
		metadata["graph_error"] = state.Agent.Error
	}
	if state.Verification.Attempted {
		metadata["verification"] = state.Verification.Message
	}

	timestamp := state.Anomaly.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	err := r.historyKB.AddRecord(ctx, &rag.HistoryRecord{
		ProblemType: state.Anomaly.Source,
		Description: state.Anomaly.Description,
		Solution:    solution,
		Steps:       append([]string(nil), state.Agent.Tools...),
		Success:     state.Agent.Error == "" && !strings.EqualFold(state.Branch.String(), "error"),
		Timestamp:   timestamp,
		Metadata:    metadata,
	})
	if err != nil {
		state.Persistence.Error = err.Error()
		return state, nil
	}
	state.Persistence.HistoryWritten = true
	return state, nil
}

func (b Branch) String() string {
	return string(b)
}
