#!/usr/bin/env bash
# Installs the pinned system tools on a Linux x86_64 CI runner: Godot,
# buf, golangci-lint and gdtoolkit. Go and uv come from their setup
# actions. Versions come from toolchain.env; downloads are checked against
# each release's published checksum file.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT/toolchain.env"
BIN="${CI_TOOLS_BIN:-$HOME/.local/bin}"
CACHE="${CI_TOOLS_CACHE:-$HOME/.cache/lorehallow-tools}"
mkdir -p "$BIN" "$CACHE"

fetch() { # fetch URL DEST
  [[ -f "$2" ]] || curl -fsSL --retry 3 "$1" -o "$2"
}

# Godot (standard build, headless-capable)
godot_zip="Godot_v${GODOT_VERSION}-stable_linux.x86_64.zip"
godot_base="https://github.com/godotengine/godot/releases/download/${GODOT_VERSION}-stable"
fetch "$godot_base/$godot_zip" "$CACHE/$godot_zip"
fetch "$godot_base/SHA512-SUMS.txt" "$CACHE/godot-${GODOT_VERSION}-SHA512-SUMS.txt"
(cd "$CACHE" && grep " $godot_zip\$" "godot-${GODOT_VERSION}-SHA512-SUMS.txt" | sha512sum -c -)
unzip -o -q "$CACHE/$godot_zip" -d "$CACHE/godot"
install -m 0755 "$CACHE/godot/Godot_v${GODOT_VERSION}-stable_linux.x86_64" "$BIN/godot"

# buf
buf_base="https://github.com/bufbuild/buf/releases/download/v${BUF_VERSION}"
fetch "$buf_base/buf-Linux-x86_64" "$CACHE/buf-${BUF_VERSION}"
fetch "$buf_base/sha256.txt" "$CACHE/buf-${BUF_VERSION}-sha256.txt"
want="$(grep ' buf-Linux-x86_64$' "$CACHE/buf-${BUF_VERSION}-sha256.txt" | cut -d' ' -f1)"
echo "$want  $CACHE/buf-${BUF_VERSION}" | sha256sum -c -
install -m 0755 "$CACHE/buf-${BUF_VERSION}" "$BIN/buf"

# golangci-lint
gl="golangci-lint-${GOLANGCI_LINT_VERSION}-linux-amd64"
gl_base="https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCI_LINT_VERSION}"
fetch "$gl_base/$gl.tar.gz" "$CACHE/$gl.tar.gz"
fetch "$gl_base/golangci-lint-${GOLANGCI_LINT_VERSION}-checksums.txt" "$CACHE/$gl-checksums.txt"
(cd "$CACHE" && grep " $gl.tar.gz\$" "$gl-checksums.txt" | sha256sum -c -)
tar -xzf "$CACHE/$gl.tar.gz" -C "$CACHE"
install -m 0755 "$CACHE/$gl/golangci-lint" "$BIN/golangci-lint"

# gdtoolkit (gdformat, gdlint)
uv tool install --quiet "gdtoolkit==${GDTOOLKIT_VERSION}"

echo "CI tools installed into $BIN"
