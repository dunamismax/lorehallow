# The host

Facts about the server behind lorehallow.com (BUILD.md 1.1). **The owner fills this in**; agents update it after every host change. Nothing secret goes here: no passwords, keys or private IPs you don't want public (the repository is public).

| Fact | Value |
|---|---|
| Provider or location (home, VPS, ...) | _to fill in_ |
| OS and version | _to fill in_ (host.sh supports Debian 12+ and Ubuntu 22.04+) |
| CPU architecture | _to fill in_ (`uname -m`: x86_64 means `LOREHALLOW_DEPLOY_ARCH=amd64`, aarch64 means `arm64`) |
| CPU model and cores | _to fill in_ (`lscpu`), so load-test results can name this hardware (`prod` in docs/design/scaling.md) |
| RAM and disk | _to fill in_ (`free -h`, `df -h`) |
| Public IPv4 and IPv6 | _to fill in_ (or: behind a home router with 80 and 443 forwarded) |
| DNS provider | _to fill in_ |
| Already running on it | _to fill in_ (anything listening on 80 or 443 conflicts with Caddy) |
| SSH | `deploy@<host>` for agents and CI, key-only after host.sh |

## Setup steps (the owner's part)

1. Fill in the table above.
2. Point A (and AAAA) records for `lorehallow.com`, `www.lorehallow.com` and `play.lorehallow.com` at the host.
3. On the host, from a checkout of the repository: add the agent and CI public keys to `/home/deploy/.ssh/authorized_keys` (create the `deploy` user first with `sudo useradd -m deploy` if you like, or let a first run create it and skip the SSH step), then:

   ```bash
   sudo LOREHALLOW_ACME_EMAIL=you@example.com deploy/bootstrap/host.sh --dry-run
   sudo LOREHALLOW_ACME_EMAIL=you@example.com deploy/bootstrap/host.sh
   ```

   It refuses to turn off SSH passwords until both `deploy` and your own account have authorized keys, so it can't lock you out.
4. Locally, create `deploy/host.env` (gitignored) with `LOREHALLOW_DEPLOY_HOST=deploy@<host>` and `LOREHALLOW_DEPLOY_ARCH=<amd64|arm64>`, then `make deploy-server`.

## Change log

| Date | Change | By |
|---|---|---|
