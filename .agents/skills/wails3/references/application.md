# Wails v3 — Application API Reference (Go side)

## Table of Contents
1. [Creating an Application](#creating-an-application)
2. [Core Methods](#core-methods)
3. [Window Management](#window-management)
4. [Event Management](#event-management)
5. [Service Management](#service-management)
6. [Dialogs](#dialogs)
7. [Other Managers](#other-managers)
8. [Logger](#logger)
9. [Raw Message Handling](#raw-message-handling)
10. [Platform-Specific Options](#platform-specific-options)
11. [Complete Example](#complete-example)

---

## Creating an Application

```go
import "github.com/wailsapp/wails/v3/pkg/application"

app := application.New(application.Options{
    Name:        "My App",
    Description: "My awesome application",
    Services: []application.Service{
        application.NewService(&MyService{}),
    },
    Mac: application.MacOptions{
        ApplicationShouldTerminateAfterLastWindowClosed: true,
    },
})
```

---

## Core Methods

### `app.Run() error`

Start the application event loop. Blocks until the app quits.

```go
if err := app.Run(); err != nil {
    log.Fatal(err)
}
```

### `app.Quit()`

Gracefully shut down the application (all services get `ServiceShutdown` called).

```go
menu.Add("Quit").OnClick(func(ctx *application.Context) {
    app.Quit()
})
```

### `app.Config() Options`

Returns the current application options.

```go
config := app.Config()
fmt.Println("App name:", config.Name)
```

---

## Window Management

### `app.Window.New() *WebviewWindow`

Create a new webview window with defaults.

```go
window := app.Window.New()
window.Show()
```

### `app.Window.NewWithOptions(opts) *WebviewWindow`

Create a window with custom options.

```go
window := app.Window.NewWithOptions(application.WebviewWindowOptions{
    Title:            "Main Window",
    Width:            1024,
    Height:           768,
    MinWidth:         800,
    MinHeight:        600,
    BackgroundColour: application.NewRGB(255, 255, 255),
    URL:              "http://wails.localhost/",
    // HTML:          "<html>...</html>",  // inline HTML
})
window.Centre()
window.Show()
```

**Key WebviewWindowOptions fields:**

| Field | Type | Description |
|---|---|---|
| `Name` | `string` | Unique identifier for `GetByName` |
| `Title` | `string` | Window title bar text |
| `Width`, `Height` | `int` | Initial dimensions |
| `MinWidth`, `MinHeight` | `int` | Minimum resize limits |
| `MaxWidth`, `MaxHeight` | `int` | Maximum resize limits |
| `URL` | `string` | URL to load (default: `http://wails.localhost/`) |
| `HTML` | `string` | Inline HTML to load |
| `BackgroundColour` | `RGBA` | Window background color |
| `Frameless` | `bool` | Remove native window frame |
| `AlwaysOnTop` | `bool` | Keep window above others |
| `Hidden` | `bool` | Start hidden |
| `Resizable` | `bool` | Allow resize |
| `StartState` | `WindowState` | `Normal`, `Minimised`, `Maximised`, `Fullscreen` |

### `app.Window.GetByName(name) Window`

```go
win := app.Window.GetByName("settings")
if win != nil { win.Show() }
```

### `app.Window.GetAll() []Window`

```go
for _, w := range app.Window.GetAll() {
    fmt.Println(w.Name())
}
```

### Window Methods

```go
window.Show()
window.Hide()
window.Close()
window.Centre()
window.SetTitle("New Title")
window.SetSize(1280, 720)
window.Minimise()
window.Maximise()
window.Fullscreen()
window.Restore()
window.SetAlwaysOnTop(true)
window.Focus()
window.EmitEvent("my-event", data)  // emit event to this window's JS
```

---

## Event Management

Events enable pub/sub communication between Go services, windows, and the frontend.

### `app.Event.Emit(name, data...)`

Broadcast to all listeners (Go and JS).

```go
app.Event.Emit("user-logged-in", map[string]interface{}{
    "username":  "john",
    "timestamp": time.Now(),
})
```

### `app.Event.On(name, callback)`

Listen for custom events (Go side).

```go
app.Event.On("user-logged-in", func(e *application.CustomEvent) {
    data := e.Data.(map[string]interface{})
    fmt.Println("User logged in:", data["username"])
})
```

### `app.Event.OnApplicationEvent(eventType, callback) func()`

Listen for application lifecycle events. Returns unsubscribe function.

```go
// Available events: EventApplicationShutdown, EventApplicationStartup,
//   EventWindowFocus, EventWindowBlur, EventWindowClose, etc.
app.Event.OnApplicationEvent(application.EventApplicationShutdown, func(e *application.ApplicationEvent) {
    fmt.Println("App shutting down — doing cleanup")
})
```

---

## Service Management

### Register at startup (recommended)

```go
app := application.New(application.Options{
    Services: []application.Service{
        application.NewService(&GreetService{}),
    },
})
```

### `app.RegisterService(service) error`

Register after app creation (e.g., when service needs the app reference).

```go
app := application.New(application.Options{})
app.RegisterService(application.NewService(NewNotificationService(app)))
```

---

## Dialogs

### Message Dialogs (chained builder API)

```go
// Info
app.Dialog.Info().SetTitle("Done").SetMessage("Saved!").Show()

// Error
app.Dialog.Error().SetTitle("Error").SetMessage("Something failed.").Show()

// Warning
app.Dialog.Warning().SetTitle("Caution").SetMessage("Cannot undo this.").Show()
```

### Question Dialog

```go
dlg := app.Dialog.Question().
    SetTitle("Confirm").
    SetMessage("Delete this item?")

yes := dlg.AddButton("Yes")
yes.OnClick(func() { deleteItem() })

no := dlg.AddButton("No")
dlg.SetDefaultButton(yes)
dlg.SetCancelButton(no)
dlg.Show()
```

### File Dialogs

```go
// Open file
path, err := app.Dialog.OpenFile().
    SetTitle("Select File").
    AddFilter("Images", "*.png;*.jpg").
    PromptForSingleSelection()

// Save file
path, err := app.Dialog.SaveFile().
    SetTitle("Save As").
    SetFilename("document.pdf").
    AddFilter("PDF", "*.pdf").
    PromptForSingleSelection()

// Open folder
path, err := app.Dialog.OpenFile().
    SetTitle("Select Folder").
    CanChooseDirectories(true).
    CanChooseFiles(false).
    PromptForSingleSelection()
```

---

## Other Managers

```go
app.Window      // WebviewWindow management
app.Menu        // Application menu
app.Dialog      // Native dialogs
app.Event       // Event pub/sub
app.Clipboard   // Clipboard read/write
app.Screen      // Screen info
app.SystemTray  // System tray icon & menu
app.Browser     // Open URLs in system browser
app.Env         // Environment variables
```

**Examples:**
```go
// Clipboard
app.Clipboard.SetText("Copied!")
text, _ := app.Clipboard.Text()

// Screen
screens := app.Screen.GetAll()
primary := app.Screen.GetPrimary()

// Browser
app.Browser.OpenURL("https://wails.io")
```

---

## Logger

Structured logger available on the app instance.

```go
app.Logger.Info("Message", "key", "value")
app.Logger.Debug("Debug info", "detail", detail)
app.Logger.Warn("Warning", "threshold", threshold)
app.Logger.Error("Error occurred", "error", err)
```

---

## Raw Message Handling

For special cases requiring direct frontend→backend communication bypassing the binding system. Only use if bindings are a confirmed performance bottleneck.

```go
app := application.New(application.Options{
    Name: "My App",
    RawMessageHandler: func(window application.Window, message string) {
        fmt.Printf("Raw msg from %s: %s\n", window.Name(), message)
        // Respond via event
        window.EmitEvent("raw-response", processMessage(message))
    },
})
```

**Frontend (JS):**
```javascript
import { System } from '@wailsio/runtime'
System.invoke(JSON.stringify({ action: "ping" }))

Events.On('raw-response', (e) => console.log(e.data))
```

---

## Platform-Specific Options

### macOS

```go
Mac: application.MacOptions{
    ActivationPolicy: application.ActivationPolicyRegular,
    ApplicationShouldTerminateAfterLastWindowClosed: true,
}
```

### Windows

```go
Windows: application.WindowsOptions{
    WndClass:                      "MyAppClass",
    WebviewUserDataPath:           "",   // default: %APPDATA%\[BinaryName.exe]
    WebviewBrowserPath:            "",   // default: system WebView2
    DisableQuitOnLastWindowClosed: false,
    EnabledFeatures:               []string{"msWebView2EnableDraggableRegions"},
    AdditionalBrowserArgs:         []string{"--remote-debugging-port=9222"},
}
```

### Linux

```go
Linux: application.LinuxOptions{
    ProgramName:                   "my-app",
    DisableQuitOnLastWindowClosed: false,
}
```

---

## Complete Example

```go
package main

import (
    "fmt"
    "log/slog"
    "time"

    "github.com/wailsapp/wails/v3/pkg/application"
)

type GreetService struct {
    app *application.Application
}

func (g *GreetService) Greet(name string) string {
    return "Hello, " + name + "!"
}

func (g *GreetService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
    slog.Info("GreetService starting up")
    return nil
}

func (g *GreetService) ServiceShutdown() error {
    slog.Info("GreetService shutting down")
    return nil
}

func main() {
    app := application.New(application.Options{
        Name:        "My Application",
        Description: "A demo Wails app",
        Mac: application.MacOptions{
            ApplicationShouldTerminateAfterLastWindowClosed: true,
        },
    })

    greetSvc := &GreetService{app: app}
    app.RegisterService(application.NewService(greetSvc))

    // Subscribe to shutdown for cleanup
    app.Event.OnApplicationEvent(application.EventApplicationShutdown, func(e *application.ApplicationEvent) {
        fmt.Println("Goodbye!")
    })

    // Create main window
    window := app.Window.NewWithOptions(application.WebviewWindowOptions{
        Title:            "My App",
        Width:            1024,
        Height:           768,
        BackgroundColour: application.NewRGB(255, 255, 255),
    })
    window.Centre()
    window.Show()

    if err := app.Run(); err != nil {
        slog.Error("App error", "error", err)
    }
}
```
