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

# Ensure output directory exists on the host (bind-mounted into the container).
New-Item -ItemType Directory -Path "$PSScriptRoot\..\out" -Force | Out-Null

docker compose run --rm -e "BUILD_VERSION=$version" -e "ARTIFACT_NAME=$artifactName" build
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

if (Test-Path $artifactPath) {
    $size = (Get-Item $artifactPath).Length
    Write-Host "Build succeeded: out/$artifactName ($size bytes, v$version)"
} else {
    Write-Error "Build failed: out/$artifactName not found"
    exit 1
}
