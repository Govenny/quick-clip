package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/tailscale/win"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	user32                         = syscall.NewLazyDLL("user32.dll")
	procAttachThreadInput          = user32.NewProc("AttachThreadInput")
	procGetWindowThreadProcessId   = user32.NewProc("GetWindowThreadProcessId")
	procEnumWindows                = user32.NewProc("EnumWindows")
	procSetLayeredWindowAttributes = user32.NewProc("SetLayeredWindowAttributes")
)

const (
	SWP_NOSIZE       = 0x0001
	SWP_NOMOVE       = 0x0002
	SWP_NOACTIVATE   = 0x0010
	SWP_SHOWWINDOW   = 0x0040
	HWND_TOPMOST     = win.HWND(^uintptr(0)) // -1
	WS_EX_NOACTIVATE = 0x08000000
	GWL_EXSTYLE      = -20
	SWP_HIDEWINDOW   = 0x0080
	GA_ROOT          = 2
	WS_EX_TOOLWINDOW = 0x00000080 // 设为工具窗口，不显示在任务栏，且减少对焦点的干扰
	LWA_ALPHA        = 0x00000002
	WS_EX_LAYERED    = 0x00080000
)

type Action struct {
	selfHwnd win.HWND
}

func NewAction() *Action {
	return &Action{}
}

// 句柄操作----------------------------------------------------------------
func (a *Action) SetSelfHwnd(hwnd win.HWND) {
	// 打印一下找到的句柄对应的标题，看是不是你的 App 标题
	var buf [256]uint16
	win.GetWindowText(hwnd, &buf[0], 256)
	title := syscall.UTF16ToString(buf[:])
	fmt.Printf("绑定句柄: %d, 标题: %s\n", hwnd, title)
	a.selfHwnd = hwnd
}

func (a *Action) FindRealWailsWindow() win.HWND {
	var targetHwnd win.HWND
	myPid := uint32(syscall.Getpid())
	cb := syscall.NewCallback(func(h win.HWND, l uintptr) uintptr {
		var pid uint32
		win.GetWindowThreadProcessId(h, &pid)
		if pid == myPid {
			// 获取窗口样式
			style := uint32(win.GetWindowLong(h, win.GWL_STYLE))
			parent := win.GetParent(h)

			// 真正的 Wails 外壳窗口必须满足：
			// 1. 没有父窗口 (或者是真正的顶层)
			// 2. 窗口是可见的 (或者曾经可见)
			// 3. 不是那种系统工具窗口 (如消息窗口)
			if parent == 0 && (style&win.WS_VISIBLE != 0) {
				targetHwnd = h
				return 0 // 找到了，停止枚举
			}
		}
		return 1 // 继续找
	})

	procEnumWindows.Call(cb, 0)
	return targetHwnd
}

func GetRootHWND(hwnd win.HWND) win.HWND {
	for {
		parent := win.GetParent(hwnd)
		if parent == 0 {
			return hwnd
		}
		hwnd = parent
	}
}

// 显示窗口----------------------------------------------------------------
func (a *Action) ShowNoActivate() {
	if a.selfHwnd == 0 {
		return
	}

	// 1. 确保样式包含 NOACTIVATE 和 TOOLWINDOW
	exStyle := win.GetWindowLong(a.selfHwnd, GWL_EXSTYLE)
	win.SetWindowLong(a.selfHwnd, GWL_EXSTYLE, exStyle|WS_EX_TOOLWINDOW)

	win.ShowWindow(a.selfHwnd, win.SW_SHOW)

	// 2. 使用 SetWindowPos 显示
	win.SetWindowPos(
		a.selfHwnd,
		HWND_TOPMOST,
		0, 0, 0, 0,
		SWP_NOMOVE|SWP_NOSIZE|SWP_SHOWWINDOW,
	)
	win.SetForegroundWindow(a.selfHwnd)
	win.SetFocus(a.selfHwnd)

	// 3. 补偿显示
	// win.ShowWindow(a.selfHwnd, 4) // SW_SHOWNOACTIVATE
}

// Hide 封装隐藏
func (a *Action) Hide() {
	win.ShowWindow(a.selfHwnd, win.SW_HIDE)
}

func (a *Action) RecordActiveWindow() (hwnd win.HWND) {
	hwnd = win.GetForegroundWindow()
	return
}

