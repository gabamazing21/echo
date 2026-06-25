---
slug: sd-url-shortener
step: 9
title: "Design: a URL shortener"
summary: Work a classic system-design question end-to-end — hashing, storage, caching and read-heavy scale.
est_min: 240
position: 2
---

# Design: a URL shortener

> **Step 9 · System design · Week 1 · Design**
> Concept: *hashing, caching, read-heavy scale*

Time to use the framework from the last lesson on a real question. "Design a URL
shortener" (think TinyURL, bit.ly) is the *Hello, World* of system-design
interviews — small enough to finish in 30 minutes, rich enough to surface every
core skill: estimation, API design, choosing an ID scheme, picking storage, and
above all **scaling a wildly read-heavy workload**. Walk it the same way every
time and it stops being scary.

## Why this matters

The shortener looks trivial — it's "a hash map with a web server." That's the
trap. The interesting engineering is entirely in the *scale and shape* of the
traffic: redirects vastly outnumber creations, so the whole design bends around
making reads cheap. Get fluent here and you have a template for half the
"design X" questions you'll ever face.

## 1. Requirements

Pin down scope first — don't build features nobody asked for.

**Functional:**
- **Shorten:** given a long URL, return a short URL (`https://sho.rt/aZ4q9`).
- **Redirect:** given the short code, send the browser to the original URL.

**Out of scope (say so explicitly):** custom vanity codes, user accounts,
analytics dashboards, link expiry. Mention them, then park them.

**Non-functional — the part that matters:**
- **Read-heavy, ~100:1 read:write.** Far more people *click* links than create
  them. This single fact drives the whole design.
- **Low-latency redirects.** A redirect should feel instant (single-digit ms).
- **High availability for reads.** A redirect failing is very visible.
- **Consistency is relaxed.** If a *brand-new* link is briefly unavailable on
  some replica, that's tolerable — we lean **AP** (see Step 7). We are not a bank.

## 2. Estimation (back-of-the-envelope)

Numbers anchor every later decision. Assume **100M new URLs/day**.

- **Writes:** 100M / 86,400s ≈ **~1,160 writes/sec**.
- **Reads:** at 100:1, ≈ **~116,000 reads/sec**. *This* is the number to design for.
- **Storage:** each row ≈ short code + long URL + metadata ≈ **~500 bytes**.
  100M/day × 500 B ≈ **50 GB/day** → ~18 TB/year. Easily fits on disk; grows
  linearly, so we'll need a plan for sharding eventually but not on day one.
- **Code space:** base62 (`a–z A–Z 0–9`). A **7-character** code gives
  62⁷ ≈ **3.5 trillion** combinations — decades of runway at 100M/day. So **7
  chars** it is.

> Don't chase precision. The goal is the *order of magnitude*: "hundreds of
> writes, ~100k reads, tens of GB a day." That tells you reads need a cache and
> writes don't.

## 3. API & data model

Two endpoints, mapping straight onto the two requirements:

```
POST /shorten
  body: { "url": "https://example.com/some/very/long/path" }
  201:  { "short": "https://sho.rt/aZ4q9" }

GET /:code
  302/301 redirect  ->  Location: <original long URL>
```

Use **301 (permanent)** if you want browsers/CDNs to cache the redirect — cheap
and fast, but you lose per-click visibility. Use **302 (temporary)** if you ever
want to count clicks or change the target. For a pure shortener, **301** wins on
performance; call out the trade-off either way.

**Data model** — one table, indexed by the code:

```
urls
  code        VARCHAR(7)   PRIMARY KEY   -- the short code
  long_url    TEXT         NOT NULL
  created_at  TIMESTAMP
```

The primary key *is* the code, so a redirect is a single point lookup — the
cheapest query a database can do.

## 4. Short-code generation — the core design decision

How do we turn a long URL into a unique 7-char code? Three approaches, each with
a different collision story. This is the part interviewers probe hardest.

**A. Hash + truncate.** Hash the long URL (MD5/SHA), base62-encode, take the
first 7 chars.
- *Pro:* stateless, same URL → same code (natural dedup).
- *Con:* **collisions** — two different URLs can truncate to the same 7 chars.
  You must check the DB and, on a clash, re-hash with a salt or grab more chars.
  Collision-handling adds a read on the write path.

**B. Counter + base62.** Keep a global incrementing integer; base62-encode it to
get the code.
- *Pro:* **collisions are impossible by construction** — every counter value is
  unique. Simple and dense.
- *Con:* needs a single source of truth for the counter (a bottleneck / single
  point of failure), and codes are *sequential* and guessable (1, 2, 3 → `b`,
  `c`, `d`), which leaks volume and invites enumeration.

**C. Key-Generation Service (KGS).** A separate service pre-generates a big pool
of unique random codes offline and hands them out on demand.
- *Pro:* **no collision check on the write path** (codes are known-unique), no
  hot counter, codes aren't sequential. Fast writes.
- *Con:* extra moving part; must track used vs. unused keys and survive the KGS
  restarting (keep a small in-memory buffer, mark keys used atomically).

