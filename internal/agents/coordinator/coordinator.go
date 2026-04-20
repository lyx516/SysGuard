package coordinator

import (
	"context"
	"fmt"
	"log"

	"github.com/sysguard/sysguard/internal/agents/inspector"
	"github.com/sysguard/sysguard/internal/agents/remediator"
	"github.com/sysguard/sysguard/internal/config"
	"github.com/sysguard/sysguard/internal/monitor"
	"github.com/sysguard/sysguard/internal/observability"
	"github.com/sysguard/sysguard/internal/rag"
	"github.com/sysguard/sysguard/internal/security"
)

type Coordinator struct {
	inspector   *inspector.Inspector
	remediator  *remediator.Remediator
	kb          *rag.KnowledgeBase
	historyKB   *rag.HistoryKnowledgeBase
	monitor     *monitor.Monitor
	interceptor *security.CommandInterceptor
	obs         *observability.GlobalCallback
	cfg         *config.Config
}

func NewCoordinator(
	cfg *config.Config,
	kb *rag.KnowledgeBase,
	historyKB *rag.HistoryKnowledgeBase,
	monitor *monitor.Monitor,
	interceptor *security.CommandInterceptor,
	obs *observability.GlobalCallback,
) *Coordinator {
	return &Coordinator{
		cfg:         cfg,
		kb:          kb,
		historyKB:   historyKB,
		monitor:     monitor,
		interceptor: interceptor,
		obs:         obs,
	}
}

func (c *Coordinator) Start(ctx context.Context) error {
	c.inspector = inspector.NewInspector(c.cfg, c.kb, c.monitor, c.obs)
	c.remediator = remediator.NewRemediator(c.cfg, c.kb, c.historyKB, c.interceptor, c.obs)
	c.remediator.SetVerifier(func(ctx context.Context, anomaly monitor.Anomaly) error {
		report, err := c.monitor.CheckHealth(ctx)
		if err != nil {
			return err
		}
		if !report.IsHealthy {
			return fmt.Errorf("health score %.2f below threshold after remediation", report.Score)
		}
		return nil
	})
	c.monitor.RegisterAnomalyHandler(c.handleAnomaly)

	if err := c.remediator.Start(ctx); err != nil {
		return fmt.Errorf("failed to start remediator: %w", err)
	}
	if err := c.inspector.Start(ctx); err != nil {
		return fmt.Errorf("failed to start inspector: %w", err)
	}

	log.Println("Coordinator: All agents started")
	return nil
}

func (c *Coordinator) Stop(ctx context.Context) error {
	if c.inspector != nil {
		if err := c.inspector.Stop(ctx); err != nil && !isContextCancellation(err) {
			return err
		}
	}
	if c.remediator != nil {
		if err := c.remediator.Stop(ctx); err != nil && !isContextCancellation(err) {
			return err
		}
	}
	return nil
}

func (c *Coordinator) handleAnomaly(ctx context.Context, anomaly monitor.Anomaly) error {
	log.Printf("Coordinator: Anomaly detected - severity=%s description=%s", anomaly.Severity, anomaly.Description)
	if err := c.remediator.Remediate(ctx, anomaly); err != nil {
		return fmt.Errorf("remediation failed: %w", err)
	}
	return nil
}

func isContextCancellation(err error) bool {
	return err == context.Canceled || err == context.DeadlineExceeded
}
