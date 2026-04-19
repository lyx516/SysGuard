# SysGuard GitHub Pages Demo

This directory contains a static, GitHub Pages friendly demo of the SysGuard dashboard.

It does not run the local Go daemon. Instead, it loads `data/snapshot.json`, a rich simulated incident replay that demonstrates:

- Inspector anomaly detection.
- Coordinator routing.
- Remediator SOP retrieval and plan generation.
- CommandInterceptor approval gates.
- ShellExecutor command execution trace.
- Repair history persistence.
- SOP and skill document evidence.

When GitHub Pages is enabled for the repository, the demo can be opened at:

```text
https://lyx516.github.io/SysGuard/demo/
```
