---
slug: ds-failure-detection
step: 7
title: Failure detection & health checking
summary: How nodes decide a peer is dead when they can't tell slow from dead — heartbeats, timeouts, phi-accrual, and SWIM gossip.
est_min: 240
position: 11
---

# Failure detection & health checking

> **Step 7 · Distributed systems theory · Interview prep**
> Concept: *heartbeats, timeouts, the slow-vs-dead problem*

Consensus (Raft) and failover (replication) both depend on a deceptively hard
question: **is that node dead, or just slow?** You can't know for sure — so failure
detection is about making a *good enough* guess and bounding the damage of being
wrong.

## Why this matters

Set timeouts too aggressively and you trigger needless failovers and flapping; too
slowly and the system is unavailable longer. It's the backbone of leader election,
load balancers, service meshes, and cluster membership. Interviewers probe it as a
follow-up to "how does your system handle a node dying?".

## 1. The fundamental limit

In an asynchronous network you **cannot** reliably distinguish a crashed node from a
slow one or a network partition (this is why consensus is hard — the FLP result).
Every failure detector trades off two errors:

- **False positive**: declaring a live-but-slow node dead → needless failover, churn.
- **False negative / slow detection**: taking too long to notice a real death →
  prolonged unavailability.

You tune where you sit on that curve; you can't eliminate both.

## 2. Heartbeats & timeouts

The basic mechanism: each node periodically sends a **heartbeat**; if a peer misses
heartbeats for a timeout window, it's suspected dead. (Raft's election timeout is
exactly this.) Push (heartbeat) vs pull (health-check probe) are duals.

Choosing the timeout is the whole art: short = fast detection but more false
positives under transient slowness; long = stable but slow to react. **Randomize**
timeouts (Raft does) so nodes don't all decide simultaneously.

## 3. Phi-accrual detection (adaptive)

Instead of a hard yes/no, the **phi-accrual** detector outputs a *suspicion level*
(φ) based on the statistical distribution of recent heartbeat arrival times. The app
picks a threshold for how much risk to accept. It **adapts** to changing network
conditions (a temporarily slow link raises the bar before declaring death). Used by
Akka and Cassandra.

## 4. SWIM gossip (scalable membership)

In a large cluster, all-to-all heartbeats are O(N²). **SWIM** scales it: each node
periodically pings a *random* peer; if no ack, it asks a few other nodes to ping it
**indirectly** (ruling out a one-off network blip) before suspecting it, then
**gossips** membership changes around. Detection load is roughly constant per node.
HashiCorp's Serf/Consul use SWIM.

## 5. Fencing the false positive

Because false positives are inevitable, you must make a wrongly-declared-dead node
*safe*: when it comes back (or was only slow), it must not act as if still in charge.
That's the **fencing token** idea from lesson 7 — the resource rejects a stale
leader's writes. Failure detection decides *who's probably alive*; fencing makes
being wrong non-catastrophic.

## Do it yourself (≈ 4 hrs)

1. Read **DDIA Ch.8** on timeouts/unreliable networks, and the [**SWIM paper**](https://www.cs.cornell.edu/projects/Quicksilver/public_pdfs/SWIM.pdf) abstract/intro.
2. Skim the [**phi-accrual failure detector**](https://doc.akka.io/docs/akka/current/typed/failure-detector.html) docs.
3. Reason about your Step-8 Raft: what election timeout did you use, and what false-positive rate does it imply under a 100ms GC pause?

## Check yourself

- Why can't you reliably tell a slow node from a dead one?
- What two errors does every failure detector trade off?
- How does phi-accrual differ from a fixed timeout, and why is that better?
- What problem does SWIM's indirect ping + gossip solve at scale?
- How do fencing tokens make an inevitable false positive safe?

## Common interview gotchas

- **You can't distinguish slow from dead — period.** Any claim of "perfect" failure detection is wrong; the honest framing is the false-positive vs detection-latency trade-off. (This is the practical face of the FLP impossibility result.)
- **Aggressive timeouts cause failover storms.** A brief GC pause or network blip wrongly evicts a healthy leader, triggering an election, more load, more timeouts — a cascading flap. Randomize and pad timeouts; prefer indirect probes before declaring death.
- **All-to-all heartbeats don't scale (O(N²)).** For large clusters use gossip/SWIM with random + indirect probing, not every node pinging every other node.
- **Detecting a death isn't enough — you must fence the zombie.** A node declared dead may still be alive and slow; without fencing tokens it can resume writing and corrupt state when it wakes.
