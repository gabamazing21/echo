---
slug: conc-patterns
step: 5
title: Concurrency patterns — pipelines & fan-in/out
summary: Compose goroutines and channels into pipelines, fan-out/fan-in, and rate-limited stages — the reusable shapes.
est_min: 360
position: 7
---

# Concurrency patterns — pipelines & fan-in/out

> **Step 5 · Concurrency in Go · Week 3**
> Concept: *composing concurrent stages*

You have the primitives — goroutines, channels, select, sync, context. Patterns are
how you *compose* them into systems that are correct and readable. These four show
up everywhere from data processing to the distributed systems in Step 8.

## Why this matters

Real workloads are stages: read → transform → write; or scatter → process →
gather. Expressing them as channel-connected stages makes each piece independently
testable and naturally concurrent. This is the vocabulary senior Go engineers think
in.

## 1. The pipeline pattern

A pipeline is a series of stages connected by channels. Each stage is a function
that receives from an inbound channel and sends to an outbound one.

```go
func gen(nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, n := range nums {
			out <- n
		}
	}()
	return out
}

func square(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			out <- n * n
		}
	}()
	return out
}

// compose: gen -> square -> consume
for n := range square(gen(2, 3, 4)) {
	fmt.Println(n) // 4, 9, 16
}
```

Each stage owns its output channel and closes it when done. Stages run
concurrently — values flow through as they're produced.

## 2. Fan-out: parallelize a slow stage

If one stage is the bottleneck, run several copies of it reading from the *same*
input channel. The runtime load-balances across them.

```go
in := gen(work...)
c1 := square(in)   // three workers, all reading from `in`
c2 := square(in)
c3 := square(in)
// ... now merge c1, c2, c3 (fan-in, below)
```

This is the worker pool from lesson 6, expressed as a pipeline stage.

## 3. Fan-in: merge many channels into one

Combine multiple producers into a single stream with a `WaitGroup` + a merged
channel:

```go
func merge(cs ...<-chan int) <-chan int {
	out := make(chan int)
	var wg sync.WaitGroup
	for _, c := range cs {
		wg.Add(1)
		go func(c <-chan int) {
			defer wg.Done()
			for v := range c {
				out <- v
			}
		}(c)
	}
	go func() { wg.Wait(); close(out) }()
	return out
}

for n := range merge(c1, c2, c3) { /* all results, any order */ }
```

Fan-out then fan-in is the canonical "process N items in parallel, collect
results" shape.

## 4. Cancellation propagation

Long pipelines must stop cleanly when the consumer quits early (an error, a
timeout). Thread a `context.Context` through every stage and `select` on
`ctx.Done()` for both sends and receives so no stage blocks forever:

```go
func square(ctx context.Context, in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			select {
			case out <- n * n:
			case <-ctx.Done():
				return           // consumer gave up; don't leak this goroutine
			}
		}
	}()
	return out
}
```

Without this, a consumer that stops early leaves upstream stages blocked on a send
forever — a goroutine leak.

## 5. Rate limiting

Sometimes you must *slow down* — respect a downstream's limits, avoid hammering a
peer. A `time.Ticker` gates a stage:

```go
limiter := time.NewTicker(200 * time.Millisecond) // ≤ 5 ops/sec
defer limiter.Stop()
for req := range requests {
	<-limiter.C       // wait for the next tick before proceeding
	handle(req)
}
```

For bursts, a **token bucket** is better — which is exactly the next (and final)
Step 5 project.

## Do it yourself (≈ 6 hrs)

1. Read the [**Go blog — Pipelines and cancellation**](https://go.dev/blog/pipelines) carefully; it's the canonical text for this lesson.
2. Build a 3-stage pipeline (generate → transform → filter) over channels.
3. Add fan-out (3 workers on the transform stage) and fan-in to merge them.
4. Add context cancellation so stopping the consumer cleanly tears down every stage;
   verify with `runtime.NumGoroutine()`.
5. (Deeper) Skim [**Concurrency in Go**](https://www.oreilly.com/library/view/concurrency-in-go/9781491941294/) (Cox-Buday).

## Check yourself

- What is a pipeline stage, and who owns/closes each stage's output channel?
- How do fan-out and fan-in combine to parallelize work and collect results?
- Why does every stage need to `select` on `ctx.Done()` — what leaks otherwise?
- When would you rate-limit a stage, and what's the simplest tool to do it?
- How does "fan-out + fan-in" relate to the worker pool you built?

Next: **the project** — a concurrent-safe token-bucket rate limiter.

## Common interview gotchas

- **"Checking `ctx.Done()` on receives is enough."** Every stage must also `select` on `ctx.Done()` for its *sends* — if the consumer dies, an upstream stage blocks forever on a send and leaks. Both directions need the cancel case.
- **"Buffer the channels to make the pipeline faster."** Buffers don't raise throughput — the pipeline runs at the rate of its *slowest stage*. Buffers only add backpressure slack to absorb bursts; the bottleneck still gates everything.
- **"Each fan-in goroutine should close the output channel when done."** Closing `out` more than once panics. Close it *exactly once*, from a single goroutine that runs `wg.Wait()` after all the merge goroutines finish.
- **"Fan-out gives me ordered results."** Fan-out + fan-in deliberately interleaves results in arbitrary order. If you need ordering, tag items with an index and reassemble downstream.
- **"A `time.Ticker` rate-limiter handles bursts."** A fixed-interval ticker smooths to a steady rate but rejects bursts; allowing short spikes up to a cap needs a token bucket (next project). Also remember to `Stop()` the ticker or it leaks.
