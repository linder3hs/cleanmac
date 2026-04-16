package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/linder3hs/cleanmac/internal/diskinfo"
	"github.com/linder3hs/cleanmac/internal/humanize"
	"github.com/spf13/cobra"
)

var diskAll bool

var diskCmd = &cobra.Command{
	Use:   "disk",
	Short: "Show current disk space usage",
	Long:  `Show disk space for mounted volumes with a visual usage bar.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mounts, err := diskinfo.Gather(diskAll)
		if err != nil {
			return err
		}
		printDiskTable(mounts)
		return nil
	},
}

func init() {
	diskCmd.Flags().BoolVar(&diskAll, "all", false, "show all mounted volumes")
}

func barStyle(pct float64) lipgloss.Style {
	switch {
	case pct >= 90:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9")) // red
	case pct >= 70:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // yellow
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
	}
}

func printDiskTable(mounts []diskinfo.Mount) {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	sizeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	pctStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))

	colMount := 22
	colSize := 10
	barWidth := 12

	header := fmt.Sprintf("  %-*s  %*s  %*s  %*s  %-*s  %s",
		colMount, "Volume",
		colSize, "Total",
		colSize, "Used",
		colSize, "Free",
		barWidth+2, "Usage",
		"",
	)
	sep := fmt.Sprintf("  %s  %s  %s  %s  %s",
		strings.Repeat("─", colMount),
		strings.Repeat("─", colSize),
		strings.Repeat("─", colSize),
		strings.Repeat("─", colSize),
		strings.Repeat("─", barWidth+8),
	)

	rows := []string{
		headerStyle.Render(header),
		dimStyle.Render(sep),
	}

	for _, d := range mounts {
		pct := d.Pct()

		bar := fmt.Sprintf("[%s]", diskinfo.UsageBar(pct, barWidth))
		styledBar := barStyle(pct).Render(bar)
		pctStr := pctStyle.Render(fmt.Sprintf("%3.0f%%", pct))

		mount := d.Path
		if len([]rune(mount)) > colMount {
			runes := []rune(mount)
			mount = "…" + string(runes[len(runes)-colMount+1:])
		}

		row := fmt.Sprintf("  %-*s  %s  %s  %s  %s %s",
			colMount, mount,
			sizeStyle.Render(fmt.Sprintf("%*s", colSize, humanize.Bytes(d.Total))),
			sizeStyle.Render(fmt.Sprintf("%*s", colSize, humanize.Bytes(d.Used))),
			sizeStyle.Render(fmt.Sprintf("%*s", colSize, humanize.Bytes(d.Free))),
			styledBar,
			pctStr,
		)
		rows = append(rows, row)
	}

	rows = append(rows, dimStyle.Render(sep))
	fmt.Println(strings.Join(rows, "\n"))
}
