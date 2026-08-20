# Docker build environment for Remotune

## Purpose and current status

Remotune is a Windows tray utility built with Wails v3, Go, and Vue. This runbook makes its build environment a Docker-only boundary: the Windows host is an editor and container orchestrator, not a place to install project toolchains or SDKs.

The Docker image includes the Wails v3 CLI pinned to `v3.0.0-beta.8`. Both `verify` and `build` generate TypeScript bindings under `frontend/bindings/`, install frontend dependencies with Bun, and build `frontend/dist/` before compiling Go. Bindings and generated production assets are intentionally ignored by Git; they are recreated in Docker on every verification or package build.

The Docker environment was established on 2026-08-14. Dockerfile, docker-compose.yml, delegate scripts, and `.dockerignore` are committed. Docker verification passed: `verify` (gofmt, go vet, compile check, platform-independent tests) and `build` (produces `out/Remotune.exe`, 11.9 MB, CGO_ENABLED=0 cross-compile from Linux). The `shell` service provides interactive diagnostics.

## Mandatory build-environment policy

**All Remotune dependency installation, code generation, verification, tests, builds, and packaging must execute inside Docker.** Do not install or invoke project build toolchains or SDKs directly on the Windows host.

The host may contain and use the editor, Git, Rancher Desktop / Docker, and `docker compose` as the orchestrator.

Do not install or invoke these tools for Remotune directly on Windows:

- Go: `go`, Go SDK, module cache, build cache, or code generators.
- Wails: the Wails 3 CLI, its generated bindings, or desktop build tooling.
- Vue frontend build tooling: Bun, Vite, Vue type checking, bundlers, and their caches.
- Windows build SDKs, compilers, linkers, or packaging dependencies added solely to build this project.

## Docker image toolchain

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.26.1 (golang:1.26.1-bookworm) | Backend compilation and tests |
| Bun | 1.2.17 | Frontend package manager and bundler (Phase 5) |
| CGO | disabled (`CGO_ENABLED=0`) | Wails v3 uses go-winloader (pure Go WebView2 bindings) — no mingw-w64 needed |
| Wails CLI | deferred to Phase 5 | Requires GTK3/WebKit2GTK dev libs; not needed until Vue frontend bindings exist |

## Required Docker contract

| Service | Responsibility |
|---------|---------------|
| `verify` | gofmt check, `go vet` (GOOS=windows), compile check (`go build`, `go test -c`), run platform-independent tests |
| `build` | Produce `out/Remotune.exe` via `GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w"` |
| `shell` | Interactive bash with the full pinned toolchain for diagnostics |

Caches are kept in Docker named volumes:

- `remotune_go-mod` — Go module cache
- `remotune_go-build` — Go build cache

Source is bind-mounted read/write from the host checkout.

## Local workflow

```powershell
# Verify: format, lint, static analysis, compile check, tests
.\scripts\verify.ps1

# Build: produce out/Remotune.exe
.\scripts\build-windows.ps1

# Shell: interactive diagnostics inside the container
.\scripts\shell.ps1
```

Each script delegates exclusively to `docker compose run`. None falls back to host `go`, `wails`, or `bun`.

## Platform-independent vs Windows-only tests

Most Remotune tests have `//go:build windows` because they use Win32 APIs. In the Docker verify pipeline:

1. **Compile check** (`go test -c` with GOOS=windows) confirms all test code compiles.
2. **Execute** (`go test` with GOOS=linux) runs only platform-independent tests (`state_test.go`, `reconstruct_test.go` once decoupled from Windows-only types).
3. **Full execution** of Windows-only tests happens on the actual Windows machine with `REMOTUNE_SYSTEM_TESTS=1`.

## Capacity and cleanup

This project shares Rancher Desktop / WSL storage with other Docker projects. Check capacity before large operations:

```powershell
Get-PSDrive C
docker system df
```

Non-destructive cleanup (removes unused images, stopped containers, build cache):

```powershell
docker system prune -af
```

To reclaim named volume space (destroys cached Go modules and build artifacts — they rebuild on next run):

```powershell
docker volume rm remotune_go-mod remotune_go-build
```

## Phase 5 additions (planned)

When the Vue frontend is added:

1. Install GTK3/WebKit2GTK dev libs in the Dockerfile (`libwebkit2gtk-4.1-dev`).
2. Install Wails v3 CLI (`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.8`).
3. Add `wails3 generate bindings` to the verify pipeline.
4. Add Bun frontend install and build to the build pipeline.
5. Add `bun-cache` named volume.

## Exception process

No standing exception for a host toolchain. A temporary host install requires explicit operator approval, a reason Docker cannot perform the work, a cleanup plan, and confirmation after removal.
