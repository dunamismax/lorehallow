#!/usr/bin/env bash
# Installs Lorehallow's own pieces on a host that is already set up
# (BUILD.md 1.1, 2.9): users, directories, the systemd unit, the Caddy site
# file, the website deploy user and a placeholder website. Idempotent. It runs on your machine and
# does everything over SSH:
#
#   deploy/bootstrap/app.sh --dry-run    # show what it would do
#   deploy/bootstrap/app.sh              # do it (make install-host)
#
# Target: LOREHALLOW_DEPLOY_HOST (env or deploy/host.env), e.g. root@ubuntu-vm.
#
# It never touches the firewall, SSH, other sites or the host's main
# Caddyfile. It adds /etc/caddy/sites/lorehallow.caddy only if the main
# Caddyfile imports /etc/caddy/sites/*.caddy, validates the whole Caddy
# config before reloading, and puts the previous file back if validation
# fails. (For a brand-new host, harden it with deploy/bootstrap/host.sh first.)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"
[[ -f deploy/host.env ]] && source deploy/host.env
HOST="${LOREHALLOW_DEPLOY_HOST:?set LOREHALLOW_DEPLOY_HOST (see deploy/HOST.md)}"
DRY=0
[[ "${1:-}" == "--dry-run" ]] && DRY=1

stage="$(mktemp -d)"
trap 'rm -rf "$stage"' EXIT
cp deploy/server/lorehallow-server.service deploy/caddy/lorehallow.caddy "$stage/"
cp -R web/placeholder "$stage/placeholder"

echo "app.sh: staging files on $HOST"
ssh -o BatchMode=yes "$HOST" 'rm -rf /tmp/lorehallow-install && mkdir -p /tmp/lorehallow-install'
COPYFILE_DISABLE=1 tar --no-xattrs -C "$stage" -czf - . | ssh -o BatchMode=yes "$HOST" 'tar -xzf - -C /tmp/lorehallow-install'

ssh -o BatchMode=yes "$HOST" "DRY=$DRY bash -s" <<'REMOTE'
set -euo pipefail
S=/tmp/lorehallow-install
say() { printf '==> %s\n' "$*"; }
run() { if [[ "$DRY" == 1 ]]; then printf '    would run: %s\n' "$*"; else printf '    %s\n' "$*"; "$@"; fi; }
[[ $EUID -eq 0 ]] || { echo "needs root (or a user with sudo)" >&2; exit 1; }
[[ "$DRY" == 1 ]] && say "DRY RUN: nothing will change"

say "service user"
id lorehallow-server >/dev/null 2>&1 || run useradd --system --no-create-home --home-dir /nonexistent --shell /usr/sbin/nologin lorehallow-server

say "website deploy user (owns the site's releases; CI deploys as it over Tailscale SSH)"
id lorehallow-deploy >/dev/null 2>&1 || run useradd --system --create-home --home-dir /var/lib/lorehallow-deploy --shell /bin/bash lorehallow-deploy

say "directories"
run install -d -m 0755 /srv/lorehallow /srv/lorehallow/server /srv/lorehallow/server/releases /srv/lorehallow/web /srv/lorehallow/web/releases
run install -d -m 0750 -o root -g lorehallow-server /etc/lorehallow
[[ -f /etc/lorehallow/server.env ]] || run install -m 0640 -o root -g lorehallow-server /dev/null /etc/lorehallow/server.env
if [[ "$(stat -c %U /srv/lorehallow/web/releases 2>/dev/null)" != lorehallow-deploy ]]; then
  run chown -R lorehallow-deploy:lorehallow-deploy /srv/lorehallow/web
fi

say "systemd unit"
if ! cmp -s "$S/lorehallow-server.service" /etc/systemd/system/lorehallow-server.service 2>/dev/null; then
  run install -m 0644 "$S/lorehallow-server.service" /etc/systemd/system/lorehallow-server.service
  run systemctl daemon-reload
fi
run systemctl enable lorehallow-server

say "placeholder website (until BUILD.md 1.9 deploys the real one)"
if [[ ! -e /srv/lorehallow/web/current ]]; then
  run install -d -m 0755 /srv/lorehallow/web/releases/placeholder
  run cp -R "$S/placeholder/." /srv/lorehallow/web/releases/placeholder/
  run ln -sfn releases/placeholder /srv/lorehallow/web/current
else
  echo "    /srv/lorehallow/web/current exists; left alone"
fi

say "caddy site file"
grep -q 'import /etc/caddy/sites/\*.caddy' /etc/caddy/Caddyfile || { echo "the main Caddyfile doesn't import /etc/caddy/sites/*.caddy; not touching Caddy" >&2; exit 1; }
target=/etc/caddy/sites/lorehallow.caddy
if cmp -s "$S/lorehallow.caddy" "$target" 2>/dev/null; then
  echo "    $target is current"
elif [[ "$DRY" == 1 ]]; then
  echo "    would install $target, validate the whole config, then reload caddy"
else
  backup=""
  [[ -f "$target" ]] && backup="$(mktemp)" && cp "$target" "$backup"
  install -m 0644 "$S/lorehallow.caddy" "$target"
  # Validate with the binary the service runs (hosts may use a custom
  # build) and as the user it runs as: validating as root would create the
  # site's log files owned by root, and the reload would then fail.
  caddy_bin="$(systemctl show -p ExecStart --value caddy | sed -n 's/.*path=\([^ ;]*\).*/\1/p' | tail -1)"
  [[ -x "$caddy_bin" ]] || caddy_bin="$(command -v caddy)"
  caddy_user="$(systemctl show -p User --value caddy)"; caddy_user="${caddy_user:-caddy}"
  install -d -m 0755 -o "$caddy_user" -g "$caddy_user" /var/log/caddy
  if ! runuser -u "$caddy_user" -- "$caddy_bin" validate --config /etc/caddy/Caddyfile --adapter caddyfile >/tmp/lorehallow-caddy-validate.log 2>&1; then
    cat /tmp/lorehallow-caddy-validate.log >&2
    if [[ -n "$backup" ]]; then cp "$backup" "$target"; else rm -f "$target"; fi
    echo "caddy validate failed: restored the previous state, caddy not reloaded" >&2
    exit 1
  fi
  echo "    installed $target (whole config validated)"
  run systemctl reload caddy
fi
rm -rf "$S"
say "done"
REMOTE
