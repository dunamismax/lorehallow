# 0016: Fonts and share cards in the Astro build

- **Status:** Accepted
- **Date:** 2026-11-07
- **BUILD.md:** 1.2

## Context

The Go site generator does three kinds of image and font work that Astro doesn't do for us ([ADR 0012](0012-website-fonts.md), [ADR 0014](0014-astro-website-and-phaser-browser-client.md)):

- it compiles the hand-drawn pixel font (`web/static/fonts/lorehallow-pixel.txt`) into TrueType and wraps it, and the Go fonts, in WOFF;
- it draws a 1200×630 share card for every page title: the `og_card` art at exactly 5x with a plaque and the title in the pixel font;
- it crops named frames from sprite sheets and builds mirrored pairs for the parallax layers, then publishes every asset under a content-hashed name.

ADR 0014 left one choice open: port that code to TypeScript, or keep it as a Go tool in `tools/` that writes committed files.

Astro's own image pipeline isn't an option for any of it. It resamples and re-encodes images, and pixel art must reach the browser with exactly the pixels it was drawn with.

## Decision

Port it to TypeScript, inside the Astro project, with no new dependencies:

- `src/lib/pixelfont.ts`: the font source parser, the outline tracer and the TrueType compiler. Its output is the same, byte for byte, as `internal/pixelfont`'s.
- `src/lib/woff.ts`: WOFF 1.0 with `node:zlib`. Deflate output differs between Go and Node, so the WOFF files differ in bytes, but every table decompresses to the same bytes as the Go build's.
- `src/lib/png.ts`: a PNG reader and writer for the 8-bit images the art pipeline writes, on `node:zlib`. It never resamples.
- `src/lib/og.ts`: the share-card renderer. Cards have the same pixels as the Go build's.
- `src/lib/assets.ts`: the content-hashed `/assets/` registry, frame crops, mirrored pairs and the responsive whole-number-scale rules.

The fonts' sources and licenses don't change. The parity check (`web/scripts/parity.ts`) compares font tables and decoded pixels between the two builds, so any drift fails `make check`.

Assets keep the Go generator's URL scheme, `/assets/<name>.<10 hex digits of SHA-256>.<ext>`, rather than Astro's `/_astro/`. Caddy's immutable-cache rule for `/assets/*` stays as it is, and the parity check pairs assets by name. Pages register what they draw while Astro renders them; a small integration (`src/integration.ts`) writes the files and the stylesheet once every page is done, because the stylesheet carries the responsive rules for exactly the images the pages used.

## Alternatives considered

- **A Go tool in `tools/` writing committed files.** The code exists and works. But every font or palette change would need a second command and a commit of generated binaries, the Astro build would depend on Go after the Go site is retired, and the owner wants Go kept to the backend.
- **npm packages** (`pngjs`, `opentype.js`, a WOFF library). Each is a dependency to audit for a few hundred lines of code we already understand, and none produces the exact bytes the Go compiler did.
- **Astro's `astro:assets`.** It optimizes and resizes, which is exactly what pixel art can't have.

## Consequences

- The whole site builds with Node alone once the Go module is gone (BUILD.md 1.9).
- PNG crops and WOFF files get new hashes in their URLs at the cutover; browsers fetch them once.
- The font compiler and share cards are now TypeScript that `make check` type-checks and the parity check compares; the Go copies are deleted with the Go module.
