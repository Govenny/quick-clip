package internal

import (
	"context"
	_ "embed"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"github.com/energye/systray"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	modTrayUser32         = syscall.NewLazyDLL("user32.dll")
	procTrayFindWindowW   = modTrayUser32.NewProc("FindWindowW")
	procTrayPostMessageW  = modTrayUser32.NewProc("PostMessageW")
	procTraySetForeground = modTrayUser32.NewProc("SetForegroundWindow")
)

const wmNull = 0x0000

func getSystrayHwnd() uintptr {
	clsName, _ := syscall.UTF16PtrFromString("SystrayClass")
	hwnd, _, _ := procTrayFindWindowW.Call(uintptr(unsafe.Pointer(clsName)), 0)
	return hwnd
}

type AppInterface interface {
	GetContent() []any
	SaveContent(content []any)
	ShowWindow()
}

//go:embed asset/icon32.png
var iconData []byte

type TrayManager struct {
	ctx        context.Context
	isQuitting bool
	action     *Action
	app        AppInterface
}

func NewTrayManager(action *Action, app AppInterface) *TrayManager {
	return &TrayManager{
		isQuitting: false,
		action:     action,
		app:        app,
	}
}

// 这个方法将在 Wails 启动时被调用
func (tm *TrayManager) Run(ctx context.Context) {
	tm.ctx = ctx
	// 关键点：在固定 OS 线程的 goroutine 中启动 systray，杜绝 Win32 消息循环在 Go 调度器中跨线程漂移导致消息丢失
	go func() {
		runtime.LockOSThread()
		systray.Run(tm.onReady, tm.onExit)
	}()
}

func (tm *TrayManager) onReady() {
	systray.SetIcon(iconData)
	systray.SetTitle("Quick-Clip")
	systray.SetTooltip("Quick-Clip")

	// 1. 添加菜单项
	mShow := systray.AddMenuItem("显示主界面", "显示窗口")
	mHotkey := systray.AddMenuItem("设置", "设置页面")
	mOut := systray.AddMenuItem("导出", "导出Json")
	mIn := systray.AddMenuItem("导入", "导入Json")
	mQuit := systray.AddMenuItem("退出", "退出程序")

	// 2. 菜单项点击事件绑定
	mShow.Click(func() {
		tm.app.ShowWindow()
	})

	mQuit.Click(func() {
		tm.isQuitting = true
		systray.Quit()
		wruntime.Quit(tm.ctx)
	})

	mHotkey.Click(func() {
		tm.app.ShowWindow()
		time.Sleep(80 * time.Millisecond)
		wruntime.EventsEmit(tm.ctx, "show-settings")
	})

	mOut.Click(func() {
		tm.action.ExportJson(tm.app.GetContent(), tm.ctx)
	})

	mIn.Click(func() {
		content := tm.action.ImportJson(tm.ctx)
		if content != nil {
			tm.app.SaveContent(content)
		}
		time.Sleep(time.Second)
		wruntime.EventsEmit(tm.ctx, "update-content")
	})

	// 托盘左键点击（显示窗口）
	systray.SetOnClick(func(menu systray.IMenu) {
		tm.app.ShowWindow()
	})

	var trayHwnd uintptr

	// 托盘右键点击（显示菜单）
	systray.SetOnRClick(func(menu systray.IMenu) {
		if tm.action != nil {
			// 若主窗口当前可见，先隐藏主窗口，防止前端失焦定时器在菜单弹出瞬间切焦点导致菜单被强制关闭
			tm.action.Hide()
		}

		if trayHwnd == 0 {
			trayHwnd = getSystrayHwnd()
		}
		if trayHwnd != 0 {
			// 1. 【微软官方规范 KB131293】先将托盘宿主窗口置于前台，确保菜单获得焦点并能捕获外部点击
			procTraySetForeground.Call(trayHwnd)
		}

		// 2. 弹出上下文菜单 (内部调用阻塞的 TrackPopupMenu)
		_ = menu.ShowMenu()

		// 3. 【核心修复 - 微软官方规范 KB131293】
		// 菜单关闭后，必须向宿主窗口投递一条 WM_NULL 消息！
		// 若缺失此调用，Windows 内部的托盘焦点状态不会释放，导致下一次/偶尔右键点击托盘时菜单闪退或不弹出！
		if trayHwnd != 0 {
			procTrayPostMessageW.Call(trayHwnd, wmNull, 0, 0)
		}
	})

	// 托盘双击（显示窗口）
	systray.SetOnDClick(func(menu systray.IMenu) {
		tm.app.ShowWindow()
	})
}

func (tm *TrayManager) onExit() {
	// 清理工作
}

func (tm *TrayManager) IsQuitting() bool {
	return tm.isQuitting
}
