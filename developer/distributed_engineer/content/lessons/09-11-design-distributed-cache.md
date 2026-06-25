---
slug: sd-distributed-cache
step: 9
title: "Design: a distributed cache"
summary: Consistent hashing, eviction, write policies, and the failure modes (stampede, penetration) behind a Redis/Memcached-style cache.
est_min: 300
position: 11
---

# Design: a distributed cache

> **Step 9 · System design · Interview prep**
> Concept: *consistent hashing, eviction, cache failure modes*

Caching shows up in *every* system-design answer ("add a cache"), so interviewers
often ask you to actually design one. It pulls together consistent hashing (Step 7),
eviction policy (LRU, Step 2), and the subtle failure modes that bite in production.

## Why this matters

A distributed cache (Redis/Memcached cluster) is the most common scaling lever, and
"how does your cache behave when it fails / a hot key appears / it's cold" separates
people who say "add a cache" from people who understand one.

## 1. Requirements

A key→value store, in memory, across many nodes: O(1) `get`/`set`, low latency, high
hit rate, horizontally scalable, tolerant of node loss. It's a *cache* — losing data
is acceptable (it's reconstructable from the source of truth), which simplifies a lot
versus a database.

## 2. Distributing keys: consistent hashing

Sharding by `hash(key) % N` remaps almost every key when N changes (a node added or
lost) — a cache-wide miss storm. **Consistent hashing** (a hash ring with virtual
nodes) means adding/removing a node only remaps ~1/N of keys. Virtual nodes also even
out load. This is *the* reason consistent hashing exists (Step 7, partitioning).

## 3. Eviction policy

Memory is bounded, so you evict. **LRU** (least-recently-used, the Step 2 challenge)
is the common default; **LFU** (least-frequently-used) suits skewed popularity;
**TTL** expires entries by time. The policy + size determine your hit rate — the
metric that justifies the cache existing.

## 4. Write policies (cache + DB consistency)

- **Cache-aside (lazy)**: app reads cache; on miss, loads from DB and populates.
  Writes go to the DB and **invalidate** the cache key. Simplest, most common.
- **Write-through**: write to cache and DB together — cache always fresh, slower
  writes.
- **Write-back**: write to cache, flush to DB later — fast, but risks data loss on
  crash. Rare for durable data.

The hard part is **invalidation** — "there are only two hard things… cache
invalidation and naming." A stale key after a DB write is the classic bug; prefer
invalidate-on-write + short TTLs.

## 5. The failure modes interviewers probe

- **Cache stampede / thundering herd**: a hot key expires and thousands of requests
  miss simultaneously and all hit the DB. Fix: a lock/single-flight so one request
  recomputes while others wait, or staggered TTLs, or async refresh-ahead.
- **Cache penetration**: queries for keys that don't exist bypass the cache and
  hammer the DB. Fix: cache negative results, or a **Bloom filter** to reject absent
  keys cheaply.
- **Cold start**: a fresh/restarted cache has 0% hit rate and overwhelms the DB. Fix:
  warm it, or rate-limit DB fallback.
- **Hot key**: one key gets disproportionate traffic, overloading its shard. Fix:
  replicate the key, or add a small local (client-side) cache.

## Do it yourself (≈ 4 hrs)

1. Read [**System Design Primer — Cache**](https://github.com/donnemartin/system-design-primer#cache) and the Redis docs on eviction.
2. Implement the in-app **LRU Cache** checker exercise (Step 2) — then explain how you'd distribute it with consistent hashing.

## Check yourself

- Why consistent hashing instead of `hash % N` for distributing keys?
- Contrast cache-aside, write-through, and write-back.
- Why is invalidation the hard part, and how do you mitigate stale reads?
- What is a cache stampede, and how do you prevent it?
- How do Bloom filters help with cache penetration?

## Common interview gotchas

- **`hash(key) % N` is the wrong sharding scheme** — changing N remaps nearly all keys and causes a cluster-wide miss storm; use consistent hashing with virtual nodes.
- **Cache invalidation is the real problem, not lookups** — a stale key after a DB write is the classic bug; pair invalidate-on-write with short TTLs, and never assume the cache and DB are in lockstep.
- **An expiring hot key causes a stampede** — thousands of simultaneous misses hit the DB at once; use single-flight/locking or staggered TTLs, not a bare TTL.
- **Cache penetration bypasses you entirely** — requests for non-existent keys never hit the cache; cache negatives or front it with a Bloom filter.
- **"Just add a cache" isn't free** — it adds an invalidation/consistency problem and a new failure mode (cold start, hot keys); name those trade-offs instead of hand-waving.
