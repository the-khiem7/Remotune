# Remotune build delegate — produces out/Remotune.exe inside Docker.
# Usage: .\scripts\build-windows.ps1
# This script ONLY orchestrates Docker. No project SDK runs on the host.

$ErrorActionPreference = 'Stop'

# Ensure output directory exists on the host (bind-mounted into the container).
New-Item -ItemType Directory -Path "$PSScriptRoot\..\out" -Force | Out-Null

docker compose run --rm build
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$exe = Join-Path $PSScriptRoot "..\out\Remotune.exe"
if (Test-Path $exe) {
    $size = (Get-Item $exe).Length
    Write-Host "Build succeeded: out/Remotune.exe ($size bytes)"
} else {
    Write-Error "Build failed: out/Remotune.exe not found"
    exit 1
}
