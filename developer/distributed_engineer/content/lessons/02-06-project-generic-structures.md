---
slug: project-generic-stack-queue-bst
step: 2
title: "Project: generic stack, queue & BST with tests"
summary: Build a type-safe generic stack, queue, and ordered BST in Go 1.21, each driven by table-driven tests that protect their invariants.
est_min: 420
position: 6
---

# Project: generic stack, queue & BST with tests

> **Step 2 · Data structures & algorithms · Week 2 · Project**
> Concept: *generics, invariants, test coverage*

This is the project where data structures stop being diagrams and become *code
you trust*. You'll implement three classics — a stack, a queue, and a binary
search tree — but **generically**, so one implementation works for `int`,
`string`, or any type you throw at it. Generics (added in Go 1.18, and the
default tool by 1.21) are how you write a container *once* and reuse it
everywhere without `interface{}` and runtime casts.

The harder, more valuable skill here is **testing invariants**: the quiet rules
each structure must always obey. A stack is LIFO. A queue is FIFO. A BST keeps
its in-order traversal sorted. Your tests exist to prove those rules can't be
broken.

## What you'll build

Three small, well-tested packages (or three files in one module — your choice):

- **`Stack[T any]`** — Last-In-First-Out. Methods: `Push(T)`, `Pop() (T, bool)`,
  `Len() int`. This one is special: it is **exactly the app's Step 2
  "generic-stack" Code Checker challenge**. Build it here, with your own tests,
  and you'll pass the checker on the first try.
- **`Queue[T any]`** — First-In-First-Out. Methods: `Enqueue(T)`,
  `Dequeue() (T, bool)`, `Len() int`.
- **`BST[T constraints.Ordered]`** — an ordered binary search tree with
  `Insert(T)`, `Contains(T) bool`, and `InOrder() []T` that returns elements in
  sorted order.

By the end you'll run:

```bash
go test ./...
# ok  	example.com/structures	0.004s
```

…with table-driven tests that cover the empty case, the happy path, and the
nasty edges.

## Why this matters

Generics are the difference between writing a stack five times (one per type) and
writing it *once*. Every serious Go codebase you'll touch — including the
consensus and storage layers later in this course — leans on generic containers
to stay both type-safe and DRY.

And invariant testing is the engineering muscle. A distributed log, a Raft state
machine, a write-ahead buffer — they all have invariants that, if violated even
once, corrupt everything downstream. Practising on a stack today is how you learn
to *think in invariants* before the stakes are a production cluster.

## 1. Set up the module

```bash
mkdir structures && cd structures
go mod init example.com/structures
```

For the BST you'll use the `constraints` package, which ships outside the
standard library in `golang.org/x/exp`:

```bash
go get golang.org/x/exp/constraints
```

Create `stack.go`, `queue.go`, `bst.go` and a `_test.go` file for each. Keep the
data structures pure — no printing, no I/O — so they're trivial to test and reuse.

## 2. The generic `Stack[T any]`

A stack is the simplest of the three, so it's where generics click. The `[T any]`
after the type name is a **type parameter**: `any` is the constraint (it means
"any type at all"), and `T` is the placeholder you use inside.

```go
package main

// Stack is a generic last-in-first-out collection.
type Stack[T any] struct {
	items []T
}

// Push adds v to the top of the stack.
func (s *Stack[T]) Push(v T) {
	s.items = append(s.items, v)
}

// Pop removes and returns the top item. The bool is false if the
// stack was empty (so the caller never mistakes a zero value for data).
func (s *Stack[T]) Pop() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	last := len(s.items) - 1
	v := s.items[last]
	s.items[last] = zero // avoid leaking a reference for GC
	s.items = s.items[:last]
	return v, true
}

// Len reports how many items the stack holds.
func (s *Stack[T]) Len() int {
	return len(s.items)
}
```

Three idioms worth absorbing:

- **`var zero T`** is how you produce the zero value of an unknown type. You
  can't write `return nil` or `return 0` — you don't know what `T` is.
- **Returning `(T, bool)`** is the Go way to say "this might have nothing." It
  spares the caller from inventing a sentinel value.
- **`s.items[last] = zero`** before truncating prevents the underlying array from
  holding onto a pointer the program no longer needs.

> This `Stack[T any]` — with `Push`, `Pop() (T, bool)`, and `Len` — *is* the
> Step 2 **"generic-stack" Code Checker challenge** in the app. Match these
> signatures exactly and the grader will go green.

## 3. Test the stack's LIFO invariant

A stack's invariant is **LIFO**: the last thing pushed is the first thing popped.
Go testing is built in — files end in `_test.go`, functions start with `Test`,
and idiomatic Go uses **table-driven tests**: a slice of cases you loop over.

