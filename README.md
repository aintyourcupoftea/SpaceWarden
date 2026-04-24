<div align="center">

# 🛡️ SpaceWarden

**Hyper-fast, concurrent disk space scanner built in Go.**

Scan hundreds of gigabytes in seconds. No dependencies. Single binary.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Zero Dependencies](https://img.shields.io/badge/Dependencies-Zero-2ea44f?style=for-the-badge)
![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS-lightgrey?style=for-the-badge)
![License](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)

---

**152 GB scanned across 41 directories in 1.5 seconds.**

</div>

## ⚡ What is SpaceWarden?

SpaceWarden is a blazing-fast CLI tool that scans all top-level directories in a given path, computes their total sizes using massively parallel goroutines, and produces two clean JSON reports:

- **`report.json`** — Every scanned folder with its size, sorted largest-first.
- **`cleanup_needed.json`** — Only folders exceeding your defined threshold.

Built for sysadmins, DevOps engineers, and anyone managing machines with terabytes of data who needs answers *now* — not in 10 minutes.

---

## 🚀 Quick Start

### Option 1 — Run directly (Go installed)

```bash
go run main.go -dir /home -threshold 20
```

### Option 2 — Build a binary

```bash
go build -o SpaceWarden main.go
./SpaceWarden -dir /home -threshold 20
```

### Option 3 — Static binary for machines without Go

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags='-extldflags "-static"' -o SpaceWarden main.go
scp SpaceWarden user@server:/usr/local/bin/
```

---

## 🔐 Deploying as Root on a Linux Machine

If you're logged in as a non-root user (e.g. `vmadmin`) with `sudo` access:

```bash
# 1. Clone the repo
cd ~
git clone <your-repo-url> SpaceWarden
cd SpaceWarden

# 2. Build the binary
go build -o SpaceWarden main.go

# 3. Install it system-wide (puts it in root's PATH)
sudo cp SpaceWarden /usr/local/bin/SpaceWarden
sudo chmod +x /usr/local/bin/SpaceWarden

# 4. Switch to root and run it
sudo -i
SpaceWarden -dir /home -threshold 20
```

> **Why `/usr/local/bin`?** It's the standard location for locally-installed binaries. It's in every user's `PATH` — including root's — and survives system updates.

---

## 🎯 Usage

```bash
SpaceWarden -dir <path> [-exclude <folders>] [-threshold <GB>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-dir` | `.` | Root directory to scan |
| `-exclude` | — | Comma-separated top-level folder names to skip |
| `-threshold` | `20` | Size threshold in GB — folders above this land in `cleanup_needed.json` |

### Examples

```bash
# Scan /home, flag anything over 50 GB
./SpaceWarden -dir /home -threshold 50

# Scan current directory, skip node_modules and .cache
./SpaceWarden -exclude node_modules,.cache

# Scan /var, flag anything over 10 GB, skip logs
./SpaceWarden -dir /var -exclude log -threshold 10
```

---

## 📊 Output

Both reports are written to `/tmp/` as pretty-printed JSON arrays, sorted by size descending.

### `/tmp/report.json`

```json
[
  { "user": ".local",      "size": "102.43 GB" },
  { "user": "Downloads",   "size": "34.13 GB"  },
  { "user": "Documents",   "size": "11.67 GB"  },
  { "user": "Development", "size": "3.06 GB"   }
]
```

### `/tmp/cleanup_needed.json`  *(threshold: 20 GB)*

```json
[
  { "user": ".local",    "size": "102.43 GB" },
  { "user": "Downloads", "size": "34.13 GB"  }
]
```

---

## 🏗️ Architecture

```
                    ┌─────────────────────────────┐
                    │         main()               │
                    │  Parse flags, read top-level  │
                    │  entries, dispatch workers     │
                    └──────────┬──────────────────┘
                               │
                    ┌──────────▼──────────────────┐
                    │     Goroutine per folder      │
                    │   Each gets its own atomic    │
                    │   counter — zero contention   │
                    └──────────┬──────────────────┘
                               │
              ┌────────────────┼────────────────┐
              ▼                ▼                ▼
        ┌──────────┐    ┌──────────┐    ┌──────────┐
        │ scanDir  │    │ scanDir  │    │ scanDir  │
        │ (recursive)   │ (recursive)   │ (recursive)
        │ goroutines│    │ goroutines│    │ goroutines│
        └──────────┘    └──────────┘    └──────────┘
              │                │                │
              └────────────────┼────────────────┘
                               │
                    ┌──────────▼──────────────────┐
                    │  Semaphore (NumCPU × 4)      │
                    │  Caps concurrency to avoid   │
                    │  file descriptor exhaustion   │
                    └──────────────────────────────┘
```

### Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **`atomic.Int64`** per folder | Lock-free size accumulation — no mutex contention across goroutines |
| **Semaphore channel** (`NumCPU × 4`) | Keeps all CPU cores saturated without exhausting OS file descriptors |
| **`os.ReadDir`** over `filepath.Walk` | Returns `DirEntry` (lazy stat), avoids unnecessary allocations |
| **Symlink skipping** | Prevents infinite cycles and double-counting in recursive walks |
| **Exclude at top-level only** | Subfolders with matching names inside other dirs are still counted |
| **Zero external dependencies** | stdlib-only — compiles anywhere Go runs, no `go mod tidy` surprises |

---

## ⚡ Performance

| Test Scenario | Directories | Total Size | Time |
|--------------|-------------|------------|------|
| `/tmp` | 2 | ~1 GB | **< 1ms** |
| `~` (home directory) | 41 | ~152 GB | **1.5s** |

> Benchmarked on a Linux machine. Results verified against `du -sh` output.

---

## 📋 Requirements

- **Go 1.21+** (to run from source)
- **OR** no requirements at all (pre-built static binary)

---

## 🤝 Contributing

1. Fork it
2. Create your feature branch (`git checkout -b feature/awesome-thing`)
3. Commit your changes (`git commit -am 'Add awesome thing'`)
4. Push to the branch (`git push origin feature/awesome-thing`)
5. Open a Pull Request

---

<div align="center">

**Built with 🧠 and Go.**

*Because `du -sh *` is too slow.*

</div>
