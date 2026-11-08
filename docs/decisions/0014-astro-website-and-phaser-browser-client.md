# 0014: An Astro website and a Phaser browser game

- **Status:** Accepted. Supersedes [ADR 0011](0011-go-website.md) and [ADR 0003](0003-gdscript-protobuf.md), and the client row of [ADR 0001](0001-stack.md). The website half is done: lorehallow.com is the Astro build and the Go generator is gone (BUILD.md 1.9). The Godot client remains until the BUILD.md 2.10 cutover.
- **Date:** 2026-11-07
- **BUILD.md:** 1.2, 1.9, 2.6, 2.10, Phase 15

## Context

Lorehallow has two frontends, and both are changing at the owner's direction.

**The website.** lorehallow.com was first planned as an Astro site. [ADR 0011](0011-go-website.md) replaced that with a Go static site generator in `web/`, with no JavaScript on any page, because the owner was moving every project to Go. That site is live: Caddy serves the files it builds, deploys are atomic with automatic rollback, and its output tests guard the no-script rule, page weight, contrast, links and metadata.

**The game client.** Phase 0 built a Godot 4 client in GDScript ([ADR 0001](0001-stack.md)), with protobuf bindings from godobuf ([ADR 0003](0003-gdscript-protobuf.md)). The plan shipped native desktop builds from Phase 2 and added a Godot web export in Phase 15. Only the skeleton exists: a title screen, the golden protocol test and the dev art scene. No networking, rendering or gameplay has been written for it.

The owner has now set a new direction:

- Go is for the backend and its supporting tools. The public website moves to Astro and TypeScript.
- The game is played in the browser at play.lorehallow.com. Native desktop clients, and everything that distributes them, are retired.
- Everything stays self-hosted on the owner's Ubuntu server, and deploys stay manual.

ADR 0011's main objection to Astro was the memory each Astro/Node site uses on the shared host (52 to 86 MB). That applies to a Node process running in production. This decision runs no Node process on the host: Node builds the files, and Caddy serves them.

## Decision

### lorehallow.com: Astro

- **Astro, used directly.** `web/` becomes an Astro project: Astro components, strict TypeScript, CSS and Markdown content collections, built with Astro's static build and its built-in Vite. No Nuxt, Vue, React, Svelte or any other UI framework integration.
- **Still no browser JavaScript.** No page includes a script. Astro adds none by default, so the settings that would add one stay off (prefetch, view transitions, client islands), stylesheets are never inlined (`style-src 'self'` refuses inline styles), and an output test fails the build if any page has a `<script>`, an inline handler or a `javascript:` URL. The public site's Content-Security-Policy keeps `script-src 'none'`.
- **The same site.** Design, art, fonts and their licenses, published posts and every stable URL carry over. That includes `/progress/` and the phase pages, `/progress.json`, `/progress.svg`, the git-backed achievement feed, devlog posts and tag pages, RSS and Atom, `sitemap.xml`, `robots.txt`, the `www` and trailing-slash redirects, a real HTTP 404, and the social metadata and cards. The accessibility, contrast and page-weight budgets stay the same.
- **One BUILD.md parser, in TypeScript.** It lives in the Astro project, and Astro's build and `make buildmd-lint` both import it. It is checked against the Go parser on the real BUILD.md and the Go parser's fixtures before the Go module is deleted.
- **Generated files are build-time endpoints.** `/progress.json`, `/progress.svg`, the feeds and the sitemap are Astro static endpoints (`src/pages/*.ts` exporting `GET`), written once at build time.
- **Drafts stay local.** Posts with `draft: true` appear in the local preview and in no public artifact: no page, tag page, feed, sitemap entry or share card.
- **The Go site stays live until the Astro site matches it.** The migration starts with a parity inventory and a local Astro build. Cutover is a separate step, after the parity checks pass and the owner approves. The Go web module is deleted only after the cutover has held.
- **Downloads becomes Play.** `/downloads/` is replaced by `/play/`, which says honestly whether the game is playable and links to play.lorehallow.com once it is. After that, `/downloads/` permanently redirects to `/play/`.

### play.lorehallow.com: Phaser

- **Phaser, strict TypeScript and Vite** in `client/`, replacing Godot and GDScript. Players open a URL to play, with nothing to install.
- **The same rules.** The client sends inputs and draws what the server reports, and predicts only its own movement. The pixel rules carry over: a 640×360 base, whole-number scales only, nearest-neighbor sampling, and positions and camera snapped to whole pixels.
- **The same protocol and maps.** TypeScript bindings are generated from `proto/` by buf, and must decode and re-encode every golden vector in `proto/testdata/` byte for byte. Maps are the same Tiled files the server reads ([ADR 0006](0006-tiled-map-format.md)).
- **Checks before removal.** Godot, the GDScript bindings, the addons, export templates and Godot's Make, CI and toolchain entries are removed only after the Phaser client's checks cover what they did: golden vectors, pixel-exact art rendering, and a browser smoke test where two sessions see each other move through the real Go server.
- **No native distribution.** No desktop exports, installers, signing, storefronts, launchers, download manifests or auto-updaters, from the prototype through v1. Browser play is part of the Phase 2 prototype. Phase 15 becomes browser releases: one versioned build, protocol compatibility, cache-safe updates, smoke checks and rollback. Release notes and version tags stay.

