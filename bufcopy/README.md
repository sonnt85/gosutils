# bufcopy

[![Go Reference](https://pkg.go.dev/badge/github.com/sonnt85/gosutils/bufcopy.svg)](https://pkg.go.dev/github.com/sonnt85/gosutils/bufcopy)

Drop-in replacement for `io.Copy` that reuses intermediate buffers via a pool — same semantics, one less allocation per call.

## Motivation

Standard `io.Copy` allocates a fresh 32 KB buffer on every call unless the source implements `io.WriterTo` or the destination implements `io.ReaderFrom`. In hot copy paths (streaming proxies, tar/zip pipelines, HTTP request forwarding, WebSocket bidirectional pumps) this shows up as a measurable allocation stream.

`bufcopy` pools those intermediate buffers with `goramcache.Pool` — TTL-based reclaim, capped size, safe for concurrent callers. The public API mirrors `io.Copy` exactly, so it's a mechanical swap:

```go
// before
n, err := io.Copy(dst, src)

// after
n, err := bufcopy.Copy(dst, src)
```

Additionally provides `Copy2Way` for bidirectional pump patterns (TCP proxy, WebSocket tunneling) in one call.

## Installation

```bash
go get github.com/sonnt85/gosutils/bufcopy
```

## Quick Start

```go
package main

import (
    "os"
    "github.com/sonnt85/gosutils/bufcopy"
)

func main() {
    src, _ := os.Open("input.bin")
    dst, _ := os.Create("output.bin")
    _, _ = bufcopy.Copy(dst, src)
}
```

## Features

- Pooled 32 KB buffers by default; reuses across calls, no per-call allocation
- Configurable buffer size via `New(default_size)` for custom instances
- Fast-path preserved: `WriterTo` / `ReaderFrom` detection just like stdlib `io.Copy`
- `Copy2Way` — bidirectional copy between two `io.ReadWriter` peers (proxy pattern)
- Optional auto-close: pass `checkCloser=true` to close `src`/`dst` when done

## Usage

### One-off copy (package-level, lazy-init pool)

```go
_, err := bufcopy.Copy(dst, src)
```

### Own pool with custom buffer size

Use case: you know your workload has 4 KB records — smaller pool reduces memory footprint.

```go
bc := bufcopy.New(4 * 1024)
_, _ = bc.Copy(dst, src)
```

### Auto-close after copy

Use case: streaming a file to an HTTP response and you want the source closed when done.

```go
resp.WriteHeader(200)
_, _ = bufcopy.Copy(resp, srcFile, true) // src closed after copy
```

### Bidirectional proxy (TCP tunnel, WebSocket relay)

Use case: two `net.Conn`s that should be piped to each other.

```go
_, err := bufcopy.Copy2Way(conn1, conn2, true) // both closed when either side ends
```

## API Reference

**Package-level (uses a global lazy-init pool)**
- `Copy(dst io.Writer, src io.Reader, checkCloser ...bool) (written int64, err error)` — semantically identical to `io.Copy`, plus optional auto-close.
- `Copy2Way(rw1, rw2 io.ReadWriter, checkCloser ...bool) (written int64, err error)` — bidirectional; returns sum of both directions.

**Instance-based (own pool)**
- `New(default_size ...int) *BufCopy` — allocate a pool; default size 32 KB.
- `(*BufCopy).Copy(dst, src, checkCloser...)` — instance variant.
- `(*BufCopy).Copy2Way(rw1, rw2, checkCloser...)` — instance variant.

**Type**
- `BufCopy struct { *goramcache.Pool[*[]byte] }` — embeds the pool; you may access pool methods (`Get`, `Put`, `Stats`) directly if needed.

## Design Decisions & Trade-offs

**Why 32 KB default?**
Matches stdlib `io.Copy` default. Large enough to amortize syscall overhead on most hardware; small enough that idle pool memory is bounded.

**Fast-path detection matches stdlib.**
If `src.(io.WriterTo)` or `dst.(io.ReaderFrom)` succeeds, the pooled buffer is bypassed entirely — same behavior as `io.Copy`. So swapping `io.Copy` → `bufcopy.Copy` never regresses fast paths, only improves the slow path.

**`Copy2Way` uses 2 goroutines and returns on FIRST error.**
When one direction errors or hits EOF, the caller unblocks and (if `checkCloser=true`) closes both peers, which causes the other goroutine to unblock too. The returned `err` is from whichever direction finished first. If you need per-direction error accounting, use `Copy` explicitly in your own goroutines.

**Pool has TTL.**
Backing `goramcache.Pool` reclaims buffers idle for >1 minute. Under sustained load buffers stay hot; during idle periods memory returns to the heap. No manual `Reset` needed.

## Performance Notes

From the repo's benchmark:

```
BenchmarkBufCopy-12    5000000    430 ns/op    3168 B/op    3 allocs/op
BenchmarkIoCopy-12     3000000    433 ns/op    3168 B/op    3 allocs/op
```

Nearly identical ns/op for small copies (the fast-path handles them via `WriteTo`/`ReadFrom`). The win shows up in workloads that hit the slow path repeatedly — allocation count for the buffer drops from 1-per-call to amortized-near-0 across many calls sharing the pool.

Measure in your workload. If your copies are small and the source/destination both implement `WriteTo`/`ReadFrom` (bytes.Buffer, os.File, net.TCPConn on some paths), `io.Copy` and `bufcopy.Copy` are indistinguishable.

## Concurrency & Thread-Safety

The pool is safe for concurrent `Get`/`Put`. Multiple goroutines calling `Copy`/`Copy2Way` on the same `*BufCopy` (or via package-level fns) share the same pool — this is the intended usage.

## Gotchas

- **`checkCloser` closes both sides in `Copy2Way`.** Do not pass an `rw` that shouldn't be closed (e.g. `os.Stdout`) with `checkCloser=true`.
- **Global pool lazy-inits on first call.** No cleanup hook — the pool lives for the process lifetime. If you need explicit lifecycle, use `New()` and hold the pointer.
- **Return value semantics match `io.Copy`.** `written` is bytes transferred; `err` may be non-nil even when some bytes were transferred (e.g. partial write). Check both.

## Dependencies

- `github.com/sonnt85/goramcache` — for `Pool[*[]byte]`. Small, no transitive network deps.

## Author

**sonnt85** — [thanhson.rf@gmail.com](mailto:thanhson.rf@gmail.com)

## License

MIT.
