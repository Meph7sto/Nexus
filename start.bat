@echo off
setlocal EnableExtensions

set "ROOT=%~dp0"
if "%ROOT:~-1%"=="\" set "ROOT=%ROOT:~0,-1%"

if not exist "%ROOT%\logs" mkdir "%ROOT%\logs"

echo.
echo Nexus Docker startup
echo ====================
echo This script builds the local Nexus Docker image, starts PostgreSQL, Redis,
echo and Nexus, then opens log windows for each service.
echo.
echo Logs:
echo   %ROOT%\logs\docker-nexus.log
echo   %ROOT%\logs\docker-postgres.log
echo   %ROOT%\logs\docker-redis.log
echo.

where docker >nul 2>nul
if errorlevel 1 (
  echo [ERROR] Docker was not found in PATH.
  echo Install and start Docker Desktop, then run this script again.
  echo.
  pause
  exit /b 1
)

docker info >nul 2>nul
if errorlevel 1 (
  echo [ERROR] Docker is not running.
  echo Start Docker Desktop, wait until it is ready, then run this script again.
  echo.
  pause
  exit /b 1
)

echo Starting Docker services. The first run can take several minutes because it builds the image.
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%ROOT%\deploy\start-windows.ps1" -UseLocalBuild
set "EXIT_CODE=%ERRORLEVEL%"
if not "%EXIT_CODE%"=="0" (
  echo.
  echo [ERROR] Docker startup failed with code %EXIT_CODE%.
  echo.
  pause
  exit /b %EXIT_CODE%
)

echo.
echo Opening log windows...
start "Nexus Docker App Logs" cmd /k ""%ROOT%\deploy\windows-docker-nexus-log.cmd""
start "Nexus Docker Postgres Logs" cmd /k ""%ROOT%\deploy\windows-docker-postgres-log.cmd""
start "Nexus Docker Redis Logs" cmd /k ""%ROOT%\deploy\windows-docker-redis-log.cmd""

echo.
echo Nexus is starting.
set "SERVER_PORT=18080"
for /f "tokens=1,* delims==" %%A in ('findstr /b /c:"SERVER_PORT=" "%ROOT%\deploy\.env" 2^>nul') do set "SERVER_PORT=%%B"
if "%SERVER_PORT%"=="" set "SERVER_PORT=18080"
echo Web UI: http://localhost:%SERVER_PORT%
echo.
echo Use stop.bat to stop the containers and close the log windows.
echo.
pause
