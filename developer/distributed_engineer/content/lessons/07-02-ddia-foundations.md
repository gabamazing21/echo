---
slug: ds-ddia-foundations
step: 7
title: "DDIA Ch.1–4: reliability, data models & storage engines"
summary: The foundations of data systems — reliability/scalability/maintainability, data models, and how storage engines actually work.
est_min: 600
position: 2
---

# DDIA Ch.1–4: reliability, data models & storage engines

> **Step 7 · Distributed systems theory · Week 1**
> Concept: *foundations of data systems*

*Designing Data-Intensive Applications* (DDIA) by Martin Kleppmann is **the** book
of this field. This lesson is your guided read of the first four chapters — the
foundations everything else builds on. Read the chapters; use this as the map and
the "make sure you got this" companion.

## Why this matters

DDIA is the most-recommended book for distributed/backend engineers, full stop. If
you internalize it, you can reason about almost any data system. Interviews assume
its vocabulary. This lesson front-loads the foundations so the harder chapters
(lesson 7) land.

## 1. (Ch.1) Reliability, scalability, maintainability

Three properties every data system is judged on:

- **Reliability** — works correctly *even when things go wrong*. Faults (hardware,
  software, human) are inevitable; a reliable system is **fault-tolerant** —
  it contains faults so they don't become failures. (Netflix's Chaos Monkey
  deliberately kills nodes to prove this.)
- **Scalability** — copes with growth in load. Define *load parameters* (requests/
  sec, read/write ratio, fan-out) and measure *performance* with **percentiles**
  (p50, p95, p99), not averages — tail latency is what users feel.
- **Maintainability** — operability, simplicity, evolvability. Most cost is in
  *ongoing* maintenance, not initial build.

Key idea: **faults vs failures.** A fault is one component misbehaving; a failure
is the whole system not delivering service. The goal is preventing faults from
cascading into failures.

## 2. (Ch.2) Data models & query languages

The data model shapes how you think. The big families:

- **Relational** — data as tables/rows; great for many-to-many relationships and
  joins; query with SQL (declarative — you say *what*, the engine decides *how*).
- **Document** (e.g. JSON stores) — good for tree-shaped, one-to-many data with
  locality; weaker at many-to-many; "schema-on-read."
- **Graph** — when many-to-many relationships dominate (social networks, routing).

The **object–relational mismatch** (objects in code vs rows in tables) is why ORMs
exist. There's no universally best model — it depends on your data's *relationships*.

## 3. (Ch.3) Storage engines — how a DB stores bytes

This is the chapter that demystifies databases. Two big families of write path:

- **B-trees** (what Postgres/most SQL DBs use) — the on-disk balanced tree from
  Step 6. Update-in-place, great for reads, the default workhorse.
- **LSM-trees** (Log-Structured Merge trees; LevelDB, RocksDB, Cassandra) — writes
  go to an in-memory *memtable*, flushed to immutable sorted files (SSTables) that
  are merged/compacted in the background. **Write-optimized**: sequential writes are
  fast.

The trade-off: **LSM-trees favor writes; B-trees favor reads.** Both rely on the
idea that *sequential* disk access is vastly faster than *random*. (You'll meet
LSM-trees again building storage in Step 8.) Also here: OLTP (transaction
processing, row-oriented) vs OLAP (analytics, column-oriented) — different storage
for different access patterns.

## 4. (Ch.4) Encoding & evolution

Systems change; data outlives code. This chapter is about **serialization formats**
and, crucially, **schema evolution**:

- Formats: JSON/XML (human-readable, verbose), and binary schemas like
  **Protocol Buffers** / **Avro** (compact, typed, fast).
- **Backward compatibility** — new code can read old data.
- **Forward compatibility** — old code can read new data (ignore unknown fields).

In a distributed system you do **rolling upgrades** — old and new versions run
*simultaneously* — so your encoding must tolerate both directions. Get this wrong
and a deploy corrupts data.

## 5. The thread through all four chapters

Every choice — model, storage engine, encoding — is a **trade-off** against your
access pattern. There is no "best database"; there's the right tool for *your*
reads, writes, relationships, and growth. DDIA's gift is teaching you to reason
about those trade-offs instead of cargo-culting.

## Do it yourself (≈ 10 hrs)

1. **Read DDIA chapters 1–4.** This is the work; budget real time. (Library, or buy
   it — it's worth owning.) [dataintensive.net](https://dataintensive.net/)
2. As a free companion, watch the relevant [**Martin Kleppmann distributed systems
   lectures**](https://www.youtube.com/playlist?list=PLeKd45zvjcDFUEv_ohr_HdUFe97RItdiB) (Cambridge).
3. After Ch.3, write one paragraph: when would you pick an LSM-tree store over a
   B-tree one, and why?
4. After Ch.4, sketch how you'd add a field to a message format without breaking a
   rolling upgrade.

## Check yourself

- Distinguish a *fault* from a *failure*. Why measure latency with p99, not average?
- When does a document model beat relational, and when does it lose?
- Contrast B-tree and LSM-tree storage: which favors reads, which writes, and why?
- What are backward vs forward compatibility, and why does a rolling upgrade need both?
- What's the single biggest idea you take from DDIA Ch.1–4?

Next: **replication** — keeping copies of your data on multiple nodes.

## Common interview gotchas

- **Averages launder the tail.** A mean of 95 ms can hide that 1-in-20 requests take a second. p99 is what a real user with *many* sub-requests actually feels — and with fan-out it gets worse: 100 leaves each 99% fast means only `0.99^100 ≈ 37%` of parents are fully fast, so ~63% hit a slow leaf. Your fan-out p99 is governed by each leaf's *p99.9*, not its p50.
- **LSM "fast writes" still pay later.** Up-front writes are cheap and sequential, but **compaction** rewrites the same keys across levels — cumulative write amplification of 10x+ — and reads probe multiple SSTables (read amplification). Compaction also competes for I/O and can quietly wreck your p99.
- **B-trees write the data twice.** Every change hits the **WAL** *and* the page, and a one-row update can rewrite a whole 4–16 KB page. The payoff is cheap reads (one tree path). Low read amplification, in exchange for that double-write.
- **A rolling upgrade needs backward AND forward compat at the same time.** Old and new code run simultaneously. The classic bug: old code reads a record with a new field, doesn't recognize it, and **drops it on re-write** — silent data loss. Old code must *ignore unknown fields and preserve them on round-trip*, not discard them (Protobuf keeps unknown fields; Avro needs defaults).
- **Schema-on-read isn't schemaless.** There's still a schema — it just lives in your application code at read time instead of being enforced by the database on write. You've moved the contract, not deleted it.
