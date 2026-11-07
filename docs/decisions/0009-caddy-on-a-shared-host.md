# 0009: Caddy as the reverse proxy, as one site on a shared host

- **Status:** Accepted
- **Date:** 2026-11-07
- **BUILD.md:** 1.1, 2.9

## Context

BUILD.md 1.1 recommends Caddy with automatic HTTPS. The real host ([deploy/HOST.md](../../deploy/HOST.md)) is the owner's home Hyper-V VM, which already runs Caddy for several other sites: a custom build (with caddy-ratelimit), an admin API on a Unix socket, and a main Caddyfile that imports one file per site from `/etc/caddy/sites/`. The firewall, SSH and updates are already hardened by the owner.

## Decision

Use the host's Caddy. Lorehallow is one site file, `/etc/caddy/sites/lorehallow.caddy` (source: [deploy/caddy/lorehallow.caddy](../../deploy/caddy/lorehallow.caddy)), serving `lorehallow.com` (static site root `/srv/lorehallow/web/current`), the `www` redirect and `play.lorehallow.com` (`/ws`, `/healthz`, `/version` proxied to the game server on `127.0.0.1:7420`). Certificates come from Caddy's automatic HTTPS; DNS points straight at the host (not proxied by Cloudflare), so client addresses are real and the game server trusts `X-Forwarded-For` only from loopback.

Rules that keep the other sites safe:

- Lorehallow never edits the main Caddyfile, other site files, the firewall or SSH. `deploy/bootstrap/app.sh` installs only its own pieces.
- Every change validates the whole configuration first, with the binary the service runs and as the `caddy` user (validating as root once created a root-owned log file and made the reload fail), and puts the previous file back if validation fails.
- The `/ws` proxy sets `stream_close_delay`, so a reload for any site doesn't drop every player.

## Alternatives considered

- **A separate Caddy (or nginx) for Lorehallow.** Ports 80 and 443 are taken by the shared Caddy; a second proxy would need its own ports behind it, doubling the TLS work for no gain.
- **Cloudflare-proxied DNS for play.lorehallow.com.** Works with WebSockets, but hides client addresses (per-IP limits would see Cloudflare's edges) and adds a hop to every snapshot. Revisit if the home IP needs hiding or DDoS protection.

## Consequences

- A Caddy reload is shared: anything that breaks the config breaks every site, which is why validation is mandatory and other sites are checked after each change.
- Lorehallow's TLS and proxy memory per connection lives in a process shared with other sites; measure it as part of Case 1 on `prod` (BUILD.md 12.7).
