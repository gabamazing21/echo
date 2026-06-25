---
slug: sys-processes-threads-memory
step: 3
title: Processes, threads, memory & the scheduler
summary: What a machine actually does — processes, threads, virtual memory, context switching and the OS scheduler that runs them all.
est_min: 300
position: 1
---

# Processes, threads, memory & the scheduler

> **Step 3 · How computers & networks work · Week 1**
> Concept: *what a machine actually does*

You've written Go. Now zoom out and look at the thing your Go binary runs *on*: a
machine. Underneath every program is an operating system juggling many programs
at once, slicing a handful of CPU cores across hundreds of tasks, and handing each
one an illusion of private memory. This lesson is that picture — **processes,
threads, virtual memory, context switching, and the scheduler**. It's the
foundation that makes goroutines (Step 5) and distributed nodes (Step 8) make
sense.

## Why this matters

A distributed system is *just a pile of machines talking over a network*. Every
node in your future cluster is one of these — a box running an OS that schedules
your process across some cores and hands it some memory. Almost every hard
question in distributed systems eventually bottoms out here:

- **Latency**: why did that request take 5ms instead of 50µs? Often the answer is
  a context switch, a cache miss, or the scheduler not running your thread yet.
- **Concurrency**: why does Go let you spawn a million goroutines but not a
  million OS threads? Because it knows what threads *cost* — and you're about to.
- **Failure**: a node "going down" usually means a process died or got starved of
  CPU. You can't reason about that without knowing what a process is.

Understand one machine deeply and the cluster stops being magic.

## 1. A process is a running program

A **program** is a file on disk. A **process** is that program *in motion* — code
loaded into memory, with its own address space, open files, and at least one
thread of execution. The OS gives every process a unique **PID** (process ID).

Your Go binary is a process the moment you run it. You can see its PID from
inside:

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("pid:", os.Getpid())   // this process's ID
	fmt.Println("ppid:", os.Getppid()) // the parent that launched it (your shell)
}
```

Run it twice and the PID changes each time — every run is a fresh process. The
**parent PID** points at whatever started it (usually your shell). Processes are
**isolated**: one process cannot read another's memory by accident. That isolation
is a safety boundary the OS enforces with hardware help, and it's why a crash in
one process doesn't take the others down.

## 2. Threads: multiple flows inside one process

A **thread** is a single sequential flow of execution. A process starts with one
thread (the one running `main`), but it can spawn more. All threads in a process
**share the same address space** — the same heap, the same global variables, the
same open files — but each has its **own stack** and its own CPU register state.

That sharing is the whole point and the whole danger:

- **Point**: threads can cooperate cheaply by reading/writing shared memory. No
  copying, no message passing required.
- **Danger**: two threads touching the same variable at the same time is a **data
  race**. This is why you'll spend real time on mutexes and channels in Step 5.

| | Process | Thread |
|---|---|---|
| Address space | Private, isolated | Shared with siblings |
| Created by | OS (`fork`/`exec`) | OS, within a process |
| Crash blast radius | Just itself | Can corrupt the whole process |
| Communication | IPC, sockets, pipes | Shared memory (needs locking) |
| Relative cost | Heavy | Lighter, but still not free |

## 3. Virtual memory: the private-address illusion

Here's a thing that surprises people: when your program reads address
`0x4000_1000`, that is **not** a real location in the RAM chips. It's a
**virtual address**. The OS, with the CPU's memory-management unit (MMU),
translates every virtual address your process uses into a physical one on the
fly, page by page (typically 4 KB pages).

This buys you three big things:

1. **Isolation** — each process gets its own virtual address space starting,
   conceptually, from zero. Process A's `0x4000` and Process B's `0x4000` map to
   *different* physical RAM. That's section 1's isolation, made real.
2. **The illusion of more memory than you have** — pages not currently needed can
   be evicted to disk (swap) and faulted back in on demand.
3. **A clean, predictable layout** — every process sees the same tidy memory map.

### Stack vs heap

Within that address space, two regions matter most to you as a programmer:

- **The stack** grows and shrinks as functions call and return. Each function
  call pushes a **stack frame** (its local variables, return address); returning
  pops it. It's fast — just bump a pointer — and automatically cleaned up. Each
  thread has its own stack.
- **The heap** is for data that must outlive the function that created it, or
  whose size isn't known at compile time. Allocation and reclamation are more
  expensive; in Go, the **garbage collector** manages the heap for you.

```go
package main

