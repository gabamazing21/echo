---
slug: ds-logical-clocks
step: 7
title: Logical clocks & causality
summary: Why wall-clocks can't order distributed events, and how Lamport timestamps, vector clocks, and HLCs capture "happened-before".
est_min: 240
position: 9
---

# Logical clocks & causality

> **Step 7 · Distributed systems theory · Interview prep**
> Concept: *happened-before, Lamport timestamps, vector clocks*

You learned in lesson 1 that you **can't trust wall-clocks** to order events across
machines — they drift, jump on NTP sync, and have no shared "now". So how *do* you
order events in a distributed system? With **logical clocks** that track causality
instead of time.

## Why this matters

"How do you order events without synchronized clocks?" is a top distributed-systems
interview question, and causality underpins replication conflict resolution, CRDTs,
and consistent snapshots. CockroachDB, DynamoDB, and Cassandra all use logical-clock
ideas in production.

## 1. The happened-before relation

Lamport defined a partial order, **happened-before** (→):

- If A and B are in the same process and A comes first, then A → B.
- If A is a *send* and B is the matching *receive*, then A → B.
- Transitive: A → B and B → C ⇒ A → C.

If neither A → B nor B → A, the events are **concurrent** — there's no causal
relationship, and no "correct" order between them. Capturing this partial order
(not a fake total order from clocks) is the whole game.

## 2. Lamport timestamps

A single counter per process that gives a **total order consistent with causality**:

- Each process keeps a counter `L`.
- On any event: `L = L + 1`.
- On send: attach `L`. On receive: `L = max(L, received) + 1`.

Then `A → B ⇒ L(A) < L(B)`. The catch: the converse is **false** — `L(A) < L(B)`
does **not** mean A caused B (they might be concurrent). Lamport clocks give you a
consistent tiebreak ordering, but they **can't detect concurrency**.

## 3. Vector clocks

To actually *detect* causality and concurrency, each process keeps a **vector** of
counters, one per process:

- On a local event: increment your own entry.
- On send: attach the whole vector. On receive: take the element-wise max, then
  increment your own entry.

Now compare vectors: `V(A) < V(B)` (every entry ≤, at least one <) ⇒ A → B. If
neither dominates, they're **concurrent** — which is exactly what you need to detect
a conflict (two replicas updated the same key independently). Cost: O(N) space per
timestamp for N nodes, which is why huge systems prune them.

## 4. Hybrid Logical Clocks (HLC)

Lamport clocks lose the wall-clock meaning; pure physical clocks lose causality.
**HLC** combines them: a timestamp is (physical-time, logical-counter), kept close to
real time but never going backwards and always respecting happened-before. CockroachDB
uses HLCs to get causally-consistent, roughly-real-time ordering without Google's
TrueTime atomic-clock hardware (which Spanner uses to bound clock uncertainty
instead).

## 5. Where this shows up

- **Conflict detection** in leaderless/multi-leader replication (lesson 3): vector
  clocks flag concurrent writes so you can resolve (LWW, merge, or CRDT).
- **Consistent snapshots** and causal consistency (lesson 5).
- **Ordering** in event logs and CRDTs (collaborative editing).

## Do it yourself (≈ 4 hrs)

1. Read **DDIA Ch.9** "Ordering Guarantees" (Lamport timestamps, total order broadcast).
2. Watch [**Martin Kleppmann's lecture**](https://www.youtube.com/playlist?list=PLeKd45zvjcDFUEv_ohr_HdUFe97RItdiB) on logical time.
3. On paper, run two processes exchanging messages; compute Lamport timestamps, then
   vector clocks, and identify which event pairs are concurrent.

## Check yourself

- Define happened-before and "concurrent" events.
- Why does `L(A) < L(B)` with Lamport clocks *not* imply A → B?
- What can vector clocks detect that Lamport clocks can't, and at what cost?
- What problem do Hybrid Logical Clocks solve that pure logical/physical clocks don't?
- Where in replication would you reach for vector clocks?

## Common interview gotchas

- **Lamport clocks give a total order but can't detect concurrency** — `L(A) < L(B)` is consistent with causality only one way; to know if two events are *concurrent* you need vector clocks.
- **Vector clocks cost O(N) per stamp** and grow with cluster membership; production systems prune/dot them (version vectors, dotted version vectors) rather than carry a full vector forever.
- **NTP does not make wall-clocks safe for ordering** — it corrects drift but can step time *backwards*, so two events can get out-of-causal-order timestamps. Logical clocks exist precisely because "just use NTP" is wrong.
- **Last-write-wins silently drops data** — using a physical timestamp to pick a winner among concurrent writes discards the loser with no merge; only acceptable when losing a write is OK.
