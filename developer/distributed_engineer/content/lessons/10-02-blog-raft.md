---
slug: jobhunt-blog-raft
step: 10
title: "Write a blog post explaining your Raft implementation"
summary: Teaching is proof of depth — turn your Raft build into a clear technical write-up that signals real understanding.
est_min: 300
position: 2
---

# Write a blog post explaining your Raft implementation

> **Step 10 · Visible proof & job hunt · Week 1 · Task**
> Concept: *teaching = proof of depth*

You implemented Raft in Step 8. That's already rare. But a green commit history
doesn't *explain itself* — a hiring manager skimming your GitHub can't tell, in
ten seconds, whether you understood the algorithm or copied a tutorial. A blog
post can. This task turns your build into a write-up that does the convincing for
you, while you sleep.

## Why this matters

There's an old teacher's saying: you don't really understand something until you
can explain it to someone else. It's true, and employers know it. **The ability
to take a genuinely hard thing — distributed consensus — and make it clear is the
strongest signal of depth you can send.** Anyone can list "Raft" as a skill.
Almost no one can walk through *why* leader election needs randomized timeouts,
or *what* breaks if you commit an entry from a previous term by counting replicas.

A good post does three jobs at once:

- **It proves understanding.** Clear writing is impossible to fake. If you can
  explain the commitment rule, you understood the commitment rule.
- **It's a portfolio piece that scales.** One afternoon of writing gets read by
  recruiters, interviewers, and strangers on the internet for years. It's the
  highest-leverage artifact in this whole step.
- **It gives the interview a script.** When someone asks "tell me about a hard
  project," you've already rehearsed the answer — in public, in order, with the
  hard parts highlighted.

You're not writing for upvotes. You're writing the thing you'll wish you had in
front of you during a system-design interview.

## What to write about

