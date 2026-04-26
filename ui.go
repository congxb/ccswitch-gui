package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// profileShortDesc profile 短描述映射
var profileShortDesc = map[string]string{
	"default":     "Anthropic 官方",
	"glm":         "智谱 GLM",
	"deepseek":    "DeepSeek",
	"kimi-kfc":    "Kimi",
	"kimi-k2":     "Kimi K2",
	"modelscope":  "ModelScope",
	"minimaxi-m2": "MiniMax M2",
	"xiaomi-mimo": "小米 Mimo",
	"anyrouter":   "AnyRouter",
}

// appUI 主界面状态
type appUI struct {
	window     fyne.Window
	configPath string
	cfg        *Config

	activeProfile string
	profileList   *widget.List
	profileNames  []string

	detailScroll *container.Scroll
	detailBox    *fyne.Container

	nameLabel    *canvas.Text
	descEntry    *widget.Entry
	urlEntry     *widget.Entry
	tokenEntry   *widget.Entry
	modelEntry   *widget.Entry
	fastEntry    *widget.Entry
	sonnetEntry  *widget.Entry
	opusEntry    *widget.Entry
	haikuEntry   *widget.Entry
	timeoutEntry *widget.Entry
	showTokenBtn *widget.Button
	tokenVisible bool

	saveBtn   *widget.Button
	switchBtn *widget.Button
	deleteBtn *widget.Button

	selectedIdx int

	claudePath    string
	claudeVersion string
	installCmd    *exec.Cmd
}

// _globalUI 全局 UI 实例
var _globalUI *appUI

// createUI 创建主界面
func createUI(window fyne.Window, configPath string) *appUI {
	ui := &appUI{
		window:     window,
		configPath: configPath,
	}
	_globalUI = ui

	// 检测 Claude Code
	ui.detectClaudeCode()

	// 加载配置
	cfg, err := loadConfig(configPath)
	if err != nil {
		// 加载失败，尝试下载预设
		presetCfg, presetErr := fetchPresets()
		if presetErr == nil {
			cfg = presetCfg
			_ = saveConfig(configPath, cfg)
		} else {
			// 使用默认空配置
			cfg = &Config{
				Profiles:     make(map[string]map[string]string),
				Descriptions: make(map[string]string),
			}
			_ = saveConfig(configPath, cfg)
		}
	}
	ui.cfg = cfg

	// 构建界面
	ui.buildUI()

	return ui
}

// detectClaudeCode 检测 Claude Code 安装路径和版本
func (ui *appUI) detectClaudeCode() {
	ui.claudePath = ""
	ui.claudeVersion = ""

	// 查找 claude 可执行文件
	cmd := exec.Command("cmd.exe", "/c", "where", "claude")
	out, err := cmd.Output()
	if err != nil {
		return
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return
	}
	ui.claudePath = strings.TrimSpace(lines[0])

	// 获取版本
	verCmd := exec.Command("cmd.exe", "/c", "claude", "--version")
	verOut, verErr := verCmd.Output()
	if verErr != nil {
		ui.claudeVersion = "未知版本"
		return
	}
	ui.claudeVersion = strings.TrimSpace(string(verOut))
}

// ======================== 安装 Claude Code ========================

