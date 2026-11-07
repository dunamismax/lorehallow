# Batch 02: brand and website art

The website's large painted images. Same palette and rules as the game, at larger sizes. The logo, emblems, section icons, progress landmarks, character lineup and 404 art are hand-drawn (`art/src/`, ADR 0004), so they have no prompts; only the painted scenes below are generated. The adventurers in the hero scene and the social card are painted out and replaced with the hand-drawn characters by `art/src/web/hero_scene.py` and `og_card.py`, so a regenerated painting only needs its figures in roughly the same places.

**How to run this batch (owner):**

1. Every prompt here attaches the style reference, `art/style/reference.png` (hand-drawn, built from the game's own tiles and sprites).
2. Open a new ChatGPT conversation with image generation.
3. For each prompt below, in order: attach `art/style/reference.png`, paste the prompt, and send.
4. Download each result and drop it, unchanged and with its original filename, into `art/inbox/batch-02/`.
5. If an image comes out obviously wrong (smooth painting, misspelled logo, text where none was asked for), ask ChatGPT to try again before downloading.

An agent then maps the files, runs `make art-intake BATCH=02` and writes corrected prompts into `intake-report.md` for anything that needs another try.

## 02-04 hero_scene: hero key art

- Target: art/processed/web/hero_scene.png
- Logical size: 240×160 art pixels (landscape 3:2)
- Attach: art/style/reference.png

```text
Create pixel art for a cozy fantasy online game called Lorehallow. Attached is the style reference; match it.

Canvas: landscape 3:2. The art is exactly 240 art pixels wide and 160 art pixels tall, every art pixel drawn as one solid square block of a single color, all blocks the same size. The art fills the whole canvas edge to edge.

The key art for the game's website. A lantern-lit fantasy village at the edge of a mossy forest at dusk, seen in top-down 3/4 view. A cobbled path winds from the bottom left toward a small square with a well. Timber-and-stone cottages with warm glowing windows, a tall old tree, fireflies, distant hills under a violet and gold dusk sky in the upper third. Four small adventurers (each about 30 art pixels tall, about one fifth of the image height, the same scale as the reference character) are on the path: one waving, one carrying a lantern, one with a shield on their back, one sitting on the well's edge. The feeling: you just arrived and other people are already here. Leave the top-left quarter calm and uncluttered (sky and hills) so a title can sit over it.

True pixel art: every art pixel is a solid square block of one color, all blocks the same size, aligned to one grid. No anti-aliasing, no gradients, no blur, no texture noise, no dithering, no soft brushes.

Palette: use only these {{PALETTE_COUNT}} colors and no others: {{PALETTE}}.

Match the style of the attached reference image exactly: the same palette, pixel size relative to the subject, 1-pixel #1A1420 outlines on characters and objects, flat 3 to 4 step shading, top-down 3/4 view and light from the top-left. Warm storybook fantasy mood: mossy forests, lantern-lit towns, old stone ruins.

No text, letters or numbers unless asked for below. No signature, no watermark, no border or frame.
```

## 02-05 parallax_far: parallax back layer

- Target: art/processed/web/parallax_far.png
- Logical size: 240×160 art pixels (landscape 3:2)
- Attach: art/style/reference.png

```text
Create pixel art for a cozy fantasy online game called Lorehallow. Attached is the style reference; match it.

Canvas: landscape 3:2. The art is exactly 240 art pixels wide and 160 art pixels tall, every art pixel drawn as one solid square block of a single color, all blocks the same size. The art fills the whole canvas edge to edge.

The back layer of a three-layer parallax background for the website, matching the hero key art. Dusk sky (deep dusk blue at the top, through dusk violet to amber and lantern gold near the horizon) with a few long flat pixel clouds and a handful of single-pixel stars near the top. Two ranges of rolling hills along the bottom third, the far one misty blue, the nearer one deep forest green with a tiny castle silhouette on one hill. No characters. Fully opaque: no background color anywhere.

True pixel art: every art pixel is a solid square block of one color, all blocks the same size, aligned to one grid. No anti-aliasing, no gradients, no blur, no texture noise, no dithering, no soft brushes.

Palette: use only these {{PALETTE_COUNT}} colors and no others: {{PALETTE}}.

Match the style of the attached reference image exactly: the same palette, pixel size relative to the subject, 1-pixel #1A1420 outlines on characters and objects, flat 3 to 4 step shading, top-down 3/4 view and light from the top-left. Warm storybook fantasy mood: mossy forests, lantern-lit towns, old stone ruins.

No text, letters or numbers unless asked for below. No signature, no watermark, no border or frame.
```

## 02-06 parallax_mid: parallax middle layer

- Target: art/processed/web/parallax_mid.png
- Logical size: 240×160 art pixels (landscape 3:2)
- Attach: art/style/reference.png

```text
Create pixel art for a cozy fantasy online game called Lorehallow. Attached is the style reference; match it.

Canvas: landscape 3:2. The art is exactly 240 art pixels wide and 160 art pixels tall, every art pixel drawn as one solid square block of a single color, all blocks the same size. The art fills the whole canvas edge to edge.

The middle layer of a three-layer parallax background, matching the hero key art. Along the lower half: the edge of a mossy forest (rounded tree crowns of different heights) on the left and right, with the roofs and chimneys of a small village with warm glowing windows between them. The top half and every gap between trees and roofs are background, so the sky layer shows through. No characters. The bottom edge of the art is a solid band of forest and roofs.

True pixel art: every art pixel is a solid square block of one color, all blocks the same size, aligned to one grid. No anti-aliasing, no gradients, no blur, no texture noise, no dithering, no soft brushes.

Palette: use only these {{PALETTE_COUNT}} colors and no others: {{PALETTE}}.

Match the style of the attached reference image exactly: the same palette, pixel size relative to the subject, 1-pixel #1A1420 outlines on characters and objects, flat 3 to 4 step shading, top-down 3/4 view and light from the top-left. Warm storybook fantasy mood: mossy forests, lantern-lit towns, old stone ruins.

No text, letters or numbers unless asked for below. No signature, no watermark, no border or frame.

Background: flat solid {{KEY}} everywhere that is not art. Never use {{KEY}} inside the art.
```

## 02-07 parallax_near: parallax front layer

- Target: art/processed/web/parallax_near.png
- Logical size: 240×160 art pixels (landscape 3:2)
- Attach: art/style/reference.png

```text
Create pixel art for a cozy fantasy online game called Lorehallow. Attached is the style reference; match it.

Canvas: landscape 3:2. The art is exactly 240 art pixels wide and 160 art pixels tall, every art pixel drawn as one solid square block of a single color, all blocks the same size. The art fills the whole canvas edge to edge.

The front layer of a three-layer parallax background, matching the hero key art. Only the bottom quarter has art: a strip of tall grass, ferns and wildflowers across the whole width, a mossy rock on the left, and a wooden lantern post with a glowing lantern on the right reaching about halfway up the image. A few fireflies (single glow-colored pixels with one amber pixel beside them) float above the grass. Everything else is background.

True pixel art: every art pixel is a solid square block of one color, all blocks the same size, aligned to one grid. No anti-aliasing, no gradients, no blur, no texture noise, no dithering, no soft brushes.

Palette: use only these {{PALETTE_COUNT}} colors and no others: {{PALETTE}}.

Match the style of the attached reference image exactly: the same palette, pixel size relative to the subject, 1-pixel #1A1420 outlines on characters and objects, flat 3 to 4 step shading, top-down 3/4 view and light from the top-left. Warm storybook fantasy mood: mossy forests, lantern-lit towns, old stone ruins.

No text, letters or numbers unless asked for below. No signature, no watermark, no border or frame.

Background: flat solid {{KEY}} everywhere that is not art. Never use {{KEY}} inside the art.
```

## 02-11 devlog_header: devlog header

- Target: art/processed/web/devlog_header.png
- Logical size: 240×80 art pixels, centered on a landscape 3:2 canvas
- Attach: art/style/reference.png

```text
Create pixel art for a cozy fantasy online game called Lorehallow. Attached is the style reference; match it.

Canvas: landscape 3:2. The art is exactly 240 art pixels wide and 80 art pixels tall, every art pixel drawn as one solid square block of a single color, all blocks the same size. Center the art band vertically; everything above and below the band is background.

A wide banner for the game's development blog: a cozy writing desk inside a timber cottage at night, seen in top-down 3/4 view. An open journal with scribbled (unreadable) lines, a quill in an inkpot, a lit candle and a small lantern casting warm light, a rolled map, a mug, a sleeping cat on a stack of books, and a window on the right showing a dusk-blue night with a few stars. The art fills the whole 240×80 band.

True pixel art: every art pixel is a solid square block of one color, all blocks the same size, aligned to one grid. No anti-aliasing, no gradients, no blur, no texture noise, no dithering, no soft brushes.

Palette: use only these {{PALETTE_COUNT}} colors and no others: {{PALETTE}}.

Match the style of the attached reference image exactly: the same palette, pixel size relative to the subject, 1-pixel #1A1420 outlines on characters and objects, flat 3 to 4 step shading, top-down 3/4 view and light from the top-left. Warm storybook fantasy mood: mossy forests, lantern-lit towns, old stone ruins.

No text, letters or numbers unless asked for below. No signature, no watermark, no border or frame.

Background: flat solid {{KEY}} everywhere that is not art. Never use {{KEY}} inside the art.
```

## 02-12 og_card: social card background

- Target: art/processed/web/og_card.png
- Logical size: 240×126 art pixels, centered on a landscape 3:2 canvas
- Attach: art/style/reference.png

```text
Create pixel art for a cozy fantasy online game called Lorehallow. Attached is the style reference; match it.

Canvas: landscape 3:2. The art is exactly 240 art pixels wide and 126 art pixels tall, every art pixel drawn as one solid square block of a single color, all blocks the same size. Center the art band vertically; everything above and below the band is background.

A background for social media link previews. A calm wide view of rolling mossy hills under a dusk sky, with a winding path leading to a small lantern-lit village on the right third and a single adventurer with a lantern walking the path. Keep the left 60 percent simple and fairly dark (sky, hills, soft clouds), because a title and text will be placed over it. The art fills the whole 240×126 band.

True pixel art: every art pixel is a solid square block of one color, all blocks the same size, aligned to one grid. No anti-aliasing, no gradients, no blur, no texture noise, no dithering, no soft brushes.

Palette: use only these {{PALETTE_COUNT}} colors and no others: {{PALETTE}}.

Match the style of the attached reference image exactly: the same palette, pixel size relative to the subject, 1-pixel #1A1420 outlines on characters and objects, flat 3 to 4 step shading, top-down 3/4 view and light from the top-left. Warm storybook fantasy mood: mossy forests, lantern-lit towns, old stone ruins.

No text, letters or numbers unless asked for below. No signature, no watermark, no border or frame.

Background: flat solid {{KEY}} everywhere that is not art. Never use {{KEY}} inside the art.
```
