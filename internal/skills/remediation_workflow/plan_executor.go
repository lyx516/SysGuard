package remediation_workflow

import (
	"context"
	"fmt"

	"github.com/sysguard/sysguard/internal/skills"
)

// PlanExecutor 计划执行工具
type PlanExecutor struct {
	skillReg *skills.SkillRegistry
}

// NewPlanExecutor 创建新的计划执行器
func NewPlanExecutor(skillReg *skills.SkillRegistry) *PlanExecutor {
	return &PlanExecutor{
		skillReg: skillReg,
	}
}

// Name 返回工具名称
func (t *PlanExecutor) Name() string {
	return "plan_executor"
}

// Description 返回工具描述
func (t *PlanExecutor) Description() string {
	return "执行修复计划中的各个步骤，调用相应的 skills"
}

// Execute 执行工具
func (t *PlanExecutor) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	plan, ok := input.Params["plan"].(*RemediationPlan)
	if !ok {
		return nil, fmt.Errorf("plan parameter is required")
	}

	results, err := t.executeSteps(ctx, plan.Steps)
	if err != nil {
		return nil, err
	}

	success := t.checkAllSuccess(results)

	return &skills.ToolOutput{
		Success: success,
		Data: map[string]interface{}{
			"steps_executed": len(plan.Steps),
			"results":        results,
			"all_success":    success,
		},
	}, nil
}

// executeSteps 执行所有步骤
func (t *PlanExecutor) executeSteps(ctx context.Context, steps []PlanStep) ([]*skills.SkillOutput, error) {
	results := make([]*skills.SkillOutput, 0, len(steps))

	for _, step := range steps {
		result, err := t.executeStep(ctx, step)
		if err != nil {
			return results, fmt.Errorf("step %d failed: %w", step.StepNumber, err)
		}

		results = append(results, result)

		// 如果步骤失败，是否继续？
		// 这里简化实现，继续执行所有步骤
	}

	return results, nil
}

// executeStep 执行单个步骤
func (t *PlanExecutor) executeStep(ctx context.Context, step PlanStep) (*skills.SkillOutput, error) {
	// 获取对应的 skill
	skill, ok := t.skillReg.Get(step.SkillName)
	if !ok {
		return &skills.SkillOutput{
			Success: false,
			Message: fmt.Sprintf("Skill not found: %s", step.SkillName),
			Data:    map[string]interface{}{},
		}, nil
	}

	// 执行 skill
	output, err := skill.Execute(ctx, &skills.SkillInput{
		Command: step.Action,
		Params:  step.Parameters,
	})

	if err != nil {
		return &skills.SkillOutput{
			Success: false,
			Message: fmt.Sprintf("Step execution error: %v", err),
			Errors:  []string{err.Error()},
		}, err
	}

	return output
}

// checkAllSuccess 检查是否所有步骤都成功
func (t *PlanExecutor) checkAllSuccess(results []*skills.SkillOutput) bool {
	for _, result := range results {
		if !result.Success {
			return false
		}
	}
	return true
}

// GetFailedSteps 获取失败的步骤
func (t *PlanExecutor) GetFailedSteps(results []*skills.SkillOutput) []int {
	failed := make([]int, 0)
	for i, result := range results {
		if !result.Success {
			failed = append(failed, i+1)
		}
	}
	return failed
}

// RetryStep 重试失败的步骤
func (t *PlanExecutor) RetryStep(ctx context.Context, step PlanStep, maxRetries int) (*skills.SkillOutput, error) {
	var lastError error
	var lastResult *skills.SkillOutput

	for attempt := 1; attempt <= maxRetries; attempt++ {
		result, err := t.executeStep(ctx, step)
		if err != nil {
			lastError = err
			continue
		}

		if result.Success {
			return result, nil
		}

		lastResult = result
	}

	if lastError != nil {
		return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastError)
	}

	return lastResult, fmt.Errorf("failed after %d attempts", maxRetries)
}
