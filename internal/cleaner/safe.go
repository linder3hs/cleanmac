package cleaner

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/linder3hs/cleanmac/internal/config"
)

// IsProtected returns true if path must not be deleted.
// Resolves symlinks before comparison to prevent symlink escape.
func IsProtected(path string, cfg *config.Config) bool {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		// Can't resolve → treat as protected.
		resolved = filepath.Clean(path)
	}

	// Minimum depth: at least 3 non-empty components (e.g. /Users/foo/something).
	parts := strings.Split(resolved, string(filepath.Separator))
	depth := 0
	for _, p := range parts {
		if p != "" {
			depth++
		}
	}
	if depth < 3 {
		return true
	}

	// System-protected paths (hardcoded, cannot be overridden by user config).
	for _, protected := range cfg.SystemProtected {
		if isUnderPath(resolved, protected) {
			return true
		}
	}

	// User-configured protected paths.
	for _, protected := range cfg.ProtectedPaths {
		expanded := expandHome(protected)
		if isUnderPath(resolved, expanded) {
			return true
		}
	}

	return false
}

// isUnderPath returns true if target equals base or is directly under it.
func isUnderPath(target, base string) bool {
	base = filepath.Clean(base)
	resolved, err := filepath.EvalSymlinks(base)
	if err == nil {
		base = resolved
	}
	return target == base || strings.HasPrefix(target, base+string(filepath.Separator))
}

// expandHome replaces a leading ~/ with the user's home directory.
func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
