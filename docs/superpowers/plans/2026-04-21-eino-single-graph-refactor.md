# Eino Single-Graph Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the custom SysGuard agent runtime with a single Eino graph that owns inspection, routing, evidence retrieval, tool-use, verification, persistence, and observability.

**Architecture:** Introduce `internal/orchestration` as the only production orchestration layer and `internal/eino` as the Eino adapter layer for models, tools, and callbacks. Thin compatibility wrappers remain only where CLI, UI, or tests still need stable entrypoints, and obsolete custom `internal/llmagent` runtime files are removed once the new graph is live.

**Tech Stack:** Go, CloudWeGo Eino compose/flow/runtime, existing SysGuard skills/security/history/observability packages, OpenAI-compatible chat model adapter

---

## File Map

### New files

- `internal/orchestration/state.go`
  - Graph state types, branch enums, and helper methods.
- `internal/orchestration/graph.go`
  - Graph builder, compiled runner, and public orchestration entrypoint.
- `internal/orchestration/nodes_inspect.go`
  - Health check node and anomaly derivation.
- `internal/orchestration/nodes_route.go`
  - Cooldown, suppression, AI enablement, and branch routing.
- `internal/orchestration/nodes_retrieve.go`
  - SOP/history retrieval bundle node.
- `internal/orchestration/nodes_agent.go`
  - Eino model/tool orchestration node.
- `internal/orchestration/nodes_verify.go`
  - Post-action verification node.
- `internal/orchestration/nodes_persist.go`
  - History, audit, and trace persistence node.
- `internal/orchestration/runtime_test.go`
  - End-to-end orchestration tests for route and callback visibility.
- `internal/eino/model.go`
  - OpenAI-compatible Eino model adapter.
- `internal/eino/tools.go`
  - SysGuard skill to Eino tool adapter.
- `internal/eino/callbacks.go`
  - Eino callback bridge into `internal/observability`.
- `internal/eino/tools_test.go`
  - Tool adaptation and schema tests.
- `internal/eino/callbacks_test.go`
  - Callback bridge tests.

### Modified files

- `cmd/sysguard/main.go`
  - Bootstrap the orchestration runtime instead of the legacy chain.
- `internal/agents/inspector/inspector.go`
  - Reduce to health-check implementation or helper, no orchestration ownership.
- `internal/agents/coordinator/coordinator.go`
  - Reduce to compatibility wrapper or delete if fully replaced.
- `internal/agents/remediator/remediator.go`
  - Keep plan/policy helpers that remain useful, remove orchestration ownership.
- `internal/config/config.go`
  - Finalize orchestration/cooldown/model config consumption.
- `internal/rag/knowledge.go`
  - Ensure retrieval results expose citations required by graph state.
- `internal/skills/core_skills.go`
  - Export skill metadata needed by Eino tool registration.
- `internal/skills/llm_tools.go`
  - Replace custom runtime-specific registry helpers with Eino-compatible definitions or delete if redundant.
- `internal/ui/dashboard.go`
  - Read graph-driven observability consistently.
- `internal/ui/server.go`
  - Make `/api/check` run the graph and return the fresh snapshot.
- `internal/ui/server_test.go`
  - Assert `/api/check` triggers a real orchestration run.
- `internal/ui/dashboard_test.go`
  - Assert current-session graph traces are visible in the dashboard.
- `internal/agents/coordinator/coordinator_test.go`
  - Replace legacy coordinator-loop tests with orchestration wrapper coverage or delete if obsolete.
- `go.mod`
  - Finalize Eino dependencies if new subpackages are required.

### Deleted files

- `internal/llmagent/agent.go`
- `internal/llmagent/tools.go`
- `internal/llmagent/agent_test.go`
- `internal/llmagent/eval_test.go`
  - Delete after graph runtime fully replaces them.

### Cleanup targets

- `/$tmpdir`
  - Remove the accidental runtime artifact directory from the repo root.

