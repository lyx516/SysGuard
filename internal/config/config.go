package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Monitor       MonitorConfig
	Orchestration OrchestrationConfig
	Execution     ExecutionConfig
	Security      SecurityConfig
	KnowledgeBase KnowledgeBaseConfig
	Observability ObservabilityConfig
	Storage       StorageConfig
	UI            UIConfig
	AI            AIConfig
	Services      []string
}

type MonitorConfig struct {
	CheckInterval   time.Duration
	HealthThreshold float64
	CPUThreshold    float64
	MemoryThreshold float64
	DiskThreshold   float64
}

type OrchestrationConfig struct {
	Interval        time.Duration
	AnomalyCooldown time.Duration
}

type ExecutionConfig struct {
	CommandTimeout         time.Duration
	AutoApproveSafe        bool
	AllowInteractiveInput  bool
	DryRun                 bool
	VerifyAfterRemediation bool
}

type SecurityConfig struct {
	DangerousCommands []string
	EnableApproval    bool
	ApprovalTimeout   time.Duration
}

type KnowledgeBaseConfig struct {
	DocsPath string
}

type ObservabilityConfig struct {
	EnableTracing bool
	TraceLogPath  string
}

type StorageConfig struct {
	HistoryPath   string
	LogPath       string
	RunsPath      string
	ApprovalsPath string
}

type UIConfig struct {
	Addr      string
	AuthToken string
}

type AIConfig struct {
	Enabled     bool
	Provider    string
	Model       string
	APIKeyEnv   string
	APIKey      string
	BaseURL     string
	Timeout     time.Duration
	MaxTokens   int
	Temperature float64
}

type pathEntry struct {
	indent int
	key    string
}

