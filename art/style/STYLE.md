# Lorehallow style guide

The rules every piece of Lorehallow art follows. If something isn't readable at a glance at 1x, it's wrong (design pillar 3). Prompts in `art/batches/` restate these rules, and `make art-intake` enforces the mechanical ones.

**Status:** the palette (v2) is approved and locked (BUILD.md 0.8). Changing it needs an ADR.

## At a glance

| Rule | Value |
|---|---|
| Tile | 32×32 px |
| Character frame | 32×48 px, feet on row 45, pivot (16, 45) |
| Base viewport | 640×360 (20 × 11¼ tiles), integer scaling only |
| Palette | the 33 colors in [`palette.hex`](palette.hex), nothing else |
| View | top-down 3/4 |
| Light | from the top-left, always |
| Outline | 1 px of ink `#1A1420` on characters, creatures, props and items; none on ground tiles |
| Shading | 3 or 4 flat steps per material; no gradients, dithering or anti-aliasing |
| Key color | `#FF00FF` background, never inside art |
| Mood | warm storybook fantasy: mossy forests, lantern-lit towns, old stone ruins |

## The palette

![Palette](palette.png)

33 colors in ramps, dark to light. [`palette.hex`](palette.hex) is the source; `palette.png` and `palette.gpl` (for Aseprite, GIMP and Krita) are generated from it with `make art-generate`.

| Ramp | Colors | Used for |
|---|---|---|
| Shadow | ink `#1A1420`, night plum `#2E2433`, dusk violet-grey `#4A3D4D` | Outlines, deepest shadows, night |
| Stone | slate `#615A68`, stone `#857882`, pale stone `#A8979A` | Rock, walls, ruins, metal |
| Earth and wood | dark soil `#3B2A24`, bark `#74402E`, earth `#965A3A`, dry path `#C08256`, sand `#D4B48A` | Soil, paths, trunks, timber, leather, hair, cloth shading |
| Lantern light | ember `#B8642A`, amber `#E09A3C`, lantern gold `#F2C45A`, glow `#FBE39B` | Fire, lanterns, gold, warm highlights |
| Moss and leaf | deep forest `#1F3326`, moss shadow `#2F4F2F`, moss `#577139`, leaf `#7A8E3C`, sunlit leaf `#ABB84E` | Grass, foliage, moss |
| Dusk and water | deep dusk `#1F2A44`, dusk blue `#2F4466`, river `#4A6A8F`, sky `#7A9CBF`, mist `#B4CDE0` | Water, sky, cloth, cool shadows |
| Skin | skin deep `#9E5A4F`, skin mid `#CC8A6C`, skin light `#EFC4A0` | Skin (with earth tones for deeper skin) |
| Accents | banner red `#8F2F2F`, berry `#C94A3A`, arcane violet `#5E4A86`, teal glass `#3F8F86` | Banners, berries, magic, potions, gems |
| Highlight | parchment white `#F4ECD8` | The brightest highlights, eyes, paper |

Rules:

- Use only these colors. Intake snaps everything to them, so off-palette art gets changed, not kept.
- A ramp step is the unit of shading. Don't invent in-between colors.
- Hue-shift shadows toward plum and blue (shadows of green grass use deep forest, then night plum), and highlights toward gold. That warmth is the look.
- Pure ink is for outlines and the deepest crevices only. Interior detail lines use the darkest step of the material's own ramp.
- Changing the palette after approval needs an ADR: it changes every asset.

### Why v2

The first draft (v1, 32 colors) was designed by eye. Batch 01 showed that the image generator ignores exact hex values and draws each material in its own colors, and intake then snaps every pixel to the nearest palette color. Where v1 had no color near the generator's, the snap jumped to a different material: brown hair and timber became banner red and skin, the tan path split into amber and pink blotches, and warm grey flagstones turned pink. About 35% of sampled cells were more than 0.04 (OKLab) from any v1 color.

v2 moves only the ramps the generator draws differently (stone slightly warmer and more violet, earth and wood warmer and more saturated, leaf slightly yellower) and adds **sand** between dry path and parchment white for paths, cloth and wood highlights. Shadows, lantern light, dusk and water, skin and accents are unchanged. On the same Batch 01 images, cells more than 0.04 away fell to 25%, and the hair, path and flagstones keep their material. Every pair of colors stays at least 0.03 apart in OKLab, and all stay far from the key color.

## Grid and scale

- **Tiles are 32×32.** The world is built on this grid. Props may cover several tiles but are drawn on the same pixel grid.
- **Characters are 32×48:** one tile wide, one and a half tall. A person is about 30 px tall from feet to top of head, so a doorway is 32 wide and 48 tall, and a tree is two to three tiles tall.
- **One art pixel is one screen pixel at 1x.** The game renders at 640×360 and scales by whole numbers (2x at 1280×720, 3x at 1920×1080, 6x at 4K). No sprite is ever drawn at a fractional scale or rotated.
- Mixing pixel sizes is not allowed: no "high-res" details on a low-res sprite.

## View and light

- **Top-down 3/4 view.** We see the ground from above at an angle: the tops of things and their front (south-facing) sides. Walls show their top edge and front face. Characters show the top of the head and shoulders and the whole front of the body.
- **Light comes from the top-left.** Lit edges face up and left; shadows fall down and right. This never changes, including at night (night is a palette shift, not a light move).
- **No cast shadows baked into sprites.** The game draws a shared shadow under characters and props, so sprites stay clean on any ground. Contact shadows inside a single tile (a stone sitting in grass) are fine.

## Outlines and shading

