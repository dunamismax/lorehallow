# deploy/

Everything needed to run Lorehallow on the owner's host behind lorehallow.com.

| Path | What | Status |
|---|---|---|
| `HOST.md` | Facts about the host and the owner's setup steps | Template; the owner fills it in (BUILD.md 1.1) |
| `bootstrap/host.sh` | Idempotent provisioning: users, key-only SSH (with a lockout guard), firewall, updates, fail2ban, Caddy, layout, unit, sudo rules. Has `--dry-run`. | Written and syntax-checked; not yet run on a real host |
| `caddy/Caddyfile` | Apex site, `www` redirect, `/api/status`, `play.lorehallow.com/ws` with `stream_close_delay` so reloads don't drop players | Passes `caddy validate` |
| `server/lorehallow-server.service` | systemd unit: dedicated user, sandboxing, `LimitNOFILE=65536` | Not yet installed on a host |
| `server/deploy.sh` | `make deploy-server` / `make rollback-server`: cross-compile, upload, atomic switch, restart, health check, automatic rollback | Not yet run against a host |
| `host.env` | Your local deploy target (gitignored) | |

Agents deploy only through the `make deploy-*` and `make rollback-*` targets. See [AGENTS.md section 10](../AGENTS.md#10-deployment-and-operations) and the [runbook](../docs/ops/runbook.md).