func Default() *Config {
	return &Config{
		Monitor: MonitorConfig{
			CheckInterval:   30 * time.Second,
			HealthThreshold: 80,
			CPUThreshold:    85,
			MemoryThreshold: 90,
			DiskThreshold:   90,
		},
		Orchestration: OrchestrationConfig{
			Interval:        30 * time.Second,
			AnomalyCooldown: 30 * time.Second,
		},
		Execution: ExecutionConfig{
			CommandTimeout:         2 * time.Minute,
			AutoApproveSafe:        true,
			AllowInteractiveInput:  true,
			DryRun:                 true,
			VerifyAfterRemediation: true,
		},
		Security: SecurityConfig{
			DangerousCommands: []string{"rm", "kill", "killall", "dd", "mkfs", "shutdown", "reboot", "launchctl unload", "systemctl stop"},
			EnableApproval:    true,
			ApprovalTimeout:   5 * time.Minute,
		},
		KnowledgeBase: KnowledgeBaseConfig{
			DocsPath: "./docs/sop",
		},
		Observability: ObservabilityConfig{
			EnableTracing: true,
			TraceLogPath:  "./logs/trace.log",
		},
		Storage: StorageConfig{
			HistoryPath:   "./data/history.json",
			LogPath:       "./logs/sysguard.log",
			RunsPath:      "./data/runs.json",
			ApprovalsPath: "./data/approvals.json",
		},
		UI: UIConfig{
			Addr: "127.0.0.1:8080",
		},
		AI: AIConfig{
			Enabled:     false,
			Provider:    "openai",
			Model:       "gpt-4.1-mini",
			APIKeyEnv:   "OPENAI_API_KEY",
			BaseURL:     "https://api.openai.com/v1",
			Timeout:     30 * time.Second,
			MaxTokens:   2048,
			Temperature: 0.2,
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := Default()
	if path == "" {
		applyEnv(cfg)
		return cfg, nil
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			applyEnv(cfg)
			return cfg, nil
		}
		return nil, err
	}
	defer file.Close()

	var stack []pathEntry
	lists := make(map[string][]string)
	values := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), " \t")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " "))
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") {
			if len(stack) == 0 {
				continue
			}
			pathKey := joinPath(stack)
			lists[pathKey] = append(lists[pathKey], strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
			continue
		}

		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}

		if value == "" {
			stack = append(stack, pathEntry{indent: indent, key: key})
			continue
		}

		pathParts := make([]pathEntry, 0, len(stack)+1)
		pathParts = append(pathParts, stack...)
		pathParts = append(pathParts, pathEntry{key: key})
		values[joinPath(pathParts)] = strings.Trim(value, `"'`)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if v := values["monitor.check_interval"]; v != "" {
		cfg.Monitor.CheckInterval = parseDuration(v, cfg.Monitor.CheckInterval)
	}
	if v := values["monitor.health_threshold"]; v != "" {
		cfg.Monitor.HealthThreshold = parseFloat(v, cfg.Monitor.HealthThreshold)
	}
	if v := values["monitor.cpu_threshold"]; v != "" {
		cfg.Monitor.CPUThreshold = parseFloat(v, cfg.Monitor.CPUThreshold)
	}
	if v := values["monitor.memory_threshold"]; v != "" {
		cfg.Monitor.MemoryThreshold = parseFloat(v, cfg.Monitor.MemoryThreshold)
	}
	if v := values["monitor.disk_threshold"]; v != "" {
		cfg.Monitor.DiskThreshold = parseFloat(v, cfg.Monitor.DiskThreshold)
	}
	if v := values["orchestration.interval"]; v != "" {
		cfg.Orchestration.Interval = parseDuration(v, cfg.Orchestration.Interval)
	}
	if v := values["orchestration.anomaly_cooldown"]; v != "" {
		cfg.Orchestration.AnomalyCooldown = parseDuration(v, cfg.Orchestration.AnomalyCooldown)
	}
	if v := values["execution.command_timeout"]; v != "" {
		cfg.Execution.CommandTimeout = parseDuration(v, cfg.Execution.CommandTimeout)
	}
	if v := values["execution.auto_approve_safe_commands"]; v != "" {
		cfg.Execution.AutoApproveSafe = parseBool(v, cfg.Execution.AutoApproveSafe)
	}
	if v := values["execution.allow_interactive_input"]; v != "" {
		cfg.Execution.AllowInteractiveInput = parseBool(v, cfg.Execution.AllowInteractiveInput)
	}
	if v := values["execution.dry_run"]; v != "" {
		cfg.Execution.DryRun = parseBool(v, cfg.Execution.DryRun)
	}
	if v := values["execution.verify_after_remediation"]; v != "" {
		cfg.Execution.VerifyAfterRemediation = parseBool(v, cfg.Execution.VerifyAfterRemediation)
	}
	if v := values["security.enable_approval"]; v != "" {
		cfg.Security.EnableApproval = parseBool(v, cfg.Security.EnableApproval)
	}
	if v := values["security.approval_timeout"]; v != "" {
		cfg.Security.ApprovalTimeout = parseDuration(v, cfg.Security.ApprovalTimeout)
	}
	if v := values["knowledge_base.docs_path"]; v != "" {
		cfg.KnowledgeBase.DocsPath = v
	}
	if v := values["observability.enable_tracing"]; v != "" {
		cfg.Observability.EnableTracing = parseBool(v, cfg.Observability.EnableTracing)
	}
	if v := values["observability.trace_log_path"]; v != "" {
		cfg.Observability.TraceLogPath = v
	}
	if v := values["storage.history_path"]; v != "" {
		cfg.Storage.HistoryPath = v
	}
	if v := values["storage.runs_path"]; v != "" {
		cfg.Storage.RunsPath = v
	}
	if v := values["storage.approvals_path"]; v != "" {
		cfg.Storage.ApprovalsPath = v
	}
	if v := values["logging.output"]; v != "" {
		cfg.Storage.LogPath = v
	}
	if v := values["ui.addr"]; v != "" {
		cfg.UI.Addr = v
	}
	if v := values["ui.auth_token"]; v != "" {
		cfg.UI.AuthToken = v
	}
	if v := values["ai.enabled"]; v != "" {
		cfg.AI.Enabled = parseBool(v, cfg.AI.Enabled)
	}
	if v := values["ai.provider"]; v != "" {
		cfg.AI.Provider = v
	}
	if v := values["ai.model"]; v != "" {
		cfg.AI.Model = v
	}
	if v := values["ai.api_key_env"]; v != "" {
		cfg.AI.APIKeyEnv = v
	}
	if v := values["ai.api_key"]; v != "" {
		cfg.AI.APIKey = v
	}
	if v := values["ai.base_url"]; v != "" {
		cfg.AI.BaseURL = v
	}
	if v := values["ai.timeout"]; v != "" {
		cfg.AI.Timeout = parseDuration(v, cfg.AI.Timeout)
	}
	if v := values["ai.max_tokens"]; v != "" {
		cfg.AI.MaxTokens = parseInt(v, cfg.AI.MaxTokens)
	}
	if v := values["ai.temperature"]; v != "" {
		cfg.AI.Temperature = parseFloat(v, cfg.AI.Temperature)
	}

	if cmds := lists["security.dangerous_commands"]; len(cmds) > 0 {
		cfg.Security.DangerousCommands = cmds
	}
	if services := lists["services.names"]; len(services) > 0 {
		cfg.Services = services
	}

	applyEnv(cfg)
	normalizePaths(cfg)
	return cfg, nil
}

