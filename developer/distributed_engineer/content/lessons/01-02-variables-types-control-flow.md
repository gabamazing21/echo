---
slug: go-variables-types-control-flow
step: 1
title: Variables, types, functions & control flow
summary: Master Go's static types, var vs :=, zero values, multi-return functions, and the if/for/switch control flow at the heart of every program.
est_min: 360
position: 2
---

# Variables, types, functions & control flow

> **Step 1 · Go fundamentals · Week 1**
> Concept: *static typing, declarations, scope*

Now that you can run Go, you need the raw material everything is built from:
**values, the types that constrain them, the functions that move them around, and
the three keywords that control flow.** Go is deliberately small here — there's
one loop, no exceptions, no operator overloading. Learn this handful of rules
cold and you can read almost any Go file on earth.

## Why this matters

Distributed systems are mostly *careful state plus careful control flow*: a node
holds typed state (a term number, a log index, a peer list) and reacts to events
with `if`/`for`/`switch`. Go being **statically typed** means the compiler catches
a whole class of bugs — passing a `string` where an `int64` term belongs — before
your code ever ships to a replica you can't easily debug. The multi-return
`value, err` pattern you learn today is how *every* fallible operation in Go
reports failure, including the network calls that hold a cluster together.

## 1. `var` vs `:=`: two ways to declare

Go gives you an explicit form and a short form. They produce the same thing.

```go
package main

import "fmt"

func main() {
	var nodes int = 3      // explicit type and value
	var leader = "node-1"  // type inferred as string
	var healthy bool       // declared, no value yet → zero value

	peers := 5             // short declaration: infer + assign, := only inside functions

	fmt.Println(nodes, leader, healthy, peers) // 3 node-1 false 5
}
```

- `var` works at **package scope and function scope**; `:=` works **only inside
  functions**.
- Use `:=` for everyday local variables. Reach for `var` when you want an explicit
  type, a package-level variable, or just the zero value.
- The compiler will **refuse to build** if you declare a variable and never use
  it. Unused locals are errors, not warnings — Go forces tidy code.

## 2. Zero values: nothing is ever uninitialised

Declare a variable without assigning it and Go gives it a well-defined **zero
value**. There is no "garbage memory" in Go.

```go
var (
	count   int     // 0
	ratio   float64 // 0
	name    string  // "" (empty string, not nil)
	ok      bool    // false
	pointer *int    // nil
)
```

This is a safety feature you'll lean on constantly: a freshly-declared struct or
counter is always in a usable, predictable state.

## 3. The basic types

```go
var i int     = 42            // platform-sized signed integer (64-bit on modern machines)
var big int64 = 9_000_000_000 // explicit width; underscores are just readability
var f float64 = 3.14          // the default float; prefer it over float32
var s string  = "consensus"   // immutable UTF-8 bytes
var b bool    = true

var r rune = '✓' // alias for int32: a single Unicode code point
var c byte = 'A' // alias for uint8: a single raw byte
```

A few rules that trip up newcomers:

- **No implicit conversions.** `int` and `int64` are different types. You must
  convert explicitly: `int64(i)`.
- A `string` is a read-only sequence of **bytes**. Range over it and you walk
  **runes** (code points), which is what you want for real text:

```go
for index, r := range "héllo" {
	fmt.Printf("%d=%c ", index, r) // indices jump because é is two bytes
}
```

## 4. Constants & `iota`

Constants are fixed at compile time. `iota` is a counter that auto-increments
within a `const` block — the idiomatic way to build enumerations.

```go
const maxRetries = 3 // untyped constant; adapts to context

type State int

const (
	Follower  State = iota // 0
	Candidate              // 1
	Leader                 // 2
)

func (s State) String() string {
	switch s {
	case Follower:
		return "follower"
	case Candidate:
		return "candidate"
	default:
		return "leader"
	}
}
```

You'll meet exactly this `Follower / Candidate / Leader` enum again in Step 8 when
you implement Raft. Constants can't be changed at runtime and can't take their
address — they're values, not storage.

## 5. Functions & multiple return values

A function lists its parameter types and its return types. Go's signature
hallmark is returning **more than one value** — almost always `result, err`.

```go
package main

import (
	"errors"
	"fmt"
)

// divide returns a quotient and an error. The caller MUST decide what to do
// with both.
func divide(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}
	return a / b, nil
}

func main() {
	q, err := divide(10, 2)
	if err != nil {
		fmt.Println("failed:", err)
		return
	}
	fmt.Println("quotient:", q) // quotient: 5
}
```

**Named returns** let you name the result variables in the signature. A bare
`return` then sends back their current values — handy with `defer`, but use it
sparingly; it can hurt readability.

```go
func split(sum int) (x, y int) {
	x = sum * 4 / 9
	y = sum - x
	return // returns x and y
}
```

Use `_` (the blank identifier) to deliberately discard a return you don't need:
`_, err := divide(10, 0)`.

## 6. `if` — and conditions can carry a statement

