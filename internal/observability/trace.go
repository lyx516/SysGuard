package observability

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// GlobalCallback 全局回调追踪器，记录工具调用链路
type GlobalCallback struct {
	mu           sync.RWMutex
	callbacks    map[string]*CallbackRecord
	probeStates  map[string]ProbeState
	enableTrace  bool
	traceLogPath string
	writeErrors  uint64
}

// CallbackRecord 回调记录
type CallbackRecord struct {
	ID        string
	StartTime time.Time
	EndTime   time.Time
	Status    string // "started", "completed", "error"
	Error     error
	Data      map[string]interface{}
}

// ProbeState 探针状态
type ProbeState struct {
	Name      string
	Status    string // "deployed", "returned", "failed"
	Timestamp time.Time
	Location  string
}

// NewGlobalCallback 创建新的全局回调追踪器
func NewGlobalCallback(enableTrace bool, traceLogPath string) (*GlobalCallback, error) {
	if enableTrace && traceLogPath != "" {
		if err := os.MkdirAll(filepath.Dir(traceLogPath), 0o755); err != nil {
			return nil, err
		}
	}

	return &GlobalCallback{
		callbacks:    make(map[string]*CallbackRecord),
		probeStates:  make(map[string]ProbeState),
		enableTrace:  enableTrace,
		traceLogPath: traceLogPath,
	}, nil
}

// OnCallbackStarted 回调开始
func (gc *GlobalCallback) OnCallbackStarted(name string) string {
	id := gc.generateID(name)

	gc.mu.Lock()
	defer gc.mu.Unlock()

	gc.callbacks[id] = &CallbackRecord{
		ID:        id,
		StartTime: time.Now(),
		Status:    "started",
		Data:      make(map[string]interface{}),
	}
	gc.writeEvent("callback_started", map[string]interface{}{
		"id":   id,
		"name": name,
	})

	return id
}

// OnCallbackCompleted 回调完成
func (gc *GlobalCallback) OnCallbackCompleted(id string, data map[string]interface{}) {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	if record, ok := gc.callbacks[id]; ok {
		record.EndTime = time.Now()
		record.Status = "completed"
		if data != nil {
			for k, v := range data {
				record.Data[k] = v
			}
		}
	}
	gc.writeEvent("callback_completed", map[string]interface{}{
		"id":   id,
		"data": data,
	})
}

// OnCallbackError 回调错误
func (gc *GlobalCallback) OnCallbackError(id string, err error) {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	if record, ok := gc.callbacks[id]; ok {
		record.EndTime = time.Now()
		record.Status = "error"
		record.Error = err
	}
	gc.writeEvent("callback_error", map[string]interface{}{
		"id":    id,
		"error": err.Error(),
	})
}

// OnProbeDeployed 探针部署
func (gc *GlobalCallback) OnProbeDeployed(name, location string) {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	gc.probeStates[name] = ProbeState{
		Name:      name,
		Status:    "deployed",
		Timestamp: time.Now(),
		Location:  location,
	}
	gc.writeEvent("probe_deployed", map[string]interface{}{
		"name":     name,
		"location": location,
	})
}

// OnProbeReturned 探针回收
func (gc *GlobalCallback) OnProbeReturned(name string) {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	if state, ok := gc.probeStates[name]; ok {
		state.Status = "returned"
		state.Timestamp = time.Now()
		gc.probeStates[name] = state
	}
	gc.writeEvent("probe_returned", map[string]interface{}{
		"name": name,
	})
}

// GetCallback 获取回调记录
func (gc *GlobalCallback) GetCallback(id string) (*CallbackRecord, bool) {
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	record, ok := gc.callbacks[id]
	return record, ok
}

// GetAllCallbacks 获取所有回调记录
func (gc *GlobalCallback) GetAllCallbacks() []*CallbackRecord {
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	records := make([]*CallbackRecord, 0, len(gc.callbacks))
	for _, record := range gc.callbacks {
		records = append(records, record)
	}

	return records
}

// GetProbeState 获取探针状态
func (gc *GlobalCallback) GetProbeState(name string) (ProbeState, bool) {
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	state, ok := gc.probeStates[name]
	return state, ok
}

