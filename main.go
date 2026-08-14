//go:build windows

package main

import (
	"log"
	"log/slog"

	"github.com/khiemnguyen/remotune/internal/lifecycle"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func main() {
	// Detect WebView2 before starting Wails — fail clearly if absent.
	if err := lifecycle.CheckWebView2(); err != nil {
		log.Fatalf("Remotune requires the WebView2 runtime: %v", err)
	}

	app := application.New(application.Options{
		Name:        "Remotune",
		Description: "Automatically tunes Windows for remote desktop sessions",
		LogLevel:    slog.LevelInfo,
		Assets:      application.AlphaAssets,
		Windows: application.WindowsOptions{
			DisableQuitOnLastWindowClosed: true,
		},
	})

	// Build the Remotune lifecycle service that owns the coordinator.
	svc := lifecycle.NewService()

	// System tray — the primary user surface.
	tray := app.SystemTray.New()
	menu := buildTrayMenu(app, svc)
	tray.SetMenu(menu)
	tray.SetTooltip("Remotune")

	// Start the coordinator loop in background once the app is running.
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(event *application.ApplicationEvent) {
		go svc.Run(app.Context())
	})

	// Run blocks until the application exits.
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}

	// After app.Run returns, perform the explicit Quit sequence:
	// stop transitions, restore owned state, clean up.
	svc.Shutdown()
}
