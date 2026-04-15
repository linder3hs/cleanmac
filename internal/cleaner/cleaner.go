package cleaner

import (
	"fmt"
	"os"

	"github.com/linder3hs/cleanmac/internal/config"
	"github.com/linder3hs/cleanmac/internal/scanner"
)

// Result holds the outcome of a clean operation.
type Result struct {
	Freed  int64
	Count  int
	Errors []error
}

// Clean deletes the given file entries, respecting dry-run and protected paths.
// progressFn is called after each successful deletion with bytes freed so far.
func Clean(entries []scanner.FileEntry, cfg *config.Config, progressFn func(freed int64, path string)) Result {
	result := Result{}

	for _, entry := range entries {
		if IsProtected(entry.Path, cfg) {
			result.Errors = append(result.Errors, fmt.Errorf("protected path skipped: %s", entry.Path))
			continue
		}

		if cfg.DryRun {
			result.Freed += entry.Size
			result.Count++
			if progressFn != nil {
				progressFn(result.Freed, entry.Path)
			}
			continue
		}

		if err := os.RemoveAll(entry.Path); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("remove %s: %w", entry.Path, err))
			continue
		}

		result.Freed += entry.Size
		result.Count++
		if progressFn != nil {
			progressFn(result.Freed, entry.Path)
		}
	}

	return result
}
