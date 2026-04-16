---
name: alerting
description: Use when creating structured incident alerts from SysGuard findings, health reports, log analysis, remediation failures, or operational events.
---

# Alerting

Use this skill to turn operational findings into structured alerts that downstream notification or audit systems can consume.

## Runtime Invocation

The Go implementation is registered by `RegisterCoreSkills` and invoked as `"alerting"`:

```go
registry.Execute(ctx, "alerting", &skills.SkillInput{Params: map[string]interface{}{
	"severity": "critical",
	"title": "Service down",
	"message": "nginx is inactive",
	"source": "sysguard",
}})
```

The result is an `Alert` with ID, severity, title, message, source, metadata, and timestamp.

## Workflow

1. Use `critical` for down services, unreachable networks, or failed remediation.
2. Use `warning` for degraded CPU, memory, disk, or suspicious logs.
3. Include source and metadata so later remediation can trace the alert.
4. Pair with `notification` when the alert needs to leave the process.

## Safety

Alerts should describe facts observed by SysGuard. Do not invent root causes when the evidence only supports symptoms.
