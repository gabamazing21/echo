---
slug: dsa-hash-maps
step: 2
title: Hash maps, under the hood
summary: How hash maps really work — hashing, buckets, collisions, load factor, resizing, and why lookup is O(1) average, O(n) worst.
est_min: 240
position: 3
---

# Hash maps, under the hood

> **Step 2 · Data structures & algorithms · Week 1**
> Concept: *hashing, collisions, O(1) average lookup*

You've used Go's `map` already — comma-ok reads, `delete`, randomized iteration.
Now you open the hood. A **hash map** is the single most important data structure
in practical software: it turns "find the value for this key" from a scan of
every element into, on average, a *constant-time* jump. Understanding *how* it
pulls that off — and *when* it stops being O(1) — is the difference between using
maps and reasoning about them.

## Why this matters

Hash maps are the quiet engine under almost everything distributed. A cache
(Redis, Memcached, your service's in-memory layer) is a hash map. A database
**index** is often a hash index. **Deduplication** — "have I seen this event ID
before?" — is a hash set lookup. **Consistent hashing**, which you'll meet in
Step 7 when you shard data across nodes, is hashing applied to the problem of
*which machine owns this key*. Get the fundamentals here and those later topics
become variations on a theme you already know.

## 1. The core idea: a function from key to slot

Start with the problem. You have keys (`"alice"`, `"bob"`) and you want to store
a value for each, then fetch it fast. An array gives you O(1) access *by integer
index* — `arr[42]` is instant. The trick of a hash map is to **turn any key into
an integer index** using a **hash function**:

```
index = hash(key) % numberOfBuckets
```

The hash function maps an arbitrary key to a big integer; the modulo squeezes
that integer into the range of available slots, called **buckets**. Store the
key/value pair in that bucket, and to look it up later you recompute the same
index and go straight there. No scanning.

A good hash function has two properties: it's **deterministic** (same key →
same hash, every time) and it **distributes uniformly** (different keys spread
evenly across buckets, so no single bucket gets overloaded).

## 2. Collisions are inevitable

