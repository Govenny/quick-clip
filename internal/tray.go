package internal

import (
	"context"
	_ "embed" // 必须引入

	"github.com/energye/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed asset/icon32.png
var iconData []byte

type TrayManager struct {
	ctx        context.Context
	isQuitting bool
}

func NewTrayManager() *TrayManager {
	return &TrayManager{isQuitting: false}
}

// 这个方法将在 Wails 启动时被调用
func (tm *TrayManager) Run(ctx context.Context) {
	tm.ctx = ctx
	// 关键点：在 goroutine 中启动 systray，避免卡死 Wails 的主线程
	go systray.Run(tm.onReady, tm.onExit)
}

func (tm *TrayManager) onReady() {
	// ---------------------------------------------------------
	// 核心修复逻辑：在内存中构建一个标准的 32x32 Windows ICO
	// ---------------------------------------------------------

	// // 1. 解码原始图片
	// img, _, err := image.Decode(bytes.NewReader(iconData))
	// if err != nil {
	// 	fmt.Println("图片解码炸了:", err)
	// 	return
	// }

	// // 2. 强制缩放到 32x32 (Windows 托盘标准大小)
	// // 如果不缩放，高清图直接扔进去 100% 报错
	// bounds := img.Bounds()
	// dst := image.NewRGBA(image.Rect(0, 0, 32, 32))

	// // 使用高质量缩放
	// draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	// // 3. 将缩放后的图转为 PNG 字节流
	// pngBuf := new(bytes.Buffer)
	// if err := png.Encode(pngBuf, dst); err != nil {
	// 	fmt.Println("PNG 编码炸了:", err)
	// 	return
	// }
	// pngBytes := pngBuf.Bytes()

	// // 4. 【魔法时刻】手动拼接 ICO 文件头
	// // Windows API 需要看到这个头才会认为它是图标
	// icoBuf := new(bytes.Buffer)

	// // ICO Header (6 bytes)
	// // Reserved=0, Type=1(Icon), Count=1(Image count)
	// binary.Write(icoBuf, binary.LittleEndian, []uint16{0, 1, 1})

	// // Icon Directory Entry (16 bytes)
	// icoBuf.WriteByte(32)                                             // Width
	// icoBuf.WriteByte(32)                                             // Height
	// icoBuf.WriteByte(0)                                              // ColorCount
	// icoBuf.WriteByte(0)                                              // Reserved
	// binary.Write(icoBuf, binary.LittleEndian, uint16(1))             // Planes
	// binary.Write(icoBuf, binary.LittleEndian, uint16(32))            // BitCount
	// binary.Write(icoBuf, binary.LittleEndian, uint32(len(pngBytes))) // SizeInBytes
	// binary.Write(icoBuf, binary.LittleEndian, uint32(22))            // Offset (6+16=22)

	// // 写入真正的 PNG 数据
	// icoBuf.Write(pngBytes)

	// // 5. 设置图标
	// finalIconData := icoBuf.Bytes()

	// // 6. 可保存到本地
	// err = os.WriteFile("D:\\DownLoad\\icon32.png", finalIconData, 0644)
	// if err != nil {
	// 	fmt.Println("写入 ICO 文件炸了:", err)
	// 	return
	// }

	systray.SetIcon(iconData)

	// ---------------------------------------------------------
	// 设置图标
	systray.SetTitle("Quick-Clip")
	systray.SetTooltip("Quick-Clip")

	// 1. 添加菜单项
	mShow := systray.AddMenuItem("显示主界面", "显示窗口")
	mQuit := systray.AddMenuItem("退出", "退出程序")

	// 2. 【核心修改】使用回调函数，而不是 Channel
	// energye/systray 的特点：使用 Click 方法传入一个函数
	mShow.Click(func() {
		// 显示窗口
		runtime.WindowShow(tm.ctx)
	})

	mQuit.Click(func() {
		// 退出程序
		tm.isQuitting = true
		systray.Quit()
		runtime.Quit(tm.ctx)
	})

	// 如果需要设置托盘左键点击（显示窗口）
	systray.SetOnClick(func(menu systray.IMenu) {
		runtime.WindowShow(tm.ctx)
	})

	// 托盘右键点击（显示菜单）
	systray.SetOnRClick(func(menu systray.IMenu) {
		menu.ShowMenu()
	})

	systray.SetOnDClick(func(menu systray.IMenu) {
		runtime.WindowShow(tm.ctx)
	})
}

func (tm *TrayManager) onExit() {
	// 清理工作
}

func (tm *TrayManager) IsQuitting() bool {
	return tm.isQuitting
}
