@echo off
setlocal EnableExtensions

set "ROOT=%~dp0"
if "%ROOT:~-1%"=="\" set "ROOT=%ROOT:~0,-1%"

echo.
echo Stopping Nexus Docker services...
echo.

taskkill /F /T /FI "WINDOWTITLE eq Nexus Docker App Logs*" >nul 2>nul
taskkill /F /T /FI "WINDOWTITLE eq Nexus Docker Postgres Logs*" >nul 2>nul
taskkill /F /T /FI "WINDOWTITLE eq Nexus Docker Redis Logs*" >nul 2>nul

where docker >nul 2>nul
if errorlevel 1 (
  echo Docker was not found. Log windows were closed if they existed.
  echo.
  pause
  exit /b 0
)

docker info >nul 2>nul
if errorlevel 1 (
  echo Docker is not running. Log windows were closed if they existed.
  echo.
  pause
  exit /b 0
)

powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%ROOT%\deploy\stop-windows.ps1" -UseLocalBuild
set "EXIT_CODE=%ERRORLEVEL%"

echo.
if "%EXIT_CODE%"=="0" (
  echo Nexus Docker services stopped. Data is kept in deploy\data, deploy\postgres_data, and deploy\redis_data.
) else (
  echo [ERROR] Stop failed with code %EXIT_CODE%.
)
echo.
pause
exit /b %EXIT_CODE%
