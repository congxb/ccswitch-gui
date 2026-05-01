package main

import (
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// addToUserPath 将指定目录添加到用户环境变量 PATH 中（永久生效）
// 使用 Windows registry API 直接操作，避免 cmd.exe 的 % 变量展开问题
func addToUserPath(dir string) error {
	if dir == "" {
		return fmt.Errorf("empty path")
	}

	// 打开用户环境变量注册表
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return fmt.Errorf("open registry: %w", err)
	}
	defer key.Close()

	// 读取当前用户 PATH
	currentPath, _, err := key.GetStringValue("Path")
	if err != nil {
		// PATH 不存在，直接创建
		if err := key.SetStringValue("Path", dir); err != nil {
			return fmt.Errorf("create PATH: %w", err)
		}
		return nil
	}

	// 检查是否已包含
	for _, p := range strings.Split(currentPath, ";") {
		if strings.EqualFold(strings.TrimSpace(p), dir) {
			return nil // 已存在
		}
	}

	// 拼接并写回
	newPath := currentPath + ";" + dir
	if err := key.SetStringValue("Path", newPath); err != nil {
		return fmt.Errorf("set PATH: %w", err)
	}

	// 通知系统环境变量已更改（让 explorer.exe 和其他程序能立即感知）
	broadcastEnvChange()

	return nil
}

// getUserPath 获取用户环境变量 PATH
func getUserPath() string {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return ""
	}
	defer key.Close()

	path, _, err := key.GetStringValue("Path")
	if err != nil {
		return ""
	}
	return path
}

// broadcastEnvChange 广播 WM_SETTINGCHANGE 通知系统环境变量已更改
func broadcastEnvChange() {
	user32 := windows.NewLazySystemDLL("user32.dll")
	sendMessageTimeout := user32.NewProc("SendMessageTimeoutW")

	// HWND_BROADCAST = 0xFFFF
	// WM_SETTINGCHANGE = 0x001A
	// SMTO_ABORTIFHUNG = 0x0002
	env, _ := windows.UTF16PtrFromString("Environment")
	sendMessageTimeout.Call(
		0xFFFF,
		0x001A,
		0,
		uintptr(unsafe.Pointer(env)),
		0x0002,
		0x2710, // 10 seconds timeout
		0,
	)
}
