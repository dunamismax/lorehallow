# Scaling model and performance contract

Lorehallow aims to host **hundreds to thousands of simultaneous, actively connected players**. This document says what that means, how it is measured and what numbers count as passing. The architecture behind it is [ADR 0005](../decisions/0005-scale-and-backend-architecture.md).

Thousands is a **target, not a capability**. Nothing in this repository, on the website or in a devlog may claim a player count that hasn't been measured and recorded in [`docs/perf/`](../perf/). Until a tier below is marked as met with a link to its result, Lorehallow has not reached it.

## Contents

1. [Three cases, kept apart](#three-cases-kept-apart)
2. [Hardware and how results count](#hardware-and-how-results-count)
3. [Budgets that apply everywhere](#budgets-that-apply-everywhere)
4. [Case 1: concurrent connections](#case-1-concurrent-connections)
5. [Case 2: active players across many zones](#case-2-active-players-across-many-zones)
6. [Case 3: a crowded area](#case-3-a-crowded-area)
7. [Bursts](#bursts)
8. [Slow and hostile clients](#slow-and-hostile-clients)
9. [Soak](#soak)
10. [Load-test scenarios](#load-test-scenarios)
11. [Status of each tier](#status-of-each-tier)
12. [When to split across processes](#when-to-split-across-processes)

## Three cases, kept apart

"How many players can it take?" has no single answer, because three different situations stress different things. A server that handles 5,000 idle sockets can still fall over when 300 people stand in one town square. Every target below belongs to exactly one case.

| Case | What it is | What it costs | What runs out first |
|---|---|---|---|
| 1. Concurrent connections | Many sockets, mostly quiet: logged in, standing still, chatting now and then, or far from anyone | Memory per connection, goroutines, buffers, heartbeats | Memory, file descriptors, ephemeral ports |
| 2. Active players across many zones | Everyone moving and fighting, spread out so each player sees a few dozen others | Total simulation and network work, spread over many zone tick loops | CPU across cores, total outbound bandwidth |
| 3. A crowded area | Hundreds of players who can all see each other | Fan-out is O(n²): n players each receive n entities every tick | Per-player bandwidth, snapshot size, encoding work in one zone's tick |

The cases share code but not targets. Case 1 is about being cheap when nothing happens. Case 2 is about using every core without the zones getting in each other's way. Case 3 is about sending less: interest management, quantized deltas, update-rate tiers and encoding each entity's state once for every recipient instead of once per recipient.

## Hardware and how results count

**Targets name the hardware they apply to.** A tick time means nothing without the CPU it was measured on.

### Reference machines

| Name | Hardware | Use |
|---|---|---|
| `dev-m5` | MacBook Pro (Mac17,2), Apple M5, 10 cores (4 performance, 6 efficiency), 32 GB RAM, macOS 27.2, Go 1.27.3, `ulimit -n` 1048576 | Every benchmark until the production host exists |
| `prod` | To be recorded in [`deploy/HOST.md`](../../deploy/HOST.md) (BUILD.md 1.1) | The real targets, once it is filled in |

Until `deploy/HOST.md` is filled in, every target is measured on `dev-m5`. When the production host is known, each tier is re-measured there and both results are kept. A result on `dev-m5` never counts as a result on `prod`.

### Rules for results

- **Only measured numbers count.** Never estimate a result, round one in our favor or copy a target into a result.
- Every result is a file in `docs/perf/` named `<YYYY-MM-DD>-<scenario>-<tier>.md`, using the template in [`docs/perf/README.md`](../perf/README.md): the hardware, the git commit, the exact command, how long it ran, the raw numbers and whether each target passed.
- Record where the load generator ran. A bot swarm on the same machine competes with the server for CPU; say so, and report the generator's own CPU use so it is clear the generator wasn't the bottleneck.
- Report p50, p99 and max, not averages. Tick time is measured on the server's monotonic clock from the start of a tick to the end of its last enqueue.
- A target is met when the scenario runs for its full duration and every listed metric passes. One failing metric fails the tier.
- A regression against a recorded result is a bug. Fix it or record why the trade is worth it.

### Machine limits to keep in mind

- One client IP talking to one server port has about 16,000 ephemeral ports on macOS (49152 to 65535) and about 28,000 on Linux by default. Load tests above that spread bots across several loopback addresses (`127.0.0.2` and up on Linux; aliases on macOS) or several generator machines.
- File descriptors: raise `ulimit -n` for both the server and the generator, and set `LimitNOFILE` in the systemd unit.
- In production, TLS ends at Caddy, so Caddy holds the TLS memory per connection and the Go server sees plain WebSocket over loopback. Measure both processes once the host exists.

## Budgets that apply everywhere

These hold in every case and every tier.

| Budget | Value | Why |
|---|---|---|
| Tick rate | 20 Hz, a 50 ms tick | AGENTS.md section 7 |
| Zone tick p99 | 12.5 ms (a quarter of the tick) in normal load; 25 ms in the crowded case; never above 50 ms for more than one tick in a row | Leaves headroom for GC, bursts and a noisy neighbor on the host |
| Tick loop never blocks on I/O | No socket writes, database calls or locks shared with connection goroutines inside a tick | One slow client or slow query must not stall a zone |
| Steady-state allocations | 0 allocations per entity per tick in the simulation step; at most 1 allocation per recipient per tick for outgoing messages, and 0 once buffers are pooled | GC work grows with allocation rate; at thousands of players it becomes the tick-time tail |
| GC | GC CPU under 5% of total server CPU at steady state; GC pause (stop-the-world) p99 under 1 ms | Go's GC is concurrent, but assist work and pauses land inside ticks |
| Goroutines per connection | At most 2 (one reader, one writer) | Bounds memory and scheduler work in Case 1 |
| Send queue per connection | Bounded (by messages and bytes); full means the client is too slow and gets evicted, never that the tick waits | See [slow clients](#slow-and-hostile-clients) |
| Snapshot latency | Tick end to bytes handed to the socket: p99 under 5 ms | Interpolation hides about 100 ms; the server shouldn't eat much of it |

## Case 1: concurrent connections

**The question:** how many connected, mostly idle players can one server process hold, and what does each cost?

A connection counts when it has completed the handshake, is in a zone and answers heartbeats. "Mostly quiet" means each player sends a heartbeat and occasionally a chat line or a single step, and sees nobody or almost nobody (they are spread out, or in a quiet area).

| Metric | Tier 1: 100 | Tier 2: 1,000 | Tier 3: 5,000 | Tier 4: 10,000 (stretch) |
|---|---|---|---|---|
| Server memory per idle connection, marginal: (RSS at N − RSS at N/2) / (N/2) | ≤ 64 KiB | ≤ 48 KiB | ≤ 48 KiB | ≤ 48 KiB |
| Goroutines: total − baseline | ≤ 2N | ≤ 2N | ≤ 2N | ≤ 2N |
| Server CPU at steady state | ≤ 2% of one core | ≤ 10% of one core | ≤ 50% of one core | ≤ 1 core |
| Idle bandwidth per connection, both directions | ≤ 200 B/s | ≤ 200 B/s | ≤ 200 B/s | ≤ 200 B/s |
| Heartbeat round trip (server Ping to Pong), p99, loopback | ≤ 50 ms | ≤ 50 ms | ≤ 100 ms | ≤ 200 ms |
| Zone tick p99 with everyone idle | ≤ 2 ms | ≤ 5 ms | ≤ 12.5 ms | ≤ 12.5 ms |
| Goroutines and memory after everyone disconnects | Back to baseline ±1% within 10 s | same | same | same |

**Scenario:** `make loadtest SCENARIO=idle N=<n>`. Bots connect at a steady rate, handshake, spawn spread across the map and then only answer heartbeats (and take one step a minute). Hold for 10 minutes, then disconnect all and watch memory and goroutines return to baseline.

**Why memory is measured as a marginal cost** (changed 2026-11-07, owner-approved, [ADR 0010](../decisions/0010-per-connection-memory.md)): the bots connect in two halves, and RSS is sampled once the first half has settled and again over the main window. Memory at N is a fixed part plus N times a per-connection part, and the difference between the two samples cancels the fixed part. The fixed part is real but not per player: the Go runtime's GC won't collect until the heap reaches 4 MB, and runtime structures grow once with the first connections. The old metric, (RSS at N − RSS at 0) / N, spread that fixed part over N, which inflated small tiers: about 30 KiB of the 72 KiB measured at 100 connections. Every result still records the old metric and the fixed part, so neither is hidden. The fixed-part estimate is noisy at large N and can come out negative when the GC's heap goal steps up between the two samples. The marginal number then reads high, so the error is on the strict side.

## Case 2: active players across many zones

**The question:** how many players can be actively playing at once, spread across the world the way real players spread out?

"Active" means moving or acting most of the time: a wandering bot sends an `Input` every tick it moves, fights now and then and chats occasionally. "Spread out" means the average player sees about 20 other entities and no area holds more than about 50. Total work is spread over many zone tick loops, so this case measures how well the server uses every core.

| Metric | Tier 1: 100 | Tier 2: 1,000 | Tier 3: 3,000 to 5,000 |
|---|---|---|---|
| Layout | 1 zone | 10 or more zones (or one large zone with players spread out) | 30 or more zones |
| Zone tick p99, every zone | ≤ 12.5 ms | ≤ 12.5 ms | ≤ 12.5 ms |
| Zone tick max over the run | ≤ 50 ms | ≤ 50 ms | ≤ 50 ms |
| Outbound bandwidth per player (WebSocket payload plus framing) | ≤ 8 KB/s | ≤ 8 KB/s | ≤ 8 KB/s |
| Inbound per player | ≤ 1 KB/s | ≤ 1 KB/s | ≤ 1 KB/s |
| Total server CPU | ≤ 1 core | ≤ 4 cores | ≤ 70% of all cores |
| Allocations per tick per zone, steady state (from `runtime/metrics`) | ≤ 1 per recipient | ≤ 1 per recipient | 0 per entity, ≤ 1 per recipient |
| GC CPU fraction | ≤ 5% | ≤ 5% | ≤ 5% |
| Input to visible result (input received to the snapshot that reflects it leaving the server), p99 | ≤ 60 ms | ≤ 60 ms | ≤ 60 ms |
| Snapshot latency (tick end to socket), p99 | ≤ 5 ms | ≤ 5 ms | ≤ 5 ms |

The bandwidth budget assumes Phase 3's quantized delta snapshots. Before they exist (Phase 2), bandwidth is measured and recorded but not held to this target.

**Scenario:** `make loadtest SCENARIO=spread N=<n> ZONES=<z>`. Bots wander, idle and follow each other, with configurable simulated latency, across every zone. Hold for 15 minutes after a 2-minute warm-up.

## Case 3: a crowded area

**The question:** what happens when hundreds of players stand in one place and can all see each other: a town square, a world event, a boss?

Here fan-out is O(n²). With 300 players in view of each other, every tick produces 300 × 300 = 90,000 entity updates. At 8 bytes each and 20 Hz that is 48 KB/s for every player and about 14 MB/s leaving the server, before framing. Spreading the work across cores doesn't help, because it all lives in one zone. The answer is to send less and to do the expensive work once:

- **Interest management:** a uniform spatial grid, so each player only receives entities within their area of interest, with hysteresis so the edge doesn't flicker.
- **Quantized deltas:** positions in fixed point, sent as small deltas against the last snapshot the client acknowledged; unchanged entities cost nothing.
- **Encode once, fan out:** each entity's per-tick update is serialized once, and each recipient's message is assembled from shared, immutable byte slices. Encoding cost is O(entities), not O(entities × recipients).
- **Update-rate tiers and a per-player budget:** nearby entities update every tick, distant ones every second or fourth tick, and a per-player byte budget with a priority accumulator decides what to send when the area is too full. Under overload the server degrades rates, never correctness.

| Metric | Tier 1: 100 in one area | Tier 2: 300 in one area | Tier 3: 1,000 in one area (degradation test) |
|---|---|---|---|
| Zone tick p99 | ≤ 12.5 ms | ≤ 25 ms | ≤ 50 ms |
| Outbound bandwidth per player | ≤ 24 KB/s | ≤ 32 KB/s (budget enforced) | ≤ 32 KB/s (budget enforced) |
| Every visible entity updated at least | 20 Hz | 5 Hz | 2 Hz, nearest 50 at 10 Hz or more |
| Encode work | Each entity encoded once per tick, independent of recipient count (benchmark) | same | same |
| Snapshot size p99 | ≤ 1,200 bytes | ≤ 1,600 bytes (budgeted) | ≤ 1,600 bytes (budgeted) |
| Players disconnected by the server | None, except clients that were too slow | same | same |
| Recovery when the crowd disperses | Back to the Case 2 numbers within 5 s | same | same |

**Scenario:** `make loadtest SCENARIO=crowd N=<n>`. Every bot spawns in one area and moves inside it, so all of them stay in view of each other. Hold for 10 minutes.

## Bursts

Real load arrives in spikes: a server comes back and everyone reconnects at once, a world event starts and everyone acts on the same tick.

| Burst | Target (Tier 2 numbers; Tier 3 scales N by 3 to 5) | Scenario |
|---|---|---|
| Login storm: 1,000 clients connect within 10 s | Every client in the world within 15 s; handshake p99 ≤ 2 s; existing zones' tick p99 stays ≤ 25 ms during the storm. From Phase 4, password hashing runs on a bounded worker pool, so a storm queues logins instead of starving tick loops | `SCENARIO=storm` |
| Reconnect storm: the server restarts with 1,000 players online | All reconnected within 30 s using jittered exponential backoff; no lost progress (from Phase 4) | `SCENARIO=reconnect` |
| Everyone acts at once: every player in a zone uses an ability on the same tick | That tick ≤ 50 ms; the next ticks back under 12.5 ms within 1 s; every action resolved, none dropped silently | `SCENARIO=burst` |
| Churn: players connecting and leaving continuously (10% of N per minute) | No growth in memory or goroutines over 30 minutes; join and leave visible to others within 1 s | `SCENARIO=churn` |

## Slow and hostile clients

The server must protect everyone else from one bad connection.

**Slow clients.** A client that reads too slowly fills its bounded send queue. The server never waits for it. When the queue is full (by message count or by bytes), the connection is closed with a clear reason and the player is removed from the world like any other disconnect.

| Metric | Target |
|---|---|
| A stalled reader is evicted | Within 2 s of its queue filling |
| Effect on everyone else, with 10% of clients reading at a quarter of the needed rate | Healthy clients' tick p99 and bandwidth within 10% of the run without slow clients |
| Memory held by a slow client | Bounded by the send-queue limit, measured |

**Hostile clients.** Every client message is untrusted (AGENTS.md section 11). The server enforces message size, message rate, input-queue depth, handshake time, connections per IP and valid protocol state, and the decoder and handlers are fuzzed.

| Attack | Target |
|---|---|
| Input flood (far above 20 inputs per second) | No effect on movement speed; extra inputs dropped and counted; repeat offenders disconnected |
| Oversized or malformed messages | Connection closed, no panic, no allocation proportional to the claimed size |
| Slowloris (connect, never finish the handshake) | Closed at the handshake timeout; a per-IP limit on pending handshakes |
| Unknown message types and invalid state (input before handshake) | Rejected and counted; disconnect after repeats |
| 100 hostile bots alongside 1,000 normal ones | Normal players' tick p99 and bandwidth within 10% of a run without them |

**Scenarios:** `make loadtest SCENARIO=slow` and `make loadtest SCENARIO=hostile` (the bot's cheat modes from BUILD.md 2.7, run as a swarm).

## Soak

Leaks and slow drift only show up over hours. A soak runs the `spread` scenario with churn for a long time and watches trends, not peaks.

| Soak | Length | Load | Targets |
|---|---|---|---|
| Short | 1 hour | Tier 2 `spread` plus `churn` | RSS slope after warm-up under 1% per hour; goroutines flat; tick p99 in the last 10 minutes within 10% of the first 10 |
| Day | 24 hours | Tier 2 on `prod` | Same, plus no errors in the logs above warning that aren't explained |
| Beta | 72 hours | Tier 2 bots plus real players on `prod` | Same, plus no restarts |

**Scenario:** `make loadtest SCENARIO=soak DURATION=<d>`.

## Load-test scenarios

`lorehallow-bot` (BUILD.md 2.7 and 3.4) is the headless Go client that drives every scenario. It speaks the real protocol over a real WebSocket, so a load test exercises the same code path as players. One generator process must run thousands of bots cheaply (one goroutine per bot plus a shared timer wheel, not a busy loop per bot) and report its own CPU and memory alongside the server's numbers.

`make loadtest SCENARIO=<name> [N=..] [ZONES=..] [DURATION=..] [LATENCY=..]` starts a server built from the current commit (or targets `ADDR=`), runs the scenario, scrapes the server's metrics before, during and after, and writes a draft result file to `docs/perf/` for an agent to check and complete.

| Scenario | Case | What the bots do |
|---|---|---|
| `idle` | 1 | Connect, handshake, answer heartbeats, take one step a minute |
| `spread` | 2 | Wander, idle and follow across every zone |
| `crowd` | 3 | All stay in one area, in view of each other |
| `storm` | Burst | N bots connect within a short window |
| `reconnect` | Burst | The server restarts mid-run; bots reconnect with backoff |
| `burst` | Burst | All bots in a zone act on the same tick |
| `churn` | Burst | A share of bots leaves and rejoins every minute |
| `slow` | Slow clients | A share of bots read at a fraction of the needed rate, or stop reading |
| `hostile` | Hostile clients | Cheat modes: floods, forged sequences, oversized and malformed messages, slowloris |
| `soak` | Soak | `spread` plus `churn` for hours |

## Status of each tier

Updated whenever a result lands in `docs/perf/`. A tier is met only with a linked result on the named hardware.

| Case | Tier | `dev-m5` | `prod` | Phase that must meet it |
|---|---|---|---|---|
| 1. Connections | T1 100 | **Met**: memory 42 and 46 KiB per connection, marginal (target 64); CPU, bandwidth, RTT, tick pass ([result](../perf/2026-11-07-idle-t1-3.md), [repeat](../perf/2026-11-07-idle-t1-4.md)) | Not measured | 2 |
| 1. Connections | T2 1,000 | **Not met**: memory passes (39 KiB marginal, target 48); bandwidth, CPU and tick p99 over without interest management ([result](../perf/2026-11-07-idle-t2-3.md)) | Not measured | 3 |
| 1. Connections | T3 5,000 | Not measured | Not measured | 12 |
| 1. Connections | T4 10,000 | Not measured | Not measured | Stretch |
| 2. Active, spread | T1 100 | Tick and CPU pass; bandwidth 15 KB/s not yet held (one zone, no interest management) ([result](../perf/2026-11-07-spread-t1.md)) | Not measured | 3 |
| 2. Active, spread | T2 1,000 | Not measured | Not measured | 11 |
| 2. Active, spread | T3 3,000 to 5,000 | Not measured | Not measured | 18 (else Phase 20) |
| 3. Crowded area | T1 100 | Not measured | Not measured | 3 |
| 3. Crowded area | T2 300 | Not measured | Not measured | 7 |
| 3. Crowded area | T3 1,000 | Not measured | Not measured | 18 |
| Bursts | Tier 2 | Not measured | Not measured | 4 (storm, reconnect), 6 (burst), 3 (churn) |
| Slow and hostile | Tier 2 | Not measured | Not measured | 3 |
| Soak | 1 h / 24 h / 72 h | Not measured | Not measured | 3 / 12 / 18 |

## When to split across processes

ADR 0005 starts with **one process with many zones**, each zone owned by one goroutine. That design is meant to reach every Tier 2 target and Case 2 Tier 3 on one good machine. The process boundary is designed in from the start (zones talk to the rest of the server only through messages), so splitting later moves code rather than rewriting it.

Splitting zones into several processes, and then several machines, is BUILD.md Phase 20. It starts only when a measurement recorded in `docs/perf/` shows a single-process limit that tuning can't fix, for example:

- total CPU above 70% of the host's cores at a tier the project needs, after a profiling pass;
- memory or file descriptors running out below the Case 1 tier the project needs;
- GC CPU or pause targets missed at scale after allocation work and GC tuning (`GOGC`, `GOMEMLIMIT`);
- one zone crash or deploy taking every player down being unacceptable at the player counts reached.

The decision is recorded in an ADR that links the measurement. Microservices, Kubernetes and distributed databases stay out unless a measured bottleneck justifies them, also recorded in an ADR. Note that splitting processes does not help Case 3: a crowded area lives in one zone, and its fix is sending less, not more machines.
