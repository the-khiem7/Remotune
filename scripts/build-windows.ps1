# Remotune build delegate — produces out/remotune-v<version>.exe inside Docker.
# Usage: .\scripts\build-windows.ps1
# This script ONLY orchestrates Docker. No project SDK runs on the host.

$ErrorActionPreference = 'Stop'

# Read version from version.txt
$versionFile = Join-Path $PSScriptRoot "..\version.txt"
$version = (Get-Content $versionFile -Raw).Trim()
if (-not ($version -match '^\d+\.\d+\.\d+$')) {
    Write-Error "Invalid version in version.txt: '$version'"
    exit 1
}

$artifactName = "remotune-v${version}.exe"
$artifactPath = Join-Path $PSScriptRoot "..\out\$artifactName"

# Check if destination is locked (running exe).
if (Test-Path $artifactPath) {
    try {
        [IO.File]::OpenWrite($artifactPath).Close()
    } catch {
        Write-Error "Cannot overwrite $artifactName - the file is locked (app may be running). Close it and retry."
        exit 1
    }
}

# Generate bindings and production frontend assets on the host.
$wails = Join-Path (go env GOPATH) 'bin\wails3.exe'
if (-not (Test-Path $wails)) {
    Write-Error "Wails CLI v3.0.0-beta.8 is required at $wails"
    exit 1
}
& $wails generate bindings -ts
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Push-Location (Join-Path $PSScriptRoot '..\frontend')
try {
    npm install --package-lock=false
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    npm run build
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
} finally { Pop-Location }

New-Item -ItemType Directory -Path "$PSScriptRoot\..\out" -Force | Out-Null
& $wails generate syso -arch amd64 -icon build/windows/icon.ico -manifest build/windows/wails.exe.manifest -out wails_windows_amd64.syso
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
go build -trimpath -ldflags "-s -w -H windowsgui -X main.version=$version" -o $artifactPath .
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

if (Test-Path $artifactPath) {
    $size = (Get-Item $artifactPath).Length
    Write-Host "Build succeeded: out/$artifactName ($size bytes, v$version)"
} else {
    Write-Error "Build failed: out/$artifactName not found"
    exit 1
}
