# Remotune host-native verification — generates Wails assets, then runs format,
# static analysis, frontend checks, and tests.
# Usage: .\scripts\verify.ps1
# Docker is retired infrastructure and is not used by this verification path.

$ErrorActionPreference = 'Stop'
$repoRoot = Join-Path $PSScriptRoot '..'
$wails = Join-Path (go env GOPATH) 'bin\wails3.exe'

Push-Location $repoRoot
try {
    if (-not (Test-Path $wails)) { throw "Wails CLI v3.0.0-beta.8 is required at $wails" }
    & $wails generate bindings -ts
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    Push-Location frontend
    try {
        npm ci
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        npm run build
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    } finally { Pop-Location }
    & $wails generate syso -arch amd64 -icon build/windows/icon.ico -manifest build/windows/wails.exe.manifest -out wails_windows_amd64.syso
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    $unformatted = @(gofmt -l . | Where-Object { $_ -notmatch '^(engine[\\/])' })
    if ($unformatted.Count -gt 0) { throw "Files need gofmt:`n$($unformatted -join [Environment]::NewLine)" }
    go vet ./...
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    go test ./... -count=1 -short
    exit $LASTEXITCODE
} finally { Pop-Location }
