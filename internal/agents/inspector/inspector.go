package inspector

import (
	"context"
	"log"
	"time"

	"github.com/sysguard/sysguard/internal/monitor"
	"github.com/sysguard/sysguard/internal/observability"
	"github.com/sysguard/sysguard/internal/rag"
)

// Inspector 巡检员，负责高频健康度检查和结构化日志输出
type Inspector struct {
	kb       *rag.KnowledgeBase
	monitor  *monitor.Monitor
	obs      *observability.GlobalCallback
	interval time.Duration
	stopCh   chan struct{}
}

// NewInspector 创建新的巡检员
func NewInspector(kb *rag.KnowledgeBase, monitor *monitor.Monitor, obs *observability.GlobalCallback) *Inspector {
	return &Inspector{
		kb:       kb,
		monitor:  monitor,
		obs:      obs,
		interval: 30 * time.Second, // 默认30秒检查一次
		stopCh:   make(chan struct{}),
	}
}

// Start 启动巡检员
func (i *Inspector) Start(ctx context.Context) error {
	log.Println("Inspector: Starting health checks")

	ticker := time.NewTicker(i.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-i.stopCh:
			log.Println("Inspector: Stopped")
			return nil
		case <-ticker.C:
			if err := i.runHealthCheck(ctx); err != nil {
				log.Printf("Inspector: Health check failed - %v", err)
			}
		}
	}
}

// Stop 停止巡检员
func (i *Inspector) Stop(ctx context.Context) error {
	close(i.stopCh)
	return nil
}

// runHealthCheck 执行健康检查
func (i *Inspector) runHealthCheck(ctx context.Context) error {
	// 记录回调开始
	callbackID := i.obs.OnCallbackStarted("Inspector.healthCheck")

	// 执行健康检查
	healthReport, err := i.monitor.CheckHealth(ctx)
	if err != nil {
		i.obs.OnCallbackError(callbackID, err)
		return err
	}

	// 输出结构化日志
	log.Printf("Inspector: Health check completed - %+v", healthReport)

	// 检测异常
	if !healthReport.IsHealthy {
		i.obs.OnCallbackCompleted(callbackID, map[string]interface{}{
			"anomaly": healthReport,
		})
	}

	i.obs.OnCallbackCompleted(callbackID, nil)
	return nil
}

// GetSystemStatus 获取系统状态
func (i *Inspector) GetSystemStatus(ctx context.Context) (*monitor.HealthReport, error) {
	return i.monitor.CheckHealth(ctx)
}
