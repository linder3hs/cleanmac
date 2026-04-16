package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/linder3hs/cleanmac/internal/cleaner"
	"github.com/linder3hs/cleanmac/internal/config"
	"github.com/linder3hs/cleanmac/internal/diskinfo"
	"github.com/linder3hs/cleanmac/internal/humanize"
	"github.com/linder3hs/cleanmac/internal/scanner"
)

// allCategoryNames is the canonical display order for the scan progress screen.
var allCategoryNames = []string{
	"System & App Caches",
	"Log Files",
	"Browser Caches",
	"Dev Artifacts",
	"Large Files",
	"Trash",
	"iOS Backups",
	"Mail Downloads",
}

// State represents the current TUI screen.
type State int

const (
	StateScanning State = iota
	StateSelect
	StateFiles       // was StateExpanded — now a full bubbles/table view
	StateConfirm     // confirm category-level (single or multi) clean
	StateFileConfirm // confirm deletion of a single file from the file list
	StateCleaning
	StateSummary
	StateMenu // hub menu (entry for `cleanmac init`)
	StateDisk // disk-space view inside the TUI
)

// menuItem describes one option in the hub menu.
type menuItem struct {
	label string
	desc  string
	key   string // single-char shortcut
}

var menuItems = []menuItem{
	{label: "Scan & Clean", desc: "Scan categories and choose what to delete", key: "s"},
	{label: "Disk Space", desc: "Show disk space for all volumes", key: "d"},
	{label: "Quit", desc: "Exit cleanmac", key: "q"},
}

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

	spinner spinner.Model
	scanCh  <-chan scanner.CategoryResult

	categories    []scanner.CategoryResult
	scanTotal     int64
	categoryTable table.Model
	fileTable     table.Model
	fileEntries   []scanner.FileEntry // sorted entries mirroring fileTable rows

	selected  map[int]bool
	cleanMode string // "single" | "multi"
	cleanIdx  int    // category index for single-clean or file detail view

	// Persistent disk widget data (root volume), shown in every view header.
	diskMain    diskinfo.Mount
	lastErr     string // transient error/info shown briefly under headers

	progress    progress.Model
	cleanTotal  int64
	cleanFreed  int64
	cleanResult cleaner.Result

	// Hub-menu state. Set when the model is constructed via NewWithMenu.
	hasMenu    bool
	menuCursor int
	mounts     []diskinfo.Mount

	err error
}

// New creates a new TUI model that boots straight into scanning.
func New(cfg *config.Config) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot

	p := progress.New(progress.WithDefaultGradient())

	m := Model{
		state:    StateScanning,
		cfg:      cfg,
		spinner:  s,
		selected: map[int]bool{},
		cleanIdx: -1,
		progress: p,
	}
	m.refreshDisk()
	return m
}

// refreshDisk re-reads the root mount usage. Called on init and after any
// deletion so the persistent widget stays accurate.
func (m *Model) refreshDisk() {
	mounts, err := diskinfo.Gather(false)
	if err != nil {
		return
	}
	for _, mt := range mounts {
		if mt.Path == "/" {
			m.diskMain = mt
			return
		}
	}
	if len(mounts) > 0 {
		m.diskMain = mounts[0]
	}
}

// NewWithMenu creates a TUI model that boots into the hub menu (StateMenu).
// Used by `cleanmac init` so flows return to the menu instead of quitting.
func NewWithMenu(cfg *config.Config) Model {
	m := New(cfg)
	m.state = StateMenu
	m.hasMenu = true
	return m
}

// Init starts the spinner and (when not in menu mode) the scan.
func (m Model) Init() tea.Cmd {
	if m.state == StateMenu {
		return m.spinner.Tick
	}
	ch := scanner.ScanAllStream(m.cfg)
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg { return initScanMsg{ch: ch} },
	)
}

// beginScan resets all scan-related state and returns a cmd that starts a
// fresh scan stream. Used when entering scan from the menu (possibly multiple
// times in one session).
func (m *Model) beginScan() tea.Cmd {
	m.categories = nil
	m.scanTotal = 0
	m.selected = map[int]bool{}
	m.cleanIdx = -1
	m.cleanFreed = 0
	m.cleanResult = cleaner.Result{}
	m.cleanMode = ""
	m.state = StateScanning
	ch := scanner.ScanAllStream(m.cfg)
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg { return initScanMsg{ch: ch} },
	)
}

