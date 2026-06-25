---
slug: jobhunt-applying
step: 10
title: "CV, LinkedIn & landing the interviews"
summary: Package the proof you built into a CV and LinkedIn, then apply and reach out with signal, not spam.
est_min: 360
position: 4
---

# CV, LinkedIn & landing the interviews

> **Step 10 · Visible proof & job hunt · Week 2 · Task**
> Concept: *visibility = opportunity*

You built the proof. A Raft implementation. A replicated key-value store. Real
APIs with tests and a deploy. A blog post that explains a hard idea clearly. None
of it gets you hired until the right people *see* it. This lesson is about
turning six to eight months of work into interviews — and interviews into an
offer. It's the last task on the roadmap, and it's the one that pays for all the
others.

## Why this matters

Hiring is a search problem with terrible signal. A recruiter skims your CV for
six seconds; a hiring manager opens twenty profiles and closes eighteen. Most
candidates are indistinguishable on paper — a degree, some buzzwords, "passionate
about technology." You are not most candidates. You have *artifacts*: code that
runs, systems that handle failure, writing that teaches. Your entire job now is
to make that proof impossible to miss and effortless to verify.

**Visibility = opportunity.** The best engineer who is invisible loses to the
average engineer who is easy to find and easy to evaluate. You've done the hard
part. Don't fall at the last hurdle by hiding it.

## The CV

One page. Results-oriented. Projects first. That's the whole brief, and almost
nobody does it.

**Lead with what you built, not where you sat.** For someone coming through this
roadmap, the strongest section is *Projects*, and it goes near the top — above
education, above any unrelated work history.

A project entry has three parts: what it is, what it demonstrates, and a link.

```
Raft consensus implementation — Go
  Leader election, log replication, and persistence following the Raft
  paper; survives node crashes and network partitions in a 5-node cluster.
  Tested with a fault-injection harness (300+ table-driven cases).
  github.com/you/raft-kv
```

Notice what that does:

- **Names the hard thing** ("leader election, log replication") so a distributed
  engineer reading it knows you're not bluffing.
- **Quantifies** — 5 nodes, 300+ cases. Numbers read as truth; adjectives read as
  filler.
- **Links** straight to the code. The reviewer can verify in one click.

Rules that keep a CV strong:

- **One page.** If you're early-career, two pages means you padded. Cut.
- **Lead every bullet with a verb and an outcome**, not a responsibility.
  *"Built a replicated KV store that stays consistent across node failures"* beats
  *"Responsible for working on distributed systems."*
- **Quantify wherever honest:** latency, throughput, node counts, test counts,
  uptime, request volume. No real number? Describe the capability precisely.
- **Cut the noise:** no "References available on request," no objective statement,
  no skills bar charts, no photo. A clean *Skills* line (Go, gRPC, Postgres,
  Docker, Linux, distributed systems) is plenty.
- **Plain formatting.** Recruiters paste CVs into tracking systems that mangle
  fancy layouts. One column, standard fonts, export to PDF.

Order for a roadmap graduate: **Name + contact + GitHub/LinkedIn → Projects →
Experience → Skills → Education.** Projects *are* your experience. Treat them
that way.

## LinkedIn

Recruiters live on LinkedIn. A profile that mirrors your CV but is *findable* is
the highest-leverage hour you'll spend.

- **Headline** — this is what shows in search results. Not "Aspiring developer."
  Write what you do: *"Backend / Distributed Systems Engineer — Go, Raft, gRPC,
  Postgres."* The keywords are how recruiters find you at all.
- **About** — three short paragraphs: who you are, the systems you build, what
  you're looking for. Write like a person, not a press release.
- **Featured / Projects** — pin the Raft repo, the KV store, and the blog post.
  The blog post is your differentiator: it proves you can *explain*, which is half
  of every senior engineer's job.
- **Experience** — reuse the strongest CV bullets. Keep the numbers.
- **Open to work** — turn it on, set it to backend/distributed/platform roles, and
  set location/remote honestly.

Then *use* it: connect with engineers at companies you admire, follow the people
who write the systems you study, and post the occasional short note about what you
shipped. A profile that's updated and active outranks a dormant one.

## Where & how to apply

Target roles that match the proof you built: **backend engineer, distributed
systems engineer, platform / infrastructure engineer, site reliability
engineer.** These are exactly the jobs your projects argue for. Don't scatter
applications across every "software engineer" listing — aim where your evidence
lands hardest.

