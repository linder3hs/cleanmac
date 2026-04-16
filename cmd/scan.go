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
	scanFiles    bool
	scanCategory string
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan and report disk usage by category",
	Long: `Scan all categories and show a summary table.

Use --files to see every individual file/folder that would be deleted.
Use --type to scope to a single category (caches, logs, dev, browser, large, trash, ios, mail).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(os.Stderr, "Scanning... (this may take a moment)")

		result := scanner.ScanAll(cfg)

		// Filter by category if requested.
		if scanCategory != "" {
			var filtered []scanner.CategoryResult
			for _, c := range result.Categories {
				if string(c.ID) == scanCategory {
					filtered = append(filtered, c)
				}
			}
			result.Categories = filtered
			result.TotalSize = 0
			for _, c := range filtered {
				result.TotalSize += c.TotalSize
			}
		}

		// Sort by size descending.
		sort.Slice(result.Categories, func(i, j int) bool {
			return result.Categories[i].TotalSize > result.Categories[j].TotalSize
		})

		if scanJSON {
			return printJSON(result, scanFiles)
		}

		printTable(result)

		if scanFiles {
			fmt.Println()
			printFileDetail(result)
		}

		return nil
	},
}

func init() {
	scanCmd.Flags().BoolVar(&scanJSON, "json", false, "output as JSON")
	scanCmd.Flags().BoolVarP(&scanFiles, "files", "f", false, "show individual files/folders per category")
	scanCmd.Flags().StringVar(&scanCategory, "type", "", "only scan one category: caches|logs|dev|browser|large|trash|ios|mail")
}

func printTable(result *scanner.ScanResult) {
	sizeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	safeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	pctStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	pctHighStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))

	colName := 26
	colSize := 10
	colPct := 5
	colFiles := 7

	header := fmt.Sprintf("  %-*s  %*s  %*s  %*s  %s",
		colName, "Category",
		colSize, "Size",
		colPct, "%",
		colFiles, "Files",
		"Risk",
	)
	sep := fmt.Sprintf("  %s  %s  %s  %s  %s",
		strings.Repeat("─", colName),
		strings.Repeat("─", colSize),
		strings.Repeat("─", colPct),
		strings.Repeat("─", colFiles),
		strings.Repeat("─", 8),
	)

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

		var pct float64
		if result.TotalSize > 0 {
			pct = float64(cat.TotalSize) / float64(result.TotalSize) * 100
		}
		pctStr := fmt.Sprintf("%*.0f%%", colPct-1, pct)
		var styledPct string
		if pct >= 50 {
			styledPct = pctHighStyle.Render(pctStr)
		} else {
			styledPct = pctStyle.Render(pctStr)
		}

		fileCount := fmt.Sprintf("%*d", colFiles, len(cat.Files))

		sizeStr := sizeStyle.Render(fmt.Sprintf("%*s", colSize, humanize.Bytes(cat.TotalSize)))
		row := fmt.Sprintf("  %-*s  %s  %s  %s  %s",
			colName, cat.DisplayName,
			sizeStr,
			styledPct,
			dimStyle.Render(fileCount),
			riskStr,
		)
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

func printFileDetail(result *scanner.ScanResult) {
	sizeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	catStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	zeroStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)

	for _, cat := range result.Categories {
		// Category header.
		riskTag := ""
		if cat.Risk == scanner.RiskWarning {
			riskTag = "  " + warnStyle.Render("⚠ review carefully")
		}
		fmt.Printf("\n%s%s\n", catStyle.Render("  "+cat.DisplayName), riskTag)
		fmt.Println(dimStyle.Render("  " + strings.Repeat("─", 60)))

		if len(cat.Files) == 0 {
			fmt.Println(zeroStyle.Render("    (nothing found)"))
			continue
		}

		// Sort files by size descending.
		files := make([]scanner.FileEntry, len(cat.Files))
		copy(files, cat.Files)
		sort.Slice(files, func(i, j int) bool {
			return files[i].Size > files[j].Size
		})

		colPath := 52
		for _, f := range files {
			name := f.Name
			if len([]rune(name)) > colPath {
				runes := []rune(name)
				name = "…" + string(runes[len(runes)-colPath+1:])
			}
			sizeStr := sizeStyle.Render(fmt.Sprintf("%10s", humanize.Bytes(f.Size)))
			fmt.Printf("    %s  %s\n", sizeStr, name)
		}

		fmt.Printf(dimStyle.Render("    Total: %s (%d items)\n"),
			humanize.Bytes(cat.TotalSize), len(cat.Files))
	}
}

// JSON types.

type jsonFile struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	Size  int64  `json:"size_bytes"`
	Human string `json:"size_human"`
}

type jsonCategory struct {
	ID    string     `json:"id"`
	Name  string     `json:"name"`
	Size  int64      `json:"size_bytes"`
	Human string     `json:"size_human"`
	Risk  string     `json:"risk"`
	Files []jsonFile `json:"files,omitempty"`
}

type jsonResult struct {
	Categories []jsonCategory `json:"categories"`
	Total      int64          `json:"total_bytes"`
	TotalHuman string         `json:"total_human"`
}

func printJSON(result *scanner.ScanResult, includeFiles bool) error {
	cats := make([]jsonCategory, 0, len(result.Categories))
	for _, c := range result.Categories {
		risk := "safe"
		if c.Risk == scanner.RiskWarning {
			risk = "warning"
		} else if c.Risk == scanner.RiskDanger {
			risk = "danger"
		}

		jc := jsonCategory{
			ID:    string(c.ID),
			Name:  c.DisplayName,
			Size:  c.TotalSize,
			Human: humanize.Bytes(c.TotalSize),
			Risk:  risk,
		}

		if includeFiles {
			for _, f := range c.Files {
				jc.Files = append(jc.Files, jsonFile{
					Path:  f.Path,
					Name:  f.Name,
					Size:  f.Size,
					Human: humanize.Bytes(f.Size),
				})
			}
		}

		cats = append(cats, jc)
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
