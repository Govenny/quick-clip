package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config 定义你的配置项
type Config struct {
	Hotkey     string `json:"hotkey"`
	Theme      string `json:"theme"`
	AutoStart  bool   `json:"autoStart"`
	PasteDelay int    `json:"pasteDelay"`
}

type ConfigManager struct {
	Path string
}

func NewConfigManager() *ConfigManager {
	// 获取系统用户配置目录
	configDir, _ := os.UserConfigDir()
	appConfigDir := filepath.Join(configDir, "quick-clip") // 替换为你的应用名

	// 确保文件夹存在
	os.MkdirAll(appConfigDir, 0755)

	return &ConfigManager{
		Path: filepath.Join(appConfigDir, "config", "config.json"),
	}
}

// Load 读取配置
func (m *ConfigManager) Load() (*Config, error) {
	data, err := os.ReadFile(m.Path)
	if err != nil {
		// 如果文件不存在，返回默认配置
		return &Config{
			Hotkey:     "Alt+Space",
			Theme:      "dark",
			AutoStart:  false,
			PasteDelay: 150,
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