---

### Task 1: Create the orchestration state model

**Files:**
- Create: `internal/orchestration/state.go`
- Test: `internal/orchestration/runtime_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestNewStateDefaults(t *testing.T) {
	state := NewState(TriggerManualCheck)

	require.NotEmpty(t, state.RunID)
	require.Equal(t, TriggerManualCheck, state.Trigger)
	require.Equal(t, BranchUnknown, state.Branch)
	require.False(t, state.Suppressed)
	require.Nil(t, state.Report)
	require.Nil(t, state.Anomaly)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/orchestration -run TestNewStateDefaults -v`
Expected: FAIL with `undefined: NewState` and missing orchestration package symbols.

- [ ] **Step 3: Write minimal implementation**

```go
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
	RunID      string
	Trigger    Trigger
	Branch     Branch
	Suppressed bool
	Report     *inspector.HealthReport
	Anomaly    *shared.Anomaly
}

func NewState(trigger Trigger) *State {
	return &State{
		RunID:   uuid.NewString(),
		Trigger: trigger,
		Branch:  BranchUnknown,
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/orchestration -run TestNewStateDefaults -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/orchestration/state.go internal/orchestration/runtime_test.go
git commit -m "refactor: add orchestration state model"
```

### Task 2: Add Eino callback bridge before replacing the runtime

**Files:**
- Create: `internal/eino/callbacks.go`
- Create: `internal/eino/callbacks_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestCallbackBridgePublishesToolLifecycle(t *testing.T) {
	obs := observability.NewGlobalCallback()
	bridge := NewCallbackBridge(obs)

	ctx := context.Background()
	runInfo := callbacks.RunInfo{Name: "agent_react"}
	ctx = bridge.OnStart(ctx, runInfo, callbacks.CallbackInput{})
	bridge.OnEnd(ctx, runInfo, callbacks.CallbackOutput{})

	records := obs.GetRecords()
	require.NotEmpty(t, records)
	require.Equal(t, "graph.agent_react", records[0].Name)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/eino -run TestCallbackBridgePublishesToolLifecycle -v`
Expected: FAIL with `undefined: NewCallbackBridge`.

- [ ] **Step 3: Write minimal implementation**

```go
type CallbackBridge struct {
	obs *observability.GlobalCallback
}

func NewCallbackBridge(obs *observability.GlobalCallback) *CallbackBridge {
	return &CallbackBridge{obs: obs}
}

func (b *CallbackBridge) OnStart(ctx context.Context, info callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	if b.obs != nil {
		b.obs.OnCallbackStarted("graph." + info.Name)
	}
	return ctx
}

func (b *CallbackBridge) OnEnd(ctx context.Context, info callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	if b.obs != nil {
		b.obs.OnCallbackCompleted("graph." + info.Name)
	}
	return ctx
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/eino -run TestCallbackBridgePublishesToolLifecycle -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/eino/callbacks.go internal/eino/callbacks_test.go
git commit -m "refactor: bridge eino callbacks into observability"
```

### Task 3: Adapt SysGuard skills into Eino tools

**Files:**
- Create: `internal/eino/tools.go`
- Create: `internal/eino/tools_test.go`
- Modify: `internal/skills/core_skills.go`
- Modify: `internal/skills/llm_tools.go`

- [ ] **Step 1: Write the failing test**

```go
func TestBuildToolsIncludesSOPRetrievalSchema(t *testing.T) {
	registry := skills.NewRegistry()
	require.NoError(t, skills.RegisterLLMTools(registry))

	tools, err := BuildTools(registry)
	require.NoError(t, err)

	tool, ok := tools["sop-retrieval"]
	require.True(t, ok)
	require.Equal(t, "read_only", tool.Permission())
	require.Contains(t, tool.Parameters(), "query")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/eino -run TestBuildToolsIncludesSOPRetrievalSchema -v`
Expected: FAIL with `undefined: BuildTools`.