```go
package main

import "testing"

func TestStackPushPop(t *testing.T) {
	cases := []struct {
		name string
		push []int
		want []int // expected pop order
	}{
		{"empty", nil, nil},
		{"single", []int{1}, []int{1}},
		{"lifo order", []int{1, 2, 3}, []int{3, 2, 1}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var s Stack[int]
			for _, v := range tc.push {
				s.Push(v)
			}
			for _, want := range tc.want {
				got, ok := s.Pop()
				if !ok {
					t.Fatalf("Pop() returned ok=false, wanted %d", want)
				}
				if got != want {
					t.Errorf("Pop() = %d, want %d", got, want)
				}
			}
			if _, ok := s.Pop(); ok {
				t.Errorf("Pop() on drained stack returned ok=true")
			}
		})
	}
}
```

Run it:

```bash
go test ./...
```

> **Your turn:** the table above only tests `int`. Add a second test —
> `TestStackString` — that pushes `"a", "b", "c"` and asserts they pop in
> reverse. Watch how the *same* `Stack` type works for a different `T` with zero
> code changes. That is the whole point of generics; prove it to yourself.

## 4. The generic `Queue[T any]`

A queue flips the rule: **FIFO**, first in, first out. The simplest correct
implementation appends to the back and removes from the front.

```go
package main

// Queue is a generic first-in-first-out collection.
type Queue[T any] struct {
	items []T
}

// Enqueue adds v to the back of the queue.
func (q *Queue[T]) Enqueue(v T) {
	q.items = append(q.items, v)
}

// Dequeue removes and returns the front item; ok is false if empty.
func (q *Queue[T]) Dequeue() (T, bool) {
	var zero T
	if len(q.items) == 0 {
		return zero, false
	}
	v := q.items[0]
	q.items[0] = zero
	q.items = q.items[1:]
	return v, true
}

func (q *Queue[T]) Len() int {
	return len(q.items)
}
```

> **Your turn:** write `queue_test.go`. Mirror the stack's table-driven style,
> but assert **FIFO**: push `1, 2, 3`, expect to dequeue `1, 2, 3` *in that
> order*. Add the empty case and the drain-past-empty case. The test should read
> almost like the stack test with one telling difference — the expected order.
>
> Bonus invariant to think about: re-slicing with `q.items[1:]` leaves the
> dropped element's backing memory allocated until the slice grows past it. For a
> long-lived, high-churn queue, how would you fix that? (Search "ring buffer" or
> a head/tail index pair — not required, but worth knowing it's a real concern.)

## 5. The ordered BST: `BST[T constraints.Ordered]`

A binary search tree needs to *compare* its elements, and not every type is
comparable with `<`. So we constrain `T` with **`constraints.Ordered`** — the set
of types that support `<`, `>`, `<=`, `>=` (all the integer, float, and string
types). This is a stricter constraint than `any`, and that's the point: it
unlocks the `<` operator inside the methods.

The BST invariant: for every node, all keys in the **left** subtree are smaller
and all keys in the **right** subtree are larger. Maintain that, and an in-order
traversal is automatically sorted.

```go
package main

import "golang.org/x/exp/constraints"

type node[T constraints.Ordered] struct {
	value       T
	left, right *node[T]
}

// BST is an ordered binary search tree.
type BST[T constraints.Ordered] struct {
	root *node[T]
}

// Insert adds v, keeping the search-tree invariant. Duplicates are ignored.
func (t *BST[T]) Insert(v T) {
	t.root = insert(t.root, v)
}

func insert[T constraints.Ordered](n *node[T], v T) *node[T] {
	if n == nil {
		return &node[T]{value: v}
	}
	switch {
	case v < n.value:
		n.left = insert(n.left, v)
	case v > n.value:
		n.right = insert(n.right, v)
	}
	// v == n.value: duplicate, do nothing
	return n
}

// Contains reports whether v is in the tree.
func (t *BST[T]) Contains(v T) bool {
	n := t.root
	for n != nil {
		switch {
		case v < n.value:
			n = n.left
		case v > n.value:
			n = n.right
		default:
			return true
		}
	}
	return false
}
```

`InOrder` is the method that *proves* the invariant: walk left, visit, walk
right, and the result comes out sorted.

```go
// InOrder returns every value in ascending order.
func (t *BST[T]) InOrder() []T {
	var out []T
	var walk func(*node[T])
	walk = func(n *node[T]) {
		if n == nil {
			return
		}
		walk(n.left)
		out = append(out, n.value)
		walk(n.right)
	}
	walk(t.root)
	return out
}
```

## 6. Test the BST invariant

The most powerful BST test is almost philosophical: **insert values in any
order, and `InOrder()` must always come back sorted.** That single property
catches a huge class of bugs.

```go
package main

import (
	"reflect"
	"testing"
)

func TestBSTInOrderIsSorted(t *testing.T) {
	cases := []struct {
		name   string
		insert []int
		want   []int
	}{
		{"empty", nil, nil},
		{"single", []int{42}, []int{42}},
		{"already sorted", []int{1, 2, 3}, []int{1, 2, 3}},
		{"reverse", []int{3, 2, 1}, []int{1, 2, 3}},
		{"scrambled with dup", []int{5, 1, 3, 1, 4, 2}, []int{1, 2, 3, 4, 5}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var bst BST[int]
			for _, v := range tc.insert {
				bst.Insert(v)
			}
			if got := bst.InOrder(); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("InOrder() = %v, want %v", got, tc.want)
			}
		})
	}
}
```

> **Your turn:** add a `TestBSTContains` table. Insert a known set, then assert
> `Contains` is `true` for every value you inserted and `false` for a value you
> didn't. Include the empty-tree case (`Contains` on an empty BST must be
> `false`). Then add one more invariant test of your own design — for example,
> that inserting the same value twice does **not** make it appear twice in
> `InOrder()`.

