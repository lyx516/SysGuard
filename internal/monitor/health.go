package monitor

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/sysguard/sysguard/internal/config"
)

type Anomaly struct {
	Timestamp   time.Time
	Severity    string
	Description string
	Source      string
	Metadata    map[string]string
}

type HealthReport struct {
	Timestamp  time.Time
	IsHealthy  bool
	Score      float64
	Components map[string]ComponentStatus
}

type ComponentStatus struct {
	Name      string
	Status    string
	Message   string
	Metrics   map[string]interface{}
}

type Monitor struct {
	cfg             *config.Config
	anomalyHandlers []AnomalyHandler
}

type AnomalyHandler func(ctx context.Context, anomaly Anomaly) error

func NewMonitor(cfg *config.Config, _ interface{}, _ interface{}) *Monitor {
	return &Monitor{
		cfg:             cfg,
		anomalyHandlers: make([]AnomalyHandler, 0),
	}
}

func (m *Monitor) CheckHealth(ctx context.Context) (*HealthReport, error) {
	report := &HealthReport{
		Timestamp:  time.Now().UTC(),
		Components: make(map[string]ComponentStatus),
	}

	report.Components["cpu"] = m.checkCPU(ctx)
	report.Components["memory"] = m.checkMemory(ctx)
	report.Components["disk"] = m.checkDisk(ctx)
	report.Components["network"] = m.checkNetwork(ctx)
	report.Components["services"] = m.checkServices(ctx)
	report.Score = m.calculateScore(report.Components)
	report.IsHealthy = report.Score >= m.cfg.Monitor.HealthThreshold
	return report, nil
}

func (m *Monitor) BuildAnomaly(report *HealthReport) Anomaly {
	degraded := make([]string, 0)
	metadata := make(map[string]string)
	severity := "warning"
	for name, component := range report.Components {
		if component.Status == "healthy" {
			continue
		}
		degraded = append(degraded, fmt.Sprintf("%s=%s", name, component.Message))
		if component.Status == "down" {
			severity = "critical"
		}
		if failedService, ok := component.Metrics["failed_service"].(string); ok && failedService != "" {
			metadata["service_name"] = failedService
		}
	}
	return Anomaly{
		Timestamp:   report.Timestamp,
		Severity:    severity,
		Description: strings.Join(degraded, "; "),
		Source:      "monitor",
		Metadata:    metadata,
	}
}

func (m *Monitor) checkCPU(ctx context.Context) ComponentStatus {
	loadAvg, cores, err := readLoadAverage(ctx)
	if err != nil || cores == 0 {
		return degradedStatus("cpu", "unable to collect CPU load", map[string]interface{}{"error": errorString(err)})
	}

	usage := (loadAvg / float64(cores)) * 100
	status := "healthy"
	message := fmt.Sprintf("CPU load %.2f%%", usage)
	if usage >= m.cfg.Monitor.CPUThreshold {
		status = "degraded"
		message = fmt.Sprintf("CPU load high: %.2f%%", usage)
	}

	return ComponentStatus{
		Name:    "cpu",
		Status:  status,
		Message: message,
		Metrics: map[string]interface{}{
			"usage": usage,
			"cores": cores,
			"load1": loadAvg,
		},
	}
}

func (m *Monitor) checkMemory(ctx context.Context) ComponentStatus {
	usage, totalMB, usedMB, err := readMemoryUsage(ctx)
	if err != nil {
		return degradedStatus("memory", "unable to collect memory usage", map[string]interface{}{"error": err.Error()})
	}

	status := "healthy"
	message := fmt.Sprintf("Memory usage %.2f%%", usage)
	if usage >= m.cfg.Monitor.MemoryThreshold {
		status = "degraded"
		message = fmt.Sprintf("Memory usage high: %.2f%%", usage)
	}

	return ComponentStatus{
		Name:    "memory",
		Status:  status,
		Message: message,
		Metrics: map[string]interface{}{
			"usage": usage,
			"total": totalMB,
			"used":  usedMB,
		},
	}
}

