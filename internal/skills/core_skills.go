package skills

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/sysguard/sysguard/internal/config"
	"github.com/sysguard/sysguard/internal/monitor"
	"github.com/sysguard/sysguard/internal/security"
	"github.com/sysguard/sysguard/internal/workflow"
	"github.com/sysguard/sysguard/pkg/utils"
)

type CoreSkillDependencies struct {
	Config      *config.Config
	Monitor     *monitor.Monitor
	Interceptor *security.CommandInterceptor
	HTTPClient  *http.Client
}

func RegisterCoreSkills(registry *SkillRegistry, deps CoreSkillDependencies) error {
	if registry == nil {
		return fmt.Errorf("registry is required")
	}
	cfg := deps.Config
	if cfg == nil {
		cfg = config.Default()
	}
	mon := deps.Monitor
	if mon == nil {
		mon = monitor.NewMonitor(cfg, deps.Interceptor, nil)
	}
	interceptor := deps.Interceptor
	if interceptor == nil {
		interceptor = security.NewCommandInterceptor(cfg.Security.DangerousCommands)
	}

	coreSkills := []Skill{
		NewLogAnalysisSkill(),
		NewHealthCheckSkill(mon),
		NewServiceManagementSkill(cfg, interceptor),
		NewAlertingSkill(),
		NewMetricsCollectionSkill(mon),
		NewNetworkDiagnosisSkill(),
		NewDatabaseOperationSkill(),
		NewFileOperationSkill(),
		NewNotificationSkill(deps.HTTPClient),
	}
	for _, skill := range coreSkills {
		if err := registry.Register(skill); err != nil {
			return err
		}
	}
	return nil
}

type LogAnalysisSkill struct{}

func NewLogAnalysisSkill() *LogAnalysisSkill { return &LogAnalysisSkill{} }

func (s *LogAnalysisSkill) Name() string { return "log-analysis" }

func (s *LogAnalysisSkill) Description() string {
	return "Analyze log files by chunking lines and filtering relevant keywords"
}

func (s *LogAnalysisSkill) Execute(ctx context.Context, input *SkillInput) (*SkillOutput, error) {
	params := normalizeParams(input)
	path, err := requiredString(params, "path")
	if err != nil {
		return nil, err
	}
	chunkSize := intParam(params, "chunk_size", 1000)
	if chunkSize <= 0 {
		return nil, fmt.Errorf("chunk_size must be positive")
	}
	keywords := stringSliceParam(params, "keywords", []string{"error", "failed", "warning", "critical", "exception", "timeout"})

	graph := workflow.NewLogAnalysisGraph(chunkSize, keywords)
	result, err := graph.Analyze(ctx, path)
	if err != nil {
		return nil, err
	}
	return successOutput(result), nil
}

type HealthCheckSkill struct {
	monitor *monitor.Monitor
}

func NewHealthCheckSkill(mon *monitor.Monitor) *HealthCheckSkill {
	return &HealthCheckSkill{monitor: mon}
}

func (s *HealthCheckSkill) Name() string { return "health-check" }

func (s *HealthCheckSkill) Description() string {
	return "Run the configured host and service health checks"
}

func (s *HealthCheckSkill) Execute(ctx context.Context, input *SkillInput) (*SkillOutput, error) {
	if s.monitor == nil {
		return nil, fmt.Errorf("monitor dependency is required")
	}
	report, err := s.monitor.CheckHealth(ctx)
	if err != nil {
		return nil, err
	}
	return successOutput(report), nil
}

type ServiceManagementSkill struct {
	cfg         *config.Config
	interceptor *security.CommandInterceptor
	executor    *utils.ShellExecutor
}

func NewServiceManagementSkill(cfg *config.Config, interceptor *security.CommandInterceptor) *ServiceManagementSkill {
	timeout := 2 * time.Minute
	if cfg != nil && cfg.Execution.CommandTimeout > 0 {
		timeout = cfg.Execution.CommandTimeout
	}
	return &ServiceManagementSkill{
		cfg:         cfg,
		interceptor: interceptor,
		executor:    utils.NewShellExecutor(timeout),
	}
}

