package orchestration

import (
	"fmt"
	"time"

	"github.com/sysguard/sysguard/internal/monitor"
	"github.com/sysguard/sysguard/internal/rag"
)

type Trigger string

const (
	TriggerStartup     Trigger = "startup"
	TriggerPeriodic    Trigger = "periodic"
	TriggerManualCheck Trigger = "manual_check"
)

type Branch string

const (
	BranchUnknown    Branch = "unknown"
	BranchHealthy    Branch = "healthy"
	BranchSuppressed Branch = "suppressed"
	BranchAlertOnly  Branch = "alert_only"
	BranchAI         Branch = "ai"
)

type State struct {
	RunID             string
	Trigger           Trigger
	StartedAt         time.Time
	CompletedAt       time.Time
	Branch            Branch
	Suppressed        bool
	SuppressionReason string
	Report            *monitor.HealthReport
	Anomaly           *monitor.Anomaly
	Evidence          EvidenceBundle
	Agent             AgentOutcome
	Verification      VerificationOutcome
	Persistence       PersistenceOutcome
}

type EvidenceBundle struct {
	SOP     []rag.EvidenceChunk
	History []*rag.HistoryRecord
}

type AgentOutcome struct {
	Final string
	Tools []string
	Error string
}

type VerificationOutcome struct {
	Attempted bool
	Healthy   bool
	Message   string
}

type PersistenceOutcome struct {
	HistoryWritten bool
	Error          string
}

func NewState(trigger Trigger) *State {
	now := time.Now().UTC()
	return &State{
		RunID:     fmt.Sprintf("run-%d", now.UnixNano()),
		Trigger:   trigger,
		StartedAt: now,
		Branch:    BranchUnknown,
	}
}
