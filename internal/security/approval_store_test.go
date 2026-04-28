package security

import (
	"context"
	"path/filepath"
	"testing"
)

func TestApprovalStoreCreatesDecidesConsumesAndPersists(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "approvals.json")
	store, err := NewApprovalStore(path)
	if err != nil {
		t.Fatalf("new approval store: %v", err)
	}
	req, err := store.Create(context.Background(), ApprovalRequest{
		Tool:    "service-management",
		Action:  "restart",
		Command: "systemctl restart nginx",
	})
	if err != nil {
		t.Fatalf("create approval: %v", err)
	}
	if req.Status != ApprovalPending || req.ID == "" {
		t.Fatalf("unexpected created request: %#v", req)
	}
	if _, err := store.Decide(context.Background(), req.ID, true, "test"); err != nil {
		t.Fatalf("approve request: %v", err)
	}

	reloaded, err := NewApprovalStore(path)
	if err != nil {
		t.Fatalf("reload approval store: %v", err)
	}
	if _, err := reloaded.Consume(context.Background(), req.ID, "systemctl restart nginx"); err != nil {
		t.Fatalf("consume request: %v", err)
	}
	items, err := reloaded.List(context.Background(), 10)
	if err != nil {
		t.Fatalf("list approvals: %v", err)
	}
	if len(items) != 1 || items[0].Status != ApprovalUsed {
		t.Fatalf("unexpected approvals: %#v", items)
	}
}
