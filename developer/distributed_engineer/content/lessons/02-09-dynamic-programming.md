---
slug: dsa-dynamic-programming
step: 2
title: Dynamic programming
summary: Spotting overlapping subproblems and optimal substructure — memoization, tabulation, and the classic DP patterns interviews love.
est_min: 600
position: 9
---

# Dynamic programming

> **Step 2 · Data structures & algorithms · Interview prep**
> Concept: *overlapping subproblems, optimal substructure*

Dynamic programming (DP) is the topic candidates fear most and interviewers love
most. It's not magic — it's **recursion plus remembering answers you've already
computed**. Once you see the pattern, a whole category of "hard" problems becomes
routine.

## Why this matters

DP is one of the highest-frequency hard-topic areas at Google, Meta, Amazon, and
friends. It's also genuinely useful: edit distance powers diffs, knapsack powers
resource allocation, and the *thinking* — break a problem into subproblems and
reuse — is everywhere. This lesson is the one Step 2 was missing for full interview
coverage.

## 1. The two signals

A problem is a DP candidate when it has both:

- **Overlapping subproblems** — the naive recursion solves the *same* smaller
  problem many times. (Fibonacci recomputes `fib(3)` exponentially often.)
- **Optimal substructure** — the optimal answer is built from optimal answers to
  subproblems. (Shortest path through a node uses shortest paths to it.)

If subproblems *don't* overlap, it's plain divide-and-conquer (merge sort), not DP.

## 2. From exponential recursion to DP

Naive Fibonacci is O(2ⁿ) — it recomputes everything:

```go
func fib(n int) int { if n < 2 { return n }; return fib(n-1) + fib(n-2) }
```

**Memoization (top-down):** keep the recursion, cache each answer:

```go
func fib(n int, memo map[int]int) int {
	if n < 2 { return n }
	if v, ok := memo[n]; ok { return v }
	memo[n] = fib(n-1, memo) + fib(n-2, memo)
	return memo[n]
}
```

**Tabulation (bottom-up):** fill a table from the base cases up:

```go
func fib(n int) int {
	if n < 2 { return n }
	dp := make([]int, n+1)
	dp[1] = 1
	for i := 2; i <= n; i++ { dp[i] = dp[i-1] + dp[i-2] }
	return dp[n]
}
```

Both are **O(n)**. Often you can drop the table to **O(1) space** by keeping only
the last few values (here, two variables). That space optimization is a frequent
interview follow-up.

## 3. The five-step DP recipe

For any DP problem, answer these in order:

1. **State** — what does `dp[i]` (or `dp[i][j]`) *mean*? (The hardest, most
   important step.)
2. **Recurrence** — how does a state depend on smaller states?
3. **Base cases** — the smallest states you know outright.
4. **Order** — fill so dependencies are computed first (bottom-up) or recurse +
   memo (top-down).
5. **Answer** — which state holds the final result?

If you can state #1 and #2 clearly out loud, you've basically solved it — say them
to the interviewer before coding.

## 4. The patterns that cover most interviews

- **1-D sequence**: climbing stairs, house robber, max subarray (Kadane is DP),
  longest increasing subsequence. State = best answer ending at / up to index `i`.
- **2-D grid**: unique paths, min path sum. State = best to reach cell `(i,j)`.
- **Two sequences**: longest common subsequence, **edit distance**. State =
  `dp[i][j]` over prefixes of both strings.
- **Knapsack / subset**: 0/1 knapsack, coin change, partition equal subset. State =
  best using first `i` items with capacity `c`.
- **Intervals**: matrix-chain, burst balloons. State = best over a range `(i,j)`.

Recognizing *which family* a problem belongs to is 80% of the battle.

## 5. Worked mini-example: coin change

"Fewest coins to make amount A from given denominations." State: `dp[a]` = fewest
coins for amount `a`. Recurrence: `dp[a] = 1 + min(dp[a-c])` over each coin `c ≤ a`.
Base: `dp[0] = 0`. Answer: `dp[A]` (or "impossible" if unreached). That's the whole
solution — O(A · #coins).

## Do it yourself (≈ 10 hrs)

1. Watch the [**NeetCode 1-D and 2-D DP**](https://neetcode.io/roadmap) sections — the explanations are the best free DP teaching anywhere.
2. Solve these in order (each is in the app's **Interview prep** below where possible): Climbing Stairs, House Robber, Coin Change, Longest Common Subsequence, Word Break, Edit Distance.
3. For each, **write the state and recurrence in a comment first**, then code. Then do the space-optimized version.
4. Pass the in-app **Climbing Stairs** checker challenge — your first tabulation.

## Check yourself

- What two properties make a problem a DP problem?
- Memoization vs tabulation — what's the difference, and when is each easier?
- Walk through turning O(2ⁿ) Fibonacci into O(n), then O(1) space.
- For coin change, state the meaning of `dp[a]` and the recurrence.
- Name the DP family for: edit distance, house robber, unique paths.

Next bonus lesson: **Graphs** — BFS, DFS, topological sort, and union-find.

## Common interview gotchas

- **The hard part is the *state definition*, not the code.** If you can't say in one sentence what `dp[i]` *means*, your recurrence will be wrong — articulate the state before writing a single line.
- **Greedy does not solve coin change.** Taking the largest coin first fails for denominations like `{1, 3, 4}` making 6 (greedy gives 4+1+1=3 coins, optimal is 3+3=2) — only DP guarantees the minimum.
- **Memoization vs tabulation isn't just style.** Top-down only computes states you actually reach (good for sparse state spaces); bottom-up avoids recursion-stack overflow on deep inputs — pick for the constraint, not taste.
- **O(1)-space rolling requires that `dp[i]` depend only on a fixed window.** Fibonacci needs the last two values, so two variables suffice; if a state reaches arbitrarily far back, you can't roll it away.
- **1-indexed `dp` over strings/arrays is an off-by-one minefield.** With `dp[i][j]` over *prefixes*, `dp[i]` usually means "first `i` chars," so you index the original string at `s[i-1]`, not `s[i]` — mixing the two silently corrupts the table.
