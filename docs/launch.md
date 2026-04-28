# Launch Notes

Use this when publishing SysGuard to GitHub, V2EX, Hacker News, Reddit, X, or a blog.

## Repository Description

```text
Local AI SRE agent in Go: host health checks, SOP-grounded diagnosis, approval-gated remediation, dashboard, and auditable graph runs.
```

## Suggested GitHub Topics

```text
sre
ai-agent
observability
incident-response
runbook
llmops
eino
golang
devops
automation
approval-workflow
site-reliability-engineering
```

## Short Launch Post

```text
I built SysGuard, a local AI SRE agent in Go.

It runs host and service health checks, detects anomalies, retrieves SOP/runbook evidence, routes incidents through an Eino graph, gates privileged remediation through an approval queue, and records every graph run for audit.

Default mode is conservative: AI off, dry-run on, side-effecting tools require approval.

Demo:
  make demo

GitHub:
  https://github.com/lyx516/SysGuard
```

## Show HN Style Title

```text
Show HN: SysGuard, a local AI SRE agent in Go with approval-gated remediation
```

## Screenshot Checklist

Before posting, capture one image that shows:

- Health score degraded by the demo missing service
- The `运行记录` page
- One graph run detail drawer
- Pending approvals if using a non-dry-run test

Set `docs/images/sysguard-social-preview.svg` as the GitHub social preview image if you want a clean default sharing card.
