---
slug: db-postgres-pgx
step: 4
title: PostgreSQL from Go — pgx & migrations
summary: Connection pools, parameterized queries, row scanning and golang-migrate — the exact data stack this app runs on.
est_min: 480
position: 4
---

# PostgreSQL from Go — pgx & migrations

> **Step 4 · Build real backend APIs · Week 2**
> Concept: *connection pools, prepared statements*

An in-memory map forgets everything on restart. Real services persist to a
database. For Go + Postgres, the modern choice is **pgx**.

> **This is the Consensus stack.** This app uses `pgx/v5` + `pgxpool` and
> `golang-migrate` with embedded migrations. Read `internal/db/db.go` and the
> `migrations/` folder alongside this lesson.

## Why this matters

Persistence is where correctness gets real: connection limits, SQL injection,
transactions, migrations. Every backend role expects fluency here, and Step 6 goes
deep on the database itself. This lesson is the Go-side plumbing.

## 1. database/sql vs pgx

Go ships `database/sql` (a generic interface) but you still need a driver. **pgx**
is both a driver *and* a richer Postgres-native API. Use `pgxpool` for servers:

```go
import "github.com/jackc/pgx/v5/pgxpool"

pool, err := pgxpool.New(ctx, "postgres://user@localhost:5432/mydb?sslmode=disable")
if err != nil { /* handle */ }
defer pool.Close()
```

## 2. Connection pools — why you don't open one connection

Opening a TCP+auth connection per query is slow, and Postgres caps total
connections. A **pool** keeps a set of live connections and hands them out:

```go
cfg, _ := pgxpool.ParseConfig(url)
cfg.MaxConns = 10           // bound concurrency to the DB
pool, _ := pgxpool.NewWithConfig(ctx, cfg)
```

Every query borrows a connection and returns it. Size `MaxConns` to your workload,
not infinity — the database is the shared bottleneck.

## 3. Querying: always parameterized

**Never** build SQL with string concatenation. Use `$1, $2` placeholders — the
driver sends values separately, so user input can never become SQL. This is how
you defeat SQL injection.

```go
// SELECT one row
var b Bookmark
err := pool.QueryRow(ctx,
	`SELECT id, url, title FROM bookmarks WHERE id=$1`, id).
	Scan(&b.ID, &b.URL, &b.Title)
if errors.Is(err, pgx.ErrNoRows) {
	return nil, ErrNotFound      // map to your domain error -> 404
}

// INSERT and get the generated id back
err = pool.QueryRow(ctx,
	`INSERT INTO bookmarks (url, title) VALUES ($1,$2) RETURNING id`,
	b.URL, b.Title).Scan(&b.ID)
```

❌ Never do this:

```go
// SQL INJECTION WAITING TO HAPPEN — do not concatenate input
pool.Query(ctx, "SELECT * FROM bookmarks WHERE title = '"+title+"'")
```

## 4. Querying many rows

```go
rows, err := pool.Query(ctx, `SELECT id, url, title FROM bookmarks ORDER BY id`)
if err != nil { return nil, err }
defer rows.Close()

var out []*Bookmark
for rows.Next() {
	var b Bookmark
	if err := rows.Scan(&b.ID, &b.URL, &b.Title); err != nil {
		return nil, err
	}
	out = append(out, &b)
}
return out, rows.Err()   // <- ALWAYS check rows.Err() after the loop
```

`Exec` is for writes where you don't need rows back (UPDATE/DELETE); it returns a
tag with `RowsAffected()`.

## 5. Migrations with golang-migrate

Your schema must evolve in versioned, repeatable steps — not hand-run SQL. Each
change is a pair of files:

```
migrations/
  0001_create_bookmarks.up.sql     # CREATE TABLE bookmarks (...)
  0001_create_bookmarks.down.sql   # DROP TABLE bookmarks
```

```sql
-- 0001_create_bookmarks.up.sql
CREATE TABLE bookmarks (
    id    BIGSERIAL PRIMARY KEY,
    url   TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT ''
);
```

Apply them on boot. This app embeds them in the binary and runs `Up()` at startup —
already-applied migrations are skipped (it's idempotent). That's why deploying is
just shipping one binary.

## 6. Prepared statements & context

pgx prepares and caches statements for you. The important habit is **passing
`ctx`** (from `c.Request().Context()`) into every call — so a cancelled request or
a deadline actually stops the query instead of holding a pooled connection hostage.

## Do it yourself (≈ 8 hrs)

1. Read the [**pgx docs**](https://pkg.go.dev/github.com/jackc/pgx/v5) (pgxpool, QueryRow/Query/Exec) and skim [**golang-migrate**](https://github.com/golang-migrate/migrate).
2. If SQL is rusty, work through [**PostgreSQL Tutorial**](https://www.postgresqltutorial.com/) (SELECT/INSERT/UPDATE/DELETE, JOINs).
3. Create a local db, write a `0001` up/down migration for a `bookmarks` table, apply it.
4. Write a tiny program that inserts two bookmarks (parameterized) and lists them.
5. Read `internal/db/db.go` and a migration pair in the Consensus repo and match each line to this lesson.

## Check yourself

- Why use a connection *pool* instead of opening a connection per query?
- How do parameterized queries (`$1`) prevent SQL injection?
- What does `pgx.ErrNoRows` mean and what HTTP status should it become?
- Why must every migration have a matching `.down.sql`?
- What breaks if you forget to pass `ctx` into your queries?

Next: **the project** — swap your bookmarks store from the in-memory map to Postgres.

## Common interview gotchas

- **"Bigger pool = more throughput."** Past a point a huge `MaxConns` *slows* you down — Postgres context-switches and contends on locks. A small pool (often single-digit per instance) usually beats a large one; the DB is the shared bottleneck, not your app.
- **"Escaping the input prevents injection."** Manual escaping is fragile and you'll miss a case. **Parameterized `$1`** sends values out-of-band from the SQL text, so input *can never* be parsed as SQL — that's the only real defense.
- **"I'll reuse `ctx` from app startup for queries."** Then a cancelled HTTP request keeps querying and holds a pooled connection hostage. Thread the *request's* ctx into every call so cancellation/deadlines actually stop the query.
- **"`ErrNoRows` is an error to log and 500."** It's the normal "not found" signal — map `pgx.ErrNoRows` to your domain `ErrNotFound` → **404**. A real DB failure is the 500.
- **"`CREATE INDEX` is safe to run anytime."** Plain `CREATE INDEX` takes a write lock and blocks the table; on a live system use `CREATE INDEX CONCURRENTLY` (slower, no exclusive lock — and can't run inside a transaction).
