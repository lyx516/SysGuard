---
name: notification
description: Use when sending SysGuard alerts, summaries, remediation outcomes, or operational messages to stdout, logs, or webhook targets.
---

# Notification

Use this skill to deliver already-formed operational messages. Pair it with `alerting` when the message represents an incident.

## Runtime Invocation

The Go implementation is registered by `RegisterCoreSkills` and invoked as `"notification"`:

```go
registry.Execute(ctx, "notification", &skills.SkillInput{Params: map[string]interface{}{
	"channel": "webhook",
	"target": "https://example.com/hook",
	"message": "nginx restart completed",
}})
```

Supported channels are `stdout`, `log`, and `webhook`.

## Workflow

1. Confirm the destination channel and target.
2. Keep messages factual and include incident IDs when available.
3. For webhook delivery, check non-2xx responses and report failure.
4. Avoid sending secrets, DSNs, credentials, or full logs.

## Safety

Treat notifications as external disclosure. Redact credentials and sensitive host details unless the user explicitly requests them and the destination is trusted.
