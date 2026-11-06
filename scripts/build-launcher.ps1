# Build the Windows GUI launcher with the Firefly Atlas icon.
# Run from anywhere:
#   powershell -ExecutionPolicy Bypass -File scripts\build-launcher.ps1
$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$icon = Join-Path $root 'assets\firefly-atlas.ico'
$launcherDir = Join-Path $root 'launcher'
$syso = Join-Path $launcherDir 'rsrc_windows_amd64.syso'
$out = Join-Path $root 'Firefly Atlas.exe'

if (-not (Test-Path $icon)) { throw "Icon not found: $icon" }

$rsrc = Join-Path $env:USERPROFILE 'go\bin\rsrc.exe'
if (-not (Test-Path $rsrc)) {
  Write-Host 'Installing rsrc...'
  go install github.com/akavel/rsrc@latest
  if ($LASTEXITCODE -ne 0) { throw 'go install rsrc failed' }
}

& $rsrc -ico $icon -o $syso
if ($LASTEXITCODE -ne 0) { throw 'rsrc failed' }

Push-Location $launcherDir
try {
  go build -trimpath -ldflags '-s -w -H=windowsgui' -o $out .
  if ($LASTEXITCODE -ne 0) { throw 'launcher build failed' }
} finally {
  Pop-Location
}

Write-Host "Built: $out" -ForegroundColor Green
