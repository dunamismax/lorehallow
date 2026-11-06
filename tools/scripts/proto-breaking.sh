#!/usr/bin/env bash
# Runs `buf breaking` against a git ref (default main). A breaking change
# is allowed only when proto/PROTOCOL_VERSION was bumped relative to that
# ref: that is what makes it "planned". Called by `make proto-check`.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
REF="${1:-main}"
cd "$ROOT"

if ! git cat-file -e "$REF:proto/buf.yaml" 2>/dev/null; then
  echo "buf breaking: $REF has no proto/buf.yaml yet, skipping"; exit 0
fi

if buf breaking proto --against ".git#ref=$REF,subdir=proto"; then
  echo "buf breaking: no breaking changes against $REF"; exit 0
fi

old="$(git show "$REF:proto/PROTOCOL_VERSION" 2>/dev/null | tr -d '[:space:]' || echo 0)"
new="$(tr -d '[:space:]' <proto/PROTOCOL_VERSION)"
if (( new > old )); then
  echo "buf breaking: breaking changes accepted because PROTOCOL_VERSION went $old -> $new"; exit 0
fi
echo "buf breaking: unplanned breaking change. Fix it, or bump proto/PROTOCOL_VERSION ($old) if it is intended." >&2
exit 1
