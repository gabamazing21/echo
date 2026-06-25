---
slug: net-http-deeper
step: 3
title: HTTP, deeper — methods, status, headers
summary: Go beyond "GET a page" — learn HTTP methods, status families, headers, statelessness and idempotency that make safe retries possible.
est_min: 240
position: 4
---

# HTTP, deeper — methods, status, headers

> **Step 3 · How computers & networks work · Week 2**
> Concept: *statelessness, idempotency*

You've sent a request and gotten a page back. Now we open HTTP up and look at the
parts that actually matter when you build *systems* on top of it: the **methods**
and what each one promises, the **status codes** a server uses to talk back, the
**headers** that carry everything-but-the-body, and two ideas — **statelessness**
and **idempotency** — that quietly govern whether your distributed system can
recover from failure. Don't skim this one; Step 4 (REST APIs) is built directly
on it.

## Why this matters

In a distributed system, **requests fail and get retried** — that's not an edge
case, it's the normal weather. A network blip, a timeout, a load balancer
reaping a connection: any of these can leave you unsure whether your request
*actually happened*. The single most important question becomes: *is it safe to
send it again?* HTTP answers that with **idempotency**. Get the methods and their
guarantees right and retries are trivially safe; get them wrong and a retry
double-charges a customer or creates two orders. This lesson is where that
intuition starts.

## 1. HTTP is a request/response protocol

Every exchange is one **request** (method + path + headers + optional body) and
one **response** (status code + headers + optional body). A raw request is just
text:

```
GET /nodes/3 HTTP/1.1
Host: api.example.com
Accept: application/json
```

And a response:

```
HTTP/1.1 200 OK
Content-Type: application/json
Content-Length: 27

{"id":3,"status":"online"}
```

That's the whole shape. Everything below is detail layered on these two
messages.

## 2. Methods and what they *mean*

The method (also called the *verb*) tells the server what you intend to do with
the resource at that path. The five you'll use constantly:

| Method | Intent | Has body? | Typical success |
|---|---|---|---|
| `GET` | Read a resource | no | `200 OK` |
| `POST` | Create / submit, server decides identity | yes | `201 Created` |
| `PUT` | Replace a resource at a known URL | yes | `200`/`204` |
| `PATCH` | Partially update a resource | yes | `200`/`204` |
| `DELETE` | Remove a resource | no | `204 No Content` |

These are **conventions with teeth**: proxies, caches, and clients all assume
they behave as described. Break the convention (e.g. a `GET` that deletes data)
and you'll fight the entire ecosystem.

## 3. Safety & idempotency — the two properties that matter most

Two independent properties describe a method's guarantees:

- **Safe** — the request *does not change server state*. It's read-only. `GET`
  is safe; you can issue it as many times as you like with no side effects.
- **Idempotent** — making the request **N times has the same effect as making
  it once**. The server may do work each time, but the end *state* is identical.

| Method | Safe | Idempotent | Why |
|---|---|---|---|
| `GET` | yes | yes | Pure read |
| `PUT` | no | **yes** | "Set resource to *this*" — repeat lands the same value |
| `DELETE` | no | **yes** | Delete-again is a no-op; resource is still gone |
| `PATCH` | no | usually not | Depends on the patch (e.g. "increment by 1" is not) |
| `POST` | no | **no** | "Create a new one" — repeat creates *another* one |

This is the heart of the lesson. A failed-and-retried `PUT /nodes/3` is **safe**
— worst case you set the same value twice. A failed-and-retried `POST /orders`
might create **two orders**. That's why robust APIs make creation idempotent with
an **idempotency key** (a client-generated ID the server uses to deduplicate),
turning an unsafe retry into a safe one. You'll see this pattern everywhere in
production systems.

## 4. Status codes: how the server talks back

The status code is a three-digit number whose **first digit names a family**:

| Family | Meaning | You'll see |
|---|---|---|
| `1xx` | Informational | `100 Continue`, `101 Switching Protocols` |
| `2xx` | Success | `200 OK`, `201 Created`, `204 No Content` |
| `3xx` | Redirection | `301 Moved Permanently`, `304 Not Modified` |
| `4xx` | **Client** error — *you* sent something wrong | `400`, `401`, `403`, `404`, `429` |
| `5xx` | **Server** error — the server failed | `500`, `502`, `503`, `504` |