// installClaudeCode 五步骤安装流程
func (ui *appUI) installClaudeCode() {
	installWin := ui.window

	stepLabel := widget.NewLabel("")
	stepLabel.TextStyle.Bold = true
	statusLabel := widget.NewLabel("")
	timeLabel := widget.NewLabel("")
	timeLabel.Alignment = fyne.TextAlignTrailing

	progressBar := widget.NewProgressBar()
	progressBar.Min = 0
	progressBar.Max = 5

	pctLabel := widget.NewLabel("0%")
	pctLabel.Alignment = fyne.TextAlignCenter

	logEntry := widget.NewEntry()
	logEntry.MultiLine = true
	logEntry.Wrapping = fyne.TextWrapBreak
	logEntry.SetPlaceHolder("安装日志...")
	logEntry.Disable()

	bottomBox := container.NewHBox()

	logScroll := container.NewVScroll(logEntry)
	logScroll.SetMinSize(fyne.NewSize(600, 200))

	content := container.NewVBox(
		container.NewHBox(
			container.NewVBox(stepLabel, statusLabel),
			layout.NewSpacer(),
			container.NewVBox(timeLabel),
		),
		progressBar,
		pctLabel,
		logScroll,
		bottomBox,
	)

	dialogWin := dialog.NewCustom("安装 Claude Code", "取消", content, installWin)

	// 取消标记
	cancelled := false

	// 关闭时取消安装（包括点击取消按钮和关闭窗口）
	dialogWin.SetOnClosed(func() {
		cancelled = true
		if ui.installCmd != nil && ui.installCmd.Process != nil {
			ui.installCmd.Process.Kill()
		}
	})

	dialogWin.Show()

	// 辅助函数
	setProgress := func(step, total float64, status string) {
		progressBar.SetValue(step)
		pct := int(step / total * 100)
		pctLabel.SetText(fmt.Sprintf("%d%%", pct))
		statusLabel.SetText(status)
	}

	appendLog := func(msg string) {
		if logEntry.Text == "" {
			logEntry.SetText(msg)
		} else {
			logEntry.SetText(logEntry.Text + "\n" + msg)
		}
		// 滚动到底部
		logScroll.Offset.Y = 99999
		logScroll.Refresh()
	}

	setBottomBtn := func(btn fyne.CanvasObject) {
		bottomBox.RemoveAll()
		bottomBox.Add(btn)
		bottomBox.Refresh()
	}

	// runCmdAndWait 启动命令并等待完成，实时读取输出
	runCmdAndWait := func(name string, args ...string) (string, error) {
		ui.installCmd = exec.Command(name, args...)
		ui.installCmd.Env = os.Environ()

		stdoutPipe, err := ui.installCmd.StdoutPipe()
		if err != nil {
			return "", fmt.Errorf("创建 stdout pipe 失败: %w", err)
		}
		stderrPipe, err := ui.installCmd.StderrPipe()
		if err != nil {
			return "", fmt.Errorf("创建 stderr pipe 失败: %w", err)
		}

		if err := ui.installCmd.Start(); err != nil {
			return "", fmt.Errorf("启动命令失败: %w", err)
		}

		// 实时读取 stdout
		var stdoutBuf strings.Builder
		go func() {
			reader := bufio.NewReader(stdoutPipe)
			for {
				line, err := reader.ReadString('\n')
				if line != "" {
					line = strings.TrimRight(line, "\r\n")
					appendLog(line)
					stdoutBuf.WriteString(line + "\n")
				}
				if err != nil {
					break
				}
			}
		}()

		// 实时读取 stderr
		go func() {
			reader := bufio.NewReader(stderrPipe)
			for {
				line, err := reader.ReadString('\n')
				if line != "" {
					line = strings.TrimRight(line, "\r\n")
					appendLog("[stderr] " + line)
				}
				if err != nil {
					break
				}
			}
		}()

		err = ui.installCmd.Wait()
		return stdoutBuf.String(), err
	}

	// getNpmPrefix 获取 npm 全局安装前缀
	getNpmPrefix := func() (string, error) {
		out, err := runCmdAndWait("cmd.exe", "/c", "npm", "config", "get", "prefix")
		if err != nil {
			return "", fmt.Errorf("获取 npm prefix 失败: %w", err)
		}
		return strings.TrimSpace(out), nil
	}

	// ensureInPath 确保目录在 PATH 中
	ensureInPath := func(dir string) error {
		pathEnv := os.Getenv("PATH")
		for _, p := range strings.Split(pathEnv, ";") {
			if strings.EqualFold(strings.TrimSpace(p), dir) {
				appendLog("  目录已在 PATH 中: " + dir)
				return nil
			}
		}
		// 写入注册表（永久生效）
		appendLog("  添加到用户 PATH: " + dir)
		if err := addToUserPath(dir); err != nil {
			return fmt.Errorf("添加到 PATH 失败: %w", err)
		}
		// 更新当前进程的 PATH
		newPath := pathEnv + ";" + dir
		os.Setenv("PATH", newPath)
		appendLog("  已更新进程 PATH")
		return nil
	}

	// 启动安装流程（在 goroutine 中执行）
	go func() {
		defer func() {
			ui.installCmd = nil
		}()

		startTime := time.Now()
		updateTime := func() {
			elapsed := time.Since(startTime).Round(time.Second)
			timeLabel.SetText("耗时: " + elapsed.String())
		}
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		go func() {
			for range ticker.C {
				updateTime()
			}
		}()

		// ---- 步骤 1/5：检查 npm ----
		if cancelled {
			return
		}
			stepLabel.SetText("步骤 1/5：检查 Node.js / npm")
			setProgress(0, 5, "正在检查 npm...")
			setBottomBtn(container.NewHBox())
		appendLog("检查 npm 是否可用...")

		npmCheck := exec.Command("cmd.exe", "/c", "npm", "--version")
		if err := npmCheck.Run(); err != nil {
			appendLog("npm 未安装，正在通过 winget 安装 Node.js LTS...")
				setProgress(0.1, 5, "正在安装 Node.js LTS...")

			_, wingetErr := runCmdAndWait("cmd.exe", "/c", "winget", "install", "OpenJS.NodeJS.LTS", "--accept-package-agreements", "--accept-source-agreements")
			if wingetErr != nil {
				appendLog("winget 安装 Node.js 失败: " + wingetErr.Error())
					dialog.ShowError(fmt.Errorf("安装 Node.js 失败，请手动安装 Node.js 后重试"), installWin)
					dialogWin.Hide()
				return
			}

			// 安装后确保 nodejs 在 PATH 中
			nodejsDir := `C:\Program Files\nodejs`
			if err := ensureInPath(nodejsDir); err != nil {
				appendLog("警告: 添加 nodejs 到 PATH 失败: " + err.Error())
			}

			appendLog("Node.js 安装完成")
		} else {
			appendLog("npm 已就绪")
		}
			setProgress(1, 5, "npm 就绪")

		// ---- 步骤 2/5：检查/安装 Git for Windows ----
			if cancelled {
				return
			}
				stepLabel.SetText("步骤 2/5：检查/安装 Git for Windows")
				setProgress(1.5, 5, "正在检查 Git for Windows...")
			appendLog("检查 Git for Windows 是否已安装...")

			gitBash := `C:\Program Files\Git\bin\bash.exe`
			_, gitExists := os.Stat(gitBash)
			if gitExists == nil {
				appendLog("Git for Windows 已安装")
			} else {
				appendLog("Git for Windows 未安装，正在从华为云镜像下载...")
					setProgress(1.7, 5, "正在下载 Git for Windows...")

				tempPath := filepath.Join(os.TempDir(), "Git-2.47.1-64-bit.exe")
				gitURL := "https://mirrors.huaweicloud.com/git-for-windows/v2.47.1.windows.1/Git-2.47.1-64-bit.exe"
				_, downloadErr := runCmdAndWait("cmd.exe", "/c", "curl", "-fSL", "-o", tempPath, gitURL)
				if downloadErr != nil {
					appendLog("下载 Git for Windows 失败: " + downloadErr.Error())
						dialog.ShowError(fmt.Errorf("下载 Git for Windows 失败: %w", downloadErr), installWin)
						dialogWin.Hide()
					return
				}

				appendLog("下载完成，正在静默安装 Git for Windows...")
					setProgress(1.9, 5, "正在安装 Git for Windows...")

				installGitCmd := exec.Command(tempPath, "/VERYSILENT", "/NORESTART", "/SP-", "/DIR=C:\\Program Files\\Git")
				installGitCmd.Env = os.Environ()
				installErr := installGitCmd.Run()
				if installErr != nil {
					appendLog("安装 Git for Windows 失败: " + installErr.Error())
					os.Remove(tempPath)
						dialog.ShowError(fmt.Errorf("安装 Git for Windows 失败: %w", installErr), installWin)
						dialogWin.Hide()
					return
				}

				// 清理临时安装包
				os.Remove(tempPath)

				// 验证安装
				_, gitVerifyErr := os.Stat(gitBash)
				if gitVerifyErr != nil {
					appendLog("Git for Windows 安装后验证失败: " + gitVerifyErr.Error())
						dialog.ShowError(fmt.Errorf("Git for Windows 安装后验证失败"), installWin)
						dialogWin.Hide()
					return
				}
				appendLog("Git for Windows 安装完成")
			}
				setProgress(2, 5, "Git for Windows 就绪")


		// ---- 步骤 3/5：配置环境变量 ----
		if cancelled {
			return
		}
			stepLabel.SetText("步骤 3/5：配置环境变量")
			setProgress(2.5, 5, "获取 npm 全局路径...")
		appendLog("获取 npm 全局安装前缀...")

		npmPrefix, err := getNpmPrefix()
		if err != nil {
			appendLog("获取 npm prefix 失败: " + err.Error())
				dialog.ShowError(fmt.Errorf("获取 npm 配置失败: %w", err), installWin)
				dialogWin.Hide()
			return
		}
		appendLog("npm prefix: " + npmPrefix)

		// 确保 npm 全局 bin 目录在 PATH 中
		if err := ensureInPath(npmPrefix); err != nil {
			appendLog("警告: 添加 npm prefix 到 PATH 失败: " + err.Error())
		}

		// 确保 Git 目录在 PATH 中
		if err := ensureInPath(`C:\Program Files\Git\cmd`); err != nil {
			appendLog("警告: 添加 Git cmd 到 PATH 失败: " + err.Error())
		}
		if err := ensureInPath(`C:\Program Files\Git\bin`); err != nil {
			appendLog("警告: 添加 Git bin 到 PATH 失败: " + err.Error())
		}

			setProgress(3, 5, "环境变量已配置")
		appendLog("环境变量配置完成")

		// ---- 步骤 4/5：安装 Claude Code ----
		if cancelled {
			return
		}
			stepLabel.SetText("步骤 4/5：安装 Claude Code")
			setProgress(3.5, 5, "正在安装 @anthropic-ai/claude-code...")
		appendLog("开始安装 Claude Code...")

		_, installErr := runCmdAndWait("cmd.exe", "/c", "npm", "install", "-g", "@anthropic-ai/claude-code")
		if installErr != nil {
			appendLog("安装 Claude Code 失败: " + installErr.Error())
				dialog.ShowError(fmt.Errorf("安装 Claude Code 失败: %w", installErr), installWin)
				dialogWin.Hide()
			return
		}
		appendLog("Claude Code 安装完成")
			setProgress(4, 5, "Claude Code 已安装")

		// ---- 步骤 5/5：验证安装 ----
		if cancelled {
			return
		}
			stepLabel.SetText("步骤 5/5：验证安装")
			setProgress(4.5, 5, "正在验证...")
		appendLog("验证 Claude Code 安装...")

		ui.detectClaudeCode()
		if ui.claudePath == "" {
			// 直接检查 npmPrefix 下的 claude.cmd
			appendLog("where claude 未找到，尝试直接检查 npmPrefix...")
			possiblePath := npmPrefix + "\\claude.cmd"
			if _, statErr := os.Stat(possiblePath); statErr == nil {
				ui.claudePath = possiblePath
				appendLog("找到 Claude Code: " + possiblePath)

				// 尝试获取版本
				verCmd := exec.Command("cmd.exe", "/c", possiblePath, "--version")
				verOut, verErr := verCmd.Output()
				if verErr == nil {
					ui.claudeVersion = strings.TrimSpace(string(verOut))
					appendLog("版本: " + ui.claudeVersion)
				}
			}
		}

		if ui.claudePath != "" {
			appendLog("验证成功！Claude Code 路径: " + ui.claudePath)
				setProgress(5, 5, "安装完成！")
				setBottomBtn(widget.NewButton("完成", func() {
					dialogWin.Hide()
					// 刷新主界面
					ui.buildUI()
				}))
		} else {
			appendLog("验证失败，未找到 claude 命令")
				dialog.ShowError(fmt.Errorf("安装完成但未找到 claude 命令，请重启终端后重试"), installWin)
				dialogWin.Hide()
				ui.buildUI()
		}
	}()
}

