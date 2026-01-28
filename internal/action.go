package internal

import (
	"github.com/go-vgo/robotgo"
	"github.com/tailscale/win"
)

type Action struct {
}

func NewAction() *Action {
	return &Action{}
}

func (a *Action) GetMousePosition() (x int, y int) {
	x, y = robotgo.Location()
	return
}

func (a *Action) RecordActiveWindow() (title win.HWND) {
	title = robotgo.GetHWND()
	return
}

func (a *Action) RestoreFocus(title win.HWND) {
	robotgo.SetFocus(title)
}