- Characters, creatures, props and items get a **1-pixel ink outline**. On the lit (top-left) side the outline may use the material's darkest step instead of ink, to soften it. Never use a 2-pixel outline.
- Ground tiles have **no outline**, so they tile seamlessly.
- **3 or 4 flat shading steps** per material: shadow, base, light, and an optional highlight. Shade by form (where light hits a shape from the top-left), not from the center outward ("pillow shading" is wrong).
- **No anti-aliasing.** Edges are hard. Don't add in-between colors to smooth a curve.
- **No dithering** in game art, no gradients, no noise textures. Texture comes from deliberate clusters of 2 to 6 pixels.
- **Clean lines:** curves step in a steady pattern (1-1-2-2-3), not jagged 1-3-1-2. Avoid single orphan pixels unless they are a deliberate sparkle or eye.

## Characters

Frame: 32×48. Pivot and foot baseline at (16, 45): the lowest foot pixel sits on row 45 (counting from 0 at the top), leaving rows 46 and 47 empty. Characters are centered on column 16.

Proportions (storybook, about 2½ heads tall):

| Part | Size |
|---|---|
| Total height | 28 to 32 px, feet to top of head (hats and hair can go higher) |
| Head | 12 to 14 px tall, 12 to 14 px wide |
| Body width | 14 to 18 px at the shoulders |
| Eyes | 1×2 or 2×2 px, parchment white with an ink pupil when facing down |

Readability first: a strong silhouette, one dominant color per character, and a clear front, back and side.

### Sheets and animation

Character sheets are 6 columns × 4 rows of 32×48 frames (192×192):

| Row | Direction | Columns 0–1 | Columns 2–5 |
|---|---|---|---|
| 0 | down (facing the viewer) | idle_0, idle_1 | walk_0 … walk_3 |
| 1 | up (facing away) | idle_0, idle_1 | walk_0 … walk_3 |
| 2 | left | idle_0, idle_1 | walk_0 … walk_3 |
| 3 | right | idle_0, idle_1 | walk_0 … walk_3 |

- **Idle:** 2 frames, about 500 ms each. A gentle breath: the body and head move 1 px; the feet never move.
- **Walk:** 4 frames, about 150 ms each: contact (left foot forward), passing, contact (right foot forward), passing. The head bobs at most 1 px.
- **At least one foot touches row 45 in every frame.** Intake rejects sheets whose frames don't share the baseline.
- **Right may be the mirror of left.** Asymmetric details (a sword on one hip) should be drawn so mirroring still reads; the engine can mirror the left row when the right row isn't worth a separate drawing.
- Attack, cast, hurt and death animations are added in later batches with the same frame size and baseline.

## Tiles

- **Seamless:** every ground tile tiles with itself and with its variants in every direction. Keep strong features away from the edges.
- **2 to 4 variants** per ground type, with small differences (a tuft, a pebble) so large areas don't look like wallpaper.
- **Low contrast inside ground tiles,** so characters stay readable on top of them. The busiest tile is never busier than the characters.

### Transition sets

Where two ground types meet (grass and dirt, grass and stone) we use a 16-tile set on a 4×4 grid (128×128). "Inner" ground is the patch (dirt in a grass field); "outer" is around it.

| | col 0 | col 1 | col 2 | col 3 |
|---|---|---|---|---|
| row 0 | edge_tl | edge_t | edge_tr | inner_tl |
| row 1 | edge_l | center | edge_r | inner_tr |
| row 2 | edge_bl | edge_b | edge_br | inner_bl |
| row 3 | edge_t_b | center_b | edge_b_b | inner_br |

- The 3×3 block in the top-left, placed together, forms one complete patch of the inner ground surrounded by the outer ground.
- **Inner corners** are the opposite: the outer ground pokes into a corner of the tile (used where a patch has a concave corner). `inner_tl` has outer ground only in its top-left corner.
- Row 3 holds variants: another top edge, another center and another bottom edge.
- Edges are organic (wavy, with a 1 to 2 px lip of the darker ground on the shadow side), but meet the tile border at the same points in every tile so they connect.

## Props, items and UI

- Props (trees, lanterns, fences, crates) follow the character rules: ink outline, top-left light, sit on the 32 px grid.
- Item icons are 16×16 or 32×32 with an ink outline, centered, readable as a silhouette.
- UI frames are 9-slice pixel art from the same palette. Text uses a pixel font at integer sizes (chosen in Phase 1).

## Web art

Website art (Batch 02 and later) uses the same palette, outline and light rules, at larger logical sizes (for example a 240×160 hero scene shown at 4x or 6x). It is always displayed at whole-number scales with `image-rendering: pixelated`.

## Do and don't

| Do | Don't |
|---|---|
| Flat color clusters from the palette | Gradients, soft brushes, noise, dithering |
| Hard 1 px ink outlines on sprites | Blurry or anti-aliased edges, 2 px outlines |
| Light from the top-left every time | Light from the right, rim light from behind |
| Feet on row 45 in every frame | Bobbing the whole sprite up and down while walking |
| Low-contrast ground, high-contrast characters | Busy ground that swallows characters |
| Warm highlights (gold) and cool shadows (plum, blue) | Pure black shadows, grey highlights |
| Whole-number scaling everywhere | Rotating, stretching or 1.5x scaling sprites |
| `#FF00FF` behind everything that isn't art | `#FF00FF` (or anything near it) inside the art |

## Delivering art (for generated images)

Generated images are drawn on a flat `#FF00FF` background at any resolution; intake finds the pixel grid, downsamples, removes the background and snaps to the palette. Prompts ask for complete sheets in strict grids, and the owner drops the results unchanged into `art/inbox/batch-NN/`. See [AGENTS.md section 8](../../AGENTS.md#8-art-pipeline) and [tools/art/README.md](../../tools/art/README.md).
