@echo off
setlocal EnableExtensions

set "SCRIPT_DIR=%~dp0"
set "ACTION=%~1"

if "%ACTION%"=="" goto menu
goto dispatch

:menu
echo.
echo Nexus Windows Control
echo =====================
echo 1. Start
echo 2. Start and follow logs
echo 3. Stop
echo 4. Restart
echo 5. Logs
echo 6. Status
echo 0. Exit
echo.
set /p "CHOICE=Choose an option: "

if "%CHOICE%"=="1" set "ACTION=start" & goto dispatch
if "%CHOICE%"=="2" set "ACTION=start-logs" & goto dispatch
if "%CHOICE%"=="3" set "ACTION=stop" & goto dispatch
if "%CHOICE%"=="4" set "ACTION=restart" & goto dispatch
if "%CHOICE%"=="5" set "ACTION=logs" & goto dispatch
if "%CHOICE%"=="6" set "ACTION=status" & goto dispatch
if "%CHOICE%"=="0" exit /b 0

echo Invalid option.
exit /b 1

:dispatch
if /I "%ACTION%"=="start" goto start
if /I "%ACTION%"=="up" goto start
if /I "%ACTION%"=="start-logs" goto start_logs
if /I "%ACTION%"=="stop" goto stop
if /I "%ACTION%"=="down" goto stop
if /I "%ACTION%"=="restart" goto restart
if /I "%ACTION%"=="logs" goto logs
if /I "%ACTION%"=="status" goto status
if /I "%ACTION%"=="ps" goto status

echo Usage:
echo   nexus-windows.bat [start^|start-logs^|stop^|restart^|logs^|status]
exit /b 1

:start
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%start-windows.ps1" -UseLocalBuild
exit /b %ERRORLEVEL%

:start_logs
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%start-windows.ps1" -UseLocalBuild -Logs
exit /b %ERRORLEVEL%

:stop
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%stop-windows.ps1" -UseLocalBuild
exit /b %ERRORLEVEL%

:restart
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%stop-windows.ps1" -UseLocalBuild
if errorlevel 1 exit /b %ERRORLEVEL%
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%start-windows.ps1" -UseLocalBuild
exit /b %ERRORLEVEL%

:logs
pushd "%SCRIPT_DIR%"
docker compose -f docker-compose.local.yml -f docker-compose.windows.yml logs -f nexus
set "EXIT_CODE=%ERRORLEVEL%"
popd
exit /b %EXIT_CODE%

:status
pushd "%SCRIPT_DIR%"
docker compose -f docker-compose.local.yml -f docker-compose.windows.yml ps
set "EXIT_CODE=%ERRORLEVEL%"
popd
exit /b %EXIT_CODE%
