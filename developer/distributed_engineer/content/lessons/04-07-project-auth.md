---
slug: project-auth-signup-login
step: 4
title: "Project: user signup, login & protected routes"
summary: Add users, bcrypt signup, JWT login and middleware so each user only sees their own bookmarks.
est_min: 480
position: 7
---

# Project: user signup, login & protected routes

> **Step 4 · Build real backend APIs · Week 3 · Project**
> Concept: *secure auth flow*

Turn your bookmarks API into a multi-user service. Users sign up, log in, get a
token, and can only see and edit *their own* bookmarks. This is the real,
end-to-end auth flow — the same shape as production systems like Lokatalent.

## What you'll build

```
POST /signup   {email, password}  -> 201 (creates a user, bcrypt-hashed password)
POST /login    {email, password}  -> 200 {token}  (or 401)
GET  /api/bookmarks               -> only the caller's bookmarks   (needs token)
POST /api/bookmarks               -> creates one owned by the caller (needs token)
```

## Why this matters

Almost every backend feature lives behind auth. Doing the full loop — hash on
signup, verify on login, sign a token, guard routes, scope data by owner — is the
backend skill interviews and jobs assume you have.

## 1. Users table + migration

```sql
-- 0002_create_users.up.sql
CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- and give bookmarks an owner:
ALTER TABLE bookmarks ADD COLUMN user_id BIGINT NOT NULL REFERENCES users(id);
```

## 2. Signup — hash, then store

```go
func (a *API) signup(c echo.Context) error {
	var in struct{ Email, Password string }
	if err := c.Bind(&in); err != nil || in.Email == "" || len(in.Password) < 8 {
		return echo.NewHTTPError(http.StatusBadRequest, "email and 8+ char password required")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	id, err := a.users.Create(c.Request().Context(), in.Email, string(hash))
	if err != nil {
		// a duplicate email is a 409, not a 500 — detect the unique-violation
		return echo.NewHTTPError(http.StatusConflict, "email already registered")
	}
	return c.JSON(http.StatusCreated, map[string]int64{"id": id})
}
```

## 3. Login — verify, then sign a token

```go
func (a *API) login(c echo.Context) error {
	var in struct{ Email, Password string }
	if err := c.Bind(&in); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "bad body")
	}
	u, err := a.users.ByEmail(c.Request().Context(), in.Email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)) != nil {
		// Same 401 for "no such user" AND "wrong password" — don't leak which.
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}
	token := signJWT(u.ID, a.secret) // exp 24h, sub = user id (from the auth lesson)
	return c.JSON(http.StatusOK, map[string]string{"token": token})
}
```

> **Security detail:** return the *same* error for unknown-email and wrong-password.
> Different errors let an attacker enumerate which emails are registered.

## 4. Protect the routes + scope by owner

```go
protected := e.Group("/api", RequireAuth(a.secret)) // middleware from the auth lesson
protected.GET("/bookmarks", a.list)
protected.POST("/bookmarks", a.create)

func (a *API) list(c echo.Context) error {
	uid := c.Get("userID").(int64)               // set by RequireAuth
	bs, err := a.store.ListByUser(c.Request().Context(), uid) // WHERE user_id=$1
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, bs)
}
```

The ownership scope (`ListByUser`, and an ownership check on get/update/delete) is
**authorization** — without it, any logged-in user could read everyone's data.

```go
// YOUR TURN:
//  - users store: Create(email, hash) (id, error) [map unique violation -> conflict],
//    ByEmail(email) (*User, error)
//  - signJWT / RequireAuth (reuse the auth lesson)
//  - ListByUser, and ownership checks on get/update/delete (404 or 403 if not owner)
```

## Stretch goals

- Add a `refresh token` flow and a `POST /logout`.
- Rate-limit `/login` (reuse the Step 5 rate limiter) to slow brute force.
- Add roles (`user`/`admin`) and an admin-only route — real authorization.
- Set the token in an `HttpOnly` cookie instead of a header and compare the trade-offs.

## Done when

- [ ] Signup stores a **bcrypt hash**, never the plaintext password.
- [ ] Login returns a signed JWT on success and **401** on bad credentials (same message for both failure modes).
- [ ] `/api/bookmarks` rejects requests with no/invalid token (**401**).
- [ ] A user can only see and modify **their own** bookmarks.
- [ ] You can explain which parts are authentication and which are authorization.

FREE source: [**Let's Go**](https://lets-go.alexedwards.net/) (the gold standard for production Go web auth).

Next step: **Step 5 — Concurrency in Go.**

## Common interview gotchas

- **"Login errors can say 'no such user' vs 'wrong password'."** That's account enumeration. Also watch **timing**: bailing early for unknown emails (skipping the bcrypt compare) is a side channel — run a dummy compare so both paths take the same time.
- **"401 vs 403 vs 404 — pick whichever."** 401 = not authenticated (who are you?), 403 = authenticated but not allowed (authorization), 404 = doesn't exist. To avoid leaking that a resource exists to a non-owner, returning **404 instead of 403** is often the right call.
- **"Auth middleware passed, so the request is safe."** Authentication ≠ authorization. The #1 web vuln (broken access control) is a missing `WHERE user_id = $me` — without it any logged-in user reads everyone's data. Scope every query by owner.
- **"Rate-limit login by IP."** Per-IP alone misses distributed/botnet attacks and punishes users behind a shared NAT. Limit per-**account** AND per-IP, and add backoff after repeated failures.
- **"Duplicate email is a 500."** Catch the unique-violation and return **409 Conflict** — but be aware a distinct "already registered" response is itself an enumeration vector; some designs deliberately blur it.
