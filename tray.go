//go:build windows

package main

import (
	"fmt"
	"log/slog"

	"github.com/khiemnguyen/remotune/internal/lifecycle"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// buildTrayMenu creates the system tray context menu that is Remotune's primary
// user surface until Phase 5 adds a Vue UI window.
func buildTrayMenu(app *application.App, svc *lifecycle.Service) *application.Menu {
	menu := app.NewMenu()

	// Status line (read-only, updated dynamically).
	statusItem := menu.Add("Remotune: Starting...")
	statusItem.SetEnabled(false)

	menu.AddSeparator()

	// Pause / Resume toggle.
	pauseItem := menu.Add("Pause Automation")
	pauseItem.OnClick(func(ctx *application.Context) {
		status := svc.Status()
		if status.Paused {
			if err := svc.Resume(); err != nil {
				slog.Error("resume failed", "error", err)
			}
		} else {
			if err := svc.Pause(); err != nil {
				slog.Error("pause failed", "error", err)
			}
		}
		refreshTrayMenu(statusItem, pauseItem, svc)
	})

	// Restore Now.
	menu.Add("Restore Now").OnClick(func(ctx *application.Context) {
		if err := svc.RestoreNow(); err != nil {
			slog.Error("restore now failed", "error", err)
		}
		refreshTrayMenu(statusItem, pauseItem, svc)
	})

	menu.AddSeparator()

	// Start with Windows.
	autostartItem := menu.Add("Start with Windows")
	autostartStatus, _ := lifecycle.GetAutostartStatus()
	autostartItem.SetChecked(autostartStatus.Registered && autostartStatus.PathMatch)
	autostartItem.OnClick(func(ctx *application.Context) {
		current, _ := lifecycle.GetAutostartStatus()
		enable := !(current.Registered && current.PathMatch)
		if err := lifecycle.SetAutostart(enable); err != nil {
			slog.Error("autostart toggle failed", "error", err)
			return
		}
		autostartItem.SetChecked(enable)
	})

	menu.AddSeparator()

	// Quit.
	menu.Add("Quit").OnClick(func(ctx *application.Context) {
		app.Quit()
	})

	// Periodic status refresh via a background goroutine (Phase 5 will use Wails
	// events for reactive updates; this is a lightweight interim solution).
	go func() {
		// The initial refresh happens once the app context is available.
		<-waitForContext(app)
		refreshTrayMenu(statusItem, pauseItem, svc)
	}()

	return menu
}

// refreshTrayMenu updates the status line and pause/resume label to reflect
// the current coordinator state.
func refreshTrayMenu(statusItem *application.MenuItem, pauseItem *application.MenuItem, svc *lifecycle.Service) {
	status := svc.Status()
	line := fmt.Sprintf("CRD: %s | %s", status.CRD, status.Tuning)
	if status.Paused {
		line += " (Paused)"
	}
	statusItem.SetLabel(line)

	if status.Paused {
		pauseItem.SetLabel("Resume Automation")
	} else {
		pauseItem.SetLabel("Pause Automation")
	}
}

// waitForContext returns a channel that closes once app.Context() is non-nil.
// This is a minimal helper until we can use ApplicationStarted event.
func waitForContext(app *application.App) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		for {
			if app.Context() != nil {
				close(ch)
				return
			}
		}
	}()
	return ch
}
