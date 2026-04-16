package skills

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sysguard/sysguard/internal/config"
	"github.com/sysguard/sysguard/internal/monitor"
	"github.com/sysguard/sysguard/internal/workflow"
)

func TestRegisterCoreSkillsRegistersAllSkills(t *testing.T) {
	registry := NewSkillRegistry()
	if err := RegisterCoreSkills(registry, CoreSkillDependencies{}); err != nil {
		t.Fatalf("register core skills: %v", err)
	}

	expected := []string{
		"log-analysis",
		"health-check",
		"service-management",
		"alerting",
		"metrics-collection",
		"network-diagnosis",
		"container-management",
		"database-operation",
		"file-operation",
		"notification",
	}
	for _, name := range expected {
		skill, err := registry.Get(name)
		if err != nil {
			t.Fatalf("expected skill %q to be registered: %v", name, err)
		}
		if skill.Description() == "" {
			t.Fatalf("expected skill %q to have a description", name)
		}
	}
}

func TestFileOperationSkillSupportsSafeReadStatListAndTail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	content := "line 1\nline 2\nline 3\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	skill := NewFileOperationSkill()
	ctx := context.Background()

	readOut, err := skill.Execute(ctx, &SkillInput{Params: map[string]interface{}{
		"operation": "read",
		"path":      path,
	}})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if readOut.Result.(map[string]interface{})["content"] != content {
		t.Fatalf("unexpected read content: %#v", readOut.Result)
	}

	statOut, err := skill.Execute(ctx, &SkillInput{Params: map[string]interface{}{
		"operation": "stat",
		"path":      path,
	}})
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if statOut.Result.(map[string]interface{})["is_dir"].(bool) {
		t.Fatalf("expected file stat, got directory")
	}

	listOut, err := skill.Execute(ctx, &SkillInput{Params: map[string]interface{}{
		"operation": "list",
		"path":      dir,
	}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listOut.Result.(map[string]interface{})["entries"].([]map[string]interface{})) != 1 {
		t.Fatalf("unexpected list result: %#v", listOut.Result)
	}

	tailOut, err := skill.Execute(ctx, &SkillInput{Params: map[string]interface{}{
		"operation": "tail",
		"path":      path,
		"lines":     2,
	}})
	if err != nil {
		t.Fatalf("tail: %v", err)
	}
	if tailOut.Result.(map[string]interface{})["content"] != "line 2\nline 3" {
		t.Fatalf("unexpected tail content: %#v", tailOut.Result)
	}
}

func TestLogAnalysisSkillUsesWorkflowGraph(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sysguard.log")
	if err := os.WriteFile(path, []byte("ok\nwarning disk\nerror service\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	skill := NewLogAnalysisSkill()
	out, err := skill.Execute(context.Background(), &SkillInput{Params: map[string]interface{}{
		"path":       path,
		"chunk_size": 2,
		"keywords":   []string{"error", "warning"},
	}})
	if err != nil {
		t.Fatalf("log analysis: %v", err)
	}

	result := out.Result.(*workflow.AnalysisResult)
	if result.Total != 2 {
		t.Fatalf("expected 2 matching lines, got %d", result.Total)
	}
	if len(result.Chunks) != 2 {
		t.Fatalf("expected 2 chunks with matches, got %d", len(result.Chunks))
	}
}

func TestAlertingAndNotificationSkillsReturnStructuredPayloads(t *testing.T) {
	ctx := context.Background()

	alertOut, err := NewAlertingSkill().Execute(ctx, &SkillInput{Params: map[string]interface{}{
		"severity": "critical",
		"title":    "Service down",
		"message":  "nginx is inactive",
		"source":   "test",
	}})
	if err != nil {
		t.Fatalf("alerting: %v", err)
	}
	alert := alertOut.Result.(Alert)
	if alert.Severity != "critical" || alert.Title != "Service down" || alert.Timestamp.IsZero() {
		t.Fatalf("unexpected alert: %#v", alert)
	}

	notificationOut, err := NewNotificationSkill(nil).Execute(ctx, &SkillInput{Params: map[string]interface{}{
		"channel": "stdout",
		"message": "hello",
	}})
	if err != nil {
		t.Fatalf("notification: %v", err)
	}
	notification := notificationOut.Result.(Notification)
	if notification.Channel != "stdout" || notification.Message != "hello" || !notification.Sent {
		t.Fatalf("unexpected notification: %#v", notification)
	}
}

func TestHealthAndMetricsSkillsUseMonitorDependency(t *testing.T) {
	cfg := config.Default()
	cfg.Services = nil
	mon := monitor.NewMonitor(cfg, nil, nil)
	registry := NewSkillRegistry()
	if err := RegisterCoreSkills(registry, CoreSkillDependencies{Monitor: mon}); err != nil {
		t.Fatalf("register core skills: %v", err)
	}

	healthOut, err := registry.Execute(context.Background(), "health-check", &SkillInput{})
	if err != nil {
		t.Fatalf("health check: %v", err)
	}
	if healthOut.Result.(*monitor.HealthReport).Timestamp.IsZero() {
		t.Fatalf("expected health report timestamp")
	}

	metricsOut, err := registry.Execute(context.Background(), "metrics-collection", &SkillInput{})
	if err != nil {
		t.Fatalf("metrics collection: %v", err)
	}
	metrics := metricsOut.Result.(map[string]interface{})
	if metrics["timestamp"].(time.Time).IsZero() {
		t.Fatalf("expected metrics timestamp")
	}
	if len(metrics["components"].(map[string]monitor.ComponentStatus)) == 0 {
		t.Fatalf("expected component metrics")
	}
}
