# Wails v3 CLI Reference

All commands: `wails3 <command>`

## Table of Contents
1. [Core Commands](#core-commands) — `init`, `dev`, `build`, `package`, `task`, `doctor`
2. [Generate Commands](#generate-commands) — `generate bindings`, `generate icons`, etc.
3. [Service Commands](#service-commands) — `service init`
4. [Update Commands](#update-commands) — `update cli`, `update build-assets`
5. [Tool Commands](#tool-commands) — `tool version`, `tool package`, etc.
6. [Utility Commands](#utility-commands) — `docs`, `version`, `sponsor`

---

## Core Commands

### `wails3 init`
Create a new Wails project.

```bash
wails3 init [flags]
```

| Flag | Description | Default |
|---|---|---|
| `-n` | Project name | *(required)* |
| `-t` | Template name or URL | `vanilla` |
| `-d` | Output directory | `.` |
| `-p` | Package name | `main` |
| `-git` | Git repo URL (also sets module name) | |
| `-productname` | Product name | `My Product` |
| `-productversion` | Product version | `0.1.0` |
| `-productdescription` | Description | |
| `-productcompany` | Company | `My Company` |
| `-productcopyright` | Copyright | |
| `-productidentifier` | Bundle ID | |
| `-q` | Suppress output | `false` |
| `-l` | List available templates | `false` |
| `-skipgomodtidy` | Skip `go mod tidy` | `false` |

`-git` accepts: `https://`, `git@`, `ssh://git@`, `git://`, `file:///`

```bash
wails3 init -n myapp
wails3 init -n myapp -t svelte -git github.com/user/myapp
```

---

### `wails3 dev`
Run in development mode with live reload.

```bash
wails3 dev [flags]
```

| Flag | Description | Default |
|---|---|---|
| `-config` | Build config file | `./build/config.yml` |
| `-port` | Vite dev server port | `9245` |
| `-s` | Enable HTTPS | `false` |

- Frontend changes reload instantly
- Go code changes trigger rebuild + relaunch
- Bindings are auto-regenerated on Go changes

---

### `wails3 build`
Build a debug binary for the current platform.

```bash
wails3 build [CLI variables...]

wails3 build PLATFORM=linux CONFIG=production
```

Equivalent to `wails3 task build`. Variables are forwarded to Taskfile tasks.

---

### `wails3 package`
Create a platform-specific distribution package.

```bash
wails3 package [CLI variables...]
```

| Platform | Output |
|---|---|
| Windows | NSIS `.exe` installer |
| macOS | `.app` bundle |
| Linux | `.AppImage`, `.deb`, `.rpm`, `.archlinux` |

Linux specific:
```bash
wails3 task linux:create:appimage
wails3 task linux:create:deb
wails3 task linux:create:rpm
wails3 task linux:create:aur
```

---

### `wails3 task`
Run tasks defined in `Taskfile.yml`.

```bash
wails3 task [taskname] [KEY=VALUE...] [flags]
```

| Flag | Description |
|---|---|
| `-list` | List tasks with descriptions |
| `-list-all` | List all tasks |
| `-w` | Watch mode |
| `-p` | Run tasks in parallel |
| `-v` | Verbose output |
| `-dry` | Print tasks without executing |
| `-summary` | Show task summary |
| `-f` | Force run even if up-to-date |

```bash
wails3 task              # run default task
wails3 task build PLATFORM=windows ARCH=amd64
wails3 task --list
wails3 task -p task1 task2 task3
```

---

### `wails3 doctor`
Check system dependencies and report issues.

```bash
wails3 doctor
```

---

## Generate Commands

Base: `wails3 generate <command>`

### `generate bindings`
Auto-generate JavaScript/TypeScript bindings from Go services.

```bash
wails3 generate bindings [flags] [patterns...]
```

| Flag | Description | Default |
|---|---|---|
| `-ts` | Generate TypeScript | `false` |
| `-d` | Output directory | `frontend/bindings` |
| `-i` | Use TS interfaces (not classes) | `false` |
| `-models` | Models filename | `models` |
| `-index` | Index filename | `index` |
| `-b` | Use bundled runtime | `false` |
| `-names` | Use method names (not IDs) | `false` |
| `-noindex` | Skip index files | `false` |
| `-clean` | Clean output dir first | `false` |
| `-dry` | Dry run | `false` |
| `-silent` | Silent output | `false` |
| `-v` | Debug output | `false` |
| `-f` | Extra Go build flags | |

```bash
wails3 generate bindings            # JS output
wails3 generate bindings -ts        # TS output
wails3 generate bindings -ts -d ./src/api -clean
```

---

### `generate icons`
Generate app icons from a PNG source.

```bash
wails3 generate icons [flags]
```

| Flag | Description | Default |
|---|---|---|
| `-input` | Source PNG | *(required)* |
| `-sizes` | Icon sizes | `256,128,64,48,32,16` |
| `-windowsfilename` | Windows output name | |
| `-macfilename` | macOS output name | |
| `-example` | Generate example icon | `false` |

---

### `generate build-assets`
Generate build assets (icons, manifests, etc.).

```bash
wails3 generate build-assets [flags]
```

| Flag | Description |
|---|---|
| `-name` | Project name |
| `-dir` | Output dir (default: `build`) |
| `-company`, `-productname`, `-description`, `-version`, `-identifier`, `-copyright`, `-comments` | Metadata |
| `-silent` | Suppress output |

---

### `generate syso`
Generate Windows `.syso` resource file.

```bash
wails3 generate syso -manifest <file> -icon <file> [-info <file>] [-arch <arch>] [-out <file>]
```

---

### `generate .desktop`
Generate Linux `.desktop` file.

```bash
wails3 generate .desktop [flags]
```

| Flag | Description | Default |
|---|---|---|
| `-name` | App name | *(required)* |
| `-exec` | Executable path | *(required)* |
| `-icon` | Icon path | |
| `-categories` | Categories | `Utility` |
| `-terminal` | Run in terminal | `false` |
| `-output` | Output filename | `[name].desktop` |

---

### `generate runtime`
Generate a pre-built `runtime.js` bundle (for projects not using npm).

```bash
wails3 generate runtime
```

---

### `generate constants`
Generate JavaScript constants from Go constants.

```bash
wails3 generate constants
```

---

### `generate appimage`
Generate a Linux AppImage.

```bash
wails3 generate appimage -binary <path> -icon <path> -desktop <path> [-output <dir>]
```

---

## Service Commands

### `service init`
Scaffold a new Wails service package.

```bash
wails3 service init [flags]
```

| Flag | Description | Default |
|---|---|---|
| `-n` | Service name | `example_service` |
| `-d` | Description | `Example service` |
| `-p` | Package name | |
| `-o` | Output directory | `.` |
| `-a` | Author | |
| `-v` | Version | |
| `-r` | Repository URL | |
| `-l` | License | |

---

## Update Commands

### `update cli`
Update the Wails CLI.

```bash
wails3 update cli [flags]
```

| Flag | Description |
|---|---|
| `-pre` | Update to latest pre-release |
| `-version` | Update to a specific version |

After updating CLI, also update `go.mod`:
```bash
go get github.com/wailsapp/wails/v3@latest
```

---

### `update build-assets`
Update project build assets from config.

```bash
wails3 update build-assets -config <path> [flags]
```

---

## Tool Commands

### `tool version`
Bump a semantic version string.

```bash
wails3 tool version -v <version> [-major|-minor|-patch|-prerelease]

wails3 tool version -v 1.2.3 -minor    # → 1.3.0
wails3 tool version -v v3.0.0-alpha.5 -prerelease  # → v3.0.0-alpha.6
```

---

### `tool package`
Create Linux packages (deb, rpm, archlinux).

```bash
wails3 tool package -name <binary> -config <path> [-format deb|rpm|archlinux] [-out <dir>]
```

---

### `tool watcher`
Watch files and run a command when they change.

```bash
wails3 tool watcher [-config <path>] [-ignore <patterns>] [-include <patterns>]
```

---

### `tool checkport`
Check if a port is open (useful to verify Vite is running).

```bash
wails3 tool checkport [-port 9245] [-host localhost]
```

---

### `tool buildinfo`
Show build information embedded in the binary.

```bash
wails3 tool buildinfo
```

---

## Utility Commands

```bash
wails3 version          # print CLI version
wails3 docs             # open docs in browser
wails3 releasenotes     # show release notes
wails3 sponsor          # open sponsor page
```
