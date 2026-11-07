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

It is a hobby project with two goals of equal weight:

1. **Make a small online world that's fun to be in**, even with only a few players.
2. **Learn real multiplayer and backend engineering by building it.** Every system should teach something. Prefer the approach that teaches the concept clearly over a black box that hides it, and explain it in the phase walkthrough.

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
- Mobile clients
- Supporting thousands of concurrent players. Dozens is the real target; hundreds is a stretch goal.

---

## 2. Roles and autonomy

**The owner** (dunamismax) sets direction, generates art in ChatGPT, owns everything involving money, accounts, credentials, DNS and host access, playtests builds, and approves public announcements. Items the owner must do are tagged `HUMAN` in BUILD.md.

**Agents** do everything else: design, code, tests, infrastructure, deploys, art prompts, art intake and integration, and documentation. **Work autonomously.** The owner wants agents to make decisions and keep moving, not to ask permission for engineering choices.

### Decide it yourself

Make the call, record it, and continue when the question is about:

- Libraries, code structure, algorithms, data formats, file layout and naming
- Protocol message design, database schema and migrations
- Game tuning numbers (speeds, cooldowns, XP curves), which playtests will adjust anyway
- Tooling, CI, scripts and test strategy
- UI layout and website design within the art direction
- Adding, splitting or rewording unchecked BUILD.md items to reflect real work