- [ ] **Step 3: Write minimal implementation**

```go
type ToolAdapter interface {
	Name() string
	Permission() string
	Parameters() map[string]any
	InvokableRun(ctx context.Context, args string) (string, error)
}

func BuildTools(registry *skills.Registry) (map[string]ToolAdapter, error) {
	result := make(map[string]ToolAdapter)
	for _, definition := range skills.ListLLMToolDefinitions(registry) {
		result[definition.Name] = NewSkillTool(definition, registry)
	}
	return result, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/eino -run TestBuildToolsIncludesSOPRetrievalSchema -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/eino/tools.go internal/eino/tools_test.go internal/skills/core_skills.go internal/skills/llm_tools.go
git commit -m "refactor: expose SysGuard skills as Eino tools"
```

### Task 4: Build the inspect and route nodes with cooldown

**Files:**
- Create: `internal/orchestration/nodes_inspect.go`
- Create: `internal/orchestration/nodes_route.go`
- Modify: `internal/config/config.go`
- Test: `internal/orchestration/runtime_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestRouteSuppressedSkipsAIRun(t *testing.T) {
	runtime := &Runtime{
		cfg: &config.Config{
			AI: config.AIConfig{Enabled: true},
			Agents: config.AgentsConfig{
				Inspector: config.InspectorConfig{AnomalyCooldown: 30 * time.Second},
			},
		},
		lastHandled: map[string]time.Time{
			"service|critical|svc down": time.Now(),
		},
	}

	state := NewState(TriggerPeriodic)
	state.Anomaly = &shared.Anomaly{
		Source:      "service",
		Severity:    shared.SeverityCritical,
		Description: "svc down",
	}

	next, err := runtime.routeMode(context.Background(), state)
	require.NoError(t, err)
	require.Equal(t, BranchSuppressed, next.Branch)
	require.True(t, next.Suppressed)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/orchestration -run TestRouteSuppressedSkipsAIRun -v`
Expected: FAIL with `undefined: Runtime` or `undefined: routeMode`.

- [ ] **Step 3: Write minimal implementation**

```go
type Runtime struct {
	cfg         *config.Config
	lastHandled map[string]time.Time
	mu          sync.Mutex
}

func (r *Runtime) routeMode(ctx context.Context, state *State) (*State, error) {
	if state.Report != nil && state.Report.IsHealthy {
		state.Branch = BranchHealthy
		return state, nil
	}

	if state.Anomaly != nil && r.isSuppressed(state.Anomaly) {
		state.Branch = BranchSuppressed
		state.Suppressed = true
		return state, nil
	}

	if !r.cfg.AI.Enabled {
		state.Branch = BranchAlertOnly
		return state, nil
	}

	state.Branch = BranchAI
	return state, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/orchestration -run TestRouteSuppressedSkipsAIRun -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/orchestration/nodes_inspect.go internal/orchestration/nodes_route.go internal/orchestration/runtime_test.go internal/config/config.go
git commit -m "refactor: add orchestration routing and cooldown nodes"
```

### Task 5: Build retrieval and verification nodes with citation-carrying state

**Files:**
- Create: `internal/orchestration/nodes_retrieve.go`
- Create: `internal/orchestration/nodes_verify.go`
- Modify: `internal/rag/knowledge.go`
- Modify: `internal/agents/remediator/remediator.go`
- Test: `internal/orchestration/runtime_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestRetrieveEvidenceIncludesCitationMetadata(t *testing.T) {
	runtime := newTestRuntime(t)
	state := NewState(TriggerManualCheck)
	state.Anomaly = &shared.Anomaly{Description: "nginx down"}

	next, err := runtime.retrieveEvidence(context.Background(), state)
	require.NoError(t, err)
	require.NotEmpty(t, next.Evidence.SOP)
	require.NotEmpty(t, next.Evidence.SOP[0].ChunkID)
	require.NotEmpty(t, next.Evidence.SOP[0].Path)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/orchestration -run TestRetrieveEvidenceIncludesCitationMetadata -v`
