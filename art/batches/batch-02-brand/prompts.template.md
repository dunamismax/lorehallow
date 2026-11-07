# Batch 02: brand and website art

The public face of Lorehallow: the logo, emblem and the website's art. Same palette and rules as the game, at larger sizes.

**How to run this batch (owner):**

1. Batch 01 must be accepted first: every prompt here attaches its reference sheet.
2. Open a new ChatGPT conversation with image generation.
3. For each prompt below, in order: attach `art/style/reference.png`, paste the prompt, and send.
4. Download each result and drop it, unchanged and with its original filename, into `art/inbox/batch-02/`.
5. If an image comes out obviously wrong (smooth painting, misspelled logo, text where none was asked for), ask ChatGPT to try again before downloading.

An agent then maps the files, runs `make art-intake BATCH=02`, exports the favicon set and writes corrected prompts into `report.md` for anything that needs another try. The logo and hero art also need the owner's approval (BUILD.md 0.9).

## 02-01 logo_wordmark: logo wordmark

- Target: art/processed/web/logo_wordmark.png
- Logical size: 192×64 art pixels, centered on a landscape 3:2 canvas
- Attach: art/style/reference.png

```text
Create pixel art for a cozy fantasy online game called Lorehallow. Attached is the style reference; match it.

Canvas: landscape 3:2. The art is exactly 192 art pixels wide and 64 art pixels tall, every art pixel drawn as one solid square block of a single color, all blocks the same size. Center the art; everything above and below it is background.

The word "Lorehallow" as a pixel-art logo wordmark, hand-drawn letters, not a font. Chunky, friendly fantasy capitals and lowercase, about 28 to 34 art pixels tall, filling most of the width. Letters in lantern gold with amber and ember shading, glow highlights on the top-left edges, a 1-pixel #1A1420 outline and a 1 to 2 pixel dark night plum drop to the bottom-right. A small vine of moss-green leaves curls around the first L and a tiny glowing lantern hangs from the end of the w. Spell it exactly: L-o-r-e-h-a-l-l-o-w.

True pixel art: every art pixel is a solid square block of one color, all blocks the same size, aligned to one grid. No anti-aliasing, no gradients, no blur, no texture noise, no dithering, no soft brushes.

Palette: use only these {{PALETTE_COUNT}} colors and no others: {{PALETTE}}.

Match the style of the attached reference image exactly: the same palette, pixel size relative to the subject, 1-pixel #1A1420 outlines on characters and objects, flat 3 to 4 step shading, top-down 3/4 view and light from the top-left. Warm storybook fantasy mood: mossy forests, lantern-lit towns, old stone ruins.

No text, letters or numbers unless asked for below. No signature, no watermark, no border or frame.

Background: flat solid {{KEY}} everywhere that is not art. Never use {{KEY}} inside the art.
```

## 02-02 emblem: square emblem

- Target: art/processed/icons/emblem.png
- Logical size: 32×32 art pixels (square)
- Attach: art/style/reference.png

```text
Create pixel art for a cozy fantasy online game called Lorehallow. Attached is the style reference; match it.

Canvas: square. The emblem is exactly 32 art pixels wide and 32 art pixels tall, every art pixel drawn as one solid square block of a single color, all blocks the same size. Draw it in the middle of the canvas, about 40 percent of the canvas width, with big chunky art pixels that are clearly visible; everything around it is background.

A square emblem for the game, readable as a tiny favicon: a warm glowing lantern hanging from a curled branch with two moss-green leaves, in front of a round dusk-blue disc. The disc touches the edges of the 32×32 square; the corners outside the disc are background. Strong silhouette, high contrast, 1-pixel #1A1420 outline around the disc and the lantern. No letters.

True pixel art: every art pixel is a solid square block of one color, all blocks the same size, aligned to one grid. No anti-aliasing, no gradients, no blur, no texture noise, no dithering, no soft brushes.

Palette: use only these {{PALETTE_COUNT}} colors and no others: {{PALETTE}}.

Match the style of the attached reference image exactly: the same palette, pixel size relative to the subject, 1-pixel #1A1420 outlines on characters and objects, flat 3 to 4 step shading, top-down 3/4 view and light from the top-left. Warm storybook fantasy mood: mossy forests, lantern-lit towns, old stone ruins.

No text, letters or numbers unless asked for below. No signature, no watermark, no border or frame.

Background: flat solid {{KEY}} everywhere that is not art. Never use {{KEY}} inside the art.
```

## 02-03 emblem_16: 16 px emblem

