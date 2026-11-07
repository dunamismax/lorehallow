# 0012: Website fonts: a hand-drawn pixel font and the Go fonts

- **Status:** Accepted
- **Date:** 2026-11-07
- **BUILD.md:** 1.2

## Context

lorehallow.com needs a pixel font for headings, navigation and buttons, and a readable font for body text. AGENTS.md rules out third-party requests, so both are self-hosted, and the license of each must be committed next to it. The site is built by the Go generator in `web/` (ADR 0011), and its only third-party dependency so far is goldmark.

The art rules apply to a heading font too. It should share the pixel grid and the hand-drawn look of the rest of the art, and look crisp at whole-number scales.

## Decision

**Headings: Lorehallow Pixel, drawn by hand for this site.** The source is `web/static/fonts/lorehallow-pixel.txt`: one text grid per glyph, `#` for a pixel, in the same spirit as the `.pxl.yaml` sprites (ADR 0004). It covers ASCII plus the typographic quotes, dashes, the ellipsis, ×, · and two arrows. Capitals are 7 pixels tall, lowercase has a 5-pixel x-height and 2-pixel descenders, and every glyph is proportional with one pixel of spacing. It is licensed CC BY-NC 4.0 with the rest of the art, and that license is written into the font's name table.

`web/internal/pixelfont` compiles the grid into a TrueType font at build time. It traces each glyph's pixels into clockwise outlines with the holes counter-clockwise, merges straight runs, and writes the ten required tables with fixed timestamps so the output is reproducible. `web/internal/woff` wraps it in WOFF 1.0. WOFF 1.0 uses zlib, which is in the standard library; WOFF 2 would need Brotli. One font pixel is 100 units and the em is 10 pixels, so the font is pixel-exact at 20, 30 and 40px, and the stylesheet uses only those sizes. The same parser draws page titles onto the Open Graph cards and the `/progress.svg` badge, so they need no font at all.

**Body: the Go fonts** (Go Regular, Italic, Bold and Go Mono for code), designed by Bigelow & Holmes for the Go project and released under the Go BSD license. The TTFs come from the `golang.org/x/image/font/gofont/ttfs` module, are committed in `web/static/fonts/` with `LICENSE-GoFonts.txt`, and are served as WOFF. They are humanist and very legible at small sizes, and they fit a site whose toolchain is Go.

All fonts are embedded in the generator with `go:embed`, published under `/assets/` with a content hash in the name, and loaded with `font-display: swap`. The pixel font and Go Regular are preloaded.

## Alternatives considered

- **An open-licensed pixel font (Silkscreen, Pixelify Sans, Press Start 2P and others, all SIL OFL).** These are good fonts, but they bring a third-party look to a project whose art is hand-drawn, and they're designed on other grids. Drawing about a hundred glyphs is a small job next to the 21 landmarks.
- **A serif body font (Literata, Source Serif) for a storybook feel.** These would need downloading from outside the Go module proxy. The warmth comes from the palette, the pixel headings and the parchment devlog pages, and the Go fonts read well on all of them.
- **WOFF 2.** About 30% smaller, but it needs a Brotli encoder, which isn't in the standard library. All the fonts together are about 300 KB as WOFF, well inside the landing page's 1 MB budget.
- **Subsetting the Go fonts.** That would save about half their size, but it needs a font subsetter in Go. Revisit if the page weight gets tight.

## Consequences

- The pixel font can be edited like any other art: change the grid and rebuild. Its tests check that it parses, covers the characters headings use, and compiles to identical bytes every time.
- Characters missing from the pixel font fall back to Go per glyph, so non-ASCII text in headings still renders.
- The Godot client can use the same TTF later. If it does, the source should move to `art/src/` with a sync step, like the rest of the art.
