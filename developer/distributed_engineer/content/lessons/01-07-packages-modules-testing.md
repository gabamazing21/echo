---
slug: go-packages-modules-testing
step: 1
title: Packages, modules & testing with the standard library
summary: Organise code into packages and modules, manage dependencies, and write table-driven tests with Go's stdlib testing package.
est_min: 360
position: 7
---

# Packages, modules & testing with the standard library

> **Step 1 · Go fundamentals · Week 3**
> Concept: *go test, table-driven tests, package layout*

You've written functions and types. Now you'll learn how Go *organises* them —
into **packages** and **modules** — and how to prove they work with **tests**.
This is the lesson that separates "I can write Go" from "I can ship Go that other
people trust." Take it slowly; the habits here are the ones you'll keep forever.

## Why this matters

Go has a **test-first culture** baked into the language. There is no third-party
test framework you have to argue about — the standard library ships `testing`,
and `go test` is one of the four commands you already know. Idiomatic Go code
comes with `_test.go` files sitting right next to it, and the dominant style is
the **table-driven test**: one test, many cases, all in a slice.

The **Consensus app you're using right now** is built this way. Its handlers,
its quiz scoring, its progress tracking — all covered by table-driven tests run
with `go test ./...` on every commit. Great engineers don't ship code and *hope*.
They ship code with a test that says *why* it's correct. That's the bar.

## 1. Packages: how Go groups code

Every Go file declares the package it belongs to with `package <name>` on line
one. Files in the **same directory** must share the **same package name**, and
together they form one package.

```go
// file: mathutil/mathutil.go
package mathutil

// Add returns the sum of two integers.
func Add(a, b int) int {
	return a + b
}

// double is a helper used only inside this package.
func double(n int) int {
	return n * 2
}
```

You import a package by its **module path + directory**, then call its functions
through the package name:

```go
package main

import (
	"fmt"

	"example.com/myapp/mathutil"
)

func main() {
	fmt.Println(mathutil.Add(2, 3)) // 5
}
```

- `package main` is special: it produces an executable. Every other package is a
  library that gets imported.
- The import path is the **module path** (next section) joined to the directory.
- The standard convention: directory name == package name. Keep them matching.

## 2. Exported vs unexported: the capitalization rule

Go has no `public`/`private` keywords. **Visibility is decided by the first
letter of the identifier.**

- **Capitalized** (`Add`, `User`, `MaxRetries`) → **exported**: visible to code
  in other packages.
- **lowercase** (`double`, `userID`, `maxRetries`) → **unexported**: visible
  only *inside the same package*.

```go
package store

type User struct {
	Name  string // exported field — readable from other packages
	email string // unexported field — package-private
}

func New(name string) *User { // exported constructor
	return &User{Name: name, email: deriveEmail(name)}
}

func deriveEmail(name string) string { // unexported helper
	return name + "@consensus.dev"
}
```

From another package you can use `store.User`, `store.New`, and `u.Name` — but
**not** `u.email` or `store.deriveEmail`. The rule applies to functions, types,
struct fields, methods, and constants alike. This single convention *is* Go's
encapsulation model. Design your packages so the exported surface is small and
the messy internals stay lowercase.

## 3. Modules: `go.mod` and `go.sum`

A **module** is a tree of packages versioned together, named by a `go.mod` file
at its root. You met `go mod init` in Lesson 1:

```bash
go mod init example.com/myapp
```

That writes `go.mod`:

```
module example.com/myapp

go 1.21
```

When you add a third-party dependency, `go.mod` records the requirement and
`go.sum` records cryptographic checksums so builds are reproducible and
tamper-evident. You rarely edit either by hand — the tooling maintains them.

| Command | What it does |
|---|---|
| `go get example.com/pkg@v1.2.3` | Add (or upgrade) a dependency at a version |
| `go mod tidy` | Add missing requirements, drop unused ones, sync `go.sum` |
| `go list -m all` | List every module in the build |

A typical flow: you write `import "github.com/google/uuid"` in your code, then
run `go mod tidy`. Go fetches it, pins a version in `go.mod`, and writes the
checksums to `go.sum`. Commit **both** files.

## 4. Writing your first test

Tests live in files ending `_test.go`, in the **same package** as the code they
test. A test is a function named `TestXxx` taking a single `*testing.T`:

```go
// file: mathutil/mathutil_test.go
package mathutil

import "testing"

func TestAdd(t *testing.T) {
	got := Add(2, 3)
	want := 5
	if got != want {
		t.Errorf("Add(2, 3) = %d; want %d", got, want)
	}
}
```

Run it:

```bash
go test ./...        # every package, recursively
go test -v ./...     # verbose: show each test name and PASS/FAIL
```

Key points:

- Use `t.Errorf` to report a failure and **keep going**; use `t.Fatalf` to
  report and **stop this test immediately** (e.g. after a setup step failed).
- Always print *got* vs *want* so a failure tells you what happened.
- A test file in `package mathutil` can reach unexported identifiers; a file in
  `package mathutil_test` (a sibling "external" test package) can only use the
  exported API — handy for testing your package as a real consumer would.

## 5. Table-driven tests with `t.Run` subtests

The idiomatic Go pattern: define a slice of cases, loop over them, and run each
as a named **subtest** with `t.Run`. One test, many inputs, clear output.

