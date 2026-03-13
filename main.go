package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// FolderReport is the JSON output structure for each folder.
type FolderReport struct {
	Folder string `json:"user"`
	Size   string `json:"size"`
}

func main() {
	dir := flag.String("dir", ".", "Root directory to scan")
	exclude := flag.String("exclude", "", "Comma-separated list of top-level folder names to exclude")
	threshold := flag.Float64("threshold", 20, "Size threshold in GB for cleanup report")
	flag.Parse()

	// Build the exclude set for O(1) lookups.
	excludeSet := make(map[string]bool)
	if *exclude != "" {
		for _, name := range strings.Split(*exclude, ",") {
			trimmed := strings.TrimSpace(name)
			if trimmed != "" {
				excludeSet[trimmed] = true
			}
		}
	}

	// Read top-level entries.
	entries, err := os.ReadDir(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot read directory %s: %v\n", *dir, err)
		os.Exit(1)
	}

	// Filter to directories only, excluding the exclude set.
	type folderJob struct {
		name string
		path string
		size atomic.Int64
	}

	var jobs []*folderJob
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if excludeSet[name] {
			continue
		}
		jobs = append(jobs, &folderJob{
			name: name,
			path: filepath.Join(*dir, name),
		})
	}

	if len(jobs) == 0 {
		fmt.Println("No directories to scan.")
		writeJSON("/tmp/report.json", []FolderReport{})
		writeJSON("/tmp/cleanup_needed.json", []FolderReport{})
		return
	}

	// Semaphore to cap concurrent goroutines and avoid fd exhaustion.
	// NumCPU * 4 keeps all cores saturated while staying within OS limits.
	semSize := runtime.NumCPU() * 4
	if semSize < 16 {
		semSize = 16
	}
	sem := make(chan struct{}, semSize)

	start := time.Now()
	fmt.Printf("🚀 Scanning %d directories in %s (workers: %d)...\n", len(jobs), *dir, semSize)

	// Launch all top-level scans concurrently.
	var wg sync.WaitGroup
	for _, job := range jobs {
		wg.Add(1)
		go func(j *folderJob) {
			defer wg.Done()
			scanDir(j.path, &j.size, sem)
		}(job)
	}
	wg.Wait()

	elapsed := time.Since(start)

	// Build reports.
	report := make([]FolderReport, 0, len(jobs))
	cleanup := make([]FolderReport, 0)
	thresholdBytes := int64(*threshold * 1024 * 1024 * 1024)

	for _, j := range jobs {
		sizeBytes := j.size.Load()
		entry := FolderReport{
			Folder: j.name,
			Size:   formatSize(sizeBytes),
		}
		report = append(report, entry)
		if sizeBytes > thresholdBytes {
			cleanup = append(cleanup, entry)
		}
	}

	// Sort reports by size descending for quick readability.
	sortBySize := func(slice []FolderReport) {
		sort.Slice(slice, func(i, k int) bool {
			return parseBack(slice[i].Size) > parseBack(slice[k].Size)
		})
	}
	sortBySize(report)
	sortBySize(cleanup)

	// Write output.
	writeJSON("/tmp/report.json", report)
	writeJSON("/tmp/cleanup_needed.json", cleanup)

	fmt.Printf("✅ Done in %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("   📄 /tmp/report.json          — %d folders\n", len(report))
	fmt.Printf("   ⚠️  /tmp/cleanup_needed.json  — %d folders > %g GB\n", len(cleanup), *threshold)
}

// scanDir recursively walks a directory, accumulating file sizes atomically.
// It uses a semaphore channel to cap concurrency and avoid fd exhaustion.
func scanDir(path string, totalSize *atomic.Int64, sem chan struct{}) {
	// Acquire semaphore slot.
	sem <- struct{}{}
	entries, err := os.ReadDir(path)
	<-sem // Release immediately after ReadDir — the expensive syscall.

	if err != nil {
		// Permission denied, broken symlinks, etc. — skip silently.
		return
	}

	var wg sync.WaitGroup

	for _, entry := range entries {
		// Skip symlinks entirely to avoid cycles and double-counting.
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}

		if entry.IsDir() {
			wg.Add(1)
			go func(name string) {
				defer wg.Done()
				scanDir(filepath.Join(path, name), totalSize, sem)
			}(entry.Name())
		} else {
			// Use entry.Info() which calls lstat — fast for regular files.
			info, err := entry.Info()
			if err != nil {
				continue
			}
			totalSize.Add(info.Size())
		}
	}

	wg.Wait()
}

// formatSize converts bytes to GB string.
func formatSize(bytes int64) string {
	const GB = 1024 * 1024 * 1024
	return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
}

// parseBack extracts the numeric GB value from a formatted size string for sorting.
func parseBack(s string) float64 {
	var val float64
	fmt.Sscanf(s, "%f", &val)
	return val
}

// writeJSON marshals data to pretty JSON and writes it to the given path.
func writeJSON(path string, data any) {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: json marshal: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error: write %s: %v\n", path, err)
		os.Exit(1)
	}
}
