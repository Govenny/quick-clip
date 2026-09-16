package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/tailscale/win"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// 增加必要的结构体
type POINT struct {
	X, Y int32
}

type RECT struct {
	Left, Top, Right, Bottom int32
}
type MONITORINFO struct {
	Size    uint32
	Monitor RECT
	Work    RECT
	Flags   uint32
}

var (
	user32                         = syscall.NewLazyDLL("user32.dll")
	procAttachThreadInput          = user32.NewProc("AttachThreadInput")
	procGetWindowThreadProcessId   = user32.NewProc("GetWindowThreadProcessId")
	procEnumWindows                = user32.NewProc("EnumWindows")
	procSetLayeredWindowAttributes = user32.NewProc("SetLayeredWindowAttributes")
	procGetCursorPos               = user32.NewProc("GetCursorPos")
	procMonitorFromPoint           = user32.NewProc("MonitorFromPoint")
	procGetMonitorInfoW            = user32.NewProc("GetMonitorInfoW")
	procGetAncestor                = user32.NewProc("GetAncestor")
	procKeybdEvent                 = user32.NewProc("keybd_event")
	procBringWindowToTop           = user32.NewProc("BringWindowToTop")
	procIsWindow                   = user32.NewProc("IsWindow")
	procGetAsyncKeyState           = user32.NewProc("GetAsyncKeyState")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
)

const (
	SWP_NOSIZE               = 0x0001
	SWP_NOMOVE               = 0x0002
	SWP_NOACTIVATE           = 0x0010
	SWP_SHOWWINDOW           = 0x0040
	HWND_TOPMOST             = win.HWND(^uintptr(0)) // -1
	WS_EX_NOACTIVATE         = 0x08000000
	GWL_EXSTYLE              = -20
	SWP_HIDEWINDOW           = 0x0080
	GA_ROOT                  = 2
	WS_EX_TOOLWINDOW         = 0x00000080 // 设为工具窗口，不显示在任务栏，且减少对焦点的干扰
	LWA_ALPHA                = 0x00000002
	WS_EX_LAYERED            = 0x00080000
	MONITOR_DEFAULTTONEAREST = 0x00000002
)

type Action struct {
	selfHwnd win.HWND

	oldWndProc     uintptr
	newWndProc     uintptr
	onResized      func(int32, int32)
	resizeSuppress bool
	resizeTimer    *time.Timer
	memoryTrimmer  *MemoryTrimmer
}

func NewAction() *Action {
	return &Action{
		memoryTrimmer: NewMemoryTrimmer(),
	}
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

func (a *Action) GetSelfHwnd() win.HWND {
	return a.selfHwnd
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
	if a.memoryTrimmer != nil {
		a.memoryTrimmer.CancelTrim()
	}

	if a.selfHwnd == 0 {
		return
	}

	// 1. 获取鼠标坐标
	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	// 2. 【核心修复】打包 POINT 结构体为一个 uintptr
	// 在 64 位系统下，POINT (X, Y) 各占 32 位，正好填满一个 64 位的 uintptr
	// 低 32 位存 X, 高 32 位存 Y
	packedPt := uintptr(uint32(pt.X)) | uintptr(uint32(pt.Y))<<32

	// MONITOR_DEFAULTTONEAREST = 2
	hMonitor, _, _ := procMonitorFromPoint.Call(packedPt, 2)

	// 3. 获取显示器信息
	var mi MONITORINFO
	mi.Size = uint32(unsafe.Sizeof(mi))
	procGetMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(&mi)))

	// 4. 获取窗口大小
	var rect win.RECT
	win.GetWindowRect(a.selfHwnd, &rect)
	appW := rect.Right - rect.Left
	appH := rect.Bottom - rect.Top

	// 5. 计算坐标 (mi.Work.Left/Top 是多屏坐标的关键偏移)
	monW := mi.Work.Right - mi.Work.Left
	monH := mi.Work.Bottom - mi.Work.Top

	x := mi.Work.Left + (monW-appW)/2
	y := mi.Work.Top + (monH-appH)/2

	// 6. 移动并显示
	// 确保移除 SWP_NOMOVE
	win.SetWindowPos(
		a.selfHwnd,
		win.HWND_TOPMOST,
		x, y, 0, 0,
		win.SWP_NOSIZE|win.SWP_SHOWWINDOW|win.SWP_NOACTIVATE,
	)

	win.ShowWindow(a.selfHwnd, win.SW_SHOWNOACTIVATE)
	win.SetForegroundWindow(a.selfHwnd)
	win.SetFocus(a.selfHwnd)
}

// Hide 封装隐藏
func (a *Action) Hide() {
	win.ShowWindow(a.selfHwnd, win.SW_HIDE)
	if a.memoryTrimmer != nil {
		a.memoryTrimmer.ScheduleTrim(120 * time.Second)
	}
}

func (a *Action) RecordActiveWindow() (hwnd win.HWND) {
	hwnd = win.GetForegroundWindow()
	return
}

