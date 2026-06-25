---
slug: ds-replication
step: 7
title: Replication — leader/follower, sync vs async
summary: Keeping copies of data on multiple nodes — leader-based replication, the sync/async trade-off, and replication lag.
est_min: 420
position: 3
---

# Replication — leader/follower, sync vs async

> **Step 7 · Distributed systems theory · Week 2**
> Concept: *keeping copies consistent*

The first tool of distribution: keep the **same data on multiple nodes**. Why?
Fault tolerance (a node dies, others have the data), read scaling (serve reads from
many copies), and locality (a copy near each region). The hard part is keeping the
copies *in agreement* when writes keep arriving.

## Why this matters

Replication is how systems survive machine failure and scale reads — the two things
you distribute *for*. It's also where you first feel the consistency-vs-availability
tension that CAP (lesson 5) formalizes and Raft (lesson 6) resolves.

## 1. Leader-based (single-leader) replication

The most common scheme, used by Postgres, MySQL, MongoDB, etc.:

- One replica is the **leader** (primary). All **writes** go to it.
- The leader streams its changes (a **replication log**) to the **followers**.
- **Reads** can be served by the leader *or* any follower.

```
            writes
   client ────────▶  LEADER ──log──▶ FOLLOWER (read replica)
                       │      ──log──▶ FOLLOWER (read replica)
   reads  ◀────────────┴───────────── (any replica)
```

Single-leader is simple because there's one place that decides write order — no
write conflicts. That simplicity is why it dominates.

## 2. Synchronous vs asynchronous replication

When the leader gets a write, does it wait for followers before telling the client
"done"?

- **Synchronous** — leader waits for the follower to confirm before acking the
  client. Guarantees the follower has the data. **But** if that follower is slow or
  down, writes *block*. Strong durability, worse availability.
- **Asynchronous** — leader acks immediately, ships to followers in the background.
  Fast and available. **But** if the leader crashes before a write propagates, that
  write is **lost**, and followers may serve stale data.

Real systems use **semi-synchronous**: one synchronous follower (guaranteed up-to-
date copy), the rest async. You're trading durability against latency/availability —
the recurring theme of this whole step.

## 3. Replication lag & its anomalies

With async replication, followers trail the leader by some **lag** (ms to seconds,
sometimes more). That lag creates user-visible weirdness — and named consistency
guarantees that fix each:

- **Read-your-own-writes**: you post a comment (write → leader), then refresh
  (read → a lagging follower) and *your own comment is missing*. Fix: read your own
  recent writes from the leader.
- **Monotonic reads**: refresh twice, see a comment, then it *disappears* (you hit a
  more-lagged follower the second time — time going backwards). Fix: pin a user to
  one replica.
- **Consistent prefix reads**: you see an answer before the question (writes
  observed out of causal order). Fix: ensure causally-related writes land in order.

These aren't exotic — they're the everyday cost of async replication, and knowing
their names is interview gold.

## 4. Failover: promoting a new leader

If the leader dies, a follower must be **promoted** to leader (failover). This is
deceptively dangerous:

- Which follower? Ideally the most up-to-date one.
- **Lost writes**: async writes not yet replicated are gone.
- **Split brain**: two nodes both think they're leader → conflicting writes →
  corruption. Must be prevented.
- **Timeout tuning**: declare the leader dead too eagerly and you cause needless
  failovers; too slowly and you're down longer.

Doing failover *correctly and automatically* is exactly the problem **consensus**
(Raft, lesson 6) solves — electing one agreed leader, safely.

## 5. Multi-leader & leaderless (a glimpse)

- **Multi-leader**: multiple nodes accept writes (e.g. one leader per datacenter).
  Better write availability, but now you get **write conflicts** that need
  resolution (last-write-wins, CRDTs, app logic). Complex.
- **Leaderless** (Dynamo-style, e.g. Cassandra): any node takes writes; clients
  write to several and read from several (**quorums**: if writes-to W + reads-from R
  > N replicas, a read overlaps a recent write). Highly available, eventually
  consistent.

Single-leader is the default; reach for the others only when write availability
demands it.

## Do it yourself (≈ 7 hrs)

1. Read **DDIA Chapter 5 (Replication)** — the canonical treatment of everything above.
2. Watch the replication lecture from [**Kleppmann's course**](https://www.youtube.com/playlist?list=PLeKd45zvjcDFUEv_ohr_HdUFe97RItdiB).
3. (Hands-on) Set up a Postgres read replica locally (primary + standby) and watch
   streaming replication; measure the lag.
4. For each replication-lag anomaly, write a concrete one-sentence user story.

## Check yourself

- Why does single-leader replication avoid write conflicts?
- Sync vs async replication: what does each trade, and what's semi-sync?
- Explain "read-your-own-writes" and "monotonic reads" with a real example each.
- What is split brain, and why is failover genuinely hard?
- When would you accept the complexity of multi-leader or leaderless replication?

## Common interview gotchas

- **Async replication loses *acknowledged* writes.** If the leader acks the client and crashes before the write propagates, failover promotes a follower that never saw it — the write is silently discarded. "Acked" ≠ "durable" under async.
- **R + W > N guarantees overlap, not linearizability.** The condition only forces a read set and write set to share a node. Concurrent writes still need conflict resolution (vector clocks / LWW), and a **sloppy quorum** with hinted handoff can satisfy R+W>N while reading stale data, because the W writes landed on substitute nodes, not the R home nodes.
- **Split brain needs fencing tokens, not just "promote the freshest follower."** Choosing the most up-to-date follower decides *who* leads; it does nothing to stop a GC-paused or partitioned old leader from still writing. Only a monotonically increasing token (Raft term, ZooKeeper `zxid`) checked at the storage layer fences the zombie out.
- **Read-your-writes ≠ strong consistency.** It's a *per-user* guarantee about your *own* writes — other users can still see stale data or observe writes in different orders. Linearizability is global and far more expensive.
- **Semi-sync still loses writes if both the leader and the sync follower fail.** One synchronous follower narrows the loss window but doesn't close it; durability scales only with the number of *independent* copies that must confirm before the ack.

Next: **partitioning** — splitting data that's too big for one node.
