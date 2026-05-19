@echo off
setlocal
title POBLATION

set "ROOT=%~dp0"
set "LAUNCHER=%USERPROFILE%\.poblation\launcher\bin\poblation-launcher.exe"
set "INSTALLER=%ROOT%poblation_v1.0.0.2_launcher_installer.exe"

echo.
echo POBLATION
echo =========
echo.

if exist "%LAUNCHER%" (
  echo Opening installed launcher...
  "%LAUNCHER%"
  pause
  exit /b %ERRORLEVEL%
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
  "%LAUNCHER%"
  pause
  exit /b %ERRORLEVEL%
)

echo Could not find the launcher or installer.
echo Keep OPEN POBLATION.bat in the same folder as:
echo   poblation_v1.0.0.2_launcher_installer.exe
echo.
echo If you downloaded only the game exe, open a terminal and run:
echo   poblation_windows_amd64.exe
pause
exit /b 1
