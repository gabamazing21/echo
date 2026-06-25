---
slug: sd-framework
step: 9
title: The system design interview framework
summary: A repeatable structure for any system-design question — requirements, estimation, API, data model, high-level design, then scale.
est_min: 300
position: 1
---

# The system design interview framework

> **Step 9 · System design · Week 1**
> Concept: *how to structure the answer*

You've spent Steps 6 and 7 building a toolbox: indexing, replication, partitioning,
caching, CAP. A system design interview is where you *assemble* those tools into a
working system on a whiteboard — under time pressure, with an interviewer watching.
The single biggest failure mode isn't not knowing the pieces. It's having no
**structure** to fit them into, so the answer wanders and never lands. This lesson
gives you that structure: one repeatable framework you run top to bottom on *any*
question — "design a URL shortener," "design Twitter," "design a rate limiter."

## Why this matters

There is **no single right answer** to a system design question. The interview is
not a quiz with a hidden solution; it's a simulation of how you'd lead a real design
discussion. They're watching whether you can take a vague prompt, turn it into
concrete requirements, reason quantitatively about scale, and *defend your
trade-offs*. Candidates who memorize one architecture and recite it fail. Candidates
who **drive the conversation** with a calm, repeatable process pass — even when they
don't know everything.

So: **you** own the structure. State your assumptions out loud, sketch boxes, and
narrate your reasoning. The framework below is the script you run so your brain is
free to think about the actual problem.

## 1. Clarify requirements & constraints

Never start drawing. Spend the first several minutes turning a one-line prompt into a
written list. Separate two kinds:

- **Functional requirements** — what the system *does*. "Users shorten a long URL and
  get a short one. Visiting the short URL redirects." Pick the **core** features and
  explicitly defer the rest ("analytics — out of scope for now, happy to come back").
- **Non-functional requirements** — the *qualities*: scale (how many users / requests),
  latency targets, availability, consistency needs, read/write ratio, durability.

Then nail down **constraints**: how many daily active users? Read-heavy or
write-heavy? Is stale data acceptable (this is your CAP lean from Step 7)?

> **Scoping is the whole game here.** A system designed for 1,000 users and one for
> 100 million look nothing alike. Ask, write the number down, and design to *that*
> number — don't guess silently.

## 2. Back-of-the-envelope estimation

Now put numbers on it. This takes five minutes and earns enormous credit, because it
proves your design is grounded in reality rather than vibes. Estimate, in order:

1. **QPS (queries per second)** — derive from daily volume. A handy rule:
   `1 million requests/day ≈ 12 requests/second`. Compute the **average** QPS, then a
   **peak** (2–10× average).
2. **Read/write ratio** — state it explicitly. A URL shortener might be **100:1**
   reads to writes. This single number drives most later decisions (caching, read
   replicas).
3. **Storage** — `writes/day × bytes/record × retention`. "500M new URLs/year × 500
   bytes ≈ 250 GB/year."
4. **Bandwidth** — `QPS × payload size`, for both ingress and egress.

```text
Example — URL shortener
  100M new URLs / day
  write QPS  = 100M / 86,400        ≈ 1,160 writes/sec
  read QPS   = 1,160 × 100 (ratio)  ≈ 116,000 reads/sec   ← read-heavy!
  storage    = 100M × 500 B × 365   ≈ 18 TB / year
```

That "read-heavy" conclusion isn't trivia — it's the justification for the cache and
read replicas you'll add in step 6. **Estimation feeds design.**

## 3. Define the API

Before any boxes, agree on the **contract** the system exposes. This forces clarity
about inputs, outputs, and who calls whom. A few endpoints is enough:

```text
POST /urls
  body:  { "longUrl": "https://...", "customAlias": "optional" }
  →      { "shortUrl": "https://sho.rt/aB3xZ" }

GET /{shortCode}
  →      302 redirect to the original long URL
```

State the **method, path, key parameters, and response**. Mention auth and rate
limiting at the boundary if relevant. Keep it small — you're defining the shape, not
writing OpenAPI.

## 4. Data model & schema

What do you store, and how is it keyed? This is where Step 6 (indexing) pays off.

```text
url_mapping
  short_code   VARCHAR  PRIMARY KEY   -- the lookup key; index here
  long_url     TEXT
  user_id      BIGINT
  created_at   TIMESTAMP
  expires_at   TIMESTAMP NULL
```

Decide **SQL vs NoSQL** *and justify it* against your access pattern: a key-value
lookup by `short_code` is a natural fit for a KV store; relational joins and
transactions argue for SQL. Call out which columns get **indexes** (Step 6) and
which field you'll **partition / shard** on later (Step 7) — for a shortener, the
`short_code` is the obvious shard key.

## 5. High-level design

*Now* you draw. Sketch the major components as boxes and connect them with the
request flow. A typical starting picture:

```text
        ┌─────────┐
        │ Clients │
        └────┬────┘
             │
      ┌──────▼───────┐
      │ Load balancer│
      └──────┬───────┘
             │
   ┌─────────▼─────────┐        ┌────────┐
   │   App servers     │◄──────►│ Cache  │
   │ (stateless, x N)  │        └────────┘
   └───┬───────────┬───┘
       │           │
 ┌─────▼────┐  ┌───▼─────┐
 │ Database │  │  Queue  │──► async workers
 └──────────┘  └─────────┘
```

