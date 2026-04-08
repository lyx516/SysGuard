package coordinator

import (
	"context"
	"fmt"
	"log"

	"github.com/sysguard/sysguard/internal/agents/inspector"
	"github.com/sysguard/sysguard/internal/agents/remediator"
	"github.com/sysguard/sysguard/internal/monitor"
	"github.com/sysguard/sysguard/internal/observability"
	"github.com/sysguard/sysguard/internal/rag"
	"github.com/sysguard/sysguard/internal/security"
	"github.com/sysguard/sysguard/internal/skills"
	"github.com/sysguard/sysguard/internal/skills/remediation_workflow"
)

// Coordinator 协调器，负责管理 Inspector 和 Remediator 的协同工作
type Coordinator struct {
	inspector   *inspector.Inspector
	remediator  *remediator.Remediator
	kb          *rag.KnowledgeBase
	historyKB   *rag.HistoryKnowledgeBase
	skillReg    *skills.SkillRegistry
	monitor     *monitor.Monitor
	interceptor  *security.CommandInterceptor
	obs         *observability.GlobalCallback
}

// NewCoordinator 创建新的协调器
func NewCoordinator(
	kb *rag.KnowledgeBase,
	monitor *monitor.Monitor,
	interceptor *security.CommandInterceptor,
	obs *observability.GlobalCallback,
) *Coordinator {
	return &Coordinator{
		kb:         kb,
		monitor:    monitor,
		interceptor: interceptor,
		obs:        obs,
	}
}

// Start 启动协调器
func (c *Coordinator) Start(ctx context.Context) error {
	// 1. 初始化历史知识库
	historyPath := "./docs/history"
	maxHistoryRecords := 1000
	historyKB, err := rag.NewHistoryKnowledgeBase(historyPath, maxHistoryRecords)
	if err != nil {
		log.Printf("Warning: Failed to initialize history KB: %v", err)
		// 继续启动，历史功能可能不可用
	}
	c.historyKB = historyKB

	// 2. 初始化 Skills 注册表
	skillReg := skills.NewDefaultRegistry()
	c.skillReg = skillReg

	// 3. 注册新的 Remediation Workflow Skill
	if err := skills.RegisterWorkflowSkill(skillReg, historyKB, c.kb); err != nil {
		log.Printf("Warning: Failed to register workflow skill: %v", err)
	}

	// 4. 初始化 Inspector
	c.inspector = inspector.NewInspector(c.kb, c.monitor, c.obs)

	// 5. 初始化 Remediator（使用新的 Workflow Skill）
	c.remediator = remediator.NewRemediator(workflowSkill, c.interceptor, c.obs)

	// 6. 注册异常回调
	c.monitor.RegisterAnomalyHandler(c.handleAnomaly)

	// 7. 启动 Inspector
	if err := c.inspector.Start(ctx); err != nil {
		return fmt.Errorf("failed to start inspector: %w", err)
	}

	log.Println("Coordinator: All agents started")
	log.Printf("Coordinator: History KB loaded with %d records", historyKB.GetRecordCount())

	return nil
}

// Stop 停止协调器
func (c *Coordinator) Stop(ctx context.Context) error {
	if c.inspector != nil {
		if err := c.inspector.Stop(ctx); err != nil {
			return err
		}
	}

	if c.remediator != nil {
		if err := c.remediator.Stop(ctx); err != nil {
			return err
		}
	}

	return nil
}

// handleAnomaly 处理异常，唤醒 Remediator
func (c *Coordinator) handleAnomaly(ctx context.Context, anomaly monitor.Anomaly) error {
	log.Printf("Coordinator: Anomaly detected - %v", anomaly)

	// 唤醒 Remediator 进行修复
	if err := c.remediator.Remediate(ctx, anomaly); err != nil {
		return fmt.Errorf("remediation failed: %w", err)
	}

	return nil
}