type initScanMsg struct{ ch <-chan scanner.CategoryResult }

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
		if len(m.categories) > 0 {
			cur := m.categoryTable.Cursor()
			m.categoryTable = buildCategoryTable(m.categories, m.selected, m.scanTotal)
			m.categoryTable.SetCursor(cur)
		}
		if m.cleanIdx >= 0 && m.cleanIdx < len(m.categories) {
			m.fileTable = buildFileTable(m.categories[m.cleanIdx].Files, m.width, m.height)
		}

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
		cat := scanner.CategoryResult(msg)
		m.categories = append(m.categories, cat)
		m.scanTotal += cat.TotalSize
		return m, listenScan(m.scanCh)

	case scanDoneMsg:
		m.categoryTable = buildCategoryTable(m.categories, m.selected, m.scanTotal)
		m.state = StateSelect
		return m, nil

	case cleanDoneMsg:
		m.cleanResult = cleaner.Result(msg)
		m.refreshDisk()
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

	case StateMenu:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.menuCursor > 0 {
				m.menuCursor--
			}
		case "down", "j":
			if m.menuCursor < len(menuItems)-1 {
				m.menuCursor++
			}
		case "s", "1":
			m.menuCursor = 0
			return m.runMenuItem()
		case "d", "2":
			m.menuCursor = 1
			return m.runMenuItem()
		case "enter":
			return m.runMenuItem()
		}

	case StateDisk:
		switch msg.String() {
		case "esc", "m", "q":
			if m.hasMenu {
				m.state = StateMenu
			} else {
				return m, tea.Quit
			}
		case "ctrl+c":
			return m, tea.Quit
		}

	case StateSelect:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		// Delegate navigation to the table component.
		case "up", "k", "down", "j", "pgup", "pgdown", "home", "end":
			var cmd tea.Cmd
			m.categoryTable, cmd = m.categoryTable.Update(msg)
			return m, cmd

		case " ":
			idx := m.categoryTable.Cursor()
			m.selected[idx] = !m.selected[idx]
			cur := m.categoryTable.Cursor()
			m.categoryTable = buildCategoryTable(m.categories, m.selected, m.scanTotal)
			m.categoryTable.SetCursor(cur)

		case "a":
			if len(m.selected) == len(m.categories) {
				m.selected = map[int]bool{}
			} else {
				for i := range m.categories {
					m.selected[i] = true
				}
			}
			cur := m.categoryTable.Cursor()
			m.categoryTable = buildCategoryTable(m.categories, m.selected, m.scanTotal)
			m.categoryTable.SetCursor(cur)

		case "enter":
			idx := m.categoryTable.Cursor()
			if idx < len(m.categories) {
				m.enterFiles(idx)
			}

		case "c":
			// Quick clean — skip file view, go straight to confirm.
			idx := m.categoryTable.Cursor()
			if idx < len(m.categories) {
				m.cleanIdx = idx
				m.cleanMode = "single"
				m.state = StateConfirm
			}

		case "d":
			if len(m.selected) > 0 {
				m.cleanMode = "multi"
				m.state = StateConfirm
			}
		}

	case StateFiles:
		switch msg.String() {
		case "esc", "q":
			m.state = StateSelect
			m.lastErr = ""

		case "up", "k", "down", "j", "pgup", "pgdown", "home", "end":
			var cmd tea.Cmd
			m.fileTable, cmd = m.fileTable.Update(msg)
			return m, cmd

		case "g":
			m.fileTable.GotoTop()

		case "G":
			m.fileTable.GotoBottom()

		case "c":
			m.cleanMode = "single"
			m.state = StateConfirm

		case "x", "delete", "backspace":
			// Delete just the file under the cursor — go to confirm.
			if len(m.fileEntries) > 0 {
				m.state = StateFileConfirm
			}
		}

	case StateFileConfirm:
		switch msg.String() {
		case "y", "Y":
			m.deleteCurrentFile()
			if len(m.fileEntries) == 0 {
				m.state = StateSelect
			} else {
				m.state = StateFiles
			}
		case "n", "N", "esc", "q":
			m.state = StateFiles
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
		case "ctrl+c":
			return m, tea.Quit
		case "esc", "m", "enter":
			if m.hasMenu {
				m.state = StateMenu
				return m, nil
			}
			return m, tea.Quit
		case "q":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *Model) startCleanCmd() tea.Cmd {
	type catEntries struct {
		name    string
		entries []scanner.FileEntry
	}

	var toClean []catEntries

	if m.cleanMode == "single" && m.cleanIdx >= 0 && m.cleanIdx < len(m.categories) {
		cat := m.categories[m.cleanIdx]
		toClean = append(toClean, catEntries{cat.DisplayName, cat.Files})
	} else {
		for i, sel := range m.selected {
			if sel && i < len(m.categories) {
				cat := m.categories[i]
				toClean = append(toClean, catEntries{cat.DisplayName, cat.Files})
			}
		}
	}

	total := int64(0)
	for _, c := range toClean {
		for _, e := range c.entries {
			total += e.Size
		}
	}
	m.cleanTotal = total

	cfg := m.cfg
	return func() tea.Msg {
		result := cleaner.Result{}
		for _, c := range toClean {
			r := cleaner.Clean(c.entries, cfg, nil)
			result.Freed += r.Freed
			result.Count += r.Count
			result.Errors = append(result.Errors, r.Errors...)
			result.Categories = append(result.Categories, cleaner.CategoryFreed{
				Name:  c.name,
				Freed: r.Freed,
				Count: r.Count,
			})
		}
		return cleanDoneMsg(result)
	}
}

