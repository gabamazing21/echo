---
slug: dsa-linear-structures
step: 2
title: Arrays, linked lists, stacks & queues
summary: Build the core linear data structures in Go — dynamic arrays, linked lists, stacks and queues — and reason about their costs.
est_min: 480
position: 2
---

# Arrays, linked lists, stacks & queues

> **Step 2 · Data structures & algorithms · Week 1**
> Concept: *pointers, references, dynamic structures*

You can write Go now. This lesson is where you start thinking like an engineer
who *chooses* data structures on purpose. Every program you've ever used —
including the infrastructure you'll build later — is held together by four
**linear structures**: the dynamic array, the linked list, the stack, and the
queue. Learn how each one stores data, what each operation *costs*, and when to
reach for which. This is the vocabulary the rest of the algorithms step is
written in.

## Why this matters

Real infra code is full of these structures wearing different clothes. A **log
buffer** that holds the last N lines is a slice (dynamic array) or a ring. A
**request queue** in front of a worker pool is a FIFO. An **undo history** or a
call stack is a LIFO. A connection's **write buffer** drains in order — FIFO
again. When a service falls over under load, the root cause is very often the
*wrong* structure: an O(n) operation in a hot loop, or a queue that's secretly a
slice doing an O(n) shift on every dequeue. Knowing the cost of each operation —
not memorising it, *understanding* it — is what lets you read a profiler and know
where to look. That's the whole game.

## 1. Dynamic arrays: the slice you already know

A **dynamic array** is a contiguous block of memory that grows on demand. In Go
that's the **slice**, backed by an array, carrying a `len` and a `cap`. You met
slices in Step 1; here we name their costs.

```go
package main

import "fmt"

func main() {
	logs := make([]string, 0, 4) // pre-size: 4 slots, no reallocation yet
	logs = append(logs, "node up")     // O(1) amortised
	logs = append(logs, "leader=3")    // O(1) amortised
	fmt.Println(logs[0])               // O(1) random access by index
	fmt.Println(len(logs), cap(logs))  // 2 4
}
```

Why "amortised O(1)"? Most appends just write into spare capacity — constant
time. Occasionally the array is full, so Go allocates a bigger one (typically
doubling) and copies everything — that single append is O(n). But because the
copies get rarer as the array grows, the *average* cost per append stays
constant. That is the central trade of a dynamic array.

| Operation | Cost | Why |
|---|---|---|
| Index `s[i]` | O(1) | Contiguous memory, direct offset |
| Append at end | O(1) amortised | In place, except on resize |
| Insert/delete at front or middle | O(n) | Every later element must shift |

That last row is the catch. Removing the front element of a slice
(`s = s[1:]` leaks, `s = append(s[:0], s[1:]...)` shifts) is O(n). When you need
cheap removal from the front, you want a different structure.

## 2. Pointers: the tool that makes dynamic structures possible

A **pointer** holds the *address* of a value rather than the value itself. `&x`
takes the address of `x`; `*p` follows ("dereferences") the pointer to reach the
value. This is the mechanism every node-based structure is built from: a node
holds a pointer to the *next* node.

```go
package main

import "fmt"

func main() {
	x := 10
	p := &x      // p points at x
	*p = 20      // write through the pointer
	fmt.Println(x) // 20 — we changed x without naming it

	var nilp *int            // zero value of a pointer is nil
	fmt.Println(nilp == nil) // true — "points at nothing"
}
```

The **zero value of any pointer is `nil`** — it points at nothing. That nil is
how we'll mark "the end of the list" in the structures below. Dereferencing a nil
pointer panics, so node code is really a discipline of "is this pointer nil yet?"

## 3. Singly linked list: nodes joined by pointers

A **linked list** stores each element in its own **node**, and each node holds a
pointer to the next one. There's no contiguous block and no index — to reach the
5th element you walk from the head through four `next` pointers. In exchange,
inserting at the front is O(1): you just make a new node point at the old head.

```go
package main

import "fmt"

// Node is one element plus a pointer to the next node.
type Node struct {
	Val  int
	Next *Node
}

// List keeps the head; an empty list has head == nil.
type List struct {
	Head *Node
}

// PushFront inserts a value at the head in O(1).
func (l *List) PushFront(v int) {
	l.Head = &Node{Val: v, Next: l.Head}
}

// Append walks to the tail and links a new node — O(n) without a tail pointer.
func (l *List) Append(v int) {
	n := &Node{Val: v}
	if l.Head == nil {
		l.Head = n
		return
	}
	cur := l.Head
	for cur.Next != nil { // walk until the last node
		cur = cur.Next
	}
	cur.Next = n
}

// Print walks every node from head to nil.
func (l *List) Print() {
	for cur := l.Head; cur != nil; cur = cur.Next {
		fmt.Printf("%d -> ", cur.Val)
	}
	fmt.Println("nil")
}

func main() {
	l := &List{}
	l.Append(1)
	l.Append(2)
	l.PushFront(0)
	l.Print() // 0 -> 1 -> 2 -> nil
}
```

