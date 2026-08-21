# Remotune native Windows development entrypoint.
# Runs Wails dev mode with Vite HMR and automatic Go rebuild/relaunch.

$ErrorActionPreference = 'Stop'
$wails = Join-Path (go env GOPATH) 'bin\wails3.exe'

if (-not (Test-Path $wails)) {
    Write-Error "Wails CLI v3.0.0-beta.8 is required at $wails. Install it with: go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.8"
    exit 1
}

$env:Path = "$(Split-Path $wails);$env:Path"
& $wails dev -config "$PSScriptRoot\..\build\config.yml"
exit $LASTEXITCODE
