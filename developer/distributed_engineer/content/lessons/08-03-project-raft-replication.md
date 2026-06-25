---
slug: project-raft-replication
step: 8
title: "Lab: Raft log replication & persistence (6.824 Lab 2B/2C)"
summary: Extend Raft with log replication, commitment by majority, and crash-safe persistence — the hard part.
est_min: 840
position: 3
---

# Lab: Raft log replication & persistence (6.824 Lab 2B/2C)

> **Step 8 · Build distributed systems · Week 3 · Lab**
> Concept: *the hard part of Raft*

In Lab 2A you got an election working: a single leader emerges, sends
heartbeats, and survives crashes. That was the warm-up. **This lab is where Raft
gets hard.** You'll make the leader actually *replicate a log*, commit entries
once a majority agree, apply them to a state machine, and — in 2C — survive a
crash and come back with its memory intact.

The election code barely changes. What changes is that every edge case the paper
warns about now becomes a test that fails at 3am unless you got the rules exactly
right. Treat **Figure 2 of the Raft paper as law**: not a summary, not a
suggestion — a literal specification you implement line by line.

## What you'll build

Building on your Lab 2A `Raft` struct (and the consensus theory from Step 7),
you'll add:

- **`AppendEntries`** that carries real log entries, performs the
  `prevLogIndex`/`prevLogTerm` **consistency check**, and backs up on mismatch.
- A **commit rule**: the leader advances `commitIndex` once a *majority* of peers
  have replicated an entry — but only commits entries from its **current term**.
- An **apply loop** that pushes committed entries, in order, down an
  `applyCh` to the caller's state machine.
- **Persistence (2C)**: `currentTerm`, `votedFor`, and the `log` are saved to
  stable storage *before* you reply to any RPC, and restored on restart.

By the end, `go test -run 2B` and `go test -run 2C` pass — repeatedly, and with
`-race`.

## Why this matters

Lab 2A proved you could elect a leader. But a leader that can't durably agree on
a *sequence* of commands is useless — that agreement is the entire point of
consensus (Step 7). Log replication is what etcd, Consul, and CockroachDB do
millions of times a second. And persistence is the difference between a toy that
forgets everything on reboot and a real system that a database can trust its data
to.

This is also the lab that teaches you to **think adversarially**. The 6.824 test
harness deliberately partitions the network, kills leaders mid-replication,
delays and reorders messages, and then checks that no two nodes ever applied a
*different* command at the same log index. If your code has a single off-by-one
or a missed persist, the tests *will* find it — sometimes on the 50th run. That
paranoia is the job.

## Background: the log

A log entry is tiny. The shape the tests expect:

```go
type LogEntry struct {
	Term    int         // term in which the leader created this entry
	Command interface{} // opaque to Raft; meaningful to the state machine
}
```

Index your log so that **index 1 is the first real entry**. The cleanest trick
is a dummy entry at index 0 (term 0) so `rf.log[0]` always exists and
`prevLogIndex = 0` is a valid, always-matching base case. Decide this now and
write a helper `lastLogIndex()` / `lastLogTerm()` — half the bugs in this lab are
disagreements between two pieces of your own code about what "index" means.

## Milestone 1 — AppendEntries carries entries

Extend the RPC from heartbeat-only to the full Figure 2 shape:

```go
type AppendEntriesArgs struct {
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
	// 2B optimization fields — see Milestone 4
}
```

The follower's handler runs these checks **in order**:

1. Reply `false` if `args.Term < rf.currentTerm` (stale leader).
2. Reset the election timer (you heard from a current leader).
3. **Consistency check:** reply `false` if your log has no entry at
   `PrevLogIndex`, or the term there isn't `PrevLogTerm`.
4. If an existing entry conflicts with a new one (same index, different term),
   delete it *and everything after it*, then append the new entries.
5. If `LeaderCommit > commitIndex`, set
   `commitIndex = min(LeaderCommit, index of last new entry)`.

Step 4 has a trap: do **not** blindly truncate. If the RPC is a stale duplicate,
truncating throws away good entries you already committed. Only truncate at the
*first* point of disagreement. The Students' Guide spells this out — read it.

## Milestone 2 — the leader replicates

The leader keeps two arrays, **reinitialized after every election**:

- `nextIndex[i]` — the next log index to send peer `i` (init to `lastLogIndex+1`).
- `matchIndex[i]` — the highest index known replicated on `i` (init to 0).

For each follower, send `log[nextIndex[i]:]` with the matching `PrevLogIndex`/
`PrevLogTerm`. On the reply:

- **Success:** advance `matchIndex[i]` and `nextIndex[i]` based on **what you
  sent**, not on your current log length (which may have grown). A common bug is
  `matchIndex[i] = len(rf.log)-1`; use `args.PrevLogIndex + len(args.Entries)`.
- **Failure** (consistency check failed): back `nextIndex[i]` up and retry.

## Milestone 3 — commit by majority, apply in order

After updating `matchIndex`, the leader scans for an index `N` such that a
**majority** of `matchIndex` values are `>= N`. The critical rule:

```go
// Only commit an entry from the CURRENT term directly.
if rf.log[N].Term == rf.currentTerm && majorityHave(N) {
	rf.commitIndex = N
}
```

That `Term == currentTerm` guard is **not optional** and not an optimization. It
closes the Figure-8 hole from the paper: a leader counting replicas of a
*previous* term's entry can commit something that a later leader then overwrites.
Older entries get committed *indirectly* — once a current-term entry on top of
them commits, they're safe.

