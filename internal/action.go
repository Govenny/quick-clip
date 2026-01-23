package internal

import (
	"github.com/go-vgo/robotgo"
	"github.com/tailscale/win"
)

func GetMousePosition() (x int, y int) {
	x, y = robotgo.Location()
	return
}

func RecordActiveWindow() (title win.HWND) {
	title = robotgo.GetHWND()
	return
}

func RestoreFocus(title win.HWND) {
	robotgo.SetFocus(title)
}
