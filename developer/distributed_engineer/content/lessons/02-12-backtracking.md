---
slug: dsa-backtracking
step: 2
title: Backtracking
summary: Systematically build candidates and abandon dead ends — the template behind permutations, subsets, combinations, and N-Queens.
est_min: 360
position: 12
---

# Backtracking

> **Step 2 · Data structures & algorithms · Interview prep**
> Concept: *exhaustive search with pruning*

Backtracking is how you explore *all* configurations — every subset, permutation,
or board placement — without writing a tangle of nested loops. It's DFS over a
"decision tree": choose, recurse, then **undo the choice** and try the next. One
template solves a whole family of interview problems.

## Why this matters

Permutations, combinations, subsets, word search, N-Queens, sudoku, generating
parentheses — these recur constantly in interviews, and they intimidate candidates
who don't know the template. Once you internalize choose → explore → un-choose,
they become mechanical. It's also the basis of constraint solvers and search.

## 1. The mental model: a decision tree

At each step you make a **choice**, recurse to make the next choice, and when you
return, **undo** that choice so you can try the alternatives. You're doing DFS over
the tree of all partial solutions, **pruning** branches that can't lead to a valid
answer.

## 2. The universal template

```go
func backtrack(state, choices) {
	if isComplete(state) {
		record(state)        // found a full solution
		return
	}
	for _, choice := range choices {
		if !valid(state, choice) { continue } // prune
		apply(state, choice)     // choose
		backtrack(state, next)   // explore
		undo(state, choice)      // un-choose (the "backtrack")
	}
}
```

The three lines — **choose, explore, un-choose** — are the whole pattern. The
`undo` is what makes it backtracking rather than plain recursion. **Append a copy**
when recording, since the working slice keeps mutating.

## 3. Subsets

Each element is either in or out — 2ⁿ subsets. At index `i`, branch on include vs
skip:

```go
func subsets(nums []int) [][]int {
	var res [][]int
	var cur []int
	var bt func(i int)
	bt = func(i int) {
		if i == len(nums) {
			res = append(res, append([]int{}, cur...)) // copy!
			return
		}
		cur = append(cur, nums[i]); bt(i + 1) // include
		cur = cur[:len(cur)-1]; bt(i + 1)     // exclude (undo, then skip)
	}
	bt(0)
	return res
}
```

## 4. Permutations

All orderings — n! of them. Track which elements are used; choose each unused one:

```go
// for each position, try every unused number, mark used, recurse, unmark
```

The `used []bool` (mark/unmark around the recursive call) is the choose/un-choose.

## 5. Complexity & pruning

Backtracking is **exponential** by nature (2ⁿ subsets, n! permutations) — that's
inherent to "generate all". The art is **pruning**: cut branches early when they
can't succeed (N-Queens: skip a column/diagonal already attacked; combination-sum:
stop when the running sum exceeds the target). Good pruning is the difference
between "passes" and "times out". Sort first when it enables earlier cutoffs.

## 6. The problem family

- **Subsets** (#78), **Subsets II** (with duplicates — sort + skip equal siblings).
- **Permutations** (#46), **Permutations II**.
- **Combinations** (#77), **Combination Sum** (#39).
- **Generate Parentheses** (#22) — prune by open/close counts.
- **Word Search** (#79) — backtrack over a grid (DFS + un-visit).
- **N-Queens** (#51), **Sudoku Solver** (#37) — the showcase pruning problems.

They're all the same template with different `valid`, `isComplete`, and `choices`.

## Do it yourself (≈ 6 hrs)

1. Read [**NeetCode — Backtracking**](https://neetcode.io/roadmap).
2. Solve in order: Subsets, Combination Sum, Permutations, Generate Parentheses,
   Word Search, then N-Queens. Write the choose/explore/un-choose comments first.
3. For one problem, add pruning and observe the speedup vs the un-pruned version.

## Check yourself

- What three steps make up the backtracking template, and which one *is* the "backtrack"?
- Why must you append a **copy** of the working slice when recording a solution?
- Subsets are 2ⁿ and permutations are n! — why is that inherent, not a bug?
- What is pruning, and give one concrete example (N-Queens or combination-sum)?
- How does Word Search apply the same template over a grid?

That completes the core DSA interview topics. Back to the **Roadmap** to keep going.

## Common interview gotchas

- **Record a *copy* of the working slice, not the slice itself.** `append(res, cur)` stores a reference that keeps mutating as you backtrack — you must `append(res, append([]int{}, cur...))` or every result ends up identical (or empty).
- **The *un-choose* is what makes it backtracking.** Drop the `undo` line and you carry state into sibling branches — the recursion still runs but produces wrong/duplicated results; it's the restore that isolates each path.
- **Pruning is the line between AC and TLE, not an optimization.** Subsets-II and Combination Sum *must* skip equal siblings / stop past the target — without it the same exponential tree is explored fully and times out on the larger cases.
- **2ⁿ subsets and n! permutations are inherent, not a bug to optimize away.** You can't beat the output size when you must *generate all* — interviewers want you to state this bound confidently, then focus optimization on pruning the invalid branches.
- **Sort first only when it enables a cutoff.** Sorting helps skip duplicates or break early past a target, but it doesn't make exhaustive generation cheaper by itself — know *why* you're sorting.
