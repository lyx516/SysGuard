# SysGuard - 智能自动化运维与诊断 Agent

<div align="center">

![Version](https://img.shields.io/badge/version-0.1.0-blue)
![License](https://img.shields.io/badge/license-MIT-green)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8)
![Status](https://img.shields.io/badge/status-stable-success)

基于 Go 语言构建的智能运维助手，集成实时监控、故障自愈与大日志分析能力

[快速开始](#快速开始) • [功能特性](#核心功能) • [架构设计](#架构设计) • [配置说明](#配置说明)

</div>

---

## 📋 项目信息

- **项目名称**: SysGuard
- **版本**: v0.1.0
- **时间周期**: 2026.01 - 至今
- **开发模式**: 独立设计开发
- **技术栈**: Go 1.21+、Eino、RAG、Shell
- **许可证**: MIT License

## 🌟 项目简介

SysGuard 是一个基于 Go 语言构建的智能运维助手，采用双智能体架构，集成实时监控、故障自愈与大日志分析能力。系统通过 RAG（检索增强生成）技术从运维知识库中检索相关信息，结合历史记录学习能力，实现智能化的运维问题诊断与自动修复。

### 核心优势

- 🤖 **双智能体协同**: Inspector 负责监控巡检，Remediator 负责故障修复
- 🛡️ **多层安全防护**: 高危命令拦截、人工审批流程、容错机制
- 📚 **知识驱动**: 基于 RAG 的知识检索，强制遵循 SOP 标准
- 🧠 **学习能力**: 历史记录学习，持续优化修复策略
- 🔍 **全链路可观测**: 完整的操作追踪与审计日志

## 🚀 快速开始

### 前置要求

- **Go**: 1.21 或更高版本
- **Docker**: (可选) 用于容器化部署
- **LLM API**: OpenAI API 密钥或兼容的 LLM 服务

### 安装步骤

#### 1. 克隆仓库

```bash
git clone https://github.com/lyx516/SysGuard.git
cd SysGuard
```

#### 2. 安装依赖

```bash
go mod download
```

#### 3. 配置环境

编辑 `configs/config.yaml` 配置文件：

```yaml
# 监控配置
monitor:
  check_interval: 30s
  health_threshold: 80.0

# 安全配置
security:
  dangerous_commands:
    - rm
    - kill
    - killall
  enable_approval: true
  approval_timeout: 5m
```

#### 4. 启动服务

```bash
# 构建项目
make build

# 运行 SysGuard
./bin/sysguard
```

#### 5. Docker 部署 (可选)

```bash
# 构建镜像
docker build -t sysguard:latest .

# 运行容器
docker run -d \
  --name sysguard \
  -v $(pwd)/configs:/app/configs \
  -v $(pwd)/logs:/app/logs \
  sysguard:latest
```

## 🏗️ 项目结构

```
SysGuard/
├── cmd/                    # 命令行入口
│   └── sysguard/
│       └── main.go         # 主程序入口
├── internal/               # 内部模块
│   ├── agents/             # Agent 实现
│   │   ├── inspector/      # 巡检员 Agent
│   │   ├── remediator/     # 修复员 Agent
│   │   └── coordinator/    # 协调器
│   ├── rag/                # RAG 模块
│   │   ├── knowledge.go    # 知识库
│   │   └── history.go     # 历史记录管理
│   ├── security/           # 安全模块
│   │   ├── interceptor.go  # 命令拦截器
│   │   └── approval.go     # 审批流程
│   ├── workflow/           # 工作流
│   │   ├── graph.go        # 日志分析图
│   │   ├── chunks.go       # 分块处理
│   │   └── filter.go       # 关键词过滤
│   ├── monitor/            # 监控模块
│   │   ├── health.go       # 健康检查
│   │   └── logger.go       # 日志记录
│   ├── skills/             # Skills 框架
│   │   ├── registry.go     # 技能注册表
│   │   └── remediation_workflow.go  # 修复工作流
│   ├── context/            # 上下文管理
│   │   ├── summary.go      # 递归摘要
│   │   └── manager.go      # 上下文管理器
│   └── observability/      # 可观测性
│       ├── trace.go        # 追踪
│       └── callback.go     # 回调管理
├── pkg/                    # 公共包
│   ├── middleware/         # 中间件
│   │   ├── error.go        # 错误处理
│   │   └── recovery.go     # 容错
│   └── utils/              # 工具函数
│       └── shell.go        # Shell 工具
├── configs/                # 配置文件
│   └── config.yaml
├── docs/                   # 运维手册
│   ├── sop/                # SOP 文档
│   │   └── example-sop.md
│   └── history/            # 历史记录
├── logs/                   # 日志输出
├── go.mod
└── go.sum
```

## 🎯 核心功能

### 1. 双智能体协同

#### Inspector (巡检员)
- **职责**: 实时系统监控与健康检查
- **功能**:
  - 高频健康度检查 (默认30秒间隔)
  - 结构化日志输出与分析
  - 异常检测与告警
  - 系统状态探针管理
- **特点**: 轻量级、高频率、低侵入

#### Remediator (修复员)
- **职责**: 异常修复与问题解决
- **功能**:
  - 基于异常自动唤醒
  - 三步修复流程：分析→执行→文档化
  - 历史学习与智能决策
  - 节点维护与部署
- **特点**: 智能化、安全优先、可学习

#### Coordinator (协调器)
- **职责**: 智能体之间的协调管理
- **功能**:
  - 异常回调处理
  - 智能体生命周期管理
  - 工作流调度
  - 状态同步

### 2. 安全防御与容错机制

#### 安全层次
```
┌─────────────────────────────────────┐
│   人工审批流程 (Human Approval)     │
├─────────────────────────────────────┤
│   高危命令拦截 (Command Interceptor)│
├─────────────────────────────────────┤
│   容错中间件 (Fault Tolerance)      │
├─────────────────────────────────────┤
│   Agent 自我纠错 (Self-Correction) │
└─────────────────────────────────────┘
```

**安全特性**:
- **高危命令拦截器**: 自动识别并拦截 `rm`、`kill`、`killall`、`dd`、`mkfs` 等危险命令
- **人工审批流程**: 强制关键操作需人工审批，支持超时自动拒绝
- **容错中间件**: 自动捕获工具执行错误，支持自动重试和熔断
- **Agent 自我纠错**: 通过反馈机制保障生产环境安全

### 3. 三步修复流程

#### Step 1: 问题分析 (Problem Analysis)
```go
analysis := &ProblemAnalysis{
    ProblemType:  "ServiceFailure",
    Description:  "Web service not responding",
    RootCause:    "Process crash detected",
    Severity:     "high",
    Environment:  "Production",
}

// 调用 skills 进行深度分析
healthCheck := registry.Execute("health-check", input)
logAnalysis := registry.Execute("log-analysis", input)
metricsAnalysis := registry.Execute("metrics-collection", input)
```

#### Step 2: 计划执行 (Plan Execution)
```go
// 搜索历史相似问题
similarRecords := historyKB.SearchSimilarRecords(description, 0.8)

// 如果有历史记录，复用成功方案
if len(similarRecords) > 0 {
    plan := adaptFromHistory(similarRecords[0])
    executePlan(plan)
} else {
    // 使用默认修复策略
    plan := createDefaultRemediationPlan()
    executePlan(plan)
}
```

#### Step 3: 文档生成 (Documentation)
```go
// 首次处理问题，生成文档
if len(similarRecords) == 0 && result.Success {
    record := &HistoryRecord{
        ProblemType:  problemType,
        Description:  description,
        RootCause:    rootCause,
        Solution:     solution,
        Steps:        executedSteps,
        Success:      true,
        Timestamp:    time.Now(),
    }
    historyKB.AddRecord(record)
}
```

### 4. Skills 框架

SysGuard 提供了10个生产级别的 Skills，涵盖运维主要场景：

| Skill | 功能描述 | 用途 |
|-------|---------|------|
| **SystemMonitoring** | 实时系统健康检查 | CPU、内存、磁盘、网络监控 |
| **LogAnalysis** | 智能日志解析分析 | 日志关键词过滤、异常检测 |
| **ContainerManagement** | Docker容器操作 | 容器启停、状态查询 |
| **ServiceManagement** | 服务生命周期管理 | 服务启停、重启、状态监控 |
| **MetricsCollection** | 性能指标收集 | 时序数据收集与分析 |
| **AlertEvaluation** | 告警规则评估 | 阈值检测、告警触发 |
| **HealthCheck** | 综合健康诊断 | 端到端健康检查 |
| **RestartService** | 服务重启能力 | 优雅重启服务 |
| **CleanResources** | 资源清理操作 | 临时文件、缓存清理 |
| **ExecuteShell** | 安全Shell执行 | 安全命令执行框架 |

### 5. 上下文管理与可观测性

#### 递归摘要机制
```
Long Conversation (10000+ tokens)
    ↓
Recursive Summary (500 tokens)
    ↓
Compressed Context + Latest Messages
```

**优势**:
- 降低 Token 消耗
- 保留关键信息
- 提升响应速度

#### 全局回调追踪
```go
callbackID := obs.OnCallbackStarted("Remediator.remediate")
// ... 执行操作
obs.OnCallbackCompleted(callbackID, result)
```

**追踪内容**:
- 探针下发与回收
- 工具调用链路
- 异常发生时间点
- 性能指标统计

## 🏛️ 架构设计

### 系统架构图

```
┌─────────────────────────────────────────────────────────────┐
│                    SysGuard System                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────┐         ┌──────────────┐               │
│  │  Inspector   │         │  Remediator  │               │
│  │  (巡检员)    │         │  (修复员)    │               │
│  └──────┬───────┘         └──────┬───────┘               │
│         │                         │                         │
│         │   Anomaly              │                         │
│         │◄──────────────────────│                         │
│         │                         │                         │
│         ▼                         ▼                         │
│  ┌─────────────────────────────────────────┐               │
│  │         Coordinator (协调器)           │               │
│  └──────────────┬──────────────────────┘               │
│                 │                                      │
└─────────────────┼──────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────────┐
│                    Core Components                        │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐│
│  │Skills Registry│  │History Knowledge│ │RAG Knowledge ││
│  │  (技能注册)  │  │Base (历史记录) │ │Base (SOP)   ││
│  └──────────────┘  └──────────────┘  └──────────────┘│
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐│
│  │   Security   │  │   Monitor    │  │Observability ││
│  │  (安全模块)  │  │  (监控模块)  │  │  (可观测性)  ││
│  └──────────────┘  └──────────────┘  └──────────────┘│
└─────────────────────────────────────────────────────────────┘
```

### 数据流图

```
┌─────────┐
│Monitor  │ 定期检查
└────┬────┘
     │
     ▼
┌─────────┐
│Inspector│ 检测异常
└────┬────┘
     │
     ▼
┌─────────────┐
│Coordinator  │ 触发修复
└────┬────────┘
     │
     ▼
┌───────────────────────┐
│   Remediator         │
│ ┌─────────────────┐   │
│ │1. Problem Analysis│   │
│ │  - Health Check  │   │
│ │  - Log Analysis │   │
│ │  - Metrics      │   │
│ └────────┬────────┘   │
│          ▼            │
│ ┌─────────────────┐   │
│ │2. Plan Execution│   │
│ │  - Search History│   │
│ │  - Execute Plan │   │
│ │  - Safety Check │   │
│ └────────┬────────┘   │
│          ▼            │
│ ┌─────────────────┐   │
│ │3. Documentation│   │
│ │  - Record Issue │   │
│ │  - Save Solution│   │
│ └────────┬────────┘   │
└──────────┼────────────┘
           │
           ▼
    ┌──────────────┐
    │  History KB  │ 学习记录
    └──────────────┘
```

## ⚙️ 配置说明

### 完整配置文件示例

```yaml
# SysGuard Configuration

# 监控配置
monitor:
  check_interval: 30s           # 检查间隔
  health_threshold: 80.0        # 健康度阈值
  probe_timeout: 10s            # 探针超时时间

# 日志分析配置
log_analysis:
  chunk_size: 1000             # 分块大小
  keywords:                     # 关键词过滤
    - error
    - failed
    - warning
    - critical
    - exception
    - timeout
  max_file_size: 100MB         # 最大文件大小

# Agent 配置
agents:
  inspector:
    interval: 30s              # 巡检间隔
    max_retries: 3              # 最大重试次数
    timeout: 5m                 # 超时时间

  remediator:
    auto_approve_safe_commands: true  # 自动审批安全命令
    max_retries: 3                    # 最大重试次数
    step_timeout: 5m                  # 单步超时
    enable_history_learning: true       # 启用历史学习

# 安全配置
security:
  dangerous_commands:         # 高危命令列表
    - rm
    - kill
    - killall
    - dd
    - mkfs
    - shutdown
    - reboot

  enable_approval: true       # 启用审批
  approval_timeout: 5m        # 审批超时
  enable_interceptor: true    # 启用命令拦截

# 知识库配置
knowledge_base:
  docs_path: "./docs/sop"      # SOP 文档路径
  reload_interval: 1h          # 重载间隔
  similarity_threshold: 0.8     # 相似度阈值

  history:
    enable: true               # 启用历史记录
    storage_path: "./docs/history"
    max_records: 1000          # 最大记录数

# 上下文管理配置
context:
  max_tokens: 8000            # 最大 Token 数
  summary_threshold: 7000     # 摘要阈值
  keep_recent_messages: 5      # 保留最近消息数

# 可观测性配置
observability:
  enable_tracing: true         # 启用追踪
  trace_log_path: "./logs/trace.log"
  enable_probes: true          # 启用探针
  metrics_interval: 1m         # 指标采集间隔

# 容错配置
fault_tolerance:
  enable_retry: true           # 启用重试
  max_retry_attempts: 3        # 最大重试次数
  retry_delay: 1s             # 重试延迟
  enable_circuit_breaker: true  # 启用熔断器
  circuit_breaker_threshold: 5  # 熔断阈值

# 日志配置
logging:
  level: info                 # 日志级别: debug, info, warn, error
  format: json               # 日志格式: json, text
  output: ./logs/sysguard.log
  rotation:
    max_size: 100MB
    max_age: 30d
    max_backups: 10
```

### 环境变量

```bash
# LLM API 配置
export OPENAI_API_KEY="your-api-key"
export OPENAI_BASE_URL="https://api.openai.com/v1"

# 数据库配置 (可选)
export DATABASE_URL="postgresql://user:pass@localhost:5432/sysguard"

# Redis 配置 (可选)
export REDIS_URL="redis://localhost:6379"
export REDIS_PASSWORD=""

# 日志配置
export LOG_LEVEL="info"
export LOG_OUTPUT="./logs/sysguard.log"
```

## 📝 使用示例

### 1. 基本监控

```go
package main

import (
    "context"
    "log"

    "github.com/sysguard/sysguard/internal/monitor"
    "github.com/sysguard/sysguard/internal/observability"
)

func main() {
    ctx := context.Background()

    // 创建监控器
    monitor := monitor.NewMonitor()

    // 注册健康检查
    monitor.RegisterHealthCheck("web-service", func() error {
        // 检查 Web 服务状态
        return checkWebService()
    })

    monitor.RegisterHealthCheck("database", func() error {
        // 检查数据库状态
        return checkDatabase()
    })

    // 启动监控
    if err := monitor.Start(ctx); err != nil {
        log.Fatal(err)
    }

    // 等待异常
    anomaly := <-monitor.AnomalyChannel()
    log.Printf("Anomaly detected: %v", anomaly)
}
```

### 2. 自定义 Skill

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/sysguard/sysguard/internal/skills"
)

// CustomSkill 自定义技能
type CustomSkill struct {
    name string
}

func (s *CustomSkill) Name() string {
    return s.name
}

func (s *CustomSkill) Description() string {
    return "Custom skill for specific operations"
}

func (s *CustomSkill) Execute(ctx context.Context, input *skills.SkillInput) (*skills.SkillOutput, error) {
    // 执行自定义逻辑
    result := performCustomOperation(input.Params)

    return &skills.SkillOutput{
        Success: true,
        Result:  result,
    }, nil
}

func main() {
    registry := skills.NewSkillRegistry()

    // 注册自定义技能
    customSkill := &CustomSkill{name: "custom-operation"}
    if err := registry.Register(customSkill); err != nil {
        log.Fatal(err)
    }

    // 执行技能
    output, err := registry.Execute(ctx, "custom-operation", &skills.SkillInput{
        Params: map[string]interface{}{
            "param1": "value1",
        },
    })

    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Result: %v", output.Result)
}
```

### 3. 添加自定义 SOP

在 `docs/sop/` 目录下创建 Markdown 文件：

```markdown
# Web 服务故障处理

## 问题识别
- Web 服务无响应
- HTTP 5xx 错误率 > 5%
- 响应时间 > 5s

## 分析步骤
1. 检查服务进程状态
2. 查看应用日志
3. 检查资源使用情况
4. 检查依赖服务状态

## 修复步骤
1. 如果进程停止，重启服务
2. 如果资源不足，扩容或优化
3. 如果依赖服务故障，联系相关团队
4. 监控修复效果

## 验证步骤
1. 检查服务健康状态
2. 验证正常请求响应
3. 检查错误率是否下降
```

## 🐛 故障排除

### 常见问题

#### 1. Inspector 检测不到异常

**可能原因**:
- 检查间隔设置过长
- 健康检查逻辑不正确
- 权限不足

**解决方案**:
```yaml
monitor:
  check_interval: 10s  # 减小检查间隔
  health_threshold: 70.0  # 降低阈值
```

#### 2. Remediator 修复失败

**可能原因**:
- 技能未正确注册
- 命令被拦截
- 审批超时

**解决方案**:
```yaml
remediator:
  auto_approve_safe_commands: false  # 手动审批
  max_retries: 5  # 增加重试次数
```

#### 3. 内存占用过高

**可能原因**:
- 历史记录过多
- 日志文件过大
- 上下文未及时清理

**解决方案**:
```yaml
knowledge_base:
  history:
    max_records: 500  # 减少历史记录数

context:
  max_tokens: 5000  # 减少最大 Token 数
```

## 📊 性能指标

| 指标 | 说明 | 目标值 |
|------|------|--------|
| Inspector 延迟 | 异常检测延迟 | < 1s |
| Remediator 响应时间 | 修复响应时间 | < 30s |
| 历史检索准确率 | 相似问题检索准确率 | > 85% |
| 命令拦截率 | 高危命令拦截成功率 | 100% |
| Token 消耗 | 平均对话 Token 消耗 | < 8000 |

## 🤝 贡献指南

欢迎贡献代码、文档、问题报告！

- 查看 [CONTRIBUTING.md](CONTRIBUTING.md) 了解贡献流程
- 遵循 MIT License 协议
- 代码风格遵循 Go 标准

## 📜 许可证

本项目采用 MIT License 许可证 - 详见 [LICENSE](LICENSE) 文件

## 📞 联系方式

- **GitHub Issues**: [提交问题](https://github.com/lyx516/SysGuard/issues)
- **Email**: support@sysguard.dev
- **Discussions**: [参与讨论](https://github.com/lyx516/SysGuard/discussions)

---

<div align="center">

**如果觉得项目有用，请给个 Star ⭐**

Made with ❤️ by SysGuard Team

</div>
