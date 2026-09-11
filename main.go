package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"quick-clip/internal"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// 优化 Chromium / WebView2 启动参数，显著降低后台常驻与网络开销，同时保证流畅响应
	existingArgs := os.Getenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS")
	optArgs := []string{
		"--renderer-process-limit=1",
		"--disable-background-networking",
		"--disable-component-update",
		"--disable-domain-reliability",
		"--disable-sync",
		"--disable-features=Translate,MediaRouter,OptimizationHints",
		"--disk-cache-size=10485760",
	}
	if existingArgs != "" {
		os.Setenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS", existingArgs+" "+strings.Join(optArgs, " "))
	} else {
		os.Setenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS", strings.Join(optArgs, " "))
	}

	action := internal.NewAction()
	configManager := internal.NewConfigManager()
	config, _ := configManager.Load()
	fmt.Println(config)
	app := NewApp(action, configManager, config)
	trayMgr := internal.NewTrayManager(action, app)
	appService := internal.NewAppService()

	configDir, _ := os.UserConfigDir()
	appRoot := filepath.Join(configDir, "quick-clip")

	// Create application with options
	windowWidth := config.Window.Width
	if windowWidth <= 0 {
		windowWidth = 420
	}
	windowHeight := config.Window.Height
	if windowHeight <= 0 {
		windowHeight = 580
	}

	err := wails.Run(&options.App{
		Title:     "quick-clip",
		Width:     windowWidth,
		Height:    windowHeight,
		MinWidth:  380,
		MinHeight: 400,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 0},
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			// 如果不是通过托盘点击“退出”的，就只隐藏窗口，不关闭
			if !trayMgr.IsQuitting() {
				//runtime.WindowHide(ctx) // 隐藏窗口（任务栏图标也会消失）
				app.action.Hide()
				return true // 返回 true 阻止默认的关闭行为
			}
			return false // 返回 false 允许关闭
		},
		OnStartup: func(ctx context.Context) {
			// 如果你有 App 的 startup 逻辑，先执行
			app.startup(ctx)
			trayMgr.Run(ctx)
		},
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
			action,
			appService,
		},
		Frameless:   true,
		AlwaysOnTop: true,
		Windows: &windows.Options{
			// 允许窗口在失去焦点时继续渲染
			WebviewGpuIsDisabled: false,
			WebviewUserDataPath:  filepath.Join(appRoot, "cache"),
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			// 可选：设置背景类型（如 Mica, Acrylic 等磨砂效果）
			BackdropType: windows.Mica,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
