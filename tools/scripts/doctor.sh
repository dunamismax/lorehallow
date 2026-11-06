#!/usr/bin/env bash
# Checks that every required tool is installed at the pinned version.
# Pins live in toolchain.env. A tool passes when its major.minor matches
# the pin and its patch is the same or newer (Node: same major, newer or
# equal overall). Exit status is non-zero if anything is missing or wrong.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT/toolchain.env"
GODOT="${GODOT:-}"

case "$(uname -s)" in
  Darwin) PKG=brew ;;
  Linux) PKG=apt ;;
  *) PKG=other ;;
esac

fail=0
ok() { printf '  \033[32mok\033[0m    %-14s %s\n' "$1" "$2"; }
bad() {
  printf '  \033[31mFAIL\033[0m  %-14s %s\n' "$1" "$2"
  printf '        install: %s\n' "$3"
  fail=1
}

# version_ok HAVE WANT MODE -> 0 if acceptable.
# MODE=minor: same major.minor, patch >= wanted. MODE=major: same major, >= wanted.
version_ok() {
  local have="$1" want="$2" mode="$3"
  IFS=. read -r hM hm hp <<<"$have"
  IFS=. read -r wM wm wp <<<"$want"
  hp="${hp:-0}"; wp="${wp:-0}"; hm="${hm:-0}"
  [[ "$hM" == "$wM" ]] || return 1
  if [[ "$mode" == minor ]]; then
    [[ "$hm" == "$wm" ]] || return 1
    (( hp >= wp )) || return 1
  else
    (( hm > wm )) && return 0
    (( hm == wm && hp >= wp )) || return 1
  fi
  return 0
}

hint() { # hint BREW APT
  case "$PKG" in
    brew) echo "brew install $1" ;;
    apt) echo "$2" ;;
    *) echo "see AGENTS.md section 13" ;;
  esac
}

check() { # check NAME WANT MODE INSTALL_HINT CMD...
  local name="$1" want="$2" mode="$3" install="$4"; shift 4
  local out have
  if ! out="$("$@" 2>/dev/null)"; then
    bad "$name" "not found (want $want)" "$install"; return
  fi
  have="$(grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' <<<"$out" | head -1)"
  if version_ok "$have" "$want" "$mode"; then ok "$name" "$have"
  else bad "$name" "found $have, want $want" "$install"; fi
}

find_godot() {
  if [[ -n "$GODOT" ]]; then echo "$GODOT"; return; fi
  for c in godot godot4 /Applications/Godot.app/Contents/MacOS/Godot; do
    if command -v "$c" >/dev/null 2>&1; then command -v "$c"; return; fi
  done
  echo godot
}

echo "Lorehallow toolchain (pins in toolchain.env)"
check go "$GO_VERSION" minor "$(hint go 'https://go.dev/dl/ (tarball into /usr/local/go)')" go version
check godot "$GODOT_VERSION" minor "$(hint 'godot --cask' 'https://godotengine.org/download/linux/ (put the binary on PATH or set GODOT=)')" "$(find_godot)" --version
check node "$NODE_VERSION" major "$(hint node 'https://nodejs.org/en/download (or mise/nvm)')" node --version
check uv "$UV_VERSION" minor "$(hint uv 'curl -LsSf https://astral.sh/uv/install.sh | sh')  (or: uv self update)" uv --version
check python "$PYTHON_VERSION" minor "uv python install $PYTHON_VERSION" uv run --no-project --python "$PYTHON_VERSION" python --version
check buf "$BUF_VERSION" minor "$(hint bufbuild/buf/buf 'https://buf.build/docs/cli/installation/')" buf --version
# shellcheck disable=SC2016 # $(go env GOPATH) is printed for the user to run
check golangci-lint "$GOLANGCI_LINT_VERSION" minor "$(hint golangci-lint 'curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(go env GOPATH)/bin v'"$GOLANGCI_LINT_VERSION")" golangci-lint --version
check gdtoolkit "$GDTOOLKIT_VERSION" minor "uv tool install gdtoolkit==$GDTOOLKIT_VERSION" gdformat --version
check make 4.0 major "$(hint make 'sudo apt install make') (GNU Make 4+; on macOS put gnubin first on PATH)" make --version
check git 2.30 major "$(hint git 'sudo apt install git')" git --version

# Project-local dependencies are installed by `make setup`, which runs
# after doctor on a fresh clone, so they are reported but never fail.
note() { printf '  \033[33mnote\033[0m  %-14s %s\n' "$1" "$2"; }
for addon in "gut:$GUT_VERSION" "godobuf:$GODOBUF_VERSION"; do
  name="${addon%%:*}"; want="${addon#*:}"
  stamp="$ROOT/client/addons/$name/.lorehallow-version"
  if [[ -f "$stamp" && "$(cat "$stamp")" == "$want" ]]; then ok "$name" "$want"
  else note "$name" "not installed at $want yet; make setup installs it"; fi
done

if (( fail )); then
  echo; echo "Some tools are missing or at the wrong version."; exit 1
fi
echo; echo "All tools present."
