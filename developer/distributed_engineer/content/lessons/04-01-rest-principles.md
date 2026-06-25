---
slug: api-rest-principles
step: 4
title: REST principles & JSON APIs
summary: How resources, HTTP verbs, status codes and idempotency combine into the JSON APIs every backend service exposes.
est_min: 240
position: 1
---

# REST principles & JSON APIs

> **Step 4 · Build real backend APIs · Week 1**
> Concept: *resources, verbs, status codes, idempotency*

You now know how HTTP works on the wire (Step 3). REST is the *discipline* layered
on top of it — a small set of conventions that turn raw HTTP into a predictable,
self-describing API. Almost every backend you'll ever build or call speaks it.

## Why this matters

A distributed system is a mesh of services calling each other's APIs. If those
APIs are consistent — nouns for things, verbs for actions, honest status codes —
the whole system stays comprehensible and *retry-safe*. If they're ad-hoc, every
integration becomes guesswork. REST is the shared grammar.

## 1. Resources are nouns, identified by URLs

REST models your domain as **resources** — things — each addressable by a URL.
Use plural nouns, not verbs:

```
GET    /bookmarks          # the collection
GET    /bookmarks/42       # one bookmark
POST   /bookmarks          # create one
PUT    /bookmarks/42       # replace one
PATCH  /bookmarks/42       # partially update one
DELETE /bookmarks/42       # remove one
```

`GET /getBookmark?id=42` is *not* REST — the verb belongs in the HTTP method, not
the path. The path names the thing; the method says what to do to it.

## 2. The verbs and their contracts

| Verb | Meaning | Safe? | Idempotent? |
|---|---|---|---|
| GET | read | ✅ yes | ✅ yes |
| POST | create / action | ❌ no | ❌ no |
| PUT | replace whole resource | ❌ no | ✅ yes |
| PATCH | partial update | ❌ no | usually |
| DELETE | remove | ❌ no | ✅ yes |

- **Safe** = no side effects (a GET must never change state).
- **Idempotent** = doing it twice has the same effect as once.

These aren't pedantry. In a distributed system, **requests get retried** — a
timeout doesn't tell you whether the server processed the request or not. If
DELETE is idempotent, a retry is harmless. If POST isn't, a retry might create a
duplicate. (You'll fix that with *idempotency keys* later.)

## 3. Status codes tell the truth

The status code is the first thing a caller reads. Use the right one:

| Range | Meaning | Common codes |
|---|---|---|
| 2xx | success | 200 OK, 201 Created, 204 No Content |
| 3xx | redirect | 301, 304 Not Modified |
| 4xx | **client** error | 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 409 Conflict, 422 Unprocessable |
| 5xx | **server** error | 500 Internal, 503 Unavailable |

The 4xx/5xx split is a promise: *4xx means you (the caller) sent something wrong;
5xx means I (the server) failed.* Returning 200 with `{"error": "..."}` in the
body breaks that promise and every client that trusts status codes.

```go
// In an Echo handler — the status code IS the API contract
return c.JSON(http.StatusCreated, bookmark)        // 201 after a successful POST
return c.JSON(http.StatusNotFound, errBody)        // 404 when /bookmarks/42 doesn't exist
return c.NoContent(http.StatusNoContent)           // 204 after a successful DELETE
```

## 4. JSON request & response shapes

REST APIs almost always speak JSON. Two rules keep you sane:

1. **Be consistent.** Same field names, same casing (`snake_case` or `camelCase` —
   pick one), same error shape everywhere.
2. **Validate input, never trust it.** A missing or malformed field is a `400`,
   not a `500`.

A conventional error body:

```json
{ "error": "url is required", "field": "url" }
```

A conventional list response with pagination:

```json
{ "data": [ ... ], "next_cursor": "eyJpZCI6NDJ9" }
```

## 5. Statelessness

Each request carries everything the server needs to handle it (auth token, body,
params). The server keeps **no per-client session state in memory** between
requests. Why? Because in a distributed system there are *many* server instances
behind a load balancer — your next request might hit a different one. Statelessness
is what lets you scale horizontally by just adding more identical instances.

## Do it yourself (≈ 4 hrs)

1. Read the MDN overview of [**HTTP methods**](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods) and [**status codes**](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status).
2. Skim the [**Echo guide**](https://echo.labstack.com/docs) intro — you'll use it in the next lesson.
3. On paper, design the full REST API for a "notes" service: list the URL + method + success status + error cases for create, read-one, list, update, delete.
4. For each endpoint, mark whether it's safe and idempotent, and decide what a *retry* should do.

## Check yourself

- Why is `GET /deleteUser?id=5` bad REST — and what should it be?
- What does "idempotent" mean, and why does it matter when requests get retried?
- A client sends an invalid email on signup. Which status code — and which *range* — and why?
- Why must a REST server be stateless to scale horizontally?
- What's the difference between `PUT` and `PATCH`?

Next: **the Echo framework** — the toolkit you'll build these APIs with (and the one this very app is written in).

## Common interview gotchas

- **"POST can't be made safe to retry."** Wrong — attach an **idempotency key** (client-generated, e.g. a UUID header); the server dedupes so a retried create doesn't double-charge. This is how Stripe/payments survive timeouts.
- **"Just return 200 with `{"error": ...}` in the body."** This silently breaks every client that routes on status code (retries, circuit breakers, monitoring). The 4xx/5xx split is a *contract*, not decoration.
- **"4xx vs 5xx is about how bad it is."** No — it's about *whose fault*: 4xx = caller sent something wrong (don't retry unchanged), 5xx = server failed (safe to retry). A bad request returned as 500 makes clients retry a doomed call.
- **"PUT and POST both create, so they're interchangeable."** PUT is idempotent (same body twice = one resource, you supply the id); POST is not (twice = two resources). Use PUT when the client owns the id, POST when the server mints it.
- **"Stateless just means no login state."** It means *no per-client state in server memory* — so any instance behind the LB can serve any request. One sticky in-memory cache and you've broken horizontal scaling and lost data on the next deploy.
