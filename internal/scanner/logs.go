package scanner

import (
	"os"
	"path/filepath"

	"github.com/linder3hs/cleanmac/internal/config"
)

// ScanLogs scans log directories.
func ScanLogs(cfg *config.Config) CategoryResult {
	home, _ := os.UserHomeDir()

	roots := []string{
		filepath.Join(home, "Library", "Logs"),
		"/var/log",
	}

	var files []FileEntry
	for _, root := range roots {
		files = append(files, subDirSizes(root)...)
	}

	// Also include individual log files at root of ~/Library/Logs.
	userLogsRoot := filepath.Join(home, "Library", "Logs")
	if entries, err := os.ReadDir(userLogsRoot); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			full := filepath.Join(userLogsRoot, e.Name())
			info, err := e.Info()
			if err != nil || info.Size() == 0 {
				continue
			}
			files = append(files, FileEntry{
				Path: full,
				Name: e.Name(),
				Size: info.Size(),
			})
		}
	}

	return CategoryResult{
		ID:          CategoryLogs,
		DisplayName: "Log Files",
		TotalSize:   totalSize(files),
		Files:       files,
		Risk:        RiskSafe,
	}
}
