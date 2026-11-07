<!-- Generated from prompts.template.md by `make art-prompts`. Edit the template, not this file. -->

# Batch 01: style foundation

Five images that set the look of Lorehallow: a style reference vignette, one character concept and three terrain tilesets.

**How to run this batch (owner):**

1. Open a new ChatGPT conversation with image generation.
2. Paste each prompt below in order, one per message. There is no reference sheet to attach yet: this batch creates it.
3. Download each result and drop it, unchanged and with its original filename, into `art/inbox/batch-01/`.
4. If an image comes out obviously wrong (smooth painting, wrong grid, text on it), ask ChatGPT to try again before downloading.

An agent then maps the files, runs `make art-intake BATCH=01` and writes corrected prompts into `report.md` for anything that needs another try.

Every prompt repeats the full rules on purpose: image models follow instructions in the same message far better than earlier ones.

## 01-01 style_reference: style reference vignette

- Target: art/processed/style/reference.png (then copied to art/style/reference.png)
- Logical size: 240×160 art pixels (landscape, 3:2)
- Attach: nothing

```text
Create a pixel-art scene for a cozy fantasy online game called Lorehallow. This image will be the style reference for every later piece of art, so follow the rules exactly.

Canvas: landscape 3:2. The whole image is exactly 240 art pixels wide and 160 art pixels tall. Every art pixel is drawn as one solid square block of a single color, all blocks the same size, aligned to one grid that fills the canvas edge to edge.

True pixel art: every art pixel is a solid square block of one color. No anti-aliasing, no gradients, no blur, no texture noise, no dithering, no soft brushes.

Palette: use only these 33 colors and no others: #1A1420, #2E2433, #4A3D4D, #615A68, #857882, #A8979A, #3B2A24, #74402E, #965A3A, #C08256, #D4B48A, #B8642A, #E09A3C, #F2C45A, #FBE39B, #1F3326, #2F4F2F, #577139, #7A8E3C, #ABB84E, #1F2A44, #2F4466, #4A6A8F, #7A9CBF, #B4CDE0, #9E5A4F, #CC8A6C, #EFC4A0, #8F2F2F, #C94A3A, #5E4A86, #3F8F86, #F4ECD8.

View and light: top-down 3/4 view, like a classic top-down fantasy RPG. Light always comes from the top-left: lit edges face up and left, shadows fall down and right.

Scene: a small clearing at the edge of a mossy forest at golden hour, built on a 32×32 tile grid (7½ tiles wide, 5 tiles tall). Soft grass with a few tufts and small flowers, a winding dry dirt path from the bottom left to the right, a patch of old cracked flagstones with moss in the cracks, two or three trees with round leafy crowns (each about 2 to 3 tiles tall), a mossy rock, a wooden lantern post with a warm glowing lantern, and the corner of a small timber-and-stone cottage with a wooden door. One adventurer stands on the path facing the viewer: 32 art pixels wide and 48 tall including empty space, about 30 art pixels tall from feet to the top of the head, about 2½ heads tall, with a short green cloak, brown leather boots, a satchel and a simple tunic.

Outlines: characters, trees, the rock, the lantern and the cottage have a 1-pixel outline in #1A1420. Ground (grass, dirt, flagstones) has no outline.

Shading: 3 or 4 flat steps per material, shaded by form from the top-left light. Warm gold highlights, cool plum and blue shadows. No cast shadows except small contact shadows directly under objects.

Mood: warm storybook fantasy, peaceful, inviting, readable at a glance.

No text, no letters, no signature, no watermark, no border or frame.
```

## 01-02 concept_hero: character concept

- Target: art/processed/chars/concept_hero.png
- Logical size: 32×48 art pixels (portrait, 2:3)
- Attach: nothing

```text
Create a single pixel-art character sprite for a cozy fantasy online game called Lorehallow.

Canvas: portrait 2:3. The whole image is exactly 32 art pixels wide and 48 art pixels tall. Every art pixel is drawn as one solid square block of a single color, all blocks the same size, aligned to one grid that fills the canvas edge to edge.

True pixel art: every art pixel is a solid square block of one color. No anti-aliasing, no gradients, no blur, no texture noise, no dithering.

Palette: use only these 33 colors and no others: #1A1420, #2E2433, #4A3D4D, #615A68, #857882, #A8979A, #3B2A24, #74402E, #965A3A, #C08256, #D4B48A, #B8642A, #E09A3C, #F2C45A, #FBE39B, #1F3326, #2F4F2F, #577139, #7A8E3C, #ABB84E, #1F2A44, #2F4466, #4A6A8F, #7A9CBF, #B4CDE0, #9E5A4F, #CC8A6C, #EFC4A0, #8F2F2F, #C94A3A, #5E4A86, #3F8F86, #F4ECD8.

The character: a young adventurer standing still, facing the viewer (facing down the screen), arms relaxed, in top-down 3/4 view, so we see the top of the head and the whole front of the body. Storybook proportions, about 2½ heads tall: head about 13 art pixels tall, total height about 30 art pixels from the feet to the top of the hair. Short green cloak over a cream tunic, brown leather belt and boots, a small satchel at the hip, warm brown hair. Eyes are 2 art pixels tall with a dark pupil.

Placement: centered horizontally. The lowest pixel of the feet sits exactly on row 45 counting from 0 at the top, so the bottom two rows are empty background.

Light from the top-left. 1-pixel outline in #1A1420 around the whole character. 3 or 4 flat shading steps per material. No shadow on the ground.

Background: flat solid #FF00FF everywhere that is not the character. Never use #FF00FF inside the character.

No text, no signature, no watermark, no border or frame.
```

