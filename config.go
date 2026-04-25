package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config 兼容 huangdijia/ccswitch 的 ccs.json 格式
type Config struct {
	SettingsPath string                       `json:"settingsPath"`
	Default      string                       `json:"default"`
	Profiles     map[string]map[string]string `json:"profiles"`
	Descriptions map[string]string            `json:"descriptions,omitempty"`
}

var defaultModelKeys = []string{
	"ANTHROPIC_DEFAULT_HAIKU_MODEL",
	"ANTHROPIC_DEFAULT_OPUS_MODEL",
	"ANTHROPIC_DEFAULT_SONNET_MODEL",
	"ANTHROPIC_SMALL_FAST_MODEL",
}

// defaultConfigPath 返回 ccs.json 的默认路径
func defaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ccswitch", "ccs.json")
}

// defaultSettingsPath 返回 Claude settings.json 的默认路径
func defaultSettingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "settings.json")
}

// loadConfig 从文件加载配置
func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	cfg := &Config{
		Profiles:     make(map[string]map[string]string),
		Descriptions: make(map[string]string),
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]map[string]string)
	}
	if cfg.Descriptions == nil {
		cfg.Descriptions = make(map[string]string)
	}

	return cfg, nil
}

// saveConfig 保存配置到文件
func saveConfig(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// getProfileWithDefaults 获取 profile 配置，填充缺失的 model 子键
func getProfileWithDefaults(cfg *Config, name string) map[string]string {
	env, exists := cfg.Profiles[name]
	if !exists {
		return make(map[string]string)
	}

	result := make(map[string]string)
	for k, v := range env {
		result[k] = v
	}

	// 用 ANTHROPIC_MODEL 填充缺失的子模型键
	if model, ok := result["ANTHROPIC_MODEL"]; ok {
		for _, key := range defaultModelKeys {
			if _, exists := result[key]; !exists {
				result[key] = model
			}
		}
	}

	return result
}

// isPlaceholderToken 判断 token 是否是占位符（非真实 token）
func isPlaceholderToken(token string) bool {
	if token == "" {
		return true
	}
	// 常见的占位符（纯前缀，没有真实内容）
	placeholders := []string{"sk-", "sk-kimi-", "ms-", "sk-ant-"}
	for _, p := range placeholders {
		if token == p {
			return true
		}
	}
	// 真实 token 通常较长（>= 15 字符）
	return len(token) < 15
}

// maskToken 对敏感值进行脱敏处理
func maskToken(val string) string {
	if val == "" {
		return ""
	}
	if len(val) <= 8 {
		return strings.Repeat("*", len(val))
	}
	return val[:4] + strings.Repeat("*", len(val)-8) + val[len(val)-4:]
}

// detectActiveProfile 检测当前激活的 profile（通过比较 settings.json 的 env 和各 profile）
func detectActiveProfile(cfg *Config) string {
	settingsPath := defaultSettingsPath()
	if cfg.SettingsPath != "" {
		settingsPath = expandHome(cfg.SettingsPath)
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return ""
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return ""
	}

	envObj, ok := settings["env"].(map[string]interface{})
	if !ok {
		return ""
	}

	// 获取 settings 中的 base URL 用于匹配
	settingsURL, _ := envObj["ANTHROPIC_BASE_URL"].(string)
	if settingsURL == "" {
		return ""
	}

	// 遍历所有 profile，匹配 base URL
	for name, profile := range cfg.Profiles {
		if profileURL, ok := profile["ANTHROPIC_BASE_URL"]; ok && profileURL == settingsURL {
			return name
		}
	}

	return ""
}

func expandHome(path string) string {
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}