func (s *ServiceManagementSkill) Name() string { return "service-management" }

func (s *ServiceManagementSkill) Description() string {
	return "Manage services with explicit status, start, stop, restart, and logs operations"
}

func (s *ServiceManagementSkill) Execute(ctx context.Context, input *SkillInput) (*SkillOutput, error) {
	params := normalizeParams(input)
	service, err := requiredString(params, "service")
	if err != nil {
		return nil, err
	}
	operation := stringParam(params, "operation", "status")

	var command string
	if runtime.GOOS == "linux" {
		switch operation {
		case "status":
			command = fmt.Sprintf("systemctl status %s", service)
		case "start", "stop", "restart":
			command = fmt.Sprintf("systemctl %s %s", operation, service)
		case "logs":
			lines := intParam(params, "lines", 100)
			command = fmt.Sprintf("journalctl -u %s -n %d --no-pager", service, lines)
		default:
			return nil, fmt.Errorf("unsupported service operation %q", operation)
		}
	} else {
		if operation != "status" {
			return nil, fmt.Errorf("operation %q is only supported on linux", operation)
		}
		command = fmt.Sprintf("pgrep -x %s", service)
	}

	result, err := executeManagedCommand(ctx, s.executor, s.interceptor, command, boolParam(params, "allow_dangerous", false))
	if err != nil {
		return nil, err
	}
	return successOutput(result), nil
}

type Alert struct {
	ID        string            `json:"id"`
	Severity  string            `json:"severity"`
	Title     string            `json:"title"`
	Message   string            `json:"message"`
	Source    string            `json:"source"`
	Metadata  map[string]string `json:"metadata"`
	Timestamp time.Time         `json:"timestamp"`
}

type AlertingSkill struct{}

func NewAlertingSkill() *AlertingSkill { return &AlertingSkill{} }

func (s *AlertingSkill) Name() string { return "alerting" }

func (s *AlertingSkill) Description() string {
	return "Create structured alerts from incidents or skill outputs"
}

func (s *AlertingSkill) Execute(ctx context.Context, input *SkillInput) (*SkillOutput, error) {
	params := normalizeParams(input)
	title, err := requiredString(params, "title")
	if err != nil {
		return nil, err
	}
	message, err := requiredString(params, "message")
	if err != nil {
		return nil, err
	}
	severity := stringParam(params, "severity", "warning")
	source := stringParam(params, "source", "sysguard")
	alert := Alert{
		ID:        fmt.Sprintf("alert-%d", time.Now().UnixNano()),
		Severity:  severity,
		Title:     title,
		Message:   message,
		Source:    source,
		Metadata:  stringMapParam(params, "metadata"),
		Timestamp: time.Now().UTC(),
	}
	return successOutput(alert), nil
}

type MetricsCollectionSkill struct {
	monitor *monitor.Monitor
}

func NewMetricsCollectionSkill(mon *monitor.Monitor) *MetricsCollectionSkill {
	return &MetricsCollectionSkill{monitor: mon}
}

func (s *MetricsCollectionSkill) Name() string { return "metrics-collection" }

func (s *MetricsCollectionSkill) Description() string {
	return "Collect host and service metrics from the health monitor"
}

func (s *MetricsCollectionSkill) Execute(ctx context.Context, input *SkillInput) (*SkillOutput, error) {
	if s.monitor == nil {
		return nil, fmt.Errorf("monitor dependency is required")
	}
	report, err := s.monitor.CheckHealth(ctx)
	if err != nil {
		return nil, err
	}
	return successOutput(map[string]interface{}{
		"timestamp":  report.Timestamp,
		"score":      report.Score,
		"is_healthy": report.IsHealthy,
		"components": report.Components,
	}), nil
}

type NetworkDiagnosisSkill struct{}

func NewNetworkDiagnosisSkill() *NetworkDiagnosisSkill { return &NetworkDiagnosisSkill{} }

