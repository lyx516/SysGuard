package observability

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTraceEventsRedactSensitivePayloadValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.log")
	callbacks, err := NewGlobalCallback(true, path)
	if err != nil {
		t.Fatalf("new callback: %v", err)
	}

	id := callbacks.OnCallbackStarted("test")
	callbacks.OnCallbackCompleted(id, map[string]interface{}{
		"auth_token": "secret-token",
		"nested": map[string]interface{}{
			"password": "database-password",
		},
	})

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}
	text := string(data)
	if strings.Contains(text, "secret-token") || strings.Contains(text, "database-password") {
		t.Fatalf("trace contains unredacted secret: %s", text)
	}
	if !strings.Contains(text, "[REDACTED]") {
		t.Fatalf("trace should contain redaction marker: %s", text)
	}
}

func TestTraceWriteErrorsAreCounted(t *testing.T) {
	dir := t.TempDir()
	callbacks, err := NewGlobalCallback(true, dir)
	if err != nil {
		t.Fatalf("new callback: %v", err)
	}

	callbacks.OnCallbackStarted("test")
	if callbacks.TraceWriteErrors() == 0 {
		t.Fatal("expected trace write error counter to increase")
	}
}
