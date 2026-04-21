package orchestration

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/sysguard/sysguard/internal/config"
	syseino "github.com/sysguard/sysguard/internal/eino"
	"github.com/sysguard/sysguard/internal/monitor"
	"github.com/sysguard/sysguard/internal/observability"
	"github.com/sysguard/sysguard/internal/rag"
	"github.com/sysguard/sysguard/internal/security"
	"github.com/sysguard/sysguard/internal/skills"
)

type Runtime struct {
	cfg         *config.Config
	kb          *rag.KnowledgeBase
	historyKB   *rag.HistoryKnowledgeBase
	monitor     *monitor.Monitor
	interceptor *security.CommandInterceptor
	obs         *observability.GlobalCallback

	graph       compose.Runnable[*State, *State]
	callbacks   compose.Option
	mu          sync.Mutex
	lastHandled map[string]time.Time
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

func NewRuntime(
	ctx context.Context,
	cfg *config.Config,
	kb *rag.KnowledgeBase,
	historyKB *rag.HistoryKnowledgeBase,
	monitor *monitor.Monitor,
	interceptor *security.CommandInterceptor,
	obs *observability.GlobalCallback,
) (*Runtime, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if monitor == nil {
		return nil, fmt.Errorf("monitor is required")
	}
	r := &Runtime{
		cfg:         cfg,
		kb:          kb,
		historyKB:   historyKB,
		monitor:     monitor,
		interceptor: interceptor,
		obs:         obs,
		callbacks:   compose.WithCallbacks(syseino.NewCallbackBridge(obs)),
		lastHandled: make(map[string]time.Time),
	}
	if cfg.AI.Enabled {
		if _, err := syseino.NewChatModel(ctx, cfg.AI); err != nil {
			return nil, err
		}
	}
	graph, err := r.buildGraph(ctx)
	if err != nil {
		return nil, err
	}
	r.graph = graph
	return r, nil
}

func (r *Runtime) buildGraph(ctx context.Context) (compose.Runnable[*State, *State], error) {
	graph := compose.NewGraph[*State, *State]()
	nodes := []struct {
		key string
		fn  func(context.Context, *State) (*State, error)
	}{
		{"inspect", r.inspect},
		{"detect_anomaly", r.detectAnomaly},
		{"route_mode", r.routeMode},
		{"retrieve_evidence", r.retrieveEvidence},
		{"agent_react", r.agentReact},
		{"verify_result", r.verifyResult},
		{"persist_result", r.persistResult},
	}
	for _, node := range nodes {
		if err := graph.AddLambdaNode(node.key, compose.InvokableLambda(node.fn), compose.WithNodeName(node.key)); err != nil {
			return nil, err
		}
	}
	if err := graph.AddEdge(compose.START, "inspect"); err != nil {
		return nil, err
	}
	for i := 0; i < len(nodes)-1; i++ {
		if err := graph.AddEdge(nodes[i].key, nodes[i+1].key); err != nil {
			return nil, err
		}
	}
	if err := graph.AddEdge(nodes[len(nodes)-1].key, compose.END); err != nil {
		return nil, err
	}
	return graph.Compile(ctx, compose.WithGraphName("SysGuardSingleGraph"), compose.WithMaxRunSteps(32))
}

func (r *Runtime) Run(ctx context.Context, trigger Trigger) (*State, error) {
	return r.RunState(ctx, NewState(trigger))
}

func (r *Runtime) RunState(ctx context.Context, state *State) (*State, error) {
	if state == nil {
		state = NewState(TriggerPeriodic)
	}
	out, err := r.graph.Invoke(ctx, state, r.callbacks)
	if err != nil {
		if out == nil {
			out = state
		}
		if out.Agent.Error == "" {
			out.Agent.Error = err.Error()
		}
		out.CompletedAt = time.Now().UTC()
		out, _ = r.persistResult(ctx, out)
		return out, err
	}
	if out != nil {
		out.CompletedAt = time.Now().UTC()
	}
	return out, err
}

func (r *Runtime) Start(ctx context.Context) error {
	runCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		log.Println("Orchestration: starting Eino single graph")
		r.runPeriodic(runCtx, TriggerStartup)
		ticker := time.NewTicker(r.checkInterval())
		defer ticker.Stop()
		for {
			select {
			case <-runCtx.Done():
				log.Println("Orchestration: stopped")
				return
			case <-ticker.C:
				r.runPeriodic(runCtx, TriggerPeriodic)
			}
		}
	}()
	return nil
}

func (r *Runtime) Stop(ctx context.Context) error {
	if r.cancel != nil {
		r.cancel()
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		r.wg.Wait()
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (r *Runtime) runPeriodic(ctx context.Context, trigger Trigger) {
	if _, err := r.Run(ctx, trigger); err != nil {
		log.Printf("Orchestration: graph run failed: %v", err)
	}
}

func (r *Runtime) checkInterval() time.Duration {
	if r.cfg.Orchestration.Interval > 0 {
		return r.cfg.Orchestration.Interval
	}
	return r.cfg.Monitor.CheckInterval
}

func (r *Runtime) coreSkillDefinitions(ctx context.Context) ([]skills.ToolDefinition, error) {
	registry := skills.NewSkillRegistry()
	if err := skills.RegisterCoreSkills(registry, skills.CoreSkillDependencies{
		Config:      r.cfg,
		Monitor:     r.monitor,
		Interceptor: r.interceptor,
	}); err != nil {
		return nil, err
	}
	return skills.CoreSkillToolDefinitions(registry)
}
