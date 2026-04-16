---
name: file-operation
description: Use when reading, statting, listing, or tailing local files for SysGuard diagnostics without modifying filesystem contents.
---

# File Operation

Use this skill for safe local file inspection during diagnostics.

## Runtime Invocation

The Go implementation is registered by `RegisterCoreSkills` and invoked as `"file-operation"`:

```go
registry.Execute(ctx, "file-operation", &skills.SkillInput{Params: map[string]interface{}{
	"operation": "tail",
	"path": "/var/log/syslog",
	"lines": 100,
}})
```

Supported operations are `read`, `stat`, `list`, and `tail`.

## Workflow

1. Use `stat` to confirm path type and size before reading large files.
2. Prefer `tail` for logs and potentially large files.
3. Use `list` for directory inspection.
4. Use `log-analysis` when keyword filtering is needed.

## Safety

This skill is read-only. Do not create, overwrite, rename, chmod, or delete files through this skill.
