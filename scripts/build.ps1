param(
  [switch]$SkipFrontend,
  [switch]$SkipBackend
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot

if (-not $SkipFrontend) {
  Write-Host '==> building frontend' -ForegroundColor Cyan
  Push-Location (Join-Path $root 'web')
  try {
    npm run build:copy
    if ($LASTEXITCODE -ne 0) { throw 'frontend build failed' }
  } finally {
    Pop-Location
  }
}

if (-not $SkipBackend) {
  Write-Host '==> building backend' -ForegroundColor Cyan
  Push-Location (Join-Path $root 'service')
  try {
    go build -trimpath -ldflags '-s -w' -o firefly-atlas.exe .
    if ($LASTEXITCODE -ne 0) { throw 'backend build failed' }
  } finally {
    Pop-Location
  }
}

Write-Host "done: $root\service\firefly-atlas.exe" -ForegroundColor Green
Write-Host "run : `$env:DATA_DIR='$root\data'; .\service\firefly-atlas.exe"
