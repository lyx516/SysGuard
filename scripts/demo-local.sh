#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEMO_DIR="${ROOT_DIR}/.demo"
CONFIG_PATH="${DEMO_DIR}/config.yaml"
PORT="${SYSGUARD_DEMO_PORT:-18080}"

mkdir -p "${DEMO_DIR}" "${ROOT_DIR}/build"

if [[ -z "${OPENAI_API_KEY:-}" && -z "${SYSGUARD_AI_API_KEY:-}" ]]; then
  cat <<MSG
SysGuard now always runs through the AI Agent path.

Set an API key before starting the demo:
  export OPENAI_API_KEY="your-api-key"

Optional provider overrides:
  export SYSGUARD_AI_MODEL="gpt-4.1-mini"
  export SYSGUARD_AI_BASE_URL="https://api.openai.com/v1"
MSG
  exit 1
fi

cat >"${CONFIG_PATH}" <<YAML
monitor:
  check_interval: 10s
  health_threshold: 80.0
  cpu_threshold: 95.0
  memory_threshold: 95.0
  disk_threshold: 95.0

orchestration:
  interval: 10s
  anomaly_cooldown: 5s

execution:
  command_timeout: 30s
  dry_run: true
  verify_after_remediation: true

security:
  dangerous_commands:
    - rm
    - kill
    - killall
    - dd
    - mkfs
    - shutdown
    - reboot
  enable_approval: true
  approval_timeout: 5m

knowledge_base:
  docs_path: "${ROOT_DIR}/docs/sop"

observability:
  enable_tracing: true
  trace_log_path: "${DEMO_DIR}/trace.log"

ui:
  addr: "127.0.0.1:${PORT}"
  auth_token: ""

ai:
  provider: openai
  model: \${SYSGUARD_AI_MODEL:-gpt-4.1-mini}
  api_key_env: OPENAI_API_KEY
  base_url: \${SYSGUARD_AI_BASE_URL:-https://api.openai.com/v1}
  timeout: 30s
  max_tokens: 2048
  temperature: 0.2

storage:
  history_path: "${DEMO_DIR}/history.json"
  runs_path: "${DEMO_DIR}/runs.json"
  approvals_path: "${DEMO_DIR}/approvals.json"

logging:
  level: info
  format: json
  output: "${DEMO_DIR}/sysguard.log"

services:
  names:
    - sysguard-demo-missing-service
YAML

printf 'Building SysGuard demo binaries...\n'
go build -o "${ROOT_DIR}/build/sysguard-ui" "${ROOT_DIR}/cmd/sysguard-ui"

cat <<MSG

SysGuard demo is starting.

Open:
  http://127.0.0.1:${PORT}

Then click:
  立即巡检

Demo state files:
  ${DEMO_DIR}/runs.json
  ${DEMO_DIR}/history.json
  ${DEMO_DIR}/trace.log

Press Ctrl+C to stop the dashboard.

MSG

exec "${ROOT_DIR}/build/sysguard-ui" -config "${CONFIG_PATH}"
