package config

import (
	"os"
	"path/filepath"
)

// systemProtectedPaths returns paths that can NEVER be deleted, regardless of config.
// Symlinks are resolved before comparison in safe.go.
func systemProtectedPaths() []string {
	home := os.Getenv("HOME")
	return []string{
		"/",
		"/System",
		"/usr",
		"/bin",
		"/sbin",
		"/private/etc",
		"/private/var",
		"/Applications",
		"/Library",
		home,
		filepath.Join(home, ".ssh"),
		filepath.Join(home, ".gnupg"),
		filepath.Join(home, ".config"),
		filepath.Join(home, "Documents"),
		filepath.Join(home, "Desktop"),
		filepath.Join(home, "Downloads"),
		filepath.Join(home, "Movies"),
		filepath.Join(home, "Music"),
		filepath.Join(home, "Pictures"),
	}
}

// DefaultConfig returns a Config with safe defaults.
func DefaultConfig() *Config {
	return &Config{
		Version:              1,
		LargeFileThresholdMB: 100,
		DryRun:               false,
		ProtectedPaths:       []string{},
		ExcludePatterns:      []string{".git/**"},
	}
}
