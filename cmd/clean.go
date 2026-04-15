package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/linder3hs/cleanmac/internal/tui"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Interactively select and clean categories",
	RunE: func(cmd *cobra.Command, args []string) error {
		m := tui.New(cfg)
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("tui: %w", err)
		}
		return nil
	},
}