// GetAllProbeStates 获取所有探针状态
func (gc *GlobalCallback) GetAllProbeStates() map[string]ProbeState {
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	states := make(map[string]ProbeState, len(gc.probeStates))
	for k, v := range gc.probeStates {
		states[k] = v
	}

	return states
}

func (gc *GlobalCallback) TraceWriteErrors() uint64 {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	return gc.writeErrors
}

// GetTraceLog 获取追踪日志
func (gc *GlobalCallback) GetTraceLog() string {
	var sb strings.Builder

	sb.WriteString("=== Callback Trace Log ===\n")
	sb.WriteString(fmt.Sprintf("Total callbacks: %d\n", len(gc.callbacks)))

	for _, record := range gc.callbacks {
		sb.WriteString(fmt.Sprintf("\nCallback: %s\n", record.ID))
		sb.WriteString(fmt.Sprintf("  Status: %s\n", record.Status))
		sb.WriteString(fmt.Sprintf("  Start: %s\n", record.StartTime.Format(time.RFC3339)))
		if !record.EndTime.IsZero() {
			duration := record.EndTime.Sub(record.StartTime)
			sb.WriteString(fmt.Sprintf("  End: %s\n", record.EndTime.Format(time.RFC3339)))
			sb.WriteString(fmt.Sprintf("  Duration: %v\n", duration))
		}
		if record.Error != nil {
			sb.WriteString(fmt.Sprintf("  Error: %v\n", record.Error))
		}
		if len(record.Data) > 0 {
			sb.WriteString("  Data:\n")
			for k, v := range record.Data {
				sb.WriteString(fmt.Sprintf("    %s: %v\n", k, v))
			}
		}
	}

	sb.WriteString("\n=== Probe States ===\n")
	sb.WriteString(fmt.Sprintf("Total probes: %d\n", len(gc.probeStates)))

	for name, state := range gc.probeStates {
		sb.WriteString(fmt.Sprintf("\nProbe: %s\n", name))
		sb.WriteString(fmt.Sprintf("  Status: %s\n", state.Status))
		sb.WriteString(fmt.Sprintf("  Timestamp: %s\n", state.Timestamp.Format(time.RFC3339)))
		sb.WriteString(fmt.Sprintf("  Location: %s\n", state.Location))
	}

	return sb.String()
}

// generateID 生成回调 ID
func (gc *GlobalCallback) generateID(name string) string {
	return fmt.Sprintf("%s-%d", name, time.Now().UnixNano())
}

// Reset 重置追踪器
func (gc *GlobalCallback) Reset() {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	gc.callbacks = make(map[string]*CallbackRecord)
	gc.probeStates = make(map[string]ProbeState)
}

func (gc *GlobalCallback) writeEvent(kind string, payload map[string]interface{}) {
	if !gc.enableTrace || gc.traceLogPath == "" {
		return
	}

	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"type":      kind,
		"payload":   redactPayload(payload),
	}
	data, err := json.Marshal(entry)
	if err != nil {
		gc.writeErrors++
		return
	}

	file, err := os.OpenFile(gc.traceLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		gc.writeErrors++
		return
	}
	defer file.Close()

	if _, err := file.Write(append(data, '\n')); err != nil {
		gc.writeErrors++
	}
}

func redactPayload(payload map[string]interface{}) map[string]interface{} {
	if payload == nil {
		return nil
	}
	redacted := make(map[string]interface{}, len(payload))
	for k, v := range payload {
		if isSensitiveKey(k) {
			redacted[k] = "[REDACTED]"
			continue
		}
		redacted[k] = redactValue(v)
	}
	return redacted
}

func redactValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		return redactPayload(typed)
	case map[string]string:
		redacted := make(map[string]string, len(typed))
		for k, v := range typed {
			if isSensitiveKey(k) {
				redacted[k] = "[REDACTED]"
			} else {
				redacted[k] = v
			}
		}
		return redacted
	case []interface{}:
		redacted := make([]interface{}, len(typed))
		for i, item := range typed {
			redacted[i] = redactValue(item)
		}
		return redacted
	default:
		return value
	}
}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, marker := range []string{"password", "passwd", "token", "secret", "credential"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
