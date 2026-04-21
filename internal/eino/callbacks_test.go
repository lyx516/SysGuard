package eino

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/callbacks"
	"github.com/sysguard/sysguard/internal/observability"
)

func TestCallbackBridgePublishesLifecycleRecords(t *testing.T) {
	obs, err := observability.NewGlobalCallback(false, "")
	if err != nil {
		t.Fatalf("NewGlobalCallback() error = %v", err)
	}
	bridge := NewCallbackBridge(obs)
	info := &callbacks.RunInfo{Name: "inspect", Component: "Lambda"}

	ctx := bridge.OnStart(context.Background(), info, nil)
	bridge.OnEnd(ctx, info, nil)

	records := obs.GetAllCallbacks()
	if len(records) != 1 {
		t.Fatalf("callback records = %d, want 1", len(records))
	}
	if records[0].Status != "completed" {
		t.Fatalf("callback status = %s, want completed", records[0].Status)
	}
}
