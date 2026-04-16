// Package diskinfo provides disk space gathering for darwin filesystems.
package diskinfo

import (
	"os"
	"strings"
	"syscall"
)

// Mount represents a mounted volume's space usage.
type Mount struct {
	Path  string
	Total int64
	Used  int64
	Free  int64
}

// Pct returns the used percentage (0-100).
func (m Mount) Pct() float64 {
	if m.Total <= 0 {
		return 0
	}
	return float64(m.Used) / float64(m.Total) * 100
}

func statfsInfo(path string) (Mount, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return Mount{}, err
	}
	total := int64(stat.Blocks) * int64(stat.Bsize)
	free := int64(stat.Bavail) * int64(stat.Bsize)
	used := total - int64(stat.Bfree)*int64(stat.Bsize)
	return Mount{Path: path, Total: total, Used: used, Free: free}, nil
}

// Gather returns all relevant mounted volumes. When includeAll is false,
// pseudo/tiny filesystems (<1 GB) are filtered out.
func Gather(includeAll bool) ([]Mount, error) {
	paths := []string{"/"}

	if entries, err := os.ReadDir("/Volumes"); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				paths = append(paths, "/Volumes/"+e.Name())
			}
		}
	}

	var result []Mount
	seen := map[int32]bool{}

	for _, p := range paths {
		info, err := statfsInfo(p)
		if err != nil {
			continue
		}
		var stat syscall.Statfs_t
		if syscall.Statfs(p, &stat) == nil {
			if seen[stat.Fsid.Val[0]] {
				continue
			}
			seen[stat.Fsid.Val[0]] = true
		}
		if !includeAll && info.Total < 1<<30 {
			continue
		}
		result = append(result, info)
	}

	return result, nil
}

// UsageBar returns a width-character bar string filled per pct.
func UsageBar(pct float64, width int) string {
	filled := int(pct / 100.0 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}
