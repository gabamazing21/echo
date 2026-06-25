---
slug: ds-ddia-distributed
step: 7
title: "DDIA Ch.5–9: transactions, trouble & consensus"
summary: Tying the theory together — distributed transactions, the trouble with distributed systems, and consistency & consensus.
est_min: 720
position: 7
---

# DDIA Ch.5–9: transactions, trouble & consensus

> **Step 7 · Distributed systems theory · Week 4**
> Concept: *tying theory together*

You've met the pieces — replication, partitioning, CAP, Raft. DDIA's second half
ties them into a coherent worldview. This guided read closes out the theory step so
you walk into the Step 8 labs with the full picture.

## Why this matters

This is where it all *connects*: why distributed transactions are hard, why
"the trouble with distributed systems" deserves its own chapter, and how
consensus underpins everything. Finishing DDIA is a milestone that genuinely
separates engineers who *use* distributed systems from those who *understand* them.

## 1. (Ch.5–6 recap) Replication & partitioning

You did these in lessons 3–4 against the book. Before moving on, make sure you can,
from memory: explain single-leader replication and the sync/async trade-off; name
the replication-lag anomalies and their fixes; contrast range vs hash partitioning;
and explain rebalancing without a full reshuffle. If any are fuzzy, re-skim Ch.5–6 —
the later chapters lean on them.

## 2. (Ch.7) Transactions, revisited at scale

Step 6 taught single-node transactions and isolation levels. Ch.7 deepens it:

- The **anomalies** (dirty/non-repeatable reads, phantoms) and what each isolation
  level prevents — now with the *why*.
- **Snapshot isolation** (how MVCC implements Repeatable Read) in full.
- **Write skew** — a subtle anomaly serializable isolation prevents but weaker
  levels don't (two transactions each read an overlapping set and make decisions
  that are individually fine but jointly violate an invariant — e.g. both on-call
  doctors clock off because each sees the other is still on).
- Approaches to **serializability**: actual serial execution, two-phase locking
  (2PL), and serializable snapshot isolation (SSI).

## 3. (Ch.8) The trouble with distributed systems

The chapter that makes you appropriately paranoid — the rigorous version of lesson
1's fallacies:

- **Unreliable networks** — packets lost/delayed/reordered; you cannot distinguish a
  slow node from a dead one. Timeouts are guesses.
- **Unreliable clocks** — time-of-day clocks jump (NTP); monotonic clocks don't sync
  across machines. **Never use wall-clock timestamps to order events across nodes**
  or to decide leadership. (Google's Spanner needs special hardware — TrueTime — just
  to bound clock uncertainty.)
- **Process pauses** — a GC pause or VM migration can freeze a node mid-operation for
  *seconds*; when it wakes, it may wrongly believe it's still the leader. Hence
  **fencing tokens** (monotonic numbers that let a resource reject a stale leader's
  request).
- **Knowledge, truth, and lies** — a node can't trust its own view; truth is defined
  by a **majority/quorum**, not any single node. This is the philosophical root of
  why consensus uses majorities.

## 4. (Ch.9) Consistency & consensus — the payoff chapter

- **Linearizability** (lesson 5) in full rigor — the strongest single-object model
  and its cost.
- **Ordering & causality** — capturing "happened-before" with **Lamport timestamps**
  and **version vectors**; total order broadcast.
- **Consensus** — the equivalence of total-order broadcast, linearizable
  compare-and-set, and leader election (they're all the same problem). **Raft and
  Paxos** are how it's done; this is exactly lesson 6, now in its theoretical home.
- **Distributed transactions & 2PC** — the **two-phase commit** protocol for atomic
  commit across nodes, its **coordinator** as a single point of failure, and why
  distributed transactions are expensive and often avoided.
- **Membership & coordination services** — how **ZooKeeper/etcd** package consensus
  into a reusable service for leader election, locks, and config (the thing lesson 4
  said the partition router relies on).

## 5. The unifying insight

DDIA's whole arc lands on one idea: **distributed systems are about coping with
uncertainty — failure, delay, and disagreement — and consensus is the tool that
manufactures agreement out of that uncertainty.** Everything you'll build in Step 8
is an application of this. When you can explain *why* a majority quorum, *why* logical
clocks, and *why* linearizability costs availability, you've got the theory.

## 6. Self-test: the distributed-systems vocabulary

You should now be able to define and connect: replication, partitioning, quorum,
leader election, replicated log, linearizability, eventual/causal consistency, CAP,
split brain, fencing token, 2PC, total order broadcast, Lamport timestamp. If you
can teach each to a peer, you're ready for the labs.

## Do it yourself (≈ 12 hrs)

1. **Read DDIA chapters 7, 8, and 9.** This is substantial — pace it across the week.
2. Watch the matching [**Kleppmann lectures**](https://www.youtube.com/playlist?list=PLeKd45zvjcDFUEv_ohr_HdUFe97RItdiB) on ordering, consensus, and 2PC.
3. Write a one-page summary in your own words of "why distributed transactions are
   hard" — this becomes your blog post seed for Step 10.
4. Re-read the **Raft paper** (lesson 6) now that Ch.9 gives it context — you start
   implementing it next step.

## Check yourself

- What is write skew, and which isolation level prevents it?
- Why can't you order cross-node events by wall-clock time — and what do you use instead?
- What's a process pause, and how does a fencing token defend against a stale leader?
- Why is "truth" defined by a majority/quorum rather than any single node?
- Why is two-phase commit expensive, and what's its single point of failure?

Next step: **Step 8 — Build distributed systems.** Theory becomes running code: MapReduce, Raft, and a replicated key-value store.

## Common interview gotchas

- **A distributed lock with a TTL is *not* safe against process pauses.** A GC pause or VM migration can freeze the holder past the lease; it wakes as a zombie leader and writes over the new holder. Bumping the TTL never fixes it — only **fencing tokens** (monotonic numbers the *resource* checks and rejects when stale) make the lock correct.
- **"Exactly-once delivery" is a myth.** The network forces redelivery, so what you build is **at-least-once + idempotent processing** (effectively-once): stable idempotency keys plus a dedup store, with the side effect and the dedup record committed atomically.
- **2PC's coordinator is a SPOF *and* a blocking point.** If it dies after participants vote "yes" in prepare, they sit holding locks — unable to commit or abort — until it recovers. That blocking is why 2PC scales poorly.
- **Saga buys availability but gives up isolation.** No global locks and no coordinator blocking, but other transactions can read uncommitted intermediate state (dirty reads), and there's no automatic rollback — you must write **compensating transactions** by hand.
- **ZooKeeper/etcd don't "store your data."** They keep *small, critical metadata* — leases, locks, config, membership — strongly consistent via a majority quorum. Bulk and high-throughput data stays in your database; ZK/etcd are the coordination brain, not the warehouse.
