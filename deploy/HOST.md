# The host

Facts about the server behind lorehallow.com (BUILD.md 1.1). Agents update this after every host change. Nothing secret goes here: the repository is public.

| Fact | Value |
|---|---|
| Name | `ubuntu-server-vm`, reached as `ubuntu-vm` over Tailscale (`root@ubuntu-vm`) |
| Where | A Hyper-V VM on the owner's home server; the owner can add cores and RAM |
| OS | Ubuntu 26.04.1 LTS, systemd |
| CPU | AMD Ryzen 7 5800H, 6 vCPUs given to the VM, x86_64 (`LOREHALLOW_DEPLOY_ARCH=amd64`) |
| RAM, disk | 7.2 GiB RAM; 124 GB root disk, 73 GB free (2026-11-07) |
| Network | LAN 192.168.50.2 behind the home router, which forwards 80 and 443 to it; public IP 71.215.196.157 (home connection, may change) |
| DNS | Cloudflare. `lorehallow.com` and `www.lorehallow.com`: A records to the home IP, DNS only (not proxied). `play.lorehallow.com`: **missing, to add** |
| Email | Catch-all forwarding on Cloudflare: any address `@lorehallow.com` reaches the owner's personal inbox. Use a role address (for example `hello@lorehallow.com`, `security@lorehallow.com`) wherever the project needs a contact; never publish the personal address. |
| Shared with | Other self-hosted sites (dunamismax.com, graceandfootnotes.com, xrayservice.net and more) and a RustDesk relay. Lorehallow must never disturb them. |
| Web server | Caddy 2.11.4, a custom build at `/usr/local/lib/caddy/caddy` (with caddy-ratelimit), admin API on a Unix socket. The main `/etc/caddy/Caddyfile` imports `/etc/caddy/sites/*.caddy`; Lorehallow's file is `/etc/caddy/sites/lorehallow.caddy`. Automatic HTTPS (Let's Encrypt). |
| Firewall | ufw, already configured by the owner: 80/tcp, 443/tcp, 443/udp open; SSH only from the Tailscale range (100.64.0.0/10); RustDesk ports |
| SSH | Tailscale SSH in check mode (the owner approves new sessions in a browser); OpenSSH password logins off |
| Already hardened | fail2ban and unattended-upgrades active. `deploy/bootstrap/host.sh` is **not** needed here and must not be run (it's for fresh hosts). |

As a load-test target this machine is `prod` in [docs/design/scaling.md](../docs/design/scaling.md): 6 vCPUs of a Ryzen 7 5800H, shared with the owner's other services.

## Lorehallow on this host

| What | Where |
|---|---|
| Game server | systemd `lorehallow-server` (user `lorehallow-server`), `127.0.0.1:7420`, admin `127.0.0.1:7421` |
| Server releases | `/srv/lorehallow/server/releases/<timestamp>/`, `current` symlink |
| Website | `/srv/lorehallow/web/current` (a placeholder page until BUILD.md 1.9) |
| Settings | `/etc/lorehallow/server.env` (0640 root:lorehallow-server, empty: the unit's defaults apply) |
| Logs | `journalctl -u lorehallow-server`; Caddy access logs `/var/log/caddy/lorehallow*-access.log` |

Agents deploy as `root@ubuntu-vm` over Tailscale SSH (`deploy/host.env`, gitignored). A dedicated `deploy` user would need its own Tailscale SSH rule; that, and CI deploys (which need a Tailscale auth key), are later steps.

Install or update Lorehallow's pieces: `make install-host` (deploy/bootstrap/app.sh). Deploy the server: `make deploy-server`.

## Change log

| Date | Change | By |
|---|---|---|
| 2026-11-07 | Installed with app.sh: user `lorehallow-server`, `/srv/lorehallow`, `/etc/lorehallow`, the systemd unit, `/etc/caddy/sites/lorehallow.caddy`, placeholder website. lorehallow.com serving over HTTPS. | agent |
| 2026-11-07 | Owner set up catch-all email forwarding for lorehallow.com (Cloudflare). | owner |
| 2026-11-07 | First validation ran `caddy validate` as root, which created `/var/log/caddy/lorehallow-play-access.log` owned by root, so `systemctl reload caddy` failed (the running config and every other site stayed up). Fixed the ownership and reloaded; app.sh now validates as the caddy user with the service's binary. | agent |
