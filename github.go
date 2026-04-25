package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const presetURL = "https://raw.githubusercontent.com/huangdijia/ccswitch/main/config/preset.json"

// fetchPresets 从 GitHub 下载预设配置
func fetchPresets() (*Config, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get(presetURL)
	if err != nil {
		return nil, fmt.Errorf("下载预设失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("下载预设失败: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取预设数据失败: %w", err)
	}

	cfg := &Config{
		Profiles:     make(map[string]map[string]string),
		Descriptions: make(map[string]string),
	}

	if err := json.Unmarshal(body, cfg); err != nil {
		return nil, fmt.Errorf("解析预设数据失败: %w", err)
	}

	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]map[string]string)
	}
	if cfg.Descriptions == nil {
		cfg.Descriptions = make(map[string]string)
	}

	return cfg, nil
}

// importPreset 从预设导入指定 profile 到本地配置
func importPreset(localCfg *Config, presetCfg *Config, profileNames []string) []string {
	imported := make([]string, 0)
	for _, name := range profileNames {
		if env, ok := presetCfg.Profiles[name]; ok {
			if _, exists := localCfg.Profiles[name]; !exists {
				localCfg.Profiles[name] = env
				if desc, ok := presetCfg.Descriptions[name]; ok {
					localCfg.Descriptions[name] = desc
				}
				imported = append(imported, name)
			}
		}
	}
	return imported
}

// presetProviderName 从 URL 推断提供商名称
func presetProviderName(baseURL string) string {
	switch {
	case strings.Contains(baseURL, "anthropic.com"):
		return "Anthropic"
	case strings.Contains(baseURL, "bigmodel.cn"):
		return "智谱 (Zhipu)"
	case strings.Contains(baseURL, "deepseek.com"):
		return "DeepSeek"
	case strings.Contains(baseURL, "kimi.com"):
		return "Kimi (KFC)"
	case strings.Contains(baseURL, "moonshot.cn"):
		return "Kimi K2"
	case strings.Contains(baseURL, "modelscope.cn"):
		return "ModelScope"
	case strings.Contains(baseURL, "minimaxi.com"):
		return "MiniMax"
	case strings.Contains(baseURL, "xiaomimimo.com"):
		return "小米 Mimo"
	case strings.Contains(baseURL, "anthropic.com"):
		return "Anthropic"
	default:
		return "自定义"
	}
}
