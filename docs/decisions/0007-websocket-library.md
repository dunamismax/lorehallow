# 0007: coder/websocket for the WebSocket transport

- **Status:** Accepted, amended by [ADR 0010](0010-per-connection-memory.md) (the session runs on its own goroutine, with smaller buffers)
- **Date:** 2026-11-07
- **BUILD.md:** 2.3

## Context

The game protocol runs over WebSocket (ADR 0001). The Go standard library has no WebSocket server, so the server needs a library or its own implementation. The choice matters for the scale targets ([scaling.md](../design/scaling.md)): Case 1 budgets at most 2 goroutines and 48 KiB per idle connection, and the send path must not allocate per message where it can avoid it.

## Decision

Use **github.com/coder/websocket v1.8.15** (formerly nhooyr.io/websocket), with compression disabled, a read limit of `wire.MaxClientMessage`, and origin checks through `AcceptOptions.OriginPatterns`.

Checked in its source before choosing:

- **No goroutines of its own per connection.** Context cancellation for reads and writes uses `context.AfterFunc` (since v1.8.13; earlier versions ran a `timeoutLoop` goroutine per connection). A Lorehallow connection is therefore exactly two goroutines: the HTTP handler goroutine, which runs the read loop, and one writer.
- **Reuses net/http's buffers.** After the hijack, the connection keeps the server's 4 KiB `bufio.Reader` and `bufio.Writer` instead of allocating its own.
- **Zero dependencies**, context-aware API, a correct close handshake, `CloseNow` for paths that must not wait, and active maintenance under Coder.

How the server uses it:

- Reads use `Conn.Reader` into a reused buffer, never `Conn.Read` (which allocates a new slice per message).
- There are no per-read timeouts. A heartbeat in the writer goroutine closes connections that go silent, and a full send queue closes slow ones; closing the connection unblocks any read or write in progress.
- Graceful closes (shutdown, protocol mismatch) send a `Disconnect` and use `Close` with the close code `4000 + DisconnectReason`. Evictions (too slow, timed out, flooding) send a best-effort `Disconnect` and then `CloseNow`, so a hostile or stuck client can't hold the server's goroutines for the close handshake's 10-second timeouts.

## Alternatives considered

- **gorilla/websocket.** Mature and fast, with `PreparedMessage` for identical fan-out and deadline-based I/O. But per-recipient snapshots differ by a few header bytes, so `PreparedMessage` doesn't apply, its API is older and not context-aware, and coder/websocket now matches it on goroutines per connection.
- **gobwas/ws.** Lowest level and lowest memory (raw frames on a `net.Conn`, usable with an epoll loop for hundreds of thousands of connections). That scale is far beyond the targets, and it hands every protocol detail (fragmentation, control frames, close handshake) to us.
- **Write our own RFC 6455 server.** A good lesson, but security-sensitive parsing code with no payoff the targets need. Revisit only if a profile shows the library is the bottleneck.

## Consequences

- One small dependency, recorded in `server/go.mod`.
- If Case 1 measurements miss the memory budget because of the 8 KiB of bufio buffers per connection, the options are smaller buffers through a custom `http.Server` connection setup, or gobwas/ws. That decision would supersede this ADR.
- They did. [ADR 0010](0010-per-connection-memory.md) keeps this library, swaps in 1 KiB and 2 KiB buffers at the hijack, and runs the session off the HTTP handler goroutine.
