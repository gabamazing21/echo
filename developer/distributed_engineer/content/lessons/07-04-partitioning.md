---
slug: ds-partitioning
step: 7
title: Partitioning & sharding
summary: Splitting data across nodes — key-range vs hash partitioning, hot spots, rebalancing, and routing requests.
est_min: 360
position: 4
---

# Partitioning & sharding

> **Step 7 · Distributed systems theory · Week 2**
> Concept: *splitting data across nodes*

Replication copies *the same* data to many nodes. **Partitioning** (a.k.a.
**sharding**) splits *different* data across nodes — so your dataset and write
throughput can exceed any single machine. The two are used together: each partition
is also replicated.

## Why this matters

There's a ceiling to how big and how busy one machine can be. Partitioning is how
you blow past it — and it's the core of countless system-design answers (Step 9).
The catch is doing it without creating hot spots or making queries impossible.

## 1. The goal: spread load evenly

Each record lives on one partition (then replicated). You want the data — and the
*traffic* — spread evenly. A partition that gets disproportionate load is a **hot
spot** or "hot partition," and it bottlenecks the whole system as if you hadn't
partitioned at all. Even distribution is the whole game.

## 2. Partition by key range

Assign contiguous ranges of the key to partitions (like volumes of an encyclopedia:
A–C, D–F, …).

- ✅ **Range scans are efficient** (`WHERE key BETWEEN a AND b` hits few partitions).
- ❌ **Hot spots** if the key is sequential (e.g. timestamp): all *new* writes hit
  the latest partition. Writing by time? The "now" partition is always the hot one.

## 3. Partition by hash of key

Hash the key and assign by hash range (or modulo). This is the common default.

- ✅ **Even distribution** — a good hash scatters keys uniformly, killing hot spots.
- ❌ **Loses range queries** — adjacent keys land on different partitions, so range
  scans must hit *all* of them.

Trade-off in one line: **range partitioning keeps range queries but risks hot spots;
hash partitioning kills hot spots but loses range queries.** Some systems use a
*compound* key (hash the first part, range the second) to get both.

> ⚠️ **Don't use plain `hash(key) % N`** for assignment: changing `N` (adding a node)
> remaps almost *every* key → a massive reshuffle. Use **consistent hashing** or a
> fixed large number of partitions instead (next section).

## 4. Rebalancing when you add/remove nodes

When the cluster grows, partitions must move to the new node — **rebalancing** —
without (a) moving more data than necessary or (b) taking the system down.

- **Fixed number of partitions**: create many more partitions than nodes up front
  (e.g. 1000 partitions on 10 nodes). Adding a node just *reassigns whole partitions*
  to it — only the moved partitions' data transfers. Simple and popular.
- **Consistent hashing**: keys and nodes both map onto a ring; adding a node only
  reassigns the keys between it and its neighbor — minimal movement.

Both exist to avoid the "change N → remap everything" disaster.

## 5. Routing: which node has this key?

A client asks for key K — who serves it? Three approaches to this **request routing**
problem:

1. Clients can hit any node, which forwards to the right one.
2. A **routing tier** (proxy) knows the partition map and routes.
3. **Clients know the map** and connect directly.

Whichever you pick, *someone* must hold the authoritative partition-to-node mapping
and keep it current as rebalancing happens — and that metadata is often itself kept
consistent by a **consensus** service (e.g. ZooKeeper/etcd). Consensus keeps showing
up; lesson 6 is why.

## 6. Secondary indexes get tricky

Partitioning by primary key is clean. But "find all bookmarks tagged `go`" (a
*secondary* index) spans partitions: either each partition indexes its own local
data (scatter/gather reads across all partitions) or you maintain a global index
(harder writes). There's no free lunch — the access pattern decides.

## Do it yourself (≈ 6 hrs)

1. Read **DDIA Chapter 6 (Partitioning)**.
2. Watch the partitioning/sharding material in [**Kleppmann's course**](https://www.youtube.com/playlist?list=PLeKd45zvjcDFUEv_ohr_HdUFe97RItdiB).
3. Implement **consistent hashing** in Go: a hash ring with virtual nodes; add/remove
   a node and measure what fraction of keys move (should be ~1/N, not ~all).
4. Pick a real system (URL shortener, chat) and decide range vs hash partitioning for
   its main key — and justify it.

## Check yourself

- What's the difference between replication and partitioning, and why use both?
- What is a hot spot, and how does each partitioning scheme cause or avoid one?
- State the range-vs-hash trade-off in one sentence.
- Why is `hash(key) % N` a bad assignment scheme, and what do you use instead?
- Why does request routing often depend on a consensus system?

## Common interview gotchas

- **A good hash spreads keys, but does *nothing* for a single hot key.** One celebrity user is *one key* — it always hashes to the same partition. Even distribution fixes *aggregate* skew, not a hot *key*; you need application-level **key splitting** (random suffix → N sub-keys), dedicated partitions, or caching/replicas.
- **Naive consistent hashing (one ring point per node) has high load variance.** Random arc lengths mean some nodes own 2–3× the average. **Virtual nodes** (100s of points per node) average those arcs out (variance ≈ 1/√V) and let a leaving node's keys spread across many successors.
- **Adding the Nth node moves ≈ 1/N of keys with consistent hashing — vs. nearly *all* keys with `hash % N`.** Knowing that ratio is the whole reason the ring exists.
- **Global secondary indexes make *writes* cross-partition; local ones make *reads* scatter/gather.** Global (term-partitioned) reads hit one partition but a write updates index entries on several (distributed txn, often async). Local (document-partitioned) writes are cheap but every read queries all partitions. Read/write ratio decides.
- **"Fixed partition count" is effectively permanent — choose it high up front.** Too few and you can't spread over future nodes; too many and per-partition overhead piles up. You can't easily change it later without rehashing everything.

Next: **CAP, consistency models & linearizability** — the fundamental trade-off.
