package internal

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	hook "github.com/robotn/gohook"
)

type ShortcutManager struct {
	ctx context.Context
}

func NewShortcutManager(ctx context.Context) *ShortcutManager {
	return &ShortcutManager{
		ctx: ctx,
	}
}

// RegisterGlobalShortcut 注册全局快捷键
func (sm *ShortcutManager) RegisterGlobalShortcut(hotkey string) error {
	// 监听全局快捷键
	hook.Register(
		hook.KeyDown,
		[]string{hotkey},
		func(e hook.Event) {
			// 唤醒主窗口
			runtime.WindowShow(sm.ctx)
			runtime.WindowUnminimise(sm.ctx)
		})

	// 开始监听（在单独的goroutine中）
	go func() {
		s := hook.Start()
		<-hook.Process(s)
	}()

	return nil
}

// RegisterShowHideShortcut 注册显示/隐藏窗口的全局快捷键
func (sm *ShortcutManager) RegisterShowHideShortcut(showKey string, hideKey string) {
	// 注册显示窗口的快捷键（全局）
	hook.Register(hook.KeyDown, []string{showKey}, func(e hook.Event) {
		runtime.WindowShow(sm.ctx)
		runtime.WindowUnminimise(sm.ctx)
	})

	// 注册隐藏窗口的快捷键
	hook.Register(hook.KeyDown, []string{hideKey}, func(e hook.Event) {
		runtime.WindowHide(sm.ctx)
	})
}
