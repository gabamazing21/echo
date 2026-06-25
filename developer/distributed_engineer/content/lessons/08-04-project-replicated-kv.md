---
slug: project-replicated-kv
step: 8
title: "Project: a replicated key-value store on Raft"
summary: Put a Get/Put/Append KV state machine on top of your Raft log — a real fault-tolerant service.
est_min: 720
position: 4
---

# Project: a replicated key-value store on Raft

> **Step 8 · Build distributed systems · Week 4 · Project**
> Concept: *putting it together*

You implemented Raft in the last three labs. On its own, Raft is just an
agreed-upon **log of bytes** — impressive, but not yet a service anyone can use.
This project is the payoff: you wrap that log in a tiny **key-value store** so a
client can `Get`, `Put`, and `Append` strings, and the cluster keeps serving
those operations *correctly* even when the leader crashes and a follower takes
over. This is MIT 6.824 **Lab 3** in spirit, and it is where consensus stops
being theory and starts being a database.

This is a **guide, not a solution.** The snippets below show shape, not the full
implementation — the learning is in filling the gaps and watching the lab tests
go green.

## What you'll build

A replicated KV service with two layers:

- A **client** (clerk) that exposes `Get(key)`, `Put(key, value)`, and
  `Append(key, value)` and hides all the messy retrying.
- A **server** that sits on top of your Raft peer. Every client operation is
  submitted to Raft, committed by a majority, then **applied in log order** to an
  in-memory `map[string]string` on *every* replica — so all replicas hold
  identical state.

By the end, a value `Put` while node 0 was leader is still readable after node 0
is killed and node 2 becomes leader. The data survived a leadership change
because it lives in the *replicated log*, not in any one server's memory.

## Why this matters

This is what **etcd** and **Consul** literally *are*. Strip away the HTTP API,
the watch streams, and the TLS, and etcd is a key-value map applied on top of a
Raft log — exactly the thing you are about to build. Kubernetes stores its
entire cluster state in etcd; when you understand this project, you understand
the beating heart of Kubernetes. There is no more convincing line on a
distributed-engineer résumé than "I built a linearizable replicated KV store on
my own Raft."

It also forces you to confront a problem that pure Raft let you ignore:
**exactly-once semantics for client requests.** That problem — and its fix — is
the real intellectual content of this lab.

## 1. The core loop: submit, commit, apply

Every operation follows the same path. The client sends an op to the server it
thinks is leader. The server hands it to Raft via `Start()`, which returns
immediately — the op is *not done yet*, only appended to the leader's log. Raft
replicates it; once a majority store it, Raft commits it and delivers it back to
**every** server through a channel (the `applyCh`). A background goroutine on
each server reads that channel and applies ops to the map *in log order*.

Crucially, the server does **not** touch the map when `Start()` returns. It waits
until the op comes back out of `applyCh`. That is the whole discipline: Raft
decides the order, and you only mutate state when Raft tells you an entry is
committed.

```go
type Op struct {
	Kind     string // "Get", "Put", or "Append"
	Key      string
	Value    string
	ClientID int64  // who sent it
	Seq      int64  // this client's request number (for dedup)
}
```

`Op` is the thing you log. It must be serialisable, because Raft persists it and
ships it over RPC — keep it to plain exported fields (register it with
`labgob` if your lab harness uses gob).

## 2. The apply loop

This single goroutine is the heart of the server. It is the *only* place the map
is mutated, which is what keeps every replica identical.

```go
func (kv *KVServer) applier() {
	for msg := range kv.applyCh {
		if !msg.CommandValid {
			continue // snapshot or no-op; handle separately
		}
		op := msg.Command.(Op)

		kv.mu.Lock()
		// Dedup: only apply a write the first time we see this Seq.
		if op.Kind != "Get" && op.Seq > kv.lastSeq[op.ClientID] {
			switch op.Kind {
			case "Put":
				kv.store[op.Key] = op.Value
			case "Append":
				kv.store[op.Key] += op.Value
			}
			kv.lastSeq[op.ClientID] = op.Seq
		}
		// Wake the RPC handler waiting on this log index, if any.
		if ch, ok := kv.waiters[msg.CommandIndex]; ok {
			ch <- applyResult{value: kv.store[op.Key]}
		}
		kv.mu.Unlock()
	}
}
```