import "fmt"

func main() {
	x := 42        // typically lives on the stack — local, fixed size
	p := &x        // we take its address...
	fmt.Println(*p) // 42
	// In C, returning &x would be a bug. In Go, the compiler does
	// "escape analysis": if a value's address escapes the function,
	// Go quietly moves it to the heap so it stays valid. Safe by default.
}
```

You'll rarely allocate manually in Go — but knowing *where* data lives explains
GC pauses, why huge stacks are bad, and why passing pointers isn't always free.

## 4. Context switching: how one core runs many threads

A single CPU core can only execute **one** thread at any instant. So how does your
laptop run a browser, an editor, music, and your Go program "at the same time" on
8 cores? The OS rapidly switches each core between threads — fast enough to *feel*
simultaneous. Switching from one thread to another is a **context switch**:

1. Stop the running thread.
2. Save its CPU registers and program counter (its "context") to memory.
3. Load the next thread's saved context.
4. Resume it — it has no idea it was ever paused.

Context switches are **not free**. Saving/restoring state costs time, and worse,
the new thread's data probably isn't in the CPU caches, so it runs slowly until
the caches warm up. Thousands of needless context switches per second is a real
performance bug. This cost is *exactly* why Go multiplexes many cheap goroutines
onto few OS threads — minimizing kernel context switches (more in Step 5).

## 5. The scheduler: who runs next, and for how long

The **scheduler** is the part of the OS that decides which thread gets a core, and
for how long. Modern desktop and server OSes use **preemptive multitasking**: the
scheduler can **forcibly pause** a running thread — it doesn't wait politely for
the thread to give up the core. (The old alternative, *cooperative* multitasking,
let one stuck program freeze the whole machine.)

The basic mechanism is the **time slice** (or *quantum*): each thread gets a small
slice of CPU time (often a few milliseconds). A hardware **timer interrupt** fires
when the slice expires, the kernel runs, the scheduler picks the next runnable
thread, and a context switch happens. Repeat, thousands of times a second.

The scheduler constantly balances goals you'll recognize from distributed systems
too:

- **Throughput** — get the most total work done.
- **Latency / responsiveness** — don't let any one task wait too long.
- **Fairness** — don't starve anyone.

How many *hardware* threads can truly run in parallel is just the core count. Go
exposes this:

```go
package main

import (
	"fmt"
	"runtime"
)

