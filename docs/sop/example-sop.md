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
