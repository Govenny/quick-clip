package internal

import (
	"os"
	"path/filepath"

	"github.com/emersion/go-autostart"
)

type AppService struct {
}

func NewAppService() *AppService {
	return &AppService{}
}

// GetAppPath 获取当前运行的可执行文件绝对路径
func (a *AppService) getAppPath() string {
	ex, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Clean(ex)
}

// IsAutoStartCheck 检查是否已经设置了自启
func (a *AppService) IsAutoStartCheck() bool {
	app := &autostart.App{
		Name:        "quick-clip", // 替换为你的应用名
		DisplayName: "quick-clip",
		Exec:        []string{a.getAppPath()},
	}
	return app.IsEnabled()
}

// ToggleAutoStart 开启或关闭自启
func (a *AppService) ToggleAutoStart(enable bool) error {
	app := &autostart.App{
		Name:        "quick-clip",
		DisplayName: "quick-clip",
		Exec:        []string{a.getAppPath()},
	}

	if enable {
		if !app.IsEnabled() {
			return app.Enable()
		}
	} else {
		if app.IsEnabled() {
			return app.Disable()
		}
	}
	return nil
}