- Target: art/processed/icons/emblem_16.png
- Logical size: 16×16 art pixels (square)
- Attach: art/style/reference.png

```text
Create pixel art for a cozy fantasy online game called Lorehallow. Attached is the style reference; match it.

Canvas: square. The emblem is exactly 16 art pixels wide and 16 art pixels tall, every art pixel drawn as one solid square block of a single color, all blocks the same size. Draw it in the middle of the canvas, about 30 percent of the canvas width, with very big chunky art pixels: only 16 across; everything around it is background.

The same emblem as the previous image, redrawn by hand for 16×16 art pixels: a round dusk-blue disc with a glowing lantern in front and one leaf. Only the essential shapes; every pixel counts. 1-pixel #1A1420 outline around the disc. The corners outside the disc are background. No letters.

True pixel art: every art pixel is a solid square block of one color, all blocks the same size, aligned to one grid. No anti-aliasing, no gradients, no blur, no texture noise, no dithering, no soft brushes.

Palette: use only these {{PALETTE_COUNT}} colors and no others: {{PALETTE}}.

Match the style of the attached reference image exactly: the same palette, pixel size relative to the subject, 1-pixel #1A1420 outlines on characters and objects, flat 3 to 4 step shading, top-down 3/4 view and light from the top-left. Warm storybook fantasy mood: mossy forests, lantern-lit towns, old stone ruins.

No text, letters or numbers unless asked for below. No signature, no watermark, no border or frame.

Background: flat solid {{KEY}} everywhere that is not art. Never use {{KEY}} inside the art.
```

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

## 02-08 character_lineup: character lineup

- Target: art/processed/web/character_lineup.png
- Logical size: 192×128 art pixels (landscape 3:2)
- Attach: art/style/reference.png

```text
Create pixel art for a cozy fantasy online game called Lorehallow. Attached is the style reference; match it.

Canvas: landscape 3:2. The art is exactly 192 art pixels wide and 128 art pixels tall, every art pixel drawn as one solid square block of a single color, all blocks the same size. The art fills the whole canvas edge to edge.

Five adventurers standing side by side on a short strip of grass and dirt along the bottom, facing the viewer, in top-down 3/4 view, each at exactly the same scale as the character in the reference (about 30 art pixels from feet to top of head, about one quarter of the image height, 2½ heads tall), all feet on the same row near the bottom. Evenly spaced across the width. From left to right: a knight in slate armor with a banner-red tabard and round shield; a ranger in a moss-green hood with a bow; a scholar-mage in arcane violet robes with a glowing staff; a cheerful bard in amber with a lute; a healer in mist and parchment white with a teal glass lantern. Varied skin tones and hair. Each has a strong, distinct silhouette. Above them is background.

True pixel art: every art pixel is a solid square block of one color, all blocks the same size, aligned to one grid. No anti-aliasing, no gradients, no blur, no texture noise, no dithering, no soft brushes.

Palette: use only these {{PALETTE_COUNT}} colors and no others: {{PALETTE}}.

Match the style of the attached reference image exactly: the same palette, pixel size relative to the subject, 1-pixel #1A1420 outlines on characters and objects, flat 3 to 4 step shading, top-down 3/4 view and light from the top-left. Warm storybook fantasy mood: mossy forests, lantern-lit towns, old stone ruins.

No text, letters or numbers unless asked for below. No signature, no watermark, no border or frame.

Background: flat solid {{KEY}} everywhere that is not art. Never use {{KEY}} inside the art.
```

## 02-09 section_icons: section icons

- Target: art/processed/icons/section_icons.png
- Logical size: 4 columns × 4 rows of 16×16 icons (64×64 total, square)
- Attach: art/style/reference.png