func (m Model) multiSelectedSize() int64 {
	var n int64
	for i, sel := range m.selected {
		if sel && i < len(m.categories) {
			n += m.categories[i].TotalSize
		}
	}
	return n
}

// enterFiles opens the file-detail view for the given category index.
// It also stores the sorted entries so the cursor maps back correctly when
// the user wants to delete a single file.
func (m *Model) enterFiles(idx int) {
	m.cleanIdx = idx
	cat := m.categories[idx]

	sorted := make([]scanner.FileEntry, len(cat.Files))
	copy(sorted, cat.Files)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Size > sorted[j].Size
	})
	m.fileEntries = sorted

	m.fileTable = buildFileTable(cat.Files, m.width, m.height)
	m.lastErr = ""
	m.state = StateFiles
}

// deleteCurrentFile deletes the file under the cursor in the file table,
// updates category/scan totals, rebuilds tables, and refreshes disk usage.
func (m *Model) deleteCurrentFile() {
	if m.cleanIdx < 0 || m.cleanIdx >= len(m.categories) {
		return
	}
	cur := m.fileTable.Cursor()
	if cur < 0 || cur >= len(m.fileEntries) {
		return
	}

	entry := m.fileEntries[cur]
	r := cleaner.Clean([]scanner.FileEntry{entry}, m.cfg, nil)

	if len(r.Errors) > 0 && r.Count == 0 {
		m.lastErr = r.Errors[0].Error()
		return
	}

	// Remove from sorted slice.
	m.fileEntries = append(m.fileEntries[:cur], m.fileEntries[cur+1:]...)

	// Remove from underlying category Files (match by Path).
	cat := m.categories[m.cleanIdx]
	for i, f := range cat.Files {
		if f.Path == entry.Path {
			cat.Files = append(cat.Files[:i], cat.Files[i+1:]...)
			break
		}
	}
	cat.TotalSize -= r.Freed
	if cat.TotalSize < 0 {
		cat.TotalSize = 0
	}
	m.categories[m.cleanIdx] = cat
	m.scanTotal -= r.Freed
	if m.scanTotal < 0 {
		m.scanTotal = 0
	}

	// Rebuild tables.
	curCat := m.categoryTable.Cursor()
	m.categoryTable = buildCategoryTable(m.categories, m.selected, m.scanTotal)
	m.categoryTable.SetCursor(curCat)

	m.fileTable = buildFileTable(cat.Files, m.width, m.height)
	if cur >= len(m.fileEntries) && cur > 0 {
		m.fileTable.SetCursor(cur - 1)
	} else {
		m.fileTable.SetCursor(cur)
	}

	m.refreshDisk()
	m.lastErr = fmt.Sprintf("deleted %s (%s)", entry.Name, humanize.Bytes(r.Freed))
}

