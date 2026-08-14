# Remotune verify delegate — runs format, lint, static analysis, and tests inside Docker.
# Usage: .\scripts\verify.ps1
# This script ONLY orchestrates Docker. No project SDK runs on the host.

$ErrorActionPreference = 'Stop'
docker compose run --rm verify
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