```text
Create pixel art for a cozy fantasy online game called Lorehallow. Attached is the style reference; match it.

Canvas: square. The art is exactly 64 art pixels wide and 64 art pixels tall, every art pixel drawn as one solid square block of a single color, all blocks the same size. A grid of 4 columns × 4 rows of icons, each exactly 16×16 art pixels, no gaps or borders between them, filling the canvas edge to edge.

Sixteen small pixel-art icons for a website, one per cell, each centered in its cell with a 1-pixel #1A1420 outline and readable as a silhouette at 16×16. In reading order (left to right, top to bottom):
1 two small figures side by side (a shared world); 2 a sprouting seedling (small but alive); 3 a magnifying glass over a pixel grid (readable art); 4 a crown on a stone pedestal (the server decides); 5 a small flag on a hill (playable milestones);
6 a compass (explore); 7 a scroll with a seal (quest); 8 three overlapping round shields (group up); 9 crossed swords (fight); 10 a quill and inkpot (devlog);
11 a winding road with a flag at the end (progress); 12 a wooden chest with a downward arrow (downloads); 13 a tiny globe made of a leaf and land (world); 14 an open book (about); 15 a speech bubble with a question mark (FAQ); 16 an open scroll with angle brackets (source code).
Every cell's background is the background color.

True pixel art: every art pixel is a solid square block of one color, all blocks the same size, aligned to one grid. No anti-aliasing, no gradients, no blur, no texture noise, no dithering, no soft brushes.

Palette: use only these {{PALETTE_COUNT}} colors and no others: {{PALETTE}}.

Match the style of the attached reference image exactly: the same palette, pixel size relative to the subject, 1-pixel #1A1420 outlines on characters and objects, flat 3 to 4 step shading, top-down 3/4 view and light from the top-left. Warm storybook fantasy mood: mossy forests, lantern-lit towns, old stone ruins.

No text, letters or numbers unless asked for below. No signature, no watermark, no border or frame.

Background: flat solid {{KEY}} everywhere that is not art. Never use {{KEY}} inside the art.
```

## 02-10 progress_landmarks: progress map landmarks

- Target: art/processed/web/progress_landmarks.png
- Logical size: 5 columns × 4 rows of 32×32 landmarks (160×128 total)
- Attach: art/style/reference.png

```text
Create pixel art for a cozy fantasy online game called Lorehallow. Attached is the style reference; match it.

Canvas: landscape. The art is exactly 160 art pixels wide and 128 art pixels tall, every art pixel drawn as one solid square block of a single color, all blocks the same size. A grid of 5 columns × 4 rows of cells, each exactly 32×32 art pixels, no gaps or borders between them. Center the grid; anything outside it is background.

Twenty small landmark buildings and places for a quest map that shows the journey from an empty campsite to a finished castle. One landmark per cell, each centered and standing on a tiny patch of grass at the bottom of its cell, in top-down 3/4 view, with a 1-pixel #1A1420 outline, all at a consistent scale, each a clear silhouette. In reading order (left to right, top to bottom):
1 a campfire with two logs; 2 a wooden signpost with two arrows; 3 a crossroads stone with two paths; 4 a market stall with a striped awning; 5 a cozy inn with a hanging sign;
6 a bell tower; 7 two straw training dummies; 8 a tall wooden watchtower; 9 an open treasure chest with gold; 10 a quest board with pinned notes;
11 a camp of two tents with a banner; 12 an arched stone bridge over a stream; 13 a lighthouse with a warm light; 14 a guild hall with banners; 15 a small stage with a lute on it;
16 a harbor pier with a small sailboat; 17 a stone city gate with a portcullis; 18 a mountain pass between two peaks; 19 a giant ancient tree with lanterns in its branches; 20 a grand castle with banner-red flags.
Every cell's empty space is the background color. Notes and banners have no readable text.

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

## 02-13 not_found: 404 page art

- Target: art/processed/web/not_found.png
- Logical size: 160×120 art pixels, centered on a landscape 3:2 canvas
- Attach: art/style/reference.png

```text
Create pixel art for a cozy fantasy online game called Lorehallow. Attached is the style reference; match it.

Canvas: landscape 3:2. The art is exactly 160 art pixels wide and 120 art pixels tall, every art pixel drawn as one solid square block of a single color, all blocks the same size. Center the art; anything outside it is background.

A lost adventurer (same scale and style as the reference character: about 30 art pixels tall, one quarter of the image height) standing in a misty forest clearing, scratching their head and holding a map upside down, beside a broken wooden signpost whose arrows point in different directions. A curious little fox peeks from behind a mossy rock. Top-down 3/4 view, gentle humor, warm storybook mood. The art fills the whole 160×120 area.

True pixel art: every art pixel is a solid square block of one color, all blocks the same size, aligned to one grid. No anti-aliasing, no gradients, no blur, no texture noise, no dithering, no soft brushes.

Palette: use only these {{PALETTE_COUNT}} colors and no others: {{PALETTE}}.

Match the style of the attached reference image exactly: the same palette, pixel size relative to the subject, 1-pixel #1A1420 outlines on characters and objects, flat 3 to 4 step shading, top-down 3/4 view and light from the top-left. Warm storybook fantasy mood: mossy forests, lantern-lit towns, old stone ruins.

No text, letters or numbers unless asked for below. No signature, no watermark, no border or frame.
```
