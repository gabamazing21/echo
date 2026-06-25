---
slug: jobhunt-polish-projects
step: 10
title: "Polish your 3 best projects for GitHub"
summary: Turn your strongest projects into portfolio pieces — clear READMEs, tests, and CI that prove the skill.
est_min: 480
position: 1
---

# Polish your 3 best projects for GitHub

> **Step 10 · Visible proof & job hunt · Week 1 · Task**
> Concept: *demonstrated, not claimed, skill*

You've spent ten steps writing real Go: an HTTP API, concurrency primitives, a
consensus engine. You *know* this stuff. But a recruiter scanning your GitHub for
forty seconds doesn't know it yet — they only see what's in front of them. This
task is about closing that gap. You will take your three strongest projects and
turn them into **portfolio pieces**: repos that prove, on sight, that you can
build and ship.

## Why this matters

The job market runs on a brutal asymmetry. You spent weeks on a project; a
hiring manager gives your repo a glance. "I'm good at Go and distributed systems"
is a *claim* — and every applicant makes it. A clean repo with a real README,
passing tests, and a green CI badge is **proof**. It's the difference between
*telling* someone you can do the work and *showing* them code that already does.

This is the whole concept of this step: **demonstrated, not claimed, skill.** A
résumé line is a claim. A link to a repo that builds, tests, and passes CI in
public is evidence. Engineers who get interviews are the ones whose GitHub does
the arguing for them before a human ever reads their cover letter.

## What makes a repo credible

Open any repo you respect and notice what's there. A credible repo signals
professionalism in the first ten seconds:

- **A README that orients you instantly** — what it is, why it exists, how to run it.
- **Tests that actually run** — `go test ./...` passes, and there are enough tests to show you think about correctness.
- **A green CI badge** at the top of the README — proof the tests pass *right now*, not just on your machine.
- **A sensible layout** — `cmd/`, package directories, a `go.mod`, no committed binaries or `node_modules`-style junk.
- **A real commit history** — small, described commits beat one giant "initial commit."

The opposite — no README, a wall of untested code, a `main.go` with everything
in it — reads as "student project." You're past that. Your repos should read as
"this person ships."

## The README template

The README is the single highest-leverage file in the repo, because it's the one
recruiters *actually read*. Write it for a stranger who has thirty seconds. Here's
an outline that works for every project you've built:

```markdown
# Project Name

> One-sentence description of what this does and why it's interesting.

![CI](https://github.com/you/project/actions/workflows/ci.yml/badge.svg)

## What & why

A short paragraph: what problem this solves and what makes it worth a look.
For the bookmarks API: "A REST API for saving and tagging bookmarks, built to
practice idiomatic Go HTTP handlers, middleware, and Postgres."

## Demo

A GIF of the CLI/UI in action, or a copy-pasteable example:

\`\`\`bash
curl -s localhost:8080/bookmarks | jq
\`\`\`

## Running it

\`\`\`bash
git clone https://github.com/you/project
cd project
go run ./cmd/server
\`\`\`

## Architecture

A few sentences (or a small diagram) on how it's put together: the main
packages, the request flow, where state lives. For the worker pool: "N worker
goroutines pull jobs off a buffered channel; a token-bucket limiter gates
submission. See `pool/pool.go`."

## What I learned

Two or three honest bullets. This section is gold for interviews — it shows
reflection. "Learned why a buffered channel alone isn't backpressure," etc.

## Running the tests

\`\`\`bash
go test ./...
\`\`\`
```

Keep it tight. A README that's too long doesn't get read either. The **what/why**,
a **demo**, **how to run**, and an **architecture** note are the load-bearing
sections. The **what I learned** section is what makes *you* memorable.

> A demo GIF is worth a hundred lines of prose. Record your terminal with a tool
> like `asciinema` or `vhs`, or capture a screen GIF, and drop it right under the
> title. A recruiter who *sees* it work doesn't need to clone anything.

## Adding CI

Continuous integration runs your tests automatically on every push, on a clean
machine that isn't yours. That last part matters: "works on my machine" is a
punchline, and CI is the cure. When the badge is green, anyone — including you in
six months — knows the project still builds and passes.

GitHub Actions is free for public repos and needs exactly one file. Create
`.github/workflows/ci.yml`:

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.21"

      - name: Vet
        run: go vet ./...

      - name: Test
        run: go test ./...
