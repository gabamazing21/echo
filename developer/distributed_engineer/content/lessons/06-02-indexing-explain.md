---
slug: db-indexing-explain
step: 6
title: Indexing — B-trees & EXPLAIN ANALYZE
summary: Why queries get slow, how B-tree indexes fix them, what indexes cost, and how to read a query plan.
est_min: 360
position: 2
---

# Indexing — B-trees & EXPLAIN ANALYZE

> **Step 6 · Databases, deep · Week 1**
> Concept: *query planning, index selection*

A query on a million rows can take 2 milliseconds or 2 seconds — same SQL, same
data. The difference is almost always an **index**. Understanding indexes is the
single highest-leverage database skill.

## Why this matters

Slow queries are the most common production performance problem, and indexing is
the most common fix. "It worked on 100 rows, it died on 10 million" is a rite of
passage. Learn to read the plan and add the right index, and you'll fix the bug
everyone else just throws hardware at.

## 1. Without an index: the sequential scan

To find `WHERE email = 'x@y.com'` with no index, the database must check **every
row** — a *sequential scan*, O(n). Fine for 100 rows, catastrophic for 10 million.

## 2. The B-tree index

The default index is a **B-tree** (balanced search tree, the on-disk cousin of the
BST from Step 2). It keeps the indexed column sorted in a shallow, wide tree, so
lookups are **O(log n)** — a handful of disk reads even for billions of rows.

```sql
CREATE INDEX idx_users_email ON users (email);
-- now WHERE email = '...' is an index scan, not a seq scan
```

B-trees serve equality (`=`), ranges (`<`, `>`, `BETWEEN`), sorting (`ORDER BY`),
and prefix matches (`LIKE 'abc%'`) on the indexed column.

## 3. Indexes are not free

Every index you add:
- **Speeds up reads** that can use it.
- **Slows down writes** — every INSERT/UPDATE/DELETE must also update the index.
- **Costs storage** — the index is a second copy of the column(s), sorted.

So you don't index everything. You index the columns you **filter, join, or sort
by** in your hot queries — and skip the rest.

## 4. Composite indexes and column order

An index on multiple columns is a **composite** index, and order matters — it's
sorted by the first column, then the second:

```sql
CREATE INDEX idx_bm_user_created ON bookmarks (user_id, created_at);
```

This serves `WHERE user_id = $1` and `WHERE user_id = $1 ORDER BY created_at`, but
**not** `WHERE created_at = $1` alone (the leading column is missing). Rule of
thumb: equality columns first, then the range/sort column. (This is the
"leftmost prefix" rule.)

## 5. Reading the plan: EXPLAIN ANALYZE

Don't guess — ask the database what it's doing. `EXPLAIN` shows the planned
strategy; `EXPLAIN ANALYZE` actually runs it and reports real timings.

```sql
EXPLAIN ANALYZE SELECT * FROM bookmarks WHERE user_id = 42 ORDER BY created_at DESC;
```

What to look for:
- **`Seq Scan`** on a big table in a hot query → usually a missing index. ⚠️
- **`Index Scan` / `Index Only Scan`** → the index is being used. ✅
- **`rows=`** estimated vs actual far apart → stale statistics (`ANALYZE` the table).
- **`Sort`** that's expensive → maybe a composite index can provide the order for free.

A **covering index** (one that includes every column the query needs) enables an
*Index Only Scan* — the DB never even touches the table. The fastest read is one
that reads only the index.

## 6. When indexes don't help

- Very small tables (a seq scan is already fast).
- Low-selectivity columns (`WHERE active = true` where half the rows match — the
  index isn't worth it).
- Functions on the column (`WHERE lower(email) = ...`) unless you make an
  *expression index* (`CREATE INDEX ON users (lower(email))`).

## Do it yourself (≈ 6 hrs)

1. Read [**Use The Index, Luke!**](https://use-the-index-luke.com/) — the anatomy of an index, the where clause, and the leftmost-prefix sections.
2. Watch the indexing lecture from [**CMU 15-445**](https://15445.courses.cs.cmu.edu/).
3. Create a table, insert ~1M rows (`generate_series`), and `EXPLAIN ANALYZE` a
   filtered query. Note the seq-scan time.
4. Add the index, run it again, and measure the speedup. See the plan switch from
   `Seq Scan` to `Index Scan`.

## Check yourself

- Why is a lookup with a B-tree index O(log n) instead of O(n)?
- Name the three costs of adding an index.
- For `WHERE user_id=$1 ORDER BY created_at`, what composite index helps, and why
  does column order matter?
- What does a `Seq Scan` in an `EXPLAIN` plan on a large hot table usually mean?
- What is a covering index / Index Only Scan, and why is it the fastest read?

Next: **concurrency control** — what happens when two transactions hit the same row.

## Common interview gotchas

- **"A composite index on `(a, b)` helps any query touching a or b."** Only the **leftmost prefix** is usable: it serves `a` and `a, b`, but a query filtering on `b` alone still seq-scans. Column order is the whole game.
- **"More indexes = faster database."** Each index taxes every INSERT/UPDATE/DELETE and consumes storage; too many can make a write-heavy table slower overall. Index for the hot read paths, not reflexively.
- **"`WHERE lower(email)=$1` / `WHERE col + 0 = $1` / `LIKE '%abc'` can use the index on the column."** None of these are **sargable** — wrapping the column in a function or leading with a wildcard defeats the B-tree. You need an expression index, or to rewrite the predicate to leave the column bare.
- **"Index Scan and Index Only Scan are the same thing."** Index Only Scan reads *only* the index (a **covering** index includes every column the query needs) and skips the heap entirely — much faster. A plain Index Scan still hops to the heap for the missing columns.
- **"Trust the EXPLAIN row estimates."** Estimates come from `pg_statistic` and go stale after bulk loads; compare **actual vs estimated rows** in `EXPLAIN ANALYZE`. A wild mismatch means run `ANALYZE` — the planner is choosing a bad plan on bad numbers.
