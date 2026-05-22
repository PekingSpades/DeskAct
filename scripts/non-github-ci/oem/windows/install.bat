@echo off
setlocal
set OEM_LOG=C:\OEM\deskact-oem.cmd.log
echo [%DATE% %TIME%] starting DeskAct Windows OEM build > "%OEM_LOG%"
powershell.exe -NoProfile -ExecutionPolicy Bypass -File C:\OEM\run-deskact-build.ps1 >> "%OEM_LOG%" 2>&1
echo [%DATE% %TIME%] finished DeskAct Windows OEM build with %ERRORLEVEL% >> "%OEM_LOG%"
exit /b %ERRORLEVEL%
