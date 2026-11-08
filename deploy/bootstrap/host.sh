#!/usr/bin/env bash
# Hardens a FRESH host for Lorehallow (BUILD.md 1.1). Idempotent. Not
# needed on a host that is already hardened (ubuntu-vm is: see
# deploy/HOST.md); Lorehallow's own pieces are installed by
# deploy/bootstrap/app.sh either way. The owner reviews it and runs it
# with sudo:
#
#   sudo LOREHALLOW_ACME_EMAIL=you@example.com ./host.sh --dry-run   # print what it would do
#   sudo LOREHALLOW_ACME_EMAIL=you@example.com ./host.sh
#
# Supported: Debian 12+ and Ubuntu 22.04+ (apt, systemd). It stops on
# anything else rather than guess.
#
# What it does, in order:
#   1. packages: unattended-upgrades, fail2ban, ufw, curl, Caddy (official repo)
#   2. users: `deploy` (SSH, deploys) and `lorehallow-server` (runs the game server, no login)
#   3. SSH: key-only, no root login. REFUSES unless `deploy` already has an
#      authorized key AND the user running sudo has one, so nobody is locked out.
#   4. firewall: allow 22, 80, 443 (TCP) and 443 (UDP, HTTP/3); deny the rest.
#      Port 22 is allowed before the firewall turns on.
#   5. layout: /srv/lorehallow/{server,web}/releases, /etc/lorehallow (0750)
#   6. systemd unit for the game server, Caddyfile, sudo rules for `deploy`
#
# Changes nothing in DNS. Records every step it took in /var/log/lorehallow-bootstrap.log.
set -euo pipefail

DRY=0
[[ "${1:-}" == "--dry-run" ]] && DRY=1
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(cd "$HERE/../.." && pwd)"
LOG=/var/log/lorehallow-bootstrap.log

say() { printf '\033[1m==> %s\033[0m\n' "$*"; }
run() {
  if ((DRY)); then printf '    would run: %s\n' "$*"; return 0; fi
  printf '    %s\n' "$*"
  echo "$(date -Is) $*" >>"$LOG"
  "$@"
}
die() { printf '\033[31mhost.sh: %s\033[0m\n' "$*" >&2; exit 1; }

# --- Preflight -----------------------------------------------------------------
[[ $EUID -eq 0 ]] || die "run with sudo"
[[ -r /etc/os-release ]] || die "no /etc/os-release; unsupported system"
. /etc/os-release
case "${ID:-}" in
debian | ubuntu) ;;
*) die "unsupported distribution '${ID:-unknown}'; this script supports Debian and Ubuntu" ;;
esac
command -v systemctl >/dev/null || die "systemd is required"
[[ -n "${LOREHALLOW_ACME_EMAIL:-}" ]] || die "set LOREHALLOW_ACME_EMAIL (Caddy's certificate account address)"
for f in "$REPO/deploy/server/lorehallow-server.service" "$REPO/deploy/caddy/lorehallow.caddy"; do
  [[ -f "$f" ]] || die "missing $f: run from a checkout of the repository"
done
((DRY)) && say "DRY RUN: nothing will change"

# --- 1. Packages ------------------------------------------------------------------
say "packages"
export DEBIAN_FRONTEND=noninteractive
run apt-get update -q
run apt-get install -y -q unattended-upgrades fail2ban ufw curl gnupg debian-keyring debian-archive-keyring apt-transport-https
run dpkg-reconfigure -f noninteractive unattended-upgrades
if ! command -v caddy >/dev/null; then
  # Caddy's official apt repository (https://caddyserver.com/docs/install#debian-ubuntu-raspbian)
  if ((DRY)); then
    echo "    would add the Caddy apt repository and install caddy"
  else
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor --yes -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' >/etc/apt/sources.list.d/caddy-stable.list
    run apt-get update -q
    run apt-get install -y -q caddy
  fi
fi
run systemctl enable --now fail2ban

# --- 2. Users -----------------------------------------------------------------------
say "users"
if ! id deploy >/dev/null 2>&1; then
  run useradd --create-home --shell /bin/bash deploy
fi
if ! id lorehallow-server >/dev/null 2>&1; then
  run useradd --system --no-create-home --home-dir /nonexistent --shell /usr/sbin/nologin lorehallow-server
fi
run install -d -m 0700 -o deploy -g deploy /home/deploy/.ssh
run touch /home/deploy/.ssh/authorized_keys
run chown deploy:deploy /home/deploy/.ssh/authorized_keys
run chmod 0600 /home/deploy/.ssh/authorized_keys

