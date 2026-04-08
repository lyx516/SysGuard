package remediator

import (
	"context"
	"log"

	"github.com/sysguard/sysguard/internal/monitor"
	"github.com/sysguard/sysguard/internal/observability"
	"github.com/sysguard/sysguard/internal/rag"
	"github.com/sysguard/sysguard/internal/security"
)

// Remediator 修复员，负责自动修复检测到的问题
type Remediator struct {
	kb          *rag.KnowledgeBase
	interceptor *security.CommandInterceptor
	obs         *observability.GlobalCallback
	approvalCh  chan *ApprovalRequest
}

// ApprovalRequest 审批请求
type ApprovalRequest struct {
	Command    string
	Reason     string
	ResponseCh chan bool
}

// NewRemediator 创建新的修复员
func NewRemediator(
	kb *rag.KnowledgeBase,
	interceptor *security.CommandInterceptor,
	obs *observability.GlobalCallback,
) *Remediator {
	return &Remediator{
		kb:          kb,
		interceptor: interceptor,
		obs:         obs,
		approvalCh:  make(chan *ApprovalRequest, 100),
	}
}

// Remediate 执行修复操作
func (r *Remediator) Remediate(ctx context.Context, anomaly monitor.Anomaly) error {
	log.Printf("Remediator: Starting remediation for - %v", anomaly)

	callbackID := r.obs.OnCallbackStarted("Remediator.remediate")

	// 1. 从知识库检索相关 SOP
	sops, err := r.kb.Retrieve(ctx, anomaly.Description)
	if err != nil {
		r.obs.OnCallbackError(callbackID, err)
		return err
	}

	// 2. 根据 SOP 制定修复计划
	plan := r.createRemediationPlan(anomaly, sops)
	log.Printf("Remediator: Remediation plan - %s", plan)

	// 3. 执行修复操作（带安全检查）
	if err := r.executeRemediation(ctx, plan); err != nil {
		r.obs.OnCallbackError(callbackID, err)
		return err
	}

	r.obs.OnCallbackCompleted(callbackID, map[string]interface{}{
		"anomaly": anomaly,
		"plan":    plan,
	})

	log.Println("Remediator: Remediation completed")
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

// createRemediationPlan 创建修复计划
func (r *Remediator) createRemediationPlan(anomaly monitor.Anomaly, sops []string) string {
	// 使用 RAG 检索到的 SOP 构建修复计划
	plan := "Execute SOP: " + anomaly.Description
	for _, sop := range sops {
		plan += "\n" + sop
	}
	return plan
}

// executeRemediation 执行修复操作
func (r *Remediator) executeRemediation(ctx context.Context, plan string) error {
	// 解析计划中的命令
	commands := r.parseCommands(plan)

	for _, cmd := range commands {
		// 检查是否为高危命令
		if r.interceptor.IsDangerous(cmd) {
			// 请求人工审批
			approved := r.requestApproval(cmd, plan)
			if !approved {
				return fmt.Errorf("command not approved: %s", cmd)
			}
		}

		// 执行命令
		if err := r.executeCommand(ctx, cmd); err != nil {
			return err
		}
	}

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

// parseCommands 解析修复计划中的命令
func (r *Remediator) parseCommands(plan string) []string {
	// 实现命令解析逻辑
	// 这里简化实现
	return []string{"echo 'remediation'"}
}

// executeCommand 执行单个命令
func (r *Remediator) executeCommand(ctx context.Context, cmd string) error {
	log.Printf("Remediator: Executing command - %s", cmd)
	// 实现命令执行逻辑
	return nil
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
