---
title: "Laying the foundations"
date: 2026-11-06
summary: "Before there is a world to walk around in, there has to be a workshop. Phase 0 builds the tools Lorehallow will be made with, including a way to draw its pixel art by hand, in text."
tags: [phase-0, tooling, art]
cover: null
draft: true
---

<!-- DRAFT for the owner to edit and publish. Written in the owner's voice; change anything. -->

Lorehallow is a small pixel-art MMO built around one feeling: you log in, and other people are already there. There's nothing to play yet. This first stretch of work, Phase 0, is about building the workshop before building the world.

## One command to check everything

The project mixes four languages: Go for the server, GDScript for the Godot client, Python for the art tools, and soon TypeScript for the website. It would be easy for each corner to drift into its own habits. Instead there's one entry point: `make check` runs every formatter, linter and test in the repository, and the CI server runs exactly the same command on every push. If it's green on my machine, it's green everywhere.

## Teaching two languages to agree

The server and the client talk in Protocol Buffers, a compact binary format described by a shared schema. The Go side gets an official code generator. GDScript doesn't, so the client uses a community generator, and I didn't want to just trust it. So the server writes a set of "golden" messages to disk, including awkward ones like the largest and smallest 64-bit numbers, and both the server and the client have to read them back and write out the exact same bytes. If either side ever disagrees, the build fails.

## Real pixel art, drawn by hand

The art started in an image generator, which is surprisingly good at the look of pixel art but never quite produces the real thing: a 32-pixel-wide character arrives as a 1024-pixel image where every "pixel" is a slightly blurry blob of slightly-off color.

So I first built an intake pipeline that recovers the real pixels. The trick for finding the hidden grid is a small piece of signal processing: if you add up where the color changes from one column to the next, you get a row of spikes, one at every block boundary, and a Fourier transform finds their spacing even when the blocks are 5.33 screen pixels wide and wobble a bit. After that, every block is snapped to a locked 33-color palette and the magenta background is removed.

It worked, and it showed exactly what a generator can't do. Characters came out half again too tall. Tiles that should repeat seamlessly had visible edges. A 16-pixel icon was an unreadable smudge. So the art changed hands: the AI agents that write Lorehallow's code now draw its pixel art too, pixel by pixel, as plain text files in the repository. A sprite is a little grid of letters, one letter per palette color; ground tiles are short scripts that place clumps of grass or cobbles on a canvas that wraps around its edges, so a tile is seamless by construction. One command turns the text into images and checks every rule: exact size, palette only, feet on row 45. The build fails if an image ever drifts from its source.

Everything from the first two batches is now redrawn that way: the terrain, the hero, a tree, a rock, a lantern post, a cottage, the logo, the icons, twenty landmarks for the progress map and a lineup of five adventurers. Only the big painted scenes for the website are still generated.

## The look

Lorehallow's palette is 33 colors: mossy greens, lantern golds, old-stone greys, dusk blues, warm earth and a few accents for banners and magic. Light always comes from the top-left, characters get a one-pixel dark outline, and everything sits on a 32-pixel grid. The style reference is a small scene built from the real game tiles and sprites, so the picture every new piece of art is checked against is made of the same pieces the game will use.

## What's next

With the first art in, Phase 1 builds lorehallow.com: the site you're reading this on, with a live progress page that updates itself from the project's build plan every time a task is checked off.
