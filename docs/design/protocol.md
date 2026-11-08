# The wire protocol

How the client and server talk: transport, framing, the handshake, every message with its size budget, and the limits the server enforces. The schema is [`proto/lorehallow/v1/envelope.proto`](../../proto/lorehallow/v1/envelope.proto); the rules for changing it are in [`proto/README.md`](../../proto/README.md). Scale targets are in [scaling.md](scaling.md).

## Transport and framing

- WebSocket (RFC 6455) over TLS in production (`wss://play.lorehallow.com/ws`, TLS ends at Caddy), plain `ws://127.0.0.1:7420/ws` locally.
- **One `Envelope` per binary WebSocket message.** The `payload` oneof says what it is. Text messages are a protocol error.
- No WebSocket compression (permessage-deflate): it costs memory per connection, and the messages are small binary protobuf that compress poorly.
- WebSocket framing adds 2 bytes per server message up to 125 bytes and 4 bytes up to 64 KiB (server frames are unmasked), and 6 or 8 bytes per client message (client frames carry a 4-byte mask).

## Limits

| Limit | Value | At the limit |
|---|---|---|
| Client message size | 512 bytes (`wire.MaxClientMessage`) | Rejected before decoding; `Disconnect` `BAD_MESSAGE` |
| Server message size | 64 KiB (`wire.MaxServerMessage`); clients must accept this | The server never sends more |
| Handshake time | `ClientHello` within 5 s of the upgrade | `Disconnect` `TIMED_OUT` |
| Display name | 1 to 16 characters, letters, digits, spaces, `'` and `-`, no leading or trailing space | `Disconnect` `INVALID_HELLO` |
| Inbound message rate | 60 messages per second sustained, bursts of 120 (token bucket) | Dropped and counted; `Disconnect` `RATE_LIMITED` after repeated overflow |
| Pending inputs per player | 16 | Newest dropped and counted |
| Send queue per player | 256 messages or 256 KiB | `Disconnect` `TOO_SLOW` |
| Silence | No message for 15 s (the server pings every 2 s) | `Disconnect` `TIMED_OUT` |

The exact values live in `internal/session` and `internal/gameserver` as named constants and are covered by tests that push past each one.

## Session flow

```
client                                   server
  │ ── WebSocket upgrade (/ws) ─────────► │  origin checked, connection limit checked
  │ ── ClientHello ─────────────────────► │  protocol version, name validated
  │ ◄───────────────────── ServerHello ── │  player id, tick rate, spawn, map id
  │ ◄────────── EntitiesJoined (batch) ── │  at the next tick: everyone in the zone
  │ ◄─────── Snapshot (up to 20 Hz) ───── │  ticks where something changed, or a keepalive each second
  │ ── PlayerInput (≈20 Hz) ────────────► │  applied in seq order, one tick of movement each
  │ ◄──── Ping ─── / ─── Pong ──────────► │  heartbeats both ways
  │ ◄──── EntitiesJoined / EntitiesLeft ─ │  each tick's arrivals and departures, one batch each
  │ ◄────────────────────── Disconnect ── │  before any server-side close, with a reason
```

Before `ServerHello`, the server accepts only `ClientHello` and `Ping`. Anything else is `BAD_MESSAGE`.

A mismatched `protocol_version` gets `Disconnect` `PROTOCOL_MISMATCH` with the text "Update required", and the client shows its update screen.

When the server closes a connection, the WebSocket close code is `4000 + DisconnectReason` (for example 4003 for `TOO_SLOW`), so the reason survives even if the `Disconnect` message is lost.

## Units

- **Positions** are fixed-point integers in **subpixels**: 16 per pixel, 512 per 32-pixel tile, measured from the map's top-left corner to the entity's feet. Integers keep the server and client simulations identical (Phase 3 prediction replays the same arithmetic).
- **Ticks** count from 0 at server start, at `tick_rate` (20) per second.
- **Times** in `Ping` and `Pong` are milliseconds on the sender's clock; only differences on one clock are meaningful.
- **Ids** are `uint32`, unique among live entities. Small ids encode smaller.

## Messages and byte budgets

Sizes are the `Envelope` bytes, measured from the golden vectors in `proto/testdata/`, before WebSocket framing.