> **Verdict for the interview:** start with **B (counter/base62)** as the simple
> baseline, then propose **C (KGS)** as the scalable answer when they push on the
> counter bottleneck and guessability. Mention A to show you know the hashing
> angle and why collision-checking makes it less clean at scale.

## 5. High-level design (in words)

Trace the two flows through the boxes:

```
            +----------+      +-----------+      +-----------+
 client --> |   LB     | ---> |  app/API  | ---> |  cache    |
            +----------+      +-----------+      +-----------+
                                   |                  | miss
                                   v                  v
                              +-----------+      +-----------+
                              |    KGS    |      | datastore |
                              | (writes)  |      | (primary) |
                              +-----------+      +-----------+
```

- **Write path (`POST /shorten`):** app server asks the **KGS** for an unused
  code, writes `(code, long_url)` to the datastore, returns the short URL. Low
  volume (~1k/s) — no exotic scaling needed.
- **Read path (`GET /:code`):** app server checks the **cache** first; on a hit,
  redirect immediately. On a miss, read the datastore, populate the cache, then
  redirect. This is where all the load lives.

## 6. Scaling & trade-offs — making reads cheap

~116k reads/sec is the whole game. Layer the defences:

- **Cache the hot codes.** Link popularity is heavily skewed (a few viral links
  get most clicks), so a relatively small cache (Redis/Memcached) absorbs the
  vast majority of reads. **Cache-aside + LRU eviction**: on a miss, read DB and
  backfill. Mappings are immutable, so entries never go stale — caching is easy
  here. Aim for a high hit ratio and the DB barely sees read traffic.
- **Read replicas.** Point cache-miss reads at **follower replicas** of the
  datastore; keep the leader for writes. Reads scale horizontally by adding
  followers. Replication lag is fine — we already accepted **AP** (a just-created
  link missing from a replica for a moment is acceptable).
- **CDN / edge for redirects.** With **301** responses, a CDN can cache the
  redirect itself at the edge, serving repeat clicks without ever touching your
  app. The cheapest request is the one that never reaches your servers.
- **Stateless app tier.** App servers hold no state, so put them behind a load
  balancer and scale out horizontally as traffic grows.
- **Sharding (later).** When the dataset outgrows one machine, **shard by code**
  (consistent hashing on the code). Point lookups by primary key shard cleanly;
  this is a "when we're huge" concern, not day one.

**The trade-offs to say out loud:**
- **301 vs 302:** edge-cacheable speed *vs.* click analytics and editability.
- **Strong vs eventual consistency:** we chose **eventual/AP** — availability and
  latency over a freshly-created link being instantly visible everywhere.
- **Counter vs KGS:** simplicity *vs.* removing the write bottleneck and
  un-guessable codes.
- **Cache size vs hit ratio:** more memory buys a higher hit ratio and less DB
  load — tune to the skew of your traffic.

## Do it yourself (≈ the rest of the session)

1. On a whiteboard (or paper), reproduce the **six sections** from memory:
   requirements → estimation → API & data model → code generation → high-level
   design → scaling. Time yourself.
2. Redo the estimation assuming **1B URLs/day**. Does **7 chars** still suffice?
   What breaks first — code space, storage, or read throughput?
3. Skim the **URL-shortener** and **caching** chapters of the free
   [**System Design Primer**](https://github.com/donnemartin/system-design-primer) —
   compare its choices to yours and note where they differ.
4. Defend one decision you'd change if reads were only 2:1 instead of 100:1.

## Done when

You can **whiteboard the whole design in ~30 minutes**, out loud, hitting:

- the two endpoints and the 100:1 read:write shape;
- an order-of-magnitude estimate (reads ≈ 100× writes; 7-char base62 codes);
- the three code-generation approaches and the **collision** story for each;
- the read-path stack — **cache → replicas → CDN** — and *why* each layer exists;
- why **eventual consistency (AP)** is the right call here.

Next: **another design exercise — a rate limiter.**

## Common interview gotchas

- **Claiming hash+truncate is collision-free.** Truncating a hash to 7 chars *can* collide — two URLs mapping to the same code. You must check the DB and retry with a salt or more chars on a clash, which adds a read to the write path. Forgetting this is an instant red flag.
- **Sequential counter codes are a security hole.** A global counter gives codes 1,2,3 → `b,c,d`: trivially enumerable. An attacker can crawl every link and your creation volume leaks. Reach for a KGS or randomized codes when guessability matters — don't hand-wave it.
- **301 vs 302 is not interchangeable.** A 301 (permanent) lets browsers and CDNs cache the redirect — fast and cheap, but you *never see the repeat clicks*, so analytics die. Use 302 when you need per-click counting or editable targets. State which and why.
- **Designing the write path for the read load.** It's ~100:1 read-heavy. Writes are ~1k/s and need no heroics; the entire design effort belongs on making the ~100k/s reads cheap — cache → replicas → CDN. Spending the interview on write scaling misreads the problem.
- **Sharding without consistent hashing.** Naively sharding by `code % N` means re-sharding rehomes almost every key when you add a node. Shard with consistent hashing so only a slice moves — and flag that re-sharding is the expensive part, not the steady state.
