---
id: service-restart
risk_level: privileged
required_approval: true
signals:
  - service status is down
  - process is not found
diagnosis_steps:
  - check service status
  - inspect recent service logs
execution_steps:
  - stop the service only after approval
  - start the service after prechecks pass
verification_steps:
  - verify service status
  - inspect recent logs after restart
rollback_steps:
  - stop the faulty service
  - restore previous configuration backup
  - restart service and verify health
steps:
  - id: collect-status
    title: 服务状态检查
    type: diagnosis
    intent: 确认服务是否由 systemd 标记为 failed/inactive，并收集退出原因。
    tool: service-management
    action: status
    preconditions:
      - 已确认目标服务名来自受管服务配置或人工输入
      - 当前操作处于只读诊断阶段
    risks:
      - status 输出可能包含环境变量、路径或启动参数中的敏感信息
    verification:
      - systemctl status 能返回目标服务的当前状态
      - 异常状态、退出码或最近失败时间已被记录
    rollback:
      - 只读操作无需回滚
  - id: inspect-logs
    title: 检查近期服务日志
    type: diagnosis
    intent: 通过最近日志判断故障是否由配置、依赖、端口占用或资源不足引起。
    tool: service-management
    action: logs
    parameters:
      - lines=100
    preconditions:
      - 服务状态检查已完成
      - 操作者有读取 journal 的权限
    risks:
      - 日志可能包含 token、用户数据或内部地址
      - 日志量过大时可能影响响应时间
    verification:
      - 最近错误日志已归类为可解释原因或未知原因
      - 未发现必须先回滚配置的明显信号
    rollback:
      - 只读操作无需回滚
  - id: restart-service
    title: 审批后重启服务
    type: execution
    intent: 在诊断支持且风险可接受时，通过受控重启恢复服务。
    tool: service-management
    action: restart
    requires_approval: true
    preconditions:
      - 服务确认为 down 或无法健康响应
      - 最近日志没有显示重启会扩大故障的信号
      - 已获得生产环境审批
      - dry-run 状态和执行窗口已确认
    risks:
      - 现有连接可能被中断
      - 如果根因是配置错误，重启可能继续失败
      - 依赖服务未恢复时可能触发级联告警
    verification:
      - systemctl is-active 返回 active
      - 健康检查分数恢复到阈值以上
      - 重启后的日志没有新的 critical/error
    rollback:
      - 停止故障服务
      - 恢复最近一次已知可用配置备份
      - 再次启动服务并重新执行健康检查
---
# 服务重启标准作业程序 (SOP)

## 目的
规范化服务重启流程，确保服务重启的安全性和可靠性。

## 适用范围
适用于所有核心服务的重启操作。

## 前置条件
1. 确认服务状态异常
2. 获取必要的权限和审批
3. 通知相关人员

## 操作步骤

### 1. 服务状态检查
```bash
systemctl status <service_name>
```

### 2. 检查相关日志
```bash
journalctl -u <service_name> -n 100 --no-pager
```

### 3. 优雅停止服务
```bash
systemctl stop <service_name>
```

### 4. 等待服务完全停止
```bash
while systemctl is-active --quiet <service_name>; do sleep 1; done
```

### 5. 检查端口释放
```bash
netstat -tuln | grep <port>
```

### 6. 启动服务
```bash
systemctl start <service_name>
```

### 7. 验证服务状态
```bash
systemctl status <service_name>
```

### 8. 检查服务日志
```bash
journalctl -u <service_name> -f --no-pager
```

## 注意事项
1. 在生产环境执行前，必须获得审批
2. 重启操作应在低峰期进行
3. 如遇异常，立即回滚
4. 记录操作日志

## 应急回滚
如重启后服务异常，执行以下步骤：
1. 停止服务
2. 恢复配置备份
3. 重新启动服务
4. 验证功能正常

## 联系人
- 运维团队: ops@example.com
- 开发团队: dev@example.com
