package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/compose"
	flowagent "github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	syseino "github.com/sysguard/sysguard/internal/eino"
	"github.com/sysguard/sysguard/internal/rag"
	"github.com/sysguard/sysguard/internal/skills"
)

func (r *Runtime) agentReact(ctx context.Context, state *State) (*State, error) {
	if state.Branch != BranchAI || state.Anomaly == nil {
		return state, nil
	}

	chatModel, err := syseino.NewChatModel(ctx, r.cfg.AI)
	if err != nil {
		state.Agent.Error = err.Error()
		return nil, err
	}
	definitions, err := r.agentToolDefinitions(ctx, state)
	if err != nil {
		state.Agent.Error = err.Error()
		return nil, err
	}
	tools := syseino.BuildTools(definitions)

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools:               tools,
			UnknownToolsHandler: unknownToolHandler,
		},
		MaxStep: 8,
	})
	if err != nil {
		state.Agent.Error = err.Error()
		return nil, err
	}

	options, err := react.WithTools(ctx, tools...)
	if err != nil {
		state.Agent.Error = err.Error()
		return nil, err
	}
	options = append(options, flowagent.WithComposeOptions(r.callbacks))

	msg, err := agent.Generate(ctx, []*schema.Message{
		schema.SystemMessage(agentSystemPrompt()),
		schema.UserMessage(r.agentUserPrompt(state)),
	}, options...)
	if err != nil {
		state.Agent.Error = err.Error()
		return nil, err
	}
	if msg != nil {
		state.Agent.Final = msg.Content
	}
	state.Agent.Tools = r.collectToolNames(state.StartedAt, definitions)
	return state, nil
}

func (r *Runtime) agentToolDefinitions(ctx context.Context, state *State) ([]skills.ToolDefinition, error) {
	core, err := r.coreSkillDefinitions(ctx)
	if err != nil {
		return nil, err
	}
	defs := append([]skills.ToolDefinition{}, core...)
	defs = append(defs, r.sopRetrievalTool(state), r.historySearchTool(state))
	return defs, nil
}

