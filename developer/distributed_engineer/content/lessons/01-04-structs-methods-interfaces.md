---
slug: go-structs-methods-interfaces
step: 1
title: Structs, methods, interfaces & pointers
summary: Model data with structs, attach behaviour with methods, and let interfaces give Go its quiet, implicit polymorphism.
est_min: 480
position: 4
---

# Structs, methods, interfaces & pointers

> **Step 1 · Go fundamentals · Week 2**
> Concept: *value vs pointer receivers, interface satisfaction*

You now know variables, types, and control flow. This lesson is where Go stops
being a calculator and starts being a *systems language*. Structs let you model
the things in your system — a node, a message, a log entry. Methods give those
things behaviour. And **interfaces** — Go's whole answer to polymorphism — let
you write code against *what something does* instead of *what it is*. Take your
time here; this is the conceptual core of the whole language.

## Why this matters

Distributed-systems code is built out of interfaces. The standard library hands
you `io.Reader` and `io.Writer` — two tiny interfaces — and *everything*
composes through them: files, network sockets, in-memory buffers, gzip streams,
HTTP bodies. Your Raft node in Step 8 will talk to its peers through an interface
so you can swap a real TCP transport for a fake one in tests. `error` itself is
an interface. Once interfaces *click*, huge swathes of Go stop looking like magic
and start looking like the same simple idea repeated everywhere.

## 1. Structs: grouping data that belongs together

A **struct** is a typed collection of fields. It's how you name a concept in your
program:

```go
package main

import "fmt"

type Server struct {
	Host string
	Port int
	Up   bool
}

func main() {
	// Struct literal with field names — clearest, order-independent.
	s := Server{Host: "10.0.0.1", Port: 8080, Up: true}

	// Access and update fields with a dot.
	s.Up = false
	fmt.Printf("%s:%d up=%v\n", s.Host, s.Port, s.Up)

	// The zero value of a struct has every field at its own zero value.
	var empty Server
	fmt.Printf("%+v\n", empty) // {Host: Port:0 Up:false}
}
```

Note `%+v` — it prints field names too, which is invaluable when debugging.

## 2. Pointers: `&` takes an address, `*` follows it

A **pointer** holds the memory address of a value. You get one with `&`, and you
read through it with `*` (called *dereferencing*).

```go
package main

import "fmt"

func main() {
	count := 41
	p := &count // p is a *int — "pointer to int"

	*p = *p + 1   // follow p, change what it points at
	fmt.Println(count) // 42 — count itself changed
}
```

Why pointers at all? Two reasons: to **mutate** a value a function received, and
to **avoid copying** large structs. By default Go passes everything *by value* (a
copy), so without a pointer a function can't change the caller's data.

> **No pointer arithmetic.** Unlike C, you cannot do `p++` to walk through memory.
> Go pointers can only point *at* a value or be `nil` — they can't be nudged to a
> neighbouring address. This is deliberate: it removes a whole category of memory
> bugs and lets the garbage collector and race detector reason about your program.

## 3. Methods: functions with a receiver

A **method** is a function with a special *receiver* argument attached before the
name. That receiver binds the function to a type:

```go
type Server struct {
	Host string
	Port int
}

// (s Server) is the receiver. Address reads the server's address.
func (s Server) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// Called like: srv.Address()
```

You can define methods on any type *you* declared in the same package — not just
structs:

```go
type Celsius float64

func (c Celsius) Fahrenheit() float64 {
	return float64(c)*9/5 + 32
}
```

## 4. Value vs pointer receivers — the decision that trips everyone up

A receiver can be a **value** (`func (s Server)`) or a **pointer**
(`func (s *Server)`). The difference is the same as for any argument: a value
receiver gets a *copy*, a pointer receiver gets the *original*.

```go
package main

import "fmt"

type Counter struct{ n int }

// Value receiver: operates on a COPY. The caller never sees the change.
func (c Counter) IncBroken() { c.n++ }

// Pointer receiver: operates on the ORIGINAL. The change sticks.
func (c *Counter) Inc() { c.n++ }

func main() {
	c := Counter{}
	c.IncBroken()
	fmt.Println(c.n) // 0 — mutation was lost on the copy

	c.Inc()
	fmt.Println(c.n) // 1 — pointer receiver mutated the real thing
}
```

