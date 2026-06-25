---
slug: dsa-sorting-binary-search
step: 2
title: Sorting & binary search
summary: Divide and conquer in practice — merge sort, quicksort, stability, Go's sort.Slice, and the binary search pattern you'll go pass.
est_min: 360
position: 5
---

# Sorting & binary search

> **Step 2 · Data structures & algorithms · Week 2**
> Concept: *divide and conquer — and when to use which*

Two of the oldest, most-studied problems in computing are *put data in order* and
*find a thing in ordered data*. They matter here for two reasons. First, sorting
and binary search are the canonical introduction to **divide and conquer** — the
strategy of splitting a problem into smaller copies of itself, solving those, and
combining the answers. Second, the binary search you write at the end of this
lesson is **literally the app's Step 2 "binary-search" Code Checker challenge**.
By the end you'll have written code you can paste straight into the checker and
watch go green.

## Why this matters

Distributed systems sort and search constantly: ordering log entries by term and
index before replication, merging sorted streams from many shards, finding the
right peer in a sorted membership ring, range-scanning a sorted key space in an
LSM-tree. The *ideas* here — split, recurse, combine; halve the search space each
step — show up again at the systems scale. And the off-by-one bugs you learn to
avoid in a 12-line binary search are the exact bugs that take down production
range queries. Small problem, large lesson.

## 1. Divide and conquer

The pattern has three moves:

1. **Divide** — split the input into smaller subproblems.
2. **Conquer** — solve each subproblem (usually by recursing).
3. **Combine** — stitch the sub-answers into the full answer.

Merge sort divides an array in half, sorts each half, then *merges*. Quicksort
partitions around a pivot, then sorts each side. Binary search divides the search
range in half and *throws one half away*. Same skeleton, different bones.

A quick way to reason about the cost: if each step halves the problem, you get
about **log₂ n** levels. If each level also touches all *n* elements (as merging
does), the total is **O(n log n)**. If each level only touches *one* element (as
binary search does), the total is **O(log n)**. Hold those two shapes in your head.

## 2. Merge sort — stable, O(n log n) worst case

Merge sort's superpower is *predictability*: it is **O(n log n) in the worst
case**, every time, and it is **stable** (equal elements keep their original
relative order). The cost is **O(n) extra memory** for the merge buffer.

```go
package main

import "fmt"

// mergeSort returns a new sorted slice; it does not mutate its input.
func mergeSort(a []int) []int {
	if len(a) <= 1 { // base case: 0 or 1 element is already sorted
		return a
	}
	mid := len(a) / 2
	left := mergeSort(a[:mid])  // divide + conquer the left half
	right := mergeSort(a[mid:]) // divide + conquer the right half
	return merge(left, right)   // combine
}

// merge fuses two already-sorted slices into one sorted slice.
func merge(left, right []int) []int {
	out := make([]int, 0, len(left)+len(right))
	i, j := 0, 0
	for i < len(left) && j < len(right) {
		// "<=" (not "<") is what makes the sort STABLE: when values are equal,
		// the element from the left (earlier) slice is taken first.
		if left[i] <= right[j] {
			out = append(out, left[i])
			i++
		} else {
			out = append(out, right[j])
			j++
		}
	}
	out = append(out, left[i:]...)  // drain whichever side has leftovers
	out = append(out, right[j:]...)
	return out
}

func main() {
	in := []int{5, 2, 9, 2, 1, 5, 6}
	fmt.Println(mergeSort(in)) // [1 2 2 5 5 6 9]
}
```

Run it. Trace the recursion once on paper for `[5 2 9]`: it splits to `[5]` and
`[2 9]`, the right splits again to `[2]` and `[9]`, those merge to `[2 9]`, and
the top merges `[5]` with `[2 9]` to `[2 5 9]`. That hand-trace is worth more than
re-reading the code.

## 3. Quicksort — in-place, O(n log n) average / O(n²) worst

Quicksort flips the trade-offs. It sorts **in place** (O(log n) stack, no big
buffer) and is usually the fastest sort in practice. But it is **not stable**, and
on a bad pivot choice it degrades to **O(n²)** — classically when the input is
already sorted and you naively pick the last element as the pivot.

