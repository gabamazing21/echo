---
slug: dsa-graphs
step: 2
title: Graphs — BFS, DFS, topological sort & union-find
summary: Represent graphs, traverse them with BFS/DFS, order dependencies with topological sort, and group with union-find.
est_min: 600
position: 10
---

# Graphs — BFS, DFS, topological sort & union-find

> **Step 2 · Data structures & algorithms · Interview prep**
> Concept: *traversal, ordering, connectivity*

Graphs model anything with relationships: networks, dependencies, maps, social
follows, the partition map from Step 7. A huge slice of medium/hard interview
questions are graph problems in disguise ("number of islands" is a grid graph).
Master four tools and most of them fall.

## Why this matters

Graph traversal is core interview territory at every big-tech company, and it's the
backbone of real systems — shortest paths (maps/routing), dependency resolution
(build systems, schedulers), and connectivity (clustering, networks). It also ties
straight into distributed systems: a cluster *is* a graph.

## 1. Representation

- **Adjacency list** (default): `map[int][]int` or `[][]int` — neighbors per node.
  Space O(V+E), great for sparse graphs (most real ones).
- **Adjacency matrix**: `[V][V]bool` — O(V²) space, O(1) edge lookup; only for dense
  graphs or when you need fast "is there an edge?".
- A **grid** (like "number of islands") is an implicit graph: each cell is a node,
  edges to its 4 neighbors.

## 2. BFS — shortest path in an unweighted graph

Breadth-first explores level by level using a **queue**; the first time you reach a
node is via the *fewest edges*.

```go
func bfs(adj map[int][]int, start int) {
	visited := map[int]bool{start: true}
	q := []int{start}
	for len(q) > 0 {
		node := q[0]; q = q[1:]
		for _, nb := range adj[node] {
			if !visited[nb] { visited[nb] = true; q = append(q, nb) }
		}
	}
}
```

Use BFS for shortest path / minimum steps in unweighted graphs and level-order
problems. **Mark visited when you enqueue**, not when you dequeue, or you'll add
nodes twice.

## 3. DFS — explore deep, detect structure

Depth-first goes as deep as possible (recursion or an explicit stack). Use it for
connectivity, cycle detection, path existence, and anything "explore the whole
component."

```go
func dfs(adj map[int][]int, node int, visited map[int]bool) {
	visited[node] = true
	for _, nb := range adj[node] {
		if !visited[nb] { dfs(adj, nb, visited) }
	}
}
```

"Number of islands" = count how many DFS/BFS calls it takes to visit every land
cell. Both BFS and DFS are **O(V+E)**.

## 4. Topological sort — order a DAG

For a **directed acyclic graph** (dependencies), a topological order lists every
node before the nodes that depend on it — course schedules, build order, task
pipelines. Two ways:

- **Kahn's algorithm (BFS)**: repeatedly remove a node with in-degree 0, decrement
  its neighbors. If you can't remove all nodes, there's a **cycle** (no valid order).
- **DFS post-order**: push nodes as DFS finishes them, then reverse.

"Course Schedule" (can you finish all courses given prerequisites?) is exactly cycle
detection on a directed graph — a top interview classic.

## 5. Union-Find (Disjoint Set Union)

For "are these two connected?" / "how many groups?" without repeated traversal.
Each element points to a parent; `find` returns the root, `union` merges two roots.
With **path compression** + **union by rank**, operations are near-O(1) (inverse
Ackermann).

```go
func find(p []int, x int) int { for p[x] != x { p[x] = p[p[x]]; x = p[x] }; return x }
func union(p []int, a, b int)  { p[find(p,a)] = find(p,b) }
```

Use it for connected components, cycle detection in *undirected* graphs, "number of
provinces", and Kruskal's MST.

## 6. Weighted shortest path: Dijkstra (briefly)

For non-negative weighted edges, **Dijkstra** = BFS with a **min-heap** (priority
queue) by distance — pop the closest unsettled node, relax its edges. O((V+E) log V).
(Heaps are the next bonus lesson.) For graphs with negative edges, Bellman-Ford.

## Do it yourself (≈ 10 hrs)

1. Work the [**NeetCode Graphs**](https://neetcode.io/roadmap) section.
2. Solve in order: Number of Islands, Clone Graph, Course Schedule (cycle detect),
   Pacific Atlantic Water Flow, Number of Provinces (union-find), Rotting Oranges
   (multi-source BFS). The in-app **Number of Islands** checker is below.
3. Implement BFS, DFS, Kahn's topo sort, and union-find from scratch once each —
   you want them in muscle memory.

## Check yourself

- When do you use BFS vs DFS? Which gives shortest path in an unweighted graph?
- Why mark a node visited on enqueue (BFS), not dequeue?
- How does topological sort detect a cycle?
- What does union-find do that repeated DFS does inefficiently?
- What's the one change that turns BFS into Dijkstra?

Next bonus lesson: **Heaps & Top-K** — the priority queue and its killer pattern.

## Common interview gotchas

- **Mark visited on *enqueue*, not on dequeue.** If you mark only when popping, the same node gets pushed multiple times before it's processed — blowing up the queue and breaking the shortest-path guarantee.
- **Recursive DFS overflows the stack on large/deep graphs.** A linked-list-shaped graph of 10⁵ nodes will crash recursion — switch to an explicit stack (iterative DFS) when depth can be huge.
- **Cycle detection differs by direction.** Undirected: a "visited neighbor that isn't your parent" is a cycle (or use union-find). Directed: you need three colors / a recursion-stack set — a node merely being visited is *not* a back edge.
- **Union-find without path compression *and* union by rank degrades to O(n) per op.** A naive `find` that walks a long chain makes the whole thing quadratic — both optimizations are required for the near-O(1) (inverse Ackermann) claim.
- **Dijkstra breaks on negative edges.** Once a node is popped it's treated as settled, so a later cheaper path is never reconsidered — use Bellman-Ford (or SPFA) when weights can be negative.
