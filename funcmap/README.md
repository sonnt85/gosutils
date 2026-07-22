# funcmap

[![Go Reference](https://pkg.go.dev/badge/github.com/sonnt85/gosutils/funcmap.svg)](https://pkg.go.dev/github.com/sonnt85/gosutils/funcmap)

Generic task wrapper — bind a function together with its arguments into a single reusable `*Task[K]` value. Call it later, wait for completion, retrieve results by task ID.

## Motivation

Go doesn't have a built-in way to say "here is a function, here are its arguments, remember them, and give me a handle I can pass around, execute later, or store in a map." The closest options each fall short:

- **Anonymous closure** (`func() { doWork(a, b) }`) — captures args, but the result is opaque: no ID, no way to update args, no built-in "wait for done" signal, no result retrieval, no panic recovery, no `IsFinish` check.
- **`errgroup.Group` / `sync.WaitGroup`** — coordinate N goroutines, but don't wrap a specific function-with-args as a first-class value. You still need your own registry to look up "task 42".
- **Command pattern by hand** — works, but every project reinvents the same struct (name, id, params, results, done channel, mutex).

`funcmap.Task[K]` fills that gap: **one struct that owns the function + arguments + results + completion state**, so you can put many of them in a `map[K]*Task[K]` (which is exactly what `goramcache.CacheFuncs` and `goring.EventWorker[K]` do internally).

## Installation

```bash
go get github.com/sonnt85/gosutils/funcmap
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/sonnt85/gosutils/funcmap"
)

func greet(prefix, name string) (int, string) {
    return 9, prefix + name
}

func main() {
    // Bind function + args into a task; nil fid → auto-generated ID
    task, _ := funcmap.NewTask[uint32]("greeting", nil, greet, "hello ", "world")

    results, _ := task.Call()             // execute
    fmt.Println(task.Id, results)         // e.g. 3221225472 [9 hello world]
}
```

## Features

- Generic ID type: `string` (UUID), `uint32`/`uint64` (random via `endec`), or any `constraints.Ordered`
- One handle owns: function, args, results, done flag, ignore flag, message, error
- Runtime parameter update — call the same task with different args later
- Panic-safe: `Call()` recovers panics and returns them as `error`
- Blocking wait for completion via `sync.Cond` broadcast
- Introspection: current args, current results, done state, custom message, ignore flag

## Usage

### Bind now, execute later, wait from another goroutine

Use case: schedule work now, kick it off later, block elsewhere until it finishes.

```go
task, _ := funcmap.NewTask[string]("job-42", nil, slowFn, arg1, arg2)

go func() {
    _, _ = task.Call() // blocks the goroutine until slowFn returns
}()

// somewhere else — blocks until Call() completes
results := task.WaitTaskFinishThenGetValues()
```

### Store many tasks in a registry, dispatch by ID

Use case: named callback registry, plugin dispatcher, cron event dispatcher.

```go
registry := map[string]*funcmap.Task[string]{}

t1, _ := funcmap.NewTask[string]("on_login", nil, handleLogin, user, session)
registry[t1.Id] = t1

t2, _ := funcmap.NewTask[string]("on_logout", nil, handleLogout, user)
registry[t2.Id] = t2

// Later, on an event:
if t, ok := registry[eventID]; ok {
    _, _ = t.Call()
}
```

### Update parameters and re-run

Use case: a memoized computation whose input changes over time.

```go
task, _ := funcmap.NewTask[uint32]("compute", nil, expensiveFn, initialArg)

for _, arg := range newArgs {
    _ = task.ParamsUpdate(arg)
    results, _ := task.Call()
    process(results)
}
```

## API Reference

Grouped by role, one-line summaries. See `go doc github.com/sonnt85/gosutils/funcmap` for full signatures.

**Task creation**
- `NewTask[K](name, fid, f, params...)` — bind function + args. Validates arity against `f`'s signature (variadic-aware).
- `FIDAuto[K](fid)` — return `fid` or a default generator (UUID / random uint).

**Execution & completion**
- `Task.Call()` — invoke wrapped function; results stored on the task, `done` set, waiters broadcast.
- `Task.WaitTaskFinishThenGetValues()` — block until `done`, return results.
- `Task.IsFinish()` — non-blocking done check.

**Parameters & introspection**
- `Task.ParamsUpdate(params...)` — replace args; re-validate against `f`.
- `Task.GetFuncDetail()` — snapshot of `(f, params, results, err)`.
- `Task.GetRetValues()` — non-blocking `(results, ok)` where `ok == done`.

**Flags & metadata**
- `Task.SetIgnore(bool)` / `Task.IsIgnore()` — external skip flag; funcmap itself does not consume it. Consumers like `EventWorker` may skip ignored tasks in the dispatcher.
- `Task.SetMsg(string)` / `Task.GetMsg()` — free-form message slot for progress/error context.

**Lifecycle**
- `Task.ResetParasIfFinish()` — clear params IF done; returns whether it happened.
- `Task.GCTask()` — clear params + `f` reflect.Value; call after fully drained to let GC free captured closures sooner.

**Errors**
- `ErrParamsNotAdapted` — returned by `NewTask` / `ParamsUpdate` when arg count doesn't match `f`'s arity.

## Design Decisions & Trade-offs

**Why `reflect` instead of generic function types?**
Generic function types (`func(A, B) (R, E)`) would force each task to be strongly typed at declaration site. That precludes storing heterogeneous tasks in `map[K]*Task[K]` — which is the whole point. `reflect` pays a per-call cost but gains a uniform type across all tasks.

**Why `sync.Cond` (not `chan struct{}`)?**
`Call()` may run many times over a task's lifetime (`ParamsUpdate` + `Call` again). A `chan struct{}` would need to be recreated after each run. `sync.Cond` naturally supports "wait for the next completion" semantics without re-init.

**Why `[]interface{}` return, not a typed generic return?**
Return values are extracted via `reflect.Value.Call`, which yields `[]reflect.Value`. Converting to a typed generic result would require the caller's type at the task-creation site — undermining the map-of-heterogeneous-tasks use case. Type-assert at the call site instead: `results[0].(int)`.

**Why K constrained to `constraints.Ordered`?**
IDs are used as map keys and often need natural comparison (e.g. `SubmitWithTimeout` in `EventWorker` may sort by expiration). `Ordered` covers `string`, all integer types, `float`. Sufficient for practical task IDs.

## Concurrency & Thread-Safety

Every `Task[K]` holds its own `*sync.RWMutex` and `*sync.Cond`.

- All `Task` methods lock as needed.
- `Call()` uses a deferred lock+broadcast so waiters wake up whether the function returned or panicked.
- Multiple goroutines may `Call` the same task **serially**, but concurrent `Call` on the same task is not defined — the internal `params` slice is read+copied under `RLock`, but concurrent writes would race with `ParamsUpdate`.

## Performance Notes

- `Call()` uses `reflect.Value.Call` — allocation-heavy, ~microsecond overhead per invocation. Fine for coarse-grained work (HTTP handlers, cron ticks, background jobs). Do not use in tight loops.
- No benchmarks published in-repo. Measure in your workload.

## Gotchas

- **Not zero-value safe.** `Task[K]{}` panics on any method (nil `RWMutex`/`Cond`). Always use `NewTask`.
- **Arg kinds not fully validated at construction.** `NewTask` checks arity but not per-argument kind. Passing `func(int)` and `params = []interface{}{"str"}` will panic at `Call()`, recovered as `err`. Prefer testing early.
- **`WaitTaskFinishThenGetValues` has no timeout.** Wrap with `context.Context` at the caller if you need a deadline.
- **`Call()` result is retained on the task.** Repeated calls without `ResetParasIfFinish` or `GCTask` accumulate references to the last result. Not a leak for small results; matters if results are large blobs.

## Ecosystem Usage

Two libraries in the same author's ecosystem build on `*Task[K]`:

- **`goramcache.CacheFuncs`** — cache of named tasks with TTL; `AddFunc(name, fn, params...)` returns `*funcmap.Task[string]`; `CallFunc(name)` looks up and invokes synchronously.
- **`goring.EventWorker[K]`** — bounded worker pool; `Submit(...)` wraps `f` + params in a `*funcmap.Task[K]` and dispatches to a worker. All the `Task` methods above are available on the returned handle.

If you're using either of those libs, you're already using `funcmap` transitively — this doc is the reference for what `*Task[K]` can do.

## Author

**sonnt85** — [thanhson.rf@gmail.com](mailto:thanhson.rf@gmail.com)

## License

MIT.
