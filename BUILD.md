# Lorehallow Build Plan

The single source of truth for what gets built, in what order, and what "done" means. It takes Lorehallow from an empty repository (Phase 0) to the public v1.0 release (Phase 19), then a post-1.0 track that splits the server across machines when measurements call for it (Phase 20). Agents work through it and check boxes as work lands, and lorehallow.com reads it on every push to render the live progress tracker.

Read [AGENTS.md](AGENTS.md) before changing this file.

## How this file works

- **Phases run in order.** The current phase is the lowest-numbered phase with an unchecked box. Within a phase, sections run roughly top to bottom, but independent items can be done in any order or in parallel.
- **Website first.** Phase 1 is finished, live and polished before any more work on the server phases (the owner's call, AGENTS.md decision 38). Until then, server work is limited to keeping `make check` green and the live server healthy.
- **Two frontend migrations** ([ADR 0014](docs/decisions/0014-astro-website-and-phaser-browser-client.md)). Phase 1 moves lorehallow.com from the Go site generator to Astro; its first item is the parity inventory in 1.2. Phase 2 replaces the Godot client with a Phaser browser game; its first client item is the Phaser scaffold in 2.6. While Phase 1 waits only on an owner approval, the Phaser work in `client/` (2.6, the client map import in 2.2, 2.7, 2.8 and the Godot retirement in 2.10) may go ahead, because none of it changes lorehallow.com. Deploying the game waits until Phase 1 is done. Superseded checked work is kept as history in [Superseded work](#superseded-work).
- **Performance criteria are measured, never estimated.** They point at targets in [docs/design/scaling.md](docs/design/scaling.md), name the hardware (`dev-m5` until the production host exists, then `prod`) and are checked only with a result in `docs/perf/`.
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
| 1 | lorehallow.com | The website, rebuilt in Astro | Linux hosting, DNS, TLS, static sites, migrating a live site, deploys and rollback |
| 2 | Two Players, One World | v0.2.0, playable in the browser | WebSockets, connection lifecycle, backpressure, tick loops, interpolation, a browser client |
| 3 | Many Players | v0.3.0 | Prediction, interest management, bandwidth, load testing, profiling |
| 4 | Accounts and Characters | v0.4.0 | Auth, sessions, PostgreSQL, login storms, backups |
| 5 | Chat and Presence | v0.5.0 | Fan-out messaging, rate limiting |
| 6 | Combat | v0.6.0 | Server-side game logic, AI, tick budgets |
| 7 | Crowds and a Larger World | v0.7.0 | Spatial queries, bandwidth budgets, graceful degradation |
| 8 | Items and Loot | v0.8.0 | Transactional persistence, data integrity, idempotency |
| 9 | Quests, NPCs and Progression | v0.9.0 | Data-driven content |
| 10 | Grouping | v0.10.0 | Cross-entity state, consistency |
| 11 | Zones | v0.11.0 | Ownership boundaries, handoff, failure isolation |
| 12 | Operations | v0.12.0 | Observability, soak testing, zero-downtime deploys |
| 13 | Classes and the v1 World | v0.13.0 | Content production, balance tooling |
| 14 | Audio, UI and Polish | v0.14.0 | Game feel, UX, accessibility |
| 15 | Browser Releases | v0.15.0 | Release engineering for a web client: versioned builds, compatibility, cache-safe updates, rollback |
| 16 | Security and Hardening | v0.16.0 | Threat modeling, fuzzing, abuse prevention |
| 17 | Closed Alpha | v0.17.0 | Playtesting, triage, live tuning |
| 18 | Open Beta | v0.18.0 | Capacity planning at scale, community operations |
| 19 | Launch | v1.0.0 | Shipping and supporting a live game |
| 20 | Many Machines | v1.1.0 | Gateways, inter-process handoff, partial failure |

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
- [x] `make setup` installs project-local dependencies (Go modules, uv environments, Godot addons) and is safe to re-run.
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
- [x] Golden vectors: a Go program writes encoded messages and JSON descriptions to `proto/testdata/`; Go and GDScript tests decode and re-encode them and compare bytes.
- [x] `make proto` regenerates everything; `make check` fails when generated code is stale or `buf breaking` reports an unplanned breaking change against `main`.

### 0.6 Client skeleton

The Godot skeleton built here is superseded by the Phaser client (2.6, [ADR 0014](docs/decisions/0014-astro-website-and-phaser-browser-client.md)); its items are in [Superseded work](#superseded-work). The pixel rules it settled carry over.

- [x] Client pixel rules settled for the game: base viewport 640×360, integer scaling only, nearest-neighbor filtering, no mipmaps, positions snapped to whole pixels (AGENTS.md section 7).

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
- [x] `make sync` mirrors `art/processed/` into `client/assets/art/` and `web/assets/art/`.
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

### 0.9 Art Batch 02: brand and website art

- [x] Write Batch 02 prompts and manifest: logo wordmark, square emblem (favicon and app icon), hero key-art scene, 2 or 3 parallax background layers, a character lineup, section icons, a sheet of 20 progress-map landmarks (one per phase, from a starting campfire to a castle for v1.0), devlog header art, an Open Graph card background (1200×630) and 404 art.
- [x] `HUMAN` Generate Batch 02 and drop it into `art/inbox/batch-02/`.
- [x] Run intake on Batch 02 (same palette, larger logical sizes allowed for web art) and iterate until accepted.
- [x] Export the favicon set (16, 32, 180, 192 and 512 px) and the client app icon from the emblem.
- [x] `HUMAN` Approve the logo and hero art as the public face of the project.

### 0.10 Continuous integration

- [x] GitHub Actions workflow runs `make check` on every push to `main` (Linux), caching Go and uv.
- [x] CI status badge in README.md.
- [x] Document in AGENTS.md that CI runs on GitHub and Codeberg is a mirror.

### 0.11 Walkthrough

- [x] `docs/walkthroughs/phase-00-foundations.md`: repository tour, the command runner, how protobuf codegen works on both sides, and how the art pipeline turns generated images into true pixel art.
- [x] Devlog draft for Phase 0 in `docs/devlog-drafts/`.

### 0.12 Hand-drawn art

Agents draw the art by hand from now on ([ADR 0004](docs/decisions/0004-hand-drawn-pixel-art.md)). Build the tools, then redraw the Phase 0 art that isn't a large painted image.

- [x] `ADR` Hand-drawn pixel art by agents is the default; image generation only for large painted web images.
- [x] `art/src/` sources and `make art-draw BATCH=<id>`: `.pxl.yaml` pixel grids (legend by palette name, one grid per frame) and `.py` draw scripts, rendered deterministically into `art/processed/` via a `source:` manifest field, with intake's checks (size, palette, frames, baseline, pivot) and a 4x and 8x contact sheet per batch.
- [x] Drawing helpers in `tools/art` for draw scripts: palette by name, seeded random, pixel, line, rect and ellipse, 1 px outline, mirror, wrap-around drawing for seamless tiles, stamping `.pxl` parts, and compositing finished assets.
- [x] `make check` fails when a processed file no longer matches its hand-drawn source, and when a source uses an unknown color name or the wrong size.
- [x] Redraw the base terrain tileset (16 tiles): every tile seamless with itself and its variants, low contrast, checked repeated 3×3.
- [x] Redraw the grass-to-dirt and grass-to-flagstone transition sets so every edge and center tile self-tiles and the 3×3 patch closes.
- [x] Redraw the concept hero (32×48, feet on row 45, 28 to 32 px tall to the top of the head).
- [x] Draw the props the reference scene needs (a tree, a mossy rock, a lantern post, a cottage corner) and rebuild the style reference vignette (240×160) as a composite of the hand-drawn tiles, hero and props; `art/style/reference.png` follows.
- [x] Redraw the logo wordmark (192×64), the 32 and 16 px emblems, the 16 section icons and the 20 progress landmarks, then re-export the favicons and app icon.
- [x] Redraw the character lineup with five hand-drawn adventurers at the in-game 32×48 scale.
- [x] Keep the generated hero scene, parallax layers, devlog header, social card and 404 art; check they still sit well next to the hand-drawn art, and note any that should be regenerated.
- [x] Update the Phase 0 walkthrough and devlog draft to describe hand-drawn art.

### Acceptance criteria

- [x] On a fresh clone, `make doctor && make setup && make check` passes on macOS, and `make check` passes in CI on Linux.
- [x] Batch 02 assets are processed and ready in `art/processed/web/`.
- [x] The Phase 0 walkthrough exists and every `ADR` item in this phase has its ADR.

---

## Phase 1: lorehallow.com

> **Goal:** A fast, beautiful, self-hosted website at lorehallow.com that sells the dream, links to the public repositories, tracks BUILD.md automatically, hosts a devlog and points players to the browser game, built with Astro and no browser JavaScript, replacing the Go-generated site without breaking a URL.
> **Ships:** The lorehallow.com website, rebuilt in Astro
> **Lesson:** Linux server hardening, DNS, TLS, reverse proxies, static site generation, migrating a live site by parity checks and cutover, and atomic deploys with rollback
> **Depends on:** Phase 0 sections 0.2, 0.3 and 0.9 (the rest of Phase 0 can finish in parallel)

The Go-generated site was built first and is live ([ADR 0011](docs/decisions/0011-go-website.md)). The Astro migration ([ADR 0014](docs/decisions/0014-astro-website-and-phaser-browser-client.md)) starts with the parity inventory in 1.2 and a local Astro build. The Go site stays live until the Astro build passes the parity checks and the owner approves a separate cutover (1.9). Checked items that describe the live site stay checked: the parity checks keep them true across the cutover.

### 1.1 Host provisioning and hardening

- [x] `HUMAN` Fill in `deploy/HOST.md` (or give an agent the facts to write it): OS and version, CPU architecture, RAM and disk, network setup (public IP, or ports 80 and 443 forwarded if it's at home), and anything else already running on it.
- [x] `HUMAN` Point DNS A records (and AAAA if the host has public IPv6) for `lorehallow.com`, `www.lorehallow.com` and `play.lorehallow.com` at the host, and note the DNS provider in `deploy/HOST.md`.
- [ ] Write `deploy/bootstrap/host.sh`, an idempotent provisioning script: create the `deploy` and service users, key-only SSH with root login off, firewall allowing only 22, 80 and 443, automatic security updates, fail2ban or equivalent, Caddy, and the `/srv/lorehallow` and `/etc/lorehallow` layout.
- [ ] `HUMAN` Review and run the bootstrap script with sudo (or let an agent run it), and authorize SSH keys for agent sessions on the `deploy` user. CI never gets a key (AGENTS.md section 10).
- [x] `ADR` Reverse proxy: Caddy with automatic HTTPS (recommended). Config in `deploy/caddy/lorehallow.caddy` (a site file on the shared host, ADR 0009), site root `/srv/lorehallow/web/current`.
- [x] HTTPS works with valid certificates; HTTP redirects to HTTPS and `www` redirects to the apex.
- [x] Security headers (HSTS, CSP, X-Content-Type-Options, Referrer-Policy, Permissions-Policy), compression, and long cache lifetimes for fingerprinted assets.
- [x] Every host change documented in `deploy/HOST.md`.

### 1.2 Site scaffold

- [x] Design tokens generated from `art/style/palette.hex` into CSS custom properties, so the site and the game share one palette.
- [x] `ADR` Fonts: an open-licensed pixel font for headings and a readable body font, both self-hosted with license files committed.
- [x] Base layout: header with logo and navigation (Home, World, Progress, Devlog, Downloads, About), footer linking GitHub, Codeberg, RSS and the licenses.
- [x] Pixel image component: integer scaling, `image-rendering: pixelated`, explicit dimensions, lazy loading below the fold.
- [ ] Parity inventory in `docs/design/website-parity.md`, taken from the live Go-built site and `web/dist/`: every published URL (pages, phase pages, tag pages, posts, feeds, `/progress.json`, `/progress.svg`, `sitemap.xml`, `robots.txt`, favicons, the 404 page), the `www` and trailing-slash redirects, every generated asset (fingerprinted CSS, fonts, art, share cards), every meta tag, every output test and budget in `web/README.md`, and for each one how the Astro build is checked against it.
- [ ] `ADR` Frontend toolchain for `web/` and `client/`: one pinned Node LTS and one package manager (npm workspaces with one committed lockfile recommended), plus the TypeScript, lint, format and test tools both frontends use, pinned in `toolchain.env`, checked by `make doctor` and listed in AGENTS.md section 13. Node is for development, builds and CI only, never on the host.
- [ ] Astro project in `web/`, next to the Go generator until 1.9: Astro components, strict TypeScript (`astro check` and `tsc` clean), CSS and static output only, no UI framework integrations, URL settings (`trailingSlash`, `build.format`) that keep every current URL, stylesheets never inlined, prefetch, view transitions and client islands off. It writes to its own output folder so both builds can be compared until cutover.
- [ ] `make web-dev`, `web-build`, `web-test` and `web-lint` run the Astro project (`astro check`, the linters, formatter check, tests and the package manager's audit); the Go generator's targets keep running under other names until 1.9 retires them, so `make check` covers both sites throughout.
- [ ] `ADR` Fonts and share cards in the Astro build: port the pixel-font compiler (TrueType and WOFF) and the 1200×630 Open Graph card renderer to TypeScript (recommended), or keep them as a Go tool in `tools/` that writes committed files. Fonts and their licenses don't change, and the outputs match the Go generator's byte for byte or differ only as recorded in the parity inventory.
- [ ] Astro layout, navigation, footer, pixel image component and palette tokens and color roles, with the same markup, classes and CSS as the Go templates.

### 1.3 Landing page

- [x] Hero: key art with gentle CSS-only parallax layers, the wordmark, a one-line pitch and calls to action (Follow the build, Read the devlog, View the source).
- [x] "A world that's already there": what Lorehallow is and the feeling it's going for, in plain language.
- [x] Design pillars with pixel icons.
- [x] "What you'll do" teaser: explore, quest, group up and fight with other people around you, clearly marked as planned.
- [x] Live progress widget: overall progress as an XP bar, the current phase, and a link to /progress.
- [x] Latest three devlog posts.
- [x] Art showcase strip built from processed art.
- [x] "Built in the open" section linking the GitHub and Codeberg repositories and BUILD.md.
- [x] Ambient touches (drifting fireflies, lantern flicker) in CSS only, switched off under `prefers-reduced-motion`.
- [x] No JavaScript: the page has no scripts at all, and everything above is plain HTML and CSS.
- [ ] The landing page rebuilt in Astro, matching the Go-built page section by section, including the CSS-only motion and its `prefers-reduced-motion` switch.

### 1.4 Progress tracker (/progress)

- [x] Parser unit tests, including malformed input, and a `buildmd-lint` command in `make check` that fails on any format violation.
- [x] Overall progress XP bar and per-phase progress bars, computed from the boxes.
- [x] Quest map: one landmark per phase along a winding road; finished landmarks lit, the current one marked "You are here", future ones dimmed.
- [x] Hand-draw a 21st landmark for Phase 20 (a far road beyond the castle) to join the 20 from Batch 02.
- [x] Phase pages (`/progress/phase-N`) with goal, lesson, sections, checklists and `HUMAN` and `ADR` badges.
- [x] Achievement feed: at build time, walk the `git log` of BUILD.md to find when each box was checked, and show the most recent as "quest complete" entries with dates and commit links.
- [x] "Waiting on the owner" panel listing unchecked `HUMAN` items in the current phase.
- [x] Machine-readable `/progress.json` and a `/progress.svg` badge, with the badge added to README.md.
- [x] Each phase links to its walkthrough on GitHub once it exists.
- [ ] TypeScript BUILD.md parser in the Astro project, one module imported by Astro's build and by `make buildmd-lint` (run by Node directly, no build step), implementing the format contract above and reporting every problem with its line number.
- [ ] Parser parity: on the real BUILD.md, every fixture in the Go parser's tests and its fuzz seed corpus, and randomly broken copies of BUILD.md, the TypeScript parser returns the same phases, sections, items, tags, checked state and problems as the Go parser. The test runs in `make check` until the Go parser is retired.
- [ ] /progress, the phase pages, the quest map, the git-backed achievement feed (walking `git log -p BUILD.md`, so builds need full history), the "Waiting on the owner" panel and the walkthrough links rebuilt in Astro with the same content and URLs.
- [ ] `/progress.json` and `/progress.svg` as Astro build-time endpoints that match the Go output for the same commit (the README badge and the uptime check read them).

### 1.5 Devlog (/devlog)

- [x] Posts with `draft: true` show only in `serve` and are never written by `build`.
- [x] Index, tag pages and post pages with pixel-art styling and comfortable reading typography.
- [x] RSS and Atom feeds.
- [x] Move the Phase 0 devlog draft into the collection.
- [ ] `HUMAN` Edit and publish the first devlog post.
- [ ] Devlog as an Astro content collection reading `web/content/devlog/` in place: the same front-matter schema, a build failure naming the file and field for bad front matter, the same slugs, and raw HTML in posts still dropped.
- [ ] Drafts appear only in `make web-dev`. A test proves no draft reaches any public artifact: pages, tag pages, feeds, the sitemap, share cards or any file in the build output.
- [ ] Devlog index, tag and post pages, plus RSS and Atom as build-time endpoints, with the same URLs, entry ids and dates, so feed readers see no duplicates after the cutover.

### 1.6 Play (/play)

Downloads is replaced by a Play entry point: Lorehallow is played in the browser at play.lorehallow.com, and there are no native builds ([ADR 0014](docs/decisions/0014-astro-website-and-phaser-browser-client.md)).

- [ ] /play page with no script: before the browser game is live, it says plainly that nothing is playable yet, that playing will mean opening play.lorehallow.com with nothing to install, and links to /progress and the devlog. Once v0.2.0 is live (2.10), it links to play.lorehallow.com with the current version and the supported browsers.
- [ ] `/downloads/` permanently redirects (301, in the Caddy site file) to `/play/` once /play is live, and the redirect stays; the sitemap lists /play and drops /downloads.
- [ ] Navigation and footer say Play instead of Downloads, and the landing page's calls to action, the FAQ (platforms: any current desktop browser; when can I play) and the about page's tech stack describe browser play, not downloads.
- [ ] Replace `web/data/releases.json` (platform files, sizes, checksums) with the release data /play and the patch notes need (version, date, channel, notes link, protocol version), with its schema documented in `web/data/README.md` and validated at build time.
- [ ] The deploy health check loads `/play/` and expects the `/downloads/` redirect, instead of loading /downloads.

### 1.7 More pages

- [x] /world: the setting and mood (mossy forests, lantern-lit towns, old stone ruins) with an art gallery, marked as early and subject to change.
- [x] /about: the project's goals (a small world worth being in, an excellent Go MMO server as the showcase, and learning by building it), building in the open, how it's made (the owner's direction, AI agents writing the code and hand-drawing the pixel art, a few large painted images generated and run through the processing pipeline) and the tech stack.
- [x] /faq: is it free, when can I play, which platforms, how many players (the goal of hundreds to thousands, stated as a target, with only measured numbers given as facts), how to follow along.
- [x] Custom 404 page with the hand-drawn 404 art (`web/not_found.png`: the hero lost at a broken signpost).
- [x] Open Graph and social card meta on every page with generated share images, plus `sitemap.xml`, `robots.txt` and canonical URLs.
- [x] No third-party trackers, scripts or fonts on any page.
- [ ] /world, /about, /faq and the 404 page rebuilt in Astro with the same content and art; Caddy serves the 404 page with a real HTTP 404 status.
- [ ] Social metadata parity: every page keeps its title, description, canonical URL, Open Graph and Twitter tags and its 1200×630 share card; `sitemap.xml` and `robots.txt` are build-time endpoints with the same entries.

### 1.8 Quality gates

- [ ] Lighthouse audit on the live Astro site after the cutover (1.9), on the landing page, /progress and a devlog post: 95 or higher for performance, accessibility, best practices and SEO on mobile, recorded in `docs/perf/`. Lighthouse runs locally under Node against the live site; it isn't part of `make check` or CI.
- [ ] The Go output tests ported to TypeScript and run on the Astro build in `make web-test`: every check listed in `web/README.md` (no scripts, iframes, inline handlers, `style` attributes or `javascript:` URLs; alt text and whole-number image sizes; links, fragments and CSS `url()`s resolve; no other origins; the 1 MB landing page budget; WCAG AA pairs and colors only through roles; titles, descriptions, canonical URLs, Open Graph cards and repository links on every page; valid feeds and sitemap; no drafts; valid release data).
- [ ] Parity check, run in `make check` until 1.9: builds both sites and compares URL sets, page text, meta tags, feeds, `/progress.json`, `/progress.svg`, the sitemap and asset references. Every difference is fixed or listed as accepted in the parity inventory.
- [x] Output tests on the built site in `make web-test`: no `<script>` tags or inline event handlers on any page, alt text and explicit dimensions on every image, and WCAG AA contrast for every palette pair the CSS uses.
- [x] Accessibility: alt text on all art, full keyboard navigation, visible focus, WCAG AA contrast.
- [x] Responsive from 360 px phones to 4K screens, with art always scaled by whole numbers.
- [x] Internal link checker in `make check`.
- [x] Landing page transfers under 1 MB, enforced by a test on the built output.

### 1.9 Deploys and cutover

- [x] `deploy/web/deploy.sh`: build, upload to a new timestamped release directory, switch the `current` symlink atomically, keep the last five releases.
- [x] `make deploy-web` and `make rollback-web` for manual use.
- [x] `ADR` Uptime monitoring that alerts the owner (for example ntfy) when the site has been down for five minutes.
- [x] `HUMAN` Pick an ntfy topic, subscribe to it on your phone and add it as the `NTFY_TOPIC` secret (docs/ops/website.md), then send a test alert.
- [ ] `deploy/web/deploy.sh` builds the Astro site on the machine running the deploy (Node never goes to the host) and keeps the same release model: timestamped release, atomic `current` switch, health check through the host's Caddy, automatic rollback, five releases kept.
- [ ] The deploy health check covers what Astro changes: hashed assets load with immutable cache headers and pages revalidate, canonical URLs and share-card URLs are absolute and correct, the RSS and Atom feeds, `/progress.json` and `/progress.svg` load, `/play/` loads, `/downloads/` redirects, and a missing page returns HTTP 404.
- [ ] Caddy site file for the Astro output: cache rules match Astro's asset folder, the public site's CSP stays `script-src 'none'` and is kept separate from play.lorehallow.com's game CSP, and the `/downloads/` redirect; validated as the caddy user before every reload ([ADR 0009](docs/decisions/0009-caddy-on-a-shared-host.md)).
- [ ] `HUMAN` Approve the cutover after reviewing the local Astro build and the parity report. The Go-generated site stays live until then.
- [ ] Cutover: deploy the Astro build with `make deploy-web`, confirm the live checks, keep the last Go-built release as the rollback target, and rehearse the automatic rollback with an Astro release (`make rehearse-rollback-web`).
- [ ] Retire the Go web module once the cutover has held for a week with no rollback: delete its Go code, templates, `go.mod` and `go.sum`, its Make targets, its CI cache path and its govulncheck run, keep the parser fixtures the TypeScript tests use, and update every doc that describes it.

### 1.10 Launch

- [x] `docs/walkthroughs/phase-01-website.md`: DNS, TLS and Caddy, static builds, the deploy pipeline, and how BUILD.md becomes the progress page.
- [x] Devlog draft announcing the website.
- [ ] `docs/walkthroughs/phase-01-website-migration.md`: moving a live static site to a new generator without breaking a URL: the parity inventory, one shared parser, build-time endpoints, the separate cutover and the rollback target. The original Phase 1 walkthrough stays as history.
- [ ] `HUMAN` Review the live Astro site on desktop and phone after the cutover and approve it.

### Acceptance criteria

- [x] https://lorehallow.com serves the landing page with a valid certificate, and HTTP and `www` redirect correctly.
- [x] Every page links to the GitHub and Codeberg repositories.
- [ ] The devlog has at least one published post and a valid RSS feed.
- [ ] Lighthouse budgets pass and no page ships any JavaScript.
- [x] A deliberately bad deploy was rolled back with one command.
- [ ] lorehallow.com is served from the Astro build, and every URL in the parity inventory answers as it did before the cutover: the same content, the same redirects, and a real 404 for a missing page.
- [ ] /play exists and accurately says whether the game is playable, and /downloads redirects to it.
- [ ] No draft appears in any public artifact.
- [ ] The Go web module is gone, `make check` runs only the TypeScript site checks, and the progress tracker, feeds and badge still update on every deploy.

---

## Phase 2: Two Players, One World

> **Goal:** Two players on different machines open play.lorehallow.com in their browsers and see each other move smoothly around one shared 30×20 map, drawn by a Phaser client, on a connection layer that already survives slow and hostile clients and a bot swarm that measures it.
> **Ships:** v0.2.0, playable in the browser at play.lorehallow.com
> **Lesson:** WebSockets, message framing, connection lifecycle, backpressure, authoritative simulation, tick loops, interpolation, a browser game client, and deploying a Go service and a static game build behind TLS
> **Depends on:** Phase 0 and Phase 1 (website first). The Phaser client needs only the frontend toolchain ADR from 1.2, and its work in `client/` may go ahead while Phase 1 waits on an owner approval (see the preamble). The deploys in 2.9 and 2.10 wait until Phase 1 is done.

The server sections (2.1 to 2.5) were built first. The client is now a Phaser browser game ([ADR 0014](docs/decisions/0014-astro-website-and-phaser-browser-client.md)); the Godot skeleton from Phase 0 is retired in 2.10, once the Phaser checks cover it.

### 2.1 Protocol v1

- [x] Handshake: `ClientHello` (protocol version, client version, display name) and `ServerHello` (player id, tick rate, server tick, spawn state); mismatched protocol versions get a clear rejection.
- [x] `PlayerInput` (not `Input`, a Godot class name): sequence number, client tick and 8-way movement intent. Never a position.
- [x] `Snapshot`: server tick, the last input sequence processed for the receiving player, and entity states (id, position, facing, animation state).
- [x] `EntityJoined`, `EntityLeft`, `Ping` and `Pong`.
- [x] `Disconnect`: a reason code (protocol mismatch, server full, too slow, kicked, shutting down, timed out, bad message) and readable text, sent before the server closes a connection.
- [x] Framing, limits and a byte budget per message (one `Envelope` per binary WebSocket message, maximum sizes per direction) documented in `docs/design/protocol.md`.
- [x] Round-trip golden tests for every new message.
- [x] Hardened decoding: size checked before decoding, unknown payloads rejected, and fuzz targets for envelope decoding with a committed seed corpus, run by `make fuzz`.

### 2.2 Shared map format

- [x] `ADR` Map source of truth: Tiled JSON (`.tmj`) in `content/maps/` (recommended), read by the server for collision and spawn points and imported by the client for rendering.
- [x] Server map loader for tile layers, the collision layer and spawn points, tested against a small fixture map.
- [ ] Phaser client map import: loads `content/maps/*.tmj` and the processed tilesets synced into the client, draws the tile layers in file order, never draws the `collision` layer, and is tested against the same fixture map as the server loader.
- [x] Map validator in `make check`: tilesets exist, collision layer present, at least one spawn point.
- [x] `make sync` also mirrors `content/` into `client/assets/content/`.

### 2.3 Connection lifecycle and backpressure

- [x] `ADR` WebSocket library (coder/websocket recommended); endpoint `/ws` with origin checks, read limits and compression off.
- [x] Connection lifecycle as an explicit state machine (handshaking, in world, closing, closed): handshake timeout, player id assignment, spawn selection, and one cleanup path for every exit (close, error, timeout, eviction, shutdown).
- [x] One reader and one writer goroutine per connection: the reader decodes, validates and forwards to the zone's bounded inbox; the writer drains a send queue bounded by message count and bytes.
- [x] Slow-client eviction: a full send queue closes the connection with `Disconnect` ("too slow"), and the tick loop never waits on a socket (tested with a stalled reader).
- [x] Heartbeats: the server pings every connection, a read deadline drops silent ones, and round-trip time is recorded per connection.
- [x] Limits: maximum connections (beyond it, `Disconnect` "server full"), pending handshakes per IP, message size and inbound message rate, each with a test that proves the limit holds.
- [x] Graceful shutdown sends `Disconnect` ("shutting down") to everyone, drains, and leaves no goroutines behind (leak test).

### 2.4 Server simulation

- [x] One goroutine owns the zone: a fixed 20 Hz tick on a monotonic clock, measuring tick duration and warning on overruns, with no locks around world state.
- [x] Per-connection input queues drained each tick; inputs applied in sequence order; stale and duplicate inputs dropped.
- [x] Movement: fixed speed in tiles per second, axis-separated collision against the collision layer.
- [x] Validation: input floods, impossible speeds and wall-walking have no effect; repeat offenders are disconnected and logged.
- [x] Snapshot broadcast every tick to every connected player, each entity's state encoded once per tick and shared by every recipient (interest management comes in Phase 3).
- [x] `ADR` Snapshots carry only the entities that changed, and a player with nothing new gets no Snapshot but a keepalive each second (TCP is reliable, so no acknowledged baselines), measured before and after with `make loadtest`.
- [x] Disconnects remove the entity for everyone within one second (`EntityLeft` on close, heartbeat timeout for silent drops).
- [x] Deterministic tests of the simulation with a fake clock: movement, collision, validation, join and leave.
- [x] Benchmarks with `ReportAllocs` for the tick, snapshot encoding and input decoding, and an allocation test that fails `make check` if the steady-state simulation step allocates.

### 2.5 Bots and load generation

- [x] `server/cmd/lorehallow-bot`: a headless Go client that connects, handshakes and wanders; `-n` runs many from one process cheaply, and it reports its own CPU and memory.
- [x] Cheat mode for the bot: input flooding, forged sequence numbers, oversized messages, unknown message types and a handshake that never finishes.
- [x] Integration test proving cheat mode never moves a bot faster than allowed or through walls.
- [x] Integration test: server plus two bots; each sees the other move, and sees the leave within one second of a disconnect.
- [x] `make loadtest SCENARIO=<name>` runs a [scaling.md](docs/design/scaling.md) scenario against a fresh server and writes a draft result to `docs/perf/`; `idle` and `spread` work.
- [x] Baseline on `dev-m5`: Case 1 Tier 1 (100 idle connections) and 100 bots in `spread`, recorded in `docs/perf/` with memory per connection, goroutines, tick p99 and bytes per player.

### 2.6 Phaser browser client

The first client task is the scaffold. It shares `client/` with the Godot project until 2.10, so Godot must be kept out of the Phaser folders and `make check` runs both clients' checks in the meantime.

- [ ] Phaser scaffold in `client/`: Phaser, strict TypeScript (`strict` plus unchecked index access; `tsc --noEmit` clean) and Vite on the 1.2 frontend toolchain, the folder layout from AGENTS.md section 7, and lint, format check and unit tests in `make check`.
- [ ] Local development: `make client-dev` serves the client with Vite and proxies `/ws`, `/healthz`, `/version` and `/api/` to the local Go server, so the dev page is same-origin like production; `make client-build` writes the static build with the version, commit and protocol version embedded.
- [ ] TypeScript protobuf bindings generated by `make proto` with buf (committed, marked generated), plus `PROTOCOL_VERSION` as a constant; a test decodes and re-encodes every golden vector in `proto/testdata/` byte for byte and checks its JSON description, in `make check`.
- [ ] Crisp integer pixel scaling: a 640×360 base, the canvas sized to the largest whole-number multiple that fits the window and letterboxed, never stretched by CSS; nearest-neighbor sampling, rounded pixels and a pixel-snapped camera; a test covers the scale choice at common window sizes.
- [ ] Title screen showing the build version and commit (replacing the Godot one).
- [ ] Dev art scene that draws every synced image at 1x to 4x with a 3×3 seam patch per tileset, and `make client-art-check` rendering it in a headless browser and checking every pixel against the processed art.
- [ ] Connect screen: server choice (play.lorehallow.com by default, plus local), display name, connect button and clear error states, including each `Disconnect` reason and its close code.
- [ ] Net module: a browser WebSocket with binary frames, envelope encode and decode through the generated bindings, reconnect with jittered exponential backoff, and typed events per message type.
- [ ] Input sampled at a fixed rate and sent as `PlayerInput` messages with sequence numbers.
- [ ] Local player drawn at the server's authoritative position (prediction comes in Phase 3).
- [ ] Remote players drawn about 100 ms behind the latest snapshot, interpolating between buffered snapshots, with capped extrapolation for short gaps.
- [ ] Idle and walk animations per direction, driven by movement state.
- [ ] Debug overlay (F3): RTT, snapshot rate, interpolation buffer depth, server tick and entity count.
- [ ] Dev network simulation: configurable delay, jitter and drops, on the server and in the client.
- [ ] Browser smoke test: starts the real Go server and two headless browser sessions of the client build; each sees the other join and move, and sees the other leave within one second of closing. It runs in `make check` and CI.

### 2.7 Art Batch 03: player character and starter tiles

- [ ] Draw Batch 03 by hand (manifest and sources in `art/src/`): the player character sheet (32×48, four directions, idle 2 frames, walk 4 frames) and the extra tiles the starter map needs (trees, rocks, path edges, water edges and a few props).
- [ ] Run intake, check frame alignment, iterate until accepted, and `make sync`.
- [ ] Character animations in the Phaser client built from the sheet's sidecar JSON (frame size, order and pivot); placeholder art fully replaced.

### 2.8 Starter map

- [ ] Hand-build a 30×20 map from Batch 01 and 03 tiles, with paths, obstacles worth testing collision against, and at least four spawn points.
- [ ] Camera follows the local player with pixel-snapped movement and stops at map edges.

### 2.9 Deploy the game server

- [x] systemd unit `deploy/server/lorehallow-server.service`: dedicated user, restart on failure, sandboxing options, `LimitNOFILE` raised for many connections, env file at `/etc/lorehallow/server.env`.
- [x] Caddy site for `play.lorehallow.com` proxying `/ws` WebSocket upgrades to the server on localhost.
- [x] `make deploy-server`: cross-compile for the host, upload, switch, restart, check `/healthz`, and roll back automatically on failure.
- [ ] Public `GET /api/status` (online count, uptime, version, protocol version), cached briefly, proxied by Caddy at `play.lorehallow.com/api/status` and, read-only, at `lorehallow.com/api/status`.
- [ ] Server state and player count shown on the game's own pages at play.lorehallow.com; lorehallow.com stays script-free and shows only what is known at build time, linking to the game for live numbers.
- [x] Logs in journald, and `docs/ops/runbook.md` started with start, stop, logs, deploy and rollback commands.
- [ ] Caddy serves the client's static build at play.lorehallow.com from `/srv/lorehallow/client/current`, keeping `/ws`, `/healthz`, `/version` and `/api/` proxied to the Go server; `index.html` revalidates and hashed assets are cached as immutable.
- [ ] Game CSP on play.lorehallow.com, separate from the website's: scripts only from `'self'`, no inline scripts and no `unsafe-eval`, `connect-src` limited to the game's own origin and `wss://play.lorehallow.com`, and only the image and media sources Phaser needs, each with a reason in the site file.
- [ ] `make deploy-client` and `make rollback-client`: build the client on the machine running the deploy, upload a timestamped release, switch `current` atomically, check that the page, its assets and the `/ws` handshake work through Caddy, roll back on failure, keep five releases; documented in `docs/ops/runbook.md`.

### 2.10 Browser play and retiring Godot

- [ ] The v0.2.0 browser build is live at play.lorehallow.com (`make deploy-client`), and /play on lorehallow.com links to it as "Prototype".
- [ ] Remove the Godot client once the Phaser checks (2.6's golden vector test, art check and browser smoke test) pass in `make check`: the Godot project files, GDScript sources and GUT tests, the GDScript bindings in `client/net/pb/` and their generation in `tools/scripts/gen-proto.sh`, the GUT and godobuf addons and their install script, export presets and templates, and the Godot import cache rules.
- [ ] Remove Godot from the tooling in the same change: the Godot, GUT, godobuf and gdtoolkit pins in `toolchain.env`, their `make doctor` and `make setup` steps, the Godot Make targets and variables, the Godot steps in CI, and the AGENTS.md toolchain rows. `make check` stays green before and after.
- [ ] Update every doc that describes the Godot client (client and proto READMEs, AGENTS.md, README.md, the protocol design note) and mark the remaining Godot decisions superseded; the Phase 0 walkthrough stays as history.

### 2.11 Walkthrough

- [ ] `docs/walkthroughs/phase-02-two-players.md`: the tick loop, the snapshot format, interpolation, the connection lifecycle and backpressure, authoritative validation, the browser client and the deploys, with pointers into the code.
- [ ] Devlog draft for v0.2.0 with a short clip of two players moving.
- [ ] Release notes in `docs/releases/v0.2.0.md` and tag `v0.2.0`.

### Acceptance criteria

- [ ] `HUMAN` Playtest: two people on different machines open play.lorehallow.com in their browsers and connect to the server over `wss://`, with nothing to install.
- [ ] The browser smoke test passes in CI: two sessions see each other move through the real Go server.
- [ ] The repository has no Godot project, GDScript, Godot addons or Godot toolchain pins, and `make doctor` doesn't ask for Godot.
- [ ] Each player sees the other moving smoothly with no teleporting or jitter under normal latency, and also with 100 ms simulated latency and 20 ms jitter.
- [ ] Closing a client removes that player for everyone within one second.
- [ ] A modified client sending forged or flooded input cannot move faster than allowed (integration test).
- [ ] A stalled client is evicted without raising anyone else's tick time, and the server shuts down with no goroutines left behind (integration tests).
- [x] Case 1 Tier 1 in [scaling.md](docs/design/scaling.md) (100 idle connections) is met on `dev-m5`, with the result in `docs/perf/`.
- [ ] The walkthrough explains the tick loop, the snapshot format, interpolation and backpressure.

---

## Phase 3: Many Players

> **Goal:** 1,000 bots and 10 or more real players share the world, your own movement feels instant thanks to prediction and reconciliation, and each player only receives what is near them.
> **Ships:** v0.3.0
> **Lesson:** Latency hiding, client prediction, reconciliation, interest management, bandwidth, load testing and profiling
> **Depends on:** Phase 2

### 3.1 Shared deterministic movement

- [ ] `ADR` Position representation: fixed-point integers (for example 1/16 pixel) so client and server compute identical results.
- [ ] Movement extracted into a pure step function with identical rules on both sides, documented in `docs/design/movement.md`.
- [ ] Shared test vectors: Go generates input sequences and expected positions; the TypeScript client replays them and must match exactly.

### 3.2 Prediction and reconciliation

- [ ] Client applies local input immediately and keeps a buffer of unacknowledged inputs.
- [ ] On each snapshot, the client rewinds to the server's state for its player and replays pending inputs.
- [ ] Small errors are smoothed out visually; large errors snap. Corrections are counted in the debug overlay.
- [ ] Prediction can be toggled from the debug menu for side-by-side comparison.

### 3.3 Interest management

- [ ] `ADR` Spatial index: a uniform grid (recommended) with a cell size chosen by measurement, updated as entities move.
- [ ] Per-player area of interest with hysteresis (enter radius smaller than leave radius) so entities don't flicker at the edge.
- [ ] Enter and leave messages driven by area changes; snapshots include only entities in the area.
- [ ] Tests at cell boundaries and with fast movement, and a benchmark of grid updates and queries with 1,000 entities.

### 3.4 Bandwidth

- [ ] Measure bytes per snapshot and per client per second, shown in the debug overlay and exported by the server.
- [ ] `ADR` Snapshot optimization on top of ADR 0008's changed-only snapshots: quantization and smaller encodings of changed entities (for example position deltas). Implement it and measure the savings.
- [ ] Encode once with interest management: each changed entity encoded once per tick and shared by every recipient that can see it, with a benchmark showing encode cost independent of recipient count.
- [ ] Per-client bandwidth with 100 bots spread across the map, before and after interest management, in `docs/perf/phase-03.md`.

### 3.5 Metrics and profiling

- [ ] Prometheus text-format metrics on a localhost-only admin port: connections, players per zone, tick duration histogram, inbox and send-queue depth, drops, evictions, messages and bytes in and out, goroutines and GC.
- [ ] pprof on the same localhost-only admin port; one profiling pass at 1,000 bots completed and any hotspot fixed, with before and after in `docs/perf/`.
- [ ] `ADR` GC settings: measure `GOGC` and `GOMEMLIMIT` under Case 2 load and record the chosen defaults.

### 3.6 Load testing

- [ ] Bot swarm: `lorehallow-bot -n 100` with wander, follow and idle behaviors and configurable latency.
- [ ] The bot generator runs 5,000 bots from one process, with its own CPU and memory reported, and can spread bots across several source addresses to get past ephemeral-port limits.
- [ ] `make loadtest` covers the scaling.md scenarios `idle`, `spread`, `crowd`, `churn`, `slow` and `hostile` (`storm` and `reconnect` join in Phase 4, `burst` in Phase 6).
- [ ] Load test script reporting tick time p50 and p99, CPU, memory and bandwidth at 10, 50 and 100 bots, with results in `docs/perf/phase-03.md`.
- [ ] Every Tier 1 target in scaling.md, and Case 1 Tier 2 (1,000 connections), measured on `dev-m5` and recorded in `docs/perf/phase-03.md`.
- [ ] Race detector clean during a load test run.
- [ ] One-hour soak (scaling.md short soak) on `dev-m5`.

### 3.7 Art Batch 04: a crowd of people

- [ ] Draw Batch 04 by hand: three or four player appearance variants and a townsfolk placeholder, so a crowd reads as different people.
- [ ] Intake and integrate; bots pick random appearances.

### 3.8 Walkthrough

- [ ] `docs/walkthroughs/phase-03-many-players.md`: prediction, reconciliation, fixed-point simulation, interest management, and measuring and reducing bandwidth.
- [ ] Devlog draft, release notes, tag `v0.3.0` and deploy the browser build.

### Acceptance criteria

- [ ] With 150 ms simulated latency, your own movement responds on the next frame and corrections aren't noticeable in normal play.
- [ ] With 100 bots spread across the map, per-client bandwidth is at least 70% lower than broadcasting everything.
- [ ] Case 1 Tier 2 (1,000 connections), Case 2 Tier 1 (100 active players) and Case 3 Tier 1 (100 in one area) in scaling.md are met on `dev-m5`.
- [ ] With 10% slow clients and 100 hostile bots, healthy players' tick p99 and bandwidth stay within 10% of a clean run (scaling.md).
- [ ] The one-hour soak passes: no memory growth, flat goroutines, no tick drift.
- [ ] 100 bots on the production host keep server tick p99 under 12.5 ms (a quarter of the tick budget).
- [ ] `HUMAN` Playtest: at least 3 real players plus 200 bots on the live server feels smooth.
- [ ] The walkthrough explains prediction, reconciliation and interest management with diagrams.

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
- [ ] A bounded connection pool, a context timeout on every query, and no database call on a zone goroutine: zones hand saves to persistence workers through a bounded queue.

### 4.2 Authentication

- [ ] Registration: username rules, password of at least 10 characters checked against a common-password list, Argon2id hashing with parameters recorded in an ADR.
- [ ] HTTPS endpoints `POST /api/auth/register`, `/login` and `/logout` returning an opaque session token, stored only as a hash.
- [ ] The WebSocket handshake carries the session token; invalid or expired tokens are rejected; a new login ends any older game session for that account.
- [ ] Rate limits and lockouts on login and registration per IP and per account, proven by tests.
- [ ] Password hashing on a bounded worker pool, so a login storm queues instead of starving tick loops; `make loadtest SCENARIO=storm` meets the scaling.md login-storm target on `dev-m5`.
- [ ] `HUMAN` Decide on account recovery for v1 (no email, or optional email with an SMTP provider). An agent writes the options up as an ADR first.

### 4.3 Characters

- [ ] Character creation: name rules (unique, allowed characters, length, reserved and offensive words filtered) and appearance chosen from the Batch 04 variants.
- [ ] Character select: up to four characters per account, with delete behind a typed confirmation (soft delete).
- [ ] Position and state saved on logout, on disconnect and every 60 seconds, and restored on login.
- [ ] A failure in the middle of a save (connection killed, transaction aborted, server killed) leaves the last committed state intact, tested against a real Postgres.
- [ ] Session resume: a client that reconnects within a grace window with its session token returns to its character without logging in again; `make loadtest SCENARIO=reconnect` meets the scaling.md reconnect-storm target on `dev-m5`.
- [ ] Client login, registration, character select and creation screens with clear errors; the session token lives in memory, or in a Secure, HttpOnly, SameSite=Strict cookie scoped to play.lorehallow.com when "remember me" is chosen, never in script-readable storage.

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
- [ ] Devlog draft, release notes, tag `v0.4.0` and deploy the browser build.

### Acceptance criteria

- [ ] A new player can register, create a character, play, quit, and log back in at the same spot.
- [ ] Passwords exist only as Argon2id hashes and session tokens only as hashes.
- [ ] Brute-force login attempts are rate limited (integration test).
- [ ] A backup was restored successfully from the off-site copy.
- [ ] CI runs the database integration tests against a real Postgres.
- [ ] The login-storm and reconnect-storm targets in scaling.md are met on `dev-m5`, with results in `docs/perf/`.

---

## Phase 5: Chat and Presence

> **Goal:** Players can talk and see who's around: say, zone chat, whispers, nameplates, emotes and a who list.
> **Ships:** v0.5.0
> **Lesson:** Fan-out messaging, rate limiting and moderation basics
> **Depends on:** Phase 4

### 5.1 Chat on the server

- [ ] Channels: say (range-limited), zone, whisper (by character name), emote and system, with server timestamps.
- [ ] Fan-out by channel with length limits and per-player token-bucket rate limits that reply with a friendly "slow down".
- [ ] Each chat message encoded once for all its recipients, with a benchmark of zone chat to 1,000 players.
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
- [ ] Intake and integrate, with one UI theme module in the client used by all UI.

### 5.5 Walkthrough

- [ ] `docs/walkthroughs/phase-05-chat.md`: fan-out, rate limiting and moderation design.
- [ ] Devlog draft, release notes, tag `v0.5.0` and deploy the browser build.

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
- [ ] Tick time broken down by system (input, movement, AI, combat, snapshots) in the metrics, with a budget for each.

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
- [ ] Deterministic combat: the same inputs and random seed produce the same combat log, tested tick by tick.
- [ ] `make loadtest SCENARIO=burst`: every player in a zone uses an ability on the same tick, within the scaling.md burst target on `dev-m5`.

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
- [ ] Devlog draft, release notes, tag `v0.6.0` and deploy the browser build.

### Acceptance criteria

- [ ] A player can target, fight and kill a mob, which respawns on its timer.
- [ ] A player can die and respawn at the respawn point.
- [ ] Abilities can't be used early, out of range or without resources, even by a modified client (tests).
- [ ] Mobs leash and reset when kited too far.
- [ ] 100 bots fighting mobs still meet Case 2 Tier 1 in scaling.md on `dev-m5`.
- [ ] `HUMAN` Playtest: two players fighting the same mob feels good and reads clearly.

---

## Phase 7: Crowds and a Larger World

> **Goal:** A town square holds hundreds of players who can all see each other, with each player's bandwidth held to a budget, on a much larger map where spatial queries drive chat, combat and AI.
> **Ships:** v0.7.0
> **Lesson:** Spatial queries, bandwidth budgets, priority and update-rate tiers, and graceful degradation under overload
> **Depends on:** Phase 6

### 7.1 Spatial queries everywhere

- [ ] Say chat range and combat events use the same spatial queries.
- [ ] Mob aggro, leashing and AI perception use the zone's spatial grid instead of scanning every entity.
- [ ] Benchmarks of the grid's queries with 1,000 players and 1,000 mobs in one zone.

### 7.2 Bandwidth budgets for crowds

- [ ] Per-player byte budget with a priority accumulator: nearby, moving and targeted entities first, everyone else as the budget allows (scaling.md Case 3).
- [ ] Update-rate tiers by distance (every tick, every second tick, every fourth tick), and a cap on full-rate entities per player.
- [ ] Graceful degradation under overload: as a zone's tick nears its budget, the server lowers update rates for distant entities first and logs it; game rules never degrade.
- [ ] Tests: the budget is never exceeded, every visible entity is updated at least at its tier's minimum rate, and nothing flickers at area edges.

### 7.3 A larger map

- [ ] Build a large map (about 200×150 tiles) with a town, a forest edge and ruins, as the first real zone layout.
- [ ] `ADR` Measure client rendering and memory on the large map, and decide whether chunked loading is needed.
- [ ] Minimap showing nearby terrain and other players.

### 7.4 Art Batch 07: zone tiles

- [ ] Draw Batch 07 by hand: forest variants, town building exteriors, lanterns, ruins, water and bridges.
- [ ] Intake and integrate.

### 7.5 Measurement

- [ ] `make loadtest SCENARIO=crowd` at Case 3 Tier 1 and Tier 2 (100 and 300 players in one area) on `dev-m5`, before and after the budgets, in `docs/perf/phase-07.md`.
- [ ] Case 3 Tier 3 (1,000 in one area) run as a degradation test: the server stays up, holds its budgets and recovers when the crowd disperses.
- [ ] Client frame rate with 300 players on screen measured, and fixed if it can't hold 60 FPS on `dev-m5`.

### 7.6 Walkthrough

- [ ] `docs/walkthroughs/phase-07-crowds.md`: why fan-out is O(n²), spatial partitioning, hysteresis, priority and bandwidth budgets, and degrading gracefully.
- [ ] Devlog draft, release notes, tag `v0.7.0` and deploy the browser build.

### Acceptance criteria

- [ ] 300 players in one area meet Case 3 Tier 2 in scaling.md on `dev-m5`: tick p99 under 25 ms, at most 32 KB/s per player, every visible entity updated at least 5 times a second.
- [ ] Entities don't visibly pop in or out at area edges during normal movement.
- [ ] Server tick p99 stays under 12.5 ms with 100 bots spread across the large map.

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
- [ ] The client is told about an item change only after it commits, so a crash between the game event and the commit never shows an item the player doesn't have.
- [ ] Request ids survive reconnects: a command replayed after a reconnect is applied once.
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
- [ ] Devlog draft, release notes, tag `v0.8.0` and deploy the browser build.

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
- [ ] Devlog draft, release notes, tag `v0.9.0` and deploy the browser build.

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

- [ ] Party model: invite, accept, decline, leave, kick, promote and disband, up to five members, owned by one goroutine outside any zone and changed only by messages.
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
- [ ] Devlog draft, release notes, tag `v0.10.0` and deploy the browser build.

### Acceptance criteria

- [ ] Five players form a party, beat the elite camp, and all get quest credit and experience.
- [ ] Disconnecting and reconnecting within the grace period keeps party membership.
- [ ] Concurrency tests: simultaneous invites, kicks and leaves never corrupt a party.
- [ ] `HUMAN` Playtest: a group of three or more takes on the elite camp together.

---

## Phase 11: Zones

> **Goal:** The world spans multiple maps with quick transitions; each zone is an isolated simulation owned by its own goroutine, and handoff is a message protocol ready to cross processes later.
> **Ships:** v0.11.0
> **Lesson:** Ownership boundaries, handoff protocols and failure isolation
> **Depends on:** Phase 10

### 11.1 Architecture

- [ ] `ADR` Zone architecture: one process with many zone goroutines and a world coordinator (ADR 0005), with the internal messages between them written down in `docs/design/zones.md` so they can run between processes later (Phase 20). Write up the tradeoffs.
- [ ] Zone registry and routing in the world coordinator; a connection belongs to exactly one zone at a time.
- [ ] Handoff protocol: save state, transfer and spawn in the new zone, rolling back to the old zone on failure, tested with a failure injected at every step.
- [ ] Zone isolation: a panic in one zone's tick is recovered and logged and stops only that zone; its players are moved to a safe point or disconnected cleanly.
- [ ] Zone and world chat, whispers and parties work across zones.
- [ ] Zones start on demand and an empty zone stops ticking.

### 11.2 The world

- [ ] Zone transitions at map edges and doors, with a short loading screen.
- [ ] Interiors (an inn, a house) as small zones.
- [ ] World map window showing the zones and where you are.
- [ ] The three v1 zones blocked out (starting town and fields, Mosswood, the Old Ruins; working names), to be filled with content in Phase 13.

### 11.3 Resilience and scale

- [ ] Bots cross zones continuously for an hour with no lost characters or items.
- [ ] Crashing a zone mid-handoff leaves every player in a safe, recoverable state.
- [ ] `make loadtest SCENARIO=spread` at Case 2 Tier 2 (1,000 active players across 10 or more zones) on `dev-m5`, with CPU per core and zone tick p99 for every zone, in `docs/perf/phase-11.md`.

### 11.4 Walkthrough

- [ ] `docs/walkthroughs/phase-11-zones.md`: ownership boundaries, handoff and failure handling, and why the boundary is a message protocol.
- [ ] Devlog draft, release notes, tag `v0.11.0` and deploy the browser build.

### Acceptance criteria

- [ ] Players walk between at least three zones with transitions under two seconds.
- [ ] Parties, whispers and quests work across zones.
- [ ] A zone crash affects only that zone's players, who can reconnect.
- [ ] Case 2 Tier 2 in scaling.md (1,000 active players across zones) is met on `dev-m5`.

---

## Phase 12: Operations

> **Goal:** Lorehallow runs like a real live service, with metrics, logs, alerts, tested backups and deploys that don't throw everyone out.
> **Ships:** v0.12.0
> **Lesson:** Observability, alerting and release engineering for a live service
> **Depends on:** Phase 11

### 12.1 Metrics and dashboards

- [ ] Metrics completed for every service (the Phase 3 set plus database latency and errors, persistence queue depth and handoffs), each documented in `docs/ops/metrics.md`.
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
- [ ] Minimum client version announced by the server; an outdated browser tab shows "Update available" and reloads into the current build.

### 12.5 Runbooks

- [ ] Runbooks in `docs/ops/` for: server down, database full, bad deploy, compromised account, restoring from backup and the server at capacity.
- [ ] Game-day exercise: simulate two incidents and follow the runbooks, fixing whatever was wrong in them.

### 12.6 Status page

- [ ] /status shows 30-day uptime and recent incidents.

### 12.7 Soak and capacity

- [ ] 24-hour soak (scaling.md day soak) on `prod`.
- [ ] Case 1 Tier 3 (5,000 concurrent connections) measured on `dev-m5` and `prod`, including Caddy's memory per connection.
- [ ] A nightly scheduled job runs the fuzz targets for an hour and a short `spread` load test, and alerts on failure.

### 12.8 Walkthrough

- [ ] `docs/walkthroughs/phase-12-operations.md`: metrics, alerts, deploy strategy and incident response.
- [ ] Devlog draft, release notes, tag `v0.12.0` and deploy the browser build.

### Acceptance criteria

- [ ] Deploying a server update during a 20-bot session loses no progress.
- [ ] Each alert type was fired in a test and reached the owner.
- [ ] Dashboards show live player counts and tick times.
- [ ] The 24-hour soak passed on `prod`.

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
- [ ] Devlog draft, release notes, tag `v0.13.0` and deploy the browser build.

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

- [ ] Settings menu: fullscreen, integer scale, FPS cap, audio, gameplay, chat filters and UI scale.
- [ ] Key rebinding with conflict detection, saved in the browser's local storage with the other settings.
- [ ] `ADR` Controller support for v1, or deferred.

### 14.3 Onboarding

- [ ] First-time tips (movement, targeting, chat, quests) that can be turned off.
- [ ] Clear messages for every connection and login failure.

### 14.4 Accessibility

- [ ] Shapes as well as colors for hostility and item quality, text size options, a screen shake toggle and reduced flashing.

### 14.5 Polish

- [ ] Game feel pass: hit feedback, footstep dust and smooth UI transitions.
- [ ] Performance: 60 FPS at 1080p in current Chrome, Firefox and Safari on a five-year-old integrated-graphics laptop with 30 players on screen; profile and fix.
- [ ] Every placeholder asset and placeholder string is gone.
- [ ] All player-facing strings go through one string table, with English as the only shipped language.

### 14.6 Walkthrough

- [ ] `docs/walkthroughs/phase-14-polish.md`: audio architecture, settings persistence and performance profiling.
- [ ] Devlog draft, release notes, tag `v0.14.0` and deploy the browser build.

### Acceptance criteria

- [ ] `HUMAN` Play for 30 minutes and sign off on sound and feel.
- [ ] Every setting persists and applies without a restart where possible.
- [ ] The performance target is met and recorded in `docs/perf/phase-14.md`.

---

## Phase 15: Browser Releases

> **Goal:** Every release is one versioned browser build at play.lorehallow.com that updates cleanly in open tabs, stays compatible with the server it talks to, is checked live after every deploy and rolls back with one command.
> **Ships:** v0.15.0
> **Lesson:** Release engineering for a web client: versioned immutable builds, cache-safe updates, protocol compatibility, smoke checks and rollback
> **Depends on:** Phase 14

There are no native builds, installers, signing, storefronts, launchers, download manifests or auto-updaters ([ADR 0014](docs/decisions/0014-astro-website-and-phaser-browser-client.md)). Browser play has existed since Phase 2; this phase makes releasing it routine and safe.

### 15.1 Versioned builds

- [ ] `make release-client VERSION=vX.Y.Z` builds one browser build from a clean tag: version, commit and protocol version embedded, every asset fingerprinted, and build metadata (toolchain versions, lockfile hash, commit) recorded in the release.
- [ ] The same tag rebuilds to the same files, checked by building twice.

### 15.2 Protocol compatibility

- [ ] The server announces its protocol version and the minimum client version; deploy order (server first, then client) and the compatibility window are written down in `docs/ops/releases.md`.
- [ ] A test runs the previous release's client build against the new server: compatible changes keep it playing, and an incompatible change gets the "Update available" screen, never a broken session.

### 15.3 Cache-safe updates

- [ ] `index.html` always revalidates and hashed assets are immutable; the previous release's assets stay served for a while after a deploy, so a tab opened before the deploy never loads a missing file.
- [ ] An open tab learns of a new build (from the server's announcement or a reconnect) and offers to reload at a safe moment, forced only when the protocol is incompatible.

### 15.4 Smoke checks and rollback

- [ ] `make deploy-client` runs a browser smoke check against the live play.lorehallow.com after every switch (the page loads, assets load, the handshake completes) and rolls back automatically on failure. It runs from the deploying machine; CI never touches the host.
- [ ] `make rehearse-rollback-client` deploys a deliberately broken build and checks that it rolls back on its own; the last rehearsal is recorded in `deploy/HOST.md`.

### 15.5 Release notes and tags

- [ ] Release notes in `docs/releases/` for every version, rendered on lorehallow.com at /patch-notes, and the /play page shows the live version from the release data.
- [ ] Annotated `vX.Y.Z` tags with their release notes as GitHub and Codeberg releases (notes only, no binaries).

### 15.6 Browser support

- [ ] Supported browsers decided (current Chrome, Firefox and Safari on desktop) and stated on /play and in the FAQ, with a friendly message for unsupported ones.
- [ ] The browser smoke test runs in CI against Chromium and Firefox (and WebKit where the CI runner allows).

### 15.7 Walkthrough

- [ ] `docs/walkthroughs/phase-15-browser-releases.md`: versioned builds, cache-safe updates, protocol compatibility windows, smoke checks and rollback.
- [ ] Devlog draft, release notes and tag `v0.15.0` through the new release flow.

### Acceptance criteria

- [ ] A release goes from tag to live with `make release-client` and `make deploy-client`, run by hand, and is rolled back with one command.
- [ ] A tab left open across a deploy keeps working or is told to reload; it never shows a broken page.
- [ ] An outdated client is told to update.
- [ ] `HUMAN` The game loads and plays in current Chrome, Firefox and Safari (owner or friends verify).

---

## Phase 16: Security and Hardening

> **Goal:** Lorehallow is safe to open to strangers.
> **Ships:** v0.16.0
> **Lesson:** Threat modeling, fuzzing and abuse prevention
> **Depends on:** Phase 15

### 16.1 Threat model

- [ ] `docs/design/threat-model.md`: assets, attackers, entry points (WebSocket, HTTP APIs, admin tools, the host) and mitigations.

### 16.2 Protocol and server

- [ ] Fuzz targets for every message handler (envelope decoding is fuzzed from Phase 2), run briefly in CI and for an hour nightly.
- [ ] Limits everywhere (connections per IP, messages per second, message size, pending inputs, names, chat length), each with a test.
- [ ] Authority audit: every client message type reviewed for trust assumptions, with findings fixed.
- [ ] Review the Phase 2 slow-client, idle and handshake timeouts against real traffic; origin checks and TLS-only connections in production; both CSPs (website and game) audited, with no `unsafe-eval` or inline scripts in the game.
- [ ] Server-side anomaly logging (speed, impossible actions, abnormal rates) with an admin report.

### 16.3 Accounts and data

- [ ] Session expiry and revocation, "log out everywhere" and password change.
- [ ] Self-service account deletion that honors the privacy policy.
- [ ] Least-privilege database roles, verified encrypted backups, and secrets rotated with the process documented.

### 16.4 Host and supply chain

- [ ] Host audit (open ports, services, users, SSH, updates, file permissions) with findings fixed and documented.
- [ ] Automated dependency updates (Renovate or Dependabot) for the server's Go module, the Node lockfile, the art tools' Python lockfile and GitHub Actions, plus `govulncheck` and the package manager's audit in CI.

### 16.5 Walkthrough

- [ ] `docs/walkthroughs/phase-16-security.md`: the threat model, fuzzing and the authority audit.
- [ ] Devlog draft, release notes, tag `v0.16.0` and deploy the browser build.

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
- [ ] Release notes, tag `v0.17.0` and deploy the browser build.

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

- [ ] 72-hour soak test with 1,000 bots plus real players: no memory growth and no tick degradation.
- [ ] Capacity test for each scaling.md case on `prod`, finding where each target fails, documented in `docs/perf/phase-18.md`, with the player cap set safely below it.
- [ ] Case 2 Tier 3 (3,000 to 5,000 active players) and Case 3 Tier 3 (1,000 in one area) measured on `prod`; if one process can't meet them after tuning, write the ADR that starts Phase 20.
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
- [ ] Release notes, tag `v0.18.0` and deploy the browser build.

### Acceptance criteria

- [ ] The 72-hour soak test passed.
- [ ] Every Tier 2 target in scaling.md is met on `prod`.
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
- [ ] Full regression checklist in `docs/releases/regression.md` (every system, every class from 1 to 10, every quest, every supported browser), run and passed.
- [ ] Final backup and restore drill, and a written rollback plan for the launch build.
- [ ] Performance and security checks re-run against the release candidate, with every scaling.md tier met so far re-measured on `prod`.

### 19.2 Launch preparation

- [ ] Launch-day runbook: what to watch, rollback steps and how to communicate problems.
- [ ] Website launch update: "Play now" hero, final screenshots and a gameplay trailer, updated FAQ, and a /press page (description, fact sheet, logos, screenshots, contact).
- [ ] `HUMAN` Choose the launch date.
- [ ] `HUMAN` Approve the launch devlog post and announcements.
- [ ] Character wipe carried out if decided in Phase 18, with a backup taken first.

### 19.3 Launch

- [ ] Tag `v1.0.0`, deploy the 1.0 browser build with `make deploy-client`, and /play and /progress show 1.0.
- [ ] `HUMAN` Publish the launch announcements.
- [ ] Monitor closely for the first week and ship `v1.0.x` hotfixes as needed.

### 19.4 After launch

- [ ] Post-mortem in `docs/releases/v1.0-postmortem.md`: what went well, what didn't and what was learned.
- [ ] `docs/walkthroughs/phase-19-launch.md`: the whole architecture end to end as it stands at 1.0.
- [ ] Post-1.0 roadmap draft in `docs/design/roadmap-post-1.0.md`, built from Deferred and player feedback.

### Acceptance criteria

- [ ] v1.0.0 is playable at play.lorehallow.com in current Chrome, Firefox and Safari, with nothing to install.
- [ ] The first launch week had no data loss and 99% or better uptime.
- [ ] Every earlier phase in this file is complete.
- [ ] `HUMAN` Declare Lorehallow 1.0 shipped.

---

## Phase 20: Many Machines

> **Goal:** When measurements show one process is the limit, zones run in separate processes and then on separate machines, behind gateways that hold the client connections.
> **Ships:** v1.1.0
> **Lesson:** Distributed systems for games: gateways, service registries, inter-process handoff, partial failure and rolling deploys
> **Depends on:** Phase 19, and an ADR linking a measured single-process limit (see [scaling.md](docs/design/scaling.md#when-to-split-across-processes))

This is the post-1.0 track. It starts only when a result in `docs/perf/` shows a limit that tuning one process can't fix. If Phase 18 finds that limit before 1.0, this phase moves ahead of the launch with the owner's approval.

### 20.1 The case for splitting

- [ ] `ADR` Record the measured single-process limit that justifies splitting, the options (more tuning, a bigger host, a gateway plus zone processes) and the choice.

### 20.2 Gateway and zone processes

- [ ] `lorehallow-gateway`: terminates client WebSockets, authenticates, and routes each player to a zone process; clients don't reconnect on handoff.
- [ ] Internal protocol between gateways and zone processes in `proto/` (package `lorehallow.internal.v1`), with golden vectors, over one persistent connection per process pair.
- [ ] Service registry: which zone runs where, health checks, and draining a zone process before a deploy.
- [ ] Cross-process handoff using the Phase 11 protocol, rolling back when either side fails.
- [ ] Chat, whispers and parties across processes.
- [ ] `make server` runs the whole set locally; production gets a systemd unit per service.

### 20.3 Multiple machines

- [ ] Zone processes on a second machine over a private network, with mutual TLS between processes.
- [ ] Partial-failure tests: kill a zone process, a gateway, or the network between them; players end up safe and can reconnect.
- [ ] Rolling deploys: drain and restart one zone process at a time with no lost progress.

### 20.4 Measurement

- [ ] Re-run every scaling.md case against the split deployment, measure the cost of the extra hop, and record before and after in `docs/perf/phase-20.md`.

### 20.5 Walkthrough

- [ ] `docs/walkthroughs/phase-20-many-machines.md`: gateways, inter-process handoff, partial failure, and what splitting did and didn't fix.
- [ ] Devlog draft, release notes and tag `v1.1.0`.

### Acceptance criteria

- [ ] The limit recorded in 20.1 is gone, with before and after results on named hardware.
- [ ] A zone process crash affects only its zones' players, who are back in the world within 30 seconds.
- [ ] A rolling deploy across every zone process loses no progress.

---

## Deferred

Ideas and cut scope parked until after 1.0. Nothing here is counted or scheduled. Native desktop and mobile clients are not deferred but retired: Lorehallow is played in the browser ([ADR 0014](docs/decisions/0014-astro-website-and-phaser-browser-client.md)).

- Visible equipment on character sprites (layered paper-doll art)
- Guilds
- Player trading and an auction house
- Gathering and crafting professions
- Dungeons and instancing
- PvP: duels and arenas
- Mounts
- Player housing
- Controller support, if deferred in Phase 14
- Languages beyond English
- More zones and a higher level cap

## Superseded work

Checked work replaced by the move to an Astro website and a Phaser browser game ([ADR 0014](docs/decisions/0014-astro-website-and-phaser-browser-client.md), 2026-11-07). Each item was done and verified when it was checked, and is kept here as history with its original section. The replacements are in the phases above, unchecked until they are done. Nothing here is counted, and these are plain bullets because the format contract allows no checkboxes outside a phase.

The rule used: an item moved here when what it built is the Go site generator, the Godot client or native distribution itself. Checked items that only mention the old tools in passing, or that describe features the live site keeps through the parity checks, stayed in place.

### Phase 0: the Godot client

- 0.5 `ADR` GDScript code generation: try godobuf on the pinned Godot 4 version first; if it can't pass the round-trip tests below, write `tools/protoc-gen-gdscript` (a Go protoc plugin) for the proto3 subset Lorehallow uses. ([ADR 0003](docs/decisions/0003-gdscript-protobuf.md); replaced by the TypeScript bindings in 2.6.)
- 0.5 GDScript code generation into `client/net/pb/` (committed, marked generated).
- 0.6 Create the Godot 4 project in `client/` with the project name, a placeholder icon and the folder layout from AGENTS.md. (Replaced by the Phaser scaffold in 2.6.)
- 0.6 Rendering settings: base viewport 640×360, stretch mode `viewport`, integer scaling, default texture filter Nearest, no mipmaps, 2D pixel snapping. (The engine-neutral pixel rules stay checked in 0.6.)
- 0.6 Pixel-art import preset (filter off, mipmaps off, lossless) as the project default.
- 0.6 Main scene with a placeholder title screen showing the build version.
- 0.6 Autoload skeletons: `Config`, `Log` and `Net` (empty for now).
- 0.6 GUT installed in `addons/` with one passing test; `make test` runs GUT headless.
- 0.6 gdtoolkit through uv; `make fmt` and `make lint` cover `client/`, excluding generated code and addons.
- 0.6 Static typing enforced: the project warns on untyped declarations.
- 0.6 `make client` launches the client (Godot found through a `GODOT` variable with sensible macOS and Linux defaults).
- 0.8 Dev scene `client/scenes/dev/art_test.tscn` renders the processed tiles and character at 1x to 4x with no blur, seams or off-palette pixels. (Replaced by the Phaser dev art scene and `make client-art-check` in 2.6.)
- 0.10 Godot headless tests run in CI with the pinned Godot version.
- Phase 0 acceptance: `make server` runs the server and `make client` launches the client, which shows the title screen and build version.
- Phase 0 acceptance: Changing a `.proto` file and running `make proto` updates Go and GDScript code, and golden round-trip tests pass on both sides.
- Phase 0 acceptance: Batch 01 tiles and the character render pixel-perfect in Godot at every integer scale.

### Phase 1: the Go site generator and Downloads

- 1.2 `ADR` Website stack: a Go static site generator in `web/` (its own module) with no JavaScript on any page ([ADR 0011](docs/decisions/0011-go-website.md)). (Replaced by the Astro project in 1.2.)
- 1.2 `cmd/lorehallow-web` with `build` (writes the site to `web/dist/`) and `serve` (local preview that rebuilds on change, drafts included); templates (`html/template`), CSS and fonts embedded with `go:embed`.
- 1.2 Make targets `web-dev`, `web-build`, `web-test` and `web-lint` (gofmt, go vet, golangci-lint, govulncheck), with the tests and linters included in `make check`.
- 1.4 BUILD.md parser in `web/internal/buildmd/` implementing the format contract: phases, metadata, sections, items, tags and checked state. (Replaced by the TypeScript parser in 1.4, checked against this one.)
- 1.5 Devlog posts as Markdown files with front matter in `web/content/devlog/`, rendered with goldmark and validated at build time (title, date, summary, tags, cover image, draft). (Replaced by the Astro content collection in 1.5.)
- 1.6 Platform cards for Windows, macOS and Linux with an honest "Not available yet. Follow the progress." state. (Replaced by /play in 1.6.)
- 1.6 Page driven by `web/data/releases.json` (version, date, channel, platform, URL, size, SHA-256, notes link), with the schema documented for the release automation that writes it later.
- 1.6 System requirements and "verify your download" sections that appear once releases exist.
- Phase 1 acceptance: /downloads exists and accurately says no builds are available yet.

### Unchecked plans dropped

Native distribution was planned but never started, so it was replaced in place rather than recorded here: the Phase 2 desktop exports and /downloads listing (2.10), and Phase 15's desktop packaging, signing, download manifests, update checks, itch.io listing, install guides and Godot web export. The launcher left Deferred.
