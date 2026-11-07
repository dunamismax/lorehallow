# 0004: Agents hand-draw the pixel art

- **Status:** Accepted
- **Date:** 2026-11-06
- **BUILD.md:** 0.12

## Context

Phase 0 made all of its art by prompting an image generator and running the results through an intake pipeline (grid detection, palette quantization, slicing). Two batches showed the limits of that approach:

- The generator draws "pixel-art style" images, not pixel art. Block sizes drift, painted scenes mix block sizes, and the generator ignores the requested logical size: sprites came out 40 to 60 px tall instead of 30, and sheets came out 156 or 250 cells across instead of 128 or 192.
- It ignores the palette, so every pixel is snapped to the nearest palette color after the fact, and the palette had to be refitted (v2) to stop browns turning red.
- It can't hold a layout or exact rules: tile edges that don't self-tile, drawn grid lines, wrong tile order, a 16×16 emblem that was an unreadable smudge.
- Each image costs a round trip through ChatGPT, often several, and a person in the loop.

The 16×16 emblem was redrawn by hand, pixel by pixel, as a text grid in a few minutes. It was the most readable asset in the batch. An agent working in text can place every pixel deliberately: exact sizes, exact palette, exact baselines, seamless edges, and every rule in STYLE.md.

## Decision

**Agents hand-draw Lorehallow's pixel art.** That is the default for every asset: tiles, characters, animations, mobs, NPCs, items, icons, UI, effects, the logo and small web art.

Image generation (the Phase 0 prompt and intake pipeline) is kept only for large painterly images where hand placement isn't practical and exact pixels don't matter: website key art, parallax backgrounds, banners and social cards. It is the exception, and any new use of it is justified in the batch manifest's notes.

### Source formats

Hand-drawn art lives as text sources under `art/src/<category>/`, mirroring `art/processed/<category>/`. A manifest asset points at its source with `source:`. `make art-draw` renders every source into `art/processed/<target>` deterministically. Two formats:

1. **Pixel grids** (`<name>.pxl.yaml`) for sprites, icons, frames and anything drawn pixel by pixel:

   ```yaml
   size: [32, 48]
   legend:            # one character per color; palette names from palette.hex
     ".": transparent
     K: ink
     S: skin mid
     G: leaf
   frames:            # one grid per frame, in reading order (a single sprite has one)
     - |
       ................................
       ...
   ```

   Colors are referred to by their palette names (`ink`, `moss shadow`, `lantern gold`), so off-palette pixels are impossible. Sheets list several frames, and the renderer lays them out on the manifest grid.

2. **Draw scripts** (`<name>.py`) for art that is better described by rules than by listing pixels: textured ground tiles with seeded noise, seamless tiling, variants, palette swaps, mirrored directions, and composites (a scene built from finished tiles and sprites). A script defines `draw(ctx) -> Image` and uses the helpers in `tools/art` (palette lookup by name, a seeded random generator, pixel, line, rectangle and ellipse primitives, 1 px outline, mirroring, wrap-around drawing for seamless tiles, and stamping `.pxl` parts). Scripts never read the clock or unseeded randomness.

Both formats go through the same checks as intake: logical size, palette only, frame count, empty trailing cells, the foot baseline, and the pivot. `make check` fails when a processed file doesn't match its source.

### Review

Agents check their own art by looking at it: `make art-draw` writes a contact sheet per batch at 4x and 8x, and the agent views it next to `art/style/reference.png` and the other assets it must sit beside, at 1x and at zoom, on light and dark backgrounds, before accepting it. Tiles are also checked repeated 3×3, so seams show.

## Alternatives considered

- **Keep generating everything and improve intake.** Intake can't add what the generator didn't draw: correct scale, layout, seamless edges or palette discipline. Every fix so far has been a workaround for one of those.
- **An external pixel editor (Aseprite) with binary sources.** Agents can't edit binary files well, and reviewers can't diff them. Text sources diff cleanly and can be generated, refactored and mirrored by code.
- **Draw scripts only.** Faces, letters, icons and silhouettes are clearer as explicit grids than as code. Both formats share one renderer, so each asset uses whichever reads better.

## Consequences

- Art becomes code: reviewable diffs, deterministic builds, and no owner round trip for most assets. The `HUMAN` "generate Batch NN" steps go away for hand-drawn batches.
- Quality depends on the agent's eye. The style guide, the reference sheet and the contact-sheet review step carry more weight, and the owner's playtest feedback is the final check.
- Phase 0's generated tiles, character, logo, emblem, icons, landmarks and lineup are redrawn by hand (BUILD.md 0.12). The large painted web images stay generated, and the style reference vignette is rebuilt as a composite of the hand-drawn assets.
- The intake pipeline, prompts and inbox stay for the exceptions above. Palette v2 stays: it was fitted to generated art, but it is a good warm palette and changing it needs another ADR.
