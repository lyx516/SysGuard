package ui

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sysguard/sysguard/internal/config"
	"github.com/sysguard/sysguard/internal/monitor"
	"github.com/sysguard/sysguard/internal/observability"
	"github.com/sysguard/sysguard/internal/orchestration"
	"github.com/sysguard/sysguard/internal/rag"
	"github.com/sysguard/sysguard/internal/security"
)

const maxRecentItems = 12

type Collector struct {
	cfg        *config.Config
	monitor    healthChecker
	obs        *observability.GlobalCallback
	historyKB  *rag.HistoryKnowledgeBase
	runner     graphRunner
	skillsPath string
}

type healthChecker interface {
	CheckHealth(ctx context.Context) (*monitor.HealthReport, error)
}

type anomalyTrigger interface {
	BuildAnomaly(report *monitor.HealthReport) monitor.Anomaly
	NotifyAnomaly(ctx context.Context, anomaly monitor.Anomaly) error
}

type graphRunner interface {
	Run(ctx context.Context, trigger orchestration.Trigger) (*orchestration.State, error)
}

type runLister interface {
	ListRuns(ctx context.Context, limit int) ([]orchestration.RunRecord, error)
	GetRun(ctx context.Context, runID string) (orchestration.RunRecord, bool, error)
}

type Snapshot struct {
	GeneratedAt time.Time       `json:"generated_at"`
	System      SystemOverview  `json:"system"`
	Agents      []AgentRuntime  `json:"agents"`
	Tools       ToolSummary     `json:"tools"`
	Logs        LogSummary      `json:"logs"`
	History     HistorySummary  `json:"history"`
	Runs        RunSummary      `json:"runs"`
	Approvals   ApprovalSummary `json:"approvals"`
	Documents   DocumentLibrary `json:"documents"`
	Timeline    []TimelineEvent `json:"timeline"`
}

type SystemOverview struct {
	HealthScore     float64               `json:"health_score"`
	IsHealthy       bool                  `json:"is_healthy"`
	ManagedServices int                   `json:"managed_services"`
	Components      []ComponentView       `json:"components"`
	Config          map[string]string     `json:"config"`
	Collected       map[string]MetricView `json:"collected"`
}

type ComponentView struct {
	Name    string                 `json:"name"`
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Metrics map[string]interface{} `json:"metrics"`
}

type MetricView struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

type AgentRuntime struct {
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	Runs      int       `json:"runs"`
	Errors    int       `json:"errors"`
	LastEvent string    `json:"last_event"`
	LastSeen  time.Time `json:"last_seen"`
}

type ToolSummary struct {
	Total  int        `json:"total"`
	Errors int        `json:"errors"`
	Recent []ToolCall `json:"recent"`
}

type ToolCall struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Status         string                 `json:"status"`
	StartedAt      time.Time              `json:"started_at"`
	DurationMillis int64                  `json:"duration_millis"`
	Summary        string                 `json:"summary"`
	Data           map[string]interface{} `json:"data"`
	Error          string                 `json:"error"`
	Events         []TraceEventView       `json:"events"`
}

type TraceEventView struct {
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
}

type LogSummary struct {
	Total    int        `json:"total"`
	Errors   int        `json:"errors"`
	Warnings int        `json:"warnings"`
	Recent   []LogEntry `json:"recent"`
}

type LogEntry struct {
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

type HistorySummary struct {
	Total   int                 `json:"total"`
	Success int                 `json:"success"`
	Failed  int                 `json:"failed"`
	Recent  []HistoryRecordView `json:"recent"`
}

type RunSummary struct {
	Total   int                       `json:"total"`
	Running int                       `json:"running"`
	Failed  int                       `json:"failed"`
	Recent  []orchestration.RunRecord `json:"recent"`
}

type ApprovalSummary struct {
	Total   int                        `json:"total"`
	Pending int                        `json:"pending"`
	Recent  []security.ApprovalRequest `json:"recent"`
}

type HistoryRecordView struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Solution    string    `json:"solution"`
	Success     bool      `json:"success"`
	Timestamp   time.Time `json:"timestamp"`
	Steps       []string  `json:"steps"`
}

type TimelineEvent struct {
	Time    time.Time `json:"time"`
	Source  string    `json:"source"`
	Level   string    `json:"level"`
	Message string    `json:"message"`
}

