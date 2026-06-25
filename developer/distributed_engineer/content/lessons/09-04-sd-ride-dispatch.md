---
slug: sd-ride-dispatch
step: 9
title: "Design: ride dispatch & proximity matching"
summary: Match riders to nearby drivers in real time — geospatial indexing, sharding by location, and live updates.
est_min: 300
position: 4
---

# Design: ride dispatch & proximity matching

> **Step 9 · System design · Week 2 · Design**
> Concept: *geospatial sharding, real-time matching*

This one is personal. You build **Lokatalent**, where the whole product is
*matching* — connecting a request to the best available provider, fast. A
ride-dispatch system is the same problem stripped to its purest form: thousands of
drivers streaming their location, a rider taps "request," and you must find the
nearest free driver in well under a second. Everything you design here — the
geospatial index, the location-update firehose, the dispatch flow — maps almost
directly onto matching and dispatch at Lokatalent. Treat it as a rehearsal for your
own system.

We'll run it through the **framework** from lesson 09-01: requirements first, then
the core technical challenge, a high-level design, sharding, deep dives, and
trade-offs.

## 1. Requirements

Pin down scope before drawing boxes. Interviewers (and your own architecture)
reward this.

**Functional:**

- Drivers continuously **stream their GPS location** (every few seconds) while online.
- A rider requests a ride at a pickup point; the system **matches them to the
  nearest available driver** and dispatches the request.
- A driver can **accept or decline**; on decline, offer the next-nearest.
- Both sides see **live location updates** during the trip.

**Non-functional:**

- **Low latency** on matching — a match decision in ~hundreds of milliseconds, not seconds.
- **High write throughput** — every online driver writes location every 3–5s. A
  million online drivers ≈ 200k–300k location writes/second.
- **Availability over strict consistency** — a slightly stale driver location is
  fine; refusing to dispatch is not (recall CAP from Step 8: we lean AP here).
- **Geographic scale** — the system spans cities and regions, but any single match
  is *local*. That locality is the key we'll exploit.

**Rough scale estimate:** 1M concurrent drivers, location every 4s → ~250k writes/s
of tiny payloads (driver id, lat, lng, timestamp). Rider requests are far rarer
(thousands/s). So this is a **write-heavy location pipeline** with a **read-heavy
proximity query** layered on top.

## 2. The proximity problem & geospatial indexing

The naive match is: "for this pickup point, scan every driver, compute distance,
take the minimum." With a million drivers that's a million distance calculations
*per request* — hopelessly slow. Worse, you can't even index it with a normal
B-tree, because "nearest" is a 2-D question and a B-tree on `lat` plus a B-tree on
`lng` doesn't compose into "near this point."

The core challenge is **"find nearby" efficiently**. The whole design hinges on a
**geospatial index** that lets you fetch only the handful of drivers in the pickup
neighborhood. Three common approaches:

- **Geohashing.** Encode (lat, lng) into a short string where a **shared prefix
  means physical proximity** — `9q8yy` and `9q8yz` are adjacent cells. To find
  nearby drivers, compute the rider's geohash, then query that cell plus its 8
  neighbors. Cells are fixed-size squares; precision is chosen by prefix length.
  Easy to store as a plain indexed string column.
- **Quadtrees.** Recursively subdivide space into four quadrants, splitting a cell
  only when it holds too many points. Dense downtown areas get fine-grained cells;
  empty highways stay coarse. Adapts to density, but the tree must be maintained as
  points move.
- **S2 cells (Google) / H3 (Uber).** Map the globe onto a space-filling curve and
  divide it into hierarchical cells with stable IDs. This is what real
  ride-hailing systems use — clean hierarchy, good neighbor math, fixed cell IDs
  you can shard on.

For a database-backed version, **PostGIS** (the spatial extension to Postgres you
already run) gives you a `geography`/`geometry` column with a **GiST spatial
index**, and `ST_DWithin(driver_loc, pickup, radius)` does the radius query using
that index — no full scan. That's the pragmatic starting point: you almost
certainly have Postgres already.

The shared idea across all four: **convert "near" into a cheap lookup** — a prefix
match, a tree descent, or an indexed radius query — so you touch tens of drivers
instead of a million.

## 3. High-level design

