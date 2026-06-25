---
slug: project-distributed-lock
step: 8
title: "Project: a distributed lock on your KV store"
summary: Build a coordination primitive — a distributed lock — on top of your replicated key-value store.
est_min: 360
position: 5
---

# Project: a distributed lock on your KV store

> **Step 8 · Build distributed systems · Week 4 · Project**
> Concept: *coordination primitive*

You have a replicated key-value store (08-04) sitting on top of your Raft log.
That's already a serious distributed system. Now you'll build something *on top*
of it: a **distributed lock** — the primitive that lets many processes agree on
"who gets to act right now." This is the same job ZooKeeper and etcd do for the
rest of the industry, and you're about to do it with the machinery you already
have.

This is a **guide, not a solution.** The snippets sketch the shape; the design
decisions — and the subtle correctness traps — are yours to work through.

## What you'll build

A lock service exposed by your KV cluster with three operations:

- `Acquire(name, holderID) (token, ok)` — try to take the lock `name`. Succeeds
  only if nobody else holds it. Returns a **fencing token** (more on that below).
- `Release(name, holderID)` — give the lock back, but only if *you* hold it.
- A **lease/TTL** so a holder that crashes doesn't freeze the lock forever.

A lock here is nothing exotic: it's **a key whose value is the holder's id.**
The lock `deploy` is held iff the key `lock/deploy` exists with some holder's id
as its value. Acquiring is "set this key, but only if it's empty." Releasing is
"delete this key, but only if it's mine."

```go
// What a lock entry looks like in your KV store.
type LockValue struct {
    Holder  string // who holds it, e.g. "node-A:pid-1234"
    Token   uint64 // monotonic fencing token, handed out on each acquire
    Expires int64  // unix-nano lease deadline; 0 = no lease
}
```

## Why this matters

Coordination is the hardest thing to get right in a distributed system, and
almost nobody should implement it from scratch in production — they reach for a
**coordination service** instead. etcd and ZooKeeper exist precisely so that
*application* engineers can say "give me a lock" or "elect me a leader" without
re-deriving consensus. A distributed lock is the canonical example of what those
services provide. Building one yourself is how you understand what they're
actually doing — and, just as importantly, what they *can't* promise.

## The one idea that makes it work: atomic compare-and-set

A lock is only correct if "check that it's free **and** take it" happens as a
*single indivisible step.* If two nodes both read "free" and then both write
"mine," you have two holders — the bug the lock exists to prevent.