type DocumentLibrary struct {
	Total  int               `json:"total"`
	ByKind map[string]int    `json:"by_kind"`
	Items  []DocumentSummary `json:"items"`
}

type DocumentSummary struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"`
	Title    string   `json:"title"`
	Path     string   `json:"path"`
	Preview  string   `json:"preview"`
	Commands []string `json:"commands"`
}

func NewCollector(cfg *config.Config, monitor healthChecker, obs *observability.GlobalCallback, historyKB *rag.HistoryKnowledgeBase) *Collector {
	return &Collector{cfg: cfg, monitor: monitor, obs: obs, historyKB: historyKB, skillsPath: "./skills"}
}

func NewCollectorWithRunner(cfg *config.Config, monitor healthChecker, obs *observability.GlobalCallback, historyKB *rag.HistoryKnowledgeBase, runner graphRunner) *Collector {
	collector := NewCollector(cfg, monitor, obs, historyKB)
	collector.runner = runner
	return collector
}

func (c *Collector) Snapshot(ctx context.Context) (*Snapshot, error) {
	return c.snapshot(ctx, nil)
}

func (c *Collector) TriggerCheck(ctx context.Context) (*Snapshot, error) {
	if c.runner != nil {
		state, err := c.runner.Run(ctx, orchestration.TriggerManualCheck)
		if err != nil {
			return nil, err
		}
		if state != nil {
			return c.snapshot(ctx, state.Report)
		}
		return c.snapshot(ctx, nil)
	}
	if c.monitor == nil {
		return c.snapshot(ctx, nil)
	}
	report, err := c.monitor.CheckHealth(ctx)
	if err != nil {
		return nil, err
	}
	if !report.IsHealthy {
		if trigger, ok := c.monitor.(anomalyTrigger); ok {
			if err := trigger.NotifyAnomaly(ctx, trigger.BuildAnomaly(report)); err != nil {
				return nil, err
			}
		}
	}
	return c.snapshot(ctx, report)
}

func (c *Collector) snapshot(ctx context.Context, report *monitor.HealthReport) (*Snapshot, error) {
	now := time.Now().UTC()
	sessionStart := detectCurrentRunStart(c.cfg.Storage.LogPath)
	snapshot := &Snapshot{
		GeneratedAt: now,
		System: SystemOverview{
			ManagedServices: len(c.cfg.Services),
			Config: map[string]string{
				"check_interval": c.cfg.Monitor.CheckInterval.String(),
				"trace_log":      c.cfg.Observability.TraceLogPath,
				"history_path":   c.cfg.Storage.HistoryPath,
				"runs_path":      c.cfg.Storage.RunsPath,
				"approvals_path": c.cfg.Storage.ApprovalsPath,
			},
			Collected: make(map[string]MetricView),
		},
		Agents: []AgentRuntime{
			{Name: "Eino.Graph", Role: "单图编排运行", Status: "standby"},
			{Name: "Eino.Lambda", Role: "巡检、路由、检索、验证节点", Status: "standby"},
			{Name: "Eino.ChatModel", Role: "模型推理节点", Status: "standby"},
			{Name: "Eino.Tools", Role: "受控工具调用节点", Status: "standby"},
		},
	}

	if c.monitor != nil {
		if report == nil {
			var err error
			report, err = c.monitor.CheckHealth(ctx)
			if err != nil {
				snapshot.Timeline = append(snapshot.Timeline, TimelineEvent{
					Time: now, Source: "monitor", Level: "error", Message: err.Error(),
				})
			}
		}
		if report != nil {
			snapshot.System.HealthScore = report.Score
			snapshot.System.IsHealthy = report.IsHealthy
			snapshot.System.Components = componentViews(report.Components)
			snapshot.System.Collected = collectedMetrics(report.Components)
		}
	}

	snapshot.Tools = c.collectTools(sessionStart)
	snapshot.Agents = enrichAgents(snapshot.Agents, snapshot.Tools.Recent)
	snapshot.Logs = readLogs(c.cfg.Storage.LogPath, sessionStart)
	snapshot.History = c.collectHistory(ctx, sessionStart)
	snapshot.Runs = c.collectRuns(ctx)
	snapshot.Approvals = c.collectApprovals(ctx)
	snapshot.Documents = c.collectDocuments()
	snapshot.Timeline = mergeTimeline(snapshot.Timeline, snapshot.Tools.Recent, snapshot.Logs.Recent, snapshot.History.Recent)
	normalizeSnapshot(snapshot)

	return snapshot, nil
}