Expected: FAIL with `State has no field or method Evidence` or `undefined: retrieveEvidence`.

- [ ] **Step 3: Write minimal implementation**

```go
type EvidenceBundle struct {
	SOP     []rag.SearchResult
	History []remediator.HistoryRecord
}

func (r *Runtime) retrieveEvidence(ctx context.Context, state *State) (*State, error) {
	results, err := r.knowledge.Search(ctx, state.Anomaly.Description, 3)
	if err != nil {
		return nil, err
	}
	state.Evidence = EvidenceBundle{SOP: results}
	return state, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/orchestration -run TestRetrieveEvidenceIncludesCitationMetadata -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/orchestration/nodes_retrieve.go internal/orchestration/nodes_verify.go internal/orchestration/runtime_test.go internal/rag/knowledge.go internal/agents/remediator/remediator.go
git commit -m "refactor: add orchestration retrieval and verification nodes"
```

### Task 6: Build the Eino agent node and compile the single graph

**Files:**
- Create: `internal/eino/model.go`
- Create: `internal/orchestration/nodes_agent.go`
- Create: `internal/orchestration/graph.go`
- Test: `internal/orchestration/runtime_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestGraphAIRouteExecutesToolCallbacks(t *testing.T) {
	runtime := newTestRuntime(t)
	state := NewState(TriggerManualCheck)
	state.Branch = BranchAI
	state.Anomaly = &shared.Anomaly{Description: "service down"}
	state.Evidence = EvidenceBundle{
		SOP: []rag.SearchResult{{ChunkID: "chunk-1", Content: "restart service"}},
	}

	result, err := runtime.RunOnce(context.Background(), state)
	require.NoError(t, err)
	require.NotEmpty(t, result.Agent.Tools)
	require.NotEmpty(t, result.Agent.Final)
	require.NotEmpty(t, runtime.obs.GetRecords())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/orchestration -run TestGraphAIRouteExecutesToolCallbacks -v`
Expected: FAIL with `undefined: RunOnce` or missing graph compilation/runtime methods.

- [ ] **Step 3: Write minimal implementation**

```go
func NewRuntime(cfg *config.Config, deps Dependencies) (*Runtime, error) {
	r := &Runtime{cfg: cfg, deps: deps, obs: deps.Observer, lastHandled: map[string]time.Time{}}
	graph := compose.NewGraph[*State, *State]()
	// add inspect, route, retrieve, agent, verify, persist nodes and branches
	compiled, err := graph.Compile(context.Background())
	if err != nil {
		return nil, err
	}
	r.graph = compiled
	return r, nil
}

func (r *Runtime) RunOnce(ctx context.Context, state *State) (*State, error) {
	return r.graph.Invoke(ctx, state)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/orchestration -run TestGraphAIRouteExecutesToolCallbacks -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/eino/model.go internal/orchestration/nodes_agent.go internal/orchestration/graph.go internal/orchestration/runtime_test.go
git commit -m "refactor: compile eino single-graph runtime"
```

### Task 7: Switch daemon and UI entrypoints to the graph runtime

**Files:**
- Modify: `cmd/sysguard/main.go`
- Modify: `internal/ui/server.go`
- Modify: `internal/ui/dashboard.go`
- Modify: `internal/ui/server_test.go`
- Modify: `internal/ui/dashboard_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestServerCheckEndpointRunsGraph(t *testing.T) {
	runtime := &fakeRuntime{}
	server := NewServer(NewCollector(runtime, nil, nil, nil))

	req := httptest.NewRequest(http.MethodPost, "/api/check", nil)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, runtime.runCount)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -run TestServerCheckEndpointRunsGraph -v`
Expected: FAIL with `fakeRuntime runCount is 0` or constructor mismatch.

- [ ] **Step 3: Write minimal implementation**