Here's the catch. You're mapping a huge space of possible keys onto a small
number of buckets, so two different keys *will* eventually land in the same
bucket. That's a **collision**. It's not a bug or a rare edge case — by the
[pigeonhole principle](https://neetcode.io/roadmap) it's guaranteed once you
have more keys than buckets. The entire engineering of a hash map is really the
engineering of *how it handles collisions*.

There are two classic strategies.

## 3. Collision resolution I: separate chaining

In **separate chaining**, each bucket holds a *list* of entries. On collision,
you just append to that bucket's list. To look up a key, you hash to the bucket,
then walk its (usually tiny) list comparing keys:

```go
package main

import "fmt"

// A deliberately tiny, illustrative hash table using separate chaining.
// NOT for production — Go's built-in map is far better. This is to see inside.
type entry struct {
	key string
	val int
}

type HashTable struct {
	buckets [][]entry // each bucket is a slice of entries (the "chain")
}

func New(size int) *HashTable {
	return &HashTable{buckets: make([][]entry, size)}
}

// hash: a simple polynomial rolling hash over the bytes of the key.
func (h *HashTable) hash(key string) int {
	var sum uint32
	for i := 0; i < len(key); i++ {
		sum = sum*31 + uint32(key[i])
	}
	return int(sum) % len(h.buckets)
}

func (h *HashTable) Put(key string, val int) {
	i := h.hash(key)
	for j := range h.buckets[i] { // update if key already present
		if h.buckets[i][j].key == key {
			h.buckets[i][j].val = val
			return
		}
	}
	h.buckets[i] = append(h.buckets[i], entry{key, val}) // else append to chain
}

func (h *HashTable) Get(key string) (int, bool) {
	i := h.hash(key)
	for _, e := range h.buckets[i] { // walk the chain in this bucket
		if e.key == key {
			return e.val, true
		}
	}
	return 0, false
}

func main() {
	h := New(4)
	h.Put("alice", 30)
	h.Put("bob", 25)
	h.Put("carol", 41)
	v, ok := h.Get("bob")
	fmt.Println(v, ok) // 25 true
	_, ok = h.Get("dave")
	fmt.Println(ok) // false
}
```

This is the whole idea in ~40 lines. Each bucket's chain stays short *as long as*
the table isn't too full — which brings us to load factor.

## 4. Collision resolution II: open addressing

The other family is **open addressing**: every entry lives directly in the bucket
array (no side lists). On collision, you **probe** for the next free slot by a
fixed rule. The simplest is **linear probing** — try `i`, then `i+1`, `i+2`, …
wrapping around — until you find an empty slot (to insert) or your key (to read):

```go
package main

import "fmt"

// Open addressing with linear probing. Fixed capacity, no resize, for clarity.
type slot struct {
	key  string
	val  int
	used bool
}

type OAMap struct {
	slots []slot
}

func NewOA(size int) *OAMap { return &OAMap{slots: make([]slot, size)} }

func (m *OAMap) hash(key string) int {
	var sum uint32
	for i := 0; i < len(key); i++ {
		sum = sum*31 + uint32(key[i])
	}
	return int(sum) % len(m.slots)
}

func (m *OAMap) Put(key string, val int) {
	i := m.hash(key)
	for n := 0; n < len(m.slots); n++ {
		j := (i + n) % len(m.slots) // probe forward, wrapping
		if !m.slots[j].used || m.slots[j].key == key {
			m.slots[j] = slot{key, val, true}
			return
		}
	}
	// full — a real implementation would resize before this happens.
}

func (m *OAMap) Get(key string) (int, bool) {
	i := m.hash(key)
	for n := 0; n < len(m.slots); n++ {
		j := (i + n) % len(m.slots)
		if !m.slots[j].used {
			return 0, false // empty slot => key not present
		}
		if m.slots[j].key == key {
			return m.slots[j].val, true
		}
	}
	return 0, false
}

func main() {
	m := NewOA(8)
	m.Put("x", 1)
	m.Put("y", 2)
	fmt.Println(m.Get("x")) // 1 true
	fmt.Println(m.Get("z")) // 0 false
}
```

Open addressing is cache-friendly (everything is in one contiguous array) but
degrades badly when the table gets crowded, because probe sequences get long.

| | Separate chaining | Open addressing |
|---|---|---|
| Storage | array of lists | one flat array |
| On collision | append to bucket's list | probe for next slot |
| Memory locality | poorer (pointers) | excellent (contiguous) |
| Tolerates high fullness | gracefully | poorly — probes explode |

## 5. Load factor and resizing

The **load factor** is `entries / buckets` — how full the table is. It's the
single number that governs performance. Low load factor → short chains / short
probes → near-O(1). High load factor → long chains / long probes → drifting
toward O(n).

So a real hash map watches its load factor and, when it crosses a threshold
(commonly ~0.75), it **resizes**: allocate a bigger bucket array (typically
double) and **rehash** every existing entry into it, since the bucket index
depends on `numberOfBuckets`. Resizing is O(n) for that one operation, but it
happens rarely enough that, *amortized* over all the cheap inserts, the average
cost per insert stays O(1).

```
load factor = 6 / 8 = 0.75  -> threshold hit, grow to 16 buckets, rehash all
```

## 6. Why average O(1), worst case O(n)

Put it together:

- **Average case — O(1).** With a good hash function and a controlled load
  factor, each key lands in a near-empty bucket. Hashing is constant work and
  the chain/probe you walk is tiny. Lookup, insert, and delete are all expected
  O(1).
- **Worst case — O(n).** If the hash function is bad (or an attacker crafts keys
  that all collide), *every* key piles into one bucket. Now a lookup degenerates
  into scanning a list of all `n` entries — the same as a plain slice. O(1) is an
  *expectation under good hashing*, not a guarantee.

This is why hash-flooding attacks matter: adversarial keys that force collisions
can turn a service's O(1) lookups into O(n) and exhaust CPU. Production hash maps
defend with **randomized hash seeds** so an attacker can't predict which keys
collide.

## 7. Go's built-in map, briefly

Go's `map` is a highly tuned hash table with separate chaining, but with a twist:
each bucket holds **8 key/value pairs** in a small struct, not a linked list.
This keeps related entries contiguous (cache-friendly) while still chaining to
**overflow buckets** when a bucket fills past 8. A few facts worth carrying:

- Go hashes map keys with a **random seed** chosen at startup, defending against
  hash-flooding — a direct application of section 6.
- It grows when the load factor gets too high, rehashing **incrementally** so a
  single insert never has to move the whole table at once.
- **Iteration order is randomized on purpose** (you saw this last lesson), partly
  so you never accidentally depend on the internal bucket layout.
- A map value is a small header (a pointer to the internal `hmap`); copying a map
  variable does *not* copy the contents — both names refer to the same table.

You don't implement any of this — you just `make(map[K]V)` and trust it. But now
you know what "trust it" means.

## 8. Using maps well in Go

Two idioms you'll use constantly. First, **comma-ok** to distinguish *absent*
from *present-with-zero-value* (recap from last lesson, but it's the heartbeat of
map code):

```go
package main

import "fmt"

func main() {
	seen := map[string]int{"a": 0}
	v, ok := seen["a"]
	fmt.Println(v, ok) // 0 true  — present, value happens to be 0
	v, ok = seen["b"]
	fmt.Println(v, ok) // 0 false — absent
}
```

Second, the **set** idiom. Go has no built-in set, so you use a map whose values
carry no information. `map[T]struct{}` is the idiomatic choice because an empty
struct occupies **zero bytes** — you pay only for keys:

```go
package main

import "fmt"

func main() {
	// A set of seen event IDs — classic dedup in a distributed pipeline.
	seen := make(map[string]struct{})

	events := []string{"e1", "e2", "e1", "e3", "e2"}
	for _, id := range events {
		if _, dup := seen[id]; dup {
			continue // already processed — skip
		}
		seen[id] = struct{}{} // mark as seen; the value is empty
		fmt.Println("processing", id)
	}
	// processing e1 / e2 / e3 — duplicates filtered
	fmt.Println("unique count:", len(seen)) // 3
}
```

That `map[T]struct{}` set *is* the deduplication layer of countless real
systems, and it's just hashing under the hood.

## Do it yourself

1. Create a scratch program (`mkdir hashmaps && cd hashmaps && go mod init example.com/hashmaps`) and a `main.go` you can run with `go run .`.
2. Type in the separate-chaining `HashTable` from section 3 and run it. Then shrink the table to `New(1)` so *everything* collides into one bucket — convince yourself `Get` still works but is now O(n).
3. Add a `LoadFactor()` method and a `Put` that resizes (double the buckets and rehash all entries) when the load factor passes 0.75. Print bucket count as you insert and watch it grow.
4. Build a word-frequency counter over a slice of strings using a `map[string]int` and comma-ok, then a *unique-words* set using `map[string]struct{}`. Compare `len` of each.
5. Read the Go blog [**Go maps in action**](https://go.dev/blog/maps) — the canonical guide to using maps idiomatically in Go.
6. Open the [**NeetCode roadmap**](https://neetcode.io/roadmap) and start the **Arrays & Hashing** section. Do *Contains Duplicate* and *Two Sum* — both are hash-set / hash-map problems and will cement why O(1) lookup changes the game.

## Check yourself

You're ready to move on when you can answer, *without looking*:

- What does a hash function do, and what two properties make one "good"?
- Why are collisions guaranteed, not just possible?
- Explain separate chaining vs open addressing, and one trade-off between them.
- What is load factor, why does resizing cost O(n) but amortize to O(1), and why does rehashing every entry become necessary?
- Why is hash-map lookup O(1) on average but O(n) in the worst case — and how does Go's random hash seed relate to that worst case?
- Why is `map[T]struct{}` the idiomatic Go set?

When all six feel obvious, commit this task on your **Roadmap** and take the
Step 2 quiz. Next lesson: **trees and binary search.**

## Common interview gotchas

- **`map[T]struct{}` is the Go set.** Zero-byte values mean you pay only for keys — prefer it over `map[T]bool`. This idiom is the right answer for *Contains Duplicate*, dedup, and "have I seen this?" filters; reaching for it is a signal interviewers listen for.
- **Hash-flooding / algorithmic DoS.** An attacker who can predict your hash crafts keys that all collide into one bucket, degrading lookups from O(1) to **O(n)** and burning CPU. Go defends with a **random hash seed** chosen at startup (also why iteration order is randomized). O(1) is an *expectation under an unpredictable hash*, not a guarantee.
- **Chaining vs open addressing.** Chaining degrades *gracefully* under high load but costs pointer chasing; open addressing is cache-friendly but degrades *badly* near-full (probe sequences explode) and needs tombstones for deletes. Load factor is the dial; both resize (~double + rehash) past a threshold.
- **Anagram counting: bytes vs runes.** Indexing a string yields **bytes**, which mis-counts multi-byte Unicode. For `a–z only` a fixed `[26]int` gives O(1) space; for full Unicode, iterate `[]rune`.
- **Top-K: heap vs bucket sort.** A size-k min-heap is O(n log k) (best when k is small); bucket sort by frequency is O(n) (best when k is large). Naming the trade-off beats the naive "sort everything by count" (O(n log n)).