func (s *Snapshot) AgentByName(name string) AgentRuntime {
	for _, agent := range s.Agents {
		if agent.Name == name {
			return agent
		}
	}
	return AgentRuntime{}
}

func componentViews(components map[string]monitor.ComponentStatus) []ComponentView {
	views := make([]ComponentView, 0, len(components))
	for _, component := range components {
		views = append(views, ComponentView{
			Name:    component.Name,
			Status:  component.Status,
			Message: component.Message,
			Metrics: component.Metrics,
		})
	}
	sort.Slice(views, func(i, j int) bool { return views[i].Name < views[j].Name })
	return views
}

func collectedMetrics(components map[string]monitor.ComponentStatus) map[string]MetricView {
	metrics := make(map[string]MetricView)
	addUsage := func(component string, label string) {
		if status, ok := components[component]; ok {
			if value, ok := numeric(status.Metrics["usage"]); ok {
				metrics[component] = MetricView{Label: label, Value: value, Unit: "%"}
			}
		}
	}
	addUsage("cpu", "CPU")
	addUsage("memory", "内存")
	addUsage("disk", "磁盘")
	return metrics
}

func (c *Collector) collectTools(since time.Time) ToolSummary {
	byID := make(map[string]ToolCall)
	if c.obs != nil {
		for _, record := range c.obs.GetAllCallbacks() {
			if !since.IsZero() && record.StartTime.Before(since) {
				continue
			}
			byID[record.ID] = ToolCall{
				ID:        record.ID,
				Name:      callbackName(record.ID),
				Status:    record.Status,
				StartedAt: record.StartTime,
				Summary:   summarizeCallback(record),
				Data:      cloneMap(record.Data),
			}
			call := byID[record.ID]
			if !record.EndTime.IsZero() {
				call.DurationMillis = record.EndTime.Sub(record.StartTime).Milliseconds()
			}
			if record.Error != nil {
				call.Error = record.Error.Error()
			}
			byID[record.ID] = call
		}
	}
	for _, call := range readTraceToolCalls(c.cfg.Observability.TraceLogPath, since) {
		if existing, ok := byID[call.ID]; ok && !existing.StartedAt.IsZero() {
			continue
		}
		byID[call.ID] = call
	}

	calls := make([]ToolCall, 0, len(byID))
	summary := ToolSummary{Total: len(byID)}
	for _, call := range byID {
		if call.Status == "error" {
			summary.Errors++
		}
		calls = append(calls, call)
	}
	sort.Slice(calls, func(i, j int) bool { return calls[i].StartedAt.After(calls[j].StartedAt) })
	summary.Recent = limitToolCalls(calls)
	return summary
}

func readTraceToolCalls(path string, since time.Time) []ToolCall {
	if path == "" {
		return nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	calls := make(map[string]ToolCall)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event TraceEventView
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}
		id, _ := event.Payload["id"].(string)
		if id == "" {
			continue
		}
		call := calls[id]
		if call.ID == "" {
			name, _ := event.Payload["name"].(string)
			if name == "" {
				name = callbackName(id)
			}
			call = ToolCall{
				ID:        id,
				Name:      name,
				Status:    "started",
				StartedAt: event.Timestamp,
				Summary:   name + ": started",
				Data:      make(map[string]interface{}),
			}
		}
		call.Events = append(call.Events, event)
		switch event.Type {
		case "callback_started":
			name, _ := event.Payload["name"].(string)
			if name == "" {
				name = callbackName(id)
			}
			call.Name = name
			call.Status = "started"
			call.StartedAt = event.Timestamp
			call.Summary = name + ": started"
		case "callback_completed":
			call.Status = "completed"
			if !call.StartedAt.IsZero() {
				call.DurationMillis = event.Timestamp.Sub(call.StartedAt).Milliseconds()
			}
			if data, ok := event.Payload["data"].(map[string]interface{}); ok {
				call.Data = data
			}
			call.Summary = call.Name + ": completed"
		case "callback_error":
			call.Status = "error"
			if !call.StartedAt.IsZero() {
				call.DurationMillis = event.Timestamp.Sub(call.StartedAt).Milliseconds()
			}
			errMsg, _ := event.Payload["error"].(string)
			call.Error = errMsg
			call.Summary = call.Name + ": " + errMsg
		}
		if !since.IsZero() && !call.StartedAt.IsZero() && call.StartedAt.Before(since) {
			continue
		}
		calls[id] = call
	}
	result := make([]ToolCall, 0, len(calls))
	for _, call := range calls {
		result = append(result, call)
	}
	return result
}