func (s *NetworkDiagnosisSkill) Name() string { return "network-diagnosis" }

func (s *NetworkDiagnosisSkill) Description() string {
	return "Run DNS, TCP, interface, and ping network diagnostics"
}

func (s *NetworkDiagnosisSkill) Execute(ctx context.Context, input *SkillInput) (*SkillOutput, error) {
	params := normalizeParams(input)
	operation := stringParam(params, "operation", "interfaces")
	switch operation {
	case "interfaces":
		interfaces, err := net.Interfaces()
		if err != nil {
			return nil, err
		}
		items := make([]map[string]interface{}, 0, len(interfaces))
		for _, iface := range interfaces {
			items = append(items, map[string]interface{}{
				"name":  iface.Name,
				"flags": iface.Flags.String(),
				"mtu":   iface.MTU,
			})
		}
		return successOutput(map[string]interface{}{"interfaces": items}), nil
	case "dns":
		host, err := requiredString(params, "host")
		if err != nil {
			return nil, err
		}
		addrs, err := net.DefaultResolver.LookupHost(ctx, host)
		if err != nil {
			return nil, err
		}
		return successOutput(map[string]interface{}{"host": host, "addresses": addrs}), nil
	case "tcp":
		host, err := requiredString(params, "host")
		if err != nil {
			return nil, err
		}
		port := intParam(params, "port", 0)
		if port <= 0 {
			return nil, fmt.Errorf("port must be positive")
		}
		timeout := durationParam(params, "timeout", 5*time.Second)
		dialer := net.Dialer{Timeout: timeout}
		conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err != nil {
			return successOutput(map[string]interface{}{"host": host, "port": port, "reachable": false, "error": err.Error()}), nil
		}
		_ = conn.Close()
		return successOutput(map[string]interface{}{"host": host, "port": port, "reachable": true}), nil
	case "ping":
		host, err := requiredString(params, "host")
		if err != nil {
			return nil, err
		}
		command := "ping -c 3 " + host
		if runtime.GOOS == "windows" {
			command = "ping -n 3 " + host
		}
		result, err := utils.NewShellExecutor(durationParam(params, "timeout", 10*time.Second)).Execute(ctx, command)
		if err != nil {
			return successOutput(map[string]interface{}{"host": host, "reachable": false, "stderr": safeStderr(result)}), nil
		}
		return successOutput(map[string]interface{}{"host": host, "reachable": true, "stdout": result.Stdout}), nil
	default:
		return nil, fmt.Errorf("unsupported network operation %q", operation)
	}
}

type DatabaseOperationSkill struct{}

func NewDatabaseOperationSkill() *DatabaseOperationSkill { return &DatabaseOperationSkill{} }

func (s *DatabaseOperationSkill) Name() string { return "database-operation" }

func (s *DatabaseOperationSkill) Description() string {
	return "Run safe database ping and read-only query operations through database/sql"
}

func (s *DatabaseOperationSkill) Execute(ctx context.Context, input *SkillInput) (*SkillOutput, error) {
	params := normalizeParams(input)
	driver, err := requiredString(params, "driver")
	if err != nil {
		return nil, err
	}
	dsn, err := requiredString(params, "dsn")
	if err != nil {
		return nil, err
	}
	operation := stringParam(params, "operation", "ping")

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	switch operation {
	case "ping":
		if err := db.PingContext(ctx); err != nil {
			return nil, err
		}
		return successOutput(map[string]interface{}{"operation": "ping", "ok": true}), nil
	case "query":
		query, err := requiredString(params, "query")
		if err != nil {
			return nil, err
		}
		if !isReadOnlySQL(query) {
			return nil, fmt.Errorf("only read-only SELECT/WITH queries are allowed")
		}
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		result, err := collectRows(rows, intParam(params, "limit", 100))
		if err != nil {
			return nil, err
		}
		return successOutput(result), nil
	default:
		return nil, fmt.Errorf("unsupported database operation %q", operation)
	}
}

