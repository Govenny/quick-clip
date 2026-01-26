package main

import (
	"context"
	"encoding/json"
	"os"

	jsoniter "github.com/json-iterator/go"
)

// App struct
type App struct {
	ctx     context.Context
	content []any
	keys    string
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
}

// shutdown is called when the app is about to close
func (a *App) shutdown(ctx context.Context) {
	a.CloseAll(a.content)
}

func (a *App) GetContent() []any {
	return a.content
}

func (a *App) SaveContent(data []any) {
	a.content = data
}

func (a *App) CloseAll(data []any) {
	// map -> []byte
	byteData, err := json.Marshal(data)
	if err != nil {
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
