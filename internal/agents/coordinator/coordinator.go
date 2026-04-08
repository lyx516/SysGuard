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
)

// Coordinator 协调器，负责管理 Inspector 和 Remediator 的协同工作
type Coordinator struct {
	inspector  *inspector.Inspector
	remediator *remediator.Remediator
	kb         *rag.KnowledgeBase
	monitor    *monitor.Monitor
	interceptor *security.CommandInterceptor
	obs        *observability.GlobalCallback
}

// NewCoordinator 创建新的协调器
func NewCoordinator(
	kb *rag.KnowledgeBase,
	monitor *monitor.Monitor,
	interceptor *security.CommandInterceptor,
	obs *observability.GlobalCallback,
) *Coordinator {
	return &Coordinator{
		kb:          kb,
		monitor:     monitor,
		interceptor: interceptor,
		obs:         obs,
	}
}

// Start 启动协调器
func (c *Coordinator) Start(ctx context.Context) error {
	// 初始化 Inspector
	c.inspector = inspector.NewInspector(c.kb, c.monitor, c.obs)

	// 初始化 Remediator
	c.remediator = remediator.NewRemediator(c.kb, c.interceptor, c.obs)

	// 注册异常回调
	c.monitor.RegisterAnomalyHandler(c.handleAnomaly)

	// 启动 Inspector
	if err := c.inspector.Start(ctx); err != nil {
		return fmt.Errorf("failed to start inspector: %w", err)
	}

	log.Println("Coordinator: All agents started")
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
