---
slug: project-raft-election
step: 8
title: "Lab: Raft leader election (6.824 Lab 2A)"
summary: Implement Raft leader election — terms, RequestVote, randomized timeouts and heartbeats, in Go.
est_min: 840
position: 2
---

# Lab: Raft leader election (6.824 Lab 2A)

> **Step 8 · Build distributed systems · Week 2 · Lab**
> Concept: *consensus in practice*

This is the lab the whole roadmap has been pointing at. In **Step 7 (ds-raft)**
you read the Raft paper and built the mental model: one leader, one log, terms
as a logical clock, majority quorums. Now you *implement* it. Lab 2A is the
first slice — **leader election only**, no log replication yet. Get this right
and the rest of Raft (2B replication, 2C persistence, 2D snapshots) is built on
solid ground. Get the concurrency wrong here and you'll fight phantom bugs for
weeks.

This is a **guide, not a solution.** It tells you what 2A demands, sketches the
structures, and points at the traps. You write the code — that's the whole
point.

## What you'll build

The MIT 6.824 skeleton gives you a `raft` package with stub methods. For Lab 2A
you fill in just enough to make a cluster of `Raft` peers **elect exactly one
leader** and keep electing one as leaders fail. Concretely, 2A's tests check
three things:

- **Elect a leader once, and only one.** Start N peers; within a fraction of a
  second exactly one should declare itself leader for the term.
- **Re-elect after the leader fails.** Disconnect the leader; the survivors must
  pick a new leader. Reconnect the old one; it must *step down* (it'll see a
  higher term) and not cause a second leader.
- **Respect terms.** Terms only increase. A node that sees a higher term in any
  message immediately reverts to follower and adopts that term.

No client commands, no log entries to apply yet — heartbeats in 2A are **empty
AppendEntries** RPCs whose only job is to say "the leader is alive, reset your
timer."

## Why this matters

Leader election is where the paper's prose becomes running code, and it is the
single richest source of distributed-systems bugs you'll ever debug: races,
deadlocks, split votes, stale leaders. Doing it under the 6.824 test harness —
which injects network failures, delays, and partitions — is exactly the skill
that makes "I implemented Raft" a credible line on a résumé. Companies don't run
your toy KV store; they trust that you understand *why* the lock discipline and
term rules are the way they are.

And it's unforgiving in a useful way: the tests are deterministic about
correctness but stochastic about timing, so a subtle race shows up as a *flaky*
failure. Learning to make `go test -race -run 2A` pass **repeatedly** teaches
you more about concurrency than any tutorial.

## Read these first (and keep them open)

