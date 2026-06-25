---
slug: project-worker-pool
step: 5
title: "Project: a worker pool with graceful shutdown"
summary: N workers draining a job queue, with bounded concurrency, clean shutdown via context, and verified race-free.
est_min: 480
position: 6
---

# Project: a worker pool with graceful shutdown

> **Step 5 · Concurrency in Go · Week 2 · Project**
> Concept: *backpressure, -race, leaks*

The worker pool is *the* foundational concurrency pattern: a fixed number of
workers pulling jobs off a queue. It bounds concurrency (you don't spawn a million
goroutines), provides backpressure, and shuts down cleanly. You'll reuse this shape
constantly.

## What you'll build

A pool of `N` workers that consume jobs from a channel, produce results to another,
shut down gracefully when the work is done *or* when a context is cancelled, and
pass `go test -race` with no goroutine leaks.

## Why this matters

Every real system has bounded resources. "Spawn a goroutine per item" works until
the item count explodes; a pool caps in-flight work at `N`. Graceful shutdown and
leak-freedom are the difference between a toy and production code — exactly what
Steps 7–8 will demand.

## 1. Jobs, results, and the worker

```go
type Job struct{ ID int }
type Result struct {
	JobID int
	Out   int
	Err   error
}

func worker(ctx context.Context, id int, jobs <-chan Job, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return                         // cancelled — stop immediately
		case j, ok := <-jobs:
			if !ok {
				return                     // jobs channel closed & drained — done
			}
			results <- Result{JobID: j.ID, Out: do(j)}
		}
	}
}
```

Two exit paths: the `jobs` channel is closed (normal completion) **or** the context
is cancelled (early shutdown). Both must be handled or the worker leaks.

## 2. Start the pool

```go
func RunPool(ctx context.Context, n int, jobs []Job) []Result {
	jobCh := make(chan Job)
	resCh := make(chan Result)
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {            // start N workers
		wg.Add(1)
		go worker(ctx, i, jobCh, resCh, &wg)
	}

	// Feed jobs in a goroutine so we can read results concurrently.
	go func() {
		defer close(jobCh)             // closing jobCh tells workers "no more work"
		for _, j := range jobs {
			select {
			case <-ctx.Done():
				return                 // stop feeding if cancelled
			case jobCh <- j:
			}
		}
	}()

	// Close results once all workers have exited.
	go func() { wg.Wait(); close(resCh) }()

	var out []Result
	for r := range resCh {             // drain results until closed
		out = append(out, r)
	}
	return out
}
```

Trace the shutdown choreography: feeder closes `jobCh` → workers finish draining and
return → `wg.Wait()` unblocks → `resCh` closes → the `range` in `RunPool` ends. No
goroutine is left blocked. That's a *clean* shutdown.

## 3. Prove it's correct

```go
// YOUR TURN:
//  - implement do(Job) int
//  - write a test that submits 100 jobs to a pool of 5 and asserts all 100
//    results come back exactly once (use a map/set to check for dupes & misses)
//  - run it with:  go test -race ./...
//  - add a test that cancels the context mid-run and asserts the pool returns
//    promptly without deadlocking
```

## Stretch goals

- **Backpressure**: make `jobCh` buffered and observe how a slow consumer slows the
  producer — that's flow control. Discuss what an unbounded queue would risk.
- Add a per-job timeout with a child `context.WithTimeout`.
- Detect goroutine leaks: print `runtime.NumGoroutine()` before and after; it should
  return to baseline.
- Generalize to `Pool[T, R any]` with generics (ties back to Step 2).

## Done when

- [ ] N workers process all jobs, each result delivered exactly once.
- [ ] `go test -race ./...` passes — no data races.
- [ ] Cancelling the context stops the pool promptly with no deadlock.
- [ ] After completion, the goroutine count returns to baseline (no leaks).
- [ ] You can narrate the close-channel shutdown sequence end to end.

FREE source: [**Go blog — Pipelines and cancellation**](https://go.dev/blog/pipelines).

Next: **concurrency patterns** — composing pools and stages into pipelines.

## Common interview gotchas

- **"A `recover()` in the request handler catches a worker panic."** No — a panic in a worker goroutine unwinds *that goroutine's* stack and crashes the whole process; `recover` only works in the same goroutine. Wrap each worker's loop body in its own `defer recover()`.
- **"Stop the pool by setting a flag / using a bool channel."** The clean signal is to `close(jobs)`: workers see `ok == false` on receive and exit after draining. Closing once broadcasts "done" to all N workers without races.
- **"It returns the right answers, so the pool is correct."** Correctness includes no leaks and no races — verify with `go test -race` *and* check `runtime.NumGoroutine()` returns to baseline after shutdown.
- **"A full job queue means the pool is broken."** A full *bounded* queue is backpressure working as designed — the producer slows to match the workers. An *unbounded* queue is the real bug (unbounded memory growth, no flow control).
- **"Cancelling context drains remaining jobs first."** Workers should `select` on `ctx.Done()` and bail *immediately* on cancel — graceful shutdown on cancel means stop promptly, not finish the backlog. Don't deadlock by blocking on a send to a channel no one is draining.
