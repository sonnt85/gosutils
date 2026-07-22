# lockedfile

[![Go Reference](https://pkg.go.dev/badge/github.com/sonnt85/gosutils/lockedfile.svg)](https://pkg.go.dev/github.com/sonnt85/gosutils/lockedfile)

Atomic file operations with **cross-process** file locking — reads, writes, and read-modify-write cycles that stay consistent across concurrent processes on the same host.

## Motivation

Go's `os` package gives you file I/O, and `sync.Mutex` gives you in-process mutual exclusion — but neither handles the common case: **two processes on the same machine, writing to the same file, must not corrupt each other**.

Concrete examples:
- Two invocations of a CLI tool sharing a JSON cache file (`~/.mytool/cache.json`).
- A daemon and a control command mutating the same PID or state file.
- A build system where several `go build` runs share a `go.sum`-style manifest.

Without OS-level file locks (`flock` on Unix, `LockFileEx` on Windows), interleaved writes silently corrupt the file. `lockedfile` bundles the platform-specific lock syscalls behind a simple API:

```go
lockedfile.Read(path)             // lock, read, unlock
lockedfile.Write(path, data, 0644) // lock, overwrite, unlock
lockedfile.Transform(path, fn)     // lock, read, transform, write, unlock — one atomic RMW
```

This is a port/derivative of Go's `internal/lockedfile` (from the `cmd/go` toolchain), exposed as an external package.

## Installation

```bash
go get github.com/sonnt85/gosutils/lockedfile
```

## Quick Start

```go
package main

import (
    "bytes"
    "fmt"
    "github.com/sonnt85/gosutils/lockedfile"
)

func main() {
    // Write with exclusive lock (creates file if missing).
    _ = lockedfile.Write("/tmp/counter.txt", bytes.NewReader([]byte("0\n")), 0644)

    // Atomic read-modify-write — no other process can interleave.
    _ = lockedfile.Transform("/tmp/counter.txt", func(cur []byte) ([]byte, error) {
        // increment
        n := 0
        fmt.Sscanf(string(cur), "%d", &n)
        return []byte(fmt.Sprintf("%d\n", n+1)), nil
    })

    data, _ := lockedfile.Read("/tmp/counter.txt")
    fmt.Println(string(data)) // "1\n"
}
```

## Features

- **Atomic read-modify-write** via `Transform` — the file stays locked for the duration of the transform function
- **Cross-process file locks** — Unix `fcntl` / Linux `flock`; Windows `LockFileEx`; Plan 9 supported
- **Read locks** (shared) and **write locks** (exclusive) — multiple readers, one writer
- **Timeouts** via `LockTimeout` / `RLockTimeout` — bounded wait for contended locks
- **`Mutex` type** — general-purpose cross-process mutex around any well-known lock file
- **`File` type** — locked `*os.File` for arbitrary I/O; `Close()` releases lock
- Best-effort restore on transform failure

## Usage

### Concurrent-safe cache read

Multiple processes can call `Read` at the same time (shared lock); a writer blocks until all readers finish.

```go
data, err := lockedfile.Read("/var/lib/app/cache.json")
```

### Concurrent-safe overwrite

Excludes all readers and other writers.

```go
err := lockedfile.Write(
    "/var/lib/app/state.bin",
    bytes.NewReader(newState),
    0644,
)
```

### Atomic increment / update

Use case: shared counter, config patch, JSON edit that must never lose an update.

```go
err := lockedfile.Transform("/var/lib/app/counter.json", func(cur []byte) ([]byte, error) {
    var c struct{ N int }
    if len(cur) > 0 {
        _ = json.Unmarshal(cur, &c)
    }
    c.N++
    return json.Marshal(c)
})
```

If the transform function returns an error, the file is left as-is (best effort).

### Cross-process mutex

Use case: guard a resource that isn't a file — a directory operation, a shared external tool, etc.

```go
mu := &lockedfile.Mutex{Path: "/var/run/mytool.lock"}
unlock, err := mu.Lock()
if err != nil { return err }
defer unlock()

// exclusive access — no other process holding mu.Lock can proceed
doExpensiveOp()
```

### Lock with timeout

Use case: don't wait forever for a stuck lock.

```go
f, unlock, err := lockedfile.LockTimeout(
    "/var/run/mytool.lock",
    5*time.Second,
    100*time.Millisecond, // poll interval
)
if err != nil {
    // lock not acquired in 5 seconds
    return err
}
defer unlock()
```

### Long-lived locked handle

Use case: hold a lock while streaming, then release.

```go
f, err := lockedfile.OpenFile("/var/log/app.jsonl", os.O_WRONLY|os.O_APPEND, 0644)
if err != nil { return err }
defer f.Close() // releases lock

_, _ = fmt.Fprintln(f, `{"event": "start"}`)
// ... more writes ...
```

## API Reference

**One-shot helpers**
- `Read(name) ([]byte, error)` — read-lock, read whole file, unlock.
- `Write(name, content io.Reader, perm fs.FileMode) error` — write-lock, overwrite, unlock. Creates file if missing.
- `Transform(name, t func([]byte) ([]byte, error)) error` — write-lock, read, call `t`, write result, unlock. Best-effort restore on `t` error.

**Long-lived handles**
- `Open(name) (*File, error)` — read-lock; use `File` for I/O; `Close` releases lock.
- `Create(name) (*File, error)` — write-lock, like `os.Create`.
- `Edit(name) (*File, error)` — write-lock, does not truncate; `O_RDWR`.
- `OpenFile(name, flag, perm) (*File, error)` — flags decide read vs write lock (`O_WRONLY`/`O_RDWR` → write-lock).

**Lock primitives (no file I/O required)**
- `Lock(path) (f *File, unlock func(), err error)` — exclusive lock on `path`. Call `unlock()` when done.
- `LockTimeout(path, timeout, intervalCheck ...) (f, unlock, err)` — same, bounded wait.
- `RLockTimeout(path, timeout, intervalCheck ...) (f, unlock, err)` — shared (read) lock with timeout.

**Cross-process mutex**
- `Mutex{Path string}` — set `Path` to a well-known lock file. `mu.Lock()` returns `(unlock, err)`. Not documented in `go doc -all` output above but is exported.

**File methods**
- `(*File).Close() error` — release lock and close descriptor. Subsequent `Close` calls return non-nil error.
- File embeds `*os.File` — use standard I/O methods (`Read`, `Write`, `Seek`, `Sync`, `Truncate`, ...).

**Errors**
- `MISSING_FILE` — returned when a path argument is empty.

## Design Decisions & Trade-offs

**Cross-process locks are advisory, not mandatory.**
Any process that bypasses `lockedfile` (uses plain `os.OpenFile`) can still corrupt the file. All participants must cooperate — this is a Unix limitation, not a library one.

**Locks are per-file-handle, not per-path.**
The lock is bound to the OS file descriptor. If a `*File` is closed, the lock is released. Do not stash the underlying `*os.File` and close it separately.

**`Transform` reads the whole file into memory.**
Not suitable for multi-GB files. For those, use `OpenFile` + explicit incremental I/O with the lock held.

**Best-effort restore on `Transform` error.**
If `t(cur)` returns an error, the library tries to leave the file with its pre-transform contents. "Best-effort" because a partial write may have already happened on some platforms. Do not rely on it for critical rollback semantics — use `os.Rename` from a temp file if you need true atomicity.

**Timeouts poll.**
`LockTimeout` / `RLockTimeout` use polling with `intervalCheck` (default 100ms if not supplied). No native "wait until lock available" primitive across all supported OSes.

## Concurrency & Thread-Safety

- Multiple goroutines in the SAME process calling `Read`/`Write`/`Transform` on the same path serialize correctly through the OS lock — but each goroutine gets its own file descriptor. Within a process, prefer `sync.Mutex` for goroutine coordination on top of `lockedfile` for cross-process safety.
- The `Mutex` type coordinates across processes only; add a `sync.Mutex` for in-process mutual exclusion if the same process might contend with itself.

## Platform Support

- **Linux, macOS, other Unix**: `fcntl(F_SETLK)`-style advisory locks.
- **Windows**: `LockFileEx` (mandatory in Windows semantics — a locked file cannot be accessed at all by non-participating processes).
- **Plan 9**: dedicated `lockedfile_plan9.go`.

Behavior differs slightly per platform (mandatory vs advisory) but the Go API is uniform.

## Gotchas

- **Lock leaks on `os.Exit` / kill -9.** OS may take some time to release the lock. If your program can be killed abruptly, callers must handle "lock file exists but no owner" cases (e.g., stale PID file cleanup).
- **`Transform` overwrite is single-file.** No rename-based atomic swap; the file being transformed IS the file being overwritten. A power failure mid-write can corrupt the file. For critical data, wrap with your own tmpfile + `os.Rename`.
- **`Close()` errors are informational.** Second close returns error; don't rely on it for control flow.
- **Read lock does not prevent read by non-participating readers.** As noted in Design — advisory locks only affect processes that also use `lockedfile`.

## Origin

Derived from `cmd/go/internal/lockedfile` in the Go toolchain, exposed as an external package.

## Author

**sonnt85** — [thanhson.rf@gmail.com](mailto:thanhson.rf@gmail.com)

## License

BSD-style (Go project license).
