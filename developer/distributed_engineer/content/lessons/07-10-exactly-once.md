---
slug: ds-exactly-once
step: 7
title: Exactly-once & idempotent delivery
summary: Why "exactly-once delivery" is a myth, and how at-least-once + idempotency gives you effectively-once processing.
est_min: 240
position: 10
---

# Exactly-once & idempotent delivery

> **Step 7 · Distributed systems theory · Interview prep**
> Concept: *at-least-once + idempotency = effectively-once*

"Does your queue guarantee exactly-once delivery?" is a trap question. The honest
answer — and the one interviewers want — is that **exactly-once *delivery* is
impossible** over an unreliable network, but **exactly-once *processing* (effect)**
is achievable with idempotency. Understanding the difference marks a senior engineer.

## Why this matters

Every message queue, payment system, and event pipeline faces this. Get it wrong and
you double-charge customers or drop orders. It ties together idempotency (Step 4),
dedup (Step 8 KV store), and transactions (Step 6).

## 1. Why exactly-once delivery is impossible

A sender transmits a message and waits for an ack. If the ack is lost (network),
the sender can't tell whether the receiver got it. Two choices:

- **Don't resend** → risk losing the message (**at-most-once**).
- **Resend** → risk delivering twice (**at-least-once**).

There's no third option at the delivery layer — the **two-generals problem**. So
real systems pick **at-least-once** (never lose data) and make duplicates harmless.

## 2. The three delivery semantics

- **At-most-once**: fire and forget. Fast, lossy. OK for metrics/telemetry.
- **At-least-once**: retry until acked. No loss, possible duplicates. The default for
  anything that matters.
- **Exactly-once (effect)**: at-least-once delivery + **idempotent processing**, so
  duplicates produce the same result as one. This is what "exactly-once" really means
  in practice.

## 3. Idempotency keys — the core technique

The producer attaches a stable, unique **idempotency key** to each operation. The
consumer records processed keys; a duplicate key is recognized and skipped (or
returns the prior result).

```
1. client generates key = uuid (stable across retries of the SAME op)
2. server: INSERT INTO processed (key) ... ON CONFLICT DO NOTHING
3. if it was a new row -> do the side effect; else -> it's a retry, skip/return cached
```

Stripe's API works exactly this way (`Idempotency-Key` header). The key must be the
*same* across retries of one logical operation and *different* across distinct ones.

## 4. The atomicity catch

The side effect and the dedup record must commit **together**, or you reintroduce
the bug: do the work, crash before recording the key, retry → double effect. Fixes:

- Put the side effect and the dedup insert in **one transaction** (Step 6) when
  they're in the same database.
- The **transactional outbox** pattern when the effect is "publish an event": write
  the event to an outbox table in the same transaction as the state change, then a
  relay publishes it (at-least-once) — consumers dedup by event id.

## 5. Kafka's "exactly-once"

Kafka offers "exactly-once semantics" via **idempotent producers** (dedup by
producer id + sequence number) and **transactions** (atomic write across partitions
+ offset commit). Crucially, it's exactly-once *within Kafka's processing*, not a
magic delivery guarantee to arbitrary external systems — those still need their own
idempotency.

## Do it yourself (≈ 4 hrs)

1. Read **DDIA Ch.11** (stream processing, exactly-once) and Stripe's [**idempotency docs**](https://stripe.com/docs/api/idempotent_requests).
2. Read about the [**transactional outbox**](https://microservices.io/patterns/data/transactional-outbox.html) pattern.
3. Sketch an order-placement endpoint that's safe to retry: where does the idempotency key live, and how do the effect + dedup commit atomically?

## Check yourself

- Why is exactly-once *delivery* impossible? Which semantic do real systems choose?
- What is an idempotency key, and what makes a good one?
- Why must the side effect and the dedup record commit atomically — what breaks otherwise?
- What problem does the transactional outbox pattern solve?
- What does Kafka's "exactly-once" actually cover, and what does it not?

## Common interview gotchas

- **"Our broker guarantees exactly-once delivery" is a red flag** — delivery can't be exactly-once over a lossy network; what's achievable is at-least-once delivery + idempotent processing. Say "effectively-once".
- **Idempotency without atomicity is still buggy** — if you perform the effect and then crash before recording the key, the retry repeats the effect. The dedup write and the effect must be in one transaction (or use an outbox).
- **A bad idempotency key defeats the whole thing** — generating a new key per retry (instead of per logical operation) makes every retry look new; reusing one key across different operations drops legitimate ones.
- **Dedup tables grow forever** — you need a retention/TTL policy for processed keys, and clients must tolerate a key expiring (rare, but design for it).
