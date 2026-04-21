package orchestration

import (
	"context"
	"fmt"
)

func (r *Runtime) verifyResult(ctx context.Context, state *State) (*State, error) {
	if state.Branch != BranchAI || !r.cfg.Execution.VerifyAfterRemediation {
		return state, nil
	}
	state.Verification.Attempted = true
	report, err := r.monitor.CheckHealth(ctx)
	if err != nil {
		state.Verification.Message = err.Error()
		return state, nil
	}
	state.Verification.Healthy = report.IsHealthy
	state.Verification.Message = fmt.Sprintf("health_score=%.2f healthy=%t", report.Score, report.IsHealthy)
	return state, nil
}
