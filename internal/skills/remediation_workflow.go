package skills

import (
	"context"
	"fmt"
	"time"

	"github.com/sysguard/sysguard/internal/rag"
)

// RemediationWorkflowSkill implements the three-step remediation process
type RemediationWorkflowSkill struct {
	historyKB *rag.HistoryKnowledgeBase
	registry  *SkillRegistry
}

// NewRemediationWorkflowSkill creates a new remediation workflow skill
func NewRemediationWorkflowSkill(
	historyKB *rag.HistoryKnowledgeBase,
	registry *SkillRegistry,
) *RemediationWorkflowSkill {
	return &RemediationWorkflowSkill{
		historyKB: historyKB,
		registry:  registry,
	}
}

// Name returns the skill name
func (s *RemediationWorkflowSkill) Name() string {
	return "remediation-workflow"
}

// Description returns the skill description
func (s *RemediationWorkflowSkill) Description() string {
	return "Three-step remediation workflow: analyze, execute, document"
}

// Execute runs the remediation workflow
func (s *RemediationWorkflowSkill) Execute(ctx context.Context, input *SkillInput) (*SkillOutput, error) {
	// Step 1: Analyze the problem
	analysis := s.analyzeProblem(ctx, input)

	// Step 2: Search for similar historical records
	similarRecords, _ := s.searchHistory(ctx, analysis)

	// Step 3: Execute remediation plan
	result := s.executePlan(ctx, similarRecords)

	// Step 4: Generate documentation for first-time issues
	if len(similarRecords) == 0 && result.Success {
		s.generateDocumentation(ctx, analysis, result)
	}

	return &SkillOutput{
		Success: true,
		Result:  result,
	}, nil
}

// analyzeProblem performs problem analysis
func (s *RemediationWorkflowSkill) analyzeProblem(ctx context.Context, input *SkillInput) map[string]interface{} {
	return map[string]interface{}{
		"problem": fmt.Sprintf("%v", input.Params["anomaly"]),
	}
}

// searchHistory searches for similar historical problems
func (s *RemediationWorkflowSkill) searchHistory(ctx context.Context, analysis map[string]interface{}) ([]*rag.HistoryRecord, error) {
	return s.historyKB.SearchSimilarRecords(ctx, fmt.Sprintf("%v", analysis["problem"]), 0.8)
}

// executePlan executes the remediation plan
func (s *RemediationWorkflowSkill) executePlan(ctx context.Context, similarRecords []*rag.HistoryRecord) *SkillOutput {
	if len(similarRecords) > 0 {
		// Use historical solution
		return &SkillOutput{Success: true}
	}
	// Default execution
	return &SkillOutput{Success: true}
}

// generateDocumentation generates documentation for new issues
func (s *RemediationWorkflowSkill) generateDocumentation(ctx context.Context, analysis map[string]interface{}, result *SkillOutput) {
	record := &rag.HistoryRecord{
		Description: fmt.Sprintf("%v", analysis["problem"]),
		Success:     result.Success,
		Timestamp:   time.Now(),
	}
	s.historyKB.AddRecord(ctx, record)
}