Applying is separate from committing. Run a single dedicated goroutine (or a
condition variable) that, whenever `commitIndex > lastApplied`, increments
`lastApplied` and sends one `ApplyMsg` per entry, **strictly in index order**:

```go
msg := ApplyMsg{
	CommandValid: true,
	Command:      rf.log[rf.lastApplied].Command,
	CommandIndex: rf.lastApplied,
}
applyCh <- msg
```

Never send on `applyCh` while holding the lock — the receiver may block, and
you'll deadlock. Copy what you need under the lock, release it, then send.

## Milestone 4 — fast log backup

The naive "decrement `nextIndex` by one per failed RPC" works but is achingly
slow when a follower is far behind (a test will time out). Add the optimization
the Students' Guide describes: the follower returns a `ConflictTerm` and the
first index it stored for that term, letting the leader **skip an entire
conflicting term** in one round-trip. Get correctness with the slow version
first, then add this if 2B times out.

## Milestone 5 — persistence (2C)

Whenever `currentTerm`, `votedFor`, or `log` changes, call `rf.persist()` —
encode them with `labgob` into the provided `Persister`. Read them back in
`readPersist()` at startup.

The rule that matters: **persist before you reply.** If a follower grants a vote
or accepts entries, then crashes before persisting, on restart it could vote
again or lose entries — and your cluster has two leaders or a hole in the log.
Call `persist()` at the end of any handler that mutated that state, *before* the
`return`.

```go
func (rf *Raft) persist() {
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.currentTerm)
	e.Encode(rf.votedFor)
	e.Encode(rf.log)
	rf.persister.Save(w.Bytes(), nil)
}
```

## Common pitfalls

- **Off-by-one log indices.** Mixing 0-based slices with 1-based log indices is
  the #1 time sink. Pick the dummy-entry-at-0 convention and use helpers
  everywhere; never index `rf.log` with a raw number you computed inline.
- **Applying out of order (or twice).** Only one goroutine may send on
  `applyCh`, and it must go strictly `lastApplied+1, +2, …`. Sending from inside
  the AppendEntries handler races with the leader path — centralize it.
- **Not persisting before replying.** Persisting in the wrong place, or
  forgetting it after `votedFor` changes, passes 2B and silently fails 2C under
  crashes. Audit every state mutation.
- **Livelock / repeated elections.** If you reset the election timer on a
  *rejected* AppendEntries, or forget to reset it when granting a vote, the
  cluster keeps re-electing and never makes progress. Reset only on the events
  Figure 2 names.
- **`matchIndex` from log length.** Set it from what you *sent*, not from
  `len(rf.log)` — the log grows under you while RPCs are in flight.
- **Sending on `applyCh` under the lock.** Classic deadlock. Release first.

## Done when

- [ ] `go test -run 2B` passes — log replication, including with a follower far
      behind and with an unreliable network.
- [ ] `go test -run 2C` passes — state survives crash and restart.
- [ ] Both pass **repeatedly** (run them 20–50 times in a loop, not once).
- [ ] Both pass with **`go test -race -run 2B`** and `-race -run 2C` — no data
      races reported.
- [ ] No two nodes ever apply a different command at the same index (the harness
      checks this; if it ever fires, your commit or consistency logic is wrong).

```bash
go test -run 2B
go test -run 2C
go test -race -run 2C
for i in $(seq 1 30); do go test -run 2C || break; done
```

If a test fails one run in thirty, you are not done — that flake is a real bug
the harness will reproduce under grading. Chase it.

## Common interview gotchas

- **"A leader commits an entry as soon as a majority store it."** Only for entries from its *own* term. Committing a replicated prior-term entry by counting replicas is the Figure-8 bug — a future leader can still overwrite it. Older entries commit indirectly, on the back of a current-term entry above them.
- **"Persist whenever you get a free moment / after replying."** Persist `currentTerm`, `votedFor`, and `log` *before* the RPC reply leaves. A crash after replying-but-before-persisting lets a node vote twice or lose accepted entries — two leaders or a log hole.
- **"`commitIndex` and `lastApplied` are basically the same thing."** Commit is "a majority has it"; apply is "the state machine ran it." They advance independently, apply lags commit, and apply must be strictly in index order from a single point — or replicas diverge.
- **"On an AppendEntries rejection, just resend the same thing."** You must back `nextIndex` down (ideally by whole conflicting term) and retry; advance `matchIndex`/`nextIndex` from what you *sent*, never from `len(log)`, which grows under in-flight RPCs.
- **"Re-applying a committed entry after restart is harmless."** Only if apply is idempotent. Raft guarantees an entry is *committed* once, not *applied* once across a crash — non-idempotent state machines double-apply unless you track `lastApplied` durably or dedup downstream.

## Free resources

- [**6.824 Lab 2**](https://pdos.csail.mit.edu/6.824/labs/lab-raft.html) — the
  official lab spec. Read the 2B and 2C sections word for word; the hints are
  earned in blood.
- [**The Raft paper, Figure 2**](https://raft.github.io/raft.pdf) — your
  specification. Keep it open. Every rule in this lab is on that one page.
- [**Students' Guide to Raft**](https://thesquareplanet.com/blog/students-guide-to-raft/)
  — written by a 6.824 TA, it lists the exact mistakes you're about to make
  (the apply loop, the commit rule, the backup optimization). Read it *before*
  you start, then again when you're stuck.

When 2B and 2C are green and stable under `-race`, commit this on your
**Roadmap**. You've built the hard core of a real consensus engine. Next: **Lab
2D — snapshots** to compact that ever-growing log, then a replicated key-value
store on top.
