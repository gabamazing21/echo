---
slug: db-concurrency-mvcc
step: 6
title: Concurrency control — locks, MVCC & lost updates
summary: What happens when two transactions touch the same row — locking, the lost-update bug, optimistic versioning, and MVCC.
est_min: 360
position: 3
---

# Concurrency control — locks, MVCC & lost updates

> **Step 6 · Databases, deep · Week 2**
> Concept: *what happens when two writes collide*

Isolation (lesson 1) promised transactions don't corrupt each other. This lesson is
*how* the database delivers that when two transactions want the same row at the same
time — and the classic bug that bites you when you don't think about it.

## Why this matters

Race conditions don't only live in Go (Step 5) — they live in your data. Two users
editing the same record, two workers claiming the same job: get concurrency control
wrong and you silently lose writes or double-process. This is also the direct
on-ramp to *distributed* consensus in Step 7.

## 1. The lost-update problem

Two transactions read the same value, both modify it, both write back. One update
silently vanishes:

```
T1: read balance (100)                 T2: read balance (100)
T1: write 100 - 10 = 90
                                       T2: write 100 + 50 = 150   ← clobbers T1!
```

The -10 is lost. The final balance is 150, should be 140. Neither transaction did
anything obviously wrong — the bug is the *interleaving*.

## 2. Fix 1 — do the math in the database

Often the simplest fix: don't read-then-write in your app; let the DB compute
atomically.

```sql
UPDATE accounts SET balance = balance + 50 WHERE id = 1;
```

The row is locked for the duration of the statement, so concurrent increments
serialize correctly. Prefer this whenever the new value is a function of the old.

## 3. Fix 2 — pessimistic locking (SELECT FOR UPDATE)

When you must read, decide in app code, then write, **lock the row on read** so
no one else can touch it until you commit:

```sql
BEGIN;
  SELECT balance FROM accounts WHERE id = 1 FOR UPDATE; -- locks the row
  -- ... compute in app ...
  UPDATE accounts SET balance = $1 WHERE id = 1;
COMMIT;                                                  -- lock released
```

T2's `SELECT … FOR UPDATE` blocks until T1 commits — they serialize. "Pessimistic"
= assume conflict, lock up front. Cost: contention; risk: **deadlock** (two
transactions each holding a lock the other wants — the DB detects it and aborts
one, which your app must retry).

## 4. Fix 3 — optimistic concurrency (version column)

Assume conflicts are *rare*; don't lock. Add a `version` column and only write if it
hasn't changed since you read it:

```sql
UPDATE accounts SET balance = $1, version = version + 1
WHERE id = 1 AND version = $2;          -- $2 = version you read
```

If `RowsAffected() == 0`, someone else updated it first — you re-read and retry.
Great for low-contention, read-heavy workloads (no locks held); wasteful if
conflicts are frequent (lots of retries).

## 5. MVCC — how Postgres avoids read locks entirely

Postgres uses **Multi-Version Concurrency Control**: each write creates a *new
version* of the row rather than overwriting it. Each transaction sees a consistent
**snapshot** as of when it started. The headline consequence:

> **Readers never block writers, and writers never block readers.**

A `SELECT` reads the version visible to its snapshot while an `UPDATE` writes a new
version alongside — no read lock needed. Writers still conflict with *other writers*
on the same row (one waits or aborts). Old versions are cleaned up later by
`VACUUM`. This is why Postgres handles read-heavy concurrency so gracefully, and
how it implements the isolation levels from lesson 1.

## 6. Foreshadowing: this is consensus in miniature

"Two parties want to update the same state; how do we agree on the result without
losing writes?" — on one machine, that's locks and MVCC. Across *many* machines
with unreliable networks, that same question becomes **distributed consensus**
(Raft, Step 7). Same problem, harder setting. Hold that thought.

## Do it yourself (≈ 6 hrs)

1. Read the Postgres [**MVCC docs**](https://www.postgresql.org/docs/current/mvcc.html) (intro + transaction isolation).
2. Watch the concurrency-control lectures from [**CMU 15-445**](https://15445.courses.cs.cmu.edu/).
3. Reproduce a lost update with two `psql` sessions doing read-modify-write on the
   same row, then prevent it three ways: in-DB math, `SELECT … FOR UPDATE`, and a
   `version` column.
4. Cause a deadlock (two sessions lock two rows in opposite order) and read how
   Postgres aborts one.

## Check yourself

- Walk through a lost update and say exactly which write is lost.
- When can you avoid the whole problem by doing the math in SQL?
- Contrast pessimistic (`FOR UPDATE`) vs optimistic (version column) — when is each better?
- What does MVCC let readers and writers do that lock-based control doesn't?
- How is a single-row write conflict a tiny version of distributed consensus?

Next: **the project** — index, EXPLAIN, and wrap a multi-step write in a transaction.

## Common interview gotchas

- **"Wrapping a read-modify-write in `BEGIN…COMMIT` prevents lost updates."** At Read Committed it does **not** — both txns can read the old value and the second clobbers the first. Do the math in SQL (`SET balance = balance + $1`), or lock with `FOR UPDATE`, or use a version column.
- **"`SELECT … FOR UPDATE` is the safe default."** It serializes writers but introduces **deadlock** risk if two transactions lock the same rows in *different order*. Always acquire locks in a consistent order (e.g. by ascending id) to avoid the cycle.
- **"MVCC means I never have to worry about cleanup."** Every UPDATE/DELETE leaves a dead tuple; without **VACUUM** (autovacuum) the table and its indexes **bloat**, slowing scans and risking transaction-ID wraparound. MVCC trades read-locking for a garbage-collection obligation.
- **"Readers never block writers, so there's no contention."** True for reads vs writes — but two writers to the *same row* still serialize: one waits, or aborts under Serializable. MVCC doesn't remove write-write conflicts, it removes read locks.
- **"Optimistic (version-column) concurrency is always cheaper than locking."** Only under **low contention**. When conflicts are frequent, `RowsAffected()==0` triggers re-read-and-retry storms; pessimistic `FOR UPDATE` is cheaper for hot rows.
