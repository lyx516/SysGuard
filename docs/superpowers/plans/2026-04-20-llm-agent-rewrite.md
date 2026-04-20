# LLM Agent Rewrite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rework SysGuard so operational decisions are made by an LLM agent, while Go provides safe tools, auditing, and execution controls.

**Architecture:** Add an LLM decision loop with an OpenAI-compatible client, a small JSON decision protocol, and a tool registry. Inspector still collects health data; Coordinator routes anomalies to the LLM agent; tools expose health checks, SOP retrieval, history search, shell execution, and verification.

**Tech Stack:** Go 1.21 standard library HTTP client, existing SysGuard monitor/rag/security/shell packages, OpenAI-compatible chat completions API.

---

### Batch 1: LLM Decision Protocol

**Files:**
- Create: `internal/llmagent/types.go`
- Create: `internal/llmagent/client.go`
- Create: `internal/llmagent/client_test.go`

- [ ] Add decision JSON schema: `action`, `tool`, `args`, `final_answer`, `thought`.
- [ ] Add OpenAI-compatible chat client that parses JSON decisions.
- [ ] Test parsing plain and fenced JSON decisions.
- [ ] Run `go test ./internal/llmagent`.
- [ ] Commit and push.

### Batch 2: Tool Registry And Agent Loop

**Files:**
- Create: `internal/llmagent/agent.go`
- Create: `internal/llmagent/tools.go`
- Create: `internal/llmagent/agent_test.go`

- [ ] Add a registry for named tools with JSON-like arguments.
- [ ] Add an agent loop that calls the LLM, runs selected tools, feeds results back, and stops on final answer.
- [ ] Enforce max steps and unknown-tool errors.
- [ ] Run `go test ./internal/llmagent`.
- [ ] Commit and push.

### Batch 3: SysGuard Tool Set And Coordinator Wiring

**Files:**
- Create: `internal/agents/llmops/tools.go`
- Modify: `internal/agents/coordinator/coordinator.go`
- Modify: `internal/config/config.go`
- Modify: `configs/config.yaml`
- Test: `internal/agents/llmops/tools_test.go`
- Test: `internal/agents/coordinator/coordinator_test.go`

- [ ] Expose health, SOP retrieval, history search, shell execution, and verification as tools.
- [ ] Make Coordinator use LLM agent when `ai.enabled` is true.
- [ ] Refuse autonomous remediation when AI is disabled, instead of silently making rule-based decisions.
- [ ] Run `go test ./...`.
- [ ] Commit and push.
