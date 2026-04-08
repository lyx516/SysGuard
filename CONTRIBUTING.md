# 贡献指南

感谢您对 SysGuard 项目的关注！我们欢迎任何形式的贡献。

## 如何贡献

### 报告问题
如果您发现了 bug 或者有功能建议，请：
1. 检查是否已经存在相关 issue
2. 创建新的 issue，详细描述问题或建议
3. 提供复现步骤、环境信息等

### 提交代码
1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

### 代码规范
- 遵循 Go 语言代码规范
- 运行 `make fmt` 格式化代码
- 运行 `make vet` 和 `make lint` 检查代码质量
- 为新功能添加测试

### 提交信息规范
- 使用清晰的提交信息
- feat: 新功能
- fix: 修复 bug
- docs: 文档更新
- style: 代码格式调整
- refactor: 重构
- test: 测试相关
- chore: 构建/工具相关

## 开发环境设置

```bash
# 克隆仓库
git clone https://github.com/lyx516/SysGuard.git
cd SysGuard

# 安装依赖
make deps

# 运行测试
make test

# 构建
make build
```

## 许可证

通过贡献代码，您同意您的贡献将根据 MIT 许可证进行许可。
