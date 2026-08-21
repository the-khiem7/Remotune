//go:build windows

package main

import (
	"fmt"
	"log/slog"

	"github.com/khiemnguyen/remotune/internal/lifecycle"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func buildTrayMenu(app *application.App, mainWindow *application.WebviewWindow, svc *lifecycle.Service) *application.Menu {
	menu := app.NewMenu()
	statusItem := menu.Add("Remotune: Starting...")
	statusItem.SetEnabled(false)

	menu.AddSeparator()
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
	menu.Add("Restore Now").OnClick(func(ctx *application.Context) {
		if err := svc.RestoreNow(); err != nil {
			slog.Error("restore now failed", "error", err)
		}
		refreshTrayMenu(statusItem, pauseItem, svc)
	})

	menu.AddSeparator()
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

	menu.Add("Open").OnClick(func(ctx *application.Context) {
		mainWindow.Show()
		mainWindow.Focus()
	})

	menu.AddSeparator()
	menu.Add("Quit").OnClick(func(ctx *application.Context) {
		if err := lifecycle.Shutdown(svc); err != nil {
			statusItem.SetLabel("Quit blocked: restore failed")
			slog.Error("quit blocked because restore failed", "error", err)
			return
		}
		app.Quit()
	})
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(event *application.ApplicationEvent) {
		refreshTrayMenu(statusItem, pauseItem, svc)
	})

	return menu
}
func refreshTrayMenu(statusItem *application.MenuItem, pauseItem *application.MenuItem, svc *lifecycle.Service) {
	status := svc.Status()
	line := fmt.Sprintf("CRD: %s | %s", status.CRD, status.Tuning)
	if status.Detector.Health == "Degraded" {
		line += " | Detector degraded"
	}
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
