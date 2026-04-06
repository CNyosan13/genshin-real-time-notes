# Hoyo Monitor - Compilation Guide

This guide describes how to compile the Hoyo Monitor suite from source. The easiest way is to use the `makefile`, but manual steps are also provided.

## Prerequisites
1. **Go 1.20+**
2. **.NET 8.0 SDK** (with Desktop Development workload)
3. **go-winres** (`go install github.com/tc-hib/go-winres@latest`)
4. **Make** (standard on Linux/macOS; use [GnuWin32](http://gnuwin32.sourceforge.net/packages/make.htm) or Git Bash on Windows)

---

## 1. Automated Build (Recommended)
The project includes a `makefile` that automates the complex cross-platform build and asset synchronization.

1. Open a terminal in the root folder (e.g., Git Bash).
2. Run:
   ```bash
   make all
   ```
This will:
- Compile the C# `WebViewLogin` helper as a single-file executable.
- Synchronize all native dependencies (`WebView2Loader.dll`).
- "Bake" icons into `hoyo.exe`, `resin.exe`, `stamina.exe`, and `charge.exe`.

---

## 2. Manual Compilation (Fallback)
If you cannot use `make`, follow these steps:

### A. Compile the C# WebView Helper
1. Run:
   ```bash
   dotnet publish LoginForm/WinFormsApp1/WebViewLogin.csproj -c Release -p:PublishProfile=FolderProfile
   ```
2. Copy `LoginForm/WinFormsApp1/bin/Release/net8.0-windows/publish/win-x64/WebViewLogin.exe` to `embedded/login/WebViewLogin-v0.0.8.exe`.
3. Copy `WebView2Loader.dll` from the `bin` folder's native runtimes into `embedded/login/`.

### B. Compile Go Binaries with Icons
To ensure icons are "baked" correctly, you **must** build from within each app's directory:
1. `cd cmd/hoyo`
2. `go-winres make`
3. `go build -ldflags "-H=windowsgui" -o ../../hoyo.exe .`
4. `rm *.syso` (optional cleanup)

Repeat for `cmd/resin`, `cmd/stamina`, and `cmd/charge`.
