---
slug: go-toolchain-first-program
step: 1
title: The Go toolchain & your first program
summary: Install Go, understand go run / build / mod, and ship your first program — the way real Go projects are laid out.
est_min: 180
position: 1
---

# The Go toolchain & your first program

> **Step 1 · Go fundamentals · Week 1**
> Concept: *toolchain, `go run` / `go build`, `go.mod`*

Before you write a single distributed system, you write Go. And before you write
*good* Go, you need to feel at home in the **toolchain** — the small set of
commands that compile, run, test, and manage every Go project on earth. This is
the muscle memory everything else sits on. Don't skim it.

## Why this matters

Go's whole personality is *simplicity that scales*. One command builds a single
static binary with no runtime to install on the server. That property — `scp` a
binary and run it — is exactly why Go dominates infrastructure: Docker,
Kubernetes, etcd, Terraform, and the Raft implementations you'll study in Step 8
are all written in Go. You're learning the language the field is *built in*.

## 1. A program is a package

Every Go file starts by declaring the package it belongs to. A program you can
*run* must have `package main` and a `func main()`:

```go
package main

import "fmt"

func main() {
	fmt.Println("nodes online")
}
```

- `package main` — this is an executable, not a library.
- `import "fmt"` — pull in the standard-library formatting package.
- `func main()` — the entry point. Execution starts here.

## 2. The four commands you'll use every day

| Command | What it does | When you reach for it |
|---|---|---|
| `go run .` | Compile **and** run, throwing the binary away | Fast feedback while learning |
| `go build` | Compile to a binary in the current dir | You want an artifact to ship/run |
| `go test ./...` | Run every test in every package | Constantly — Go culture is test-first |
| `go mod tidy` | Sync dependencies with your imports | After adding/removing an import |

The `./...` pattern means "this directory and everything under it." You'll type
it a thousand times.

## 3. Modules: `go.mod` is your project's identity

A **module** is a collection of packages with a name and a dependency list,
declared in a `go.mod` file. You create one with:

```bash
mkdir hello && cd hello
go mod init example.com/hello
```

That writes:

```
module example.com/hello

go 1.21
```

The module path (`example.com/hello`) is how *other* code imports yours, and how
Go resolves your own internal packages. For a real project it's usually your repo
URL, e.g. `github.com/you/project`.

> **The Consensus app you're using right now** was bootstrapped with exactly this:
> `go mod init consensus`. Open its `go.mod` and you'll recognise everything.

## 4. Do it yourself (≈ the rest of the session)

1. Install Go (you already have **Go 1.21** — check with `go version`).
2. `mkdir hello && cd hello && go mod init example.com/hello`.
3. Create `main.go` with the program above. Run it with `go run .`.
4. Change the message, run again. Notice there's no separate "compile" step in your head — `go run` does both.
5. Now `go build` and run the produced `./hello` binary directly. **This binary is self-contained** — that's the superpower.
6. Read the official [**A Tour of Go**](https://go.dev/tour/welcome/1) — *Welcome* and *Basics* sections. It runs in your browser; do every exercise.

## Check yourself

You're ready to move on when you can answer, *without looking*:

- What must a file declare to be a runnable program?
- What's the difference between `go run` and `go build`?
- What is `go.mod` for, and what's a "module path"?
- Why is a Go binary easy to deploy compared to, say, a Python script?

When all four feel obvious, commit this task on your **Roadmap** and take the
Step 1 quiz. Next lesson: **variables, types, and control flow.**

## Common interview gotchas

- **"Go binaries are always static" is half-true** — the moment you import `net` or `os/user`, the default build links libc via cgo and you get a *dynamically* linked binary. Set `CGO_ENABLED=0` (and often a `netgo`/`osusergo` build tag) to guarantee a truly static binary for a `FROM scratch` container.
- **`go run` and `go build` don't behave identically** — `go run` compiles to a temp dir and runs from there, so relative paths, `os.Args[0]`, and anything that locates files next to the binary can differ. Reproduce deployment bugs with the built binary, not `go run`.
- **`go.sum` is not a lockfile** — it's a tamper-evidence ledger of checksums; `go.mod` pins the versions. A passing build with a populated `go.sum` doesn't mean reproducible — `go mod tidy` can still change `go.mod`. The checksum *database* (sum.golang.org) is what protects you from a swapped dependency.
- **`GOPROXY=off` vs an unreachable proxy** — by default `go build` reaches out to the module proxy; in a hermetic CI box that hangs or fails. Vendor (`go mod vendor` + `-mod=vendor`) or set `GOFLAGS=-mod=mod` deliberately rather than discovering it on a red pipeline.
- **`go build` caches aggressively** — a green build can hide a change you didn't actually recompile. The build cache is keyed on inputs, so this is usually safe, but `go clean -cache` is the answer when "it works on my machine" smells like a stale artifact.
