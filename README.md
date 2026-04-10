# SysGuard

智能自动化运维与诊断 Agent，面向“持续巡检 -> 异常发现 -> 安全修复 -> 历史沉淀”这条闭环。

<div align="center">

![Version](https://img.shields.io/badge/version-0.2.0-blue)
![License](https://img.shields.io/badge/license-MIT-green)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8)
![Status](https://img.shields.io/badge/status-production--ready-success)

[快速开始](#快速开始) • [运行机制](#运行机制) • [配置说明](#配置说明) • [部署建议](#部署建议)

</div>

## 项目简介

SysGuard 是一个用 Go 编写的运维守护进程。它会定期检查主机和受管服务的健康状态，当检测到异常时：

1. 生成结构化异常信息。
2. 从本地 SOP 知识库中检索相关修复文档。
3. 优先复用历史成功修复方案。
4. 按安全规则执行修复命令。
5. 记录 trace 和历史修复结果，供后续复用与审计。

当前版本已经实现了可运行的生产主链路，而不只是概念原型。

## 当前能力

- 双 Agent 协同：
  `Inspector` 负责定时巡检，`Remediator` 负责修复，`Coordinator` 负责调度。
- 真实健康检查：
  支持 CPU、内存、磁盘、网络和受管服务状态检查，不再使用固定模拟数据。
- SOP/RAG 检索：
  从 `docs/sop` 目录加载 Markdown 运维文档，并基于关键词召回相关 SOP。
- 历史知识库：
  成功修复会写入本地历史记录，后续相似故障优先复用历史步骤。
- 安全执行：
  危险命令拦截、交互式人工审批、审批超时自动拒绝。
- 可观测性：
  trace 事件和运行日志落盘，便于审计和排障。
- 优雅停止：
  支持 `SIGINT` / `SIGTERM`，可安全关闭巡检和修复流程。

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
危险命令审批
    ->
执行修复
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
- Linux 上检测到服务异常时，会优先尝试 `journalctl` + `systemctl restart` 的修复流程。
- macOS 上默认只做进程存在性检测，不自动执行服务重启。

### 4. 构建

```bash
go build -o build/sysguard ./cmd/sysguard
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

- 危险命令前缀拦截。
- 审批超时自动拒绝。
- 无交互终端时拒绝需要审批的操作。
- 命令解析时过滤 `|`、`;`、`&`、重定向等高风险字符。
- 审计日志和 trace 落盘。

建议：

- 先在测试环境验证 SOP。
- 只把必要命令加入知识库。
- 谨慎维护 `dangerous_commands` 列表。

## 验证状态

当前仓库已验证：

```bash
go build ./...
go test ./...
```

并补充了以下基础测试：

- 配置解析测试
- 历史记录持久化测试
- 危险命令识别测试

## Docker

仓库包含 [Dockerfile](/Users/liyuxuan/Desktop/SysGuard/Dockerfile)，可用于容器化构建：

```bash
docker build -t sysguard:latest .
docker run --rm -it sysguard:latest
```

注意：

- 容器模式下是否能检查宿主服务，取决于挂载和权限设计。
- 如果要让 SysGuard 管理宿主机服务，通常更适合直接部署为主机守护进程，而不是默认容器模式。

## 部署建议

生产环境更推荐：

1. 以 systemd 或类似进程管理器运行。
2. 将 `configs/`、`docs/sop/`、`logs/`、`data/` 放到持久化目录。
3. 将审批终端接入值班环境，避免高危命令因无交互输入而被自动拒绝。
4. 先从少量明确的受管服务开始接入，不要一开始就覆盖整台机器。

## 局限与后续方向

当前版本已经可用，但仍有明确边界：

- 还没有 HTTP API 或 Web 控制台。
- 还没有外部告警通道集成。
- 还没有分布式多节点调度能力。
- 当前 RAG 仍是本地关键词召回，不是向量检索。

这些能力适合作为下一阶段演进。

## License

MIT
