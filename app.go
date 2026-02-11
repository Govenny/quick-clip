package main

import (
	"context"
	"fmt"

	"os"
	"path/filepath"
	"quick-clip/internal"
	"time"

	"github.com/tailscale/win"
	"golang.design/x/hotkey" // 注意：这个库通常要求在主线程初始化
)

// App struct
type App struct {
	ctx           context.Context
	content       []any
	keys          string
	action        *internal.Action
	isVisible     bool
	lastHwnd      win.HWND
	configManager *internal.ConfigManager
	config        *internal.Config
	dataPath      string
}

// NewApp creates a new App application struct
func NewApp(action *internal.Action, configManager *internal.ConfigManager, config *internal.Config) *App {
	configDir, _ := os.UserConfigDir()
	appConfigDir := filepath.Join(configDir, "quick-clip", "data") // 替换为你的应用名
	os.MkdirAll(appConfigDir, 0755)
	dataPath := filepath.Join(appConfigDir, "resource.json")

	return &App{
		keys:          "11112222111122221111222211112222",
		action:        action,
		isVisible:     false,
		configManager: configManager,
		config:        config,
		dataPath:      dataPath,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	resource := internal.ReadContent(a.dataPath, a.keys)
	if resource == nil {
		// 如果读取失败（可能是解密失败），初始化为空数组，防止程序崩溃
		a.content = make([]any, 0)
	} else {
		a.content = resource
	}

	// 根据config初始化注册相关配置
	a.RegisterGlobalHotkey(a.config.Shortcuts.WakeUp[0], a.config.Shortcuts.WakeUp[1])
	a.action.SetTransparency(uint8(a.config.Appearance.Opacity))

	// 注册窗口句柄
	go func() {
		for i := 0; i < 10; i++ {
			time.Sleep(1000 * time.Millisecond)

			hwnd := a.action.FindRealWailsWindow()
			if hwnd != 0 {
				rootHwnd := internal.GetRootHWND(hwnd)
				a.action.SetSelfHwnd(rootHwnd)

				// 初始化完成后，先用原生方式藏起来
				// 这样 Wails 认为窗口是“显示”的，WebView2 会继续工作
				// 但用户看不见
				a.action.Hide()
				break
			}
		}
	}()

}

// shutdown is called when the app is about to close
func (a *App) shutdown(ctx context.Context) {
	internal.SaveContent(a.dataPath, a.keys, a.content)
}

func (a *App) GetContent() []any {
	return a.content
}

func (a *App) SaveContent(data []any) {
	a.content = data
	fmt.Println(a.content)
	internal.SaveContent(a.dataPath, a.keys, a.content)
}

func (a *App) RegisterGlobalHotkey(key1 string, key2 string) {
	go func() {
		// 从映射中获取 Modifier 和 Key
		modifier, ok1 := internal.HotKeyMap[key1].(hotkey.Modifier)
		key, ok2 := internal.HotKeyMap[key2].(hotkey.Key)
		if !ok1 || !ok2 {
			return
		}

		hk := hotkey.New([]hotkey.Modifier{modifier}, key)
		err := hk.Register()
		if err != nil {
			return
		}

		// 监听热键事件
		for range hk.Keydown() {
			a.ToggleWindow()
		}
	}()
}

// 你的热键触发逻辑
func (a *App) ToggleWindow() {
	if a.isVisible {
		a.isVisible = false
		a.action.Hide()
	} else {
		a.lastHwnd = a.action.RecordActiveWindow()
		a.isVisible = true
		a.action.ShowNoActivate()
	}
}

func (a *App) HideWindow() {
	a.isVisible = false
	a.action.Hide()
}

func (a *App) PasteAndHide() {
	// 1. 隐藏窗口
	a.isVisible = false
	a.action.Hide()

	// 2. 恢复焦点
	a.action.RestoreFocus(a.lastHwnd)

	// 3. 等待并粘贴
	time.Sleep(time.Duration(a.config.Shortcuts.PasteWaitTime) * time.Millisecond)
	go a.action.SendPaste()
}

// 进入设置模式：变大
func (a *App) EnterSettingsMode() {
	a.action.SetSizeNative(600, 450)
}

// 退出设置模式：变回紧凑小窗口
func (a *App) ExitSettingsMode() {
	a.action.SetSizeNative(320, 480)
}

// GetConfig 供前端获取当前配置
func (a *App) GetConfig() *internal.Config {
	return a.config
}

// UpdateConfig 供前端更新配置, 存在写入操作
func (a *App) UpdateConfig(newCfg *internal.Config) string {
	a.config = newCfg
	err := a.configManager.Save(newCfg)
	if err != nil {
		return err.Error()
	}
	// 这里可以触发一些逻辑更新，比如修改了热键后重新注册热键
	return "success"
}

func (a *App) SetOpacity(opacity uint8) {
	a.config.Appearance.Opacity = opacity
	a.action.SetTransparency(opacity)
}

func (a *App) GetDataPath() string {
	return a.dataPath
}

func (a *App) GetKeys() string {
	return a.keys
}
