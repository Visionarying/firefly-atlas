# Recreate the Windows shortcut with the Firefly Atlas icon.
# Run from anywhere:
#   powershell -ExecutionPolicy Bypass -File scripts\create-shortcut.ps1
$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$icon = Join-Path $root 'assets\firefly-atlas.ico'
$vbs = Join-Path $PSScriptRoot 'start-firefly-atlas.vbs'
$wscript = Join-Path $env:SystemRoot 'System32\wscript.exe'

if (-not (Test-Path $icon)) { throw "Icon not found: $icon" }
if (-not (Test-Path $vbs)) { throw "Launcher not found: $vbs" }
if (-not (Test-Path $wscript)) { throw "wscript.exe not found: $wscript" }

$shell = New-Object -ComObject WScript.Shell
$linkPath = Join-Path $root 'Firefly Atlas.lnk'
$link = $shell.CreateShortcut($linkPath)
$link.TargetPath = $wscript
$link.Arguments = '"' + $vbs + '"'
$link.WorkingDirectory = $root
$link.IconLocation = "$icon,0"
$link.Description = '启动 Firefly Atlas 导航面板'
$link.WindowStyle = 1
$link.Save()

$desktop = [Environment]::GetFolderPath('Desktop')
if ($desktop) {
  Copy-Item $linkPath (Join-Path $desktop 'Firefly Atlas.lnk') -Force
}

Write-Host "Created: $linkPath"
if ($desktop) { Write-Host "Desktop: $(Join-Path $desktop 'Firefly Atlas.lnk')" }
