package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/linder3hs/cleanmac/internal/config"
)

// ScanTrash scans ~/.Trash and external volume trash directories.
func ScanTrash(cfg *config.Config) CategoryResult {
	home, _ := os.UserHomeDir()

	var files []FileEntry

	// Main trash.
	mainTrash := filepath.Join(home, ".Trash")
	if entries, err := os.ReadDir(mainTrash); err == nil {
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".") {
				continue
			}
			full := filepath.Join(mainTrash, e.Name())
			var size int64
			if e.IsDir() {
				size = dirSize(full)
			} else {
				if info, err := e.Info(); err == nil {
					size = info.Size()
				}
			}
			if size == 0 {
				continue
			}
			files = append(files, FileEntry{
				Path: full,
				Name: e.Name(),
				Size: size,
			})
		}
	}

	// External volume trash directories: /Volumes/*/.Trashes/<uid>
	uid := os.Getuid()
	volumesDir := "/Volumes"
	if vols, err := os.ReadDir(volumesDir); err == nil {
		for _, vol := range vols {
			if !vol.IsDir() {
				continue
			}
			trashDir := filepath.Join(volumesDir, vol.Name(), ".Trashes", fmt.Sprint(uid))
			if entries, err := os.ReadDir(trashDir); err == nil {
				for _, e := range entries {
					full := filepath.Join(trashDir, e.Name())
					var size int64
					if e.IsDir() {
						size = dirSize(full)
					} else {
						if info, err := e.Info(); err == nil {
							size = info.Size()
						}
					}
					if size == 0 {
						continue
					}
					files = append(files, FileEntry{
						Path: full,
						Name: vol.Name() + " Trash: " + e.Name(),
						Size: size,
					})
				}
			}
		}
	}

	return CategoryResult{
		ID:          CategoryTrash,
		DisplayName: "Trash",
		TotalSize:   totalSize(files),
		Files:       files,
		Risk:        RiskSafe,
	}
}
