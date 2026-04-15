package scanner

import (
	"os"
	"path/filepath"

	"github.com/linder3hs/cleanmac/internal/config"
)

// ScanIOSBackups finds iOS device backups.
func ScanIOSBackups(cfg *config.Config) CategoryResult {
	home, _ := os.UserHomeDir()
	backupRoot := filepath.Join(home, "Library", "Application Support", "MobileSync", "Backup")

	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		return CategoryResult{
			ID:          CategoryIOSBackups,
			DisplayName: "iOS Backups",
			Risk:        RiskSafe,
		}
	}

	var files []FileEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		full := filepath.Join(backupRoot, e.Name())
		size := dirSize(full)
		if size == 0 {
			continue
		}
		// Backup dirs are named by UDID — show that as the name.
		files = append(files, FileEntry{
			Path: full,
			Name: "Backup: " + e.Name()[:8] + "…",
			Size: size,
		})
	}

	return CategoryResult{
		ID:          CategoryIOSBackups,
		DisplayName: "iOS Backups",
		TotalSize:   totalSize(files),
		Files:       files,
		Risk:        RiskWarning,
	}
}
