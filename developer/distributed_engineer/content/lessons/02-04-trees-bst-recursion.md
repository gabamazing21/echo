---
slug: dsa-trees-bst-recursion
step: 2
title: Trees, binary search trees & recursion
summary: Build the recursion mental model, then implement a binary search tree with insert, search, and in-order plus BFS traversals.
est_min: 420
position: 4
---

# Trees, binary search trees & recursion

> **Step 2 · Data structures & algorithms · Week 2**
> Concept: *recursion, tree traversal, BST invariants*

So far your data has been *flat* — slices and maps line everything up in a row.
But a huge amount of the world is **hierarchical**: a filesystem, an org chart, a
parsed expression, the index pages of a database. The data structure for
hierarchy is the **tree**, and the natural way to walk a tree is **recursion**.
Master both together and a whole class of problems stops being scary.

## Why this matters

Trees are everywhere in the systems you're going to build. When you reach
**Step 6 (storage engines)** you'll meet the **B-tree** — the balanced,
high-fan-out tree that virtually every relational database uses to index data on
disk so a lookup touches a handful of pages instead of scanning millions of rows.
That's just a tree with the invariants from this lesson, generalised. Recursion,
meanwhile, is the thinking tool you'll use for traversals, divide-and-conquer
sorts, parsers, and the consensus log structures later in the roadmap. This
lesson is foundational — don't rush it.

## 1. Recursion: the mental model

A **recursive** function is one that calls itself on a *smaller* version of the
same problem. Every correct recursion has two parts:

- A **base case** — a problem small enough to answer directly, with *no* further
  recursion. This is what stops the process.
- A **recursive case** — reduce the problem toward the base case, call yourself,
  and combine the result.

The classic warm-up is factorial:

```go
package main

import "fmt"

func factorial(n int) int {
	if n <= 1 { // base case: 0! and 1! are 1 — stop here
		return 1
	}
	return n * factorial(n-1) // recursive case: shrink toward the base
}

func main() {
	fmt.Println(factorial(5)) // 120
}
```

Trace it: `factorial(5)` waits on `factorial(4)`, which waits on `factorial(3)`,
down to `factorial(1)` which returns `1` immediately. Then the answers multiply
back up. If you ever write a recursion with **no base case** (or one you never
reach), it calls itself forever and Go eventually crashes with a *stack
overflow*. The base case is not optional.

### The call stack

Each call gets its own **stack frame** holding its arguments and local variables.
Calls pile frames *on*; returns pop frames *off*. The "shape" of a recursion is
literally the shape of the stack over time. Holding that picture in your head —
frames pushed on the way down, popped on the way back up — is what makes
recursive code readable instead of magical.

## 2. Binary trees: the structure

A **binary tree** is made of **nodes**. Each node holds a value and up to two
children: a **left** child and a **right** child. The top node is the **root**; a
node with no children is a **leaf**. A missing child is just `nil`.

```go
type Node struct {
	Val   int
	Left  *Node
	Right *Node
}
```

Notice the type is **self-referential**: a `Node` points to other `Node`s. That
recursive *shape* is exactly why recursive *functions* fit trees so naturally —
the data is defined in terms of itself, so the code can be too.

## 3. Binary search trees & their invariant

A **binary search tree (BST)** is a binary tree with one rule — the **BST
invariant** — that must hold at *every* node:

> For any node, **every** value in its **left** subtree is **less than** the
> node's value, and **every** value in its **right** subtree is **greater
> than** it.

That single ordering rule is what makes a BST useful: at each node you can throw
away half the remaining tree. To find a value you compare, then go left or right
— never both. To insert, you walk the same path to the spot where the value
*would* be and attach a new leaf there. Both operations follow one root-to-leaf
path, so on a **balanced** tree they cost **O(log n)**.

## 4. A BST in Go: insert & search

Here is a complete, runnable program. `Insert` and `Search` are written
recursively — each returns by working on a *subtree*, the smaller version of the
problem.