// ======================== 构建主界面 ========================

// buildUI 构建完整界面
func (ui *appUI) buildUI() {
	// 标题栏
	title := canvas.NewText("CCSwitch", theme.ForegroundColor())
	title.TextSize = 20
	title.TextStyle.Bold = true
	subtitle := canvas.NewText("天机图35岁程序员开源", theme.DisabledColor())
	subtitle.TextSize = 12
	titleBar := container.NewHBox(title, layout.NewSpacer(), subtitle)

	// 左侧 profile 列表
	ui.refreshProfileList()
	leftPanel := container.NewBorder(
		widget.NewLabelWithStyle("配置列表", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		ui.profileList,
	)

	// 右侧详情面板
	ui.buildDetailPanel()

	// 左右分栏
	split := container.NewHSplit(leftPanel, ui.detailScroll)
	split.SetOffset(0.25)

	// Claude Code 信息栏
	claudeInfoBar := ui.buildClaudeInfoBar()

	// 底部工具栏
	addBtn := widget.NewButtonWithIcon("新增配置", theme.ContentAddIcon(), func() {
		ui.showAddDialog()
	})
	presetBtn := widget.NewButtonWithIcon("下载预设", theme.DownloadIcon(), func() {
		ui.showPresetDialog()
	})
	resetBtn := widget.NewButtonWithIcon("重置设置", theme.ContentUndoIcon(), func() {
		ui.showResetConfirm()
	})
	toolbar := container.NewHBox(addBtn, presetBtn, layout.NewSpacer(), resetBtn)

	// 主内容
	mainContent := container.NewBorder(titleBar, claudeInfoBar, nil, nil, split)

	// 组合底部
	bottom := container.NewVBox(toolbar)
	fullContent := container.NewBorder(nil, bottom, nil, nil, mainContent)
	fullContent.Resize(fyne.NewSize(700, 650))

	ui.window.SetContent(fullContent)
}

// buildClaudeInfoBar 构建 Claude Code 信息栏
func (ui *appUI) buildClaudeInfoBar() *fyne.Container {
	if ui.claudePath != "" {
		// 有 Claude Code
		pathLabel := canvas.NewText("Claude Code: "+ui.claudePath, theme.ForegroundColor())
		pathLabel.TextSize = 11
		verLabel := canvas.NewText(" ("+ui.claudeVersion+")", theme.DisabledColor())
		verLabel.TextSize = 11

		copyBtn := widget.NewButtonWithIcon("复制路径", theme.ContentCopyIcon(), func() {
			ui.window.Clipboard().SetContent(ui.claudePath)
			dialog.ShowInformation("已复制", "Claude Code 路径已复制到剪贴板", ui.window)
		})

		return container.NewHBox(pathLabel, verLabel, layout.NewSpacer(), copyBtn)
	}

	// 没有 Claude Code
	notFoundLabel := canvas.NewText("未检测到 Claude Code", theme.ErrorColor())
	notFoundLabel.TextSize = 12
	notFoundLabel.TextStyle.Bold = true

	installBtn := widget.NewButtonWithIcon("一键安装 Claude Code", theme.DownloadIcon(), func() {
		ui.installClaudeCode()
	})
	installBtn.Importance = widget.HighImportance

	return container.NewHBox(notFoundLabel, layout.NewSpacer(), installBtn)
}

// buildDetailPanel 构建右侧详情面板
func (ui *appUI) buildDetailPanel() {
	// profile 名称
	ui.nameLabel = canvas.NewText("", theme.ForegroundColor())
	ui.nameLabel.TextSize = 18
	ui.nameLabel.TextStyle.Bold = true

	// 描述
	ui.descEntry = widget.NewEntry()
	ui.descEntry.SetPlaceHolder("配置描述（可选）")

	// API Base URL
	ui.urlEntry = widget.NewEntry()
	ui.urlEntry.SetPlaceHolder("https://api.example.com")

	// API Token（多行）
	ui.tokenEntry = widget.NewEntry()
	ui.tokenEntry.MultiLine = true
	ui.tokenEntry.SetPlaceHolder("sk-xxx...")
	ui.tokenEntry.Password = true

	// 显示/隐藏 token 按钮
	ui.tokenVisible = false
	ui.showTokenBtn = widget.NewButtonWithIcon("", theme.VisibilityIcon(), func() {
		ui.tokenVisible = !ui.tokenVisible
		ui.tokenEntry.Password = !ui.tokenVisible
		if ui.tokenVisible {
			ui.showTokenBtn.SetIcon(theme.VisibilityOffIcon())
		} else {
			ui.showTokenBtn.SetIcon(theme.VisibilityIcon())
		}
	})

	// 主模型
	ui.modelEntry = widget.NewEntry()
	

	// 子模型
	ui.fastEntry = widget.NewEntry()
	
	ui.sonnetEntry = widget.NewEntry()
	
	ui.opusEntry = widget.NewEntry()
	
	ui.haikuEntry = widget.NewEntry()
	

	// 超时
	ui.timeoutEntry = widget.NewEntry()
	

	// 操作按钮
	ui.saveBtn = widget.NewButtonWithIcon("保存", theme.DocumentSaveIcon(), func() {
		ui.saveCurrentProfile()
	})
	ui.switchBtn = widget.NewButtonWithIcon("切换到此配置", theme.MediaReplayIcon(), func() {
		ui.switchToCurrentProfile()
	})
	ui.switchBtn.Importance = widget.HighImportance
	ui.deleteBtn = widget.NewButtonWithIcon("删除", theme.DeleteIcon(), func() {
		ui.deleteCurrentProfile()
	})
	ui.deleteBtn.Importance = widget.DangerImportance

	// 表单布局
	form := container.NewVBox(
		ui.nameLabel,
		widget.NewSeparator(),
		widget.NewForm(
			widget.NewFormItem("描述", ui.descEntry),
			widget.NewFormItem("API Base URL", ui.urlEntry),
			widget.NewFormItem("API Token", container.NewBorder(nil, nil, nil, ui.showTokenBtn, ui.tokenEntry)),
			widget.NewFormItem("主模型", ui.modelEntry),
			widget.NewFormItem("快速模型", ui.fastEntry),
			widget.NewFormItem("Sonnet 模型", ui.sonnetEntry),
			widget.NewFormItem("Opus 模型", ui.opusEntry),
			widget.NewFormItem("Haiku 模型", ui.haikuEntry),
			widget.NewFormItem("超时", ui.timeoutEntry),
		),
		widget.NewSeparator(),
		container.NewHBox(ui.saveBtn, ui.switchBtn, layout.NewSpacer(), ui.deleteBtn),
	)

	ui.detailBox = container.NewPadded(form)
	ui.detailScroll = container.NewVScroll(ui.detailBox)
	ui.detailScroll.SetMinSize(fyne.NewSize(300, 550))

	// 默认显示空
	ui.clearDetail()
}

// refreshProfileList 刷新 profile 列表
func (ui *appUI) refreshProfileList() {
	ui.profileNames = make([]string, 0, len(ui.cfg.Profiles))
	for name := range ui.cfg.Profiles {
		ui.profileNames = append(ui.profileNames, name)
	}
	ui.activeProfile = detectActiveProfile(ui.cfg)

	ui.profileList = widget.NewList(
		func() int {
			return len(ui.profileNames)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			name := ui.profileNames[i]
			shortDesc := ui.getShortDesc(name)
			if shortDesc != "" {
				label.SetText(name + " - " + shortDesc)
			} else {
				label.SetText(name)
			}

			// 高亮激活的 profile
			if name == ui.activeProfile {
				label.TextStyle.Bold = true
			} else {
				label.TextStyle.Bold = false
			}
		},
	)

	ui.profileList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(ui.profileNames) {
			return
		}
		ui.selectedIdx = id
		ui.showProfileDetail(ui.profileNames[id])
	}

	ui.profileList.OnUnselected = func(id widget.ListItemID) {
		ui.selectedIdx = -1
		ui.clearDetail()
	}
}

