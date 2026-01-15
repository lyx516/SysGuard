# SysGuard - 智能自动化运维与诊断 Agent

## 项目信息
- **时间周期**: 2024.03 - 至今
- **开发模式**: 独立设计开发
- **技术栈**: Go、Eino、RAG、Shell

## 项目背景
基于 Go 语言 Eino 框架构建的智能运维助手，集成实时监控、故障自愈与大日志分析能力。

## 核心功能

### 1. 双智能体协同
- **Inspector（巡检员）**: 实现高频健康度检查与结构化日志输出
- **Remediator（修复员）**: 在检测到异常时自动唤醒，自动维护、部署节点
- 智能协调机制实现两者之间的无缝协作

### 2. 安全防御与容错机制
- **高危命令拦截器**: 拦截 `rm`、`kill` 等危险命令
- **人工审批流程**: 强制关键操作需人工审批
- **容错中间件**: 自动捕获工具执行错误
- **Agent 自我纠错**: 保障生产环境安全

### 3. 确定性编排与 RAG 优化
- **日志分析图**: 基于工作流构建
- **大文件分块读取**: 避免大模型 Token 溢出
- **关键词过滤**: 提升日志分析效率
- **动态加载运维手册**: 支持 Markdown 格式
- **SOP 约束**: 强制 Agent 严格遵循标准作业程序

### 4. 上下文管理与可观测性
- **递归摘要机制**: 处理长对话，降低 Token 消耗
- **全局回调追踪**: 监控探针的下发与回收
- **完整工具调用链路**: 记录所有操作轨迹

## 项目结构

```
SysGuard/
├── cmd/                    # 命令行入口
│   └── sysguard/
│       └── main.go
├── internal/               # 内部模块
│   ├── agents/             # Agent 实现
│   │   ├── inspector/      # 巡检员 Agent
│   │   ├── remediator/     # 修复员 Agent
│   │   └── coordinator/    # 协调器
│   ├── rag/                # RAG 模块
│   │   ├── loader.go       # 文档加载器
│   │   ├── retriever.go    # 检索器
│   │   └── knowledge.go    # 知识库
│   ├── security/           # 安全模块
│   │   ├── interceptor.go  # 命令拦截器
│   │   ├── approval.go     # 审批流程
│   │   └── whitelist.go    # 命令白名单
│   ├── workflow/           # 工作流
│   │   ├── graph.go        # 日志分析图
│   │   ├── chunks.go       # 分块处理
│   │   └── filter.go       # 关键词过滤
│   ├── monitor/            # 监控模块
│   │   ├── health.go       # 健康检查
│   │   ├── logger.go       # 日志记录
│   │   └── probe.go        # 探针管理
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
│   └── manuals/            # 操作手册
├── go.mod
└── go.sum
```

## 技术亮点

1. **双 Agent 架构设计**: 实现监控与修复的职责分离
2. **生产安全机制**: 多层防护确保运维安全
3. **智能日志处理**: 优化大文件处理和上下文管理
4. **全链路可观测**: 完整的操作追踪和审计