Read the loop conditions carefully — that walk-until-`nil` pattern *is* linked
list code. Notice the costs are the mirror image of a slice: front insert is
O(1), but random access by position is O(n) because there's no index, only
pointers to follow.

A **doubly linked list** adds a `Prev` pointer to each node (and usually a `Tail`
on the list). That buys O(1) removal of a *known* node and O(1) append, at the
cost of more bookkeeping on every insert. Go's standard library ships one as
[`container/list`](https://pkg.go.dev/container/list) — worth reading once you've
written your own.

| Operation | Slice | Singly linked list |
|---|---|---|
| Index / random access | O(1) | O(n) |
| Insert at front | O(n) | **O(1)** |
| Append at end | O(1) amortised | O(n) (O(1) with a tail pointer) |
| Memory layout | contiguous, cache-friendly | scattered nodes, pointer overhead |

That last row matters more than it looks: slices keep elements next to each other
in memory, so the CPU cache loves them. Linked lists scatter nodes across the
heap. In practice a slice often *wins* even where Big-O says the list should —
another reason to measure, not assume.

## 4. Stack (LIFO): last in, first out

A **stack** only lets you touch one end. You **push** onto the top and **pop**
from the top — the *last* thing in is the *first* thing out (LIFO). Think of the
call stack, an undo history, or a depth-first traversal's pending work. A slice
makes a perfect stack because append/remove at the *end* are both O(1).

```go
package main

import "fmt"

// Stack is LIFO, backed by a slice (the end is the top).
type Stack[T any] struct {
	items []T
}

// Push adds to the top — O(1) amortised.
func (s *Stack[T]) Push(v T) {
	s.items = append(s.items, v)
}

// Pop removes and returns the top. ok is false if the stack is empty.
func (s *Stack[T]) Pop() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	top := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1] // shrink by one — O(1)
	return top, true
}

func (s *Stack[T]) Len() int { return len(s.items) }

func main() {
	var s Stack[string]
	s.Push("a")
	s.Push("b")
	s.Push("c")
	for s.Len() > 0 {
		v, _ := s.Pop()
		fmt.Print(v, " ") // c b a — reverse of insertion order
	}
	fmt.Println()
}
```

This uses Go **generics** (`[T any]`) so one stack works for any element type.
Both operations work at the slice's end, which is exactly why they're O(1) — no
shifting. Returning `(T, bool)` instead of panicking on empty is the idiomatic
way to signal "nothing there."

## 5. Queue (FIFO): first in, first out

A **queue** is the opposite discipline: you **enqueue** at the back and
**dequeue** from the front — *first* in, *first* out (FIFO). This is the shape of
almost every request pipeline: jobs are served in arrival order. The naive
slice-backed queue has a trap, so read the dequeue carefully.

```go
package main

import "fmt"

// Queue is FIFO. Enqueue at the back, dequeue from the front.
type Queue[T any] struct {
	items []T
}

// Enqueue appends to the back — O(1) amortised.
func (q *Queue[T]) Enqueue(v T) {
	q.items = append(q.items, v)
}

// Dequeue removes from the front. Re-slicing (items[1:]) is O(1) here, but
// the dropped front elements are not reclaimed until the slice is replaced.
func (q *Queue[T]) Dequeue() (T, bool) {
	var zero T
	if len(q.items) == 0 {
		return zero, false
	}
	front := q.items[0]
	q.items = q.items[1:] // advance the view past the consumed element
	return front, true
}

func (q *Queue[T]) Len() int { return len(q.items) }

func main() {
	var q Queue[int]
	q.Enqueue(1)
	q.Enqueue(2)
	q.Enqueue(3)
	for q.Len() > 0 {
		v, _ := q.Dequeue()
		fmt.Print(v, " ") // 1 2 3 — same order they arrived
	}
	fmt.Println()
}
```

The subtle cost: `q.items = q.items[1:]` looks O(1), and it is — but it leaves
the consumed elements stranded in the backing array, which never shrinks. A
long-lived queue that enqueues and dequeues forever will hold memory for every
element it ever saw. The shift-everything alternative
(`append(q.items[:0], q.items[1:]...)`) reclaims memory but makes dequeue O(n).

The real fix is a **ring buffer** (circular buffer): a fixed-size slice with a
`head` and `tail` index that wrap around with modulo arithmetic. Both ends are
O(1) *and* memory is bounded — which is exactly why production log buffers and
bounded request queues use one. Implementing it is your stretch goal below.

| Structure | Add | Remove | Order out |
|---|---|---|---|
| Stack (LIFO) | push top — O(1) | pop top — O(1) | reverse of insertion |
| Queue (FIFO) | enqueue back — O(1) | dequeue front — O(1)* | same as insertion |

\* O(1) per call with re-slicing, but unbounded memory until you switch to a ring
buffer.

## 6. Choosing the right one

The decision is almost always about *which operation is hot*:

- Need fast **random access by index** and cache-friendly iteration? **Slice.**
- Need cheap **insert/remove at the front** (or splicing mid-list)? **Linked list.**
- Need **most-recent-first** / nested undo / DFS work tracking? **Stack.**
- Need **arrival-order** processing / a work pipeline? **Queue** (ring buffer if it's long-lived and bounded).

When in doubt, start with a slice — it's the simplest and usually the fastest —
and switch only when a profiler or a clear access pattern tells you to.

## Do it yourself

1. Create a scratch module (`mkdir linear && cd linear && go mod init example.com/linear`) and a `main.go` you can run with `go run .`.
2. Type out the singly linked list from section 3 *from memory*, then add a `Reverse()` method that flips the `Next` pointers in place. This is *the* classic pointer exercise — do not skip it.
3. Add a `Tail *Node` field to the list so `Append` becomes O(1). Confirm you keep `Tail` correct in `PushFront` too.
4. Build the generic stack and queue. Write a tiny program that uses the stack to check whether a string of brackets `()[]{}` is balanced.
5. **Stretch:** replace the slice-backed queue with a fixed-size **ring buffer** — a `[]T`, a `head`, a `tail`, and a `count`, with both indices wrapping via `% len(buf)`. This is a real log-buffer.
6. Read the *Grokking Algorithms* chapters on arrays vs linked lists and on stacks/queues — [**Grokking Algorithms**](https://www.manning.com/books/grokking-algorithms). It draws the trade-offs better than any prose.
7. Work through the slices, structs, pointers, and generics examples on [**Go by Example**](https://gobyexample.com/) — short, runnable, and exactly the patterns above.

## Check yourself

You're ready to move on when you can answer, *without looking*:

- Why is append to a slice "amortised O(1)" and not just O(1)? What single operation is the expensive one?
- A linked list and a slice have opposite cost profiles for two operations — name both operations and say which structure wins each.
- What does it mean for a pointer to be `nil`, and how does a linked list use nil?
- Explain LIFO vs FIFO in one sentence each, and give a real infra example of each.
- What is the hidden cost of a slice-backed queue that dequeues with `items[1:]`, and what structure fixes it?

When all five feel obvious, commit this task on your **Roadmap** and take the
Step 2 quiz. Next lesson: **hash tables & trees.**

## Common interview gotchas

- **Reversing a linked list: save `next` first.** The iterative reverse carries `prev`/`cur`/`next` and *must* stash `cur.Next` before flipping it, or you lose the rest of the list. The recursive version is O(n) **space** (call stack) — interviewers usually want the O(1)-space iterative one.
- **The dummy-head trick.** Merging or building lists, use a sentinel head node. It removes the "which node is first?" special case so you don't write branchy head-handling code. `valid-parentheses` has the mirror trick: the stack must be **empty at the end** to catch unclosed openers.
- **Slice-queue memory leak.** `q = q[1:]` is O(1) but only advances the header — the backing array pins every element ever enqueued (and any pointers they hold), so a long-lived queue leaks unboundedly. Fix with a **ring buffer** (head/tail/count + modulo): O(1) both ends *and* bounded memory.
- **LRU cache needs a *doubly* linked list.** O(1) get/put = hash map (lookup) + doubly linked list (recency). Singly linked won't do — you must unlink an arbitrary accessed node in O(1), which needs its `prev` pointer. Use a sentinel head/tail to kill null-checks.
- **Slices beat lists more often than Big-O suggests.** Contiguous memory is cache-friendly; linked-list nodes scatter across the heap. Default to a slice and switch only when a profiler or a clear front-insert pattern demands it.
