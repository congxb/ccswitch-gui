# CCSwitch GUI

<p align="center">
   <img src="screenshot.png" alt="CCSwitch GUI 截图" width="705">
</p>

<p align="center">
  <strong>新手小白也能一键安装 Claude Code 的可视化 API 配置管理器</strong>
</p>

## 这是什么？

Claude Code 是 Anthropic 推出的 AI 编程助手 CLI 工具，功能强大，但安装配置对新手来说门槛较高：

- 需要手动安装 Node.js、npm
- 需要配置环境变量
- 国内网络访问 GitHub 下载困难
- 如果想切换不同的 API 提供商（如智谱、DeepSeek、Kimi 等），需要手动修改配置文件

**CCSwitch GUI** 解决了以上所有问题。这是一个 Windows 桌面应用，提供图形化界面，让新手也能：

1. **一键安装 Claude Code** — 自动安装 Node.js、Git、Claude Code，自动配置环境变量，全程可视化进度
2. **一键切换 API 配置** — 在不同 API 提供商之间快速切换，无需手动编辑任何配置文件
3. **可视化管理** — 添加、编辑、删除 API 配置，所有操作都有界面引导

## 支持的环境

| 项目 | 要求 |
|------|------|
| 操作系统 | Windows 10 / Windows 11 |
| 运行依赖 | **无**（单 exe 文件，双击即用） |
| 网络 | 安装 Claude Code 时需要网络连接 |
| 架构 | x86_64 (64 位) |

> 无需安装 .NET、Java、Python 等任何运行时环境。

## 功能一览

### 一键安装 Claude Code

全自动五步安装流程：

1. 检查并安装 Node.js（通过 winget）
2. 检查并安装 Git for Windows（从华为云镜像下载，国内快速）
3. 自动配置 npm 全局路径和 Git 路径到系统环境变量
4. 安装 Claude Code（`npm install -g @anthropic-ai/claude-code`）
5. 验证安装结果

全程可视化进度条和日志输出，每一步都清晰可见。

### API 配置管理

- **配置列表** — 左侧展示所有 API 配置，当前激活的配置高亮显示
- **一键切换** — 点击按钮即可切换 API 提供商，自动写入 `~/.claude/settings.json`
- **编辑保存** — 右侧表单可修改 Base URL、Token、模型名称等所有字段
- **新增配置** — 图形化表单添加新的 API 提供商
- **删除配置** — 确认对话框防误删
- **下载预设** — 从 GitHub 下载社区预设配置，一键导入
- **重置设置** — 一键清空 Claude Code 的环境变量配置

### Token 安全

- API Token 默认以密码形式显示（`***`）
- 点击眼睛图标切换显示/隐藏

### 配置文件兼容

完全兼容 [huangdijia/ccswitch](https://github.com/huangdijia/ccswitch) 的配置格式，配置文件位于 `~/.ccswitch/ccs.json`。

## 快速开始

### 方式一：直接下载

从 [Releases](https://github.com/congxb/ccswitch-gui/releases) 下载最新的 `ccswitch-gui.exe`，双击运行即可。

### 方式二：从源码构建

```bash
# 1. 克隆仓库
git clone https://github.com/congxb/ccswitch-gui.git
cd ccswitch-gui

# 2. 安装 MinGW-w64 GCC（构建时需要）
winget install mingw

# 3. 构建依赖
go mod tidy

# 4. 编译（产出单个 exe）
CGO_ENABLED=1 go build -ldflags "-s -w -H windowsgui" -o ccswitch-gui.exe .
```

## 项目结构

```
ccswitch-gui/
├── main.go          # 入口，创建窗口
├── config.go        # ccs.json 配置文件读写
├── settings.go      # ~/.claude/settings.json 读写
├── ui.go            # 主界面和安装流程
├── github.go        # GitHub 预设配置下载
├── winpath.go       # Windows 注册表 PATH 管理
├── go.mod
└── go.sum
```

## 内置预置配置

开箱即用，内置多家国内 AI 提供商的 API 配置：

| 配置名称 | 提供商 | 说明 |
|---------|--------|------|
| glm | 智谱 AI | GLM 系列 |
| deepseek | DeepSeek | DeepSeek 系列 |
| kimi-k2 | Moonshot | Kimi K2 |
| kimi-kfc | Kimi | Kimi for Coding |
| minimaxi-m2 | MiniMax | MiniMax M2 |
| modelscope | ModelScope | Qwen 系列 |
| xiaomi-mimo | 小米 | MiMo 系列 |
| default | Anthropic | Anthropic 官方 |

## 工作原理

CCSwitch 通过修改 `~/.claude/settings.json` 中的 `env` 字段来切换 API 提供商：

```json
{
    "env": {
        "ANTHROPIC_BASE_URL": "https://open.bigmodel.cn/api/anthropic",
        "ANTHROPIC_AUTH_TOKEN": "your-token",
        "ANTHROPIC_MODEL": "glm-5-turbo"
    },
    "model": "glm-5-turbo"
}
```

切换配置后，重启 Claude Code 即可使用新的 API 提供商。

## 致谢

本项目基于以下开源项目构建：

- [Fyne v2](https://fyne.io/) — Go 跨平台 GUI 框架
- [huangdijia/ccswitch](https://github.com/huangdijia/ccswitch) — Claude Code API 切换工具（配置格式参考）
- [Anthropic Claude Code](https://docs.anthropic.com/en/docs/claude-code) — AI 编程助手
- [Git for Windows](https://gitforwindows.org/) — Windows 下的 Git 环境
- [Node.js](https://nodejs.org/) — JavaScript 运行时

## License

MIT
