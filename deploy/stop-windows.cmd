@echo off
setlocal
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0stop-windows.ps1" -UseLocalBuild %*
set "EXIT_CODE=%ERRORLEVEL%"
echo.
exit /b %EXIT_CODE%
