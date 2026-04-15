package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/linder3hs/cleanmac/internal/cleaner"
	"github.com/linder3hs/cleanmac/internal/config"
	"github.com/linder3hs/cleanmac/internal/humanize"
	"github.com/linder3hs/cleanmac/internal/scanner"
)

// State represents the current TUI screen.
type State int

const (
	StateScanning State = iota
	StateSelect
	StateExpanded
	StateConfirm
	StateCleaning
	StateSummary
)

// Messages.
type (
	scanResultMsg scanner.CategoryResult
	scanDoneMsg   struct{}
	cleanDoneMsg  cleaner.Result
)

// Model is the root Bubbletea model.
type Model struct {
	state  State
	cfg    *config.Config
	width  int
	height int

	spinner    spinner.Model
	scanCh     <-chan scanner.CategoryResult
	categories []scanner.CategoryResult

	cursor     int
	selected   map[int]bool
	expanded   int
	fileScroll int // scroll offset for expanded file list

	progress   progress.Model
	cleanTotal int64
	cleanFreed int64
	cleanResult cleaner.Result

	err error
}

// New creates a new TUI model.
func New(cfg *config.Config) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot

	p := progress.New(progress.WithDefaultGradient())

	return Model{
		state:    StateScanning,
		cfg:      cfg,
		spinner:  s,
		selected: map[int]bool{},
		expanded: -1,
		progress: p,
	}
}

// Init starts the spinner and scanning.
func (m Model) Init() tea.Cmd {
	ch := scanner.ScanAllStream(m.cfg)
	// We can't mutate m here (value receiver), so return a cmd that sends
	// an initScanMsg carrying the channel.
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg { return initScanMsg{ch: ch} },
	)
}

type initScanMsg struct{ ch <-chan scanner.CategoryResult }

// listenScan reads one result from ch and returns it as a tea.Msg.
func listenScan(ch <-chan scanner.CategoryResult) tea.Cmd {
	return func() tea.Msg {
		result, ok := <-ch
		if !ok {
			return scanDoneMsg{}
		}
		return scanResultMsg(result)
	}
}

// Update handles all messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = m.width - 10

	case tea.KeyMsg:
		return m.handleKey(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case initScanMsg:
		m.scanCh = msg.ch
		return m, listenScan(m.scanCh)

	case scanResultMsg:
		m.categories = append(m.categories, scanner.CategoryResult(msg))
		return m, listenScan(m.scanCh)

	case scanDoneMsg:
		m.state = StateSelect
		return m, nil

	case cleanDoneMsg:
		m.cleanResult = cleaner.Result(msg)
		m.state = StateSummary
		return m, nil

	case progress.FrameMsg:
		pm, cmd := m.progress.Update(msg)
		m.progress = pm.(progress.Model)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {

	case StateSelect:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.categories)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case " ":
			m.selected[m.cursor] = !m.selected[m.cursor]
		case "a":
			if len(m.selected) == len(m.categories) {
				m.selected = map[int]bool{}
			} else {
				for i := range m.categories {
					m.selected[i] = true
				}
			}
		case "enter":
			if m.expanded == m.cursor {
				m.expanded = -1
				m.fileScroll = 0
				m.state = StateSelect
			} else {
				m.expanded = m.cursor
				m.fileScroll = 0
				m.state = StateExpanded
			}
		case "d":
			if len(m.selected) > 0 {
				m.state = StateConfirm
			}
		}

	case StateExpanded:
		files := []scanner.FileEntry{}
		if m.expanded >= 0 && m.expanded < len(m.categories) {
			files = m.categories[m.expanded].Files
		}
		visibleLines := m.visibleFileLines()

		switch msg.String() {
		case "esc", "q":
			m.state = StateSelect
			m.expanded = -1
			m.fileScroll = 0
		case "j", "down":
			// Scroll file list down.
			if m.fileScroll < len(files)-visibleLines {
				m.fileScroll++
			}
		case "k", "up":
			// Scroll file list up.
			if m.fileScroll > 0 {
				m.fileScroll--
			}
		case "pgdown", " ":
			m.fileScroll += visibleLines
			if m.fileScroll > len(files)-visibleLines {
				m.fileScroll = max(0, len(files)-visibleLines)
			}
		case "pgup":
			m.fileScroll -= visibleLines
			if m.fileScroll < 0 {
				m.fileScroll = 0
			}
		case "G":
			m.fileScroll = max(0, len(files)-visibleLines)
		case "g":
			m.fileScroll = 0
		}

	case StateConfirm:
		switch msg.String() {
		case "y", "Y":
			m.state = StateCleaning
			return m, m.startCleanCmd()
		case "n", "N", "esc", "q":
			m.state = StateSelect
		}

	case StateSummary:
		switch msg.String() {
		case "q", "ctrl+c", "enter", "esc":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *Model) startCleanCmd() tea.Cmd {
	var entries []scanner.FileEntry
	for i, sel := range m.selected {
		if sel && i < len(m.categories) {
			entries = append(entries, m.categories[i].Files...)
		}
	}

	total := int64(0)
	for _, e := range entries {
		total += e.Size
	}
	m.cleanTotal = total

	cfg := m.cfg
	return func() tea.Msg {
		result := cleaner.Clean(entries, cfg, nil)
		return cleanDoneMsg(result)
	}
}

func (m Model) selectedSize() int64 {
	var n int64
	for i, sel := range m.selected {
		if sel && i < len(m.categories) {
			n += m.categories[i].TotalSize
		}
	}
	return n
}

