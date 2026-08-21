//go:build windows

package main

import (
	"embed"
	"fmt"
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
	var openCustomEffectsEditor func() error
	svc := lifecycle.NewService(func() error {
		if openCustomEffectsEditor == nil {
			return fmt.Errorf("Custom Visual Effects editor is not initialized")
		}
		return openCustomEffectsEditor()
	})

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
	customEffectsWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:          "custom-effects",
		Title:         "Remotune Custom Visual Effects",
		URL:           "http://wails.localhost/?view=custom",
		Width:         430,
		Height:        620,
		MinWidth:      400,
		MinHeight:     520,
		Hidden:        true,
		DisableResize: false,
	})
	customEffectsWindow.OnWindowEvent(events.Common.WindowClosing, func(event *application.WindowEvent) {
		event.Cancel()
		customEffectsWindow.Hide()
	})
	openCustomEffectsEditor = func() error {
		showCustomEffectsEditor(mainWindow, customEffectsWindow)
		return nil
	}
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

func showCustomEffectsEditor(mainWindow, editorWindow *application.WebviewWindow) {
	screen, err := mainWindow.GetScreen()
	if err != nil || screen == nil {
		screen = nil
	}
	mainX, mainY := mainWindow.Position()
	mainWidth, _ := mainWindow.Size()
	editorWidth, editorHeight := editorWindow.Size()
	if editorWidth == 0 || editorHeight == 0 {
		editorWidth, editorHeight = 430, 620
	}
	if screen != nil {
		const gap = 12
		x := mainX - editorWidth - gap
		if x < screen.WorkArea.X {
			x = mainX + mainWidth + gap
		}
		maxX := screen.WorkArea.X + screen.WorkArea.Width - editorWidth
		if x > maxX {
			x = maxX
		}
		if x < screen.WorkArea.X {
			x = screen.WorkArea.X
		}
		y := mainY
		maxY := screen.WorkArea.Y + screen.WorkArea.Height - editorHeight
		if y > maxY {
			y = maxY
		}
		if y < screen.WorkArea.Y {
			y = screen.WorkArea.Y
		}
		editorWindow.SetPosition(x, y)
	}
	editorWindow.Show()
	editorWindow.Focus()
}