The 4xx/5xx split carries real operational meaning. A **4xx** says *don't retry
unchanged — fix the request* (a `400 Bad Request` won't get better on its own). A
**5xx** says *the server had a problem*; these are often **transient**, and
combined with an idempotent method they're exactly what you retry — ideally with
**exponential backoff** so you don't stampede a struggling server. `429 Too Many
Requests` is the server explicitly asking you to slow down.

Key codes worth memorising: `200 OK`, `201 Created`, `204 No Content`,
`301`/`304` (redirect / not-modified), `400` (bad request), `401 Unauthorized`
(you're not authenticated), `403 Forbidden` (authenticated but not allowed),
`404 Not Found`, `429`, `500 Internal Server Error`, `503 Service Unavailable`.

## 5. Setting status and methods in Go

In `net/http`, you read the method off the request and write the status to the
response. A handler that only accepts `POST`:

```go
package main

import (
	"net/http"
)

func createNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// Tell the client which methods *are* allowed, then 405.
		w.Header().Set("Allow", "POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusCreated) // 201
	w.Write([]byte(`{"id":3,"status":"online"}`))
}

func main() {
	http.HandleFunc("/nodes", createNode)
	http.ListenAndServe(":8080", nil)
}
```

Two things to internalise. First, the method names are constants
(`http.MethodPost`, `http.MethodGet`, …) — use them, don't hand-type strings.
Second, **`WriteHeader` may be called only once**, and once you write *any* body
the status is locked in. If you never call `WriteHeader`, Go sends `200 OK`
automatically on the first `Write`.

## 6. Headers: everything that isn't the body

Headers are key/value metadata on both requests and responses. A few families
you'll touch immediately:

- **Content negotiation** — `Content-Type` says what the body *is*
  (`application/json`), `Accept` (on the request) says what the client *wants*.
- **Caching** — `Cache-Control: max-age=60` lets clients and proxies reuse a
  response; `ETag` + `If-None-Match` let a server reply `304 Not Modified` with
  no body when nothing changed. Caching is one of HTTP's biggest performance
  wins and it's all header-driven.
- **Auth** — `Authorization: Bearer <token>` carries credentials on each
  request (see the next section on *why* it's every request).

Setting headers in Go — **always before** `WriteHeader`/`Write`, because once the
status line goes out the headers are already on the wire:

```go
package main

import "net/http"

func getNode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=60")
	w.WriteHeader(http.StatusOK) // 200
	w.Write([]byte(`{"id":3,"status":"online"}`))
}