**The rule of thumb:**

- Use a **pointer receiver** if the method needs to **modify** the receiver, or
  if the struct is **large** (copying it is wasteful).
- Use a **value receiver** for small, immutable types where a copy is cheap and
  you want to signal "this method doesn't change anything."
- **Be consistent.** If *any* method on a type uses a pointer receiver, use
  pointer receivers for *all* of them. Mixing the two is a well-known source of
  confusion and subtle bugs.

Go is friendly about the call site: you can call a pointer-receiver method on an
addressable value (`c.Inc()` above) and Go inserts the `&` for you. The reverse
also works. But the *copy vs original* semantics still apply exactly as shown.

## 5. Interfaces & implicit satisfaction

An **interface** is a set of method signatures. Any type that has those methods
*automatically* satisfies the interface — there is **no `implements` keyword**.
This is the single most important idea in this lesson.

```go
package main

import (
	"fmt"
	"math"
)

// An interface: anything with an Area() float64 method is a Shape.
type Shape interface {
	Area() float64
}

type Rectangle struct{ W, H float64 }
type Circle struct{ R float64 }

func (r Rectangle) Area() float64 { return r.W * r.H }
func (c Circle) Area() float64    { return math.Pi * c.R * c.R }

// totalArea works on ANY Shape — it never names Rectangle or Circle.
func totalArea(shapes []Shape) float64 {
	var sum float64
	for _, s := range shapes {
		sum += s.Area()
	}
	return sum
}

func main() {
	shapes := []Shape{Rectangle{W: 3, H: 4}, Circle{R: 2}}
	fmt.Printf("%.2f\n", totalArea(shapes)) // 24.57
}
```

`Rectangle` and `Circle` never *say* they are `Shape`s. Because each has an
`Area() float64` method, they simply *are*. This is called **structural** or
**duck typing**, checked at compile time. The payoff: you can make a type from
*someone else's package* satisfy *your* interface, and you can define an
interface that existing types already satisfy — no edits to the original code.

> **Receiver type matters for satisfaction.** If a method has a *pointer*
> receiver, only the pointer (`*T`) satisfies the interface, not the value (`T`).
> So `var s Shape = &myStruct` may compile where `var s Shape = myStruct`
> doesn't. When an interface assignment "mysteriously" fails to compile, this is
> usually why.

## 6. The empty interface & `any`

The interface with *no* methods is satisfied by **every** type. Historically
written `interface{}`, Go 1.18 added the alias `any` — they are identical.

```go
func describe(v any) {
	fmt.Printf("value=%v type=%T\n", v, v)
}

describe(42)        // value=42 type=int
describe("hello")   // value=hello type=string
describe([]int{1})  // value=[1] type=[]int
```

`any` is the escape hatch for "I'll accept literally anything" (e.g. logging,
`encoding/json`). Reach for it sparingly — you lose all compile-time type
safety and have to recover the real type by hand, which is the next section.

## 7. Type assertions & type switches

To pull a concrete type back out of an interface value, use a **type assertion**.
Always use the two-result form so a wrong guess doesn't panic:

```go
var v any = "consensus"

if s, ok := v.(string); ok {
	fmt.Println("string of length", len(s))
}
```

When you need to branch over several possibilities, use a **type switch**:

```go
func kind(v any) string {
	switch x := v.(type) {
	case int:
		return fmt.Sprintf("int: %d", x)
	case string:
		return fmt.Sprintf("string: %q", x)
	case Shape:
		return fmt.Sprintf("shape with area %.2f", x.Area())
	default:
		return "unknown"
	}
}
```

Inside each `case`, `x` already has the matched type — no extra assertion needed.

## 8. Two interfaces you'll meet constantly: `Stringer` and `error`

The standard library defines small interfaces that your types can opt into. The
most common is `fmt.Stringer` — implement `String() string` and your type prints
nicely everywhere:

```go
type Server struct {
	Host string
	Port int
}

func (s Server) String() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// Now fmt.Println(server) prints "10.0.0.1:8080" automatically.
```

And `error` is *just an interface* — nothing special:

```go
type error interface {
	Error() string
}
```

So you can define your own error types by giving them an `Error()` method:

```go
type DialError struct {
	Host string
}

func (e *DialError) Error() string {
	return "could not dial " + e.Host
}

func connect(host string) error {
	return &DialError{Host: host} // satisfies error implicitly
}
```

This is *exactly* how the errors you've already been checking (`if err != nil`)
work under the hood. There's no magic — just an interface with one method.

## Do it yourself

Spend the bulk of this session writing code, not reading. Type every example;
don't copy-paste.

1. Do the **Tour of Go** section
   [*Methods and interfaces*](https://go.dev/tour/methods/1) end to end — every
   slide and exercise, including the `Stringer` and `error` exercises near the
   end. This is the single best free walkthrough of today's material.
2. Build a tiny program: define a `Notifier` interface with a
   `Notify(msg string) error` method, then write two types — `EmailNotifier`
   and `SlackNotifier` — that satisfy it. Write a function
   `broadcast(ns []Notifier, msg string)` that calls all of them. Notice you
   never wrote `implements` anywhere.
3. Deliberately break it: give one notifier a **pointer** receiver and try to
   store the *value* in a `[]Notifier`. Read the compile error until it makes
   sense — that's section 5's warning in action.
4. Read the **interfaces** part of
   [*Effective Go*](https://go.dev/doc/effective_go#interfaces) — short, dense,
   and written by the Go team. It explains *why* interfaces are small.
5. Work through the early chapters of
   [*Learn Go with Tests*](https://quii.gitbook.io/learn-go-with-tests),
   especially the *Structs, methods & interfaces* chapter, writing the tests
   yourself. This pairs the concepts with Go's test-first culture.

## Check yourself

You're ready to move on when you can answer, *without looking*:

- When should a method use a **pointer receiver** instead of a value receiver,
  and what goes wrong if you mutate through a value receiver?
- Why does Go have *no* `implements` keyword — how does a type come to satisfy an
  interface?
- What's the difference between `&x` and `*p`, and why can't you do pointer
  arithmetic in Go?
- What is `any` (a.k.a. `interface{}`), and how do you safely recover the
  concrete type stored in it?
- `error` and `fmt.Stringer` are "just interfaces" — what one method does each
  require?

When all five feel obvious, commit this task on your **Roadmap** and take the
Step 1 quiz. Next up: **goroutines, channels, and the `sync` package** — where
Go earns its reputation for concurrency.

## Common interview gotchas

- **A non-nil interface holding a nil pointer is *not* `== nil`** — return a `*MyError` that happens to be nil into an `error`, and `if err != nil` is *true* because the interface carries a type. This typed-nil trap silently turns "no error" into "error." Return a literal `nil`, never a typed nil pointer.
- **Pointer-receiver methods aren't in the value's method set** — `T` only has value-receiver methods; `*T` has both. So a value stored in an interface (or a non-addressable value like a map element) can fail to satisfy an interface its pointer would. This is why interface assignments "mysteriously" don't compile.
- **"Accept interfaces, return concrete types"** — taking an interface param keeps callers flexible, but *returning* an interface hides the real type, blocks callers from new methods, and invites the typed-nil trap above. Return the struct (or `*struct`); let the caller narrow to an interface.
- **Embedding is composition, not inheritance** — a promoted method called on the outer type still runs with the *embedded* type as its receiver; it can't see the outer type's fields or overrides. There's no virtual dispatch up the chain. Don't reach for embedding expecting `super`-style behavior.
- **`struct{}{}` costs zero bytes — use it for set membership and signaling** — `map[string]struct{}` is the idiomatic set (a `map[string]bool` wastes a byte per entry and invites `false`-vs-absent confusion), and `chan struct{}` signals without sending data.