### Shared

- **Go stays for everything behind the frontends:** the authoritative game server, bots, admin and content tools, PostgreSQL, versioned Protobuf over WebSocket, the shared Tiled maps and the pixel-art pipeline (Python for the art tools, as before).
- **One Node toolchain.** One pinned Node LTS and one package manager cover `web/` and `client/`, pinned in `toolchain.env` and checked by `make doctor`. Node is used for frontend development, builds and CI, and never runs on the host. The exact versions, the package manager and the lint, format and test tools are recorded in the BUILD.md 1.2 toolchain ADR.
- **Hosting.** Caddy on the owner's server serves two independent static builds: the website at lorehallow.com and the game at play.lorehallow.com. On play.lorehallow.com it proxies `/ws`, `/healthz`, `/version` and the game's HTTP API routes (`/api/`) to the Go server. The game's APIs share the game's origin, so the browser needs no CORS and a session cookie can be scoped to the game. `lorehallow.com/api/status` may stay as a public read-only route to the same server.
- **Two CSPs.** The website's policy refuses all scripts. The game's policy allows only its own scripts from `'self'`, connects only to its own origin (`wss://play.lorehallow.com`), and never allows `unsafe-eval` or inline scripts. Each is set in its own Caddy site block.
- **Deploys.** Both builds use the release model the website already has: a timestamped release directory, an atomic `current` symlink switch, a health check through the host's Caddy, automatic rollback, and five releases kept. They are run by hand with `make deploy-web` and `make deploy-client`. CI builds and tests, and never touches the host.

## Alternatives considered

- **Keep the Go site generator.** It works and is live. But the owner wants Go kept to the backend and its tools, and the site is the obvious frontend to move. Astro gives components, content collections and endpoints for exactly this kind of site, and the output can stay script-free.
- **Astro with islands or a UI framework.** Nothing on the public site needs client-side code. Islands would bring scripts and break the no-JavaScript rule and its CSP.
- **Godot with its web export.** It keeps one engine, but it ships a large WebAssembly runtime and loader script, needs a script and wasm exception in the CSP, and keeps GDScript bindings that need workarounds (ADR 0003's naming traps). The owner has decided the game is browser-only, and a browser-native engine fits that better.
- **PixiJS, or raw canvas and WebGL.** Both render well, but scenes, input, tilemaps, cameras, animation and audio would all have to be built. Phaser provides them, and Lorehallow's showcase is the server, not a client engine.
- **Keep native desktop builds next to the browser build.** That keeps signing, installers, per-platform testing and an update path for no player benefit, since the browser build reaches every desktop platform.
- **A Node server for either frontend in production.** Nothing needs server rendering. Static files behind Caddy are faster, simpler to roll back and use no memory on the shared host.

## Consequences

- Node returns to the toolchain ([ADR 0002](0002-toolchain-versions.md)) for development, builds and CI only. `npm audit` (or the chosen package manager's equivalent) joins `govulncheck` in CI.
- Until each cutover, both stacks live in the checkout and `make check` runs both. Old and new targets must not collide: the Astro build writes to its own output folder until cutover, and Godot is kept out of the Phaser folders.
- The Go web module's output tests, parser tests and fixtures become the parity spec. They are ported to TypeScript, not dropped.
- [ADR 0012](0012-website-fonts.md)'s fonts stay: the hand-drawn pixel font and the Go fonts, with their licenses. The code that compiles the pixel font and draws the share cards is Go today and has to move. The BUILD.md 1.2 ADR decides whether it is ported to TypeScript (the default) or becomes a Go tool in `tools/` that writes committed files.
- godobuf's naming traps (no field called `message`, no messages named like Godot classes) no longer apply once the GDScript bindings are gone. `PlayerInput` keeps its name.
- `make sync` and the art check change targets: the art and content mirrors move into the Phaser client's asset folder, and the pixel-exact art check runs in a headless browser instead of a Godot window.
- Client performance targets (Phase 7 and Phase 14) are measured in current desktop browsers.
- The decision log rows for the Godot client (1), GUT and gdtoolkit (15), godobuf (16), the Godot renderer and typing rule (23), the Go website (39) and the API host (17) are superseded.
