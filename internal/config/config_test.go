package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadParsesKeySettings(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `
monitor:
  check_interval: 45s
  health_threshold: 75
security:
  dangerous_commands:
    - rm
    - shutdown
services:
  names:
    - nginx
execution:
  command_timeout: 90s
storage:
  runs_path: ./data/test-runs.json
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Monitor.CheckInterval.String() != "45s" {
		t.Fatalf("unexpected check interval: %s", cfg.Monitor.CheckInterval)
	}
	if cfg.Monitor.HealthThreshold != 75 {
		t.Fatalf("unexpected threshold: %v", cfg.Monitor.HealthThreshold)
	}
	if len(cfg.Security.DangerousCommands) != 2 {
		t.Fatalf("unexpected dangerous command count: %d", len(cfg.Security.DangerousCommands))
	}
	if len(cfg.Services) != 1 || cfg.Services[0] != "nginx" {
		t.Fatalf("unexpected services: %#v", cfg.Services)
	}
	if cfg.Execution.CommandTimeout.String() != "1m30s" {
		t.Fatalf("unexpected timeout: %s", cfg.Execution.CommandTimeout)
	}
	if filepath.Base(cfg.Storage.RunsPath) != "test-runs.json" {
		t.Fatalf("unexpected runs path: %s", cfg.Storage.RunsPath)
	}
}

func TestLoadParsesProductionGuardrails(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `
execution:
  dry_run: true
  verify_after_remediation: true
ui:
  addr: "127.0.0.1:9090"
  auth_token: "local-token"
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if !cfg.Execution.DryRun {
		t.Fatal("expected execution dry-run mode to be enabled")
	}
	if !cfg.Execution.VerifyAfterRemediation {
		t.Fatal("expected post-remediation verification to be enabled")
	}
	if cfg.UI.Addr != "127.0.0.1:9090" {
		t.Fatalf("unexpected UI addr: %q", cfg.UI.Addr)
	}
	if cfg.UI.AuthToken != "local-token" {
		t.Fatalf("unexpected UI auth token: %q", cfg.UI.AuthToken)
	}
}

func TestLoadParsesAIConfigAndEnvKey(t *testing.T) {
	t.Setenv("SYSGUARD_AI_API_KEY", "test-key")

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `
ai:
  enabled: true
  provider: openai
  model: gpt-4.1-mini
  api_key_env: SYSGUARD_AI_API_KEY
  base_url: "https://api.openai.com/v1"
  timeout: 45s
  max_tokens: 2048
  temperature: 0.2
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if !cfg.AI.Enabled {
		t.Fatal("expected AI config to be enabled")
	}
	if cfg.AI.Provider != "openai" {
		t.Fatalf("provider = %q, want openai", cfg.AI.Provider)
	}
	if cfg.AI.Model != "gpt-4.1-mini" {
		t.Fatalf("model = %q, want gpt-4.1-mini", cfg.AI.Model)
	}
	if cfg.AI.APIKeyEnv != "SYSGUARD_AI_API_KEY" {
		t.Fatalf("api key env = %q, want SYSGUARD_AI_API_KEY", cfg.AI.APIKeyEnv)
	}
	if cfg.AI.APIKey != "test-key" {
		t.Fatalf("api key was not loaded from environment")
	}
	if cfg.AI.BaseURL != "https://api.openai.com/v1" {
		t.Fatalf("base url = %q", cfg.AI.BaseURL)
	}
	if cfg.AI.Timeout.String() != "45s" {
		t.Fatalf("timeout = %s, want 45s", cfg.AI.Timeout)
	}
	if cfg.AI.MaxTokens != 2048 {
		t.Fatalf("max tokens = %d, want 2048", cfg.AI.MaxTokens)
	}
	if cfg.AI.Temperature != 0.2 {
		t.Fatalf("temperature = %v, want 0.2", cfg.AI.Temperature)
	}
}

func TestLoadParsesInlineAIAPIKey(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `
ai:
  enabled: true
  provider: openai
  model: Qwen/Qwen3.6-35B-A3B
  api_key: direct-secret
  base_url: "https://api.siliconflow.cn/v1"
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.AI.APIKey != "direct-secret" {
		t.Fatalf("api key = %q, want direct-secret", cfg.AI.APIKey)
	}
	if cfg.AI.BaseURL != "https://api.siliconflow.cn/v1" {
		t.Fatalf("base url = %q", cfg.AI.BaseURL)
	}
}
