package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds user-configurable settings loaded from ~/.config/cleanmac/config.yaml.
type Config struct {
	Version              int      `yaml:"version"`
	ProtectedPaths       []string `yaml:"protected_paths"`
	CustomInclude        []string `yaml:"custom_include"`
	LargeFileThresholdMB int64    `yaml:"large_file_threshold_mb"`
	DryRun               bool     `yaml:"dry_run"`
	ExcludePatterns      []string `yaml:"exclude_patterns"`

	// SystemProtected is populated at runtime from defaults.go — not from yaml.
	SystemProtected []string `yaml:"-"`
}

// DefaultPath returns the default config file path.
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "cleanmac", "config.yaml")
}

// Load reads the config file at path. If the file doesn't exist, returns defaults.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		cfg.SystemProtected = systemProtectedPaths()
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	cfg.SystemProtected = systemProtectedPaths()
	return cfg, nil
}

// Save writes the config to disk, creating parent dirs if needed.
func Save(cfg *Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
