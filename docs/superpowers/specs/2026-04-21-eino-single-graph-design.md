# Eino Single-Graph Refactor Design

**Date:** 2026-04-21

**Status:** Draft for review

**Goal**

Replace the current custom `llmagent` loop and hand-wired `Inspector -> Coordinator -> Remediator` runtime with a single Eino `compose` graph that owns the full orchestration path: inspect, detect, suppress, retrieve, decide, act, verify, persist, and emit observability data.

## Background

The repository currently depends on Eino but does not use it as the actual agent execution framework. The AI path is driven by custom code in `internal/llmagent` and direct calls from `Coordinator`. The UI exposes A2UI-style resources, but the runtime is not an Eino graph, so model runs and tool calls are not naturally represented through Eino callbacks.

This creates three practical problems:

1. The execution framework and the observability story are split.
2. Routing, cooldown, and remediation are spread across multiple packages rather than encoded as one executable graph.
3. The project claims Eino-style operations while the most important runtime path is still custom.

## Scope

This refactor covers the runtime orchestration layer and the AI remediation path.

Included:

- Replace `internal/llmagent` as the primary agent runtime.
- Rebuild the `Inspector / Coordinator / Remediator` scheduling relationship as a single Eino graph.
- Register existing SysGuard skills as Eino tools.
- Route model/tool callbacks through Eino callback handlers and bridge them into the existing dashboard trace format.
- Move anomaly suppression and routing into graph nodes rather than scattered control flow.
- Make `/api/check` trigger a real graph run for an immediate check.

Not included in this refactor:

- Rewriting the dashboard frontend.
- Replacing the current knowledge-base storage format.
- Reworking command-policy semantics beyond what is needed for Eino tool execution.
- Multi-node or distributed orchestration.

## Non-Goals

- Do not preserve the current custom `llmagent` API as a first-class runtime abstraction.
- Do not add a second agent framework in parallel.
- Do not expand the product scope beyond the current single-host SysGuard model.

## Architecture Summary

The new runtime will use one Eino `compose` graph as the single source of truth for orchestration. Each legacy "agent" becomes a graph responsibility rather than an independent scheduler:

- `Inspector` becomes the inspect stage and health-report producer.
- `Coordinator` becomes the route, suppression, and policy stage.
- `Remediator` becomes the retrieval, tool-use, verification, and persistence stage.

The graph will operate on a shared state object that accumulates health data, anomaly metadata, evidence, tool outputs, verification results, and persistence status.

## Graph Shape

The graph will be compiled once at startup and invoked in two ways:

- periodic runs from the daemon scheduler
- on-demand runs from `/api/check`

The graph stages are:

1. `inspect`
   Run host and service health checks and produce a `HealthReport`.

2. `detect_anomaly`
   Convert an unhealthy report into a structured `Anomaly`. Healthy reports bypass remediation.

3. `cooldown_guard`
   Compute an anomaly signature and suppress repeated incidents within a configured cooldown window.

4. `route_mode`
   Choose one of these branches:
   - healthy -> `finish_noop`
   - suppressed -> `finish_suppressed`
   - AI disabled -> `alert_only`
   - AI enabled -> `retrieve_evidence`

5. `retrieve_evidence`
   Query SOP chunks and prior history into a single evidence bundle. This stage is deterministic and separate from model execution.

6. `agent_react`
   Execute the Eino agent/tool-calling stage using registered SysGuard tools. This stage may call read-only or privileged tools depending on routing and policy.

7. `verify_result`
   Verify post-action state when remediation or diagnosis produced actions that require confirmation.

8. `persist_result`
   Write history, trace-compatible summaries, and audit metadata.

9. `emit_snapshot`
   Surface graph outputs in a form consumable by the existing dashboard and A2UI responses.

## Runtime State Model

The graph state should be explicit and serializable enough for testing. It should include:

- run metadata:
  - run ID
  - trigger source (`periodic`, `manual_check`, `startup`)
  - timestamps
- health data:
  - `HealthReport`
  - derived metrics summary
- anomaly data:
  - anomaly struct
  - anomaly signature
  - suppression decision and reason
- AI mode:
  - enabled/disabled
  - selected model configuration
- retrieval data:
  - SOP evidence chunks with citations
  - history matches
- agent execution data:
  - selected tools
  - tool outputs
  - final answer / remediation narrative
- verification data:
  - verify attempted
  - verify result
- persistence data:
  - history write status
  - trace/audit payload
- user-facing observability data:
  - summarized node outcomes
  - callback-friendly records

## Package Layout

The refactor should introduce a dedicated orchestration package so the graph does not live inside `Coordinator`.

Planned structure:

- `internal/orchestration/graph.go`
  Build and compile the Eino graph.

- `internal/orchestration/state.go`
  Define graph state and helper methods.

- `internal/orchestration/nodes_inspect.go`
  Health-check node implementation.

- `internal/orchestration/nodes_route.go`
  Detection, suppression, and branch selection.

- `internal/orchestration/nodes_retrieve.go`
  SOP/history retrieval bundling.

- `internal/orchestration/nodes_agent.go`
  Eino model and tool-calling stage.

- `internal/orchestration/nodes_verify.go`
  Verification logic.

- `internal/orchestration/nodes_persist.go`
  History, trace, and audit persistence.

- `internal/eino/`
  Eino-specific adapters:
  - tool registration
  - model wiring
  - callback bridge into existing observability

The existing `internal/agents` package should be reduced to thin wrappers or compatibility façades during migration. The long-term direction is that orchestration logic no longer lives there.

