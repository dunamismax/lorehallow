# 0013: Uptime monitoring from GitHub Actions, with ntfy alerts

- **Status:** Accepted
- **Date:** 2026-11-07
- **BUILD.md:** 1.9

## Context

lorehallow.com and play.lorehallow.com run on a VM on the owner's home server, behind a home connection whose IP can change (deploy/HOST.md). The owner should hear about it when the site has been down for five minutes. A monitor on the host itself can't see the most likely failures (the home connection, the router, the VM), so the check has to run somewhere else. Anything that needs a new account or a paid plan is the owner's call (AGENTS.md section 2).

## Decision

**A scheduled GitHub Actions workflow checks the site every five minutes and alerts through ntfy.**

- `.github/workflows/uptime.yml` runs `deploy/uptime.sh` on a `*/5 * * * *` schedule from GitHub's runners, outside the owner's network.
- It checks three things: the home page answers and was built by `lorehallow-web`, `/progress.json` has its phases, and `play.lorehallow.com/healthz` answers. A failure is retried once a minute; the site counts as down only when it is still failing five minutes after the first failure.
- It remembers the last state in the Actions cache, so it alerts once when something goes down and once when it comes back, not on every run.
- Alerts are a high-priority push to `https://ntfy.sh/<topic>` (ntfy needs no account; the topic name is the secret and lives in the `NTFY_TOPIC` repository secret). The run that first finds the site down also fails, so GitHub's own failed-run email reaches the owner even before ntfy is set up.
- `workflow_dispatch` with `test_alert` sends one test notification, to check the phone gets it.

## Alternatives considered

- **A hosted monitor (UptimeRobot, Better Stack, Healthchecks.io).** More precise timing and status pages, but each needs an account in the owner's name. It can be added later without changing anything here.
- **A check on the host (a systemd timer).** It can't see the home connection or the VM going down, which are the likeliest outages.
- **The game server's own metrics and alerting.** That's Phase 12 (observability). This is a simple external probe and stays useful when that exists.

## Consequences

- GitHub runs scheduled workflows on a best-effort basis: runs can start several minutes late when GitHub is busy. An alert can arrive 5 to 15 minutes after the site goes down, not exactly at five minutes.
- GitHub disables scheduled workflows in a repository with no activity for 60 days. Pushes to this repository keep it alive; if work pauses for two months, re-enable the workflow in the Actions tab.
- Each check uses a minute or two of Actions time on the free public-repository allowance (up to six minutes when something is down).
- The owner must pick a topic, add the `NTFY_TOPIC` secret and subscribe to the topic in the ntfy app (docs/ops/website.md). Until then, alerts are the failed-run emails only.