// View renders the current screen.
func (m Model) View() string {
	switch m.state {
	case StateScanning:
		return m.viewScanning()
	case StateSelect, StateExpanded:
		return m.viewSelect()
	case StateConfirm:
		return m.viewConfirm()
	case StateCleaning:
		return m.viewCleaning()
	case StateSummary:
		return m.viewSummary()
	}
	return ""
}

func (m Model) viewScanning() string {
	lines := []string{
		TitleStyle.Render("  CleanMac") + DimStyle.Render(" — scanning..."),
		"",
	}
	for _, cat := range m.categories {
		sizeStr := SizeStyle.Render(humanize.Bytes(cat.TotalSize))
		lines = append(lines, fmt.Sprintf("  %s %-24s %s",
			CheckedStyle.Render("✓"), cat.DisplayName, sizeStr))
	}
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s scanning...", m.spinner.View()))
	return strings.Join(lines, "\n")
}

func (m Model) viewSelect() string {
	var sb strings.Builder

	sb.WriteString(TitleStyle.Render("  CleanMac") + DimStyle.Render(" — select categories") + "\n\n")

	for i, cat := range m.categories {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		checkbox := UncheckedStyle.Render("[ ]")
		if m.selected[i] {
			checkbox = CheckedStyle.Render("[✓]")
		}

		name := cat.DisplayName
		if i == m.cursor {
			name = BoldStyle.Render(name)
		}

		var riskStr string
		switch cat.Risk {
		case scanner.RiskWarning:
			riskStr = " " + WarningStyle.Render("⚠")
		case scanner.RiskDanger:
			riskStr = " " + DangerStyle.Render("✗")
		}

		sizeStr := SizeStyle.Render(humanize.Bytes(cat.TotalSize))
		row := fmt.Sprintf("%s%s %-24s %s%s", cursor, checkbox, name, sizeStr, riskStr)

		if i == m.cursor {
			sb.WriteString(SelectedRowStyle.Render(row) + "\n")
		} else {
			sb.WriteString(row + "\n")
		}

		// Expanded file list (scrollable).
		if m.state == StateExpanded && m.expanded == i {
			files := cat.Files
			visible := m.visibleFileLines()
			start := m.fileScroll
			if start > len(files) {
				start = 0
			}
			end := start + visible
			if end > len(files) {
				end = len(files)
			}

			for _, f := range files[start:end] {
				sizeStr := SizeStyle.Render(fmt.Sprintf("%10s", humanize.Bytes(f.Size)))
				sb.WriteString(fmt.Sprintf("       %s  %s\n",
					sizeStr, DimStyle.Render(truncate(f.Name, 45))))
			}

			// Scroll indicator.
			scrollInfo := fmt.Sprintf("       showing %d–%d of %d  (j/k scroll, g/G top/bottom, esc back)",
				start+1, end, len(files))
			sb.WriteString(DimStyle.Render(scrollInfo) + "\n")
		}
	}

	sb.WriteString("\n")

	if len(m.selected) > 0 {
		sel := humanize.Bytes(m.selectedSize())
		sb.WriteString(StatusBarStyle.Render(
			fmt.Sprintf(" Selected: %s  |  d=delete  space=toggle  a=all  q=quit", sel)))
	} else {
		sb.WriteString(StatusBarStyle.Render(
			" space=select  enter=expand  a=all  d=delete  q=quit"))
	}

	return BorderStyle.Render(sb.String())
}

func (m Model) viewConfirm() string {
	sel := humanize.Bytes(m.selectedSize())
	dryNote := ""
	if m.cfg.DryRun {
		dryNote = "  " + DimStyle.Render("(dry-run — nothing will be deleted)")
	}
	body := fmt.Sprintf("\n  Delete %s?%s\n\n  %s Yes   %s No\n",
		DangerStyle.Render(sel),
		dryNote,
		CheckedStyle.Render("[y]"),
		UncheckedStyle.Render("[n]"),
	)
	return BorderStyle.Render(TitleStyle.Render("  Confirm") + "\n" + body)
}

func (m Model) viewCleaning() string {
	bar := m.progress.View()
	freed := SizeStyle.Render(humanize.Bytes(m.cleanFreed))
	return BorderStyle.Render(fmt.Sprintf(
		"\n  %s Cleaning...\n\n  %s\n\n  Freed: %s\n",
		m.spinner.View(), bar, freed,
	))
}

func (m Model) viewSummary() string {
	r := m.cleanResult
	dryNote := ""
	if m.cfg.DryRun {
		dryNote = DimStyle.Render(" (dry-run)")
	}
	lines := []string{
		TitleStyle.Render("  Done!") + dryNote,
		"",
		fmt.Sprintf("  Freed:   %s", SizeStyle.Render(humanize.Bytes(r.Freed))),
		fmt.Sprintf("  Items:   %d deleted", r.Count),
	}
	if len(r.Errors) > 0 {
		lines = append(lines, fmt.Sprintf("  Errors:  %d skipped", len(r.Errors)))
	}
	lines = append(lines, "", DimStyle.Render("  Press q or enter to exit"))
	return BorderStyle.Render(strings.Join(lines, "\n"))
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}

// visibleFileLines returns how many file entries fit in the expanded panel.
func (m Model) visibleFileLines() int {
	// Total height minus: border(2) + title(2) + category rows + statusbar(1) + scroll indicator(1) + padding(2)
	overhead := len(m.categories) + 8
	n := m.height - overhead
	if n < 5 {
		return 5
	}
	return n
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