```go
package main

import "fmt"

type Node struct {
	Val   int
	Left  *Node
	Right *Node
}

// Insert returns the (possibly new) subtree root with val added.
// The "return the root" pattern lets the parent re-link cleanly.
func Insert(root *Node, val int) *Node {
	if root == nil { // base case: found the empty spot — create the leaf
		return &Node{Val: val}
	}
	if val < root.Val {
		root.Left = Insert(root.Left, val) // recurse left, re-attach
	} else if val > root.Val {
		root.Right = Insert(root.Right, val) // recurse right, re-attach
	}
	// val == root.Val: already present, do nothing (no duplicates here)
	return root
}

// Search reports whether val exists in the tree.
func Search(root *Node, val int) bool {
	if root == nil { // base case: ran off the tree — not found
		return false
	}
	if val == root.Val {
		return true
	}
	if val < root.Val {
		return Search(root.Left, val) // discard the right half
	}
	return Search(root.Right, val) // discard the left half
}

func main() {
	var root *Node // nil tree to start
	for _, v := range []int{8, 3, 10, 1, 6, 14, 4, 7, 13} {
		root = Insert(root, v)
	}

	fmt.Println(Search(root, 7))  // true
	fmt.Println(Search(root, 99)) // false
}
```

Two things to internalise. First, `Insert(root.Left, val)` followed by
`root.Left = ...` is the standard recursive-tree pattern: you recurse on a child
and **re-assign the returned subtree**, which handles "this child was `nil`,
create it" without special cases. Second, each call to `Search` *halves* the
search space — that's the BST payoff.

## 5. A note on the delete invariant

Deletion is the trickiest BST operation because you must keep the invariant
intact after removing a node. Three cases:

- **Leaf** (no children): just detach it — set the parent's pointer to `nil`.
- **One child**: splice the node out by linking its parent straight to its only
  child.
- **Two children**: you can't simply remove it. Replace its value with its
  **in-order successor** (the smallest value in the right subtree — the
  left-most node over there), then delete *that* successor node, which by
  construction has at most one child. The successor is the next value in sorted
  order, so the ordering rule still holds everywhere.

You don't need to memorise delete code today, but understand *why* the
two-child case reaches for the in-order successor: it's the only value that can
sit in that slot without breaking the invariant on either side.

## 6. Traversals: visiting every node

Walking a whole tree is a **traversal**. The three depth-first orders differ only
in *when* you visit the current node relative to its children:

| Order | Visit sequence | Common use |
|---|---|---|
| **Pre-order** | node, left, right | copy/serialise a tree |
| **In-order** | left, node, right | **sorted output** of a BST |
| **Post-order** | left, right, node | free/aggregate children first |

The killer property: an **in-order** traversal of a BST visits values in
**sorted ascending order**. That falls straight out of the invariant — everything
left is smaller, everything right is larger, so "left, me, right" is "smaller, me,
larger" all the way down.

```go
// InOrder appends values left, node, right -> sorted for a BST.
func InOrder(root *Node, out *[]int) {
	if root == nil { // base case: nothing to visit
		return
	}
	InOrder(root.Left, out)        // all smaller values first
	*out = append(*out, root.Val)  // then this node
	InOrder(root.Right, out)       // then all larger values
}
```

Call it like this (continuing the `main` from section 4):

```go
var sorted []int
InOrder(root, &sorted)
fmt.Println(sorted) // [1 3 4 6 7 8 10 13 14] — sorted, for free
```

Swap the order of the three lines and you get pre-order or post-order — same
recursion, different visit point.

## 7. Level-order (BFS) with a queue

Depth-first traversals dive *down* first. **Level-order** — also called
**breadth-first search (BFS)** — visits the tree *layer by layer*: the root, then
everything one level down, then the next level, and so on. There's no clean
recursive form for this; instead you use an explicit **queue** (FIFO). You dequeue
a node, visit it, then enqueue its children — so nodes come out in the exact order
they were discovered.

```go
// BFS returns values level by level, top to bottom, left to right.
func BFS(root *Node) []int {
	var out []int
	if root == nil {
		return out
	}
	queue := []*Node{root} // a slice used as a FIFO queue
	for len(queue) > 0 {
		node := queue[0]   // front of the queue
		queue = queue[1:]  // dequeue
		out = append(out, node.Val)
		if node.Left != nil {
			queue = append(queue, node.Left) // enqueue children
		}
		if node.Right != nil {
			queue = append(queue, node.Right)
		}
	}
	return out
}
```

