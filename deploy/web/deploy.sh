#!/usr/bin/env bash
# Deploys lorehallow.com (BUILD.md 1.9). Called by `make deploy-web`,
# `make rollback-web` and the deploy-web GitHub Actions workflow.
#
#   deploy/web/deploy.sh deploy     build, upload a new release, switch, check, roll back on failure
#   deploy/web/deploy.sh rollback   switch back to the release before the current one
#   deploy/web/deploy.sh rehearse-rollback
#                                   deploy a release with a broken stylesheet link and check
#                                   that the health check fails and the previous release comes
#                                   back on its own (the site is wrong for a few seconds)
#
# Needs LOREHALLOW_DEPLOY_HOST (environment or deploy/host.env), an ssh
# target such as root@ubuntu-vm, and Go to build the site.
#
# On the host:
#   /srv/lorehallow/web/releases/<timestamp>/   one directory per release
#   /srv/lorehallow/web/current -> releases/<timestamp>   what Caddy serves
#
# The switch is one rename of a symlink, so a visitor gets either the old
# release or the new one, never a mix. The last five releases are kept.
# Other directories in releases/ (the placeholder) are left alone.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"
[[ -f deploy/host.env ]] && source deploy/host.env

ACTION="${1:-deploy}"
HOST="${LOREHALLOW_DEPLOY_HOST:?set LOREHALLOW_DEPLOY_HOST (see deploy/HOST.md)}"
BASE=/srv/lorehallow/web
KEEP=5
SITE=lorehallow.com
STAMP_RE='^[0-9]{8}T[0-9]{6}Z$'

remote() { ssh -o BatchMode=yes -o ConnectTimeout=20 "$HOST" "$@"; }

releases() { # timestamped releases on the host, oldest first
  remote "ls -1 $BASE/releases" | grep -E "$STAMP_RE" | sort || true
}

current() { remote "readlink $BASE/current 2>/dev/null | xargs -r basename" || true; }

switch_to() { # $1: release directory name
  remote "ln -sfn releases/$1 $BASE/current.new && mv -T $BASE/current.new $BASE/current"
}

# check_live <short commit>: fetch pages through the host's own Caddy (so
# DNS and the home router aren't part of the test) and check that the
# expected release is the one being served, with its stylesheet and data.
check_live() {
  remote "bash -s" -- "$1" "$SITE" <<'CHECK'
set -euo pipefail
want="$1"; site="$2"
get() { curl -fsS --max-time 10 --resolve "$site:443:127.0.0.1" "https://$site$1"; }
for i in $(seq 1 10); do
  page="$(get / 2>/dev/null)" && grep -q "lorehallow-web $want" <<<"$page" && break
  [[ $i == 10 ]] && { echo "check: / isn't serving $want" >&2; exit 1; }
  sleep 1
done
css="$(grep -o '/assets/site\.[0-9a-f]*\.css' <<<"$page" | head -n1)"
[[ -n "$css" ]] || { echo "check: no stylesheet link on /" >&2; exit 1; }
get "$css" >/dev/null || { echo "check: $css doesn't load" >&2; exit 1; }
for p in /progress/ /devlog/ /downloads/ /progress.json /devlog/rss.xml; do
  get "$p" >/dev/null || { echo "check: $p doesn't load" >&2; exit 1; }
done
code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 10 --resolve "$site:443:127.0.0.1" "https://$site/no-such-page")"
[[ "$code" == 404 ]] || { echo "check: a missing page returned $code, not 404" >&2; exit 1; }
CHECK
}

BROKEN=0
if [[ "$ACTION" == rehearse-rollback ]]; then
  BROKEN=1
  ACTION=deploy
fi

case "$ACTION" in
deploy)
  commit="$(git rev-parse HEAD)"
  short="${commit:0:7}"
  if [[ -n "$(git status --porcelain -- web BUILD.md art/style/palette.hex docs/walkthroughs)" ]]; then
    echo "deploy-web: the site's sources have uncommitted changes; commit first" >&2; exit 1
  fi
  stamp="$(date -u +%Y%m%dT%H%M%SZ)"
  out="$(mktemp -d)"
  trap 'rm -rf "$out"' EXIT
  echo "deploy-web: building $short"
  (cd web && go run ./cmd/lorehallow-web build -out "$out/site")
  if [[ "$BROKEN" == 1 ]]; then
    echo "deploy-web: rehearsal: removing the stylesheet so this release fails its check"
    rm -f "$out"/site/assets/site.*.css
  fi

  previous="$(current)"
  if [[ "$BROKEN" == 1 && ! "$previous" =~ $STAMP_RE ]]; then
    echo "deploy-web: the rehearsal needs a working release to roll back to; deploy first" >&2; exit 1
  fi
  echo "deploy-web: uploading release $stamp (previous: ${previous:-none})"
  remote "mkdir -p $BASE/releases/$stamp"
  COPYFILE_DISABLE=1 tar --no-xattrs -C "$out/site" -czf - . | remote "tar -xzf - --no-same-owner -C $BASE/releases/$stamp"
  # mktemp directories are private (0700) and tar keeps that mode; Caddy
  # must be able to read the release.
  remote "chmod -R u=rwX,go=rX $BASE/releases/$stamp"
  # Releases belong to the website deploy user, so CI (which deploys as
  # that user) can clean up releases an agent deployed as root.
  remote "if [ \"\$(id -u)\" = 0 ] && id lorehallow-deploy >/dev/null 2>&1; then chown -R lorehallow-deploy:lorehallow-deploy $BASE/releases/$stamp; fi"
  switch_to "$stamp"
  if check_live "$short"; then
    if [[ "$BROKEN" == 1 ]]; then
      echo "deploy-web: rehearsal FAILED: a release without its stylesheet passed the check" >&2
      switch_to "$previous"
      remote "rm -rf $BASE/releases/$stamp"
      exit 1
    fi
    echo "deploy-web: $short is live at https://$SITE/ (release $stamp)"
    old="$(releases | awk -v keep="$KEEP" '{r[NR] = $0} END {for (i = 1; i <= NR - keep; i++) print r[i]}')"
    if [[ -n "$old" ]]; then
      echo "deploy-web: removing old releases: $(tr "\n" " " <<<"$old")"
      for r in $old; do remote "rm -rf $BASE/releases/$r"; done
    fi
    exit 0
  fi
  echo "deploy-web: health check failed" >&2
  if [[ -n "$previous" ]]; then
    echo "deploy-web: rolling back to $previous" >&2
    switch_to "$previous"
    remote "rm -rf $BASE/releases/$stamp"
    if [[ "$BROKEN" == 1 ]]; then
      echo "deploy-web: rehearsal passed: the bad release was rolled back automatically"
      exit 0
    fi
  fi
  exit 1
  ;;
rollback)
  cur="$(current)"
  prev="$(releases | grep -B1 -x "$cur" | head -n1 || true)"
  if [[ -z "$prev" || "$prev" == "$cur" ]]; then
    echo "rollback-web: no release before ${cur:-the current one}" >&2; exit 1
  fi
  echo "rollback-web: $cur -> $prev"
  switch_to "$prev"
  remote "curl -fsS --max-time 10 --resolve $SITE:443:127.0.0.1 https://$SITE/ >/dev/null"
  echo "rollback-web: $prev is live. The next deploy-web publishes a new release."
  ;;
*)
  echo "usage: $0 deploy|rollback|rehearse-rollback" >&2; exit 2
  ;;
esac
