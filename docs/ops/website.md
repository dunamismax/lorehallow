# Runbook: the website

How lorehallow.com is built, deployed, rolled back and watched. The site is an Astro project in [web/](../../web/README.md); host facts are in [deploy/HOST.md](../../deploy/HOST.md); the Caddy site file is [deploy/caddy/lorehallow.caddy](../../deploy/caddy/lorehallow.caddy). Update this file whenever a procedure changes.

Since the cutover (BUILD.md 1.9, [ADR 0014](../decisions/0014-astro-website-and-phaser-browser-client.md)) the live site is the Astro build. The Go generator stays in `web/` as the rollback target until it is retired, a week after the cutover; see [The Astro cutover](#the-astro-cutover).

## Layout on the host

| Path | What |
|---|---|
| `/srv/lorehallow/web/releases/<timestamp>/` | One build of the site (the Astro output). Five are kept. |
| `/srv/lorehallow/web/releases/go-final/` | A copy of the last Go-built release, kept as the cutover's rollback target until the Go module is retired. Never deleted by deploys. |
| `/srv/lorehallow/web/current` | Symlink to the live release; Caddy's site root. Switched with one rename. |
| `/srv/lorehallow/web/releases/placeholder/` | The old holding page. Never deleted by deploys. |
| `lorehallow-deploy` | The user that owns `/srv/lorehallow/web`. Deploys run as root over SSH and hand each release to it. |

Everything under `/assets/` has a content hash in its name and is cached for a year (`immutable`). Everything else is revalidated on every request, so a deploy shows up at once.

## Build and preview locally

```bash
make web-dev        # the Astro site at http://127.0.0.1:4321, drafts included
make web-build      # the Astro site into web/dist-astro/
make web-go-dev     # the Go-built site at http://127.0.0.1:4322, drafts included
make web-go-build   # the Go-built site into web/dist/ (the rollback target until it's retired)
make web-test       # both sites' tests, and the parity check between them
```

## Deploy

```bash
make deploy-web
```

It refuses to run with uncommitted changes to the site's sources, then:

1. builds the site at the current commit with Astro, on the machine running the deploy (it needs Node and `make setup`; Node never goes to the host),
2. uploads it to a new `releases/<timestamp>/` and hands it to `lorehallow-deploy`,
3. switches `current` to it (one rename, so visitors never see half a release),
4. checks it through the host's own Caddy: `/` is served by this commit; its stylesheet and share card load, the stylesheet with immutable caching; `/` revalidates; the canonical and share-card URLs are absolute; `/progress/`, `/devlog/`, `/play/`, `/progress.json`, `/progress.svg`, both feeds and the sitemap load; `/downloads/` answers 301 to `/play/`; and a missing page returns 404,
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

## The Astro cutover

Done on 2026-11-08 (BUILD.md 1.9). What happened, so it can be repeated or undone:

1. `make web-parity` passed: the Go and Astro builds of the same commit matched on every file, apart from the differences accepted in [docs/design/website-parity.md](../design/website-parity.md).
2. The last Go-built release was copied to `releases/go-final/` on the host, so no deploy can rotate it away.
3. `make deploy-web` published the Astro build, and its health check passed.
4. `make rehearse-rollback-web` deployed an Astro release with its stylesheet missing; the check failed and the previous Astro release came back on its own.

To go back to the Go-built site (only while the Go module still exists):

```bash
ssh root@ubuntu-vm 'ln -sfn releases/go-final /srv/lorehallow/web/current.new && mv -T /srv/lorehallow/web/current.new /srv/lorehallow/web/current'
```

A week after the cutover with no rollback, the Go web module is deleted (BUILD.md 1.9) and so is `releases/go-final/`.
