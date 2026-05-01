package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
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

var _globalUI *appUI

func createUI(window fyne.Window, configPath string) *appUI {
	ui := &appUI{
		window:     window,
		configPath: configPath,
	}
	_globalUI = ui

	ui.detectClaudeCode()

	cfg, err := loadConfig(configPath)
	if err != nil {
		presetCfg, presetErr := fetchPresets()
		if presetErr == nil {
			cfg = presetCfg
			_ = saveConfig(configPath, cfg)
		} else {
			cfg = &Config{
				Profiles:     make(map[string]map[string]string),
				Descriptions: make(map[string]string),
			}
			_ = saveConfig(configPath, cfg)
		}
	}
	ui.cfg = cfg

	ui.buildUI()

	return ui
}

func (ui *appUI) detectClaudeCode() {
	ui.claudePath = ""
	ui.claudeVersion = ""

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "cmd.exe", "/c", "where", "claude")
	out, err := cmd.Output()
	if err != nil {
		return
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return
	}
	ui.claudePath = strings.TrimSpace(lines[0])

	verCtx, verCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer verCancel()
	verCmd := exec.CommandContext(verCtx, "cmd.exe", "/c", "claude", "--version")
	verOut, verErr := verCmd.Output()
	if verErr != nil {
		ui.claudeVersion = "未知版本"
		return
	}
	ui.claudeVersion = strings.TrimSpace(string(verOut))
}