// showProfileDetail 显示指定 profile 的详情
func (ui *appUI) showProfileDetail(name string) {
	env, exists := ui.cfg.Profiles[name]
	if !exists {
		ui.clearDetail()
		return
	}

	ui.nameLabel.Text = name
	ui.nameLabel.Refresh()

	// 描述
	desc := ""
	if d, ok := ui.cfg.Descriptions[name]; ok {
		desc = d
	}
	ui.descEntry.SetText(desc)

	// 各字段
	ui.urlEntry.SetText(env["ANTHROPIC_BASE_URL"])
	ui.modelEntry.SetText(env["ANTHROPIC_MODEL"])
	ui.fastEntry.SetText(env["ANTHROPIC_SMALL_FAST_MODEL"])
	ui.sonnetEntry.SetText(env["ANTHROPIC_DEFAULT_SONNET_MODEL"])
	ui.opusEntry.SetText(env["ANTHROPIC_DEFAULT_OPUS_MODEL"])
	ui.haikuEntry.SetText(env["ANTHROPIC_DEFAULT_HAIKU_MODEL"])
	ui.timeoutEntry.SetText(env["ANTHROPIC_TIMEOUT"])

	// Token 处理（兼容 API_KEY 和 AUTH_TOKEN）
	token := env["ANTHROPIC_AUTH_TOKEN"]
	if token == "" {
		token = env["ANTHROPIC_API_KEY"]
	}
	ui.tokenEntry.SetText(token)
	// 默认隐藏 token
	ui.tokenVisible = false
	ui.tokenEntry.Password = true
	ui.showTokenBtn.SetIcon(theme.VisibilityIcon())

	// 如果是激活的 profile，高亮切换按钮
	if name == ui.activeProfile {
		ui.switchBtn.SetText("当前激活")
		ui.switchBtn.Disable()
	} else {
		ui.switchBtn.SetText("切换到此配置")
		ui.switchBtn.Enable()
	}

	ui.saveBtn.Enable()
	ui.deleteBtn.Enable()
}

