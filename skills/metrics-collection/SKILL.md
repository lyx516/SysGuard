---
name: metrics-collection
description: Use when collecting structured CPU, memory, disk, network, service, or health-score metrics from SysGuard for dashboards or reports.
---

# Metrics Collection

Use this skill when the user needs metric-shaped data rather than a narrative health diagnosis.

## Runtime Invocation

The Go implementation is registered by `RegisterCoreSkills` and invoked as `"metrics-collection"`:

```go
registry.Execute(ctx, "metrics-collection", &skills.SkillInput{})
```

The result includes timestamp, health score, health boolean, and component metrics from the monitor.

## Workflow

1. Run the runtime skill for a current metrics snapshot.
2. Preserve component names and metric keys from the returned structure.
3. Summarize trends only if historical data is actually provided.
4. Use `alerting` if thresholds are breached and the user wants incident output.

## Safety

This skill is read-only. Do not remediate based solely on one metric unless the user explicitly asks for action.