```
                 +------------------+
  driver app --> | Location Ingest  | --> Location Index (in-memory / Redis GEO)
   (GPS every    |    service       |         keyed by geo-cell
    few sec)     +------------------+              ^
                                                   | "drivers near pickup?"
  rider app ---> +------------------+              |
   (request) --> | Matching /       |--------------+
                 | Dispatch service |--> offers ride to nearest driver
                 +------------------+        |
                          |                  v
                          +--------> Trip store (Postgres) + notifications
```

The pieces:

- **Location Ingest service** — absorbs the firehose of location updates and writes
  them into the location index. Stateless and horizontally scaled.
- **Location Index** — the geospatial structure (Redis `GEOADD`/`GEOSEARCH`, or an
  in-memory cell map, or PostGIS) holding *current* driver positions. Optimized for
  fast proximity reads.
- **Matching/Dispatch service** — on a ride request, queries the index for nearby
  available drivers, ranks them (distance, ETA, rating), and offers the ride.
- **Trip store** — durable record of trips and state, in Postgres. The location
  index is ephemeral; the trip lifecycle is the source of truth.

Notice we **separate the hot, ephemeral location data from the durable trip data**.
Driver positions churn 250k times a second and you only ever need the *latest* — no
reason to durably persist every ping. Trips are rare, valuable, and must survive a
crash. Different stores for different access patterns.

## 4. Sharding by location

A single index can't hold a million constantly-updating drivers, and you don't want
a query for pickups in Lagos scanning drivers in Nairobi. So **shard the location
index by geography** — a direct application of partitioning (Step 7).

- **Partition key = geo-cell** (geohash prefix, S2 cell, or region id), *not* driver
  id. All drivers in a region live on the same shard, so a proximity query for that
  region hits **one shard** instead of scattering across all of them. This is the
  whole reason to shard by location rather than by driver: it preserves **locality**
  so reads stay local.
- This is essentially **range partitioning on a space-filling curve** — and from
  Step 7 you know range partitioning risks **hot spots**. Here the hot spot is
  literal: a downtown cell at rush hour, a stadium at closing time. Dense cells get
  far more traffic than rural ones.
- **Mitigations:** use finer cells in dense areas (quadtree-style adaptive
  granularity) so no single shard owns too much load; replicate hot shards for read
  scaling; size cells so the busiest is still serveable.
- **Cross-boundary queries:** a pickup near a cell edge has nearby drivers in the
  *adjacent* cell. So a proximity query fans out to the cell **and its neighbors**
  (the 8 surrounding geohash cells), then merges — a small, bounded scatter/gather,
  not a global one.

Someone must own the **cell → shard mapping** and keep it current as you rebalance,
exactly the routing problem from Step 7 — often backed by a consensus store like
etcd.

## 5. Deep dive: the location-update rate

The 250k writes/second of location pings is the part that quietly breaks naive
designs, so dig in.

- **Don't write every ping to Postgres.** Disk-backed durable writes can't sustain
  that rate cheaply, and you don't need the history. Keep current positions in an
  **in-memory store** — Redis with its native `GEO` commands, or your own in-memory
  cell map per shard. Overwrite-in-place: you only care about *now*.
- **Last-write-wins** is fine for position. Two pings from the same driver? Keep the
  newest by timestamp; no coordination needed. This is the kind of conflict where
  LWW is actually *correct*, not a compromise.
- **Tune the ping interval.** Faster pings = fresher matches but more load. 4–5s is a
  common sweet spot; you can even adapt it (ping more often when a match is imminent
  or the driver is moving fast, less when parked).
- **Coalesce and expire.** Drop pings that haven't moved meaningfully. Use a TTL so a
  driver who goes offline silently disappears from the index instead of being
  matched as a ghost.
- **Decouple ingest from indexing.** Front the pipeline with a queue/stream (e.g.
  Kafka) so a burst of pings is absorbed and the index updates asynchronously — the
  same back-pressure idea from the rate-limiter/queue lessons.

## 6. Deep dive: the matching flow

When a rider requests a ride:

1. Compute the pickup's geo-cell; query that shard for **available drivers in the
   cell and its neighbors**.
2. **Rank** the candidates — straight-line distance is a first cut, but real systems
   use **road ETA** (the nearest driver as the crow flies may be across a river).
   Factor in driver rating, direction of travel, and fairness.
