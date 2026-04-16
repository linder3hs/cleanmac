package cmd

import (
	"fmt"
	"os"

	"github.com/linder3hs/cleanmac/internal/config"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	dryRun  bool
	noColor bool

	cfg *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "cleanmac",
	Short: "Interactive terminal cleaner for macOS",
	Long:  "CleanMac — a fast, interactive terminal replacement for CleanMyMac.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		path := cfgFile
		if path == "" {
			path = config.DefaultPath()
		}

		loaded, err := config.Load(path)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if dryRun {
			loaded.DryRun = true
		}

		cfg = loaded
		return nil
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ~/.config/cleanmac/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "n", false, "show what would be deleted, never delete")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable color output")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(diskCmd)
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("cleanmac v0.1.0")
	},
}