## Treatment Of Existing Packages

### `internal/llmagent`

This package should be retired from the primary runtime path. It may remain temporarily during migration for tests or compatibility, but the target architecture is that production code does not depend on it.

### `internal/skills`

This package remains the domain-tool implementation layer. Its responsibility changes from "custom registry used by custom agent loop" to "tool implementations registered into Eino tooling."

### `internal/observability`

This package remains as the dashboard-facing record sink, but it should be fed by Eino callback handlers rather than direct manual instrumentation spread throughout the runtime.

### `internal/ui`

The UI should continue to read the same trace/history abstractions, but those records should now be populated from Eino node/model/tool callbacks and graph-level run outcomes.

## Eino Integration Strategy

### Graph

Use Eino `compose` as the orchestration backbone. The graph should define:

- typed state input/output
- branch routing for healthy/suppressed/AI-disabled/AI-enabled paths
- graph compile once, invoke many times

### Model

Use Eino model abstractions to wrap the current OpenAI-compatible endpoint configuration already present in SysGuard config.

### Tools

Adapt SysGuard skills into Eino tools with:

- JSON-schema-like parameter definitions
- permission metadata preserved in local wrappers
- explicit read-only vs privileged separation

### Callbacks

Use Eino callbacks as the primary observability source. Bridge them into current SysGuard callback records so the dashboard still works without a total UI rewrite.

The bridge should emit normalized records for:

- graph run start/end/error
- node start/end/error
- model call start/end/error
- tool call start/end/error

This is the key fix for the current "AI path runs but UI cannot see it" problem.

## Cooldown And Dedup Strategy

Cooldown should no longer be an incidental behavior in `Coordinator`. It should become an explicit graph node with deterministic semantics.

Rules:

- anomaly signature uses source, severity, description, and stable metadata
- suppression is checked before retrieval/model/tool execution
- suppression outcome is still observable and queryable in the dashboard
- suppressed runs do not invoke the model or write duplicate history entries

Cooldown duration should be configurable in `agents.inspector.anomaly_cooldown` to preserve the existing configuration surface.

## `/api/check` Semantics

After the refactor, `/api/check` must perform a real one-shot graph invocation. It should no longer be a snapshot alias.

Expected behavior:

- run one immediate orchestration graph pass
- update observability/history if the graph reaches those stages
- return the fresh snapshot after that run completes

## Safety Requirements

The Eino refactor must not weaken the existing safety model.

The following protections stay in force:

- command policy allowlist
- parameter validation
- permission tiers for tools
- dry-run behavior
- post-remediation verification
- history/audit persistence

Privileged tools must continue to pass through existing policy enforcement rather than bypassing it through Eino.

## Migration Plan

The refactor should happen in phases to keep the repository runnable.

### Phase 1: Introduce Eino orchestration skeleton

- add orchestration package and state model
- compile a trivial graph
- keep old runtime intact

### Phase 2: Move periodic and manual check entry points onto the graph

- daemon periodic checks invoke graph
- `/api/check` invokes graph
- dashboard still works through existing observability structures

### Phase 3: Replace custom AI loop with Eino model/tool calling

- register skills as Eino tools
- remove `internal/llmagent` from production path
- bridge callbacks

### Phase 4: Remove legacy orchestration logic

- minimize or delete redundant scheduling code in `Coordinator`
- reduce `Inspector` / `Remediator` to compatibility shells or remove unused runtime code

## Testing Strategy

The refactor needs regression coverage at graph, node, and API levels.

Required tests:

1. graph routing:
   - healthy path
   - suppressed path
   - AI-disabled path
   - AI-enabled path

2. callback visibility:
   - graph run visible
   - model call visible
   - tool calls visible
   - errors visible

3. cooldown behavior:
   - repeated anomaly within cooldown is suppressed
   - anomaly outside cooldown runs again

4. `/api/check`:
   - triggers one real graph run
   - notifies anomaly path when unhealthy

5. persistence:
   - history written once for unsuppressed incident
   - duplicate suppressed incidents do not create duplicate history

6. dashboard integration:
   - Coordinator/AI path appears in snapshot
   - tool list includes Eino-driven tool invocations

## Risks

### Risk 1: Over-coupling graph state to UI

Mitigation:
Keep graph state domain-oriented. Build dashboard summaries in adapters, not in node logic.

### Risk 2: Partial migration leaves two runtimes alive

Mitigation:
Define a hard cut-over point where the daemon entry uses only the Eino graph.

### Risk 3: Callback duplication

Mitigation:
Use one callback bridge path. Avoid mixing manual callback emission and Eino callback emission for the same runtime stages.

### Risk 4: Tool safety regression

Mitigation:
Retain current policy enforcement inside tool adapters and remediation execution code.

## Final Decisions

This refactor will use the following concrete decisions:

1. Legacy `internal/agents` will remain only as thin compatibility wrappers during migration. New orchestration logic lives in `internal/orchestration`, and the goal at the end of the refactor is that those wrappers contain no scheduling or decision logic.
2. Eino model integration will use the OpenAI-compatible chat path already represented by SysGuard config, so current provider settings and base URLs remain valid after the refactor.
3. Eino callbacks will not write directly into UI code. They will first pass through an adapter layer under `internal/eino/`, which then writes normalized records into `internal/observability`.

Use a single Eino graph as the production runtime, keep `internal/skills` and existing safety/persistence layers, and bridge Eino callbacks into the existing observability store. This gives the project a real Eino core without forcing a simultaneous rewrite of UI, persistence, and domain tools.
