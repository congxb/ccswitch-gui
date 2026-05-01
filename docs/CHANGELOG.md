# CCSwitch GUI v1.1.0 更新说明

## 概述

本次更新主要解决了一键安装 Claude Code 流程中的多个关键问题，包括安装方式重构、Git 安装弹窗阻断、环境变量未生效等，同时新增 CLI 模式用于远程调试和安装日志记录。

---

## 改动详情

### 1. Node.js 安装方式重构（修复安装失败）

**问题**：原方案使用 winget 安装 Node.js MSI，在国内网络环境下经常失败（下载 0 字节、MSI 数据库损坏 error 2203）。

**修复**：改用 zip 包下载 + PowerShell 解压方案：
- 从 npmmirror（国内镜像）下载 Node.js LTS zip 包
- 通过 PowerShell `Expand-Archive` 解压到 `C:\` 根目录
- `Move-Item` 移动到 `C:\Program Files\nodejs`
- 写入 ps1 脚本文件执行，避免 shell 引号嵌套问题

**涉及文件**：`src/ui.go`（步骤 1）、`src/main.go`（CLI `cliInstall`）

### 2. Git for Windows 安装弹窗问题

**问题**：安装 Git 时，如果系统存在旧版本，安装程序会弹窗提示"无法卸载旧版本"，需要手动确认才能继续，导致自动化安装流程被阻断。

**修复**：
- 安装前先 `taskkill` 杀掉所有 Git 相关进程（git.exe、git-credential-manager.exe 等）
- 通过注册表查找旧版 Git 的卸载命令，以 `/VERYSILENT /SUPPRESSMSGBOXES` 参数静默卸载
- 新版 Git 安装命令增加 `/SUPPRESSMSGBOXES` 参数，抑制所有弹窗

**涉及文件**：`src/ui.go`（步骤 2）

### 3. Claude Code 环境变量未生效

**问题**：安装完成后，`where claude` 找不到命令，因为新添加的 PATH 条目未被当前进程和系统感知。

**修复**：
- 步骤 3 配置环境变量后，立即调用 `broadcastEnvChange()` 广播 `WM_SETTINGCHANGE` 通知
- 从注册表重新读取最新 PATH 并更新当前进程环境变量，确保后续 `exec.Command` 能找到 `claude.cmd`

**涉及文件**：`src/ui.go`（步骤 3→4 之间）

### 4. 新增安装日志文件记录

**问题**：安装过程中如果出错，错误信息只显示在 GUI 对话框中，无法通过远程 SSH 查看日志定位问题。

**修复**：
- 所有安装步骤的日志同时写入 `%TEMP%\ccswitch-install.log` 文件，每行带时间戳
- `runCmdAndWait` 的 stdout/stderr 输出也写入日志文件
- goroutine panic 信息同时写入 `ccswitch-install-panic.log` 和安装日志文件

**涉及文件**：`src/ui.go`（`appendLog`、panic defer）

### 5. 新增 CLI 模式（远程调试）

**问题**：通过 SSH 远程调试 GUI 安装流程非常困难，无法点击按钮触发安装。

**修复**：新增 `ccswitch-gui.exe -cli` 命令行模式，支持以下命令：
- `detect` — 检测 Node.js、npm、Git、Claude Code 安装状态
- `install` — 执行完整的一键安装流程
- `env` — 显示当前环境变量（PATH 等）
- `uninstall --force` — 卸载 Claude Code

**涉及文件**：`src/main.go`

### 6. 项目结构重组

**改动**：将源码从根目录迁移到 `src/` 子目录，规范化项目结构：

```
ccswitch-gui/
├── src/                    # Go 源码
│   ├── main.go
│   ├── config.go
│   ├── settings.go
│   ├── ui.go
│   ├── github.go
│   └── winpath.go
├── scripts/                # 辅助脚本
│   ├── 一键卸载Claude.bat      # 交互式卸载（带确认提示）
│   └── 自动清理测试环境.bat     # 一键清理（无交互）
├── assets/                 # 静态资源
│   └── screenshot.png
├── docs/                   # 文档
│   ├── README.md
│   └── CHANGELOG.md
├── go.mod
└── go.sum
```

### 7. 卸载脚本修复

**问题**：原卸载脚本使用 `rmdir /s /q` 删除 `C:\Program Files\nodejs` 时，因权限问题静默失败。

**修复**：
- 改用 PowerShell `Remove-Item -Recurse -Force` 替代 `rmdir`，更可靠
- 修复 Windows 下错误使用 `/dev/null`（Linux 语法）的问题，改为 `>nul 2>&1`
- PATH 清理增加 npm/nodejs 相关条目的模糊匹配过滤

**涉及文件**：`scripts/一键卸载Claude.bat`、`scripts/自动清理测试环境.bat`

### 8. 启动崩溃定位增强

**改动**：在 `main()` 函数中增加分步日志记录（`startLog`），写入 `%TEMP%\ccswitch-startup.log`，用于定位 GUI 启动闪退的具体位置。

**涉及文件**：`src/main.go`

---

## 文件变更清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `src/ui.go` | 修改 | 安装流程重构、日志记录、Git 弹窗修复、环境变量广播 |
| `src/main.go` | 修改 | 新增 CLI 模式、启动日志 |
| `src/winpath.go` | 修改 | 无代码变更，迁移至 src/ |
| `src/config.go` | 修改 | 无代码变更，迁移至 src/ |
| `src/settings.go` | 修改 | 无代码变更，迁移至 src/ |
| `src/github.go` | 修改 | 无代码变更，迁移至 src/ |
| `scripts/一键卸载Claude.bat` | 修改 | 修复删除命令和 PATH 清理 |
| `scripts/自动清理测试环境.bat` | 新增 | 无交互一键清理脚本 |
| `docs/README.md` | 新增 | 项目文档（从根目录迁移） |
| `docs/CHANGELOG.md` | 新增 | 本更新说明 |

---

## 待办 / 已知问题

- [ ] 步骤 1 中 `ensureInPath` 被调用了两次（重复），需要清理
- [ ] Git 安装卸载旧版本时，如果旧版安装目录不是默认路径，可能卸载不彻底
- [ ] CLI 模式下 Git 安装被跳过，需要手动处理
- [ ] 建议在 GitHub Releases 中提供独立的安装日志查看说明
