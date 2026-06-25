---
slug: conc-channels-select
step: 5
title: Channels & select
summary: How goroutines communicate safely — buffered/unbuffered channels, closing, directions, and the select statement.
est_min: 420
position: 2
---

# Channels & select

> **Step 5 · Concurrency in Go · Week 1**
> Concept: *communicating via channels, not shared memory*

Go's motto: **"Don't communicate by sharing memory; share memory by
communicating."** Channels are how. Instead of multiple goroutines poking the same
variable (and racing), they pass values to each other through typed pipes.

## Why this matters

Channels turn scary concurrent coordination into something you can reason about
like a conveyor belt. Pipelines, worker pools, cancellation, fan-in/out — every
pattern in Steps 5–8 is built from channels and `select`.

## 1. A channel is a typed pipe

```go
ch := make(chan int) // a channel that carries ints
ch <- 42             // send 42 into the channel
x := <-ch            // receive a value from the channel
```

The arrow points in the direction the value flows. Sends and receives are the
synchronization — no locks needed.

## 2. Unbuffered: a handshake

An **unbuffered** channel has no capacity. A send blocks until *another goroutine
is ready to receive*, and vice versa. It's a synchronization point — a rendezvous.

```go
done := make(chan struct{})

go func() {
	fmt.Println("working...")
	done <- struct{}{}   // signal completion
}()

<-done                   // block until the goroutine signals — a proper "wait"
```

`chan struct{}` (an empty struct, zero bytes) is the idiom for a pure signal that
carries no data. This is the correct fix for the Step 5.1 "main exits too early"
problem.

## 3. Buffered: a queue with capacity

A **buffered** channel holds up to N values. Sends block only when the buffer is
*full*; receives block only when it's *empty*.

```go
ch := make(chan int, 3) // capacity 3
ch <- 1; ch <- 2; ch <- 3 // none of these block
ch <- 4                   // BLOCKS — buffer full, until someone receives
```

Use a buffer when you want the producer to run ahead of the consumer by a bounded
amount. The bound is a feature — it provides *backpressure* (Step 5.7).

## 4. Closing and ranging

The sender `close`s a channel to say "no more values." Receivers can loop with
`range` until it's closed, and detect closure with the comma-ok form.

```go
ch := make(chan int)
go func() {
	for i := 0; i < 3; i++ {
		ch <- i
	}
	close(ch)            // sender closes when done
}()

for v := range ch {      // loops until ch is closed and drained
	fmt.Println(v)
}

v, ok := <-ch            // ok == false once closed and empty
```

Rules that prevent panics:
- **Only the sender closes** a channel, never the receiver.
- **Never send on a closed channel** (it panics).
- Closing is optional — only needed to signal "done" to rangers.

## 5. Direction types

Restrict a channel to send-only or receive-only in function signatures — the
compiler enforces correct use and documents intent:

```go
func produce(out chan<- int) { out <- 1 }   // send-only
func consume(in <-chan int)  { <-in }        // receive-only
```

## 6. select — wait on multiple channels

`select` blocks until *one* of its cases can proceed. It's the heart of concurrent
control flow:

```go
select {
case v := <-a:
	fmt.Println("from a:", v)
case b <- 1:
	fmt.Println("sent to b")
case <-time.After(time.Second):
	fmt.Println("timeout!")        // don't wait forever
default:
	fmt.Println("nothing ready")   // non-blocking: take this if no case is ready
}
```

- `time.After` gives you **timeouts** — essential in distributed systems where a
  peer may never reply.
- `default` makes the select **non-blocking**.
- If multiple cases are ready, one is chosen at random (don't depend on order).

## 7. Deadlocks

If every goroutine is blocked waiting and none can proceed, the runtime panics with
`fatal error: all goroutines are asleep - deadlock!`. The usual causes: sending on
an unbuffered channel with no receiver, or forgetting to `close` a ranged channel.
Read the message — it's telling you a goroutine is stuck forever.

## Do it yourself (≈ 7 hrs)

1. Work the channels sections of [**Learn Go with Tests — Concurrency**](https://quii.gitbook.io/learn-go-with-tests/go-fundamentals/concurrency) and [**A Tour of Go — Concurrency**](https://go.dev/tour/concurrency/1).
2. Replace a `time.Sleep`-based "wait" from lesson 1 with a `chan struct{}` signal.
3. Build a producer goroutine that sends 0..9 and closes; consume with `range`.
4. Write a `select` that reads from two channels and times out after 500ms.
5. Deliberately cause a deadlock, read the runtime message, then fix it.

## Check yourself

- What's the difference between an unbuffered and a buffered channel's send behavior?
- Who is allowed to `close` a channel, and what happens if you send after close?
- How do you detect that a channel has been closed?
- What three things can a `select` do that a plain receive can't (timeout, multiplex, non-block)?
- What does a "deadlock" panic actually tell you?

Next: **the project** — fetch many web pages concurrently.

## Common interview gotchas

- **"A nil channel just errors."** A send or receive on a nil channel blocks *forever* — which is actually a feature: set a select case's channel to nil to dynamically disable that case.
- **"Closing tells receivers to stop."** Closing only signals "no more values"; receivers keep draining buffered values first, and a receive on a closed channel returns the zero value immediately (use comma-ok to tell them apart). Sending on a closed channel *panics*.
- **"Either side can close the channel."** Only the *sender* (or sole owner) closes — closing from a receiver, or closing twice, panics. With multiple senders, coordinate close via a separate done signal or WaitGroup, never from a sender.
- **"`select` checks cases top-to-bottom."** When multiple cases are ready it picks one *uniformly at random*, so never rely on ordering for priority. A `default` makes the whole select non-blocking.
- **"Add a buffer to fix the deadlock."** A buffer only defers the block until it's full; it hides the real coordination bug and can mask backpressure. Reach for it for known burst sizes, not to silence a deadlock.
