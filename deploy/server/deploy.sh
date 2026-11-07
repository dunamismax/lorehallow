#!/usr/bin/env bash
# Deploys the game server to the host (BUILD.md 2.9). Called by
# `make deploy-server` and `make rollback-server`.
#
#   deploy/server/deploy.sh deploy     build, upload, switch, restart, check, roll back on failure
#   deploy/server/deploy.sh rollback   switch back to the previous release and restart
#   deploy/server/deploy.sh rehearse-rollback
#                                      deploy a release that can't start (no map) and
#                                      check that the automatic rollback restores the
#                                      previous one; the game is down for about 20 s
#
# Needs (from deploy/HOST.md, set in the environment or deploy/host.env):
#   LOREHALLOW_DEPLOY_HOST   ssh target, for example deploy@lorehallow.com
#   LOREHALLOW_DEPLOY_ARCH   amd64 or arm64 (the host's CPU)
#
# On the host (set up by deploy/bootstrap/host.sh):
#   /srv/lorehallow/server/releases/<timestamp>/   one directory per release
#   /srv/lorehallow/server/current -> releases/<timestamp>
#   the deploy user may run exactly `sudo systemctl restart|status lorehallow-server`
#
# A release is the server binary plus content/ (maps and data). The last
# five releases are kept.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"
[[ -f deploy/host.env ]] && source deploy/host.env

ACTION="${1:-deploy}"
HOST="${LOREHALLOW_DEPLOY_HOST:?set LOREHALLOW_DEPLOY_HOST (see deploy/HOST.md)}"
ARCH="${LOREHALLOW_DEPLOY_ARCH:?set LOREHALLOW_DEPLOY_ARCH to amd64 or arm64 (see deploy/HOST.md)}"
BASE=/srv/lorehallow/server
KEEP=5

remote() { ssh -o BatchMode=yes "$HOST" "$@"; }

check_health() {
  # The server listens on loopback; check it from the host. Up to 15 s.
  remote 'for i in $(seq 1 30); do curl -fsS --max-time 1 http://127.0.0.1:7420/healthz >/dev/null && exit 0; sleep 0.5; done; exit 1'
}

switch_to() { # $1: release directory name
  remote "ln -sfn releases/$1 $BASE/current.new && mv -T $BASE/current.new $BASE/current"
  remote "sudo systemctl restart lorehallow-server"
}

BROKEN=0
if [[ "$ACTION" == rehearse-rollback ]]; then
  BROKEN=1
  ACTION=deploy
fi

case "$ACTION" in
deploy)
  version="$(git describe --tags --always --dirty)"
  commit="$(git rev-parse HEAD)"
  if [[ "$version" == *-dirty ]]; then
    echo "deploy: the working tree has uncommitted changes; commit first" >&2; exit 1
  fi
  stamp="$(date -u +%Y%m%dT%H%M%SZ)"
  out="$(mktemp -d)"
  trap 'rm -rf "$out"' EXIT
  echo "deploy: building $version for linux/$ARCH"
  (cd server && CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" go build -trimpath \
    -ldflags "-s -w -X github.com/dunamismax/lorehallow/server/internal/buildinfo.Version=$version -X github.com/dunamismax/lorehallow/server/internal/buildinfo.Commit=$commit" \
    -o "$out/lorehallow-server" ./cmd/lorehallow-server)
  cp -R content "$out/content"
  echo "$version $commit" >"$out/RELEASE"
  if [[ "$BROKEN" == 1 ]]; then
    echo "deploy: rehearsal: removing the map so this release can't start"
    rm -rf "$out/content/maps"
  fi

  previous="$(remote "readlink $BASE/current 2>/dev/null | xargs -r basename" || true)"
  if [[ "$BROKEN" == 1 && -z "$previous" ]]; then
    echo "deploy: rehearsal needs a working release to roll back to" >&2; exit 1
  fi
  echo "deploy: uploading release $stamp (previous: ${previous:-none})"
  remote "mkdir -p $BASE/releases/$stamp"
  COPYFILE_DISABLE=1 tar --no-xattrs -C "$out" -czf - . | remote "tar -xzf - --no-same-owner -C $BASE/releases/$stamp"
  # The staging directory is private (0700) and tar keeps that mode; the
  # service user must be able to enter and read the release.
  remote "chown -R root:root $BASE/releases/$stamp && chmod -R u=rwX,go=rX $BASE/releases/$stamp"
  switch_to "$stamp"
  if check_health; then
    if [[ "$BROKEN" == 1 ]]; then
      echo "deploy: rehearsal FAILED: a release without a map passed the health check" >&2
      switch_to "$previous"
      check_health || true
      remote "rm -rf $BASE/releases/$stamp"
      exit 1
    fi
    echo "deploy: $version is live"
    remote "cd $BASE/releases && ls -1 | sort | head -n -$KEEP | xargs -r rm -rf"
    exit 0
  fi
  echo "deploy: health check failed" >&2
  if [[ -n "$previous" ]]; then
    echo "deploy: rolling back to $previous" >&2
    switch_to "$previous"
    if check_health; then
      echo "deploy: rolled back to $previous" >&2
      if [[ "$BROKEN" == 1 ]]; then
        remote "rm -rf $BASE/releases/$stamp"
        echo "deploy: rehearsal passed: the bad release was rolled back automatically"
        exit 0
      fi
    fi
  fi
  remote "sudo systemctl status --no-pager lorehallow-server" >&2 || true
  exit 1
  ;;
rollback)
  current="$(remote "readlink $BASE/current | xargs basename")"
  previous="$(remote "ls -1 $BASE/releases | sort | grep -B1 -x '$current' | head -n 1")"
  if [[ -z "$previous" || "$previous" == "$current" ]]; then
    echo "rollback: no release before $current" >&2; exit 1
  fi
  echo "rollback: $current -> $previous"
  switch_to "$previous"
  check_health && echo "rollback: $previous is live"
  ;;
*)
  echo "usage: $0 deploy|rollback|rehearse-rollback" >&2; exit 2
  ;;
esac
