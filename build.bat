@echo off
REM ConfigChief build script (no wails CLI / no npm required)
REM Builds one program (spec 15): build\bin\ConfigChief.exe
REM   started alone or with a case folder  -> the window
REM   started with a command               -> the command line, no window
setlocal
set SCRIPT_DIR=%~dp0
cd /d "%SCRIPT_DIR%"

echo go vet ...
go vet ./...
if errorlevel 1 (
  echo VET FAILED
  exit /b 1
)
echo go test ...
go test ./...
if errorlevel 1 (
  echo TEST FAILED
  exit /b 1
)
echo Building ConfigChief.exe ...
go build -trimpath -buildvcs=false -tags production -ldflags="-H windowsgui -s -w" -o build\bin\ConfigChief.exe .
if errorlevel 1 (
  echo BUILD FAILED
  exit /b 1
)
echo.
echo Done: build\bin\ConfigChief.exe
endlocal
