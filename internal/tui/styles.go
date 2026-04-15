package tui

import "github.com/charmbracelet/lipgloss"

var (
	ColorGreen  = lipgloss.Color("10")
	ColorYellow = lipgloss.Color("11")
	ColorRed    = lipgloss.Color("9")
	ColorCyan   = lipgloss.Color("14")
	ColorGray   = lipgloss.Color("8")
	ColorWhite  = lipgloss.Color("15")
	ColorBlue   = lipgloss.Color("62")

	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBlue).
			Padding(0, 1)

	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite)

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBlue)

	CheckedStyle = lipgloss.NewStyle().
			Foreground(ColorGreen)

	UncheckedStyle = lipgloss.NewStyle().
			Foreground(ColorGray)

	SizeStyle = lipgloss.NewStyle().
			Foreground(ColorCyan)

	DangerStyle = lipgloss.NewStyle().
			Foreground(ColorRed)

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorYellow)

	SafeStyle = lipgloss.NewStyle().
			Foreground(ColorGreen)

	StatusBarStyle = lipgloss.NewStyle().
			Background(ColorBlue).
			Foreground(lipgloss.Color("0")).
			Padding(0, 1)

	DimStyle = lipgloss.NewStyle().
			Foreground(ColorGray)

	BoldStyle = lipgloss.NewStyle().
			Bold(true)

	SelectedRowStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("236")).
				Foreground(ColorWhite)
)
