# idle t1: 100 bots, at least one target failed

- **Date:** 2026-11-07
- **Commit:** bd830c2d63ad9f6ede02937e7e4648f4061e7ee4
- **Hardware:** `dev-m5` (MacBook Pro Mac17,2, Apple M5, 10 cores (4 performance, 6 efficiency), 32 GB, macOS 27.2)
- **Go:** go1.27.3, default GOGC and GOMEMLIMIT, GOMAXPROCS=10
- **Generator:** `lorehallow-loadtest` in-process swarm on the same machine as the server; generator CPU 16.7 s over the whole run (0.12 cores averaged over connect, warm-up and window), peak RSS 28.4 MB
- **Command:** `make loadtest SCENARIO=idle N=100 DURATION=2m0s`
- **Duration:** connect 1.0 s (ramp 1s), warm-up 20s, measured window 120.0 s
- **Map:** `content/maps/starter.tmj` (one zone; Phase 2 sends every entity to every player)

## Results

Targets from [scaling.md](../design/scaling.md) for this case and tier. "n/a" rows are recorded, not held. Tick times are bucket upper bounds (an over-estimate by at most ×1.41).

| Metric | Target | Measured | Pass |
|---|---|---|---|
| Server memory per idle connection (RSS) | ≤ 64 KiB | 135.3 KiB | no |
| Server CPU at steady state | ≤ 2% of one core | 4.6% of one core | no |
| Idle bandwidth per connection, both directions | ≤ 200 B/s | 29159 B/s | no |
| Heartbeat round trip p99, loopback | ≤ 50ms | ≤ 3.20 ms | yes |
| Zone tick p99 with everyone idle | ≤ 2ms | ≤ 0.57 ms | yes |
| Players connected at the end of the window | 100 | 100 | yes |
| Zone tick p50 | (recorded) | ≤ 0.20 ms | n/a |
| Ticks over the 50 ms budget | (recorded) | 0 of 2400 | n/a |
| Evicted as too slow | 0 | 0 | yes |
| Server goroutines above baseline | ≤ 2N = 200 | 199 (1.99 per connection) | yes |
| Server memory per connection, RSS ((mean RSS − baseline) / N) | (see below) | 135.3 KiB | n/a |
| Server heap + stacks per connection | (recorded) | 73.9 KiB | n/a |
| Server allocations per tick (whole process) | (recorded) | 426 | n/a |
| Server CPU | (see below) | 0.046 cores | n/a |
| GC CPU fraction (since start) | ≤ 5% | 0.01% | yes |
| Outbound per player (payload) | (see below) | 29150 B/s (20.5 messages/s) | n/a |
| Inbound per player (payload) | (see below) | 9 B/s | n/a |
| Back to baseline after everyone left | ≤ 10 s | 0.0 s | yes |

Server RSS: baseline 15.0 MB, mean 28.9 MB and peak 29.8 MB during the window. Generator saw 29150 B/s in and 9 B/s out per bot.

## Notes

The baseline before ADR 0008, measured from the server built at bd830c2 with the bots speaking protocol 1 (`-protocol-version 1`). Every entity went to every player every tick, so 100 idle players received 29 KB/s each. Kept as the "before" for ADR 0008.
