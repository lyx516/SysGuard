---
name: network-diagnosis
description: Use when diagnosing DNS, TCP connectivity, ping reachability, network interfaces, service ports, or network-related SysGuard incidents.
---

# Network Diagnosis

Use this skill to isolate whether an incident is caused by name resolution, interface state, TCP connectivity, or host reachability.

## Runtime Invocation

The Go implementation is registered by `RegisterCoreSkills` and invoked as `"network-diagnosis"`:

```go
registry.Execute(ctx, "network-diagnosis", &skills.SkillInput{Params: map[string]interface{}{
	"operation": "tcp",
	"host": "example.com",
	"port": 443,
	"timeout": "5s",
}})
```

Supported operations are `interfaces`, `dns`, `tcp`, and `ping`.

## Workflow

1. Start with `interfaces` for local network suspicion.
2. Use `dns` when hostnames fail or resolve unexpectedly.
3. Use `tcp` for service-port reachability.
4. Use `ping` only as a coarse reachability signal; ICMP can be blocked.

## Safety

Do not run scanning or broad port sweeps from this skill. Diagnose only the host and port relevant to the incident.
