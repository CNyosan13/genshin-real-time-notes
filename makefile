VERSION := v0.0.8

.PHONY: hoyo resin stamina charge all clean login legacy

ENV := CGO_ENABLED=1 GOOS=windows GOARCH=amd64
LDFLAGS := -ldflags "-H=windowsgui"

# Default: build everything from scratch
all: login hoyo legacy

# Build and sync the C# UI helper
login:
	dotnet publish LoginForm/WinFormsApp1/WebViewLogin.csproj -c Release -p:PublishProfile=FolderProfile
	rm -f embedded/login/*.exe embedded/login/*.dll
	mkdir -p embedded/login
	cp "LoginForm/WinFormsApp1/bin/Release/net8.0-windows/publish/win-x64/WebViewLogin.exe" "embedded/login/WebViewLogin-$(VERSION).exe"
	cp "LoginForm/WinFormsApp1/bin/Release/net8.0-windows/win-x64/runtimes/win-x64/native/WebView2Loader.dll" "embedded/login/WebView2Loader.dll"

# Build individual apps with correct "Icon Baking"
hoyo:
	cd cmd/hoyo && go-winres make
	cd cmd/hoyo && ${ENV} go build $(LDFLAGS) -o ../../hoyo.exe .
	rm -f cmd/hoyo/*.syso

resin:
	cd cmd/resin && go-winres make
	cd cmd/resin && ${ENV} go build $(LDFLAGS) -o ../../resin.exe .
	rm -f cmd/resin/*.syso

stamina:
	cd cmd/stamina && go-winres make
	cd cmd/stamina && ${ENV} go build $(LDFLAGS) -o ../../stamina.exe .
	rm -f cmd/stamina/*.syso

charge:
	cd cmd/charge && go-winres make
	cd cmd/charge && ${ENV} go build $(LDFLAGS) -o ../../charge.exe .
	rm -f cmd/charge/*.syso

legacy: resin stamina charge

clean:
	rm -rf hoyo*.exe resin*.exe stamina*.exe charge*.exe
	rm -f embedded/login/*.exe embedded/login/*.dll