type FileOperationSkill struct{}

func NewFileOperationSkill() *FileOperationSkill { return &FileOperationSkill{} }

func (s *FileOperationSkill) Name() string { return "file-operation" }

func (s *FileOperationSkill) Description() string {
	return "Perform safe file read, stat, list, and tail operations"
}

func (s *FileOperationSkill) Execute(ctx context.Context, input *SkillInput) (*SkillOutput, error) {
	params := normalizeParams(input)
	operation := stringParam(params, "operation", "stat")
	path, err := requiredString(params, "path")
	if err != nil {
		return nil, err
	}
	cleanPath := filepath.Clean(path)

	switch operation {
	case "read":
		data, err := os.ReadFile(cleanPath)
		if err != nil {
			return nil, err
		}
		return successOutput(map[string]interface{}{"path": cleanPath, "content": string(data)}), nil
	case "stat":
		info, err := os.Stat(cleanPath)
		if err != nil {
			return nil, err
		}
		return successOutput(map[string]interface{}{
			"path":     cleanPath,
			"name":     info.Name(),
			"size":     info.Size(),
			"mode":     info.Mode().String(),
			"is_dir":   info.IsDir(),
			"modified": info.ModTime(),
		}), nil
	case "list":
		entries, err := os.ReadDir(cleanPath)
		if err != nil {
			return nil, err
		}
		items := make([]map[string]interface{}, 0, len(entries))
		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				return nil, err
			}
			items = append(items, map[string]interface{}{
				"name":   entry.Name(),
				"is_dir": entry.IsDir(),
				"size":   info.Size(),
				"mode":   info.Mode().String(),
			})
		}
		return successOutput(map[string]interface{}{"path": cleanPath, "entries": items}), nil
	case "tail":
		lines := intParam(params, "lines", 100)
		if lines <= 0 {
			return nil, fmt.Errorf("lines must be positive")
		}
		content, err := tailFile(cleanPath, lines)
		if err != nil {
			return nil, err
		}
		return successOutput(map[string]interface{}{"path": cleanPath, "content": content}), nil
	default:
		return nil, fmt.Errorf("unsupported file operation %q", operation)
	}
}

type Notification struct {
	Channel   string            `json:"channel"`
	Target    string            `json:"target"`
	Message   string            `json:"message"`
	Metadata  map[string]string `json:"metadata"`
	Sent      bool              `json:"sent"`
	Timestamp time.Time         `json:"timestamp"`
}

type NotificationSkill struct {
	client *http.Client
}

func NewNotificationSkill(client *http.Client) *NotificationSkill {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &NotificationSkill{client: client}
}

func (s *NotificationSkill) Name() string { return "notification" }

func (s *NotificationSkill) Description() string {
	return "Send notifications to stdout, log-style output, or webhook endpoints"
}

func (s *NotificationSkill) Execute(ctx context.Context, input *SkillInput) (*SkillOutput, error) {
	params := normalizeParams(input)
	message, err := requiredString(params, "message")
	if err != nil {
		return nil, err
	}
	channel := stringParam(params, "channel", "stdout")
	target := stringParam(params, "target", "")
	notification := Notification{
		Channel:   channel,
		Target:    target,
		Message:   message,
		Metadata:  stringMapParam(params, "metadata"),
		Timestamp: time.Now().UTC(),
	}

	switch channel {
	case "stdout", "log":
		fmt.Println(message)
		notification.Sent = true
	case "webhook":
		if target == "" {
			return nil, fmt.Errorf("target is required for webhook notifications")
		}
		body, err := json.Marshal(notification)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := s.client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("webhook returned status %d", resp.StatusCode)
		}
		notification.Sent = true
	default:
		return nil, fmt.Errorf("unsupported notification channel %q", channel)
	}
	return successOutput(notification), nil
}

