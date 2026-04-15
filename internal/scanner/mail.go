package scanner

import (
	"os"
	"path/filepath"

	"github.com/linder3hs/cleanmac/internal/config"
)

// ScanMail scans Mail app download/attachment storage.
func ScanMail(cfg *config.Config) CategoryResult {
	home, _ := os.UserHomeDir()

	roots := []string{
		filepath.Join(home, "Library", "Containers", "com.apple.mail", "Data", "Library", "Mail Downloads"),
		filepath.Join(home, "Library", "Mail"),
	}

	var files []FileEntry
	for _, root := range roots {
		if _, err := os.Stat(root); err != nil {
			continue
		}
		size := dirSize(root)
		if size == 0 {
			continue
		}
		files = append(files, FileEntry{
			Path: root,
			Name: filepath.Base(root),
			Size: size,
		})
	}

	return CategoryResult{
		ID:          CategoryMail,
		DisplayName: "Mail Downloads",
		TotalSize:   totalSize(files),
		Files:       files,
		Risk:        RiskWarning,
	}
}