## 01-03 terrain_base: base terrain tileset

- Target: art/processed/tiles/terrain_base.png
- Logical size: 4 columns × 4 rows of 32×32 tiles (128×128 total, square)
- Attach: nothing

```text
Create a pixel-art tileset for a cozy fantasy online game called Lorehallow.

Canvas: square. The whole image is exactly 128 art pixels wide and 128 art pixels tall: a grid of 4 columns × 4 rows of tiles, each tile exactly 32×32 art pixels, with no gaps, borders or labels between tiles. Every art pixel is drawn as one solid square block of a single color, all blocks the same size, aligned to one grid that fills the canvas edge to edge.

True pixel art: every art pixel is a solid square block of one color. No anti-aliasing, no gradients, no blur, no texture noise, no dithering.

Palette: use only these 33 colors and no others: #1A1420, #2E2433, #4A3D4D, #615A68, #857882, #A8979A, #3B2A24, #74402E, #965A3A, #C08256, #D4B48A, #B8642A, #E09A3C, #F2C45A, #FBE39B, #1F3326, #2F4F2F, #577139, #7A8E3C, #ABB84E, #1F2A44, #2F4466, #4A6A8F, #7A9CBF, #B4CDE0, #9E5A4F, #CC8A6C, #EFC4A0, #8F2F2F, #C94A3A, #5E4A86, #3F8F86, #F4ECD8.

View and light: top-down 3/4 view of the ground. Light from the top-left.

Every tile is seamless: it tiles with itself and with the other tiles of the same ground type on all four sides. Keep strong features away from the tile edges. Keep contrast inside each tile low so characters stay readable on top. No outlines on ground tiles.

Tiles in reading order (left to right, top to bottom):
Row 1: soft grass, grass variant with slightly different blade clusters, grass variant 3, grass with a few tiny flowers.
Row 2: grass with one taller tuft, packed dry dirt, dirt variant, dirt with a few pebbles.
Row 3: grey stone ground, stone variant, cracked stone, stone with moss in the cracks.
Row 4: cobblestones, cobblestone variant, large old flagstones, grass with a few small stones.

Grass uses the moss and leaf greens with deep forest shadows; dirt uses the earth browns; stone uses the slates and greys. Warm storybook mood.

No text, no letters, no numbers, no signature, no watermark, no border or frame.
```

## 01-04 terrain_grass_dirt: grass to dirt transitions

- Target: art/processed/tiles/terrain_grass_dirt.png
- Logical size: 4 columns × 4 rows of 32×32 tiles (128×128 total, square)
- Attach: art/batches/batch-01-style/guides/terrain_grass_dirt.png (layout guide), then art/style/reference.png

