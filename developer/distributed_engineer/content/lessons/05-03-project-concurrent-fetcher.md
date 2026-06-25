---
slug: project-concurrent-fetcher
step: 5
title: "Project: a concurrent web-page fetcher"
summary: Fetch many URLs at once with goroutines and channels — concurrency applied to real, I/O-bound work.
est_min: 360
position: 3
---

# Project: a concurrent web-page fetcher

> **Step 5 · Concurrency in Go · Week 1 · Project**
> Concept: *goroutines + channels in anger*

Fetching URLs is the perfect first concurrency project: it's **I/O-bound**, so
doing them in parallel is a massive speedup, and it forces you to coordinate
goroutines with channels.

## What you'll build

A CLI that takes a list of URLs, fetches them **concurrently**, and prints per-URL
status, size, and timing — finishing in roughly the time of the *slowest* request,
not the *sum* of all of them.

## Why this matters

Fan-out — "do these N independent things at once and collect the results" — is one
of the most common shapes in real systems: scatter a query to many shards, health-
check many nodes, call many downstream services. You're learning the template.

## 1. The result type and a single fetch

```go
type Result struct {
	URL     string
	Status  int
	Bytes   int
	Elapsed time.Duration
	Err     error
}

func fetch(ctx context.Context, url string) Result {
	start := time.Now()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Result{URL: url, Err: err, Elapsed: time.Since(start)}
	}
	defer resp.Body.Close()
	n, _ := io.Copy(io.Discard, resp.Body) // count bytes without storing
	return Result{URL: url, Status: resp.StatusCode, Bytes: int(n), Elapsed: time.Since(start)}
}
```

Always use `http.NewRequestWithContext` so a timeout/cancel can stop the request.

## 2. Fan out: one goroutine per URL, results over a channel

```go
func fetchAll(ctx context.Context, urls []string) []Result {
	results := make(chan Result)          // workers send here
	var wg sync.WaitGroup

	for _, u := range urls {
		wg.Add(1)
		go func(url string) {             // pass url as an arg (capture gotcha!)
			defer wg.Done()
			results <- fetch(ctx, url)
		}(u)
	}

	// Close the results channel once all workers are done, in a separate goroutine
	go func() { wg.Wait(); close(results) }()

	var out []Result
	for r := range results {              // drain until closed
		out = append(out, r)
	}
	return out
}
```

Two things to internalize here:
- **The loop-variable capture gotcha:** pass `u` into the goroutine as a parameter.
  (Go 1.22+ fixed the per-iteration scoping, but passing it is still the clearest,
  portable habit.)
- **The close-in-a-goroutine pattern:** `wg.Wait()` blocks, so it runs in its own
  goroutine; closing `results` lets the `range` in `main` terminate.

## 3. Wire up main and print

```go
// YOUR TURN:
//  - read URLs from os.Args (or a file)
//  - call fetchAll with a context that has an overall timeout
//    (context.WithTimeout — you'll go deep on this in lesson 5)
//  - print a tidy table; report errors per URL without aborting the others
```

Compare wall-clock time vs a sequential `for` loop over `fetch` — the difference is
the whole point.

## Stretch goals

- **Bound concurrency**: don't launch 10,000 goroutines for 10,000 URLs. Use a
  buffered "semaphore" channel (`sem := make(chan struct{}, 20)`) or a worker pool
  (next project) to cap it at N in flight.
- Add a per-request timeout via `context.WithTimeout` and report which URLs timed out.
- Add a `-c N` flag for the concurrency limit.
- Retry failed fetches with a small backoff.

## Done when

- [ ] N URLs are fetched concurrently and total time ≈ the slowest one, not the sum.
- [ ] Each goroutine sends a `Result`; `main` collects them all via a channel.
- [ ] One failing URL doesn't crash or block the others.
- [ ] You used the `wg.Wait(); close(ch)` pattern and can explain why the close is
      in its own goroutine.

FREE source: [**Go blog — Pipelines and cancellation**](https://go.dev/blog/pipelines).

Next: **the sync package** — the other side of concurrency, shared memory done safely.

## Common interview gotchas

- **"One goroutine per URL scales fine."** At 10k+ URLs you exhaust file descriptors and sockets and trigger connection storms. Bound in-flight work with a semaphore channel (`make(chan struct{}, N)`) or a worker pool.
- **"First-result-wins is easy — just return from the first goroutine."** If the result channel is *unbuffered*, the losing goroutines block forever on their send and leak. Use a *buffered* channel (capacity = number of workers) and/or cancel the rest via context.
- **"Loop-variable capture is solved, so don't pass it in."** Only on Go 1.22+. On older versions all goroutines see the final value; passing `u` as an argument is the portable, unambiguous habit regardless of version.
- **"Just `close(results)` after the loop."** You must `close` *after* all senders finish — run `wg.Wait(); close(results)` in its own goroutine while `main` drains, or you either close too early (send-on-closed panic) or deadlock.
- **"The HTTP timeout will stop a slow fetch."** Only if you build the request with `http.NewRequestWithContext` and the context carries the deadline — a bare `http.Get` ignores your cancellation entirely.
