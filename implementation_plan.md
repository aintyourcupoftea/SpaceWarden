# Hyper-Fast Directory Size Scanner (spacewarden2.0)

A high-performance, multi-threaded Go CLI tool to scan all top-level directories within a given folder, compute their sizes in parallel, and produce JSON reports. Designed to handle **8TB+ directories**.

## Performance Strategy

- **Worker pool** of `runtime.NumCPU()` goroutines walking subdirectories concurrently
- **`syscall.Stat_t`** for raw stat calls — avoids overhead of `os.Stat` / `fs.FileInfo` wrappers
- **`os.ReadDir`** (returns `DirEntry`) instead of `os.ReadFile` — avoids unnecessary allocations
- **Atomic counters** (`atomic.Int64`) per top-level folder — no mutexes, no contention
- **Exclude matching only at the top level** — subfolders with the same name inside other dirs are still counted
- **Buffered channel work queue** — folders are enqueued and consumed by workers with zero coordination overhead

## CLI Interface

```
spacewarden2.0 -dir /tmp -exclude folder2,node_modules -threshold 20
```

| Flag | Default | Description |
|------|---------|-------------|
| `-dir` | `.` | Root directory to scan |
| `-exclude` | (none) | Comma-separated list of top-level folder names to skip |
| `-threshold` | `20` | Size threshold in GB — folders above this go into `cleanup_needed.json` |

## Output Files

### `/tmp/report.json`
```json
[
  { "folder": "folder1", "size": "4.23 GB" },
  { "folder": "folder3", "size": "128.91 GB" }
]
```

### `/tmp/cleanup_needed.json`
```json
[
  { "folder": "folder3", "size": "128.91 GB" }
]
```

## Proposed Changes

### Main Package

#### [NEW] [main.go](file:///Users/aintyourcupoftea/Development/spacewarden2.0/main.go)

Single-file implementation containing:

1. **`main()`** — parses flags, reads top-level entries, dispatches workers, writes JSON reports
2. **`scanDir(path string, size *atomic.Int64, sem chan struct{})`** — recursively walks a directory tree using `os.ReadDir`, spawns goroutines for subdirs bounded by the semaphore, accumulates file sizes via atomic add
3. **`formatSize(bytes int64) string`** — converts bytes → human-readable string (GB/MB/KB)
4. **JSON structs** — `FolderReport { Folder string; Size string }`

Key design decisions:
- Semaphore channel (`chan struct{}`) sized to `runtime.NumCPU() * 4` caps goroutine count to avoid fd exhaustion while keeping all cores saturated
- Exclude set stored as `map[string]bool` for O(1) lookup — only checked against top-level folder basenames
- `sync.WaitGroup` coordinates completion of all recursive walks before writing output

#### [NEW] [go.mod](file:///Users/aintyourcupoftea/Development/spacewarden2.0/go.mod)

Standard Go module file — no external dependencies (stdlib only).

## Verification Plan

### Automated Test

Run the tool against `/tmp` and verify it produces valid JSON:

```bash
cd /Users/aintyourcupoftea/Development/spacewarden2.0
go build -o spacewarden2 . && ./spacewarden2 -dir /tmp -threshold 20
cat /tmp/report.json | python3 -m json.tool
cat /tmp/cleanup_needed.json | python3 -m json.tool
```

### Manual Verification

1. Build and run against a known directory with a few folders
2. Cross-check one folder's reported size against `du -sh <folder>` output
3. Verify excluded folders do not appear in reports
4. Verify folders above threshold appear in `cleanup_needed.json`
