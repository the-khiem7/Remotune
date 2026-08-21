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

// version is set at build time via -ldflags "-X main.version=..."
var version = "dev"

// frontendAssets is the production Vue build. The host-native build pipeline creates
// frontend/dist before compiling this package, so the shipped executable remains a
// single portable file.
//
//go:embed all:frontend/dist
var frontendAssets embed.FS

//go:embed assets/app/remotune-256.png
var appIcon []byte

//go:embed assets/tray/remotune-32.png
var trayIcon []byte

func main() {
	// Detect WebView2 before starting Wails — fail clearly if absent.
	if err := lifecycle.CheckWebView2(); err != nil {
		log.Fatalf("Remotune requires the WebView2 runtime: %v", err)
	}

	// The lifecycle service is the only backend surface exposed to Vue. It owns no
	// Win32 logic itself; every state change continues through the coordinator.
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

	// The compact control surface stays hidden until the user opens Remotune from
	// its tray icon. Closing it hides the window; the background service remains.
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

	// System tray — the primary user surface.
	tray := app.SystemTray.New()
	tray.SetIcon(trayIcon)
	menu := buildTrayMenu(app, mainWindow, svc)
	tray.SetMenu(menu)
	tray.SetTooltip("Remotune v" + version)

	// Left-click opens the compact control surface.
	tray.OnClick(func() {
		mainWindow.Show()
		mainWindow.Focus()
	})

	// Run blocks until the application exits.
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}

}
