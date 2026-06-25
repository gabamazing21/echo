---
slug: dsa-practice-patterns
step: 2
title: How to practice — the patterns that recur
summary: Recognise the handful of patterns behind most problems, grind them with a deliberate method, and manage interview time well.
est_min: 840
position: 7
---

# How to practice — the patterns that recur

> **Step 2 · Data structures & algorithms · Week 3 · Practice**
> Concept: *pattern recognition under time pressure*

You've learned the containers and the core algorithms. This lesson is different:
it's not about *one* algorithm, it's about **how to practice** so that, under a
ticking clock and a watching interviewer, the right approach surfaces in your head
within two minutes. The secret nobody tells beginners is that you are not solving
hundreds of unique puzzles. You're learning to recognise a small set of
**patterns** that recur over and over. Master the patterns, and most "hard"
problems become "oh, it's *that* shape again."

## Why this matters

Interview problems feel infinite when every one looks brand new. They stop feeling
infinite the moment you can classify a problem by its *structure* rather than its
story. "Find the longest substring without repeats" and "smallest subarray summing
to at least K" sound unrelated — they're the same **sliding window** pattern. The
goal of grinding is not to *memorise solutions*; memorised solutions evaporate
under stress. The goal is to build a reliable **classifier** in your head: read a
problem, feel which pattern fits, and start coding the skeleton you've drilled a
dozen times. That's what separates people who freeze from people who flow.

## 1. The high-frequency patterns

Almost everything you'll meet maps onto this short list. Learn to *spot the
trigger* for each — the phrase or constraint that whispers which one to reach for.

| Pattern | Typical trigger | Mental model |
|---|---|---|
| **Two pointers** | sorted array, pair/triplet, "in place" | one index from each end, or fast/slow |
| **Sliding window** | "substring/subarray", "longest/shortest", contiguous | grow right, shrink left, track a running stat |
| **Hashing / frequency map** | "count", "seen before", "anagram", O(1) lookup | `map[T]int` or `map[T]bool` |
| **Binary search** | sorted input, or "minimise the max" / monotonic answer | halve the search space each step |
| **BFS / DFS (trees & graphs)** | levels, reachability, connected components, paths | queue for BFS, recursion/stack for DFS |
| **Stack-based** | matching pairs, "next greater", nesting, undo | push context, pop when resolved |
| **Intervals** | meetings, ranges, overlaps, merging | sort by start, compare with previous end |
| **Dynamic programming (intro)** | "number of ways", "min/max cost", overlapping subproblems | define a state, write the recurrence, then cache |

Don't try to master all eight at once. Work them roughly top-to-bottom — two
pointers and sliding window pay off fastest — and let the harder ones (graphs, DP)
build on the easier ones.

## 2. A study method that actually sticks

Volume alone doesn't work. *Deliberate* practice does. Use this loop for every
problem:

1. **Attempt cold.** Read the problem, decide the pattern, and start coding. No
   peeking.
2. **Struggle for 20–30 minutes.** This is where the learning happens. Being stuck
   is the workout — your brain is forming the retrieval paths you'll use in the
   real interview. Don't cut it short.
3. **Read the solution** *only after* the timer. Read until you understand *why*,
   not just *what*. Identify which pattern it was and what trigger you missed.
4. **Re-implement from scratch.** Close the solution. Type it out from your own
   understanding. If you can't, you didn't understand it — re-read and retry.
5. **Spaced repetition.** Revisit the same problem after ~3 days, then ~1 week. A
   problem you can re-solve a week later from a blank file is a problem you
   genuinely *own*. Keep a simple list: problem, pattern, date last solved.

The 20–30 minute struggle cap matters in both directions: long enough to actually
grapple, short enough that you don't burn an evening spinning on one problem.

## 3. Pattern in code: two pointers

The trigger here is a **sorted** array and a pair-sum question. Instead of the
O(n²) double loop, walk one pointer from each end and let the sortedness steer you:

```go
// twoSumSorted returns the indices of two values that add to target,
// or (-1, -1) if none exist. Input must be sorted ascending.
func twoSumSorted(nums []int, target int) (int, int) {
	lo, hi := 0, len(nums)-1
	for lo < hi {
		sum := nums[lo] + nums[hi]
		switch {
		case sum == target:
			return lo, hi
		case sum < target:
			lo++ // need a bigger sum, move the low end up
		default:
			hi-- // need a smaller sum, move the high end down
		}
	}
	return -1, -1
}
```

One pass, O(n) time, O(1) space. The instant you see "sorted" plus "find a pair,"
your hand should reach for this skeleton.

## 4. Pattern in code: sliding window

The trigger is a **contiguous** subarray/substring asking for a longest or
shortest run. Grow the window on the right, shrink it from the left when a
constraint breaks, and track the best you've seen:

```go
// longestUniqueSubstr returns the length of the longest substring of s
// containing no repeated bytes.
func longestUniqueSubstr(s string) int {
	last := make(map[byte]int) // byte -> most recent index seen
	best, left := 0, 0
	for right := 0; right < len(s); right++ {
		c := s[right]
		// If we've seen c inside the current window, jump left past it.
		if i, ok := last[c]; ok && i >= left {
			left = i + 1
		}
		last[c] = right
		if w := right - left + 1; w > best {
			best = w
		}
	}
	return best
}
```

Notice the window never moves backwards — `left` only ever increases — so this is
O(n), not O(n²). That "two indices sweeping forward" feel is the heart of the
pattern.

## 5. Managing time in the interview

Knowing the pattern isn't enough; you have to *spend the clock well*. A reliable
order under pressure:

1. **Clarify (1–2 min).** Restate the problem. Ask about input size, ranges,
   duplicates, empty input. This buys thinking time *and* scores points.
2. **Brute force out loud (2–3 min).** State the naive solution and its complexity.
   It proves you understand the problem and gives you something to optimise *from*.
3. **Name the pattern (1–2 min).** "The sorted constraint suggests two pointers."
   Talk through the approach before coding.
4. **Code the skeleton.** Write the structure you've drilled, narrating as you go.
   Silence makes interviewers nervous; a running commentary reassures them.
5. **Test on a small example by hand.** Walk one concrete input through your code.
   This is where you catch off-by-one and empty-input bugs.
6. **State complexity.** Finish by giving time and space Big-O. Always.

If you're stuck at minute 10, *say so* and reason aloud — "let me consider a hash
map to make the lookup O(1)." Interviewers hire for how you think when stuck, not
for instant brilliance.

## 6. Tracking your weak spots

You improve fastest by attacking what you're *worst* at, which means you have to
know what that is. Keep a tiny log — a spreadsheet or a text file is plenty — with
four columns:

- **Problem** (name/link)
- **Pattern** (from section 1)
- **Outcome** (solved cold / needed a hint / read solution)
- **Date last solved**

After a couple of weeks the pattern jumps out: maybe you crush two pointers but
freeze on anything graph-shaped. Now your practice has direction — spend the next
sessions on BFS/DFS until that column turns green too. Drilling your strengths
feels good and teaches you nothing; drilling your weak column is the whole game.

## Do it yourself

1. Open the [**NeetCode roadmap**](https://neetcode.io/roadmap) — it groups problems
   by exactly the patterns in section 1, in a sensible learning order. Use it as
   your curriculum.
2. Create a free [**LeetCode**](https://leetcode.com/) account. Filter by *Easy*
   and start with the Arrays & Hashing and Two Pointers groups.
3. Run the section 2 loop on every problem: attempt cold, struggle 20–30 min, read,
   re-implement from a blank file, schedule a re-visit.
4. Start your weak-spot log today, on your very first problem. Don't wait until you
   "have data" — the log *is* how you get data.
5. For each problem, before coding, write a one-line comment naming the pattern and
   its trigger. Force yourself to classify *first*.

## Done when

- You've solved **30 Easy + 10 Medium** problems using the section 2 loop.
- You can **name the pattern within 2 minutes** of reading a fresh problem, before
  writing any code.
- You can re-implement two-pointers and sliding-window skeletons from a **blank
  file**, no reference.
- Your weak-spot log has **at least one green column** you turned around from red.
- In a timed mock, you naturally run the section 5 sequence — clarify, brute force,
  pattern, code, test, complexity — without thinking about it.

When all five are true, commit this task on your **Roadmap** and take the Step 2
quiz. Next step: **Step 3 — concurrency in Go.**

## Common interview gotchas

- **One constraint flips the pattern.** "Find a pair summing to target" is the **hashing** pattern when unsorted (O(n) time, O(n) space) but the **two-pointers** pattern when sorted (O(n) time, O(1) space). Train yourself to spot the trigger word ("sorted") rather than memorizing solutions.
- **Exponential blowup → dynamic programming.** "Number of ways / min-max cost / overlapping subproblems" signals DP. Climbing Stairs is Fibonacci: naive recursion is O(2ⁿ); memoize for O(n), or roll two variables for O(1) space. The recipe: define state → write recurrence → base case → cache.
- **Kadane's init trap.** Maximum Subarray with one pass (`cur = max(x, cur+x)`) is O(n)/O(1), but seed `best` with the first element (or -∞), *not* 0 — an all-negative array wrongly returns 0 otherwise.
- **Grid / "connected" / "regions" → graph traversal.** Number of Islands is connected components: DFS or BFS flooding each unvisited cell, marking visited in place. BFS (queue) avoids deep-recursion stack overflow on huge grids; Union-Find is a third valid approach.
- **Union-Find is near-constant with two optimizations.** Disjoint-set fires on grouping / cycle-detection / dynamic-connectivity problems. **Union by rank/size** + **path compression** give O(α(n)) — effectively O(1) — and often beat BFS/DFS when edges arrive incrementally.
