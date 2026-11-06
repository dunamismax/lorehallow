#!/usr/bin/env bash
# Installs the pinned Godot addons (GUT, godobuf) into client/addons/.
# Archives are verified against the SHA-256 sums in toolchain.env.
# Safe to re-run: an addon already at the pinned version is left alone.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT/toolchain.env"
ADDONS="$ROOT/client/addons"
mkdir -p "$ADDONS"

sha256() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | cut -d' ' -f1
  else shasum -a 256 "$1" | cut -d' ' -f1; fi
}

# install NAME VERSION SHA URL ARCHIVE_SUBDIR STAMP
install_addon() {
  local name="$1" version="$2" sum="$3" url="$4" subdir="$5"
  local dest="$ADDONS/$name" stamp="$ADDONS/$name/.lorehallow-version"
  if [[ -f "$stamp" && "$(cat "$stamp")" == "$version" ]]; then
    echo "  $name $version already installed"; return
  fi
  local tmp; tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' RETURN
  echo "  downloading $name $version"
  curl -fsSL "$url" -o "$tmp/a.tgz"
  local got; got="$(sha256 "$tmp/a.tgz")"
  if [[ "$got" != "$sum" ]]; then
    echo "checksum mismatch for $name: got $got, want $sum" >&2; exit 1
  fi
  mkdir -p "$tmp/x"
  tar xzf "$tmp/a.tgz" -C "$tmp/x"
  rm -rf "$dest"
  cp -R "$tmp"/x/*/"$subdir" "$dest"
  echo "$version" >"$stamp"
  echo "  installed $name $version"
}

echo "Godot addons:"
install_addon gut "$GUT_VERSION" "$GUT_SHA256" \
  "https://github.com/bitwes/Gut/archive/refs/tags/v$GUT_VERSION.tar.gz" addons/gut
install_addon godobuf "$GODOBUF_VERSION" "$GODOBUF_SHA256" \
  "https://github.com/oniksan/godobuf/archive/refs/tags/v$GODOBUF_VERSION.tar.gz" addons/godobuf
