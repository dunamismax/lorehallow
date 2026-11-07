# 0010: Sessions off the HTTP goroutine, with smaller transport buffers

- **Status:** Accepted
- **Date:** 2026-11-07
- **BUILD.md:** 2.3, Phase 2 acceptance (Case 1 Tier 1)
- **Amends:** [ADR 0007](0007-websocket-library.md)

## Context

Case 1 in [scaling.md](../design/scaling.md) budgets server memory per idle connection: 64 KiB at Tier 1, 48 KiB from Tier 2. The first idle runs measured 84 KiB at 100 connections and 60 KiB at 1,000 ([idle t1](../perf/2026-11-07-idle-t1.md), [idle t2](../perf/2026-11-07-idle-t2.md)).

An exact heap profile of 1,000 idle connections (`GODEBUG=memprofilerate=1`, `/debug/pprof/heap?gc=1`) put live heap at 18.6 KB per connection, plus about 10 KiB of goroutine stacks:

| What | Per connection | Why it was alive |
|---|---|---|
| net/http's `bufio.Reader` and `bufio.Writer` | 8 KiB | Handed over by `Hijack`; coder/websocket keeps both for the connection's life (ADR 0007 noted this) |
| The upgrade request, its headers, URL and contexts, the `response` and net/http's `conn` | about 4 KB | The HTTP handler goroutine ran the session, so its frames kept the request and net/http's state reachable until the player left |
| Session, send queue, websocket `Conn`, timers | about 6 KB | Needed |

Messages are small. A client message is at most 512 bytes (`wire.MaxClientMessage`). A Phase 2 Snapshot is tens of bytes, and a crowded Phase 3 one a few hundred. The 4 KiB buffers are sized for HTTP, not for this traffic.

## Decision

1. **The session runs on its own goroutine.** The `/ws` handler admits the connection (limits, the upgrade), starts a goroutine that runs `session.Serve`, and returns. net/http then lets go of its request and connection state, and the handler goroutine's deep stack is gone. The count is still two goroutines per connection: the session goroutine reads, and `Serve` starts the writer. A `slot` in `internal/gameserver` holds the connection's share of the limits (session count, connection count, pending handshake) and gives it back exactly once.
2. **Smaller transport buffers.** The handler gives coder/websocket a `ResponseWriter` whose `Hijack` swaps net/http's 4 KiB buffers for a 1 KiB reader (`gameserver.ReadBufferSize`, two of the largest client messages) and a 2 KiB writer (`gameserver.WriteBufferSize`). bufio writes anything larger than its buffer straight to the socket, so a big message (a newcomer's `EntitiesJoined`) costs one extra write call, never a copy or a failure. If a client sent bytes before the `101` response, or the writer still holds data, the original buffer is kept so nothing is lost.
3. The session's read buffer starts at 64 bytes instead of 512 and grows on demand up to the read limit.

## Alternatives considered

- **gobwas/ws or our own RFC 6455 layer.** Lower memory and full control of buffers, but it means owning frame parsing, fragmentation, control frames and the close handshake, the security-sensitive part ADR 0007 chose not to own. The two changes above recover most of what that would save.
- **A custom `http.Server` connection setup** (ADR 0007's other option). The `Hijack` wrapper does the same with a few lines and no change to how the server listens.
- **Tune `GOGC` or `GOMEMLIMIT` to pass the test.** That hides memory rather than removing it, and BUILD.md 3.5 already plans that decision under Case 2 load, where it belongs.
- **One goroutine per connection** (reader and writer merged around a poller). It would save a stack, but blocking writes would then hold up reads, and it means building an event loop the targets don't need.

## Consequences

- Measured on `dev-m5`: live heap per idle connection 18.6 → 10.0 KB at 1,000 connections, and RSS per idle connection 84.1 → 72.3 KiB at Tier 1 ([result](../perf/2026-11-07-idle-t1-2.md)) and 60.3 → 38.7 KiB at Tier 2, which now meets its 48 KiB budget ([result](../perf/2026-11-07-idle-t2-2.md)).
- Tier 1 is still not met. At 100 connections the live heap is about 1 MB, under Go's 4 MB minimum GC goal, so memory freed after the handshakes stays resident until the scavenger returns it. That is about 3 MB of runtime slack, which the metric divides by 100 and reports as about 30 KiB per connection. Shrinking that slack is a GC-settings question for the BUILD.md 3.5 ADR. Whether the Tier 1 metric should subtract a fixed runtime floor is a contract question for the owner.
- `r.Context()` is cancelled once the handler returns. Sessions never used it: they run on the server's own shutdown context. Code added later must not either.
- net/http's original 4 KiB buffers become garbage after each upgrade instead of staying in use. That adds a little allocation per connect, which is cheap next to the handshake.
