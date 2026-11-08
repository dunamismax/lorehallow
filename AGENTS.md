# AGENTS.md

The operating manual for every AI agent (and human) working on Lorehallow. Read all of it before you change anything, then read [BUILD.md](BUILD.md). When this file and an older document disagree, this file wins. When this file and BUILD.md disagree about *what* to build, BUILD.md wins; when they disagree about *how* to work, this file wins.

> **No AI attribution, ever.** Commit messages must never mention Claude, Anthropic, ChatGPT, OpenAI or any other AI tool or model. No `Co-Authored-By` trailers, no "Generated with ..." lines, no AI credit of any kind. The same goes for tags, release notes, code, comments, docs and file headers. Commits are authored by the owner's configured git identity and nothing else. See [section 5](#5-git-rules).

Contents:

1. [What Lorehallow is](#1-what-lorehallow-is)
2. [Roles and autonomy](#2-roles-and-autonomy)
3. [Session workflow](#3-session-workflow)
4. [Working with BUILD.md](#4-working-with-buildmd)
5. [Git rules](#5-git-rules)
6. [Parallel agents](#6-parallel-agents)
7. [Architecture and conventions](#7-architecture-and-conventions)
8. [Art pipeline](#8-art-pipeline)
9. [Quality gates](#9-quality-gates)
10. [Deployment and operations](#10-deployment-and-operations)
11. [Security baseline](#11-security-baseline)
12. [Documentation](#12-documentation)
13. [Toolchain versions](#13-toolchain-versions)
14. [Decision log](#14-decision-log)

---

## 1. What Lorehallow is

A pixel-art fantasy MMO built around one feeling: **a shared world.** You log in and other people are already there. It takes inspiration from early World of Warcraft (persistent world, zones, quests, grouping, a world that exists without you) rendered in a warm, readable, top-down pixel-art style.

It is a hobby project with three goals:

1. **Make a small online world that's fun to be in**, even with only a few players.
2. **Build an excellent MMO server.** The Go backend and its networking stack are the project's main achievement: fast, scalable, reliable, secure and maintainable. Players see a small, beautiful world; the infrastructure behind it is the showcase.
3. **Learn real multiplayer and backend engineering by building it.** Every system should teach something. Prefer the approach that teaches the concept clearly over a black box that hides it, and explain it in the phase walkthrough.

**When there is a real choice** of what to build next or how to build it (once the website is done, [section 3](#3-session-workflow)), favor backend and networking quality over content, quest count or gameplay depth. The world stays small and handcrafted; the server behind it is built to a professional standard.

### Scale goal

Lorehallow aims to host **hundreds to thousands of simultaneous, actively connected players**. The architecture grows from one process with many zones to several processes or machines when measurements call for it ([ADR 0005](docs/decisions/0005-scale-and-backend-architecture.md)). [docs/design/scaling.md](docs/design/scaling.md) is the performance contract: three cases (concurrent connections, active players across many zones, a crowded area), tiered targets from about 100 to 1,000 to several thousand, the metric and load test for each, and the hardware they are measured on.

**Thousands is a target, not a capability.** Never claim a player count, in code, docs, the website or a devlog, until it has been measured and recorded in `docs/perf/`.

### Design pillars

Use these to settle any design question.

1. **The world is shared.** Every feature should make other players more visible, more useful or more present.
2. **Small but alive.** One handcrafted zone that feels alive beats ten empty ones. Grow the world only after the core feels good.
3. **Readable pixel art.** Clear silhouettes, a locked palette and consistent scale. If it can't be read at a glance on screen, it's wrong.
4. **The server decides.** The server owns the truth. Clients send intents and draw what the server reports.
5. **Playable every milestone.** Every phase ends with something you can launch, play and show someone.

### Non-goals

Do not build these, and push back on anything that drifts toward them:

- Monetization, cash shops or anything pay-to-win
- 3D, isometric or multiple art styles
- A large world before one zone feels good
- Native clients of any kind: desktop or mobile apps, installers, signing, storefront builds, launchers or auto-updaters. Lorehallow is played in the browser at play.lorehallow.com ([ADR 0014](docs/decisions/0014-astro-website-and-phaser-browser-client.md))
- Browser JavaScript on the public website, or UI framework integrations (Nuxt, Vue, React, ...) in either frontend
- Game-server frameworks (Nakama or similar) that hide the server the project exists to build
- Microservices, Kubernetes, service meshes or distributed databases, until a measured bottleneck justifies one in an ADR
- Claims about scale that haven't been measured

---

## 2. Roles and autonomy

**The owner** (dunamismax) sets direction, generates the occasional large painted image in ChatGPT, and owns everything involving money, accounts, credentials, DNS and host access. Items that once needed the owner are tagged `HUMAN` in BUILD.md; the owner has pre-approved every approval and decision among them (below).

**Agents** do everything else: design, code, tests, infrastructure, deploys, **drawing the pixel art by hand**, art prompts and intake for the rare generated image, documentation, and the devlog. **Work autonomously.** The owner wants agents to make decisions and keep moving, not to ask permission for engineering choices.

### Decide it yourself

Make the call, record it, and continue when the question is about:

- Libraries, code structure, algorithms, data formats, file layout and naming
- Protocol message design, database schema and migrations
- Game tuning numbers (speeds, cooldowns, XP curves), which playtests will adjust anyway
- Tooling, CI, scripts and test strategy
- UI layout and website design within the art direction
- Adding, splitting or rewording unchecked BUILD.md items to reflect real work

Record any decision that a future session would wonder about: an ADR in `docs/decisions/` for anything significant, plus a row in the [decision log](#14-decision-log). Items tagged `ADR` in BUILD.md always need one.

### Standing pre-approval from the owner

The owner has pre-approved everything an agent needs to keep moving (2026-11-07, decision 47). Nothing waits on the owner:

- **Every decision is the agent's:** architecture, libraries, design, tuning, content, art, UI and website copy. Make the call, record it (an ADR for anything significant, plus a decision-log row) and keep going. Changing the design pillars or the non-goals, which settle every other question, still needs an ADR that says why.
- **Every `HUMAN` approval or decision in BUILD.md is already granted:** cutovers, live reviews, class design, legal text (privacy policy, terms), announcements, release dates, version tags, wipes and any other "approve" or "decide" item. Do the work, then check the item with "pre-approved by the owner, 2026-11-07" in the commit body.
- **Agents publish in the owner's voice:** devlog posts and announcements ([section 12](#12-documentation)).
- **Deploys are the agent's:** `make deploy-*` and `make rollback-*`, run by hand from a session, whenever the work calls for it.

Some `HUMAN` items need the owner physically: a multi-person playtest, a credential an agent doesn't have, or anything that costs money or needs a new account.

- Pick the option that needs no money, account or credential wherever one exists (CC0 audio, no email recovery, and so on).
- Otherwise build everything up to that one step and write the exact remaining instruction in [docs/ops/owner-todo.md](docs/ops/owner-todo.md).
- For playtests, run the strongest automated substitute available (for example headless browsers against the live server) and record how it was verified.
- Leave the box unchecked and move straight on. These items never block anything. **Never check a box that isn't really done**: the public progress tracker reads them.

### Hard limits

These are safety rules, not approvals, and nothing above loosens them:

- No AI attribution anywhere ([section 5](#5-git-rules)). Use the configured git identity and never change git config.
- `main` only, and never force-push.
- Don't disturb the other sites on the shared host. Never edit the main Caddyfile. Validate Caddy as the caddy user before every reload ([ADR 0009](docs/decisions/0009-caddy-on-a-shared-host.md)).
- Never change SSH, firewall or DNS settings in a way that could lock the owner out.
- CI never deploys and never reaches the host.
- No secrets in the repository.
- Before any destructive change to production data, take a verified backup and make the change reversible.
- If a deploy breaks something, roll it back first, then fix it.

### When you are blocked

Never sit idle. An unchecked owner-only item never blocks the next item or the next phase: write the remaining step in `docs/ops/owner-todo.md` and move on.

---

## 3. Session workflow

### Start of session

1. `git pull --rebase origin main`
2. Read this file, then BUILD.md. The **current phase** is the lowest-numbered phase with an unchecked box. **Website first:** the owner wants lorehallow.com live, complete and beautiful before the game goes much further, so Phase 1 is finished before any more work on the server phases. Until every Phase 1 box is checked, server work is limited to keeping `make check` green and the live server healthy.
   - **The two frontend migrations** ([ADR 0014](docs/decisions/0014-astro-website-and-phaser-browser-client.md)): Phase 1 rebuilds lorehallow.com in Astro, starting with the parity inventory in BUILD.md 1.2. The Go-generated site stays live until the Astro build passes its parity checks and the owner approves the cutover (1.9). Phase 2 replaces the Godot client with a Phaser browser game, starting with the Phaser scaffold in 2.6.
   - **Owner-only items never block.** Every owner approval is pre-granted ([section 2](#standing-pre-approval-from-the-owner)); an item that needs the owner physically is written up in `docs/ops/owner-todo.md` and left unchecked, and work moves on to the next item or phase.
   - **While Phase 1 waits only on an owner approval** (the cutover approval in 1.9, the live review in 1.10 or the first devlog post in 1.5), agents may work on the Phaser client in `client/`: 2.6, the client map import in 2.2, 2.7, 2.8 and the Godot retirement in 2.10. None of it changes lorehallow.com. The frontend toolchain ADR in 1.2 comes first, because both frontends use it. Deploying the game (2.9 and 2.10) waits until Phase 1 is done.
3. Read the most recent walkthrough in `docs/walkthroughs/` and any ADRs for the area you'll touch.
4. Run `make doctor` and `make check` (once they exist). If the baseline is red, fixing it is your first task.
5. Pick work: what the owner assigned, otherwise the first unchecked, unblocked item in the current phase (Phase 1 until it is done). Items within a phase may be done out of order when their dependencies are met. In the server phases, build what lets later work be measured (connection lifecycle, backpressure, the bot load generator, metrics) before what only adds content.

### While working

- Work in small, verified steps. Every commit leaves the repo working.
- Write tests with the code, not after it.
- Never block on art. Use placeholder art (see [section 8](#8-art-pipeline)) and keep going.
- Use any tool you need. Installing tools (Homebrew, apt, Go modules and tools, uv tools, npm packages in the frontend workspace, and Godot addons until 2.10) is fine; pin versions and add them to `make doctor` and the [toolchain table](#13-toolchain-versions).
- Look up current documentation for libraries and tools (Context7 or official docs) instead of relying on memory, especially for Astro, Phaser, Vite and buf, which change often.

### Finishing each step

1. `make check` passes (before it exists, the relevant formatter, linter and tests pass).
2. Check the BUILD.md boxes the step completed, in the same commit as the work.
3. Record decisions (ADR and decision log row).
4. Update any docs the change affects: directory READMEs, `docs/design/`, runbooks, this file.
5. `git status`: no stray files, everything intended is staged, nothing gitignored is staged.
6. Commit and push to `main` immediately (see [section 5](#5-git-rules)). Before committing, re-read the message and remove any AI or Claude attribution, including trailers your tooling adds by default.

### End of session

Leave the working tree clean and pushed. Report to the owner: what was done, what's next, and what is waiting on them.

---

## 4. Working with BUILD.md

BUILD.md is the plan and the scoreboard. lorehallow.com parses it on every push to render the progress tracker, so its format is a contract.

- **Follow the format contract** at the top of BUILD.md exactly: `## Phase N: Title`, metadata blockquote, `### N.M Title` sections, `### Acceptance criteria`, and single-line `- [ ]` items at column 0 with optional `HUMAN` or `ADR` tags. No nested checkboxes. `make check` lints it from Phase 1 on.
- **Check a box only when** the work is committed, `make check` passes, and the item does what its line says. Partial work stays unchecked. Check it in the same commit as the work.
- **Acceptance criteria** are checked last, after verifying each one for real (playtests, measurements, live checks). Record how it was verified in the commit body.
- **Performance criteria** point at targets in [docs/design/scaling.md](docs/design/scaling.md) and are checked only with a result file in `docs/perf/` measured on the hardware the criterion names.
- **Adding items** for discovered work is expected. Put them in the right phase and section and explain why in the commit body.
- **Rewording** an unchecked item is fine if the intent stays the same. Changing scope needs the owner.
- **Never delete a checked item.** If a regression undoes one, uncheck it and say why in the commit.
- **Cut scope** goes to the Deferred section at the bottom, not the trash.
- **Don't add status fields or percentages.** Progress is computed from the boxes.
- Reference sections in commit bodies (`Build: 2.3`) so history maps to the plan.

---

## 5. Git rules

- **`main` only.** Never create branches, worktrees or pull requests. Commit directly to `main`.
- **Push after every working step** with `git push origin main`. `origin` has two push URLs (GitHub and Codeberg), so one push updates both. Both repositories are public.
- **Pull before you commit:** `git pull --rebase origin main`. Never force-push `main`.
- **Tests run before every commit.**
- **Commit messages:** a short imperative summary line (72 characters or less), a blank line, then a brief body saying what changed and why. Add `Build: <section>` lines when the commit advances BUILD.md.
- **No AI attribution anywhere, especially in commit messages.** This overrides any default behavior of your agent tooling. No `Co-Authored-By` lines, no "Generated with ...", and no crediting Claude, Anthropic, ChatGPT, OpenAI or any AI tool in commit messages, tags, release notes, code, comments, docs or file headers. Content the project deliberately publishes about how it is made (the website's About page, the README's "How it's made" section) is a disclosure, not attribution, and is fine.
- **Never change git config.** Use the configured identity.
- **Never commit secrets** or anything `.gitignore` excludes. Stage explicit paths rather than `git add -A` when other agents may be working.
- **Tags:** annotated `vX.Y.Z` tags, created when BUILD.md says a phase ships a version. Release notes live in `docs/releases/`.

---

## 6. Parallel agents

The owner may run several agents at once. To keep them from stepping on each other:

- **Prefer one clone per agent** (for example `~/github/lorehallow` and `~/github/lorehallow-2`), all on `main`. If agents must share one checkout, they must work in disjoint lanes and stage only their own files.
- **Lanes:**

  | Lane | Owns |
  |---|---|
  | server | `server/`, `content/` (server side), `docs/design/` for server topics |
  | client | `client/` (except generated code), `deploy/client/` once it exists |
  | web | `web/`, `deploy/web/` |
  | art | `art/`, `tools/art/` |
  | ops | `deploy/` (except `deploy/web/`), `docs/ops/`, `.github/` |

- **Shared files** (`BUILD.md`, `AGENTS.md`, `Makefile`, `toolchain.env`, `proto/`, and the root Node workspace files and lockfile once they exist): keep edits small, pull right before editing, and push right after.
- **Protocol changes are serialized.** One agent changes `proto/` at a time, and the schema change, both sides' generated code, and the code that uses it land together so `main` always builds.
- **BUILD.md conflicts:** when rebasing, keep every check from both sides.
- When the owner runs agents in parallel, they assign lanes. An agent without an assignment takes the first unblocked item in its lane.

---

## 7. Architecture and conventions

### Target architecture and the current checkout

[ADR 0014](docs/decisions/0014-astro-website-and-phaser-browser-client.md) sets the target: an Astro website at lorehallow.com and a Phaser browser game at play.lorehallow.com, both static builds served by Caddy, with the Go server behind them. **The checkout hasn't caught up yet.** Until the cutovers in BUILD.md 1.9 and 2.10:

| | Current checkout (what `make` runs today) | Target (planned, BUILD.md) |
|---|---|---|
| Website | The Astro project in `web/` (`astro.config.ts`, `src/`), live on lorehallow.com since the cutover; the Go generator (`cmd/lorehallow-web`, `internal/site`) stays as the rollback target, held to the Astro output by the parity check | The Astro project alone, once the Go module is retired (1.9) |
| BUILD.md parser | TypeScript (`web/src/lib/buildmd.ts`), shared by Astro's build and `make buildmd-lint`; the Go parser in `web/internal/buildmd` stays for the Go site and the parity test | The TypeScript parser alone (1.9) |
| Game client | Godot 4 skeleton in GDScript in `client/` (title screen, golden protocol test, art scene) | Phaser, strict TypeScript and Vite in `client/`, played at play.lorehallow.com (2.6 to 2.10) |
| Client protocol bindings | GDScript from godobuf in `client/net/pb/` | TypeScript generated by buf (2.6) |
| Frontend toolchain | Node 26.13.0 and npm 11.20.0 workspaces, pinned in `toolchain.env` and `package-lock.json` ([ADR 0015](docs/decisions/0015-frontend-toolchain.md)) | The same |
| Commands | `make web-dev`, `web-build` (Astro), `web-go-dev`, `web-go-build` (Go), `web-test`, `web-lint`, `web-parity`, `buildmd-lint` (both sites); `make client`, `client-test`, `client-lint`, `client-art-check` (Godot) | `web-*` without the Go ones; `make client-dev`, `client-build`, `client-test`, `client-lint`, `client-art-check`, `deploy-client`, `rollback-client` on Phaser |

Write docs and code for what exists now, and label planned commands as planned. During each transition `make check` runs the old and the new checks side by side, and an old piece is deleted only after its replacement's checks pass.

### Repository layout

The target layout. Until 2.10, `client/` holds the Godot project (`project.godot`, `addons/`, `autoload/`, `net/pb/`, GUT `tests/`) next to the Phaser sources; until 1.9, `web/` holds the Go generator (`cmd/`, `internal/`, `templates/`, `static/`) next to the Astro project.

```
lorehallow/
├── AGENTS.md, BUILD.md, README.md, CLAUDE.md, Makefile, toolchain.env
├── package.json, lockfile # One Node workspace for web/ and client/ (planned, BUILD.md 1.2)
├── client/                # Phaser browser game: strict TypeScript, Vite (planned, BUILD.md 2.6)
│   ├── src/               # Game code by feature: net/ (net/pb/ generated), world/, entities/, ui/, scenes/
│   ├── assets/            # art/ and content/ (synced), audio/, fonts/
│   ├── index.html         # The page play.lorehallow.com serves
│   └── tests/             # Unit tests, the golden vector test, the browser smoke test
├── server/                # Go module github.com/dunamismax/lorehallow/server
│   ├── cmd/               # lorehallow-server, lorehallow-bot (load generator), lorehallow-admin, ...
│   ├── internal/          # sim/, zone/, world/, session/, wire/, bot/, metrics/, auth/, db/, pb/ (generated), ...
│   └── migrations/        # SQL migrations (Phase 4 on)
├── proto/                 # lorehallow/v1/*.proto, buf config, testdata/ golden vectors
├── content/               # Canonical game data: maps/, items/, mobs/, quests/, ...
├── web/                   # lorehallow.com: Astro, strict TypeScript, no browser JavaScript (planned, BUILD.md 1.2)
│   ├── src/               # pages/ (pages and build-time endpoints), components/, layouts/, lib/ (BUILD.md parser, progress)
│   ├── content/devlog/    # Devlog posts: Markdown, read in place by a content collection
│   ├── data/              # Release data for /play and patch notes
│   └── assets/art/        # Synced from art/processed
├── tools/                 # art/ (Python, uv), other helpers
├── art/                   # style/, batches/, inbox/, processed/
├── deploy/                # HOST.md, bootstrap/, caddy/, server/, web/, client/ (planned), ...
├── docs/                  # decisions/, walkthroughs/, design/, ops/, perf/, releases/
└── .github/workflows/     # CI and uptime checks (never deploys)
```

Generated or synced copies (`client/assets/art`, `client/assets/content`, `web/assets/art`, `server/internal/pb`, the client's protobuf bindings) are committed but never edited by hand. Edit the source and run `make sync` or `make proto`.

### Networking model

The architecture is [ADR 0005](docs/decisions/0005-scale-and-backend-architecture.md); the targets are [docs/design/scaling.md](docs/design/scaling.md).

- **Server-authoritative.** Clients send *inputs* ("moving northeast", "use ability 2 on target 41"), never positions or outcomes. The server simulates and validates everything.
- **Fixed tick rate:** 20 Hz. The server sends state snapshots every tick; distant entities may update at a lower rate under the bandwidth budget (Phase 3 on).
- **Deterministic movement:** positions are fixed-point integers and movement is one pure step function, identical on both sides and checked by shared test vectors (Phase 3).
- **Interpolation:** remote entities are drawn about 100 ms behind the latest snapshot, interpolating between buffered snapshots, with capped extrapolation for short gaps.
- **Prediction:** the local player's movement is predicted and reconciled against server corrections (Phase 3). Nothing else is predicted.
- **Interest management:** a uniform spatial grid per zone; players only receive entities in their area of interest, with hysteresis at the edge (Phase 3, extended to chat, combat and AI in Phase 7).
- **Snapshots are budgeted:** quantized deltas against the last snapshot the client acknowledged, each entity encoded once per tick and shared by every recipient, and a per-player byte budget for crowded areas (Phase 3 and Phase 7).
- **Connection lifecycle:** handshake with a timeout, heartbeats both ways, a bounded inbox into the zone, a bounded send queue out, slow-client eviction, and a clean, logged removal on every exit path. Reconnects use jittered exponential backoff.
- **Endpoints:** the game page and its static build at `https://play.lorehallow.com/`; game traffic at `wss://play.lorehallow.com/ws`; the game's HTTP APIs (auth, status) under `https://play.lorehallow.com/api/`, the same origin as the game, so no CORS is needed; a read-only `https://lorehallow.com/api/status` may proxy to the same server. Local dev at `ws://127.0.0.1:7420/ws`, reached through the client dev server's proxy (planned, BUILD.md 2.6). Metrics and pprof listen on localhost only.

### Protocol rules

- proto3, package `lorehallow.v1`, files in `proto/lorehallow/v1/`, managed with `buf` (lint and breaking-change checks).
- One `Envelope` message per binary WebSocket message, with a `oneof` payload.
- Never reuse or renumber field numbers. Mark removed fields `reserved`.
- `PROTOCOL_VERSION` is a single integer. Bump it on any incompatible change; the server rejects mismatched clients with a clear "update required" message.
- Every message type gets golden round-trip tests: Go writes vectors to `proto/testdata/`, and both Go and the client decode and re-encode them byte for byte. Today the client side is GDScript (godobuf); the TypeScript bindings replace it in BUILD.md 2.6 and must pass the same vectors before the GDScript ones are removed.
- Generated code is committed and never hand-edited. `make check` fails when it is stale.
- **A protocol change lands as one commit:** the schema, both sides' generated code, the golden vectors, the code that uses it on both sides, and a `PROTOCOL_VERSION` bump if it breaks compatibility.
- **Every message has a size budget**, written next to it in `docs/design/protocol.md`. Hot messages (`Input`, `Snapshot`) are designed for bytes: small field numbers (1 to 15 encode in one byte), packed repeated fields, `sint32` for values that can be negative, and quantized integers instead of floats.

### Server conventions (Go)

- Standard layout: `cmd/` for binaries, `internal/` for packages. Prefer the standard library; any new dependency needs a reason in the commit body.
- **The simulation is pure.** `internal/sim` does no I/O, takes an injected clock and is deterministic, so it can be tested tick by tick.
- **One goroutine owns a zone's world state**: the tick loop. Connection goroutines talk to it through channels. Don't share mutable world state behind locks.
- **Channels at the edges, bounded everywhere.** Every queue, channel, buffer and map that a client can grow has a limit, and what happens at the limit (drop and count, or disconnect) is written down and tested. No unbounded `append` driven by client input.
- **The tick never blocks.** No socket writes, database calls, DNS, disk or locks shared with other goroutines inside a tick. The tick enqueues; writers and persistence workers do the I/O.
- **Every goroutine has an owner and an exit.** Whoever starts a goroutine stops it (a `context.Context` or a closed channel) and waits for it. Tests check that goroutines return to baseline.
- **Hot paths are measured, not guessed.** The tick, snapshot encoding and fan-out, input decoding and the send path have `testing.B` benchmarks with `ReportAllocs`; allocation-free paths get a `testing.AllocsPerRun` test so a regression fails `make check`. Profile with pprof before optimizing.
- **Metrics are part of the feature.** Anything with a queue, a limit or a latency exports it (depth, drops, evictions, histograms) from Phase 3 on.
- `log/slog` for logging (text locally, JSON in production). Wrap errors with `%w`. Pass `context.Context` through I/O. Log per-connection events with the connection and player ids; never log per tick or per message at info level.
- Configuration from flags and `LOREHALLOW_*` environment variables, documented in `server/README.md` and `.env.example`.
- Tests are table-driven, run with `-race`, and never `sleep`; use the fake clock, or `testing/synctest` for code with goroutines and timers.
- `gofmt`/`goimports`, `go vet`, `golangci-lint` and `govulncheck` must be clean.

### Frontend conventions (both)

- **One Node toolchain** for `web/` and `client/`: one pinned Node LTS and one package manager, in `toolchain.env` and checked by `make doctor` (BUILD.md 1.2 ADR). Node is for development, builds and CI only. Nothing Node runs on the host: Caddy serves the built files.
- **Strict TypeScript everywhere**: `strict` on, no `any` without a comment saying why, `tsc --noEmit` clean. The linter, the formatter check and the unit tests run in `make check`.
- **Dependencies are few and justified.** Any new npm package needs a reason in its commit body, the lockfile is committed, and the package manager's audit runs in CI.
- Generated code (protobuf bindings) is excluded from lint and format checks and never edited by hand.

### Client conventions (Phaser, TypeScript)

The client in the checkout is still the Godot 4 skeleton in statically typed GDScript (`gdformat`, `gdlint` and GUT in `make check`; see [client/README.md](client/README.md)). These conventions are for the Phaser client that replaces it (BUILD.md 2.6); keep the Godot checks green until 2.10 removes them.

- **Phaser, strict TypeScript and Vite.** No UI framework. DOM UI, if any, is plain TypeScript and CSS inside the game page.
- Files and folders `snake_case` (matching the art and content ids); classes and types `PascalCase`; events in past tense (`player_joined`).
- Organize by feature (see layout above). Keep game logic out of scenes and UI; networking goes through one Net module that wraps the generated bindings.
- The client never decides outcomes. It renders server state, and predicts only its own movement.
- **Pixel rules:** base 640×360, the canvas at the largest whole-number scale that fits and letterboxed (never stretched by CSS), nearest-neighbor sampling (`pixelArt` on), no mipmaps, positions rounded to whole pixels on screen, camera snapped.
- **Tests:** unit tests for logic, the golden vector test for the protocol, a pixel check of the dev art scene in a headless browser, and a browser smoke test that runs two sessions against the real Go server.
- **Browser only.** No native wrappers, installers or update checks; a new version is a new static build.

### Web conventions (lorehallow.com)

The site in the checkout and live today is built by the Go generator in `web/` ([ADR 0011](docs/decisions/0011-go-website.md), [web/README.md](web/README.md)): `html/template`, goldmark and Go output tests, with gofmt, vet, golangci-lint, govulncheck and `go test -race` in `make check`. Keep it working and live until the Astro cutover (BUILD.md 1.9). These conventions are for the Astro site that replaces it; the rules about output (no scripts, static, palette, self-hosted, pixel art) already apply to both.

- **Astro, used directly** ([ADR 0014](docs/decisions/0014-astro-website-and-phaser-browser-client.md)): Astro components, strict TypeScript, CSS and Markdown content collections, built with Astro's static build and its built-in Vite. No Nuxt, Vue, React or other UI framework integrations, and no client islands.
- **No JavaScript on any page.** No `<script>` tags, inline handlers or third-party scripts. Keep the Astro features that inject scripts off (prefetch, view transitions, islands) and stylesheets un-inlined (the CSP is `style-src 'self'`). Caddy's CSP refuses scripts, and an output test fails the build if one appears. Motion (parallax, fireflies, lantern flicker) is CSS only and stops under `prefers-reduced-motion`. Anything that seems to need a script is built at build time, done with plain HTML forms and links, or written up as an ADR first.
- **Static output.** Everything dynamic (progress, the achievement feed, release data, feeds, sitemap, `/progress.json`, `/progress.svg`) is computed at build time from BUILD.md, git history and files in `web/content/` and `web/data/`; with Astro, generated files are build-time endpoints. `make deploy-web` rebuilds and publishes it, run by hand.
- **Stable URLs.** Every URL in the parity inventory (`docs/design/website-parity.md`, BUILD.md 1.2) keeps working. A page that moves gets a permanent redirect in the Caddy site file.
- **Drafts** (`draft: true`) appear in the local preview only, never in any public artifact.
- **Shared palette.** Colors come from tokens generated from `art/style/palette.hex`, so the site and the game share one palette.
- **Self-hosted.** No third-party requests: no trackers, CDNs or remotely loaded fonts.
- **Pixel art** is shown at integer scales with `image-rendering: pixelated` and explicit dimensions read from the image. Never let an image pipeline resample or re-encode pixel art.
- **The BUILD.md parser** implements the format contract. Today it is `web/internal/buildmd/` (Go); after BUILD.md 1.4 it is one TypeScript module shared by the Astro build and `make buildmd-lint`. Change the parser, its tests and the contract together.

### Content conventions

- Game data lives in `content/` as YAML, plus Tiled JSON (`.tmj`) for maps unless an ADR says otherwise. IDs are `snake_case`, stable and never reused.
- Every content type has a validator that runs in `make check`: references resolve, quests are completable, numbers are in range.

### Versions

- Game builds are versioned `v0.<phase>.<patch>` until 1.0 (Phase 2 ships v0.2.0, Phase 18 ships v0.18.0), and Phase 19 ships v1.0.0. Post-1.0 phases (Phase 20 on) ship `v1.<minor>.<patch>`.
- The version and git commit are embedded in both the server and client builds and shown in the client title screen and `/version`.

---

## 8. Art pipeline

### Locked art direction

| Rule | Value |
|---|---|
| Tiles | 32×32 px |
| Characters | 32×48 px frames, 4 directions (down, up, left, right; right mirrored from left where possible) |
| Animations | idle 2 frames, walk 4 frames; attack, cast, hurt and death added later |
| Palette | one fixed palette of 33 colors in `art/style/palette.hex`, referred to by name in every source |
| Perspective | top-down 3/4 view |
| Light | from the top-left, always |
| Rendering | integer scaling only, no anti-aliasing, no sub-pixel blur |
| Base viewport | 640×360 |
| Mood | warm storybook fantasy: mossy forests, lantern-lit towns, old stone ruins |
| Key color | `#FF00FF` flat background for keying (never used in the palette) |

`art/style/STYLE.md` (written in Phase 0) holds the full style guide, and `art/style/reference.png` is the reference sheet: the look every new asset must match.

### Where art comes from

**Agents hand-draw the pixel art.** This is the standing rule ([ADR 0004](docs/decisions/0004-hand-drawn-pixel-art.md)): tiles, characters, animations, mobs, NPCs, items, icons, UI, effects, the logo and small web art are drawn pixel by pixel by the agent, as text sources in the repo. Placing every pixel deliberately gives exact sizes, exact palette, exact baselines and seamless edges, which image generators can't.

Image generation is the exception, kept only for **large painterly images** where hand placement isn't practical and exact pixels don't matter: website key art, parallax backgrounds, banners and social cards. Say why in the asset's manifest notes. Those go through prompts and intake (below).

### Batches

Art is produced in numbered batches, each tied to the feature that needs it. Batch numbers follow the order BUILD.md lists them (01 style foundation, 02 brand and website art, 03 player character, and so on). Each batch has a folder:

```
art/batches/batch-03-player/
├── manifest.yaml        # Machine-readable spec for every expected asset (source: for hand-drawn ones)
├── report.md            # Every asset's status and problems (written by make art-draw)
├── preview.png          # Contact sheet at 4x, plus preview-8x.png for hand-drawn art (make art-draw)
├── intake-report.md     # Only for generated assets: intake results (make art-intake)
├── prompts.template.md  # Only for generated assets: source of the prompts
├── prompts.md           # Generated by `make art-generate` from the template
├── fixes.py             # Optional: scripted cleanups applied on every intake run
└── guides/              # Layout guides to attach to prompts (generated from the manifest)
```

Hand-drawn sources live in `art/src/<category>/`, mirroring `art/processed/<category>/`.

### Drawing by hand (agents, the default)

Two source formats, both rendered by `make art-draw` into `art/processed/` (formats in [ADR 0004](docs/decisions/0004-hand-drawn-pixel-art.md), helpers and checks in [tools/art/README.md](tools/art/README.md)). A manifest asset points at its source with `source: <category>/<name>.pxl.yaml` (or `.py`), relative to `art/src/`; intake skips it.

- **Pixel grids**, `art/src/<category>/<name>.pxl.yaml`: a legend of one character per palette color (by palette name) and one text grid per frame. Use for sprites, faces, letters, icons, item art and anything where each pixel is a decision.
- **Draw scripts**, `art/src/<category>/<name>.py`: `draw(ctx)` using the `tools/art` drawing helpers, with seeded randomness only. Use for textured ground, seamless tiles and their variants, mirrored directions, palette swaps and composites built from finished parts.

How to draw well:

1. Read STYLE.md and look at `art/style/reference.png` and every finished asset the new one will sit next to.
2. Block in the silhouette first, then the 1 px ink outline, then 3 or 4 flat shading steps per material with light from the top-left, then a few highlight pixels. Readable at 1x beats detailed at 8x.
3. Reuse: build sheets from shared parts (one head, one body, mirrored directions), so a character looks the same in every frame.
4. Run `make art-draw BATCH=NN REVIEW=<dir outside the repo>` and **look at the result**: the contact sheets (`preview.png` at 4x, `preview-8x.png` at 8x, each with 1x copies on dark and light), the review images (tiles repeated 3×3 to show seams, each asset at 1x and 3x on dark and light), and animations frame by frame. Fix what looks wrong and repeat. Don't accept art you haven't looked at. `make check` fails when a processed file no longer matches its source, so always re-run `make art-draw` after editing one.
5. `make sync`, then check it in the dev art scene and in real use.

### Generated art (the exception)

For the large painted images above only.

Edit `prompts.template.md`, never `prompts.md`: generating the prompts from the template keeps the palette in every prompt in step with `art/style/palette.hex`, and `make check` fails if they drift.

#### Writing prompts

ChatGPT image generation can produce good painterly pixel-art scenes, but it doesn't follow exact sizes, palettes or layouts. Every prompt states:

- **Exact logical size** (a 32×32 tile, a 32×48 frame) and a request for "true pixel art: every art pixel a solid square block of one color, no anti-aliasing, no gradients, no blur, no texture noise"
- **Palette:** the full Lorehallow hex list, "use only these colors"
- **Perspective and light:** top-down 3/4 view, light from the top-left
- **Layout:** single sprite, tileset grid, or sprite sheet with N columns and M rows, the frame order (reading order, which row is which direction), equal cells, and the same foot baseline in every cell
- **Background:** flat solid `#FF00FF` everywhere that isn't the sprite
- **Target asset id and filename**, so intake knows what each image is meant to be
- **"Attach `art/style/reference.png`"** as an instruction to the owner, once the reference sheet exists

Use this shape in `prompts.md`:

````markdown
## 03-01 player_base: walk and idle sheet

- Target: art/processed/chars/player_base.png
- Logical size: 6 columns × 4 rows of 32×48 frames (192×192 total)
- Attach: art/style/reference.png

```text
<the full prompt, ready to paste>
```
````

#### The owner's part

Paste each prompt into ChatGPT in order (attaching the reference sheet when told to), download the results, and drop them **unchanged** into `art/inbox/batch-NN/`. No renaming or editing needed. An agent with image generation of its own may do this step instead.

#### Intake (agents)

1. Look at every image in the inbox folder and match it to its manifest entry. Write the mapping to `art/inbox/batch-NN/map.yaml`.
2. Run `make art-intake BATCH=NN`. It detects the real pixel grid, downsamples to the logical size, removes the key color, quantizes to the palette, slices sheets, checks frame counts and alignment, and writes `art/processed/` plus `intake-report.md`.
3. Inspect the results at 4x. Painted full-bleed scenes usually need `grid: fixed` in `map.yaml` (see `tools/art/README.md`). Small fixes (stray pixels, nudging a frame onto the baseline) go in a per-batch fix script in the batch folder so re-running intake reproduces them. Never edit processed files by hand without a script.
4. Anything that can't be fixed gets a corrected prompt in `intake-report.md`. The owner regenerates it into the same inbox folder.
5. `make sync`, then test in-game in the dev art scene and in real use. Check for blur, seams, palette violations and alignment at 1x to 4x.

**`art/inbox/` is never edited.** It stays as generated so intake can always be re-run.

### Naming and import

- Processed paths: `art/processed/<category>/<name>.png`, categories `tiles`, `chars`, `mobs`, `npcs`, `props` (trees, rocks, posts, buildings), `items`, `ui`, `vfx`, `icons`, `web`. Names are `snake_case` (`tiles/terrain_grass.png`, `chars/player_base.png`). Sheets get a sidecar `.json` with frame size, frame order, pivot and source.
- The client loads art as plain PNGs with nearest-neighbor sampling and no mipmaps (Phaser's `pixelArt` mode); the website shows it with `image-rendering: pixelated`. Nothing resamples or re-encodes processed art. (The Godot skeleton imports with its pixel-art preset, filter off, mipmaps off and lossless, until BUILD.md 2.10.)

### Placeholders

Code never waits on art. Drawing is fast, so draw the real asset when you can; otherwise use placeholder sprites (flat-colored shapes or simple generated sprites at the correct size) named with a `_placeholder` suffix. `make art-placeholders` writes one for every manifest asset that has no processed output yet (with sidecars, so sheets keep their frame layout and baseline) and removes each one when the real asset lands. Real art replaces them within the same phase, and Phase 14 removes every remaining placeholder.

---

## 9. Quality gates

`make check` is the single gate, run locally before every commit and in CI on every push. CI runs on **GitHub Actions** (`.github/workflows/ci.yml`, Linux, same pinned versions from `toolchain.env`). **Codeberg is a mirror**: it receives every push but runs no CI. It covers:

- **Server:** `gofmt` check, `go vet`, `golangci-lint`, `go test -race ./...` (which also runs every fuzz target's seed corpus and every allocation test), `govulncheck`
- **Protocol:** `buf lint`, `buf breaking`, generated code up to date, golden round-trip tests on both sides
- **Client:** today `gdformat --check`, `gdlint` and GUT tests headless (Godot). Planned (BUILD.md 2.6): `tsc --noEmit`, the linter and formatter check, unit tests, the golden vector test, the pixel check of the dev art scene and the two-session browser smoke test against the real Go server. Both sets run until 2.10 removes Godot.
- **Web:** today the Go generator's `gofmt` check, `go vet`, `golangci-lint`, `govulncheck` and `go test -race` (including its output checks: no scripts, page weight, alt text, image sizes, internal links). Planned (BUILD.md 1.2 to 1.9): `astro check`, `tsc`, the linter and formatter check, the TypeScript parser and its parity test against the Go parser, the output tests ported to TypeScript, the parity check between the two builds, and the package manager's audit. Both sets run until 1.9 retires the Go module.
- **Frontend dependencies:** the package manager's audit on the Node workspace, once it exists
- **Tools:** `ruff`, `pytest`
- **Content and docs:** content validators, BUILD.md lint

Outside `make check`, because they take longer:

- `make fuzz`: every fuzz target for a short run (CI runs it on every push; a nightly job runs longer).
- `make bench`: hot-path benchmarks with allocations reported. Results that back a claim go in `docs/perf/`.
- `make loadtest SCENARIO=<name>`: the scaling.md scenarios against a real server.

Testing expectations:

- Server game logic gets real unit tests, driven tick by tick with the fake clock.
- Every protocol change gets round-trip tests.
- Anything security-related (auth, rate limits, validation, cheating) gets a test that proves the attack fails.
- Anything touching persistence gets tests against a real Postgres (CI service container), including failure in the middle of an operation.
- Bugs get a regression test before the fix.

### Backend engineering standards

Code that compiles isn't done. The server is the showcase, so each of these needs a test that tries to break it, not just one that shows it working:

| Concern | What the test does | Tool |
|---|---|---|
| Concurrency | Runs the real goroutines together and checks the result is the same every time | `-race`, `testing/synctest` |
| Timing | Drives ticks, timeouts and heartbeats by advancing time, never by waiting for it | the fake clock or `testing/synctest`; no `time.Sleep` in tests |
| Leaks | Starts and stops connections, zones and servers, then checks goroutines are back to baseline | `testing/synctest` bubbles, or the leak helper in `internal/testutil` |
| Backpressure | Fills every bounded queue and checks the limit holds, the drop is counted and the tick doesn't wait | slow and stalled fake clients |
| Authorization | Sends each message from a client that isn't allowed to (wrong state, wrong owner, not logged in) | attack tests |
| Validation | Sends malformed, oversized, out-of-range and unknown input to the decoder and every handler | fuzzing (`go test -fuzz`) with a committed seed corpus |
| Duplicates and replays | Sends the same command twice, out of order and after a reconnect | attack tests with request ids |
| Transactions | Kills the operation between steps and checks nothing was lost or duplicated | real Postgres, failure injected mid-transaction |
| Recovery | Crashes or restarts the component and checks players end up in a safe, recoverable state | integration tests with the bot client |
| Performance | Measures the hot path and fails on allocation regressions | `testing.B` with `ReportAllocs`; `testing.AllocsPerRun` in tests; pprof |

A change to the server's hot path or concurrency model includes its benchmark or load-test result in the commit body, measured on named hardware ([scaling.md](docs/design/scaling.md#hardware-and-how-results-count)).

**Definition of done** for any BUILD.md item: it works, it's tested (with the tests above where they apply), `make check` is green, docs are updated, and it's pushed.

---

## 10. Deployment and operations

### The host

The owner's own server sits behind lorehallow.com. Its details (OS, architecture, resources, network, what else runs there) live in `deploy/HOST.md`, which the owner fills in during Phase 1. Until it says otherwise, assume Linux with systemd. If something conflicts with this section (for example an existing reverse proxy), adapt and record an ADR.

- **Domains:** `lorehallow.com` (the public website, a static build, plus a read-only `/api/status`), `www.lorehallow.com` (redirects to the apex), `play.lorehallow.com` (the game: today the WebSocket and health routes only; planned in BUILD.md 2.9, the game's static build too, with `/ws`, `/healthz`, `/version` and `/api/` proxied to the Go server).
- **Reverse proxy:** Caddy, with automatic HTTPS. Config lives in `deploy/caddy/`. Caddy serves the two static builds itself; only the Go server runs as a Lorehallow process.
- **Layout on the host:** releases in `/srv/lorehallow/<service>/releases/<timestamp>/` with a `current` symlink switched atomically, for `server`, `web` and (planned) `client`; secrets in `/etc/lorehallow/<service>.env` (mode 0640); services run as dedicated users under systemd units from `deploy/`.
- **No Node on the host.** Both frontends are built on the machine running the deploy, and only their output is uploaded.
- **Access:** agents connect as the `deploy` user over SSH with the key the owner provides. Privileged setup goes through idempotent scripts in `deploy/bootstrap/` that the owner reviews and runs (or temporarily lets an agent run).

### What agents may do in production

- Deploy with the `make deploy-*` targets after `make check` passes, and roll back with `make rollback-*`. **Deploys are always run by hand from a session. Never deploy from GitHub Actions or any other CI, and never give a CI job access to the host** (the owner's standing rule).
- Read logs, check status, and restart services.
- Run forward migrations that are part of a deploy.

### What agents never do in production without the owner

- Run destructive SQL, wipe data or delete backups
- Change SSH, firewall or DNS settings
- Expose admin, metrics or database ports to the internet
- Put secrets anywhere except GitHub Actions secrets and the host's env files

Runbooks live in `docs/ops/`. Every new operational procedure gets written down there.

---

## 11. Security baseline

- TLS everywhere in production (`wss://` and `https://` only).
- Passwords hashed with Argon2id; session tokens random, stored only as hashes, and expiring.
- Every client message is untrusted: validate type, size, rate and authority on the server.
- Limits on everything: connections per IP, messages per second, message size, name and chat length.
- Secrets never in the repo. `.env.example` documents variable names only.
- Admin and metrics endpoints listen on localhost only, reached through SSH tunnels.
- Dependencies scanned in CI: `govulncheck` on the Go modules, and the package manager's audit on the Node workspace once it exists.
- **Two Content-Security-Policies**, one per site block in the Caddy file. lorehallow.com: `script-src 'none'`, nothing third-party. play.lorehallow.com: scripts only from `'self'`, no inline scripts and no `unsafe-eval`, connections only to the game's own origin and `wss://play.lorehallow.com`. Never loosen one to suit the other.
- Automated, encrypted, off-site Postgres backups from Phase 4 on, with restores actually tested.

---

## 12. Documentation

- **Decisions:** `docs/decisions/NNNN-short-title.md` (four-digit numbers), with sections Context, Decision, Alternatives considered, Consequences. Add a row to the decision log below.
- **Walkthroughs:** every phase ends with `docs/walkthroughs/phase-NN-slug.md`, written to teach. Sections: What we built; How to try it; How it works; The key concept (the networking or backend lesson, with diagrams where they help); Read the code (specific files and functions, in reading order); What's next.
- **Design notes:** `docs/design/` for the scaling model ([scaling.md](docs/design/scaling.md)), the protocol, movement, combat, classes, lore and the threat model.
- **Operations:** `docs/ops/` for runbooks, backups and incidents.
- **Performance:** `docs/perf/` for measured results only, one file per run, with hardware and commit ([template](docs/perf/README.md)).
- **Releases:** `docs/releases/vX.Y.Z.md`, which become patch notes on the website.
- **Devlog:** posts in `web/content/devlog/` ([format](web/content/devlog/README.md)). **Agents write and publish them** in the owner's voice: first person, plain, warm and specific, 200 to 500 words, at most one per milestone (a phase shipping, the website cutover, the game first playable in the browser, or another real milestone). Every claim comes from real commits, measurements and screenshots, and no player count or performance figure appears unless it is recorded in `docs/perf/`. To publish: set `draft: false`, run `make check`, commit, and run `make deploy-web`. A post with `draft: true` shows only in `make web-dev`.
- **Directory READMEs:** every top-level directory has a short README saying what belongs there and how to work in it.
- **Style:** plain language and short sentences. The landing page can sell; docs explain.

---

## 13. Toolchain versions

Pinned in Phase 0 (BUILD.md 0.3) to the latest stable releases at that time ([ADR 0002](docs/decisions/0002-toolchain-versions.md)). The frontend toolchain is [ADR 0015](docs/decisions/0015-frontend-toolchain.md): Node and npm are in `toolchain.env`, and every npm package is pinned in `package-lock.json`. Rows marked _pin in_ are still planned; the Godot rows stay until BUILD.md 2.10 removes them. The machine-readable source is [`toolchain.env`](toolchain.env); `make doctor` checks against it. Update `toolchain.env` and this table in the same commit as any version bump.

| Tool | Version | Notes |
|---|---|---|
| Go | 1.27.3 | |
| Godot | 4.7.2-stable | Standard build. Removed with the Godot client in BUILD.md 2.10 |
| Python | 3.14.9 | Managed by uv |
| uv | 0.13.7 | |
| buf | 1.75.0 | |
| golangci-lint | 2.15.0 | |
| gdtoolkit | 4.5.0 | gdformat and gdlint, via uv. Removed in 2.10 |
| GUT | 9.7.1 | Godot addon, installed by `make setup`. Removed in 2.10 |
| godobuf | 0.7.0 | Godot addon for GDScript protobuf codegen, installed by `make setup`. Removed in 2.10 |
| protoc-gen-go | 1.36.12 | `tool` directive in `server/go.mod` |
| govulncheck | 1.9.0 | `tool` directive in `server/go.mod` |
| Node.js (LTS) | 26.13.0 | Frontend development, builds and CI only; never on the host ([ADR 0015](docs/decisions/0015-frontend-toolchain.md)) |
| npm | 11.20.0 | Ships with Node 26.13.0. One workspace for both frontends, one committed `package-lock.json` |
| TypeScript | 6.0.3 | In the lockfile. Not 7 yet: `@astrojs/check` supports 5 and 6 only |
| Biome | 2.6.6 | Lint and format for both frontends, in the lockfile |
| Tests | `node --test` | Built into Node; TypeScript run by Node's type stripping |
| Astro | 7.4.7 | `web/`, with its built-in Vite; `@astrojs/check` 0.9.12 |
| Phaser, Vite | _pin in 2.6_ | `client/` |
| TypeScript protobuf plugin | _pin in 2.6_ | Run by buf from `make proto` |
| Headless browser for tests | _pin in 2.6_ | Art check and browser smoke test |
| Caddy | _pin in 1.1_ | On the host |
| PostgreSQL | _pin in 4.1_ | On the host and in CI |

---

## 14. Decision log

One row per decision. "Default" means the choice stands unless the ADR written for that BUILD.md item finds a good reason to change it.

| # | Decision | Status | Why | Record |
|---|---|---|---|---|
| 1 | Client in Godot 4 with GDScript | Superseded by 44 | Fast iteration, excellent 2D, simple export to Mac, Windows and Linux | Project brief |
| 2 | Game server in Go | Decided | Built for network servers, safe concurrency, fast builds, single-binary deploys | Project brief |
| 3 | Transport is WebSocket over TLS | Decided | Reliable and ordered like WoW's TCP model; works in the browser, where the game is now played | Project brief |
| 4 | Protocol is Protobuf with schemas in `proto/`, managed by buf | Decided | One versioned source of truth for every message | Project brief |
| 5 | PostgreSQL for persistence, from Phase 4 | Decided | Characters, inventory and world state need real transactions | Project brief |
| 6 | Self-hosted on the owner's server behind lorehallow.com | Decided | Learn deployment, TLS, supervision, logs and backups for real | Project brief |
| 7 | Server-authoritative, 20 Hz tick, ~100 ms interpolation, prediction for own movement only | Decided | Cheat resistance and a clear model to learn from | Project brief |
| 8 | 32×32 tiles, 32×48 characters, ~32-color palette, 640×360 base viewport with integer scaling | Decided | Readable pixel art at 2x, 3x and 4x on common displays | Project brief, this file |
| 9 | Website before gameplay: phases are 0 scaffold, 1 website, 2+ game | Decided | Public, followable progress from day one | Owner |
| 10 | Website is an Astro static site; BUILD.md parsed at build time | Superseded by 39 | Static output is fast and simple to self-host; Markdown-native | BUILD.md 1.2 |
| 11 | Website deploys run by hand with `make deploy-web`; no CI job ever deploys or reaches the host | Decided | The owner's rule: nothing in CI touches the server | Owner |
| 12 | Caddy as reverse proxy with automatic HTTPS: Lorehallow is one site file on the owner's shared Caddy; never touch the main Caddyfile, other sites, firewall or SSH; validate as the caddy user before every reload | Decided | The host already runs Caddy for other sites | [ADR 0009](docs/decisions/0009-caddy-on-a-shared-host.md) |
| 13 | GNU Make as the single command runner | Default | Already installed on macOS and Linux; one entry point for humans, agents and CI | BUILD.md 0.3 |
| 14 | Art tooling in Python managed by uv (Pillow, numpy) | Default | Best image-processing ecosystem; uv keeps it reproducible | BUILD.md 0.7 |
| 15 | GUT for Godot tests; gdtoolkit for GDScript format and lint | Superseded by 44 | Mature, headless-friendly | BUILD.md 0.6 |
| 16 | GDScript protobuf via godobuf 0.7.0; custom protoc plugin only if a golden test fails | Superseded by 44 | Passed every golden round-trip vector on Godot 4.7 | [ADR 0003](docs/decisions/0003-gdscript-protobuf.md) |
| 17 | Game traffic on `play.lorehallow.com`, HTTP APIs under `lorehallow.com/api/` | Superseded by 46 | Separates game traffic from the website; easier to move later | This file |
| 18 | Game versions `v0.<phase>.<patch>` until 1.0 | Decided | Version numbers map to BUILD.md phases | This file |
| 19 | `main`-only git, commit and push after every verified step, no AI attribution | Decided | Owner's rules | Owner |
| 20 | Stack decisions recorded in one ADR | Decided | One place for later ADRs to reference | [ADR 0001](docs/decisions/0001-stack.md) |
| 21 | Toolchain pinned in `toolchain.env`; doctor accepts newer patch releases | Decided | One source for Make, doctor and CI | [ADR 0002](docs/decisions/0002-toolchain-versions.md) |
| 22 | `PROTOCOL_VERSION` lives in `proto/PROTOCOL_VERSION`; `buf breaking` failures pass only with a bump | Decided | One integer, generated into both languages; "planned" breaks are explicit | [proto/README.md](proto/README.md) |
| 23 | Client uses the Compatibility renderer; untyped GDScript is an error, not a warning | Superseded by 44 | Widest hardware support for 2D; headless runs don't print warnings, so only an error is enforceable in CI | [client/README.md](client/README.md) |
| 24 | Local dev server listens on 127.0.0.1:7420 | Default | Unprivileged, unlikely to clash | [server/README.md](server/README.md) |
| 25 | Code under AGPL-3.0-only; art and writing under CC BY-NC 4.0 | Decided | Forks must share server changes; art and lore can be shared and remixed but not sold | Owner, [README.md](README.md#license) |
| 26 | Palette v2: earth, stone and leaf ramps fitted to what the image generator draws, plus sand (33 colors) | Decided | v1 had gaps that sent browns to red and skin and grey stone to pink during quantization | [STYLE.md](art/style/STYLE.md#why-v2) |
| 27 | Agents hand-draw the pixel art as text sources (`.pxl.yaml` grids and draw scripts); image generation only for large painted web images | Decided | Exact size, palette, layout and seamless edges; generated art missed all four in Phase 0 | [ADR 0004](docs/decisions/0004-hand-drawn-pixel-art.md) |
| 28 | `make art-draw` renders `source:` assets and owns each batch's `report.md` and previews (intake writes `intake-report.md`); `make check` re-renders every source and fails on any drift; world objects live in a `props` category; `make client-art-check` verifies rendering in the client pixel by pixel (in Godot today; in a headless browser after BUILD.md 2.6) | Decided | One report per batch covering every asset; processed art can never silently diverge from its source; rendering checked by measurement, not by eye | [tools/art/README.md](tools/art/README.md) |
| 29 | Scale goal: hundreds to thousands of simultaneous, active players, measured against tiered targets in `docs/design/scaling.md`; thousands is a target, not a capability | Decided | The owner made the Go backend the project's main achievement | [ADR 0005](docs/decisions/0005-scale-and-backend-architecture.md) |
| 30 | One process with many zones: one goroutine owns each zone, channels at the edges, bounded send queues with slow-client eviction, encode once then fan out, spatial-grid interest management; split across processes (Phase 20) only when a measurement says so | Decided | Patterns that hold up at thousands of connections, built in before the first tick loop | [ADR 0005](docs/decisions/0005-scale-and-backend-architecture.md) |
| 31 | Phase 1 runs in parallel with the server phases; backend work never waits for the website | Superseded by 38 | Phase 1 was mostly blocked on the owner's host setup | Owner, [section 3](#3-session-workflow) |
| 32 | The BUILD.md parser and linter are plain TypeScript run by Node's type stripping, with `node --test` | Superseded by 39 | No build step or test framework for a parser the site and `make check` share | [web/README.md](web/README.md) |
| 33 | Maps are Tiled JSON in `content/maps/` with external tilesets named after processed art; a `collision` tile layer and a `spawns` object layer; validated by `lorehallow-content` in `make check` | Decided | One file both sides read, editable in a real editor | [ADR 0006](docs/decisions/0006-tiled-map-format.md) |
| 34 | coder/websocket for the transport: two goroutines per connection (handler reads, one writer), no per-read timeouts (heartbeats and eviction close stuck connections), close code 4000 + reason | Decided | No goroutines of its own, reuses net/http buffers, zero dependencies | [ADR 0007](docs/decisions/0007-websocket-library.md) |
| 35 | Snapshots carry only changed entities, quiet players get a keepalive each second, and arrivals and departures are batched per tick (`EntitiesJoined`, `EntitiesLeft`); no per-client baselines because TCP is reliable and ordered; `PROTOCOL_VERSION` 2 | Decided | Measured: unchanged state cost 29 KB/s per idle player, and per-arrival messages evicted newcomers past 256 players | [ADR 0008](docs/decisions/0008-changed-only-snapshots.md) |
| 36 | Each session runs on its own goroutine after the upgrade (the HTTP handler returns), and the hijacked connection gets 1 KiB read and 2 KiB write buffers instead of net/http's 4 KiB each | Decided | Measured: net/http's buffers and retained request were 12 of 18.6 KB live per idle connection | [ADR 0010](docs/decisions/0010-per-connection-memory.md) |
| 37 | Case 1 memory per connection is the marginal cost, (RSS at N − RSS at N/2) / (N/2), with the fixed runtime overhead and the old metric still recorded in every result | Decided | The old metric spread Go's fixed 2 to 3 MB of minimum-heap slack over N, about 30 KiB per connection at 100 | Owner, [ADR 0010](docs/decisions/0010-per-connection-memory.md), [scaling.md](docs/design/scaling.md) |
| 38 | Website first: Phase 1 is finished, live and polished before more server-phase work; the server phases resume after it | Decided | The owner wants a live, beautiful lorehallow.com before the game goes much further; the host, DNS and Caddy are already set up | Owner, [section 3](#3-session-workflow) |
| 39 | The website is a Go static site generator in `web/` (its own module: `html/template`, goldmark, `go:embed`) with no JavaScript on any page; the BUILD.md parser and linter are Go; Node, npm and TypeScript leave the toolchain | Superseded by 43 | The owner is moving every project to Go; the site is static and needs no Node or browser scripts | Owner, [ADR 0011](docs/decisions/0011-go-website.md) |
| 40 | Website fonts: a hand-drawn pixel font for headings, compiled to TrueType and WOFF by the generator, and the Go fonts (BSD) for body text, all self-hosted with licenses committed; the fonts carry over to Astro, and where the compiler runs is decided in BUILD.md 1.2 | Decided | Matches the hand-drawn art, pixel-exact at 2x to 4x, no downloads outside the Go module proxy | [ADR 0012](docs/decisions/0012-website-fonts.md) |
| 41 | Website deploys: timestamped release directories under `/srv/lorehallow/web/releases/`, an atomic `current` symlink switch, a health check through the host's own Caddy with automatic rollback, five releases kept; run by hand with `make deploy-web` | Decided | Same model as the server deploy; a bad release can't stay live | [docs/ops/website.md](docs/ops/website.md) |
| 42 | Uptime monitoring: a GitHub Actions check every five minutes from outside the home network; one ntfy alert when down for five minutes and one on recovery, plus GitHub's failed-run email | Decided | No account needed, sees home-network outages that a check on the host can't | [ADR 0013](docs/decisions/0013-uptime-monitoring.md) |
| 43 | The website is Astro in `web/`: Astro components, strict TypeScript, CSS and Markdown content collections, static build with Astro's built-in Vite, no UI framework integrations and no browser JavaScript; one TypeScript BUILD.md parser shared by the build and the linter, checked against the Go parser; JSON, SVG and feeds as build-time endpoints; the Go site stays live until parity checks pass and the owner approves the cutover | Decided, not yet built | The owner keeps Go for the backend and its tools; production still serves static files, so no Node runs on the host | Owner, [ADR 0014](docs/decisions/0014-astro-website-and-phaser-browser-client.md) |
| 44 | The game is a Phaser browser client in `client/` (strict TypeScript, Vite) at play.lorehallow.com, with TypeScript protobuf bindings checked against the golden vectors; native desktop clients and their whole distribution pipeline are retired; Godot is removed once equivalent checks cover it | Decided, not yet built | Players open a URL to play; one build reaches every desktop platform | Owner, [ADR 0014](docs/decisions/0014-astro-website-and-phaser-browser-client.md) |
| 45 | One pinned Node LTS and one package manager for both frontends, for development, builds and CI only | Decided | One toolchain, one lockfile, nothing new on the host | [ADR 0014](docs/decisions/0014-astro-website-and-phaser-browser-client.md) |
| 46 | Caddy serves two independent static builds (lorehallow.com and play.lorehallow.com) and proxies the game's `/ws`, `/healthz`, `/version` and `/api/` routes on play.lorehallow.com to the Go server; each site has its own CSP (the website refuses all scripts); deploys stay manual with atomic releases and rollback | Decided | Same-origin APIs need no CORS; one site's needs never weaken the other's policy | [ADR 0014](docs/decisions/0014-astro-website-and-phaser-browser-client.md) |
| 47 | The owner pre-approves every decision, every `HUMAN` approval, devlog publishing and deploys; items that need the owner physically go to `docs/ops/owner-todo.md` and never block; only the hard limits in section 2 remain | Decided | The owner wants agents moving through BUILD.md without waiting | Owner, [section 2](#standing-pre-approval-from-the-owner) |
| 48 | Frontend toolchain: Node 26.13.0 LTS and npm 11.20.0 workspaces with one lockfile and exact versions; TypeScript 6.0.3 run directly by Node's type stripping; Biome for lint and format; `node --test` for tests; `npm audit` in `make web-lint` | Decided | Fewest tools and dependencies that cover both frontends; TypeScript 7 waits for `@astrojs/check` | [ADR 0015](docs/decisions/0015-frontend-toolchain.md) |
| 49 | The Astro build ports the pixel-font compiler, WOFF, PNG crops, mirrored pairs and share cards to TypeScript with no new dependencies, keeps the `/assets/<name>.<hash>` URL scheme, and writes assets and the stylesheet from an integration after every page renders | Decided | Pixel art can't go through Astro's resampling image pipeline; the site must build with Node alone after the Go module goes | [ADR 0016](docs/decisions/0016-fonts-and-share-cards-in-typescript.md) |
