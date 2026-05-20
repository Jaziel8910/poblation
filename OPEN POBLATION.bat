@echo off
setlocal
title POBLATION
chcp 65001 >nul

set "ROOT=%~dp0"
set "LAUNCHER=%USERPROFILE%\.poblation\launcher\bin\poblation-launcher.exe"
set "INSTALLER=%ROOT%poblation_v1.0.0.2_launcher_installer.exe"

echo.
echo POBLATION
echo =========
echo.

if exist "%LAUNCHER%" (
  echo Opening installed launcher in a better terminal...
  call :open_launcher ""
  exit /b 0
)

if exist "%INSTALLER%" (
  echo Launcher is not installed yet.
  echo Installing launcher now...
  "%INSTALLER%"
  if errorlevel 1 (
    echo.
    echo Install failed. Read TUTORIAL.txt or send the screenshot/output.
    pause
    exit /b 1
  )
  echo.
  echo Opening launcher...
  call :open_launcher ""
  exit /b 0
)

echo Could not find the launcher or installer.
echo Keep OPEN POBLATION.bat in the same folder as:
echo   poblation_v1.0.0.2_launcher_installer.exe
echo.
echo If you downloaded only the game exe, open a terminal and run:
echo   poblation_windows_amd64.exe
pause
exit /b 1

:open_launcher
set "ARG=%~1"
where wt.exe >nul 2>nul
if not errorlevel 1 (
  start "" wt.exe -w 0 nt --title "POBLATION" powershell.exe -NoLogo -NoExit -ExecutionPolicy Bypass -Command "$Host.UI.RawUI.WindowTitle='POBLATION'; [Console]::OutputEncoding=[Text.UTF8Encoding]::UTF8; & '%LAUNCHER%' %ARG%"
  exit /b 0
)
start "POBLATION" powershell.exe -NoLogo -NoExit -ExecutionPolicy Bypass -Command "$Host.UI.RawUI.WindowTitle='POBLATION'; [Console]::OutputEncoding=[Text.UTF8Encoding]::UTF8; & '%LAUNCHER%' %ARG%"
exit /b 0