// clearDetail 清空详情面板
func (ui *appUI) clearDetail() {
	ui.nameLabel.Text = "请选择一个配置"
	ui.nameLabel.Refresh()
	ui.descEntry.SetText("")
	ui.urlEntry.SetText("")
	ui.tokenEntry.SetText("")
	ui.modelEntry.SetText("")
	ui.fastEntry.SetText("")
	ui.sonnetEntry.SetText("")
	ui.opusEntry.SetText("")
	ui.haikuEntry.SetText("")
	ui.timeoutEntry.SetText("")
	ui.saveBtn.Disable()
	ui.switchBtn.Disable()
	ui.switchBtn.SetText("切换到此配置")
	ui.deleteBtn.Disable()
}

// saveCurrentProfile 保存当前编辑的 profile
func (ui *appUI) saveCurrentProfile() {
	if ui.selectedIdx < 0 || ui.selectedIdx >= len(ui.profileNames) {
		dialog.ShowError(fmt.Errorf("请先选择一个配置"), ui.window)
		return
	}

	name := ui.profileNames[ui.selectedIdx]
	env, exists := ui.cfg.Profiles[name]
	if !exists {
		env = make(map[string]string)
		ui.cfg.Profiles[name] = env
	}

	env["ANTHROPIC_BASE_URL"] = strings.TrimSpace(ui.urlEntry.Text)
	env["ANTHROPIC_MODEL"] = strings.TrimSpace(ui.modelEntry.Text)
	env["ANTHROPIC_AUTH_TOKEN"] = strings.TrimSpace(ui.tokenEntry.Text)
	delete(env, "ANTHROPIC_API_KEY")
	env["ANTHROPIC_SMALL_FAST_MODEL"] = strings.TrimSpace(ui.fastEntry.Text)
	env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = strings.TrimSpace(ui.sonnetEntry.Text)
	env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = strings.TrimSpace(ui.opusEntry.Text)
	env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = strings.TrimSpace(ui.haikuEntry.Text)
	env["ANTHROPIC_TIMEOUT"] = strings.TrimSpace(ui.timeoutEntry.Text)

	// 保存描述
	desc := strings.TrimSpace(ui.descEntry.Text)
	if desc != "" {
		ui.cfg.Descriptions[name] = desc
	} else {
		delete(ui.cfg.Descriptions, name)
	}

	if err := saveConfig(ui.configPath, ui.cfg); err != nil {
		dialog.ShowError(fmt.Errorf("保存失败: %w", err), ui.window)
		return
	}

	dialog.ShowInformation("保存成功", "配置 '"+name+"' 已保存", ui.window)
}