func (c *Collector) collectDocuments() DocumentLibrary {
	library := DocumentLibrary{
		ByKind: make(map[string]int),
		Items:  make([]DocumentSummary, 0),
	}
	library.Items = append(library.Items, scanMarkdownDocs("sop", c.cfg.KnowledgeBase.DocsPath)...)
	library.Items = append(library.Items, scanMarkdownDocs("skill", c.skillsPath)...)
	sort.Slice(library.Items, func(i, j int) bool {
		if library.Items[i].Kind == library.Items[j].Kind {
			return library.Items[i].Title < library.Items[j].Title
		}
		return library.Items[i].Kind < library.Items[j].Kind
	})
	library.Total = len(library.Items)
	for _, item := range library.Items {
		library.ByKind[item.Kind]++
	}
	return library
}

func scanMarkdownDocs(kind, root string) []DocumentSummary {
	if root == "" {
		return nil
	}
	var docs []DocumentSummary
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)
		idPath := path
		if rel, err := filepath.Rel(root, path); err == nil {
			idPath = rel
		}
		docs = append(docs, DocumentSummary{
			ID:       kind + ":" + filepath.ToSlash(idPath),
			Kind:     kind,
			Title:    markdownTitle(content, filepath.Base(path)),
			Path:     path,
			Preview:  previewText(content, 220),
			Commands: extractCodeCommands(content),
		})
		return nil
	})
	return docs
}

func enrichAgents(agents []AgentRuntime, calls []ToolCall) []AgentRuntime {
	for i := range agents {
		for _, call := range calls {
			if call.Name != agents[i].Name && !strings.HasPrefix(call.Name, agents[i].Name+".") {
				continue
			}
			agents[i].Runs++
			if call.Status == "error" {
				agents[i].Errors++
			}
			if call.StartedAt.After(agents[i].LastSeen) {
				agents[i].LastSeen = call.StartedAt
				agents[i].LastEvent = call.Summary
				agents[i].Status = agentStatus(call.Status)
			}
		}
	}
	return agents
}

func (c *Collector) collectHistory(ctx context.Context, since time.Time) HistorySummary {
	if c.historyKB == nil {
		return HistorySummary{}
	}
	records, err := c.historyKB.ListAll(ctx)
	if err != nil {
		return HistorySummary{}
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Timestamp.After(records[j].Timestamp) })
	summary := HistorySummary{}
	for _, record := range records {
		if !since.IsZero() && record.Timestamp.Before(since) {
			continue
		}
		summary.Total++
		if record.Success {
			summary.Success++
		} else {
			summary.Failed++
		}
		if len(summary.Recent) < maxRecentItems {
			summary.Recent = append(summary.Recent, HistoryRecordView{
				ID:          record.ID,
				Description: record.Description,
				Solution:    record.Solution,
				Success:     record.Success,
				Timestamp:   record.Timestamp,
				Steps:       append([]string(nil), record.Steps...),
			})
		}
	}
	return summary
}

func (c *Collector) collectRuns(ctx context.Context) RunSummary {
	var records []orchestration.RunRecord
	if lister, ok := c.runner.(runLister); ok {
		listed, err := lister.ListRuns(ctx, maxRecentItems)
		if err == nil {
			records = listed
		}
	} else if c.cfg != nil && c.cfg.Storage.RunsPath != "" {
		store, err := orchestration.NewRunStore(c.cfg.Storage.RunsPath)
		if err == nil {
			records, _ = store.List(ctx, maxRecentItems)
		}
	}
	summary := RunSummary{Total: len(records)}
	for _, record := range records {
		if record.Status == orchestration.RunStatusRunning {
			summary.Running++
		}
		if record.Status == orchestration.RunStatusFailed {
			summary.Failed++
		}
	}
	summary.Recent = records
	return summary
}

