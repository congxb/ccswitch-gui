# 贡献指南

感谢你对 CCSwitch GUI 的关注！欢迎提交 Issue 和 PR。

## 开发环境

- [Go 1.21+](https://go.dev/dl/)
- [MinGW-w64 GCC](https://www.mingw-w64.org/)（Fyne GUI 编译需要）

## 构建

```bash
go mod tidy
CGO_ENABLED=1 go build -ldflags "-s -w -H windowsgui" -o ccswitch-gui.exe .
```

## 提交 PR

1. Fork 本仓库
2. 创建特性分支：`git checkout -b feature/your-feature`
3. 提交改动：`git commit -m "feat: 描述你的改动"`
4. 推送分支：`git push origin feature/your-feature`
5. 提交 Pull Request

## 提交信息规范

| 前缀 | 用途 |
|------|------|
| `feat:` | 新功能 |
| `fix:` | 修复 Bug |
| `docs:` | 文档更新 |
| `refactor:` | 代码重构 |
| `chore:` | 构建/工具变更 |

## Issue 反馈

提交 Issue 时请附上：
- 操作系统版本
- 问题复现步骤
- 期望行为 vs 实际行为
- 相关截图或日志
