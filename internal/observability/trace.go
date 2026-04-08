package observability

import (
	"fmt"
	"sync"
	"time"
)

// GlobalCallback 全局回调追踪器，记录工具调用链路
type GlobalCallback struct {
	mu          sync.RWMutex
	callbacks   map[string]*CallbackRecord
	probeStates map[string]ProbeState
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
func NewGlobalCallback() (*GlobalCallback, error) {
	return &GlobalCallback{
		callbacks:   make(map[string]*CallbackRecord),
		probeStates: make(map[string]ProbeState),
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
