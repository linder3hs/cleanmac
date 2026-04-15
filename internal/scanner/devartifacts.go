package scanner

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/karrick/godirwalk"
	"github.com/linder3hs/cleanmac/internal/config"
)

type devTarget struct {
	searchName  string   // directory name to match (e.g. "node_modules")
	searchRoots []string // where to look
	maxDepth    int      // how deep to search for the dir
	displayName string   // human-readable label
}

func buildDevTargets(home string) []devTarget {
	return []devTarget{
		{
			searchName: "node_modules",
			searchRoots: func() []string {
				roots := []string{}
				// Only look in known dev directories, not entire home.
				for _, d := range []string{"dev", "projects", "code", "src", "work", "repos"} {
					p := filepath.Join(home, d)
					if _, err := os.Stat(p); err == nil {
						roots = append(roots, p)
					}
				}
				if len(roots) == 0 {
					roots = []string{home}
				}
				return roots
			}(),
			maxDepth:    4,
			displayName: "node_modules",
		},
		{
			searchName:  "DerivedData",
			searchRoots: []string{filepath.Join(home, "Library", "Developer", "Xcode")},
			maxDepth:    1,
			displayName: "Xcode DerivedData",
		},
		{
			searchName:  "registry",
			searchRoots: []string{filepath.Join(home, ".cargo")},
			maxDepth:    1,
			displayName: "Cargo Registry",
		},
		{
			searchName:  "git",
			searchRoots: []string{filepath.Join(home, ".cargo")},
			maxDepth:    1,
			displayName: "Cargo Git Cache",
		},
		{
			searchName:  "mod",
			searchRoots: []string{filepath.Join(home, "go", "pkg")},
			maxDepth:    1,
			displayName: "Go Module Cache",
		},
		{
			searchName:  "caches",
			searchRoots: []string{filepath.Join(home, ".gradle")},
			maxDepth:    1,
			displayName: "Gradle Cache",
		},
		{
			searchName:  "repository",
			searchRoots: []string{filepath.Join(home, ".m2")},
			maxDepth:    1,
			displayName: "Maven Repository",
		},
		{
			searchName:  "pip",
			searchRoots: []string{filepath.Join(home, "Library", "Caches")},
			maxDepth:    1,
			displayName: "Pip Cache",
		},
		{
			searchName:  "uv",
			searchRoots: []string{filepath.Join(home, "Library", "Caches")},
			maxDepth:    1,
			displayName: "uv Cache",
		},
	}
}

// ScanDevArtifacts finds development build artifacts (node_modules, DerivedData, etc).
func ScanDevArtifacts(cfg *config.Config) CategoryResult {
	home, _ := os.UserHomeDir()
	targets := buildDevTargets(home)

	var files []FileEntry
	seen := map[string]bool{}

	for _, t := range targets {
		for _, root := range t.searchRoots {
			if _, err := os.Stat(root); err != nil {
				continue
			}
			found := findDirs(root, t.searchName, t.maxDepth)
			for _, path := range found {
				if seen[path] {
					continue
				}
				seen[path] = true
				size := dirSize(path)
				if size == 0 {
					continue
				}
				files = append(files, FileEntry{
					Path: path,
					Name: t.displayName + ": " + shortenPath(path, home),
					Size: size,
				})
			}
		}
	}

	return CategoryResult{
		ID:          CategoryDevArtifacts,
		DisplayName: "Dev Artifacts",
		TotalSize:   totalSize(files),
		Files:       files,
		Risk:        RiskSafe,
	}
}

// findDirs searches root up to maxDepth for directories named target.
func findDirs(root, target string, maxDepth int) []string {
	var results []string

	godirwalk.Walk(root, &godirwalk.Options{ //nolint:errcheck
		Callback: func(path string, de *godirwalk.Dirent) error {
			if !de.IsDir() {
				return nil
			}
			depth := strings.Count(strings.TrimPrefix(path, root), string(filepath.Separator))
			if depth > maxDepth {
				return godirwalk.SkipThis
			}
			if de.Name() == target {
				results = append(results, path)
				return godirwalk.SkipThis // don't recurse into it
			}
			return nil
		},
		ErrorCallback: func(_ string, _ error) godirwalk.ErrorAction {
			return godirwalk.SkipNode
		},
		Unsorted: true,
	})

	return results
}

func shortenPath(path, home string) string {
	short := strings.TrimPrefix(path, home)
	if short != path {
		return "~" + short
	}
	return path
}