// switchToCurrentProfile 切换到当前选中的 profile
func (ui *appUI) switchToCurrentProfile() {
	if ui.selectedIdx < 0 || ui.selectedIdx >= len(ui.profileNames) {
		dialog.ShowError(fmt.Errorf("请先选择一个配置"), ui.window)
		return
	}

	name := ui.profileNames[ui.selectedIdx]

	dialog.ShowConfirm("确认切换",
		"确定要切换到配置 '"+name+"' 吗？\n这将修改 Claude 的 settings.json 文件。",
		func(confirmed bool) {
			if !confirmed {
				return
			}
			// 先保存
			ui.saveCurrentProfile()
			// 切换
			if err := switchProfile(ui.cfg, name); err != nil {
				dialog.ShowError(fmt.Errorf("切换失败: %w", err), ui.window)
				return
			}
			ui.cfg.Default = name
			_ = saveConfig(ui.configPath, ui.cfg)

			dialog.ShowInformation("切换成功", "已切换到配置 '"+name+"'", ui.window)

			// 刷新界面
			ui.activeProfile = detectActiveProfile(ui.cfg)
			ui.refreshProfileList()
			ui.buildUI()
		}, ui.window)
}

// deleteCurrentProfile 删除当前选中的 profile
func (ui *appUI) deleteCurrentProfile() {
	if ui.selectedIdx < 0 || ui.selectedIdx >= len(ui.profileNames) {
		dialog.ShowError(fmt.Errorf("请先选择一个配置"), ui.window)
		return
	}

	name := ui.profileNames[ui.selectedIdx]

	dialog.ShowConfirm("确认删除",
		"确定要删除配置 '"+name+"' 吗？\n此操作不可撤销。",
		func(confirmed bool) {
			if !confirmed {
				return
			}
			delete(ui.cfg.Profiles, name)
			delete(ui.cfg.Descriptions, name)
			if ui.cfg.Default == name {
				ui.cfg.Default = ""
			}

			if err := saveConfig(ui.configPath, ui.cfg); err != nil {
				dialog.ShowError(fmt.Errorf("删除失败: %w", err), ui.window)
				return
			}

			dialog.ShowInformation("删除成功", "配置 '"+name+"' 已删除", ui.window)

			// 刷新界面
			ui.activeProfile = detectActiveProfile(ui.cfg)
			ui.refreshProfileList()
			ui.buildUI()
		}, ui.window)
}

