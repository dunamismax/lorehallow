---
title: "Same site, new engine"
date: 2026-11-08
summary: "lorehallow.com is now built with Astro instead of a Go program. Nothing you can see changed, and that was the whole point."
tags: [website, phase-1]
cover: null
draft: false
---

If you visited lorehallow.com this week and noticed nothing, good. The site you're reading was rebuilt from scratch and swapped in underneath you, and every page, feed and picture came out the same.

## Why move it

The first version of the site was a small Go program. It worked well, but I want Go to stay where it earns its keep in this project: the game server and the tools around it. The website is a frontend, and Astro is built for exactly this kind of site. It still produces plain files, still ships no JavaScript, and my server still just hands those files out. Node only runs on the machine that builds the site.

## Proving nothing broke

Before writing any Astro, I made a list of everything the old site published: all 40 pages, the feeds, the progress badge, every image, every link preview, every redirect. That list is the definition of "nothing broke".

Then the new site was built next to the old one, and a check compares the two, file by file, on every commit: the same text on every page, the same feed entries, the same pixels in every image. The first comparison found one real difference: which day the footer said the site was built, because the two programs read the clock in different time zones. Once that was fixed, all 153 files matched. Only then did the new build go live.

The pixel font in the headings was the satisfying part. It's drawn by hand as little grids of text, and the build turns it into a real font file. The new code produces exactly the same font file as the old code did, byte for byte.

## Downloads is now Play

Lorehallow is going to be a browser game. When it's ready you'll open play.lorehallow.com and play, with nothing to download or install. So the Downloads page became [Play](/play/), which says honestly that there's nothing to play yet. The old address still works and takes you there.

## Same safety net

Deploys work the way they always have: each release is a new folder, one switch makes it live, a health check makes sure it actually works, and a bad release puts the previous one back by itself. I tested that again with a deliberately broken Astro release, and it rolled back on its own. The last Go-built version is kept on the server too, just in case.

Lighthouse scores the new site 99 to 100 on mobile, the same as before.

## What's next

The game client. It's being rebuilt the same way: a Phaser browser game that has to pass the same protocol and pixel checks as the old Godot skeleton before the old one goes. Then the first real milestone: two players, one small world, in your browser.
