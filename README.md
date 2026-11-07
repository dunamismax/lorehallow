# Lorehallow

**A small pixel-art fantasy MMO built around a shared world.** You log in and other people are already there. Someone runs past on the road, a group is fighting at the edge of the woods, and the town square is busy.

Lorehallow draws on the early days of World of Warcraft: a persistent world, zones, quests, grouping, and the sense that the world goes on without you. It is drawn in a warm, readable, top-down pixel-art style: mossy forests, lantern-lit towns and old stone ruins.

[![CI](https://github.com/dunamismax/lorehallow/actions/workflows/ci.yml/badge.svg)](https://github.com/dunamismax/lorehallow/actions/workflows/ci.yml)
[![Lorehallow progress](https://lorehallow.com/progress.svg)](https://lorehallow.com/progress/)

| | |
|---|---|
| **Website** | [lorehallow.com](https://lorehallow.com) |
| **Progress** | [BUILD.md](BUILD.md), tracked live at [lorehallow.com/progress](https://lorehallow.com/progress) |
| **Devlog** | [lorehallow.com/devlog](https://lorehallow.com/devlog) |
| **Source** | [GitHub](https://github.com/dunamismax/lorehallow) · [Codeberg](https://codeberg.org/dunamismax/lorehallow) |

## Status

**Phase 1: lorehallow.com. Pre-alpha, nothing to play yet.**

The tooling is in place: the server and client skeletons, the shared protocol pipeline, the art pipeline and the BUILD.md linter, all behind one `make check`. [lorehallow.com](https://lorehallow.com) is live: a static site built by a Go program in `web/`, with no JavaScript, a progress tracker drawn from BUILD.md and a devlog. What's left of Phase 1 is the first published devlog post and the owner's review. Then work returns to the game server's networking core (Phase 2), which is already partly built. [BUILD.md](BUILD.md) is the full plan from an empty repo to the v1.0 release, with every step as a checkbox.

## The project

Lorehallow is a hobby project with three goals:

1. **Make a small online world that's fun to be in**, even with only a few players.
2. **Build an excellent MMO server.** The Go backend and its networking stack are the project's main achievement: fast, scalable, reliable, secure and maintainable. Players see a small, beautiful world; the infrastructure behind it is the showcase.
3. **Learn real multiplayer and backend engineering by building it**: networking, authoritative servers, persistence, deployment and operations. Every system should teach something, and every phase ends with a written walkthrough in [`docs/walkthroughs/`](docs/walkthroughs/).

### Scale

The goal is **hundreds to thousands of simultaneous, actively connected players.** The server starts as one process with many zones, each zone owned by a single goroutine, and grows to several processes or machines when measurements call for it ([ADR 0005](docs/decisions/0005-scale-and-backend-architecture.md)).

[`docs/design/scaling.md`](docs/design/scaling.md) is the performance contract. It separates three very different problems (many quiet connections, many active players spread across zones, and hundreds of players crowded into one place who can all see each other) and sets measurable targets for each, from about 100 to 1,000 to several thousand. Thousands is a target, not a claim: no player count is stated as fact until it has been measured, and every measurement lives in [`docs/perf/`](docs/perf/) with the hardware it ran on.

### Design pillars

1. **The world is shared.** Every feature should make other players more visible, more useful or more present.
2. **Small but alive.** One handcrafted zone that feels alive beats ten empty ones.
3. **Readable pixel art.** Clear silhouettes, a locked palette and consistent scale.
4. **The server decides.** The server owns the truth. Clients send what the player is trying to do and draw what the server reports back.
5. **Playable every milestone.** Every phase ends with something you can launch, play and show someone.

### What it is not

- No monetization, cash shop or pay-to-win. Ever.
- No 3D, isometric or mixed art styles.
- No mobile clients.
- No game-server framework doing the hard parts. The server is built from the ground up in Go, because that's the point.
- No microservices, Kubernetes or distributed databases unless a measurement proves they're needed.

## Roadmap

The full plan, with acceptance criteria for every phase, is in [BUILD.md](BUILD.md).

| Phase | What | Version |
|---|---|---|
| 0 | Foundations: repo, toolchain, protobuf pipeline, art pipeline, first art | |
| 1 | lorehallow.com: landing page, live progress tracker, devlog, downloads | |
| 2 | Two players, one world: connection lifecycle, backpressure, bot swarm | v0.2.0 |
| 3 | Many players: prediction, interest management, bandwidth, load testing | v0.3.0 |
| 4 | Accounts and characters | v0.4.0 |
| 5 | Chat and presence | v0.5.0 |
| 6 | Combat | v0.6.0 |
| 7 | Crowds and a larger world: bandwidth budgets, graceful degradation | v0.7.0 |
| 8 | Items and loot | v0.8.0 |
| 9 | Quests, NPCs and progression | v0.9.0 |
| 10 | Grouping | v0.10.0 |
| 11 | Zones: isolated zone simulations and handoff | v0.11.0 |
| 12 | Operations | v0.12.0 |
| 13 | Classes and the v1 world | v0.13.0 |
| 14 | Audio, UI and polish | v0.14.0 |
| 15 | Distribution and releases, plus a browser build | v0.15.0 |
| 16 | Security and hardening | v0.16.0 |
| 17 | Closed alpha | v0.17.0 |
| 18 | Open beta | v0.18.0 |
| 19 | Launch | **v1.0.0** |
| 20 | Many machines: gateways and zone processes, when measurements call for it | v1.1.0 |

## Technology

| Layer | Choice |
|---|---|
| Client | Godot 4 + GDScript (Windows, macOS, Linux, and later the browser through Godot's web export) |
| Game server | Go |
| Transport | WebSocket over TLS (`wss://`) |
| Protocol | Protobuf, schemas in [`proto/`](proto/) shared by client and server |
| Persistence | PostgreSQL |
| Website | Go static site generator, no JavaScript (ADR 0011) |
| Hosting | Self-hosted behind lorehallow.com (Caddy, systemd) |

**Networking model:** the server is authoritative and simulates at 20 Hz. Clients send inputs, never positions. Remote players are interpolated about 100 ms behind the latest snapshot, your own character is predicted and reconciled, and each player only receives updates about what is near them, found through a spatial grid.

**Server model:** one goroutine owns each zone and nothing else touches its state. Connections talk to zones through bounded channels, every connection has a bounded send queue, and a client too slow to keep up is disconnected instead of slowing anyone else down. Each entity's state is encoded once per tick and shared by everyone who can see it. A headless bot swarm (`lorehallow-bot`) drives load tests for every case in the scaling model.

## Repository layout

```
lorehallow/
├── AGENTS.md      # How work gets done here: workflow, conventions, decisions
├── BUILD.md       # The build plan, phase 0 to v1.0, tracked by the website
├── client/        # Godot 4 project (GDScript)
├── server/        # Go module: game server, bots, admin tools
├── proto/         # Protobuf schemas and golden test vectors
├── content/       # Data-driven game content: maps, items, quests, mobs
├── web/           # lorehallow.com
├── tools/         # Art intake pipeline, codegen and dev helpers
├── art/
│   ├── style/     # Locked palette, style guide, reference sheet
│   ├── batches/   # Image prompts, manifests and intake reports per batch
│   ├── src/       # Hand-drawn art sources: pixel grids and draw scripts
│   ├── inbox/     # Raw generated images, never edited
│   └── processed/ # Cleaned, grid-snapped, palette-locked assets
├── deploy/        # Host setup, Caddy, systemd units, deploy scripts
└── docs/          # Walkthroughs, design notes, decisions, runbooks
```

Each directory has its own README.

## Development

Run `make help` for the full list. (`make web-dev` arrives with the website in Phase 1.)

```bash
make doctor   # check the toolchain is installed at the pinned versions
make setup    # install project dependencies
make check    # format checks, linters and every test suite (what CI runs)
make server   # run the game server locally
make client   # launch the Godot client against the local server
make web-dev  # run the website locally
```

Prerequisites: Go, Godot 4, [uv](https://docs.astral.sh/uv/) (which manages Python), [buf](https://buf.build), golangci-lint, gdtoolkit and GNU Make 4+. Exact versions are pinned in [`toolchain.env`](toolchain.env) and listed in [AGENTS.md](AGENTS.md#13-toolchain-versions); `make doctor` tells you what's missing and how to install it.

## How it's made

Lorehallow is built in the open, one phase at a time. The project owner sets the direction, chooses what comes next and playtests every build. AI coding agents do most of the design, programming, testing and infrastructure work, following [AGENTS.md](AGENTS.md) and checking off [BUILD.md](BUILD.md) as they go. The pixel art is drawn by the agents pixel by pixel, as text files in the repository: grids of named palette colors and small drawing scripts, rendered and checked by the build. A few large painted images for the website are AI-generated and run through an intake pipeline that snaps them to a true pixel grid and the locked palette.

## Contributing

This is a personal learning project, so outside code contributions are not being accepted for now. Bug reports, ideas and kind words are welcome as issues on [GitHub](https://github.com/dunamismax/lorehallow/issues) or [Codeberg](https://codeberg.org/dunamismax/lorehallow/issues).

## License

Lorehallow uses two licenses, one for code and one for art and writing.

**Code: [GNU Affero General Public License v3.0](LICENSE)** (AGPL-3.0-only). Anyone can study, change and fork it. If you run a modified version of the server for other people to play on, you must publish your changes under the same license. This covers the game client, the server, the protocol, tools, scripts, build and deploy files, and the website's code.

**Art and writing: [Creative Commons Attribution-NonCommercial 4.0](LICENSE-ART)** (CC BY-NC 4.0). You can share and remix it with credit, but not sell it. This covers:

- everything in `art/`, and the synced copies in `client/assets/art/` and `web/assets/art/`
- the game's written content in `content/`: lore, quest and dialogue text, item, creature and place names and descriptions
- the devlog (the website's posts in `web/content/devlog/`)

Credit it as "Lorehallow (lorehallow.com)" with a link to this repository. Game data in `content/` that isn't prose (map layouts, numbers, IDs) is part of the code and under the AGPL. Releases of the game bundle both, so a build is free to share but its art and writing can't be sold.