Record any decision that a future session would wonder about: an ADR in `docs/decisions/` for anything significant, plus a row in the [decision log](#14-decision-log). Items tagged `ADR` in BUILD.md always need one.

### Stop and ask the owner

- The next step is a `HUMAN` item, or you discover a new owner dependency. Add it to BUILD.md tagged `HUMAN`, then keep working on anything that isn't blocked.
- Anything that spends money, creates an account, accepts terms of service or needs a credential you don't have.
- Destructive or irreversible production actions: deleting or rewriting production data, wiping characters, deleting backups, or changing SSH, firewall or DNS settings in a way that could lock the owner out.
- Changing the design pillars, the non-goals, the v1.0 scope or the phase order.
- Publishing anything in the owner's voice: devlog posts and announcements are written as drafts and the owner publishes them. (Routine site updates such as the progress tracker, release listings and patch notes generated by the pipeline are fine.)
- Legal text: licenses, privacy policy and terms are drafted by agents and approved by the owner.

### When you are blocked

Never sit idle. Prepare the blocked item so the owner's part takes minutes (prompts written, scripts ready, exact instructions), say clearly what you need, then move to the next unblocked item in the current phase. If the whole current phase is blocked on `HUMAN` items, you may start independent work in the next phase.

---

## 3. Session workflow

### Start of session

1. `git pull --rebase origin main`
2. Read this file, then BUILD.md. The **current phase** is the lowest-numbered phase with an unchecked box.
3. Read the most recent walkthrough in `docs/walkthroughs/` and any ADRs for the area you'll touch.
4. Run `make doctor` and `make check` (once they exist). If the baseline is red, fixing it is your first task.
5. Pick work: what the owner assigned, otherwise the first unchecked, unblocked item in the current phase. Items within a phase may be done out of order when their dependencies are met.

### While working

- Work in small, verified steps. Every commit leaves the repo working.
- Write tests with the code, not after it.
- Never block on art. Use placeholder art (see [section 8](#8-art-pipeline)) and keep going.
- Use any tool you need. Installing tools (Homebrew, apt, Go and npm packages, uv tools, Godot addons) is fine; pin versions and add them to `make doctor` and the [toolchain table](#13-toolchain-versions).
- Look up current documentation for libraries and tools (Context7 or official docs) instead of relying on memory, especially for Godot 4, Astro and buf, which change often.

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
  | client | `client/` (except generated code) |
  | web | `web/`, `deploy/web/` |
  | art | `art/`, `tools/art/` |
  | ops | `deploy/` (except `deploy/web/`), `docs/ops/`, `.github/` |

- **Shared files** (`BUILD.md`, `AGENTS.md`, `Makefile`, `proto/`): keep edits small, pull right before editing, and push right after.
- **Protocol changes are serialized.** One agent changes `proto/` at a time, and the schema change, both sides' generated code, and the code that uses it land together so `main` always builds.
- **BUILD.md conflicts:** when rebasing, keep every check from both sides.
- When the owner runs agents in parallel, they assign lanes. An agent without an assignment takes the first unblocked item in its lane.

---

## 7. Architecture and conventions

### Repository layout

```
lorehallow/
├── AGENTS.md, BUILD.md, README.md, CLAUDE.md, Makefile
├── client/                # Godot 4 project
│   ├── addons/            # Third-party addons (GUT, ...), not linted
│   ├── assets/            # art/ (synced from art/processed), audio/, fonts/, content/ (synced)
│   ├── autoload/          # Singletons: Config, Net, Log, ...
│   ├── net/               # Networking; net/pb/ is generated protobuf code
│   ├── world/             # Maps, camera, world rendering
│   ├── entities/          # Players, mobs, NPCs
│   ├── ui/                # Screens, HUD, windows, theme
│   ├── scenes/dev/        # Test and debug scenes
│   └── tests/             # GUT tests
├── server/                # Go module github.com/dunamismax/lorehallow/server
│   ├── cmd/               # lorehallow-server, lorehallow-bot, lorehallow-admin, ...
│   ├── internal/          # sim/, net/, world/, auth/, db/, pb/ (generated), ...
│   └── migrations/        # SQL migrations (Phase 4 on)
├── proto/                 # lorehallow/v1/*.proto, buf config, testdata/ golden vectors
├── content/               # Canonical game data: maps/, items/, mobs/, quests/, ...
├── web/                   # lorehallow.com (Astro)
├── tools/                 # art/ (Python, uv), other helpers
├── art/                   # style/, batches/, inbox/, processed/
├── deploy/                # HOST.md, bootstrap/, caddy/, server/, web/, ...
├── docs/                  # decisions/, walkthroughs/, design/, ops/, perf/, releases/
└── .github/workflows/     # CI, deploys, releases
```

Generated or synced copies (`client/assets/art`, `client/assets/content`, `web/src/assets/art`, `server/internal/pb`, `client/net/pb`) are committed but never edited by hand. Edit the source and run `make sync` or `make proto`.

### Networking model

- **Server-authoritative.** Clients send *inputs* ("moving northeast", "use ability 2 on target 41"), never positions or outcomes. The server simulates and validates everything.
- **Fixed tick rate:** 20 Hz. The server broadcasts state snapshots every tick.
- **Interpolation:** remote entities are drawn about 100 ms behind the latest snapshot, interpolating between buffered snapshots.
- **Prediction:** the local player's movement is predicted and reconciled against server corrections (Phase 3). Nothing else is predicted.
- **Interest management:** players only receive updates about entities near them (Phase 7).
- **Endpoints:** game traffic at `wss://play.lorehallow.com/ws`; HTTP APIs (status, auth) under `https://lorehallow.com/api/`; local dev at `ws://localhost:<port>/ws`.

### Protocol rules

- proto3, package `lorehallow.v1`, files in `proto/lorehallow/v1/`, managed with `buf` (lint and breaking-change checks).
- One `Envelope` message per binary WebSocket message, with a `oneof` payload.
- Never reuse or renumber field numbers. Mark removed fields `reserved`.
- `PROTOCOL_VERSION` is a single integer. Bump it on any incompatible change; the server rejects mismatched clients with a clear "update required" message.
- Every message type gets golden round-trip tests: Go writes vectors to `proto/testdata/`, and both Go and GDScript decode and re-encode them byte for byte.
- Generated code is committed and never hand-edited. `make check` fails when it is stale.

### Server conventions (Go)

- Standard layout: `cmd/` for binaries, `internal/` for packages. Prefer the standard library; any new dependency needs a reason in the commit body.
- **The simulation is pure.** `internal/sim` does no I/O, takes an injected clock and is deterministic, so it can be tested tick by tick.
- **One goroutine owns a zone's world state**: the tick loop. Connection goroutines talk to it through channels. Don't share mutable world state behind locks.
- `log/slog` for logging (text locally, JSON in production). Wrap errors with `%w`. Pass `context.Context` through I/O.
- Configuration from flags and `LOREHALLOW_*` environment variables, documented in `server/README.md` and `.env.example`.
- Tests are table-driven, run with `-race`, and never `sleep`; use the fake clock.
- `gofmt`/`goimports`, `go vet`, `golangci-lint` and `govulncheck` must be clean.

### Client conventions (Godot 4, GDScript)

- GDScript only, with **static typing everywhere** (typed variables, parameters and return values). The project warns on untyped code.
- Files and folders `snake_case`; `class_name` in `PascalCase`; signals in past tense (`player_joined`); private members prefixed `_`.
- Organize by feature (see layout above). Keep game logic out of UI scripts.
- The client never decides outcomes. It renders server state, and predicts only its own movement.
- **Pixel rules:** base viewport 640×360, integer scaling only, nearest-neighbor filtering, no mipmaps, positions snapped to whole pixels on screen, camera snapped.
- `gdformat` and `gdlint` must be clean (generated code and `addons/` excluded). Tests use GUT and run headless in `make check`.

### Web conventions (lorehallow.com)

- Astro static site in `web/`, TypeScript strict. Client-side JavaScript only as progressive enhancement; every page works without it.
- Colors come from tokens generated from `art/style/palette.hex`, so the site and the game share one palette.
- No third-party requests: no trackers, CDNs or remotely loaded fonts. Everything is self-hosted.
- Pixel art is shown at integer scales with `image-rendering: pixelated` and explicit dimensions.
- The BUILD.md parser in `web/src/lib/buildmd/` implements the format contract. Change both together.

### Content conventions

- Game data lives in `content/` as YAML, plus Tiled JSON (`.tmj`) for maps unless an ADR says otherwise. IDs are `snake_case`, stable and never reused.
- Every content type has a validator that runs in `make check`: references resolve, quests are completable, numbers are in range.

### Versions

- Game builds are versioned `v0.<phase>.<patch>` until 1.0 (Phase 2 ships v0.2.0, Phase 18 ships v0.18.0), and Phase 19 ships v1.0.0.
- The version and git commit are embedded in both the server and client builds and shown in the client title screen and `/version`.

---

## 8. Art pipeline

### Locked art direction

| Rule | Value |
|---|---|
| Tiles | 32×32 px |
| Characters | 32×48 px frames, 4 directions (down, up, left, right; right mirrored from left where possible) |
| Animations | idle 2 frames, walk 4 frames; attack, cast, hurt and death added later |
| Palette | one fixed palette of about 32 colors in `art/style/palette.hex`, used in every prompt |
| Perspective | top-down 3/4 view |
| Light | from the top-left, always |
| Rendering | integer scaling only, no anti-aliasing, no sub-pixel blur |
| Base viewport | 640×360 |
| Mood | warm storybook fantasy: mossy forests, lantern-lit towns, old stone ruins |
| Key color | `#FF00FF` flat background for keying (never used in the palette) |

`art/style/STYLE.md` (written in Phase 0) holds the full style guide, and `art/style/reference.png` is the reference sheet attached to every prompt.

### Batches

Art is produced in numbered batches, each tied to the feature that needs it. Batch numbers follow the order BUILD.md lists them (01 style foundation, 02 brand and website art, 03 player character, and so on). Each batch has a folder:

```
art/batches/batch-03-player/
├── prompts.template.md  # Source of the prompts; {{PALETTE}}, {{PALETTE_COUNT}} and {{KEY}} are filled in
├── prompts.md           # Generated by `make art-generate`: ready-to-paste prompts, one per image, in order
├── manifest.yaml        # Machine-readable spec for every expected asset
├── fixes.py             # Optional: scripted cleanups applied on every intake run
├── guides/              # Layout guides to attach to prompts (generated from the manifest)
├── report.md            # Intake results and corrected prompts for rejects (generated)
└── preview.png          # 4x contact sheet (generated)
```

Edit `prompts.template.md`, never `prompts.md`: generating the prompts from the template keeps the palette in every prompt in step with `art/style/palette.hex`, and `make check` fails if they drift.

### Writing prompts

ChatGPT image generation is good at pixel art and can produce a whole multi-frame sprite sheet in one image. Ask for complete sheets in strict grids rather than single frames. Every prompt states:

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

### The owner's part

Paste each prompt into ChatGPT in order (attaching the reference sheet when told to), download the results, and drop them **unchanged** into `art/inbox/batch-NN/`. No renaming or editing needed.

### Intake (agents)

1. Look at every image in the inbox folder and match it to its manifest entry. Write the mapping to `art/inbox/batch-NN/map.yaml`.
2. Run `make art-intake BATCH=NN`. It detects the real pixel grid, downsamples to the logical size, removes the key color, quantizes to the palette, slices sheets, checks frame counts and alignment, and writes `art/processed/` plus `report.md`.
3. Inspect the results at 4x. Small fixes (stray pixels, nudging a frame onto the baseline) go in a per-batch fix script in the batch folder so re-running intake reproduces them. Never edit processed files by hand without a script.
4. Anything that can't be fixed gets a corrected prompt in `report.md`. The owner regenerates it into the same inbox folder.
5. `make sync`, then test in-game in the dev art scene and in real use. Check for blur, seams, palette violations and alignment at 1x to 4x.

**`art/inbox/` is never edited.** It stays as generated so intake can always be re-run.

### Naming and import

- Processed paths: `art/processed/<category>/<name>.png`, categories `tiles`, `chars`, `mobs`, `npcs`, `items`, `ui`, `vfx`, `icons`, `web`. Names are `snake_case` (`tiles/terrain_grass.png`, `chars/player_base.png`). Sheets get a sidecar `.json` with frame size, frame order, pivot and source.
- Godot imports use the pixel-art preset: filter off, mipmaps off, lossless compression.

### Placeholders

Code never waits on art. Use placeholder sprites (flat-colored shapes or simple generated sprites at the correct size) named with a `_placeholder` suffix. `make art-placeholders` writes one for every manifest asset that has no processed output yet (with sidecars, so sheets keep their frame layout and baseline) and removes each one when the real asset lands. Real art replaces them within the same phase, and Phase 14 removes every remaining placeholder.

---

## 9. Quality gates

`make check` is the single gate, run locally before every commit and in CI on every push. CI runs on **GitHub Actions** (`.github/workflows/ci.yml`, Linux, same pinned versions from `toolchain.env`). **Codeberg is a mirror**: it receives every push but runs no CI. It covers:

- **Server:** `gofmt` check, `go vet`, `golangci-lint`, `go test -race ./...`, `govulncheck`
- **Protocol:** `buf lint`, `buf breaking`, generated code up to date, golden round-trip tests on both sides
- **Client:** `gdformat --check`, `gdlint`, GUT tests headless
- **Web:** `astro check`, lint, unit tests, build, internal link check
- **Tools:** `ruff`, `pytest`
- **Content and docs:** content validators, BUILD.md lint

Testing expectations:

- Server game logic gets real unit tests, driven tick by tick with the fake clock.
- Every protocol change gets round-trip tests.
- Anything security-related (auth, rate limits, validation, cheating) gets a test that proves the attack fails.
- Anything touching persistence gets tests against a real Postgres (CI service container), including failure in the middle of an operation.
- Bugs get a regression test before the fix.

**Definition of done** for any BUILD.md item: it works, it's tested, `make check` is green, docs are updated, and it's pushed.

---

## 10. Deployment and operations

### The host

The owner's own server sits behind lorehallow.com. Its details (OS, architecture, resources, network, what else runs there) live in `deploy/HOST.md`, which the owner fills in during Phase 1. Until it says otherwise, assume Linux with systemd. If something conflicts with this section (for example an existing reverse proxy), adapt and record an ADR.

- **Domains:** `lorehallow.com` (website and `/api/`), `www.lorehallow.com` (redirects to the apex), `play.lorehallow.com` (game WebSocket traffic).
- **Reverse proxy:** Caddy, with automatic HTTPS. Config lives in `deploy/caddy/`.
- **Layout on the host:** releases in `/srv/lorehallow/<service>/releases/<timestamp>/` with a `current` symlink switched atomically; secrets in `/etc/lorehallow/<service>.env` (mode 0640); services run as dedicated users under systemd units from `deploy/`.
- **Access:** agents connect as the `deploy` user over SSH with the key the owner provides. Privileged setup goes through idempotent scripts in `deploy/bootstrap/` that the owner reviews and runs (or temporarily lets an agent run).

### What agents may do in production

- Deploy with the `make deploy-*` targets after `make check` passes, and roll back with `make rollback-*`.
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
- Dependencies scanned in CI (`govulncheck`, `npm audit`).
- Automated, encrypted, off-site Postgres backups from Phase 4 on, with restores actually tested.

---

## 12. Documentation

- **Decisions:** `docs/decisions/NNNN-short-title.md` (four-digit numbers), with sections Context, Decision, Alternatives considered, Consequences. Add a row to the decision log below.
- **Walkthroughs:** every phase ends with `docs/walkthroughs/phase-NN-slug.md`, written to teach. Sections: What we built; How to try it; How it works; The key concept (the networking or backend lesson, with diagrams where they help); Read the code (specific files and functions, in reading order); What's next.
- **Design notes:** `docs/design/` for the protocol, movement, combat, classes, lore and the threat model.
- **Operations:** `docs/ops/` for runbooks, backups and incidents. `docs/perf/` for measurements.
- **Releases:** `docs/releases/vX.Y.Z.md`, which become patch notes on the website.
- **Devlog drafts:** before the website exists, `docs/devlog-drafts/`; afterwards, `web/src/content/devlog/` with `draft: true`. The owner edits and publishes.
- **Directory READMEs:** every top-level directory has a short README saying what belongs there and how to work in it.
- **Style:** plain language and short sentences. The landing page can sell; docs explain.

---

## 13. Toolchain versions

Pinned in Phase 0 (BUILD.md 0.3) to the latest stable releases at that time ([ADR 0002](docs/decisions/0002-toolchain-versions.md)). The machine-readable source is [`toolchain.env`](toolchain.env); `make doctor` checks against it. Update `toolchain.env` and this table in the same commit as any version bump.

| Tool | Version | Notes |
|---|---|---|
| Go | 1.27.3 | |
| Godot | 4.7.2-stable | Standard build, plus matching export templates |
| Node.js | 26.13.0 | Current LTS ("Lithium"); used from Phase 1 |
| Python | 3.14.9 | Managed by uv |
| uv | 0.13.7 | |
| buf | 1.75.0 | |
| golangci-lint | 2.15.0 | |
| gdtoolkit | 4.5.0 | gdformat and gdlint, via uv |
| GUT | 9.7.1 | Godot addon, installed by `make setup` |
| godobuf | 0.7.0 | Godot addon for GDScript protobuf codegen, installed by `make setup` |
| protoc-gen-go | 1.36.12 | `tool` directive in `server/go.mod` |
| govulncheck | 1.9.0 | `tool` directive in `server/go.mod` |
| Caddy | _pin in 1.1_ | On the host |
| PostgreSQL | _pin in 4.1_ | On the host and in CI |

---

## 14. Decision log

One row per decision. "Default" means the choice stands unless the ADR written for that BUILD.md item finds a good reason to change it.

| # | Decision | Status | Why | Record |
|---|---|---|---|---|
| 1 | Client in Godot 4 with GDScript | Decided | Fast iteration, excellent 2D, simple export to Mac, Windows and Linux | Project brief |
| 2 | Game server in Go | Decided | Built for network servers, safe concurrency, fast builds, single-binary deploys | Project brief |
| 3 | Transport is WebSocket over TLS | Decided | Reliable and ordered like WoW's TCP model; leaves the door open for a web build | Project brief |
| 4 | Protocol is Protobuf with schemas in `proto/`, managed by buf | Decided | One versioned source of truth for every message | Project brief |
| 5 | PostgreSQL for persistence, from Phase 4 | Decided | Characters, inventory and world state need real transactions | Project brief |
| 6 | Self-hosted on the owner's server behind lorehallow.com | Decided | Learn deployment, TLS, supervision, logs and backups for real | Project brief |
| 7 | Server-authoritative, 20 Hz tick, ~100 ms interpolation, prediction for own movement only | Decided | Cheat resistance and a clear model to learn from | Project brief |
| 8 | 32×32 tiles, 32×48 characters, ~32-color palette, 640×360 base viewport with integer scaling | Decided | Readable pixel art at 2x, 3x and 4x on common displays | Project brief, this file |
| 9 | Website before gameplay: phases are 0 scaffold, 1 website, 2+ game | Decided | Public, followable progress from day one | Owner |
| 10 | Website is an Astro static site; BUILD.md parsed at build time | Default | Static output is fast and simple to self-host; Markdown-native | BUILD.md 1.2 |
| 11 | Website deploys from GitHub Actions to the host over SSH on every relevant push | Default | Makes the progress tracker update automatically | BUILD.md 1.9 |
| 12 | Caddy as reverse proxy with automatic HTTPS | Default | Automatic certificates and simple WebSocket proxying | BUILD.md 1.1 |
| 13 | GNU Make as the single command runner | Default | Already installed on macOS and Linux; one entry point for humans, agents and CI | BUILD.md 0.3 |
| 14 | Art tooling in Python managed by uv (Pillow, numpy) | Default | Best image-processing ecosystem; uv keeps it reproducible | BUILD.md 0.7 |
| 15 | GUT for Godot tests; gdtoolkit for GDScript format and lint | Default | Mature, headless-friendly | BUILD.md 0.6 |
| 16 | GDScript protobuf via godobuf 0.7.0; custom protoc plugin only if a golden test fails | Decided | Passed every golden round-trip vector on Godot 4.7 | [ADR 0003](docs/decisions/0003-gdscript-protobuf.md) |
| 17 | Game traffic on `play.lorehallow.com`, HTTP APIs under `lorehallow.com/api/` | Default | Separates game traffic from the website; easier to move later | This file |
| 18 | Game versions `v0.<phase>.<patch>` until 1.0 | Decided | Version numbers map to BUILD.md phases | This file |
| 19 | `main`-only git, commit and push after every verified step, no AI attribution | Decided | Owner's rules | Owner |
| 20 | Stack decisions recorded in one ADR | Decided | One place for later ADRs to reference | [ADR 0001](docs/decisions/0001-stack.md) |
| 21 | Toolchain pinned in `toolchain.env`; doctor accepts newer patch releases | Decided | One source for Make, doctor and CI | [ADR 0002](docs/decisions/0002-toolchain-versions.md) |
| 22 | `PROTOCOL_VERSION` lives in `proto/PROTOCOL_VERSION`; `buf breaking` failures pass only with a bump | Decided | One integer, generated into both languages; "planned" breaks are explicit | [proto/README.md](proto/README.md) |
| 23 | Client uses the Compatibility renderer; untyped GDScript is an error, not a warning | Decided | Widest hardware support for 2D; headless runs don't print warnings, so only an error is enforceable in CI | [client/README.md](client/README.md) |
| 24 | Local dev server listens on 127.0.0.1:7420 | Default | Unprivileged, unlikely to clash | [server/README.md](server/README.md) |
| 25 | Code under AGPL-3.0-only; art and writing under CC BY-NC 4.0 | Decided | Forks must share server changes; art and lore can be shared and remixed but not sold | Owner, [README.md](README.md#license) |
| 26 | Palette v2: earth, stone and leaf ramps fitted to what the image generator draws, plus sand (33 colors) | Decided | v1 had gaps that sent browns to red and skin and grey stone to pink during quantization | [STYLE.md](art/style/STYLE.md#why-v2) |
