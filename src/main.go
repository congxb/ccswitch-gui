package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
)

func main() {
	// CLI 模式：ccswitch-gui.exe -cli
	cliMode := flag.Bool("cli", false, "命令行模式（无GUI，用于远程调试）")
	flag.Parse()

	if *cliMode {
		runCLI()
		return
	}

	// 启动日志（定位闪退点）
	startLog := func(step string) {
		msg := fmt.Sprintf("[%s] %s\n", time.Now().Format("15:04:05"), step)
		f, _ := os.OpenFile(os.TempDir()+"\\ccswitch-startup.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if f != nil {
			f.WriteString(msg)
			f.Close()
		}
	}

	startLog("1-enter-main")
	defer func() {
		if r := recover(); r != nil {
			logPath := fmt.Sprintf("%s\\ccswitch-panic.log", os.TempDir())
			msg := fmt.Sprintf("PANIC: %v\n\nStack:\n%s", r, debug.Stack())
			os.WriteFile(logPath, []byte(msg), 0644)
			startLog(fmt.Sprintf("PANIC: %v", r))
		}
	}()

	startLog("2-before-env")
	// 强制使用软件渲染器，避免 OpenGL GPU 兼容性问题导致闪崩
	os.Setenv("FYNE_FORCE_SOFTWARE_RENDERER", "1")

	startLog("3-before-newapp")
	a := app.NewWithID("com.ccswitch.gui")
	startLog("4-after-newapp")
	a.Settings().SetTheme(theme.DarkTheme())

	startLog("5-before-newwindow")
	w := a.NewWindow("CCSwitch - Claude Code API 配置管理器")
	startLog("6-after-newwindow")
	w.Resize(fyne.NewSize(700, 500))

	startLog("7-before-createUI")
	configPath := defaultConfigPath()
	ui := createUI(w, configPath)
	startLog(fmt.Sprintf("8-after-createUI ui=%v", ui == nil))

	startLog("9-before-ShowAndRun")
	w.ShowAndRun()
	startLog("10-after-ShowAndRun")
}

// ======================== CLI 模式 ========================

func runCLI() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ltime)

	args := flag.Args()
	if len(args) == 0 {
		printCLIHelp()
		return
	}

	switch args[0] {
	case "detect":
		cliDetect()
	case "install":
		cliInstall()
	case "env":
		cliShowEnv()
	case "uninstall":
		cliUninstall(args[1:])
	default:
		fmt.Printf("未知命令: %s\n\n", args[0])
		printCLIHelp()
	}
}

func printCLIHelp() {
	fmt.Println("CCSwitch CLI - 远程调试工具")
	fmt.Println()
	fmt.Println("用法: ccswitch-gui.exe -cli <命令>")
	fmt.Println()
	fmt.Println("命令:")
	fmt.Println("  detect     检测 Claude Code 安装状态")
	fmt.Println("  install    执行一键安装流程")
	fmt.Println("  env        显示当前环境变量（PATH 等）")
	fmt.Println("  uninstall  卸载 Claude Code")
}

func cliDetect() {
	log.Println("=== 检测 Claude Code ===")

	// 检测 node
	log.Print("node: ")
	nodeVer, err := runShell("cmd.exe", "/c", "node", "--version")
	if err != nil {
		log.Println("未安装")
	} else {
		log.Println(nodeVer)
	}

	// 检测 npm
	log.Print("npm: ")
	npmVer, err := runShell("cmd.exe", "/c", "npm", "--version")
	if err != nil {
		log.Println("未安装")
	} else {
		log.Println(npmVer)
	}

	// 检测 git
	log.Print("git: ")
	gitVer, err := runShell("cmd.exe", "/c", "git", "--version")
	if err != nil {
		log.Println("未安装")
	} else {
		log.Println(gitVer)
	}

	// 检测 winget
	log.Print("winget: ")
	wingetVer, err := runShell("cmd.exe", "/c", "winget", "--version")
	if err != nil {
		log.Println("未安装")
	} else {
		log.Println(wingetVer)
	}

	// 检测 claude
	log.Print("claude: ")
	claudeVer, err := runShell("cmd.exe", "/c", "claude", "--version")
	if err != nil {
		log.Println("未安装")
	} else {
		log.Println(claudeVer)
	}

	// 显示 npm prefix
	log.Print("npm prefix: ")
	prefix, err := runShell("cmd.exe", "/c", "npm", "config", "get", "prefix")
	if err != nil {
		log.Println("获取失败:", err)
	} else {
		log.Println(prefix)
	}

	// 显示用户 PATH
	log.Println("=== 用户 PATH ===")
	userPath := getUserPath()
	if userPath != "" {
		for _, p := range splitPath(userPath) {
			log.Println(" ", p)
		}
	} else {
		log.Println("  (空)")
	}
}

