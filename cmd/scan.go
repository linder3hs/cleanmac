package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/linder3hs/cleanmac/internal/humanize"
	"github.com/linder3hs/cleanmac/internal/scanner"
	"github.com/spf13/cobra"
)

var (
	scanJSON     bool
	scanCategory string
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan and report disk usage by category",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(os.Stderr, "Scanning... (this may take a moment)")

		result := scanner.ScanAll(cfg)

		// Sort by size descending.
		sort.Slice(result.Categories, func(i, j int) bool {
			return result.Categories[i].TotalSize > result.Categories[j].TotalSize
		})

		if scanJSON {
			return printJSON(result)
		}

		printTable(result)
		return nil
	},
}

func init() {
	scanCmd.Flags().BoolVar(&scanJSON, "json", false, "output as JSON")
	scanCmd.Flags().StringVar(&scanCategory, "type", "", "only scan specific category")
}

func printTable(result *scanner.ScanResult) {
	sizeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	safeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))

	colName := 26
	colSize := 12

	// Build rows as plain strings first, then style per-cell (avoids ANSI width issues).
	header := fmt.Sprintf("  %-*s  %*s  %s", colName, "Category", colSize, "Size", "Risk")
	sep := fmt.Sprintf("  %-*s  %*s  %s", colName, strings.Repeat("─", colName), colSize, strings.Repeat("─", colSize), strings.Repeat("─", 8))

	rows := []string{
		headerStyle.Render(header),
		dimStyle.Render(sep),
	}

	for _, cat := range result.Categories {
		var riskStr string
		switch cat.Risk {
		case scanner.RiskWarning:
			riskStr = warnStyle.Render("⚠ review ")
		case scanner.RiskDanger:
			riskStr = warnStyle.Render("✗ danger ")
		default:
			riskStr = safeStyle.Render("✓ safe   ")
		}
		sizeStr := sizeStyle.Render(fmt.Sprintf("%*s", colSize, humanize.Bytes(cat.TotalSize)))
		// Plain name padded, then styled size and risk appended.
		row := fmt.Sprintf("  %-*s  %s  %s", colName, cat.DisplayName, sizeStr, riskStr)
		rows = append(rows, row)
	}

	rows = append(rows, dimStyle.Render(sep))
	rows = append(rows, fmt.Sprintf("  %-*s  %s",
		colName, "Total",
		sizeStyle.Render(fmt.Sprintf("%*s", colSize, humanize.Bytes(result.TotalSize)))))

	if cfg.DryRun {
		rows = append(rows, "", dimStyle.Render("  dry-run mode — no files will be deleted"))
	}

	fmt.Println(strings.Join(rows, "\n"))
}

type jsonCategory struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Size  int64  `json:"size_bytes"`
	Human string `json:"size_human"`
	Risk  string `json:"risk"`
}

type jsonResult struct {
	Categories []jsonCategory `json:"categories"`
	Total      int64          `json:"total_bytes"`
	TotalHuman string         `json:"total_human"`
}

func printJSON(result *scanner.ScanResult) error {
	cats := make([]jsonCategory, 0, len(result.Categories))
	for _, c := range result.Categories {
		risk := "safe"
		if c.Risk == scanner.RiskWarning {
			risk = "warning"
		} else if c.Risk == scanner.RiskDanger {
			risk = "danger"
		}
		cats = append(cats, jsonCategory{
			ID:    string(c.ID),
			Name:  c.DisplayName,
			Size:  c.TotalSize,
			Human: humanize.Bytes(c.TotalSize),
			Risk:  risk,
		})
	}

	out := jsonResult{
		Categories: cats,
		Total:      result.TotalSize,
		TotalHuman: humanize.Bytes(result.TotalSize),
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
