# Walkthrough — spacewarden2.0

## What Was Built

A **hyper-fast, multi-threaded Go CLI** that scans all top-level directories in a given folder, computes their total sizes concurrently, and outputs two JSON reports.

### Files Changed

| File | Action |
|------|--------|
| [go.mod](file:///Users/aintyourcupoftea/Development/spacewarden2.0/go.mod) | **Created** — Go module, zero external deps |
| [main.go](file:///Users/aintyourcupoftea/Development/spacewarden2.0/main.go) | **Created** — Full scanner implementation |

## Usage

```bash
# Basic scan
./spacewarden2 -dir /path/to/scan -threshold 20

# With exclusions
./spacewarden2 -dir /path/to/scan -exclude node_modules,.cache -threshold 20
```

| Flag | Default | Description |
|------|---------|-------------|
| `-dir` | `.` | Root directory to scan |
| `-exclude` | — | Comma-separated top-level folders to skip |
| `-threshold` | `20` | GB threshold for `cleanup_needed.json` |

## Performance Results

| Test | Dirs | Time |
|------|------|------|
| `/tmp` (2 dirs) | 2 | **< 1ms** |
| `~` (41 dirs, ~152 GB) | 41 | **1.5s** |

Workers spawned: **40** (NumCPU × 4)

## Verification

- ✅ **Build**: Clean compile, zero warnings
- ✅ **JSON validity**: Both outputs are valid JSON arrays
- ✅ **Exclude flag**: Excluded folders don't appear in reports
- ✅ **Threshold filter**: `cleanup_needed.json` correctly captures folders > threshold
- ✅ **Cross-check with `du -sh`**: Sizes match system utility output
- ✅ **Sorted output**: Results sorted by size descending

### Sample Output — `/tmp/cleanup_needed.json` (threshold 1 GB)

```json
[
  { "folder": ".local",       "size": "102.43 GB" },
  { "folder": "Downloads",    "size": "34.13 GB"  },
  { "folder": "Documents",    "size": "11.67 GB"  },
  { "folder": "Development",  "size": "3.06 GB"   }
]
```
