package inspector

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/sysguard/sysguard/internal/config"
	"github.com/sysguard/sysguard/internal/monitor"
	"github.com/sysguard/sysguard/internal/observability"
	"github.com/sysguard/sysguard/internal/rag"
)

type Inspector struct {
	kb       *rag.KnowledgeBase
	monitor  *monitor.Monitor
	obs      *observability.GlobalCallback
	interval time.Duration
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func NewInspector(cfg *config.Config, kb *rag.KnowledgeBase, monitor *monitor.Monitor, obs *observability.GlobalCallback) *Inspector {
	interval := cfg.Monitor.CheckInterval
	if cfg.Agents.Inspector.Interval > 0 {
		interval = cfg.Agents.Inspector.Interval
	}

	return &Inspector{
		kb:       kb,
		monitor:  monitor,
		obs:      obs,
		interval: interval,
	}
}

func (i *Inspector) Start(ctx context.Context) error {
	runCtx, cancel := context.WithCancel(ctx)
	i.cancel = cancel
	i.wg.Add(1)

	go func() {
		defer i.wg.Done()
		log.Println("Inspector: Starting health checks")
		ticker := time.NewTicker(i.interval)
		defer ticker.Stop()

		i.runHealthCheck(runCtx)
		for {
			select {
			case <-runCtx.Done():
				log.Println("Inspector: Stopped")
				return
			case <-ticker.C:
				i.runHealthCheck(runCtx)
			}
		}
	}()

	return nil
}

func (i *Inspector) Stop(ctx context.Context) error {
	if i.cancel != nil {
		i.cancel()
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		i.wg.Wait()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (i *Inspector) runHealthCheck(ctx context.Context) {
	callbackID := i.obs.OnCallbackStarted("Inspector.healthCheck")

	healthReport, err := i.monitor.CheckHealth(ctx)
	if err != nil {
		i.obs.OnCallbackError(callbackID, err)
		log.Printf("Inspector: Health check failed - %v", err)
		return
	}

	log.Printf("Inspector: Health check completed - score=%.2f healthy=%t", healthReport.Score, healthReport.IsHealthy)
	if !healthReport.IsHealthy {
		anomaly := i.monitor.BuildAnomaly(healthReport)
		if err := i.monitor.NotifyAnomaly(ctx, anomaly); err != nil {
			i.obs.OnCallbackError(callbackID, err)
			log.Printf("Inspector: Failed to notify anomaly - %v", err)
			return
		}
		i.obs.OnCallbackCompleted(callbackID, map[string]interface{}{
			"anomaly": anomaly,
			"score":   healthReport.Score,
		})
		return
	}

	i.obs.OnCallbackCompleted(callbackID, map[string]interface{}{
		"score": healthReport.Score,
	})
}

func (i *Inspector) GetSystemStatus(ctx context.Context) (*monitor.HealthReport, error) {
	return i.monitor.CheckHealth(ctx)
}
