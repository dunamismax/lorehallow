# Phase 1 walkthrough, part 2: moving a live site without breaking a URL

The first [Phase 1 walkthrough](phase-01-website.md) built lorehallow.com with a small Go program. This one moves that live site to Astro ([ADR 0014](../decisions/0014-astro-website-and-phaser-browser-client.md)) without changing a single URL, feed entry or pixel, and replaces Downloads with Play along the way. The original walkthrough stays as history: most of what it explains (Caddy, TLS, the release model) didn't change.

## What we built

- **A parity inventory** ([docs/design/website-parity.md](../design/website-parity.md)): every URL, redirect, status code, generated asset, meta tag and output test the live site had, and how the new build is checked against each.
- **One toolchain for both frontends** ([ADR 0015](../decisions/0015-frontend-toolchain.md)): Node 26 LTS, npm workspaces with one lockfile, TypeScript run directly by Node, Biome, and `node --test`.
- **One BUILD.md parser, in TypeScript** (`web/src/lib/buildmd.ts`), shared by the Astro build and `make buildmd-lint`, held to the Go parser by a parity test.
- **The Astro site** in `web/`: the same pages, components and stylesheet, the devlog as a content collection read in place, and `/progress.json`, `/progress.svg`, the feeds and the sitemap as build-time endpoints. No browser JavaScript.
- **The image and font code, ported** ([ADR 0016](../decisions/0016-fonts-and-share-cards-in-typescript.md)): the pixel-font compiler (byte for byte the same TrueType), WOFF, a PNG reader and writer, frame crops, mirrored pairs and the share cards.
- **A parity check** (`make web-parity`) that builds both sites from the same commit and compares all 153 files.
- **/play** in place of /downloads, with a 301 redirect for the old address.
- **The cutover**: the deploy builds with Astro, a stronger health check, the last Go-built release kept as a rollback target, and a rehearsed automatic rollback.

## How to try it

```bash
make web-dev                 # the Astro site at http://127.0.0.1:4321, drafts included
make web-parity              # build both sites from this commit and compare them
make web-test                # the output tests, the parser parity test, and the parity check
make deploy-web              # build with Astro, upload, switch, check, roll back on failure
make rehearse-rollback-web   # prove the automatic rollback still works
```

Change one word in a page under `web/src/pages/` and run `make web-parity`: it names the file and shows both builds around the first difference.

## How it works

### Inventory first

Before writing any Astro, the inventory listed everything a visitor, a feed reader or a search engine could see: 40 pages, the feeds, the data files, the icons, every `/assets/` file, every meta tag, the redirects Caddy answers (`www`, `http`, a directory without its slash) and the 404's status code. Each line says how it will be checked. That list is the definition of "nothing broke", written while the old site was still the only truth.

### Port the code, keep the output

The Astro project isn't a redesign. Each Go template became an Astro page with the same markup, and each Go package a TypeScript module that produces the same bytes, or at least the same meaning:

- The BUILD.md parser had to agree with the Go one on every input, including broken ones. Two details mattered: Go's `\s` and `\S` are ASCII only while JavaScript's are Unicode, and Go's `strings.TrimSpace` and JavaScript's `trim()` disagree on a few characters. The TypeScript regexes spell out `[\t\n\f\r ]` and the parser has its own `trimSpace`.
- The font compiler is pure arithmetic, so it reproduces the Go compiler's TrueType file exactly, checksums and all.
- PNG and WOFF files go through zlib, and Go's and Node's deflate pick different encodings. The bytes differ; the pixels and font tables don't. The parity check compares what matters: decoded pixels and decompressed tables.
- Assets keep the old URL scheme, `/assets/<name>.<hash>.<ext>`, so Caddy's year-long cache rule didn't need to change.

One piece needed a different shape. The stylesheet carries a responsive whole-number-scale rule for every pixel-art image the pages drew, so it can only be written after the last page. Astro renders pages in its own bundled modules, so the pages register assets in a registry on `globalThis`, and a small integration (`web/src/integration.ts`) writes the files and the stylesheet in `astro:build:done`, then points every page at the stylesheet's hashed name. In `astro dev` the same integration serves those files from memory.

### The parity check

```
       commit X
      /        \
 Go build    Astro build
      \        /
   compare every file
   (HTML and CSS with whitespace collapsed, feeds, JSON,
    PNG pixels, font tables, everything else byte for byte)
```

The check runs in `make check`, so for as long as both builds existed, any change that made them disagree failed CI. It found real differences on the way, such as the footer's build date (Go formats it in the machine's time zone, and the first port used UTC). Differences that are meant to be there are listed with their reason in the script and the inventory: the build timestamp in `/progress.json`, and the one sentence on About that says which generator built the site.

The checker was itself checked: changing one word in one built page made it fail, and a draft post with its `draft` flag removed showed up in three files of the output, so the "no drafts" test isn't passing by accident.

### Changing the site during a migration

Play replaced Downloads in the middle of the move. Rather than teach the parity check to ignore a new page, a new nav label and new FAQ answers, the same change landed in both builds in one commit, and the check kept comparing everything. The live (still Go-built) site got /play and the redirect a few minutes later.

### The cutover

1. Copy the last Go-built release to `releases/go-final/` on the host, where no deploy will rotate it away.
2. `make deploy-web`, which now builds with Astro on the machine running the deploy. Node never goes to the host; Caddy only ever serves files.
3. The health check, now stricter: the stylesheet and share card load with immutable caching, pages revalidate, canonical and share-card URLs are absolute, the feeds, data files, /play and the sitemap load, `/downloads/` answers 301, and a missing page answers 404.
4. Rehearse the rollback with an Astro release: a release without its stylesheet fails the check and the previous one comes back on its own.
5. Sweep every URL from outside the host.

A week later, with no rollback, the Go module is deleted.

## The key concept: the old system is the specification

A rewrite usually starts from a description of what the system should do, and the description is never complete. A migration can do better, because the old system is running and its output is the most complete specification there is. Build the new one next to it, feed both the same input, and compare outputs mechanically. Every difference is then either a bug in the new system or a decision you write down. Only when the two agree do you switch traffic, and even then the old one stays ready to take over.

The same pattern shows up all over backend work: shadow traffic for a new service, dual writes during a database migration, golden files for a protocol. Lorehallow's protocol already uses it (Go and the client must re-encode the same golden vectors byte for byte), and the Phaser client will meet the same bar before Godot is removed (BUILD.md 2.6).

## Read the code

1. [docs/design/website-parity.md](../design/website-parity.md): what had to stay the same.
2. `web/src/lib/buildmd.ts`, then `web/tests/buildmd-parity.test.ts`: one parser, held to another on broken input.
3. `web/scripts/parity.ts`: the parity check, `compare` and `ACCEPTED`.
4. `web/src/lib/state.ts`, `web/src/lib/assets.ts` and `web/src/integration.ts`: the asset registry and why the stylesheet comes last.
5. `web/src/lib/pixelfont.ts`: tracing pixels into TrueType outlines (`contours`) and writing the font tables.
6. `web/src/layouts/Layout.astro` and `web/src/pages/progress/[phase].astro`: what a ported page looks like.
7. `deploy/web/deploy.sh`, `check_live`: the cutover's health check.

## What's next

Phase 2 builds the game client the same way: a Phaser browser game in `client/` that has to pass the same golden protocol vectors and pixel checks as the Godot skeleton before Godot is removed, and then two players see each other move at play.lorehallow.com.
