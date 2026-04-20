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
agents:
  remediator:
    command_timeout: 90s
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
	if cfg.Agents.Remediator.CommandTimeout.String() != "1m30s" {
		t.Fatalf("unexpected timeout: %s", cfg.Agents.Remediator.CommandTimeout)
	}
}

func TestLoadParsesProductionGuardrails(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `
agents:
  remediator:
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

	if !cfg.Agents.Remediator.DryRun {
		t.Fatal("expected remediator dry-run mode to be enabled")
	}
	if !cfg.Agents.Remediator.VerifyAfterRemediation {
		t.Fatal("expected post-remediation verification to be enabled")
	}
	if cfg.UI.Addr != "127.0.0.1:9090" {
		t.Fatalf("unexpected UI addr: %q", cfg.UI.Addr)
	}
	if cfg.UI.AuthToken != "local-token" {
		t.Fatalf("unexpected UI auth token: %q", cfg.UI.AuthToken)
	}
}
