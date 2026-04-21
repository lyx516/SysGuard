package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/sysguard/sysguard/internal/config"
	"github.com/sysguard/sysguard/internal/monitor"
	"github.com/sysguard/sysguard/internal/observability"
	"github.com/sysguard/sysguard/internal/orchestration"
	"github.com/sysguard/sysguard/internal/rag"
	"github.com/sysguard/sysguard/internal/security"
)

func main() {
	configPath := flag.String("config", "./configs/config.yaml", "Path to SysGuard config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logFile, err := setupLogging(cfg.Storage.LogPath)
	if err != nil {
		log.Fatalf("Failed to initialize logging: %v", err)
	}
	if logFile != nil {
		defer logFile.Close()
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// 初始化可观测性
	obs, err := observability.NewGlobalCallback(cfg.Observability.EnableTracing, cfg.Observability.TraceLogPath)
	if err != nil {
		log.Fatalf("Failed to initialize observability: %v", err)
	}

	// 初始化知识库
	kb, err := rag.NewKnowledgeBase(ctx, cfg.KnowledgeBase.DocsPath)
	if err != nil {
		log.Fatalf("Failed to initialize knowledge base: %v", err)
	}

	historyKB, err := rag.NewHistoryKnowledgeBase(cfg.Storage.HistoryPath)
	if err != nil {
		log.Fatalf("Failed to initialize history knowledge base: %v", err)
	}

	// 初始化安全拦截器
	interceptor := security.NewCommandInterceptor(cfg.Security.DangerousCommands)

	// 初始化监控器
	monitor := monitor.NewMonitor(cfg, interceptor, obs)

	runtime, err := orchestration.NewRuntime(ctx, cfg, kb, historyKB, monitor, interceptor, obs)
	if err != nil {
		log.Fatalf("Failed to initialize Eino orchestration runtime: %v", err)
	}

	// 启动系统
	if err := runtime.Start(ctx); err != nil {
		log.Fatalf("Failed to start orchestration runtime: %v", err)
	}

	log.Println("SysGuard started successfully")

	<-ctx.Done()

	// 优雅关闭
	if err := runtime.Stop(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("SysGuard stopped")
}

func setupLogging(path string) (*os.File, error) {
	if path == "" {
		return nil, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	log.SetOutput(io.MultiWriter(os.Stdout, file))
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.LUTC)
	return file, nil
}