// showAddDialog 显示新增配置对话框
func (ui *appUI) showAddDialog() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("配置名称（英文）")

	descEntry := widget.NewEntry()
	descEntry.SetPlaceHolder("配置描述（可选）")

	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("API Base URL")

	tokenEntry := widget.NewEntry()
	tokenEntry.MultiLine = true
	tokenEntry.SetPlaceHolder("API Token")
	tokenEntry.Password = true

	modelEntry := widget.NewEntry()
	modelEntry.SetPlaceHolder("主模型")

	form := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("配置名称", nameEntry),
			widget.NewFormItem("描述", descEntry),
			widget.NewFormItem("API Base URL", urlEntry),
			widget.NewFormItem("API Token", tokenEntry),
			widget.NewFormItem("主模型", modelEntry),
		),
	)

	dialogWin := dialog.NewCustomConfirm("新增配置", "创建", "取消", form, func(confirmed bool) {
		if !confirmed {
			return
		}

		name := strings.TrimSpace(nameEntry.Text)
		if name == "" {
			dialog.ShowError(fmt.Errorf("配置名称不能为空"), ui.window)
			return
		}

		if _, exists := ui.cfg.Profiles[name]; exists {
			dialog.ShowError(fmt.Errorf("配置 '%s' 已存在", name), ui.window)
			return
		}

		env := make(map[string]string)
		env["ANTHROPIC_BASE_URL"] = strings.TrimSpace(urlEntry.Text)
		env["ANTHROPIC_AUTH_TOKEN"] = strings.TrimSpace(tokenEntry.Text)
		env["ANTHROPIC_MODEL"] = strings.TrimSpace(modelEntry.Text)

		ui.cfg.Profiles[name] = env
		desc := strings.TrimSpace(descEntry.Text)
		if desc != "" {
			ui.cfg.Descriptions[name] = desc
		}

		if err := saveConfig(ui.configPath, ui.cfg); err != nil {
			dialog.ShowError(fmt.Errorf("保存失败: %w", err), ui.window)
			return
		}

		dialog.ShowInformation("创建成功", "配置 '"+name+"' 已创建", ui.window)

		// 刷新界面
		ui.refreshProfileList()
		ui.buildUI()
	}, ui.window)
	dialogWin.Resize(fyne.NewSize(600, 450))
	dialogWin.Show()
}