func joinPath(stack []pathEntry) string {
	parts := make([]string, 0, len(stack))
	for _, item := range stack {
		parts = append(parts, item.key)
	}
	return strings.Join(parts, ".")
}

func parseDuration(v string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func parseFloat(v string, fallback float64) float64 {
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

func parseInt(v string, fallback int) int {
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

func parseBool(v string, fallback bool) bool {
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func applyEnv(cfg *Config) {
	if services := strings.TrimSpace(os.Getenv("SYSGUARD_SERVICES")); services != "" {
		parts := strings.Split(services, ",")
		cfg.Services = cfg.Services[:0]
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				cfg.Services = append(cfg.Services, trimmed)
			}
		}
	}
	if token := strings.TrimSpace(os.Getenv("SYSGUARD_UI_AUTH_TOKEN")); token != "" {
		cfg.UI.AuthToken = token
	}
	if enabled := strings.TrimSpace(os.Getenv("SYSGUARD_AI_ENABLED")); enabled != "" {
		cfg.AI.Enabled = parseBool(enabled, cfg.AI.Enabled)
	}
	if provider := strings.TrimSpace(os.Getenv("SYSGUARD_AI_PROVIDER")); provider != "" {
		cfg.AI.Provider = provider
	}
	if model := strings.TrimSpace(os.Getenv("SYSGUARD_AI_MODEL")); model != "" {
		cfg.AI.Model = model
	}
	if apiKeyEnv := strings.TrimSpace(os.Getenv("SYSGUARD_AI_API_KEY_ENV")); apiKeyEnv != "" {
		cfg.AI.APIKeyEnv = apiKeyEnv
	}
	if baseURL := strings.TrimSpace(os.Getenv("SYSGUARD_AI_BASE_URL")); baseURL != "" {
		cfg.AI.BaseURL = baseURL
	}
	if cfg.AI.APIKeyEnv != "" {
		if envValue := strings.TrimSpace(os.Getenv(cfg.AI.APIKeyEnv)); envValue != "" {
			cfg.AI.APIKey = envValue
		}
	}
	if apiKey := strings.TrimSpace(os.Getenv("SYSGUARD_AI_API_KEY")); apiKey != "" {
		cfg.AI.APIKey = apiKey
	}
}

func normalizePaths(cfg *Config) {
	cfg.KnowledgeBase.DocsPath = absPath(cfg.KnowledgeBase.DocsPath)
	cfg.Observability.TraceLogPath = absPath(cfg.Observability.TraceLogPath)
	cfg.Storage.HistoryPath = absPath(cfg.Storage.HistoryPath)
	cfg.Storage.LogPath = absPath(cfg.Storage.LogPath)
	cfg.Storage.RunsPath = absPath(cfg.Storage.RunsPath)
	cfg.Storage.ApprovalsPath = absPath(cfg.Storage.ApprovalsPath)
}

func absPath(value string) string {
	if value == "" || filepath.IsAbs(value) {
		return value
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		return value
	}
	return filepath.Clean(absolute)
}
