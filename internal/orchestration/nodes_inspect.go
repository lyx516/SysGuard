package orchestration

import (
	"context"
)

func (r *Runtime) inspect(ctx context.Context, state *State) (*State, error) {
	report, err := r.monitor.CheckHealth(ctx)
	if err != nil {
		return nil, err
	}
	state.Report = report
	return state, nil
}

func (r *Runtime) detectAnomaly(ctx context.Context, state *State) (*State, error) {
	if state.Report == nil || state.Report.IsHealthy {
		return state, nil
	}
	anomaly := r.monitor.BuildAnomaly(state.Report)
	state.Anomaly = &anomaly
	return state, nil
}