No parentheses around the condition; braces are mandatory. `if` can run a short
statement first, scoping a variable to the `if`/`else`:

```go
if q, err := divide(20, 4); err == nil {
	fmt.Println("ok:", q) // q and err exist only inside this if/else
} else {
	fmt.Println("err:", err)
}
```

This keeps error-only variables out of the surrounding scope — very common Go
style.

## 7. `for` — the only loop

Go has exactly one looping keyword. It wears several outfits:

```go
// classic three-part
for i := 0; i < 3; i++ {
	fmt.Println(i)
}

// while-style (no init/post)
n := 1
for n < 100 {
	n *= 2
}

// infinite loop; break out of it
for {
	if n > 1000 {
		break
	}
	n *= 2
}

// range over a collection
peers := []string{"node-1", "node-2", "node-3"}
for i, p := range peers {
	fmt.Println(i, p)
}
```

`break` and `continue` work as you'd expect. There is no `while` or `do` — once
you internalise that `for` covers all of them, the language feels smaller.

## 8. `switch` — clean, no fallthrough

Go's `switch` does **not** fall through by default — no `break` needed. Cases can
be expressions, and a tagless `switch true` reads like a clean `if/else` ladder.

```go
func classify(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "success"
	case code >= 400 && code < 500:
		return "client error"
	case code >= 500:
		return "server error"
	default:
		return "other"
	}
}

// value switch with a deliberate fallthrough (rare, must be explicit)
switch x := 1; x {
case 1:
	fmt.Println("one")
	fallthrough // explicitly run the next case too
case 2:
	fmt.Println("two")
}
```

## 9. Scope & shadowing

A variable lives inside the **block** (the nearest `{ }`) where it's declared.
Declaring a name that already exists in an outer block creates a *new* variable
that **shadows** the outer one — a classic source of subtle bugs.

```go
x := "outer"
{
	x := "inner" // a different variable, only visible in this block
	fmt.Println(x) // inner
}
fmt.Println(x) // outer
```

Watch for accidental shadowing with `:=` inside an `if`: if you meant to assign to
an outer `err` but write `:=`, you create a fresh `err` that the outer code never
sees. Run `go vet` — it flags many of these.

## Do it yourself (≈ the rest of the session)

1. In your `hello` module, write a program that declares one variable of every
   basic type using both `var` and `:=`, and prints each with `fmt.Printf("%T %v\n", v, v)`
   to see its type and value.
2. Write `func divmod(a, b int) (int, int, error)` returning quotient,
   remainder, and an error for `b == 0`. Call it and handle the error.
3. Loop over a `[]string` of node names with `range`; use a `switch` to print a
   different message for the first, last, and middle nodes.
4. Deliberately shadow a variable, then run `go vet ./...` and read what it says.
5. Work through **A Tour of Go — "Basics"**:
   [go.dev/tour/basics/1](https://go.dev/tour/basics/1). Do *every* exercise in
   the browser; it covers packages, variables, types, and functions hands-on.
6. Skim the matching examples on **Go by Example**:
   [gobyexample.com](https://gobyexample.com/) — *Variables*, *Constants*, *For*,
   *If/Else*, *Switch*, and *Functions / Multiple Return Values*.

## Check yourself

You're ready to move on when you can answer, *without looking*:

- When can you use `:=`, and when must you use `var` instead?
- What is the zero value of `int`, `string`, `bool`, and a pointer?
- Why does Go make you convert between `int` and `int64` explicitly?
- How does a Go function return an error, and what must the caller do with it?
- Why doesn't Go's `switch` need `break`, and what does `switch true` (tagless) let you write?

When all five feel obvious, commit this task on your **Roadmap** and take the
Step 1 quiz. Next lesson: **pointers, structs, slices, and maps.**

## Common interview gotchas

- **`:=` inside an `if`/`for` silently shadows an outer `err`** — `if x, err := f(); ...` creates a *new* `err` in that block; the outer one stays `nil` and the failure vanishes. This is the single most common Go bug `go vet -vettool=shadow` exists to catch.
- **Writing to a nil map panics; reading doesn't** — the zero value of a map is `nil`, and `m[k]` on it returns a zero value happily, lulling you in. The first `m[k] = v` blows up at runtime. Always `make` (or use a literal) before writing.
- **Integer division truncates toward zero and overflow wraps silently** — `7/2 == 3`, and an `int8` at 127 plus 1 becomes -128 with no panic. There are no checked-arithmetic exceptions; size your types and guard ranges yourself.
- **`iota` resets per `const` block and counts lines, not values** — skipping a value (`_`), a blank line, or a multi-name line shifts everything after it. And since `iota` enums are just ints, an unset field silently equals the *first* constant — make the zero value an explicit `Unknown`/`Invalid`.
- **A bare `return` with named results returns whatever those variables currently hold** — combined with `defer` that mutates them, the returned value can differ from what the `return` line "looks like" it sends. Read the defers before trusting a named-return signature.