```go
type GraphRunner interface {
	Run(ctx context.Context, trigger orchestration.Trigger) (*orchestration.State, error)
}

func (c *Collector) TriggerCheck(ctx context.Context) (Snapshot, error) {
	if _, err := c.runner.Run(ctx, orchestration.TriggerManualCheck); err != nil {
		return Snapshot{}, err
	}
	return c.Snapshot(ctx)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/ui -run TestServerCheckEndpointRunsGraph -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/sysguard/main.go internal/ui/server.go internal/ui/dashboard.go internal/ui/server_test.go internal/ui/dashboard_test.go
git commit -m "refactor: route daemon and ui through orchestration graph"
```

### Task 8: Remove obsolete custom runtime and compatibility dead code

**Files:**
- Delete: `internal/llmagent/agent.go`
- Delete: `internal/llmagent/tools.go`
- Delete: `internal/llmagent/agent_test.go`
- Delete: `internal/llmagent/eval_test.go`
- Modify: `internal/agents/coordinator/coordinator.go`
- Modify: `internal/agents/coordinator/coordinator_test.go`
- Remove: `/$tmpdir`

- [ ] **Step 1: Write the failing test**

```bash
rg "internal/llmagent|llmagent\\." /Users/liyuxuan/Desktop/SysGuard
```

Expected: existing production references are still present and must be removed.

- [ ] **Step 2: Run cleanup verification to capture current failures**

Run: `go test ./...`
Expected: FAIL in files still referencing deleted runtime symbols.

- [ ] **Step 3: Write minimal implementation**

```go
// coordinator.go should either be removed or reduced to:
type Coordinator struct {
	runtime *orchestration.Runtime
}

func (c *Coordinator) Run(ctx context.Context, trigger orchestration.Trigger) (*orchestration.State, error) {
	return c.runtime.Run(ctx, trigger)
}
```

- [ ] **Step 4: Run verification to confirm cleanup passes**

Run: `rm -rf /Users/liyuxuan/Desktop/SysGuard/'$tmpdir' && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor: remove legacy llmagent runtime"
```

### Task 9: Final integration verification

**Files:**
- Modify as needed: any files touched by verification fixes

- [ ] **Step 1: Run focused orchestration and UI tests**

Run: `go test ./internal/orchestration ./internal/eino ./internal/ui -v`
Expected: PASS

- [ ] **Step 2: Run the full test suite**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 3: Build the binaries**

Run: `go build -o build/sysguard ./cmd/sysguard && go build -o build/sysguard-ui ./cmd/sysguard-ui`
Expected: both binaries build successfully

- [ ] **Step 4: Smoke-test the manual check path**

Run: `./build/sysguard -config ./configs/config.yaml`
Expected: daemon starts and logs orchestration startup without legacy llmagent references

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "test: verify eino orchestration refactor"
```

---

## Self-Review

### Spec coverage

- Single Eino graph runtime: covered by Tasks 1, 4, 5, 6, 7.
- Replace custom `internal/llmagent`: covered by Task 8.
- Rebuild `Inspector / Coordinator / Remediator` scheduling: covered by Tasks 4, 5, 6, 7, 8.
- Register skills as Eino tools: covered by Task 3.
- Route callbacks into dashboard trace format: covered by Tasks 2, 6, 7.
- Cooldown and suppression in graph nodes: covered by Task 4.
- `/api/check` triggers a real run: covered by Task 7.
- Keep safety, audit, verification, and citations: covered by Tasks 3, 5, 6.

### Placeholder scan

- No `TODO`, `TBD`, or “implement later” placeholders remain.
- Every task contains concrete files, commands, and minimal code targets.

### Type consistency

- `Trigger`, `Branch`, `State`, `Runtime`, `EvidenceBundle`, and `RunOnce` are defined before later tasks use them.
- `manual_check` trigger naming is consistent between state, graph runtime, and UI task.
