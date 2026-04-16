---
name: container-management
description: Use when inspecting, listing, reading logs from, starting, stopping, or restarting Docker containers through SysGuard.
---

# Container Management

Use this skill when the incident is container-scoped or the user explicitly asks for Docker container operations.

## Runtime Invocation

The Go implementation is registered by `RegisterCoreSkills` and invoked as `"container-management"`:

```go
registry.Execute(ctx, "container-management", &skills.SkillInput{Params: map[string]interface{}{
	"operation": "logs",
	"container": "web",
}})
```

Supported operations are `ps`, `inspect`, `logs`, `restart`, `stop`, and `start`.

## Workflow

1. Use `ps`, `inspect`, or `logs` before changing container state.
2. For `restart`, `stop`, or `start`, state the exact container and expected impact.
3. Pass `allow_dangerous=true` only after explicit approval.
4. Verify with `ps` or `inspect` after changes.

## Safety

Do not execute arbitrary `docker exec` commands from this skill. Keep operations to the runtime-supported allowlist.
