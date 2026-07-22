# sregexp

[![Go Reference](https://pkg.go.dev/badge/github.com/sonnt85/gosutils/sregexp.svg)](https://pkg.go.dev/github.com/sonnt85/gosutils/sregexp)

Lazy-compiled regexp wrapper — declare `regexp` variables at package level without paying the compilation cost until first use.

## Motivation

The idiomatic way to reuse a compiled `*regexp.Regexp` in Go is a package-level variable:

```go
var uuidRe = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-...`)
```

This compiles the pattern at **package init** — every time the binary starts, even if the regexp is never used in this run. For programs with many rarely-used regexes (linters, CLI tools with dozens of pattern-matching flags, code generators), init time balloons.

`sregexp.New(pattern)` defers compilation to the first method call. If the process never exercises that code path, the regexp is never compiled. If it does, compilation happens once, then subsequent calls are as fast as `regexp.Regexp` directly.

**Bonus**: when the code is running under `go test`, compilation happens immediately (at `New`) — so tests still catch invalid patterns without waiting for a lazy path.

This is a direct port of `internal/lazyregexp` from the Go toolchain itself. `sregexp` makes it available outside `internal/`.

## Installation

```bash
go get github.com/sonnt85/gosutils/sregexp
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/sonnt85/gosutils/sregexp"
)

// Declared but not compiled yet.
var uuidRe = sregexp.New(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)

func main() {
    // Compilation happens here, on the first method call.
    fmt.Println(uuidRe.MatchString("550e8400-e29b-41d4-a716-446655440000")) // true
    fmt.Println(uuidRe.MatchString("not-a-uuid"))                            // false
}
```

## Features

- Same API surface as `regexp.Regexp` for common ops (`Find*`, `Match*`, `Replace*`, `Split`, `SubexpNames`)
- Deferred compilation via `sync.Once`
- Under `go test`, compiles immediately so tests catch invalid patterns
- Zero cost if the regexp is never invoked
- Retrieve the underlying `*regexp.Regexp` via `.Regexp()` if you need something the wrapper doesn't expose

## Usage

### Package-level regexes in a CLI tool

Use case: 20 subcommand flags each with their own validation regex, only 1-2 exercised per invocation.

```go
var (
    emailRe   = sregexp.New(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
    hostnameRe = sregexp.New(`^[a-z0-9][a-z0-9-]{0,63}$`)
    pathRe     = sregexp.New(`^/[^\x00]*$`)
    // ... 17 more
)

func cmdCheckEmail(input string) bool { return emailRe.MatchString(input) }
```

Only the regex(es) actually used compile; the rest stay dormant.

### Interop with stdlib functions expecting `*regexp.Regexp`

```go
re := sregexp.New(`(\d+)`)
underlying := re.Regexp() // forces compile now
someFunc(underlying)      // pass to code that wants *regexp.Regexp
```

## API Reference

- `New(pattern string) *Regexp` — declare a lazy regexp; does not compile until first use.
- `(*Regexp).Regexp() *regexp.Regexp` — compile if needed, return the underlying regexp for advanced use.

**Match / Find** (mirror `regexp.Regexp` behavior):
- `Find(b []byte) []byte`, `FindAll(b []byte, n int) [][]byte`, `FindAllIndex(b []byte, n int) [][]int`
- `FindString(s string) string`, `FindAllString(s string, n int) []string`
- `FindStringSubmatch(s string) []string`, `FindAllStringSubmatch(s string, n int) [][]string`, `FindStringSubmatchIndex(s string) []int`
- `FindSubmatch(s []byte) [][]byte`
- `Match(b []byte) bool`, `MatchString(s string) bool`

**Replace / Split** (see `go doc github.com/sonnt85/gosutils/sregexp` for exact signatures):
- `ReplaceAll(src, repl []byte) []byte`, `ReplaceAllString(src, repl string) string`
- `ReplaceAllStringFunc(src string, repl func(string) string) string`
- `Split(s string, n int) []string`
- `SubexpNames() []string`

## Design Decisions & Trade-offs

**Why not just call `regexp.MustCompile` inline in the function?**
- `MustCompile` inside a function recompiles on every call — slower than a package-level `MustCompile` executed once at init.
- `sregexp.New(...)` at package level gives you the best of both: compiled once, but only when first needed.

**Why compile immediately under `go test`?**
Package init already runs during tests, so the "startup cost" argument doesn't apply. Compiling immediately gives you loud, immediate failure on an invalid pattern (compile panic), instead of a deferred surprise the first time a rarely-executed code path is hit. The test detection uses `testing.Testing()` (or similar) — see source.

**Same public methods as `regexp.Regexp`.**
Migration from `regexp` → `sregexp` is mechanical: change `regexp.MustCompile(...)` to `sregexp.New(...)`. All call sites keep working.

## Concurrency & Thread-Safety

`sync.Once`-guarded compilation makes `New`-then-concurrent-use safe. The underlying `*regexp.Regexp` is itself safe for concurrent use by many goroutines.

## Gotchas

- **Compile error surfaces on first use, not on `New`.** If your program has a bad pattern in a rarely-invoked code path, you won't see it until that path fires — in production, at 3am. **Always cover each `sregexp.New` with a test** that at least calls `.MatchString("")` on it (that forces compile) or use `.Regexp()` at init if you want eager panic behavior.
- **Not a drop-in for `regexp.Compile` (returns `(*Regexp, error)`).** `sregexp.New` returns just `*Regexp` and panics on invalid pattern at first use — like `MustCompile`.
- **Slight per-call overhead.** Each method call checks `sync.Once`. Negligible in practice, but if you're microbenchmarking regex-heavy hot paths, verify.

## Origin

Ported from the Go toolchain's `internal/lazyregexp`. This wrapper exists solely to make it usable outside the standard library.

## Author

**sonnt85** — [thanhson.rf@gmail.com](mailto:thanhson.rf@gmail.com)

## License

MIT.
