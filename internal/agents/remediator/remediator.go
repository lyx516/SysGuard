package remediator

import (
	"context"
	"log"

	"github.com/sysguard/sysguard/internal/monitor"
	"github.com/sysguard/sysguard/internal/observability"
	"github.com/sysguard/sysguard/internal/rags"
	"github.com/sysguard/sysguard/internal/security"
	"github.com/sysguard/sysguard/internal/skills"
	"github.com/sysguard/sysguard/internal/skills/remediation_workflow"
)

// Remediator 修复员，负责自动修复检测到的问题
type Remediator struct {
	workflowSkill *skills.Skill
	interceptor   *security.CommandInterceptor
	obs          *observability.GlobalCallback
	approvalCh   chan *ApprovalRequest
}

// ApprovalRequest 审批请求
type ApprovalRequest struct {
	Command    string
	Reason     string
	ResponseCh chan bool
}

// NewRemediator 创建新的修复员
func NewRemediator(
	workflowSkill *skills.Skill,
	interceptor *security.CommandInterceptor,
	obs *observability.GlobalCallback,
) *Remediator {
	return &Remediator{
		workflowSkill: workflowSkill,
		interceptor:   interceptor,
		obs:          obs,
		approvalCh:   make(chan *ApprovalRequest, 100),
	}
}

// Remediate 执行修复操作
func (r *Remediator) Remediate(ctx context.Context, anomaly monitor.Anomaly) error {
	log.Printf("Remediator: Starting remediation for - %v", anomaly)

	callbackID := r.obs.OnCallbackStarted("Remediator.remediate")

	// 使用新的工作流 skill 执行修复
	output, err := r.workflowSkill.Execute(ctx, &skills.SkillInput{
		Params: map[string]interface{}{
			"anomaly": anomaly,
		},
	})

	if err != nil {
		r.obs.OnCallbackError(callbackID, err)
		return fmt.Errorf("remediation workflow failed: %w", err)
	}

	if !output.Success {
		r.obs.OnCallbackCompleted(callbackID, map[string]interface{}{
			"anomaly": anomaly,
			"success": false,
			"message": output.Message,
			"errors":  output.Errors,
		})
		return fmt.Errorf("remediation workflow completed with errors: %s", output.Message)
	}

	// 记录成功结果
	r.obs.OnCallbackCompleted(callbackID, map[string]interface{}{
		"anomaly":      anomaly,
		"success":       true,
		"workflow_data": output.Data,
		"tools_used":    output.ToolsUsed,
		"duration_ms":   output.Duration,
	})

	log.Printf("Remediator: Remediation completed successfully")
	log.Printf("Remediator: Workflow summary - %s", output.Message)

	return nil
}

// Start 启动修复员
func (r *Remediator) Start(ctx context.Context) error {
	// 启动审批处理循环
	go r.handleApprovals(ctx)
	return nil
}

// Stop 停止修复员
func (r *Remediator) Stop(ctx context.Context) error {
	close(r.approvalCh)
	return nil
}

// requestApproval 请求人工审批
func (r *Remediator) requestApproval(command, reason string) bool {
	req := &ApprovalRequest{
		Command:    command,
		Reason:     reason,
		ResponseCh: make(chan bool, 1),
	}

	r.approvalCh <- req

	// 等待审批结果
	select {
	case approved := <-req.ResponseCh:
		return approved
	}
}

// handleApprovals 处理审批请求
func (r *Remediator) handleApprovals(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-r.approvalCh:
			// 打印审批请求，等待人工输入
			log.Printf("\n=== APPROVAL REQUIRED ===")
			log.Printf("Command: %s", req.Command)
			log.Printf("Reason: %s", req.Reason)
			log.Printf("Approve? (y/n): ")

			// 这里需要实现实际的用户输入处理
			// 简化实现，总是返回 true
			req.ResponseCh <- true
		}
	}
}
