package scanner

import (
	"sync"

	"github.com/linder3hs/cleanmac/internal/config"
)

// ScanAll runs all category scanners concurrently and returns a ScanResult.
// Results stream in via channel as each scanner completes.
func ScanAll(cfg *config.Config) *ScanResult {
	scanners := []func(*config.Config) CategoryResult{
		ScanCaches,
		ScanLogs,
		ScanDevArtifacts,
		ScanBrowser,
		ScanTrash,
		ScanLargeFiles,
		ScanIOSBackups,
		ScanMail,
	}

	results := make([]CategoryResult, len(scanners))
	var wg sync.WaitGroup

	for i, fn := range scanners {
		wg.Add(1)
		go func(idx int, scanFn func(*config.Config) CategoryResult) {
			defer wg.Done()
			results[idx] = scanFn(cfg)
		}(i, fn)
	}

	wg.Wait()

	sr := &ScanResult{Categories: results}
	for _, r := range results {
		sr.TotalSize += r.TotalSize
	}
	return sr
}

// ScanAllStream runs all category scanners concurrently, streaming results via a channel.
// The channel is closed when all scanners complete.
func ScanAllStream(cfg *config.Config) <-chan CategoryResult {
	ch := make(chan CategoryResult, 10)

	scanners := []func(*config.Config) CategoryResult{
		ScanCaches,
		ScanLogs,
		ScanDevArtifacts,
		ScanBrowser,
		ScanTrash,
		ScanLargeFiles,
		ScanIOSBackups,
		ScanMail,
	}

	var wg sync.WaitGroup
	for _, fn := range scanners {
		wg.Add(1)
		go func(scanFn func(*config.Config) CategoryResult) {
			defer wg.Done()
			ch <- scanFn(cfg)
		}(fn)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	return ch
}