The standard boxes, and why each exists:

- **Load balancer** — spreads traffic across app servers; enables horizontal scale.
- **App servers** — keep them **stateless** so any server handles any request.
- **Database** — the source of truth.
- **Cache** — absorbs the read-heavy load you measured in step 2.
- **Queue** — decouples slow/async work (analytics, emails) from the request path.

Walk the interviewer through one full request: client → LB → app server → cache
miss → DB → response. Narrate it.

## 6. Deep-dive & scale

The interviewer will now push: "It's working — now you have 100× the traffic." This
is where you spend your Step 6–7 toolbox. Reach for these in roughly this order:

- **Caching** — put hot reads in Redis/Memcached in front of the DB. For a 100:1
  read ratio, a cache is the highest-leverage move you can make. Discuss eviction
  (LRU) and TTLs.
- **CDN** — push static content and even cacheable redirects to edges near users,
  cutting latency and origin load.
- **Read replicas** *(Step 7)* — fan reads out to replica copies; accept replication
  lag, which means eventual consistency on reads.
- **Sharding / partitioning** *(Step 7)* — when one DB can't hold the data or the
  write load, split it by a shard key (e.g. hash of `short_code`). Mention the
  re-sharding and hot-key pain.
- **Async via queues** — move anything not needed for the response off the hot path:
  the write returns immediately, a worker does the slow part later.

You won't apply all of these — apply the ones your *estimates* justify, and say why.

## 7. Identify bottlenecks & trade-offs

Close by being your own critic. Strong candidates volunteer the weaknesses before the
interviewer finds them:

- Where's the **single point of failure**? (One DB primary → add replicas / failover.)
- What's the **CAP lean** *(Step 7)*? When a partition hits, does this system favor
  consistency or availability — and is that right for the use case?
- What's the **consistency** story? Cache and replicas mean clients may read stale
  data; is that acceptable here? (For a redirect — yes. For a bank balance — no.)
- What breaks **first** under 10× more load, and what's the next move?

There is no single right answer — there are **defensible trade-offs**. "I chose AP
here because a slightly stale redirect is fine, and availability matters more than a
brief inconsistency" is a *great* sentence. Saying it shows you understand the system,
not just the diagram.

## The framework at a glance

| # | Step | One-line goal |
|---|---|---|
| 1 | Requirements | Functional + non-functional + constraints, in writing |
| 2 | Estimation | QPS, storage, bandwidth, read/write ratio |
| 3 | API | The contract — methods, paths, responses |
| 4 | Data model | Schema, keys, SQL vs NoSQL, indexes |
| 5 | High-level design | Boxes: LB, app servers, DB, cache, queue |
| 6 | Deep-dive & scale | Cache, CDN, replicas, sharding, async |
| 7 | Bottlenecks & trade-offs | SPOFs, CAP lean, consistency, defend choices |

Run these seven steps in order, out loud, on every question. The structure is the
skill.

## Do it yourself (≈ 5 hrs)

1. Skim the **[System Design Primer](https://github.com/donnemartin/system-design-primer)** —
   the README's "study guide" and the "back-of-the-envelope" cheat sheet. Bookmark it;
   it's the single best free resource for this step.
2. Watch two or three walkthroughs on the **[ByteByteGo YouTube channel](https://www.youtube.com/@ByteByteGo)**.
   Notice the presenter follows the *same* structure every time — that's the framework
   in action.
3. Take **"design a URL shortener"** and write all seven steps yourself on paper. Do
   the estimation math. Don't peek until you've produced a full answer.
4. Now redo it for **"design a pastebin"** — and notice how much of your structure
   carries over even though the details change. That reuse *is* the point.

## Check yourself

You're ready to move on when you can, *without looking*:

- List the seven framework steps in order and say the goal of each.
- Convert "200M requests/day" into an average QPS in your head (≈ 2,300).
- Explain why you estimate the read/write ratio *before* choosing where to add caching.
- Name three scaling techniques from Steps 6–7 and the symptom each one treats.
- Defend a CAP lean for a system of your choice in one sentence.

When all five feel obvious, commit this task on your **Roadmap** and take the Step 9
quiz. Next lesson: **applying the framework end-to-end — designing a URL shortener.**

## Common interview gotchas

- **Drawing boxes before clarifying scope.** The most common reflex and the fastest way to fail. Turn the one-line prompt into written functional + non-functional requirements *and* a back-of-envelope estimate before a single box. A design for 1k users and one for 100M share almost nothing.
- **Burning half the clock on requirements.** The opposite failure: time-box it. ~5 min clarify, ~5 min estimate — don't spend 20 of 45 minutes gold-plating the problem statement. Running out of time before the deep dive is itself a scored failure mode.
- **Going silent while you think.** The interviewer can't grade an empty whiteboard. Narrate every decision and use them as a resource — thinking aloud lets them redirect you off a wrong path before you waste ten minutes on it.
- **Treating every technique as free.** Cache, replica, shard, and queue each buy something *and cost something* (staleness, lag, re-sharding pain, complexity). Name the cost as you add each one — that's the trade-off score.
- **No answer to "now scale it 10×."** They *will* push. Have a ready story for what breaks first and the next move, and be willing to re-evaluate an earlier choice rather than defend it to the death. Adapting under new constraints is the senior signal.