// diskWidget returns a compact, colored "free / total [bar] pct%" string
// for the root volume — shown in every screen header.
func (m Model) diskWidget() string {
	d := m.diskMain
	if d.Total == 0 {
		return ""
	}
	pct := d.Pct()
	bar := diskinfo.UsageBar(pct, 10)

	var styledBar string
	switch {
	case pct >= 90:
		styledBar = DangerStyle.Render("[" + bar + "]")
	case pct >= 70:
		styledBar = WarningStyle.Render("[" + bar + "]")
	default:
		styledBar = SafeStyle.Render("[" + bar + "]")
	}

	return fmt.Sprintf("%s %s %s",
		DimStyle.Render("disk"),
		styledBar,
		SizeStyle.Render(fmt.Sprintf("%s free / %s", humanize.Bytes(d.Free), humanize.Bytes(d.Total))),
	)
}

// headerLine returns "<title>     <diskWidget>" right-padded to fit the
// terminal width. When width is unknown, both halves are concatenated with
// two spaces.
func (m Model) headerLine(title string) string {
	left := TitleStyle.Render("  " + title)
	right := m.diskWidget()
	if right == "" {
		return left
	}
	if m.width <= 0 {
		return left + "    " + right
	}
	// BorderStyle adds 1 char border + 1 char padding each side = 4 chars.
	inner := m.width - 4
	gap := inner - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 2 {
		gap = 2
	}
	return left + strings.Repeat(" ", gap) + right
}

// runMenuItem executes the action for the current menuCursor.
func (m Model) runMenuItem() (tea.Model, tea.Cmd) {
	switch m.menuCursor {
	case 0: // Scan & Clean
		cmd := m.beginScan()
		return m, cmd
	case 1: // Disk Space
		mounts, _ := diskinfo.Gather(false)
		m.mounts = mounts
		m.state = StateDisk
		return m, nil
	case 2: // Quit
		return m, tea.Quit
	}
	return m, nil
}

// View renders the current screen.
func (m Model) View() string {
	switch m.state {
	case StateMenu:
		return m.viewMenu()
	case StateDisk:
		return m.viewDisk()
	case StateScanning:
		return m.viewScanning()
	case StateSelect:
		return m.viewSelect()
	case StateFiles:
		return m.viewFiles()
	case StateConfirm:
		return m.viewConfirm()
	case StateFileConfirm:
		return m.viewFileConfirm()
	case StateCleaning:
		return m.viewCleaning()
	case StateSummary:
		return m.viewSummary()
	}
	return ""
}

func (m Model) viewMenu() string {
	var sb strings.Builder

	sb.WriteString(m.headerLine("CleanMac") + "\n")
	sb.WriteString(DimStyle.Render("  Interactive disk cleaner for macOS") + "\n\n")

	for i, item := range menuItems {
		var prefix, label string
		if i == m.menuCursor {
			prefix = CheckedStyle.Render("  ▸ ")
			label = BoldStyle.Render(item.label)
		} else {
			prefix = "    "
			label = item.label
		}
		shortcut := DimStyle.Render(fmt.Sprintf("  [%s]", item.key))
		sb.WriteString(prefix + label + shortcut + "\n")
		sb.WriteString(DimStyle.Render("       "+item.desc) + "\n\n")
	}

	sb.WriteString(StatusBarStyle.Render(
		" ↑↓/jk navigate  enter select  s/d quick  q quit "))

	return BorderStyle.Render(sb.String())
}

