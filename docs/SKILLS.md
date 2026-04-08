# Skills 文档

SysGuard 基于 LangChain Skills 框架设计，提供丰富的运维自动化能力。

## 架构设计

### Skill 概念

**Skill** 是可复用的 Agent 能力单元，包含：
- 一组工具（Tools）
- 使用这些工具的说明/提示词
- 可选的推理模式

### 设计参考

本项目参考了以下主流 AI Skills 平台的架构：

1. **[LangChain Skills Framework](https://python.langchain.com/docs/latest/modules/agents/tools/)** - 2025-2026 最新架构
2. **OpenAI Function Calling** - 原生工具调用
3. **CrewAI** - 多 Agent 协作框架
4. **AutoGen** - Microsoft 多 Agent 框架
5. **LlamaIndex Tools** - 数据连接器和工具

## 内置 Skills

### 1. 日志分析 (log_analysis)

分析日志文件，提取关键信息和异常模式。

**工具**:
- `log_reader` - 读取和解析日志文件
- `keyword_filter` - 根据关键词过滤日志
- `pattern_matcher` - 正则表达式模式匹配

**功能**:
- 大文件分块读取
- 关键词过滤
- 异常模式识别
- 统计分析

**使用示例**:
```go
input := &skills.SkillInput{
    Params: map[string]interface{}{
        "file_path": "/var/log/syslog",
    },
}
output, _ := logAnalysisSkill.Execute(ctx, input)
```

### 2. 健康检查 (health_check)

检查系统各组件的健康状态。

**工具**:
- `cpu_checker` - CPU 使用率检查
- `memory_checker` - 内存使用情况
- `disk_checker` - 磁盘空间检查
- `network_checker` - 网络连通性检查
- `service_checker` - 服务状态检查

**功能**:
- 全面的系统健康度评估
- 组件级别监控
- 阈值告警

**使用示例**:
```go
input := &skills.SkillInput{
    Params: map[string]interface{}{
        "check_type": "full",
    },
}
output, _ := healthCheckSkill.Execute(ctx, input)
```

### 3. 服务管理 (service_management)

管理系统服务的启动、停止、重启和状态查询。

**工具**:
- `systemctl_start` - 启动服务
- `systemctl_stop` - 停止服务
- `systemctl_restart` - 重启服务
- `systemctl_status` - 查询状态
- `systemctl_list` - 列出所有服务

**功能**:
- systemd 服务管理
- 批量操作
- 状态监控

**使用示例**:
```go
input := &skills.SkillInput{
    Params: map[string]interface{}{
        "action":  "restart",
        "service": "nginx",
    },
}
output, _ := serviceManagementSkill.Execute(ctx, input)
```

### 4. 告警 (alerting)

发送和管理告警通知。

**工具**:
- `email_notifier` - 邮件通知
- `slack_notifier` - Slack 通知
- `webhook_notifier` - Webhook 通知

**功能**:
- 多渠道告警
- 告警级别管理
- 告警历史查询

**使用示例**:
```go
input := &skills.SkillInput{
    Params: map[string]interface{}{
        "action": "send",
        "level":  "critical",
        "title":  "CPU High",
        "message": "CPU usage exceeded 90%",
    },
}
output, _ := alertingSkill.Execute(ctx, input)
```

### 5. 指标收集 (metrics)

收集和查询系统指标数据。

**工具**:
- `prometheus` - Prometheus 指标
- `node_exporter` - 系统指标
- `custom_collector` - 自定义指标

**功能**:
- 实时指标收集
- 历史数据查询
- 指标导出

**使用示例**:
```go
input := &skills.SkillInput{
    Params: map[string]interface{}{
        "action": "collect",
        "types": []string{"cpu", "memory", "disk"},
    },
}
output, _ := metricsSkill.Execute(ctx, input)
```

### 6. 网络诊断 (network_diagnosis)

诊断网络连接和性能问题。

**工具**:
- `ping` - 连通性测试
- `traceroute` - 路径追踪
- `nslookup/dig` - DNS 查询
- `netstat` - 网络状态
- `iperf` - 带宽测试

**功能**:
- 网络连通性检查
- 延迟测试
- 带宽测试
- DNS 解析测试

**使用示例**:
```go
input := &skills.SkillInput{
    Params: map[string]interface{}{
        "type":   "latency",
        "target": "8.8.8.8",
    },
}
output, _ := networkDiagnosisSkill.Execute(ctx, input)
```

### 7. 容器管理 (container_management)

管理 Docker 和 Kubernetes 容器。

**工具**:
- `docker_*` - Docker 操作
- `kubectl_*` - Kubernetes 操作

**功能**:
- 容器生命周期管理
- Pod 管理
- 服务部署
- 日志查看

**使用示例**:
```go
input := &skills.SkillInput{
    Params: map[string]interface{}{
        "action":   "list",
        "type":     "docker",
    },
}
output, _ := containerManagementSkill.Execute(ctx, input)
```

### 8. 数据库操作 (database_operations)

执行数据库查询和管理操作。

**工具**:
- `mysql_client` - MySQL 客户端
- `postgresql_client` - PostgreSQL 客户端
- `redis_client` - Redis 客户端
- `mongodb_client` - MongoDB 客户端

**功能**:
- SQL 查询执行
- 数据库备份/恢复
- 状态监控

**使用示例**:
```go
input := &skills.SkillInput{
    Params: map[string]interface{}{
        "action":  "query",
        "db_type": "mysql",
        "query":   "SELECT * FROM users LIMIT 10",
    },
}
output, _ := databaseOperationsSkill.Execute(ctx, input)
```

### 9. 文件操作 (file_operations)

安全的文件操作和管理。

**工具**:
- `file_reader` - 文件读取
- `file_writer` - 文件写入
- `directory_lister` - 目录列表
- `file_searcher` - 文件搜索
- `file_copier` - 文件复制
- `file_mover` - 文件移动
- `file_deleter` - 文件删除

**功能**:
- 文件读写
- 目录操作
- 文件搜索
- 权限检查

**使用示例**:
```go
input := &skills.SkillInput{
    Params: map[string]interface{}{
        "action":    "read",
        "file_path": "/var/log/syslog",
        "lines":     100,
    },
}
output, _ := fileOperationsSkill.Execute(ctx, input)
```

### 10. 通知 (notification)

发送各种类型的通知消息。

**工具**:
- `smtp_client` - SMTP 邮件
- `slack_webhook` - Slack Webhook
- `http_client` - HTTP Webhook
- `sms_provider` - 短信服务
- `telegram_bot` - Telegram Bot
- `discord_webhook` - Discord Webhook

**功能**:
- 多渠道通知
- 模板支持
- 重试机制

**使用示例**:
```go
input := &skills.SkillInput{
    Params: map[string]interface{}{
        "type":    "slack",
        "channel": "#alerts",
        "message": "System alert",
    },
}
output, _ := notificationSkill.Execute(ctx, input)
```

## Skill 注册表

### 创建默认注册表

```go
registry := skills.NewDefaultRegistry()
```

### 注册自定义 Skill

```go
registry := skills.NewSkillRegistry()
registry.Register(myCustomSkill)
```

### 查询 Skills

```go
// 列出所有 Skills
allSkills := registry.List()

// 按类别查询
monitoringSkills := registry.Search("monitoring", nil)

// 按标签查询
networkSkills := registry.Search("", []string{"network"})

// 获取指定 Skill
skill, ok := registry.Get("log_analysis")
```

## 创建自定义 Skill

```go
type MyCustomSkill struct {
    version string
}

func (s *MyCustomSkill) Name() string {
    return "my_custom_skill"
}

func (s *MyCustomSkill) Description() string {
    return "My custom skill description"
}

func (s *MyCustomSkill) Execute(ctx context.Context, input *skills.SkillInput) (*skills.SkillOutput, error) {
    // 实现执行逻辑
    return &skills.SkillOutput{
        Success: true,
        Message: "Operation completed",
        Data:    map[string]interface{}{},
    }, nil
}

func (s *MyCustomSkill) Tools() []skills.Tool {
    return []skills.Tool{
        &MyTool{},
    }
}

func (s *MyCustomSkill) Metadata() *skills.SkillMetadata {
    return &skills.SkillMetadata{
        Version:     s.version,
        Category:    "custom",
        Tags:        []string{"custom", "automation"},
        Author:      "Your Name",
        Permissions: []string{},
    }
}
```

## 技术栈

- **Go 1.21+**
- **Eino 框架** - Agent 框架
- **LangChain 风格架构** - Skills 框架设计参考

## 参考资料

- [LangChain Tools Documentation](https://python.langchain.com/docs/latest/modules/agents/tools/)
- [LangChain 2025-2026 Roadmap](https://www.analyticsvidhya.com/blog/2024/12/langchain-2025-roadmap)
- [OpenAI Function Calling](https://platform.openai.com/docs/guides/function-calling)
- [CrewAI](https://www.crewai.com/)
- [AutoGen](https://microsoft.github.io/autogen/)
- [LlamaIndex](https://docs.llamaindex.ai/en/stable/)
