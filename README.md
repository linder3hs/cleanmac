# CleanMac

Fast, interactive terminal disk cleaner for macOS. Built in Go with a Bubbletea TUI.

Think CleanMyMac, but free, open source, scriptable, and lives in your terminal.

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go) ![License](https://img.shields.io/badge/license-MIT-blue) ![Platform](https://img.shields.io/badge/platform-macOS-lightgrey)

---

## Features

- **Interactive hub menu** (`cleanmac init`) — arrow-key navigation, no flags to memorize
- **Live disk widget** — root volume usage shown in every screen, updates after each delete
- **Multi-category scan** — caches, logs, browser data, dev artifacts, large files, trash, iOS backups, mail
- **Granular control** — delete a single file, a whole category, or multiple selected categories
- **Dry-run mode** — preview every byte before touching disk
- **Safe by default** — system paths and personal data (Documents, Desktop, .ssh, Keychains, Mail, …) hardcoded as protected
- **Standalone disk view** — quick `df`-style table with colored usage bars
- **Streaming results** — categories appear as they finish scanning

---

## Install

### Homebrew (recommended)

```bash
brew tap linder3hs/cleanmac
brew install cleanmac
```

### From source

```bash
git clone https://github.com/linder3hs/cleanmac.git
cd cleanmac
make install         # builds and copies to /usr/local/bin
```

Or just:

```bash
go install github.com/linder3hs/cleanmac@latest
```

### Universal binary (Intel + Apple Silicon)

```bash
make build-universal # output: dist/cleanmac
```

---

## Usage

### Interactive mode (recommended)

```bash
cleanmac init
```

Opens the hub menu. Pick **Scan & Clean** or **Disk Space**, do your thing, return to the menu. `q` quits.

### Direct commands

| Command | What it does |
|---|---|
| `cleanmac init` | Interactive hub menu |
| `cleanmac scan` | Scan all categories, print summary |
| `cleanmac clean` | Scan + interactive cleaner (no menu loop) |
| `cleanmac disk` | Show disk usage for all volumes |
| `cleanmac version` | Print version |

### Flags (global)

| Flag | Description |
|---|---|
| `-n`, `--dry-run` | Show what would be deleted, never delete |
| `--config PATH` | Config file (default `~/.config/cleanmac/config.yaml`) |
| `--no-color` | Disable colored output |

### Examples

```bash
cleanmac init                    # full interactive
cleanmac clean --dry-run         # preview only
cleanmac disk                    # quick disk check
cleanmac --config ./my.yaml scan # custom config
```

---

## Keybindings (TUI)

### Menu

| Key | Action |
|---|---|
| `↑` / `↓` / `j` / `k` | Navigate |
| `enter` | Select |
| `s` | Scan & Clean (shortcut) |
| `d` | Disk Space (shortcut) |
| `q` / `ctrl+c` | Quit |

### Category select

| Key | Action |
|---|---|
| `↑` / `↓` / `j` / `k` | Navigate |
| `space` | Toggle select |
| `a` | Toggle all |
| `enter` | View files in category |
| `c` | Clean current category |
| `d` | Clean all selected |
| `esc` / `q` | Back / quit |

### File list

| Key | Action |
|---|---|
| `↑` / `↓` / `j` / `k` | Scroll |
| `g` / `G` | Top / bottom |
| `x` / `delete` | Delete file under cursor |
| `c` | Clean entire category |
| `esc` | Back |

---

## Categories scanned

| Category | Path examples | Risk |
|---|---|---|
| System & App Caches | `~/Library/Caches/*` | safe |
| Log Files | `~/Library/Logs/*`, `/var/log/*` | safe |
| Browser Caches | Chrome, Safari, Firefox, Brave caches | safe |
| Dev Artifacts | `node_modules`, `.next`, `target`, `__pycache__` | safe |
| Large Files | Files > 100 MB (configurable) | review |
| Trash | `~/.Trash` | safe |
| iOS Backups | `~/Library/Application Support/MobileSync/Backup` | review |
| Mail Downloads | `~/Library/Containers/com.apple.mail/.../Downloads` | safe |

---

## Configuration

Default location: `~/.config/cleanmac/config.yaml`

```yaml
version: 1
large_file_threshold_mb: 100
dry_run: false
protected_paths:
  - ~/Projects/important
  - /opt/critical-data
exclude_patterns:
  - .git/**
  - node_modules/important/**
```

### Always-protected paths (cannot be overridden)

`/`, `/System`, `/usr`, `/bin`, `/sbin`, `/private/etc`, `/private/var`, `/Applications`, `/Library`, `~/Documents`, `~/Desktop`, `~/Downloads`, `~/Movies`, `~/Music`, `~/Pictures`, `~/.ssh`, `~/.gnupg`, `~/.config`, `~/Library/Keychains`, `~/Library/Mail`, `~/Library/Messages`.

---

## Safety model

1. Paths shorter than 3 components are blocked (e.g. `/Users/foo`)
2. Symlinks resolved before comparison — no escape via `ln -s /System ~/cache`
3. System-protected list hardcoded in [`internal/config/defaults.go`](internal/config/defaults.go)
4. Dry-run never touches disk
5. Per-file `os.RemoveAll` errors surface in summary, never silent

---

## Architecture

```
cmd/                  cobra subcommands (init, scan, clean, disk)
internal/
  config/             yaml config + protected-paths defaults
  scanner/            concurrent category scanners (channel-streamed)
  cleaner/            deletion + protection check
  diskinfo/           statfs wrapper, usage bars
  humanize/           byte formatting
  tui/                bubbletea model, state machine, views
main.go
```

State machine: `Menu → Scanning → Select → Files → (Confirm | FileConfirm) → Cleaning → Summary → Menu`.

---

## Build & develop

```bash
make build           # binary in ./cleanmac
make test            # go test ./...
make bench           # micro-benchmarks
go run . init        # run without installing
```

Requirements: Go 1.25+, macOS (uses `syscall.Statfs`).

---

## Contributing

PRs welcome. Suggested first issues:

- Linux support (replace `Statfs_t` darwin fields)
- More scanners (Docker images, Xcode DerivedData, Android SDK)
- Background scheduled cleans
- Settings TUI screen for editing config in-app

Run `go vet ./... && go test ./...` before submitting.

---

## License

MIT — see [LICENSE](LICENSE).

---

## Credits

Built on:

- [Cobra](https://github.com/spf13/cobra) — CLI framework
- [Bubbletea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) — terminal styling
- [Bubbles](https://github.com/charmbracelet/bubbles) — TUI components

Inspired by [CleanMyMac](https://macpaw.com/cleanmymac), [ncdu](https://dev.yorhel.nl/ncdu), [diskonaut](https://github.com/imsnif/diskonaut).
