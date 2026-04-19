package ui

import "time"

type A2UIMessage struct {
	Type      string      `json:"type"`
	ID        string      `json:"id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload,omitempty"`
}

type A2UIRenderTree struct {
	Kind     string                 `json:"kind"`
	Title    string                 `json:"title,omitempty"`
	Props    map[string]interface{} `json:"props,omitempty"`
	Children []A2UIRenderTree       `json:"children,omitempty"`
}

func NewA2UIDashboardMessage(snapshot *Snapshot) A2UIMessage {
	return A2UIMessage{
		Type:      "data_model_update",
		ID:        "sysguard-dashboard",
		Timestamp: time.Now().UTC(),
		Payload: map[string]interface{}{
			"model":   snapshot,
			"surface": dashboardSurface(snapshot),
		},
	}
}

func dashboardSurface(snapshot *Snapshot) A2UIRenderTree {
	return A2UIRenderTree{
		Kind:  "dashboard",
		Title: "SysGuard Operations Dashboard",
		Props: map[string]interface{}{
			"health_score": snapshot.System.HealthScore,
			"healthy":      snapshot.System.IsHealthy,
			"agents":       len(snapshot.Agents),
			"tools":        snapshot.Tools.Total,
			"errors":       snapshot.Tools.Errors + snapshot.Logs.Errors,
		},
		Children: []A2UIRenderTree{
			{Kind: "metric_strip", Title: "系统占用", Props: map[string]interface{}{"metrics": snapshot.System.Collected}},
			{Kind: "agent_board", Title: "Agent 运行过程", Props: map[string]interface{}{"agents": snapshot.Agents}},
			{Kind: "tool_table", Title: "工具调用", Props: map[string]interface{}{"calls": snapshot.Tools.Recent}},
			{Kind: "history_table", Title: "问题解决历史", Props: map[string]interface{}{"records": snapshot.History.Recent}},
			{Kind: "log_timeline", Title: "日志统计", Props: map[string]interface{}{"logs": snapshot.Logs, "timeline": snapshot.Timeline}},
		},
	}
}