func (m Model) viewDisk() string {
	var sb strings.Builder

	sb.WriteString(TitleStyle.Render("  Disk Space") + "\n\n")

	colMount := 22
	colSize := 10
	barWidth := 12

	hdr := fmt.Sprintf("  %-*s  %*s  %*s  %*s  %-*s",
		colMount, "Volume",
		colSize, "Total",
		colSize, "Used",
		colSize, "Free",
		barWidth+8, "Usage",
	)
	sep := fmt.Sprintf("  %s  %s  %s  %s  %s",
		strings.Repeat("─", colMount),
		strings.Repeat("─", colSize),
		strings.Repeat("─", colSize),
		strings.Repeat("─", colSize),
		strings.Repeat("─", barWidth+8),
	)

	sb.WriteString(HeaderStyle.Render(hdr) + "\n")
	sb.WriteString(DimStyle.Render(sep) + "\n")

	for _, d := range m.mounts {
		pct := d.Pct()
		bar := fmt.Sprintf("[%s]", diskinfo.UsageBar(pct, barWidth))

		var styledBar string
		switch {
		case pct >= 90:
			styledBar = DangerStyle.Render(bar)
		case pct >= 70:
			styledBar = WarningStyle.Render(bar)
		default:
			styledBar = SafeStyle.Render(bar)
		}

		mount := d.Path
		if len([]rune(mount)) > colMount {
			runes := []rune(mount)
			mount = "…" + string(runes[len(runes)-colMount+1:])
		}

		row := fmt.Sprintf("  %-*s  %s  %s  %s  %s %s",
			colMount, mount,
			SizeStyle.Render(fmt.Sprintf("%*s", colSize, humanize.Bytes(d.Total))),
			SizeStyle.Render(fmt.Sprintf("%*s", colSize, humanize.Bytes(d.Used))),
			SizeStyle.Render(fmt.Sprintf("%*s", colSize, humanize.Bytes(d.Free))),
			styledBar,
			fmt.Sprintf("%3.0f%%", pct),
		)
		sb.WriteString(row + "\n")
	}

	sb.WriteString(DimStyle.Render(sep) + "\n\n")

	footer := " esc/m back to menu  q quit "
	if !m.hasMenu {
		footer = " q/esc quit "
	}
	sb.WriteString(StatusBarStyle.Render(footer))

	return BorderStyle.Render(sb.String())
}

func (m Model) viewScanning() string {
	lines := []string{
		TitleStyle.Render("  CleanMac") + "  " + m.spinner.View() + "  " + DimStyle.Render("scanning..."),
		"",
		DimStyle.Render(fmt.Sprintf("  %-28s  %s", "Category", "Size")),
		DimStyle.Render("  " + strings.Repeat("─", 28) + "  " + strings.Repeat("─", 10)),
	}

	done := map[string]scanner.CategoryResult{}
	for _, c := range m.categories {
		done[c.DisplayName] = c
	}

	for _, name := range allCategoryNames {
		if cat, ok := done[name]; ok {
			sizeStr := SizeStyle.Render(fmt.Sprintf("%-10s", humanize.Bytes(cat.TotalSize)))
			lines = append(lines, fmt.Sprintf("  %s %-28s  %s",
				CheckedStyle.Render("✓"), name, sizeStr))
		} else {
			lines = append(lines, fmt.Sprintf("  %s %-28s  %s",
				DimStyle.Render(" "), name, DimStyle.Render("...")))
		}
	}

	prog := fmt.Sprintf("  %d / %d categories scanned", len(m.categories), len(allCategoryNames))
	lines = append(lines, "", DimStyle.Render(prog))

	return strings.Join(lines, "\n")
}

func (m Model) viewSelect() string {
	var sb strings.Builder

	sb.WriteString(m.headerLine("CleanMac — select categories") + "\n\n")
	sb.WriteString(m.categoryTable.View())
	sb.WriteString("\n\n")

	var statusMsg string
	if len(m.selected) > 0 {
		n := 0
		for _, v := range m.selected {
			if v {
				n++
			}
		}
		sel := humanize.Bytes(m.multiSelectedSize())
		statusMsg = fmt.Sprintf(" %d selected (%s)  │  space=toggle  a=all  enter=files  c=clean this  d=delete  q=quit", n, sel)
	} else {
		statusMsg = " ↑↓/jk=navigate  space=select  enter=view files  c=clean this  a=all  d=delete  q=quit"
	}
	sb.WriteString(StatusBarStyle.Render(statusMsg))

	return BorderStyle.Render(sb.String())
}