func main() {
	fmt.Println("logical CPUs:", runtime.NumCPU())
	// GOMAXPROCS caps how many OS threads run Go code simultaneously.
	// It defaults to NumCPU. Reading it with -1 returns the current value
	// without changing it.
	fmt.Println("GOMAXPROCS:", runtime.GOMAXPROCS(-1))
}
```

`runtime.NumCPU()` tells you how many cores the OS reports. `GOMAXPROCS` is Go's
own knob for how many OS threads may execute Go code at once — it defaults to the
core count. This is the seam between *Go's* scheduler and the *OS's* scheduler,
which is the whole story of the next bullet.

## 6. Foreshadowing: goroutines and Go's runtime scheduler

Everything above is about the **OS**. Go adds its **own** scheduler on top, and
that's why goroutines feel free:

- A **goroutine** is *not* an OS thread. It's a much lighter unit the Go runtime
  manages, starting with a tiny ~2 KB stack that grows on demand. You can run
  hundreds of thousands of them.
- Go's runtime **multiplexes** many goroutines onto a small pool of OS threads
  (sized by `GOMAXPROCS`). When a goroutine blocks (say, on a network read), the
  runtime parks it and runs another goroutine on that thread — **no kernel
  context switch needed**. This is often called *M:N scheduling* (M goroutines on
  N OS threads).
- So Go gets the *programming model* of "spawn a thread per task" without paying
  the *cost* of an OS thread per task — by reusing the exact concepts in this
  lesson, one level up.

When you reach Step 5 and write `go doWork()`, remember: there's a scheduler
underneath doing context switches and time-slicing on the OS threads, and a Go
scheduler above doing the same trick on goroutines. Same ideas, two layers.

## Do it yourself

1. Create a scratch program (`mkdir machine && cd machine && go mod init example.com/machine`) and a `main.go` you can run with `go run .`.
2. Print `os.Getpid()` and `os.Getppid()`. Run the program several times and watch the PID change but the parent PID stay (it's your shell).
3. Print `runtime.NumCPU()` and `runtime.GOMAXPROCS(-1)`. Confirm they match. Then try running with `GOMAXPROCS=1 go run .` and `GOMAXPROCS=2 go run .` and observe the reported value change.
4. With your program running, open another terminal and find it: `ps aux | grep machine` (or `top` / Activity Monitor). See your PID, memory usage, and CPU% — the OS's view of your process.
5. Read the **free** OSTEP book — [**Operating Systems: Three Easy Pieces**](https://pages.cs.wisc.edu/~remzi/OSTEP/). Start with the *Virtualization* part: the chapters on processes, the process API, scheduling, and address spaces map directly onto sections 1–5 above.
6. Skim Stanford's free [**CS144 (Introduction to Computer Networking)**](https://cs144.github.io/) intro — it frames *why* you, as a future distributed-systems engineer, need this machine-level model before you reason about networks of machines.

## Check yourself

You're ready to move on when you can answer, *without looking*:

- What's the difference between a program, a process, and a thread?
- What do threads in the same process **share**, and what does each thread keep **private** — and why is the shared part dangerous?
- What is virtual memory giving you, and how do the stack and heap differ?
- Walk through what happens during a context switch, and name one reason it's expensive.
- What does *preemptive* multitasking mean, and what's a *time slice*?
- What does `GOMAXPROCS` control, and how does it connect the OS scheduler to Go's runtime scheduler?

When all of these feel obvious, commit this task on your **Roadmap** and take the
Step 3 quiz. Next lesson: **how data crosses the wire — bytes, sockets, and the
network stack.**

## Common interview gotchas

- **A context switch has *two* costs.** Everyone names the direct cost (saving/restoring registers, running the scheduler). The cost interviewers are listening for is the **indirect** one: the new thread's data isn't in the CPU caches or TLB, so it runs slowly until they warm up — often the larger cost. A process switch also flushes the TLB; a thread switch within the same process doesn't.
- **"Stack or heap?" → "it depends, decided by escape analysis."** A Go slice header is small; its *backing array* goes on the heap only if it **escapes** the function (returned, stored in a heap object, captured by a goroutine, or grown to unknown size). Inspect with `go build -gcflags=-m`. Don't claim slices are "always heap."
- **Go's GC is mostly concurrent, not stop-the-world.** It's a concurrent tri-color mark-and-sweep with **two brief STW pauses** (mark setup, mark termination) and a write barrier in between; pauses are microseconds to low milliseconds. You tune *frequency* with `GOGC`/`GOMEMLIMIT`, not the algorithm.
- **Goroutines beat threads by avoiding the *kernel*.** ~2 KB growable stacks plus **M:N (G-M-P) scheduling** in user space: when a goroutine blocks on a channel/mutex/network read the runtime parks it and runs another on the same OS thread — *no kernel context switch*. That's the whole reason a million goroutines is fine but a million OS threads isn't.
- **A shared variable is not communication.** Per-core caches and CPU/compiler reordering mean one goroutine may never see another's write without a **happens-before** edge (mutex, channel, `sync/atomic`). Cite the Go memory model and `go test -race`.
