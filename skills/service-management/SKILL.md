---
name: service-management
description: Use when inspecting, starting, stopping, restarting, or reading logs for managed Linux/macOS services through SysGuard.
---

# Service Management

Use this skill for explicit service operations after health checks or log analysis identify a service-level issue.

## Runtime Invocation

The Go implementation is registered by `RegisterCoreSkills` and invoked as `"service-management"`:

```go
registry.Execute(ctx, "service-management", &skills.SkillInput{Params: map[string]interface{}{
	"operation": "status",
	"service": "nginx",
	"lines": 100,
}})
```

Supported operations are `status`, `logs`, `start`, `stop`, and `restart`. Non-Linux systems currently support `status` via process lookup.

## Workflow

1. Prefer `status` or `logs` before changing service state.
2. For `stop`, `start`, or `restart`, explain the intended action and approval requirement.
3. Pass `allow_dangerous=true` only after explicit user/operator approval.
4. Verify with `status` after a state-changing operation.

## Safety

State-changing commands may be blocked by the runtime `CommandInterceptor`. Never bypass dangerous-command checks by constructing arbitrary shell commands.
