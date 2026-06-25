---
slug: sd-news-feed
step: 9
title: "Design: a news feed (fan-out)"
summary: The push-vs-pull fan-out trade-off, the celebrity problem, and the hybrid design behind Twitter/Facebook timelines.
est_min: 300
position: 9
---

# Design: a news feed (fan-out)

> **Step 9 · System design · Interview prep**
> Concept: *fan-out on write vs read, the celebrity problem*

The home-timeline ("show me recent posts from everyone I follow") is one of the most
asked system-design questions, because its central trade-off — **fan-out on write vs
on read** — is a clean, deep design decision that scales in interesting ways.

## Why this matters

It's a canonical Meta/Twitter/LinkedIn design prompt, and the push/pull reasoning
generalizes to notifications, activity feeds, and any "merge many sources" problem.

## 1. Requirements & scale

Functional: post; view a home feed of followees' recent posts, newest-first; follow/
unfollow. Non-functional: feed load must be **fast** (read-heavy — people scroll far
more than they post), eventual consistency is fine (a post appearing a few seconds
late is OK). Estimate: hundreds of millions of users, read:write maybe 100:1.

## 2. The core decision: fan-out on write vs read

**Fan-out on write (push):** when you post, immediately write the post id into the
**precomputed feed** (a per-user list, e.g. in Redis) of every follower.
- ✅ Reads are O(1) — just read your precomputed feed. Great for the common case.
- ❌ A post by someone with 50M followers = 50M writes (a "write storm"); wasteful
  for inactive followers.

**Fan-out on read (pull):** compute the feed at read time by querying recent posts
from everyone you follow and merging.
- ✅ No write amplification; always fresh.
- ❌ Reads are expensive (merge across many followees) — bad for the hot read path.

## 3. The hybrid (the expected answer)

Use **push for most users** and **pull for celebrities**: don't fan-out a
mega-account's post to all followers; instead, at read time, merge a user's
precomputed (pushed) feed with a *pull* of the handful of celebrities they follow.
This caps the write storm while keeping normal reads cheap. State this trade-off
explicitly — it's the point of the question.

## 4. High-level design

- **Post service** → write to the post store; enqueue a fan-out job.
- **Fan-out workers** → push post ids into followers' feed lists (skip celebrities).
- **Feed service** → on read, return the cached feed list (+ merge celebrity pulls),
  hydrate post ids into full posts from a cache, paginate by **cursor** (not offset).
- **Caches/CDN** for hot posts and media; **read replicas** for the post store.

## 5. Deep dives interviewers push on

- **Ranking**: chronological is simplest; ML-ranked feeds add a scoring service and
  change caching (feeds become per-user and time-sensitive).
- **Pagination**: cursor/keyset (Step 4) so infinite scroll is stable.
- **Inactive users**: don't precompute feeds for users who never log in (lazy).
- **Consistency**: feeds are eventually consistent; a brief delay is acceptable.

## Do it yourself (≈ 4 hrs)

1. Read [**System Design Primer**](https://github.com/donnemartin/system-design-primer) and watch [**ByteByteGo — News Feed**](https://bytebytego.com/courses/system-design-interview/design-the-news-feed-system).
2. Sketch the hybrid design end-to-end and compute the write amplification for a 50M-follower post under pure push.

## Check yourself

- Contrast fan-out on write vs read — what does each optimize and cost?
- What is the celebrity problem, and how does the hybrid fix it?
- Why is a feed read-heavy, and how does that shape the design?
- Why cursor pagination over offset for infinite scroll?
- Where does ML ranking change the caching story?

## Common interview gotchas

- **Pure fan-out-on-write dies on celebrities** — a 50M-follower post is 50M writes; you must special-case high-fan-out accounts with pull, or the write path melts.
- **Pure fan-out-on-read dies on the hot path** — merging hundreds of followees per feed load is too slow at scale; precompute for the common case.
- **Feeds are eventually consistent and that's fine** — don't over-engineer strong consistency for a timeline; a post showing up 2s late is acceptable, which buys you huge scalability.
- **Offset pagination breaks infinite scroll** — inserts shift offsets, duplicating/skipping posts; use a cursor over (timestamp, id).
