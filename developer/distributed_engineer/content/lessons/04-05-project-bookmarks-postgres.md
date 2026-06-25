---
slug: project-bookmarks-postgres
step: 4
title: "Project: wire bookmarks to Postgres + migrations"
summary: Swap the in-memory store for a real Postgres-backed repository with parameterized queries and migrations.
est_min: 480
position: 5
---

# Project: wire bookmarks to Postgres + migrations

> **Step 4 · Build real backend APIs · Week 2 · Project**
> Concept: *persistence, SQL injection safety*

Your bookmarks API works but forgets everything on restart. Now you'll give it a
real memory: Postgres. Because the handlers depend on the `Store` *interface*, you
write one new implementation and change *nothing* in the handlers.

## What you'll build

A `PGStore` that satisfies the same `Store` interface, backed by a `bookmarks`
table created via a migration. Same endpoints, same responses — now durable.

## Why this matters

This is the single most common backend task: persist a resource safely. You'll
practice migrations, parameterized queries (injection-safe), and the
repository pattern that keeps SQL out of your handlers — the structure real teams
(and your Lokatalent codebase) use.

## 1. The migration

```sql
-- migrations/0001_create_bookmarks.up.sql
CREATE TABLE bookmarks (
    id         BIGSERIAL PRIMARY KEY,
    url        TEXT NOT NULL,
    title      TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

```sql
-- migrations/0001_create_bookmarks.down.sql
DROP TABLE bookmarks;
```

Apply it (CLI for now; embed it later like the Consensus app does):

```bash
migrate -path migrations -database "$DATABASE_URL" up
```

## 2. The Postgres store

```go
type PGStore struct{ pool *pgxpool.Pool }

func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

func (s *PGStore) Create(ctx context.Context, b *Bookmark) (*Bookmark, error) {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO bookmarks (url, title) VALUES ($1,$2) RETURNING id, created_at`,
		b.URL, b.Title).Scan(&b.ID, &b.CreatedAt)
	return b, err
}

func (s *PGStore) Get(ctx context.Context, id int64) (*Bookmark, error) {
	var b Bookmark
	err := s.pool.QueryRow(ctx,
		`SELECT id, url, title, created_at FROM bookmarks WHERE id=$1`, id).
		Scan(&b.ID, &b.URL, &b.Title, &b.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &b, err
}

// YOUR TURN: List (ORDER BY id), Update (UPDATE ... WHERE id=$1, map no-row to
// ErrNotFound via RowsAffected or RETURNING), Delete (DELETE ... ; 404 if 0 rows).
```

> **Note the signature change.** Your store methods now take `ctx context.Context`
> as the first argument. Update the `Store` interface and the in-memory version to
> match, and pass `c.Request().Context()` from each handler. This is the habit from
> the context lesson coming up in Step 5 — start it now.

## 3. Detecting "not found" on UPDATE/DELETE

A DELETE that matched no rows isn't an error — it's a 404. Use the command tag:

```go
func (s *PGStore) Delete(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM bookmarks WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
```

## 4. Swap it in — one line

```go
pool, _ := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
defer pool.Close()

api := &API{store: NewPGStore(pool)}   // was: NewMemStore()
```

Handlers untouched. That's the payoff of coding to an interface.

## Stretch goals

- Add `created_at` to the JSON and sort the list newest-first.
- Add an index on `url` (preview of Step 6) and a `UNIQUE` constraint; map the
  duplicate-key error to a `409 Conflict`.
- Write an integration test that spins the store against a throwaway database.
- Embed the migrations in the binary and run `Up()` on boot (study `internal/db/db.go`).

## Done when

- [ ] All five endpoints work against Postgres and survive a server restart.
- [ ] Every query is parameterized — no string concatenation anywhere.
- [ ] UPDATE/DELETE on a missing id returns **404**.
- [ ] Switching `MemStore` ↔ `PGStore` is a one-line change.

FREE sources: [**pgx docs**](https://pkg.go.dev/github.com/jackc/pgx/v5) and [**golang-migrate**](https://github.com/golang-migrate/migrate).

Next: **authentication** — who is making the request, and what are they allowed to do?

## Common interview gotchas

- **"`ON CONFLICT` will just handle duplicates."** It only fires against a **unique index/constraint** on the conflict target — without one there's nothing to conflict on and the UPSERT silently inserts a duplicate. The constraint is the contract.
- **"My UPDATE/DELETE worked because there was no error."** Updating a non-existent id returns *no error* and zero rows. Check `tag.RowsAffected() == 0` (or `RETURNING` + `ErrNoRows`) → **404**; otherwise you 200 on a no-op.
- **"Loop and INSERT each row."** N round-trips for a bulk load. Use `COPY` (pgx's `CopyFrom`) or a single multi-row INSERT — orders of magnitude faster.
- **"Soft delete is just an `is_deleted` column."** Now *every* query must remember `WHERE deleted_at IS NULL`, your `UNIQUE(email)` blocks re-signup after deletion, and indexes bloat with dead rows. Use a **partial index** (`WHERE deleted_at IS NULL`) and partial uniques — and accept the per-query tax.
- **"`RETURNING id` and `LastInsertId()` are the same."** Postgres has no auto `LastInsertId`; you must use `RETURNING` to get the generated id back in the same round-trip.
