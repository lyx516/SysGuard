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
	"github.com/sysguard/sysguard/internal/rag"
	"github.com/sysguard/sysguard/internal/security"
	"github.com/sysguard/sysguard/internal/ui"
)

func main() {
	configPath := flag.String("config", "./configs/config.yaml", "Path to SysGuard config file")
	addr := flag.String("addr", "", "UI listen address")
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

	obs, err := observability.NewGlobalCallback(cfg.Observability.EnableTracing, cfg.Observability.TraceLogPath)
	if err != nil {
		log.Fatalf("Failed to initialize observability: %v", err)
	}
	historyKB, err := rag.NewHistoryKnowledgeBase(cfg.Storage.HistoryPath)
	if err != nil {
		log.Fatalf("Failed to initialize history knowledge base: %v", err)
	}
	interceptor := security.NewCommandInterceptor(cfg.Security.DangerousCommands)
	healthMonitor := monitor.NewMonitor(cfg, interceptor, obs)

	listenAddr := cfg.UI.Addr
	if *addr != "" {
		listenAddr = *addr
	}
	collector := ui.NewCollector(cfg, healthMonitor, obs, historyKB)
	server := ui.NewServer(listenAddr, collector)

	log.Printf("SysGuard UI started at http://%s", listenAddr)
	if err := server.ListenAndServe(ctx); err != nil {
		log.Fatalf("SysGuard UI stopped with error: %v", err)
	}
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
