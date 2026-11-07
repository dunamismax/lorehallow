# Lorehallow Build Plan

The single source of truth for what gets built, in what order, and what "done" means. It takes Lorehallow from an empty repository (Phase 0) to the public v1.0 release (Phase 19). Agents work through it and check boxes as work lands, and lorehallow.com reads it on every push to render the live progress tracker.

Read [AGENTS.md](AGENTS.md) before changing this file.

## How this file works

- **Phases run in order.** The current phase is the lowest-numbered phase with an unchecked box. Within a phase, sections run roughly top to bottom, but independent items can be done in any order or in parallel.
- **Check a box only when it's really done:** committed, `make check` green, and doing what the line says. Partial work stays unchecked.
- **Every phase ends with acceptance criteria.** A phase is done when every box in it is checked, acceptance criteria included.
- **`HUMAN` items need the owner:** generating art, money, accounts, DNS, credentials, approvals and playtests. Agents prepare everything around them and keep working on whatever isn't blocked.
- **`ADR` items are decision points.** The agent makes the call, writes `docs/decisions/NNNN-*.md` and adds a row to the decision log in AGENTS.md. Where an item names a recommended option, that is the default unless the ADR finds a good reason otherwise.
- **The plan can change.** Agents add, split and reword unchecked items as real work demands, and explain why in the commit body. Checked items are never deleted. Cut scope moves to [Deferred](#deferred).

### Format contract

lorehallow.com parses this file. From Phase 1 on, `make check` lints it.

- Phase heading: `## Phase <N>: <Title>`
- Phase metadata: blockquote lines directly under the heading: `> **Goal:** ...`, `> **Ships:** ...`, `> **Lesson:** ...`, `> **Depends on:** ...`
- Section heading: `### <N>.<M> <Title>`, or `### Acceptance criteria`
- Item: a single line starting at column 0 with `- [ ] ` or `- [x] `, optionally followed by a tag in backticks (`HUMAN` or `ADR`)
- No nested checkboxes. Indented plain bullets under an item are notes; the parser ignores them.
- Everything outside a phase (this preamble, the overview and Deferred) is not counted.
- If the format must change, change the parser, the linter and this contract in the same commit.

## Phase overview

| Phase | Title | Ships | Core lesson |
|---|---|---|---|
| 0 | Foundations | Scaffold, pipelines, first art | Project structure, codegen, image processing |
| 1 | lorehallow.com | The website | Linux hosting, DNS, TLS, CI/CD |
| 2 | Two Players, One World | v0.2.0 | WebSockets, tick loops, interpolation |
| 3 | Many Players | v0.3.0 | Prediction, reconciliation, load testing |
| 4 | Accounts and Characters | v0.4.0 | Auth, sessions, PostgreSQL, backups |
| 5 | Chat and Presence | v0.5.0 | Fan-out messaging, rate limiting |
| 6 | Combat | v0.6.0 | Server-side game logic, AI, cooldowns |
| 7 | Interest Management | v0.7.0 | Spatial partitioning, bandwidth |
| 8 | Items and Loot | v0.8.0 | Transactional persistence, data integrity |
| 9 | Quests, NPCs and Progression | v0.9.0 | Data-driven content |
| 10 | Grouping | v0.10.0 | Cross-entity state, consistency |
| 11 | Zones | v0.11.0 | Distributed services, handoff |
| 12 | Operations | v0.12.0 | Observability, zero-downtime deploys |
| 13 | Classes and the v1 World | v0.13.0 | Content production, balance tooling |
| 14 | Audio, UI and Polish | v0.14.0 | Game feel, UX, accessibility |
| 15 | Distribution and Releases | v0.15.0 | Release engineering, signing |
| 16 | Security and Hardening | v0.16.0 | Threat modeling, fuzzing, abuse prevention |
| 17 | Closed Alpha | v0.17.0 | Playtesting, triage, live tuning |
| 18 | Open Beta | v0.18.0 | Capacity planning, community operations |
| 19 | Launch | v1.0.0 | Shipping and supporting a live game |

---

## Phase 0: Foundations

> **Goal:** An organized repository where one command runs the server, one launches the client, protobuf code generates for both sides, the art pipeline is proven end to end, and the art the website needs exists.
> **Ships:** Repo scaffold, toolchain, protobuf pipeline, art pipeline, Art Batches 01 and 02
> **Lesson:** Project structure, reproducible toolchains, code generation and image processing
> **Depends on:** Nothing

### 0.1 Owner kickoff

- [x] Write the foundational docs: README.md, AGENTS.md and BUILD.md.
- [x] `HUMAN` Choose licenses: one for code (for example MIT or AGPL-3.0) and one for art and written content (for example CC BY-NC 4.0, or all rights reserved).
- [x] `HUMAN` Confirm a ChatGPT plan with image generation is available for the art batches.
- [x] Verify `git push origin main` reaches both GitHub and Codeberg and that both repositories are public.

### 0.2 Repository skeleton

- [x] Add `CLAUDE.md` that points Claude Code sessions at AGENTS.md.
- [x] Create the directory layout from AGENTS.md section 7, each top-level directory with a short README saying what belongs there.
- [x] Add `.gitignore` covering Go build output, Godot `.godot/` and export output, `node_modules/` and `dist/`, Python `.venv/` and caches, `.env*` (except `.env.example`), and OS cruft.
- [x] Add `.editorconfig` and `.gitattributes` (LF line endings, images and other binaries marked binary).
- [x] Add LICENSE files for the licenses chosen in 0.1 and fill in the license section of README.md.
- [x] Add `docs/decisions/0000-template.md` and ADR 0001 recording the stack decisions already in the AGENTS.md decision log.

### 0.3 Toolchain and command runner

- [x] `ADR` Pin toolchain versions (Go, Godot 4.x, Node LTS, Python, uv, buf, golangci-lint, gdtoolkit, GUT) to the latest stable releases and fill in the AGENTS.md toolchain table.
- [x] Root `Makefile` with at least: `help`, `doctor`, `setup`, `fmt`, `lint`, `test`, `check`, `server`, `client`, `proto`, `sync`, `art-intake`. `make help` lists every target with a one-line description.
- [x] `make doctor` checks every required tool and version and prints how to install anything missing (Homebrew on macOS, apt on Debian/Ubuntu).
- [x] `make setup` installs project-local dependencies (Go modules, npm packages, uv environments, Godot addons) and is safe to re-run.
- [x] `make check` runs every format check, linter and test suite in the repo, and is exactly what CI runs.

### 0.4 Go server skeleton

- [x] Create the Go module in `server/` (`github.com/dunamismax/lorehallow/server`) with `cmd/lorehallow-server/` and `internal/`.
- [x] Configuration from flags and `LOREHALLOW_*` environment variables, with local-dev defaults, documented in `server/README.md` and `.env.example`.
- [x] Structured logging with `log/slog` (text locally, JSON in production).
- [x] HTTP listener with `GET /healthz` and `GET /version` (build version, git commit, protocol version).
- [x] Graceful shutdown on SIGINT and SIGTERM with a timeout.
- [x] Build version and commit injected with `-ldflags`.
- [x] `gofmt`, `go vet`, `golangci-lint`, `go test -race ./...` and `govulncheck` wired into `make lint` and `make test`.
- [x] First unit tests (configuration parsing, health handler).
- [x] `make server` builds and runs the server on a documented local port.

### 0.5 Protobuf pipeline

- [x] `buf.yaml` and `buf.gen.yaml` in `proto/`, package `lorehallow.v1`, buf lint enabled.
- [x] First schema in `proto/lorehallow/v1/`: an `Envelope` with a `oneof` payload, `Ping` and `Pong`, and the `PROTOCOL_VERSION` convention documented in `proto/README.md`.
- [x] Go code generation into `server/internal/pb/` (committed, marked generated).
- [x] `ADR` GDScript code generation: try godobuf on the pinned Godot 4 version first; if it can't pass the round-trip tests below, write `tools/protoc-gen-gdscript` (a Go protoc plugin) for the proto3 subset Lorehallow uses.
- [x] GDScript code generation into `client/net/pb/` (committed, marked generated).
- [x] Golden vectors: a Go program writes encoded messages and JSON descriptions to `proto/testdata/`; Go and GDScript tests decode and re-encode them and compare bytes.
- [x] `make proto` regenerates everything; `make check` fails when generated code is stale or `buf breaking` reports an unplanned breaking change against `main`.

### 0.6 Godot client skeleton

- [x] Create the Godot 4 project in `client/` with the project name, a placeholder icon and the folder layout from AGENTS.md.
- [x] Rendering settings: base viewport 640×360, stretch mode `viewport`, integer scaling, default texture filter Nearest, no mipmaps, 2D pixel snapping.
- [x] Pixel-art import preset (filter off, mipmaps off, lossless) as the project default.
- [x] Main scene with a placeholder title screen showing the build version.
- [x] Autoload skeletons: `Config`, `Log` and `Net` (empty for now).
- [x] GUT installed in `addons/` with one passing test; `make test` runs GUT headless.
- [x] gdtoolkit through uv; `make fmt` and `make lint` cover `client/`, excluding generated code and addons.
- [x] Static typing enforced: the project warns on untyped declarations.
- [x] `make client` launches the client (Godot found through a `GODOT` variable with sensible macOS and Linux defaults).

### 0.7 Art pipeline tooling

- [x] `tools/art/` Python project managed by uv (Pillow, numpy, PyYAML, pytest, ruff).
- [x] Batch manifest schema (`art/batches/<batch>/manifest.yaml`): per asset id, target path, kind (tile, tileset, sprite, sheet, ui, web), logical size, grid, frame layout and count, key color and palette.
- [x] Inbox mapping file (`art/inbox/<batch>/map.yaml`) linking each raw file to a manifest asset id.
- [x] Grid detection: find the real pixel block size and offset of generated "pixel art" and downsample to the logical size by sampling each block (center or mode).
- [x] Background removal: key color and near-key fringe become transparent.
- [x] Palette quantization: nearest palette color in a perceptual color space, no dithering, with a report of pixels changed.
- [x] Sheet slicing: split into frames and check frame count, frame size and foot-baseline alignment across frames, failing with a clear message when they don't line up.
- [x] Output: write to `art/processed/` with project names, plus a sidecar `.json` for sheets (frames, order, pivot, source file, batch).
- [x] Intake report (`art/batches/<batch>/report.md`): status per asset (accepted, fixed, regenerate), a 4x preview contact sheet, and corrected prompts for anything to regenerate.
- [x] Per-batch fix scripts are supported so small cleanups are reproducible when intake re-runs.
- [x] `make sync` mirrors `art/processed/` into `client/assets/art/` and `web/src/assets/art/`.
- [x] Tests with synthetic fixtures (upscaled, jittered, off-grid, off-palette and misaligned images with known expected output).
- [x] `make art-intake BATCH=<id>` runs the whole pipeline for one batch, and re-running it gives the same result.

### 0.8 Art Batch 01: style foundation

- [x] Draft the locked palette (about 32 colors for warm storybook fantasy: moss and leaf greens, lantern ambers, stone greys, dusk blues, earth browns, skin tones, a dark outline color and a near-white) as `art/style/palette.hex`, a swatch PNG and a `.gpl` palette file.
- [x] `HUMAN` Approve the palette or ask for changes.
- [x] Write `art/style/STYLE.md`: grid, scale, perspective, light, outlines, shading steps, character proportions, animation rules, and do and don't examples.
- [x] Write Batch 01 prompts and manifest: a style reference sheet, one character concept (32×48) and a grass, dirt and stone tileset (32×32, including transitions).
- [x] `HUMAN` Generate Batch 01 in ChatGPT and drop the files unchanged into `art/inbox/batch-01/`.
- [x] Run intake on Batch 01 and iterate with corrected prompts until every asset is accepted.
- [x] Save the accepted reference sheet as `art/style/reference.png` for every later prompt to attach.
- [ ] Dev scene `client/scenes/dev/art_test.tscn` renders the processed tiles and character at 1x to 4x with no blur, seams or off-palette pixels.

### 0.9 Art Batch 02: brand and website art

- [x] Write Batch 02 prompts and manifest: logo wordmark, square emblem (favicon and app icon), hero key-art scene, 2 or 3 parallax background layers, a character lineup, section icons, a sheet of 20 progress-map landmarks (one per phase, from a starting campfire to a castle for v1.0), devlog header art, an Open Graph card background (1200×630) and 404 art.
- [x] `HUMAN` Generate Batch 02 and drop it into `art/inbox/batch-02/`.
- [x] Run intake on Batch 02 (same palette, larger logical sizes allowed for web art) and iterate until accepted.
- [x] Export the favicon set (16, 32, 180, 192 and 512 px) and the client app icon from the emblem.
- [ ] `HUMAN` Approve the logo and hero art as the public face of the project.

### 0.10 Continuous integration

- [x] GitHub Actions workflow runs `make check` on every push to `main` (Linux), caching Go, npm and uv.
- [x] Godot headless tests run in CI with the pinned Godot version.
- [x] CI status badge in README.md.
- [x] Document in AGENTS.md that CI runs on GitHub and Codeberg is a mirror.

### 0.11 Walkthrough

- [x] `docs/walkthroughs/phase-00-foundations.md`: repository tour, the command runner, how protobuf codegen works on both sides, and how the art pipeline turns generated images into true pixel art.
- [x] Devlog draft for Phase 0 in `docs/devlog-drafts/`.

### 0.12 Hand-drawn art

Agents draw the art by hand from now on ([ADR 0004](docs/decisions/0004-hand-drawn-pixel-art.md)). Build the tools, then redraw the Phase 0 art that isn't a large painted image.

- [x] `ADR` Hand-drawn pixel art by agents is the default; image generation only for large painted web images.
- [ ] `art/src/` sources and `make art-draw BATCH=<id>`: `.pxl.yaml` pixel grids (legend by palette name, one grid per frame) and `.py` draw scripts, rendered deterministically into `art/processed/` via a `source:` manifest field, with intake's checks (size, palette, frames, baseline, pivot) and a 4x and 8x contact sheet per batch.
- [ ] Drawing helpers in `tools/art` for draw scripts: palette by name, seeded random, pixel, line, rect and ellipse, 1 px outline, mirror, wrap-around drawing for seamless tiles, stamping `.pxl` parts, and compositing finished assets.
- [ ] `make check` fails when a processed file no longer matches its hand-drawn source, and when a source uses an unknown color name or the wrong size.
- [ ] Redraw the base terrain tileset (16 tiles): every tile seamless with itself and its variants, low contrast, checked repeated 3×3.
- [ ] Redraw the grass-to-dirt and grass-to-flagstone transition sets so every edge and center tile self-tiles and the 3×3 patch closes.
- [ ] Redraw the concept hero (32×48, feet on row 45, 28 to 32 px tall to the top of the head).
- [ ] Draw the props the reference scene needs (a tree, a mossy rock, a lantern post, a cottage corner) and rebuild the style reference vignette (240×160) as a composite of the hand-drawn tiles, hero and props; `art/style/reference.png` follows.
- [ ] Redraw the logo wordmark (192×64), the 32 and 16 px emblems, the 16 section icons and the 20 progress landmarks, then re-export the favicons and app icon.
- [ ] Redraw the character lineup with five hand-drawn adventurers at the in-game 32×48 scale.
- [ ] Keep the generated hero scene, parallax layers, devlog header, social card and 404 art; check they still sit well next to the hand-drawn art, and note any that should be regenerated.
- [ ] Update the Phase 0 walkthrough and devlog draft to describe hand-drawn art.

### Acceptance criteria

- [x] On a fresh clone, `make doctor && make setup && make check` passes on macOS, and `make check` passes in CI on Linux.
- [x] `make server` runs the server and `make client` launches the client, which shows the title screen and build version.
- [x] Changing a `.proto` file and running `make proto` updates Go and GDScript code, and golden round-trip tests pass on both sides.
- [ ] Batch 01 tiles and the character render pixel-perfect in Godot at every integer scale.
- [x] Batch 02 assets are processed and ready in `art/processed/web/`.
- [x] The Phase 0 walkthrough exists and every `ADR` item in this phase has its ADR.

---

## Phase 1: lorehallow.com

> **Goal:** A fast, beautiful, self-hosted website at lorehallow.com that sells the dream, links to the public repositories, tracks BUILD.md automatically, hosts a devlog and has a downloads page ready for the first builds.
> **Ships:** The lorehallow.com website
> **Lesson:** Linux server hardening, DNS, TLS, reverse proxies, static site builds and automated deploys
> **Depends on:** Phase 0 sections 0.2, 0.3 and 0.9 (the rest of Phase 0 can finish in parallel)

### 1.1 Host provisioning and hardening

- [ ] `HUMAN` Fill in `deploy/HOST.md` (or give an agent the facts to write it): OS and version, CPU architecture, RAM and disk, network setup (public IP, or ports 80 and 443 forwarded if it's at home), and anything else already running on it.
- [ ] `HUMAN` Point DNS A and AAAA records for `lorehallow.com`, `www.lorehallow.com` and `play.lorehallow.com` at the host, and note the DNS provider in `deploy/HOST.md`.
- [ ] Write `deploy/bootstrap/host.sh`, an idempotent provisioning script: create the `deploy` and service users, key-only SSH with root login off, firewall allowing only 22, 80 and 443, automatic security updates, fail2ban or equivalent, Caddy, and the `/srv/lorehallow` and `/etc/lorehallow` layout.
- [ ] `HUMAN` Review and run the bootstrap script with sudo (or let an agent run it), and authorize SSH keys for agent sessions and for CI on the `deploy` user.
- [ ] `ADR` Reverse proxy: Caddy with automatic HTTPS (recommended). Config in `deploy/caddy/Caddyfile`, site root `/srv/lorehallow/web/current`.
- [ ] HTTPS works with valid certificates; HTTP redirects to HTTPS and `www` redirects to the apex.
- [ ] Security headers (HSTS, CSP, X-Content-Type-Options, Referrer-Policy, Permissions-Policy), compression, and long cache lifetimes for fingerprinted assets.
- [ ] Every host change documented in `deploy/HOST.md`.

### 1.2 Site scaffold

- [ ] `ADR` Website stack: Astro static site with TypeScript strict in `web/` (recommended).
- [ ] Design tokens generated from `art/style/palette.hex` into CSS custom properties, so the site and the game share one palette.
- [ ] `ADR` Fonts: an open-licensed pixel font for headings and a readable body font, both self-hosted with license files committed.
- [ ] Base layout: header with logo and navigation (Home, World, Progress, Devlog, Downloads, About), footer linking GitHub, Codeberg, RSS and the licenses.
- [ ] Pixel image component: integer scaling, `image-rendering: pixelated`, explicit dimensions, lazy loading below the fold.
- [ ] Make targets `web-dev`, `web-build`, `web-test` and `web-lint` (astro check, linter, formatter), included in `make check`.

### 1.3 Landing page

- [ ] Hero: key art with gentle parallax layers, the wordmark, a one-line pitch and calls to action (Follow the build, Read the devlog, View the source).
- [ ] "A world that's already there": what Lorehallow is and the feeling it's going for, in plain language.
- [ ] Design pillars with pixel icons.
- [ ] "What you'll do" teaser: explore, quest, group up and fight with other people around you, clearly marked as planned.
- [ ] Live progress widget: overall progress as an XP bar, the current phase, and a link to /progress.
- [ ] Latest three devlog posts.
- [ ] Art showcase strip built from processed art.
- [ ] "Built in the open" section linking the GitHub and Codeberg repositories and BUILD.md.
- [ ] Ambient touches (drifting fireflies, lantern flicker) that switch off under `prefers-reduced-motion`.
- [ ] The page works fully without JavaScript; scripts only add enhancements.

### 1.4 Progress tracker (/progress)

- [ ] BUILD.md parser in `web/src/lib/buildmd/` implementing the format contract: phases, metadata, sections, items, tags and checked state.
- [ ] Parser unit tests, including malformed input, and a `buildmd-lint` command in `make check` that fails on any format violation.
- [ ] Overall progress XP bar and per-phase progress bars, computed from the boxes.
- [ ] Quest map: the 20 phase landmarks from Batch 02 along a winding road; finished landmarks lit, the current one marked "You are here", future ones dimmed.
- [ ] Phase pages (`/progress/phase-N`) with goal, lesson, sections, checklists and `HUMAN` and `ADR` badges.
- [ ] Achievement feed: at build time, walk the `git log` of BUILD.md to find when each box was checked, and show the most recent as "quest complete" entries with dates and commit links.
- [ ] "Waiting on the owner" panel listing unchecked `HUMAN` items in the current phase.
- [ ] Machine-readable `/progress.json` and a `/progress.svg` badge, with the badge added to README.md.
- [ ] Each phase links to its walkthrough on GitHub once it exists.

### 1.5 Devlog (/devlog)

- [ ] Content collection in `web/src/content/devlog/` with a schema (title, date, summary, tags, cover image, draft).
- [ ] Posts with `draft: true` show in local dev only and are never published.
- [ ] Index, tag pages and post pages with pixel-art styling and comfortable reading typography.
- [ ] RSS and Atom feeds.
- [ ] Move the Phase 0 devlog draft into the collection.
- [ ] `HUMAN` Edit and publish the first devlog post.

### 1.6 Downloads (/downloads)

- [ ] Platform cards for Windows, macOS and Linux with an honest "Not available yet. Follow the progress." state.
- [ ] Page driven by `web/src/data/releases.json` (version, date, channel, platform, URL, size, SHA-256, notes link), with the schema documented for the release automation that writes it later.
- [ ] System requirements and "verify your download" sections that appear once releases exist.

### 1.7 More pages

- [ ] /world: the setting and mood (mossy forests, lantern-lit towns, old stone ruins) with an art gallery, marked as early and subject to change.
- [ ] /about: the project's two goals, building in the open, how it's made (the owner's direction, AI agents writing the code, AI-generated art run through the processing pipeline) and the tech stack.
- [ ] /faq: is it free, when can I play, which platforms, how many players, how to follow along.
- [ ] Custom 404 page with the 404 art.
- [ ] Open Graph and social card meta on every page with generated share images, plus `sitemap.xml`, `robots.txt` and canonical URLs.
- [ ] No third-party trackers, scripts or fonts on any page.

### 1.8 Quality gates

- [ ] Lighthouse CI on the landing page, /progress and a devlog post: 95 or higher for performance, accessibility, best practices and SEO on mobile.
- [ ] Accessibility: alt text on all art, full keyboard navigation, visible focus, WCAG AA contrast.
- [ ] Responsive from 360 px phones to 4K screens, with art always scaled by whole numbers.
- [ ] Internal link checker in `make check`.
- [ ] Landing page transfers under 1 MB.

### 1.9 Deploy automation

- [ ] `deploy/web/deploy.sh`: build, upload to a new timestamped release directory, switch the `current` symlink atomically, keep the last five releases.
- [ ] `make deploy-web` and `make rollback-web` for manual use.
- [ ] GitHub Actions workflow that deploys on every push to `main` touching `BUILD.md`, `web/`, `art/processed/web/` or `docs/walkthroughs/`, with full git history checked out for the achievement feed.
- [ ] `HUMAN` Add the CI deploy key and host details as GitHub Actions secrets.
- [ ] Nightly scheduled rebuild so date-based content stays fresh with no pushes.
- [ ] `ADR` Uptime monitoring that alerts the owner (for example ntfy) when the site has been down for five minutes.

### 1.10 Launch

- [ ] `docs/walkthroughs/phase-01-website.md`: DNS, TLS and Caddy, static builds, the deploy pipeline, and how BUILD.md becomes the progress page.
- [ ] Devlog draft announcing the website.
- [ ] `HUMAN` Review the live site on desktop and phone and approve it.

### Acceptance criteria

- [ ] https://lorehallow.com serves the landing page with a valid certificate, and HTTP and `www` redirect correctly.
- [ ] Checking a box in BUILD.md and pushing to `main` updates /progress within 10 minutes with no manual steps.
- [ ] Every page links to the GitHub and Codeberg repositories.
- [ ] The devlog has at least one published post and a valid RSS feed.
- [ ] /downloads exists and accurately says no builds are available yet.
- [ ] Lighthouse budgets pass and every page works with JavaScript disabled.
- [ ] A deliberately bad deploy was rolled back with one command.

---

## Phase 2: Two Players, One World

> **Goal:** Two players on different machines connect to the server at play.lorehallow.com and see each other move smoothly around one shared 30×20 map.
> **Ships:** v0.2.0
> **Lesson:** WebSockets, message framing, authoritative simulation, tick loops, interpolation and deploying a Go service behind TLS
> **Depends on:** Phase 1

### 2.1 Protocol v1

- [ ] Handshake: `ClientHello` (protocol version, client version, display name) and `ServerHello` (player id, tick rate, server tick, spawn state); mismatched protocol versions get a clear rejection.
- [ ] `Input`: sequence number, client tick and 8-way movement intent. Never a position.
- [ ] `Snapshot`: server tick, the last input sequence processed for the receiving player, and entity states (id, position, facing, animation state).
- [ ] `EntityJoined`, `EntityLeft`, `Ping` and `Pong`.
- [ ] Framing and limits (one `Envelope` per binary WebSocket message, maximum sizes) documented in `docs/design/protocol.md`.
- [ ] Round-trip golden tests for every new message.

### 2.2 Shared map format

- [ ] `ADR` Map source of truth: Tiled JSON (`.tmj`) in `content/maps/` (recommended), read by the server for collision and spawn points and imported by the client for rendering.
- [ ] Server map loader for tile layers, the collision layer and spawn points, tested against a small fixture map.
- [ ] Client map import that builds TileMapLayers from the processed tilesets.
- [ ] Map validator in `make check`: tilesets exist, collision layer present, at least one spawn point.
- [ ] `make sync` also mirrors `content/` into `client/assets/content/`.

### 2.3 Server simulation

- [ ] `ADR` WebSocket library (coder/websocket recommended); endpoint `/ws` with origin checks and read limits.
- [ ] Connection lifecycle: handshake timeout, player id assignment, spawn selection, cleanup on close or error.
- [ ] Fixed 20 Hz tick loop on a monotonic clock, measuring tick duration and warning on overruns.
- [ ] Per-connection input queues drained each tick; inputs applied in sequence order; stale and duplicate inputs dropped.
- [ ] Movement: fixed speed in tiles per second, axis-separated collision against the collision layer.
- [ ] Validation: input floods, impossible speeds and wall-walking have no effect; repeat offenders are disconnected and logged.
- [ ] Snapshot broadcast every tick to every connected player (interest management comes in Phase 7).
- [ ] Disconnects remove the entity for everyone within one second (`EntityLeft` on close, heartbeat timeout for silent drops).
- [ ] Deterministic tests of the simulation with a fake clock: movement, collision, validation, join and leave.

### 2.4 Client networking

- [ ] Connect screen: server choice (default `wss://play.lorehallow.com/ws`, plus local), display name, connect button and clear error states.
- [ ] `Net` autoload: WebSocketPeer connection, envelope encode and decode, reconnect with backoff, signals per message type.
- [ ] Input sampled at a fixed rate and sent as `Input` messages with sequence numbers.
- [ ] Local player drawn at the server's authoritative position (prediction comes in Phase 3).
- [ ] Remote players drawn about 100 ms behind the latest snapshot, interpolating between buffered snapshots, with capped extrapolation for short gaps.
- [ ] Idle and walk animations per direction, driven by movement state.
- [ ] Debug overlay (F3): RTT, snapshot rate, interpolation buffer depth, server tick and entity count.
- [ ] Dev network simulation: configurable delay, jitter and drops, on the server and in the client.

### 2.5 Art Batch 03: player character and starter tiles

- [ ] Draw Batch 03 by hand (manifest and sources in `art/src/`): the player character sheet (32×48, four directions, idle 2 frames, walk 4 frames) and the extra tiles the starter map needs (trees, rocks, path edges, water edges and a few props).
- [ ] Run intake, check frame alignment, iterate until accepted, and `make sync`.
- [ ] Character SpriteFrames and animation setup in the client; placeholder art fully replaced.

### 2.6 Starter map

- [ ] Hand-build a 30×20 map from Batch 01 and 03 tiles, with paths, obstacles worth testing collision against, and at least four spawn points.
- [ ] Camera follows the local player with pixel-snapped movement and stops at map edges.

### 2.7 Bots and cheat tests

- [ ] `server/cmd/lorehallow-bot`: a headless Go client that connects, handshakes and wanders; `-n` runs many.
- [ ] Cheat mode for the bot: input flooding, forged sequence numbers, oversized messages and unknown message types.
- [ ] Integration test proving cheat mode never moves a bot faster than allowed or through walls.
- [ ] Integration test: server plus two bots; each sees the other move, and sees the leave within one second of a disconnect.

### 2.8 Deploy the game server

- [ ] systemd unit `deploy/server/lorehallow-server.service`: dedicated user, restart on failure, sandboxing options, env file at `/etc/lorehallow/server.env`.
- [ ] Caddy site for `play.lorehallow.com` proxying `/ws` WebSocket upgrades to the server on localhost.
- [ ] `make deploy-server`: cross-compile for the host, upload, switch, restart, check `/healthz`, and roll back automatically on failure.
- [ ] Public `GET /api/status` (online count, uptime, version, protocol version) at `lorehallow.com/api/status`, cached briefly.
- [ ] Website /status page and a landing-page indicator showing server state and player count (progressive enhancement).
- [ ] Logs in journald, and `docs/ops/runbook.md` started with start, stop, logs, deploy and rollback commands.

### 2.9 Playtest builds

- [ ] `make export` builds Windows, macOS and Linux clients headless with the pinned export templates.
- [ ] Publish the v0.2.0 builds as GitHub and Codeberg releases and list them on /downloads as "Prototype".

### 2.10 Walkthrough

- [ ] `docs/walkthroughs/phase-02-two-players.md`: the tick loop, the snapshot format, interpolation, authoritative validation and the deploy, with pointers into the code.
- [ ] Devlog draft for v0.2.0 with a short clip of two players moving.
- [ ] Release notes in `docs/releases/v0.2.0.md` and tag `v0.2.0`.

### Acceptance criteria

- [ ] `HUMAN` Playtest: two clients on different machines connect to the server at play.lorehallow.com over `wss://`.
- [ ] Each player sees the other moving smoothly with no teleporting or jitter under normal latency, and also with 100 ms simulated latency and 20 ms jitter.
- [ ] Closing a client removes that player for everyone within one second.
- [ ] A modified client sending forged or flooded input cannot move faster than allowed (integration test).
- [ ] The walkthrough explains the tick loop, the snapshot format and interpolation.

---

## Phase 3: Many Players

> **Goal:** 10 or more real players and 100 bots share the world smoothly, and your own movement feels instant thanks to prediction and reconciliation.
> **Ships:** v0.3.0
> **Lesson:** Latency hiding, client prediction, reconciliation, load testing and profiling
> **Depends on:** Phase 2

### 3.1 Shared deterministic movement

- [ ] `ADR` Position representation: fixed-point integers (for example 1/16 pixel) so client and server compute identical results.
- [ ] Movement extracted into a pure step function with identical rules on both sides, documented in `docs/design/movement.md`.
- [ ] Shared test vectors: Go generates input sequences and expected positions; GDScript replays them and must match exactly.

### 3.2 Prediction and reconciliation

- [ ] Client applies local input immediately and keeps a buffer of unacknowledged inputs.
- [ ] On each snapshot, the client rewinds to the server's state for its player and replays pending inputs.
- [ ] Small errors are smoothed out visually; large errors snap. Corrections are counted in the debug overlay.
- [ ] Prediction can be toggled from the debug menu for side-by-side comparison.

### 3.3 Bandwidth

- [ ] Measure bytes per snapshot and per client per second, shown in the debug overlay and logged by the server.
- [ ] `ADR` Snapshot optimization: quantization, deltas against the last acknowledged snapshot, or both. Implement it and measure the savings.

### 3.4 Load testing

- [ ] Bot swarm: `lorehallow-bot -n 100` with wander, follow and idle behaviors and configurable latency.
- [ ] Load test script reporting tick time p50 and p99, CPU, memory and bandwidth at 10, 50 and 100 bots, with results in `docs/perf/phase-03.md`.
- [ ] pprof on a localhost-only admin port; one profiling pass completed and any hotspot fixed.
- [ ] Race detector clean during a load test run.

### 3.5 Art Batch 04: a crowd of people

- [ ] Draw Batch 04 by hand: three or four player appearance variants and a townsfolk placeholder, so a crowd reads as different people.
- [ ] Intake and integrate; bots pick random appearances.

### 3.6 Walkthrough

- [ ] `docs/walkthroughs/phase-03-many-players.md`: prediction, reconciliation, fixed-point simulation, and measuring and reducing bandwidth.
- [ ] Devlog draft, release notes, tag `v0.3.0` and publish builds.

### Acceptance criteria

- [ ] With 150 ms simulated latency, your own movement responds on the next frame and corrections aren't noticeable in normal play.
- [ ] 100 bots on the production host keep server tick p99 under 12.5 ms (a quarter of the tick budget).
- [ ] `HUMAN` Playtest: at least 3 real players plus 20 bots on the live server feels smooth.
- [ ] The walkthrough explains prediction and reconciliation with diagrams.

---

## Phase 4: Accounts and Characters

> **Goal:** Players create an account, log in, make a character and come back later to find it where they left it.
> **Ships:** v0.4.0
> **Lesson:** Authentication, sessions, PostgreSQL, migrations and backups
> **Depends on:** Phase 3

### 4.1 Database

- [ ] PostgreSQL on the host through the bootstrap script: pinned major version, local connections only, separate roles for migrations and the app.
- [ ] `ADR` Database access: pgx with sqlc for typed queries and goose for migrations (recommended).
- [ ] Migrations in `server/migrations/`, run by `make migrate` locally and as a deploy step in production.
- [ ] Local dev database with Docker Compose (`make db-up`, `make db-reset`) and a Postgres service container in CI.
- [ ] Schema: accounts, sessions and characters (name, appearance, zone, position, facing, timestamps).

### 4.2 Authentication

- [ ] Registration: username rules, password of at least 10 characters checked against a common-password list, Argon2id hashing with parameters recorded in an ADR.
- [ ] HTTPS endpoints `POST /api/auth/register`, `/login` and `/logout` returning an opaque session token, stored only as a hash.
- [ ] The WebSocket handshake carries the session token; invalid or expired tokens are rejected; a new login ends any older game session for that account.
- [ ] Rate limits and lockouts on login and registration per IP and per account, proven by tests.
- [ ] `HUMAN` Decide on account recovery for v1 (no email, or optional email with an SMTP provider). An agent writes the options up as an ADR first.

### 4.3 Characters

- [ ] Character creation: name rules (unique, allowed characters, length, reserved and offensive words filtered) and appearance chosen from the Batch 04 variants.
- [ ] Character select: up to four characters per account, with delete behind a typed confirmation (soft delete).
- [ ] Position and state saved on logout, on disconnect and every 60 seconds, and restored on login.
- [ ] Client login, registration, character select and creation screens with clear errors; the token lives in memory unless "remember me" is chosen.

### 4.4 Backups

- [ ] Nightly `pg_dump`, compressed and encrypted, keeping 7 daily and 4 weekly copies.
- [ ] `HUMAN` Choose an off-site backup destination and put its credentials on the host.
- [ ] Every backup copied off-site automatically, with an alert if a backup fails.
- [ ] Restore drill: restore the latest off-site backup into a scratch database and verify it, documented in `docs/ops/backups.md`.

### 4.5 Website

- [ ] Privacy policy page (what's stored, why, for how long, how to delete an account) and terms of use with a player code of conduct.
- [ ] `HUMAN` Review and approve the privacy policy and terms.

### 4.6 Walkthrough

- [ ] `docs/walkthroughs/phase-04-accounts.md`: password hashing, sessions, migrations, transactions and backups.
- [ ] Devlog draft, release notes, tag `v0.4.0` and publish builds.

### Acceptance criteria

- [ ] A new player can register, create a character, play, quit, and log back in at the same spot.
- [ ] Passwords exist only as Argon2id hashes and session tokens only as hashes.
- [ ] Brute-force login attempts are rate limited (integration test).
- [ ] A backup was restored successfully from the off-site copy.
- [ ] CI runs the database integration tests against a real Postgres.

---

## Phase 5: Chat and Presence

> **Goal:** Players can talk and see who's around: say, zone chat, whispers, nameplates, emotes and a who list.
> **Ships:** v0.5.0
> **Lesson:** Fan-out messaging, rate limiting and moderation basics
> **Depends on:** Phase 4

### 5.1 Chat on the server

- [ ] Channels: say (range-limited), zone, whisper (by character name), emote and system, with server timestamps.
- [ ] Fan-out by channel with length limits and per-player token-bucket rate limits that reply with a friendly "slow down".
- [ ] Persisted ignore list and whisper reply.
- [ ] `/who` listing online characters in the zone.
- [ ] Text hygiene: Unicode normalization, control characters stripped, combining marks capped.
- [ ] Chat logs kept for 30 days for moderation and then deleted, as the privacy policy states.

### 5.2 Moderation

- [ ] Account roles (player, moderator, admin) with server-side permission checks.
- [ ] Moderator commands: timed mute, kick, and timed or permanent ban, all written to an audit log.
- [ ] `/report <name> <reason>` stored for review.
- [ ] `lorehallow-admin` CLI on the host for role grants, bans and report review.

### 5.3 Chat in the client

- [ ] Chat window with channel filters, scrollback, `/` commands, optional timestamps and clickable names to whisper.
- [ ] Speech bubbles above characters for say chat.
- [ ] Nameplates with a readable outlined pixel font.
- [ ] Emotes `/wave`, `/sit` and `/dance` with animations.

### 5.4 Art Batch 05: UI kit

- [ ] `ADR` Game UI font: an open-licensed pixel font with full Latin coverage, ideally matching the website.
- [ ] Draw Batch 05 by hand: 9-slice window frames, buttons, chat bubble, nameplate backing, channel icons and emote frames (sit, wave, dance).
- [ ] Intake and integrate, with one Godot theme at `client/ui/theme.tres` used by all UI.

### 5.5 Walkthrough

- [ ] `docs/walkthroughs/phase-05-chat.md`: fan-out, rate limiting and moderation design.
- [ ] Devlog draft, release notes, tag `v0.5.0` and publish builds.

### Acceptance criteria

- [ ] Say reaches only nearby players, zone chat reaches the zone, and whispers reach only their target.
- [ ] A bot spamming chat gets rate limited without affecting anyone else (test).
- [ ] A moderator can mute, kick and ban, and every action appears in the audit log.
- [ ] `HUMAN` Playtest: a small group chats comfortably and the UI reads well at 2x and 3x.

---

## Phase 6: Combat

> **Goal:** Tab-target a creature, fight it with an auto-attack and one ability, die and respawn, and watch it respawn too.
> **Ships:** v0.6.0
> **Lesson:** Server-side game logic, entity models, cooldowns, threat and AI state machines
> **Depends on:** Phase 5

### 6.1 Entity model

- [ ] `ADR` Entity architecture: components and systems, or plain structs with interfaces. Pick whatever keeps the tick loop clear and testable.
- [ ] One entity model for players and mobs: id, kind, position, health, faction and state.
- [ ] Stats (health, attack power, armor, movement speed) with derived values computed in one place.
- [ ] Combat events to clients: damage, heal, death, respawn, cast start and cast stop.

### 6.2 Targeting and abilities

- [ ] Tab-target cycling (nearest hostile in front first), click to target, Escape to clear; the server validates targets.
- [ ] Auto-attack with a swing timer, range check and facing.
- [ ] Data-defined abilities in `content/abilities/` (cast time, cooldown, global cooldown, range, cost, effects), with one ability working end to end.
- [ ] A resource (stamina or mana) with regeneration.
- [ ] Server validation on every ability use: cooldowns, range, line of sight, resource and a living target.

### 6.3 Mobs and AI

- [ ] Mob definitions in `content/mobs/` and spawn points in maps, with respawn timers.
- [ ] AI state machine: idle, wander, aggro on proximity, chase, attack, and leash back with a full reset when pulled too far.
- [ ] Threat table: damage builds threat and mobs attack whoever is on top.
- [ ] Player death, a respawn point and a short respawn delay; mob death with a fading corpse.

### 6.4 Combat in the client

- [ ] Player frame and target frame with health and resource bars.
- [ ] Action bar on keys 1 to 6 with cooldown sweeps and an out-of-range tint.
- [ ] Floating combat text, hit flashes and death animations.
- [ ] Health bars over damaged entities and a target selection ring.
- [ ] Combat log window.

### 6.5 Art Batch 06: combat

- [ ] Draw Batch 06 by hand: player attack and cast animations in four directions, hurt and death frames, the first forest mob (idle, walk, attack and death in four directions), hit and ability effects, ability icons and the target ring.
- [ ] Intake and integrate.

### 6.6 Walkthrough

- [ ] `docs/walkthroughs/phase-06-combat.md`: server-authoritative combat, cooldowns, threat and AI state machines.
- [ ] Devlog draft, release notes, tag `v0.6.0` and publish builds.

### Acceptance criteria

- [ ] A player can target, fight and kill a mob, which respawns on its timer.
- [ ] A player can die and respawn at the respawn point.
- [ ] Abilities can't be used early, out of range or without resources, even by a modified client (tests).
- [ ] Mobs leash and reset when kited too far.
- [ ] `HUMAN` Playtest: two players fighting the same mob feels good and reads clearly.

---

## Phase 7: Interest Management

> **Goal:** A much larger map where each player only receives updates about what's near them, so bandwidth stays flat as the world grows.
> **Ships:** v0.7.0
> **Lesson:** Spatial partitioning, areas of interest and bandwidth budgets
> **Depends on:** Phase 6

### 7.1 Areas of interest

- [ ] `ADR` Spatial index: a uniform grid (recommended) with a chosen cell size, updated as entities move.
- [ ] Per-player area of interest with hysteresis (enter radius smaller than leave radius) so entities don't flicker at the edge.
- [ ] Enter and leave messages driven by area changes; snapshots include only entities in the area.
- [ ] Say chat range and combat events use the same spatial queries.
- [ ] Tests at cell boundaries and with fast movement.

### 7.2 A larger map

- [ ] Build a large map (about 200×150 tiles) with a town, a forest edge and ruins, as the first real zone layout.
- [ ] `ADR` Measure client rendering and memory on the large map, and decide whether chunked loading is needed.
- [ ] Minimap showing nearby terrain and other players.

### 7.3 Art Batch 07: zone tiles

- [ ] Draw Batch 07 by hand: forest variants, town building exteriors, lanterns, ruins, water and bridges.
- [ ] Intake and integrate.

### 7.4 Measurement

- [ ] Per-client bandwidth with 100 bots spread across the map, before and after interest management, in `docs/perf/phase-07.md`.

### 7.5 Walkthrough

- [ ] `docs/walkthroughs/phase-07-interest-management.md`: spatial partitioning, hysteresis and bandwidth budgets.
- [ ] Devlog draft, release notes, tag `v0.7.0` and publish builds.

### Acceptance criteria

- [ ] With 100 bots spread across the map, per-client bandwidth is at least 70% lower than broadcasting everything.
- [ ] Entities don't visibly pop in or out at area edges during normal movement.
- [ ] Server tick p99 stays under 12.5 ms with 100 bots.

---

## Phase 8: Items and Loot

> **Goal:** Mobs drop loot, players carry it in bags, wear gear that changes their stats and trade with vendors, without an item ever being lost or duplicated.
> **Ships:** v0.8.0
> **Lesson:** Transactional persistence, data integrity and idempotency
> **Depends on:** Phase 7

### 8.1 Item data

- [ ] Item definitions in `content/items/` (id, name, icon, quality, slot, stats, stack size, vendor price, flavor text) with a validator in `make check`.
- [ ] Weighted loot tables in `content/loot/`, with tests of drop distributions.
- [ ] Item instances in Postgres with unique ids, plus inventory and equipment tables; every move is one transaction.

### 8.2 Items on the server

- [ ] `ADR` Loot model: personal loot per eligible player, or a shared corpse. Pick one and record why.
- [ ] Looting with range checks and a loot timeout.
- [ ] Inventory operations (move, split, stack, destroy) made idempotent with request ids.
- [ ] Equipment slots (head, chest, legs, feet, hands, main hand, off hand, trinket) feeding the stat system.
- [ ] Gold, and vendor NPCs that buy, sell and offer buyback for the session.
- [ ] Inventory changes persisted immediately in transactions, not on the periodic save.
- [ ] Duplication tests: concurrent requests, disconnects mid-operation and replayed messages never duplicate or lose an item.

### 8.3 Items in the client

- [ ] Bags window and character window with a paper-doll preview, drag and drop, and tooltips with stat comparison.
- [ ] Loot window and loot notifications.
- [ ] Vendor window.

### 8.4 Art Batch 08: items

- [ ] Draw Batch 08 by hand: 32×32 item icon sheets, quality borders, bag and character window art, coin icons and a vendor NPC.
- [ ] Intake and integrate.

### 8.5 Walkthrough

- [ ] `docs/walkthroughs/phase-08-items.md`: transactions, idempotency and how duplication bugs happen and are prevented.
- [ ] Devlog draft, release notes, tag `v0.8.0` and publish builds.

### Acceptance criteria

- [ ] Kill, loot, equip, see the stat change, and sell to a vendor all work end to end.
- [ ] Duplication and loss tests pass, including killing the server mid-transaction.
- [ ] Inventories survive server restarts and come back intact from a backup restore.

---

## Phase 9: Quests, NPCs and Progression

> **Goal:** NPCs offer quests with objectives and rewards, and characters earn experience and level from 1 to 10.
> **Ships:** v0.9.0
> **Lesson:** Data-driven content, content pipelines and validation
> **Depends on:** Phase 8

### 9.1 Progression

- [ ] Experience from kills and quests, a level 1 to 10 curve in `content/progression.yaml`, and stat growth per level.
- [ ] Level-up event with effects in the client, and levels on nameplates and frames.
- [ ] Mob level relative to player level scales experience (far lower mobs give none).

### 9.2 NPCs and dialogue

- [ ] NPC definitions in `content/npcs/` and placement in maps.
- [ ] Dialogue trees in `content/dialogue/` with conditions (quest state, level) and actions (offer quest, open vendor).
- [ ] Interact key and range checks.

### 9.3 Quests

- [ ] Quest definitions in `content/quests/`: prerequisites; objectives (kill, collect, talk, explore, use object); rewards (experience, gold, a choice of items).
- [ ] Server quest state machine persisted per character; quest items drop only while the quest is active.
- [ ] `!` and `?` markers over quest givers.
- [ ] Quest log window and an on-screen objective tracker.
- [ ] Content validator in `make check`: every reference resolves, every quest can be completed, no orphaned items.
- [ ] Dev commands to set quest state for testing.

### 9.4 Starter content

- [ ] Start `docs/design/lore.md`: the setting, its tone, naming rules and a short history.
- [ ] Ten starter quests that teach movement, combat, looting and talking to NPCs, written in the Lorehallow voice.

### 9.5 Art Batch 09: NPCs

- [ ] Draw Batch 09 by hand: four to six NPCs (innkeeper, guard, smith and quest givers), quest markers, an interaction cursor and a level-up effect.
- [ ] Intake and integrate.

### 9.6 Walkthrough

- [ ] `docs/walkthroughs/phase-09-quests.md`: data-driven design, content validation and quest state machines.
- [ ] Devlog draft, release notes, tag `v0.9.0` and publish builds.

### Acceptance criteria

- [ ] A new character can reach level 4 by completing the starter quests.
- [ ] A headless bot test completes the whole starter quest chain through the real server.
- [ ] Quest progress survives logout and server restart.

---

## Phase 10: Grouping

> **Goal:** Players form parties of up to five, see each other on the map, and share quest credit and experience.
> **Ships:** v0.10.0
> **Lesson:** Cross-entity state, consistency and disconnect edge cases
> **Depends on:** Phase 9

### 10.1 Parties on the server

- [ ] Party model: invite, accept, decline, leave, kick, promote and disband, up to five members.
- [ ] Party membership survives a brief disconnect (grace period) and stays consistent for every member.
- [ ] Shared kill credit, experience split by level for members in range, and shared quest objective credit nearby.
- [ ] Party loot rules that follow the Phase 8 loot model.
- [ ] Party chat channel.
- [ ] Low-rate party member updates (health, position) even outside the area of interest.

### 10.2 Parties in the client

- [ ] Party frames with health and resource bars.
- [ ] Invite through a right-click menu and `/invite`.
- [ ] Party members shown on the minimap and at screen edges.

### 10.3 A reason to group

- [ ] An elite camp in the zone, tuned for three players, with a rare drop.
- [ ] Draw Batch 10 by hand: the elite mobs and camp props.
- [ ] Intake and integrate.

### 10.4 Walkthrough

- [ ] `docs/walkthroughs/phase-10-grouping.md`: cross-entity state, consistency and disconnect handling.
- [ ] Devlog draft, release notes, tag `v0.10.0` and publish builds.

### Acceptance criteria

- [ ] Five players form a party, beat the elite camp, and all get quest credit and experience.
- [ ] Disconnecting and reconnecting within the grace period keeps party membership.
- [ ] Concurrency tests: simultaneous invites, kicks and leaves never corrupt a party.
- [ ] `HUMAN` Playtest: a group of three or more takes on the elite camp together.

---

## Phase 11: Zones

> **Goal:** The world spans multiple maps with quick transitions, and zones run as separate services.
> **Ships:** v0.11.0
> **Lesson:** Distributed services, handoff and service boundaries
> **Depends on:** Phase 10

### 11.1 Architecture

- [ ] `ADR` Zone architecture: one process with many zone simulations, or a gateway plus one process per zone (the learning goal favors trying the multi-process design). Write up the tradeoffs.
- [ ] Zone registry and routing (the gateway keeps the client connection, or clients reconnect with a handoff token).
- [ ] Handoff protocol: save state, transfer and spawn in the new zone, rolling back to the old zone on failure.
- [ ] Zone and world chat, whispers and parties work across zones.
- [ ] `make server` runs the whole set locally; production gets a systemd unit per service.

### 11.2 The world

- [ ] Zone transitions at map edges and doors, with a short loading screen.
- [ ] Interiors (an inn, a house) as small zones.
- [ ] World map window showing the zones and where you are.
- [ ] The three v1 zones blocked out (starting town and fields, Mosswood, the Old Ruins; working names), to be filled with content in Phase 13.

### 11.3 Resilience

- [ ] Bots cross zones continuously for an hour with no lost characters or items.
- [ ] Killing a zone process mid-handoff leaves every player in a safe, recoverable state.

### 11.4 Walkthrough

- [ ] `docs/walkthroughs/phase-11-zones.md`: service boundaries, handoff and failure handling.
- [ ] Devlog draft, release notes, tag `v0.11.0` and publish builds.

### Acceptance criteria

- [ ] Players walk between at least three zones with transitions under two seconds.
- [ ] Parties, whispers and quests work across zones.
- [ ] A zone crash affects only that zone's players, who can reconnect.

---

## Phase 12: Operations

> **Goal:** Lorehallow runs like a real live service, with metrics, logs, alerts, tested backups and deploys that don't throw everyone out.
> **Ships:** v0.12.0
> **Lesson:** Observability, alerting and release engineering for a live service
> **Depends on:** Phase 11

### 12.1 Metrics and dashboards

- [ ] Prometheus metrics from every service: connected players, tick duration histogram, messages and bytes in and out, database latency and errors.
- [ ] `ADR` Monitoring stack on the host (Prometheus and Grafana, or something lighter), reachable only through an SSH tunnel or an authenticated proxy.
- [ ] Dashboards for server health, player activity and performance.

### 12.2 Logs

- [ ] Structured logs carrying session and request ids, a retention policy, and documented ways to search them.

### 12.3 Alerts

- [ ] Alerts to the owner for: a service down, tick overruns, disk over 80%, a failed backup and certificates close to expiry.

### 12.4 Deploys

- [ ] `ADR` Zero-downtime strategy: drain and hand connections off through the gateway, or scheduled restarts with in-game countdowns and automatic reconnects.
- [ ] Implement it: a server deploy loses no progress and causes at most a brief, automatic reconnect.
- [ ] Database migrations follow expand-and-contract so the previous server version still works, documented in `docs/ops/migrations.md`.
- [ ] Minimum client version announced by the server; outdated clients get an "Update required" screen linking to /downloads.

### 12.5 Runbooks

- [ ] Runbooks in `docs/ops/` for: server down, database full, bad deploy, compromised account and restoring from backup.
- [ ] Game-day exercise: simulate two incidents and follow the runbooks, fixing whatever was wrong in them.

### 12.6 Status page

- [ ] /status shows 30-day uptime and recent incidents.

### 12.7 Walkthrough

- [ ] `docs/walkthroughs/phase-12-operations.md`: metrics, alerts, deploy strategy and incident response.
- [ ] Devlog draft, release notes, tag `v0.12.0` and publish builds.

### Acceptance criteria

- [ ] Deploying a server update during a 20-bot session loses no progress.
- [ ] Each alert type was fired in a test and reached the owner.
- [ ] Dashboards show live player counts and tick times.

---

## Phase 13: Classes and the v1 World

> **Goal:** Three classes and three zones of handcrafted content take a character from level 1 to 10, ending in a group challenge.
> **Ships:** v0.13.0
> **Lesson:** Content production at scale and balance tooling
> **Depends on:** Phase 12

### 13.1 Classes

- [ ] `ADR` Class design in `docs/design/classes.md`: three classes (working names Warden for melee and tanking, Ranger for ranged damage, Lorekeeper for spells and healing) with roles, resources and identity.
- [ ] `HUMAN` Approve the class design.
- [ ] Five or six data-defined abilities per class, unlocked across levels 1 to 10.
- [ ] Class trainer NPCs who teach abilities for gold.
- [ ] Balance simulator: time-to-kill per class against mobs at each level, with all classes within 15% of each other solo.

### 13.2 Zone content

- [ ] Zone 1, starting town and fields (levels 1 to 4): final layout, 12 or more quests, three or four mob types, and town services (inn, vendors, trainers).
- [ ] Zone 2, Mosswood (levels 4 to 7): 12 or more quests, four mob types and a small outpost.
- [ ] Zone 3, the Old Ruins (levels 7 to 10): 12 or more quests, four mob types, an elite area and a group boss with a simple mechanic that ends the main quest chain.
- [ ] A recurring world event (for example a creature incursion at the town gate) that pulls players together.
- [ ] Ambient life: townsfolk going about their day, critters, and a day and night cycle with lantern light in towns.
- [ ] `docs/design/lore.md` complete, and all quest text proofread for consistency.

### 13.3 Art for the v1 world

- [ ] Write the class batch (class looks, ability effects and icons) and one batch per zone (tiles, props, mobs, NPCs, boss).
- [ ] `HUMAN` Generate the class batch.
- [ ] `HUMAN` Generate the three zone batches.
- [ ] Intake and integrate every batch, then review consistency across all batches against the reference sheet.

### 13.4 Validation

- [ ] Scripted bot playthrough from level 1 to 10 for each class along the quest route with no blockers.
- [ ] `HUMAN` Play one class from 1 to 10 and write notes; agents triage the notes into this phase.

### 13.5 Walkthrough

- [ ] `docs/walkthroughs/phase-13-content.md`: the content pipeline, validators and balance tooling.
- [ ] Devlog draft, release notes, tag `v0.13.0` and publish builds.

### Acceptance criteria

- [ ] Every class reaches level 10 through quests in roughly 5 to 8 hours of play.
- [ ] All content passes the validators, with no unfinished quests.
- [ ] Three to five level-10 players can beat the group boss.

---

## Phase 14: Audio, UI and Polish

> **Goal:** Lorehallow sounds and feels finished, with music, sound effects, settings, onboarding and accessibility options.
> **Ships:** v0.14.0
> **Lesson:** Game feel, UX and accessibility
> **Depends on:** Phase 13

### 14.1 Audio

- [ ] `HUMAN` Choose music and sound sources (generated, commissioned, or licensed or CC0 packs) and confirm the licenses.
- [ ] Audio buses (master, music, effects, UI, ambient) with volume settings.
- [ ] Music per zone and for combat with crossfades, plus ambient loops (forest, town, wind in the ruins).
- [ ] Sound effects: footsteps by terrain, hits, abilities, UI, loot, level up, quest complete and the whisper ping.
- [ ] Audio credits and licenses listed in the game and on the website.

### 14.2 Settings and controls

- [ ] Settings menu: window mode, integer scale, vsync, FPS cap, audio, gameplay, chat filters and UI scale.
- [ ] Key rebinding with conflict detection, saved to the user config.
- [ ] `ADR` Controller support for v1, or deferred.

### 14.3 Onboarding

- [ ] First-time tips (movement, targeting, chat, quests) that can be turned off.
- [ ] Clear messages for every connection and login failure.

### 14.4 Accessibility

- [ ] Shapes as well as colors for hostility and item quality, text size options, a screen shake toggle and reduced flashing.

### 14.5 Polish

- [ ] Game feel pass: hit feedback, footstep dust and smooth UI transitions.
- [ ] Performance: 60 FPS at 1080p on a five-year-old integrated-graphics laptop with 30 players on screen; profile and fix.
- [ ] Every placeholder asset and placeholder string is gone.
- [ ] All player-facing strings go through Godot's translation system, with English as the only shipped language.

### 14.6 Walkthrough

- [ ] `docs/walkthroughs/phase-14-polish.md`: audio architecture, settings persistence and performance profiling.
- [ ] Devlog draft, release notes, tag `v0.14.0` and publish builds.

### Acceptance criteria

- [ ] `HUMAN` Play for 30 minutes and sign off on sound and feel.
- [ ] Every setting persists and applies without a restart where possible.
- [ ] The performance target is met and recorded in `docs/perf/phase-14.md`.

---

## Phase 15: Distribution and Releases

> **Goal:** Anyone can download, verify, install and update Lorehallow from lorehallow.com on Windows, macOS and Linux.
> **Ships:** v0.15.0
> **Lesson:** Release engineering, code signing and supply-chain hygiene
> **Depends on:** Phase 14

### 15.1 Builds

- [ ] Release workflow on `v*` tags: export every platform in CI, embed the version, package (zip for Windows, dmg or zip for macOS, tar.gz for Linux) and write SHA-256 checksums.
- [ ] `HUMAN` Decide on macOS signing and notarization (the Apple Developer Program costs money) and Windows code signing, or choose to ship unsigned with clear instructions.
- [ ] If signing was approved, sign and notarize in CI using secrets.
- [ ] Pinned export templates and recorded build metadata for every release.

### 15.2 Publishing

- [ ] Publish to GitHub Releases and Codeberg releases, and self-host the files on lorehallow.com with checksums.
- [ ] The workflow updates `web/src/data/releases.json` and redeploys, so /downloads shows the latest build, checksums and older versions.
- [ ] `/downloads/latest.json` manifest for the client update check.
- [ ] /patch-notes built from `docs/releases/`.

### 15.3 In-client updates

- [ ] The client checks `latest.json` at startup and shows "New version available" with a download link, mandatory below the server's minimum version.
- [ ] `ADR` Launcher or auto-updater for v1, or the update prompt alone (the prompt alone is recommended for v1).

### 15.4 Storefronts

- [ ] `HUMAN` Decide whether to also list Lorehallow on itch.io.
- [ ] If yes, upload with butler from the release workflow.

### 15.5 Install guides

- [ ] Install guides per platform on the website (including any unsigned-app warnings) and final system requirements.

### 15.6 Walkthrough

- [ ] `docs/walkthroughs/phase-15-distribution.md`: the release pipeline, checksums, signing and update checks.
- [ ] Devlog draft, release notes and tag `v0.15.0` through the new pipeline.

### Acceptance criteria

- [ ] Pushing a release tag produces builds for all three platforms, publishes them everywhere and updates /downloads with no manual steps.
- [ ] `HUMAN` Fresh installs from the downloads page work on Windows, macOS and Linux (owner or friends verify).
- [ ] An outdated client is told to update.

---

## Phase 16: Security and Hardening

> **Goal:** Lorehallow is safe to open to strangers.
> **Ships:** v0.16.0
> **Lesson:** Threat modeling, fuzzing and abuse prevention
> **Depends on:** Phase 15

### 16.1 Threat model

- [ ] `docs/design/threat-model.md`: assets, attackers, entry points (WebSocket, HTTP APIs, admin tools, the host) and mitigations.

### 16.2 Protocol and server

- [ ] Go fuzz tests for envelope decoding and every message handler, run briefly in CI and longer nightly.
- [ ] Limits everywhere (connections per IP, messages per second, message size, pending inputs, names, chat length), each with a test.
- [ ] Authority audit: every client message type reviewed for trust assumptions, with findings fixed.
- [ ] Slow-client and idle timeouts; origin checks and TLS-only connections in production.
- [ ] Server-side anomaly logging (speed, impossible actions, abnormal rates) with an admin report.

### 16.3 Accounts and data

- [ ] Session expiry and revocation, "log out everywhere" and password change.
- [ ] Self-service account deletion that honors the privacy policy.
- [ ] Least-privilege database roles, verified encrypted backups, and secrets rotated with the process documented.

### 16.4 Host and supply chain

- [ ] Host audit (open ports, services, users, SSH, updates, file permissions) with findings fixed and documented.
- [ ] Automated dependency updates (Renovate or Dependabot) for Go, npm and GitHub Actions, plus `govulncheck` and `npm audit` in CI.

### 16.5 Walkthrough

- [ ] `docs/walkthroughs/phase-16-security.md`: the threat model, fuzzing and the authority audit.
- [ ] Devlog draft, release notes, tag `v0.16.0` and publish builds.

### Acceptance criteria

- [ ] Fuzzers run for an hour with no crashes.
- [ ] Every threat in the model has a mitigation or an accepted risk with a reason.
- [ ] No known vulnerabilities in dependencies.

---

## Phase 17: Closed Alpha

> **Goal:** Invited players play the whole game on the live server, and their feedback shapes the beta.
> **Ships:** v0.17.0
> **Lesson:** Running playtests, triage and live tuning
> **Depends on:** Phase 16

### 17.1 Preparation

- [ ] Invite codes required at registration, generated with `lorehallow-admin`.
- [ ] In-game `/bug` and `/feedback` commands that store reports with character state and client version, plus an admin review command.
- [ ] Privacy-respecting aggregate telemetry (session length, levels reached, quest completion, deaths by location) with an opt-out, described in the privacy policy.
- [ ] Alpha wipe policy decided and stated on the website.
- [ ] Alpha section on the website and a devlog draft announcing it.

### 17.2 Running the alpha

- [ ] `HUMAN` Invite 5 to 20 alpha testers.
- [ ] `HUMAN` Run at least two scheduled group playtest sessions.
- [ ] Triage all feedback into new items under 17.3, ordered by priority.

### 17.3 Alpha fixes

- [ ] Fix every crash and data-loss bug found during the alpha.
- [ ] Balance pass using telemetry: experience curve, mob difficulty and class parity.
- [ ] UX fixes from tester feedback.

### 17.4 Wrap-up

- [ ] `docs/walkthroughs/phase-17-alpha.md`: what testers found, what changed and what the telemetry showed.
- [ ] Release notes, tag `v0.17.0` and publish builds.

### Acceptance criteria

- [ ] Two alpha sessions with five or more concurrent real players completed.
- [ ] No open crash or data-loss bugs.
- [ ] `HUMAN` Decide that the alpha's goals are met.

---

## Phase 18: Open Beta

> **Goal:** Anyone can sign up and play, and the server proves it can handle the crowd.
> **Ships:** v0.18.0
> **Lesson:** Capacity planning and community operations
> **Depends on:** Phase 17

### 18.1 Capacity

- [ ] 72-hour soak test with 100 bots plus real players: no memory growth and no tick degradation.
- [ ] Capacity test to find where tick p99 exceeds budget, documented in `docs/perf/phase-18.md`, with the player cap set safely below it.
- [ ] Login queue when the server is full.

### 18.2 Community

- [ ] `HUMAN` Decide on a community space (Discord, Matrix, a forum, or none) and set it up.
- [ ] Link the community space from the website and the game.
- [ ] Moderation process documented for moderators in `docs/ops/moderation.md`.

### 18.3 Opening the doors

- [ ] Remove the invite requirement and switch the website's calls to action to "Play the beta".
- [ ] `HUMAN` Approve and publish the open beta announcement.
- [ ] Watch the first beta week closely and ship hotfixes through the release pipeline.

### 18.4 Beta fixes

- [ ] Triage beta feedback into new items in this section and fix them by priority.
- [ ] Final content and balance polish.
- [ ] `ADR` Whether characters are wiped at 1.0, announced on the website and in game.

### 18.5 Wrap-up

- [ ] `docs/walkthroughs/phase-18-beta.md`: capacity planning, the soak test and lessons from running a public beta.
- [ ] Release notes, tag `v0.18.0` and publish builds.

### Acceptance criteria

- [ ] The 72-hour soak test passed.
- [ ] A peak of 25 or more concurrent real players (or a target the owner sets) was handled without degradation.
- [ ] No open critical bugs for two weeks.

---

## Phase 19: Launch

> **Goal:** Lorehallow 1.0 is publicly released, stable and supported.
> **Ships:** v1.0.0
> **Lesson:** Shipping and supporting a live game
> **Depends on:** Phase 18

### 19.1 Release candidate

- [ ] Feature freeze and tag `v1.0.0-rc.1`; only fixes from here on.
- [ ] Full regression checklist in `docs/releases/regression.md` (every system, every class from 1 to 10, every quest, every platform), run and passed.
- [ ] Final backup and restore drill, and a written rollback plan for the launch build.
- [ ] Performance and security checks re-run against the release candidate.

### 19.2 Launch preparation

- [ ] Launch-day runbook: what to watch, rollback steps and how to communicate problems.
- [ ] Website launch update: "Play now" hero, final screenshots and a gameplay trailer, updated FAQ, and a /press page (description, fact sheet, logos, screenshots, contact).
- [ ] `HUMAN` Choose the launch date.
- [ ] `HUMAN` Approve the launch devlog post and announcements.
- [ ] Character wipe carried out if decided in Phase 18, with a backup taken first.

### 19.3 Launch

- [ ] Tag `v1.0.0`; the release pipeline publishes every build, and /downloads and /progress show 1.0.
- [ ] `HUMAN` Publish the launch announcements.
- [ ] Monitor closely for the first week and ship `v1.0.x` hotfixes as needed.

### 19.4 After launch

- [ ] Post-mortem in `docs/releases/v1.0-postmortem.md`: what went well, what didn't and what was learned.
- [ ] `docs/walkthroughs/phase-19-launch.md`: the whole architecture end to end as it stands at 1.0.
- [ ] Post-1.0 roadmap draft in `docs/design/roadmap-post-1.0.md`, built from Deferred and player feedback.

### Acceptance criteria

- [ ] v1.0.0 can be downloaded for Windows, macOS and Linux from lorehallow.com.
- [ ] The first launch week had no data loss and 99% or better uptime.
- [ ] Every earlier phase in this file is complete.
- [ ] `HUMAN` Declare Lorehallow 1.0 shipped.

---

## Deferred

Ideas and cut scope parked until after 1.0. Nothing here is counted or scheduled.

- Visible equipment on character sprites (layered paper-doll art)
- Guilds
- Player trading and an auction house
- Gathering and crafting professions
- Dungeons and instancing
- PvP: duels and arenas
- Mounts
- Player housing
- A web client (the WebSocket transport already allows it)
- A launcher with automatic patching
- Controller support, if deferred in Phase 14
- Languages beyond English
- More zones and a higher level cap
