---
slug: db-transactions-acid
step: 6
title: Transactions & ACID
summary: What a transaction guarantees, what ACID really means, and how isolation levels trade safety for speed.
est_min: 360
position: 1
---

# Transactions & ACID

> **Step 6 · Databases, deep · Week 1**
> Concept: *atomicity, consistency, isolation, durability*

You can read and write rows (Step 4). Now go deeper: how does a database keep your
data *correct* when multiple operations — and multiple clients — hit it at once? The
answer is the **transaction**, and the guarantees it makes are **ACID**.

## Why this matters

A transaction is the unit of "all or nothing." Transfer money, place an order,
sign up a user-and-create-their-workspace — these are multi-step writes that must
never half-happen. ACID is the contract that lets you reason about correctness, and
its limits are exactly where distributed systems (Step 7) get hard.

## 1. A transaction is all-or-nothing

```sql
BEGIN;
  UPDATE accounts SET balance = balance - 100 WHERE id = 1;
  UPDATE accounts SET balance = balance + 100 WHERE id = 2;
COMMIT;   -- both happen, or (on ROLLBACK / crash) neither does
```

If anything fails between `BEGIN` and `COMMIT`, you `ROLLBACK` and the database is
as if nothing happened. No state where money left account 1 but never arrived at 2.

## 2. ACID, concretely

- **Atomicity** — all statements in the transaction succeed, or none do. (The
  transfer above.)
- **Consistency** — the transaction moves the DB from one valid state to another;
  constraints (foreign keys, `CHECK`, `UNIQUE`) are never left violated.
- **Isolation** — concurrent transactions don't see each other's half-finished
  work; the result is *as if* they ran one after another (to a degree set by the
  isolation level — section 4).
- **Durability** — once `COMMIT` returns, the data survives a crash (it's on disk /
  in the write-ahead log).

Atomicity and isolation are the two you'll think about daily.

## 3. Transactions from Go (pgx)

```go
tx, err := pool.Begin(ctx)
if err != nil {
	return err
}
defer tx.Rollback(ctx) // no-op if we already committed; safety net on early return

if _, err := tx.Exec(ctx, `UPDATE accounts SET balance = balance - $1 WHERE id=$2`, amt, from); err != nil {
	return err // deferred Rollback fires
}
if _, err := tx.Exec(ctx, `UPDATE accounts SET balance = balance + $1 WHERE id=$2`, amt, to); err != nil {
	return err // deferred Rollback fires
}
return tx.Commit(ctx)
```

The `defer tx.Rollback(ctx)` pattern is the idiom: if any step returns early,
rollback runs; if you reach `Commit`, the rollback becomes a no-op. (The Consensus
seeder and `ResetProgress` use exactly this shape.)

## 4. Isolation levels — the safety/speed dial

Full isolation (every transaction acts as if alone) is expensive. SQL defines
levels that trade safety for concurrency, each permitting certain **anomalies**:

| Level | Dirty read | Non-repeatable read | Phantom |
|---|---|---|---|
| Read Uncommitted | possible | possible | possible |
| Read Committed *(Postgres default)* | no | possible | possible |
| Repeatable Read | no | no | possible* |
| Serializable | no | no | no |

- **Dirty read** — you see another transaction's *uncommitted* write.
- **Non-repeatable read** — you read a row twice in one transaction and get
  different values (someone committed a change between).
- **Phantom** — you run the same query twice and new rows *appear* (someone
  inserted matching rows). (*Postgres's Repeatable Read prevents these too via MVCC.)

Higher level = fewer anomalies = more locking/aborts = less concurrency. Most apps
run at Read Committed and reach for Serializable only on the few transactions that
truly need it.

```sql
BEGIN ISOLATION LEVEL SERIALIZABLE;
  -- ... critical multi-step read-then-write ...
COMMIT;  -- may fail with a serialization error → your app retries the whole tx
```

At Serializable, the database may *abort* a transaction to preserve correctness;
your code must be ready to **retry** it.

## 5. Keep transactions short

A transaction holds resources (and possibly locks) until it ends. Long-running
transactions hurt concurrency and can deadlock. Do slow work (HTTP calls, heavy
compute) *outside* the transaction; keep `BEGIN…COMMIT` tight around the writes.

## Do it yourself (≈ 6 hrs)

1. Watch the transactions/ACID lectures of [**CMU 15-445**](https://15445.courses.cs.cmu.edu/) — the best free DB course there is.
2. Read the Postgres [**transactions tutorial**](https://www.postgresql.org/docs/current/tutorial-transactions.html).
3. Open two `psql` sessions. In one, `BEGIN` and update a row but don't commit; in
   the other, read it. Observe Read Committed hiding the uncommitted change.
4. Write the Go transfer transaction above against a 2-row `accounts` table; force
   the second statement to fail and confirm the first is rolled back.

## Check yourself

- What does each letter of ACID guarantee, in your own words?
- Walk through why the bank transfer must be atomic.
- What's Postgres's default isolation level, and which anomaly does it still allow?
- What must your application code do differently when running at Serializable?
- Why should transactions be kept short?

Next: **indexing** — why some of those queries are slow, and how to fix them.

## Common interview gotchas

- **"Postgres default is Serializable / fully safe."** No — it's Read Committed, which still permits non-repeatable reads, phantoms, *and* lost updates on app-side read-modify-write. Safety you didn't ask for, you don't get.
- **"Serializable just makes it slower."** It also *aborts* conflicting transactions with a serialization failure (SQLSTATE `40001`). Without an application **retry loop**, you've traded a silent corruption for a runtime error.
- **"I'll send the confirmation email / call the payment API inside the transaction."** Don't — side effects can't be rolled back, and they make the tx long and un-retryable. Do external I/O *after* commit; keep `BEGIN…COMMIT` tight so a retry is safe and idempotent.
- **"ACID's Consistency is the same as CAP's Consistency."** Different words. ACID's C = your constraints/invariants hold; CAP's C = every read sees the latest write across nodes (linearizability). Conflating them is a classic distributed-systems tell.
- **"Durability means it's flushed to the data files on COMMIT."** It means it's in the durable **write-ahead log**; the heap pages may still be dirty in memory. Also, `synchronous_commit=off` trades a small durability window for throughput — COMMIT can return before the WAL is fsynced.
