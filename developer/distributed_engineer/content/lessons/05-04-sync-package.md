---
slug: conc-sync-package
step: 5
title: The sync package — mutexes, WaitGroup, Once
summary: When channels aren't the right tool — protecting shared state with mutexes, WaitGroup, Once, and the race detector.
est_min: 360
position: 4
---

# The sync package — mutexes, WaitGroup, Once

> **Step 5 · Concurrency in Go · Week 2**
> Concept: *shared state, race conditions*

Channels are great for *passing ownership* of data. But sometimes goroutines
genuinely need to share state — a counter, a cache, a connection pool. For that you
reach for the `sync` package and a **mutex**.

## Why this matters

Shared mutable state is where concurrency bugs breed. A **data race** — two
goroutines touching the same memory, at least one writing, with no coordination —
produces corruption that's nondeterministic and hellish to debug. Knowing when and
how to lock is core competence.

## 1. The bug: a data race

```go
var count int
var wg sync.WaitGroup
for i := 0; i < 1000; i++ {
	wg.Add(1)
	go func() { defer wg.Done(); count++ }() // RACE: read-modify-write, unsynchronized
}
wg.Wait()
fmt.Println(count) // almost never 1000 — increments get lost
```

`count++` is not atomic — it's load, add, store. Two goroutines interleave and one
update clobbers the other. The result is wrong *and* varies run to run.

## 2. The detector: -race

Go ships a race detector. **Use it in tests and CI.**

```bash
go run -race .
go test -race ./...
```

It instruments memory access and reports the exact two goroutines and lines that
raced. If you remember one tool from this lesson, it's `-race`.

## 3. Mutex — mutual exclusion

A `sync.Mutex` lets only one goroutine into the critical section at a time:

```go
type Counter struct {
	mu sync.Mutex
	n  int
}

func (c *Counter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()  // always unlock — defer makes it panic-safe
	c.n++
}

func (c *Counter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}
```

`Lock`/`Unlock` around every access to the shared field — reads too, not just
writes. (The Consensus app guards its in-memory bits this way.)

## 4. RWMutex — many readers, one writer

When reads vastly outnumber writes, `sync.RWMutex` lets readers proceed in parallel
and only blocks them for writes:

```go
var mu sync.RWMutex
mu.RLock(); _ = cache[key]; mu.RUnlock()   // concurrent readers OK
mu.Lock();  cache[key] = val; mu.Unlock()  // exclusive for writes
```

Use it only when you've measured read-heavy contention; a plain `Mutex` is simpler
and often just as fast.

## 5. WaitGroup — wait for a set of goroutines

You've already used this. `Add(n)` before launching, `Done()` (deferred) inside
each goroutine, `Wait()` to block until all finish:

```go
var wg sync.WaitGroup
for _, job := range jobs {
	wg.Add(1)
	go func(j Job) { defer wg.Done(); process(j) }(job)
}
wg.Wait() // returns when the counter hits zero
```

Pitfall: call `Add` *before* the `go`, never inside the goroutine (it may not run
before `Wait`).

## 6. Once — exactly-once initialization

`sync.Once` runs a function exactly once, even if many goroutines call it
concurrently — perfect for lazy singletons:

```go
var (
	once sync.Once
	conn *DB
)
func getDB() *DB {
	once.Do(func() { conn = connect() }) // connect() runs once, ever
	return conn
}
```

## 7. Channels or mutexes?

A practical rule of thumb:
- **Passing data / ownership between goroutines, pipelines, signaling** → channels.
- **Protecting a small piece of shared state (counter, map, cache)** → mutex.

Don't force a channel where a one-line mutex is clearer, or vice versa. Use whichever
makes the code obviously correct.

## Do it yourself (≈ 6 hrs)

1. Run the racy counter above with `go run -race .` and read the report.
2. Fix it three ways: a `sync.Mutex`, then `sync/atomic.AddInt64`, then a channel —
   compare clarity.
3. Build a concurrency-safe in-memory cache (`map` + `RWMutex`) with `Get`/`Set`.
4. Work the relevant parts of [**Learn Go with Tests — Concurrency**](https://quii.gitbook.io/learn-go-with-tests/go-fundamentals/concurrency).
5. Skim the [**sync package docs**](https://pkg.go.dev/sync).

## Check yourself

- What exactly is a data race, and why is `count++` one?
- How do you run the race detector, and when should you?
- Why must you `Unlock` (ideally via `defer`), and why lock reads too?
- When is `RWMutex` worth it over `Mutex`?
- Give the rule of thumb for choosing channels vs mutexes.

Next: **context** — cancelling and bounding all this concurrent work.

## Common interview gotchas

- **"It worked in my test, so it's race-free."** A data race is nondeterministic — passing once proves nothing. Only `go test -race` (and `-race` in CI) instruments memory access to actually catch it.
- **"Only writes need the lock; reads are safe."** A read concurrent with a write is still a data race. Lock *reads* too — or use `RWMutex` if reads dominate. `-race` will flag the unguarded read.
- **"`-race` catches deadlocks too."** No — the race detector finds *unsynchronized access*, not lock-ordering deadlocks. Inconsistent lock ordering across goroutines is its own bug class, found by reasoning/`go vet`/timeouts, not `-race`.
- **"`atomic` makes my struct thread-safe."** An atomic op protects exactly *one* variable's read-modify-write. Multiple fields that must change together still need a mutex; atomics don't compose into a transaction.
- **"`defer Unlock` is just style."** Without it, an early return or panic inside the critical section leaves the mutex locked forever, deadlocking every future caller. `defer` makes unlock panic-safe.