```text
Create a pixel-art transition tileset for a cozy fantasy online game called Lorehallow: a patch of packed dirt inside a grass field.

Canvas: square. The whole image is exactly 128 art pixels wide and 128 art pixels tall: a grid of 4 columns × 4 rows of tiles, each tile exactly 32×32 art pixels, with no gaps, borders or labels between tiles. Every art pixel is drawn as one solid square block of a single color, all blocks the same size, aligned to one grid that fills the canvas edge to edge.

True pixel art: every art pixel is a solid square block of one color. No anti-aliasing, no gradients, no blur, no texture noise, no dithering.

Palette: use only these 33 colors and no others: #1A1420, #2E2433, #4A3D4D, #615A68, #857882, #A8979A, #3B2A24, #74402E, #965A3A, #C08256, #D4B48A, #B8642A, #E09A3C, #F2C45A, #FBE39B, #1F3326, #2F4F2F, #577139, #7A8E3C, #ABB84E, #1F2A44, #2F4466, #4A6A8F, #7A9CBF, #B4CDE0, #9E5A4F, #CC8A6C, #EFC4A0, #8F2F2F, #C94A3A, #5E4A86, #3F8F86, #F4ECD8.

View and light: top-down 3/4 view of the ground. Light from the top-left. No outlines.

Two images are attached. The first is a layout guide in two flat colors: green is where grass goes, brown is where dirt goes. Follow its shapes and positions exactly, tile by tile, but draw real textured ground in place of the flat colors and make the borders organic instead of straight. The second is the style reference: match its grass, ground colors, pixel size and mood.

The 16 tiles form one continuous image. Do not draw any lines, gaps, outlines, shadows or color changes along the tile boundaries; a viewer should not be able to see where one tile ends and the next begins, except where the layout itself changes.

Layout, in reading order:
- The 3×3 block in the top-left (columns 1 to 3, rows 1 to 3) is one complete dirt patch surrounded by grass when the nine tiles sit together: top-left corner, top edge, top-right corner; left edge, full dirt center, right edge; bottom-left corner, bottom edge, bottom-right corner. Grass is on the outside, dirt on the inside.
- Column 4, rows 1 to 4: inner corners, where the tile is dirt except for a small rounded bite of grass in one corner: grass in the top-left corner, then top-right, then bottom-left, then bottom-right.
- Row 4, columns 1 to 3: variants: another top edge, another full dirt center, another bottom edge.

The grass-to-dirt border is organic and slightly wavy, with a 1 to 2 pixel darker lip of grass on the shadow side, but it meets the tile borders at the same positions in every tile so all tiles connect seamlessly. Grass matches soft green meadow grass; dirt is packed dry earth brown. Low contrast, warm storybook mood.

No text, no letters, no numbers, no signature, no watermark, no border or frame.
```

## 01-05 terrain_grass_stone: grass to flagstone transitions

- Target: art/processed/tiles/terrain_grass_stone.png
- Logical size: 4 columns × 4 rows of 32×32 tiles (128×128 total, square)
- Attach: art/batches/batch-01-style/guides/terrain_grass_stone.png (layout guide), then art/style/reference.png

```text
Create a pixel-art transition tileset for a cozy fantasy online game called Lorehallow: an area of old grey flagstones inside a grass field, like the floor of a ruin being reclaimed by nature.

Canvas: square. The whole image is exactly 128 art pixels wide and 128 art pixels tall: a grid of 4 columns × 4 rows of tiles, each tile exactly 32×32 art pixels, with no gaps, borders or labels between tiles. Every art pixel is drawn as one solid square block of a single color, all blocks the same size, aligned to one grid that fills the canvas edge to edge.

True pixel art: every art pixel is a solid square block of one color. No anti-aliasing, no gradients, no blur, no texture noise, no dithering.

Palette: use only these 33 colors and no others: #1A1420, #2E2433, #4A3D4D, #615A68, #857882, #A8979A, #3B2A24, #74402E, #965A3A, #C08256, #D4B48A, #B8642A, #E09A3C, #F2C45A, #FBE39B, #1F3326, #2F4F2F, #577139, #7A8E3C, #ABB84E, #1F2A44, #2F4466, #4A6A8F, #7A9CBF, #B4CDE0, #9E5A4F, #CC8A6C, #EFC4A0, #8F2F2F, #C94A3A, #5E4A86, #3F8F86, #F4ECD8.

View and light: top-down 3/4 view of the ground. Light from the top-left. No outlines.

Two images are attached. The first is a layout guide in two flat colors: green is where grass goes, grey is where flagstone goes. Follow its shapes and positions exactly, tile by tile, but draw real textured ground in place of the flat colors and make the borders organic instead of straight. The second is the style reference: match its grass, ground colors, pixel size and mood.

The 16 tiles form one continuous image. Do not draw any lines, gaps, outlines, shadows or color changes along the tile boundaries; a viewer should not be able to see where one tile ends and the next begins, except where the layout itself changes.

Layout, in reading order:
- The 3×3 block in the top-left (columns 1 to 3, rows 1 to 3) is one complete flagstone area surrounded by grass when the nine tiles sit together: top-left corner, top edge, top-right corner; left edge, full flagstone center, right edge; bottom-left corner, bottom edge, bottom-right corner. Grass is on the outside, flagstones on the inside.
- Column 4, rows 1 to 4: inner corners, where the tile is flagstone except for a small rounded bite of grass in one corner: grass in the top-left corner, then top-right, then bottom-left, then bottom-right.
- Row 4, columns 1 to 3: variants: another top edge, another full flagstone center, another bottom edge.

Flagstones are large, slightly irregular grey slabs with thin dark seams and a little moss in the seams. Grass creeps a few pixels over the stone at the border, which is irregular but meets the tile borders at the same positions in every tile so all tiles connect seamlessly. Low contrast, warm storybook mood.

No text, no letters, no numbers, no signature, no watermark, no border or frame.
```
