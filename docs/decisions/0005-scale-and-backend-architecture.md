# 0005: Scale targets and the backend architecture

- **Status:** Accepted
- **Date:** 2026-11-07
- **BUILD.md:** Phases 2, 3, 7, 11 and 20; supersedes the scale non-goal in AGENTS.md section 1

## Context

The project began with a modest scale goal: "Dozens is the real target; hundreds is a stretch goal", and a non-goal against thousands of players. The owner has raised it. Lorehallow now aims for **hundreds to thousands of simultaneous, actively connected players**, and the Go server and its networking stack are the project's main engineering achievement. Players see a small, beautiful world; the infrastructure behind it is the showcase.

Two things follow. First, "how many players" needs a precise meaning, measurable targets and named hardware, or the goal can't be tested and will drift into marketing. Second, the server's shape has to be chosen now, before Phase 2 writes the first tick loop, because the patterns that hold up at thousands of connections (ownership of state, backpressure, encode-once fan-out) are hard to retrofit.

The stack stays as decided in [ADR 0001](0001-stack.md): Go, Godot 4 with GDScript, WebSocket over TLS, Protobuf with buf, PostgreSQL, Caddy, self-hosted. No game-server framework (Nakama or similar): the point is to build and understand the server.

## Decision

### Targets

The scaling model and performance contract is [docs/design/scaling.md](../design/scaling.md). It keeps three cases apart (concurrent connections, active players across many zones, a crowded area), sets tiered targets from about 100 to 1,000 to several thousand for each, names the metrics and the load-test scenario for each, and covers bursts, slow and hostile clients and soak. Targets name their hardware; until `deploy/HOST.md` exists they are measured on the development machine and recorded with its specs. Results go in `docs/perf/`. **Thousands is a target, not a capability: nothing claims a player count that hasn't been measured.**

### Architecture direction

One server process with many zones, built so it can later split across processes and machines when measurements say so.

```
            ┌──────────────────────── lorehallow-server (one process) ────────────────────────┐
 client ──► │ conn reader goroutine ──► zone inbox (bounded chan) ──► zone goroutine (tick)    │
  (wss,     │                                                           │  owns all zone state │
  Caddy)    │ conn writer goroutine ◄── send queue (bounded) ◄──────────┘  encode once, fan out│
            │                                                                                  │
            │ world coordinator goroutine: zone registry, handoff, cross-zone routing          │
            │ persistence workers (bounded queue) ──► PostgreSQL     metrics, pprof (localhost)│
            └──────────────────────────────────────────────────────────────────────────────────┘
```

1. **One goroutine owns each zone.** A zone's tick loop is the only code that reads or writes that zone's entities, positions, combat state and spatial index. No mutexes around world state. The simulation inside it (`internal/sim`) is pure and deterministic: no I/O, an injected clock, the same inputs give the same outputs, tick by tick.

2. **Channels at the edges.** Everything else talks to a zone by message. A connection's reader goroutine decodes and validates a client message, then puts it on the zone's bounded inbox; the tick loop drains its inbox at the start of each tick and applies inputs in sequence order. A full inbox drops the message and counts it (and repeated drops from one client are a disconnect), so a flood can't grow memory or delay the tick. Cross-zone work (handoff, whispers, parties) goes through a world coordinator goroutine, also by message.

3. **Bounded send queues with slow-client eviction.** Each connection has one writer goroutine and a send queue bounded by message count and bytes. The tick loop enqueues without blocking. If a queue is full, the client is too slow: it is evicted with a clear close reason and removed from the world like any other disconnect. The tick loop never waits for a socket.

4. **Encode once, then fan out.** Each tick, every entity's update is serialized once into an immutable byte slice. A recipient's snapshot is assembled from the shared slices of the entities in its area of interest plus a small per-recipient header (its acknowledged input and tick). The same applies to chat and events sent to many players. Encoding cost grows with entities, not with entities × recipients. Buffers are pooled so steady-state allocation per tick stays near zero.

5. **Spatial grid interest management.** A uniform grid per zone, with a cell size chosen in an ADR (Phase 3), answers "who is near this point" for snapshots, say chat, combat events and AI. Each player's area of interest has hysteresis (enter radius smaller than leave radius). Snapshots are quantized deltas against the last snapshot the client acknowledged, with update-rate tiers by distance and a per-player byte budget for the crowded case.

6. **Zone handoff, then splitting across processes when measurements say so.** Moving a player between zones is a message protocol between zone goroutines through the coordinator: freeze and serialize the player in the old zone, spawn in the new one, roll back on failure. Because zones already talk only through messages and serialized state, the same protocol can later run between processes: a gateway keeps client connections and zone processes run the simulations. That split (BUILD.md Phase 20) happens only when a measurement in `docs/perf/` shows a single-process limit that tuning can't fix, recorded in its own ADR. The triggers are listed in scaling.md.

7. **Persistence never runs in a tick.** Zones hand state changes to a bounded persistence queue drained by a small worker pool. Item and currency changes are transactional and idempotent (request ids), and survive a crash between the game event and the commit.

8. **Measurement is part of the design.** A headless bot load generator (`lorehallow-bot`) that scales to thousands of clients arrives in Phase 2, `make loadtest` scenarios cover every case in scaling.md, and the server exposes tick-time histograms, queue depths, drops, evictions and bytes in and out from Phase 3. pprof and metrics listen on localhost only.

### What stays out

No microservices, Kubernetes, service mesh or distributed database until a measured bottleneck justifies one, recorded in an ADR. One PostgreSQL database. No UDP for now: WebSocket over TLS keeps the browser build possible and a tab-target game doesn't need it.

## Alternatives considered

- **Shared world state behind locks.** Simple at first, but lock contention, deadlocks and non-deterministic ordering get worse with every system added, and the simulation can't be tested tick by tick.
- **A goroutine (actor) per entity or per player.** Elegant in theory; in practice interactions between entities need cross-goroutine messaging on every tick, ordering becomes non-deterministic, and thousands of entities pay scheduler cost for work a single loop does in a cache-friendly pass.
- **One process per zone from day one.** Teaches distributed systems early, but adds an internal protocol, failure modes and deploy complexity before any measurement shows a need. The message-passing zone boundary keeps the option open at low cost.
- **A game-server framework (Nakama, Colyseus-style servers).** Hides exactly the parts the project exists to build and teach.
- **An ECS library.** The entity architecture is its own ADR in Phase 6; nothing here forces or forbids components and systems inside a zone's loop.
- **Encode per recipient.** Simplest code, but O(n²) serialization in the crowded case is the first thing to fall over.
- **Unbounded send queues, or blocking writes from the tick.** The classic way one slow client freezes or bloats a whole server.

## Consequences

- Every phase from 2 on has measurable performance acceptance criteria pointing at scaling.md, and backend quality is the deciding factor when there is a real choice of what to build next.
- Connection lifecycle, backpressure, the bot generator and metrics come early (Phases 2 and 3), so later features can be measured as they land.
- The zone boundary is a message protocol from the start, which costs some indirection in Phase 2 but makes Phase 11 handoff and Phase 20 splitting a move rather than a rewrite.
- Testing gets stricter: race detector, fake clock, fuzzing of the decoder and handlers, allocation benchmarks, goroutine-leak checks, attack tests and mid-transaction failures against real Postgres (AGENTS.md section 9).
- The old non-goal is gone. The website, README and docs say "hundreds to thousands" as a goal, and only measured numbers as capabilities.