func (m *Monitor) checkDisk(ctx context.Context) ComponentStatus {
	usage, totalGB, usedGB, mount, err := readDiskUsage(ctx, "/")
	if err != nil {
		return degradedStatus("disk", "unable to collect disk usage", map[string]interface{}{"error": err.Error()})
	}

	status := "healthy"
	message := fmt.Sprintf("Disk usage %.2f%% on %s", usage, mount)
	if usage >= m.cfg.Monitor.DiskThreshold {
		status = "degraded"
		message = fmt.Sprintf("Disk usage high: %.2f%% on %s", usage, mount)
	}

	return ComponentStatus{
		Name:    "disk",
		Status:  status,
		Message: message,
		Metrics: map[string]interface{}{
			"usage": usage,
			"total": totalGB,
			"used":  usedGB,
			"mount": mount,
		},
	}
}

func (m *Monitor) checkNetwork(ctx context.Context) ComponentStatus {
	interfaces, err := net.Interfaces()
	if err != nil {
		return degradedStatus("network", "unable to enumerate network interfaces", map[string]interface{}{"error": err.Error()})
	}

	active := 0
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagUp != 0 {
			active++
		}
	}
	if active == 0 {
		return ComponentStatus{
			Name:    "network",
			Status:  "down",
			Message: "No active non-loopback network interfaces",
			Metrics: map[string]interface{}{"active_interfaces": 0},
		}
	}

	return ComponentStatus{
		Name:    "network",
		Status:  "healthy",
		Message: fmt.Sprintf("%d active network interface(s)", active),
		Metrics: map[string]interface{}{"active_interfaces": active},
	}
}

func (m *Monitor) checkServices(ctx context.Context) ComponentStatus {
	if len(m.cfg.Services) == 0 {
		return ComponentStatus{
			Name:    "services",
			Status:  "healthy",
			Message: "No managed services configured",
			Metrics: map[string]interface{}{"configured": 0},
		}
	}

	failed := make([]string, 0)
	for _, service := range m.cfg.Services {
		ok, detail := serviceRunning(ctx, service)
		if !ok {
			failed = append(failed, fmt.Sprintf("%s (%s)", service, detail))
		}
	}

	if len(failed) > 0 {
		metrics := map[string]interface{}{
			"configured":     len(m.cfg.Services),
			"failed":         len(failed),
			"failed_service": m.cfg.Services[0],
		}
		if len(failed) > 0 {
			metrics["failed_service"] = strings.Split(failed[0], " ")[0]
		}
		return ComponentStatus{
			Name:    "services",
			Status:  "down",
			Message: "Service check failed: " + strings.Join(failed, ", "),
			Metrics: metrics,
		}
	}

	return ComponentStatus{
		Name:    "services",
		Status:  "healthy",
		Message: fmt.Sprintf("All %d managed service(s) are running", len(m.cfg.Services)),
		Metrics: map[string]interface{}{"configured": len(m.cfg.Services)},
	}
}

func (m *Monitor) calculateScore(components map[string]ComponentStatus) float64 {
	if len(components) == 0 {
		return 0
	}

	total := 0.0
	for _, comp := range components {
		switch comp.Status {
		case "healthy":
			total += 100
		case "degraded":
			total += 60
		default:
			total += 0
		}
	}
	return total / float64(len(components))
}

func (m *Monitor) RegisterAnomalyHandler(handler AnomalyHandler) {
	m.anomalyHandlers = append(m.anomalyHandlers, handler)
}

func (m *Monitor) NotifyAnomaly(ctx context.Context, anomaly Anomaly) error {
	for _, handler := range m.anomalyHandlers {
		if err := handler(ctx, anomaly); err != nil {
			return err
		}
	}
	return nil
}

type Probe interface {
	Execute(ctx context.Context) (*ProbeResult, error)
}

type ProbeResult struct {
	Name      string
	Success   bool
	Message   string
	Value     interface{}
	Timestamp time.Time
}

func degradedStatus(name, message string, metrics map[string]interface{}) ComponentStatus {
	return ComponentStatus{
		Name:    name,
		Status:  "degraded",
		Message: message,
		Metrics: metrics,
	}
}

func readLoadAverage(ctx context.Context) (float64, int, error) {
	if runtime.GOOS == "linux" {
		data, err := os.ReadFile("/proc/loadavg")
		if err != nil {
			return 0, 0, err
		}
		fields := strings.Fields(string(data))
		if len(fields) < 1 {
			return 0, 0, fmt.Errorf("unexpected /proc/loadavg format")
		}
		load, err := strconv.ParseFloat(fields[0], 64)
		if err != nil {
			return 0, 0, err
		}
		return load, runtime.NumCPU(), nil
	}

	out, err := exec.CommandContext(ctx, "sysctl", "-n", "vm.loadavg").Output()
	if err != nil {
		return 0, 0, err
	}
	trimmed := strings.Trim(strings.TrimSpace(string(out)), "{}")
	fields := strings.Fields(trimmed)
	if len(fields) < 1 {
		return 0, 0, fmt.Errorf("unexpected vm.loadavg format")
	}
	load, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, 0, err
	}
	return load, runtime.NumCPU(), nil
}

