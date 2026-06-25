---
slug: project-mapreduce
step: 8
title: "Lab: MapReduce in Go (MIT 6.824 Lab 1)"
summary: Build a distributed MapReduce — a coordinator handing map/reduce tasks to workers, tolerant of worker crashes.
est_min: 720
position: 1
---

# Lab: MapReduce in Go (MIT 6.824 Lab 1)

> **Step 8 · Build distributed systems · Week 1 · Lab**
> Concept: *distributed computation, coordination*

You've read the theory; now you build the real thing. This is **MIT 6.824 Lab 1**
— the entry point to one of the best distributed-systems courses on the planet,
free and online. You'll implement **MapReduce**: the paper from Google (2004)
that launched the big-data era. The lab is genuinely hard, and that's the point.
This guide is a **map of the territory**, not a solution — the learning lives in
the parts you write yourself.

> This guide will **not** hand you a working implementation. That violates the
> course policy and, more importantly, robs you of the one thing that makes a
> distributed-systems résumé credible: having actually done it. We sketch the
> shapes; you fill them in.

## What you'll build

A single-machine MapReduce that *behaves* like a distributed one:

- One **coordinator** process that owns the list of tasks and hands them out.
- Several **worker** processes that ask the coordinator for work, run your `Map`
  or `Reduce` function, and report back.
- Communication over **Go RPC** (`net/rpc`) — workers call the coordinator like
  a local function, but it crosses a process boundary.
- **Crash tolerance**: if a worker dies mid-task (the test script kills some on
  purpose), the coordinator notices via a **timeout** and re-hands that task to
  someone else. The final output must be correct anyway.

The deliverables you write are `mr/coordinator.go`, `mr/worker.go`, and
`mr/rpc.go`. The course gives you the `main/` wrappers, the sequential reference,
and a set of plugin applications (word count, indexer, etc.).

## Why this matters

This is the bridge from "I understand distributed systems" to "I have built
one." MapReduce is the simplest *real* distributed computation: it has a
coordinator, workers, RPC, partial failure, and a correctness contract — the
same ingredients as every system in the steps ahead (and the Raft labs that
follow). If you can make the coordinator survive a worker dying at the worst
possible moment, you understand partial failure in your fingers, not just your
head.

It also ties straight back to **Step 1**. Remember `WordCount(s string)
map[string]int`? The canonical MapReduce application *is* word count: `Map`
emits `(word, "1")` for every word, `Reduce` sums the `"1"`s for a key. You
already wrote the single-machine version. Now you're spreading it across workers.

## The MapReduce model

Three phases, in order:

1. **Map.** Each input split is fed to your `Map(filename, contents)`, which
   emits a list of intermediate key/value pairs. For word count it emits one
   `{Key: word, Value: "1"}` per word.
2. **Shuffle.** All intermediate pairs are grouped by key, so every pair with
   the same key lands at the same reduce task. This is the quiet, crucial step —
   it's why the framework exists. You implement it by **partitioning** each map
   output into `nReduce` buckets with `ihash(key) % nReduce`.
3. **Reduce.** Each reduce task receives one key and *all* its values, and emits
   the final result. For word count: `Reduce(word, ["1","1","1"]) = "3"`.

```text
input files ──Map──> intermediate (k,v) ──shuffle by key──> Reduce ──> output
   pg-*.txt          mr-X-Y files                            mr-out-Y
```

The map and reduce functions come from a plugin (`wc.go`); your framework just
*schedules and moves data*. Keep that boundary clean in your head.

## Architecture

One coordinator, many workers, talking over RPC:

```text
        ┌─────────────┐
        │ Coordinator │   owns task list + state, hands out work
        └─────────────┘
          ▲   ▲   ▲
   RPC    │   │   │   RPC
        ┌─┘   │   └─┐
   ┌────────┐┌────────┐┌────────┐
   │ Worker ││ Worker ││ Worker │   ask for a task, run Map/Reduce, report done
   └────────┘└────────┘└────────┘
```

The workers drive the loop: a worker calls the coordinator, gets a task, does it,
calls back to say "done," then asks for the next one. The coordinator is mostly
**passive** — it answers RPCs and tracks state. Because several workers call at
once, **every field the coordinator touches must be guarded by a `sync.Mutex`.**

A task struct you'll keep on the coordinator might look like:

```go
type TaskKind int

const (
	MapTask TaskKind = iota
	ReduceTask
	WaitTask // no work available right now; worker should sleep and retry
	ExitTask // all done; worker should shut down
)

type TaskState int

const (
	Idle TaskState = iota
	InProgress
	Completed
)

type Task struct {
	Kind     TaskKind
	Index    int       // which map or reduce task this is
	File     string    // input file, for map tasks
	NReduce  int       // number of reduce buckets
	NMap     int       // number of map tasks (reduce needs this)
	Deadline time.Time // when this in-progress task is considered crashed
}
```

## RPC: the wire between them

Go's `net/rpc` lets a worker call a coordinator method as if it were local. You
define an **argument struct** and a **reply struct** for each call; both must use
**exported (capitalized) fields** or they won't survive encoding. A single
"request a task" call is usually enough:

```go
// in mr/rpc.go — shared by both sides
type RequestTaskArgs struct {
	WorkerID int
}

type RequestTaskReply struct {
	Task Task
}

type ReportDoneArgs struct {
	Kind  TaskKind
	Index int
}

type ReportDoneReply struct{}
```

On the coordinator, the handler signature is fixed by `net/rpc`:

```go
func (c *Coordinator) RequestTask(args *RequestTaskArgs, reply *RequestTaskReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// pick an idle task, mark it InProgress, set its Deadline, fill reply.Task
	return nil
}
```

Don't over-design the protocol. Two methods — "give me a task" and "I finished
task N" — carry the whole lab.

## Approach: build it in milestones

Resist writing it all at once. Each milestone is runnable and testable.

### Milestone 1 — sequential, in one process

Before any RPC, get the *computation* right. Read the provided
`mrsequential.go`: it runs all maps, sorts, then all reduces, in one process.
Make sure you understand how it loads the `wc` plugin and how `Map`/`Reduce` are
shaped. This is your correctness oracle — its output is what your distributed
version must match.

### Milestone 2 — coordinator hands out map tasks

Start the coordinator with one map task per input file. Implement `RequestTask`
so it returns idle map tasks. The worker runs `Map`, then partitions its output
into `nReduce` files named **`mr-X-Y`** (X = map index, Y = reduce bucket) using
`ihash(key) % nReduce`. Encode each bucket as JSON lines — `json.NewEncoder` over
a file is the easy path:

```go
enc := json.NewEncoder(file)
for _, kv := range bucketY {
	enc.Encode(&kv) // KeyValue{Key, Value string}
}
```

### Milestone 3 — reduce phase

Only hand out reduce tasks once **all** map tasks are `Completed`. A reduce task
`Y` reads every `mr-X-Y` file (for all map indices X), sorts the pairs by key,
groups equal keys, calls `Reduce(key, values)`, and writes `mr-out-Y`. When all
reduce tasks finish, `Done()` returns true and the coordinator (and workers)
exit.

### Milestone 4 — crash tolerance

This is the real lab. The test script kills workers mid-task. Your coordinator
must **not** wait forever for a dead worker. The standard rule: **if a task has
been `InProgress` for more than ~10 seconds, assume the worker died and reset it
to `Idle`** so another worker picks it up.

Run a background goroutine that periodically scans for expired tasks:

```go
func (c *Coordinator) reaper() {
	for {
		time.Sleep(time.Second)
		c.mu.Lock()
		for i := range c.tasks {
			t := &c.tasks[i]
			if t.State == InProgress && time.Now().After(t.Deadline) {
				t.State = Idle // hand it out again
			}
		}
		c.mu.Unlock()
	}
}
```

### The atomic-write trap

Two workers might run the *same* reduce task — the slow one you gave up on, plus
its replacement — and both write `mr-out-Y`. If they interleave, you get a
corrupt file. The fix the paper itself uses: **write to a uniquely named
temporary file, then `os.Rename` it into place.** Rename is atomic on the same
filesystem, so a reader ever sees either the old file or a complete new one,
never a half-written one.

```go
tmp, _ := os.CreateTemp(".", "mr-out-tmp-*")
// ... write all output to tmp ...
tmp.Close()
os.Rename(tmp.Name(), fmt.Sprintf("mr-out-%d", reduceIndex))
```

Do the same for the intermediate `mr-X-Y` files. This single technique is most of
the credit on the crash tests.

## Common pitfalls

- **Forgetting the lock.** Every read or write of coordinator state goes through
  the mutex. The race detector (`go run -race`) will catch you; run it.
- **Handing out reduce before map is done.** A reduce task that runs before all
  `mr-X-Y` files exist produces wrong output. Gate the phase transition.
