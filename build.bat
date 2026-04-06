@echo off
echo Compiling Resources...
for /f "tokens=*" %%i in ('go env GOPATH') do set "GOPATH=%%i"
pushd cmd\hoyo
"%GOPATH%\bin\go-winres.exe" make
echo Building Unified HoyoLAB Monitor...
go build -ldflags "-H=windowsgui -X resin/pkg/config.VERSION=v0.0.8" -o ..\..\hoyo.exe .
del *.syso
popd
if %errorlevel% neq 0 (
    echo BUILD FAILED!
    pause
    exit /b %errorlevel%
)
echo SUCCESS - hoyo.exe created with embedded icon!
pause