// GetWindowContext extracts the process name and window title from a given HWND.
func (a *Action) GetWindowContext(hwnd win.HWND) (procName string, title string) {
	if hwnd == 0 {
		return "", ""
	}

	// 1. 获取窗口标题
	var buf [512]uint16
	win.GetWindowText(hwnd, &buf[0], 512)
	title = syscall.UTF16ToString(buf[:])

	// 2. 获取进程 ID
	var pid uint32
	win.GetWindowThreadProcessId(hwnd, &pid)
	if pid == 0 {
		return "", title
	}

	// 3. 打开进程获取可执行文件名 (PROCESS_QUERY_LIMITED_INFORMATION = 0x1000)
	hProc, _, _ := procOpenProcess.Call(0x1000, 0, uintptr(pid))
	if hProc != 0 {
		defer procCloseHandle.Call(hProc)
		var imgBuf [1024]uint16
		var size uint32 = 1024
		ret, _, _ := procQueryFullProcessImageNameW.Call(
			hProc,
			0,
			uintptr(unsafe.Pointer(&imgBuf[0])),
			uintptr(unsafe.Pointer(&size)),
		)
		if ret != 0 {
			fullPath := syscall.UTF16ToString(imgBuf[:size])
			procName = filepath.Base(fullPath)
		}
	}

	return procName, title
}

// GenerateContextKey produces a clean, consistent scene fingerprint from process name and title.
func GenerateContextKey(procName, title string) string {
	procName = strings.ToLower(strings.TrimSpace(procName))
	title = strings.TrimSpace(title)

	// 常见浏览器集合
	browsers := map[string]bool{
		"chrome.exe":        true,
		"msedge.exe":        true,
		"firefox.exe":       true,
		"brave.exe":         true,
		"opera.exe":         true,
		"360chrome.exe":     true,
		"sogouexplorer.exe": true,
		"qqbrowser.exe":     true,
	}

	if browsers[procName] {
		cleanTitle := CleanBrowserTitle(title)
		if cleanTitle != "" {
			return procName + "::" + cleanTitle
		}
		return procName
	}

	// 通用桌面应用直接以进程名为主场景指纹
	if procName != "" {
		return procName
	}
	if title != "" {
		return title
	}
	return "default"
}

var (
	reUnreadPrefix   = regexp.MustCompile(`^[\(\[\{]\d+\+?[\)\]\}]\s*`)
	reEdgeTabCountZh = regexp.MustCompile(`(?i)\s+(和另外|和其余)\s*\d+\s*个(页面|标签页).*$`)
	reEdgeTabCountEn = regexp.MustCompile(`(?i)\s+and\s+\d+\s+other\s+tabs?.*$`)
	reProfileSuffix  = regexp.MustCompile(`(?i)\s*-\s*(用户\s*\d+|个人|工作|Work|Profile\s*\d+|Default)\s*-\s*(Microsoft\s*Edge|Google\s*Chrome).*$`)
	reBrandSuffix    = regexp.MustCompile(`(?i)\s*-\s*(Microsoft\s*Edge|Google\s*Chrome|Mozilla\s*Firefox|Brave|Opera|Vivaldi|360.+|QQ浏览器|搜狗.+).*$`)
)

// CleanBrowserTitle strips browser branding suffixes, tab count noise, and unread badge prefixes.
func CleanBrowserTitle(title string) string {
	title = strings.TrimSpace(title)

	// 1. 剔除未读前缀，如 (3) 或 [5条]
	title = reUnreadPrefix.ReplaceAllString(title, "")

	// 2. 彻底剔除 Edge 垂直标签页的动态计数尾巴（如 " 和另外 11 个页面 - 用户1 - Microsoft Edge"）
	title = reEdgeTabCountZh.ReplaceAllString(title, "")
	title = reEdgeTabCountEn.ReplaceAllString(title, "")

	// 3. 剔除带有用户配置文件名的浏览器后缀（如 " - 用户1 - Microsoft Edge"）
	title = reProfileSuffix.ReplaceAllString(title, "")

	// 4. 剔除常规浏览器品牌后缀（如 " - Google Chrome"）
	title = reBrandSuffix.ReplaceAllString(title, "")

	title = strings.TrimSpace(title)

	// 5. 限制最大长度（避免过长的动态参数标题）
	runes := []rune(title)
	if len(runes) > 60 {
		title = string(runes[:60])
	}
	return title
}

func isWindow(hwnd win.HWND) bool {
	if hwnd == 0 {
		return false
	}
	ret, _, _ := procIsWindow.Call(uintptr(hwnd))
	return ret != 0
}

func isKeyDown(vk int) bool {
	ret, _, _ := procGetAsyncKeyState.Call(uintptr(vk))
	return (ret & 0x8000) != 0
}

