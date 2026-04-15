package scanner

import (
	"container/heap"
	"os"
	"path/filepath"
	"strings"

	"github.com/karrick/godirwalk"
	"github.com/linder3hs/cleanmac/internal/config"
)

const defaultTopN = 200

// fileHeap is a min-heap of FileEntry by size (keeps the N largest).
type fileHeap []FileEntry

func (h fileHeap) Len() int            { return len(h) }
func (h fileHeap) Less(i, j int) bool  { return h[i].Size < h[j].Size }
func (h fileHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *fileHeap) Push(x interface{}) { *h = append(*h, x.(FileEntry)) }
func (h *fileHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// skipDirs are directory names to skip during large-file scanning (too deep / too many files).
var largeFileSkipDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	".gradle":      true,
	"DerivedData":  true,
	".cargo":       true,
}

// ScanLargeFiles finds the largest files under the home directory.
// Skips known artifact dirs (node_modules etc) to stay fast.
func ScanLargeFiles(cfg *config.Config) CategoryResult {
	home, _ := os.UserHomeDir()
	threshold := cfg.LargeFileThresholdMB * 1024 * 1024

	h := &fileHeap{}
	heap.Init(h)

	godirwalk.Walk(home, &godirwalk.Options{ //nolint:errcheck
		Callback: func(path string, de *godirwalk.Dirent) error {
			if de.IsDir() {
				name := de.Name()
				// Skip hidden dirs and known artifact dirs.
				if (strings.HasPrefix(name, ".") && path != home) || largeFileSkipDirs[name] {
					return filepath.SkipDir
				}
				return nil
			}
			if !de.IsRegular() {
				return nil
			}
			info, err := os.Lstat(path)
			if err != nil || info.Size() < threshold {
				return nil
			}
			entry := FileEntry{
				Path: path,
				Name: shortenPath(path, home),
				Size: info.Size(),
			}
			if h.Len() < defaultTopN {
				heap.Push(h, entry)
			} else if (*h)[0].Size < info.Size() {
				heap.Pop(h)
				heap.Push(h, entry)
			}
			return nil
		},
		ErrorCallback: func(_ string, _ error) godirwalk.ErrorAction {
			return godirwalk.SkipNode
		},
		Unsorted: true,
	})

	// Sort descending by size.
	files := make([]FileEntry, h.Len())
	for i := len(files) - 1; i >= 0; i-- {
		files[i] = heap.Pop(h).(FileEntry)
	}

	return CategoryResult{
		ID:          CategoryLargeFiles,
		DisplayName: "Large Files",
		TotalSize:   totalSize(files),
		Files:       files,
		Risk:        RiskWarning,
	}
}