3. **Offer** the ride to the top candidate and wait for accept/decline with a short
   timeout. On decline or timeout, offer the next. This is a sequential dispatch
   loop.
4. **Lock the driver** on acceptance so they can't be double-booked. This is the one
   spot that needs **stronger consistency** — two riders must not both be told they
   got the same driver. A short-lived lock / conditional update (compare-and-set on
   driver state) handles it; the rest of the system can stay loose.
5. Record the trip in Postgres and start streaming live positions to both apps.

Note the consistency split: **reads of driver location are eventually consistent**
(stale-by-seconds is fine), but the **accept step is a critical section** requiring
a real lock. Knowing *where* to spend consistency is the senior move.

## 7. Trade-offs

- **Cell size.** Big cells → few shards, simple, but more drivers scanned per query
  and coarser hot-spot control. Small cells → tighter queries and better load
  spreading, but more neighbor-cells to fan out to and more shards to manage.
  Adaptive sizing splits the difference at the cost of complexity.
- **Freshness vs load.** Frequent pings give accurate matches and live tracking but
  drive up write volume and cost. The ping interval is a dial between the two.
- **Consistency vs availability.** We chose **AP** for locations: a slightly stale
  position is acceptable, refusing to dispatch is not. We pay for it only at the
  driver-lock step, where we *do* demand consistency. Don't make the whole system
  pay consensus costs for a guarantee you only need in one place.
- **Surge.** When demand outstrips nearby supply, you can't conjure drivers. Surge
  pricing is the relief valve — it dampens demand and pulls in supply, and it leans
  on the same proximity index to measure local supply/demand imbalance per cell.
- **Build vs buy.** PostGIS gets you a correct geospatial index today on
  infrastructure you already run; Redis GEO gets you speed; S2/H3 + custom shards is
  where you go at true scale. Start simple, measure, then specialize.

## Do it yourself (≈ 5 hrs)

1. Read the **geospatial index** and **system-design case studies** sections of the
   free [**System Design Primer**](https://github.com/donnemartin/system-design-primer).
2. In Postgres, enable **PostGIS**, create a `drivers(location geography(Point))`
   table with a **GiST** index, and run a `ST_DWithin` query — confirm with
   `EXPLAIN` that it uses the spatial index, not a seq scan.
3. Implement a tiny **geohash** in Go: encode (lat, lng) to a prefix, and write a
   function that returns a cell's 8 neighbors. Use it to do a fan-out proximity
   query over an in-memory driver map.
4. **Tie it to Lokatalent:** sketch how your matching service would shard providers
   by region, where its hot spots would be, and where it needs a lock vs where stale
   reads are fine. Write the one-paragraph design.

## Check yourself

- Why can't you answer "nearest driver" with an ordinary B-tree index, and what does
  a geospatial index do instead?
- Name three geospatial indexing approaches and one production system that uses each.
- Why shard the location index **by geo-cell** rather than by driver id?
- Where in this design do you need strong consistency, and where is eventual
  consistency fine — and why?
- How does this map onto matching/dispatch in your own Lokatalent system?

Next: **Step 9 wrap-up** — assembling the framework into answers you can give cold.

## Common interview gotchas

- **Querying only the rider's own cell.** A pickup near a cell boundary has its nearest driver in the *adjacent* cell. Always fan out to the cell **plus its 8 neighbors** and merge — otherwise you silently miss the closest driver and return a worse match.
- **Queueing location pings under back-pressure.** Positions are last-write-wins; only *now* matters. When ingest falls behind, load-shed — keep the latest ping and drop stale ones — rather than queueing a backlog you'll just overwrite. Buffering old positions wastes work and serves ghosts.
- **Letting two riders claim one driver.** The accept step is a real critical section. Lock the driver with an atomic compare-and-set on their state; the loser retries with the next candidate. Reads of location stay eventually consistent — spend consistency *only* here, not system-wide.
- **Treating cell size as free.** Big cells = fewer shards but more drivers scanned and coarse hot-spot control; small cells = tighter queries but more neighbor fan-out and shards to manage. It's a recall-vs-fan-out dial you must name, not a default.
- **Uniform cells over a non-uniform world.** A downtown cell at rush hour is a literal hot partition while rural cells sit idle. Use adaptive granularity (finer cells where dense) and replicate hot shards — fixed-size cells guarantee a hot-spot at scale.