Note what `applier` does *not* do: it does not check whether this server is the
leader. Followers run the exact same loop and reach the exact same map. The
leader is special only in that it has an RPC handler *waiting* (via `waiters`) to
reply to the client.

## 3. The RPC handler: submit and wait

When a client RPC arrives, the server calls `Start()`. If it isn't the leader,
`Start()` says so and the handler returns `ErrWrongLeader` immediately — the
client will go find the real leader. If it is the leader, the handler registers
a waiter channel for the returned log index and blocks until the applier signals
that *that index* committed (or a timeout fires).

```go
func (kv *KVServer) Submit(args *Op) (string, Err) {
	index, _, isLeader := kv.rf.Start(*args)
	if !isLeader {
		return "", ErrWrongLeader
	}

	kv.mu.Lock()
	ch := make(chan applyResult, 1)
	kv.waiters[index] = ch
	kv.mu.Unlock()

	select {
	case res := <-ch:
		return res.value, OK
	case <-time.After(500 * time.Millisecond):
		return "", ErrTimeout // lost leadership? client will retry
	}
}
```

The timeout matters: if this server *was* leader when it called `Start()` but
lost the election before the entry committed, the entry may be overwritten and
will never come back on `applyCh`. The timeout lets the handler give up so the
client can retry elsewhere.

## 4. The client (clerk): retry until it sticks

The client's job is to make the cluster *look* like a single reliable machine.
It remembers who it thinks the leader is, and on `ErrWrongLeader` or
`ErrTimeout` it simply tries the next server, looping forever until one accepts.

```go
func (ck *Clerk) Put(key, value string) {
	args := Op{
		Kind:     "Put",
		Key:      key,
		Value:    value,
		ClientID: ck.id,
		Seq:      atomic.AddInt64(&ck.seq, 1), // unique, increasing
	}
	for {
		srv := ck.servers[ck.leader]
		var reply Reply
		ok := srv.Call("KVServer.PutAppend", &args, &reply)
		if ok && reply.Err == OK {
			return
		}
		ck.leader = (ck.leader + 1) % len(ck.servers) // try next
	}
}
```

That `Seq` field is the key. The clerk picks a **monotonically increasing
sequence number per request** and *reuses the same number on every retry of the
same operation*. The server uses it to tell a genuinely new `Put` from a retry of
one it already applied.

## 5. The duplicate-request problem

Here is the subtle bug that this lab is really about. A client sends `Append("x",
"a")`. The leader logs it, commits it, applies it — `x` becomes `…a` — and then
**crashes before the reply reaches the client.** The client times out and
retries the *same* `Append` against the new leader. If the server applies it
again, `x` becomes `…aa`. The operation was committed exactly once by Raft, but
*applied twice* by the state machine. Reads are harmless to re-run; `Put` is
idempotent if the value is the same; but `Append` is destructive when duplicated.

This is **idempotency from Step 4**, now at the system level. The fix is a
**dedup table**: each server remembers the highest `Seq` it has applied for each
client, and refuses to re-apply anything at or below that number.

```go
// Per server, mutated only inside the apply loop:
lastSeq map[int64]int64 // clientID -> highest Seq applied
```

That single check in the applier (`op.Seq > kv.lastSeq[op.ClientID]`) is what
turns "Raft committed it once" into "the map applied it once." Because the dedup
table is updated *inside the apply loop*, every replica makes the same decision
in the same order — so the dedup state is itself replicated and consistent. A
client only advances its `Seq` once a request *succeeds*, so there is never more
than one outstanding request per client to worry about.

## Milestones

Build it in stages and test after each — do not write the whole thing then
debug a wall of red.

1. **Single-server, no crashes.** Wire up `Op`, the apply loop, the RPC handlers,
   and the clerk. Get `Put`/`Append`/`Get` working against one server. The lab's
   basic tests should pass.
2. **Multi-server, leader changes.** Make the clerk find the leader by retrying.
   Confirm a value survives killing the leader. This exercises your Raft from the
   earlier labs hard — many Lab 3 "bugs" are actually Lab 2 bugs surfacing.
3. **Dedup table.** Add `ClientID`/`Seq` and `lastSeq`. Pass the tests that send
   duplicate requests under unreliable networks (these fail loudly without dedup).
