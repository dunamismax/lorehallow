# Runbook: the website

How lorehallow.com is built, deployed, rolled back and watched. The generator is in [web/](../../web/README.md); host facts are in [deploy/HOST.md](../../deploy/HOST.md); the Caddy site file is [deploy/caddy/lorehallow.caddy](../../deploy/caddy/lorehallow.caddy). Update this file whenever a procedure changes.

Everything here describes the Go-built site that is live today. The planned Astro site keeps the same release model; what changes is in [Planned: the Astro cutover](#planned-the-astro-cutover) at the end.

## Layout on the host

| Path | What |
|---|---|
| `/srv/lorehallow/web/releases/<timestamp>/` | One build of the site (the contents of `web/dist/`). Five are kept. |
| `/srv/lorehallow/web/current` | Symlink to the live release; Caddy's site root. Switched with one rename. |
| `/srv/lorehallow/web/releases/placeholder/` | The old holding page. Never deleted by deploys. |
| `lorehallow-deploy` | The user that owns `/srv/lorehallow/web`. Deploys run as root over SSH and hand each release to it. |

Everything under `/assets/` has a content hash in its name and is cached for a year (`immutable`). Everything else is revalidated on every request, so a deploy shows up at once.

## Build and preview locally

```bash
make web-dev        # the Astro site at http://127.0.0.1:4321, drafts included
make web-build      # the Astro site into web/dist-astro/
make web-go-dev     # the Go-built site at http://127.0.0.1:4322, drafts included
make web-go-build   # the Go-built site into web/dist/, exactly what deploys until the cutover (BUILD.md 1.9)
make web-test       # both sites' tests, and the parity check between them
```

## Deploy

```bash
make deploy-web
```

It refuses to run with uncommitted changes to the site's sources, then:

1. builds the site at the current commit,
2. uploads it to a new `releases/<timestamp>/` and hands it to `lorehallow-deploy`,
3. switches `current` to it (one rename, so visitors never see half a release),
4. checks it through the host's own Caddy: `/` is served by this commit, its stylesheet loads, `/progress/`, `/devlog/`, `/downloads/`, `/progress.json` and the RSS feed load, and a missing page returns 404,
5. if any check fails, switches back to the previous release and deletes the new one,
6. deletes all but the newest five releases.

Deploys are always run by hand, from the owner's machine or an agent's session, over SSH. `make check` in CI builds and tests the site on every push but never deploys it.

## Roll back

```bash
make rollback-web
```

It switches `current` to the release before the live one. The newer release stays on disk, so the next `make deploy-web` (or the next push) moves forward again. To rehearse the automatic rollback, which deploys a release with its stylesheet missing and checks that the previous release comes back on its own:

```bash
make rehearse-rollback-web
```

Last rehearsed: see the change log in [deploy/HOST.md](../../deploy/HOST.md).

## Check it by hand

```bash
curl -sI https://lorehallow.com/ | grep -iE 'HTTP|content-security|cache-control'
curl -s https://lorehallow.com/progress.json | head
ssh root@ubuntu-vm 'readlink /srv/lorehallow/web/current; ls /srv/lorehallow/web/releases'
```

Caddy's access log: `/var/log/caddy/lorehallow-access.log` *(host)*.

## Uptime alerts

`.github/workflows/uptime.yml` checks the home page, `/progress.json` and `play.lorehallow.com/healthz` every five minutes from GitHub's runners ([ADR 0013](../decisions/0013-uptime-monitoring.md)). After five minutes of failures it sends one ntfy alert and fails the run (GitHub emails the owner about failed runs); it sends one more alert when everything is back.

When an alert comes in:

1. `curl -sI https://lorehallow.com/` from somewhere outside the home network (a phone on mobile data).
2. If nothing on the host answers, the cause is upstream: the home connection, the router's port forwarding, or a changed home IP (the Cloudflare A records must follow it; deploy/HOST.md).
3. If the host answers but lorehallow.com doesn't: `systemctl status caddy` *(host)* and the Caddy logs. Never edit other sites' files.
4. If the page is wrong rather than down: `make rollback-web`.
5. If only play.lorehallow.com is down, see the [game server runbook](runbook.md#when-its-down).

## Owner setup: uptime alerts on your phone

1. Pick a topic name nobody will guess: `echo "lorehallow-$(openssl rand -hex 8)"`.
2. Install the ntfy app (iOS or Android) and subscribe to that topic on `ntfy.sh`.
3. Store it: `gh secret set NTFY_TOPIC --body 'lorehallow-…'`.
4. Send a test: `gh workflow run uptime.yml -f test_alert=true`. The phone should show "Lorehallow uptime: test".

## Planned: the Astro cutover

Not done yet (BUILD.md 1.9, [ADR 0014](../decisions/0014-astro-website-and-phaser-browser-client.md)). When it happens, this section becomes the procedure and the rest of the file is updated to match.

- **Same release model.** `make deploy-web` still uploads a timestamped release, switches `current` atomically, checks the release through the host's Caddy, rolls back on failure and keeps five releases. The difference is the build step: it runs the Astro build on the machine doing the deploy. Node never goes to the host.
- **More checks.** The health check adds Astro's hashed assets (served with immutable cache headers, while pages revalidate), absolute canonical and share-card URLs, both feeds, `/progress.json`, `/progress.svg`, `/play/` and the `/downloads/` redirect. It still requires a missing page to return HTTP 404.
- **Caddy.** The cache rule follows Astro's asset folder. `/downloads/` gets a permanent redirect to `/play/`. The CSP stays `script-src 'none'`, separate from play.lorehallow.com's game CSP. Validate as the caddy user before reloading, as always.
- **Cutover order.** First, the owner reviews the local Astro build and the parity report and approves. Then `make deploy-web` publishes the Astro build and the live checks are confirmed. The last Go-built release stays as the rollback target, so `make rollback-web` returns to it. After that, `make rehearse-rollback-web` is run with an Astro release.
- **After a week with no rollback,** the Go web module is deleted (BUILD.md 1.9).
