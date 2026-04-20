# Production Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move SysGuard from a single-host demo toward safer production operation with guarded remediation, auditable state, protected UI access, stronger configuration, and basic deployment hygiene.

**Architecture:** Keep the existing Go daemon and UI shape, but add production guardrails around the current behavior instead of replacing the project wholesale. Each batch is independently testable and committed after a quick verification run.

**Tech Stack:** Go 1.21, standard library HTTP/process APIs, local JSONL/JSON storage, existing SysGuard packages.

---

### Batch 1: Safe Defaults And Config Surface

**Files:**
- Modify: `internal/config/config.go`
- Modify: `configs/config.yaml`
- Modify: `cmd/sysguard/main.go`
- Test: `internal/config/config_test.go`

- [ ] Add remediator `dry_run` and `verify_after_remediation` config fields.
- [ ] Add UI `addr` and `auth_token` config fields.
- [ ] Make the daemon accept `-config`, matching the UI binary.
- [ ] Run `go test ./internal/config ./cmd/sysguard`.
- [ ] Commit and push.

### Batch 2: Remediation Execution Safety

**Files:**
- Modify: `internal/agents/remediator/remediator.go`
- Modify: `internal/monitor/health.go` if verification needs monitor support
- Test: `internal/agents/remediator/remediator_test.go`

- [ ] Add dry-run execution behavior that records planned commands without running them.
- [ ] Add command result records with status, stdout/stderr snippets, and duration.
- [ ] Add post-remediation verification hook before writing successful history.
- [ ] Run `go test ./internal/agents/remediator ./internal/rag ./pkg/utils`.
- [ ] Commit and push.

### Batch 3: UI Access Control And Safer Defaults

**Files:**
- Modify: `cmd/sysguard-ui/main.go`
- Modify: `internal/ui/server.go`
- Test: `internal/ui/server_test.go`

- [ ] Default UI bind address to `127.0.0.1:8080`.
- [ ] Add optional bearer token protection for all API/SSE/A2UI endpoints.
- [ ] Add request method guards where endpoints are read-only.
- [ ] Run `go test ./internal/ui ./cmd/sysguard-ui`.
- [ ] Commit and push.

### Batch 4: Audit Storage And Observability Hygiene

**Files:**
- Modify: `internal/rag/history.go`
- Modify: `internal/observability/trace.go`
- Test: `internal/rag/history_test.go`
- Test: `internal/observability/trace_test.go`

- [ ] Write history files atomically with `0600` permissions.
- [ ] Add trace write error accounting instead of silently swallowing failures.
- [ ] Redact obvious secrets from trace/log payloads before writing.
- [ ] Run `go test ./internal/rag ./internal/observability`.
- [ ] Commit and push.

### Batch 5: Deployment And Production Docs

**Files:**
- Modify: `README.md`
- Modify: `Dockerfile`
- Create: `deploy/systemd/sysguard.service`
- Create: `deploy/systemd/sysguard-ui.service`

- [ ] Document safe production modes: observe-only, approved remediation, and unattended remediation.
- [ ] Add systemd unit examples with non-root defaults and explicit config paths.
- [ ] Update Docker notes to clarify host-agent limitations.
- [ ] Run `go test ./...`.
- [ ] Commit and push.
