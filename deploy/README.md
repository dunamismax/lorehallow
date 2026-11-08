# deploy/

Everything needed to run Lorehallow on the owner's host behind lorehallow.com.

| Path | What | Status |
|---|---|---|
| `HOST.md` | Facts about the host (`ubuntu-vm`) and its change log | Filled in |
| `bootstrap/app.sh` | `make install-host`: Lorehallow's own pieces over SSH (users, directories, unit, Caddy site file, placeholder site). Never touches the firewall, SSH, other sites or the main Caddyfile; validates the whole Caddy config before reloading. `--dry-run`. | Run on ubuntu-vm 2026-11-07 |
| `bootstrap/host.sh` | Hardening for a fresh host only (users, key-only SSH with a lockout guard, firewall, updates, fail2ban, Caddy). Skips an already-active firewall and never replaces a main Caddyfile. | Not used on ubuntu-vm (already hardened) |
| `caddy/lorehallow.caddy` | Site file: apex, `www` redirect, `play.lorehallow.com/ws` with `stream_close_delay` so reloads don't drop players. Planned ([ADR 0014](../docs/decisions/0014-astro-website-and-phaser-browser-client.md)): the Astro site's cache rules and `/downloads/` redirect (BUILD.md 1.9), and the game's static build, `/api/` proxy and its own CSP on play.lorehallow.com (2.9) | Installed on ubuntu-vm |
| `server/lorehallow-server.service` | systemd unit: dedicated user, sandboxing, `LimitNOFILE=65536` | Installed on ubuntu-vm |
| `server/deploy.sh` | `make deploy-server` / `make rollback-server`: cross-compile, upload, atomic switch, restart, health check, automatic rollback | |
| `client/deploy.sh` | Planned (BUILD.md 2.9): `make deploy-client` / `make rollback-client` for the Phaser browser game's static build, the same release model as the website | Not written yet |
| `web/deploy.sh` | Run by hand, never from CI: `make deploy-web` / `make rollback-web` / `make rehearse-rollback-web`: build the site, upload a timestamped release, switch `current` atomically, check it through the host's Caddy, roll back on failure, keep five releases | |
| `uptime.sh` | The Uptime workflow's check: home page, `/progress.json` and `play.lorehallow.com/healthz`, one ntfy alert per outage ([ADR 0013](../docs/decisions/0013-uptime-monitoring.md)) | |
| `host.env` | Your local deploy target (gitignored) | |

Agents deploy only through the `make deploy-*` and `make rollback-*` targets. See [AGENTS.md section 10](../AGENTS.md#10-deployment-and-operations) and the [runbook](../docs/ops/runbook.md).