func (r *Runtime) sopRetrievalTool(state *State) skills.ToolDefinition {
	return skills.ToolDefinition{
		Name:             "sop-retrieval",
		Description:      "Retrieve cited SOP evidence chunks for the current SysGuard anomaly.",
		Permission:       skills.PermissionReadOnly,
		Toolset:          "knowledge",
		SideEffects:      false,
		RequiresApproval: false,
		AllowedPlatforms: []string{"linux", "darwin"},
		OutputBudget:     4000,
		RedactionPolicy:  "return only relevant SOP chunks with source citations",
		Parameters: skills.JSONSchema{
			Type:     "object",
			Required: []string{"query"},
			Properties: map[string]skills.JSONSchemaProperty{
				"query": {Type: "string", Description: "Search query"},
				"limit": {Type: "number", Description: "Maximum evidence chunks"},
			},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (skills.ToolResult, error) {
			query := fmt.Sprintf("%v", args["query"])
			if strings.TrimSpace(query) == "" && state.Anomaly != nil {
				query = state.Anomaly.Description
			}
			limit := 5
			if raw, ok := args["limit"].(float64); ok && raw > 0 {
				limit = int(raw)
			}
			if r.kb == nil {
				return skills.ToolResult{Success: true, Data: []rag.EvidenceChunk{}}, nil
			}
			chunks, err := r.kb.RetrieveEvidence(ctx, query, limit)
			if err != nil {
				return skills.ToolResult{}, err
			}
			return skills.ToolResult{Success: true, Data: chunks}, nil
		},
	}
}

func (r *Runtime) historySearchTool(state *State) skills.ToolDefinition {
	return skills.ToolDefinition{
		Name:             "history-search",
		Description:      "Search prior remediation records related to the current anomaly.",
		Permission:       skills.PermissionReadOnly,
		Toolset:          "knowledge",
		SideEffects:      false,
		RequiresApproval: false,
		AllowedPlatforms: []string{"linux", "darwin"},
		OutputBudget:     4000,
		RedactionPolicy:  "redact secrets and return only incident metadata relevant to the current anomaly",
		Parameters: skills.JSONSchema{
			Type:     "object",
			Required: []string{"query"},
			Properties: map[string]skills.JSONSchemaProperty{
				"query":     {Type: "string", Description: "Search query"},
				"threshold": {Type: "number", Description: "Similarity threshold"},
			},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (skills.ToolResult, error) {
			query := fmt.Sprintf("%v", args["query"])
			if strings.TrimSpace(query) == "" && state.Anomaly != nil {
				query = state.Anomaly.Description
			}
			threshold := 0.35
			if raw, ok := args["threshold"].(float64); ok && raw >= 0 {
				threshold = raw
			}
			if r.historyKB == nil {
				return skills.ToolResult{Success: true, Data: []*rag.HistoryRecord{}}, nil
			}
			records, err := r.historyKB.SearchSimilarRecords(ctx, query, threshold)
			if err != nil {
				return skills.ToolResult{}, err
			}
			return skills.ToolResult{Success: true, Data: records}, nil
		},
	}
}

func agentSystemPrompt() string {
	return "You are SysGuard, an evidence-grounded operations agent. Use cited SOP evidence and history before privileged action. Never invent tools. Prefer read-only diagnosis first. Return a concise final answer with diagnosis, evidence citations, actions taken, verification, and rollback advice."
}

func (r *Runtime) agentUserPrompt(state *State) string {
	payload, _ := json.MarshalIndent(struct {
		Anomaly  *raglessAnomaly      `json:"anomaly"`
		SOP      []rag.EvidenceChunk  `json:"sop_evidence"`
		History  []*rag.HistoryRecord `json:"history"`
		Policy   map[string]any       `json:"tool_policy"`
		Contract map[string]any       `json:"evidence_contract"`
		Final    map[string]string    `json:"final_response_schema"`
	}{
		Anomaly: anomalyPayload(state),
		SOP:     state.Evidence.SOP,
		History: state.Evidence.History,
		Policy: map[string]any{
			"read_only_first":       true,
			"requires_approval":     "privileged or side-effecting tools require explicit approval and must respect dry-run mode",
			"unknown_tools":         "never invent tools; use only registered SysGuard tools",
			"tool_failure_handling": "treat success=false tool results as observations and continue diagnosis when possible",
		},
		Contract: map[string]any{
			"must_use_citations": true,
			"allowed_sources":    []string{"sop_evidence", "history", "tool_observations"},
			"no_evidence_rule":   "if evidence is missing or irrelevant, say so and avoid unsupported remediation",
		},
		Final: map[string]string{
			"diagnosis":     "root cause or current best hypothesis",
			"evidence":      "SOP citations, history records, and tool observations used",
			"actions":       "actions taken or proposed",
			"verification":  "post-action health or checks to run",
			"rollback":      "rollback steps if the action fails",
			"residual_risk": "remaining uncertainty or risk",
		},
	}, "", "  ")
	return "Analyze and handle this SysGuard graph run. Evidence is provided with citations and tools are available for further checks.\n" + string(payload)
}

type raglessAnomaly struct {
	Timestamp   time.Time         `json:"timestamp"`
	Severity    string            `json:"severity"`
	Description string            `json:"description"`
	Source      string            `json:"source"`
	Metadata    map[string]string `json:"metadata"`
}

func anomalyPayload(state *State) *raglessAnomaly {
	if state.Anomaly == nil {
		return nil
	}
	return &raglessAnomaly{
		Timestamp:   state.Anomaly.Timestamp,
		Severity:    state.Anomaly.Severity,
		Description: state.Anomaly.Description,
		Source:      state.Anomaly.Source,
		Metadata:    state.Anomaly.Metadata,
	}
}

func unknownToolHandler(ctx context.Context, name, input string) (string, error) {
	return fmt.Sprintf("unknown tool %q; use only the registered SysGuard tools", name), nil
}

func (r *Runtime) collectToolNames(since time.Time, definitions []skills.ToolDefinition) []string {
	if r.obs == nil {
		return nil
	}
	allowed := make(map[string]bool, len(definitions))
	for _, def := range definitions {
		allowed[def.Name] = true
	}
	seen := make(map[string]bool)
	var names []string
	for _, record := range r.obs.GetAllCallbacks() {
		if record.StartTime.Before(since) {
			continue
		}
		for name := range allowed {
			if strings.Contains(record.ID, name) && !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	return names
}
