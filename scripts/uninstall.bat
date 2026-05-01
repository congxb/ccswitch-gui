@echo off
chcp 65001 >nul 2>&1
title CCSwitch 一键卸载 Claude Code
echo.
echo ═══════════════════════════════════════
echo    CCSwitch - 一键卸载 Claude Code
echo ═══════════════════════════════════════
echo.

:: 步骤 1：卸载 Claude Code
echo [1/4] 正在卸载 Claude Code...
call npm uninstall -g @anthropic-ai/claude-code 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo       Claude Code 未通过 npm 安装，尝试直接删除...
    if exist "%APPDATA%\npm\claude.cmd" del /f /q "%APPDATA%\npm\claude.cmd" 2>nul
    if exist "%APPDATA%\npm\claude" rmdir /s /q "%APPDATA%\npm\claude" 2>nul
    if exist "%APPDATA%\npm\node_modules\@anthropic-ai" rmdir /s /q "%APPDATA%\npm\node_modules\@anthropic-ai" 2>nul
)
echo       Claude Code 已卸载
echo.

:: 步骤 2：清理用户 PATH 中的 npm 全局目录
echo [2/4] 清理用户 PATH 中的 npm 目录...
for /f "tokens=*" %%p in ('npm config get prefix 2^>nul') do set NPM_PREFIX=%%p
if not defined NPM_PREFIX set NPM_PREFIX=%APPDATA%\npm

:: 从用户 PATH 中移除 npm prefix
powershell -Command ^
    "$path = [Environment]::GetEnvironmentVariable('Path','User'); ^
     $dirs = $path -split ';' | Where-Object { $_ -ne '%NPM_PREFIX%' -and $_.Trim() -ne '' }; ^
     $newPath = $dirs -join ';'; ^
     [Environment]::SetEnvironmentVariable('Path', $newPath, 'User'); ^
     Write-Host ('       已从 PATH 移除: ' + '%NPM_PREFIX%')"

:: 从用户 PATH 中移除 Git 相关目录
powershell -Command ^
    "$path = [Environment]::GetEnvironmentVariable('Path','User'); ^
     $gitDirs = @('C:\Program Files\Git\cmd','C:\Program Files\Git\bin'); ^
     $dirs = $path -split ';' | Where-Object { $d=$_.Trim(); $gitDirs -notcontains $d }; ^
     $newPath = $dirs -join ';'; ^
     [Environment]::SetEnvironmentVariable('Path', $newPath, 'User'); ^
     Write-Host '       已从 PATH 移除 Git 目录'"

:: 广播环境变量更改
powershell -Command ^
    "Add-Type -TypeDefinition 'using System;using System.Runtime.InteropServices;public class Env{[DllImport(\"user32.dll\")]public static extern IntPtr SendMessageTimeout(IntPtr h,uint m,IntPtr w,string l,uint f,uint t,out IntPtr r);public static void Broadcast(){IntPtr r;SendMessageTimeout((IntPtr)0xffff,0x001A,IntPtr.Zero,\"Environment\",0x0002,0x2710,out r);}}'; [Env]::Broadcast()"

echo.

:: 步骤 3：卸载 Node.js（可选）
echo [3/4] 检测 Node.js...
where node >nul 2>&1
if %ERRORLEVEL% EQU 0 (
    echo       检测到 Node.js 已安装
    set /p UNINSTALL_NODE="       是否同时卸载 Node.js? (y/N): "
    if /i "!UNINSTALL_NODE!"=="y" (
        echo       正在通过 winget 卸载 Node.js...
        winget uninstall --id OpenJS.NodeJS.LTS --source winget --accept-source-agreements --disable-interactivity 2>nul
        if %ERRORLEVEL% EQU 0 (
            echo       Node.js 已卸载
        ) else (
            echo       winget 卸载失败，请手动卸载
        )
    ) else (
        echo       保留 Node.js
    )
) else (
    echo       Node.js 未安装，跳过
)
echo.

:: 步骤 4：清理配置文件（可选）
echo [4/4] 检测 Claude 配置文件...
if exist "%USERPROFILE%\.claude" (
    set /p CLEAN_CONFIG="       是否删除 .claude 配置目录? (y/N): "
    if /i "!CLEAN_CONFIG!"=="y" (
        rmdir /s /q "%USERPROFILE%\.claude" 2>nul
        echo       .claude 目录已删除
    ) else (
        echo       保留 .claude 目录
    )
) else (
    echo       .claude 目录不存在，跳过
)

if exist "%USERPROFILE%\.ccswitch" (
    set /p CLEAN_CCSWITCH="       是否删除 .ccswitch 配置目录? (y/N): "
    if /i "!CLEAN_CCSWITCH!"=="y" (
        rmdir /s /q "%USERPROFILE%\.ccswitch" 2>nul
        echo       .ccswitch 目录已删除
    ) else (
        echo       保留 .ccswitch 目录
    )
) else (
    echo       .ccswitch 目录不存在，跳过
)

echo.
echo ═══════════════════════════════════════
echo    卸载完成！
echo ═══════════════════════════════════════
echo.
echo 请关闭并重新打开终端以使环境变量生效。
echo.
pause
