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
	Capsule1      [2]string `json:"capsule1"`
	Capsule2      [2]string `json:"capsule2"`
	Capsule3      [2]string `json:"capsule3"`
}

type AppearanceConfig struct {
	Opacity       uint8 `json:"opacity"`
	FontSizeLevel int   `json:"fontSizeLevel"` // 1~5, 默认 2 (标准 13px)
}

type WindowConfig struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type Config struct {
	General    GeneralConfig    `json:"general"`
	Shortcuts  ShortcutsConfig  `json:"shortcuts"`
	Appearance AppearanceConfig `json:"appearance"`
	Window     WindowConfig     `json:"window"`
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
				Capsule1:      [2]string{"Alt", "1"},
				Capsule2:      [2]string{"Alt", "2"},
				Capsule3:      [2]string{"Alt", "3"},
			},
			AppearanceConfig{
				Opacity:       250,
				FontSizeLevel: 2,
			},
			WindowConfig{
				Width:  420,
				Height: 580,
			},
		}, nil
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if config.Appearance.FontSizeLevel < 1 || config.Appearance.FontSizeLevel > 5 {
		config.Appearance.FontSizeLevel = 2
	}
	if config.Shortcuts.Capsule1[1] == "" {
		config.Shortcuts.Capsule1 = [2]string{"Alt", "1"}
	}
	if config.Shortcuts.Capsule2[1] == "" {
		config.Shortcuts.Capsule2 = [2]string{"Alt", "2"}
	}
	if config.Shortcuts.Capsule3[1] == "" {
		config.Shortcuts.Capsule3 = [2]string{"Alt", "3"}
	}
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
