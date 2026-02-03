package main

import (
	"context"
	"encoding/json"
	"os"
	"quick-clip/internal"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/tailscale/win"
	"golang.design/x/hotkey" // 注意：这个库通常要求在主线程初始化
)

// App struct
type App struct {
	ctx       context.Context
	content   []any
	keys      string
	action    *internal.Action
	isVisible bool
	lastHwnd  win.HWND
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		keys:      "11112222111122221111222211112222",
		action:    internal.NewAction(),
		isVisible: false,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	content, err := os.ReadFile("./resource.json")
	if os.IsNotExist(err) {
		// 文件不存在，创建文件并写入初始数据
		initialData := []byte("[]") // 空的JSON数组
		err = os.WriteFile("./resource.json", initialData, 0644)
		if err != nil {
			return
		}
		content = initialData // 使用刚创建的初始数据
	} else if err != nil {
		// 其他读取错误
		return
	}

	// decrypted, err := internal.DecryptBytes(content, a.keys)
	// if err != nil {
	// 	return
	// }
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	err = json.Unmarshal(content, &a.content)

	// 注册全局热键
	a.registerGlobalHotkey()

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
	a.saveToFile()
}

func (a *App) GetContent() []any {
	return a.content
}

func (a *App) SaveContent(data []any) {
	a.content = data
	a.saveToFile()
}

func (a *App) saveToFile() {
	byteData, err := json.Marshal(a.content)
	if err != nil {
		return
	} else if byteData == nil {
		return
	}

	// resource, err := internal.EncryptBytes(byteData, a.keys)
	// if err != nil {
	// 	return
	// }

	err = os.WriteFile("./resource.json", byteData, 0644)
	if err != nil {
		return
	}
}

func (a *App) registerGlobalHotkey() {
	go func() {
		// 注册 Alt + Space
		// ModAlt, KeySpace 需要根据库的定义
		hk := hotkey.New([]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeyQ)
		err := hk.Register()
		if err != nil {
			return
		}

		// 监听热键事件
		for range hk.Keydown() {
			a.toggleWindow()
		}
	}()
}

// 你的热键触发逻辑
func (a *App) toggleWindow() {
	if a.isVisible {
		a.isVisible = false
		a.action.Hide()
	} else {
		a.lastHwnd = a.action.RecordActiveWindow()
		a.isVisible = true
		a.action.ShowNoActivate()
	}
}

func (a *App) PasteAndHide() {
	// 1. 隐藏窗口（使用我们之前写的原生 Hide）
	// 此时焦点会自动回到上一个窗口，或者配合 RestoreFocus 强行还回去
	a.isVisible = false
	a.action.Hide()
	a.action.RestoreFocus(a.lastHwnd)
	time.Sleep(100 * time.Millisecond)
	// 如果你用了 ShowNoActivate，焦点理论上还在原处
	// 但为了保险，可以显式还一下焦点（如果你记录了 lastHwnd）
	// if a.lastHwnd != 0 {
	//     a.action.RestoreFocus(a.lastHwnd)
	// }

	// 2. 模拟粘贴
	// 建议新开一个协程，避免阻塞 Wails 的前端回调
	go a.action.SendPaste()
}
