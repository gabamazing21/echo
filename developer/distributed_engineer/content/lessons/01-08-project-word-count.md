---
slug: project-cli-word-count
step: 1
title: "Project: a word-count tool, test-first"
summary: Build a wc-style CLI and a pure WordCount function, writing the tests first — the exact habit and function the Consensus checker grades.
est_min: 360
position: 8
---

# Project: a word-count tool, test-first

> **Step 1 · Go fundamentals · Week 3 · Project**
> Concept: *testing, file reading, maps for counting*

This is your first real project. No new syntax to memorise — instead you take
everything from Weeks 1–2 (functions, slices, maps, strings) and *build
something that runs*. The twist: you write the **test before the code**. That is
not a gimmick. It is the single habit that separates engineers who ship from
engineers who debug forever.

## What you'll build

A small command-line tool, `wc`-style, that reads text and reports counts:

- **lines, words, and bytes** of a file or of standard input, and
- a pure function `WordCount(s string) map[string]int` that returns the
  *frequency* of each word — how many times every distinct word appears.

By the end you will be able to run:

```bash
go run . poem.txt
#   12   84  503 poem.txt

cat poem.txt | go run .
#   12   84  503
```

…and you'll have a tested `WordCount` you can drop straight into the Consensus
Code Checker.

## Why this matters

Test-first is the ALX habit. When you write the test first you are forced to
decide *what the function should do* before you get lost in *how*. The test
becomes an executable specification — and a safety net so you can refactor
fearlessly later.

It matters here for a second reason: **`WordCount(s string) map[string]int` is a
real challenge in this app's Code Checker.** The grader feeds your function a
string and compares the map you return against the expected counts. Get it right
here, locally, with your own tests, and you'll pass the checker on the first
submission. You're not doing a toy exercise — you're building the thing the
platform is about to grade.

## 1. Set up the module

```bash
mkdir wordcount && cd wordcount
go mod init example.com/wordcount
```

Create two files: `wordcount.go` (the library code) and `wordcount_test.go`
(the tests). Keep the counting logic *pure* — no file reading, no printing — so
it's trivial to test and to reuse.

## 2. Write the test FIRST

Before `WordCount` exists, describe what it must do. Go's testing is built in:
test files end in `_test.go`, test functions start with `Test`, and idiomatic Go
uses **table-driven tests** — a slice of cases you loop over.

```go
package main

import (
	"reflect"
	"testing"
)

func TestWordCount(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want map[string]int
	}{
		{"empty", "", map[string]int{}},
		{"single", "go", map[string]int{"go": 1}},
		{"repeat", "go go go", map[string]int{"go": 3}},
		{
			"mixed spacing",
			"the quick the",
			map[string]int{"the": 2, "quick": 1},
		},
		{
			"tabs and newlines",
			"a\tb\nb",
			map[string]int{"a": 1, "b": 2},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := WordCount(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("WordCount(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
```

Run it now:

```bash
go test ./...
```

It **won't even compile** — `WordCount` doesn't exist yet. That's the "red" step
of red-green-refactor. Good. The test is telling you exactly what to build next.

## 3. Make it pass: implement `WordCount`

Two tools do almost all the work:

- `strings.Fields(s)` splits on *any* run of whitespace (spaces, tabs,
  newlines) and drops empty pieces — exactly what we want.
- a `map[string]int` accumulates the count per word.

```go
package main

import "strings"

// WordCount returns how many times each whitespace-separated word
// appears in s.
func WordCount(s string) map[string]int {
	counts := make(map[string]int)
	for _, word := range strings.Fields(s) {
		counts[word]++
	}
	return counts
}
```

The line `counts[word]++` is the idiom to know: reading a missing key from a Go
map yields the zero value (`0`), so the first `++` turns it into `1` with no
special-casing. Run `go test ./...` again — green.

> **Your turn:** add a failing case for punctuation, e.g. `"Go, go!"`. Should
> `Go,` and `go!` count as the same word? Decide the behaviour, write the test,
> *then* make it pass (hint: `strings.ToLower`, `strings.Trim`/`strings.Map`).
> This is your first taste of TDD driving a design decision.

## 4. Count lines, words, and bytes with `bufio`

The frequency map is one feature; classic `wc` reports three numbers. Reading
input efficiently is the job of **`bufio`** — buffered I/O that wraps a raw
reader so you scan it line by line without loading the whole file into memory.

`bufio.Scanner` is the friendly front door. By default it splits on lines; you
can switch it to split on words with `scanner.Split(bufio.ScanWords)`.

```go
package main

import (
	"bufio"
	"io"
)

type Counts struct {
	Lines int
	Words int
	Bytes int
}

// Count reads everything from r and reports line, word, and byte totals.
func Count(r io.Reader) (Counts, error) {
	var c Counts
	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		c.Lines++
		c.Bytes += len(line) + 1 // +1 for the newline the scanner stripped
		c.Words += len(splitWords(line))
	}
	return c, scanner.Err()
}
```

