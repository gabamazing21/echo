---
slug: project-db-tuning
step: 6
title: "Project: index, EXPLAIN & transactional writes"
summary: Make a slow query fast with the right index, prove it with EXPLAIN ANALYZE, and wrap a multi-step write in a safe transaction.
est_min: 420
position: 4
---

# Project: index, EXPLAIN & transactional writes

> **Step 6 · Databases, deep · Week 2 · Project**
> Concept: *real query tuning + safe writes*

Theory lands when you measure it yourself. You'll take the bookmarks database from
Step 4, make a real query slow, fix it with an index you *chose* from the plan, then
make a multi-step write safe with a transaction — and prove rollback works.

## What you'll build

A small, reproducible experiment: seed lots of rows → find a slow query →
`EXPLAIN ANALYZE` → add the right index → measure the speedup → then a transactional
"create bookmark and bump a per-user counter" that rolls back cleanly on failure.

## Why this matters

This *is* the day-to-day of a backend engineer: "this endpoint is slow." You'll have
done the full loop — observe, diagnose with the plan, fix, verify — and you'll have
written a correct multi-step write. That's the difference between using a database
and *understanding* one.

## 1. Seed enough rows to matter

```sql
-- give bookmarks an owner if it doesn't have one yet
ALTER TABLE bookmarks ADD COLUMN IF NOT EXISTS user_id BIGINT NOT NULL DEFAULT 1;

-- 1,000,000 rows across 1,000 users
INSERT INTO bookmarks (url, title, user_id)
SELECT 'https://example.com/' || g,
       'Bookmark ' || g,
       (g % 1000) + 1
FROM generate_series(1, 1000000) AS g;
```

## 2. Find the slow query and read its plan

```sql
EXPLAIN ANALYZE
SELECT * FROM bookmarks WHERE user_id = 42 ORDER BY id DESC LIMIT 20;
```

Note the `Seq Scan` and the execution time (likely hundreds of ms). That seq scan
is the enemy.

## 3. Choose and add the index — then re-measure

```sql
-- YOUR TURN: pick the index from what the query filters and sorts by.
-- Hint: it filters by user_id and orders by id. What composite index serves both?
CREATE INDEX idx_bookmarks_user_id ON bookmarks (user_id, id DESC);

EXPLAIN ANALYZE
SELECT * FROM bookmarks WHERE user_id = 42 ORDER BY id DESC LIMIT 20;
```

Confirm the plan switched to an `Index Scan` and the time dropped by orders of
magnitude. **Write down both timings** — that before/after is the whole lesson.

## 4. A safe multi-step write (transaction)

Suppose creating a bookmark also bumps a `bookmark_count` on the user. Both must
happen, or neither:

```go
func CreateBookmarkAndCount(ctx context.Context, pool *pgxpool.Pool, b *Bookmark) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // safety net

	if err := tx.QueryRow(ctx,
		`INSERT INTO bookmarks (url, title, user_id) VALUES ($1,$2,$3) RETURNING id`,
		b.URL, b.Title, b.UserID).Scan(&b.ID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE users SET bookmark_count = bookmark_count + 1 WHERE id = $1`,
		b.UserID); err != nil {
		return err // deferred Rollback undoes the INSERT too
	}
	return tx.Commit(ctx)
}
```

## 5. Prove rollback

```go
// YOUR TURN:
//  - force the UPDATE to fail (e.g. point it at a non-existent column, or a user id
//    that violates a constraint) and assert the bookmark row was NOT inserted
//  - then fix it and assert BOTH the insert and the counter bump happened
//  - bonus: do the increment as `balance = balance + 1` in SQL (lesson 3) and explain
//    why that's safe under concurrency
```

## Stretch goals

- Add a second filter (e.g. `WHERE user_id=$1 AND created_at > $2`) and design the
  composite index for it; verify with `EXPLAIN`.
- Make the counter update concurrency-safe and reason about lost updates (lesson 3).
- Try `SELECT … FOR UPDATE` to claim-and-process a "job" row from two sessions.
- Run `EXPLAIN (ANALYZE, BUFFERS)` and note the shared-buffer hits.

## Done when

- [ ] You have a recorded before/after timing showing the index turning a `Seq Scan`
      into an `Index Scan`.
- [ ] You can justify *which* index you added from the query's filter + sort.
- [ ] The multi-step write commits atomically and **rolls back fully** when the
      second statement fails.
- [ ] You can explain why doing the counter bump in SQL avoids a lost update.

FREE source: [**Use The Index, Luke!**](https://use-the-index-luke.com/).

Next step: **Step 7 — Distributed systems theory.** This is the summit you've been climbing toward.

## Common interview gotchas

- **"Loop the per-row query — the index makes each one fast."** That's the **N+1** trap: 1000 fast queries still lose to one round-trip. Batch with `WHERE id = ANY($1)` or a `JOIN`; the network and planning overhead per call dominates, not the index.
- **"Add an index per query and you're done."** You'll accumulate **redundant indexes** — e.g. `(user_id)` is already covered by the leftmost prefix of `(user_id, id)`. Drop the narrower one; it only costs writes. Audit with `pg_stat_user_indexes` for never-used (`idx_scan=0`) indexes too.
- **"This index is unused, drop it."** Check first whether it backs a **UNIQUE or primary-key constraint** — that index enforces correctness, and dropping it (rather than the constraint) silently removes the guarantee or fails.
- **"`EXPLAIN` told me the query is fast."** Plain `EXPLAIN` only shows *estimated* cost; it never runs the query. Use **`EXPLAIN ANALYZE`** for real timing and actual-vs-estimated rows — and remember it actually executes, so wrap writes in a transaction you roll back.
- **"The counter bump in app code is fine inside the transaction."** Read-modify-write of `bookmark_count` is the lost-update bug again; `SET bookmark_count = bookmark_count + 1` does it atomically in SQL so concurrent inserts can't clobber each other.
