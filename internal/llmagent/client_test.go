package llmagent

import "testing"

func TestParseDecisionFromPlainJSON(t *testing.T) {
	decision, err := ParseDecision(`{"action":"tool","tool":"health_check","args":{"scope":"host"},"thought":"need fresh state"}`)
	if err != nil {
		t.Fatalf("parse decision: %v", err)
	}
	if decision.Action != ActionTool {
		t.Fatalf("action = %q, want %q", decision.Action, ActionTool)
	}
	if decision.Tool != "health_check" {
		t.Fatalf("tool = %q, want health_check", decision.Tool)
	}
	if decision.Args["scope"] != "host" {
		t.Fatalf("args = %#v", decision.Args)
	}
}

func TestParseDecisionFromFencedJSON(t *testing.T) {
	decision, err := ParseDecision("```json\n{\"action\":\"final\",\"final_answer\":\"no action needed\"}\n```")
	if err != nil {
		t.Fatalf("parse decision: %v", err)
	}
	if decision.Action != ActionFinal {
		t.Fatalf("action = %q, want %q", decision.Action, ActionFinal)
	}
	if decision.FinalAnswer != "no action needed" {
		t.Fatalf("final answer = %q", decision.FinalAnswer)
	}
}

func TestParseDecisionRejectsMissingToolName(t *testing.T) {
	if _, err := ParseDecision(`{"action":"tool"}`); err == nil {
		t.Fatal("expected missing tool name to fail")
	}
}
