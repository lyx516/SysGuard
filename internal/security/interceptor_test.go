package security

import "testing"

func TestInterceptorDetectsDangerousCommands(t *testing.T) {
	interceptor := NewCommandInterceptor([]string{"rm", "shutdown"})

	if !interceptor.IsDangerous("rm -rf /tmp/demo") {
		t.Fatalf("expected rm to be dangerous")
	}
	if interceptor.IsDangerous("printf rm") {
		t.Fatalf("did not expect non-prefix command to be dangerous")
	}
}
