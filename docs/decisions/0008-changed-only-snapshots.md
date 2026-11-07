# 0008: Snapshots carry only what changed, and arrivals are batched

- **Status:** Accepted
- **Date:** 2026-11-07
- **BUILD.md:** 2.4 (and the starting point for 3.4)

## Context

The first Phase 2 server sent every entity to every player on every tick. The first load test ([docs/perf/2026-11-07-idle-t1-before.md](../perf/2026-11-07-idle-t1-before.md), 100 idle bots on `dev-m5`) measured what that costs when nobody moves:

- 29 KB/s to each idle player, against Case 1's budget of 200 B/s ([scaling.md](../design/scaling.md#case-1-concurrent-connections)),
- 4.6% of a core for 100 idle players, against 2%,
- 135 KiB of server memory per connection, against 64 KiB, much of it buffers sized for a snapshot of every entity.

A profile at 1,000 idle bots showed the same: the largest live allocations were snapshot buffers and the 7 KB of entities copied into each one, and the server was pushing 74 MB/s of unchanged positions.

The same run found a second problem: only about 515 of 1,000 bots stayed connected. A newcomer was sent one `EntityJoined` per player already in the zone, all at once, and past 256 players that overflowed its 256-message send queue, so every later newcomer was evicted as too slow (159 TOO_SLOW disconnects at 600 bots). Any login storm or crowded zone would hit the same wall.

The classic answer, quantized deltas against the last snapshot the client acknowledged (Quake 3, Source), exists because those games use UDP, where any snapshot can be lost. Lorehallow runs over WebSocket on TCP: every message arrives, in order, or the connection ends.

## Decision

A `Snapshot` carries only the entities whose visible state (position, facing, animation) changed on that tick. An entity left out keeps the state the client last heard from `ServerHello` (itself), `EntitiesJoined` (everyone else) or an earlier `Snapshot`. Because nothing is ever lost on a live connection, the server needs no acknowledgements or baselines per client: "changed since the previous tick" is the same for every recipient, so it is still encoded once and shared.

A player gets no `Snapshot` on a tick where nothing it can see changed and its acknowledged input didn't move, except a keepalive at least once a second (`zone.Config.KeepaliveTicks`), so the client keeps a server-tick reference.

**Arrivals and departures are batched per tick.** `EntityJoined` and `EntityLeft` (fields 7 and 8, now reserved) are replaced by `EntitiesJoined` and `EntitiesLeft` (fields 10 and 11). The zone collects everyone who joined or left since the last tick and announces them at the start of the next one: players already there get one `EntitiesJoined` with the newcomers and one `EntitiesLeft`, and each newcomer gets the whole zone (itself included, which clients ignore) in chunks of up to 400. Every message is encoded once and shared. However fast players arrive, a player receives a few messages per tick, not one per arrival. A player who joins and leaves before the next tick is never announced.

Both changes alter the protocol, so `PROTOCOL_VERSION` goes from 1 to 2 (no client had shipped).

How it is checked:

- `sim.Player.Changed` is set by `Step`, so "what changed" is part of the deterministic simulation and its tests.
- `TestDeltasRebuildTheWorld` (internal/zone): six players moving at random for 200 ticks; every player rebuilds the world from ServerHello, EntitiesJoined, EntitiesLeft and the Snapshots it received, and must match the server's state exactly.
- Keepalive timing and the "an ack alone sends a Snapshot" rule have their own tests.
- `TestJoinStormIsBatched`: 600 players joining between two ticks cost each existing player two batch messages and each newcomer at most four.
- Measured on `dev-m5` before and after (both in `docs/perf/`): see the results linked from [scaling.md](../design/scaling.md#status-of-each-tier).

## Alternatives considered

- **Acknowledged-baseline deltas** (per client: diff against the last snapshot it acknowledged). Needed on UDP; on TCP it adds per-client baselines and breaks encode-once for no gain.
- **Keep full snapshots and lower the rate for idle players.** Still sends unchanged state, just less often, and makes the client's timeline irregular.
- **Raise the send queue's message limit instead of batching.** The byte limit already bounds memory, but one WebSocket message and one write per arrival is the real waste: a newcomer to a 1,000-player zone would cost 1,000 writes instead of 3.
- **Wait for Phase 3.** The waste was measured now, it blocked the Phase 2 Case 1 criterion, and the fix is small; backend quality comes first when there's a choice.

## Consequences

- An idle world costs almost nothing to send: a keepalive a second and a heartbeat every two seconds per player.
- The client must treat Snapshots as deltas from day one (BUILD.md 2.6), and must never assume an absent entity left: only `EntitiesLeft` removes one.
- A newcomer sees the zone up to one tick (50 ms) after `ServerHello`, and others see it then too.
- A client that misses a message has a wrong world, so a protocol error must end the connection, and reconnecting starts over from `ServerHello`. That already holds for WebSocket.
- Phase 3.4 builds on this: quantization and smaller encodings of the changed entities (for example position deltas instead of absolute positions), and Phase 3.3's interest management decides which changed entities each player hears about, with `EntityJoined` and `EntityLeft` at the edge of the area.
- In a crowd where everyone moves every tick (Case 3), this saves nothing; that case is Phase 7's budgets.