```go
package main

import "fmt"

// quickSort sorts a[lo..hi] in place (hi is inclusive).
func quickSort(a []int, lo, hi int) {
	if lo >= hi { // 0 or 1 element in range: nothing to do
		return
	}
	p := partition(a, lo, hi) // place pivot at its final home, p
	quickSort(a, lo, p-1)     // everything left of p is <= pivot
	quickSort(a, p+1, hi)     // everything right of p is >= pivot
}

// partition uses the last element as pivot (Lomuto scheme). It returns the
// pivot's final index; all elements <= pivot end up to its left.
func partition(a []int, lo, hi int) int {
	pivot := a[hi]
	i := lo // i marks the boundary of the "<= pivot" region
	for j := lo; j < hi; j++ {
		if a[j] <= pivot {
			a[i], a[j] = a[j], a[i] // swap smaller element into place
			i++
		}
	}
	a[i], a[hi] = a[hi], a[i] // drop the pivot just after the "<=" region
	return i
}

func main() {
	in := []int{5, 2, 9, 2, 1, 5, 6}
	quickSort(in, 0, len(in)-1)
	fmt.Println(in) // [1 2 2 5 5 6 9]
}
```

The fix for the O(n²) worst case is a better pivot: pick a random index, or the
median of the first/middle/last elements, before partitioning. Real libraries do
exactly this — and switch to insertion sort for tiny ranges.

## 4. Stability, and choosing between them

**Stability** means equal keys stay in their original order. It only matters when
you sort by one key but care about a second. Example: you have records already
sorted by name, and you stable-sort by department — within each department,
people stay in name order. An unstable sort would scramble that.

| | Merge sort | Quicksort |
|---|---|---|
| Worst case | **O(n log n)** | O(n²) |
| Average | O(n log n) | O(n log n) |
| Extra memory | O(n) | O(log n) (in place) |
| Stable? | **Yes** | No |
| Reach for it when… | you need stability or a hard worst-case guarantee | you want raw in-place speed and average case is fine |

Rule of thumb: **need a stability guarantee or a bounded worst case → merge sort.
Want the fastest general in-memory sort and don't care about order of equals →
quicksort.** In practice you rarely hand-roll either; you call the standard
library, which uses a tuned hybrid.

## 5. Go's `sort.Slice`

You implemented those to *understand* them. In real Go you sort with the `sort`
package. `sort.Slice` takes the slice and a `less(i, j)` function and sorts in
place — it is **not** guaranteed stable. When you need stability, use
`sort.SliceStable`.

```go
package main

import (
	"fmt"
	"sort"
)

type person struct {
	name string
	age  int
}

func main() {
	nums := []int{5, 2, 9, 1}
	sort.Slice(nums, func(i, j int) bool { return nums[i] < nums[j] }) // ascending
	fmt.Println(nums) // [1 2 5 9]

	people := []person{{"ada", 36}, {"bob", 36}, {"cleo", 28}}
	// Stable sort by age: ada and bob (both 36) keep their input order.
	sort.SliceStable(people, func(i, j int) bool { return people[i].age < people[j].age })
	fmt.Println(people) // [{cleo 28} {ada 36} {bob 36}]
}
```

The `less` function defines *your* ordering — flip the comparison for descending,
or compare a different field to sort by it. For the common case of a plain
`[]int`, `sort.Ints(nums)` is even shorter. The whole point: lean on the library;
only hand-write a sort when you're learning or have an exotic constraint.

## 6. Binary search — O(log n) on sorted data

Binary search finds a target in a **sorted** slice by repeatedly halving the range
it could be in. Each comparison eliminates half the remaining elements, so it runs
in **O(log n)** — for a million elements, about 20 comparisons. The prerequisite
is non-negotiable: **the input must already be sorted**, or the answer is garbage.

The pattern is `lo`, `hi`, `mid`:

```go
package main

import "fmt"

// Search returns the index of target in the sorted slice a,
// or -1 if it is not present. Iterative, O(log n).
func Search(a []int, target int) int {
	lo, hi := 0, len(a)-1 // hi is INCLUSIVE: the last valid index
	for lo <= hi {        // "<=" so a 1-element range [lo==hi] is still checked
		mid := lo + (hi-lo)/2 // avoids integer overflow vs (lo+hi)/2
		switch {
		case a[mid] == target:
			return mid // found it
		case a[mid] < target:
			lo = mid + 1 // target is in the upper half; discard mid and below
		default:
			hi = mid - 1 // target is in the lower half; discard mid and above
		}
	}
	return -1 // lo passed hi: range is empty, target absent
}

func main() {
	a := []int{1, 3, 4, 7, 9, 11, 15}
	fmt.Println(Search(a, 7))  // 3
	fmt.Println(Search(a, 1))  // 0
	fmt.Println(Search(a, 15)) // 6
	fmt.Println(Search(a, 8))  // -1 (not present)
	fmt.Println(Search([]int{}, 5)) // -1 (empty slice)
}
```

### The off-by-one pitfalls

These four decisions are *exactly* where binary search goes wrong. Get them right
and it just works:

- **`hi` is inclusive** here (`len(a)-1`). That choice forces the loop condition
  to be `lo <= hi`, not `lo < hi` — otherwise the final one-element range never
  gets checked and you miss targets at the boundary.