func executeManagedCommand(ctx context.Context, executor *utils.ShellExecutor, interceptor *security.CommandInterceptor, command string, allowDangerous bool) (*utils.ExecutionResult, error) {
	if executor == nil {
		executor = utils.NewShellExecutor(2 * time.Minute)
	}
	validator := utils.NewDefaultValidator()
	if err := validator.Validate(command); err != nil {
		return nil, err
	}
	if interceptor != nil && interceptor.IsDangerous(command) && !allowDangerous {
		return nil, fmt.Errorf("dangerous command requires explicit allow_dangerous=true: %s", command)
	}
	return executor.Execute(ctx, command)
}

func successOutput(result interface{}) *SkillOutput {
	return &SkillOutput{
		Result:   result,
		Success:  true,
		Metadata: map[string]string{},
	}
}

func normalizeParams(input *SkillInput) map[string]interface{} {
	if input == nil || input.Params == nil {
		return map[string]interface{}{}
	}
	return input.Params
}

func requiredString(params map[string]interface{}, key string) (string, error) {
	value := strings.TrimSpace(stringParam(params, key, ""))
	if value == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return value, nil
}

func stringParam(params map[string]interface{}, key, fallback string) string {
	value, ok := params[key]
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func intParam(params map[string]interface{}, key string, fallback int) int {
	value, ok := params[key]
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func boolParam(params map[string]interface{}, key string, fallback bool) bool {
	value, ok := params[key]
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func durationParam(params map[string]interface{}, key string, fallback time.Duration) time.Duration {
	value, ok := params[key]
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case time.Duration:
		return typed
	case string:
		parsed, err := time.ParseDuration(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func stringSliceParam(params map[string]interface{}, key string, fallback []string) []string {
	value, ok := params[key]
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case []string:
		return typed
	case []interface{}:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if s := strings.TrimSpace(fmt.Sprintf("%v", item)); s != "" {
				result = append(result, s)
			}
		}
		if len(result) > 0 {
			return result
		}
	case string:
		parts := strings.Split(typed, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			if s := strings.TrimSpace(part); s != "" {
				result = append(result, s)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return fallback
}

func stringMapParam(params map[string]interface{}, key string) map[string]string {
	value, ok := params[key]
	if !ok || value == nil {
		return map[string]string{}
	}
	switch typed := value.(type) {
	case map[string]string:
		return typed
	case map[string]interface{}:
		result := make(map[string]string, len(typed))
		for k, v := range typed {
			result[k] = fmt.Sprintf("%v", v)
		}
		return result
	default:
		return map[string]string{}
	}
}

func tailFile(path string, lines int) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	ring := make([]string, 0, lines)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if len(ring) == lines {
			copy(ring, ring[1:])
			ring[len(ring)-1] = scanner.Text()
			continue
		}
		ring = append(ring, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return strings.Join(ring, "\n"), nil
}

func safeStderr(result *utils.ExecutionResult) string {
	if result == nil {
		return ""
	}
	return result.Stderr
}

func isReadOnlySQL(query string) bool {
	normalized := strings.TrimSpace(strings.ToLower(query))
	return strings.HasPrefix(normalized, "select ") || strings.HasPrefix(normalized, "with ")
}

func collectRows(rows *sql.Rows, limit int) (map[string]interface{}, error) {
	if limit <= 0 {
		limit = 100
	}
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	items := make([]map[string]interface{}, 0)
	for rows.Next() {
		values := make([]interface{}, len(columns))
		scanTargets := make([]interface{}, len(columns))
		for i := range values {
			scanTargets[i] = &values[i]
		}
		if err := rows.Scan(scanTargets...); err != nil {
			return nil, err
		}
		item := make(map[string]interface{}, len(columns))
		for i, column := range columns {
			if raw, ok := values[i].([]byte); ok {
				item[column] = string(raw)
			} else {
				item[column] = values[i]
			}
		}
		items = append(items, item)
		if len(items) >= limit {
			break
		}
	}
	if err := rows.Err(); err != nil && err != io.EOF {
		return nil, err
	}
	return map[string]interface{}{"columns": columns, "rows": items}, nil
}