> **Your turn:** `Count` takes an `io.Reader`, not a filename — that's
> deliberate. It means the same function works for a file, for stdin, or for a
> `strings.Reader` in a test. Write a table-driven test for `Count` that feeds
> it `strings.NewReader(...)` and checks the three numbers. You'll need to
> implement `splitWords` (`strings.Fields` is fine) — let the test tell you when
> it's right.

Note the byte count above is a simplification (it assumes every line ended in
exactly one `\n`). Fixing that edge case is part of your job below.

## 5. Wire up `main()` — read a file or stdin

Now the thin shell around your tested core. `os.Args` holds the command-line
arguments (`os.Args[0]` is the program name). If a filename is given, open it
with `os.Open`; otherwise read from `os.Stdin`.

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	var (
		r    *os.File
		name string
		err  error
	)

	if len(os.Args) > 1 {
		name = os.Args[1]
		r, err = os.Open(name)
		if err != nil {
			fmt.Fprintln(os.Stderr, "wc:", err)
			os.Exit(1)
		}
		defer r.Close()
	} else {
		r = os.Stdin
	}

	c, err := Count(r)
	if err != nil {
		fmt.Fprintln(os.Stderr, "wc:", err)
		os.Exit(1)
	}

	fmt.Printf("%5d %5d %5d %s\n", c.Lines, c.Words, c.Bytes, name)
}
```

Two habits worth absorbing: errors go to `os.Stderr` (not stdout, which is for
results), and `defer r.Close()` runs when `main` returns so the file is always
released. Build it and try it:

```bash
go build
./wordcount go.mod
echo "one two two three three three" | ./wordcount
```

## Stretch goals

Pick at least one. These are where the real learning compounds.

- **Flags like real `wc`.** Use the standard `flag` package to add `-l`, `-w`,
  `-c` so the user can ask for only lines, only words, or only bytes. Default
  (no flags) prints all three.
- **Top-N frequent words.** Add a `-top N` mode that prints the N most common
  words from `WordCount`, sorted by count then alphabetically. You'll need to
  copy map entries into a slice and `sort.Slice` them — maps have no order.
- **Multiple files.** Accept several filenames, print a line per file, and a
  `total` line at the end (again, just like `wc`).
- **Unicode-correct bytes vs. runes.** `len(s)` counts bytes; a function that
  counts *characters* needs `utf8.RuneCountInString`. Add a `-m` (char) mode and
  a test with an emoji or accented letter.
- **Benchmark it.** Add a `Benchmark...` function and run
  `go test -bench=.` on a large file.

## Free resources

- [**Learn Go with Tests**](https://quii.gitbook.io/learn-go-with-tests) — the
  best free, TDD-first Go book. Do the *Maps* and *Reading files* chapters; they
  are this project, explained in depth.
- [**`bufio` package docs**](https://pkg.go.dev/bufio) — read the `Scanner`
  section and the `ScanLines` / `ScanWords` split functions.
- [**`testing` package docs**](https://pkg.go.dev/testing) — `t.Run`, sub-tests,
  and table-driven patterns.

## Done when

- [ ] `go test ./...` passes, and you wrote the tests **before** the code.
- [ ] Your tests are **table-driven** and cover empty input, repeats, and mixed
      whitespace.
- [ ] `WordCount(s string) map[string]int` returns correct frequencies — and
      **passes the Step 1 checker challenge in the app.**
- [ ] `Count` works on a file *and* on piped stdin (`cat file | ./wordcount`).
- [ ] Errors go to stderr and the program exits non-zero on a bad filename.
- [ ] You attempted at least one stretch goal.

When the checker turns green, commit this project on your **Roadmap**. You've now
written, tested, and shipped a real tool — the same loop you'll repeat for every
distributed system in the steps ahead. Next up: **Step 2, structs and methods.**

## Common interview gotchas

- **`bufio.Scanner` silently stops on a line longer than 64KB** — its default token buffer caps at `bufio.MaxScanTokenSize`; a single huge line makes `Scan()` return false with `ErrTooLong`. Always check `scanner.Err()` after the loop (a false return can mean error *or* EOF), and call `scanner.Buffer(...)` to raise the limit. For unbounded lines use `bufio.Reader.ReadString`.
- **`len(s)` counts bytes, not characters** — a `wc -m` (char) count over `"héllo"` or an emoji is wrong with `len`; use `utf8.RuneCountInString`. Ranging a string yields runes, but indexing (`s[i]`) yields a byte. Know which one the spec wants.
- **Reading the whole file with `os.ReadFile` doesn't scale** — fine for a poem, fatal for a 10GB log; it loads everything into memory. Streaming through an `io.Reader` with `bufio` keeps memory flat regardless of input size — the reason `Count` takes a reader, not a filename.
- **Top-K is a heap, not a full sort** — sorting all N words to take the top 10 is O(N log N) and needless; a size-K min-heap (`container/heap`) is O(N log K) and bounds memory. Interviewers ask "now find the 10 most frequent words in a stream you can't fit in RAM" precisely to see if you reach for the heap.
- **Tie-breaking must be deterministic** — sorting top-K by count alone leaves equal-count words in map-iteration (random) order, so output differs run to run and tests flake. Sort by count *then* alphabetically.
