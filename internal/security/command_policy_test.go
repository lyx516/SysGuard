package security

import "testing"

func TestCommandPolicyAllowsTemplatesAndRejectsDangerousArguments(t *testing.T) {
	policy := NewCommandPolicy([]CommandTemplate{
		{Name: "service-status", Permission: PermissionPrivileged, Program: "systemctl", Args: []string{"is-active", "{service}"}},
	})

	audit, err := policy.Validate("systemctl is-active nginx")
	if err != nil {
		t.Fatalf("validate allowed command: %v", err)
	}
	if audit.Template != "service-status" || audit.Permission != PermissionPrivileged {
		t.Fatalf("unexpected audit: %#v", audit)
	}

	if _, err := policy.Validate("systemctl is-active nginx;rm"); err == nil {
		t.Fatal("expected unsafe argument to fail")
	}
	if _, err := policy.Validate("rm -rf /"); err == nil {
		t.Fatal("expected unlisted command to fail")
	}
}
