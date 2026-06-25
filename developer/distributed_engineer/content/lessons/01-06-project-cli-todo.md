---
slug: project-cli-todo-app
step: 1
title: "Project: build a command-line to-do app"
summary: Build a CLI to-do app with add/list/done commands that persists tasks to a JSON file using encoding/json.
est_min: 360
position: 6
---

# Project: build a command-line to-do app

> **Step 1 · Go fundamentals · Week 2 · Project**
> Concept: *I/O, encoding/json, slices of structs*

You've spent two weeks on syntax. Now you build something that *remembers* —
a real program that survives being closed and reopened. This is your first
encounter with the loop every backend service runs forever: **read state from
disk, change it in memory, write it back.** Treat it seriously; the muscles you
build here are the same ones you'll use to persist a Raft log in Step 8.

## What you'll build

A single-binary command-line to-do app called `todo`. It stores tasks in a
plain JSON file in the current directory and supports three subcommands:

```bash
todo add "buy milk"      # appends a new task
todo list                # prints all tasks with their status
todo done 2              # marks task #2 as completed
```

`todo list` should print something like:

```
[ ] 1  buy milk
[x] 2  email the landlord
[ ] 3  finish the to-do app
```

When you quit and run `todo list` again tomorrow, the tasks are still there.
That persistence is the whole point.

## Why this matters

This is the first program you'll write that has **state that outlives the
process**. Every service you build from here on does the same thing — it just
swaps the JSON file for a database, a message queue, or a replicated log. The
pattern is identical: deserialize bytes into structs, mutate the structs, then
serialize them back into bytes. Master `encoding/json` against a flat file now
and the leap to networked storage later is mostly plumbing.

You'll also meet **slices of structs** for the first time in anger — the
workhorse data shape of almost every Go program — and learn to read program
arguments off the command line, which is how every CLI tool on your machine
gets its instructions.

## 1. Model a task

Start with the data. A task needs an id, a title, and a done flag. The
**struct tags** in backticks tell `encoding/json` what to name each field in
the file — without them you'd get `Title` instead of the lowercase `title`
that's conventional in JSON.

```go
package main

type Task struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}
```

Your whole list is just a `[]Task` — a slice of structs. Keep that in mind:
loading the file gives you a slice, and saving writes the slice back.

## 2. Load the tasks (and survive the first run)

Loading reads the file, then unmarshals the bytes into a `[]Task`. The tricky
part is the **first run**, when the file doesn't exist yet. That's not an
error — it just means you have zero tasks. Detect it with `os.IsNotExist` and
return an empty slice.

```go
import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
)

const dataFile = "tasks.json"

func load() ([]Task, error) {
	bytes, err := os.ReadFile(dataFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []Task{}, nil // first run: no file yet, no tasks
		}
		return nil, err
	}

	var tasks []Task
	if err := json.Unmarshal(bytes, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}
```

Note `&tasks` — `Unmarshal` needs a *pointer* so it can fill in the slice you
gave it.

## 3. Save the tasks

Saving is the mirror image: marshal the slice to bytes, write the bytes.
Reach for `json.MarshalIndent` so the file is human-readable — you'll want to
peek at it while debugging.

```go
func save(tasks []Task) error {
	bytes, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dataFile, bytes, 0o644)
}
```

`0o644` is the file permission (owner read/write, others read). Open
`tasks.json` in your editor after the first `add` and confirm it looks the way
you expect.

## 4. Read the subcommand from os.Args

`os.Args` is a `[]string` of everything typed on the command line.
`os.Args[0]` is the program name itself, so the **subcommand** is
`os.Args[1]` and any further arguments follow it.

```go
func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: todo [add|list|done] ...")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "add":
		// your turn — see section 5
	case "list":
		// your turn — see section 6
	case "done":
		// your turn — see section 7
	default:
		fmt.Printf("unknown command: %q\n", os.Args[1])
		os.Exit(1)
	}
}
```

Always check `len(os.Args)` before indexing into it — reaching past the end of
a slice panics, and a half-typed command shouldn't crash your program.

## 5. Implement `add`

Here's the one command written out, so you can see the full load → mutate →
save loop end to end:

```go
case "add":
	if len(os.Args) < 3 {
		fmt.Println("usage: todo add <title>")
		os.Exit(1)
	}
	tasks, err := load()
	if err != nil {
		fmt.Println("error loading:", err)
		os.Exit(1)
	}
	task := Task{ID: len(tasks) + 1, Title: os.Args[2]}
	tasks = append(tasks, task)
	if err := save(tasks); err != nil {
		fmt.Println("error saving:", err)
		os.Exit(1)
	}
	fmt.Printf("added #%d: %s\n", task.ID, task.Title)
```

