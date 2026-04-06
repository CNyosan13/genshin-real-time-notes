@echo off
echo Compiling legacy resources...
for /f "tokens=*" %%i in ('go env GOPATH') do set "GOPATH=%%i"

pushd cmd\resin
"%GOPATH%\bin\go-winres.exe" make
echo Compiling resin.exe...
go build -ldflags "-H=windowsgui" -o ..\..\resin.exe .
del *.syso
popd

pushd cmd\stamina
"%GOPATH%\bin\go-winres.exe" make
echo Compiling stamina.exe...
go build -ldflags "-H=windowsgui" -o ..\..\stamina.exe .
del *.syso
popd

pushd cmd\charge
"%GOPATH%\bin\go-winres.exe" make
echo Compiling charge.exe...
go build -ldflags "-H=windowsgui" -o ..\..\charge.exe .
del *.syso
popd

if %errorlevel% neq 0 (
    echo BUILD FAILED!
    pause
    exit /b %errorlevel%
)
echo SUCCESS - Legacy binaries created with icons!
pause
