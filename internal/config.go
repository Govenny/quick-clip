package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type GeneralConfig struct {
	LaunchAtLogin bool `json:"launchAtLogin"`
}

type ShortcutsConfig struct {
	WakeUp        [2]string `json:"wakeUp"`
	PasteWaitTime int       `json:"pasteWaitTime"`
}

type AppearanceConfig struct {
	Opacity uint8 `json:"opacity"`
}

type Config struct {
	General    GeneralConfig    `json:"general"`
	Shortcuts  ShortcutsConfig  `json:"shortcuts"`
	Appearance AppearanceConfig `json:"appearance"`
}

// Config 定义你的配置项
type ConfigManager struct {
	Path string
}

func NewConfigManager() *ConfigManager {
	// 获取系统用户配置目录
	configDir, _ := os.UserConfigDir()
	appConfigDir := filepath.Join(configDir, "quick-clip", "config") // 替换为你的应用名

	// 确保文件夹存在
	os.MkdirAll(appConfigDir, 0755)

	return &ConfigManager{
		Path: filepath.Join(appConfigDir, "config.json"),
	}
}

// Load 读取配置
func (m *ConfigManager) Load() (*Config, error) {
	data, err := os.ReadFile(m.Path)
	if err != nil {
		// 如果文件不存在，返回默认配置
		return &Config{
			GeneralConfig{
				LaunchAtLogin: false,
			},
			ShortcutsConfig{
				WakeUp:        [2]string{"Alt", "Space"},
				PasteWaitTime: 100,
			},
			AppearanceConfig{
				Opacity: 250,
			},
		}, nil
	}

	var config Config
	err = json.Unmarshal(data, &config)
	return &config, err
}

// Save 保存配置
func (m *ConfigManager) Save(config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.Path, data, 0644)
}