> Generating the id with `len(tasks) + 1` is fine for now. **Your turn (stretch):**
> what breaks once you add a `delete` command? Think about it — there's a fix in
> the stretch goals.

## 6. Implement `list` — your turn

You write this one. It should:

1. `load()` the tasks.
2. If the slice is empty, print something friendly like `no tasks yet`.
3. Otherwise loop with `for _, t := range tasks` and print each line.

Use a checkbox that reflects `t.Done`. A `map[bool]string` or a small `if`
gets you the `[ ]` / `[x]` marker:

```go
box := "[ ]"
if t.Done {
	box = "[x]"
}
fmt.Printf("%s %d  %s\n", box, t.ID, t.Title)
```

Wire that into a loop yourself.

## 7. Implement `done` — your turn

This is the most interesting command, because you mutate the slice in place
and save it. It should:

1. Parse `os.Args[2]` into an int with `strconv.Atoi` (handle the error — a
   non-number argument shouldn't crash).
2. `load()` the tasks.
3. Find the task whose `ID` matches and set its `Done` field to `true`.
4. `save()` and confirm, or report that no task had that id.

A subtle Go gotcha to wrestle with: ranging with `for i, t := range tasks`
gives you a **copy** of each task in `t`. Setting `t.Done = true` changes the
copy, not the slice. Index back into the slice — `tasks[i].Done = true` — to
mutate the real element. Discovering this for yourself is half the lesson.

## Stretch goals

Only after the three core commands work:

- **`delete <id>`** — remove a task from the slice. Now reckon with the id
  problem from section 5: switch to a counter you persist (e.g. a `nextID`
  field in a wrapping struct), or never reuse ids.
- **Due dates** — add a `Due time.Time` field with a `json:"due"` tag and let
  `add` accept an optional date.
- **Priorities** — add a `Priority int`, and make `list` sort by it with
  `sort.Slice`.
- **A custom data path** — read the file location from a `TODO_FILE`
  environment variable via `os.Getenv`, falling back to `tasks.json`.

## Done when

- [ ] `todo add "..."` creates `tasks.json` on first run with no crash.
- [ ] `todo list` prints every task with the right `[ ]` / `[x]` marker.
- [ ] `todo done <id>` actually flips the flag *and the change survives* a
      restart (you mutated the slice, not a copy).
- [ ] A bad command, a missing argument, and a non-numeric id all print a
      helpful message instead of panicking.
- [ ] You can open `tasks.json` by hand and the JSON looks clean and indented.

## Sources

- [**Learn Go with Tests**](https://quii.gitbook.io/learn-go-with-tests) —
  free, test-driven, and the best way to internalize Go idioms. The *Reading
  files* and *JSON* chapters map directly onto this project.
- [**`encoding/json` package docs**](https://pkg.go.dev/encoding/json) — keep
  `Marshal`, `Unmarshal`, and `MarshalIndent` open in a tab as you build.

When all five boxes are checked, commit this project on your **Roadmap**. You've
just written a stateful program. Next lesson begins **Step 2: concurrency.**

## Common interview gotchas

- **Only *exported* (capitalized) fields are marshaled** — a lowercase `title string` field is invisible to `encoding/json`; it silently round-trips as empty/zero with no error. If a field must persist, it must be exported, with a struct tag if you want a lowercase JSON name.
- **`os.WriteFile` is not atomic — a crash mid-write truncates your data** — it opens, truncates, then writes; a panic or power loss in between leaves a corrupt or empty file. Write to a temp file in the *same directory* and `os.Rename` it over the target; rename is atomic on the same filesystem.
- **Two processes (or goroutines) running `load → mutate → save` will clobber each other** — last writer wins and silently drops the other's changes; there's no locking in `ReadFile`/`WriteFile`. A single CLI invocation is fine, but the moment two run concurrently you need a file lock or a real database.
- **JSON has no integers — every number unmarshals into `float64`** when the target is `any`/`interface{}`. An `int` ID read into a `map[string]any` comes back as `float64`, and large IDs lose precision past 2^53. Unmarshal into a typed struct (`ID int`) so the decoder converts correctly.
- **`json.Unmarshal` into a non-empty slice doesn't reset it** — it appends/reuses, so reusing a slice across loads can leave stale entries. Decode into a fresh `var tasks []Task` each time.
