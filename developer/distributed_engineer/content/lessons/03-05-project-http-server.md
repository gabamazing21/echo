---
slug: project-http-server
step: 3
title: "Project: a minimal HTTP server from scratch"
summary: Build a small HTTP server with net/http — handlers, ServeMux routing, query/body parsing, JSON, and the request lifecycle.
est_min: 300
position: 5
---

# Project: a minimal HTTP server from scratch

> **Step 3 · How computers & networks work · Week 2 · Project**
> Concept: *handlers, routing, the request lifecycle*

You've read about TCP, HTTP, and the request/response model. Now you *build the
server* that speaks it. No framework, no magic — just Go's standard library
`net/http`. By the end you'll have a running web server you can hit with `curl`,
and — more importantly — you'll understand exactly what happens between the moment
a request arrives and the moment a response leaves.

## What you'll build

A tiny JSON API for an in-memory list of "notes". It will:

- listen on a TCP port and serve HTTP,
- route requests by path **and** method with `http.ServeMux`,
- read a **query parameter** and a **JSON request body**,
- write **JSON responses** with correct status codes and headers.

By the end you'll run:

```bash
go run .
# server listening on :8080

curl -s localhost:8080/health
# {"status":"ok"}

curl -s 'localhost:8080/notes?contains=go'
# [{"id":1,"text":"learn go"}]

curl -s -X POST localhost:8080/notes -d '{"text":"ship it"}'
# {"id":2,"text":"ship it"}
```

## Why this matters

Every web framework you'll ever touch — Express, Flask, Rails, and **Echo, which
you'll meet in Step 4** — is a convenience layer sitting on top of exactly what
you're about to write by hand. A "handler" that takes a request and writes a
response; a "router" that maps paths to handlers; "middleware" that wraps them.
Learn the bare metal once and every framework afterward reads like sugar.

It matters for a second, closer reason: **the Consensus app you're using right now
is a Go HTTP server.** When you submit a challenge, an HTTP request travels from
your browser to a handler, which reads your code, runs the checker, and writes a
JSON response back. The thing grading you *is the thing you're building today.*

## 1. Set up the module

```bash
mkdir noteserver && cd noteserver
go mod init example.com/noteserver
```

One file is enough to start: `main.go`. Everything we need ships with Go 1.21 —
no `go get`, no dependencies.

## 2. The smallest server that runs

Three pieces make an HTTP server in Go: a **handler** (something that responds), a
**mux** (router) that decides which handler runs, and `ListenAndServe` (the loop
that accepts connections).

```go
package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok\n"))
	})

	log.Println("server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
```

Run `go run .`, then in another terminal `curl localhost:8080/health`. You have a
web server. Two types are doing all the work and you'll see them everywhere:

- **`http.ResponseWriter`** (`w`) — you *write the response into it*: status,
  headers, body.
- **`*http.Request`** (`r`) — everything *about the incoming request*: method,
  URL, headers, body.

That function signature `func(w http.ResponseWriter, r *http.Request)` is the
single most important shape in Go web programming. Burn it in.

## 3. The request lifecycle, concretely

When that `curl` ran, here's the path the bytes took:

1. `curl` opens a **TCP** connection to port 8080 and sends an HTTP request line
   (`GET /health HTTP/1.1`) plus headers.
2. `ListenAndServe` accepts the connection and parses it into an `*http.Request`.
3. The **mux** looks at `r.URL.Path` (`/health`) and picks the matching handler.
4. Your handler writes to `w`. **The first call to `w.Write` flushes the status
   line and headers** — after that you can't change them.
5. Go closes (or keeps alive) the connection; `curl` prints what came back.

> **Worth internalising:** you never call your handler — *the server does*, once
> per request, often on a **separate goroutine** per connection. That's why Go
> servers handle thousands of concurrent requests for free, and why any shared
> state your handlers touch must be safe for concurrent use (more on that below).

## 4. Write JSON, not strings

Real APIs speak JSON. Define your data as a struct with field tags, set the
`Content-Type` header, and encode straight to the `ResponseWriter`.

```go
package main

import (
	"encoding/json"
	"net/http"
)

type Note struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}

// writeJSON sets the header, status, and encodes v as the body.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
```

Two ordering rules that trip up everyone once:

- `w.Header().Set(...)` must come **before** `w.WriteHeader(...)`. Headers are
  sent with the status line; set them too late and they're silently dropped.
- `w.WriteHeader(status)` may be called **once**. If you never call it, the first
  `Write` defaults the status to `200 OK`.

> **Your turn:** rewrite the `/health` handler from step 2 to use `writeJSON` so
> it returns `{"status":"ok"}` with `Content-Type: application/json`. Confirm the
> header with `curl -i localhost:8080/health` (the `-i` prints response headers).

## 5. State, and routing by method

Our notes live in memory. Because handlers run on concurrent goroutines, guard
the shared slice with a `sync.Mutex` — skip this and you have a data race the
moment two requests arrive at once.

```go
package main

import (
	"sync"
)

type store struct {
	mu     sync.Mutex
	notes  []Note
	nextID int
}
```

Now the interesting handler. The same path `/notes` does different things for
`GET` vs `POST`, so we branch on `r.Method`.

```go
func (s *store) handleNotes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.list(w, r)
	case http.MethodPost:
		s.create(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed,
			map[string]string{"error": "method not allowed"})
	}
}
```

## 6. Read a query parameter (GET)

`r.URL.Query()` parses `?contains=go` into a map-like value. `Get` returns `""`
if the key is absent, which is a clean "no filter" default.

