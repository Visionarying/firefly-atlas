# Recreate the Windows shortcut for the Firefly Atlas GUI launcher.
# Run from anywhere:
#   powershell -ExecutionPolicy Bypass -File scripts\create-shortcut.ps1
$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$target = Join-Path $root 'Firefly Atlas.exe'
$icon = Join-Path $root 'assets\firefly-atlas.ico'

if (-not (Test-Path $target)) {
  throw "Launcher not found: $target. Run scripts\build-launcher.ps1 first."
}
if (-not (Test-Path $icon)) { throw "Icon not found: $icon" }

$shell = New-Object -ComObject WScript.Shell
$linkPath = Join-Path $root 'Firefly Atlas.lnk'
$link = $shell.CreateShortcut($linkPath)
$link.TargetPath = $target
$link.Arguments = ''
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