- **`mid := lo + (hi-lo)/2`**, not `(lo+hi)/2`. On huge slices `lo+hi` can
  overflow; this form can't. Make it a habit.
- **Always move past `mid`** (`mid+1` / `mid-1`). If you set `lo = mid` or
  `hi = mid`, a 2-element range can stop shrinking and you loop **forever**.
- **Empty / not-found** is handled by the loop simply ending: when `lo` passes
  `hi` the range is empty, so you fall through and return `-1`.

## 7. This is your Code Checker challenge — go pass it

The `Search(a []int, target int) int` function above is **exactly the app's Step 2
"binary-search" Code Checker challenge**. The signature matches, the contract
matches (return the index, or `-1` when absent), and it handles the empty slice.

Open the **binary-search** challenge in the Code Checker now, type the function in
yourself (don't just paste — typing it cements the lo/hi/mid pattern), run it
against the tests, and watch it pass. That's a real, committed win on your
Roadmap.

> Note: Go's standard library also ships `sort.SearchInts(a, target)`, but it
> returns the *insertion point* (where the value would go), not `-1` for a miss —
> a different contract. For the challenge, write your own `Search` as above.

## Do it yourself

1. Create a scratch module (`mkdir sorting && cd sorting && go mod init example.com/sorting`) and run each program above with `go run .`.
2. Hand-trace `mergeSort([5 2 9])` on paper, then add a `fmt.Println` inside `merge` to confirm your trace matches the real call order.
3. Break stability on purpose: change merge's `<=` to `<`, sort a slice with duplicate keys carried alongside a tag, and observe equal elements swapping order.
4. Trigger quicksort's O(n²) case: feed `quickSort` an already-sorted slice with the last-element pivot, add a counter to `partition`, and watch the comparison count blow up. Then switch to a random pivot and watch it drop.
5. Write `Search` from memory and run the four edge cases: target first, target last, target absent, empty slice. Then go pass the **binary-search** Code Checker challenge.
6. Re-implement everything via the standard library: `sort.Ints`, `sort.SliceStable`, and `sort.SearchInts`. Notice how little code real Go needs.

## Free resources

- [**NeetCode roadmap**](https://neetcode.io/roadmap) — work the *Binary Search* and *Sorting* sections; the videos walk the lo/hi/mid pattern and divide-and-conquer step by step.
- [**MIT 6.006 Introduction to Algorithms (Spring 2020)**](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/) — full lecture videos and notes on merge sort, recurrences (why it's O(n log n)), and binary search. The canonical free deep dive.

## Check yourself

You're ready to move on when you can answer, *without looking*:

- What are the three moves of divide and conquer, and how does each of merge sort / quicksort / binary search map onto them?
- Why is merge sort O(n log n) in the *worst* case while quicksort can hit O(n²) — and what causes quicksort's bad case?
- What does "stable" mean, when do you actually care, and which of `sort.Slice` / `sort.SliceStable` gives it to you?
- In binary search, why is the loop condition `lo <= hi` (with inclusive `hi`), and why `mid := lo + (hi-lo)/2`?
- Name the two bugs that come from setting `lo = mid` (instead of `mid+1`) or using `lo < hi` with an inclusive `hi`.

When all five feel obvious, commit this task on your **Roadmap**, make sure the
**binary-search** challenge is green, and take the Step 2 quiz. Next lesson:
**hashing & hash tables.**

## Common interview gotchas

- **The four binary-search bugs.** (1) `mid := (lo+hi)/2` can overflow — use `lo + (hi-lo)/2`. (2) `lo < hi` with an *inclusive* `hi` skips the final one-element range. (3) `lo = mid` instead of `mid+1` → infinite loop on a 2-element range. (4) Forgetting the empty/not-found case. Always hand-trace a 1- and 2-element input.
- **Rotated sorted array: one half is always sorted.** At each `mid`, decide which side is ordered, test whether the target lies in that side's range, and discard the other half — still O(log n). With duplicates (#81) the worst case degrades to O(n).
- **Merge vs quicksort.** Merge sort: O(n log n) *worst case*, stable, O(n) memory. Quicksort: O(n log n) average but **O(n²) on a bad pivot** (e.g. last-element pivot on sorted input), not stable, in place. Need stability or a hard worst-case bound → merge; want raw in-place speed → quicksort with a random/median-of-three pivot.
- **`sort.Slice` is NOT stable.** If you depend on equal elements keeping their order, call `sort.SliceStable` explicitly — don't assume it.
- **Heap sort fills a niche.** O(n log n) worst case *and* O(1) space, in place — the guarantee merge sort gives but without its O(n) buffer, and the in-place property quicksort gives without its O(n²) risk. The same heap is your **priority queue** (`container/heap`) for streaming top-k.
