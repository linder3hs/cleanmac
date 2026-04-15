package scanner

// CategoryID identifies a cleanup category.
type CategoryID string

const (
	CategoryCaches       CategoryID = "caches"
	CategoryLogs         CategoryID = "logs"
	CategoryDevArtifacts CategoryID = "dev"
	CategoryBrowser      CategoryID = "browser"
	CategoryLargeFiles   CategoryID = "large"
	CategoryDuplicates   CategoryID = "duplicates"
	CategoryTrash        CategoryID = "trash"
	CategoryIOSBackups   CategoryID = "ios"
	CategoryLangFiles    CategoryID = "lang"
	CategoryMail         CategoryID = "mail"
)

// RiskLevel indicates how safe it is to delete a category.
type RiskLevel int

const (
	RiskSafe    RiskLevel = iota // green
	RiskWarning                  // yellow — review before deleting
	RiskDanger                   // red — requires extra confirmation
)

// FileEntry is a single deletable path with its size.
type FileEntry struct {
	Path string
	Size int64
	Name string // display name (may differ from filepath.Base)
}

// CategoryResult holds scan results for one category.
type CategoryResult struct {
	ID          CategoryID
	DisplayName string
	TotalSize   int64
	Files       []FileEntry
	Risk        RiskLevel
	Error       error
}

// ScanResult holds all category results.
type ScanResult struct {
	Categories []CategoryResult
	TotalSize  int64
}
