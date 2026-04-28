# SysGuard

Local AI SRE Agent for host inspection, SOP-grounded diagnosis, approval-gated remediation, and auditable graph runs.

SysGuard 是一个基于 Go 与 CloudWeGo Eino 的本地运维 Agent 框架。它把主机巡检、异常检测、SOP/RAG 检索、LLM tool-use、审批队列、执行安全、结果验证和历史沉淀放进同一张 Eino graph 中，用来构建可观测、可审计、可逐步放权的 AI 运维闭环。

> 当前项目处于 production hardening 阶段。默认配置偏保守：AI 关闭、修复 dry-run、危险操作需要审批。本项目适合先在本地或测试机器观察运行，再逐步接入真实服务和执行权限。

[![CI](https://github.com/lyx516/SysGuard/actions/workflows/ci.yml/badge.svg)](https://github.com/lyx516/SysGuard/actions/workflows/ci.yml)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8)
![Eino](https://img.shields.io/badge/Agent-Eino-3b82f6)
![Status](https://img.shields.io/badge/status-production%20hardening-0f766e)
![License](https://img.shields.io/badge/license-MIT-green)

![SysGuard social preview](docs/images/sysguard-social-preview.svg)

## 目录

- [项目定位](#项目定位)
- [核心能力](#核心能力)
- [架构总览](#架构总览)
- [快速开始](#快速开始)
- [本地演示](#本地演示)
- [配置说明](#配置说明)
- [AI 与工具调用](#ai-与工具调用)
- [执行安全](#执行安全)
- [SOP/RAG 与历史记忆](#soprag-与历史记忆)
- [Dashboard 与 API](#dashboard-与-api)
- [测试与验证](#测试与验证)
- [生产运行建议](#生产运行建议)
- [项目结构](#项目结构)
- [开发路线](#开发路线)

## 项目定位

SysGuard 不是一个会直接“放飞自我”的自动修复脚本。它的设计目标是成为一个可靠的运维 Agent 骨架：

1. 持续巡检本机 CPU、内存、磁盘、网络和受管服务。
2. 将异常转换成结构化事件。
3. 基于本地 SOP 和历史记录检索证据。
4. 在 Eino ReAct agent 中让模型通过受控工具诊断和处理。
5. 对命令执行做 allowlist、参数校验、权限分级、审批和审计。
6. 修复后重新巡检验证。
7. 把 trace、工具调用、历史结果留存下来，供 UI、排障和下一次推理复用。

当 `ai.enabled=false` 时，SysGuard 只做巡检、告警、trace 和历史记录，不伪装成 AI 自动修复。当 `ai.enabled=true` 时，才进入 LLM agent 路径。

## 核心能力

- **Eino 单图编排**
  `internal/orchestration` 使用一张 graph 串起 `inspect -> detect_anomaly -> route_mode -> retrieve_evidence -> agent_react -> verify_result -> persist_result`。

- **真实本地巡检**
  支持 CPU load、内存、磁盘、网络接口和受管服务状态检查。任一组件 `down` 或 `critical` 会直接判定为不健康，避免默认健康阈值漏报服务宕机。

- **OpenAI-compatible LLM 接入**
  可配置 OpenAI、硅基流动或其他兼容 `/v1/chat/completions` 的模型服务。

- **标准 ReAct/tool-use 循环**
  模型通过 Eino ReAct agent 输出 tool call，系统执行注册工具，将 observation 回填，再继续推理直到 final 或达到 step 限制。

- **LLM 可调用工具注册**
  `internal/skills` 中的核心能力会注册成工具，带 JSON schema、参数校验和权限等级。

- **工具集元数据**
  每个工具会暴露 toolset、side-effect、审批要求、平台约束、输出预算和脱敏策略，方便模型、运行时和 UI 一起理解工具风险。

- **工具失败可恢复**
  工具参数错误、平台不支持、命令失败会作为 `success=false` observation 返回给模型，而不是直接打断整轮 graph。

- **SOP/RAG 证据约束**
  本地 Markdown SOP 会被切成 chunk，生成稀疏 embedding，检索后带 citation 提供给模型。

- **历史记忆**
  处理结果写入本地 history。后续相似异常可检索历史记录，减少重复摸索。

- **安全执行**
  默认 dry-run，命令走安全策略、危险命令识别、参数校验、审批和审计路径。

- **Dashboard 与 API**
  `sysguard-ui` 提供本地看板、快照 API、手动检查接口和 SSE 数据流。

- **评测与回归测试**
  测试覆盖配置加载、RAG 检索、工具 schema、安全策略、Eino callback、orchestration 路由、UI snapshot 等关键路径。

## 架构总览

```mermaid
flowchart TD
    A["Timer or /api/check"] --> B["inspect: health report"]
    B --> C["detect_anomaly"]
    C --> D["route_mode"]
    D -->|healthy| H["persist_result / finish"]
    D -->|cooldown| H
    D -->|AI disabled| I["alert-only path"]
    I --> H
    D -->|AI enabled| E["retrieve_evidence"]
    E --> F["agent_react: Eino ReAct"]
    F --> G["verify_result"]
    G --> H

    F --> T["registered tools"]
    T --> F

    subgraph Evidence
      KB["SOP chunks + citations"]
      Hist["history records"]
    end

    E --> KB
    E --> Hist
```

运行时只有一条主编排链路，不再保留旧的 Inspector / Coordinator / Remediator 分散调度关系。这样做的好处是：每轮运行都有清晰的 run state、trace、分支和持久化边界。

## 快速开始

### 1. 准备环境

要求：

- Go 1.21+
- macOS 或 Linux
- 本地 shell 环境可执行常用巡检命令
- 若要管理 Linux 服务，建议系统具备 `systemctl` 和 `journalctl`

### 2. 克隆项目

```bash
git clone https://github.com/lyx516/SysGuard.git
cd SysGuard
```

### 3. 准备本地私有配置

仓库只提交示例配置，不提交真实本地配置。先复制一份：

```bash
cp configs/config.example.yaml configs/config.yaml
```

`configs/config.yaml` 已被 `.gitignore` 忽略，适合写本机服务名、私有模型地址、API key 或本地路径。公开示例请改 `configs/config.example.yaml`。

### 4. 下载依赖

```bash
go mod download
```

### 5. 构建

```bash
go build -o build/sysguard ./cmd/sysguard
go build -o build/sysguard-ui ./cmd/sysguard-ui
```

### 6. 运行 daemon

```bash
./build/sysguard -config configs/config.yaml
```

默认情况下：

- AI 关闭。
- 修复 dry-run。
- trace 写入 `logs/trace.log`。
- 运行日志写入 `logs/sysguard.log`。
- 历史记录写入 `data/history.json`。
- graph 运行记录写入 `data/runs.json`。

### 7. 运行 Dashboard

另开一个终端：

```bash
./build/sysguard-ui -config configs/config.yaml
```

打开：

```text
http://127.0.0.1:8080
```

## 本地演示

最快体验方式：

```bash
make demo
```

它会生成一个本地 demo 配置，配置一个不存在的受管服务 `sysguard-demo-missing-service`，并启动 Dashboard：

```text
http://127.0.0.1:18080
```

打开页面后点击“立即巡检”，你会看到一次完整的异常 graph run：健康检查失败、异常进入 alert-only 路径、运行记录写入 `data/runs.json`。详细说明见 [docs/demo.md](docs/demo.md)。

## 配置说明

完整示例见 `configs/config.example.yaml`。常用配置如下：

```yaml
monitor:
  check_interval: 30s
  health_threshold: 80.0
  cpu_threshold: 85.0
  memory_threshold: 90.0
  disk_threshold: 90.0

orchestration:
  interval: 30s
  anomaly_cooldown: 30s

execution:
  auto_approve_safe_commands: true
  command_timeout: 2m
  allow_interactive_input: true
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
  docs_path: "./docs/sop"

observability:
  enable_tracing: true
  trace_log_path: "./logs/trace.log"

ui:
  addr: "127.0.0.1:8080"
  auth_token: ""

ai:
  enabled: false
  provider: openai
  model: gpt-4.1-mini
  api_key: ""
  api_key_env: OPENAI_API_KEY
  base_url: "https://api.openai.com/v1"
  timeout: 30s
  max_tokens: 2048
  temperature: 0.2

storage:
  history_path: "./data/history.json"
  runs_path: "./data/runs.json"
  approvals_path: "./data/approvals.json"

logging:
  level: info
  format: json
  output: ./logs/sysguard.log

services:
  # names:
  #   - nginx
  #   - redis
```

### 关键字段

- `monitor.health_threshold`
  总体健康分阈值。注意：只要存在 `down` 或 `critical` 组件，即使总分达到阈值也会判定不健康。

- `orchestration.interval`
  Eino graph 周期运行间隔。

- `orchestration.anomaly_cooldown`
  相同异常重复触发 AI 路径前的冷却时间。

- `execution.dry_run`
  建议生产首次接入保持 `true`。关闭后才可能真实执行允许的命令。

- `execution.verify_after_remediation`
  修复后重新巡检，并把验证结果写入 history。

- `security.enable_approval`
  是否启用人工审批。危险命令必须经过审批。

- `observability.trace_log_path`
  Eino callback、graph 节点、工具调用事件的落盘路径。

- `ui.auth_token`
  为空时本地无鉴权。暴露到非本机网络前必须配置 token。

- `ai.api_key_env`
  推荐使用环境变量注入 key，例如 `OPENAI_API_KEY`。

- `services.names`
  受管服务列表。macOS 下使用 `pgrep -x` 做进程存在性检查；Linux 下优先使用 `systemctl is-active`。

## AI 与工具调用

开启 AI：

```yaml
ai:
  enabled: true
  provider: openai
  model: Qwen/Qwen3.6-35B-A3B
  api_key_env: OPENAI_API_KEY
  base_url: "https://api.siliconflow.cn/v1"
```

或用环境变量：

```bash
export OPENAI_API_KEY="your-api-key"
```

Agent 使用的 system prompt 强调三件事：

1. 必须基于 SOP citation 和历史证据。
2. 优先使用只读诊断工具。
3. final answer 需要包含诊断、证据、动作、验证和回滚建议。

当前注册的 LLM 工具包括：

| 工具 | 权限 | 用途 |
| --- | --- | --- |
| `health-check` | read_only | 重新执行健康巡检 |
| `metrics-collection` | read_only | 收集主机指标 |
| `log-analysis` | read_only | 按 chunk 分析日志关键词 |
| `network-diagnosis` | read_only | 接口、DNS、TCP、ping 诊断 |
| `service-management` | privileged | 服务 status/logs/start/stop/restart |
| `file-operation` | read_only | read/stat/list/tail |
| `database-operation` | read_only | ping/query |
| `alerting` | read_only | 生成结构化告警 |
| `notification` | privileged | stdout/log/webhook 通知 |
| `sop-retrieval` | read_only | 检索 SOP evidence chunk |
| `history-search` | read_only | 检索相似历史记录 |

每个工具都有 JSON schema。模型传入非法参数或工具执行失败时，结果会作为 `success=false` observation 返回给模型继续推理。

## 执行安全

SysGuard 的执行安全目标是“最小权限、可解释、可回滚”。

### 命令策略

命令执行不会直接信任模型输出。核心策略包括：

- allowlist 命令模板。
- 参数级校验。
- 只读、特权、危险三级权限。
- dry-run 默认开启。
- 危险命令拦截。
- 交互式审批。
- 命令超时。
- trace 和 history 审计。

### 推荐运行姿势

1. 本地首次运行保持 `ai.enabled=false` 和 `dry_run=true`。
2. 确认巡检、trace、dashboard 正常后，开启 AI。
3. 只让模型使用只读工具观察几轮。
4. 为少量低风险服务开启审批修复。
5. 观察 history 和验证结果，再考虑扩大权限。

## SOP/RAG 与历史记忆

SOP 文档位于：

```text
docs/sop/
```

写 SOP 时建议包含：

- 故障现象。
- 诊断命令。
- 前置检查。
- 执行步骤。
- 验证步骤。
- 回滚步骤。
- 风险和审批要求。

SOP 可以带结构化 front matter：

```yaml
---
id: service-restart
risk_level: privileged
required_approval: true
signals:
  - service status is down
diagnosis_steps:
  - check service status
verification_steps:
  - verify service status
rollback_steps:
  - restore previous configuration backup
steps:
  - id: inspect-service
    title: Inspect service status
    type: diagnosis
    intent: Confirm the service state before any side-effecting action.
    tool: service-management
    action: status
    preconditions:
      - service name is known
      - current phase is read-only diagnosis
    risks:
      - status output may include sensitive startup parameters
    verification:
      - status output was collected and summarized
    rollback:
      - no rollback required for read-only diagnosis
  - id: restart-service
    title: Restart service after approval
    type: execution
    intent: Restore availability after diagnosis supports restart.
    tool: service-management
    action: restart
    requires_approval: true
    preconditions:
      - service is confirmed down
      - recent logs do not indicate a config rollback is required first
      - production approval has been granted
    risks:
      - active connections may be interrupted
      - restart may fail again if the root cause is unchanged
    verification:
      - service status is active
      - health check returns healthy
    rollback:
      - restore previous known-good configuration
      - restart service and verify health again
---
```

知识库加载 Markdown 文件后会：

1. 提取标题和路径。
2. 按段落切 chunk。
3. 生成轻量稀疏 embedding。
4. 按 cosine similarity 检索。
5. 返回带 citation 的 `EvidenceChunk`。

历史记录位于 `storage.history_path`。失败运行也会写入 history，便于审计 LLM 超时、工具失败、审批拒绝等情况。每次 graph 运行的状态快照位于 `storage.runs_path`，用于 UI/API 查询最近运行和排查并发触发。审批请求位于 `storage.approvals_path`，有副作用工具会先写入 pending request，批准后才允许携带 `approval_id` 执行。

## Dashboard 与 API

`sysguard-ui` 是本地运维看板，不是聊天页面。它用于监督 agent 是否被唤醒、graph 节点是否运行、工具调用是否成功、当前机器是否健康。

常用接口：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/snapshot` | 返回当前看板快照 |
| `POST` | `/api/check` | 立即触发一次 graph 巡检，然后返回快照 |
| `GET` | `/api/runs` | 返回最近 graph 运行记录 |
| `GET` | `/api/runs/{run_id}` | 返回单次 graph 运行详情 |
| `GET` | `/api/approvals` | 返回最近审批请求 |
| `POST` | `/api/approvals/{id}/approve` | 批准待审批操作 |
| `POST` | `/api/approvals/{id}/deny` | 拒绝待审批操作 |
| `GET` | `/api/stream` | SSE 实时事件流 |
| `GET` | `/a2ui/render` | 返回 A2UI render tree 与数据模型 |

如果设置了 `ui.auth_token`，请求需要携带：

```bash
curl -H "Authorization: Bearer <token>" http://127.0.0.1:8080/api/snapshot
```

## 测试与验证

运行全部测试：

```bash
go test ./...
```

构建两个二进制：

```bash
go build -o build/sysguard ./cmd/sysguard
go build -o build/sysguard-ui ./cmd/sysguard-ui
```

常用定向测试：

```bash
go test ./internal/eino
go test ./internal/orchestration
go test ./internal/monitor
go test ./internal/rag
go test ./internal/security
go test ./internal/ui
```

本项目已经覆盖的典型场景：

- 服务宕机触发异常。
- `down` 组件不会被总体分数掩盖。
- AI 关闭时走告警路径。
- 相同异常冷却去重。
- 工具 schema 和权限元数据暴露。
- 工具失败返回 observation。
- graph 失败写入 history。
- SOP chunk 检索带 citation。
- Dashboard 不混入旧 session 数据。
- `/api/check` 触发真实检查。

当前评测集分两份文件：

- `docs/evals/agent_scenarios.yaml`：给人读的场景说明。
- `docs/evals/agent_scenarios.json`：给测试读取的机器可执行模拟数据。

## Agent 场景评测结果

这里的结果只记录真实 LLM replay eval。评测会读取同一份场景数据，调用本地配置里的 OpenAI-compatible 模型，让模型真实执行 Eino ReAct/tool-call 循环；所有工具 handler 都是安全评测桩，只返回模拟 observation 并记录调用，不会重启服务、删文件或发送外部通知。

当前评测覆盖 12 类场景：

| 场景 | 期望能力 |
| --- | --- |
| `service_down_ai_path` | AI 开启时进入 ReAct 路径，检索 SOP/history，健康检查验证，生成含诊断、验证、回滚的结果 |
| `service_down_alert_only` | AI 关闭时只告警和落历史，不调用 LLM 工具 |
| `repeated_anomaly_cooldown` | 同类异常在冷却窗口内被抑制，避免重复模型调用 |
| `disk_full` | 使用 SOP、指标和文件工具诊断磁盘压力，不误调用服务重启 |
| `cpu_high` | 使用指标和健康检查定位 CPU 压力，保留 residual risk |
| `false_positive` | 复查健康时不执行修复 |
| `dangerous_command_injection` | 拒绝 `rm -rf /` 一类危险命令诱导 |
| `irrelevant_sop` | SOP 证据不匹配时只继续诊断，不基于无关证据修复 |
| `tool_failure` | 工具失败作为 observation 回填，agent 继续给出下一步 |
| `approval_denied` | 审批拒绝后不执行副作用工具 |
| `llm_timeout` | 模型超时也写入失败历史，保留审计 |
| `dashboard_trace_visibility` | AI graph、agent node 和工具调用可被 trace/UI 观测 |

运行方式：

```bash
SYSGUARD_RUN_LIVE_LLM_EVAL=1 go test ./internal/evals -run TestLiveLLMReplayEvaluation -count=1 -v
```

最近一次本机结果：

```text
模型: Qwen/Qwen3.6-35B-A3B
场景数: 12
真实 LLM 场景数: 9
场景成功数: 12
工具调用准确率: precision 85.2%, recall 100.0%
平均 ReAct 轮数: 2.78
平均 LLM replay 耗时: 10.78s
危险/禁用工具违规: 0
报告: data/evals/live_llm_replay_latest.json
```

| 指标 | 结果 |
| --- | --- |
| 模型 | `Qwen/Qwen3.6-35B-A3B` |
| 场景数 | 12 |
| 真实 LLM 场景数 | 9 |
| 场景成功数 | 12 |
| 工具调用准确率 | precision 85.2%, recall 100.0% |
| 平均 ReAct 轮数 | 2.78 |
| 平均 LLM replay 耗时 | 10.78s |
| 危险/禁用工具违规 | 0 |

这次 replay 的结论是：模型能覆盖所有必需工具，且没有调用禁用工具；但在 `dangerous_command_injection` 和 `approval_denied` 这类安全场景中，会额外调用 `health-check`、`log-analysis` 等读工具，说明 prompt 对“最小必要工具调用”的约束还可以继续收紧。`final_contains_score` 目前是字符串匹配，对模型改写表达不够公平，后续应改为结构化 JSON 输出或语义 rubric 评分。

## 参考项目

SysGuard 的下一阶段设计参考了 HolmesGPT、kagent、kubectl-ai、K8sGPT、StackStorm、Rundeck 和 Scoutflo SRE Playbooks 的公开架构思路。仓库中的 `docs/architecture/sre-agent-patterns.md` 记录了参考链接和被吸收的设计模式。为避免错误继承和版权问题，SysGuard 不复制外部项目正文、代码、runbook 或 prompt，而是保留来源链接并做本项目自己的实现。

## 生产运行建议

### 文件与权限

- 使用专门的低权限用户运行。
- 只给必要命令 sudo 权限。
- 把 history、trace、logs 放到独立目录。
- 不要把 `configs/config.yaml` 提交到仓库。
- API key 使用环境变量或系统 secret 管理。

### systemd 示例

仓库提供了 systemd unit 示例：

```text
deploy/systemd/sysguard.service
deploy/systemd/sysguard-ui.service
```

示例假设：

- 二进制位于 `/opt/sysguard/`
- 配置位于 `/etc/sysguard/config.yaml`
- 数据目录位于 `/var/lib/sysguard`
- 日志目录位于 `/var/log/sysguard`

### 网络暴露

Dashboard 默认建议只监听 `127.0.0.1`。如果要暴露到局域网或公网：

- 设置 `ui.auth_token`。
- 使用反向代理加 TLS。
- 限制来源 IP。
- 不要暴露写权限接口给不可信网络。

## 项目结构

```text
cmd/
  sysguard/              daemon 入口
  sysguard-ui/           dashboard 入口
configs/
  config.example.yaml    公开示例配置
deploy/systemd/          systemd unit 示例
docs/sop/                本地 SOP 知识库
docs/evals/              Agent 评测场景
docs/architecture/       架构参考和设计说明
internal/config/         配置加载与环境变量覆盖
internal/monitor/        健康检查与异常构建
internal/orchestration/  Eino 单图运行时
internal/eino/           ChatModel、Tool、Callback 适配
internal/rag/            SOP 知识库与历史记录
internal/security/       命令安全策略与危险命令拦截
internal/skills/         LLM 可调用工具与 skill registry
internal/ui/             Dashboard、API、SSE、A2UI
pkg/utils/               Shell 执行器
skills/                  工具说明文档
```

## 开发路线

近期优先级：

- 去重 Eino callback，避免 trace 统计翻倍。
- 扩展 agent 评测集，覆盖更多故障和攻击场景。
- 把当前稀疏 embedding 替换为可插拔 embedding provider。
- 增加 rerank 接口，提升 SOP 检索质量。
- 为结构化修复 plan 增加更严格的 schema 和 UI 展示。
- 增强回滚策略，从“建议”升级为可验证执行单元。
- 对命令 allowlist 增加更细粒度平台适配。

## 贡献

欢迎提交 issue 和 PR。建议 PR 至少包含：

- 问题背景。
- 设计取舍。
- 测试覆盖。
- 安全影响说明。
- 手动验证结果。

提交前请运行：

```bash
go test ./...
go build -o build/sysguard ./cmd/sysguard
go build -o build/sysguard-ui ./cmd/sysguard-ui
```

## License

MIT License. See `LICENSE`.