// RestoreFocus 根据句柄恢复目标窗口前台与焦点
func (a *Action) RestoreFocus(hwnd win.HWND) {
	if hwnd == 0 || hwnd == a.selfHwnd || !isWindow(hwnd) {
		return
	}

	// 1. 如果窗口最小化，恢复它
	if win.IsIconic(hwnd) {
		win.ShowWindow(hwnd, win.SW_RESTORE)
	}

	// 2. 获取当前前台窗口线程ID与目标窗口线程ID
	curFg := win.GetForegroundWindow()
	var curThreadID uint32
	if curFg != 0 {
		curThreadID = win.GetWindowThreadProcessId(curFg, nil)
	}
	targetThreadID := win.GetWindowThreadProcessId(hwnd, nil)
	selfThreadID := win.GetCurrentThreadId()

	// 3. 附加线程输入，将前台权限平滑移交目标窗口
	if curThreadID != 0 && curThreadID != targetThreadID {
		procAttachThreadInput.Call(uintptr(curThreadID), uintptr(targetThreadID), 1)
		defer procAttachThreadInput.Call(uintptr(curThreadID), uintptr(targetThreadID), 0)
	}
	if selfThreadID != targetThreadID {
		procAttachThreadInput.Call(uintptr(selfThreadID), uintptr(targetThreadID), 1)
		defer procAttachThreadInput.Call(uintptr(selfThreadID), uintptr(targetThreadID), 0)
	}

	// 4. 设置目标窗口为前台窗口（Windows 会自动激活它并保持/恢复输入框对焦）
	// 注意：绝对不要调用 SetFocus(hwnd)，否则会把键盘焦点从浏览器输入框抢走
	win.SetForegroundWindow(hwnd)
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
	VK_MENU         = 0x12 // Alt 键
)

// SendPaste 模拟 Ctrl + V 粘贴按键
func (a *Action) SendPaste() {
	// 1. 只有在物理 Alt 键仍被按下的情况下才释放，避免随意发送 Alt 导致浏览器输入框失焦
	if isKeyDown(VK_MENU) {
		procKeybdEvent.Call(uintptr(VK_MENU), 0, KEYEVENTF_KEYUP, 0)
		time.Sleep(5 * time.Millisecond)
	}

	// 2. 开始模拟 Ctrl + V，微小间隔保证消息队列准确捕获按键序列
	// 按下 Ctrl
	procKeybdEvent.Call(uintptr(VK_CONTROL), 0, 0, 0)
	time.Sleep(10 * time.Millisecond)

	// 按下 V
	procKeybdEvent.Call(uintptr(VK_V), 0, 0, 0)
	time.Sleep(15 * time.Millisecond)

	// 松开 V
	procKeybdEvent.Call(uintptr(VK_V), 0, KEYEVENTF_KEYUP, 0)
	time.Sleep(10 * time.Millisecond)

	// 松开 Ctrl
	procKeybdEvent.Call(uintptr(VK_CONTROL), 0, KEYEVENTF_KEYUP, 0)

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

func (a *Action) ImportJson(ctx context.Context) []any {
	filePath, err := runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{
		Title:            "导入密码文件(明文)",
		DefaultDirectory: "",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "JSON文件 (*.json)",
				Pattern:     "*.json",
			},
		},
	})

	if err != nil {
		return nil
	}

	if filePath == "" {
		return nil
	}

	// 读取文件
	byteData, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	var content []any
	err = json.Unmarshal(byteData, &content)
	if err != nil {
		return nil
	}
	return content
}

// resizeAction bridges the package-level window procedure back to the live Action.
var resizeAction *Action

// SetOnResized registers the callback invoked (debounced) after the window is resized by the user.
func (a *Action) SetOnResized(cb func(int32, int32)) {
	a.onResized = cb
}

// SetResizeSuppressed temporarily disables resize persistence, e.g. while toggling settings window size.
func (a *Action) SetResizeSuppressed(suppress bool) {
	a.resizeSuppress = suppress
}

// InstallResizeTracker subclasses the Wails shell window to observe WM_SIZE and persist user resizes.
func (a *Action) InstallResizeTracker(hwnd win.HWND) {
	if hwnd == 0 {
		return
	}
	resizeAction = a
	a.newWndProc = syscall.NewCallback(resizeWindowProc)
	a.oldWndProc = win.SetWindowLongPtr(hwnd, win.GWLP_WNDPROC, a.newWndProc)
}

func resizeWindowProc(hwnd win.HWND, msg uint32, wparam, lparam uintptr) uintptr {
	if msg == win.WM_SIZE && resizeAction != nil && wparam != 1 {
		width := int32(uint16(lparam & 0xFFFF))
		height := int32(uint16((lparam >> 16) & 0xFFFF))
		resizeAction.handleResized(width, height)
	}
	if resizeAction == nil || resizeAction.oldWndProc == 0 {
		return 0
	}
	return win.CallWindowProc(resizeAction.oldWndProc, hwnd, msg, wparam, lparam)
}

func (a *Action) handleResized(width, height int32) {
	if a.resizeSuppress || width <= 0 || height <= 0 || a.onResized == nil {
		return
	}
	if a.resizeTimer != nil {
		a.resizeTimer.Stop()
	}
	a.resizeTimer = time.AfterFunc(400*time.Millisecond, func() {
		a.onResized(width, height)
	})
}