## Stretch goals

Pick at least one. This is where the learning compounds.

- **A custom comparison function.** Not every type implements `<` (think a
  `Person` you want sorted by age). Write a variant — `BSTFunc[T any]` whose
  constructor takes a `less func(a, b T) bool` — so it works on *any* type. This
  is exactly how `slices.SortFunc` is designed.
- **`Delete(v T)` for the BST.** The genuinely tricky case is deleting a node
  with two children (hint: replace it with its in-order successor). Write the
  test *first*, then make it pass — and re-run your "InOrder is sorted" test to
  prove you didn't break the invariant.
- **`Height()` and balance check.** Add a `Height() int` method, then a test
  showing that inserting already-sorted data produces a degenerate, list-shaped
  tree. That observation is the doorway to balanced trees (AVL / red-black).
- **Property-based testing.** Use `testing/quick` or just a loop with
  `math/rand` to insert hundreds of random values and assert `InOrder()` is
  sorted every time. Random tests find edges your hand-written tables miss.
- **Benchmark `Contains`.** Add a `Benchmark...` function and run
  `go test -bench=.` to feel how lookup cost grows with tree size.

## Free resources

- [**Learn Go with Tests**](https://quii.gitbook.io/learn-go-with-tests) — the
  best free, TDD-first Go book. The **Generics** chapter builds a generic stack
  almost identical to yours; read it alongside this project.
- [**`golang.org/x/exp/constraints`**](https://pkg.go.dev/golang.org/x/exp/constraints)
  — the docs for `Ordered` and the other constraint sets.
- [**`testing` package docs**](https://pkg.go.dev/testing) — `t.Run`, sub-tests,
  and the table-driven pattern you've been using.

## Done when

- [ ] `go test ./...` passes for all three structures, with **table-driven**
      tests you wrote yourself.
- [ ] `Stack[T any]` exposes `Push`, `Pop() (T, bool)`, and `Len`, and
      **passes the Step 2 "generic-stack" checker challenge in the app.**
- [ ] Tests cover the **empty case** and the **drain-past-empty** case for both
      stack and queue (each `Pop`/`Dequeue` returns `ok=false`).
- [ ] Your stack test runs against **at least two different types** (e.g. `int`
      and `string`) — proof the generic works.
- [ ] The BST has a test asserting `InOrder()` is **sorted regardless of insert
      order**, including a duplicate that does *not* appear twice.
- [ ] You attempted at least one stretch goal.

When the "generic-stack" checker turns green, commit this project on your
**Roadmap**. You've now written generic, invariant-tested data structures — the
exact discipline you'll lean on when the structures hold a replicated log instead
of a handful of ints. Next up: **Step 3, concurrency with goroutines and
channels.**

## Common interview gotchas

- **Return `(T, bool)`, not a panic or sentinel.** On empty Pop/Dequeue, produce the unknown type's zero with `var zero T` and signal absence with the bool — the caller never mistakes a zero value for data. And zero the popped slot before truncating so the backing array doesn't pin a stale pointer (GC hygiene).
- **Queue from two stacks is amortized O(1).** Use an `in` stack and an `out` stack; only transfer `in → out` when `out` is empty. Each element moves at most twice, so a single transferring dequeue is O(n) but the *amortized* cost is O(1) — a clean amortized-analysis talking point.
- **Thread safety is a contract, not a default.** Concurrent `append`/re-slice races (`go test -race` proves it). Adding a `sync.Mutex` everywhere fixes it but slows single-threaded users and can bottleneck under contention. Idiomatic Go often leaves locking to the caller or uses a channel — *state explicitly* whether the type is safe for concurrent use.
- **Expose iteration without leaking nodes.** Offer `InOrder() []T` (eager), a `ForEach(func(T) bool)` visitor (lazy, early-stop), or a Go 1.23 `iter.Seq[T]` for `range`. Avoid channel iterators unless you handle early-stop cancellation — they leak a goroutine otherwise.
- **`constraints.Ordered` unlocks `<`.** A generic BST can't use `<` under `[T any]`; constrain it to `constraints.Ordered`. For types without `<` (sort a `Person` by age), take a `less func(a, b T) bool` instead — exactly how `slices.SortFunc` is designed.
