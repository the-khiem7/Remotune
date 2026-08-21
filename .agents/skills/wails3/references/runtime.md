# Wails v3 — Frontend Runtime API (`@wailsio/runtime`)

## Table of Contents
1. [Installation & Setup](#installation--setup)
2. [Events](#events)
3. [Window](#window)
4. [Dialogs](#dialogs)
5. [Clipboard](#clipboard)
6. [Screens](#screens)
7. [Application](#application)
8. [Browser](#browser)
9. [System](#system)
10. [WML (Wails Markup Language)](#wml-wails-markup-language)
11. [TypeScript Support](#typescript-support)
12. [Best Practices](#best-practices)

---

## Installation & Setup

```bash
npm install --save @wailsio/runtime
```

**Import API modules:**
```javascript
import { Events, Window, Dialogs, Clipboard, Screens, Application, Browser } from '@wailsio/runtime'
```

**Required side-effect import** (enables context menus + window dragging):
```javascript
import '@wailsio/runtime'
```

**Without npm (pre-built bundle):**
```bash
wails3 generate runtime   # outputs runtime.js and runtime.debug.js
```
```html
<script type="module" src="./runtime.js"></script>
<script>
    window.onload = function() { wails.Window.SetTitle("Hello") }
</script>
```
> ⚠️ Use `type="module"` and wait for `onload` — ES modules run asynchronously.

---

## Events

Communication between Go backend and JavaScript frontend.

### `Events.On(eventName, callback)` → unsubscribe function

```typescript
Events.On(eventName: string, callback: (event: WailsEvent) => void): () => void
Events.On<T>(eventType: EventType<T>, callback: (event: WailsEvent<T>) => void): () => void
```

```javascript
import { Events } from '@wailsio/runtime'

const unsub = Events.On('data-updated', (event) => {
    console.log('Data:', event.data)
})

// Stop listening
unsub()
```

**With typed events (TypeScript):**
```typescript
import { UserLogin } from './bindings/events'

Events.On(UserLogin, (event) => {
    console.log(event.data.username)  // fully typed
})
```

---

### `Events.Once(eventName, callback)` → unsubscribe function

Fires callback exactly once, then auto-unsubscribes.

```javascript
Events.Once('app-ready', () => {
    console.log('App initialized')
})
```

---

### `Events.Emit(name, data?)` → `Promise<boolean>`

Emits an event. Returns `true` if cancelled by a hook, `false` otherwise.

```javascript
await Events.Emit('button-clicked', { id: 'submit' })
```

**Typed emit (TypeScript):**
```typescript
import { UserLogin } from './bindings/events'

const cancelled = await Events.Emit(UserLogin({
    UserID: "123",
    Username: "john_doe",
    LoginTime: new Date().toISOString()
}))
```

---

### `Events.Off(...eventNames)` / `Events.OffAll()`

```javascript
Events.Off('user-logged-in', 'user-logged-out')
Events.OffAll()
```

---

### Typed Events (Go side)

Register typed events in Go so bindings generator can emit `.d.ts` types:
```go
type UserData struct {
    ID   string
    Name string
}

func init() {
    application.RegisterEvent[UserData]("user-updated")
}
```

Then run:
```bash
wails3 generate bindings -ts
```

---

## Window

Control the current window (or a named window).

```javascript
import { Window } from '@wailsio/runtime'

// Current window
await Window.SetTitle('New Title')

// Named window
const other = Window.Get('settings')
await other.Show()
```

### Visibility

| Method | Signature | Description |
|---|---|---|
| `Show()` | `(): Promise<void>` | Show window |
| `Hide()` | `(): Promise<void>` | Hide window |
| `Close()` | `(): Promise<void>` | Close window |

### Size & Position

| Method | Signature |
|---|---|
| `SetSize(w, h)` | `(width: number, height: number): Promise<void>` |
| `Size()` | `(): Promise<{ width: number, height: number }>` |
| `SetPosition(x, y)` | `(x: number, y: number): Promise<void>` |
| `Position()` | `(): Promise<{ x: number, y: number }>` |
| `Center()` | `(): Promise<void>` |

```javascript
await Window.SetSize(1024, 768)
await Window.Center()
const { width, height } = await Window.Size()
```

### Window State

| Method | Signature |
|---|---|
| `Minimise()` | `(): Promise<void>` |
| `Maximise()` | `(): Promise<void>` |
| `Fullscreen()` | `(): Promise<void>` |
| `Restore()` | `(): Promise<void>` |
| `IsMinimised()` | `(): Promise<boolean>` |
| `IsMaximised()` | `(): Promise<boolean>` |
| `IsFullscreen()` | `(): Promise<boolean>` |

### Properties

| Method | Signature |
|---|---|
| `SetTitle(title)` | `(title: string): Promise<void>` |
| `Name()` | `(): Promise<string>` |
| `SetBackgroundColour(r,g,b,a)` | `(r,g,b,a: number): Promise<void>` |
| `SetAlwaysOnTop(b)` | `(b: boolean): Promise<void>` |
| `SetResizable(b)` | `(b: boolean): Promise<void>` |

### Focus & Screen

| Method | Signature |
|---|---|
| `Focus()` | `(): Promise<void>` |
| `IsFocused()` | `(): Promise<boolean>` |
| `GetScreen()` | `(): Promise<Screen>` |

### Content

| Method | Description |
|---|---|
| `Reload()` | Reload the page |
| `ForceReload()` | Force reload (clear cache) |

### Zoom

| Method | Signature |
|---|---|
| `SetZoom(level)` | `(level: number): Promise<void>` |
| `GetZoom()` | `(): Promise<number>` |
| `ZoomIn()` | `(): Promise<void>` |
| `ZoomOut()` | `(): Promise<void>` |
| `ZoomReset()` | `(): Promise<void>` Reset to 100% |

### Print

```javascript
await Window.Print()   // opens native OS print dialog
```

---

## Dialogs

Native OS dialogs from JavaScript.

```javascript
import { Dialogs } from '@wailsio/runtime'
```

### `Dialogs.Info(options)` → `Promise<string>`

```javascript
await Dialogs.Info({ Title: 'Success', Message: 'Operation completed!' })
```

### `Dialogs.Error(options)` / `Dialogs.Warning(options)`

```javascript
await Dialogs.Error({ Title: 'Error', Message: 'Something went wrong.' })
await Dialogs.Warning({ Title: 'Warning', Message: 'Low disk space.' })
```

### `Dialogs.Question(options)` → `Promise<string>`

Returns the label of the button clicked.

```javascript
const result = await Dialogs.Question({
    Title: 'Confirm Delete',
    Message: 'Are you sure?',
    Buttons: [
        { Label: 'Delete', IsDefault: false },
        { Label: 'Cancel', IsDefault: true }
    ]
})
if (result === 'Delete') { /* proceed */ }
```

### `Dialogs.OpenFile(options)` → `Promise<string | string[]>`

```javascript
const file = await Dialogs.OpenFile({
    Title: 'Select Image',
    Filters: [
        { DisplayName: 'Images', Pattern: '*.png;*.jpg;*.jpeg' },
        { DisplayName: 'All Files', Pattern: '*.*' }
    ]
})
if (file) console.log('Selected:', file)
```

### `Dialogs.SaveFile(options)` → `Promise<string>`

```javascript
const path = await Dialogs.SaveFile({
    Title: 'Save As',
    DefaultFilename: 'document.pdf',
    Filters: [{ DisplayName: 'PDF', Pattern: '*.pdf' }]
})
```

---

## Clipboard

```javascript
import { Clipboard } from '@wailsio/runtime'

// Write
await Clipboard.SetText('Hello from Wails!')

// Read
const text = await Clipboard.Text()
```

---

## Screens

```javascript
import { Screens } from '@wailsio/runtime'

const screens = await Screens.GetAll()
const primary = await Screens.GetPrimary()
const current = await Screens.GetCurrent()

screens.forEach(s => console.log(`${s.Name}: ${s.Size.Width}x${s.Size.Height}`))
```

**Screen interface:**
```typescript
interface Screen {
    ID: string
    Name: string
    ScaleFactor: number
    X: number
    Y: number
    Size: { Width: number, Height: number }
    Bounds: { X: number, Y: number, Width: number, Height: number }
    WorkArea: { X: number, Y: number, Width: number, Height: number }
    IsPrimary: boolean
    Rotation: number
}
```

---

## Application

```javascript
import { Application } from '@wailsio/runtime'

await Application.Show()   // show all windows
await Application.Hide()   // hide all windows
await Application.Quit()   // quit the app
```

```javascript
document.getElementById('quit-btn').addEventListener('click', () => Application.Quit())
```

---

## Browser

```javascript
import { Browser } from '@wailsio/runtime'

await Browser.OpenURL('https://wails.io')
await Browser.OpenURL(new URL('https://wails.io'))
```

---

## System

Low-level raw message to the Go backend (`RawMessageHandler`).
Bypass the standard binding system — use only if bindings are a confirmed bottleneck.

```javascript
import { System } from '@wailsio/runtime'

System.invoke('my-custom-message')
System.invoke(JSON.stringify({ action: 'update', value: 42 }))
```

> Fire-and-forget. Use `Events.On` to receive the response from Go.

---

## WML (Wails Markup Language)

Declarative HTML attributes — no JavaScript required.

| Attribute | Action |
|---|---|
| `wml-event="event-name"` | Emits event on click |
| `wml-window="MethodName"` | Calls window method on click |
| `wml-target-window="name"` | Target window for `wml-window` |
| `wml-openurl="https://..."` | Opens URL in system browser |
| `wml-confirm="message"` | Confirmation dialog before action |

```html
<button wml-event="save-clicked">Save</button>
<button wml-window="Minimise">Minimize</button>
<button wml-window="Fullscreen">Fullscreen</button>
<button wml-window="Close" wml-confirm="Close window?">Close</button>
<button wml-window="Show" wml-target-window="settings">Open Settings</button>
<a href="#" wml-openurl="https://wails.io">Visit Wails</a>
```

---

## Vite Plugin for Typed Events (HMR)

```typescript
// vite.config.ts
import { defineConfig } from 'vite'
import wails from '@wailsio/runtime/plugins/vite'

export default defineConfig({
    plugins: [wails()],
})
```

Benefits: event bindings auto-reload on `wails3 generate bindings`.

---

## TypeScript Support

The runtime ships with full TypeScript definitions.

```typescript
import { Events, Window } from '@wailsio/runtime'

Events.On('custom-event', (event) => {
    console.log(event.data)    // typed as WailsEvent
    console.log(event.name)
    console.log(event.sender)
})

const size: { width: number, height: number } = await Window.Size()
```

---

## Best Practices

### ✅ Do
- Import only the modules you use
- `await` all async calls — most methods are async
- Use WML for simple click actions (cleaner than inline JS)
- Unsubscribe events when components unmount (`const unsub = Events.On(...)`)
- Handle dialog return values

### ❌ Don't
- Don't forget `await` — silent no-ops without it
- Don't block the UI thread with long sync work
- Don't ignore promise rejections