For the tree built in section 4, `BFS(root)` yields
`[8 3 10 1 6 14 4 7 13]` — every node at depth 0, then depth 1, then depth 2.
Note the difference in *machinery*: depth-first traversals lean on the **call
stack** (recursion), while breadth-first leans on an **explicit queue**. That
stack-vs-queue distinction is the whole difference between DFS and BFS, and it
shows up again on graphs in the next lesson.

## 8. Balancing: why unbalanced BSTs degrade

Every "O(log n)" claim above assumed the tree is **balanced** — roughly the same
depth on every path. But a plain BST's shape depends entirely on *insertion
order*. Insert already-sorted data and watch what happens:

```go
var root *Node
for _, v := range []int{1, 2, 3, 4, 5} { // ascending!
	root = Insert(root, v)
}
// Every value is larger than the last, so each goes right:
//   1 -> 2 -> 3 -> 4 -> 5  (a straight line)
```

That tree is a glorified linked list. Its height is `n`, not `log n`, so search
and insert degrade to **O(n)** — the worst case. This is the fundamental weakness
of a naive BST.

The fix is a **self-balancing** tree that rearranges itself on insert/delete to
keep its height near `log n` no matter the input order — **AVL trees** and
**red-black trees** do this in memory. On disk, databases use the **B-tree**
(and B+tree), a balanced tree with many keys per node to minimise page reads.
You'll implement and reason about B-trees properly in **Step 6**; for now, just
hold the key insight: *a tree is only as fast as it is balanced.*

## Do it yourself

1. Create a scratch program (`mkdir trees && cd trees && go mod init example.com/trees`) and paste the BST from section 4 into `main.go`. Run it with `go run .`.
2. Add the `InOrder` and `BFS` functions and print both for the same tree. Confirm in-order is sorted and BFS is layer-by-layer.
3. By hand, draw the tree that results from inserting `8, 3, 10, 1, 6, 14`. Then trace `Search(root, 6)` on paper — which nodes does it touch, and which whole subtree does it skip?
4. Insert the values **in sorted order** (section 8) and print the BFS. See the "straight line" degenerate tree for yourself, then explain out loud why its search is O(n).
5. Write a recursive `Height(root *Node) int` (base case: a `nil` node has height 0; otherwise `1 + max(height of children)`). Use it to measure your balanced tree vs the degenerate one.
6. Read the **recursion** and **trees / tries** sections of the [**NeetCode roadmap**](https://neetcode.io/roadmap) — it sequences the canonical practice problems in the right order.
7. Work through [**A Tour of Go**](https://go.dev/tour/) up to and including the *Methods and pointers* material so the `*Node` pointer plumbing in this lesson feels routine.

## Check yourself

You're ready to move on when you can answer, *without looking*:

- What are the two required parts of any recursion, and what happens if the base case is missing or unreachable?
- State the BST invariant precisely. Why does it let search discard half the tree at each step?
- Why does an **in-order** traversal of a BST produce sorted output?
- What data structure powers BFS, and how does that differ from what powers a recursive depth-first traversal?
- How can a BST degrade to O(n), and what category of tree fixes it (and where does that resurface in this roadmap)?

When all five feel obvious, commit this task on your **Roadmap** and take the
Step 2 quiz. Next lesson: **graphs — representations, BFS & DFS.**

## Common interview gotchas

- **Validating a BST: check whole subtrees, not just children.** The most common wrong answer compares each node only to its immediate children, which passes invalid trees. Recurse carrying `(min, max)` bounds that tighten as you descend — or check that the **in-order traversal is strictly increasing**.
- **BST delete with two children.** Replace the node's value with its **in-order successor** (smallest in the right subtree), then delete that successor (which has at most one child). It's the only value that preserves the ordering on both sides. The predecessor (max of left subtree) is the symmetric, equally valid choice.
- **Iterative traversal beats recursion on deep trees.** Recursion is O(h) call-stack space — O(n) on a skewed tree, which can **stack-overflow**. Know the explicit-stack in-order walk; recursion *is* a stack, and saying so signals depth.
- **In-order of a BST is sorted.** That property is the trigger for "kth smallest", "validate BST", and "two-sum in a BST" — recognize it rather than re-deriving each time.
- **O(log n) assumes *balanced*.** Insert sorted data into a plain BST and it degenerates to a linked list (height n → O(n)). The fix is a self-balancing tree (AVL / red-black in memory; **B-tree** on disk). A tree is only as fast as it is balanced.
