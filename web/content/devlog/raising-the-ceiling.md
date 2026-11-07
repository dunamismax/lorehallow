---
title: "Raising the ceiling"
date: 2026-11-07
summary: "Lorehallow's world stays small and handcrafted, but the server behind it is now aiming for hundreds to thousands of players at once. Here's what that means, and how I'll know if it's true."
tags: [server, networking, scaling]
cover: null
draft: true
---

<!-- DRAFT for the owner to edit and publish. Written in the owner's voice; change anything. -->

When I started Lorehallow, the plan said "dozens of players is the real target; hundreds is a stretch goal." That was a sensible, modest line for a hobby MMO. I've changed my mind about it.

The world will stay small: a few handcrafted zones, warm pixel art, a town that feels lived in. But the part of this project I'm proudest of, and the part I most want to learn from, is the server. So the goal is now **hundreds to thousands of simultaneous, active players**, on a Go server built to a professional standard: fast, reliable, secure and easy to work on.

## A target, not a claim

"Thousands of players" is a phrase that gets thrown around a lot. I don't want to be one more project saying it without proof. So there's a rule now: Lorehallow never claims a player count it hasn't measured. Every number comes from a load test, run on named hardware, with the result written down in the repository.

There's nothing to measure yet, which means the honest answer to "how many players can it handle?" today is: none. Ask again in a few weeks.

## Three different problems

"How many players" turns out to be three questions, and they stress completely different things.

1. **Lots of quiet connections.** Thousands of people logged in, mostly standing around. The cost is memory per connection, goroutines and heartbeats. Being cheap when nothing happens.
2. **Lots of active players, spread out.** Everyone moving and fighting, but spread across the world. The total work is large, but it divides neatly across zones and CPU cores.
3. **A crowd.** Three hundred players in the town square, all able to see each other. This is the hard one. Every player needs to hear about every other player, so the work grows with the square of the crowd: 300 players means 90,000 updates every tick. More CPU cores don't help, because it's all one place. The only answer is to send less: only what's near you, only what changed, the far-away people a bit less often, and each update encoded once and shared by everyone who needs it.

Each case gets its own targets, rising from about 100, to 1,000, to several thousand, and its own load test. They're written up in the [scaling model](https://github.com/dunamismax/lorehallow/blob/main/docs/design/scaling.md).

## How the server is shaped

The design is deliberately simple, and it's one I've wanted to build properly for a long time:

- **Each zone is owned by one goroutine.** Its tick loop, twenty times a second, is the only code that touches that zone's world. No locks, no races, and the game rules can be tested tick by tick with a fake clock.
- **Everything else talks to it through bounded queues.** If one player's connection is too slow to keep up, their queue fills and they're disconnected. The rest of the world never waits for them.
- **A bot army comes first.** Before most game features exist, there will be a headless bot client that can run thousands of fake players from one process, so every feature after it can be measured as it lands.
- **Splitting comes later, if it's needed.** The server starts as a single process with many zones. Zones only talk to each other by message, so when a measurement shows one process is the limit, they can move into separate processes, and then separate machines, without a rewrite. Not before.

## What changes in the plan

The build plan grew a lot of backend detail: connection lifecycles and backpressure in the very first multiplayer phase, interest management and profiling much earlier than before, measurable performance targets on almost every phase, and a new Phase 20 for running across several machines. The website keeps going in parallel, since it's waiting on server setup anyway.

The game itself doesn't get bigger. It gets a much better engine room.
