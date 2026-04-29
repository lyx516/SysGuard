package orchestration

import (
	"context"
	"encoding/json"
	"time"

	"github.com/sysguard/sysguard/internal/monitor"
)

func (r *Runtime) routeMode(ctx context.Context, state *State) (*State, error) {
	if state.Report == nil || state.Report.IsHealthy || state.Anomaly == nil {
		state.Branch = BranchHealthy
		return state, nil
	}
	if r.suppressed(state.Anomaly) {
		state.Branch = BranchSuppressed
		state.Suppressed = true
		state.SuppressionReason = "cooldown_active"
		return state, nil
	}
	state.Branch = BranchAI
	return state, nil
}

func (r *Runtime) suppressed(anomaly *monitor.Anomaly) bool {
	cooldown := r.cfg.Orchestration.AnomalyCooldown
	if anomaly == nil || cooldown <= 0 {
		return false
	}
	now := time.Now().UTC()
	signature := anomalySignature(*anomaly)

	r.mu.Lock()
	defer r.mu.Unlock()
	if last, ok := r.lastHandled[signature]; ok && now.Sub(last) < cooldown {
		return true
	}
	r.lastHandled[signature] = now
	return false
}

func anomalySignature(anomaly monitor.Anomaly) string {
	metadata, _ := json.Marshal(anomaly.Metadata)
	return anomaly.Source + "|" + anomaly.Severity + "|" + anomaly.Description + "|" + string(metadata)
}
