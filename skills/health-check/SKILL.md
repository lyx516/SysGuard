---
name: health-check
description: Use when checking host health, service health, CPU, memory, disk, network status, or determining whether SysGuard should treat a system as unhealthy.
---

# Health Check

Use this skill to get a structured SysGuard health report. It should be the first stop before remediation when the user asks whether a system or service is healthy.

## Runtime Invocation

The Go implementation is registered by `RegisterCoreSkills` and invoked as `"health-check"`:

```go
registry.Execute(ctx, "health-check", &skills.SkillInput{})
```

The result is `*monitor.HealthReport` with timestamp, score, `IsHealthy`, and component statuses.

## Workflow

1. Run the runtime skill to collect the current report.
2. Read component statuses: `cpu`, `memory`, `disk`, `network`, and `services`.
3. If `IsHealthy` is false, summarize degraded/down components and severity.
4. Use `metrics-collection` for metric-shaped output or `service-management` for explicit service action.

## Safety

This skill is read-only. Do not restart services or execute repair commands from this skill.
