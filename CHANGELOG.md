# Changelog

All notable changes to this project will be documented in this file.

## [0.0.8] - 2026-04-06

### Added
- **Unified Monitor (`hoyo.exe`)**: Consolidated Genshin, HSR, and ZZZ into a single, high-performance system tray application.
- **Login & Settings UI**: Interactive WebView2-based login helper and visual settings dashboard.
- **High-Visibility Alerting**: Introduced `!` (threshold reached) and `!!!` (capacity maxed) markers for clear status feedback.
- **Unified Build System**: Introduced a master `makefile` that automates Go icon baking, C# publishing, and asset synchronization.
- **Single-Instance Protection**: Implemented a Windows Mutex and professional "Application already running" prompt.
- **Self-Healing Assets**: Automated extraction of icons and resources from the binary if missing from disk.

### Changed
- **Login Helper Architecture**: Migrated `WebViewLogin.exe` to a single-file bundled executable for superior portability.
- **Unified Config**: Migrated to a single `hoyo_cookie.json` format for all game credentials.
- **Performance**: Optimized the refresh loop with parallel polling for HSR, ZZZ, and Genshin.
- **Repository Cleanliness**: Updated `.gitignore` to exclude build artifacts and machine-specific files.

### Fixed
- **Icon Embedding (Icon Baking)**: Resolved an issue where icons were skipped when building from the root. Now builds from package directories for perfect resource discovery.
- **WebView2 Loader Sync**: Fixed the `makefile` path for `WebView2Loader.dll` to ensure it is always correctly bundled.
- **Legacy Synchronization**: Fully synchronized `resin.exe`, `stamina.exe`, and `charge.exe` to use the new backend logic.

## [v0.0.1] - Initial Release
- Individual monitors for Genshin, HSR, and ZZZ.
