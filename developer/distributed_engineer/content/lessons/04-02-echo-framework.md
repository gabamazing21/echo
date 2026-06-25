---
slug: api-echo-framework
step: 4
title: The Echo framework — routing, middleware, binding
summary: Echo's instance, routing, Context, middleware chains and request binding — the toolkit you'll build every API with.
est_min: 420
position: 2
---

# The Echo framework — routing, middleware, binding

> **Step 4 · Build real backend APIs · Week 1**
> Concept: *middleware chains, context*

`net/http` (Step 3) is enough to build a server, but you'd hand-roll routing,
JSON binding, and middleware every time. **Echo** is a thin, fast web framework
that gives you those without hiding HTTP from you.

> **The app you're reading this in is built with Echo.** Open `cmd/server/main.go`
> and `internal/handlers/handlers.go` in the Consensus repo — everything below is
> in there, in production.

## Why this matters

Echo (or a peer like Gin/chi) is what you reach for to build real services. Your
Lokatalent backend is Echo. Learning it well means you can ship a clean, tested
HTTP API quickly — the day-to-day craft of backend engineering.

## 1. The Echo instance and a first route

```go
package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	e.Logger.Fatal(e.Start(":8080"))
}
```

A handler is `func(c echo.Context) error`. You return an `error` — Echo turns it
into an HTTP response. Returning `nil` after writing means "done".

## 2. Routing and path/query params

```go
e.GET("/bookmarks", listBookmarks)        // collection
e.POST("/bookmarks", createBookmark)
e.GET("/bookmarks/:id", getBookmark)      // :id is a path param
e.DELETE("/bookmarks/:id", deleteBookmark)

func getBookmark(c echo.Context) error {
	id := c.Param("id")          // path param  -> "42"
	q := c.QueryParam("fields")  // query param -> /bookmarks/42?fields=url
	// ...
}
```

Group related routes (and share middleware) with `e.Group`:

```go
api := e.Group("/api")
api.GET("/bookmarks", listBookmarks)   // -> /api/bookmarks
```

(The Consensus app uses exactly this — its mutation endpoints live under an
`/api` group.)

## 3. The Context

`echo.Context` (the `c`) is the per-request object. It wraps the request and
response and gives you helpers:

```go
c.Request()                      // the underlying *http.Request
c.Request().Context()            // the Go context.Context — pass this down to your DB calls
c.Param("id") / c.QueryParam(..) // params
c.Bind(&dst)                     // decode the JSON body into a struct
c.JSON(status, v)                // write a JSON response
c.String(status, s) / c.NoContent(status)
```

Always pass `c.Request().Context()` into your store/DB calls so cancellation and
deadlines propagate (you'll feel why in Step 5).

## 4. Middleware chains

Middleware wraps every handler — for logging, auth, recovery, CORS, rate limiting.
It runs in order, like an onion around the handler:

```go
e.Use(middleware.Recover())   // turn panics into 500s instead of crashing
e.Use(middleware.Logger())    // log every request

// a custom middleware
func RequireAPIKey(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if c.Request().Header.Get("X-API-Key") == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "missing api key")
		}
		return next(c) // call the next link in the chain
	}
}

api := e.Group("/api", RequireAPIKey) // applies to the whole group
```

This chain idea is how cross-cutting concerns stay out of your handlers. Auth
(lesson 6) is just middleware.

## 5. Binding and validation

`c.Bind` decodes the request body into a struct using its `json` tags:

```go
type CreateBookmark struct {
	URL   string `json:"url"   validate:"required,url"`
	Title string `json:"title"`
}

func createBookmark(c echo.Context) error {
	var in CreateBookmark
	if err := c.Bind(&in); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}
	if in.URL == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "url is required")
	}
	// ... create it, then:
	return c.JSON(http.StatusCreated, in)
}
```

For real validation, wire a validator (e.g. `go-playground/validator`) into
`e.Validator` and call `c.Validate(&in)` — Echo's docs show the few lines. Validate
at the boundary so a bad body is a clean `400`, never a `500` deeper in.

## 6. Errors, centrally

Return `echo.NewHTTPError(status, msg)` and Echo renders a JSON error. You can set
a custom `e.HTTPErrorHandler` to standardize the error body shape across the whole
API (one place, every endpoint consistent).

## Do it yourself (≈ 7 hrs)

1. Read the [**Echo guide**](https://echo.labstack.com/docs): Routing, Context, Binding, Middleware.
2. Build a tiny Echo server with `/health` (200 JSON) and `/echo/:word` (returns the word).
3. Add `middleware.Logger()` and `middleware.Recover()`. Add a handler that `panic`s and confirm Recover turns it into a 500.
4. Write a custom middleware that rejects requests without an `X-Demo` header (401).
5. Read the Consensus repo's `internal/handlers/handlers.go` and map each thing back to this lesson.
6. (Optional, recommended) Start [**Let's Go**](https://lets-go.alexedwards.net/) — the gold standard for production Go web apps.

## Check yourself

- What's the signature of an Echo handler, and what does returning an `error` do?
- How does a path param differ from a query param, and how do you read each?
- What problem does middleware solve, and in what order does a chain run?
- Why pass `c.Request().Context()` into your database calls?
- Where should input validation happen, and what status code does a bad body deserve?

Next: **your first real API** — a bookmarks CRUD service in Echo.

## Common interview gotchas

- **"`middleware.Recover()` catches all panics."** Only panics on the request goroutine. A panic in a goroutine *you* spawn (`go func(){...}()`) crashes the whole process — recover inside that goroutine yourself.
- **"Middleware order doesn't matter."** It's an onion: the order of `e.Use` calls is the order of execution. Put `Recover` before `Logger` or a panic skips your logging; put auth before the handler or you've leaked.
- **"Just use `context.Background()` in my DB call."** Then cancellation/deadlines never propagate — a client that hung up still ties up a pooled connection. Pass `c.Request().Context()` down the whole stack.
- **"`c.Bind` will populate my struct fields."** Only **exported** fields with matching tags. An unexported field, or a typo'd `json` tag, silently stays zero — and Bind won't error.
- **"Validation failures are 500s from deep in the code."** Validate at the boundary so a bad body is a clean **400**. Letting a missing field blow up three layers down (nil deref) is both a 500 and a security smell.