func main() {
	http.HandleFunc("/nodes/3", getNode)
	http.ListenAndServe(":8080", nil)
}
```

Reading a request header is just as direct:

```go
auth := r.Header.Get("Authorization") // "" if absent — Get never panics
accept := r.Header.Get("Accept")
```

## 7. Statelessness: the server remembers nothing

HTTP is **stateless**: each request is self-contained, and the server keeps **no
memory of previous requests** as part of the protocol. Whatever the server needs
to handle your request — *who you are*, *what you want* — must travel **in that
request** (typically in headers like `Authorization`, or a cookie).

This sounds like a limitation; it's actually the property that makes the web
**scale horizontally**. Because no single server holds your session in memory,
*any* server behind a load balancer can handle *any* request. Add ten more boxes
and they all serve traffic immediately — no session affinity required. In Step 8,
when you study replication and consensus, you'll appreciate how much simpler life
is when the request-handling layer carries no hidden state. (When state *is*
needed — a login session — it's pushed into a token, a cookie, or a shared store
like Redis, deliberately, not smuggled into server memory.)

## 8. Connection reuse: keep-alive

Statelessness is about *application* state. The TCP connection underneath is a
separate concern, and here HTTP/1.1 made a key optimisation: **persistent
connections** (keep-alive) are the default. Instead of paying the TCP (and TLS)
handshake cost for every request, the client and server **reuse one connection**
for many request/response pairs.

This matters more than it looks: that handshake is a full network round trip (or
several, with TLS), and on a high-latency link it dominates. Go's `http.Client`
does this for you automatically via its connection pool — which is exactly why
you're told to **create one `http.Client` and reuse it**, not make a fresh one
per request. A new client per call throws the pool away and you pay handshakes
you didn't need to.

## 9. A note on HTTP/1.1 vs HTTP/2

HTTP/1.1's weakness is **head-of-line blocking**: even with keep-alive, responses
on one connection come back in order, so a slow response stalls the ones behind
it. Browsers worked around this by opening *several* parallel connections.

**HTTP/2** fixes it at the protocol level with **multiplexing**: many concurrent
requests and responses (*streams*) share a **single** connection, interleaved so
a slow one doesn't block the rest. It also adds **header compression** (HPACK)
and binary framing. Crucially, the **semantics are identical** — same methods,
same status codes, same headers. HTTP/2 changes *how bytes move on the wire*, not
*what the messages mean*. Everything you learned above carries over unchanged;
Go's standard server even speaks HTTP/2 automatically over TLS.

## Do it yourself

1. Spin up the section-5 server (`mkdir httpdeep && cd httpdeep && go mod init example.com/httpdeep`, paste the handler, `go run .`).
2. With `curl -i` (the `-i` prints response headers), hit it three ways: `curl -i -X POST localhost:8080/nodes`, then `curl -i localhost:8080/nodes` (a `GET`). Watch the second return `405` *and* the `Allow` header.
3. Add a `GET` handler that sets `Content-Type` and `Cache-Control` (section 6). Confirm the headers appear in `curl -i` output.
4. Make a list: for each of `GET/POST/PUT/PATCH/DELETE`, write down *safe?* and *idempotent?* from memory, then check against section 3.
5. Reason through one scenario out loud: *a client sends `POST /orders`, the response is lost to a timeout, the client retries.* What can go wrong, and how does an idempotency key fix it?
6. Read MDN's [**HTTP overview and reference**](https://developer.mozilla.org/en-US/docs/Web/HTTP) — skim *Methods*, *Response status codes*, and *Headers*. It's the canonical free reference; bookmark it.
7. Read the **Primer on Latency and Bandwidth** and **HTTP/1.X / HTTP/2** chapters of [**High Performance Browser Networking**](https://hpbn.co/) (free online) for keep-alive, head-of-line blocking, and multiplexing in depth.

## Check yourself

You're ready to move on when you can answer, *without looking*:

- What's the difference between a method being **safe** and being **idempotent**, and which methods are each?
- Why is a retried `PUT` safe but a retried `POST` not — and what's the standard fix to make creation safe to retry?
- What do the `4xx` and `5xx` families mean, and which one do you retry with backoff?
- What does it mean that HTTP is **stateless**, and how does that property help a system scale horizontally?
- Why should you reuse a single `http.Client`, and what does HTTP/2 multiplexing solve that HTTP/1.1 keep-alive alone does not?

When all five feel obvious, commit this task on your **Roadmap** and take the
Step 3 quiz. Next step: **REST APIs** — where these methods and status codes
become the vocabulary you design with.

## Common interview gotchas

- **Designing a retry-safe POST = idempotency key, stored atomically.** The *client* sends a stable key (a UUID) per operation, reused on every retry; the server, in **one transaction**, dedups on the key and persists the response so retries return the *same* result instead of re-charging. Score points by noting: store the response, reject key reuse with a different body, set a TTL, and make the dedup record + side effect atomic.
- **Safe ≠ idempotent.** Safe = read-only; idempotent = N calls leave the same end *state* as one. PUT and DELETE are idempotent but **not** safe; POST is neither; PATCH usually isn't (`increment by 1`). DELETE-again is idempotent even though the *status* changes (204 → 404) — state, not status, is what counts.
- **4xx vs 5xx drives retry policy.** 4xx = client's fault, **don't retry unchanged**; 5xx = server's fault, often transient → **retry *idempotent* requests with exponential backoff + jitter**. 429 is an explicit "slow down" (honor `Retry-After`). Never return 200 with an error body.
- **CORS is enforced by the *browser*, and it's the *server's* opt-in.** The request often reaches the server; the browser hides the response unless headers allow the origin. Non-simple requests trigger a **preflight `OPTIONS`** the server must answer with `Access-Control-Allow-Origin/Methods/Headers`. Pitfall: `*` is forbidden when credentials are sent — the origin must be exact.
- **Statelessness is what enables horizontal scaling.** Each request self-contained ⇒ any box behind the LB serves any request (easy failover, rolling deploys, no affinity). Stash session in **process memory** and you force sticky sessions and lose it on redeploy — push state into a token or a shared store (Redis/Postgres) instead.
