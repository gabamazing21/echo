---
slug: sd-chat-app
step: 9
title: "Design: a chat app (WhatsApp-style)"
summary: Real-time messaging at scale — connections, fan-out, message ordering, delivery and presence.
est_min: 240
position: 3
---

# Design: a chat app (WhatsApp-style)

> **Step 9 · System design · Week 1 · Design**
> Concept: *fan-out, presence, message ordering*

A chat app *feels* trivial — type a message, it shows up over there. But "over
there" might be a phone that's offline, in a group of 500 people, behind a flaky
mobile connection, while the recipient watches your "typing…" indicator. Chat is
where **real-time delivery**, **ordering**, and **state at scale** all collide.
We'll walk it through the framework from lesson 1: pin the requirements, sketch a
high-level design, then deep-dive the three hard parts.

## Why this matters

Almost every product grows a messaging surface — DMs, notifications, live
collaboration. The core moves here (long-lived connections, a user→server
registry, per-conversation ordering, fan-out) reappear in feeds, multiplayer,
and presence systems everywhere. And it's a *perennial* interview question.

## 1. Requirements

Always start by bounding the problem out loud. Functional:

- **1:1 messaging** and **group messaging** (say up to a few hundred members).
- **Delivery & read receipts** — sent ✓, delivered ✓✓, read ✓✓ (blue).
- **Presence** — online / last-seen / "typing…".
- **Message history** — messages persist; a device that was offline syncs missed
  messages on reconnect.
- **Ordering** — within one conversation, everyone sees messages in the *same*
  order.

Non-functional (the ones that shape the design):

- **Low latency** — sub-second delivery when both parties are online.
- **High availability** — being unreachable is the cardinal sin of a chat app.
- **Durability** — an accepted message is never silently lost.
- **Scale** — assume ~100M daily users, so tens of millions of *concurrent*
  long-lived connections. That number is the whole reason this is hard.

Back-of-envelope: 100M users × ~40 messages/day ≈ 4B messages/day ≈ **~50k
writes/sec** average, with peaks several times higher. Each connection also costs
memory on whatever server holds it — that's the constraint that drives the
gateway tier below.

## 2. API & data model

The client talks over a **persistent connection** (see §3), but the operations
are simple:

- `send(conversation_id, client_msg_id, body)` → server assigns a sequence number.
- `ack(conversation_id, up_to_seq)` → delivery/read receipts.
- `subscribe(conversation_ids)` → receive pushed messages + presence.

`client_msg_id` is a client-generated UUID for **idempotency** — if the network
drops and the client retries, the server dedupes on it so the message isn't
stored twice.

Data model (conceptually):

```
users(user_id, name, last_seen, ...)
conversations(conv_id, type[1:1|group], created_at)
members(conv_id, user_id, joined_at, last_read_seq)
messages(conv_id, seq, msg_id, sender_id, body, created_at)
                  └── PRIMARY KEY (conv_id, seq)
```

The crucial choice: **partition messages by `conv_id`** (the partition-key idea
from Step 7). All messages for one conversation live together and are ordered by
`seq`. A "get my history" read is then a single-partition range scan
(`WHERE conv_id = ? AND seq > last_read`). A wide-column store (Cassandra,
DynamoDB, HBase) fits this access pattern well at chat scale.

## 3. High-level design

```
                       ┌──────────────┐
  phone ──WebSocket──▶ │              │   which server holds user U?
  phone ──WebSocket──▶ │  GATEWAY     │◀────────────┐
  phone ──WebSocket──▶ │  servers     │      ┌───────────────┐
                       │ (hold sockets│─────▶│ session/route │  registry:
                       └──────┬───────┘      │   registry    │  user → gateway
                              │              └───────────────┘
                              ▼
                       ┌──────────────┐      ┌───────────────┐
                       │ chat service │─────▶│  message store│ (by conv_id)
                       │  (logic)     │      └───────────────┘
                       └──────┬───────┘      ┌───────────────┐
                              └─────────────▶│ offline queue │ → push (APNs/FCM)
                                             └───────────────┘
```