// ======================== 安装 Claude Code ========================

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

	dialogWin := dialog.NewCustom("安装 Claude Code", "", content, installWin)

	cancelled := false

	dialogWin.SetOnClosed(func() {
		cancelled = true
		if ui.installCmd != nil && ui.installCmd.Process != nil {
			ui.installCmd.Process.Kill()
		}
	})

	dialogWin.Show()

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
		logScroll.Offset.Y = 99999
		logScroll.Refresh()
	}

	setBottomBtn := func(btn fyne.CanvasObject) {
		bottomBox.RemoveAll()
		bottomBox.Add(btn)
		bottomBox.Refresh()
	}

	installLogPath := filepath.Join(os.TempDir(), "ccswitch-install.log")
	os.WriteFile(installLogPath, []byte("=== CCSwitch 安装日志 "+time.Now().Format("2006-01-02 15:04:05")+" ===\n"), 0644)
	appendLog = func(msg string) {
		timestamped := time.Now().Format("15:04:05") + " " + msg
		if logEntry.Text == "" {
			logEntry.SetText(timestamped)
		} else {
			logEntry.SetText(logEntry.Text + "\n" + timestamped)
		}
		logScroll.Offset.Y = 99999
		logScroll.Refresh()
		// 同时写入日志文件
		f, err := os.OpenFile(installLogPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			f.WriteString(timestamped + "\n")
			f.Close()
		}
	}

	runCmdAndWait := func(name string, args ...string) (string, error) {
		ui.installCmd = exec.Command(name, args...)

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

	getNpmPrefix := func() (string, error) {
		out, err := runCmdAndWait("cmd.exe", "/c", "npm", "config", "get", "prefix")
		if err != nil {
			return "", fmt.Errorf("获取 npm prefix 失败: %w", err)
		}
		return strings.TrimSpace(out), nil
	}

	ensureInPath := func(dir string) error {
		pathEnv := os.Getenv("PATH")
		for _, p := range strings.Split(pathEnv, ";") {
			if strings.EqualFold(strings.TrimSpace(p), dir) {
				appendLog("  目录已在 PATH 中: " + dir)
				return nil
			}
		}
		appendLog("  添加到用户 PATH: " + dir)
		if err := addToUserPath(dir); err != nil {
			return fmt.Errorf("添加到 PATH 失败: %w", err)
		}
		newPath := pathEnv + ";" + dir
		os.Setenv("PATH", newPath)
		appendLog("  已更新进程 PATH")
		return nil
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				msg := fmt.Sprintf("INSTALL PANIC: %v", r)
				stack := string(debug.Stack())
				logFile := filepath.Join(os.TempDir(), "ccswitch-install-panic.log")
				os.WriteFile(logFile, []byte(msg+"\n\n"+stack), 0644)
				// 同时写入安装日志
				f, ferr := os.OpenFile(installLogPath, os.O_APPEND|os.O_WRONLY, 0644)
				if ferr == nil {
					f.WriteString("!!! PANIC !!! " + msg + "\n" + stack + "\n")
					f.Close()
				}
				log.Println(msg)
			}
			ui.installCmd = nil
		}()

		startTime := time.Now()
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		go func() {
			for range ticker.C {
				elapsed := time.Since(startTime).Round(time.Second)
				timeLabel.SetText("耗时: " + elapsed.String())
			}
		}()

		// ---- 步骤 1/5：检查 npm ----
		if cancelled {
			return
		}
		// 取消按钮（灰色，始终在右侧）
		cancelBtn := widget.NewButton("取消", func() {
			cancelled = true
			if ui.installCmd != nil && ui.installCmd.Process != nil {
				ui.installCmd.Process.Kill()
			}
			dialogWin.Hide()
		})
		cancelBtn.Importance = widget.LowImportance

		stepLabel.SetText("步骤 1/5：检查 Node.js / npm")
		setProgress(0, 5, "正在检查 npm...")
		setBottomBtn(container.NewHBox(layout.NewSpacer(), cancelBtn))
		appendLog("检查 npm 是否可用...")

		npmCheck := exec.Command("cmd.exe", "/c", "npm", "--version")
		if err := npmCheck.Run(); err != nil {
			appendLog("npm 未安装，正在下载 Node.js LTS...")
			setProgress(0.1, 5, "正在下载 Node.js LTS...")

			// 下载 Node.js zip（不依赖 winget/msi，避免 Win11 兼容问题）
			nodejsZip := filepath.Join(os.TempDir(), "nodejs-lts.zip")
			nodejsURL := "https://npmmirror.com/mirrors/node/v22.15.0/node-v22.15.0-win-x64.zip"
			_, downloadErr := runCmdAndWait("cmd.exe", "/c", "curl", "-fSL", "-o", nodejsZip, nodejsURL)
			if downloadErr != nil {
				appendLog("下载 Node.js 失败: " + downloadErr.Error())
				dialog.ShowError(fmt.Errorf("下载 Node.js 失败: %w", downloadErr), installWin)
				dialogWin.Hide()
				return
			}

			appendLog("下载完成，正在解压安装 Node.js...")
			setProgress(0.3, 5, "正在安装 Node.js...")

			nodejsDir := `C:\Program Files\nodejs`
				// 写 PowerShell 脚本到临时文件，避免引号嵌套问题
				psScript := filepath.Join(os.TempDir(), "install-nodejs.ps1")
				psContent := "Expand-Archive -Path '" + nodejsZip + "' -DestinationPath 'C:/' -Force\n" +
					"Remove-Item '" + nodejsDir + "' -Recurse -Force -ErrorAction SilentlyContinue\n" +
					"Move-Item 'C:/node-v22.15.0-win-x64' '" + nodejsDir + "' -Force\n"
				os.WriteFile(psScript, []byte(psContent), 0644)
				_, extractErr := runCmdAndWait("powershell", "-ExecutionPolicy", "Bypass", "-File", psScript)
				os.Remove(psScript)
			os.Remove(nodejsZip)
			if extractErr != nil {
				appendLog("安装 Node.js 失败: " + extractErr.Error())
				dialog.ShowError(fmt.Errorf("安装 Node.js 失败: %w", extractErr), installWin)
				dialogWin.Hide()
				return
			}

			if err := ensureInPath(nodejsDir); err != nil {
				appendLog("警告: 添加 nodejs 到 PATH 失败: " + err.Error())
			}
			if err := ensureInPath(nodejsDir); err != nil {
				appendLog("警告: 添加 nodejs 到 PATH 失败: " + err.Error())
			}

			// 验证 npm 可用
			verifyNpm := exec.Command("cmd.exe", "/c", "npm", "--version")
			if verifyErr := verifyNpm.Run(); verifyErr != nil {
				appendLog("警告: npm 安装后仍不可用，可能需要重启")
			} else {
				appendLog("Node.js + npm 安装完成")
			}
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

			// 先关闭 Git 进程，再卸载旧版，避免安装时弹窗提示"无法卸载旧版本"
			runCmdAndWait("cmd.exe", "/c", "taskkill", "/f", "/im", "git.exe")
			runCmdAndWait("cmd.exe", "/c", "taskkill", "/f", "/im", "git-credential-manager.exe")
			runCmdAndWait("cmd.exe", "/c", "taskkill", "/f", "/im", "git-credential-manager-core.exe")
			// 通过注册表查找并静默卸载旧版 Git
			runCmdAndWait("powershell", "-ExecutionPolicy", "Bypass", "-Command",
				"$git = Get-ItemProperty 'HKLM:\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\Git_is1' -ErrorAction SilentlyContinue; "+
					"if ($git) { $u = $git.UninstallString; if ($u) { Start-Process $u -ArgumentList '/VERYSILENT /NORESTART /SUPPRESSMSGBOXES' -Wait -NoNewWindow } }")

			installGitCmd := exec.Command(tempPath, "/VERYSILENT", "/NORESTART", "/SP-", "/SUPPRESSMSGBOXES", "/DIR=C:\\Program Files\\Git")
			installErr := installGitCmd.Run()
			if installErr != nil {
				appendLog("安装 Git for Windows 失败: " + installErr.Error())
				os.Remove(tempPath)
				dialog.ShowError(fmt.Errorf("安装 Git for Windows 失败: %w", installErr), installWin)
				dialogWin.Hide()
				return
			}

			os.Remove(tempPath)

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

		if err := ensureInPath(npmPrefix); err != nil {
			appendLog("警告: 添加 npm prefix 到 PATH 失败: " + err.Error())
		}
		if err := ensureInPath(`C:\Program Files\Git\cmd`); err != nil {
			appendLog("警告: 添加 Git cmd 到 PATH 失败: " + err.Error())
		}
		if err := ensureInPath(`C:\Program Files\Git\bin`); err != nil {
			appendLog("警告: 添加 Git bin 到 PATH 失败: " + err.Error())
		}

		setProgress(3, 5, "环境变量已配置")
		appendLog("环境变量配置完成")

		// 广播环境变量变更，让系统其他程序立即感知
		broadcastEnvChange()
		// 重新从注册表读取最新 PATH 并更新当前进程环境
		if latestPath := getUserPath(); latestPath != "" {
			os.Setenv("PATH", latestPath+";"+os.Getenv("PATH"))
		}

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
			appendLog("where claude 未找到，尝试直接检查 npmPrefix...")
			possiblePath := npmPrefix + "\\claude.cmd"
			if _, statErr := os.Stat(possiblePath); statErr == nil {
				ui.claudePath = possiblePath
				appendLog("找到 Claude Code: " + possiblePath)

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
			doneBtn := widget.NewButton("完成", func() {
				dialogWin.Hide()
				ui.buildUI()
			})
			doneBtn.Importance = widget.HighImportance
			setBottomBtn(container.NewHBox(layout.NewSpacer(), cancelBtn, doneBtn))
		} else {
			appendLog("验证失败，未找到 claude 命令")
			dialog.ShowError(fmt.Errorf("安装完成但未找到 claude 命令，请重启终端后重试"), installWin)
			dialogWin.Hide()
			ui.buildUI()
		}
	}()
}