// RestoreFocus 根据句柄恢复窗口焦点
func (a *Action) RestoreFocus(hwnd win.HWND) {
	if hwnd == 0 {
		return
	}

	// 1. 获取当前线程ID和目标窗口的线程ID
	// 这里的 0 是 GetCurrentThreadId 的意思（在某些封装中），但在 syscall 中我们需要显式调用
	// 既然用了 tailscale/win，我们尽量复用它的，如果没有就用 syscall

	// 获取当前线程 ID
	curThreadID := win.GetCurrentThreadId()

	// 获取目标窗口的线程 ID
	var targetProcessID uint32
	// win.GetWindowThreadProcessId 返回的是线程ID
	targetThreadID := win.GetWindowThreadProcessId(hwnd, &targetProcessID)

	// 2. 关键步骤：连接线程输入 (Attach)
	// 如果目标线程和当前线程不同，才需要 Attach
	if curThreadID != targetThreadID {
		attachThreadInput(curThreadID, targetThreadID, true)
		defer attachThreadInput(curThreadID, targetThreadID, false) // 确保函数结束时 Detach
	}

	// 3. 处理最小化情况
	if win.IsIconic(hwnd) {
		win.ShowWindow(hwnd, win.SW_RESTORE)
	} else {
		win.ShowWindow(hwnd, win.SW_SHOW)
	}

	// 4. 设置前台窗口 & 设置焦点
	// 因为已经 Attach 了线程，这时候 SetFocus 才有权限生效
	win.SetForegroundWindow(hwnd)
	win.SetFocus(hwnd)
}

// 封装 AttachThreadInput 系统调用
func attachThreadInput(idAttach, idAttachTo uint32, fAttach bool) {
	flag := 0
	if fAttach {
		flag = 1
	}
	procAttachThreadInput.Call(
		uintptr(idAttach),
		uintptr(idAttachTo),
		uintptr(flag),
	)
}

const (
	INPUT_KEYBOARD  = 1
	KEYEVENTF_KEYUP = 0x0002
	VK_CONTROL      = 0x11
	VK_V            = 0x56
)

// 如果上面的不行，试试这个原生 syscall 版本
func (a *Action) SendPaste() {
	// 1. 稍微延时，等待窗口隐藏和焦点彻底回到原位
	time.Sleep(150 * time.Millisecond)

	user32 := syscall.NewLazyDLL("user32.dll")
	keybd := user32.NewProc("keybd_event")

	const (
		VK_CONTROL = 0x11
		VK_V       = 0x56
		VK_MENU    = 0x12 // Alt 键
		KEYUP      = 0x0002
	)

	// 2. 【关键】强制松开物理 Alt 键
	// 如果用户按住 Alt+Space 触发，点击时 Alt 可能还没松开
	// 模拟一次 Alt 的 KeyUp，确保环境“干净”
	keybd.Call(uintptr(VK_MENU), 0, KEYUP, 0)

	// 3. 开始模拟 Ctrl + V
	// 按下 Ctrl
	keybd.Call(uintptr(VK_CONTROL), 0, 0, 0)
	// 按下 V
	keybd.Call(uintptr(VK_V), 0, 0, 0)

	// 松开 V
	keybd.Call(uintptr(VK_V), 0, KEYUP, 0)
	// 松开 Ctrl
	keybd.Call(uintptr(VK_CONTROL), 0, KEYUP, 0)

	fmt.Println("粘贴指令已发送")
}

func (a *Action) SetSizeNative(width, height int) {
	if a.selfHwnd == 0 {
		return
	}

	// 使用 SetWindowPos 进行缩放
	// SWP_NOMOVE: 保持当前位置
	// SWP_NOZORDER: 保持当前的层级（不改变置顶状态）
	// SWP_NOACTIVATE: 关键！缩放时不激活窗口，光标不丢失
	win.SetWindowPos(
		a.selfHwnd,
		0,    // 忽略，因为用了 SWP_NOZORDER
		0, 0, // 忽略，因为用了 SWP_NOMOVE
		int32(width),
		int32(height),
		win.SWP_NOMOVE|win.SWP_NOZORDER,
	)
}

func (a *Action) SetTransparency(alpha uint8) {
	if a.selfHwnd == 0 {
		return
	}

	// 获取当前扩展样式
	exStyle := win.GetWindowLong(a.selfHwnd, GWL_EXSTYLE)

	// 设置分层窗口样式
	win.SetWindowLong(a.selfHwnd, GWL_EXSTYLE, exStyle|WS_EX_LAYERED)

	// 设置透明度
	procSetLayeredWindowAttributes.Call(
		uintptr(a.selfHwnd),
		0, // 颜色键，这里不使用颜色键
		uintptr(alpha),
		LWA_ALPHA,
	)
}

func (a *Action) ExportJson(content []any, ctx context.Context) {
	byteData, err := json.Marshal(content)
	if err != nil || byteData == nil {
		return
	}

	filePath, err := runtime.SaveFileDialog(ctx, runtime.SaveDialogOptions{
		Title:            "导出密码文件(明文)",
		DefaultFilename:  "resource.json",
		DefaultDirectory: "",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "JSON文件 (*.json)",
				Pattern:     "*.json",
			},
		},
	})

	if err != nil {
		return
	}

	if filePath == "" {
		return
	}

	// 写入文件
	err = os.WriteFile(filePath, byteData, 0644)
	if err != nil {
		return
	}

}
