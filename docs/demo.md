# SysGuard Local Demo

This demo is designed for first-time visitors who want to see SysGuard fail safely.

It starts the Dashboard with:

- AI Agent enabled
- remediation dry-run enabled
- approval enabled
- one synthetic missing service named `sysguard-demo-missing-service`

## Run

```bash
make demo
```

Open:

```text
http://127.0.0.1:18080
```

Click `立即巡检`.

## What To Look For

- The system health score should drop because the configured service is missing.
- The graph run should take the `ai` branch and record the Agent outcome.
- The run should appear in the `运行记录` view.
- Runtime artifacts are written under `.demo/`.

## Generated Files

```text
.demo/config.yaml
.demo/runs.json
.demo/history.json
.demo/trace.log
.demo/sysguard.log
```

## Try Approval Flow

The default demo keeps `dry_run=true`, so side-effecting service operations are simulated. To inspect the approval queue path, edit `.demo/config.yaml`:

```yaml
execution:
  dry_run: false
security:
  enable_approval: true
```

Then call a state-changing service tool through an Agent run or a focused integration test. SysGuard will create a pending approval request instead of executing the command directly.

Approval API:

```bash
curl http://127.0.0.1:18080/api/approvals
curl -X POST http://127.0.0.1:18080/api/approvals/<approval_id>/approve
curl -X POST http://127.0.0.1:18080/api/approvals/<approval_id>/deny
```
