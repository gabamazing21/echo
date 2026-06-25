---
slug: dsa-heaps-topk
step: 2
title: Heaps & the Top-K pattern
summary: The priority queue — how a binary heap works, Go's container/heap, and the Top-K pattern that ends countless interviews.
est_min: 360
position: 11
---

# Heaps & the Top-K pattern

> **Step 2 · Data structures & algorithms · Interview prep**
> Concept: *priority queues, Top-K, streaming selection*

A **heap** (priority queue) gives you the smallest-or-largest element in O(log n)
without sorting everything. It's the engine behind Dijkstra (last lesson), event
schedulers, and the single most reused interview pattern: **Top-K**.

## Why this matters

"Find the K largest / most frequent / closest" appears in interviews at every
big-tech company, and the heap-based answer beats the naive sort. Heaps also power
real systems — timers, rate limiters, Dijkstra, median-of-a-stream, merge of K
sorted streams. This is a Step 2 gap worth closing.

## 1. What a heap is

A **binary heap** is a complete binary tree (stored in an array) with the heap
property: in a **min-heap**, every parent ≤ its children, so the minimum is always
at the root. Operations:

- **Peek** min/max: O(1).
- **Push** / **Pop**: O(log n) (sift up / sift down to restore the property).
- **Build** from n items: O(n).

It does *not* keep everything sorted — only the root is guaranteed. That partial
order is exactly why it's cheaper than a full sort when you only need extremes.

## 2. Go's container/heap

Go's stdlib gives you the algorithm; you supply the storage by implementing the
interface (`Len, Less, Swap, Push, Pop`):

```go
import "container/heap"

type MinHeap []int
func (h MinHeap) Len() int            { return len(h) }
func (h MinHeap) Less(i, j int) bool  { return h[i] < h[j] } // '>' for a max-heap
func (h MinHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *MinHeap) Push(x any)         { *h = append(*h, x.(int)) }
func (h *MinHeap) Pop() any           { old := *h; n := len(old); v := old[n-1]; *h = old[:n-1]; return v }

// usage:
h := &MinHeap{}; heap.Init(h)
heap.Push(h, 5); heap.Push(h, 1)
min := heap.Pop(h) // 1
```

Flip `Less` to get a max-heap. The `Push`/`Pop` you implement are the raw slice ops;
`heap.Push`/`heap.Pop` are the ones you *call* (they sift).

## 3. The Top-K pattern (the money move)

"Return the K largest (or most frequent) of n items." Naive sort is O(n log n). With
a heap it's **O(n log K)** — keep a **min-heap of size K**:

- Push each item; if the heap grows past K, **pop the smallest**.
- After processing all n, the heap holds the K largest.

```go
// K largest with a size-K min-heap
for _, x := range nums {
	heap.Push(h, x)
	if h.Len() > k { heap.Pop(h) } // evict the smallest so far
}
// h now contains the K largest
```

Counter-intuition to state in the interview: to keep the K **largest**, use a
**min**-heap (so you can cheaply discard the smallest of your current top-K). For
"top-K frequent", count first, then heap by frequency — your in-app **Top K
Frequent** challenge.

## 4. More heap classics

- **Kth largest element** (LeetCode #215) — Top-K with K-size heap, or quickselect
  for O(n) average.
- **Merge K sorted lists** (#23) — push the heads of all K lists; pop the min, push
  its successor. O(N log K).
- **Find median from a data stream** (#295) — two heaps (a max-heap of the low half,
  min-heap of the high half) balanced so the median is at the tops. A favorite hard
  question.
- **Task scheduler / meeting rooms** — heap by next-available time.

## 5. When NOT to use a heap

If you need the items *fully sorted*, just sort (O(n log n)) — a heap doesn't save
you. If K is tiny and fixed (say 1–2), a linear scan tracking the best is simpler.
If counts are bounded, **bucket sort** can do Top-K in O(n), beating the heap.

## Do it yourself (≈ 6 hrs)

1. Read [**NeetCode — Heap / Priority Queue**](https://neetcode.io/roadmap) and the [**container/heap docs**](https://pkg.go.dev/container/heap).
2. Solve: Kth Largest Element, Top K Frequent Elements (in-app checker below),
   K Closest Points to Origin, Merge K Sorted Lists, Find Median from Data Stream.
3. Implement a min-heap from scratch once (sift up/down) so `container/heap` isn't a
   black box.

## Check yourself

- What's the heap property, and which operations are O(1) vs O(log n)?
- To keep the K *largest* items, do you use a min-heap or a max-heap — and why?
- What's the time complexity of Top-K with a size-K heap vs sorting?
- How do two heaps give you a streaming median?
- When is sorting or bucket sort the better choice over a heap?

Next bonus lesson: **Backtracking** — systematically exploring all configurations.

## Common interview gotchas

- **To keep the K *largest*, use a size-K *min*-heap** so you can cheaply evict the smallest when the heap overflows — a max-heap there forces you to hold all n elements or sort.
- **The `Push`/`Pop` you implement are not the ones you call.** Your methods are raw slice ops on the underlying storage; you must call `heap.Push`/`heap.Pop` (the package functions) to actually sift — calling your own methods skips the reordering and corrupts the heap.
- **Building a heap from n items is O(n), not O(n log n).** `heap.Init` (Floyd's bottom-up sift-down) is linear; inserting one-by-one with n pushes is the O(n log n) version — don't quote the wrong bound.
- **Two-heaps median must stay *balanced* in size.** After every insert, rebalance so the heaps differ by at most one — forgetting this puts the median in the wrong place even though both heap properties hold.
- **A heap is not sorted.** Only the root is guaranteed; iterating the backing array does *not* yield sorted order — to get sorted output you must pop repeatedly (which is just heapsort).
