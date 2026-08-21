# Wails v3 — Services & Bindings Reference

## Table of Contents
1. [Service Architecture](#service-architecture)
2. [Creating Services](#creating-services)
3. [Service Lifecycle](#service-lifecycle)
4. [Generating Bindings](#generating-bindings)
5. [Using Bindings (JS/TS)](#using-bindings)
6. [Type Mapping](#type-mapping)
7. [Error Handling](#error-handling)
8. [Dependency Injection](#dependency-injection)
9. [Best Practices](#best-practices)

---

## Service Architecture

A **service** is a plain Go struct whose exported methods are automatically bound to JavaScript functions. Services are:

- Lifecycle-aware (`ServiceStartup` / `ServiceShutdown`)  
- Singletons (one instance per application)  
- Thread-shared — use mutexes for any mutable state  
- Independently testable with no Wails dependency

---

## Creating Services

### Basic Service

```go
type CalculatorService struct{}

func (c *CalculatorService) Add(a, b int) int        { return a + b }
func (c *CalculatorService) Subtract(a, b int) int   { return a - b }
func (c *CalculatorService) Divide(a, b float64) (float64, error) {
    if b == 0 { return 0, errors.New("division by zero") }
    return a / b, nil
}
```

**Register:**
```go
app := application.New(application.Options{
    Services: []application.Service{
        application.NewService(&CalculatorService{}),
    },
})
```

**Key rules:**
- Only **exported methods** (PascalCase) are bound
- Return types: `value`, `error`, or `(value, error)`
- Methods returning `error` propagate it as a JS exception

---

### Service with State

```go
type CounterService struct {
    count int
    mu    sync.Mutex
}

func (c *CounterService) Increment() int {
    c.mu.Lock(); defer c.mu.Unlock()
    c.count++; return c.count
}
func (c *CounterService) GetCount() int {
    c.mu.RLock(); defer c.mu.RUnlock()
    return c.count
}
func (c *CounterService) Reset() {
    c.mu.Lock(); defer c.mu.Unlock()
    c.count = 0
}
```

**Always use mutexes** — services are shared across all windows.

---

### Service with Dependencies

```go
type UserService struct {
    db     *sql.DB
    logger *slog.Logger
}

func NewUserService(db *sql.DB, logger *slog.Logger) *UserService {
    return &UserService{db: db, logger: logger}
}

func (u *UserService) GetUser(id int) (*User, error) {
    u.logger.Info("Getting user", "id", id)
    var user User
    err := u.db.QueryRow("SELECT * FROM users WHERE id = ?", id).Scan(&user)
    if err != nil { return nil, fmt.Errorf("failed to get user: %w", err) }
    return &user, nil
}
```

```go
db, _ := sql.Open("sqlite3", "app.db")
app := application.New(application.Options{
    Services: []application.Service{
        application.NewService(NewUserService(db, slog.Default())),
    },
})
```

---

### Custom Service Name

```go
application.NewServiceWithOptions(&MyService{}, application.ServiceOptions{
    Name: "CustomName",
})
```

Useful when registering multiple instances of the same type.

---

### HTTP Route on a Service

Services can handle HTTP requests directly:

```go
type FileService struct{ root string }

func (f *FileService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    http.FileServer(http.Dir(f.root)).ServeHTTP(w, r)
}

application.NewServiceWithOptions(&FileService{root: "./files"}, application.ServiceOptions{
    Route: "/files",
})
// Access at: http://wails.localhost/files/...
```

---

## Service Lifecycle

### `ServiceStartup(ctx, opts) error`

Called when the application starts. Services start in **registration order**.

```go
func (u *UserService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
    // Open resources, run migrations, start background goroutines
    db, err := sql.Open("sqlite3", "app.db")
    if err != nil { return fmt.Errorf("open db: %w", err) }
    u.db = db

    ctx, u.cancel = context.WithCancel(ctx)
    go u.backgroundSync(ctx)
    return nil
}
```

- Return a non-nil `error` to **abort application startup**
- Use the provided `ctx` for graceful goroutine cancellation

---

### `ServiceShutdown() error`

Called when the application is shutting down. Services shut down in **reverse registration order**.

```go
func (u *UserService) ServiceShutdown() error {
    if u.cancel != nil { u.cancel() }
    if u.db != nil { return u.db.Close() }
    return nil
}
```

- Returning an error logs a warning but **does not prevent shutdown**
- The application context is already cancelled at this point

---

## Generating Bindings

```bash
wails3 generate bindings              # JavaScript (default: frontend/bindings/)
wails3 generate bindings -ts          # TypeScript
wails3 generate bindings -d ./src/api # custom output
wails3 generate bindings -clean       # clean output dir first
```

Bindings are auto-regenerated during `wails3 dev`.

**Output structure:**
```
frontend/bindings/
└── myapp/
    ├── calculatorservice.js   ← one file per service
    ├── userservice.js
    └── index.js               ← re-exports all services
```

---

## Using Bindings

### JavaScript

```javascript
// Direct import
import { Add, Subtract, Divide } from './bindings/myapp/calculatorservice'

const sum  = await Add(5, 3)          // 8
const diff = await Subtract(10, 4)    // 6

// Error handling
try {
    const result = await Divide(10, 0)
} catch (error) {
    console.error("Error:", error)   // "division by zero"
}
```

**Namespace import via index:**
```javascript
import { CalculatorService } from './bindings/myapp'
const sum = await CalculatorService.Add(5, 3)
```

---

### TypeScript

```typescript
import { Add, Divide } from './bindings/myapp/calculatorservice'

const sum: number = await Add(5, 3)

try {
    const result = await Divide(10, 0)
} catch (error: unknown) {
    if (error instanceof Error) console.error(error.message)
}
```

Benefits: full type checking, IDE autocomplete, compile-time error detection.

---

### Generated File Example

```javascript
// frontend/bindings/myapp/calculatorservice.js

/**
 * @param {number} a
 * @param {number} b
 * @returns {Promise<number>}
 */
export function Add(a, b) {
    return window.wails.Call('CalculatorService.Add', a, b)
}
```

---

## Type Mapping

### Primitive Types

| Go | JavaScript/TypeScript |
|---|---|
| `string` | `string` |
| `bool` | `boolean` |
| `int`, `int8`…`int64` | `number` |
| `uint`, `uint8`…`uint64` | `number` |
| `float32`, `float64` | `number` |
| `byte`, `rune` | `number` |

### Complex Types

| Go | JavaScript/TypeScript |
|---|---|
| `[]T` | `T[]` |
| `[N]T` | `T[]` |
| `map[string]T` | `Record<string, T>` |
| `map[K]V` | `Map<K, V>` |
| `struct` | `class` (with exported fields) |
| `time.Time` | `Date` |
| `*T` | `T` (pointers are transparent) |
| `interface{}` | `any` |
| `error` | thrown `Error` exception |

### Unsupported Types ❌

`chan T`, `func()`, complex interfaces, unexported struct fields

**Workaround — use IDs:**
```go
var openFiles = make(map[string]*os.File)

func (s *FileService) OpenFile(path string) (string, error) {
    f, err := os.Open(path)
    if err != nil { return "", err }
    id := generateID()
    openFiles[id] = f
    return id, nil
}

func (s *FileService) ReadFile(id string) ([]byte, error) {
    return io.ReadAll(openFiles[id])
}

func (s *FileService) CloseFile(id string) error {
    f := openFiles[id]
    delete(openFiles, id)
    return f.Close()
}
```

---

## Error Handling

Go errors propagate automatically as JavaScript exceptions:

```go
// Go
func (d *DatabaseService) GetUser(id int) (*User, error) {
    if id <= 0 { return nil, errors.New("invalid user ID") }
    // ...
    return nil, fmt.Errorf("user %d not found", id)
}
```

```javascript
// JavaScript
try {
    const user = await GetUser(-1)
} catch (error) {
    console.error("Error:", error)  // "invalid user ID"
}
```

---

## Dependency Injection

### Application Injection

When your service needs to call back into the app (emit events, open dialogs):

```go
type NotificationService struct {
    app *application.Application
}

func NewNotificationService(app *application.Application) *NotificationService {
    return &NotificationService{app: app}
}

func (n *NotificationService) Notify(message string) {
    n.app.Event.Emit("notification", message)
}

// Register after app creation
app := application.New(application.Options{})
app.RegisterService(application.NewService(NewNotificationService(app)))
```

### Service-to-Service Dependencies

```go
userService  := &UserService{}
emailService := &EmailService{}
orderService := NewOrderService(userService, emailService)

app := application.New(application.Options{
    Services: []application.Service{
        application.NewService(userService),
        application.NewService(emailService),
        application.NewService(orderService),
    },
})
```

---

## Best Practices

### ✅ Do
- Single responsibility — one purpose per service
- Use constructor injection for dependencies
- Always use mutexes for shared state
- Return `error` rather than panicking
- Use `ServiceStartup` / `ServiceShutdown` for resource management

### ❌ Don't
- Don't use global state — pass dependencies explicitly
- Don't block `ServiceStartup` — start goroutines instead
- Don't skip `ServiceShutdown` cleanup — always release resources
- Don't create circular service dependencies
- Don't expose unexported methods — they won't be bound

### Performance Tips
```javascript
// ❌ Slow: N round trips
for (const item of items) { await ProcessItem(item) }

// ✅ Fast: 1 round trip
await ProcessItems(items)
```

Use events for long-running operations instead of blocking calls:
```go
func (s *MyService) ProcessLargeFile(path string) {
    go func() {
        for progress := range process(path) {
            s.app.Event.Emit("progress", progress)
        }
        s.app.Event.Emit("process-done", nil)
    }()
}
```