func readMemoryUsage(ctx context.Context) (usage float64, totalMB float64, usedMB float64, err error) {
	if runtime.GOOS == "linux" {
		file, err := os.Open("/proc/meminfo")
		if err != nil {
			return 0, 0, 0, err
		}
		defer file.Close()

		values := map[string]float64{}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			parts := strings.Split(scanner.Text(), ":")
			if len(parts) != 2 {
				continue
			}
			fields := strings.Fields(strings.TrimSpace(parts[1]))
			if len(fields) == 0 {
				continue
			}
			v, parseErr := strconv.ParseFloat(fields[0], 64)
			if parseErr == nil {
				values[parts[0]] = v
			}
		}
		if err := scanner.Err(); err != nil {
			return 0, 0, 0, err
		}

		totalKB := values["MemTotal"]
		availableKB := values["MemAvailable"]
		if totalKB == 0 {
			return 0, 0, 0, fmt.Errorf("missing MemTotal")
		}
		usedKB := totalKB - availableKB
		return usedKB / totalKB * 100, totalKB / 1024, usedKB / 1024, nil
	}

	pageSizeOut, err := exec.CommandContext(ctx, "sysctl", "-n", "hw.pagesize").Output()
	if err != nil {
		return 0, 0, 0, err
	}
	totalOut, err := exec.CommandContext(ctx, "sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0, 0, 0, err
	}
	vmStatOut, err := exec.CommandContext(ctx, "vm_stat").Output()
	if err != nil {
		return 0, 0, 0, err
	}

	pageSize, err := strconv.ParseFloat(strings.TrimSpace(string(pageSizeOut)), 64)
	if err != nil {
		return 0, 0, 0, err
	}
	totalBytes, err := strconv.ParseFloat(strings.TrimSpace(string(totalOut)), 64)
	if err != nil {
		return 0, 0, 0, err
	}

	var freePages float64
	for _, line := range strings.Split(string(vmStatOut), "\n") {
		if strings.HasPrefix(line, "Pages free:") || strings.HasPrefix(line, "Pages inactive:") || strings.HasPrefix(line, "Pages speculative:") {
			value := strings.Trim(strings.TrimSpace(strings.SplitN(line, ":", 2)[1]), ".")
			pages, parseErr := strconv.ParseFloat(value, 64)
			if parseErr == nil {
				freePages += pages
			}
		}
	}
	freeBytes := freePages * pageSize
	usedBytes := totalBytes - freeBytes
	return usedBytes / totalBytes * 100, totalBytes / (1024 * 1024), usedBytes / (1024 * 1024), nil
}

func readDiskUsage(ctx context.Context, mount string) (usage float64, totalGB float64, usedGB float64, actualMount string, err error) {
	out, err := exec.CommandContext(ctx, "df", "-Pk", mount).Output()
	if err != nil {
		return 0, 0, 0, "", err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return 0, 0, 0, "", fmt.Errorf("unexpected df output")
	}
	fields := strings.Fields(lines[len(lines)-1])
	if len(fields) < 6 {
		return 0, 0, 0, "", fmt.Errorf("unexpected df row")
	}
	totalKB, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return 0, 0, 0, "", err
	}
	usedKB, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return 0, 0, 0, "", err
	}
	pct := strings.TrimSuffix(fields[4], "%")
	usage, err = strconv.ParseFloat(pct, 64)
	if err != nil {
		return 0, 0, 0, "", err
	}
	return usage, totalKB / (1024 * 1024), usedKB / (1024 * 1024), fields[5], nil
}

func serviceRunning(ctx context.Context, service string) (bool, string) {
	if runtime.GOOS == "linux" {
		if err := exec.CommandContext(ctx, "systemctl", "is-active", "--quiet", service).Run(); err == nil {
			return true, "active"
		}
		if err := exec.CommandContext(ctx, "pgrep", "-x", service).Run(); err == nil {
			return true, "process running"
		}
		return false, "systemctl inactive"
	}

	if err := exec.CommandContext(ctx, "pgrep", "-x", service).Run(); err == nil {
		return true, "process running"
	}
	return false, "process not found"
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
