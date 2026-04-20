# SysGuard

智能自动化运维与诊断 Agent，面向“持续巡检 -> 异常发现 -> 安全修复 -> 历史沉淀”这条闭环。

<div align="center">

![Version](https://img.shields.io/badge/version-0.2.0-blue)
![License](https://img.shields.io/badge/license-MIT-green)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8)
![Status](https://img.shields.io/badge/status-production--hardening-0b7285)
![Demo](https://img.shields.io/badge/demo-GitHub%20Pages-007a7a)

[在线演示](https://lyx516.github.io/SysGuard/demo/) • [快速开始](#快速开始) • [运行机制](#运行机制) • [配置说明](#配置说明) • [部署建议](#部署建议)

### [打开 GitHub Pages 交互式演示看板](https://lyx516.github.io/SysGuard/demo/)

用预置模拟事故展示 `Inspector -> Coordinator -> Remediator -> CommandInterceptor -> ShellExecutor` 的完整异常处理链路。

</div>

## 项目简介

SysGuard 是一个用 Go 编写的运维守护进程。它会定期检查主机和受管服务的健康状态，当检测到异常时：

1. 生成结构化异常信息。
2. 从本地 SOP 知识库中检索相关修复文档。
3. 优先复用历史成功修复方案。
4. 在 dry-run、审批和危险命令规则保护下生成或执行修复计划。
5. 修复后重新巡检验证，并记录 trace 和历史结果，供后续复用与审计。

当前版本已经实现了可运行的主链路，并补充了生产接入所需的基础护栏。默认推荐以观察模式运行，再逐步开放审批修复。

## 当前能力

- 双 Agent 协同：
  `Inspector` 负责定时巡检，`Remediator` 负责修复，`Coordinator` 负责调度。
- 真实健康检查：
  支持 CPU、内存、磁盘、网络和受管服务状态检查，不再使用固定模拟数据。
- SOP/RAG 检索：
  从 `docs/sop` 目录加载 Markdown 运维文档，并基于关键词召回相关 SOP。
- AI 配置面：
  支持配置 provider、model、base URL、超时、token 数和 API Key 环境变量；当前默认关闭，后续可用于模型辅助生成修复计划。
- 历史知识库：
  修复计划和验证结果会写入本地历史记录，后续相似故障优先复用历史步骤。
- 安全执行：
  默认 dry-run、危险命令拦截、交互式人工审批、审批超时自动拒绝。
- 可观测性：
  trace 事件和运行日志落盘，trace 会脱敏常见敏感字段，便于审计和排障。
- 监控看板：
  提供本机默认监听的 UI/API/SSE/A2UI 看板，并支持 Bearer token 保护。
- 优雅停止：
  支持 `SIGINT` / `SIGTERM`，可安全关闭巡检和修复流程。

## 在线演示

SysGuard 提供了 GitHub Pages 静态演示版，用预置模拟事故展示完整异常处理链路，不需要运行本机 daemon：

- [GitHub Pages 交互式演示](https://lyx516.github.io/SysGuard/demo/)
- [模拟事故数据](docs/demo/data/snapshot.json)

演示内容包括：

- `Inspector` 发现 nginx 不可用并生成 critical anomaly。
- `Coordinator` 将异常路由给 `Remediator`。
- `Remediator` 检索 SOP、生成恢复计划并沉淀历史记录。
- `CommandInterceptor` 展示危险命令审批门禁。
- `ShellExecutor` 展示模拟命令执行、输出和 trace payload。
- 看板可点击查看 Agent 步骤、工具调用详情、SOP/技能文档、日志回放和修复历史。

如果你 fork 或部署本项目，在 GitHub 仓库的 **Settings -> Pages** 中选择 `main` 分支的 `/docs` 目录，即可发布：

```text
https://<your-github-user>.github.io/SysGuard/demo/
```

## 运行机制

### 核心流程

```text
Inspector 定时巡检
    ->
Monitor 生成健康报告
    ->
发现异常后通知 Coordinator
    ->
Remediator 检索 SOP / 历史记录
    ->
生成修复计划
    ->
dry-run / 危险命令审批
    ->
执行修复或记录计划
    ->
重新巡检验证
    ->
写入 trace 与历史记录
```

### 主要模块

```text
cmd/sysguard/main.go                 启动入口
internal/config                      配置加载
internal/monitor                     健康检查与异常构建
internal/agents/inspector            巡检 Agent
internal/agents/remediator           修复 Agent
internal/agents/coordinator          调度器
internal/rag/knowledge.go            SOP 知识库
internal/rag/history.go              历史修复记录
internal/security/interceptor.go     危险命令拦截
internal/observability/trace.go      trace 事件落盘
pkg/utils/shell.go                   Shell 执行器
```

## 快速开始

### 前置要求

- Go 1.21+
- Linux 或 macOS
- 可访问受管主机本地命令环境
- 如果要进行服务级修复：
  Linux 上建议有 `systemctl` 和 `journalctl`

### 1. 克隆仓库

```bash
git clone https://github.com/lyx516/SysGuard.git
cd SysGuard
```

### 2. 下载依赖

```bash
go mod download
```

### 3. 配置 SysGuard

编辑 [configs/config.yaml](/Users/liyuxuan/Desktop/SysGuard/configs/config.yaml)。

最小可用示例：

```yaml
monitor:
  check_interval: 30s
  health_threshold: 80
  cpu_threshold: 85
  memory_threshold: 90
  disk_threshold: 90

agents:
  inspector:
    interval: 30s
  remediator:
    command_timeout: 2m
    auto_approve_safe_commands: true
    allow_interactive_input: true
    dry_run: true
    verify_after_remediation: true

security:
  dangerous_commands:
    - rm
    - kill
    - dd
    - shutdown
    - reboot
    - systemctl stop
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
  api_key_env: OPENAI_API_KEY
  base_url: "https://api.openai.com/v1"
  timeout: 30s
  max_tokens: 2048
  temperature: 0.2

storage:
  history_path: "./data/history.json"

logging:
  output: "./logs/sysguard.log"

services:
  names:
    - nginx
    - redis
```

说明：

- `services.names` 为空时，不做受管服务检查。
- 默认 `dry_run: true`，只生成和记录修复计划，不真实执行命令。
- 如果将 `dry_run` 改为 `false`，Linux 上检测到服务异常时，会优先尝试 `journalctl` + `systemctl restart` 的修复流程。
- `ai.enabled` 当前默认关闭；API Key 请放在 `ai.api_key_env` 指向的环境变量里，例如 `OPENAI_API_KEY`。
- macOS 上默认只做进程存在性检测，不自动执行服务重启。

### 4. 构建

```bash
go build -o build/sysguard ./cmd/sysguard
go build -o build/sysguard-ui ./cmd/sysguard-ui
```

### 5. 运行

```bash
./build/sysguard
```

启动后，SysGuard 会：

- 周期性输出健康检查日志。
- 把结构化 trace 写入 `logs/trace.log`。
- 把运行日志写入 `logs/sysguard.log`。
- 把历史修复记录写入 `data/history.json`。

### 6. 启动图形化监控看板

SysGuard 提供了基于 Eino A2UI 数据流的监控看板。它不是聊天界面，而是面向运维值班人员的实时页面：

- 展示 `Coordinator`、`Inspector`、`Remediator` 的运行状态。
- 汇总 Eino callback / 工具调用链路、耗时和错误。
- 展示 CPU、内存、磁盘、服务检查结果。
- 读取 trace、运行日志和历史修复记录，形成时间线。
- 支持手动触发一次巡检。

构建并启动：

```bash
./build/sysguard-ui
```

然后打开：

```text
http://localhost:8080
```

如果配置了 `ui.auth_token` 或环境变量 `SYSGUARD_UI_AUTH_TOKEN`，访问 API/SSE/A2UI 时需要带上：

```bash
curl -H "Authorization: Bearer <token>" http://127.0.0.1:8080/api/snapshot
```

常用接口：

- `GET /api/snapshot`：当前 dashboard JSON 快照。
- `POST /api/check`：立即执行一次健康巡检并返回快照。
- `GET /api/stream`：Eino A2UI 风格的 SSE 实时数据流。
- `GET /a2ui/render`：返回当前 A2UI dashboard render tree 与数据模型。

## 配置说明

### `monitor`

- `check_interval`:
  巡检周期。
- `health_threshold`:
  总体健康分低于该阈值时触发异常。
- `cpu_threshold`:
  CPU 使用率阈值。
- `memory_threshold`:
  内存使用率阈值。
- `disk_threshold`:
  磁盘使用率阈值。

### `agents.inspector`

- `interval`:
  巡检 Agent 的执行周期，通常与 `monitor.check_interval` 保持一致。

### `agents.remediator`

- `command_timeout`:
  单条修复命令的执行超时。
- `auto_approve_safe_commands`:
  当前保留字段，安全命令默认直接执行。
- `allow_interactive_input`:
  是否允许在终端内进行审批确认。
- `dry_run`:
  是否只生成和记录修复计划而不执行命令。生产首次接入建议保持 `true`。
- `verify_after_remediation`:
  命令执行后是否重新巡检，只有验证通过才写入成功历史。

### `ui`

- `addr`:
  UI 监听地址。默认 `127.0.0.1:8080`，避免无意暴露到外网。
- `auth_token`:
  UI/API/SSE/A2UI 的 Bearer token。生产建议留空配置文件，并通过 `SYSGUARD_UI_AUTH_TOKEN` 注入。

### `ai`

- `enabled`:
  是否启用 AI 配置。当前版本只加载配置，不会默认调用模型。
- `provider`:
  模型服务商标识，例如 `openai`。
- `model`:
  模型名称，例如 `gpt-4.1-mini`。
- `api_key_env`:
  存放 API Key 的环境变量名。不要把真实 key 写入 YAML。
- `base_url`:
  模型服务 API 地址，兼容 OpenAI 风格接口时可改为代理或私有网关地址。
- `timeout`:
  单次 AI 请求超时时间。
- `max_tokens`:
  单次响应最大 token 数。
- `temperature`:
  生成随机性，运维场景建议保持较低值。

### `security`

- `dangerous_commands`:
  需要人工审批的危险命令前缀列表。
- `enable_approval`:
  是否启用审批。
- `approval_timeout`:
  审批等待超时。

### `knowledge_base`

- `docs_path`:
  SOP 文档目录，支持递归加载 `.md` 文件。

### `observability`

- `enable_tracing`:
  是否写 trace 事件。
- `trace_log_path`:
  trace 输出文件，JSON Lines 格式。

### `storage`

- `history_path`:
  历史修复记录存储位置。

### `logging`

- `output`:
  运行日志文件路径。

### `services`

- `names`:
  受管服务名列表。
  Linux 使用 `systemctl is-active` 检查，失败时回退到 `pgrep -x`。

## 知识库与修复策略

### SOP 文档

SOP 目录默认是 [docs/sop/example-sop.md](/Users/liyuxuan/Desktop/SysGuard/docs/sop/example-sop.md) 这一类 Markdown 文档。

推荐写法：

- 使用清晰标题描述问题类型。
- 用代码块列出可执行命令。
- 对命令中的变量使用 `<service_name>`、`<port>` 这类占位符。

SysGuard 会从代码块中抽取命令，并尝试用异常元数据替换占位符。

### 历史复用

当新异常与历史记录描述足够相似时，SysGuard 会优先复用历史成功步骤，而不是重新从 SOP 推导。

这意味着：

- 你的修复流程会随着运行次数逐步沉淀。
- 首次故障修复成功后，后续同类问题恢复速度会更快。

## 安全模型

SysGuard 默认不是“无条件自动执行器”，而是“带护栏的自动修复器”。

安全机制包括：

- 默认 dry-run，不直接执行修复命令。
- 危险命令前缀拦截。
- 审批超时自动拒绝。
- 无交互终端时拒绝需要审批的操作。
- 命令解析时过滤 `|`、`;`、`&`、重定向等高风险字符。
- 修复后重新巡检，验证失败不会被记录为成功修复。
- 审计历史以私有权限写入，trace 会脱敏常见 token/password/secret 字段。
- UI 默认只绑定本机地址，配置 token 后 API 需要 `Authorization: Bearer <token>`。

建议：

- 先在测试环境验证 SOP。
- 只把必要命令加入知识库。
- 谨慎维护 `dangerous_commands` 列表。
- 生产上线时先使用观察模式，再逐步开放审批执行。

### 生产运行模式

推荐按三个阶段接入：

1. **观察模式**：
   `dry_run: true`，只巡检、生成计划、记录历史，不执行命令。
2. **审批修复模式**：
   `dry_run: false`，保留 `enable_approval: true`，危险操作必须人工确认。
3. **受控无人值守模式**：
   仅对经过演练的低风险 SOP 开放自动执行，并保留 `verify_after_remediation: true`、日志审计和告警。

## 验证状态

当前仓库已验证：

```bash
go build ./...
go test ./...
```

并补充了以下基础测试：

- 配置解析测试
- dry-run 与修复后验证测试
- 历史记录持久化测试
- trace 脱敏与写入错误计数测试
- 危险命令识别测试
- UI 鉴权与 HTTP 方法限制测试

## Docker

仓库包含 [Dockerfile](/Users/liyuxuan/Desktop/SysGuard/Dockerfile)，可用于容器化构建：

```bash
docker build -t sysguard:latest .
docker run --rm -it sysguard:latest
```

注意：

- 容器模式下是否能检查宿主服务，取决于挂载和权限设计。
- 如果要让 SysGuard 管理宿主机服务，通常更适合直接部署为主机守护进程，而不是默认容器模式。
- 当前镜像默认使用非 root 用户运行，适合观察和演示；如果需要操作宿主服务，应改用 systemd 主机部署并显式配置最小权限。

## 部署建议

生产环境更推荐：

1. 以 systemd 或类似进程管理器运行。
2. 将 `configs/`、`docs/sop/`、`logs/`、`data/` 放到持久化目录。
3. 将历史和日志目录设置为 `sysguard` 用户可写，例如 `/var/lib/sysguard` 和 `/var/log/sysguard`。
4. 先从少量明确的受管服务开始接入，不要一开始就覆盖整台机器。
5. UI 保持 `127.0.0.1` 监听；远程访问通过反向代理、TLS 和统一身份认证暴露。

仓库提供 systemd 示例：

```bash
sudo install -D -m 0644 deploy/systemd/sysguard.service /etc/systemd/system/sysguard.service
sudo install -D -m 0644 deploy/systemd/sysguard-ui.service /etc/systemd/system/sysguard-ui.service
sudo systemctl daemon-reload
sudo systemctl enable --now sysguard
sudo systemctl enable --now sysguard-ui
```

## 局限与后续方向

当前版本已经可用，但仍有明确边界：

- 还没有外部告警通道集成。
- 还没有分布式多节点调度能力。
- 当前 RAG 仍是本地关键词召回，不是向量检索。
- 命令策略仍是字符串级护栏，后续应演进为结构化 Action、参数 schema、回滚动作和外部审批系统。
- 当前 UI/API 适合作为本机运维看板，远程生产暴露仍建议放在反向代理和统一身份认证之后。

这些能力适合作为下一阶段演进。

## License

MIT
