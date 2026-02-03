package main

import (
	"context"
	"embed"
	"quick-clip/internal"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	trayMgr := internal.NewTrayManager()
	app := NewApp()
	action := internal.NewAction()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "quick-clip",
		Width:  256,
		Height: 384,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
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
		},
		Frameless:   true,
		AlwaysOnTop: true,
		// Windows: &windows.Options{
		// 	// 允许窗口在失去焦点时继续渲染
		// 	WebviewUserDataFolder: "",
		// },
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
