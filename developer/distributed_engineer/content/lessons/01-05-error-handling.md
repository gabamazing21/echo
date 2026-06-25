---
slug: go-error-handling
step: 1
title: Error handling the Go way
summary: Treat errors as ordinary values — check, wrap, and inspect them with errors.Is/As instead of throwing exceptions.
est_min: 300
position: 5
---

# Error handling the Go way

> **Step 1 · Go fundamentals · Week 2**
> Concept: *errors as values, wrapping, errors.Is/As*

Go has no exceptions. There is no `try`, no `catch`, no invisible stack-unwinding
that whisks control away to some handler three layers up. Instead, a function
that can fail simply *returns* its failure alongside its result, and you decide —
right there, in plain code — what to do about it. This feels verbose at first.
It is the single most important habit you will build this week.

## Why this matters

In a single-process script, failure is the exception. In a distributed system,
failure is the **normal case**: the network drops packets, a peer is slow, a
disk fills, a leader dies mid-write. The systems you'll build in later steps —
replicated logs, consensus protocols, RPC layers — spend most of their code on
the failure paths, not the happy path.

Go's design forces failure into the open. Every `if err != nil` is a decision
you were *made* to make rather than one you forgot. That explicitness isn't
ceremony; it's the feature that keeps a 10,000-node cluster from silently
corrupting state. You're learning to treat errors as a normal part of control
flow, not an emergency.

## 1. The `error` interface

An error in Go is just a value satisfying a one-method interface from the
standard library:

```go
type error interface {
	Error() string
}
```

Anything with an `Error() string` method *is* an error. A `nil` error means "no
failure." This is the whole foundation — errors are ordinary values you can
store, compare, pass around, and return.

## 2. The `if err != nil` idiom

The convention: failable functions return the result **and** an error as the
last value. You check the error immediately.

```go
package main

import (
	"fmt"
	"strconv"
)

func main() {
	n, err := strconv.Atoi("42")
	if err != nil {
		fmt.Println("bad input:", err)
		return
	}
	fmt.Println("parsed:", n)
}
```

The rule of thumb: **handle the error or return it — never ignore it.** When a
value is genuinely irrelevant you may discard it with `_`, but do that
deliberately, not out of laziness.

## 3. Creating errors: `errors.New` and `fmt.Errorf`

For a fixed message, use `errors.New`. When you need to interpolate context, use
`fmt.Errorf`:

```go
package main

import (
	"errors"
	"fmt"
)

func withdraw(balance, amount int) (int, error) {
	if amount > balance {
		return balance, errors.New("insufficient funds")
	}
	if amount < 0 {
		return balance, fmt.Errorf("invalid amount: %d", amount)
	}
	return balance - amount, nil
}

func main() {
	if _, err := withdraw(100, 250); err != nil {
		fmt.Println(err) // insufficient funds
	}
}
```

Error strings are conventionally lowercase and without trailing punctuation,
because they're often wrapped inside other messages.

## 4. Wrapping errors with `%w`

When an error bubbles up through layers, each layer should add context *without
throwing away the original*. The `%w` verb in `fmt.Errorf` **wraps** an error,
preserving it for later inspection:

```go
package main

import (
	"errors"
	"fmt"
)

func readConfig(path string) error {
	err := errors.New("permission denied")
	// %w keeps the original error reachable; %v would flatten it to text.
	return fmt.Errorf("read config %q: %w", path, err)
}

func main() {
	err := readConfig("/etc/app.conf")
	fmt.Println(err) // read config "/etc/app.conf": permission denied
}
```

The difference between `%w` and `%v` is critical: `%v` produces a *string*,
losing the original error's identity and type. `%w` builds a chain you can walk
later with the functions in the next section.

## 5. Inspecting wrapped errors: `errors.Is` and `errors.As`

Because wrapping hides the original error inside a chain, you must **not** compare
errors with `==` once they may be wrapped. Use the two inspectors instead:

- `errors.Is(err, target)` — is `target` *anywhere* in the chain? Use it to match
  a specific sentinel value.
- `errors.As(err, &target)` — is there an error of a *specific type* in the
  chain? If so, it's assigned into `target` so you can read its fields.

```go
package main

import (
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("not found")

// A custom error type carrying structured detail.
type QueryError struct {
	Query string
	Err   error
}

func (e *QueryError) Error() string {
	return fmt.Sprintf("query %q: %v", e.Query, e.Err)
}

// Unwrap lets errors.Is / errors.As see through this type.
func (e *QueryError) Unwrap() error { return e.Err }

func lookup(user string) error {
	return &QueryError{Query: "SELECT ... " + user, Err: ErrNotFound}
}

func main() {
	err := lookup("alice")

	// errors.Is walks the chain to find the sentinel.
	if errors.Is(err, ErrNotFound) {
		fmt.Println("no such user — treating as empty result")
	}

	// errors.As extracts the typed error to read its fields.
	var qe *QueryError
	if errors.As(err, &qe) {
		fmt.Println("failed query was:", qe.Query)
	}
}
```

