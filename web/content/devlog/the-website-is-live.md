---
title: "lorehallow.com is live"
date: 2026-11-07
summary: "Lorehallow has a home on the web: a small, hand-drawn site with no JavaScript, a live map of the build plan, and this devlog."
tags: [website, phase-1]
cover: web/devlog_header.png
draft: false
---

Lorehallow has a front door. lorehallow.com is the place to see what this project is, how far along it is, and what's being built next.

## A map of the whole plan

My favorite part is the [progress page](/progress/). Every step from an empty repository to the 1.0 launch is a checkbox in one file in the repository, BUILD.md, a little over six hundred of them. The site reads that file every time it's rebuilt and draws it as a quest map: one hand-drawn landmark per phase along a winding road, from the campfire where it all started to the castle at 1.0. Finished phases are lit, the current one says "You are here", and the rest wait in the dark.

Underneath is a feed of "quests complete": every box that's been checked, with the date and a link to the exact change that did it. The site finds them by reading the file's git history, so nobody has to remember to update it.

## Made the same way as the game

The pixel art on the site is the game's art: the same 33 colors, the same characters, drawn pixel by pixel as text files in the repository. Even the font in the headings is hand-drawn, about a hundred little letter grids that the site turns into a real web font when it's built.

The site itself is built by a small Go program, the same language as the game server. It turns the plan, these posts and the art into plain HTML files, and a web server hands them out. There's no JavaScript anywhere on the site. The fireflies drifting over the village on the front page and the flickering lantern are plain CSS, and they stop if your device asks for less motion. There are no trackers and no cookies, and nothing loads from anyone else's servers.

## Running it for real

The site runs on my own server at home, next to the game server. Part of the point of this project is learning to run things properly, so the site gets the same treatment the game will:

- **Releases.** Every deploy is a new folder of files, and a single switch makes it live, so nobody ever sees half an update.
- **Checks.** A health check makes sure the new version actually works, and switches back to the old one if it doesn't. I tested that on purpose with a deliberately broken release, and it put the old site back on its own.
- **Monitoring.** A watcher outside my network checks every five minutes that the site is up, and alerts me if it isn't.

## What's next

Two changes come next. The site is being rebuilt with Astro, a tool made for exactly this kind of site. It will look the same, keep every address, and still ship no JavaScript; the Go version stays live until the new one matches it page for page. Then the game itself becomes a browser game: when Phase 2 lands, playing will mean opening play.lorehallow.com, with nothing to download or install. You can watch it happen on the [progress page](/progress/), and posts like this one will turn up here and in the [RSS feed](/devlog/rss.xml).
