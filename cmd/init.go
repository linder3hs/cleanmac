package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/linder3hs/cleanmac/internal/tui"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Open the interactive hub menu",
	Long:  "Launch the CleanMac TUI starting at the main menu, where you can choose between scanning, viewing disk space, and more.",
	RunE: func(cmd *cobra.Command, args []string) error {
		m := tui.NewWithMenu(cfg)
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("tui: %w", err)
		}
		return nil
	},
}