func (c *Collector) Runs(ctx context.Context) (RunSummary, error) {
	return c.collectRuns(ctx), nil
}

func (c *Collector) Run(ctx context.Context, runID string) (orchestration.RunRecord, bool, error) {
	if lister, ok := c.runner.(runLister); ok {
		return lister.GetRun(ctx, runID)
	}
	if c.cfg != nil && c.cfg.Storage.RunsPath != "" {
		store, err := orchestration.NewRunStore(c.cfg.Storage.RunsPath)
		if err != nil {
			return orchestration.RunRecord{}, false, err
		}
		return store.Get(ctx, runID)
	}
	return orchestration.RunRecord{}, false, nil
}

func (c *Collector) collectApprovals(ctx context.Context) ApprovalSummary {
	store, err := c.approvalStore()
	if err != nil || store == nil {
		return ApprovalSummary{}
	}
	items, err := store.List(ctx, maxRecentItems)
	if err != nil {
		return ApprovalSummary{}
	}
	summary := ApprovalSummary{Total: len(items), Recent: items}
	for _, item := range items {
		if item.Status == security.ApprovalPending {
			summary.Pending++
		}
	}
	return summary
}

func (c *Collector) Approvals(ctx context.Context) (ApprovalSummary, error) {
	return c.collectApprovals(ctx), nil
}

func (c *Collector) DecideApproval(ctx context.Context, id string, approved bool, actor string) (security.ApprovalRequest, error) {
	store, err := c.approvalStore()
	if err != nil {
		return security.ApprovalRequest{}, err
	}
	return store.Decide(ctx, id, approved, actor)
}

func (c *Collector) approvalStore() (*security.ApprovalStore, error) {
	if c.cfg == nil || c.cfg.Storage.ApprovalsPath == "" {
		return nil, nil
	}
	return security.NewApprovalStore(c.cfg.Storage.ApprovalsPath)
}

func readLogs(path string, since time.Time) LogSummary {
	if path == "" {
		return LogSummary{}
	}
	file, err := os.Open(path)
	if err != nil {
		return LogSummary{}
	}
	defer file.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		ts := parseLogTimestamp(line)
		if !since.IsZero() && !ts.IsZero() && ts.Before(since) {
			continue
		}
		level := classifyLog(line)
		entries = append(entries, LogEntry{Level: level, Message: line, Timestamp: ts})
	}
	summary := LogSummary{Total: len(entries)}
	for _, entry := range entries {
		switch entry.Level {
		case "error":
			summary.Errors++
		case "warning":
			summary.Warnings++
		}
	}
	for i := len(entries) - 1; i >= 0 && len(summary.Recent) < maxRecentItems; i-- {
		summary.Recent = append(summary.Recent, entries[i])
	}
	return summary
}

func mergeTimeline(existing []TimelineEvent, tools []ToolCall, logs []LogEntry, history []HistoryRecordView) []TimelineEvent {
	timeline := append([]TimelineEvent(nil), existing...)
	for _, tool := range tools {
		timeline = append(timeline, TimelineEvent{
			Time:    tool.StartedAt,
			Source:  "tool",
			Level:   tool.Status,
			Message: tool.Summary,
		})
	}
	for _, logEntry := range logs {
		timeline = append(timeline, TimelineEvent{
			Time:    logEntry.Timestamp,
			Source:  "log",
			Level:   logEntry.Level,
			Message: logEntry.Message,
		})
	}
	for _, record := range history {
		timeline = append(timeline, TimelineEvent{
			Time:    record.Timestamp,
			Source:  "history",
			Level:   successLevel(record.Success),
			Message: record.Solution,
		})
	}
	sort.SliceStable(timeline, func(i, j int) bool {
		return timeline[i].Time.After(timeline[j].Time)
	})
	if len(timeline) > maxRecentItems {
		return timeline[:maxRecentItems]
	}
	return timeline
}

func detectCurrentRunStart(path string) time.Time {
	if path == "" {
		return time.Time{}
	}
	file, err := os.Open(path)
	if err != nil {
		return time.Time{}
	}
	defer file.Close()

	var latest time.Time
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.Contains(line, "SysGuard started successfully") {
			continue
		}
		if ts := parseLogTimestamp(line); ts.After(latest) {
			latest = ts
		}
	}
	return latest
}

