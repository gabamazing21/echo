---
slug: jobhunt-oss-contribution
step: 10
title: "Land a real open-source contribution"
summary: Find and fix a 'good first issue' in a Go distributed project — the strongest external signal of real-world skill.
est_min: 600
position: 3
---

# Land a real open-source contribution

> **Step 10 · Visible proof & job hunt · Week 2 · Task**
> Concept: *real open-source signal*

Everything you've built so far lives in *your* repos, on *your* terms. This task
is different. You're going to walk into a real, running distributed system that
strangers depend on, find one small broken thing, and fix it well enough that a
maintainer presses **Merge**. That single green checkmark is worth more to a
hiring manager than a dozen personal projects — because you don't control it.

## Why this matters

A merged pull request to a real distributed system — **etcd**, **NATS**,
**Temporal**, **CockroachDB**, **Vitess** — is *external, verifiable proof* that
you can:

- read an unfamiliar Go codebase and find your way around it,
- work inside someone else's conventions and CI,
- take review feedback without ego and iterate,
- and ship something a careful maintainer was willing to put their name next to.

Anyone can write "proficient in Go and distributed systems" on a résumé. A link
to `github.com/etcd-io/etcd/pull/12345` with your name on it *ends the argument*.
Recruiters skim; engineers click that link and read the diff. It is the single
highest-leverage line you can add to your job hunt this week.

> You are not trying to redesign Raft. You are trying to fix **one tiny thing**,
> correctly, the way the project wants it fixed. Tiny and merged beats ambitious
> and abandoned every time.

## Choosing a project & issue

Pick a project you already *use* or *study*. You've read about Raft in Step 8 —
etcd is the canonical Go implementation and is famously welcoming to newcomers.
Good candidates, all Go, all distributed:

| Project | What it is | Why it's a good first target |
|---|---|---|
| **etcd** | Distributed key-value store (Raft) | Active `good first issue` queue, clear `CONTRIBUTING.md` |
| **NATS** | High-performance messaging system | Small focused issues, friendly maintainers |
| **Temporal** | Durable workflow engine | Lots of docs/test gaps labeled for newcomers |
| **Vitess** | Horizontally-scaled MySQL | Big surface area, many small bugs |

### Find the on-ramp: the `good first issue` label

Mature projects deliberately tag beginner-friendly work. The label is almost
always literally **`good first issue`** (sometimes `help wanted` or
`good-first-issue`). Go straight to the filtered issue list:

- [etcd good-first-issues](https://github.com/etcd-io/etcd/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22)

That URL is a template. Swap the org/repo to scan any project, or use GitHub's
search bar: `is:issue is:open label:"good first issue"`.

### What makes a *good* first issue

- **Small and bounded.** A typo in docs, a missing test case, a confusing error
  message, a flaky log line. You should be able to describe the fix in one
  sentence.
- **Unclaimed and recent.** Skip issues where someone already commented "I'll
  take this" — that's their PR to write. Prefer issues with recent activity over
  ones that have sat dead for two years.
- **Understood.** If you can't explain *why* it's a bug after reading the issue,
  it's not your first issue. Move on. There's no shame in passing.

> **Docs and tests count.** A first contribution that fixes a broken doc example
> or adds a missing test case is a *completely legitimate* merged PR. Maintainers
> love these because they're low-risk and genuinely useful. Start there.

## The contribution workflow

The mechanics are the same Git skills you already have, with extra care because
you're a guest in someone's house.

### 1. Read `CONTRIBUTING.md` first — actually read it

Every serious project has a `CONTRIBUTING.md` (and often a `DCO`/CLA step and a
`CODE_OF_CONDUCT.md`). It tells you the branch naming, the commit format, how to
run the tests, and how to sign off. **Skipping it is the fastest way to get your
PR closed.** Read it before you write a line of code.

### 2. Claim the issue — briefly and politely

Comment on the issue to signal you're working on it, and ask a *focused* question
if you genuinely have one:

```
Hi! I'd like to work on this. My plan is to add a table-driven test
covering the empty-key case in store_test.go — does that match what you
had in mind, or is there a broader scenario you'd prefer?
```

Don't ask "can someone explain how etcd works?" — that's homework, not a
question. Ask the *one* thing you can't resolve by reading the code.

### 3. Fork, clone, branch

```bash
# Fork via the GitHub UI, then:
git clone git@github.com:YOUR_USER/etcd.git
cd etcd
git remote add upstream https://github.com/etcd-io/etcd.git
git checkout -b fix/empty-key-test
```

Naming the branch after the change (`fix/...`, `docs/...`, `test/...`) keeps
things tidy and signals intent.

### 4. Make a small, focused change — and *test it*

One issue, one concern. Resist the urge to also reformat that ugly function next
door — that's scope creep and reviewers hate it. Then prove it works, using the
exact commands `CONTRIBUTING.md` specifies (often a `make test` target):

```bash
go test ./...        # or the project's documented test command
go vet ./...
gofmt -l .           # should print nothing
```

If you fixed a bug, **add a test that fails before your fix and passes after.**
That's the table-driven `t.Run` pattern from Step 1 — it's what turns "I think
this is fixed" into proof.

### 5. Commit in the project's style (often Conventional Commits)

Many Go projects use **Conventional Commits**: a `type(scope): subject` line,
imperative mood, with a body explaining *why*. Check `CONTRIBUTING.md` for the
exact rule, and remember the **sign-off** (`-s`) most projects require:

```bash
git commit -s -m "test(store): cover empty-key lookup

The empty-key path in Get had no test coverage; this adds a
table-driven case so a regression here fails CI. Fixes #12345."
```

### 6. Push and open the PR

```bash
git push origin fix/empty-key-test
```

Open the PR against `upstream/main` and write a description a busy maintainer can
approve in two minutes:

- **What** changed and **why** (link the issue: `Fixes #12345`).
- **How you verified it** — paste the passing test output.
- A note on scope: "Intentionally minimal; happy to expand if you'd like."

A clear PR description is itself a signal of seniority. Sloppy PR, sloppy
engineer — fair or not, that's how it reads.

## Review etiquette

This is where most first-timers stumble, and it has nothing to do with code.

- **Respond graciously, always.** A maintainer reviewing your PR is doing you a
  favour on their own time. "Good catch, fixed in `a1b2c3d`" is the whole game.
- **Iterate without arguing.** If they ask for a change, make it. If you
  genuinely disagree, explain your reasoning *once*, calmly, then defer — it's
  their project, their call.
- **Push follow-ups, don't force-rebase chaos.** Add commits addressing feedback
  so reviewers can see what changed; squash at the end only if they ask.
- **Be patient.** Maintainers are busy and unpaid. A polite "gentle ping — is
  there anything else you need from me here?" after a week is fine. Nagging daily
  is not.
- **Say thank you when it merges.** It costs nothing and people remember it.

> Reviewers aren't gatekeepers being difficult — they're protecting a system
> thousands of people run in production. Treat their feedback as mentorship,
> because that's exactly what it is.

## Do it yourself

1. Pick **one** Go distributed project you find interesting. Read its
   `README.md` and clone it. Get the test suite green *before* you change
   anything — `go test ./...` (or its documented equivalent).
2. Read `CONTRIBUTING.md` end to end. Note the commit format, the sign-off/CLA
   step, and the test command.
3. Browse the
   [etcd good-first-issues](https://github.com/etcd-io/etcd/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22)
   list (or your project's equivalent). Find **one** small, unclaimed, recent
   issue you fully understand.
4. Comment to claim it and ask your one focused question if you have one.
5. Fork → branch → make the smallest correct change → add or update a test →
   conventional, signed-off commit → push → open a PR with a clear description.
6. Respond to every review comment within a day or two, graciously, and iterate
   until it merges (or is clearly stalled — then move on and try another).
7. Read **[How to Contribute to Open Source](https://opensource.guide/how-to-contribute/)**
   — it is the definitive field guide to everything above and worth the hour.

## Check yourself

You're ready to call this done when you can answer, *without looking*:

- Where do you find beginner-friendly work, and what's the label called?
- Why read `CONTRIBUTING.md` *before* writing code?
- What does a good first issue look like — and why are docs/tests fair game?
- What goes in a PR description that lets a maintainer approve it in two minutes?
- How do you respond when a reviewer asks for a change you'd rather not make?

**Done when:** you have **one PR opened** against a real Go distributed project —
ideally merged, but an open, well-formed PR under active review already counts as
the signal. Drop the PR link on your **Roadmap**, add it to your résumé and
LinkedIn, and take the Step 10 quiz. A merged PR to a system the industry runs in
production is gold — guard that link and lead with it.

## Common interview gotchas

- **A "good first issue" can be stale or mislabeled.** Many are 2+ years old, already fixed, or secretly hard. Check the last-updated date and comment "is this still relevant / may I take it?" *before* you start — it shows awareness and avoids wasted work on a dead issue.
- **Read CONTRIBUTING.md and match the project's conventions, or you get auto-rejected.** Skipping the DCO/CLA, commit-message format, or test/lint requirements gets a PR closed regardless of how good the code is. Run their full check suite locally first.
- **A rejected or unmerged PR is not a failure.** A polite, well-formed PR — even one a maintainer declines — is still real signal of initiative and collaboration. Thank them, apply the feedback, move on; don't argue or ghost.
- **Don't force-push over review history or balloon the scope.** Reviewers need to see incremental changes; rewriting the branch mid-review erases their context. Keep the change small and focused — a 10-line fix that merges beats a 500-line refactor that stalls.
- **Don't claim a merged PR you only opened.** Interviewers will open the link and read the diff and the conversation. Describe your contribution honestly ("opened a PR, under review" vs "merged"); the diff itself is the proof.