```go
package mathutil

import "testing"

func TestAdd_Table(t *testing.T) {
	tests := []struct {
		name string
		a, b int
		want int
	}{
		{name: "positives", a: 2, b: 3, want: 5},
		{name: "with zero", a: 0, b: 7, want: 7},
		{name: "negatives", a: -4, b: -6, want: -10},
		{name: "mixed signs", a: -5, b: 9, want: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Add(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("Add(%d, %d) = %d; want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
```

Why this is the standard:

- **Adding a case is one line** in the slice — no copy-pasted test functions.
- `t.Run` gives each case its **own name** in the output, e.g.
  `TestAdd_Table/mixed_signs`, so a failure points straight at the bad case.
- You can run a single case: `go test -run TestAdd_Table/negatives`.
- Subtests are isolated — one failing case doesn't hide the others.

## 6. Test helpers and `t.Helper()`

When several tests share setup or assertion logic, factor it into a helper.
Call `t.Helper()` first so that, on failure, Go reports the **caller's** line
number instead of the line inside the helper:

```go
package mathutil

import "testing"

func assertEqual(t *testing.T, got, want int) {
	t.Helper() // failures point at the caller, not this line
	if got != want {
		t.Errorf("got %d; want %d", got, want)
	}
}

func TestDoubleViaHelper(t *testing.T) {
	assertEqual(t, Add(2, 2), 4)
	assertEqual(t, Add(10, 5), 15)
}
```

Without `t.Helper()`, every failure would point at the `t.Errorf` line inside
`assertEqual` — useless when you have twenty call sites. With it, you get the
exact failing assertion. Always add it to test helpers that call `t.Error`/
`t.Fatal`.

## 7. A note on coverage

Go measures how much of your code your tests exercise:

```bash
go test -cover ./...                      # print a coverage % per package
go test -coverprofile=cover.out ./...     # write a detailed profile
go tool cover -html=cover.out             # open a line-by-line HTML report
```

Coverage is a **flashlight, not a scoreboard**. A high number doesn't prove
correctness, but the HTML report's red lines show you exactly which branches no
test ever touches — often where the bugs hide. Aim to cover the logic that
*matters*, not to chase 100%.

## Do it yourself

1. Create a module: `mkdir stringkit && cd stringkit && go mod init example.com/stringkit`.
2. Write `stringkit.go` with an **exported** `Reverse(s string) string` and an
   **unexported** helper of your choice. Confirm the capitalization rule by
   trying to call the helper from a `main` package — watch it fail to compile.
3. Write `stringkit_test.go` with a **table-driven** `TestReverse` using
   `t.Run` subtests: empty string, single char, a word, and a palindrome.
4. Run `go test -v ./...` and read each subtest name in the output.
5. Add an `assertEqual`-style helper with `t.Helper()` and route your assertions
   through it. Break the implementation on purpose and confirm the failure points
   at the *caller*.
6. Run `go test -cover ./...`, then generate and open the HTML report.
7. Work through **[Learn Go with Tests](https://quii.gitbook.io/learn-go-with-tests)**
   — the *Hello, World* and *Iteration* chapters are pure gold and use exactly
   this style.
8. Skim the **[testing package docs](https://pkg.go.dev/testing)** and the
   **[Go Modules Reference](https://go.dev/ref/mod)** — you don't need to read
   either end-to-end, just know what's there to look up later.

## Check yourself

You're ready to move on when you can answer, *without looking*:

- What decides whether an identifier is exported — and what's the exact rule?
- What's the difference between `go.mod` and `go.sum`, and which command keeps
  them in sync with your imports?
- How do you turn one test into many cases, and why use `t.Run` for each?
- What does `t.Helper()` change about a failure message, and where do you put it?
- What does `go test -cover` tell you — and what does it *not* prove?

When all five feel obvious, commit this task on your **Roadmap** and take the
Step 1 quiz. Next up: **Step 2 — concurrency with goroutines and channels.**

## Common interview gotchas

- **A green test suite without `-race` is a false comfort** — concurrency bugs are invisible to ordinary `go test`; only `go test -race ./...` instruments memory access to catch them. CI that doesn't run `-race` will ship data races that pass locally every time. (The pre-Go-1.22 loop-variable capture in subtests is a classic the race detector exposes.)
- **`internal/` is enforced by the compiler, not convention** — code under `.../internal/foo` is importable only by packages rooted at `internal/`'s parent. It's how you keep an API surface private across packages; outsiders importing it get a build error, not a lint warning.
- **The exported surface *is* your compatibility contract** — capitalizing a name to "just make a test reach it" commits you to supporting it forever under Go's module compatibility promise. Test unexported logic from inside the package (`package foo`) or test the public API from a `foo_test` external package; don't export for testing.
- **`t.Run` subtests don't stop the suite on `t.Fatal` — but they do stop their goroutine** — a `t.Fatal` inside a subtest fails only that case, which is the point, but calling `t.Fatal` from a *helper goroutine* you spawned does nothing useful (it must run on the test's own goroutine). Return errors from goroutines and assert on the main one.
- **`t.Helper()` only fixes the reported line number — it doesn't make a helper a test** — forgetting it points every failure at the assertion line inside the helper, useless across many call sites. And a helper that calls `t.Fatal` must still run on the test goroutine to abort correctly.
