package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"quick-clip/internal"
	"sync"
	"time"

	"github.com/tailscale/win"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.design/x/hotkey" // 注意：这个库通常要求在主线程初始化
)

// App struct
type App struct {
	ctx               context.Context
	content           []any
	keys              string
	action            *internal.Action
	isVisible         bool
	lastHwnd          win.HWND
	configManager     *internal.ConfigManager
	config            *internal.Config
	dataPath          string
	storageReady      bool
	statsManager      *internal.StatsManager
	contextMu         sync.RWMutex
	currentContextKey string

	hotkeyMu       sync.Mutex
	currentHotkey  *hotkey.Hotkey
	hotkeyStopChan chan struct{}
}

// NewApp creates a new App application struct
func NewApp(action *internal.Action, configManager *internal.ConfigManager, config *internal.Config) *App {
	configDir, _ := os.UserConfigDir()
	appConfigDir := filepath.Join(configDir, "quick-clip", "data") // 替换为你的应用名
	os.MkdirAll(appConfigDir, 0755)
	dataPath := filepath.Join(appConfigDir, "resource.json")
	statsManager := internal.NewStatsManager(appConfigDir)

	return &App{
		keys:          "11112222111122221111222211112222",
		action:        action,
		isVisible:     false,
		configManager: configManager,
		config:        config,
		dataPath:      dataPath,
		statsManager:  statsManager,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	resource, err := internal.ReadContent(a.dataPath, a.keys)
	if err != nil {
		fmt.Printf("内容存储不可用；已禁止自动保存以保护原始文件: %v\n", err)
		a.content = make([]any, 0)
		a.storageReady = false
	} else {
		a.content = resource
		a.storageReady = true
	}

	// 根据config初始化注册相关配置
	a.RegisterGlobalHotkey(a.config.Shortcuts.WakeUp[0], a.config.Shortcuts.WakeUp[1])
	a.action.SetTransparency(uint8(a.config.Appearance.Opacity))

	// 注册窗口句柄：使用自适应 50ms 轮询快速捕获窗口并隐藏，避免 1s 阶梯睡眠
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		timeout := time.After(15 * time.Second)

		for {
			select {
			case <-timeout:
				fmt.Println("Warning: Wails window handle discovery timed out")
				return
			case <-ticker.C:
				hwnd := a.action.FindRealWailsWindow()
				if hwnd != 0 {
					rootHwnd := internal.GetRootHWND(hwnd)
					a.action.SetSelfHwnd(rootHwnd)
					a.action.SetOnResized(a.persistWindowSize)
					a.action.InstallResizeTracker(rootHwnd)

					// 初始化完成后立即隐藏
					a.action.Hide()
					return
				}
			}
		}
	}()
}

// shutdown is called when the app is about to close
func (a *App) shutdown(ctx context.Context) {
	// 1. 安全注销全局热键与后台监听 goroutine
	a.hotkeyMu.Lock()
	a.unregisterHotkeyLocked()
	a.hotkeyMu.Unlock()

	// 2. 刷新未持久化的统计数据
	if a.statsManager != nil {
		_ = a.statsManager.Flush()
	}

	// 3. 持久化密码本内容
	if !a.storageReady {
		return
	}
	if err := internal.SaveContent(a.dataPath, a.keys, a.content); err != nil {
		fmt.Printf("退出时保存内容失败: %v\n", err)
	}
}

func (a *App) GetContent() []any {
	return a.content
}

func (a *App) SaveContent(data []any) {
	if !a.storageReady {
		fmt.Println("内容存储未成功加载；拒绝保存以保护磁盘数据")
		return
	}

	a.content = data
	if err := internal.SaveContent(a.dataPath, a.keys, a.content); err != nil {
		fmt.Printf("保存内容失败: %v\n", err)
	}
}

func (a *App) unregisterHotkeyLocked() {
	if a.hotkeyStopChan != nil {
		close(a.hotkeyStopChan)
		a.hotkeyStopChan = nil
	}
	if a.currentHotkey != nil {
		_ = a.currentHotkey.Unregister()
		a.currentHotkey = nil
	}
}