Before you apply anywhere, **research compensation** so you negotiate from
knowledge, not hope. [levels.fyi](https://www.levels.fyi/) has real numbers by
company, level, and location — read it before any salary conversation.

**Tailor every application.** This does not mean rewriting your CV ten times. It
means:

- Read the job description and mirror its language. If it says "gRPC" and you used
  gRPC, make sure the word *gRPC* is on your CV.
- Reorder your project bullets so the most relevant one is first.
- Write three honest sentences in the cover note connecting *your* Raft/KV work to
  *their* problem. Generic cover letters are worse than none.

Apply in batches, track them in a simple spreadsheet (company, role, date,
contact, status), and keep a steady cadence rather than one frantic afternoon.
Quality of fit beats raw volume — ten tailored applications outperform a hundred
sprayed ones.

## Outreach that works

The hidden job market runs on warm intros and direct messages. A short, specific,
respectful note to an actual engineer beats an application into the void. This is
**signal, not spam** — and the difference is entirely in the specifics.

A good cold message is three sentences:

```
Hi Ana — I saw you work on the storage layer at Acme. I just built a
Raft-backed key-value store in Go and wrote up the log-replication part
here [link] — would love your read if you have a minute. Are you all
hiring backend engineers this quarter?
```

Why it works:

- **Specific to them** ("the storage layer," "you work on"). It's obviously not a
  template blasted to 200 people.
- **Offers something** — your work, your writing — before asking for anything.
- **One clear, easy ask.** Respect their time; make the reply effortless.

What to avoid: walls of text, attaching your CV unasked, flattery with no
substance, and following up five times. One polite follow-up after a week is fine;
after that, move on. Reach out to engineers and team leads, not just recruiters —
they often have more influence over who gets an interview, and they remember
people who engaged with their actual work.

## Interview prep

Interviews reward rehearsal. You already did the learning; now build the cadence
to perform it under pressure.

- **Data structures & algorithms** — the muscle from **Step 2**. Do a few problems
  a week, out loud, talking through your reasoning the way you would in a room.
  Consistency beats cramming; a steady habit for a few weeks carries you through
  most screens.
- **System design** — the thinking from **Step 9**. Practise sketching a system
  end to end: requirements, API, data model, scaling, failure modes. Work through
  the [System Design Primer](https://github.com/donnemartin/system-design-primer)
  — it's free, comprehensive, and exactly the vocabulary interviewers use.
- **Your projects** — this is your home-field advantage. Be ready to walk through
  the Raft implementation, the trade-offs you made, the bug that took two days,
  what you'd do differently. Interviewers love a candidate who can go *deep* on
  something real. Rehearse the story until it's smooth.
- **Behavioural** — have three or four concrete stories ready (a hard problem, a
  disagreement, a failure you owned). Keep them tied to real work.

Treat each interview as data. Note the questions, fix the gaps, and the third
interview will feel nothing like the first.

## Done when

- Your **CV** is one page, projects-first, quantified, and exported to PDF.
- Your **LinkedIn** has the keyword headline, a real About, pinned projects and
  blog post, and *Open to work* switched on.
- A first **batch of applications** is out — tailored, tracked, targeting
  backend/distributed/platform roles.
- A handful of **warm outreach messages** have gone to real engineers — specific,
  respectful, signal not spam.
- You have a weekly **prep cadence** running: DSA reps, one system-design sketch,
  and a rehearsed walkthrough of your best project.

## You did it — now keep going

Stop and look back at the distance. You started at zero. You learned Go, then
concurrency, then how to build and test real services, then how machines agree
when the network lies — and you *built the thing*: consensus, replication, APIs,
a deploy, writing that teaches. That is not a tutorial-follower's résumé. That is
an engineer's body of work.

The offer may not come on the first try. Job hunting is noisy and a little unfair,
and rejection says far more about a hiring funnel than about you. Keep the cadence:
apply, reach out, prep, ship one more small thing, repeat. The proof you built
doesn't expire, and every week you stay visible, the odds tilt your way.

You set out to go from zero to distributed engineer in six to eight months. You
did the work. Now go let the right people see it — and welcome to the field.

When your CV and LinkedIn are live and the first applications are out, mark this
final task complete on your **Roadmap**. You've finished the journey. Go get the
job.

## Common interview gotchas

- **No full-time experience is not a dealbreaker — but a thin "Projects" section is.** With limited work history, lead with **Projects** (your Raft, KV store, APIs) and a focused **Skills** section. A junior with deep, inspectable projects beats one with a vague CV. Never fabricate roles; let the demonstrable work carry it.
- **Quantify impact, don't list tasks.** "Built a Raft-backed KV store passing the MIT 6.824 test suite" beats "worked with Go and distributed systems." Numbers and verifiable outcomes (tests passing, p99 latency, throughput) read as real.
- **Warm outreach beats the application black hole.** A specific, short message to an engineer or hiring manager on a team is several times more likely to land an interview than a blind portal submission. Spend more effort on referrals/intros, less on mass applications.
- **Research the level and comp before you apply or negotiate.** Titles are inconsistent across companies; check levels.fyi for the level↔comp mapping and the team's actual scope. Walking in blind to comp is how strong candidates leave money or get mis-leveled.
- **Allocate prep to where you're weakest, not where you're comfortable.** Roadmap grads have DSA/system-design rehearsed; the usual gap is **behavioral** (STAR stories) and **project deep-dives** ("walk me through why you chose Raft, what failed, what you'd change"). Budget real time there — interviewers *will* probe your own projects and writing.
