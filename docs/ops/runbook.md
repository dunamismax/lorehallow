# Runbook: the game server

How to run, watch, deploy and roll back `lorehallow-server` on the host. Facts about the host itself are in [deploy/HOST.md](../../deploy/HOST.md); the Caddy config is [deploy/caddy/Caddyfile](../../deploy/caddy/Caddyfile); the unit is [deploy/server/lorehallow-server.service](../../deploy/server/lorehallow-server.service).

Commands marked *(host)* run on the host: `ssh root@ubuntu-vm` (Tailscale SSH; the owner approves new sessions). Update this file whenever a procedure changes.

## Layout on the host

| Path | What |
|---|---|
| `/srv/lorehallow/server/releases/<timestamp>/` | One release: `lorehallow-server`, `content/`, `RELEASE` (version and commit) |
| `/srv/lorehallow/server/current` | Symlink to the live release, switched atomically |
| `/etc/lorehallow/server.env` | Settings and secrets (mode 0640, root:lorehallow-server); names in [.env.example](../../.env.example) |
| `/etc/caddy/sites/lorehallow.caddy` | Lorehallow's Caddy site file; the host's main Caddyfile and other sites are not ours to edit |
| `127.0.0.1:7420` | Game HTTP and `/ws` (Caddy proxies `play.lorehallow.com/ws` and `lorehallow.com/api/status` here) |
| `127.0.0.1:7421` | Admin: `/metrics` and `/debug/pprof` (loopback only, never proxied) |

## Start, stop, status *(host)*

```bash
sudo systemctl start lorehallow-server
sudo systemctl stop lorehallow-server      # graceful: every player gets "server restarting"
sudo systemctl restart lorehallow-server
systemctl status lorehallow-server
cat /srv/lorehallow/server/current/RELEASE # which build is live
curl -s localhost:7420/version
```

`stop` sends SIGTERM: the server stops accepting, sends every player `Disconnect` SHUTTING_DOWN, waits for them to close, stops the zone and exits, within `LOREHALLOW_SHUTDOWN_TIMEOUT` (20 s) and systemd's `TimeoutStopSec` (30 s).

## Logs *(host)*

```bash
journalctl -u lorehallow-server -f                   # follow
journalctl -u lorehallow-server --since "1 hour ago"
journalctl -u lorehallow-server -p warning           # warnings and errors only (tick overruns, kicks)
```

Logs are JSON in production (`LOREHALLOW_LOG_FORMAT=json`). Per-connection events are at debug level; set `LOREHALLOW_LOG_LEVEL=debug` in `server.env` and restart to see joins and disconnects.

## Health and metrics

```bash
curl -s https://play.lorehallow.com/healthz            # from anywhere
curl -s https://lorehallow.com/api/status              # public: online count, uptime, version
ssh -L 7421:127.0.0.1:7421 root@ubuntu-vm              # then, locally:
curl -s localhost:7421/metrics | grep -E 'zone_tick_seconds_count|connections_open|evicted|overruns'
go tool pprof http://localhost:7421/debug/pprof/profile?seconds=30
```

What to look at first:

| Metric | Healthy | If not |
|---|---|---|
| `lorehallow_zone_tick_overruns_total` | flat | the zone can't keep 20 Hz: take a CPU profile |
| `lorehallow_connections_evicted_total` | rare | clients too slow: check bandwidth per player (`bytes_out_total`) |
| `lorehallow_disconnects_rate_limited_total`, `lorehallow_zone_kicks_total` | rare | a misbehaving or hostile client: find its IP in the logs |
| `lorehallow_goroutines` | about 2 per connection plus a few dozen | a leak: take `/debug/pprof/goroutine?debug=1` |

## Deploy

From a clean checkout on `main` with `make check` green:

```bash
make deploy-server     # build for the host, upload, switch, restart, check /healthz, roll back on failure
```

It needs `LOREHALLOW_DEPLOY_HOST=root@ubuntu-vm` and `LOREHALLOW_DEPLOY_ARCH=amd64` (in the environment or `deploy/host.env`, which is gitignored). `make install-host` installs or updates the unit and the Caddy site file. A deploy restarts the server, so every player is disconnected with "server restarting" and reconnects; zero-downtime deploys are BUILD.md 12.4.

## Roll back

```bash
make rollback-server   # switch to the previous release and restart
```

To check that a failed deploy really rolls itself back, while nobody is playing:

```bash
make rehearse-rollback-server   # deploy a release with no map; passes only if it's rolled back automatically
```

It takes the game down for 10 to 20 seconds and removes the bad release afterwards. First run: 2026-11-07, passed.

Or by hand *(host)*: `ls /srv/lorehallow/server/releases`, then `ln -sfn releases/<previous> /srv/lorehallow/server/current.new && mv -T /srv/lorehallow/server/current.new /srv/lorehallow/server/current && sudo systemctl restart lorehallow-server`.

## When it's down

1. `systemctl status lorehallow-server`: is it running, crash-looping, or stopped?
2. `journalctl -u lorehallow-server -n 100`: the last error before it stopped.
3. A bad config refuses to start with a clear message (for example `admin-addr must be a loopback address`). Fix `/etc/lorehallow/server.env` and restart.
4. If the last deploy broke it: `make rollback-server`.
5. If Caddy is the problem (`curl localhost:7420/healthz` works on the host but the public URL doesn't): `systemctl status caddy`, `journalctl -u caddy -n 100`, and validate as Caddy itself would: `runuser -u caddy -- /usr/local/lib/caddy/caddy validate --config /etc/caddy/Caddyfile`. Never run `caddy validate` as root: it can create log files Caddy then can't open. Other sites share this Caddy; check them after any reload.