```go
func (s *store) list(w http.ResponseWriter, r *http.Request) {
	contains := r.URL.Query().Get("contains")

	s.mu.Lock()
	defer s.mu.Unlock()

	out := []Note{}
	for _, n := range s.notes {
		if contains == "" || strings.Contains(n.Text, contains) {
			out = append(out, n)
		}
	}
	writeJSON(w, http.StatusOK, out)
}
```

> **Your turn:** notice `out` starts as `[]Note{}`, not `var out []Note`. A nil
> slice encodes to JSON `null`; an empty slice encodes to `[]`. Try both and see
> which one a client would rather parse. Small detail, real bug.

## 7. Read a JSON body (POST)

For `POST`, the payload is in `r.Body` (an `io.ReadCloser`). Decode it into a
struct. **Always validate** — never trust the body. Return `400` on bad input.

```go
func (s *store) create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest,
			map[string]string{"error": "invalid JSON"})
		return
	}
	if in.Text == "" {
		writeJSON(w, http.StatusBadRequest,
			map[string]string{"error": "text is required"})
		return
	}

	s.mu.Lock()
	s.nextID++
	n := Note{ID: s.nextID, Text: in.Text}
	s.notes = append(s.notes, n)
	s.mu.Unlock()

	writeJSON(w, http.StatusCreated, n) // 201, with the created note
}
```

## 8. Wire it together

Register the handlers on a mux, build the store, and serve. You still owe two
imports (`strings` for step 6; the rest you've already used) — let the compiler
tell you what's missing.

```go
func main() {
	s := &store{
		notes:  []Note{{ID: 1, Text: "learn go"}},
		nextID: 1,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", health) // your step-4 handler
	mux.HandleFunc("/notes", s.handleNotes)

	log.Println("server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
```

> **Your turn (the real work):** assemble the full program from the snippets
> above into `main.go`, fix the imports, and get all three `curl` examples from
> *What you'll build* to pass. Then add a test: `net/http/httptest` lets you call
> a handler with a fake request and inspect the response — no real network. Write
> one test that POSTs a note and asserts the status is `201` and the body has the
> right `id`.

## Stretch goals

Pick at least one — this is where it stops being a tutorial and starts being
*yours*.

- **Logging middleware.** Middleware is a function that *wraps* a handler:
  `func(http.Handler) http.Handler`. Write one that logs method, path, and
  duration of every request, then wrap your whole mux with it. This is the exact
  pattern Echo's middleware uses.
- **Graceful shutdown.** Replace `http.ListenAndServe` with an `*http.Server` and
  call `srv.Shutdown(ctx)` when you catch `SIGINT` (`os/signal` + a `context`
  with timeout). In-flight requests finish; new ones are refused. This is what
  real deployments do on every redeploy.
- **A tiny router.** `http.ServeMux` doesn't do path params like `/notes/42`.
  Write a small wrapper that matches `/notes/{id}` and extracts the id, so you can
  support `GET`/`DELETE` on a single note. (Go 1.22 adds this to `ServeMux`
  natively — building it yourself shows you what that buys you.)
- **Per-handler timeouts.** Set `ReadTimeout` and `WriteTimeout` on the server and
  observe what a slow client does without them.

## Free resources

- [**`net/http` package docs**](https://pkg.go.dev/net/http) — the canonical
  reference. Read the package overview, then `ServeMux`, `HandlerFunc`, and the
  `Request` / `ResponseWriter` types. Everything today is in here.
- [**`net/http/httptest`**](https://pkg.go.dev/net/http/httptest) — for the test
  in step 8.
- [**`encoding/json`**](https://pkg.go.dev/encoding/json) — `Encoder`/`Decoder`
  and struct field tags.

## Done when

- [ ] `go run .` starts a server and `curl localhost:8080/health` returns JSON.
- [ ] `GET /notes?contains=...` filters by query parameter and returns a JSON
      array (`[]`, never `null`, when empty).
- [ ] `POST /notes` decodes a JSON body, validates it, and returns `201` with the
      created note — and `400` on bad or empty input.
- [ ] An unsupported method on `/notes` returns `405`.
- [ ] You wrote at least one `httptest`-based test that passes.
- [ ] Shared state is guarded — no `go run -race .` warnings under concurrent
      requests.
- [ ] You attempted at least one stretch goal.

When every box is checked, commit this project on your **Roadmap**. You now
understand a web server from the socket up — which means in Step 4, when Echo
hands you `c.JSON(200, note)`, you'll know exactly what it's doing for you. Next
up: **Step 4, building real APIs with Echo.**

## Common interview gotchas

- **Go 1.22 `ServeMux` does method + path params natively.** Register `GET /notes/{id}` and read `r.PathValue("id")` — a wrong method gets an automatic **405** (with `Allow`), so you don't write that branch. Routes match by **specificity**, not registration order. Before 1.22 you'd register the `/notes/` prefix and parse `r.URL.Path` by hand; know what the feature replaced.
- **Middleware is `func(http.Handler) http.Handler`, and to read the status you must wrap the `ResponseWriter`.** The writer isn't readable, so embed it in a `statusRecorder` that captures `WriteHeader`, defaulting to 200 (a handler that never calls `WriteHeader` still sends 200). Middlewares **compose**: `A(B(C(mux)))`.
- **Graceful shutdown needs an explicit `*http.Server` + `srv.Shutdown(ctx)`.** `ListenAndServe` returns **`http.ErrServerClosed`** on a clean stop — treat it as normal, not an error. The context is the *drain deadline*; past it, remaining connections are dropped.
- **You can't change status or headers after the first `Write`.** The first write flushes the status line and headers; later `WriteHeader`/`Header().Set` are ignored ("superfluous WriteHeader" in the log). Set headers and status **before** the body.
- **Every request runs on its own goroutine.** Shared state handlers touch (in-memory maps/slices, counters) is accessed concurrently and must be guarded with a mutex/atomics — verify with `go test -race`.
