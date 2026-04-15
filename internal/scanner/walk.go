package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/karrick/godirwalk"
)

// godirwalkWalk walks path and calls fn for each regular file with its size.
func godirwalkWalk(path string, fn func(path string, size int64)) error {
	return godirwalk.Walk(path, &godirwalk.Options{
		Callback: func(p string, de *godirwalk.Dirent) error {
			if de.IsRegular() {
				info, err := os.Lstat(p)
				if err == nil {
					fn(p, info.Size())
				}
			}
			return nil
		},
		ErrorCallback: func(_ string, _ error) godirwalk.ErrorAction {
			return godirwalk.SkipNode
		},
		Unsorted: true,
	})
}

// dirSize returns the total size of a directory tree in one pass.
func dirSize(path string) int64 {
	var total int64
	_ = godirwalkWalk(path, func(_ string, size int64) {
		total += size
	})
	return total
}

// subDirSizes does ONE walk of root and returns FileEntry per immediate subdirectory.
// This is far faster than calling dirSize() per subdir (N walks → 1 walk).
func subDirSizes(root string) []FileEntry {
	if _, err := os.Stat(root); err != nil {
		return nil
	}

	// Map top-level child name → accumulated size.
	sizes := map[string]int64{}

	godirwalk.Walk(root, &godirwalk.Options{ //nolint:errcheck
		Callback: func(path string, de *godirwalk.Dirent) error {
			if path == root {
				return nil
			}
			// Determine top-level child of root.
			rel := strings.TrimPrefix(path, root+string(filepath.Separator))
			if rel == "" {
				return nil
			}
			topLevel := strings.SplitN(rel, string(filepath.Separator), 2)[0]

			if de.IsRegular() {
				info, err := os.Lstat(path)
				if err == nil {
					sizes[topLevel] += info.Size()
				}
			}
			return nil
		},
		ErrorCallback: func(_ string, _ error) godirwalk.ErrorAction {
			return godirwalk.SkipNode
		},
		Unsorted: true,
	})

	var results []FileEntry
	for name, size := range sizes {
		if size == 0 {
			continue
		}
		results = append(results, FileEntry{
			Path: filepath.Join(root, name),
			Name: name,
			Size: size,
		})
	}
	return results
}

// parallelSubDirSizes reads top-level children of root and computes each
// subtree size concurrently. Faster than subDirSizes for deep, large dirs.
func parallelSubDirSizes(root string) []FileEntry {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}

	type result struct {
		entry FileEntry
	}

	sem := make(chan struct{}, runtime.NumCPU()*2)
	results := make([]result, len(entries))
	var wg sync.WaitGroup

	for i, e := range entries {
		if !e.IsDir() {
			continue
		}
		wg.Add(1)
		idx := i
		name := e.Name()
		full := filepath.Join(root, name)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			size := dirSize(full)
			if size > 0 {
				results[idx] = result{FileEntry{Path: full, Name: name, Size: size}}
			}
		}()
	}
	wg.Wait()

	var out []FileEntry
	for _, r := range results {
		if r.entry.Size > 0 {
			out = append(out, r.entry)
		}
	}
	return out
}

// totalSize sums sizes of all FileEntry values.
func totalSize(entries []FileEntry) int64 {
	var n int64
	for _, e := range entries {
		n += e.Size
	}
	return n
}
