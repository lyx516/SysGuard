package orchestration

import (
	"context"
)

func (r *Runtime) retrieveEvidence(ctx context.Context, state *State) (*State, error) {
	if state.Branch != BranchAI || state.Anomaly == nil {
		return state, nil
	}
	if r.kb != nil {
		chunks, err := r.kb.RetrieveEvidence(ctx, state.Anomaly.Description, 5)
		if err != nil {
			return nil, err
		}
		state.Evidence.SOP = chunks
	}
	if r.historyKB != nil {
		records, err := r.historyKB.SearchSimilarRecords(ctx, state.Anomaly.Description, 0.35)
		if err != nil {
			return nil, err
		}
		state.Evidence.History = records
	}
	return state, nil
}
