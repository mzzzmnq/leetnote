@echo off
REM =============================================================
REM  LeetNote launcher - double click this file to open the menu.
REM =============================================================
REM
REM  All Chinese text lives in launcher.ps1 on purpose:
REM  cmd.exe parses .bat bytes using the active code page, and
REM  mixing non-ASCII literals with a mid-file "chcp 65001" is a
REM  well known source of garbled output. Keeping this file ASCII
REM  and letting PowerShell print the UI avoids that entirely.
REM

chcp 65001 >nul 2>&1
cd /d "%~dp0"
title LeetNote

REM A slightly wider console makes the dashboard layout line up.
mode con cols=72 lines=32 >nul 2>&1

set "PWSH_EXE=%ProgramFiles%\PowerShell\7\pwsh.exe"

if exist "%PWSH_EXE%" (
    "%PWSH_EXE%" -NoProfile -ExecutionPolicy Bypass -File "%~dp0launcher.ps1"
    goto done
)

where pwsh >nul 2>&1
if %errorlevel%==0 (
    pwsh -NoProfile -ExecutionPolicy Bypass -File "%~dp0launcher.ps1"
    goto done
)

REM Fallback: Windows PowerShell 5.1 (may render CJK less cleanly).
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0launcher.ps1"

:done
if errorlevel 1 (
    echo.
    echo [launcher exited with an error - press any key to close]
    pause >nul
)
