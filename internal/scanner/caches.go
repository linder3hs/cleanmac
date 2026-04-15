package scanner

import (
	"os"
	"path/filepath"

	"github.com/linder3hs/cleanmac/internal/config"
)

// ScanCaches scans system and user cache directories.
func ScanCaches(cfg *config.Config) CategoryResult {
	home, _ := os.UserHomeDir()

	roots := []string{
		filepath.Join(home, "Library", "Caches"),
		filepath.Join(home, "Library", "Application Support", "CrashReporter"),
		filepath.Join(home, "Library", "Logs", "DiagnosticReports"),
		"/private/tmp",
		os.TempDir(),
	}

	var files []FileEntry
	for _, root := range roots {
		files = append(files, parallelSubDirSizes(root)...)
	}

	return CategoryResult{
		ID:          CategoryCaches,
		DisplayName: "System & App Caches",
		TotalSize:   totalSize(files),
		Files:       files,
		Risk:        RiskSafe,
	}
}