func (m Model) viewFiles() string {
	if m.cleanIdx < 0 || m.cleanIdx >= len(m.categories) {
		return ""
	}
	cat := m.categories[m.cleanIdx]

	var sb strings.Builder

	riskStr := ""
	switch cat.Risk {
	case scanner.RiskWarning:
		riskStr = "  " + WarningStyle.Render("⚠ review carefully")
	case scanner.RiskDanger:
		riskStr = "  " + DangerStyle.Render("✗ danger")
	}

	title := m.headerLine(cat.DisplayName) + riskStr
	sub := DimStyle.Render(fmt.Sprintf("  %s — %d files", humanize.Bytes(cat.TotalSize), len(cat.Files)))
	sb.WriteString(title + "\n" + sub + "\n\n")

	if len(cat.Files) == 0 {
		sb.WriteString(DimStyle.Render("  (nothing found in this category)") + "\n")
	} else {
		sb.WriteString(m.fileTable.View())
	}

	if m.lastErr != "" {
		sb.WriteString("\n  " + DimStyle.Render(m.lastErr))
	}

	sb.WriteString("\n\n")
	sb.WriteString(StatusBarStyle.Render(
		" ↑↓/jk=scroll  g/G=top/bottom  x=delete file  c=clean category  esc=back"))

	return BorderStyle.Render(sb.String())
}

func (m Model) viewConfirm() string {
	dryNote := ""
	if m.cfg.DryRun {
		dryNote = "\n\n  " + DimStyle.Render("(dry-run — nothing will actually be deleted)")
	}

	var body string
	if m.cleanMode == "single" && m.cleanIdx >= 0 && m.cleanIdx < len(m.categories) {
		cat := m.categories[m.cleanIdx]
		riskWarn := ""
		if cat.Risk == scanner.RiskWarning {
			riskWarn = "\n  " + WarningStyle.Render("⚠  This category is marked review — proceed carefully.")
		}
		body = fmt.Sprintf(
			"\n  Clean: %s%s\n\n  %s — %d files%s\n\n  %s Yes   %s No\n",
			BoldStyle.Render(cat.DisplayName),
			riskWarn,
			DangerStyle.Render(humanize.Bytes(cat.TotalSize)),
			len(cat.Files),
			dryNote,
			CheckedStyle.Render("[y]"),
			UncheckedStyle.Render("[n]"),
		)
	} else {
		// Multi mode: list each selected category.
		var lines []string
		var totalSize int64
		var totalFiles int
		for i, sel := range m.selected {
			if sel && i < len(m.categories) {
				cat := m.categories[i]
				totalSize += cat.TotalSize
				totalFiles += len(cat.Files)
				lines = append(lines, fmt.Sprintf("  %s %-24s  %s",
					DimStyle.Render("•"),
					cat.DisplayName,
					SizeStyle.Render(humanize.Bytes(cat.TotalSize)),
				))
			}
		}
		catList := strings.Join(lines, "\n")
		word := "categories"
		if len(lines) == 1 {
			word = "category"
		}
		body = fmt.Sprintf(
			"\n  Clean %d %s:\n\n%s\n\n  Total: %s — %d files%s\n\n  %s Yes   %s No\n",
			len(lines),
			word,
			catList,
			DangerStyle.Render(humanize.Bytes(totalSize)),
			totalFiles,
			dryNote,
			CheckedStyle.Render("[y]"),
			UncheckedStyle.Render("[n]"),
		)
	}

	return BorderStyle.Render(TitleStyle.Render("  Confirm") + "\n" + body)
}

