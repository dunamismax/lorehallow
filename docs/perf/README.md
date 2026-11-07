# docs/perf/

Performance measurements. The targets they are measured against live in [docs/design/scaling.md](../design/scaling.md).

**Only measured numbers go here.** Never estimate, round in our favor, or copy a target into a result. A result without its hardware and commit doesn't count.

## Files

- `<YYYY-MM-DD>-<scenario>-<tier>.md`: one load-test run (`2026-11-20-spread-t2.md`). `make loadtest` writes a draft; an agent checks it and fills in the notes.
- `<YYYY-MM-DD>-bench-<area>.md`: a `go test -bench` run for hot paths (`2026-11-20-bench-snapshot.md`).
- `phase-NN.md`: a phase's summary, linking the runs that back its acceptance criteria.

When a result meets or misses a tier, update the status table at the end of scaling.md in the same commit.

## Template

````markdown
# <scenario> <tier>: <one-line result>

- **Date:** YYYY-MM-DD
- **Commit:** <full hash> (`git describe`)
- **Hardware:** `dev-m5` (MacBook Pro Mac17,2, Apple M5, 10 cores, 32 GB, macOS 27.2) or `prod` (see deploy/HOST.md)
- **Go:** 1.27.3, GOGC=<..>, GOMEMLIMIT=<..>, GOMAXPROCS=<..>
- **Generator:** where `lorehallow-bot` ran, how many processes, its own peak CPU and RSS
- **Command:** `make loadtest SCENARIO=... N=... DURATION=...`
- **Duration:** warm-up and measured window

## Results

| Metric | Target | Measured | Pass |
|---|---|---|---|
| <metric from scaling.md> | <target> | <measured value> | yes or no |

## Notes

What was surprising, what limited the run, profiles taken (`go tool pprof` output summary), and what to try next.
````
