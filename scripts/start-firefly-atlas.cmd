@echo off
setlocal
cd /d "%~dp0"

REM Always use the real data directory next to this script, not service\data.
set "DATA_DIR=%~dp0data"
if not defined PORT set "PORT=3002"
if not defined GIN_MODE set "GIN_MODE=release"

if not exist "service\firefly-atlas.exe" (
  echo.
  echo service\firefly-atlas.exe was not found.
  echo Build it first with: scripts\build.ps1
  echo.
  pause
  exit /b 1
)

REM If an instance is already running, just open the browser instead of
REM starting a second process that would fail with "address already in use".
powershell -NoProfile -Command "try { $r = Invoke-WebRequest -Uri ('http://127.0.0.1:%PORT%/api/v1/health') -UseBasicParsing -TimeoutSec 2; if ($r.StatusCode -eq 200) { exit 0 } } catch {}; exit 1"
if %errorlevel%==0 (
  echo Firefly Atlas is already running at http://127.0.0.1:%PORT%
  start "" "http://127.0.0.1:%PORT%"
  exit /b 0
)

echo.
echo Starting Firefly Atlas...
echo   URL:  http://127.0.0.1:%PORT%
echo   Data: %DATA_DIR%
echo.
echo A server window will stay open with logs. Close it to stop the panel.
echo.

REM Keep the server in its own console so errors remain readable, then wait for
REM the health endpoint before opening the browser.
start "Firefly Atlas Server" cmd /k "service\firefly-atlas.exe"
for /l %%i in (1,1,30) do (
  powershell -NoProfile -Command "try { $r = Invoke-WebRequest -Uri ('http://127.0.0.1:%PORT%/api/v1/health') -UseBasicParsing -TimeoutSec 2; if ($r.StatusCode -eq 200) { exit 0 } } catch {}; exit 1"
  if not errorlevel 1 (
    start "" "http://127.0.0.1:%PORT%"
    exit /b 0
  )
  timeout /t 1 >nul
)

echo.
echo The server did not become ready within 30 seconds.
echo Check the "Firefly Atlas Server" window for the error message.
echo.
pause