- **Workers that spin.** When no task is available but the job isn't finished,
  return a `WaitTask` and have the worker `time.Sleep` briefly before retrying —
  don't busy-loop hammering the coordinator.
- **Lower-cased RPC fields.** Unexported struct fields silently vanish over RPC.
  Capitalize everything in the args/reply structs.
- **Crash-test corruption.** If you skip temp-file-then-rename, duplicate task
  execution will occasionally leave a mangled `mr-out-Y` and the test fails
  *intermittently* — the worst kind of bug. Do the atomic write from the start.
- **Exit timing.** A worker shouldn't crash when the coordinator has already shut
  down its RPC socket; treat a failed `call()` as "the job is probably done" and
  exit cleanly.

## A timeout you'll reach for

The select-with-timeout pattern shows up whenever you wait on something that
might never answer (a worker, an RPC). Keep it in your toolkit:

```go
select {
case result := <-done:
	handle(result)
case <-time.After(10 * time.Second):
	// gave up waiting — treat the worker as crashed
}
```

In this lab you'll more often track deadlines on the coordinator (Milestone 4)
than block on a channel, but the idea is identical: **never wait unbounded on an
unreliable peer.**

## Free resources

- [**MIT 6.824 Lab 1: MapReduce**](https://pdos.csail.mit.edu/6.824/labs/lab-mr.html)
  — the official lab page. Read it top to bottom; it has the rules, the hints,
  and the exact files you edit.
- [**MIT 6.824 course site**](https://pdos.csail.mit.edu/6.824/) — lecture
  videos, schedule, and the rest of the labs (Raft is next, in Step 8 too).
- [**The MapReduce paper**](https://pdos.csail.mit.edu/6.824/papers/mapreduce.pdf)
  — Dean & Ghemawat, 2004. Section 3 (Implementation) describes the exact
  coordinator/worker/backup-task design you're building.
- [**`net/rpc` docs**](https://pkg.go.dev/net/rpc) and
  [**`encoding/json`**](https://pkg.go.dev/encoding/json) — the two standard
  libraries the lab leans on.

## Done when

- [ ] `go build -race -buildmode=plugin ../mrapps/wc.go` and your `mrcoordinator`
      + `mrworker` run the word-count job and produce correct `mr-out-*` files.
- [ ] Output matches `mrsequential.go` exactly (the test script diffs them).
- [ ] **`bash test-mr.sh` passes every case** — including the crash test and the
      parallelism tests — run it a few times, since crash bugs are intermittent.
- [ ] `go run -race` reports **no data races** in the coordinator.
- [ ] All coordinator state is mutex-guarded; reduce only starts after every map
      task completes.
- [ ] Output files are written via **temp file + `os.Rename`**, never directly.

When `test-mr.sh` prints `*** PASSED ALL TESTS`, commit this on your **Roadmap**.
You've built a real distributed computation that survives workers dying — the
hard-won intuition that makes the next lab, **Raft**, feel like implementing
something you already understand.

## Common interview gotchas

- **Skew, not key-count, is the bottleneck.** `ihash(key) % nReduce` balances
  *keys* evenly, never *work*. One hot key — `"the"`, a celebrity user id — makes
  one reducer the straggler that decides wall-clock time. Reach for a combiner,
  salting, or a skew-aware partitioner; don't just add more reduce tasks and hope.
- **Crash tolerance is paid in duplicate work, and that's fine.** A re-run task
  is correct only because output is committed via **temp-file-then-`os.Rename`**
  (atomic on one filesystem). Interviewers want "I tolerate duplicate execution
  via atomic commit," not "I prevent it" — preventing it is harder and unnecessary.
- **Know where shuffle physically lives.** There's no shuffle process: the map
  side partitions into `mr-X-Y` buckets, the reduce side pulls every `mr-X-Y` for
  its bucket. Be able to say *why* every value for a key must reach one reducer
  (the `Reduce(key, ALL values)` contract).
- **The coordinator is a deliberate SPOF.** It's one process among thousands of
  workers, so the paper spends its fault-tolerance budget on workers and lets a
  rare coordinator crash abort the job. If asked to make it HA, checkpoint its
  state or back it with consensus — which is exactly what the Raft labs build.
- **Batch vs. streaming is a when-question.** MapReduce wins on bounded,
  reproducible, latency-tolerant work; an unbounded or seconds-fresh requirement
  pushes you to windowed stream processing with watermarks for late events.
