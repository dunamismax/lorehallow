# bench snapshot: encode once is about 85× cheaper than marshalling per recipient

- **Date:** 2026-11-07
- **Commit:** the commit that adds this file (the benchmarked code, `server/internal/wire`, is in the same commit)
- **Hardware:** `dev-m5` (MacBook Pro Mac17,2, Apple M5, 10 cores (4 performance, 6 efficiency), 32 GB, macOS 27.2)
- **Go:** 1.27.3, default GOGC, GOMAXPROCS=10
- **Generator:** none (micro-benchmark)
- **Command:** `cd server && go test -run='^$' -bench=. -benchmem -count=5 ./internal/wire/`
- **Duration:** 5 runs of each benchmark, default benchtime (1 s)

## What is measured

One tick of the Phase 2 broadcast: 100 entities, each sent to 100 recipients (every player sees every player).

- `SnapshotEncodeOnce`: encode each entity once with `wire.AppendEntity`, then build each recipient's Envelope with `wire.AppendSnapshot` (its own tick and ack, the shared entity bytes copied in), into reused buffers.
- `SnapshotMarshalPerRecipient`: the naive way: build a `pb.Snapshot` per recipient and `proto.Marshal` it (deterministic), into a reused buffer.
- `DecodeInput`: decode one 10-byte `PlayerInput` Envelope with `wire.DecodeClient` into a reused `pb.Envelope`.

## Results

| Benchmark | ns/op (5 runs) | B/op | allocs/op |
|---|---|---|---|
| SnapshotEncodeOnce (100 × 100) | 4084, 4084, 4105, 4111, 4111 | 0 | 0 |
| SnapshotMarshalPerRecipient (100 × 100) | 348216, 350519, 350957, 351190, 354937 | 15200 to 15225 | 300 |
| DecodeInput | 106.9, 107.1, 107.3, 107.9, 108.3 | 72 | 2 |

No scaling.md target is met or missed by this run; it backs the "encode once" design in ADR 0005 and the allocation budget ("0 allocations per entity per tick").

## Notes

- The per-recipient work in `SnapshotEncodeOnce` is a 10 to 14 byte header plus a `memcpy` of the shared entity bytes, so at this size the cost is dominated by copying about 1.5 KB 100 times. The Phase 2 send path (BUILD.md 2.3) is designed to hand the shared body to each connection's writer goroutine to assemble its own message, so that copy runs in parallel on the writers instead of in the zone's tick.
- Decoding an input allocates twice (the oneof wrapper and the `PlayerInput`). At 1,000 players sending 20 inputs a second that is 40,000 small allocations a second, which is fine for now; a hand-written `PlayerInput` decoder with `protowire` would remove them if a profile shows it matters.
