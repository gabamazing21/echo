---
slug: sd-mock-practice
step: 9
title: Mock system-design practice
summary: How to rehearse system-design out loud — five prompts, a self-scoring rubric, and the communication habits that pass interviews.
est_min: 480
position: 5
---

# Mock system-design practice

> **Step 9 · System design · Week 2 · Practice**
> Concept: *communication under pressure*

You've learned the framework. This lesson is different: it's not about *knowing*
how to design a system, it's about **performing** that knowledge out loud, on a
clock, while a stranger watches and interrupts. The hard part of a system-design
interview is almost never the design. It's the communication — clarifying a vague
prompt, narrating your thinking, driving a 45-minute conversation without stalling,
and steering a blank whiteboard toward a coherent picture. That's a *skill*, and
like every skill it only improves with deliberate, repeated rehearsal.

## Why this matters

A system-design interview is not an exam where you submit the right answer. It's a
**simulated working session** where the interviewer is pretending to be a colleague
you're designing with. They are scoring your judgement, your communication, and how
you behave when the problem is deliberately under-specified — not whether you
produced the one true architecture (there isn't one).

This is why strong engineers still fail these interviews. They *know* about load
balancers and sharding and consistent hashing, but they freeze when the prompt is
two words ("Design Twitter"), they go silent for five minutes while thinking, or
they dive into database schemas before anyone agreed what the system even does. The
knowledge was there; the *performance* fell apart. The only fix is to rehearse the
performance until it's automatic — and you can't rehearse it silently in your head.
You have to say the words.

## How to practice

The single most important rule: **practice out loud.** Designing a system in your
head is a completely different skill from designing it while talking. Your goal is
to make the talking automatic so your brain is free to think about the design.

**Time-box the framework from lesson 09-01 to ~40 minutes.** A real interview is
45–60 minutes; rehearse at the tight end so the real thing feels roomy. A workable
split:

| Phase | Time | What you're doing |
|---|---|---|
| **Clarify requirements** | ~5 min | Functional + non-functional, scope, who uses it |
| **Estimate** | ~5 min | QPS, storage, bandwidth — back-of-envelope numbers |
| **API & data model** | ~5 min | Core endpoints, the main entities and their fields |
| **High-level design** | ~10 min | Boxes and arrows: clients, services, stores, queues |
| **Deep dive & bottlenecks** | ~10 min | Pick the hard part; scale it, find the failure modes |
| **Trade-offs & wrap-up** | ~5 min | CAP choices, what you'd do with more time |

Keep a visible timer. When it goes off, *move on* even if you're not finished —
running out of time on one section is itself an interview failure mode you need to
feel and fix.

**Think aloud, always.** Narrate every decision: "I'll use a message queue here so
writes don't block the user — the trade-off is eventual consistency on the feed."
Silence reads as being stuck. A running commentary reassures the interviewer and,
crucially, lets them *redirect* you before you waste ten minutes down a wrong path.

**Drive the conversation.** *You* own the whiteboard, not the interviewer. State
what you're going to do next ("Let me size this before I draw anything"), then do
it. Don't wait to be asked. The candidates who pass lead the session; the ones who
fail wait for instructions.

**State your assumptions out loud and write them down.** "I'll assume 100M daily
active users and a read-heavy workload, 100:1 reads to writes — stop me if that's
wrong." This turns a vague prompt into a concrete problem *and* shows judgement. If
the interviewer corrects an assumption, great — adjust and continue.

**Manage the whiteboard like a shared document.** Reserve a corner for requirements
and assumptions so they stay visible. Draw the high-level diagram once, in the
center, and leave room around it to annotate. Don't erase the big picture to do a
deep dive — use the margins. A legible board *is* part of your communication score.

**Record yourself.** Phone audio is enough; video is better. Then watch it back —
this is the part everyone skips and it's where the real gains are. You will hear the
"ums," the thirty-second silences, the moment you started coding before clarifying.
You cannot feel these in the moment; you can only catch them on the replay.

## The five prompts

Run one per session, cold, with a timer. Don't read about the system first — that
defeats the purpose. The one-line hint is the *trap* each prompt is testing, not the
answer.

1. **Design a rate limiter.**
   *Hint:* nail the algorithm trade-offs — token bucket vs. sliding window vs. fixed
   window — and where state lives (local vs. centralized/Redis) in a distributed
   setup.

2. **Design a news feed (Twitter/Facebook timeline).**
   *Hint:* the whole interview is fan-out-on-write vs. fan-out-on-read, and how you
   handle celebrity accounts with millions of followers.

3. **Design a notification service (push/email/SMS).**
   *Hint:* it's a queue-and-fan-out problem — decoupling, retries, deduplication,
   and per-user/per-channel delivery preferences.

4. **Design Pastebin (or a URL shortener).**
   *Hint:* deceptively simple — spend your time on key generation/collisions,
   read-heavy caching, and expiry, not on the obvious CRUD.

5. **Design typeahead / autocomplete (search suggestions).**
   *Hint:* low-latency reads dominate — a trie, prefix caching, and how you rank and
   update suggestions from a firehose of queries.

Cycle through all five, then start again. The second pass on the same prompt should
feel dramatically smoother — that delta is the skill being built.

## The scoring rubric

Right after each mock — before you forget — score yourself honestly out of these
seven points. Watch your recording to grade fairly; you'll be generous from memory
and accurate from the replay.

| # | Did you... | Pass condition |
|---|---|---|
| 1 | **Clarify requirements?** | Stated functional + non-functional reqs and scope *before* designing |
| 2 | **Estimate?** | Produced rough QPS / storage / bandwidth numbers, not just hand-waving |
| 3 | **Define an API & data model?** | Named the core endpoints and the main entities/fields |
| 4 | **Identify bottlenecks?** | Found where it breaks under load and did a real deep dive on one |
| 5 | **Discuss trade-offs / CAP?** | Made explicit choices (consistency vs. availability, etc.) and justified them |
| 6 | **Communicate clearly?** | Thought aloud throughout, no long silences, drove the conversation |
| 7 | **Manage time?** | Hit every phase within the ~40-minute box; nothing got skipped for time |

Log your scores in a tiny table — date, prompt, points hit. A point you miss two
sessions in a row is your homework: it's almost always estimation or time
management, the two things people most want to skip. Drilling the rubric points you
keep dropping is the whole game.

## Resources

These are free and more than enough:

- The [**System Design Primer**](https://github.com/donnemartin/system-design-primer)
  — the canonical open-source study guide. Use its worked examples to *check* your
  own attempt **after** you've done the mock cold, never before.
- [**ByteByteGo**](https://www.youtube.com/@ByteByteGo) on YouTube — short, visual
  walkthroughs of exactly these kinds of designs. Watch one *after* attempting the
  matching prompt and compare your diagram to theirs.

Use both as the "read the solution" step from your DSA practice loop: struggle
first, review second, re-attempt third.

## Done when

- You've completed **5 full mocks** — one per prompt — out loud and recorded.
- You **hit all seven rubric points** in your most recent mock, not just your best one.
- You finish **within the ~40-minute time-box** without skipping a phase.
- You can **start any of the five prompts cold** and clarify requirements within the
  first two minutes, without freezing.
- On replay, your narration is **continuous** — no silences longer than a few
  seconds, and you visibly *drove* the session rather than waiting to be led.

When all five are true, you've turned knowledge into performance. Commit this task
on your **Roadmap** and take the Step 9 quiz. Next step: **Step 10 — the behavioural
interview.**

## Common interview gotchas

- **Drilling only these five prompts.** They're a starting set, not the universe. Real interviews also pull from search, analytics pipelines, recommendations, and collaborative editing — practice the *framework* so any prompt is tractable, don't memorize five canned answers.
- **Over-weighting estimation, under-weighting trade-offs.** Candidates obsess over getting QPS math exact. The rubric rewards *identifying the bottleneck* and *making defensible trade-offs* far more — a rough order-of-magnitude estimate that feeds a real scaling decision beats precise numbers that go nowhere.
- **Going quiet while thinking.** Silence reads as being stuck even when you're reasoning hard. Narrate continuously; it both reassures the interviewer and lets them redirect you. On the replay, the long pauses are the single most damning thing you'll hear.
- **Skipping the "now scale it 10×" answer.** Always volunteer what breaks first under more load and your next move — don't wait to be asked. Closing each design by stating what you'd change at 10× is a top-of-rubric signal.
- **Practicing in your head instead of out loud.** Designing silently is a different skill from performing under a clock. Record yourself, watch it back, and grade against the rubric — the gains live in the replay everyone skips.
