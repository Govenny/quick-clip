package main

import (
	"context"
	"encoding/json"
	"os"
	"quick-clip/internal"

	jsoniter "github.com/json-iterator/go"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.design/x/hotkey" // 注意：这个库通常要求在主线程初始化
)

// App struct
type App struct {
	ctx         context.Context
	content     []any
	keys        string
	shortcutMgr *internal.ShortcutManager
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		keys: "11112222111122221111222211112222",
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
		hk := hotkey.New([]hotkey.Modifier{hotkey.ModAlt}, hotkey.KeySpace)
		err := hk.Register()
		if err != nil {
			return
		}

		// 监听热键事件
		for range hk.Keydown() {
			runtime.WindowShow(a.ctx)
			// runtime.WindowSetFocus(a.ctx)
		}
	}()
}
