# Runbook: the website

How lorehallow.com is built, deployed, rolled back and watched. The generator is in [web/](../../web/README.md); host facts are in [deploy/HOST.md](../../deploy/HOST.md); the Caddy site file is [deploy/caddy/lorehallow.caddy](../../deploy/caddy/lorehallow.caddy). Update this file whenever a procedure changes.

## Layout on the host

| Path | What |
|---|---|
| `/srv/lorehallow/web/releases/<timestamp>/` | One build of the site (the contents of `web/dist/`). Five are kept. |
| `/srv/lorehallow/web/current` | Symlink to the live release; Caddy's site root. Switched with one rename. |
| `/srv/lorehallow/web/releases/placeholder/` | The old holding page. Never deleted by deploys. |
| `lorehallow-deploy` | The user that owns `/srv/lorehallow/web`. CI deploys as it; agents deploy as root, and the deploy hands the files to it. |

Everything under `/assets/` has a content hash in its name and is cached for a year (`immutable`). Everything else is revalidated on every request, so a deploy shows up at once.

## Build and preview locally

```bash
make web-dev     # http://127.0.0.1:4321, drafts included, rebuilt on reload
make web-build   # web/dist/, exactly what deploys
make web-test    # the output tests
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

CI does the same on every push to `main` that touches the site's inputs and every night at 07:23 UTC (`.github/workflows/deploy-web.yml`), once its secrets exist (below).

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

## Owner setup: CI deploys

CI reaches the host over the owner's tailnet with Tailscale SSH and logs in as `lorehallow-deploy`, which owns nothing but the website. These steps need the Tailscale admin console and the GitHub repository settings, so they are the owner's (BUILD.md 1.9, the `HUMAN` item).

1. **Tags.** In the Tailscale admin console, Access controls, add two tags the owner (or admins) own:

   ```json
   "tagOwners": {
     "tag:ci": ["autogroup:admin"],
     "tag:lorehallow-host": ["autogroup:admin"]
   }
   ```

2. **Tag the VM.** Machines, `ubuntu-server-vm`, Edit ACL tags, add `tag:lorehallow-host`. Tailscale only lets tagged devices SSH into tagged devices. Tagging a machine moves it from the owner's identity to the tag, so check the owner's own SSH rule still covers it; if it targets `autogroup:self`, add one for the tag (the next step does).

3. **Access and SSH rules.** Let `tag:ci` reach the VM's SSH port and log in as `lorehallow-deploy` only, and keep the owner's own check-mode access:

   ```json
   "grants": [
     {"src": ["tag:ci"], "dst": ["tag:lorehallow-host"], "ip": ["tcp:22"]}
   ],
   "ssh": [
     {"action": "accept", "src": ["tag:ci"], "dst": ["tag:lorehallow-host"], "users": ["lorehallow-deploy"]},
     {"action": "check", "src": ["autogroup:member"], "dst": ["tag:lorehallow-host"], "users": ["root", "autogroup:nonroot"]}
   ]
   ```

   (If the policy file uses `acls` instead of `grants`, the equivalent is `{"action": "accept", "src": ["tag:ci"], "dst": ["tag:lorehallow-host:22"]}`.)

4. **OAuth client.** Settings, OAuth clients, Generate: scope **Auth Keys: Write**, tag `tag:ci`. Copy the client ID and secret.

5. **GitHub secrets**, from a clone of the repository:

   ```bash
   gh secret set TS_OAUTH_CLIENT_ID      # paste the client ID
   gh secret set TS_OAUTH_SECRET         # paste the client secret
   gh secret set LOREHALLOW_DEPLOY_HOST --body 'lorehallow-deploy@ubuntu-vm'
   ```

6. **Try it:**

   ```bash
   gh workflow run deploy-web.yml
   gh run watch "$(gh run list --workflow deploy-web.yml -L 1 --json databaseId -q '.[0].databaseId')"
   ```

   The log ends with `deploy-web: <commit> is live`. Until the secrets exist, the workflow still builds and tests the site and ends with a "Website not deployed" warning.

## Owner setup: uptime alerts on your phone

1. Pick a topic name nobody will guess: `echo "lorehallow-$(openssl rand -hex 8)"`.
2. Install the ntfy app (iOS or Android) and subscribe to that topic on `ntfy.sh`.
3. Store it: `gh secret set NTFY_TOPIC --body 'lorehallow-…'`.
4. Send a test: `gh workflow run uptime.yml -f test_alert=true`. The phone should show "Lorehallow uptime: test".