| Message | Direction | Rate | Measured size | Budget |
|---|---|---|---|---|
| `ClientHello` | client → server | once | 18 bytes (`client_hello_basic`) | ≤ 128 |
| `ServerHello` | server → client | once | 41 bytes (`server_hello_basic`) | ≤ 128 |
| `PlayerInput` | client → server | up to 20 per second | 6 bytes standing still, 10 moving (`input_still`, `input_basic`) | ≤ 16 |
| `Snapshot` | server → client | up to 20 per second; at least 1 per second | 4 bytes empty; header ≤ 14 bytes plus ≤ 16 bytes per changed entity in Phase 2 | Phase 2: unbudgeted, measured. Phase 3: ≤ 8 bytes per changed entity (scaling.md) |
| `EntitiesJoined` | server → client | at most one batch per tick (more only past 400 arrivals); a newcomer gets the whole zone | 24 bytes for one arrival, 60 for three (`entities_joined_one`, `entities_joined_three`); about 20 to 90 bytes per entity | ≤ 400 entities per message, far under 64 KiB |
| `EntitiesLeft` | server → client | at most one batch per tick | 5 bytes for one, 16 for five (`entities_left_one`, `entities_left_many`) | ≤ 4,000 ids per message |
| `Ping` / `Pong` | both | every 2 s | 11 and 18 bytes | ≤ 32 |
| `Disconnect` | server → client | once | 21 to 39 bytes | ≤ 256 |

**Arrivals and departures are batched per tick** ([ADR 0008](../decisions/0008-changed-only-snapshots.md)): the zone announces everyone who joined or left since the last tick at the start of the next one, in one `EntitiesJoined` and one `EntitiesLeft`, encoded once for everyone. A newcomer gets the whole zone (itself included, which clients ignore) in as few messages as fit. However fast players arrive, each player gets a few messages per tick, not one per arrival. Someone who joins and leaves within the same tick is never announced at all.

**Snapshots are deltas** ([ADR 0008](../decisions/0008-changed-only-snapshots.md)): a Snapshot lists only the entities whose position, facing or animation changed on that tick. An entity left out keeps the state the client last heard (ServerHello for itself, EntitiesJoined for others, or an earlier Snapshot); only `EntitiesLeft` removes one. TCP delivers every message in order, so no acknowledgements or per-client baselines are needed, and the changed entities are encoded once for everyone. A tick with nothing new for a player sends that player nothing, except a keepalive Snapshot at least once a second.

A changed entity costs 14 to 16 bytes in Phase 2 on the 30×20 starter map (id, two 3-byte zigzag positions, facing and animation, plus 2 bytes of field tag and length). Phase 3 quantizes and shrinks that encoding (an ADR) and moves the budget to scaling.md's numbers.

## Encode once, then fan out

`Snapshot.entities` is the last field, so a Snapshot can be built from parts. Each tick the zone encodes every changed entity once with `wire.AppendEntity` (field tag, length and body: one element of the repeated field). Every recipient's message is then `wire.AppendSnapshot`: an `Envelope` tag and length, that recipient's `tick` and `ack_input_seq`, and the shared entity bytes copied in. Protobuf allows a repeated field's elements to be concatenated, so the result is a normal message, and `internal/wire` tests check it is byte for byte what `proto.Marshal` would produce.

Measured on `dev-m5` (see [docs/perf/2026-11-07-bench-snapshot.md](../perf/2026-11-07-bench-snapshot.md)): 100 entities sent to 100 recipients takes about 4.1 µs with no allocations this way, against about 350 µs and 300 allocations when a Snapshot is built and marshalled per recipient, roughly 85 times less work.

## Validation

Every client message is untrusted. In order:

1. Size: over 512 bytes is rejected before parsing.
2. Decoding: malformed bytes, an empty envelope or an unknown payload field (`wire.ErrMalformed`, `ErrEmpty`, `ErrUnknownPayload`) are rejected.
3. State: the message must be allowed in the connection's state (only `ClientHello` and `Ping` before the handshake).
4. Values: `move_x` and `move_y` must each be -1, 0 or 1; `seq` must be greater than the last applied; names follow the rules above.
5. Rate: the inbound token bucket and the pending-input limit.

The decoder is fuzzed (`FuzzDecodeClient` in `internal/wire`, seeded with every golden vector): any input must either fail cleanly or survive a re-encode unchanged.

## Changing the protocol

See [`proto/README.md`](../../proto/README.md). In short: one commit carries the schema, both sides' generated code, the golden vectors, the code that uses it, and a `PROTOCOL_VERSION` bump if it breaks compatibility. While the GDScript bindings exist (until BUILD.md 2.10, when the Phaser client's TypeScript bindings replace them), there are two naming traps from godobuf: a field can't be called `message`, and a message can't share a name with a Godot class (`Input`, `Node`, `Object`, ...), which is why the input message is `PlayerInput`.
