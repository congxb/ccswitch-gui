package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// switchProfile 将指定 profile 的配置写入 Claude settings.json
func switchProfile(cfg *Config, profileName string) error {
	env := getProfileWithDefaults(cfg, profileName)
	if len(env) == 0 {
		return fmt.Errorf("profile '%s' 不存在", profileName)
	}

	settingsPath := defaultSettingsPath()
	if cfg.SettingsPath != "" {
		settingsPath = expandHome(cfg.SettingsPath)
	}

	// 读取现有 settings，保留未知字段
	raw := make(map[string]interface{})
	data, err := os.ReadFile(settingsPath)
	if err == nil {
		json.Unmarshal(data, &raw)
	}

	// 替换 env 和 model，统一使用 AUTH_TOKEN，移除 API_KEY 避免冲突
	envInterface := make(map[string]interface{})
	for k, v := range env {
		envInterface[k] = v
	}
	delete(envInterface, "ANTHROPIC_API_KEY")
	raw["env"] = envInterface

	if model, ok := env["ANTHROPIC_MODEL"]; ok && model != "" {
		raw["model"] = model
	} else {
		delete(raw, "model")
	}

	// 写回
	out, err := json.MarshalIndent(raw, "", "    ")
	if err != nil {
		return fmt.Errorf("序列化 settings 失败: %w", err)
	}

	
	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return fmt.Errorf("创建 settings 目录失败: %w", err)
	}
	if err := os.WriteFile(settingsPath, out, 0644); err != nil {
		return fmt.Errorf("写入 settings 失败: %w", err)
	}

	return nil
}

// resetSettings 清空 settings.json 的 env 和 model
func resetSettings(cfg *Config) error {
	settingsPath := defaultSettingsPath()
	if cfg.SettingsPath != "" {
		settingsPath = expandHome(cfg.SettingsPath)
	}

	raw := make(map[string]interface{})
	data, err := os.ReadFile(settingsPath)
	if err == nil {
		json.Unmarshal(data, &raw)
	}

	delete(raw, "env")
	delete(raw, "model")

	out, err := json.MarshalIndent(raw, "", "    ")
	if err != nil {
		return fmt.Errorf("序列化 settings 失败: %w", err)
	}

	
	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return fmt.Errorf("创建 settings 目录失败: %w", err)
	}
	if err := os.WriteFile(settingsPath, out, 0644); err != nil {
		return fmt.Errorf("写入 settings 失败: %w", err)
	}

	return nil
}
