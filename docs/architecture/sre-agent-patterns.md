# SRE Agent Reference Patterns

This document records the public projects used as architecture references for
SysGuard. It intentionally summarizes design ideas and links to the upstream
sources instead of copying upstream code, documentation, runbooks, prompts, or
tool definitions.

## Sources

- HolmesGPT: <https://github.com/robusta-dev/holmesgpt>
- HolmesGPT built-in toolsets documentation: <https://holmesgpt.dev/data-sources/builtin-toolsets/aws/>
- kagent: <https://github.com/kagent-dev/kagent>
- kagent tools documentation: <https://kagent.dev/docs/kagent/concepts/tools>
- GoogleCloudPlatform kubectl-ai: <https://github.com/GoogleCloudPlatform/kubectl-ai>
- K8sGPT: <https://github.com/k8sgpt-ai/k8sgpt>
- StackStorm: <https://github.com/StackStorm/st2>
- Rundeck: <https://github.com/rundeck/rundeck>
- Scoutflo SRE Playbooks: <https://github.com/Scoutflo/Scoutflo-SRE-Playbooks>

## Patterns Adopted In SysGuard

### Read-Only First

Production SRE agents should default to observation and diagnosis before
remediation. SysGuard encodes this in tool metadata with `permission`,
`side_effects`, and `requires_approval`.

### Toolsets

Tools are grouped into toolsets so the runtime and UI can reason about their
domain:

- `host`
- `network`
- `observability`
- `filesystem`
- `database`
- `workflow`
- `knowledge`

### Declarative Metadata

Inspired by declarative agent/tool configuration patterns, SysGuard tools now
carry operational metadata:

- permission tier
- side-effect flag
- approval requirement
- allowed platforms
- output budget
- redaction policy

### Structured Runbooks

SOP documents can include front matter with fields for risk, approval,
signals, diagnosis, execution, verification, and rollback. Retrieved evidence
keeps this metadata attached to each chunk.

### Evaluation Scenarios

SysGuard keeps a local scenario list in `docs/evals/agent_scenarios.yaml` covering
service failures, host pressure, false positives, prompt injection, irrelevant
SOP retrieval, tool failure, approval refusal, model timeout, and dashboard
trace visibility.
