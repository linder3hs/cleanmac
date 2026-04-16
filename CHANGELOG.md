# Changelog

All notable changes to this project documented here.
Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
versioning follows [SemVer](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-04-16

Initial public release.

### Added

- Interactive hub menu (`cleanmac init`) — arrow-key navigation
- Persistent disk-usage widget on every TUI screen (root volume, color by usage)
- Single-file delete from category file list (`x` / `delete` / `backspace`)
- Standalone `cleanmac disk` command with colored usage bars
- Streaming concurrent scanners — categories appear as they finish
- Category multi-select with `space` / `a`
- Per-category risk levels (safe / review / danger)
- Dry-run mode (`-n` / `--dry-run`)
- YAML config at `~/.config/cleanmac/config.yaml` for user-defined protected paths
- Hardcoded system-protected paths (Documents, Desktop, .ssh, Keychains, Mail, …)
- Symlink resolution before path comparison
- Per-category breakdown in cleanup summary

### Scanners included

- System & App Caches
- Log Files
- Browser Caches (Chrome, Safari, Firefox, Brave)
- Dev Artifacts (`node_modules`, `.next`, `target`, `__pycache__`)
- Large Files (configurable threshold)
- Trash
- iOS Backups
- Mail Downloads

[Unreleased]: https://github.com/linder3hs/cleanmac/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/linder3hs/cleanmac/releases/tag/v0.1.0
