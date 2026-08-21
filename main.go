//go:build windows

package main

import (
	"embed"
	"log"
	"log/slog"

	"github.com/khiemnguyen/remotune/internal/lifecycle"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

var version = "dev"

//go:embed all:frontend/dist
var frontendAssets embed.FS

//go:embed assets/app/remotune-256.png
var appIcon []byte

//go:embed assets/tray/remotune-32.png
var trayIcon []byte

func main() {
	if err := lifecycle.CheckWebView2(); err != nil {
		log.Fatalf("Remotune requires the WebView2 runtime: %v", err)
	}
	svc := lifecycle.NewService()

	app := application.New(application.Options{
		Name:        "Remotune",
		Description: "Automatically tunes Windows for remote desktop sessions",
		Icon:        appIcon,
		LogLevel:    slog.LevelInfo,
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(frontendAssets),
		},
		Services: []application.Service{
			application.NewService(svc),
		},
		Windows: application.WindowsOptions{
			DisableQuitOnLastWindowClosed: true,
		},
	})
	mainWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:          "main",
		Title:         "Remotune",
		Width:         420,
		Height:        560,
		MinWidth:      380,
		MaxWidth:      450,
		MinHeight:     520,
		Hidden:        true,
		DisableResize: false,
	})
	mainWindow.OnWindowEvent(events.Common.WindowClosing, func(event *application.WindowEvent) {
		event.Cancel()
		mainWindow.Hide()
	})
	tray := app.SystemTray.New()
	tray.SetIcon(trayIcon)
	menu := buildTrayMenu(app, mainWindow, svc)
	tray.SetMenu(menu)
	tray.SetTooltip("Remotune v" + version)
	tray.OnClick(func() {
		showAtWorkAreaBottomRight(app, mainWindow)
	})
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}

}

func showAtWorkAreaBottomRight(app *application.App, window *application.WebviewWindow) {
	screen := app.Screen.GetPrimary()
	if screen != nil {
		width, height := window.Size()
		if width == 0 || height == 0 {
			width, height = 420, 560
		}
		const margin = 12
		window.SetPosition(screen.WorkArea.X+screen.WorkArea.Width-width-margin, screen.WorkArea.Y+screen.WorkArea.Height-height-margin)
	}
	window.Show()
	window.Focus()
}