// showPresetDialog 显示下载预设对话框
func (ui *appUI) showPresetDialog() {
	statusLabel := widget.NewLabel("正在下载预设配置...")
	statusLabel.Alignment = fyne.TextAlignCenter

	dialogWin := dialog.NewCustom("下载预设配置", "关闭", statusLabel, ui.window)
	dialogWin.Resize(fyne.NewSize(400, 150))
	dialogWin.Show()

	go func() {
		presetCfg, err := fetchPresets()
			dialogWin.Hide()
			if err != nil {
				dialog.ShowError(fmt.Errorf("下载预设失败: %w", err), ui.window)
				return
			}

			if len(presetCfg.Profiles) == 0 {
				dialog.ShowInformation("无预设", "远程预设配置为空", ui.window)
				return
			}

			// 构建可选列表
			presetNames := make([]string, 0, len(presetCfg.Profiles))
			for name := range presetCfg.Profiles {
				presetNames = append(presetNames, name)
			}

			// 多选列表
			check := widget.NewCheckGroup(presetNames, nil)

			selectAllBtn := widget.NewButton("全选", func() {
				check.SetSelected(presetNames)
			})
			deselectAllBtn := widget.NewButton("取消全选", func() {
				check.SetSelected(nil)
			})

			form := container.NewBorder(
				widget.NewLabel("选择要导入的预设配置："),
				container.NewHBox(selectAllBtn, deselectAllBtn),
				nil, nil,
				check,
			)

			dialog.ShowCustomConfirm("导入预设配置", "导入", "取消", form, func(confirmed bool) {
				if !confirmed || len(check.Selected) == 0 {
					return
				}

				imported := importPreset(ui.cfg, presetCfg, check.Selected)
				if len(imported) == 0 {
					dialog.ShowInformation("无需导入", "所选配置已全部存在", ui.window)
					return
				}

				if err := saveConfig(ui.configPath, ui.cfg); err != nil {
					dialog.ShowError(fmt.Errorf("保存失败: %w", err), ui.window)
					return
				}

				dialog.ShowInformation("导入成功",
					fmt.Sprintf("已导入 %d 个配置:\n%s", len(imported), strings.Join(imported, ", ")),
					ui.window,
				)

				// 刷新界面
				ui.refreshProfileList()
				ui.buildUI()
			}, ui.window)
	}()
}

// showResetConfirm 显示重置确认对话框
func (ui *appUI) showResetConfirm() {
	dialog.ShowConfirm("重置设置",
		"确定要重置 Claude 的 settings.json 吗？\n这将清空所有环境变量和模型配置。",
		func(confirmed bool) {
			if !confirmed {
				return
			}
			if err := resetSettings(ui.cfg); err != nil {
				dialog.ShowError(fmt.Errorf("重置失败: %w", err), ui.window)
				return
			}
			dialog.ShowInformation("重置成功", "Claude settings.json 已重置", ui.window)

			// 刷新界面
			ui.activeProfile = detectActiveProfile(ui.cfg)
			ui.refreshProfileList()
			ui.buildUI()
		}, ui.window)
}

// getShortDesc 获取 profile 短描述
func (ui *appUI) getShortDesc(name string) string {
	if desc, ok := profileShortDesc[name]; ok {
		return desc
	}
	// 回退到 descriptions
	if desc, ok := ui.cfg.Descriptions[name]; ok {
		return desc
	}
	return ""
}

// isWindows 判断是否为 Windows 系统
func isWindows() bool {
	return runtime.GOOS == "windows"
}
