# deploy/

Everything needed to run Lorehallow on the owner's host behind lorehallow.com.

- `HOST.md`: facts about the host (filled in during Phase 1).
- `bootstrap/`: idempotent privileged setup scripts the owner reviews and runs.
- `caddy/`: reverse proxy configuration.
- `server/`, `web/`: systemd units and deploy scripts per service.

Agents deploy only through the `make deploy-*` and `make rollback-*` targets. See [AGENTS.md section 10](../AGENTS.md#10-deployment-and-operations).
