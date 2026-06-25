---
slug: conc-context
step: 5
title: The context package — cancellation & deadlines
summary: Propagate cancellation, deadlines and timeouts across goroutines and API boundaries — the spine of robust services.
est_min: 300
position: 5
---

# The context package — cancellation & deadlines

> **Step 5 · Concurrency in Go · Week 2**
> Concept: *propagating cancellation across calls*

You can start a thousand goroutines. How do you *stop* them — when a request is
cancelled, a deadline passes, or one failure means the rest of the work is
pointless? The answer is `context.Context`, and it's everywhere in real Go.

## Why this matters

In a distributed system, work spans many goroutines and many network calls. If the
client hangs up, or a 2-second SLA expires, you must abandon the in-flight work —
or you leak goroutines and waste a struggling system's resources. `context` is how
a cancellation signal flows down through every layer at once.

> **This app uses it throughout.** Every Consensus handler passes
> `c.Request().Context()` into the store and the checker, so a cancelled HTTP
> request stops the database query and the `go test` run.

## 1. What a Context carries

A `context.Context` carries, across API boundaries and goroutines:
- a **cancellation signal** (`Done()` channel),
- an optional **deadline/timeout**,
- request-scoped **values** (use sparingly).

By convention it's the **first argument** to any function that does I/O or
long work: `func DoThing(ctx context.Context, ...) error`.

## 2. Deriving contexts

You start from a root and *derive* children. Cancelling a parent cancels all its
children — that's the propagation.

```go
ctx := context.Background()              // the root (use in main/tests)

ctx, cancel := context.WithCancel(ctx)   // manual cancellation
defer cancel()                           // ALWAYS call cancel to release resources

ctx, cancel := context.WithTimeout(ctx, 2*time.Second)  // auto-cancel after 2s
defer cancel()

ctx, cancel := context.WithDeadline(ctx, t) // auto-cancel at a wall-clock time
defer cancel()
```

`defer cancel()` even on a timeout context is not optional — it frees the timer.

## 3. Reacting to cancellation

A long-running goroutine selects on `ctx.Done()`:

```go
func worker(ctx context.Context, jobs <-chan Job) {
	for {
		select {
		case <-ctx.Done():
			return                 // cancelled or timed out — stop now
		case j, ok := <-jobs:
			if !ok {
				return
			}
			process(j)
		}
	}
}
```

After cancellation, `ctx.Err()` tells you why: `context.Canceled` or
`context.DeadlineExceeded`.

## 4. It threads through the standard library

You don't usually check `Done()` by hand for I/O — you *pass the context in*, and
the library honors it:

```go
req, _ := http.NewRequestWithContext(ctx, "GET", url, nil) // cancels the HTTP call
rows, _ := pool.Query(ctx, "SELECT ...")                   // cancels the DB query
```

When `ctx` is cancelled, that HTTP request and that query abort. This is why
threading `ctx` everywhere matters — cancellation only works if it reaches the
blocking call.

## 5. Values — use sparingly

`context.WithValue` attaches request-scoped data (a request id, the authenticated
user id). Keep it to cross-cutting metadata, never to pass normal function
parameters — overuse makes data flow invisible.

```go
ctx = context.WithValue(ctx, userIDKey{}, 42)
uid, _ := ctx.Value(userIDKey{}).(int64)
```

Use an unexported key type (not a plain string) to avoid collisions.

## 6. Rules of thumb

- Pass `ctx` as the **first param**; don't store it in a struct.
- Don't pass `nil` — use `context.Background()` or `context.TODO()`.
- The caller that *creates* a cancel/timeout context owns calling `cancel()`.
- A cancelled context stays cancelled — derive a fresh one for new work.

## Do it yourself (≈ 5 hrs)

1. Read the [**Go blog — Context**](https://go.dev/blog/context) and skim the [**context docs**](https://pkg.go.dev/context).
2. Write a worker that loops until `ctx.Done()`; cancel it from `main` after 100ms
   and print `ctx.Err()`.
3. Add a `context.WithTimeout` to your Step 5.3 fetcher so slow URLs are abandoned;
   report which ones hit `DeadlineExceeded`.
4. Trace how the Consensus app passes `c.Request().Context()` from a handler into
   `internal/store` and `internal/checker`.

## Check yourself

- What three things does a `Context` carry?
- Why must you always `defer cancel()`, even with `WithTimeout`?
- How does a goroutine learn it's been cancelled, and how do you find out why?
- Why does cancellation "only work if you pass ctx all the way down"?
- When is `context.WithValue` appropriate, and when is it abuse?

Next: **the project** — a worker pool with graceful shutdown, using everything so far.

## Common interview gotchas

- **"WithTimeout auto-cancels, so I don't need `defer cancel()`."** You still must call it — the timer/resources aren't released until cancel runs (or the timeout fires). `go vet` warns on the missing cancel; skipping it leaks the context's goroutine and timer.
- **"Store the context on the struct to avoid threading it."** Anti-pattern: `ctx` is always the *first argument*, never a struct field. A stored context goes stale and breaks per-call cancellation/deadlines.
- **"context.WithValue is a handy way to pass parameters."** It's for request-scoped *metadata* (request id, auth principal) only — using it for normal arguments makes data flow invisible and type-unsafe. Use an unexported key type to avoid collisions.
- **"Cancelling the context stops the work."** Only if `ctx` actually reaches the blocking call (the DB query, HTTP request, or a `select` on `ctx.Done()`). A goroutine that never checks `Done()` keeps running after cancel.
- **"A cancelled context can be reused once the operation retries."** Once cancelled it stays cancelled forever — derive a *fresh* context for new/retried work rather than reusing the dead one.