Here is the payoff of everything in Step 7: **your KV operations go through the
Raft log in a total order.** Every node applies the same commands in the same
sequence. So a **compare-and-set** ("set this key only if its current value is
empty") is naturally atomic — it's just one command in the log, applied to one
deterministic state on every replica. There is no window for a race, because the
log *is* the serialization point.

You'll add a CAS command to your store's command set:

```go
// CAS sets key=newVal only if its current value equals oldVal.
// Returned through the Raft log, so it's atomic across the cluster.
type CASCommand struct {
    Key    string
    OldVal string // "" means "must not exist"
    NewVal string
}

// applyCAS runs on every node when this command is committed and applied.
// Because the log is totally ordered, exactly one CAS can win.
func (kv *Store) applyCAS(c CASCommand) bool {
    cur, exists := kv.data[c.Key]
    if (c.OldVal == "" && exists) || (c.OldVal != "" && cur != c.OldVal) {
        return false // current state isn't what the caller expected
    }
    kv.data[c.Key] = c.NewVal
    return true
}
```

Acquire is then "CAS the lock key from empty to my holder value." Whoever's CAS
command commits *first in the log* wins; everyone else's CAS sees a non-empty
value and fails.

```go
// Acquire tries to take the lock. Returns a fencing token on success.
func (c *Client) Acquire(name, holderID string, lease time.Duration) (uint64, bool) {
    token := c.nextToken()        // monotonic — see "fencing" below
    val := encode(LockValue{
        Holder:  holderID,
        Token:   token,
        Expires: time.Now().Add(lease).UnixNano(),
    })
    ok := c.CAS("lock/"+name, "", val) // set only if currently empty
    return token, ok
}

// Release deletes the lock — but only if the caller still holds it.
func (c *Client) Release(name, holderID string) bool {
    cur, _ := c.Get("lock/" + name)
    if decode(cur).Holder != holderID {
        return false // not yours to release
    }
    return c.CAS("lock/"+name, cur, "") // compare-and-delete
}
```

## Leases: surviving a crashed holder

What if the holder crashes while holding the lock? With the code above, the key
stays set forever and the lock is permanently stuck. The fix is a **lease**: the
lock is only valid until `Expires`. A holder that wants to keep it must
**renew** (push `Expires` forward) before the deadline.

The cleanest way to enforce expiry is to check it *at apply time*, inside the
Raft state machine, so every node agrees on whether a lease is dead:

```go
// On an Acquire CAS, treat an expired lock as if it were empty.
func (kv *Store) lockIsFree(key string, now int64) bool {
    cur, exists := kv.data[key]
    if !exists {
        return true
    }
    return decode(cur).Expires != 0 && now >= decode(cur).Expires
}
```

> **Watch the clock.** `now` must come from a deterministic source the whole
> cluster agrees on — not each node's wall clock. A clean approach: the leader
> stamps the time *into the command* before it's logged, so every node applies
> the same `now`. This is the wall-clock problem from Step 7 biting you again.

## Fencing tokens: the part everyone gets wrong

Here is the trap that leases alone do **not** solve, and it's the whole reason
this project exists. Recall Step 7, Ch.8 — *process pauses & fencing tokens*:

1. Client A acquires the lock with a 10-second lease.
2. A's process is paused — a long GC pause, a hypervisor freeze, whatever. It
   doesn't know it's paused.
3. The lease expires. Client B legitimately acquires the lock.
4. A wakes up, **still believing it holds the lock**, and writes to the
   protected resource (a file, a database row, a storage bucket).

Now A and B *both* think they're the holder. The lock failed at the exact moment
it mattered. No TTL can fix this, because A can be paused for longer than *any*
timeout you pick. **The expired-holder-still-acting problem is fundamental.**

The real fix is a **fencing token**: a number the lock service increments on
every successful acquire and returns to the holder. The holder includes its
token on every write to the protected resource, and **the resource rejects any
write whose token is lower than the highest it has already seen.**

```go
// At the protected resource (storage server, DB shim, etc.):
type Fence struct{ maxSeen uint64 }

// Write accepts the request only if its fencing token is the newest one.
func (f *Fence) Write(token uint64, payload []byte) error {
    if token < f.maxSeen {
        return fmt.Errorf("stale token %d (already saw %d)", token, f.maxSeen)
    }
    f.maxSeen = token
    return doWrite(payload)
}
```

In the scenario above, B acquired *after* A, so B's token is higher. The resource
sees B's higher token, advances `maxSeen`, and then A's delayed write — carrying
the *older*, smaller token — is rejected. The lock holder no longer has to be
trusted to be honest or awake; **the resource enforces the ordering itself.**

Making the token monotonic is easy in your design: keep a counter key in the KV
store and increment it (via the log, atomically) on each acquire, or just reuse
the Raft log index of the acquire command — it's already monotonic and totally
ordered.

## Milestones

Build it in slices, testing each before moving on:

1. **CAS command.** Add `CASCommand` to your store, route it through Raft, and
   apply it deterministically. Unit-test `applyCAS` directly.
2. **Acquire / Release.** Build the lock key convention on top of CAS. Test that
   two concurrent `Acquire`s on a free lock yield exactly one winner.
3. **Leases.** Add `Expires` and renewal; make apply-time expiry deterministic
   with a leader-stamped `now`. Test that a lock frees up after a crashed holder.
4. **Fencing tokens.** Hand out a monotonic token on acquire; build a tiny
   protected "resource" that rejects stale tokens. **Write the paused-holder
   test:** acquire as A, let the lease expire, acquire as B, then have A try to
   write — assert it's rejected.
5. **Glue.** Expose `Acquire`/`Release` over your client RPC so a real program
   can use the lock across the cluster.

## Common pitfalls

- **Lock never released on crash (no lease).** If you skip TTLs, one crash
  freezes the lock forever. The lease is mandatory, not a stretch goal.
- **Trusting the TTL alone.** This is *the* classic mistake. A lease bounds how
  long a *healthy* holder keeps the lock, but a paused holder can outlive any
  lease and still act. **Fencing tokens — not TTLs — are the real fix.** If your
  design has leases but no fencing, it is subtly broken.
- **Release by anyone.** `Release` must verify the caller is the current holder,
  or a slow client can release a lock that's since been re-acquired by someone
  else (and you're back to two holders).
- **Non-deterministic expiry.** Checking `time.Now()` independently on each node
  makes replicas disagree about whether the lease is dead — a split state. Stamp
  the time into the logged command.
- **Read-modify-write instead of CAS.** `Get` then `Put` is two log entries with
  a gap between them; another acquire can slip in. The whole point is to make it
  *one* atomic command.

## Free resources

- [**etcd**](https://github.com/etcd-io/etcd) — read how its lock recipe works:
  leases, key creation, and how clients are told to use the response's
  *revision* as a fencing token. This is your project, productionized.
- **DDIA, Ch.8 — "The Truth Is Defined by the Majority."** Kleppmann's fencing-
  token discussion (the paused-client diagram) is exactly the trap in this
  project. Re-read it after you've written your paused-holder test; it'll click.
- The [**Raft paper**](https://raft.github.io/raft.pdf) — your CAS command is
  just another log entry; revisit §5 to remember *why* that makes it atomic.

## Done when

- [ ] A `CASCommand` flows through your Raft log and `applyCAS` is correct and
      deterministic, with a direct unit test.
- [ ] Two concurrent `Acquire`s on a free lock produce **exactly one** winner.
- [ ] `Release` succeeds only for the current holder; a non-holder is rejected.
- [ ] A lock held by a **crashed** holder is reacquirable after its lease expires.
- [ ] `Acquire` returns a **monotonic fencing token**, and a protected resource
      **rejects a stale token** — proven by a paused-holder test.
- [ ] You can articulate, in one sentence, why a TTL alone is not enough.

When this passes, mark the project complete on your **Roadmap**. You've built a
coordination primitive on top of consensus — the layer real infrastructure runs
on. Next up you'll see what it takes to operate one of these for real.

## Common interview gotchas

- **"A short enough TTL makes the lock safe."** No TTL is small enough. A paused holder (GC, hypervisor freeze) can outlive *any* lease and wake up still believing it holds the lock — the expired-holder-still-acting problem is fundamental, not tunable.
- **"The lock service enforcing mutual exclusion is sufficient."** Only fencing tokens *checked at the resource* make it safe. The lock granting exclusivity means nothing if a delayed write from a stale holder still lands; the resource must reject any token below the highest it has seen.
- **"Any monotonic counter works as a fencing token."** It must be monotonic *and* derived from the replicated log (a log-backed counter or the acquire's log index). A token minted from local state or wall-clock can go backwards across leader changes and stops being a valid fence.
- **"Check the lease with `time.Now()` at apply time."** Each node's wall clock differs, so replicas disagree on whether a lease is dead — split state. The leader must stamp `now` into the command before it's logged so every replica applies the same deterministic time.
- **"Acquire is read-the-key-then-write-if-free."** That's two log entries with a gap a competing acquire slips through, yielding two holders. It must be a single atomic CAS command — the log's total order is the only thing making "check and take" indivisible.
