#!/usr/bin/env bash
# Writes client/override.cfg so the client knows its build version and
# commit. Godot loads override.cfg on top of project.godot at startup.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cat >"$ROOT/client/override.cfg" <<CFG
; Written by make (tools/scripts/write-client-buildinfo.sh). Not committed.
[application]
config/version="${1:-dev}"

[lorehallow]
build/commit="${2:-unknown}"
CFG
