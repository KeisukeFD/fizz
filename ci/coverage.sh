#!/usr/bin/env bash
# Coverage floor over the library packages. Tests of every package (including
# internal/ ones) count, but only statements of the library packages are
# measured: examples/, internal/, tools/ and docs/compat are excluded.
# Usage: ci/coverage.sh [floor-percent]   (default 90)
set -euo pipefail

floor="${1:-90}"
pkgs="$(go list ./... | grep -vE '/(examples|internal|tools)(/|$)|/docs/compat$' | paste -sd, -)"

go test -count=1 -covermode=atomic -coverpkg="$pkgs" -coverprofile=coverage.txt ./...

go tool cover -func=coverage.txt | sort -k3 -n | head -15
total="$(go tool cover -func=coverage.txt | awk '/^total:/ {sub("%","",$3); print $3}')"
echo "total coverage: ${total}% (floor ${floor}%)"
if ! awk -v t="$total" -v f="$floor" 'BEGIN { exit !(t+0 >= f+0) }'; then
  echo "::error::coverage ${total}% is below the ${floor}% floor"
  exit 1
fi
