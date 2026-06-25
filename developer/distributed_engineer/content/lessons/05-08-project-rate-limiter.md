---
slug: project-rate-limiter
step: 5
title: "Project: a concurrent-safe token-bucket rate limiter"
summary: Build an Allow() limiter that caps requests per second, refills over time, and is safe under many goroutines.
est_min: 360
position: 8
---

# Project: a concurrent-safe token-bucket rate limiter

> **Step 5 · Concurrency in Go · Week 3 · Project**
> Concept: *time, tickers, mutexes*

A rate limiter protects a service from being overwhelmed — by abusive clients, by a
retry storm, by your own enthusiastic worker pool. The **token bucket** is the
classic algorithm, and building one ties together time, mutexes, and concurrency.

## What you'll build

A `Limiter` with `Allow() bool`: it permits up to `rate` requests per second,
returning `false` when the budget is exhausted, and refilling smoothly over time —
safe to call from many goroutines at once.

## Why this matters

Rate limiting is everywhere: API gateways, login endpoints (brute-force defense,
see Step 4), polite clients to third-party APIs, backpressure between services. It's
also a compact, real exercise in protecting shared state correctly.

## 1. The token-bucket idea

Picture a bucket holding up to `burst` tokens. Every request takes one token. Tokens
refill at `rate` per second up to the cap. If the bucket is empty, the request is
denied (or must wait).

- **Steady state**: you get ~`rate` requests/second.
- **Bursts**: a full bucket lets a short spike of up to `burst` through at once.

That burst tolerance is why token bucket beats a naive "1 every 1/rate seconds".

## 2. A mutex-guarded implementation

Compute refill lazily from elapsed time — no background goroutine needed.

```go
type Limiter struct {
	mu       sync.Mutex
	tokens   float64
	max      float64
	perSec   float64
	last     time.Time
}

func NewLimiter(ratePerSec, burst float64) *Limiter {
	return &Limiter{tokens: burst, max: burst, perSec: ratePerSec, last: time.Now()}
}

func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(l.last).Seconds()
	l.last = now

	l.tokens += elapsed * l.perSec      // refill for the time that passed
	if l.tokens > l.max {
		l.tokens = l.max                // cap at burst
	}

	if l.tokens >= 1 {
		l.tokens--                      // spend one
		return true
	}
	return false
}
```

The mutex makes the read-modify-write of `tokens` atomic across goroutines —
exactly the lesson from 5.4. Note `time.Now()` is the clock; injecting a clock makes
this testable (stretch goal).

## 3. The alternative: a buffered channel + ticker

A different design worth knowing: a buffered channel holds tokens, a `time.Ticker`
refills it.

```go
// YOUR TURN: implement this variant.
//  - tokens := make(chan struct{}, burst); prefill it
//  - a goroutine on a time.Ticker tries to add a token each tick (non-blocking
//    send with select/default so it never overflows)
//  - Allow() does a non-blocking receive: select { case <-tokens: return true;
//    default: return false }
// Compare it to the mutex version: which is simpler? which needs a goroutine?
```

## 4. Prove it under concurrency

```go
// YOUR TURN:
//  - test: a limiter at 5/sec, hammered by 50 goroutines in the same second,
//    should allow roughly the burst, then deny the rest — assert the allowed
//    count is within an expected range
//  - run with: go test -race ./...   (must be clean)
```

## Stretch goals

- **Per-key limiting**: a `map[string]*Limiter` (guarded!) so each user/IP gets its
  own bucket. Now you have an HTTP middleware.
- Plug it into your Step 4 `/login` route to throttle brute-force attempts.
- Inject a `now func() time.Time` clock so tests are deterministic (no sleeps).
- Compare your implementation's behavior to `golang.org/x/time/rate`.

## Done when

- [ ] `Allow()` permits ≈ `rate`/sec in steady state and tolerates a burst.
- [ ] It's correct under many concurrent callers; `go test -race` is clean.
- [ ] You implemented (or fully understand) both the mutex and channel+ticker designs.
- [ ] You can explain why token bucket allows bursts and a fixed-interval limiter
      doesn't.

FREE source: [**Go time package docs**](https://pkg.go.dev/time) (and, for comparison, `golang.org/x/time/rate`).

Next step: **Step 6 — Databases, deep.**

## Common interview gotchas

- **"Run the limiter on every node and you've capped the global rate."** N nodes each at `rate` = `N × rate` globally. A *global* limit needs a centralized store (e.g. Redis) or a coordinated/distributed algorithm — per-instance limiters only bound per-instance traffic.
- **"Token bucket and leaky bucket are the same thing."** Token bucket *allows bursts* up to the bucket size, then falls to the steady rate; leaky bucket *smooths* output to a constant rate with no burst. Pick based on whether spikes are acceptable.
- **"Sleep in the test to let tokens refill."** Sleeps make tests slow and flaky. Inject a clock (`now func() time.Time`) so you can advance time deterministically and assert exact allow/deny counts.
- **"Use `time.Now()` for elapsed-time refill."** Wall-clock time can jump backward (NTP, DST) and produce negative elapsed or token surges. Go's monotonic clock (already embedded in `time.Now()` for `Sub`) handles this — but a custom/serialized clock must preserve monotonicity.
- **"One mutex-guarded limiter is fine for per-user limiting."** A shared `map[string]*Limiter` is itself shared state — the map access must be guarded too, and you need eviction or it grows unbounded as keys accumulate.
