package monitor

import (
	"testing"

	"github.com/sysguard/sysguard/internal/config"
)

func TestDownComponentForcesUnhealthyAtDefaultThreshold(t *testing.T) {
	mon := NewMonitor(config.Default(), nil, nil)
	components := map[string]ComponentStatus{
		"cpu":      {Name: "cpu", Status: "healthy"},
		"memory":   {Name: "memory", Status: "healthy"},
		"disk":     {Name: "disk", Status: "healthy"},
		"network":  {Name: "network", Status: "healthy"},
		"services": {Name: "services", Status: "down"},
	}

	if mon.isHealthy(components) {
		t.Fatalf("isHealthy() = true, want false when any component is down")
	}
	if score := mon.calculateScore(components); score != 80 {
		t.Fatalf("calculateScore() = %v, want 80 to exercise default threshold boundary", score)
	}
}