func cliInstall() {
	log.Println("=== 一键安装 Claude Code ===")

	// 步骤 1: npm
	log.Println("[1/4] 检查 npm...")
	_, npmErr := runShell("cmd.exe", "/c", "npm", "--version")
	if npmErr != nil {
		log.Println("  npm 未安装，正在下载 Node.js LTS zip...")
		nodejsZip := filepath.Join(os.TempDir(), "nodejs-lts.zip")
		_, dlErr := runShell("cmd.exe", "/c", "curl", "-fSL", "-o", nodejsZip,
			"https://npmmirror.com/mirrors/node/v22.15.0/node-v22.15.0-win-x64.zip")
		if dlErr != nil {
			log.Println("  下载失败:", dlErr)
			return
		}
		log.Println("  下载完成，正在解压安装...")

		nodejsDir := `C:\Program Files\nodejs`

		// 写 PowerShell 脚本到临时文件，避免引号嵌套问题
		psScript := filepath.Join(os.TempDir(), "install-nodejs.ps1")
		psContent := fmt.Sprintf(
			"Expand-Archive -Path '%s' -DestinationPath 'C:/' -Force\n"+
				"Remove-Item '%s' -Recurse -Force -ErrorAction SilentlyContinue\n"+
				"Move-Item 'C:/node-v22.15.0-win-x64' '%s' -Force\n"+
				"Write-Output DONE\n",
			nodejsZip, nodejsDir, nodejsDir)
		os.WriteFile(psScript, []byte(psContent), 0644)

		out, instErr := runShell("powershell", "-ExecutionPolicy", "Bypass", "-File", psScript)
		os.Remove(psScript)
		os.Remove(nodejsZip)
		log.Println("  PowerShell:", trimStr(out))
		if instErr != nil || !strings.Contains(out, "DONE") {
			log.Println("  安装失败:", instErr)
			return
		}

		if err := addToUserPath(nodejsDir); err != nil {
			log.Println("  警告: addToUserPath 失败:", err)
		}
		os.Setenv("PATH", os.Getenv("PATH")+";"+nodejsDir)

		_, npmCheckErr := runShell("cmd.exe", "/c", "npm", "--version")
		if npmCheckErr != nil {
			log.Println("  错误: npm 安装后仍不可用！")
			return
		}
		log.Println("  Node.js + npm 安装完成")
	} else {
		log.Println("  npm 已就绪")
	}

	// 步骤 2: Git
	log.Println("[2/4] 检查 Git...")
	if _, err := os.Stat(`C:\Program Files\Git\bin\bash.exe`); err == nil {
		log.Println("  Git for Windows 已安装")
	} else {
		log.Println("  跳过 Git 安装（CLI 模式下手动处理）")
	}

	// 步骤 3: 环境变量
	log.Println("[3/4] 配置环境变量...")
	prefix, prefixErr := runShell("cmd.exe", "/c", "npm", "config", "get", "prefix")
	if prefixErr == nil && trimStr(prefix) != "" {
		prefix = trimStr(prefix)
		if err := addToUserPath(prefix); err != nil {
			log.Println("  警告:", err)
		}
		os.Setenv("PATH", os.Getenv("PATH")+";"+prefix)
		log.Println("  npm prefix:", prefix)
	}

	// 步骤 4: 安装 Claude Code
	log.Println("[4/4] 安装 Claude Code...")
	output := execShell("cmd.exe", "/c", "npm", "install", "-g", "@anthropic-ai/claude-code")
	log.Println("  npm 输出:", output)

	// 验证
	claudeVer := execShell("cmd.exe", "/c", "claude", "--version")
	if claudeVer != "" {
		log.Println("=== 安装成功! Claude Code 版本:", claudeVer, "===")
	} else {
		log.Println("=== 安装完成但验证失败，请重启终端 ===")
	}
}


func cliShowEnv() {
	log.Println("=== 环境变量 ===")
	log.Println("PATH:")
	for _, p := range splitPath(os.Getenv("PATH")) {
		log.Println(" ", p)
	}
	log.Println()
	log.Println("用户 PATH (注册表):")
	userPath := getUserPath()
	if userPath != "" {
		for _, p := range splitPath(userPath) {
			log.Println(" ", p)
		}
	}
}

func cliUninstall(args []string) {
	force := len(args) > 0 && args[0] == "--force"
	if !force {
		log.Println("使用 -cli uninstall --force 确认卸载")
		return
	}
	log.Println("=== 卸载 Claude Code ===")
	output := execShell("cmd.exe", "/c", "npm", "uninstall", "-g", "@anthropic-ai/claude-code")
	log.Println(output)
	log.Println("卸载完成")
}

// ======================== CLI 辅助函数 ========================

func runShell(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return trimStr(string(out)), err
}

func execShell(name string, args ...string) string {
	out, _ := runShell(name, args...)
	return out
}

func trimStr(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

func splitPath(path string) []string {
	parts := strings.Split(path, ";")
	var result []string
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// 需要导入
var (
	_ = fyne.Theme(nil)
	_ = theme.ColorNamePrimary
)
