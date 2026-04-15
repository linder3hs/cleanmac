//go:build bench

package cmd

import (
	"fmt"
	"time"

	"github.com/linder3hs/cleanmac/internal/humanize"
	"github.com/linder3hs/cleanmac/internal/scanner"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(benchCmd)
}

var benchCmd = &cobra.Command{
	Use:    "bench",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		fns := []struct {
			name string
			fn   func() scanner.CategoryResult
		}{
			{"caches", func() scanner.CategoryResult { return scanner.ScanCaches(cfg) }},
			{"logs", func() scanner.CategoryResult { return scanner.ScanLogs(cfg) }},
			{"browser", func() scanner.CategoryResult { return scanner.ScanBrowser(cfg) }},
			{"trash", func() scanner.CategoryResult { return scanner.ScanTrash(cfg) }},
			{"iosbackups", func() scanner.CategoryResult { return scanner.ScanIOSBackups(cfg) }},
			{"mail", func() scanner.CategoryResult { return scanner.ScanMail(cfg) }},
			{"devartifacts", func() scanner.CategoryResult { return scanner.ScanDevArtifacts(cfg) }},
			{"largefiles", func() scanner.CategoryResult { return scanner.ScanLargeFiles(cfg) }},
		}

		for _, f := range fns {
			t := time.Now()
			r := f.fn()
			elapsed := time.Since(t)
			fmt.Printf("%-16s %10s  %v\n", f.name, humanize.Bytes(r.TotalSize), elapsed.Round(time.Millisecond))
		}
		return nil
	},
}
