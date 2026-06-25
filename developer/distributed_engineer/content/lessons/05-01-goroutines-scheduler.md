---
slug: conc-goroutines-scheduler
step: 5
title: Goroutines & the Go scheduler
summary: Goroutines, concurrency vs parallelism, and how Go's M:N scheduler runs thousands of them cheaply.
est_min: 300
position: 1
---

# Goroutines & the Go scheduler

> **Step 5 · Concurrency in Go · Week 1**
> Concept: *lightweight concurrency, M:N scheduling*

Concurrency is Go's headline feature, and the reason it dominates infrastructure.
Everything you'll build in Steps 7–8 — replication, Raft, key-value stores — is
concurrent. This is where you start thinking in *many things at once*.

## Why this matters

A server handles thousands of simultaneous requests. A distributed node talks to
many peers at the same time. Without cheap concurrency you'd need one OS thread per
task (expensive) or a tangle of callbacks. Go gives you **goroutines** — so cheap
you can have a million — and a runtime that schedules them for you.

## 1. Concurrency is not parallelism

- **Concurrency** = *structuring* a program as independently-progressing tasks.
- **Parallelism** = those tasks *actually running at the same instant* on multiple
  cores.

Concurrency is about design; parallelism is about execution. A concurrent program
runs fine on one core (tasks interleave); add cores and it *also* runs in parallel.
Rob Pike's talk on this is required watching (linked below).

## 2. Starting a goroutine

Put `go` in front of a function call. It runs concurrently; the caller doesn't wait.

```go
package main

import (
	"fmt"
	"time"
)

func main() {
	go fmt.Println("from a goroutine")  // runs concurrently
	fmt.Println("from main")
	time.Sleep(10 * time.Millisecond)   // crude: give the goroutine a moment
}
```

A goroutine is a function scheduled by the Go runtime, not the OS. Starting one
costs ~a few KB of stack (that grows on demand), versus ~1 MB for an OS thread.
That's why "spawn a goroutine per connection/request" is normal in Go.

## 3. The main-goroutine gotcha

**When `main` returns, the program exits — even if other goroutines are still
running.** The `time.Sleep` above is a hack to paper over that. Real code uses
`sync.WaitGroup` or channels (next two lessons) to *wait* properly. Never rely on
`Sleep` for correctness.

```go
// This may print nothing — main exits before the goroutine runs:
go fmt.Println("hi")
// (no wait)  <- the goroutine never gets a chance
```

## 4. The M:N scheduler

Go multiplexes **many goroutines (G)** onto a **few OS threads (M)**, across
logical processors **(P)**. This is *M:N scheduling*:

- You can have 100,000 goroutines on 8 OS threads.
- When a goroutine blocks (on I/O, a channel, a syscall), the scheduler parks it
  and runs another on that thread — no goroutine wastes a thread while waiting.
- `GOMAXPROCS` sets how many goroutines can run *in parallel* (defaults to the
  number of CPU cores).

```go
import "runtime"

runtime.NumCPU()      // logical cores available
runtime.GOMAXPROCS(0) // current parallelism setting (0 = just read it)
runtime.NumGoroutine()// how many goroutines are alive right now
```

You almost never tune the scheduler — but understanding it explains *why* Go
servers stay responsive under load: blocking one goroutine doesn't block the
others.

## 5. Goroutines are not free

Cheap ≠ free. Two failure modes to respect:

- **Goroutine leaks**: a goroutine blocked forever (waiting on a channel nobody
  sends to) never dies — memory and resources pile up. You'll learn to prevent
  this with `context` cancellation.
- **Unsynchronized sharing**: two goroutines touching the same variable without
  coordination is a **data race** (corruption + nondeterminism). Fixed with
  channels or the `sync` package — the next lessons.

## Do it yourself (≈ 5 hrs)

1. Watch Rob Pike's [**Concurrency is not Parallelism**](https://go.dev/blog/waza-talk) (~30 min).
2. Start the [**Learn Go with Tests — Concurrency**](https://quii.gitbook.io/learn-go-with-tests/go-fundamentals/concurrency) chapter.
3. Launch 5 goroutines that each print their index; observe that order is *not*
   guaranteed. Add a `time.Sleep` to let them finish, then note why that's a smell.
4. Print `runtime.NumGoroutine()` before and after starting them.
5. Write a program that starts a goroutine but exits immediately — watch it print
   nothing — then fix it (you'll do it properly with a WaitGroup soon).

## Check yourself

- In one sentence each, distinguish concurrency from parallelism.
- Why can Go run a million goroutines but not a million OS threads?
- What happens to running goroutines when `main` returns?
- What does the scheduler do when a goroutine blocks on I/O?
- Name the two classic ways goroutines go wrong (and what fixes each).

Next: **channels & select** — how goroutines safely talk to each other.

## Common interview gotchas

- **"Goroutines are free, so just spawn them."** Cheap is not free — each is a few KB of stack plus scheduler bookkeeping. Unbounded spawning still exhausts memory/fds; bound it with a pool or semaphore.
- **"A leaked goroutine eventually gets garbage-collected."** It never does — a goroutine blocked forever on a channel/lock is rooted and unreclaimable. Find leaks with `pprof` (`/debug/pprof/goroutine`) and fix the blocking path with `context` cancellation.
- **"My goroutine didn't run — that's a bug in Go."** No: when `main` returns the process exits and kills all goroutines mid-flight. The fix is to *wait* (WaitGroup/channel), never `time.Sleep`.
- **"GOMAXPROCS auto-detects my container's CPU limit."** It reads the host's core count, not the cgroup quota — so in a 1-CPU container it may set 64. Set it explicitly or use `automaxprocs`, or you get scheduler thrash and throttling.
- **"More goroutines means more parallelism."** Parallelism is capped by GOMAXPROCS; beyond that you only add concurrency (interleaving). For CPU-bound work, oversubscribing just adds context-switching overhead.