// ======================== 构建主界面 ========================

func (ui *appUI) buildUI() {
	title := canvas.NewText("CCSwitch", theme.ForegroundColor())
	title.TextSize = 20
	title.TextStyle.Bold = true
	subtitle := canvas.NewText("天机图35岁程序员开源", theme.DisabledColor())
	subtitle.TextSize = 12
	titleBar := container.NewHBox(title, layout.NewSpacer(), subtitle)

	ui.refreshProfileList()
	leftPanel := container.NewBorder(
		widget.NewLabelWithStyle("配置列表", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		ui.profileList,
	)

	ui.buildDetailPanel()

	split := container.NewHSplit(leftPanel, ui.detailScroll)
	split.SetOffset(0.25)

	claudeInfoBar := ui.buildClaudeInfoBar()

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

	mainContent := container.NewBorder(titleBar, claudeInfoBar, nil, nil, split)

	bottom := container.NewVBox(toolbar)
	fullContent := container.NewBorder(nil, bottom, nil, nil, mainContent)
	fullContent.Resize(fyne.NewSize(700, 650))

	ui.window.SetContent(fullContent)
}

func (ui *appUI) buildClaudeInfoBar() *fyne.Container {
	if ui.claudePath != "" {
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

	notFoundLabel := canvas.NewText("未检测到 Claude Code", theme.ErrorColor())
	notFoundLabel.TextSize = 12
	notFoundLabel.TextStyle.Bold = true

	installBtn := widget.NewButtonWithIcon("一键安装 Claude Code", theme.DownloadIcon(), func() {
		ui.installClaudeCode()
	})
	installBtn.Importance = widget.HighImportance

	return container.NewHBox(notFoundLabel, layout.NewSpacer(), installBtn)
}

func (ui *appUI) buildDetailPanel() {
	ui.nameLabel = canvas.NewText("", theme.ForegroundColor())
	ui.nameLabel.TextSize = 18
	ui.nameLabel.TextStyle.Bold = true

	ui.descEntry = widget.NewEntry()
	ui.descEntry.SetPlaceHolder("配置描述（可选）")

	ui.urlEntry = widget.NewEntry()
	ui.urlEntry.SetPlaceHolder("https://api.example.com")

	ui.tokenEntry = widget.NewEntry()
	ui.tokenEntry.MultiLine = true
	ui.tokenEntry.SetPlaceHolder("sk-xxx...")
	ui.tokenEntry.Password = true

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

	ui.modelEntry = widget.NewEntry()
	ui.fastEntry = widget.NewEntry()
	ui.sonnetEntry = widget.NewEntry()
	ui.opusEntry = widget.NewEntry()
	ui.haikuEntry = widget.NewEntry()
	ui.timeoutEntry = widget.NewEntry()

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

	ui.clearDetail()
}

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

func (ui *appUI) showProfileDetail(name string) {
	env, exists := ui.cfg.Profiles[name]
	if !exists {
		ui.clearDetail()
		return
	}

	ui.nameLabel.Text = name
	ui.nameLabel.Refresh()

	desc := ""
	if d, ok := ui.cfg.Descriptions[name]; ok {
		desc = d
	}
	ui.descEntry.SetText(desc)

	ui.urlEntry.SetText(env["ANTHROPIC_BASE_URL"])
	ui.modelEntry.SetText(env["ANTHROPIC_MODEL"])
	ui.fastEntry.SetText(env["ANTHROPIC_SMALL_FAST_MODEL"])
	ui.sonnetEntry.SetText(env["ANTHROPIC_DEFAULT_SONNET_MODEL"])
	ui.opusEntry.SetText(env["ANTHROPIC_DEFAULT_OPUS_MODEL"])
	ui.haikuEntry.SetText(env["ANTHROPIC_DEFAULT_HAIKU_MODEL"])
	ui.timeoutEntry.SetText(env["ANTHROPIC_TIMEOUT"])

		// Token 处理（优先 API_KEY，兼容旧 AUTH_TOKEN）
		token := env["ANTHROPIC_API_KEY"]
		if token == "" {
			token = env["ANTHROPIC_AUTH_TOKEN"]
		}
	ui.tokenEntry.SetText(token)
	ui.tokenVisible = false
	ui.tokenEntry.Password = true
	ui.showTokenBtn.SetIcon(theme.VisibilityIcon())

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
			ui.saveCurrentProfile()
			if err := switchProfile(ui.cfg, name); err != nil {
				dialog.ShowError(fmt.Errorf("切换失败: %w", err), ui.window)
				return
			}
			ui.cfg.Default = name
			_ = saveConfig(ui.configPath, ui.cfg)

			dialog.ShowInformation("切换成功", "已切换到配置 '"+name+"'", ui.window)

			ui.activeProfile = detectActiveProfile(ui.cfg)
			ui.refreshProfileList()
			ui.buildUI()
		}, ui.window)
}

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

			ui.activeProfile = detectActiveProfile(ui.cfg)
			ui.refreshProfileList()
			ui.buildUI()
		}, ui.window)
}

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

		ui.refreshProfileList()
		ui.buildUI()
	}, ui.window)
	dialogWin.Resize(fyne.NewSize(600, 450))
	dialogWin.Show()
}

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

		presetNames := make([]string, 0, len(presetCfg.Profiles))
		for name := range presetCfg.Profiles {
			presetNames = append(presetNames, name)
		}

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

			ui.refreshProfileList()
			ui.buildUI()
		}, ui.window)
	}()
}

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

			ui.activeProfile = detectActiveProfile(ui.cfg)
			ui.refreshProfileList()
			ui.buildUI()
		}, ui.window)
}

func (ui *appUI) getShortDesc(name string) string {
	if desc, ok := profileShortDesc[name]; ok {
		return desc
	}
	if desc, ok := ui.cfg.Descriptions[name]; ok {
		return desc
	}
	return ""
}

func isWindows() bool {
	return runtime.GOOS == "windows"
}