4. **Snapshots (Lab 3B).** When the log grows past a size threshold, snapshot the
   map *and the dedup table together* and hand it to Raft; restore both on
   restart. The dedup table is part of your state — snapshot it or duplicates
   return after a restart.

## Common pitfalls

- **Applying before commit.** Never mutate the map in the RPC handler or right
  after `Start()`. Only the apply loop, reading committed entries from
  `applyCh`, may touch state. Violate this and replicas diverge.
- **Duplicate application.** Skipping the dedup check makes `Append` tests fail
  intermittently under packet loss. Update `lastSeq` *atomically with* applying
  the write, inside the same locked section.
- **Leader change mid-request.** A server can call `Start()` as leader, then lose
  leadership before the entry commits. The committed entry at that index may end
  up being a *different* op. Always check the op that actually came back at your
  index matches the one you submitted before replying OK — don't trust the index
  alone.
- **Deadlock between the KV mutex and Raft.** Never hold `kv.mu` while calling
  into Raft (`kv.rf.Start`, or anything that might block on Raft's own lock), and
  never let the applier block sending on a waiter channel. Use **buffered**
  waiter channels (`make(chan ..., 1)`) and drop the result if no one is
  listening, or you will deadlock the apply loop and freeze the whole server.
- **Stale waiters.** If you registered a waiter for an index and a *different* op
  commits there (you lost leadership), the original client will hang until its
  timeout. That is fine — just make sure you eventually clean up the `waiters`
  map so it doesn't leak.

## Free resources

- [**MIT 6.824**](https://pdos.csail.mit.edu/6.824/) — the course this project
  comes from. Read the Lab 3 (KV Raft) page closely; the test names tell you
  exactly which failure each one hunts for, and the lecture on this lab walks the
  duplicate-detection design in detail.
- [**etcd source**](https://github.com/etcd-io/etcd) — the production version of
  what you just built. Look at the `raft/` package and how the server applies
  committed entries to its store; you will recognise the apply loop. Reading real
  code after building your own is where the mental model locks in.

## Done when

- [ ] A single server passes the basic `Get`/`Put`/`Append` tests.
- [ ] A value `Put` while one node is leader is **readable after that node is
      killed** and a follower takes over.
- [ ] Duplicate/retried `Append`s under an unreliable network do **not** apply
      twice — the dedup table holds.
- [ ] The map is mutated **only** in the apply loop, never in an RPC handler.
- [ ] No deadlocks: the suite runs to completion, repeatedly, without hanging.
- [ ] (3B) Snapshots include the dedup table, and state survives restart.
- [ ] The full Lab 3 test suite passes — run it many times; concurrency bugs are
      intermittent.

When this goes green you have built, end to end, a fault-tolerant replicated
service on consensus you implemented yourself. That is the capstone of the whole
track — the thing every "distributed systems" job description is gesturing at.
Commit it on your **Roadmap** with pride.

## Common interview gotchas

- **"Raft committing each op once gives exactly-once application."** No — a leader can apply then crash before replying; the client retries and the op applies twice. Exactly-once is a *state-machine* property you add with a dedup table keyed by `client_id+seq`, not something Raft hands you.
- **"The snapshot just needs the map."** It must also include the dedup table (`lastSeq` per client). Snapshot the map alone and after a restore the highest-seen seqs reset to zero, so already-applied retries re-apply — duplicates reappear precisely after recovery.
- **"A leader can serve reads straight from its local map — it's the leader."** A partitioned stale leader still thinks it's leader and returns stale data. Linearizable reads must go through the log (a no-op/read entry) or use a read lease tied to a confirmed heartbeat quorum.
- **"Matching log index at commit means it's my op."** Leadership can change between `Start()` and commit, so a *different* op may occupy your index. Verify the committed op's `client_id+seq` matches what you submitted before replying OK — trusting the index alone is a correctness bug.
- **"Bump the client's seq on every retry to keep it unique."** The opposite: reuse the *same* seq across retries of one logical op, advancing only after success. Incrementing per attempt defeats the dedup table and reintroduces the double-apply it exists to stop.

When this goes green you have built, end to end, a fault-tolerant replicated
service on consensus you implemented yourself.