// RegisterGlobalHotkey 注册全局呼出热键，具备完整的生命周期管理（支持多次重新注册且不泄露 goroutine）
func (a *App) RegisterGlobalHotkey(key1 string, key2 string) {
	a.hotkeyMu.Lock()
	defer a.hotkeyMu.Unlock()

	// 1. 先反注册并停止旧的监听 goroutine
	a.unregisterHotkeyLocked()

	// 2. 从映射中获取 Modifier 和 Key
	modifier, ok1 := internal.HotKeyMap[key1].(hotkey.Modifier)
	key, ok2 := internal.HotKeyMap[key2].(hotkey.Key)
	if !ok1 || !ok2 {
		fmt.Printf("无效的热键配置: %s + %s\n", key1, key2)
		return
	}

	hk := hotkey.New([]hotkey.Modifier{modifier}, key)
	err := hk.Register()
	if err != nil {
		fmt.Printf("注册热键 %s+%s 失败: %v\n", key1, key2, err)
		return
	}

	a.currentHotkey = hk
	stopChan := make(chan struct{})
	a.hotkeyStopChan = stopChan

	// 3. 启动受管理的事件监听循环
	go func(hk *hotkey.Hotkey, stop <-chan struct{}) {
		for {
			select {
			case <-stop:
				return
			case _, ok := <-hk.Keydown():
				if !ok {
					return
				}
				a.ToggleWindow()
			}
		}
	}(hk, stopChan)
}

// 你的热键触发逻辑
func (a *App) ToggleWindow() {
	if a.isVisible {
		a.isVisible = false
		a.action.Hide()
	} else {
		a.lastHwnd = a.action.RecordActiveWindow()
		procName, title := a.action.GetWindowContext(a.lastHwnd)
		cKey := internal.GenerateContextKey(procName, title)
		a.contextMu.Lock()
		a.currentContextKey = cKey
		a.contextMu.Unlock()

		a.isVisible = true
		a.action.ShowNoActivate()
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "window-shown", cKey)
		}
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

// HideAndRestore 只隐藏+恢复焦点，不执行粘贴
func (a *App) HideAndRestore() {
	a.isVisible = false
	a.action.Hide()
	a.action.RestoreFocus(a.lastHwnd)
}

// FontSizeWindowSizes 定义 5 个字体档位对应的推荐窗口尺寸 (宽 × 高)
var FontSizeWindowSizes = map[int][2]int{
	1: {395, 550},
	2: {420, 580},
	3: {450, 615},
	4: {480, 645},
	5: {515, 680},
}

// 进入设置模式：保持主窗口尺寸与位置不变 (方案 A 极简转场)
func (a *App) EnterSettingsMode() {
}

// 退出设置模式：保持主窗口尺寸与位置不变
func (a *App) ExitSettingsMode() {
}

// SetFontSizeLevel 设置字体档位 (1-5) 并联动同步调整并保存窗口推荐尺寸
func (a *App) SetFontSizeLevel(level int) {
	if level < 1 || level > 5 {
		level = 2
	}
	if a.config != nil {
		a.config.Appearance.FontSizeLevel = level
		if target, ok := FontSizeWindowSizes[level]; ok {
			a.config.Window.Width = target[0]
			a.config.Window.Height = target[1]
			a.action.SetSizeNative(target[0], target[1])
		}
		if a.configManager != nil {
			_ = a.configManager.Save(a.config)
		}
	}
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

// persistWindowSize is called (debounced) after a user resizes the window.
func (a *App) persistWindowSize(width, height int32) {
	if a.config == nil || a.configManager == nil {
		return
	}
	a.config.Window.Width = int(width)
	a.config.Window.Height = int(height)
	if err := a.configManager.Save(a.config); err != nil {
		fmt.Printf("保存窗口尺寸失败: %v\n", err)
	}
}

func (a *App) GetDataPath() string {
	return a.dataPath
}

func (a *App) GetKeys() string {
	return a.keys
}

// GetContextSuggestions returns the top recommended item IDs for the current active window scene.
func (a *App) GetContextSuggestions() []string {
	a.contextMu.RLock()
	cKey := a.currentContextKey
	a.contextMu.RUnlock()

	if a.statsManager == nil || cKey == "" {
		return []string{}
	}
	res := a.statsManager.GetTopItemIds(cKey, 3)
	if res == nil {
		return []string{}
	}
	return res
}

// RecordItemUsage records usage of an item under the current context.
func (a *App) RecordItemUsage(itemId string) {
	if a.statsManager == nil || itemId == "" {
		return
	}
	a.contextMu.RLock()
	cKey := a.currentContextKey
	a.contextMu.RUnlock()

	if cKey != "" {
		a.statsManager.RecordUsage(cKey, itemId)
	}
}
