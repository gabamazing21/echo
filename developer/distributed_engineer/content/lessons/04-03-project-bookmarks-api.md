---
slug: project-bookmarks-api
step: 4
title: "Project: a bookmarks REST API with Echo"
summary: Build a full CRUD JSON API with Echo over an in-memory store — your first real backend service.
est_min: 480
position: 3
---

# Project: a bookmarks REST API with Echo

> **Step 4 · Build real backend APIs · Week 1 · Project**
> Concept: *handlers, JSON binding, error responses*

Time to build the thing. A **bookmarks API**: create, read, list, update, delete.
In-memory for now — next lesson you'll swap the store for Postgres without touching
the handlers. That clean seam is the whole point.

## What you'll build

```
POST   /bookmarks        -> 201 + the created bookmark
GET    /bookmarks        -> 200 + list
GET    /bookmarks/:id    -> 200 or 404
PUT    /bookmarks/:id    -> 200 or 404
DELETE /bookmarks/:id    -> 204 or 404
```

## Why this matters

This is the shape of nearly every backend feature you'll ever write: validate input
→ touch a store → return the right status + JSON. Get this loop clean and tested and
you can build any CRUD service. It's also the exact skeleton you'll extend with a
database and auth over the next lessons.

## 1. The model and the store interface

Define the data, and an *interface* for storage so the DB swap later is painless:

```go
package main

import "errors"

type Bookmark struct {
	ID    int64  `json:"id"`
	URL   string `json:"url"`
	Title string `json:"title"`
}

var ErrNotFound = errors.New("bookmark not found")

// Store is what the handlers depend on — not a concrete DB.
type Store interface {
	Create(b *Bookmark) (*Bookmark, error)
	Get(id int64) (*Bookmark, error)
	List() ([]*Bookmark, error)
	Update(id int64, b *Bookmark) (*Bookmark, error)
	Delete(id int64) error
}
```

## 2. An in-memory implementation

A map guarded by a mutex (you'll appreciate the mutex after Step 5):

```go
import "sync"

type MemStore struct {
	mu     sync.Mutex
	items  map[int64]*Bookmark
	nextID int64
}

func NewMemStore() *MemStore {
	return &MemStore{items: map[int64]*Bookmark{}, nextID: 1}
}

func (s *MemStore) Create(b *Bookmark) (*Bookmark, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b.ID = s.nextID
	s.nextID++
	s.items[b.ID] = b
	return b, nil
}

// YOUR TURN: implement Get, List, Update, Delete.
// Get/Update/Delete must return ErrNotFound when the id is missing.
```

## 3. Handlers that bind, validate, and return honest status codes

```go
type API struct{ store Store }

func (a *API) create(c echo.Context) error {
	var in Bookmark
	if err := c.Bind(&in); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid JSON body")
	}
	if in.URL == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "url is required")
	}
	created, err := a.store.Create(&in)
	if err != nil {
		return err // becomes a 500 via the error handler
	}
	return c.JSON(http.StatusCreated, created)
}

func (a *API) get(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "id must be a number")
	}
	b, err := a.store.Get(id)
	if errors.Is(err, ErrNotFound) {
		return echo.NewHTTPError(http.StatusNotFound, "not found")
	}
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, b)
}

// YOUR TURN: write list (200 + slice), update (PUT, 404 if missing),
// delete (204 on success, 404 if missing).
```

Notice the pattern in every handler: **parse → validate → call store → map the
result (incl. ErrNotFound → 404) → write JSON**. The `errors.Is(err, ErrNotFound)`
check is exactly the wrapping idiom from Step 1.

## 4. Wire it up

```go
func main() {
	e := echo.New()
	e.Use(middleware.Recover(), middleware.Logger())

	api := &API{store: NewMemStore()}
	e.POST("/bookmarks", api.create)
	e.GET("/bookmarks", api.list)
	e.GET("/bookmarks/:id", api.get)
	e.PUT("/bookmarks/:id", api.update)
	e.DELETE("/bookmarks/:id", api.delete)

	e.Logger.Fatal(e.Start(":8080"))
}
```

Test it with `curl`:

```bash
curl -s -X POST localhost:8080/bookmarks -d '{"url":"https://go.dev","title":"Go"}' -H 'content-type: application/json'
curl -s localhost:8080/bookmarks/1
curl -s -X DELETE localhost:8080/bookmarks/1 -i   # expect 204
```

## Stretch goals

- Add a validator (`go-playground/validator`) and tag `URL` with `validate:"required,url"`.
- Return a consistent error body (`{"error": "..."}`) via a custom `e.HTTPErrorHandler`.
- Add `GET /bookmarks?q=go` filtering by title substring.
- Write a handler test using `httptest` and Echo's test recorder.

## Done when

- [ ] All five endpoints work end-to-end with `curl`.
- [ ] Missing ids return **404**, bad bodies return **400**, successful create returns **201**, delete returns **204**.
- [ ] Handlers depend on the `Store` *interface*, not the concrete `MemStore`.
- [ ] You can explain why that interface seam makes the Postgres swap trivial.

FREE source: the [**Echo guide**](https://echo.labstack.com/docs) (Binding, Routing, Error Handling).

Next: **PostgreSQL from Go** — then you'll swap `MemStore` for a real database.

## Common interview gotchas

- **"Offset pagination is fine."** `OFFSET 10000` makes Postgres scan and discard 10k rows (O(offset)), and rows inserted/deleted between pages cause **skipped or duplicated** results. Use **keyset/cursor** pagination (`WHERE id > $last ORDER BY id LIMIT n`).
- **"Handlers can depend on `MemStore` directly — I'll refactor later."** Depending on the concrete type defeats the whole interface seam; the Postgres swap stops being one line. Depend on the `Store` *interface*.
- **"I'll sort by whatever column the client passes."** Interpolating `ORDER BY ` + user input is injection. Whitelist sortable columns to a fixed map — `$N` placeholders can't parameterize identifiers.
- **"The in-memory map is just a demo, no locking needed."** Concurrent requests hit it from multiple goroutines; an unguarded `map` is a data race (and Go will fatal-crash on concurrent map writes). Guard with a `sync.Mutex` and prove it with `go test -race`.
- **"Returning the `*Bookmark` straight from the map is fine."** Callers can now mutate your stored object without the lock. Return a copy, or you've leaked the protected state past the mutex.
