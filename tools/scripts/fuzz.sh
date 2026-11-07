#!/usr/bin/env bash
# Runs every Go fuzz target for FUZZTIME each (default 10s). Called by
# `make fuzz`. `go test` can fuzz one target at a time, so this finds
# them all. The seed corpus of every target also runs in `make check`.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
FUZZTIME="${FUZZTIME:-10s}"
cd "$ROOT/server"

found=0
while IFS=: read -r file _ line; do
  name="$(sed -E 's/^func (Fuzz[A-Za-z0-9_]*)\(.*/\1/' <<<"$line")"
  pkg="./$(dirname "$file")"
  echo "fuzz: $pkg $name ($FUZZTIME)"
  go test -run='^$' -fuzz="^${name}\$" -fuzztime="$FUZZTIME" "$pkg"
  found=$((found + 1))
done < <(grep -rn --include='*_test.go' -E '^func Fuzz[A-Za-z0-9_]*\(f \*testing\.F\)' . | sed 's|^\./||' | sort)

if (( found == 0 )); then echo "fuzz: no fuzz targets found" >&2; exit 1; fi
echo "fuzz: $found target(s) passed"
