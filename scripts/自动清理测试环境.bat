@echo off
setlocal enabledelayedexpansion
title CCSwitch Clean
echo.
echo ========================================
echo    CCSwitch - Clean Test Environment
echo ========================================
echo.

echo [1/5] Uninstalling Claude Code...
if exist "C:\Program Files\nodejs\npm.cmd" (
    call "C:\Program Files\nodejs\npm.cmd" uninstall -g @anthropic-ai/claude-code 2>nul
)
if exist "%APPDATA%\npm\npm.cmd" (
    call "%APPDATA%\npm\npm.cmd" uninstall -g @anthropic-ai/claude-code 2>nul
)
if exist "C:\Program Files\nodejs\node_modules\@anthropic-ai" rmdir /s /q "C:\Program Files\nodejs\node_modules\@anthropic-ai" 2>nul
if exist "C:\Program Files\nodejs\claude.cmd" del /f /q "C:\Program Files\nodejs\claude.cmd" 2>nul
if exist "%APPDATA%\npm\claude.cmd" del /f /q "%APPDATA%\npm\claude.cmd" 2>nul
if exist "%APPDATA%\npm\node_modules\@anthropic-ai" rmdir /s /q "%APPDATA%\npm\node_modules\@anthropic-ai" 2>nul
echo       Claude Code uninstalled.
echo.

echo [2/5] Cleaning user PATH...
powershell -ExecutionPolicy Bypass -File "%~dp0clean_path.ps1"
echo.

echo [3/5] Removing Node.js...
if exist "C:\Program Files\nodejs" (
    powershell -ExecutionPolicy Bypass -Command "Remove-Item -Recurse -Force 'C:\Program Files\nodejs' -ErrorAction SilentlyContinue"
    echo       Node.js removed.
) else (
    echo       Node.js not installed, skip.
)
if exist "%APPDATA%\npm" rmdir /s /q "%APPDATA%\npm" 2>nul
if exist "C:\node-v22*" rmdir /s /q "C:\node-v22*" 2>nul
echo.

echo [4/5] Removing Git for Windows...
if exist "C:\Program Files\Git" (
    powershell -ExecutionPolicy Bypass -Command "Remove-Item -Recurse -Force 'C:\Program Files\Git' -ErrorAction SilentlyContinue"
    echo       Git for Windows removed.
) else (
    echo       Git for Windows not installed, skip.
)
echo.

echo [5/5] Cleaning config and temp files...
if exist "%USERPROFILE%\.claude" (
    powershell -ExecutionPolicy Bypass -Command "Remove-Item -Recurse -Force (Join-Path $env:USERPROFILE '.claude') -ErrorAction SilentlyContinue"
    echo       .claude dir removed.
) else (
    echo       .claude not found, skip.
)
if exist "%USERPROFILE%\.ccswitch" (
    powershell -ExecutionPolicy Bypass -Command "Remove-Item -Recurse -Force (Join-Path $env:USERPROFILE '.ccswitch') -ErrorAction SilentlyContinue"
    echo       .ccswitch dir removed.
) else (
    echo       .ccswitch not found, skip.
)
del /f /q "%TEMP%\nodejs-lts.zip" 2>nul
del /f /q "%TEMP%\nodejs-lts.msi" 2>nul
del /f /q "%TEMP%\nodejs*.zip" 2>nul
del /f /q "%TEMP%\nodejs*.msi" 2>nul
del /f /q "%TEMP%\install-nodejs.ps1" 2>nul
del /f /q "%TEMP%\ccswitch-*.log" 2>nul
del /f /q "%TEMP%\cli-install.log" 2>nul
del /f /q "%TEMP%\Git-*.exe" 2>nul
del /f /q "%TEMP%\gui-stderr.log" 2>nul
echo       Temp files cleaned.
echo.

echo ========================================
echo    Clean complete!
echo ========================================
echo.
pause
