---
slug: sd-typeahead
step: 9
title: "Design: search autocomplete (typeahead)"
summary: A trie of popular queries with precomputed top-K per node, served from memory and rebuilt from logs — low-latency suggestions at scale.
est_min: 300
position: 10
---

# Design: search autocomplete (typeahead)

> **Step 9 · System design · Interview prep**
> Concept: *trie + precomputed top-K + caching*

"Suggest completions as the user types" is a favorite Google/Amazon design prompt. It
looks simple but hides a sharp latency constraint (suggestions must appear within
tens of ms, on *every keystroke*) and a nice data-structure + data-pipeline split.

## Why this matters

It combines a classic data structure (the trie, from Step 2) with system-design
scaling (caching, sharding, offline pipelines) and the **top-K** pattern — a clean
showcase of "index + precompute + cache".

## 1. Requirements & scale

Functional: given a prefix, return the top ~5–10 most relevant completions, fast,
updated reasonably often. Non-functional: **very low latency** (it fires per
keystroke), read-heavy, eventual freshness is fine (new trending terms can lag
minutes). Scale: billions of queries feeding the suggestions; huge read QPS.

## 2. The data structure: a trie with cached top-K

A **trie** (prefix tree) maps each prefix to its node. The key optimization: at each
node, **precompute and store the top-K completions** for that prefix. Then a lookup
is a single traversal to the prefix node + return its cached list — no ranking at
query time. Without the cached top-K you'd have to gather and sort all descendant
terms per keystroke, which is far too slow.

## 3. Read path

In-memory trie replicas behind the suggestion service: traverse to the prefix node,
return its top-K. Debounce on the client (don't fire on every single keystroke) and
cap result count. Cache hot prefixes at the edge/CDN. Reads never touch disk.

## 4. Write path (the offline pipeline)

You don't update the trie on every search in real time. Instead:

- A **data pipeline** aggregates query frequencies from logs over a window (map-reduce
  / stream processing).
- Periodically **rebuild** (or incrementally update) the trie with new frequencies
  and recompute each node's top-K.
- Ship the new trie to the read replicas (atomic swap).

This separates the cheap, hot **read** path from the heavy, batched **write** path —
the recurring "precompute offline, serve from memory" idea.

## 5. Deep dives

- **Sharding**: partition the trie by first letter(s)/prefix range across servers.
- **Personalization / location**: blend a global trie with per-user/region signals
  (more state, more cost).
- **Typo tolerance**: fuzzy matching (edit distance) is expensive — usually a
  separate path, not the hot trie.
- **Freshness vs cost**: rebuild cadence trades trending-term latency against compute.

## Do it yourself (≈ 4 hrs)

1. Watch [**ByteByteGo — Search Autocomplete**](https://bytebytego.com/courses/system-design-interview/design-a-search-autocomplete-system) and skim [**System Design Primer**](https://github.com/donnemartin/system-design-primer).
2. Sketch the trie node with a cached top-K and describe the offline rebuild pipeline.

## Check yourself

- Why store a precomputed top-K at each trie node instead of ranking at query time?
- Why separate the read path from an offline write/rebuild pipeline?
- How would you shard the trie across servers?
- What makes typo tolerance expensive, and how do you handle it?
- What does the rebuild cadence trade off?

## Common interview gotchas

- **Ranking at query time is too slow** — gathering and sorting all completions under a prefix on every keystroke blows the latency budget; precompute top-K per node.
- **Don't update the trie synchronously on every search** — decouple with an offline aggregation + periodic rebuild; the read path must stay a pure in-memory traversal.
- **Debounce and cap on the client** — firing a request per keystroke with no cap multiplies QPS needlessly; wait for a short pause and limit suggestions.
- **Fuzzy/typo matching isn't the hot path** — edit-distance search is costly; keep it as a fallback service, don't bolt it onto the per-keystroke trie lookup.