- The **6.824 Lab 2 page** — [Raft labs](https://pdos.csail.mit.edu/6.824/labs/lab-raft.html).
  It defines the API (`Make`, `GetState`, `Start`, `Kill`), the RPC plumbing
  (`labrpc`), and the test commands. Read the 2A section twice.
- The **Raft paper** — [raft.pdf](https://raft.github.io/raft.pdf). You will
  live inside **Figure 2.** Print it. Tape it to your monitor. Most 2A bugs are
  a rule from Figure 2 you forgot to implement.

## The structures

A `Raft` peer is a state machine guarded by one mutex. Sketch:

```go
type State int

const (
	Follower State = iota
	Candidate
	Leader
)

type Raft struct {
	mu        sync.Mutex          // guards ALL fields below
	peers     []*labrpc.ClientEnd // RPC endpoints of every peer (incl. self)
	me        int                 // this peer's index into peers[]

	// Persistent-ish state (you'll truly persist it in 2C; track it now).
	currentTerm int  // latest term this peer has seen, starts at 0
	votedFor    int  // candidate voted for in currentTerm, -1 if none

	// Volatile state.
	state         State
	electionReset time.Time // last time we heard from a leader / granted a vote
}
```

Two notes that save hours:

- `votedFor` is **per term.** Whenever `currentTerm` changes you reset
  `votedFor` to -1. Bake that into a single helper (see below) so you can't
  forget.
- Track the election deadline however you like — a `time.Time` you compare
  against, or a resettable timer. The `time.Time` approach (a goroutine that
  wakes periodically and checks "has it been too long?") is the easiest to
  reason about and the hardest to deadlock.

## The one rule that prevents most bugs: step down on a higher term

Every RPC handler and every RPC *reply* must do the same check first. Centralise
it:

```go
// caller MUST hold rf.mu
func (rf *Raft) becomeFollowerLocked(term int) {
	rf.state = Follower
	rf.currentTerm = term
	rf.votedFor = -1
	rf.electionReset = time.Now()
}
```

If a message's term is greater than `rf.currentTerm`, call this. If it's *less*,
reject the message (reply with your own higher term so the sender steps down).
This is the "stale leader is harmless" mechanism from Step 7, made concrete.

## RequestVote — the heart of 2A

The args and reply mirror Figure 2 exactly. For 2A there are no log entries, so
the "is the candidate's log up to date" check is trivially true — but write the
fields now; 2B needs them.

```go
type RequestVoteArgs struct {
	Term         int
	CandidateID  int
	LastLogIndex int // unused in 2A, present for 2B
	LastLogTerm  int // unused in 2A, present for 2B
}

type RequestVoteReply struct {
	Term        int
	VoteGranted bool
}

func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.Term = rf.currentTerm
	if args.Term < rf.currentTerm {
		reply.VoteGranted = false // stale candidate
		return
	}
	if args.Term > rf.currentTerm {
		rf.becomeFollowerLocked(args.Term)
		reply.Term = rf.currentTerm
	}

	// Grant the vote if we haven't already this term (and, in 2B, the
	// candidate's log is at least as up-to-date as ours).
	if rf.votedFor == -1 || rf.votedFor == args.CandidateID {
		rf.votedFor = args.CandidateID
		reply.VoteGranted = true
		rf.electionReset = time.Now() // granting a vote counts as "heard from"
	} else {
		reply.VoteGranted = false
	}
}
```

The send side is a thin wrapper the harness expects:

```go
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs,
	reply *RequestVoteReply) bool {
	return rf.peers[server].Call("Raft.RequestVote", args, reply)
}
```

## The election loop

A long-running goroutine (start it in `Make`) watches the clock. When the
randomized timeout elapses without hearing from a leader, it becomes a candidate
and runs an election. The randomization is what breaks split votes — pick a
fresh timeout (e.g. 150–300 ms) every time you reset.

```go
func (rf *Raft) ticker() {
	for !rf.killed() {
		timeout := time.Duration(150+rand.Intn(150)) * time.Millisecond
		time.Sleep(10 * time.Millisecond) // wake often, decide rarely

		rf.mu.Lock()
		if rf.state != Leader &&
			time.Since(rf.electionReset) >= timeout {
			rf.startElectionLocked() // becomes candidate, fires RPCs
		}
		rf.mu.Unlock()
	}
}
```

`startElectionLocked` increments `currentTerm`, sets `state = Candidate`, votes
for itself (`votedFor = rf.me`), resets the timer, then launches **one goroutine
per peer** to call `sendRequestVote`. Each goroutine, on a granted vote, takes
the lock, re-checks it's *still* a candidate in *the same term* (it may have
stepped down or the term may have moved on), counts votes, and on reaching a
majority transitions to leader and starts heartbeats.

> **The critical detail:** the RPC call itself happens **without the lock held.**
> You read what you need under the lock, release it, do the network call, then
> re-acquire to apply the result. Holding the lock across `Call` is the classic
> Raft deadlock (below).

## Heartbeats: empty AppendEntries

Once a peer wins, it must immediately and *periodically* (every ~100 ms, well
under the election timeout) send AppendEntries to every follower. In 2A these
carry no entries — they exist purely to reset followers' election timers so they
don't start spurious elections.

```go
type AppendEntriesArgs struct {
	Term     int
	LeaderID int
	// 2B adds PrevLogIndex/Term, Entries[], LeaderCommit.
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs,
	reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.Term = rf.currentTerm
	if args.Term < rf.currentTerm {
		reply.Success = false // stale leader
		return
	}
	if args.Term > rf.currentTerm {
		rf.becomeFollowerLocked(args.Term)
	}
	rf.state = Follower            // a valid leader exists this term
	rf.electionReset = time.Now()  // <-- DO NOT forget this reset
	reply.Term = rf.currentTerm
	reply.Success = true
}
```

The heartbeat sender is a loop the new leader runs (one goroutine), firing a
round of `AppendEntries` every 100 ms while it remains leader.

## Milestones

Build it in this order; don't move on until the previous step is solid.

1. **Skeleton compiles, `Make` starts `ticker`.** Wire the struct, the state
   enum, and `becomeFollowerLocked`. `go test -run 2A` will fail, but cleanly.
2. **RequestVote handler + send wrapper.** Two peers, by hand, can grant and
   count a vote.
3. **Election fires on timeout.** A single isolated peer repeatedly times out
   and increments its term (watch it climb in logs).
4. **One leader elected.** Full cluster: exactly one peer reaches Leader for the
   first term. (`TestInitialElection2A`)
5. **Heartbeats keep the peace.** With the leader alive, no follower starts a new
   election; the term stays stable.
6. **Re-election after failure.** Kill the leader → a new one appears; reconnect
   the old → it steps down. (`TestReElection2A`)

## Common pitfalls

- **Holding the lock across an RPC — the classic deadlock.** Peer A calls
  `RequestVote` on B while holding A's lock; B's handler tries to take *its* lock,
  fine — but if your code ever calls out to a peer while holding the lock and
  that path can re-enter, you deadlock the whole cluster and tests hang. Rule:
  **read state under the lock, release, then `Call`, then re-lock to use the
  reply.**
- **Forgetting to reset the election timer.** Both on a *valid* AppendEntries and
  on *granting* a vote, you must reset `electionReset`. Miss it and followers
  start elections under a healthy leader — you'll see the term creep upward
  forever and the test reports "term changed even though there were no failures."
- **Re-check after re-locking.** When a vote reply arrives, the world may have
  moved: you might no longer be a candidate, or `currentTerm` may have advanced.
  Always verify `rf.state == Candidate && rf.currentTerm == termWhenStarted`
  before counting the vote or becoming leader. Acting on a stale reply elects two
  leaders.
- **Counting your own vote twice / off-by-one majority.** Majority is
  `len(peers)/2 + 1`. You voted for yourself when you became candidate — start
  the tally at 1, not 0.
- **Election timeout too short or not randomized.** If timeouts cluster, peers
  keep splitting the vote and no one wins. Randomize on *every* reset. Keep
  heartbeat interval well under the minimum election timeout (~100 ms heartbeat
  vs. 150–300 ms timeout).
- **Busy-waiting / too many timers.** Don't spin. A 10 ms sleep in `ticker` plus
  a deadline comparison is plenty and won't trip the harness's "too many RPCs"
  limits.
- **Data races.** *Every* read or write of a `Raft` field must be under `rf.mu`.
  Run **`go test -race`** — the race detector finds the field you accessed from a
  goroutine without the lock. A race that "seems harmless" today is the 2B bug you
  can't find next week.

> **Your turn:** before writing any code, transcribe **Figure 2** of the paper
> by hand — the RequestVote and AppendEntries rules and the "Rules for Servers"
> box. 2A only uses a subset, but copying it cements which checks are mandatory.
> Nearly every election bug is a line from Figure 2 you skipped.

## Free resources

- [**6.824 Lab 2 (Raft)**](https://pdos.csail.mit.edu/6.824/labs/lab-raft.html) —
  the lab spec, API, and the 2A test descriptions. Your source of truth.
- [**The Raft paper**](https://raft.github.io/raft.pdf) — Figure 2 is the
  implementation. Re-read §5.1–§5.2 (election) specifically.
- [**Students' Guide to Raft**](https://thesquareplanet.com/blog/students-guide-to-raft/)
  — written by a 6.824 TA; the "structure" and "locking" sections are the exact
  advice that gets people through 2A.
- The [**Raft visualization**](https://raft.github.io/) — kill the leader, watch
  re-election; it's the picture your code is reproducing.

## Done when

- [ ] You transcribed Figure 2 and can say which rules 2A exercises.
- [ ] `go test -run 2A` passes both `TestInitialElection2A` and
      `TestReElection2A`.
- [ ] `go test -race -run 2A` passes — **no** reported races.
- [ ] It passes **repeatedly.** Run it 10+ times (a shell loop, or
      `go test -run 2A -count 10`). Flaky means a real timing bug, not bad luck.
- [ ] No goroutine holds `rf.mu` across an RPC `Call`, and every field access is
      under the lock.

When 2A is green and stays green, commit it on your **Roadmap.** You have a
cluster that elects a leader and heals after failure — the foundation of every
real consensus system. Next: **Lab 2B, log replication** — now the leader has to
make everyone agree on *what happened*, not just *who's in charge*.

## Common interview gotchas

- **Randomized timeouts exist to break split votes.** With fixed timeouts every
  follower wakes at once, all become candidates, all vote for themselves, nobody
  reaches a majority, and the cluster livelocks. Draw a *fresh* random value on
  every reset, and keep the range wider than one round-trip so the first waker
  wins before the second fires.
- **Step down on a higher term — including on replies.** Any server seeing a term
  greater than its own reverts to follower, adopts the term, and clears
  `votedFor`. The classic two-leader bug is checking terms on incoming requests
  but ignoring the term in an RPC *reply*, leaving a deposed leader heartbeating.
- **A stale (partitioned) leader is harmless because it can't commit.** It only
  reaches a minority, so `commitIndex` never advances and its writes silently
  evaporate when the partition heals. It never acknowledges data that later
  disappears — write safety holds even before it learns it's been deposed.
- **Never hold `rf.mu` across an RPC `Call`.** RPCs can hang arbitrarily; holding
  the lock freezes the cluster into deadlock. Read under the lock, unlock, Call,
  re-lock — and on re-lock **re-validate** you're still a candidate in the same
  term before counting the vote, or a stale reply elects two leaders.
- **Majority means quorum overlap, and odd sizes are about cost.** Any two
  majorities share a node, which is what forbids two leaders per term. Six nodes
  tolerate the same two failures as five but pay a larger quorum — odd sizes give
  the best resilience-per-machine.