# --- 3. SSH -------------------------------------------------------------------------
say "ssh: key-only, no root login"
admin="${SUDO_USER:-}"
admin_keys=""
if [[ -n "$admin" && "$admin" != root ]]; then
  admin_home="$(getent passwd "$admin" | cut -d: -f6)"
  admin_keys="$admin_home/.ssh/authorized_keys"
fi
if [[ ! -s /home/deploy/.ssh/authorized_keys ]]; then
  echo "    SKIPPED: /home/deploy/.ssh/authorized_keys is empty. Add the public keys for agent deploy sessions (never a CI key), then run this again."
elif [[ -z "$admin_keys" || ! -s "$admin_keys" ]]; then
  echo "    SKIPPED: the user running sudo (${admin:-unknown}) has no ~/.ssh/authorized_keys. Turning off passwords now could lock you out."
else
  conf=/etc/ssh/sshd_config.d/10-lorehallow.conf
  if ((DRY)); then
    echo "    would write $conf (PasswordAuthentication no, PermitRootLogin no, KbdInteractiveAuthentication no)"
  else
    cat >"$conf" <<'SSHD'
# Written by lorehallow deploy/bootstrap/host.sh
PasswordAuthentication no
KbdInteractiveAuthentication no
PermitRootLogin no
SSHD
    sshd -t || die "sshd config test failed; $conf left in place for inspection, sshd not reloaded"
    run systemctl reload ssh 2>/dev/null || run systemctl reload sshd
  fi
fi

# --- 4. Firewall ---------------------------------------------------------------------
say "firewall: 22, 80, 443 only"
if ufw status | grep -q "Status: active"; then
  echo "    SKIPPED: ufw is already active; its rules are yours (Lorehallow needs only 80/tcp, 443/tcp and 443/udp open)."
else
run ufw allow 22/tcp
run ufw allow 80/tcp
run ufw allow 443/tcp
run ufw allow 443/udp
run ufw default deny incoming
run ufw default allow outgoing
run ufw --force enable
fi

# --- 5. Layout ------------------------------------------------------------------------
say "layout"
run install -d -m 0755 -o deploy -g deploy /srv/lorehallow /srv/lorehallow/server /srv/lorehallow/server/releases /srv/lorehallow/web /srv/lorehallow/web/releases
run install -d -m 0750 -o root -g lorehallow-server /etc/lorehallow
if [[ ! -f /etc/lorehallow/server.env ]]; then
  run install -m 0640 -o root -g lorehallow-server /dev/null /etc/lorehallow/server.env
fi

# --- 6. Services ----------------------------------------------------------------------
say "systemd unit, Caddyfile, sudo rules"
run install -m 0644 "$REPO/deploy/server/lorehallow-server.service" /etc/systemd/system/lorehallow-server.service
run systemctl daemon-reload
run systemctl enable lorehallow-server
# Never replace a main Caddyfile that serves other sites. On a fresh
# Caddy, make the main file import one file per site; app.sh adds ours.
if [[ ! -d /etc/caddy/sites ]]; then
  run install -d -m 0755 /etc/caddy/sites
  if grep -q "import /etc/caddy/sites" /etc/caddy/Caddyfile 2>/dev/null; then :; else
    run cp /etc/caddy/Caddyfile "/etc/caddy/Caddyfile.before-lorehallow.$(date +%s)"
    if ((DRY)); then echo "    would write /etc/caddy/Caddyfile: import /etc/caddy/sites/*.caddy"; else
      printf '{\n\temail %s\n}\n\nimport /etc/caddy/sites/*.caddy\n' "$LOREHALLOW_ACME_EMAIL" >/etc/caddy/Caddyfile
    fi
  fi
fi
run systemctl enable --now caddy
sudoers=/etc/sudoers.d/lorehallow-deploy
if ((DRY)); then
  echo "    would write $sudoers (deploy may restart and inspect lorehallow-server and reload caddy, nothing else)"
else
  cat >"$sudoers.tmp" <<'SUDO'
# Written by lorehallow deploy/bootstrap/host.sh: what the deploy user may do as root.
deploy ALL=(root) NOPASSWD: /usr/bin/systemctl restart lorehallow-server, /usr/bin/systemctl status lorehallow-server, /usr/bin/systemctl status --no-pager lorehallow-server, /usr/bin/systemctl reload caddy
SUDO
  visudo -cf "$sudoers.tmp" >/dev/null || die "sudoers check failed"
  run install -m 0440 "$sudoers.tmp" "$sudoers"
  rm -f "$sudoers.tmp"
fi

say "done"
echo "Next: fill in deploy/HOST.md with what this host is, add DNS records, then make deploy-server."
