# Changelog

All notable changes to this project will be documented in this file.

## [0.0.8] - 2026-04-06

### Added
- **Unified Monitor (`hoyo.exe`)**: Consolidated Genshin, HSR, and ZZZ into a single, high-performance system tray application.
- **Login & Settings UI**: Interactive WebView2-based login helper and visual settings dashboard.
- **High-Visibility Alerting**: Introduced `!` (threshold reached) and `!!!` (capacity maxed) markers for clear status feedback without relying on unstable emojis.
- **Automated Resource System**: Integrated `go-winres` into `build.bat` and `build_legacy.bat` for automatic, professional icon embedding.
- **Stealth Build Mode**: Resource "glue" files (`.syso`) are now temporary and deleted automatically after each build.
- **Single-Instance Protection**: Implemented a Windows Mutex and a professional "Application already running" prompt to prevent duplicate processes.
- **Self-Healing Assets**: Automated extraction of icons and resources from the binary if they are missing from the disk.

### Changed
- **Unified Config**: Migrated to a single `hoyo_cookie.json` format for all game credentials.
- **Performance**: Optimized the refresh loop to perform parallel polling for all active games.
- **Repository Cleanliness**: Updated `.gitignore` to exclude build artifacts and machine-specific files.

### Fixed
- **Legacy Synchronization**: Fully restored and synchronized `resin.exe`, `stamina.exe`, and `charge.exe` to use the new unified backend logic.
- **Icon Embedding**: Resolved build-context issues by pivoting to directory-based builds for perfect resource discovery.

## [v0.0.1] - Initial Release
- Individual monitors for Genshin, HSR, and ZZZ.