The pieces:

- **Transport** — clients keep a **long-lived WebSocket** open (an HTTP upgrade to
  a full-duplex TCP connection). Unlike request/response HTTP, the server can
  **push** down it the instant a message arrives — no polling. (Mobile fallback:
  if the socket can't stay up, fall back to a push notification, §6.)
- **Gateway tier** — a fleet of servers whose only job is to *hold* those millions
  of sockets and shuttle bytes. Stateless except for the connections themselves,
  so you scale it horizontally by adding boxes.
- **Chat service** — stateless business logic: assign sequence numbers, persist,
  fan out, write receipts.
- **Message store** — durable, partitioned by `conv_id` (§2).
- **Session/route registry** — the map of *which gateway currently holds user U's
  socket*. This is the heart of the routing problem (§5).

## 4. Deep dive — message ordering

"Everyone sees the same order" sounds easy until you remember messages from many
senders hit different servers at slightly different times, and clocks disagree
(the logical-ordering problem from Step 7). The fix is to make **one component the
sequencer per conversation**:

- When a message is accepted, the chat service assigns it a **monotonic sequence
  number scoped to that conversation** — `seq = 1, 2, 3, …` for `conv_id`.
- All members order by `(conv_id, seq)`. Since one logical writer hands out `seq`
  for a given conversation, there's a single source of truth for order — no
  reliance on wall-clock time across machines.
- The **`client_msg_id`** lets the sender reconcile its optimistic local copy
  with the server-assigned `seq` (and dedupe retries).
- A device that reconnects sends its highest seen `seq` and pulls everything
  after it — guaranteeing a **consistent prefix** (no gaps, no holes), the same
  guarantee we named under replication in Step 7.

Note we only need a *total order per conversation*, not globally. That's what
makes the per-`conv_id` partition both a storage and an ordering win.

## 5. Deep dive — fan-out & the routing problem

When a message lands, who needs a copy *pushed* to them right now?

**1:1** — trivial: look up the recipient, push to their socket. **Group** — the
chat service must deliver to every online member. Two strategies:

- **Fan-out on write** — at send time, push to all currently-online members and
  persist for the offline ones. Simple, fast reads. Fine for small/medium groups;
  a 100k-member broadcast would be a thundering herd (handle those specially).
- **Fan-out on read** — store once; each member's device pulls on open. Cheaper
  writes, but needs clients to actively sync. Real systems blend both.

Either way you hit **the routing problem**: to push to member U you must know
*which gateway server holds U's socket*. With millions of sockets spread over
thousands of gateways, this is exactly the **partition-routing problem from Step
7** — "which node holds this key?" — except the key is a live user and the
mapping changes every time someone reconnects.

The answer is the **session registry**: when U's socket connects to gateway G,
write `U → G` (with a TTL/heartbeat) to a fast shared store (Redis-style) or a
pub/sub layer. To deliver, the chat service looks up U's gateway and forwards;
the gateway writes to the socket. A common simplification is to put gateways
behind a **pub/sub bus** keyed by user/conversation, so the chat service just
publishes and the gateway holding U is subscribed — no explicit lookup.

## 6. Deep dive — presence & offline delivery

**Presence** ("online", "last seen", "typing…") is high-churn, low-value state —
do *not* durably persist every flip. Drive it from the connection lifecycle:

- Socket connects → mark online (a key in a fast store with a **heartbeat TTL**).
- Heartbeats stop / socket drops → TTL expires → offline, stamp `last_seen`.
- "Typing…" is a transient event fanned out to the conversation and never stored.

Presence fan-out can be expensive (every contact wants your status), so batch and
rate-limit it — nobody needs millisecond-accurate "last seen."

**Offline delivery** — the recipient has no live socket:

1. Persist the message (it's durable regardless of delivery).
2. Enqueue it in a **per-user offline queue / inbox**.
3. Fire a **push notification** (APNs for iOS, FCM for Android) to wake the app.
4. On reconnect, the device drains its queue / pulls everything after its last
   `seq`, and the server emits the delivery receipt.

This is why "✓ sent" and "✓✓ delivered" are *different* states: sent = persisted;
delivered = the recipient's device actually received it.

## 7. Scaling & trade-offs

- **Connections are the bottleneck, not CPU.** Each idle socket costs memory and a
  file descriptor. Scale gateways horizontally; keep them dumb so any can be
  replaced. Millions of sockets / tens of thousands per box → thousands of boxes.
- **Sticky-ish but recoverable.** A socket is pinned to one gateway, but if that
  gateway dies the client reconnects (to any gateway) and re-registers — the
  registry heals. Avoid hard stickiness that makes failover painful.
- **Durability vs latency.** Persist-then-ack is safe but adds a write to the hot
  path; some systems ack on accept and persist async (faster, small loss window) —
  the same sync/async trade-off from Step 7's replication.
- **Hot conversations.** A huge group (or a celebrity broadcast) is a hot
  partition — the §2/Step 7 hot-spot problem. Treat broadcasts as a separate
  fan-out-on-read path rather than pushing to millions synchronously.
- **Exactly-once is a myth; aim for at-least-once + idempotency.** Networks retry;
  `client_msg_id` and per-conv `seq` make duplicates harmless.

## Do it yourself (≈ 4 hrs)

1. Sketch the whole diagram from memory — gateways, registry, chat service,
   store, offline queue — and label where each requirement is satisfied.
2. Trace one group message end-to-end for a member who is **offline**, then
   **reconnecting**. Name every state: sent → persisted → queued → push →
   delivered → read.
3. Read the chat / "designing a chat system" walkthrough in the
   [**System Design Primer**](https://github.com/donnemartin/system-design-primer).
4. Watch a chat-system design video from
   [**ByteByteGo**](https://www.youtube.com/@ByteByteGo) and compare its choices
   to yours — especially how it handles the registry and fan-out.

## Done when

You can, on a whiteboard and without notes:

- State the functional + non-functional requirements and the rough write rate.
- Explain why connections (not compute) drive the gateway tier's sizing.
- Describe how per-`conv_id` sequence numbers give a consistent total order, and
  why that's the logical-ordering idea from Step 7.
- Solve the routing problem with a session registry, and tie it to partition
  routing from Step 7.
- Contrast fan-out on write vs on read, and explain offline delivery via queue +
  push.
- Name one durability/latency trade-off you'd make and defend it.

Next: **the design write-up** — turning this whiteboard into a crisp document.

## Common interview gotchas

- **Reaching for a global message order.** You only need a total order *per conversation*, not across the whole system. A per-`conv_id` monotonic `seq` from one logical sequencer gives that cheaply; insisting on global ordering invents a needless bottleneck and ignores that clocks across machines disagree.
- **Forgetting the session registry is a SPOF.** The user→gateway map is what makes routing work — and if it's a single unreplicated store, the whole push path dies with it. Replicate it, drive entries from heartbeats with TTLs, and let clients re-register on reconnect so it self-heals.
- **Promising exactly-once delivery.** It's a myth over real networks. Aim for at-least-once + client-side dedup on `client_msg_id` (plus per-conv `seq`), which makes inevitable retries harmless. Saying "exactly-once" unprompted signals inexperience.
- **Durably persisting presence.** "Online / typing…" is high-churn, low-value, ephemeral state — derive it from the connection lifecycle (heartbeat TTL), batch and rate-limit the fan-out. Writing every flip to a durable store is a self-inflicted write storm; nobody needs millisecond-accurate "last seen."
- **One fan-out strategy for all group sizes.** Fan-out-on-write is great for small/medium groups but a thundering herd for a 100k-member or celebrity broadcast. Treat huge groups as a separate fan-out-on-read path rather than pushing synchronously to millions.
