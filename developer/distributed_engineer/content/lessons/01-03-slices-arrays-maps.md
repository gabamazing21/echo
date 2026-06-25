---
slug: go-slices-arrays-maps
step: 1
title: Slices, arrays & maps — the core data containers
summary: Master Go's arrays, slices and maps — their internals, zero values, append, make, copy, and the sharp gotchas that bite real systems.
est_min: 360
position: 3
---

# Slices, arrays & maps — the core data containers

> **Step 1 · Go fundamentals · Week 1**
> Concept: *slices vs arrays, append, make, zero values*

You already have a runnable program and a feel for the toolchain. Now you learn
the three containers you'll reach for in *every* Go program you ever write:
**arrays**, **slices**, and **maps**. Slices and maps especially are everywhere —
request batches, in-memory caches, node membership lists, log buffers. Get their
internals into your bones now and you'll avoid a whole category of bugs later.

## Why this matters

Distributed systems are, underneath, programs that move and reshape *data*: a
batch of records to replicate, a set of peers to gossip with, a map from key to
the node that owns it. The way you store that data determines correctness and
performance. The single most common Go bug in real infrastructure code is a
**shared backing array** surprise — two slices that look independent but quietly
mutate each other. By the end of this lesson that bug will be obvious to you, not
mysterious. That's the difference between writing Go and writing *correct* Go.

## 1. Arrays: fixed size, value semantics

An **array** has a length baked into its type. `[3]int` and `[4]int` are
*different types*. The length is part of what the array *is*.

```go
package main

import "fmt"

func main() {
	var a [3]int       // zero value: all elements zeroed -> [0 0 0]
	a[0] = 10
	a[2] = 30
	fmt.Println(a, len(a)) // [10 0 30] 3

	b := [...]int{1, 2, 3} // [...] lets the compiler count for you -> [3]int
	fmt.Println(b)
}
```

The catch: arrays are **values**. Assigning one, or passing it to a function,
*copies every element*. That's rarely what you want, which is exactly why you'll
almost always use slices instead.

```go
package main

import "fmt"

func main() {
	a := [3]int{1, 2, 3}
	c := a       // full copy
	c[0] = 99
	fmt.Println(a, c) // [1 2 3] [99 2 3] — a is untouched
}
```

## 2. Slices: the workhorse, and what's inside one

A **slice** is a lightweight view *into* a backing array. Internally it's just
three words: a **pointer** to the first element, a **length** (`len`), and a
**capacity** (`cap` — how many elements exist from the pointer to the end of the
backing array).

```go
package main

import "fmt"

func main() {
	s := []int{2, 3, 5, 7, 11}
	fmt.Println(s, len(s), cap(s)) // [2 3 5 7 11] 5 5
}
```

Because a slice is a small header pointing at shared storage, copying a slice
header is cheap — and *does not* copy the underlying elements. Two slice values
can point at the same backing array. Hold that thought; section 6 is built on it.

## 3. `make`: allocating a slice (or map) up front

The slice literal in section 2 created the backing array for you. When you want
to allocate storage with a known length and/or capacity, use the built-in
`make`:

```go
package main

import "fmt"

func main() {
	s := make([]int, 3)    // len 3, cap 3 -> [0 0 0]
	t := make([]int, 0, 8) // len 0, cap 8 — empty now, room for 8 without re-allocating
	fmt.Println(s, len(s), cap(s)) // [0 0 0] 3 3
	fmt.Println(t, len(t), cap(t)) // [] 0 8
}
```

Pre-sizing capacity with `make([]T, 0, n)` is a real performance technique: if
you know roughly how many items you'll append, you avoid repeated re-allocation.

## 4. `append`: growing slices (and why nil is fine)

`append` adds elements to the end of a slice. If there's spare capacity it writes
in place; if not, it allocates a **new, larger** backing array, copies the old
elements over, and returns a slice pointing at the new storage. You **must**
assign the result back:

```go
package main

import "fmt"

func main() {
	s := make([]int, 0, 2)
	for i := 1; i <= 4; i++ {
		s = append(s, i)
		fmt.Println(s, "len", len(s), "cap", cap(s))
	}
	// cap grows (typically doubling) once you exceed it.
}
```

The **zero value of a slice is `nil`** — and a `nil` slice is perfectly usable.
You can `len` it (0), range over it (zero iterations), and crucially **append to
it**. You almost never need to "initialize" a slice before appending:

```go
package main

import "fmt"

func main() {
	var s []int            // nil slice, no make needed
	fmt.Println(s == nil, len(s)) // true 0
	s = append(s, 42)      // works: append allocates for you
	fmt.Println(s, s == nil)      // [42] false
}
```

## 5. `copy`: duplicating elements safely

`copy(dst, src)` copies `min(len(dst), len(src))` elements and returns the count.
It's how you make a genuinely independent slice when you need one:

```go
package main

import "fmt"

func main() {
	src := []int{1, 2, 3, 4}
	dst := make([]int, len(src)) // must have length to receive into
	n := copy(dst, src)
	dst[0] = 99
	fmt.Println(n, src, dst) // 4 [1 2 3 4] [99 2 3 4] — src untouched
}
```

## 6. Slicing semantics & the shared-backing-array gotcha

`s[low:high]` produces a new slice header over the **same backing array** —
length `high-low`, no element copy. So mutating through one slice is visible
through the other:

```go
package main

import "fmt"

func main() {
	full := []int{1, 2, 3, 4, 5}
	mid := full[1:3]   // view over elements 2,3 — shares storage with full
	mid[0] = 99
	fmt.Println(full)  // [1 99 3 4 5] — full changed too!
}
```

