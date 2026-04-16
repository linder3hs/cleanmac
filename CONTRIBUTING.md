# Contributing to CleanMac

Thanks for the interest. PRs and issues welcome.

## Getting started

```bash
git clone https://github.com/linder3hs/cleanmac.git
cd cleanmac
go run . init
```

Requires Go 1.25+, macOS.

## Development workflow

```bash
make build         # build binary
make test          # run tests
go vet ./...       # static checks
go run . init      # run without installing
```

Always run `go vet ./... && go test ./...` before pushing.

## Project layout

```
cmd/         cobra subcommands
internal/
  config/    yaml + protected paths
  scanner/   concurrent category scanners
  cleaner/   deletion + safety check
  diskinfo/  statfs + bars
  humanize/  byte formatting
  tui/       bubbletea model + views
```

State machine:
`Menu → Scanning → Select → Files → (Confirm | FileConfirm) → Cleaning → Summary → Menu`

## Adding a new scanner

1. Create `internal/scanner/scan_<name>.go` with a `func Scan<Name>(*config.Config) CategoryResult`
2. Register it in the `scanners` slice in `scanner/scanner.go` (both `ScanAll` and `ScanAllStream`)
3. Add display name to `allCategoryNames` in `internal/tui/model.go`
4. Set appropriate `RiskLevel` (safe / warning / danger)

## Safety rules (NEVER violate)

- Never delete a path matched by `cleaner.IsProtected`
- Never bypass `cfg.DryRun`
- Always resolve symlinks before path comparison
- New scanners must verify deletion targets are user-owned cache/log/artifact data — not user data

## Commit messages

Conventional-ish, short imperative:

```
feat(tui): add single-file delete
fix(cleaner): don't protect $HOME root
docs: update install instructions
```

## Pull requests

- One feature per PR
- Include a short description of what + why
- If touching the TUI, attach a screenshot or asciicast
- Bump version in `cmd/root.go` only if maintainer asks

## Reporting bugs

Open an issue with:

- macOS version
- Output of `cleanmac version`
- Reproduction steps
- Expected vs actual

## Code of conduct

See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