func parseLogTimestamp(line string) time.Time {
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 2 {
		return time.Time{}
	}
	prefix := parts[0] + " " + parts[1]
	for _, layout := range []string{"2006/01/02 15:04:05.000000", "2006/01/02 15:04:05"} {
		if ts, err := time.ParseInLocation(layout, prefix, time.UTC); err == nil {
			return ts
		}
	}
	return time.Time{}
}

func classifyLog(line string) string {
	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "error") || strings.Contains(lower, "failed"):
		return "error"
	case strings.Contains(lower, "warning") || strings.Contains(lower, "warn"):
		return "warning"
	default:
		return "info"
	}
}

func markdownTitle(content, fallback string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
		}
		if strings.HasPrefix(trimmed, "name:") {
			return strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "name:")), `"'`)
		}
	}
	return fallback
}

func previewText(content string, limit int) string {
	lines := make([]string, 0)
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "---") || strings.HasPrefix(trimmed, "```") {
			continue
		}
		lines = append(lines, strings.TrimPrefix(trimmed, "# "))
	}
	preview := strings.Join(lines, " ")
	if len(preview) > limit {
		return preview[:limit] + "..."
	}
	return preview
}

func extractCodeCommands(content string) []string {
	var commands []string
	inCode := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCode = !inCode
			continue
		}
		if inCode && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			commands = append(commands, trimmed)
		}
	}
	return commands
}

func callbackName(id string) string {
	if idx := strings.LastIndex(id, "-"); idx > 0 {
		return id[:idx]
	}
	return id
}

func summarizeCallback(record *observability.CallbackRecord) string {
	name := callbackName(record.ID)
	if record.Error != nil {
		return name + ": " + record.Error.Error()
	}
	if score, ok := record.Data["score"]; ok {
		data, _ := json.Marshal(score)
		return name + ": score=" + string(data)
	}
	if plan, ok := record.Data["plan"]; ok {
		return name + ": plan=" + toString(plan)
	}
	return name + ": " + record.Status
}

func agentStatus(callbackStatus string) string {
	switch callbackStatus {
	case "completed":
		return "healthy"
	case "error":
		return "error"
	case "started":
		return "running"
	default:
		return "standby"
	}
}

func successLevel(success bool) string {
	if success {
		return "completed"
	}
	return "error"
}

func numeric(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case float64:
		return v, true
	case float32:
		return float64(v), true
	default:
		return 0, false
	}
}

func toString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		data, _ := json.Marshal(v)
		return string(data)
	}
}

func cloneMap(input map[string]interface{}) map[string]interface{} {
	if len(input) == 0 {
		return map[string]interface{}{}
	}
	output := make(map[string]interface{}, len(input))
	for k, v := range input {
		output[k] = v
	}
	return output
}

func limitToolCalls(calls []ToolCall) []ToolCall {
	if len(calls) > maxRecentItems {
		return calls[:maxRecentItems]
	}
	return calls
}

func normalizeSnapshot(snapshot *Snapshot) {
	if snapshot.System.Components == nil {
		snapshot.System.Components = []ComponentView{}
	}
	if snapshot.Agents == nil {
		snapshot.Agents = []AgentRuntime{}
	}
	if snapshot.Tools.Recent == nil {
		snapshot.Tools.Recent = []ToolCall{}
	}
	if snapshot.Logs.Recent == nil {
		snapshot.Logs.Recent = []LogEntry{}
	}
	if snapshot.History.Recent == nil {
		snapshot.History.Recent = []HistoryRecordView{}
	}
	if snapshot.Runs.Recent == nil {
		snapshot.Runs.Recent = []orchestration.RunRecord{}
	}
	if snapshot.Approvals.Recent == nil {
		snapshot.Approvals.Recent = []security.ApprovalRequest{}
	}
	if snapshot.Documents.ByKind == nil {
		snapshot.Documents.ByKind = map[string]int{}
	}
	if snapshot.Documents.Items == nil {
		snapshot.Documents.Items = []DocumentSummary{}
	}
	if snapshot.Timeline == nil {
		snapshot.Timeline = []TimelineEvent{}
	}
}
