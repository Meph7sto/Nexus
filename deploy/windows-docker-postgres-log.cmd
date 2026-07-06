@echo off
setlocal EnableExtensions

title Nexus Docker Postgres Logs

set "DEPLOY_DIR=%~dp0"
set "ROOT=%DEPLOY_DIR%.."
for %%I in ("%ROOT%") do set "ROOT=%%~fI"
set "LOG_FILE=%ROOT%\logs\docker-postgres.log"

if not exist "%ROOT%\logs" mkdir "%ROOT%\logs"

cd /d "%DEPLOY_DIR%"
echo.
echo [Nexus Docker Postgres Logs]
echo Log file: %LOG_FILE%
echo.
echo ===== Postgres logs started at %DATE% %TIME% =====>> "%LOG_FILE%"
docker compose -f docker-compose.local.yml -f docker-compose.windows.yml logs -f postgres 2>&1 | powershell.exe -NoProfile -Command "$input | Tee-Object -FilePath '%LOG_FILE%' -Append"

echo.
echo Log stream ended. Window stays open so the error above is visible.
echo.
pause