Now the classic `append` trap. `mid` above has `len 2` but `cap 4` (it can still
see to the end of `full`). Appending to it writes *into `full`'s storage*,
silently clobbering data you thought was safe:

```go
package main

import "fmt"

func main() {
	full := []int{1, 2, 3, 4, 5}
	mid := full[1:3]            // len 2, cap 4
	mid = append(mid, 100)      // fits in spare cap -> overwrites full[3]!
	fmt.Println(full)           // [1 2 3 100 5] — surprise mutation
	fmt.Println(mid)            // [2 3 100]
}
```

The fix when you need an *independent* slice: either `copy` (section 5), or use a
**full slice expression** `s[low:high:max]` to cap the capacity so any later
append is forced to allocate fresh storage:

```go
package main

import "fmt"

func main() {
	full := []int{1, 2, 3, 4, 5}
	mid := full[1:3:3]          // len 2, cap 3 — no spare capacity
	mid = append(mid, 100)      // cap exceeded -> new backing array
	fmt.Println(full)           // [1 2 3 4 5] — safe, untouched
	fmt.Println(mid)            // [2 3 100]
}
```

## 7. Maps: key/value lookups and the comma-ok idiom

A **map** associates keys with values. Create one with `make` (or a literal). The
**zero value of a map is `nil`** — and unlike a slice, a nil map is read-only:
reading gives zero values, but *writing to a nil map panics*. So you must `make`
before you write.

```go
package main

import "fmt"

func main() {
	m := make(map[string]int)
	m["nodes"] = 3
	m["replicas"] = 5

	// Plain read returns the zero value for a missing key — ambiguous!
	fmt.Println(m["missing"]) // 0 ... but does "missing" exist with value 0?

	// The comma-ok idiom disambiguates: ok is true only if the key is present.
	v, ok := m["replicas"]
	fmt.Println(v, ok) // 5 true
	_, ok = m["missing"]
	fmt.Println(ok)    // false
}
```

Delete with the built-in `delete` (a no-op if the key is absent), and note that
**map iteration order is randomized** by the runtime — never rely on it:

```go
package main

import "fmt"

func main() {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	delete(m, "b")
	fmt.Println(len(m)) // 2

	for k, v := range m {
		fmt.Println(k, v) // order differs run to run — sort keys if you need stability
	}
}
```

## 8. Zero values at a glance

| Type | Zero value | Usable as-is? |
|---|---|---|
| `[N]T` array | every element zeroed | yes |
| `[]T` slice | `nil` | yes — `len`, range, **append** all work |
| `map[K]V` | `nil` | read-only — **must `make` before writing** |

This is a recurring Go theme: the zero value is designed to be useful. Lean on
it, but remember the one exception — writing to a nil map.

## Do it yourself

1. Create a scratch program (`mkdir containers && cd containers && go mod init example.com/containers`) and a `main.go` you can run with `go run .`.
2. Reproduce section 6's shared-backing-array gotcha yourself. Print `len` and `cap` at each step until the surprise mutation is fully predictable to you.
3. Fix it two ways — once with `copy`, once with the `s[low:high:max]` full slice expression — and confirm `full` stays untouched.
4. Build a `map[string]int` word counter over a slice of strings, using the comma-ok idiom to decide whether to initialize or increment.
5. Read the Go blog [**Go Slices: usage and internals**](https://go.dev/blog/slices-intro) — this is *the* canonical explanation of the ptr/len/cap header and append growth.
6. Read [**Go maps in action**](https://go.dev/blog/maps) for delete, the comma-ok idiom, and why iteration order is random.
7. Work through the Tour of Go [**More types**](https://go.dev/tour/moretypes/1) section in your browser — do every slice and map exercise.

## Check yourself

You're ready to move on when you can answer, *without looking*:

- What three fields make up a slice header, and which of them does `cap` report?
- Why must you write `s = append(s, x)` instead of just `append(s, x)`?
- What is the zero value of a slice vs a map, and which one can you write to immediately?
- Walk through *why* `append`ing to a sub-slice can mutate the original slice — and name two ways to prevent it.
- What does the comma-ok idiom (`v, ok := m[k]`) let you distinguish that a plain `m[k]` cannot?

When all five feel obvious, commit this task on your **Roadmap** and take the
Step 1 quiz. Next lesson: **structs, methods, and interfaces.**

## Common interview gotchas

- **`nil` slice and empty slice are *not* the same to JSON or `==`** — both have `len 0`, but `var s []T` marshals to `null` while `[]T{}` marshals to `[]`. APIs and clients that distinguish the two will break; pick deliberately and don't compare slices with `==` (only `nil` comparison is legal).
- **A map is not safe for concurrent use — even read+write races** — concurrent writes (or a write racing a read) trigger a fatal `concurrent map writes` that no `recover` can catch; it kills the process. Guard with a `sync.RWMutex` or use `sync.Map`. The `-race` detector is how you catch it before prod.
- **Map iteration order is deliberately randomized every run** — relying on it (even accidentally, e.g. "the first key") produces tests that pass 90% of the time. Sort the keys when you need determinism; the runtime randomizes precisely to stop you depending on it.
- **A slice kept alive pins its *entire* backing array** — `huge[:1]` holds a 1-element view but the whole array can't be GC'd. Re-slicing a giant buffer to keep a few bytes is a classic memory leak; `copy` into a fresh small slice to release the original.
- **Taking the address of a map element is illegal; struct values in a map aren't addressable** — `&m[k]` won't compile, and `m[k].Field = x` fails for struct values. Store pointers (`map[K]*V`) or read-modify-write the whole value back.
