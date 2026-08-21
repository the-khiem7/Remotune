# Remotune shell delegate — opens an interactive bash inside the build container.
# Usage: .\scripts\shell.ps1
# Provides Go 1.26, Wails CLI, Bun, and the full project source at /src.
# This script ONLY orchestrates Docker. No project SDK runs on the host.

$ErrorActionPreference = 'Stop'
Set-Location (Join-Path $PSScriptRoot '..')
& powershell.exe -NoExit -NoLogo