The subject is the [Raft paper](https://raft.github.io/raft.pdf) — *In Search of
an Understandable Consensus Algorithm* — as seen through *your* implementation.
You are not re-deriving the paper. You're reporting from the trenches: here is
what the paper says, here is what it was actually like to build, here is where my
mental model was wrong.

The richest material is the gap between reading and doing. The paper makes terms,
elections, and log matching look tidy. Your code remembers the 2 a.m. session
where a split vote wouldn't resolve, or where a follower's log silently diverged.
**That gap is your content.** Write the post you'd have wanted before you started.

## The outline

A reliable structure — adapt freely, but hit these beats in roughly this order:

1. **The problem consensus solves.** Open here, not with code. A cluster of
   unreliable machines, dropped and reordered messages, and yet they must agree on
   one ordered log. Why this is hard, in two or three sentences. This frames
   everything that follows.

2. **The big idea: one leader, one log.** How Raft tames the problem by electing a
   single leader and funneling all changes through it. Roles: leader, follower,
   candidate. Terms as a logical clock. Keep it tight — you're orienting the
   reader, not transcribing Section 5.

3. **The aha moments.** The two or three insights that made it click for you.
   Candidates: *why majorities work* (any two majorities overlap in at least one
   node); *why terms make stale leaders harmless*; *why randomized election
   timeouts break symmetry*. Pick the ones you actually felt land.

4. **A bug you hit and how you debugged it.** This is the heart of the post and
   the part interviewers love. Describe one concrete failure — a flaky election, a
   log that wouldn't converge, a liveness bug from holding a lock across an RPC.
   Show how you *found* it: the test that flapped, the log line that gave it away,
   the hypothesis you formed and confirmed. Debugging stories are where real
   engineers recognize each other.

5. **What you'd do differently.** Honest reflection. Maybe your locking was too
   coarse, your tests too slow, your RPC layer too clever. This section signals
   maturity — you can critique your own work.

A short closing that links the repo and invites questions is plenty. No grand
conclusion needed.

## Writing tips

- **Lead with the insight, not the setup.** Don't make readers earn the payoff
  through three paragraphs of preamble. State the interesting thing, *then*
  explain how you got there. "Two majorities always overlap — that single fact is
  why Raft can't elect two leaders" is a better opener than a history of consensus.
- **Show a diagram.** One picture of a cluster electing a leader, or a log
  diverging and reconverging, does more than five paragraphs. A hand-drawn photo
  or a quick boxes-and-arrows sketch is completely fine — don't let tooling stall
  you.
- **Keep code snippets small.** Five to fifteen lines that illustrate *one* idea.
  Never paste a whole file; link to it on GitHub instead. The snippet is a
  pointer, not the source of truth.
- **Be honest about the struggles.** The bug section is the most valuable part
  precisely because it's vulnerable. "I lost a day to a deadlock from holding the
  mutex during an RPC" reads as competence, not weakness. Polished-perfection
  posts are forgettable; honest ones get bookmarked.
- **Write for one specific reader** — a peer who knows Go but hasn't read the Raft
  paper. That single constraint fixes your vocabulary, your pacing, and how much
  you explain.
- **Cut ruthlessly.** A focused 1,200-word post beats a sprawling 4,000-word one.
  If a paragraph doesn't advance the story, delete it.

## Where to publish

Pick a platform and ship — the worst choice is not publishing.

- **Your personal site.** Best if you have one: you own the URL and the SEO, and
  it doubles as a portfolio. A plain static-site setup is enough.
- **[dev.to](https://dev.to/).** Free, developer-focused, friendly to Markdown and
  code blocks, with a built-in audience that reads exactly this kind of post. The
  lowest-friction place to start.
- **Medium.** Wide reach and a clean reader; consider a publication for
  distribution. Watch the metered paywall if you want it fully open.
- **Hashnode.** Developer-first, maps to your own domain for free, good code
  support. A strong middle ground between dev.to and rolling your own.

Cross-posting is fine — write once, syndicate, and point a canonical link back to
your primary copy.

Then make it *findable*:

- **Link it from your GitHub README.** Add a line near the top of the Raft repo:
  "Read the write-up →". The post explains the code; the code backs the post. They
  sell each other.
- **Post it on LinkedIn** with two or three sentences on what you learned. This is
  where recruiters actually look. A real technical post outperforms any list of
  buzzwords on your profile.

## Done when

- The post is **published** at a public URL anyone can open without an account.
- It covers the problem, the design, at least one **aha moment**, and one **real
  bug** with how you debugged it.
- A diagram is included.
- It's **linked** from your Raft repo's README and shared on LinkedIn.

When the link is live and pointing both ways with your repo, commit this task on
your **Roadmap**. You've now turned a hard build into proof anyone can verify in
two minutes — which is exactly what gets you the interview.

## Common interview gotchas

The post proves you can write; the interview tests whether you can *defend and teach* it live. Where strong writers stumble out loud:

- **Reaching for jargon to sound deep.** When asked to "explain consensus simply," start from the problem and an analogy — introduce "term" or "log" only after the intuition lands. Performing vocabulary is the opposite of the skill being tested; making a hard idea feel obvious is the win.
- **Getting defensive when a claim is challenged.** "I'm not convinced randomized timeouts are necessary — defend it" is a stress-test, not an attack. Restate the objection, argue from reasoning, and *volunteer the limits* of your claim. Doubling down with ego loses; conceding a fair point fast scores.
- **Explaining everything instead of the one thing.** "Teach me the hardest concept in two minutes" rewards compression: one load-bearing insight plus one concrete example. Depth is proven by what you leave out, not by emptying your head.
- **No example when one is asked for.** Abstract restatement ("the commitment rule ensures safety") is not teaching. Have a concrete two-then-three-node scenario ready that the listener can hold in their head.
- **"Why write a blog?" answered with "for my resume."** The honest, stronger answer: writing forced you to find and fix the gaps in your own understanding. Signal-only motivations read as shallow.
