---
slug: ds-why-fallacies
step: 7
title: Why distributed systems exist & the 8 fallacies
summary: Why we tolerate the pain of many machines, and the eight false assumptions that sink naive distributed code.
est_min: 240
position: 1
---

# Why distributed systems exist & the 8 fallacies

> **Step 7 · Distributed systems theory · Week 1**
> Concept: *latency, partial failure, unreliable networks*

Everything so far ran on **one machine**, where memory is reliable, calls don't
"half-happen," and there's a single source of truth. A distributed system gives all
that up — on purpose. This step is about why, and what new rules apply.

## Why this matters

This is the field you're training for. A distributed engineer's whole job is making
many unreliable machines behave like one reliable system. Everything downstream —
replication, CAP, Raft, system design — follows from the hard truths in this lesson.

## 1. Why distribute at all? (it's not because it's fun)

Distribution is a cost you pay for things one machine can't give you:

- **Scale** — more data/traffic than one machine can hold or serve.
- **Fault tolerance** — one machine dies; the system keeps running.
- **Latency** — put data near users around the globe.
- **Throughput** — do work in parallel across many nodes.

You distribute *only when you must*. A single beefy Postgres box handles enormous
load. Reach for distribution when scale or availability genuinely demands it — not
by default.

## 2. The fundamental new problem: partial failure

On one machine, things succeed or fail together. In a distributed system, **part of
it can fail while the rest runs** — and worse, you often **can't tell** whether a
remote call succeeded, failed, or is just slow. A timeout tells you *nothing
definite*: the request may have been processed, or not. This single fact — partial
failure under uncertainty — is the source of nearly all distributed-systems
difficulty (and why idempotency from Step 4 matters so much).

## 3. The 8 fallacies of distributed computing

A famous list (Deutsch & Gosling, Sun, 1990s) of things engineers wrongly assume.
Every one of them will bite code that ignores it:

1. **The network is reliable.** — Packets drop, connections reset. *Design for
   retries and timeouts.*
2. **Latency is zero.** — A remote call is ~10⁶× slower than a local one. *Don't
   chat in a loop across the network; batch.*
3. **Bandwidth is infinite.** — Big payloads clog links. *Page, compress, stream.*
4. **The network is secure.** — *Authenticate and encrypt; assume hostile networks.*
5. **Topology doesn't change.** — Nodes come and go, IPs change. *Use discovery, not
   hard-coded hosts.*
6. **There is one administrator.** — Many teams, many configs. *Expect inconsistency.*
7. **Transport cost is zero.** — Serialization and bandwidth cost real money/CPU.
8. **The network is homogeneous.** — Mixed hardware, OSes, versions. *Don't assume
   uniformity.*

You don't memorize these to recite them — you internalize them so your designs
*assume* the network is slow, lossy, and adversarial.

## 4. The two hard truths to carry forward

- **Networks are unreliable and asynchronous.** Messages can be lost, delayed,
  reordered, or duplicated; you can't distinguish "slow" from "dead."
- **Clocks are unreliable.** Each machine's clock drifts; "what time is it?" has no
  single answer across nodes. So you *can't* order events by wall-clock time — which
  is why distributed systems use logical ordering and consensus instead.

These two truths are why the rest of Step 7 exists. Replication, CAP, and Raft are
all responses to "the network and clocks can't be trusted."

## 5. Latency numbers worth a gut feel

| Operation | ~Time |
|---|---|
| L1 cache reference | ~1 ns |
| Main memory reference | ~100 ns |
| SSD random read | ~16 µs |
| Same-datacenter round trip | ~0.5 ms |
| Disk seek | ~2 ms |
| Cross-continent round trip | ~150 ms |

A cross-region network call is ~**a million times** slower than a memory access.
This is fallacy #2 made concrete — and why *where* data lives dominates performance.

## Do it yourself (≈ 4 hrs)

1. Read about the [**8 fallacies**](https://en.wikipedia.org/wiki/Fallacies_of_distributed_computing) and skim Peter Deutsch's original list.
2. Watch the intro lecture of [**MIT 6.824**](https://pdos.csail.mit.edu/6.824/) — the course you'll do labs from in Step 8.
3. Study Jeff Dean's "[**latency numbers every programmer should know**](https://gist.github.com/jboner/2841832)" until the orders of magnitude feel intuitive.
4. Write down, for a service you know, one concrete failure each fallacy could cause.

## Check yourself

- Give three legitimate reasons to distribute a system — and one reason *not* to.
- What is "partial failure," and why does a timeout tell you nothing definite?
- Recite at least five of the eight fallacies and the design response to each.
- Why can't you order events across nodes using wall-clock timestamps?
- Roughly how much slower is a cross-continent round trip than a memory reference?

Next: **DDIA foundations** — the book that defines this field.

## Common interview gotchas

- A timeout means the *client gave up*, not that the request failed — it may have fully succeeded with only the reply lost. That's why retries demand **idempotency** (an idempotency key), not optimism.
- Exponential backoff **without jitter** still synchronizes clients: they all failed together, compute the same delay, and stampede back in lockstep. Add randomness (full/decorrelated jitter) to *decorrelate* the herd, and always cap retries.
- Retries multiply down the stack. Three layers each retrying 3× is 4³ = **64** attempts at the bottom — uncontrolled retries turn a blip into a cascading failure. Bound retries and propagate deadlines.
- You **cannot** order cross-node events by wall-clock time: clocks drift and jump, so "later" can carry a smaller timestamp. Use logical ordering — **happened-before**, Lamport/vector clocks, or a monotonic token.
- "It works on each service's dashboard" hides the slow hop. Tail latency (p99) and a propagated **trace ID** — not per-service averages — are what locate partial failure.
