package scanner

import (
	"os"
	"path/filepath"

	"github.com/linder3hs/cleanmac/internal/config"
)

type browserTarget struct {
	name string
	// Specific cache subdirs (not parent dirs — avoids walking entire profile).
	cachePaths []string
}

func buildBrowserTargets(home string) []browserTarget {
	caches := filepath.Join(home, "Library", "Caches")
	return []browserTarget{
		{
			name: "Chrome",
			cachePaths: []string{
				filepath.Join(caches, "Google", "Chrome", "Default", "Cache"),
				filepath.Join(caches, "Google", "Chrome", "Default", "Code Cache"),
				filepath.Join(caches, "Google", "Chrome", "Default", "GPUCache"),
			},
		},
		{
			name: "Arc",
			cachePaths: []string{
				filepath.Join(caches, "Arc", "User Data", "Default", "Cache"),
				filepath.Join(caches, "Arc", "User Data", "Default", "Code Cache"),
			},
		},
		{
			name: "Firefox",
			cachePaths: []string{
				filepath.Join(home, "Library", "Caches", "Firefox", "Profiles"),
			},
		},
		{
			name: "Safari",
			cachePaths: []string{
				filepath.Join(caches, "com.apple.Safari"),
			},
		},
		{
			name: "Brave",
			cachePaths: []string{
				filepath.Join(caches, "BraveSoftware", "Brave-Browser", "Default", "Cache"),
				filepath.Join(caches, "BraveSoftware", "Brave-Browser", "Default", "Code Cache"),
			},
		},
		{
			name: "Edge",
			cachePaths: []string{
				filepath.Join(caches, "Microsoft Edge", "Default", "Cache"),
			},
		},
	}
}

// ScanBrowser scans browser cache directories.
func ScanBrowser(cfg *config.Config) CategoryResult {
	home, _ := os.UserHomeDir()
	targets := buildBrowserTargets(home)

	var files []FileEntry

	for _, browser := range targets {
		var browserSize int64

		for _, cachePath := range browser.cachePaths {
			if _, err := os.Stat(cachePath); err != nil {
				continue
			}
			size := dirSize(cachePath)
			if size == 0 {
				continue
			}
			browserSize += size
			files = append(files, FileEntry{
				Path: cachePath,
				Name: browser.name + " (" + filepath.Base(cachePath) + ")",
				Size: size,
			})
		}
		_ = browserSize
	}

	return CategoryResult{
		ID:          CategoryBrowser,
		DisplayName: "Browser Caches",
		TotalSize:   totalSize(files),
		Files:       files,
		Risk:        RiskSafe,
	}
}
