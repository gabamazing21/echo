---
slug: ds-cap-consistency
step: 7
title: CAP, consistency models & linearizability
summary: The fundamental trade-off under network partitions, the spectrum from strong to eventual consistency, and what linearizability means.
est_min: 360
position: 5
---

# CAP, consistency models & linearizability

> **Step 7 · Distributed systems theory · Week 3**
> Concept: *the fundamental tradeoff*

Replication and partitioning gave you multiple copies of data on flaky networks.
Now the deep question: **how consistent should those copies be, and what must you
give up to get it?** This lesson is the theory that frames every distributed design
decision.

## Why this matters

"Strong or eventual consistency?" is *the* defining trade-off of distributed
systems, and the most common senior-interview topic. Misunderstand CAP and you'll
either over-engineer (paying for consistency you don't need) or ship data
corruption. This is the conceptual peak before Raft makes it concrete.

## 1. The CAP theorem — and what it really says

Three properties:
- **C**onsistency — every read sees the most recent write (one up-to-date copy).
- **A**vailability — every request gets a (non-error) response.
- **P**artition tolerance — the system keeps working despite the network splitting
  nodes into groups that can't talk.

The pop-culture "pick 2 of 3" is **misleading**. In a real distributed system,
**partitions happen** — you don't get to *not* tolerate them. So the honest framing:

> **When a network partition occurs, you must choose between Consistency and
> Availability.** (When there's no partition, you can have both.)

- **CP** (choose consistency): on a partition, refuse requests that can't be made
  consistent — return errors rather than stale/conflicting data. (e.g. etcd,
  ZooKeeper, HBase.)
- **AP** (choose availability): on a partition, keep serving — accept that
  different sides may diverge and reconcile later. (e.g. Cassandra, Dynamo.)

Neither is "right"; it depends on whether stale data or downtime is worse for *your*
use case. A bank balance leans CP; a shopping cart or social feed often leans AP.

## 2. The consistency spectrum

Consistency isn't binary; it's a spectrum from strongest (most intuitive, most
expensive) to weakest (cheapest, most surprising):

- **Linearizable / strong** — behaves as if there's a single copy and operations
  happen instantaneously in real-time order. The gold standard (and section 3).
- **Sequential / causal** — preserves *some* ordering (e.g. cause before effect)
  but not real-time. Causal consistency is a sweet spot for many apps.
- **Eventual** — if writes stop, all replicas *eventually* converge. Says nothing
  about *when* or what you read meanwhile. Cheap, highly available, and the source
  of the replication-lag anomalies from lesson 3.

Stronger consistency costs latency and availability. Pick the *weakest* model your
application can correctly tolerate — but no weaker.

## 3. Linearizability, precisely

**Linearizability** is the strongest single-object guarantee. The system behaves as
if every operation took effect **atomically at a single instant** between its
invocation and completion, consistent with **real-time order**: if write W completes
before read R begins (in wall-clock real time), R *must* see W (or something newer).

In short: it looks like there is **one copy of the data, and everyone sees the same,
up-to-date value**. No "I wrote it but my next read didn't see it." That intuitive
behavior is *exactly* what's expensive to provide across replicas — it generally
requires coordination (consensus) on every operation.

> **Linearizability (a recency/single-object guarantee) is not the same as
> serializability (a transaction-isolation guarantee from Step 6).** Serializability
> says transactions appear to run one-at-a-time in *some* order; linearizability
> adds that the order respects real time, for single objects. "Strict
> serializability" is roughly both at once.

## 4. The cost, and why we still chase it

Providing linearizability means replicas must **agree** before responding — that's
network round trips, and under a partition it means *some* nodes must stop serving
(CP). So you only pay for it where correctness demands a single source of truth:
leader election, distributed locks, unique-constraint enforcement, config that all
nodes must agree on.

And that agreement — getting multiple nodes to commit to the same value despite
failures and partitions — is **consensus**. Which is the next lesson.

## 5. A practical decision guide

- Money, inventory, "exactly one leader," uniqueness → lean **strong/CP**.
- Feeds, likes, presence, caches, carts → **eventual/AP** is usually fine (and far
  more available).
- Many real systems are **mixed**: strong for the few critical invariants, eventual
  for the rest. Knowing *which data needs which* is the skill.

## Do it yourself (≈ 6 hrs)

1. Read **DDIA Chapter 9** intro on consistency + linearizability (you'll finish Ch.9
   in lesson 7).
2. Watch Kleppmann's lectures on **consistency & consensus** ([course playlist](https://www.youtube.com/playlist?list=PLeKd45zvjcDFUEv_ohr_HdUFe97RItdiB)).
3. Read the classic "[**Please stop calling databases CP or AP**](https://martin.kleppmann.com/2015/05/11/please-stop-calling-databases-cp-or-ap.html)" by Kleppmann — it sharpens the nuance.
4. For three systems you use, classify each as leaning CP or AP and justify it.

## Check yourself

- Why is "pick 2 of 3" a misleading reading of CAP? State the honest version.
- Give a use case that should lean CP and one that should lean AP — and why.
- Define linearizability in plain words. What does it feel like to a client?
- How does linearizability differ from serializability (Step 6)?
- Why does strong consistency generally require consensus (and cost availability under a partition)?

Next: **consensus — the Raft algorithm.** Everything converges here.

## Common interview gotchas

- **CAP's "C" is *linearizability*, not ACID's "C".** Same letter, unrelated guarantees. CAP-consistency is about replicas showing the latest write in real-time order; ACID-consistency is about a transaction preserving application invariants. A single-node SQL database can be perfectly ACID-consistent yet, once async-replicated, not linearizable.
- **"Pick 2 of 3" is wrong — partition tolerance isn't optional.** Real networks drop and delay messages, so P is a given. The real statement is: *during* a partition you choose C or A. Frame it that way or you'll look like you learned CAP from a slide.
- **Linearizability ≠ serializability.** Linearizability is single-object *recency* (real-time order); serializability is *transaction ordering* (some serial order, no real-time claim). "Strict serializability" is both at once — the multi-object generalization of linearizability.
- **CAP ignores the no-partition case — PACELC fixes that.** Partitions are rare; latency-vs-consistency bites you 99.9% of the time. Even a healthy linearizable system pays a coordination round-trip per operation. PACELC: *if Partition → A or C; Else → Latency or Consistency.*
- **Eventual consistency says *nothing* about WHEN.** "Replicas converge if writes stop" is a promise about the limit, not a bound. Don't treat "eventually" as "quickly" — there's no guarantee on staleness window unless the system gives you one.