Note the `Unwrap() error` method: implementing it is what lets `errors.Is` and
`errors.As` see *through* your custom type into what it wraps. (Wrapping with
`%w` gives you `Unwrap` for free.)

## 6. Sentinel errors

A **sentinel** is a named, exported error value that callers compare against. The
standard library uses them constantly — `io.EOF` to signal end of input,
`sql.ErrNoRows` when a query matched nothing:

```go
package main

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

func main() {
	r := strings.NewReader("hi")
	buf := make([]byte, 1)
	for {
		_, err := r.Read(buf)
		if errors.Is(err, io.EOF) { // EOF is expected, not a crash
			fmt.Println("done reading")
			break
		}
		if err != nil {
			fmt.Println("real error:", err)
			break
		}
		fmt.Printf("%c", buf[0])
	}
}
```

Declare your own with `var ErrThing = errors.New("thing happened")` at package
scope, and always check them with `errors.Is`, never `==`, so wrapping can't
break the comparison.

## 7. `panic` and `recover` — and when NOT to use them

Go *does* have `panic` (abort the goroutine, unwind, run deferred functions) and
`recover` (stop a panic inside a deferred call). They are **not** Go's error
handling. They exist for one thing:

> **`panic` is for programmer bugs, not expected failures.**

A nil-pointer dereference, an out-of-bounds index, an "impossible" switch case —
these are *bugs in your code* and panicking is appropriate. A missing file, a
refused connection, a malformed request — these are *expected runtime
conditions* and must be returned as `error` values.

```go
package main

import "fmt"

func mustBalance(b int) {
	if b < 0 {
		// A negative balance here would mean our own logic is broken.
		panic(fmt.Sprintf("invariant violated: balance = %d", b))
	}
}

func safeCall() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered: %v", r)
		}
	}()
	mustBalance(-5)
	return nil
}

func main() {
	fmt.Println(safeCall()) // recovered: invariant violated: balance = -5
}
```

`recover` is mainly for stopping one bad request from taking down a whole server
(an HTTP handler recovering at the top of each request). If you find yourself
using panic/recover for ordinary control flow, stop — return an error instead.

## Do it yourself

Spend the session writing and breaking small programs, not just reading.

1. Write a `divide(a, b int) (int, error)` that returns a sentinel
   `ErrDivByZero` when `b == 0`. Call it and handle the error.
2. Wrap that error one layer up with `fmt.Errorf("...: %w", err)` and confirm
   `errors.Is(wrapped, ErrDivByZero)` still returns `true`.
3. Build a custom `ValidationError` type with a `Field` string and an `Unwrap`
   method, then pull it back out of a wrapped chain with `errors.As`.
4. Read the Go blog post
   [**Working with Errors in Go 1.13**](https://go.dev/blog/go1.13-errors) — the
   canonical explanation of `%w`, `errors.Is`, and `errors.As`. Type out its
   examples.
5. Read the *Errors* section of
   [**Effective Go**](https://go.dev/doc/effective_go#errors) for the idioms and
   naming conventions the community expects.

## Check yourself

You're ready to move on when you can answer, *without looking*:

- Why does Go return errors as values instead of throwing exceptions, and why
  does that matter more in distributed systems?
- What's the difference between `%w` and `%v` in `fmt.Errorf`, and why should you
  never compare a possibly-wrapped error with `==`?
- When do you reach for `errors.Is` versus `errors.As`?
- What method must a custom error type implement so `errors.Is`/`errors.As` can
  see through it?
- Give one case where `panic` is correct and one where returning an `error` is
  correct — what's the dividing line?

When all five feel obvious, commit this task on your **Roadmap** and take the
Step 1 quiz. Next lesson: **structs, methods, and interfaces.**

## Common interview gotchas

- **`%w` is part of your API contract; `%v` is not** — wrap with `%w` and callers can `errors.Is`/`As` through your error forever, which means you can *never* stop wrapping without breaking them. Wrap with `%v` and you flatten to a string, deliberately hiding the cause. Choose on purpose, not by habit.
- **`errors.Is`/`errors.As` over `==` and type assertions** — once any layer wraps, `err == ErrNotFound` and `err.(*MyErr)` both fail because the sentinel/type is now buried in the chain. `Is`/`As` walk the `Unwrap` chain; the bare operators don't. `errors.As` also requires a pointer-to-pointer (`&target`) or it panics.
- **Sentinel errors are tighter coupling than typed errors** — `var ErrX = errors.New(...)` makes callers depend on an exact value (and you can't attach context without wrapping); a custom type with fields lets them read structured detail via `errors.As`. Prefer typed errors when callers need more than "yes/no."
- **`panic`/`recover` is not exception handling and won't cross goroutines** — a `recover` only catches a panic in *its own* goroutine; a panic in a spawned goroutine crashes the whole process regardless of any `recover` upstairs. Don't use panic for control flow, and recover at goroutine boundaries (e.g. per HTTP request).
- **A `nil` error with a non-nil result is a real signal** — `io.Reader.Read` can return `n > 0` *and* `io.EOF` in the same call; treating EOF as "stop, discard" loses the final bytes. Always process `n` bytes before inspecting the error.
