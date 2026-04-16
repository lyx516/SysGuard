---
name: database-operation
description: Use when checking database connectivity, running read-only diagnostic SQL, validating DSNs, or collecting small database health evidence through SysGuard.
---

# Database Operation

Use this skill for conservative database diagnostics. It is not a general-purpose SQL administration interface.

## Runtime Invocation

The Go implementation is registered by `RegisterCoreSkills` and invoked as `"database-operation"`:

```go
registry.Execute(ctx, "database-operation", &skills.SkillInput{Params: map[string]interface{}{
	"operation": "ping",
	"driver": "postgres",
	"dsn": "postgres://user:pass@localhost/db",
}})
```

For queries, only read-only `SELECT` or `WITH` statements are accepted by the runtime skill.

## Workflow

1. Prefer `ping` to test connectivity.
2. Use read-only `query` only when the user supplies a safe diagnostic query.
3. Keep result limits small with the `limit` parameter.
4. Escalate write, migration, backup, or restore tasks to a human-reviewed database workflow.

## Safety

Never run DDL, DML, destructive SQL, or unbounded queries from this skill.
