package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sysguard/sysguard/internal/agents/coordinator"
	"github.com/sysguard/sysguard/internal/monitor"
	"github.com/sysguard/sysguard/internal/observability"
	"github.com/sysguard/sysguard/internal/rag"
	"github.com/sysguard/sysguard/internal/security"
)

func main() {
	ctx := context.Background()

	// 初始化可观测性
	obs, err := observability.NewGlobalCallback()
	if err != nil {
		log.Fatalf("Failed to initialize observability: %v", err)
	}

	// 初始化知识库
	kb, err := rag.NewKnowledgeBase(ctx, "./docs/sop")
	if err != nil {
		log.Fatalf("Failed to initialize knowledge base: %v", err)
	}

	// 初始化安全拦截器
	interceptor := security.NewCommandInterceptor()

	// 初始化监控器
	monitor := monitor.NewMonitor(interceptor, obs)

	// 初始化协调器
	coord := coordinator.NewCoordinator(kb, monitor, interceptor, obs)

	// 启动系统
	if err := coord.Start(ctx); err != nil {
		log.Fatalf("Failed to start coordinator: %v", err)
	}

	log.Println("SysGuard started successfully")

	// 等待退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// 优雅关闭
	if err := coord.Stop(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("SysGuard stopped")
}