```

What each piece does:

- `on:` — run this workflow on every push to `main` and on every pull request.
- `actions/checkout@v4` — pull your code onto the runner.
- `actions/setup-go@v5` — install the Go toolchain you developed against.
- `go vet ./...` — catch suspicious constructs (bad `Printf` verbs, unreachable
  code) before they reach a reviewer.
- `go test ./...` — the same command you run locally, now enforced on every push.

Commit and push that file. Open the **Actions** tab on GitHub and watch the run.
Once it's green, add the badge to the top of your README (the
`actions/workflows/ci.yml/badge.svg` URL in the template above — swap in your
username and repo). That little green badge is the most efficient credibility
signal on the whole page.

> Tip: keep CI *fast* and *honest*. If a test is flaky, fix it or delete it — a
> CI badge that's sometimes red teaches you to ignore it, which defeats the
> point. A reliable green is worth more than a thorough-but-flaky suite.

## Picking your 3

Three is the right number: enough to show range, few enough that each one is
polished. Pick projects that, together, tell a story about what you can do.
A strong trio from this course:

1. **The bookmarks API** — shows you can build a real web service: HTTP handlers,
   middleware, a database, REST design. This is the most *relatable* project for
   most hiring managers, so make its README the best of the three.
2. **The worker pool / rate limiter** — shows you understand Go's concurrency
   model: goroutines, channels, backpressure, graceful shutdown. This is the
   project that says "I'm not just a CRUD developer."
3. **The Raft / KV store** — shows depth and ambition: consensus, replication,
   handling failure. Few applicants have *anything* like this. Even an honest
   "this implements leader election and log replication; partition handling is
   partial" is impressive, because it's real distributed-systems code.

Together they say: *I can build services, I understand concurrency, and I reach
for hard problems.* That's a memorable applicant. Spend the most polish on
whichever is closest to the job you want.

## Done when

Work through all three. This is a full-day task — budget the time, because the
polish is the point.

- [ ] **README** on all 3 repos, following the template: what/why, demo, how to
      run, architecture note, what I learned.
- [ ] **Real tests** in each repo — `go test ./...` passes locally, and the
      tests actually exercise the interesting logic (not just one trivial case).
- [ ] **CI** via `.github/workflows/ci.yml` on each repo, running `go vet` and
      `go test` on every push.
- [ ] **Green CI badge** at the top of each README, linking to the live Actions
      run.
- [ ] A friend or peer can clone any of the three and get it running using *only*
      the README — no asking you. If they can't, the README isn't done.

## Going further

- Read the **[GitHub Actions docs](https://docs.github.com/actions)** (free) —
  skim *Understanding GitHub Actions* and *Building and testing Go*. You don't
  need all of it; know what's there to look up.
- Add a `go test -race ./...` step to CI on the concurrency projects — the race
  detector catches bugs no amount of staring can.
- Add a short **architecture diagram** (even hand-drawn, photographed) to the
  Raft repo. Distributed-systems diagrams read as senior.

When all three repos are clean, tested, and green, commit this task on your
**Roadmap**. You now have something most applicants don't: proof. Next lesson:
**writing the résumé and the GitHub profile README that ties it together.**

## Common interview gotchas

The repo gets you the interview; the deep-dive gets you the offer. Where polished projects go wrong in the room:

- **"Walk me through your project" turns into a code tour.** Lead with the *problem*, then the big idea, then *one* level of depth — let the interviewer steer with follow-ups. Drowning them in implementation details on the opening question reads as no sense of altitude.
- **You can't survive the drill-down on your own README.** If your README claims "fault-injection harness, 300+ cases," be ready to explain exactly what it injects and what one test asserts. A claim you can't defend is worse than one you never made.
- **Overselling an honest-incomplete project.** "Partial partition handling" is a strength when you can draw the boundary precisely. Hand-waving "it mostly works" — then fumbling a follow-up — is the trap. Know exactly where your system's edges are.
- **"What would you do differently?" gets answered with "nothing."** That reads as no growth. Have two specific, *principled* improvements ready (the lesson, not just the fix) — but not ten, which reads as no judgment about what matters.
- **A green badge over a hollow suite.** If asked what the tests actually cover and the honest answer is "one trivial case," the badge backfires. Make sure your tests exercise the *interesting* logic before you point at the checkmark.
