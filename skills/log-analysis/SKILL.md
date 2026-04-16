---
name: log-analysis
description: Use when analyzing operational logs, filtering errors or warnings, summarizing incidents from log files, or finding relevant lines in SysGuard-managed logs.
---

# Log Analysis

Use this skill to inspect log files without loading entire files into context. Prefer the SysGuard Go runtime skill when working inside this repository.

## Runtime Invocation

The Go implementation is registered by `RegisterCoreSkills` and invoked as `"log-analysis"`:

```go
registry.Execute(ctx, "log-analysis", &skills.SkillInput{Params: map[string]interface{}{
	"path": "/path/to/app.log",
	"chunk_size": 1000,
	"keywords": []string{"error", "warning", "critical"},
}})
```

## Workflow

1. Identify the log file path and whether the user wants keywords or broad incident discovery.
2. Use default keywords for generic incident analysis: `error`, `failed`, `warning`, `critical`, `exception`, `timeout`.
3. Return matching chunks, total matched lines, and a concise incident summary.
4. If the result points to a service failure, hand off to `service-management` or `health-check`.

## Safety

Do not rewrite, truncate, or delete logs from this skill. For file mutation requests, use a separate reviewed file operation path.