func (m Model) viewFileConfirm() string {
	if m.cleanIdx < 0 || m.cleanIdx >= len(m.categories) {
		return ""
	}
	cur := m.fileTable.Cursor()
	if cur < 0 || cur >= len(m.fileEntries) {
		return ""
	}
	entry := m.fileEntries[cur]

	dryNote := ""
	if m.cfg.DryRun {
		dryNote = "\n\n  " + DimStyle.Render("(dry-run — nothing will actually be deleted)")
	}

	body := fmt.Sprintf(
		"\n  Delete this file?\n\n  %s  %s\n  %s%s\n\n  %s Yes   %s No\n",
		DangerStyle.Render(humanize.Bytes(entry.Size)),
		BoldStyle.Render(entry.Name),
		DimStyle.Render(entry.Path),
		dryNote,
		CheckedStyle.Render("[y]"),
		UncheckedStyle.Render("[n]"),
	)

	return BorderStyle.Render(m.headerLine("Confirm delete") + "\n" + body)
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
		m.headerLine("Done! ✓") + dryNote,
		"",
	}

	// Per-category breakdown.
	if len(r.Categories) > 0 {
		colName := 24
		colFreed := 10
		hdr := fmt.Sprintf("  %-*s  %*s  %s", colName, "Category", colFreed, "Freed", "Items")
		sep := fmt.Sprintf("  %s  %s  %s",
			strings.Repeat("─", colName),
			strings.Repeat("─", colFreed),
			strings.Repeat("─", 5))
		lines = append(lines, DimStyle.Render(hdr), DimStyle.Render(sep))
		for _, cat := range r.Categories {
			lines = append(lines, fmt.Sprintf("  %-*s  %s  %d",
				colName, cat.Name,
				SizeStyle.Render(fmt.Sprintf("%*s", colFreed, humanize.Bytes(cat.Freed))),
				cat.Count,
			))
		}
		lines = append(lines, DimStyle.Render(sep), "")
	}

	lines = append(lines,
		fmt.Sprintf("  Total freed:   %s", SizeStyle.Render(humanize.Bytes(r.Freed))),
		fmt.Sprintf("  Items deleted: %d", r.Count),
	)
	if len(r.Errors) > 0 {
		lines = append(lines, DangerStyle.Render(fmt.Sprintf("  Errors:        %d skipped", len(r.Errors))))
	}
	if m.hasMenu {
		lines = append(lines, "", DimStyle.Render("  enter/esc back to menu  q quit"))
	} else {
		lines = append(lines, "", DimStyle.Render("  Press q or enter to exit"))
	}

	return BorderStyle.Render(strings.Join(lines, "\n"))
}

// buildCategoryTable constructs the interactive category selection table.
func buildCategoryTable(cats []scanner.CategoryResult, selected map[int]bool, total int64) table.Model {
	cols := []table.Column{
		{Title: " ", Width: 3},
		{Title: "Category", Width: 22},
		{Title: "Size", Width: 10},
		{Title: "%", Width: 5},
		{Title: "Files", Width: 6},
		{Title: "Risk", Width: 9},
	}

	rows := make([]table.Row, len(cats))
	for i, cat := range cats {
		check := "[ ]"
		if selected[i] {
			check = "[✓]"
		}
		pct := 0.0
		if total > 0 {
			pct = float64(cat.TotalSize) / float64(total) * 100
		}
		risk := "✓ safe"
		if cat.Risk == scanner.RiskWarning {
			risk = "⚠ review"
		} else if cat.Risk == scanner.RiskDanger {
			risk = "✗ danger"
		}
		rows[i] = table.Row{
			check,
			cat.DisplayName,
			humanize.Bytes(cat.TotalSize),
			fmt.Sprintf("%4.0f%%", pct),
			fmt.Sprintf("%6d", len(cat.Files)),
			risk,
		}
	}

	height := len(cats)
	if height < 1 {
		height = 1
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorBlue).
		BorderBottom(true).
		Bold(true).
		Foreground(ColorWhite)
	s.Selected = s.Selected.
		Foreground(ColorWhite).
		Background(lipgloss.Color("236")).
		Bold(false)
	t.SetStyles(s)

	return t
}

// buildFileTable constructs the scrollable file detail table.
func buildFileTable(files []scanner.FileEntry, termWidth, termHeight int) table.Model {
	pathWidth := termWidth - 20
	if pathWidth < 30 {
		pathWidth = 30
	}

	cols := []table.Column{
		{Title: "Size", Width: 10},
		{Title: "Path", Width: pathWidth},
	}

	// Sort by size descending.
	sorted := make([]scanner.FileEntry, len(files))
	copy(sorted, files)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Size > sorted[j].Size
	})

	rows := make([]table.Row, len(sorted))
	for i, f := range sorted {
		name := f.Name
		if len([]rune(name)) > pathWidth {
			runes := []rune(name)
			name = "…" + string(runes[len(runes)-pathWidth+1:])
		}
		rows[i] = table.Row{humanize.Bytes(f.Size), name}
	}

	height := termHeight - 12
	if height < 5 {
		height = 5
	}
	if len(sorted) > 0 && height > len(sorted) {
		height = len(sorted)
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorBlue).
		BorderBottom(true).
		Bold(true).
		Foreground(ColorWhite)
	s.Selected = s.Selected.
		Foreground(ColorCyan).
		Background(lipgloss.Color("236")).
		Bold(false)
	t.SetStyles(s)

	return t
}
